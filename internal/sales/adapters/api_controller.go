package adapters

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ferrowin/internal/sales/domain"
	"github.com/google/uuid"
)

// Patch request types for partial updates
type patchQuoteRequest struct {
	ClientID  *string `json:"client_id"`
	ExpiresAt *string `json:"expires_at"`
}

type cancelSalesDocRequest struct {
	Reason string `json:"reason"`
}

type SalesController struct {
	service *domain.SalesService
}

func NewSalesController(service *domain.SalesService) *SalesController {
	return &SalesController{
		service: service,
	}
}

func (c *SalesController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := r.URL.Path

	// Quotes routing
	if path == "/api/v1/sales/quotes" {
		switch r.Method {
		case http.MethodGet:
			c.HandleGetQuotes(w, r)
		case http.MethodPost:
			c.HandleCreateQuote(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	if path == "/api/v1/sales/quotes/convert" {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleConvertQuoteToOrder(w, r)
		return
	}

	if strings.HasPrefix(path, "/api/v1/sales/quotes/") {
		// /api/v1/sales/quotes/{id}/cancel
		parts := strings.Split(strings.TrimPrefix(path, "/api/v1/sales/quotes/"), "/")
		if len(parts) >= 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
			c.HandleCancelQuote(w, r)
			return
		}
		if len(parts) == 1 && parts[0] != "" && parts[0] != "convert" {
			switch r.Method {
			case http.MethodGet:
				c.HandleGetQuote(w, r)
				return
			case http.MethodPatch:
				c.HandlePatchQuote(w, r)
				return
			}
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Orders routing
	if path == "/api/v1/sales/orders" {
		switch r.Method {
		case http.MethodGet:
			c.HandleGetOrders(w, r)
		case http.MethodPost:
			c.HandleCreateOrder(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	if path == "/api/v1/sales/orders/convert" {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleConvertOrderToDeliveryNote(w, r)
		return
	}

	if strings.HasPrefix(path, "/api/v1/sales/orders/") {
		parts := strings.Split(strings.TrimPrefix(path, "/api/v1/sales/orders/"), "/")
		if len(parts) >= 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
			c.HandleCancelOrder(w, r)
			return
		}
		if len(parts) == 1 && parts[0] != "" && parts[0] != "convert" {
			switch r.Method {
			case http.MethodGet:
				c.HandleGetOrder(w, r)
				return
			case http.MethodPatch:
				c.HandlePatchOrder(w, r)
				return
			}
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Delivery-notes routing
	if path == "/api/v1/sales/delivery-notes" {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleGetDeliveryNotes(w, r)
		return
	}

	if path == "/api/v1/sales/delivery-notes/process" {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleProcessDeliveryNote(w, r)
		return
	}

	if path == "/api/v1/sales/delivery-notes/convert" {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleConvertDeliveryNoteToInvoice(w, r)
		return
	}

	if strings.HasPrefix(path, "/api/v1/sales/delivery-notes/") {
		parts := strings.Split(strings.TrimPrefix(path, "/api/v1/sales/delivery-notes/"), "/")
		if len(parts) >= 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
			c.HandleCancelDeliveryNote(w, r)
			return
		}
		if len(parts) == 1 && parts[0] != "" && parts[0] != "process" && parts[0] != "convert" {
			switch r.Method {
			case http.MethodGet:
				c.HandleGetDeliveryNote(w, r)
				return
			case http.MethodPatch:
				c.HandlePatchDeliveryNote(w, r)
				return
			}
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Invoices routing
	if path == "/api/v1/sales/invoices" {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleGetInvoices(w, r)
		return
	}

	if strings.HasPrefix(path, "/api/v1/sales/invoices/") {
		parts := strings.Split(strings.TrimPrefix(path, "/api/v1/sales/invoices/"), "/")
		if len(parts) >= 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
			c.HandleCancelInvoice(w, r)
			return
		}
		if len(parts) == 1 && parts[0] != "" {
			if r.Method == http.MethodGet {
				c.HandleGetInvoice(w, r)
				return
			}
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	http.NotFound(w, r)
}

type CreateQuoteRequest struct {
	ClientID  string             `json:"client_id"`
	ExpiresAt string             `json:"expires_at"`
	Lineas    []QuoteLineRequest `json:"lineas"`
}

type QuoteLineRequest struct {
	ProductoID     string  `json:"producto_id"`
	Cantidad       float64 `json:"cantidad"`
	PrecioUnitario float64 `json:"precio_unitario"`
	CosteUnitario  float64 `json:"coste_unitario"`
}

type ConvertQuoteRequest struct {
	QuoteID           string `json:"quote_id"`
	UserID            string `json:"user_id"`
	RecalculatePrices bool   `json:"recalculate_prices"`
}

type CreateOrderRequest struct {
	QuoteID *string            `json:"quote_id,omitempty"`
	Lineas  []OrderLineRequest `json:"lineas"`
}

type OrderLineRequest struct {
	ProductoID     string  `json:"producto_id"`
	Cantidad       float64 `json:"cantidad"`
	PrecioUnitario float64 `json:"precio_unitario"`
}

type ConvertOrderRequest struct {
	OrderID     string `json:"order_id"`
	WarehouseID string `json:"warehouse_id"`
}

type ProcessDNRequest struct {
	DeliveryNoteID string `json:"delivery_note_id"`
}

type ConvertDNRequest struct {
	DeliveryNoteID    string `json:"delivery_note_id"`
	TerminalID        string `json:"terminal_id"`
	InvoicingSeriesID string `json:"invoicing_series_id"`
}

func getTenantID(r *http.Request) (uuid.UUID, error) {
	val := r.Header.Get("X-Empresa-ID")
	if val == "" {
		return uuid.Nil, errors.New("missing X-Empresa-ID header")
	}
	u, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, errors.New("invalid X-Empresa-ID format")
	}
	return u, nil
}

func (c *SalesController) HandleCreateQuote(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var req CreateQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	clientUUID, err := uuid.Parse(req.ClientID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid client_id"})
		return
	}

	expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid expires_at format"})
		return
	}

	lines := make([]domain.QuoteLine, len(req.Lineas))
	for i, l := range req.Lineas {
		prodUUID, err := uuid.Parse(l.ProductoID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("invalid producto_id at line %d", i)})
			return
		}
		lines[i] = domain.QuoteLine{
			ProductoID:     prodUUID,
			Cantidad:       l.Cantidad,
			PrecioUnitario: l.PrecioUnitario,
			CosteUnitario:  l.CosteUnitario,
		}
	}

	q, err := c.service.CreateQuote(r.Context(), empID, clientUUID, expiresAt, lines)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(q)
}

func (c *SalesController) HandleConvertQuoteToOrder(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var req ConvertQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	quoteUUID, err := uuid.Parse(req.QuoteID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid quote_id"})
		return
	}

	userUUID, err := uuid.Parse(req.UserID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid user_id"})
		return
	}

	order, err := c.service.ConvertQuoteToOrder(r.Context(), empID, quoteUUID, userUUID, domain.ConvertQuoteOptions{
		RecalculatePrices: req.RecalculatePrices,
	})
	if err != nil {
		if errors.Is(err, domain.ErrTenantMismatch) {
			w.WriteHeader(http.StatusForbidden)
		} else if errors.Is(err, domain.ErrUnauthorized) {
			w.WriteHeader(http.StatusForbidden)
		} else if errors.Is(err, domain.ErrDocumentAlreadyConverted) || errors.Is(err, domain.ErrDocumentAlreadyCancelled) {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(order)
}

func (c *SalesController) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	var quoteUUIDPtr *uuid.UUID
	if req.QuoteID != nil && *req.QuoteID != "" {
		qUUID, err := uuid.Parse(*req.QuoteID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid quote_id"})
			return
		}
		quoteUUIDPtr = &qUUID
	}

	lines := make([]domain.OrderLine, len(req.Lineas))
	for i, l := range req.Lineas {
		prodUUID, err := uuid.Parse(l.ProductoID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("invalid producto_id at line %d", i)})
			return
		}
		lines[i] = domain.OrderLine{
			ProductoID:     prodUUID,
			Cantidad:       l.Cantidad,
			PrecioUnitario: l.PrecioUnitario,
		}
	}

	o, err := c.service.CreateOrder(r.Context(), empID, quoteUUIDPtr, lines)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(o)
}

func (c *SalesController) HandleConvertOrderToDeliveryNote(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var req ConvertOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	orderUUID, err := uuid.Parse(req.OrderID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid order_id"})
		return
	}

	whUUID, err := uuid.Parse(req.WarehouseID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid warehouse_id"})
		return
	}

	dn, err := c.service.ConvertOrderToDeliveryNote(r.Context(), empID, orderUUID, whUUID)
	if err != nil {
		if errors.Is(err, domain.ErrTenantMismatch) {
			w.WriteHeader(http.StatusForbidden)
		} else if errors.Is(err, domain.ErrDocumentAlreadyConverted) || errors.Is(err, domain.ErrDocumentAlreadyCancelled) {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(dn)
}

func (c *SalesController) HandleProcessDeliveryNote(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var req ProcessDNRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	dnUUID, err := uuid.Parse(req.DeliveryNoteID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid delivery_note_id"})
		return
	}

	err = c.service.ProcessDeliveryNote(r.Context(), empID, dnUUID)
	if err != nil {
		if errors.Is(err, domain.ErrTenantMismatch) {
			w.WriteHeader(http.StatusForbidden)
		} else if strings.Contains(err.Error(), "insufficient stock") {
			w.WriteHeader(http.StatusConflict)
		} else if errors.Is(err, domain.ErrInvalidStatus) {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "delivery note processed, stock withdrawn"})
}

func (c *SalesController) HandleConvertDeliveryNoteToInvoice(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var req ConvertDNRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	dnUUID, err := uuid.Parse(req.DeliveryNoteID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid delivery_note_id"})
		return
	}

	termUUID, err := uuid.Parse(req.TerminalID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid terminal_id"})
		return
	}

	seriesUUID, err := uuid.Parse(req.InvoicingSeriesID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid invoicing_series_id"})
		return
	}

	invoice, err := c.service.ConvertDeliveryNoteToInvoice(r.Context(), empID, dnUUID, termUUID, seriesUUID)
	if err != nil {
		if errors.Is(err, domain.ErrTenantMismatch) {
			w.WriteHeader(http.StatusForbidden)
		} else if errors.Is(err, domain.ErrDocumentAlreadyConverted) || errors.Is(err, domain.ErrDocumentAlreadyCancelled) {
			w.WriteHeader(http.StatusConflict)
		} else if errors.Is(err, domain.ErrBillingServiceNil) {
			w.WriteHeader(http.StatusInternalServerError)
		} else if strings.Contains(err.Error(), "must be Processed before invoicing") {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(invoice)
}

// List response types
type salesListResponse[T any] struct {
	Data     []T   `json:"data"`
	Total    int   `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// HandleGetQuotes handles GET /api/v1/sales/quotes
func (c *SalesController) HandleGetQuotes(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	q := r.URL.Query()
	var filter domain.DocumentFilter

	if v := q.Get("estado"); v != "" {
		filter.Estado = &v
	}
	if v := q.Get("cliente_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			filter.ClientID = &id
		}
	}
	if v := q.Get("desde"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			filter.Desde = &t
		}
	}
	if v := q.Get("hasta"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			filter.Hasta = &t
		}
	}
	if v := q.Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			filter.Page = p
		}
	}
	if v := q.Get("page_size"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil {
			filter.PageSize = ps
		}
	}

	quotes, total, err := c.service.ListQuotes(r.Context(), empID, filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	resp := salesListResponse[*domain.Quote]{
		Data:     quotes,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}
	if resp.Page < 1 {
		resp.Page = 1
	}
	if resp.PageSize < 1 {
		resp.PageSize = 20
	}

	json.NewEncoder(w).Encode(resp)
}

// HandleGetQuote handles GET /api/v1/sales/quotes/{id}
func (c *SalesController) HandleGetQuote(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/sales/quotes/"), "/")
	if len(parts) == 0 || parts[0] == "" || parts[0] == "convert" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	quoteID, err := uuid.Parse(parts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid quote ID format"})
		return
	}

	quote, err := c.service.GetQuote(r.Context(), quoteID)
	if err != nil {
		if errors.Is(err, domain.ErrQuoteNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if quote.EmpresaID != empID {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "quote not found"})
		return
	}

	json.NewEncoder(w).Encode(quote)
}

// HandleGetOrders handles GET /api/v1/sales/orders
func (c *SalesController) HandleGetOrders(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	q := r.URL.Query()
	var filter domain.DocumentFilter

	if v := q.Get("estado"); v != "" {
		filter.Estado = &v
	}
	if v := q.Get("desde"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			filter.Desde = &t
		}
	}
	if v := q.Get("hasta"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			filter.Hasta = &t
		}
	}
	if v := q.Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			filter.Page = p
		}
	}
	if v := q.Get("page_size"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil {
			filter.PageSize = ps
		}
	}

	orders, total, err := c.service.ListOrders(r.Context(), empID, filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	resp := salesListResponse[*domain.Order]{
		Data:     orders,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}
	if resp.Page < 1 {
		resp.Page = 1
	}
	if resp.PageSize < 1 {
		resp.PageSize = 20
	}

	json.NewEncoder(w).Encode(resp)
}

// HandleGetOrder handles GET /api/v1/sales/orders/{id}
func (c *SalesController) HandleGetOrder(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/sales/orders/"), "/")
	if len(parts) == 0 || parts[0] == "" || parts[0] == "convert" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	orderID, err := uuid.Parse(parts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid order ID format"})
		return
	}

	order, err := c.service.GetOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if order.EmpresaID != empID {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "order not found"})
		return
	}

	json.NewEncoder(w).Encode(order)
}

// HandleGetDeliveryNotes handles GET /api/v1/sales/delivery-notes
func (c *SalesController) HandleGetDeliveryNotes(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	q := r.URL.Query()
	var filter domain.DocumentFilter

	if v := q.Get("estado"); v != "" {
		filter.Estado = &v
	}
	if v := q.Get("desde"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			filter.Desde = &t
		}
	}
	if v := q.Get("hasta"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			filter.Hasta = &t
		}
	}
	if v := q.Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			filter.Page = p
		}
	}
	if v := q.Get("page_size"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil {
			filter.PageSize = ps
		}
	}

	notes, total, err := c.service.ListDeliveryNotes(r.Context(), empID, filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	resp := salesListResponse[*domain.DeliveryNote]{
		Data:     notes,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}
	if resp.Page < 1 {
		resp.Page = 1
	}
	if resp.PageSize < 1 {
		resp.PageSize = 20
	}

	json.NewEncoder(w).Encode(resp)
}

// HandleGetDeliveryNote handles GET /api/v1/sales/delivery-notes/{id}
func (c *SalesController) HandleGetDeliveryNote(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/sales/delivery-notes/"), "/")
	if len(parts) == 0 || parts[0] == "" || parts[0] == "process" || parts[0] == "convert" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	noteID, err := uuid.Parse(parts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid delivery note ID format"})
		return
	}

	note, err := c.service.GetDeliveryNote(r.Context(), noteID)
	if err != nil {
		if errors.Is(err, domain.ErrDeliveryNoteNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if note.EmpresaID != empID {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "delivery note not found"})
		return
	}

	json.NewEncoder(w).Encode(note)
}

// HandleGetInvoices handles GET /api/v1/sales/invoices
func (c *SalesController) HandleGetInvoices(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	q := r.URL.Query()
	var filter domain.DocumentFilter

	if v := q.Get("estado"); v != "" {
		filter.Estado = &v
	}
	if v := q.Get("desde"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			filter.Desde = &t
		}
	}
	if v := q.Get("hasta"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			filter.Hasta = &t
		}
	}
	if v := q.Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			filter.Page = p
		}
	}
	if v := q.Get("page_size"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil {
			filter.PageSize = ps
		}
	}

	invoices, total, err := c.service.ListInvoices(r.Context(), empID, filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	resp := salesListResponse[*domain.Invoice]{
		Data:     invoices,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}
	if resp.Page < 1 {
		resp.Page = 1
	}
	if resp.PageSize < 1 {
		resp.PageSize = 20
	}

	json.NewEncoder(w).Encode(resp)
}

// HandleGetInvoice handles GET /api/v1/sales/invoices/{id}
func (c *SalesController) HandleGetInvoice(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/sales/invoices/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	invoiceID, err := uuid.Parse(parts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid invoice ID format"})
		return
	}

	invoice, err := c.service.GetInvoice(r.Context(), invoiceID)
	if err != nil {
		if errors.Is(err, domain.ErrInvoiceNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if invoice.EmpresaID != empID {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "invoice not found"})
		return
	}

	json.NewEncoder(w).Encode(invoice)
}

// parseSalesID extracts the document ID from paths like /api/v1/sales/quotes/{id}/...
func parseSalesID(prefix, path string) (uuid.UUID, error) {
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(parts) == 0 || parts[0] == "" {
		return uuid.Nil, fmt.Errorf("missing ID in URL")
	}
	return uuid.Parse(parts[0])
}

// HandlePatchQuote handles PATCH /api/v1/sales/quotes/{id}
func (c *SalesController) HandlePatchQuote(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	id, err := parseSalesID("/api/v1/sales/quotes/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid quote ID"})
		return
	}
	var req patchQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	input := domain.UpdateQuoteInput{ID: id}
	if req.ClientID != nil {
		clientUUID, err := uuid.Parse(*req.ClientID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid client_id format"})
			return
		}
		input.ClientID = &clientUUID
	}
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid expires_at format, use RFC3339"})
			return
		}
		input.ExpiresAt = &t
	}
	if err := c.service.UpdateQuote(r.Context(), input); err != nil {
		if errors.Is(err, domain.ErrQuoteNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	// Fetch and return updated quote
	quote, err := c.service.GetQuote(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if quote.EmpresaID != empID {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "quote not found"})
		return
	}
	json.NewEncoder(w).Encode(quote)
}

// HandlePatchOrder handles PATCH /api/v1/sales/orders/{id}
func (c *SalesController) HandlePatchOrder(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	id, err := parseSalesID("/api/v1/sales/orders/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid order ID"})
		return
	}
	// No optional fields for orders currently — but verify the order exists
	input := domain.UpdateOrderInput{ID: id}
	if err := c.service.UpdateOrder(r.Context(), input); err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	// Return the current order
	order, err := c.service.GetOrder(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if order.EmpresaID != empID {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "order not found"})
		return
	}
	json.NewEncoder(w).Encode(order)
}

// HandlePatchDeliveryNote handles PATCH /api/v1/sales/delivery-notes/{id}
func (c *SalesController) HandlePatchDeliveryNote(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	id, err := parseSalesID("/api/v1/sales/delivery-notes/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid delivery note ID"})
		return
	}
	input := domain.UpdateDeliveryNoteInput{ID: id}
	if err := c.service.UpdateDeliveryNote(r.Context(), input); err != nil {
		if errors.Is(err, domain.ErrDeliveryNoteNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	dn, err := c.service.GetDeliveryNote(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if dn.EmpresaID != empID {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "delivery note not found"})
		return
	}
	json.NewEncoder(w).Encode(dn)
}

// HandleCancelQuote handles POST /api/v1/sales/quotes/{id}/cancel
func (c *SalesController) HandleCancelQuote(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	id, err := parseSalesID("/api/v1/sales/quotes/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid quote ID"})
		return
	}
	var cancelReq cancelSalesDocRequest
	json.NewDecoder(r.Body).Decode(&cancelReq)

	err = c.service.CancelQuote(r.Context(), empID, id)
	if err != nil {
		if errors.Is(err, domain.ErrQuoteNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else if errors.Is(err, domain.ErrTenantMismatch) {
			w.WriteHeader(http.StatusForbidden)
		} else if errors.Is(err, domain.ErrInvalidStatus) {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	quote, err := c.service.GetQuote(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(quote)
}

// HandleCancelOrder handles POST /api/v1/sales/orders/{id}/cancel
func (c *SalesController) HandleCancelOrder(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	id, err := parseSalesID("/api/v1/sales/orders/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid order ID"})
		return
	}
	var cancelReq cancelSalesDocRequest
	json.NewDecoder(r.Body).Decode(&cancelReq)

	err = c.service.CancelOrder(r.Context(), empID, id)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else if errors.Is(err, domain.ErrTenantMismatch) {
			w.WriteHeader(http.StatusForbidden)
		} else if errors.Is(err, domain.ErrInvalidStatus) {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	order, err := c.service.GetOrder(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(order)
}

// HandleCancelDeliveryNote handles POST /api/v1/sales/delivery-notes/{id}/cancel
func (c *SalesController) HandleCancelDeliveryNote(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	id, err := parseSalesID("/api/v1/sales/delivery-notes/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid delivery note ID"})
		return
	}
	var cancelReq cancelSalesDocRequest
	json.NewDecoder(r.Body).Decode(&cancelReq)

	err = c.service.CancelDeliveryNote(r.Context(), empID, id)
	if err != nil {
		if errors.Is(err, domain.ErrDeliveryNoteNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else if errors.Is(err, domain.ErrTenantMismatch) {
			w.WriteHeader(http.StatusForbidden)
		} else if errors.Is(err, domain.ErrInvalidStatus) {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	dn, err := c.service.GetDeliveryNote(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(dn)
}

// HandleCancelInvoice handles POST /api/v1/sales/invoices/{id}/cancel
func (c *SalesController) HandleCancelInvoice(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	id, err := parseSalesID("/api/v1/sales/invoices/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid invoice ID"})
		return
	}
	var cancelReq cancelSalesDocRequest
	json.NewDecoder(r.Body).Decode(&cancelReq)

	err = c.service.CancelInvoice(r.Context(), empID, id)
	if err != nil {
		if errors.Is(err, domain.ErrInvoiceNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else if errors.Is(err, domain.ErrTenantMismatch) {
			w.WriteHeader(http.StatusForbidden)
		} else if errors.Is(err, domain.ErrInvalidStatus) {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	invoice, err := c.service.GetInvoice(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(invoice)
}
