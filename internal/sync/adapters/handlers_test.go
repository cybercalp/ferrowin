package adapters_test

import (
	"bytes"
	"context"
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
)

// setupHandlersTestDB creates the necessary schema for closure and stock tests.
// It uses the same tables as setupControllerTestDB with added box_closures.
func setupHandlersTestDB(t *testing.T) (*idempotency.Tracker, *inventorydomain.InventoryService, *adapters.SalesSyncController, func()) {
	t.Helper()
	db, cleanup := setupControllerTestDB(t)

	ctx := context.Background()

	// Create additional table for closures
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS box_closures (
		id TEXT PRIMARY KEY,
		opened_at DATETIME NOT NULL,
		closed_at DATETIME NOT NULL,
		cash_reported REAL NOT NULL,
		card_reported REAL NOT NULL,
		sales_total REAL NOT NULL,
		synced_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create box_closures table: %v", err)
	}

	tracker := idempotency.NewTracker(db, true)
	err = tracker.InitSchema(ctx)
	if err != nil {
		db.Close()
		t.Fatalf("failed to init idempotency schema: %v", err)
	}

	ledgerRepo := inventoryadapters.NewSQLStockLedgerRepository(db, true)
	invService := inventorydomain.NewInventoryService(ledgerRepo)

	controller := adapters.NewSalesSyncController(db, true, invService, tracker)
	controller.SetDefaultWarehouse(uuid.New())

	return tracker, invService, controller, cleanup
}

