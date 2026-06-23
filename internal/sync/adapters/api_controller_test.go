package adapters_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	inventoryadapters "ferrowin/internal/inventory/adapters"
	inventorydomain "ferrowin/internal/inventory/domain"
	"ferrowin/internal/shared/idempotency"
	"ferrowin/internal/sync/adapters"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func setupControllerTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Create tables
	queries := []string{
		`CREATE TABLE terminals (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			is_active BOOLEAN DEFAULT TRUE
		)`,
		`CREATE TABLE invoicing_series (
			id TEXT PRIMARY KEY,
			terminal_id TEXT REFERENCES terminals(id) ON DELETE RESTRICT,
			prefix TEXT UNIQUE NOT NULL,
			next_sequence INTEGER NOT NULL DEFAULT 1,
			empresa_id TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE invoice (
			id TEXT PRIMARY KEY,
			delivery_note_id TEXT,
			terminal_id TEXT REFERENCES terminals(id) ON DELETE RESTRICT,
			invoicing_series_id TEXT REFERENCES invoicing_series(id) ON DELETE RESTRICT,
			invoice_number TEXT UNIQUE NOT NULL,
			sequence_number INTEGER NOT NULL,
			total REAL NOT NULL,
			status TEXT,
			empresa_id TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			firma_registro TEXT,
			hash_anterior TEXT,
			datos_encadenamiento TEXT
		)`,
		`CREATE TABLE invoice_lineas (
			id TEXT PRIMARY KEY,
			invoice_id TEXT NOT NULL,
			item_id TEXT NOT NULL,
			cantidad REAL NOT NULL,
			precio_unitario REAL NOT NULL,
			warehouse_id TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE stock_ledger_movements (
			id TEXT PRIMARY KEY,
			item_id TEXT NOT NULL,
			warehouse_id TEXT NOT NULL,
			quantity REAL NOT NULL,
			movement_type TEXT NOT NULL CHECK (movement_type IN ('RECEIPT', 'WITHDRAWAL', 'SYNC_ADJUSTMENT', 'RETURN')),
			reference_document_type TEXT,
			reference_document_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE registro_sucesos (
			id TEXT PRIMARY KEY,
			fecha_hora TEXT NOT NULL,
			tipo_evento TEXT NOT NULL,
			detalles TEXT NOT NULL,
			estado_sincronizacion TEXT NOT NULL DEFAULT 'PENDING'
		)`,
		`CREATE TABLE registro_eventos (
			id TEXT PRIMARY KEY,
			documento_tipo TEXT NOT NULL,
			documento_id TEXT NOT NULL,
			empresa_id TEXT NOT NULL,
			accion TEXT NOT NULL,
			usuario_id TEXT,
			detalles TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, q := range queries {
		_, err = db.Exec(q)
		if err != nil {
			db.Close()
			t.Fatalf("failed to run query %q: %v", q, err)
		}
	}

	cleanup := func() {
		db.Close()
	}
	return db, cleanup
}

// mockInvoiceGenerator implements adapters.InvoiceNumberGenerator for testing.
type mockInvoiceGenerator struct {
	invoiceNumber string
	seq           int
}

func (m *mockInvoiceGenerator) GenerateFacturaNumber(ctx context.Context, terminalID uuid.UUID) (string, int, error) {
	return m.invoiceNumber, m.seq, nil
}

func TestSalesSyncController_HandleSyncSales(t *testing.T) {
	ctx := context.Background()
	db, cleanup := setupControllerTestDB(t)
	defer cleanup()

	// 1. Initialize services & controller
	tracker := idempotency.NewTracker(db, true)
	err := tracker.InitSchema(ctx)
	if err != nil {
		t.Fatalf("failed to init idempotency schema: %v", err)
	}

	ledgerRepo := inventoryadapters.NewSQLStockLedgerRepository(db, true)
	invService := inventorydomain.NewInventoryService(ledgerRepo)

	controller := adapters.NewSalesSyncController(db, true, invService, tracker, &mockInvoiceGenerator{invoiceNumber: "S1-16", seq: 16})
	defaultWarehouse := uuid.New()
	controller.SetDefaultWarehouse(defaultWarehouse)

	// 2. Seed Terminal and Invoicing Series
	terminalID := uuid.New()
	_, err = db.Exec("INSERT INTO terminals (id, name, is_active) VALUES (?, 'TPV-01', 1)", terminalID.String())
	if err != nil {
		t.Fatalf("failed to seed terminal: %v", err)
	}

	seriesID := uuid.New()
	empresaID := uuid.New()
	_, err = db.Exec("INSERT INTO invoicing_series (id, terminal_id, prefix, next_sequence, empresa_id) VALUES (?, ?, 'S1', 10, ?)", seriesID.String(), terminalID.String(), empresaID.String())
	if err != nil {
		t.Fatalf("failed to seed series: %v", err)
	}

	t.Run("Scenario: Missing or invalid idempotency key", func(t *testing.T) {
		// No key
		reqBody := `{"sales": []}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/sales", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}

		// Invalid key format
		req = httptest.NewRequest(http.MethodPost, "/api/v1/sync/sales", bytes.NewBufferString(reqBody))
		req.Header.Set("Idempotency-Key", "not-a-uuid")
		w = httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Scenario: Successful sync allows negative stock and registers movements", func(t *testing.T) {
		idemKey := uuid.New().String()
		saleID := uuid.New().String()
		itemID := uuid.New().String()

		firma := "firmar_registro_hash"
		hashAnt := "hash_anterior_val"
		datosEnc := "datos_encadenamiento_val"

		syncReq := adapters.SyncRequest{
			Sales: []adapters.SyncSale{
				{
					ID:               saleID,
					NumeroFactura:    "S1-16",
					NumeroSecuencia:  16,
					CreatedAt:        "2026-06-05T13:00:00Z",
					Total:            150.00,
					Items: []adapters.SyncItem{
						{
							ItemID:    itemID,
							Quantity:  2.0,
							UnitPrice: 75.00,
						},
					},
					FirmaRegistro:       &firma,
					HashAnterior:        &hashAnt,
					DatosEncadenamiento: &datosEnc,
				},
			},
		}

		reqBody, _ := json.Marshal(syncReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/sales", bytes.NewReader(reqBody))
		req.Header.Set("Idempotency-Key", idemKey)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp adapters.SyncResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Status != "success" {
			t.Errorf("expected status success, got %s", resp.Status)
		}
		if resp.SyncedCount != 1 {
			t.Errorf("expected synced_count 1, got %d", resp.SyncedCount)
		}
		if len(resp.ProcessedIDs) != 1 || resp.ProcessedIDs[0] != saleID {
			t.Errorf("expected processed IDs to contain %s, got %v", saleID, resp.ProcessedIDs)
		}

		// Verify database contains the invoice
		var dbInvoiceCount int
		err = db.QueryRow("SELECT COUNT(*) FROM invoice WHERE id = ?", saleID).Scan(&dbInvoiceCount)
		if err != nil {
			t.Fatalf("failed to query invoice: %v", err)
		}
		if dbInvoiceCount != 1 {
			t.Errorf("expected 1 invoice in DB, got %d", dbInvoiceCount)
		}

		// Verify database contains the invoice with chaining data
		var dbFirma, dbHashAnt, dbDatosEnc sql.NullString
		err = db.QueryRow("SELECT firma_registro, hash_anterior, datos_encadenamiento FROM invoice WHERE id = ?", saleID).Scan(&dbFirma, &dbHashAnt, &dbDatosEnc)
		if err != nil {
			t.Fatalf("failed to query invoice chaining fields: %v", err)
		}
		if !dbFirma.Valid || dbFirma.String != firma {
			t.Errorf("expected firma_registro %q, got %q", firma, dbFirma.String)
		}
		if !dbHashAnt.Valid || dbHashAnt.String != hashAnt {
			t.Errorf("expected hash_anterior %q, got %q", hashAnt, dbHashAnt.String)
		}
		if !dbDatosEnc.Valid || dbDatosEnc.String != datosEnc {
			t.Errorf("expected datos_encadenamiento %q, got %q", datosEnc, dbDatosEnc.String)
		}

		// Verify central stock drops to -2 (showing negative stock was allowed and registered)
		itemUUID, _ := uuid.Parse(itemID)
		stock, err := invService.GetAvailableStock(ctx, itemUUID, defaultWarehouse)
		if err != nil {
			t.Fatalf("failed to get available stock: %v", err)
		}
		if stock != -2.0 {
			t.Errorf("expected stock to be -2.0, got %f", stock)
		}

		// Verify that stock movements are successfully persisted in the database via the sync controller
		var movementCount int
		var movementQty float64
		var movementType, refDocType, refDocID string
		err = db.QueryRow(
			"SELECT COUNT(*), SUM(quantity), movement_type, reference_document_type, reference_document_id FROM stock_ledger_movements WHERE item_id = ?",
			itemID,
		).Scan(&movementCount, &movementQty, &movementType, &refDocType, &refDocID)
		if err != nil {
			t.Fatalf("failed to query stock_ledger_movements: %v", err)
		}
		if movementCount != 1 {
			t.Errorf("expected 1 movement in DB, got %d", movementCount)
		}
		if movementQty != -2.0 {
			t.Errorf("expected movement quantity to be -2.0, got %f", movementQty)
		}
		if movementType != "SYNC_ADJUSTMENT" {
			t.Errorf("expected movement_type to be SYNC_ADJUSTMENT, got %q", movementType)
		}
		if refDocType != "INVOICE" {
			t.Errorf("expected reference_document_type to be INVOICE, got %q", refDocType)
		}
		if refDocID != saleID {
			t.Errorf("expected reference_document_id to be %q, got %q", saleID, refDocID)
		}

		// Scenario: Duplicate sync payload with the same Idempotency-Key
		// The endpoint must return the exact same response and NOT duplicate DB records.
		reqDup := httptest.NewRequest(http.MethodPost, "/api/v1/sync/sales", bytes.NewReader(reqBody))
		reqDup.Header.Set("Idempotency-Key", idemKey)
		reqDup.Header.Set("Content-Type", "application/json")

		wDup := httptest.NewRecorder()
		controller.ServeHTTP(wDup, reqDup)

		if wDup.Code != http.StatusOK {
			t.Fatalf("expected duplicate to return 200 OK, got %d", wDup.Code)
		}

		var respDup adapters.SyncResponse
		if err := json.Unmarshal(wDup.Body.Bytes(), &respDup); err != nil {
			t.Fatalf("failed to unmarshal duplicate response: %v", err)
		}

		if respDup.Status != "success" || respDup.SyncedCount != 1 || respDup.ProcessedIDs[0] != saleID {
			t.Errorf("duplicate response does not match: %+v", respDup)
		}

		// Ensure records were not duplicated
		err = db.QueryRow("SELECT COUNT(*) FROM invoice WHERE id = ?", saleID).Scan(&dbInvoiceCount)
		if err != nil {
			t.Fatalf("failed to query invoice: %v", err)
		}
		if dbInvoiceCount != 1 {
			t.Errorf("expected still exactly 1 invoice in DB, got %d", dbInvoiceCount)
		}

		stock, err = invService.GetAvailableStock(ctx, itemUUID, defaultWarehouse)
		if err != nil {
			t.Fatalf("failed to get available stock: %v", err)
		}
		if stock != -2.0 {
			t.Errorf("expected stock to remain -2.0, got %f (indicating no duplicate movements were added)", stock)
		}
	})
}

