package adapters

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	inventoryadapters "ferrowin/internal/inventory/adapters"
	"ferrowin/internal/inventory/domain"
	"ferrowin/internal/shared/idempotency"

	"github.com/google/uuid"
)

// InvoiceNumberGenerator defines the contract for generating server-side invoice numbers.
type InvoiceNumberGenerator interface {
	GenerateFacturaNumber(ctx context.Context, terminalID uuid.UUID) (string, int, error)
}

// SalesSyncController handles synchronization of offline sales and tracks idempotency.
type SalesSyncController struct {
	db               *sql.DB
	isSQLite         bool
	inventoryService *domain.InventoryService
	idempTracker     *idempotency.Tracker
	billingService   InvoiceNumberGenerator
	defaultWarehouse uuid.UUID
}

// NewSalesSyncController creates a new SalesSyncController.
func NewSalesSyncController(db *sql.DB, isSQLite bool, inventoryService *domain.InventoryService, idempTracker *idempotency.Tracker, billingService InvoiceNumberGenerator) *SalesSyncController {
	return &SalesSyncController{
		db:               db,
		isSQLite:         isSQLite,
		inventoryService: inventoryService,
		idempTracker:     idempTracker,
		billingService:   billingService,
		defaultWarehouse: uuid.Nil,
	}
}

// SetDefaultWarehouse allows customizing the warehouse ID used for stock movements.
func (c *SalesSyncController) SetDefaultWarehouse(id uuid.UUID) {
	c.defaultWarehouse = id
}

// ServeHTTP makes the controller implement http.Handler.
func (c *SalesSyncController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1/health" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.URL.Path == "/api/v1/sync/sales" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		c.HandleSyncSales(w, r)
		return
	}
	if r.URL.Path == "/api/v1/sync/events" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		c.HandleSyncEvents(w, r)
		return
	}
	if r.URL.Path == "/api/v1/sync/voids" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		c.HandleSyncVoids(w, r)
		return
	}
	if r.URL.Path == "/api/v1/sync/closures" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		c.HandleSyncClosures(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/v1/inventory/stock/") {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		c.HandleGetStock(w, r)
		return
	}
	http.NotFound(w, r)
}

