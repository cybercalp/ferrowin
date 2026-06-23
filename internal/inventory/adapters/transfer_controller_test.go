package adapters_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ferrowin/internal/inventory/adapters"
	"ferrowin/internal/inventory/domain"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// fakeWhValidator for controller tests — wraps a map
type controllerWhValidator struct {
	warehouses map[uuid.UUID]*domain.WarehouseView
}

func (v *controllerWhValidator) GetWarehouse(_ context.Context, id uuid.UUID) (*domain.WarehouseView, error) {
	if w, ok := v.warehouses[id]; ok {
		return w, nil
	}
	return nil, errors.New("warehouse not found")
}

func setupTransferControllerTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	queries := []string{
		`CREATE TABLE traspasos_almacen (
			id TEXT PRIMARY KEY,
			empresa_id TEXT NOT NULL,
			origen_id TEXT NOT NULL,
			destino_id TEXT NOT NULL,
			estado TEXT NOT NULL CHECK (estado IN ('Borrador', 'Procesado', 'Cancelado')),
			created_at TEXT NOT NULL,
			processed_at TEXT,
			cancelled_at TEXT
		)`,
		`CREATE TABLE traspaso_almacen_lineas (
			id TEXT PRIMARY KEY,
			traspaso_almacen_id TEXT NOT NULL REFERENCES traspasos_almacen(id) ON DELETE CASCADE,
			producto_id TEXT NOT NULL,
			cantidad REAL NOT NULL CHECK (cantidad > 0)
		)`,
		`CREATE TABLE stock_ledger_movements (
			id TEXT PRIMARY KEY,
			item_id TEXT NOT NULL,
			warehouse_id TEXT NOT NULL,
			quantity REAL NOT NULL,
			movement_type TEXT NOT NULL,
			reference_document_type TEXT,
			reference_document_id TEXT,
			created_at TEXT NOT NULL
		)`,
	}

	for _, q := range queries {
		if _, err = db.Exec(q); err != nil {
			db.Close()
			t.Fatalf("failed to run query %q: %v", q, err)
		}
	}

	cleanup := func() { db.Close() }
	return db, cleanup
}

