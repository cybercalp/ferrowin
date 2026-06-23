package adapters

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ferrowin/internal/sales/domain"
	"github.com/google/uuid"
)

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

	if r.URL.Path == "/api/v1/sales/quotes" {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleCreateQuote(w, r)
		return
	}

	if r.URL.Path == "/api/v1/sales/quotes/convert" {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleConvertQuoteToOrder(w, r)
		return
	}

	if r.URL.Path == "/api/v1/sales/orders" {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleCreateOrder(w, r)
		return
	}

	if r.URL.Path == "/api/v1/sales/orders/convert" {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleConvertOrderToDeliveryNote(w, r)
		return
	}

	if r.URL.Path == "/api/v1/sales/delivery-notes/process" {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleProcessDeliveryNote(w, r)
		return
	}

	if r.URL.Path == "/api/v1/sales/delivery-notes/convert" {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleConvertDeliveryNoteToInvoice(w, r)
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