func TestSalesSyncController_HandleSyncVoids(t *testing.T) {
	db, cleanup := setupControllerTestDB(t)
	defer cleanup()

	ctx := context.Background()
	tracker := idempotency.NewTracker(db, true)
	err := tracker.InitSchema(ctx)
	if err != nil {
		t.Fatalf("failed to init idempotency schema: %v", err)
	}

	ledgerRepo := inventoryadapters.NewSQLStockLedgerRepository(db, true)
	invService := inventorydomain.NewInventoryService(ledgerRepo)
	controller := adapters.NewSalesSyncController(db, true, invService, tracker, &mockInvoiceGenerator{invoiceNumber: "S1-16", seq: 16})

	// Seed terminal, series, and invoice for void testing
	terminalID := uuid.New()
	empresaID := uuid.New()
	_, _ = db.Exec("INSERT INTO terminals (id, name, is_active) VALUES (?, 'TPV-VOID', 1)", terminalID.String())
	seriesID := uuid.New()
	_, _ = db.Exec("INSERT INTO invoicing_series (id, terminal_id, prefix, next_sequence, empresa_id) VALUES (?, ?, 'V1', 1, ?)", seriesID.String(), terminalID.String(), empresaID.String())

	saleID := uuid.New().String()
	itemID := uuid.New().String()
	warehouseID := uuid.New()
	_, _ = db.Exec(`INSERT INTO invoice (id, terminal_id, invoicing_series_id, invoice_number, sequence_number, total, status, empresa_id, created_at)
		VALUES (?, ?, ?, 'V1-1', 1, 100.00, 'Issued', ?, datetime('now'))`, saleID, terminalID.String(), seriesID.String(), empresaID.String())
	_, _ = db.Exec(`INSERT INTO invoice_lineas (id, invoice_id, item_id, cantidad, precio_unitario, warehouse_id)
		VALUES (?, ?, ?, 2.0, 50.00, ?)`, uuid.New().String(), saleID, itemID, warehouseID.String())

	// Record initial stock movement (simulate the sale that will be voided)
	docType := "INVOICE"
	saleUUID, _ := uuid.Parse(saleID)
	itemUUID, _ := uuid.Parse(itemID)
	_, _ = invService.RecordSyncAdjustment(ctx, itemUUID, warehouseID, 2.0, &docType, &saleUUID)

	t.Run("Scenario: Valid void sync returns 200 and reverses stock", func(t *testing.T) {
		idemKey := uuid.New().String()
		body := `{"sale_id":"` + saleID + `","reason":"Operator error","firma_registro":"void-sig-001","hash_anterior":"prev-hash-001"}`

		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/voids", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idemKey)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp adapters.SyncVoidResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Status != "success" {
			t.Errorf("expected status 'success', got %q", resp.Status)
		}

		// Verify ANULACION event was stored in registro_sucesos
		var tipo, estado string
		var detalles string
		err = db.QueryRow("SELECT tipo_evento, detalles, estado_sincronizacion FROM registro_sucesos WHERE tipo_evento = 'ANULACION'").Scan(&tipo, &detalles, &estado)
		if err != nil {
			t.Fatalf("failed to query void event from db: %v", err)
		}
		if tipo != "ANULACION" {
			t.Errorf("expected tipo_evento ANULACION, got %s", tipo)
		}
		if estado != "SINCRONIZADO" {
			t.Errorf("expected estado_sincronizacion SINCRONIZADO, got %s", estado)
		}
		if !strings.Contains(detalles, saleID) {
			t.Errorf("expected detalles to contain sale_id %s, got %s", saleID, detalles)
		}

		// Verify invoice status was updated to Anulado
		var invStatus string
		err = db.QueryRow("SELECT status FROM invoice WHERE id = ?", saleID).Scan(&invStatus)
		if err != nil {
			t.Fatalf("failed to query invoice status: %v", err)
		}
		if invStatus != "Anulado" {
			t.Errorf("expected invoice status 'Anulado', got %q", invStatus)
		}
	})

	t.Run("Scenario: Void non-existent sale returns 404", func(t *testing.T) {
		idemKey := uuid.New().String()
		fakeSaleID := uuid.New().String()
		body := `{"sale_id":"` + fakeSaleID + `","reason":"Ghost sale","firma_registro":"void-sig-ghost","hash_anterior":"prev-hash-ghost"}`

		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/voids", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idemKey)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found for non-existent sale, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Scenario: Void already-voided sale returns 409", func(t *testing.T) {
		idemKey := uuid.New().String()
		body := `{"sale_id":"` + saleID + `","reason":"Also wrong","firma_registro":"void-sig-002","hash_anterior":"prev-hash-002"}`

		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/voids", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idemKey)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict for already-voided sale, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "sale already voided") {
			t.Errorf("expected 'sale already voided' error, got: %s", w.Body.String())
		}
	})

	t.Run("Scenario: Duplicate void with same idempotency key returns cached response", func(t *testing.T) {
		// Create a fresh invoice for this test
		freshSaleID := uuid.New().String()
		_, _ = db.Exec(`INSERT INTO invoice (id, terminal_id, invoicing_series_id, invoice_number, sequence_number, total, status, empresa_id, created_at)
			VALUES (?, ?, ?, 'V1-2', 2, 50.00, 'Issued', ?, datetime('now'))`, freshSaleID, terminalID.String(), seriesID.String(), empresaID.String())
		_, _ = db.Exec(`INSERT INTO invoice_lineas (id, invoice_id, item_id, cantidad, precio_unitario, warehouse_id)
			VALUES (?, ?, ?, 1.0, 50.00, ?)`, uuid.New().String(), freshSaleID, itemID, warehouseID.String())

		idemKey := uuid.New().String()
		body := `{"sale_id":"` + freshSaleID + `","reason":"Duplicate test","firma_registro":"void-sig-999","hash_anterior":"prev-hash-999"}`

		// First request
		req1 := httptest.NewRequest(http.MethodPost, "/api/v1/sync/voids", bytes.NewBufferString(body))
		req1.Header.Set("Content-Type", "application/json")
		req1.Header.Set("Idempotency-Key", idemKey)
		w1 := httptest.NewRecorder()
		controller.ServeHTTP(w1, req1)
		if w1.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w1.Code, w1.Body.String())
		}

		// Duplicate request with same key
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/sync/voids", bytes.NewBufferString(body))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Idempotency-Key", idemKey)
		w2 := httptest.NewRecorder()
		controller.ServeHTTP(w2, req2)

		if w2.Code != http.StatusOK {
			t.Errorf("expected duplicate 200 OK, got %d", w2.Code)
		}
		if w1.Body.String() != w2.Body.String() {
			t.Errorf("duplicate response body differs:\noriginal: %s\nduplicate: %s", w1.Body.String(), w2.Body.String())
		}
	})

	t.Run("Scenario: Missing required fields returns 400", func(t *testing.T) {
		idemKey := uuid.New().String()

		tests := []struct {
			name string
			body string
		}{
			{"empty sale_id", `{"sale_id":"","reason":"err","firma_registro":"sig","hash_anterior":"hash"}`},
			{"empty reason", `{"sale_id":"` + uuid.New().String() + `","reason":"","firma_registro":"sig","hash_anterior":"hash"}`},
			{"empty firma_registro", `{"sale_id":"` + uuid.New().String() + `","reason":"err","firma_registro":"","hash_anterior":"hash"}`},
			{"empty hash_anterior", `{"sale_id":"` + uuid.New().String() + `","reason":"err","firma_registro":"sig","hash_anterior":""}`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/voids", bytes.NewBufferString(tt.body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Idempotency-Key", idemKey)
				w := httptest.NewRecorder()
				controller.ServeHTTP(w, req)

				if w.Code != http.StatusBadRequest {
					t.Errorf("expected 400 Bad Request, got %d", w.Code)
				}
				if !strings.Contains(w.Body.String(), "missing required fields") {
					t.Errorf("expected 'missing required fields' error, got: %s", w.Body.String())
				}
			})
		}
	})

	t.Run("Scenario: Missing Idempotency-Key returns 400", func(t *testing.T) {
		body := `{"sale_id":"` + uuid.New().String() + `","reason":"err","firma_registro":"sig","hash_anterior":"hash"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/voids", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Idempotency-Key") {
			t.Errorf("expected error about missing Idempotency-Key, got: %s", w.Body.String())
		}
	})

	t.Run("Scenario: Invalid JSON body returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/voids", bytes.NewBufferString("not-json"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.New().String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "bad request") {
			t.Errorf("expected 'bad request' error, got: %s", w.Body.String())
		}
	})

	t.Run("Scenario: Invalid Idempotency-Key format returns 400", func(t *testing.T) {
		body := `{"sale_id":"` + uuid.New().String() + `","reason":"err","firma_registro":"sig","hash_anterior":"hash"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/voids", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "not-a-uuid")
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "invalid Idempotency-Key") {
			t.Errorf("expected 'invalid Idempotency-Key' error, got: %s", w.Body.String())
		}
	})
}

