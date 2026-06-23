package adapters_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	billingadapters "ferrowin/internal/billing/adapters"
	billingdomain "ferrowin/internal/billing/domain"
	inventoryadapters "ferrowin/internal/inventory/adapters"
	inventorydomain "ferrowin/internal/inventory/domain"
	purchasesdomain "ferrowin/internal/purchases/domain"
	salesadapters "ferrowin/internal/sales/adapters"
	salesdomain "ferrowin/internal/sales/domain"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type securityStub struct{}

func (s securityStub) HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	return true, nil
}

func setupSalesTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open test SQLite DB: %v", err)
	}

	queries := []string{
		`CREATE TABLE empresas (
			id TEXT PRIMARY KEY,
			razon_social TEXT NOT NULL,
			nif TEXT UNIQUE NOT NULL,
			activa INTEGER DEFAULT 1
		)`,
		`CREATE TABLE warehouses (
			id TEXT PRIMARY KEY,
			empresa_id TEXT NOT NULL REFERENCES empresas(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			active INTEGER DEFAULT 1,
			UNIQUE(empresa_id, name)
		)`,
		`CREATE TABLE familias (
			id TEXT PRIMARY KEY,
			nombre TEXT NOT NULL,
			activo INTEGER DEFAULT 1
		)`,
		`CREATE TABLE tipos_iva (
			id TEXT PRIMARY KEY,
			nombre TEXT NOT NULL,
			porcentaje REAL NOT NULL,
			activo INTEGER DEFAULT 1
		)`,
		`CREATE TABLE productos (
			id TEXT PRIMARY KEY,
			codigo TEXT UNIQUE NOT NULL,
			nombre TEXT NOT NULL,
			precio_venta REAL NOT NULL,
			familia_id TEXT REFERENCES familias(id),
			tipo_iva_id TEXT REFERENCES tipos_iva(id),
			activo INTEGER DEFAULT 1
		)`,
		`CREATE TABLE entidades (
			id TEXT PRIMARY KEY,
			empresa_id TEXT,
			razon_social TEXT NOT NULL,
			nif TEXT UNIQUE,
			email TEXT,
			telefono TEXT,
			activo INTEGER DEFAULT 1,
			roles TEXT NOT NULL
		)`,
		`CREATE TABLE terminals (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			is_active INTEGER DEFAULT 1
		)`,
		`CREATE TABLE invoicing_series (
			id TEXT PRIMARY KEY,
			terminal_id TEXT REFERENCES terminals(id) ON DELETE RESTRICT,
			prefix TEXT UNIQUE NOT NULL,
			next_sequence INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE presupuestos (
			id TEXT PRIMARY KEY,
			empresa_id TEXT NOT NULL REFERENCES empresas(id) ON DELETE CASCADE,
			cliente_id TEXT NOT NULL REFERENCES entidades(id) ON DELETE RESTRICT,
			total REAL NOT NULL,
			estado TEXT NOT NULL CHECK (estado IN ('Borrador', 'Aprobado', 'Convertido', 'Anulado')),
			fecha_validez DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			version INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE presupuesto_lineas (
			id TEXT PRIMARY KEY,
			presupuesto_id TEXT NOT NULL REFERENCES presupuestos(id) ON DELETE CASCADE,
			producto_id TEXT NOT NULL REFERENCES productos(id) ON DELETE RESTRICT,
			cantidad REAL NOT NULL,
			precio_unitario REAL NOT NULL,
			coste_unitario REAL NOT NULL DEFAULT 0.00
		)`,
		`CREATE TABLE pedidos (
			id TEXT PRIMARY KEY,
			empresa_id TEXT NOT NULL REFERENCES empresas(id) ON DELETE CASCADE,
			presupuesto_id TEXT REFERENCES presupuestos(id) ON DELETE SET NULL,
			total REAL NOT NULL,
			estado TEXT NOT NULL CHECK (estado IN ('Borrador', 'Aprobado', 'Convertido', 'Anulado')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			version INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE pedido_lineas (
			id TEXT PRIMARY KEY,
			pedido_id TEXT NOT NULL REFERENCES pedidos(id) ON DELETE CASCADE,
			producto_id TEXT NOT NULL REFERENCES productos(id) ON DELETE RESTRICT,
			cantidad REAL NOT NULL,
			precio_unitario REAL NOT NULL
		)`,
		`CREATE TABLE albaranes (
			id TEXT PRIMARY KEY,
			empresa_id TEXT NOT NULL REFERENCES empresas(id) ON DELETE CASCADE,
			pedido_id TEXT REFERENCES pedidos(id) ON DELETE SET NULL,
			total REAL NOT NULL,
			estado TEXT NOT NULL CHECK (estado IN ('Borrador', 'Procesado', 'Convertido', 'Anulado')),
			almacen_id TEXT NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			version INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE albaran_lineas (
			id TEXT PRIMARY KEY,
			albaran_id TEXT NOT NULL REFERENCES albaranes(id) ON DELETE CASCADE,
			producto_id TEXT NOT NULL REFERENCES productos(id) ON DELETE RESTRICT,
			cantidad REAL NOT NULL,
			precio_unitario REAL NOT NULL
		)`,
		`CREATE TABLE facturas (
			id TEXT PRIMARY KEY,
			empresa_id TEXT NOT NULL REFERENCES empresas(id) ON DELETE CASCADE,
			albaran_id TEXT REFERENCES albaranes(id) ON DELETE SET NULL,
			terminal_id TEXT REFERENCES terminals(id) ON DELETE RESTRICT,
			serie_facturacion_id TEXT REFERENCES invoicing_series(id) ON DELETE RESTRICT,
			numero_factura TEXT UNIQUE NOT NULL,
			numero_secuencia INTEGER NOT NULL,
			total REAL NOT NULL,
			total_rectificado REAL NOT NULL DEFAULT 0.00,
			estado TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			version INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE factura_lineas (
			id TEXT PRIMARY KEY,
			factura_id TEXT NOT NULL REFERENCES facturas(id) ON DELETE CASCADE,
			producto_id TEXT NOT NULL REFERENCES productos(id) ON DELETE RESTRICT,
			cantidad REAL NOT NULL,
			precio_unitario REAL NOT NULL
		)`,
		`CREATE TABLE stock_ledger_movements (
			id TEXT PRIMARY KEY,
			item_id TEXT NOT NULL,
			warehouse_id TEXT NOT NULL REFERENCES warehouses(id),
			quantity REAL NOT NULL,
			movement_type TEXT NOT NULL CHECK (movement_type IN ('RECEIPT', 'WITHDRAWAL', 'SYNC_ADJUSTMENT', 'RETURN')),
			reference_document_type TEXT,
			reference_document_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, q := range queries {
		if _, err = db.Exec(q); err != nil {
			db.Close()
			t.Fatalf("failed to run query %q: %v", q, err)
		}
	}

	cleanup := func() {
		db.Close()
	}
	return db, cleanup
}

func TestSalesController_Integration(t *testing.T) {
	db, cleanup := setupSalesTestDB(t)
	defer cleanup()

	ledgerRepo := inventoryadapters.NewSQLStockLedgerRepository(db, true)
	invService := inventorydomain.NewInventoryService(ledgerRepo)
	billingRepo := billingadapters.NewSQLInvoicingSeriesRepository(db, true)
	billingServ := billingdomain.NewBillingService(billingRepo)
	salesRepo := salesadapters.NewSQLSalesRepository(db, true)
	salesService := salesdomain.NewSalesService(salesRepo, invService, securityStub{}, billingServ)
	controller := salesadapters.NewSalesController(salesService)

	// Seed catalog reference data
	ivaID := uuid.New().String()
	_, _ = db.Exec("INSERT INTO tipos_iva (id, nombre, porcentaje) VALUES (?, 'General', 21.00)", ivaID)

	productID := uuid.New().String()
	_, _ = db.Exec("INSERT INTO productos (id, codigo, nombre, precio_venta, tipo_iva_id) VALUES (?, 'PROD-01', 'Articulo Test', 10.0, ?)", productID, ivaID)

	clientA := uuid.New().String()
	_, _ = db.Exec("INSERT INTO entidades (id, razon_social, nif, roles) VALUES (?, 'Cliente A', '12345678A', 'CLIENTE')", clientA)

	terminalID := uuid.New().String()
	_, _ = db.Exec("INSERT INTO terminals (id, name) VALUES (?, 'T1')", terminalID)

	seriesID := uuid.New().String()
	_, _ = db.Exec("INSERT INTO invoicing_series (id, terminal_id, prefix, next_sequence) VALUES (?, ?, 'S1', 1)", seriesID, terminalID)

	// Setup tenant companies and warehouses
	var companyA, companyB purchasesdomain.Empresa
	var warehouseA, warehouseB purchasesdomain.Warehouse

	// Create Companies
	cABody := `{"razon_social":"Empresa A","nif":"A001"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/purchases/companies", bytes.NewBufferString(cABody)) // Wait, purchases controller handles company creation, but let's insert directly to SQL to decouple tests.
	_, _ = db.Exec("INSERT INTO empresas (id, razon_social, nif) VALUES ('00000000-0000-4000-a000-000000000001', 'Empresa A', 'A001')")
	_, _ = db.Exec("INSERT INTO empresas (id, razon_social, nif) VALUES ('00000000-0000-4000-a000-000000000002', 'Empresa B', 'B002')")
	companyA = purchasesdomain.Empresa{ID: uuid.MustParse("00000000-0000-4000-a000-000000000001")}
	companyB = purchasesdomain.Empresa{ID: uuid.MustParse("00000000-0000-4000-a000-000000000002")}

	_, _ = db.Exec("INSERT INTO warehouses (id, empresa_id, name) VALUES ('00000000-0000-4000-a000-000000000101', '00000000-0000-4000-a000-000000000001', 'Almacen A')")
	_, _ = db.Exec("INSERT INTO warehouses (id, empresa_id, name) VALUES ('00000000-0000-4000-a000-000000000102', '00000000-0000-4000-a000-000000000002', 'Almacen B')")
	warehouseA = purchasesdomain.Warehouse{ID: uuid.MustParse("00000000-0000-4000-a000-000000000101")}
	warehouseB = purchasesdomain.Warehouse{ID: uuid.MustParse("00000000-0000-4000-a000-000000000102")}

	t.Run("Create Quote", func(t *testing.T) {
		reqBody := fmt.Sprintf(`{
			"cliente_id": "%s",
			"fecha_validez": "%s",
			"lineas": [{"producto_id": "%s", "cantidad": 5.0, "precio_unitario": 10.0, "coste_unitario": 6.0}]
		}`, clientA, time.Now().Add(24*time.Hour).Format(time.RFC3339), productID)

		req = httptest.NewRequest(http.MethodPost, "/api/v1/sales/quotes", bytes.NewBufferString(reqBody))
		req.Header.Set("X-Empresa-ID", companyA.ID.String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Conversion Flow & Multi-Tenant Mismatch", func(t *testing.T) {
		// 1. Create a quote for Company A
		reqBody := fmt.Sprintf(`{
			"cliente_id": "%s",
			"fecha_validez": "%s",
			"lineas": [{"producto_id": "%s", "cantidad": 5.0, "precio_unitario": 10.0, "coste_unitario": 6.0}]
		}`, clientA, time.Now().Add(24*time.Hour).Format(time.RFC3339), productID)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/sales/quotes", bytes.NewBufferString(reqBody))
		req.Header.Set("X-Empresa-ID", companyA.ID.String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		var q salesdomain.Presupuesto
		json.Unmarshal(w.Body.Bytes(), &q)

		// 2. Attempt to convert Quote of Company A using Company B's header
		convBody := fmt.Sprintf(`{
			"presupuesto_id": "%s",
			"user_id": "%s",
			"recalculate_prices": false
		}`, q.ID.String(), uuid.New().String())

		req = httptest.NewRequest(http.MethodPost, "/api/v1/sales/quotes/convert", bytes.NewBufferString(convBody))
		req.Header.Set("X-Empresa-ID", companyB.ID.String()) // Company B
		w = httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden on tenant mismatch, got %d", w.Code)
		}

		// 3. Convert Quote of Company A using Company A's header (Authorized)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/sales/quotes/convert", bytes.NewBufferString(convBody))
		req.Header.Set("X-Empresa-ID", companyA.ID.String())
		w = httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("failed to convert quote: %d", w.Code)
		}
		var o salesdomain.Pedido
		json.Unmarshal(w.Body.Bytes(), &o)

		// 4. Convert Order to Delivery Note
		convOrderBody := fmt.Sprintf(`{
			"pedido_id": "%s",
			"almacen_id": "%s"
		}`, o.ID.String(), warehouseA.ID.String())
		req = httptest.NewRequest(http.MethodPost, "/api/v1/sales/orders/convert", bytes.NewBufferString(convOrderBody))
		req.Header.Set("X-Empresa-ID", companyA.ID.String())
		w = httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("failed to convert order: %d", w.Code)
		}
		var dn salesdomain.Albaran
		json.Unmarshal(w.Body.Bytes(), &dn)

		// 5. Attempt to Process Delivery Note -> Should Fail with Conflict due to insufficient stock (available: 0)
		processBody := fmt.Sprintf(`{"albaran_id": "%s"}`, dn.ID.String())
		req = httptest.NewRequest(http.MethodPost, "/api/v1/sales/delivery-notes/process", bytes.NewBufferString(processBody))
		req.Header.Set("X-Empresa-ID", companyA.ID.String())
		w = httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict on insufficient stock, got %d", w.Code)
		}

		// 6. Record stock addition to Warehouse A (avail stock: 10)
		refDocType := "PURCHASE_RECEIPT"
		refDocID := uuid.New()
		_, err := invService.RecordReceipt(context.Background(), uuid.MustParse(productID), warehouseA.ID, 10.0, &refDocType, &refDocID)
		if err != nil {
			t.Fatalf("failed to seed stock: %v", err)
		}

		// Verify stock before delivery note process
		stockA, _ := invService.GetAvailableStock(context.Background(), uuid.MustParse(productID), warehouseA.ID)
		if stockA != 10.0 {
			t.Fatalf("expected stock 10, got %f", stockA)
		}

		// 7. Process Delivery Note -> Should succeed now, withdrawing 5 units from Warehouse A
		req = httptest.NewRequest(http.MethodPost, "/api/v1/sales/delivery-notes/process", bytes.NewBufferString(processBody))
		req.Header.Set("X-Empresa-ID", companyA.ID.String())
		w = httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("failed to process delivery note: %d. Body: %s", w.Code, w.Body.String())
		}

		// 8. Verify stock deduction
		stockA, _ = invService.GetAvailableStock(context.Background(), uuid.MustParse(productID), warehouseA.ID)
		if stockA != 5.0 {
			t.Errorf("expected stock in Warehouse A to decrease to 5.0, got %f", stockA)
		}

		stockB, _ := invService.GetAvailableStock(context.Background(), uuid.MustParse(productID), warehouseB.ID)
		if stockB != 0.0 {
			t.Errorf("expected stock in Warehouse B to remain 0.0, got %f", stockB)
		}

		// 9. Convert processed delivery note to invoice
		convDNBody := fmt.Sprintf(`{
			"albaran_id": "%s",
			"terminal_id": "%s",
			"serie_facturacion_id": "%s"
		}`, dn.ID.String(), terminalID, seriesID)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/sales/delivery-notes/convert", bytes.NewBufferString(convDNBody))
		req.Header.Set("X-Empresa-ID", companyA.ID.String())
		w = httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("failed to convert delivery note to invoice: %d. Body: %s", w.Code, w.Body.String())
		}
		var invoice salesdomain.Factura
		json.Unmarshal(w.Body.Bytes(), &invoice)

		if invoice.NumeroFactura != "S1-1" {
			t.Errorf("expected invoice number S1-1, got %s", invoice.NumeroFactura)
		}
	})
}
