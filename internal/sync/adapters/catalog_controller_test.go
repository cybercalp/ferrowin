package adapters_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ferrowin/internal/shared/idempotency"
	"ferrowin/internal/sync/adapters"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func setupCatalogTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	queries := []string{
		`CREATE TABLE tipos_iva (
			id TEXT PRIMARY KEY,
			nombre TEXT NOT NULL,
			porcentaje REAL NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			activo INTEGER DEFAULT 1
		)`,
		`CREATE TABLE familias (
			id TEXT PRIMARY KEY,
			nombre TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			activo INTEGER DEFAULT 1
		)`,
		`CREATE TABLE productos (
			id TEXT PRIMARY KEY,
			codigo TEXT UNIQUE NOT NULL,
			nombre TEXT NOT NULL,
			precio_venta REAL NOT NULL,
			familia_id TEXT REFERENCES familias(id) ON DELETE SET NULL,
			tipo_iva_id TEXT REFERENCES tipos_iva(id) ON DELETE RESTRICT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			activo INTEGER DEFAULT 1
		)`,
		`CREATE TABLE entidades (
			id TEXT PRIMARY KEY,
			empresa_id TEXT,
			razon_social TEXT NOT NULL,
			nif TEXT,
			email TEXT,
			telefono TEXT,
			activo INTEGER DEFAULT 1,
			roles TEXT NOT NULL,
			codigo_interno TEXT,
			codigos_alternativos TEXT,
			configuracion_contable TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE cliente_ventas_recientes (
			id_factura TEXT PRIMARY KEY,
			cliente_id TEXT NOT NULL,
			fecha DATETIME NOT NULL,
			numero TEXT NOT NULL,
			total REAL NOT NULL,
			estado TEXT NOT NULL
		)`,
		`CREATE TABLE cliente_estadisticas (
			cliente_id TEXT PRIMARY KEY,
			saldo_pendiente REAL NOT NULL,
			limite_credito REAL NOT NULL,
			articulos_mas_comprados_json TEXT NOT NULL
		)`,
		`CREATE TABLE cliente_facturas_pendientes (
			id_factura TEXT PRIMARY KEY,
			cliente_id TEXT NOT NULL,
			numero_factura TEXT NOT NULL,
			importe_pendiente REAL NOT NULL,
			fecha_emision DATETIME NOT NULL
		)`,
		`CREATE TABLE cobros_recibidos (
			id TEXT PRIMARY KEY,
			cliente_id TEXT NOT NULL,
			factura_id TEXT,
			importe REAL NOT NULL,
			fecha DATETIME NOT NULL,
			metodo_pago TEXT NOT NULL,
			tipo_cobro TEXT NOT NULL,
			idempotency_key TEXT UNIQUE NOT NULL,
			synced_at DATETIME DEFAULT CURRENT_TIMESTAMP
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

func TestCatalogSyncController_HandleCatalogSync(t *testing.T) {
	db, cleanup := setupCatalogTestDB(t)
	defer cleanup()

	tracker := idempotency.NewTracker(db, true)
	err := tracker.InitSchema(context.Background())
	if err != nil {
		t.Fatalf("failed to init idempotency schema: %v", err)
	}

	controller := adapters.NewCatalogSyncController(db, true, tracker)

	// Seed data
	tipoIvaID := uuid.New().String()
	_, err = db.Exec("INSERT INTO tipos_iva (id, nombre, porcentaje, updated_at, activo) VALUES (?, 'General', 21.00, '2026-06-05T12:00:00Z', 1)", tipoIvaID)
	if err != nil {
		t.Fatalf("failed to seed tipos_iva: %v", err)
	}

	familiaID := uuid.New().String()
	_, err = db.Exec("INSERT INTO familias (id, nombre, updated_at, activo) VALUES (?, 'Herramientas', '2026-06-05T12:00:00Z', 1)", familiaID)
	if err != nil {
		t.Fatalf("failed to seed familias: %v", err)
	}

	prodID := uuid.New().String()
	_, err = db.Exec("INSERT INTO productos (id, codigo, nombre, precio_venta, familia_id, tipo_iva_id, updated_at, activo) VALUES (?, 'P001', 'Martillo', 15.50, ?, ?, '2026-06-05T12:00:00Z', 1)", prodID, familiaID, tipoIvaID)
	if err != nil {
		t.Fatalf("failed to seed productos: %v", err)
	}

	cliID := uuid.New().String()
	_, err = db.Exec("INSERT INTO entidades (id, razon_social, nif, email, updated_at, activo, roles) VALUES (?, 'Juan Perez', '12345678A', 'juan@example.com', '2026-06-05T12:00:00Z', 1, 'CLIENTE')", cliID)
	if err != nil {
		t.Fatalf("failed to seed entidades: %v", err)
	}

	// Seed one deactivated product to test delta delete
	inactiveProdID := uuid.New().String()
	_, err = db.Exec("INSERT INTO productos (id, codigo, nombre, precio_venta, familia_id, tipo_iva_id, updated_at, activo) VALUES (?, 'P002', 'Tornillo Viejo', 0.10, ?, ?, '2026-06-05T12:30:00Z', 0)", inactiveProdID, familiaID, tipoIvaID)
	if err != nil {
		t.Fatalf("failed to seed inactive product: %v", err)
	}

	t.Run("Full sync (no since param)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/sync", nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		var resp adapters.CatalogSyncResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(resp.TiposIVA) != 1 || resp.TiposIVA[0].ID != tipoIvaID {
			t.Errorf("unexpected tipos_iva count or id, got: %+v", resp.TiposIVA)
		}
		if len(resp.Familias) != 1 || resp.Familias[0].ID != familiaID {
			t.Errorf("unexpected familias count or id, got: %+v", resp.Familias)
		}
		if len(resp.Productos) != 1 || resp.Productos[0].ID != prodID {
			t.Errorf("unexpected active productos count or id, got: %+v", resp.Productos)
		}
		if len(resp.Clientes) != 1 || resp.Clientes[0].ID != cliID {
			t.Errorf("unexpected clientes count or id, got: %+v", resp.Clientes)
		}
		// In full sync, inactive products are returned under Eliminados
		if len(resp.Eliminados.Productos) != 1 || resp.Eliminados.Productos[0] != inactiveProdID {
			t.Errorf("expected 1 inactive product in Eliminados, got: %+v", resp.Eliminados.Productos)
		}
	})

	t.Run("Delta sync with since param", func(t *testing.T) {
		// since is 2026-06-05T12:15:00Z (after the active records, but before the inactive product)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/sync?since=2026-06-05T12:15:00Z", nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		var resp adapters.CatalogSyncResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Active records are updated_at = 12:00:00, so they shouldn't show up in the delta
		if len(resp.TiposIVA) != 0 {
			t.Errorf("expected 0 tipos_iva, got %d", len(resp.TiposIVA))
		}
		if len(resp.Familias) != 0 {
			t.Errorf("expected 0 familias, got %d", len(resp.Familias))
		}
		if len(resp.Productos) != 0 {
			t.Errorf("expected 0 productos, got %d", len(resp.Productos))
		}
		if len(resp.Clientes) != 0 {
			t.Errorf("expected 0 clientes, got %d", len(resp.Clientes))
		}

		// The inactive product is updated_at = 12:30:00 (which is after since), so it should be in Eliminados
		if len(resp.Eliminados.Productos) != 1 || resp.Eliminados.Productos[0] != inactiveProdID {
			t.Errorf("expected 1 inactive product in Eliminados, got: %+v", resp.Eliminados.Productos)
		}
	})
}

func TestCatalogSyncController_HandleClientDossier(t *testing.T) {
	db, cleanup := setupCatalogTestDB(t)
	defer cleanup()

	tracker := idempotency.NewTracker(db, true)
	err := tracker.InitSchema(context.Background())
	if err != nil {
		t.Fatalf("failed to init idempotency schema: %v", err)
	}

	controller := adapters.NewCatalogSyncController(db, true, tracker)

	cliID := uuid.New().String()
	facturaID := uuid.New().String()

	// Seed client dossier tables
	_, err = db.Exec(`INSERT INTO cliente_ventas_recientes (id_factura, cliente_id, fecha, numero, total, estado)
		VALUES (?, ?, '2026-06-05T10:00:00Z', 'FAC-001', 100.50, 'Emitida')`, facturaID, cliID)
	if err != nil {
		t.Fatalf("failed to seed recent sales: %v", err)
	}

	_, err = db.Exec(`INSERT INTO cliente_estadisticas (cliente_id, saldo_pendiente, limite_credito, articulos_mas_comprados_json)
		VALUES (?, 250.00, 1000.00, '[]')`, cliID)
	if err != nil {
		t.Fatalf("failed to seed stats: %v", err)
	}

	_, err = db.Exec(`INSERT INTO cliente_facturas_pendientes (id_factura, cliente_id, numero_factura, importe_pendiente, fecha_emision)
		VALUES (?, ?, 'FAC-001', 100.50, '2026-06-05T10:00:00Z')`, facturaID, cliID)
	if err != nil {
		t.Fatalf("failed to seed pending invoices: %v", err)
	}

	reqBody := map[string][]string{
		"client_ids": {cliID},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/catalog/clients/dossier", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	controller.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp adapters.ClientDossierResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp.VentasRecientes) != 1 || resp.VentasRecientes[0].IDFactura != facturaID {
		t.Errorf("unexpected VentasRecientes, got: %+v", resp.VentasRecientes)
	}
	if len(resp.Estadisticas) != 1 || resp.Estadisticas[0].SaldoPendiente != 250.00 {
		t.Errorf("unexpected Estadisticas, got: %+v", resp.Estadisticas)
	}
	if len(resp.FacturasPendientes) != 1 || resp.FacturasPendientes[0].ImportePendiente != 100.50 {
		t.Errorf("unexpected FacturasPendientes, got: %+v", resp.FacturasPendientes)
	}
}

func TestCatalogSyncController_HandleSyncPayments(t *testing.T) {
	db, cleanup := setupCatalogTestDB(t)
	defer cleanup()

	tracker := idempotency.NewTracker(db, true)
	err := tracker.InitSchema(context.Background())
	if err != nil {
		t.Fatalf("failed to init idempotency schema: %v", err)
	}

	controller := adapters.NewCatalogSyncController(db, true, tracker)

	cliID := uuid.New().String()
	facturaID := uuid.New().String()
	paymentID := uuid.New().String()
	idemKey := uuid.New().String()

	payment := adapters.SyncPayment{
		ID:             paymentID,
		ClienteID:      cliID,
		FacturaID:      &facturaID,
		Importe:        50.00,
		Fecha:          "2026-06-05T11:00:00Z",
		MetodoPago:     "Efectivo",
		TipoCobro:      "FACTURA",
		IdempotencyKey: idemKey,
	}

	reqPayload := adapters.SyncPaymentsRequest{
		Payments: []adapters.SyncPayment{payment},
	}
	bodyBytes, _ := json.Marshal(reqPayload)

	t.Run("Sync Payments success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/payments", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idemKey)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp adapters.SyncPaymentsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Status != "success" || resp.SyncedCount != 1 || resp.ProcessedIDs[0] != paymentID {
			t.Errorf("unexpected sync payments response: %+v", resp)
		}

		// Check database
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM cobros_recibidos WHERE id = ?", paymentID).Scan(&count)
		if err != nil {
			t.Fatalf("failed to check DB: %v", err)
		}
		if count != 1 {
			t.Error("expected payment record in DB, got 0")
		}
	})

	t.Run("Sync Payments idempotency duplicate key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/payments", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idemKey) // same key
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK (idempotent cached response), got %d", w.Code)
		}

		var resp adapters.SyncPaymentsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Status != "success" || resp.SyncedCount != 1 {
			t.Errorf("unexpected cached response: %+v", resp)
		}
	})
}