func TestSalesSyncController_HandleSyncClosures(t *testing.T) {
	_, _, controller, cleanup := setupHandlersTestDB(t)
	defer cleanup()

	validID := uuid.New().String()

	t.Run("Scenario: Missing Idempotency-Key header", func(t *testing.T) {
		body := `{"id":"` + validID + `","opened_at":"2026-06-05T08:00:00Z","closed_at":"2026-06-05T18:00:00Z","cash_reported":1200.50,"card_reported":800.25,"sales_total":2000.75}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/closures", bytes.NewBufferString(body))
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

	t.Run("Scenario: Invalid Idempotency-Key format", func(t *testing.T) {
		body := `{"id":"` + validID + `","opened_at":"2026-06-05T08:00:00Z","closed_at":"2026-06-05T18:00:00Z","cash_reported":1200.50,"card_reported":800.25,"sales_total":2000.75}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/closures", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "not-a-uuid")
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "invalid Idempotency-Key") {
			t.Errorf("expected error about invalid Idempotency-Key, got: %s", w.Body.String())
		}
	})

	t.Run("Scenario: Invalid JSON payload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/closures", bytes.NewBufferString("not-json"))
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

	t.Run("Scenario: Invalid closure ID (not a UUID)", func(t *testing.T) {
		body := `{"id":"not-a-uuid","opened_at":"2026-06-05T08:00:00Z","closed_at":"2026-06-05T18:00:00Z","cash_reported":100,"card_reported":50,"sales_total":150}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/closures", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.New().String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "invalid closure ID") {
			t.Errorf("expected 'invalid closure ID' error, got: %s", w.Body.String())
		}
	})

	t.Run("Scenario: Successful closure sync", func(t *testing.T) {
		closureID := uuid.New().String()
		idemKey := uuid.New().String()

		body := `{
			"id":"` + closureID + `",
			"opened_at":"2026-06-05T08:00:00Z",
			"closed_at":"2026-06-05T18:00:00Z",
			"cash_reported":1200.50,
			"card_reported":800.25,
			"sales_total":2000.75
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/closures", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idemKey)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Status string `json:"status"`
			ID     string `json:"id"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Status != "success" {
			t.Errorf("expected status 'success', got %q", resp.Status)
		}
		if resp.ID != closureID {
			t.Errorf("expected closure ID %q, got %q", closureID, resp.ID)
		}
	})

	t.Run("Scenario: Duplicate idempotency key returns cached response", func(t *testing.T) {
		idemKey := uuid.New().String()
		closureID := uuid.New().String()

		body := `{
			"id":"` + closureID + `",
			"opened_at":"2026-06-05T08:00:00Z",
			"closed_at":"2026-06-05T18:00:00Z",
			"cash_reported":500.00,
			"card_reported":300.00,
			"sales_total":800.00
		}`

		// First request
		req1 := httptest.NewRequest(http.MethodPost, "/api/v1/sync/closures", bytes.NewBufferString(body))
		req1.Header.Set("Content-Type", "application/json")
		req1.Header.Set("Idempotency-Key", idemKey)
		w1 := httptest.NewRecorder()
		controller.ServeHTTP(w1, req1)
		if w1.Code != http.StatusOK {
			t.Fatalf("expected first request 200 OK, got %d: %s", w1.Code, w1.Body.String())
		}

		// Duplicate request with same key
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/sync/closures", bytes.NewBufferString(body))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Idempotency-Key", idemKey)
		w2 := httptest.NewRecorder()
		controller.ServeHTTP(w2, req2)

		if w2.Code != http.StatusOK {
			t.Errorf("expected duplicate 200 OK, got %d", w2.Code)
		}

		// Both responses should be identical
		if w1.Body.String() != w2.Body.String() {
			t.Errorf("duplicate response body differs from original:\noriginal: %s\nduplicate: %s", w1.Body.String(), w2.Body.String())
		}
	})

	t.Run("Scenario: Missing fields in payload (opened_at parse fallback)", func(t *testing.T) {
		closureID := uuid.New().String()
		idemKey := uuid.New().String()

		// Provide an invalid opened_at — the handler will fallback to time.Now()
		body := `{
			"id":"` + closureID + `",
			"opened_at":"invalid-date",
			"closed_at":"2026-06-05T18:00:00Z",
			"cash_reported":100.00,
			"card_reported":50.00,
			"sales_total":150.00
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/closures", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idemKey)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK even with invalid date (fallback), got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Status string `json:"status"`
			ID     string `json:"id"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Status != "success" {
			t.Errorf("expected status 'success', got %q", resp.Status)
		}
	})
}

func TestSalesSyncController_HandleGetStock(t *testing.T) {
	_, invService, controller, cleanup := setupHandlersTestDB(t)
	defer cleanup()

	itemID := uuid.New()
	defaultWarehouse := uuid.New()
	controller.SetDefaultWarehouse(defaultWarehouse)

	// Register a receipt so stock is positive
	ctx := context.Background()
	_, err := invService.RecordReceipt(ctx, itemID, defaultWarehouse, 100.0, nil, nil)
	if err != nil {
		t.Fatalf("failed to seed stock receipt: %v", err)
	}

	t.Run("Scenario: Successful stock retrieval", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/stock/"+itemID.String(), nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			ItemID string  `json:"item_id"`
			Stock  float64 `json:"stock"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.ItemID != itemID.String() {
			t.Errorf("expected item_id %q, got %q", itemID.String(), resp.ItemID)
		}
		if resp.Stock != 100.0 {
			t.Errorf("expected stock 100.0, got %f", resp.Stock)
		}
	})

	t.Run("Scenario: Zero stock for unknown item", func(t *testing.T) {
		unknownID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/stock/"+unknownID.String(), nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			ItemID string  `json:"item_id"`
			Stock  float64 `json:"stock"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Stock != 0.0 {
			t.Errorf("expected stock 0.0 for unknown item, got %f", resp.Stock)
		}
	})

	t.Run("Scenario: Trailing slash yields empty item ID", func(t *testing.T) {
		// /api/v1/inventory/stock/ splits into ["", "api", "v1", "inventory", "stock", ""]
		// parts[5] is "" which fails uuid.Parse -> "invalid item ID format"
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/stock/", nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "invalid item ID format") {
			t.Errorf("expected 'invalid item ID format' error, got: %s", w.Body.String())
		}
	})



	t.Run("Scenario: Invalid item ID format (not a UUID)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/stock/not-a-uuid", nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for invalid item ID, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "invalid item ID format") {
			t.Errorf("expected 'invalid item ID format' error, got: %s", w.Body.String())
		}
	})
}

func TestSalesSyncController_HandleSyncSales_InvalidPayload(t *testing.T) {
	_, _, controller, cleanup := setupHandlersTestDB(t)
	defer cleanup()

	t.Run("Scenario: Invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/sales", bytes.NewBufferString("not-json"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.New().String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for invalid JSON, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "bad request") {
			t.Errorf("expected 'bad request' error, got: %s", w.Body.String())
		}
	})

	t.Run("Scenario: Empty sales array succeeds (nothing to sync)", func(t *testing.T) {
		body := `{"sales": []}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/sales", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.New().String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK for empty sales, got %d: %s", w.Code, w.Body.String())
		}

		var resp adapters.SyncResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Status != "success" {
			t.Errorf("expected status 'success', got %q", resp.Status)
		}
		if resp.SyncedCount != 0 {
			t.Errorf("expected synced_count 0, got %d", resp.SyncedCount)
		}
	})

	t.Run("Scenario: Invalid sale ID (not a UUID)", func(t *testing.T) {
		body := `{"sales": [{"id":"not-a-uuid","invoice_number":"S1-16","sequence_number":1,"created_at":"2026-06-05T13:00:00Z","total":100.0,"items":[]}]}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/sales", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.New().String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for invalid sale ID, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "invalid sale ID") {
			t.Errorf("expected 'invalid sale ID' error, got: %s", w.Body.String())
		}
	})

	t.Run("Scenario: Invalid invoice number format", func(t *testing.T) {
		saleID := uuid.New().String()
		body := `{"sales": [{"id":"` + saleID + `","invoice_number":"INVALID","sequence_number":1,"created_at":"2026-06-05T13:00:00Z","total":100.0,"items":[]}]}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/sales", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.New().String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for invalid invoice number, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "invalid invoice number format") {
			t.Errorf("expected 'invalid invoice number format' error, got: %s", w.Body.String())
		}
	})

	t.Run("Scenario: Invoicing series not found for prefix", func(t *testing.T) {
		saleID := uuid.New().String()
		// Prefix "XX" has no invoicing series seeded
		body := `{"sales": [{"id":"` + saleID + `","invoice_number":"XX-1","sequence_number":1,"created_at":"2026-06-05T13:00:00Z","total":100.0,"items":[]}]}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/sales", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.New().String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for unknown series prefix, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "invoicing series not found") {
			t.Errorf("expected 'invoicing series not found' error, got: %s", w.Body.String())
		}
	})
}

func TestSalesSyncController_HandleSyncEvents_InvalidPayload(t *testing.T) {
	_, _, controller, cleanup := setupHandlersTestDB(t)
	defer cleanup()

	t.Run("Scenario: Invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/events", bytes.NewBufferString("not-json"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.New().String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for invalid JSON, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "bad request") {
			t.Errorf("expected 'bad request' error, got: %s", w.Body.String())
		}
	})

	t.Run("Scenario: Invalid event ID (not a UUID)", func(t *testing.T) {
		body := `{"events": [{"id":"not-a-uuid","fecha_hora":"2026-06-05T13:00:00Z","tipo_evento":"TEST","detalles":"test"}]}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/events", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.New().String())
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for invalid event ID, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "invalid event ID") {
			t.Errorf("expected 'invalid event ID' error, got: %s", w.Body.String())
		}
	})
}