func TestSyncSale_RoundtripWithFinancialFields(t *testing.T) {
	firma := "firma-hash"
	hashAnt := "hash-ant"

	sale := adapters.SyncSale{
		ID:                uuid.New().String(),
		NumeroFactura:     "TPV-0042",
		NumeroSecuencia:   42,
		CreatedAt:           "2026-06-20T09:30:00Z",
		Total:               150.00,
		Items:               []adapters.SyncItem{{ItemID: uuid.New().String(), Quantity: 2, UnitPrice: 75.00}},
		FirmaRegistro:       &firma,
		HashAnterior:        &hashAnt,
		DatosEncadenamiento: nil,
		Subtotal:            150.00,
		TaxTotal:            31.50,
		DiscountTotal:       0.00,
		Status:              "COMPLETED",
		VoidReason:          nil,
		VoidedAt:            nil,
		Payments: []adapters.SyncSalePayment{
			{Method: "cash", Amount: 100.00},
			{Method: "card", Amount: 50.00},
		},
	}

	// Marshal
	data, err := json.Marshal(sale)
	if err != nil {
		t.Fatalf("failed to marshal SyncSale: %v", err)
	}

	// Verify JSON contains new fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}
	if raw["subtotal"].(float64) != 150.00 {
		t.Errorf("expected subtotal 150.00 in JSON, got %v", raw["subtotal"])
	}
	if raw["tax_total"].(float64) != 31.50 {
		t.Errorf("expected tax_total 31.50 in JSON, got %v", raw["tax_total"])
	}
	if raw["discount_total"].(float64) != 0.00 {
		t.Errorf("expected discount_total 0.00 in JSON, got %v", raw["discount_total"])
	}
	if raw["status"].(string) != "COMPLETED" {
		t.Errorf("expected status COMPLETED in JSON, got %v", raw["status"])
	}

	// Unmarshal
	var decoded adapters.SyncSale
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal SyncSale: %v", err)
	}

	// Verify fields
	if decoded.Subtotal != 150.00 {
		t.Errorf("expected subtotal 150.00, got %f", decoded.Subtotal)
	}
	if decoded.TaxTotal != 31.50 {
		t.Errorf("expected tax_total 31.50, got %f", decoded.TaxTotal)
	}
	if decoded.DiscountTotal != 0.00 {
		t.Errorf("expected discount_total 0.00, got %f", decoded.DiscountTotal)
	}
	if decoded.Status != "COMPLETED" {
		t.Errorf("expected status COMPLETED, got %s", decoded.Status)
	}
	if decoded.VoidReason != nil {
		t.Errorf("expected void_reason nil, got %v", *decoded.VoidReason)
	}
	if decoded.VoidedAt != nil {
		t.Errorf("expected voided_at nil, got %v", *decoded.VoidedAt)
	}
	if len(decoded.Payments) != 2 {
		t.Fatalf("expected 2 payments, got %d", len(decoded.Payments))
	}
	if decoded.Payments[0].Method != "cash" || decoded.Payments[0].Amount != 100.00 {
		t.Errorf("unexpected payment[0]: %+v", decoded.Payments[0])
	}
	if decoded.Payments[1].Method != "card" || decoded.Payments[1].Amount != 50.00 {
		t.Errorf("unexpected payment[1]: %+v", decoded.Payments[1])
	}

	// Test with void fields populated
	voidReason := "Operator error"
	voidedAt := "2026-06-20T10:00:00Z"
	voidSale := adapters.SyncSale{
		Status:     "VOIDED",
		VoidReason: &voidReason,
		VoidedAt:   &voidedAt,
	}
	data2, _ := json.Marshal(voidSale)
	var decoded2 adapters.SyncSale
	if err := json.Unmarshal(data2, &decoded2); err != nil {
		t.Fatalf("failed to unmarshal void SyncSale: %v", err)
	}

	if decoded2.Status != "VOIDED" {
		t.Errorf("expected status VOIDED, got %s", decoded2.Status)
	}
	if decoded2.VoidReason == nil || *decoded2.VoidReason != "Operator error" {
		t.Errorf("expected void_reason 'Operator error', got %v", decoded2.VoidReason)
	}
	if decoded2.VoidedAt == nil || *decoded2.VoidedAt != voidedAt {
		t.Errorf("expected voided_at %s, got %v", voidedAt, decoded2.VoidedAt)
	}

	// Test that omitempty excludes nil void fields from JSON
	var raw2 map[string]interface{}
	json.Unmarshal(data2, &raw2)
	if _, exists := raw2["void_reason"]; !exists {
		t.Errorf("expected void_reason in JSON when non-nil, but it's missing")
	}
}

