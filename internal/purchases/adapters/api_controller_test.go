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

	inventoryadapters "ferrowin/internal/inventory/adapters"
	inventorydomain "ferrowin/internal/inventory/domain"
	purchasesadapters "ferrowin/internal/purchases/adapters"
	purchasesdomain "ferrowin/internal/purchases/domain"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
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
		`CREATE TABLE entidades (
			id TEXT PRIMARY KEY,
			empresa_id TEXT NOT NULL REFERENCES empresas(id) ON DELETE CASCADE,
			razon_social TEXT NOT NULL,
			nif TEXT NOT NULL,
			email TEXT,
			telefono TEXT,
			activo INTEGER DEFAULT 1,
			roles TEXT NOT NULL,
			UNIQUE(empresa_id, nif)
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
		`CREATE TABLE pedidos_compra (
			id TEXT PRIMARY KEY,
			empresa_id TEXT NOT NULL REFERENCES empresas(id) ON DELETE CASCADE,
			proveedor_id TEXT NOT NULL REFERENCES entidades(id) ON DELETE RESTRICT,
			numero_pedido TEXT NOT NULL,
			fecha DATETIME DEFAULT CURRENT_TIMESTAMP,
			estado TEXT NOT NULL CHECK (estado IN ('Borrador', 'Aprobado', 'Recibido', 'Parcial', 'Cancelado')),
			total REAL NOT NULL DEFAULT 0.00,
			version INTEGER NOT NULL DEFAULT 1,
			UNIQUE(empresa_id, numero_pedido)
		)`,
		`CREATE TABLE pedido_compra_lineas (
			id TEXT PRIMARY KEY,
			pedido_compra_id TEXT NOT NULL REFERENCES pedidos_compra(id) ON DELETE CASCADE,
			producto_id TEXT NOT NULL REFERENCES productos(id) ON DELETE RESTRICT,
			cantidad REAL NOT NULL,
			precio_unitario REAL NOT NULL,
			recibido REAL NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE recepciones_compra (
			id TEXT PRIMARY KEY,
			empresa_id TEXT NOT NULL REFERENCES empresas(id) ON DELETE CASCADE,
			pedido_compra_id TEXT REFERENCES pedidos_compra(id) ON DELETE SET NULL,
			proveedor_id TEXT NOT NULL REFERENCES entidades(id) ON DELETE RESTRICT,
			numero_albaran TEXT NOT NULL,
			fecha DATETIME DEFAULT CURRENT_TIMESTAMP,
			estado TEXT NOT NULL CHECK (estado IN ('Borrador', 'Procesado', 'Cancelado')),
			warehouse_id TEXT NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
			version INTEGER NOT NULL DEFAULT 1,
			UNIQUE(empresa_id, numero_albaran)
		)`,
		`CREATE TABLE recepcion_compra_lineas (
			id TEXT PRIMARY KEY,
			recepcion_compra_id TEXT NOT NULL REFERENCES recepciones_compra(id) ON DELETE CASCADE,
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

func TestPurchaseController_Integration(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ledgerRepo := inventoryadapters.NewSQLStockLedgerRepository(db, true)
	invService := inventorydomain.NewInventoryService(ledgerRepo)
	purchaseRepo := purchasesadapters.NewSQLPurchaseRepository(db, true)
	purchaseService := purchasesdomain.NewPurchaseService(purchaseRepo, invService)
	controller := purchasesadapters.NewPurchaseController(purchaseService)

	// Seed product reference data
	ivaID := uuid.New().String()
	_, err := db.Exec("INSERT INTO tipos_iva (id, nombre, porcentaje) VALUES (?, 'General', 21.00)", ivaID)
	if err != nil {
		t.Fatalf("failed to seed IVA: %v", err)
	}

	productID := uuid.New().String()
	_, err = db.Exec("INSERT INTO productos (id, codigo, nombre, precio_venta, tipo_iva_id) VALUES (?, 'PROD-01', 'Tornillo M6', 1.50, ?)", productID, ivaID)
	if err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	var companyA, companyB purchasesdomain.Empresa
	var warehouseA, warehouseB purchasesdomain.Warehouse
	var supplierA, supplierB purchasesdomain.Proveedor

	t.Run("Create Companies", func(t *testing.T) {
		// Company A
		reqBody := `{"razon_social":"Empresa A S.L.","nif":"A11111111"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/purchases/companies", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d. Body: %s", w.Code, w.Body.String())
		}
		json.Unmarshal(w.Body.Bytes(), &companyA)

		// Company B
		reqBody = `{"razon_social":"Empresa B S.L.","nif":"B22222222"}`
		req = httptest.NewRequest(http.MethodPost, "/api/v1/purchases/companies", bytes.NewBufferString(reqBody))
		w = httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d", w.Code)
		}
		json.Unmarshal(w.Body.Bytes(), &companyB)
	})

	t.Run("Create Warehouses", func(t *testing.T) {
		// Warehouse A (Company A)
		reqBody := `{"name":"Almacen Norte"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/purchases/warehouses", bytes.NewBufferString(reqBody))
		req.Header.Set("X-Empresa-ID", companyA.ID.String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d", w.Code)
		}
		json.Unmarshal(w.Body.Bytes(), &warehouseA)

		// Warehouse B (Company B)
		reqBody = `{"name":"Almacen Sur"}`
		req = httptest.NewRequest(http.MethodPost, "/api/v1/purchases/warehouses", bytes.NewBufferString(reqBody))
		req.Header.Set("X-Empresa-ID", companyB.ID.String())
		w = httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d", w.Code)
		}
		json.Unmarshal(w.Body.Bytes(), &warehouseB)
	})

	t.Run("Create Suppliers & Verify Isolation", func(t *testing.T) {
		// Supplier A
		reqBody := `{"razon_social":"Proveedor A","cif":"A99999999","email":"a@prov.com"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/purchases/suppliers", bytes.NewBufferString(reqBody))
		req.Header.Set("X-Empresa-ID", companyA.ID.String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d", w.Code)
		}
		json.Unmarshal(w.Body.Bytes(), &supplierA)

		// Supplier B
		reqBody = `{"razon_social":"Proveedor B","cif":"B99999999","email":"b@prov.com"}`
		req = httptest.NewRequest(http.MethodPost, "/api/v1/purchases/suppliers", bytes.NewBufferString(reqBody))
		req.Header.Set("X-Empresa-ID", companyB.ID.String())
		w = httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d", w.Code)
		}
		json.Unmarshal(w.Body.Bytes(), &supplierB)

		// Get suppliers for Company A
		req = httptest.NewRequest(http.MethodGet, "/api/v1/purchases/suppliers", nil)
		req.Header.Set("X-Empresa-ID", companyA.ID.String())
		w = httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		var listA []purchasesdomain.Proveedor
		json.Unmarshal(w.Body.Bytes(), &listA)
		if len(listA) != 1 || listA[0].ID != supplierA.ID {
			t.Errorf("expected only Supplier A, got %d items", len(listA))
		}
	})

	t.Run("Create Purchase Order with Tenant Mismatch", func(t *testing.T) {
		// Create order for Company A but using Supplier B (Company B)
		reqBody := fmt.Sprintf(`{
			"proveedor_id": "%s",
			"numero_pedido": "PO-001",
			"lineas": [{"producto_id": "%s", "cantidad": 10.0, "precio_unitario": 5.0}]
		}`, supplierB.ID.String(), productID)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/purchases/orders", bytes.NewBufferString(reqBody))
		req.Header.Set("X-Empresa-ID", companyA.ID.String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden on tenant mismatch, got %d", w.Code)
		}
	})

	t.Run("Purchase Cycle & Stock Addition", func(t *testing.T) {
		// 1. Create Purchase Order for Company A / Supplier A
		reqBody := fmt.Sprintf(`{
			"proveedor_id": "%s",
			"numero_pedido": "PO-A01",
			"lineas": [{"producto_id": "%s", "cantidad": 50.0, "precio_unitario": 2.50}]
		}`, supplierA.ID.String(), productID)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/purchases/orders", bytes.NewBufferString(reqBody))
		req.Header.Set("X-Empresa-ID", companyA.ID.String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("failed to create order: %d. Body: %s", w.Code, w.Body.String())
		}
		var po purchasesdomain.PedidoCompra
		json.Unmarshal(w.Body.Bytes(), &po)

		// 2. Approve Purchase Order
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/purchases/orders/approve/%s", po.ID.String()), nil)
		req.Header.Set("X-Empresa-ID", companyA.ID.String())
		w = httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("failed to approve order: %d", w.Code)
		}

		// Verify stock before receipt is 0
		stockA, err := invService.GetAvailableStock(context.Background(), uuid.MustParse(productID), warehouseA.ID)
		if err != nil || stockA != 0 {
			t.Fatalf("stock in warehouse A should be 0, got %f", stockA)
		}

		// 3. Create Purchase Receipt referencing the order
		reqBody = fmt.Sprintf(`{
			"proveedor_id": "%s",
			"pedido_compra_id": "%s",
			"numero_albaran": "ALB-01",
			"warehouse_id": "%s",
			"lineas": [{"producto_id": "%s", "cantidad": 50.0, "precio_unitario": 2.50}]
		}`, supplierA.ID.String(), po.ID.String(), warehouseA.ID.String(), productID)

		req = httptest.NewRequest(http.MethodPost, "/api/v1/purchases/receipts", bytes.NewBufferString(reqBody))
		req.Header.Set("X-Empresa-ID", companyA.ID.String())
		w = httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("failed to create receipt: %d. Body: %s", w.Code, w.Body.String())
		}
		var rc purchasesdomain.RecepcionCompra
		json.Unmarshal(w.Body.Bytes(), &rc)

		// 4. Process Receipt (which records stock movements)
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/purchases/receipts/process/%s", rc.ID.String()), nil)
		req.Header.Set("X-Empresa-ID", companyA.ID.String())
		w = httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("failed to process receipt: %d", w.Code)
		}

		// 5. Verify stock additions
		stockA, err = invService.GetAvailableStock(context.Background(), uuid.MustParse(productID), warehouseA.ID)
		if err != nil || stockA != 50.0 {
			t.Errorf("expected stock in Warehouse A to be 50.0, got %f", stockA)
		}

		stockB, err := invService.GetAvailableStock(context.Background(), uuid.MustParse(productID), warehouseB.ID)
		if err != nil || stockB != 0.0 {
			t.Errorf("expected stock in Warehouse B to be 0.0, got %f", stockB)
		}

		// Verify Purchase Order status is transitioned to "Recibido"
		updatedPO, err := purchaseRepo.GetPurchaseOrder(context.Background(), po.ID)
		if err != nil || updatedPO.Estado != "Recibido" {
			t.Errorf("expected PO state to be 'Recibido', got %s", updatedPO.Estado)
		}
	})
}