func TestTransferController_Integration(t *testing.T) {
	db, cleanup := setupTransferControllerTestDB(t)
	defer cleanup()

	// Fix clock for deterministic tests
	now := time.Now()

	transferRepo := adapters.NewSQLTransferRepository(db, true)
	whValidator := &controllerWhValidator{
		warehouses: make(map[uuid.UUID]*domain.WarehouseView),
	}

	transferSvc := domain.NewTransferService(transferRepo, whValidator)
	transferSvc.Now = func() time.Time { now = now.Add(1 * time.Second); return now }

	controller := adapters.NewTransferController(transferSvc)

	// Seed data
	empresaID := uuid.New()
	origenID := uuid.New()
	destinoID := uuid.New()
	productoID := uuid.New()

	whValidator.warehouses[origenID] = &domain.WarehouseView{ID: origenID, EmpresaID: empresaID}
	whValidator.warehouses[destinoID] = &domain.WarehouseView{ID: destinoID, EmpresaID: empresaID}

	t.Run("HandleCreate — 201 Created", func(t *testing.T) {
		reqBody := fmt.Sprintf(`{
			"origen_id": "%s",
			"destino_id": "%s"
		}`, origenID.String(), destinoID.String())

		req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/transfers", bytes.NewBufferString(reqBody))
		req.Header.Set("X-Empresa-ID", empresaID.String())
		w := httptest.NewRecorder()

		controller.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp domain.TraspasoAlmacen
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Estado != domain.TraspasoBorrador {
			t.Errorf("expected estado Borrador, got %s", resp.Estado)
		}
		if resp.OrigenID != origenID {
			t.Errorf("expected origen %s, got %s", origenID, resp.OrigenID)
		}
		if resp.DestinoID != destinoID {
			t.Errorf("expected destino %s, got %s", destinoID, resp.DestinoID)
		}
	})

	t.Run("HandleCreate — 400 Same warehouse", func(t *testing.T) {
		reqBody := fmt.Sprintf(`{
			"origen_id": "%s",
			"destino_id": "%s"
		}`, origenID.String(), origenID.String())

		req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/transfers", bytes.NewBufferString(reqBody))
		req.Header.Set("X-Empresa-ID", empresaID.String())
		w := httptest.NewRecorder()

		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("HandleCreate — 401 Missing X-Empresa-ID", func(t *testing.T) {
		reqBody := fmt.Sprintf(`{
			"origen_id": "%s",
			"destino_id": "%s"
		}`, origenID.String(), destinoID.String())

		req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/transfers", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		controller.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("HandleCreate — 400 Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/transfers", bytes.NewBufferString(`{bad json`))
		req.Header.Set("X-Empresa-ID", empresaID.String())
		w := httptest.NewRecorder()

		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	// Create a transfer for subsequent tests
	var createdTransfer domain.TraspasoAlmacen
	func() {
		reqBody := fmt.Sprintf(`{
			"origen_id": "%s",
			"destino_id": "%s"
		}`, origenID.String(), destinoID.String())
		req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/transfers", bytes.NewBufferString(reqBody))
		req.Header.Set("X-Empresa-ID", empresaID.String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("failed to create transfer for sub-tests: %d. Body: %s", w.Code, w.Body.String())
		}
		json.Unmarshal(w.Body.Bytes(), &createdTransfer)
	}()

	t.Run("HandleAddLine — 200 OK", func(t *testing.T) {
		reqBody := fmt.Sprintf(`{
			"producto_id": "%s",
			"cantidad": 10.0
		}`, productoID.String())

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/inventory/transfers/%s/lines", createdTransfer.ID.String()),
			bytes.NewBufferString(reqBody))
		req.Header.Set("X-Empresa-ID", empresaID.String())
		w := httptest.NewRecorder()

		controller.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp domain.TraspasoAlmacen
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if len(resp.Lineas) != 1 {
			t.Fatalf("expected 1 line, got %d", len(resp.Lineas))
		}
		if resp.Lineas[0].ProductoID != productoID {
			t.Errorf("expected producto %s, got %s", productoID, resp.Lineas[0].ProductoID)
		}
		if resp.Lineas[0].Cantidad != 10.0 {
			t.Errorf("expected cantidad 10.0, got %f", resp.Lineas[0].Cantidad)
		}
	})

	t.Run("HandleAddLine — 400 Invalid UUID", func(t *testing.T) {
		reqBody := `{"producto_id": "bad-uuid", "cantidad": 5.0}`
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/inventory/transfers/%s/lines", createdTransfer.ID.String()),
			bytes.NewBufferString(reqBody))
		req.Header.Set("X-Empresa-ID", empresaID.String())
		w := httptest.NewRecorder()

		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("HandleAddLine — 400 Invalid cantidad (zero)", func(t *testing.T) {
		reqBody := fmt.Sprintf(`{"producto_id": "%s", "cantidad": 0}`, productoID.String())
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/inventory/transfers/%s/lines", createdTransfer.ID.String()),
			bytes.NewBufferString(reqBody))
		req.Header.Set("X-Empresa-ID", empresaID.String())
		w := httptest.NewRecorder()

		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("HandleRemoveLine — 200 OK", func(t *testing.T) {
		// First add then remove
		reqBody := fmt.Sprintf(`{"producto_id": "%s", "cantidad": 7.0}`, uuid.New().String())
		reqAdd := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/inventory/transfers/%s/lines", createdTransfer.ID.String()),
			bytes.NewBufferString(reqBody))
		reqAdd.Header.Set("X-Empresa-ID", empresaID.String())
		wAdd := httptest.NewRecorder()
		controller.ServeHTTP(wAdd, reqAdd)
		if wAdd.Code != http.StatusOK {
			t.Fatalf("failed to add line for removal: %d", wAdd.Code)
		}

		var afterAdd domain.TraspasoAlmacen
		json.Unmarshal(wAdd.Body.Bytes(), &afterAdd)
		if len(afterAdd.Lineas) == 0 {
			t.Fatal("no lines after add")
		}
		lineID := afterAdd.Lineas[len(afterAdd.Lineas)-1].ID

		reqDel := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/inventory/transfers/%s/lines/%s", createdTransfer.ID.String(), lineID.String()), nil)
		reqDel.Header.Set("X-Empresa-ID", empresaID.String())
		wDel := httptest.NewRecorder()
		controller.ServeHTTP(wDel, reqDel)

		if wDel.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. Body: %s", wDel.Code, wDel.Body.String())
		}

		var afterDel domain.TraspasoAlmacen
		json.Unmarshal(wDel.Body.Bytes(), &afterDel)
		for _, l := range afterDel.Lineas {
			if l.ID == lineID {
				t.Errorf("line %s should have been removed", lineID)
			}
		}
	})

	t.Run("HandleProcess — 200 OK", func(t *testing.T) {
		// Create a fresh transfer for processing
		reqBody := fmt.Sprintf(`{"origen_id": "%s", "destino_id": "%s"}`, origenID.String(), destinoID.String())
		reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/transfers", bytes.NewBufferString(reqBody))
		reqCreate.Header.Set("X-Empresa-ID", empresaID.String())
		wCreate := httptest.NewRecorder()
		controller.ServeHTTP(wCreate, reqCreate)
		if wCreate.Code != http.StatusCreated {
			t.Fatalf("failed to create: %d", wCreate.Code)
		}

		var fresh domain.TraspasoAlmacen
		json.Unmarshal(wCreate.Body.Bytes(), &fresh)

		// Add a line
		lineBody := fmt.Sprintf(`{"producto_id": "%s", "cantidad": 5.0}`, productoID.String())
		reqLine := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/inventory/transfers/%s/lines", fresh.ID.String()),
			bytes.NewBufferString(lineBody))
		reqLine.Header.Set("X-Empresa-ID", empresaID.String())
		wLine := httptest.NewRecorder()
		controller.ServeHTTP(wLine, reqLine)
		if wLine.Code != http.StatusOK {
			t.Fatalf("failed to add line: %d", wLine.Code)
		}

		// Process
		reqProcess := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/inventory/transfers/%s/process", fresh.ID.String()), nil)
		reqProcess.Header.Set("X-Empresa-ID", empresaID.String())
		wProcess := httptest.NewRecorder()
		controller.ServeHTTP(wProcess, reqProcess)

		if wProcess.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. Body: %s", wProcess.Code, wProcess.Body.String())
		}

		var processed domain.TraspasoAlmacen
		json.Unmarshal(wProcess.Body.Bytes(), &processed)
		if processed.Estado != domain.TraspasoProcesado {
			t.Errorf("expected estado Procesado, got %s", processed.Estado)
		}
		if processed.ProcessedAt == nil {
			t.Error("expected ProcessedAt to be set")
		}
	})

	t.Run("HandleProcess — 409 Already processed", func(t *testing.T) {
		// Create, add line, process, try again
		reqBody := fmt.Sprintf(`{"origen_id": "%s", "destino_id": "%s"}`, origenID.String(), destinoID.String())
		reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/transfers", bytes.NewBufferString(reqBody))
		reqCreate.Header.Set("X-Empresa-ID", empresaID.String())
		wCreate := httptest.NewRecorder()
		controller.ServeHTTP(wCreate, reqCreate)

		var fresh domain.TraspasoAlmacen
		json.Unmarshal(wCreate.Body.Bytes(), &fresh)

		lineBody := fmt.Sprintf(`{"producto_id": "%s", "cantidad": 3.0}`, productoID.String())
		reqLine := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/inventory/transfers/%s/lines", fresh.ID.String()),
			bytes.NewBufferString(lineBody))
		reqLine.Header.Set("X-Empresa-ID", empresaID.String())
		wLine := httptest.NewRecorder()
		controller.ServeHTTP(wLine, reqLine)

		reqProcess := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/inventory/transfers/%s/process", fresh.ID.String()), nil)
		reqProcess.Header.Set("X-Empresa-ID", empresaID.String())
		wProcess1 := httptest.NewRecorder()
		controller.ServeHTTP(wProcess1, reqProcess)

		// Second process should fail
		wProcess2 := httptest.NewRecorder()
		controller.ServeHTTP(wProcess2, reqProcess)
		if wProcess2.Code != http.StatusConflict {
			t.Errorf("expected 409, got %d. Body: %s", wProcess2.Code, wProcess2.Body.String())
		}
	})

	t.Run("HandleGetByID — 200 OK", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/inventory/transfers/%s", createdTransfer.ID.String()), nil)
		req.Header.Set("X-Empresa-ID", empresaID.String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp domain.TraspasoAlmacen
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.ID != createdTransfer.ID {
			t.Errorf("expected ID %s, got %s", createdTransfer.ID, resp.ID)
		}
	})

	t.Run("HandleGetByID — 404 Not Found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/inventory/transfers/%s", uuid.New().String()), nil)
		req.Header.Set("X-Empresa-ID", empresaID.String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Code)
		}
	})

	t.Run("HandleGetByID — 400 Bad UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/transfers/bad-uuid", nil)
		req.Header.Set("X-Empresa-ID", empresaID.String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("HandleList — 200 with filters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/inventory/transfers?estado=Borrador&origen_id=%s", origenID.String()), nil)
		req.Header.Set("X-Empresa-ID", empresaID.String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data     []*domain.TraspasoAlmacen `json:"data"`
			Total    int                       `json:"total"`
			Page     int                       `json:"page"`
			PageSize int                       `json:"page_size"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Total < 1 {
			t.Errorf("expected at least 1 transfer, got total %d", resp.Total)
		}
		if resp.Page < 1 {
			t.Errorf("expected page >= 1, got %d", resp.Page)
		}
		if resp.PageSize < 1 {
			t.Errorf("expected page_size >= 1, got %d", resp.PageSize)
		}
	})

	t.Run("HandleList — 401 Missing X-Empresa-ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/transfers", nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("MethodNotAllowed — 405 on PUT", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/inventory/transfers", nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Code)
		}
	})

	t.Run("404 on unknown path prefix", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/transfers/some-id/nonexistent-action", nil)
		req.Header.Set("X-Empresa-ID", empresaID.String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Code)
		}
	})
}