func TestSalesSyncController_HandleSyncSales_FIFOReconciliation(t *testing.T) {
	ctx := context.Background()
	db, cleanup := setupControllerTestDB(t)
	defer cleanup()

	// 1. Initialize services & controller
	tracker := idempotency.NewTracker(db, true)
	err := tracker.InitSchema(ctx)
	if err != nil {
		t.Fatalf("failed to init idempotency schema: %v", err)
	}

	ledgerRepo := inventoryadapters.NewSQLStockLedgerRepository(db, true)
	invService := inventorydomain.NewInventoryService(ledgerRepo)

	controller := adapters.NewSalesSyncController(db, true, invService, tracker, &mockInvoiceGenerator{invoiceNumber: "S1-16", seq: 16})
	defaultWarehouse := uuid.New()
	controller.SetDefaultWarehouse(defaultWarehouse)

	// 2. Seed Terminal and Invoicing Series
	terminalID := uuid.New()
	_, err = db.Exec("INSERT INTO terminals (id, name, is_active) VALUES (?, 'TPV-01', 1)", terminalID.String())
	if err != nil {
		t.Fatalf("failed to seed terminal: %v", err)
	}

	seriesID := uuid.New()
	empresaID := uuid.New()
	_, err = db.Exec("INSERT INTO invoicing_series (id, terminal_id, prefix, next_sequence, empresa_id) VALUES (?, ?, 'S1', 10, ?)", seriesID.String(), terminalID.String(), empresaID.String())
	if err != nil {
		t.Fatalf("failed to seed series: %v", err)
	}

	// 3. Synchronize an offline sale of 2 units of item A (stock becomes -2.0)
	idemKey := uuid.New().String()
	saleID := uuid.New().String()
	itemID := uuid.New().String()
	itemUUID, _ := uuid.Parse(itemID)

	firma := "firmar_registro_hash"
	hashAnt := "hash_anterior_val"
	datosEnc := "datos_encadenamiento_val"

	syncReq := adapters.SyncRequest{
		Sales: []adapters.SyncSale{
			{
				ID:               saleID,
				NumeroFactura:    "S1-16",
				NumeroSecuencia:  16,
				CreatedAt:        "2026-06-05T13:00:00Z",
				Total:            150.00,
				Items: []adapters.SyncItem{
					{
						ItemID:    itemID,
						Quantity:  2.0,
						UnitPrice: 75.00,
					},
				},
				FirmaRegistro:       &firma,
				HashAnterior:        &hashAnt,
				DatosEncadenamiento: &datosEnc,
			},
		},
	}

	reqBody, _ := json.Marshal(syncReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/sales", bytes.NewReader(reqBody))
	req.Header.Set("Idempotency-Key", idemKey)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	controller.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	// Verify stock is -2.0
	stock, err := invService.GetAvailableStock(ctx, itemUUID, defaultWarehouse)
	if err != nil {
		t.Fatalf("failed to get available stock: %v", err)
	}
	if stock != -2.0 {
		t.Errorf("expected stock to be -2.0, got %f", stock)
	}

	// 4. Register a stock receipt of 10 units
	receipt, err := invService.RecordReceipt(ctx, itemUUID, defaultWarehouse, 10.0, nil, nil)
	if err != nil {
		t.Fatalf("failed to record receipt: %v", err)
	}

	// 5. Reconcile FIFO and assert the 2 synced items are matched against the stock receipt
	allocs, err := invService.ReconcileFIFO(ctx, itemUUID, defaultWarehouse)
	if err != nil {
		t.Fatalf("failed to run FIFO reconciliation: %v", err)
	}

	if len(allocs) != 1 {
		t.Fatalf("expected 1 allocation, got %d", len(allocs))
	}

	alloc := allocs[0]
	if alloc.ReceiptID != receipt.ID {
		t.Errorf("expected allocated receipt ID to be %s, got %s", receipt.ID, alloc.ReceiptID)
	}
	if alloc.QtyAllocated != 2.0 {
		t.Errorf("expected allocated quantity to be 2.0, got %f", alloc.QtyAllocated)
	}
}


func TestSalesSyncController_HandleSyncEvents(t *testing.T) {
	ctx := context.Background()
	db, cleanup := setupControllerTestDB(t)
	defer cleanup()

	tracker := idempotency.NewTracker(db, true)
	err := tracker.InitSchema(ctx)
	if err != nil {
		t.Fatalf("failed to init idempotency schema: %v", err)
	}

	ledgerRepo := inventoryadapters.NewSQLStockLedgerRepository(db, true)
	invService := inventorydomain.NewInventoryService(ledgerRepo)
	controller := adapters.NewSalesSyncController(db, true, invService, tracker, &mockInvoiceGenerator{invoiceNumber: "S1-16", seq: 16})

	t.Run("Scenario: Successful events sync registers events in DB with status SINCRONIZADO", func(t *testing.T) {
		idemKey := uuid.New().String()
		eventID := uuid.New().String()

		eventsReq := adapters.SyncEventsRequest{
			Events: []adapters.SyncEvent{
				{
					ID:         eventID,
					FechaHora:  "2026-06-05T13:00:00Z",
					TipoEvento: "ALTA_FACTURA",
					Detalles:   "Factura emitida offline: S1-16",
				},
			},
		}

		reqBody, _ := json.Marshal(eventsReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/events", bytes.NewReader(reqBody))
		req.Header.Set("Idempotency-Key", idemKey)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp adapters.SyncResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Status != "success" || resp.SyncedCount != 1 || resp.ProcessedIDs[0] != eventID {
			t.Errorf("unexpected response: %+v", resp)
		}

		// Verify event exists in DB
		var tipo, detalles, estado string
		err = db.QueryRow("SELECT tipo_evento, detalles, estado_sincronizacion FROM registro_sucesos WHERE id = ?", eventID).Scan(&tipo, &detalles, &estado)
		if err != nil {
			t.Fatalf("failed to query event from db: %v", err)
		}
		if tipo != "ALTA_FACTURA" || detalles != "Factura emitida offline: S1-16" || estado != "SINCRONIZADO" {
			t.Errorf("unexpected event data in DB: tipo=%s, detalles=%s, estado=%s", tipo, detalles, estado)
		}

		// Verify idempotency
		reqDup := httptest.NewRequest(http.MethodPost, "/api/v1/sync/events", bytes.NewReader(reqBody))
		reqDup.Header.Set("Idempotency-Key", idemKey)
		reqDup.Header.Set("Content-Type", "application/json")

		wDup := httptest.NewRecorder()
		controller.ServeHTTP(wDup, reqDup)

		if wDup.Code != http.StatusOK {
			t.Fatalf("expected duplicate to return 200 OK, got %d", wDup.Code)
		}

		var respDup adapters.SyncResponse
		if err := json.Unmarshal(wDup.Body.Bytes(), &respDup); err != nil {
			t.Fatalf("failed to unmarshal duplicate response: %v", err)
		}

		if respDup.Status != "success" || respDup.SyncedCount != 1 || respDup.ProcessedIDs[0] != eventID {
			t.Errorf("duplicate response does not match: %+v", respDup)
		}
	})
}