// SyncItem represents a single item in an offline sale.
type SyncItem struct {
	ItemID    string  `json:"item_id"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

// SyncSalePayment represents a payment method in an offline POS sale.
type SyncSalePayment struct {
	Method string  `json:"method"`
	Amount float64 `json:"amount"`
}

// SyncSale represents an offline sale document being synchronized.
type SyncSale struct {
	ID                  string        `json:"id"`
	NumeroFactura       string        `json:"invoice_number"`
	NumeroSecuencia     int           `json:"sequence_number"`
	CreatedAt           string        `json:"created_at"`
	Total               float64       `json:"total"`
	Items               []SyncItem    `json:"items"`
	FirmaRegistro       *string       `json:"firma_registro"`
	HashAnterior        *string       `json:"hash_anterior"`
	DatosEncadenamiento *string       `json:"datos_encadenamiento"`
	Subtotal            float64       `json:"subtotal"`
	TaxTotal            float64       `json:"tax_total"`
	DiscountTotal       float64       `json:"discount_total"`
	Status              string        `json:"status"`
	VoidReason          *string       `json:"void_reason,omitempty"`
	VoidedAt            *string       `json:"voided_at,omitempty"`
	Payments            []SyncSalePayment `json:"payments"`
}

// SyncRequest is the request payload for POS sales synchronization.
type SyncRequest struct {
	Sales []SyncSale `json:"sales"`
}

// SyncEvent represents an audit event log being synchronized.
type SyncEvent struct {
	ID         string `json:"id"`
	FechaHora  string `json:"fecha_hora"`
	TipoEvento string `json:"tipo_evento"`
	Detalles   string `json:"detalles"`
}

// SyncEventsRequest is the request payload for POS events synchronization.
type SyncEventsRequest struct {
	Events []SyncEvent `json:"events"`
}

// SyncVoidRequest is the request payload for synchronizing a void/ANULACION event.
type SyncVoidRequest struct {
	SaleID        string `json:"sale_id"`
	Motivo        string `json:"reason"`
	FirmaRegistro string `json:"firma_registro"`
	HashAnterior  string `json:"hash_anterior"`
}

// SyncVoidResponse is the response payload for void synchronization.
type SyncVoidResponse struct {
	Status string `json:"status"`
}

// SyncResponse is the response payload for POS sales synchronization.
type SyncResponse struct {
	Status         string            `json:"status"`
	SyncedCount    int               `json:"synced_count"`
	ProcessedIDs   []string          `json:"processed_ids"`
	InvoiceNumbers map[string]string `json:"invoice_numbers,omitempty"`
}

// HandleSyncSales processes the incoming POST request to synchronize POS offline sales.
func (c *SalesSyncController) HandleSyncSales(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		http.Error(w, "missing Idempotency-Key header", http.StatusBadRequest)
		return
	}

	if !c.idempTracker.IsValidKey(idemKey) {
		http.Error(w, "invalid Idempotency-Key format", http.StatusBadRequest)
		return
	}

	// Check if idempotency key exists
	found, savedBody, err := c.idempTracker.GetResponse(r.Context(), idemKey)
	if err != nil {
		http.Error(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if found && savedBody != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(savedBody))
		return
	}

	// Reserve key (insert empty string response first to prevent concurrent execution)
	if !found {
		err = c.idempTracker.ReserveKey(r.Context(), idemKey)
		if err != nil {
			http.Error(w, "duplicate key or reservation failed: "+err.Error(), http.StatusConflict)
			return
		}
	}

	// Start database transaction
	tx, err := c.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	txCtx := inventoryadapters.WithTx(r.Context(), tx)

	// Parse body
	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	var processedIDs []string
	invoiceNumbers := make(map[string]string)

	// Define queries dynamically for SQLite vs Postgres
	var selectSeriesQuery string
	var insertInvoiceQuery string

	if c.isSQLite {
		selectSeriesQuery = "SELECT id, terminal_id FROM invoicing_series WHERE prefix = ?"
		insertInvoiceQuery = `INSERT INTO invoice (id, delivery_note_id, terminal_id, invoicing_series_id, invoice_number, sequence_number, total, status, created_at, firma_registro, hash_anterior, datos_encadenamiento)
							  VALUES (?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	} else {
		selectSeriesQuery = "SELECT id, terminal_id FROM invoicing_series WHERE prefix = $1"
		insertInvoiceQuery = `INSERT INTO invoice (id, delivery_note_id, terminal_id, invoicing_series_id, invoice_number, sequence_number, total, status, created_at, firma_registro, hash_anterior, datos_encadenamiento)
							  VALUES ($1, NULL, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	}

	for _, sale := range req.Sales {
		saleUUID, err := uuid.Parse(sale.ID)
		if err != nil {
			http.Error(w, "invalid sale ID: "+sale.ID, http.StatusBadRequest)
			return
		}

		parts := strings.Split(sale.NumeroFactura, "-")
		if len(parts) < 2 {
			http.Error(w, "invalid invoice number format: "+sale.NumeroFactura, http.StatusBadRequest)
			return
		}
		prefix := parts[0]

		var seriesIDStr, terminalIDStr string
		err = tx.QueryRowContext(r.Context(), selectSeriesQuery, prefix).Scan(&seriesIDStr, &terminalIDStr)
		if err == sql.ErrNoRows {
			http.Error(w, "invoicing series not found for prefix: "+prefix, http.StatusBadRequest)
			return
		} else if err != nil {
			http.Error(w, "database error querying series: "+err.Error(), http.StatusInternalServerError)
			return
		}

		seriesUUID, _ := uuid.Parse(seriesIDStr)
		terminalUUID, _ := uuid.Parse(terminalIDStr)

		// Generate server-side invoice number using the billing service
		invoiceNumber, seq, err := c.billingService.GenerateFacturaNumber(r.Context(), terminalUUID)
		if err != nil {
			http.Error(w, "failed to generate invoice number: "+err.Error(), http.StatusInternalServerError)
			return
		}

		parsedCreatedAt, err := time.Parse(time.RFC3339, sale.CreatedAt)
		if err != nil {
			parsedCreatedAt, err = time.Parse("2006-01-02T15:04:05Z", sale.CreatedAt)
			if err != nil {
				parsedCreatedAt = time.Now()
			}
		}

		// Insert Invoice with server-generated invoice_number and sequence_number
		_, err = tx.ExecContext(r.Context(), insertInvoiceQuery,
			saleUUID.String(),
			terminalUUID.String(),
			seriesUUID.String(),
			invoiceNumber,
			seq,
			sale.Total,
			"Issued",
			parsedCreatedAt.UTC(),
			sale.FirmaRegistro,
			sale.HashAnterior,
			sale.DatosEncadenamiento,
		)
		if err != nil {
			http.Error(w, "failed to insert invoice: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Insert Movements
		for _, item := range sale.Items {
			itemUUID, err := uuid.Parse(item.ItemID)
			if err != nil {
				http.Error(w, "invalid item ID: "+item.ItemID, http.StatusBadRequest)
				return
			}

			docType := "INVOICE"
			_, err = c.inventoryService.RecordSyncAdjustment(
				txCtx,
				itemUUID,
				c.defaultWarehouse,
				item.Quantity,
				&docType,
				&saleUUID,
			)
			if err != nil {
				http.Error(w, "failed to record stock movement: "+err.Error(), http.StatusInternalServerError)
				return
			}

			_, err = c.inventoryService.ReconcileFIFO(
				txCtx,
				itemUUID,
				c.defaultWarehouse,
			)
			if err != nil {
				http.Error(w, "failed to reconcile FIFO: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		invoiceNumbers[sale.ID] = invoiceNumber
		processedIDs = append(processedIDs, sale.ID)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare response
	resp := SyncResponse{
		Status:         "success",
		SyncedCount:    len(processedIDs),
		ProcessedIDs:   processedIDs,
		InvoiceNumbers: invoiceNumbers,
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "failed to marshal response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Save response to idempotency tracker
	_ = c.idempTracker.SaveResponse(r.Context(), idemKey, string(respBytes))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}

// SyncClosureRequest is the request payload for POS box closure synchronization.
type SyncClosureRequest struct {
	ID           string  `json:"id"`
	OpenedAt     string  `json:"opened_at"`
	ClosedAt     string  `json:"closed_at"`
	CashReported float64 `json:"cash_reported"`
	CardReported float64 `json:"card_reported"`
	SalesTotal   float64 `json:"sales_total"`
}

// SyncClosureResponse is the response payload for POS box closure synchronization.
type SyncClosureResponse struct {
	Status string `json:"status"`
	ID     string `json:"id"`
}

// HandleSyncClosures processes the incoming POST request to synchronize POS offline box closures.
func (c *SalesSyncController) HandleSyncClosures(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		http.Error(w, "missing Idempotency-Key header", http.StatusBadRequest)
		return
	}

	if !c.idempTracker.IsValidKey(idemKey) {
		http.Error(w, "invalid Idempotency-Key format", http.StatusBadRequest)
		return
	}

	// Check if idempotency key exists
	found, savedBody, err := c.idempTracker.GetResponse(r.Context(), idemKey)
	if err != nil {
		http.Error(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if found && savedBody != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(savedBody))
		return
	}

	// Reserve key (insert empty string response first to prevent concurrent execution)
	if !found {
		err = c.idempTracker.ReserveKey(r.Context(), idemKey)
		if err != nil {
			http.Error(w, "duplicate key or reservation failed: "+err.Error(), http.StatusConflict)
			return
		}
	}

	// Start database transaction
	tx, err := c.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Parse body
	var req SyncClosureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	closureUUID, err := uuid.Parse(req.ID)
	if err != nil {
		http.Error(w, "invalid closure ID: "+req.ID, http.StatusBadRequest)
		return
	}

	parsedOpenedAt, err := time.Parse(time.RFC3339, req.OpenedAt)
	if err != nil {
		parsedOpenedAt = time.Now()
	}

	parsedClosedAt, err := time.Parse(time.RFC3339, req.ClosedAt)
	if err != nil {
		parsedClosedAt = time.Now()
	}

	// Insert into box_closures
	var insertQuery string
	if c.isSQLite {
		insertQuery = `INSERT INTO box_closures
			(id, opened_at, closed_at, cash_reported, card_reported, sales_total, synced_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`
	} else {
		insertQuery = `INSERT INTO box_closures
			(id, opened_at, closed_at, cash_reported, card_reported, sales_total, synced_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`
	}

	_, err = tx.ExecContext(r.Context(), insertQuery,
		closureUUID.String(),
		parsedOpenedAt.UTC(),
		parsedClosedAt.UTC(),
		req.CashReported,
		req.CardReported,
		req.SalesTotal,
		time.Now().UTC(),
	)
	if err != nil {
		http.Error(w, "failed to insert box closure: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare response
	resp := SyncClosureResponse{
		Status: "success",
		ID:     req.ID,
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "failed to marshal response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Save response to idempotency tracker
	_ = c.idempTracker.SaveResponse(r.Context(), idemKey, string(respBytes))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}

// HandleGetStock returns the stock level for a specific item.
func (c *SalesSyncController) HandleGetStock(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		http.Error(w, "bad request: missing item ID", http.StatusBadRequest)
		return
	}
	itemIDStr := parts[5]
	itemUUID, err := uuid.Parse(itemIDStr)
	if err != nil {
		http.Error(w, "invalid item ID format", http.StatusBadRequest)
		return
	}

	stock, err := c.inventoryService.GetAvailableStock(r.Context(), itemUUID, c.defaultWarehouse)
	if err != nil {
		http.Error(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := struct {
		ItemID string  `json:"item_id"`
		Stock  float64 `json:"stock"`
	}{
		ItemID: itemIDStr,
		Stock:  stock,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// HandleSyncVoids processes the incoming POST request to synchronize void/ANULACION events.
func (c *SalesSyncController) HandleSyncVoids(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		http.Error(w, "missing Idempotency-Key header", http.StatusBadRequest)
		return
	}

	if !c.idempTracker.IsValidKey(idemKey) {
		http.Error(w, "invalid Idempotency-Key format", http.StatusBadRequest)
		return
	}

	// Check if idempotency key exists
	found, savedBody, err := c.idempTracker.GetResponse(r.Context(), idemKey)
	if err != nil {
		http.Error(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if found && savedBody != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(savedBody))
		return
	}

	// Reserve key (insert empty string response first to prevent concurrent execution)
	if !found {
		err = c.idempTracker.ReserveKey(r.Context(), idemKey)
		if err != nil {
			http.Error(w, "duplicate key or reservation failed: "+err.Error(), http.StatusConflict)
			return
		}
	}

	// Parse body
	var req SyncVoidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.SaleID == "" || req.Motivo == "" || req.FirmaRegistro == "" || req.HashAnterior == "" {
		http.Error(w, "missing required fields: sale_id, reason, firma_registro, hash_anterior", http.StatusBadRequest)
		return
	}

	// Open transaction for atomic check + insert
	tx, err := c.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Check if sale_id already voided
	var searchQuery string
	searchStr := `"sale_id":"` + req.SaleID + `"`
	if c.isSQLite {
		searchQuery = "SELECT COUNT(*) FROM registro_sucesos WHERE tipo_evento = 'ANULACION' AND detalles LIKE '%' || ? || '%'"
	} else {
		searchQuery = "SELECT COUNT(*) FROM registro_sucesos WHERE tipo_evento = 'ANULACION' AND detalles LIKE '%' || $1 || '%'"
	}

	var voidCount int
	err = tx.QueryRowContext(r.Context(), searchQuery, searchStr).Scan(&voidCount)
	if err != nil {
		http.Error(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if voidCount > 0 {
		http.Error(w, "sale already voided", http.StatusConflict)
		return
	}

	// Insert void event into registro_sucesos
	voidID := uuid.New().String()
	ahora := time.Now().UTC().Format(time.RFC3339)

	detallesMap := map[string]string{
		"sale_id":        req.SaleID,
		"reason":         req.Motivo,
		"firma_registro": req.FirmaRegistro,
		"hash_anterior":  req.HashAnterior,
	}
	detallesBytes, _ := json.Marshal(detallesMap)

	var insertQuery string
	if c.isSQLite {
		insertQuery = `INSERT INTO registro_sucesos (id, fecha_hora, tipo_evento, detalles, estado_sincronizacion)
						VALUES (?, ?, 'ANULACION', ?, 'SINCRONIZADO')`
	} else {
		insertQuery = `INSERT INTO registro_sucesos (id, fecha_hora, tipo_evento, detalles, estado_sincronizacion)
						VALUES ($1, $2, 'ANULACION', $3, 'SINCRONIZADO')`
	}

	_, err = tx.ExecContext(r.Context(), insertQuery,
		voidID,
		ahora,
		string(detallesBytes),
	)
	if err != nil {
		http.Error(w, "failed to insert void event: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare response
	resp := SyncVoidResponse{
		Status: "success",
	}
	respBytes, _ := json.Marshal(resp)

	// Save response to idempotency tracker
	_ = c.idempTracker.SaveResponse(r.Context(), idemKey, string(respBytes))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}

// HandleSyncEvents processes the incoming POST request to synchronize POS audit events.
func (c *SalesSyncController) HandleSyncEvents(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		http.Error(w, "missing Idempotency-Key header", http.StatusBadRequest)
		return
	}

	if !c.idempTracker.IsValidKey(idemKey) {
		http.Error(w, "invalid Idempotency-Key format", http.StatusBadRequest)
		return
	}

	// Check if idempotency key exists
	found, savedBody, err := c.idempTracker.GetResponse(r.Context(), idemKey)
	if err != nil {
		http.Error(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if found && savedBody != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(savedBody))
		return
	}

	// Reserve key (insert empty string response first to prevent concurrent execution)
	if !found {
		err = c.idempTracker.ReserveKey(r.Context(), idemKey)
		if err != nil {
			http.Error(w, "duplicate key or reservation failed: "+err.Error(), http.StatusConflict)
			return
		}
	}

	// Start database transaction
	tx, err := c.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Parse body
	var req SyncEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	var processedIDs []string

	// Define queries dynamically for SQLite vs Postgres
	var insertEventQuery string
	if c.isSQLite {
		insertEventQuery = `INSERT INTO registro_sucesos (id, fecha_hora, tipo_evento, detalles, estado_sincronizacion)
							VALUES (?, ?, ?, ?, 'SINCRONIZADO')`
	} else {
		insertEventQuery = `INSERT INTO registro_sucesos (id, fecha_hora, tipo_evento, detalles, estado_sincronizacion)
							VALUES ($1, $2, $3, $4, 'SINCRONIZADO')`
	}

	for _, event := range req.Events {
		eventUUID, err := uuid.Parse(event.ID)
		if err != nil {
			http.Error(w, "invalid event ID: "+event.ID, http.StatusBadRequest)
			return
		}

		parsedFechaHora, err := time.Parse(time.RFC3339, event.FechaHora)
		if err != nil {
			parsedFechaHora, err = time.Parse("2006-01-02T15:04:05Z", event.FechaHora)
			if err != nil {
				parsedFechaHora = time.Now()
			}
		}

		_, err = tx.ExecContext(r.Context(), insertEventQuery,
			eventUUID.String(),
			parsedFechaHora.UTC(),
			event.TipoEvento,
			event.Detalles,
		)
		if err != nil {
			http.Error(w, "failed to insert event: "+err.Error(), http.StatusInternalServerError)
			return
		}

		processedIDs = append(processedIDs, event.ID)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare response
	resp := SyncResponse{
		Status:       "success",
		SyncedCount:  len(processedIDs),
		ProcessedIDs: processedIDs,
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "failed to marshal response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Save response to idempotency tracker
	_ = c.idempTracker.SaveResponse(r.Context(), idemKey, string(respBytes))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}