func TestSalesSyncController_ServeHTTP_Routing(t *testing.T) {
	_, _, controller, cleanup := setupHandlersTestDB(t)
	defer cleanup()

	t.Run("Method not allowed for GET on /api/v1/sync/sales", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/sales", nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 Method Not Allowed, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "method not allowed") {
			t.Errorf("expected 'method not allowed' error, got: %s", w.Body.String())
		}
	})

	t.Run("Method not allowed for PUT on /api/v1/sync/events", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/sync/events", nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 Method Not Allowed, got %d", w.Code)
		}
	})

	t.Run("Method not allowed for POST on /api/v1/inventory/stock/xxx", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/stock/"+uuid.New().String(), nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 Method Not Allowed for POST on stock endpoint, got %d", w.Code)
		}
	})

	t.Run("Method not allowed for DELETE on /api/v1/sync/closures", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/sync/closures", nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 Method Not Allowed, got %d", w.Code)
		}
	})

	t.Run("Method not allowed for GET on /api/v1/sync/voids", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/voids", nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 Method Not Allowed, got %d", w.Code)
		}
	})

	t.Run("Unknown path returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown/path", nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", w.Code)
		}
	})

	t.Run("Health check endpoint returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK for health check, got %d", w.Code)
		}
	})
}

func TestCatalogSyncController_ServeHTTP_Routing(t *testing.T) {
	db, cleanup := setupCatalogTestDB(t)
	defer cleanup()

	tracker := idempotency.NewTracker(db, true)
	err := tracker.InitSchema(context.Background())
	if err != nil {
		t.Fatalf("failed to init idempotency schema: %v", err)
	}

	controller := adapters.NewCatalogSyncController(db, true, tracker)

	t.Run("Method not allowed for POST on /api/v1/catalog/sync", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/catalog/sync", nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 Method Not Allowed, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "method not allowed") {
			t.Errorf("expected 'method not allowed' error, got: %s", w.Body.String())
		}
	})

	t.Run("Method not allowed for GET on /api/v1/catalog/clients/dossier", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/clients/dossier", nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 Method Not Allowed, got %d", w.Code)
		}
	})

	t.Run("Method not allowed for GET on /api/v1/sync/payments", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/payments", nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 Method Not Allowed, got %d", w.Code)
		}
	})

	t.Run("Unknown path returns 404 for catalog controller", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/nonexistent", nil)
		w := httptest.NewRecorder()
		controller.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", w.Code)
		}
	})
}
