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
type patchPresupuestoRequest struct {
	ClienteID  *string `json:"cliente_id"`
	FechaValidez *string `json:"fecha_validez"`
}

type cancelSalesDocRequest struct {
	Motivo string `json:"motivo"`
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

	// Presupuestos routing
	if path == "/api/v1/sales/quotes" {
		switch r.Method {
		case http.MethodGet:
			c.HandleGetPresupuestos(w, r)
		case http.MethodPost:
			c.HandleCreatePresupuesto(w, r)
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
		c.HandleConvertPresupuestoToPedido(w, r)
		return
	}

	if strings.HasPrefix(path, "/api/v1/sales/quotes/") {
		// /api/v1/sales/quotes/{id}/cancel
		parts := strings.Split(strings.TrimPrefix(path, "/api/v1/sales/quotes/"), "/")
		if len(parts) >= 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
			c.HandleCancelPresupuesto(w, r)
			return
		}
		if len(parts) == 1 && parts[0] != "" && parts[0] != "convert" {
			switch r.Method {
			case http.MethodGet:
				c.HandleGetPresupuesto(w, r)
				return
			case http.MethodPatch:
				c.HandlePatchPresupuesto(w, r)
				return
			}
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Pedidos routing
	if path == "/api/v1/sales/orders" {
		switch r.Method {
		case http.MethodGet:
			c.HandleGetPedidos(w, r)
		case http.MethodPost:
			c.HandleCreatePedido(w, r)
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
		c.HandleConvertPedidoToAlbaran(w, r)
		return
	}

	if strings.HasPrefix(path, "/api/v1/sales/orders/") {
		parts := strings.Split(strings.TrimPrefix(path, "/api/v1/sales/orders/"), "/")
		if len(parts) >= 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
			c.HandleCancelPedido(w, r)
			return
		}
		if len(parts) == 1 && parts[0] != "" && parts[0] != "convert" {
			switch r.Method {
			case http.MethodGet:
				c.HandleGetPedido(w, r)
				return
			case http.MethodPatch:
				c.HandlePatchPedido(w, r)
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
		c.HandleGetAlbarans(w, r)
		return
	}

	if path == "/api/v1/sales/delivery-notes/process" {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleProcessAlbaran(w, r)
		return
	}

	if path == "/api/v1/sales/delivery-notes/convert" {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleConvertAlbaranToFactura(w, r)
		return
	}

	if strings.HasPrefix(path, "/api/v1/sales/delivery-notes/") {
		parts := strings.Split(strings.TrimPrefix(path, "/api/v1/sales/delivery-notes/"), "/")
		if len(parts) >= 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
			c.HandleCancelAlbaran(w, r)
			return
		}
		if len(parts) == 1 && parts[0] != "" && parts[0] != "process" && parts[0] != "convert" {
			switch r.Method {
			case http.MethodGet:
				c.HandleGetAlbaran(w, r)
				return
			case http.MethodPatch:
				c.HandlePatchAlbaran(w, r)
				return
			}
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Facturas rectificativas routing
	if path == "/api/v1/sales/facturas-rectificativas" {
		switch r.Method {
		case http.MethodGet:
			c.HandleGetFacturasRectificativas(w, r)
		case http.MethodPost:
			c.HandleCreateFacturaRectificativa(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	if strings.HasPrefix(path, "/api/v1/sales/facturas-rectificativas/") {
		parts := strings.Split(strings.TrimPrefix(path, "/api/v1/sales/facturas-rectificativas/"), "/")
		if len(parts) == 1 && parts[0] != "" {
			if r.Method == http.MethodGet {
				c.HandleGetFacturaRectificativa(w, r)
				return
			}
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Facturas routing
	if path == "/api/v1/sales/invoices" {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleGetFacturas(w, r)
		return
	}

	if strings.HasPrefix(path, "/api/v1/sales/invoices/") {
		parts := strings.Split(strings.TrimPrefix(path, "/api/v1/sales/invoices/"), "/")
		if len(parts) >= 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
			c.HandleCancelFactura(w, r)
			return
		}
		if len(parts) == 1 && parts[0] != "" {
			if r.Method == http.MethodGet {
				c.HandleGetFactura(w, r)
				return
			}
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	http.NotFound(w, r)
}

type CreatePresupuestoRequest struct {
	ClienteID  string             `json:"cliente_id"`
	FechaValidez string             `json:"fecha_validez"`
	Lineas    []PresupuestoLineaRequest `json:"lineas"`
}

type PresupuestoLineaRequest struct {
	ProductoID     string  `json:"producto_id"`
	Cantidad       float64 `json:"cantidad"`
	PrecioUnitario float64 `json:"precio_unitario"`
	CosteUnitario  float64 `json:"coste_unitario"`
}

type ConvertPresupuestoRequest struct {
	PresupuestoID     string                   `json:"presupuesto_id"`
	UserID            string                   `json:"user_id"`
	RecalculatePrices bool                     `json:"recalculate_prices"`
	Lineas            []ConversionLineaRequest `json:"lineas,omitempty"`
}

type ConversionLineaRequest struct {
	ProductoID string  `json:"producto_id"`
	Cantidad   float64 `json:"cantidad"`
}

type CreatePedidoRequest struct {
	PresupuestoID *string            `json:"presupuesto_id,omitempty"`
	Lineas  []PedidoLineaRequest `json:"lineas"`
}

type PedidoLineaRequest struct {
	ProductoID     string  `json:"producto_id"`
	Cantidad       float64 `json:"cantidad"`
	PrecioUnitario float64 `json:"precio_unitario"`
}

type ConvertPedidoRequest struct {
	PedidoID  string                   `json:"pedido_id"`
	AlmacenID string                   `json:"almacen_id"`
	Lineas    []ConversionLineaRequest `json:"lineas,omitempty"`
}

type ProcessDNRequest struct {
	AlbaranID string `json:"albaran_id"`
}

type ConvertDNRequest struct {
	AlbaranID          string                   `json:"albaran_id"`
	TerminalID         string                   `json:"terminal_id"`
	SerieFacturacionID string                   `json:"serie_facturacion_id"`
	Lineas             []ConversionLineaRequest `json:"lineas,omitempty"`
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

func (c *SalesController) HandleCreatePresupuesto(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var req CreatePresupuestoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	clientUUID, err := uuid.Parse(req.ClienteID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid client_id"})
		return
	}

	expiresAt, err := time.Parse(time.RFC3339, req.FechaValidez)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid expires_at format"})
		return
	}

	lines := make([]domain.PresupuestoLinea, len(req.Lineas))
	for i, l := range req.Lineas {
		prodUUID, err := uuid.Parse(l.ProductoID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("invalid producto_id at line %d", i)})
			return
		}
		lines[i] = domain.PresupuestoLinea{
			ProductoID:     prodUUID,
			Cantidad:       l.Cantidad,
			PrecioUnitario: l.PrecioUnitario,
			CosteUnitario:  l.CosteUnitario,
		}
	}

	q, err := c.service.CreatePresupuesto(r.Context(), empID, clientUUID, expiresAt, lines)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(q)
}

func (c *SalesController) HandleConvertPresupuestoToPedido(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var req ConvertPresupuestoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	quoteUUID, err := uuid.Parse(req.PresupuestoID)
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

	var lineas []domain.ConversionLineInput
	for _, l := range req.Lineas {
		prodUUID, err := uuid.Parse(l.ProductoID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid producto_id in lineas"})
			return
		}
		lineas = append(lineas, domain.ConversionLineInput{
			ProductoID: prodUUID,
			Cantidad:   l.Cantidad,
		})
	}

	input := domain.ConvertPresupuestoInput{
		PresupuestoID:     quoteUUID,
		UserID:            userUUID,
		RecalculatePrices: req.RecalculatePrices,
		Lineas:            lineas,
	}

	order, err := c.service.ConvertPresupuestoToPedido(r.Context(), empID, input)
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

func (c *SalesController) HandleCreatePedido(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var req CreatePedidoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	var quoteUUIDPtr *uuid.UUID
	if req.PresupuestoID != nil && *req.PresupuestoID != "" {
		qUUID, err := uuid.Parse(*req.PresupuestoID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid quote_id"})
			return
		}
		quoteUUIDPtr = &qUUID
	}

	lines := make([]domain.PedidoLinea, len(req.Lineas))
	for i, l := range req.Lineas {
		prodUUID, err := uuid.Parse(l.ProductoID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("invalid producto_id at line %d", i)})
			return
		}
		lines[i] = domain.PedidoLinea{
			ProductoID:     prodUUID,
			Cantidad:       l.Cantidad,
			PrecioUnitario: l.PrecioUnitario,
		}
	}

	o, err := c.service.CreatePedido(r.Context(), empID, quoteUUIDPtr, lines)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(o)
}

func (c *SalesController) HandleConvertPedidoToAlbaran(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var req ConvertPedidoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	orderUUID, err := uuid.Parse(req.PedidoID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid order_id"})
		return
	}

	whUUID, err := uuid.Parse(req.AlmacenID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid warehouse_id"})
		return
	}

	var lineas []domain.ConversionLineInput
	for _, l := range req.Lineas {
		prodUUID, err := uuid.Parse(l.ProductoID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid producto_id in lineas"})
			return
		}
		lineas = append(lineas, domain.ConversionLineInput{
			ProductoID: prodUUID,
			Cantidad:   l.Cantidad,
		})
	}

	input := domain.ConvertPedidoInput{
		PedidoID:  orderUUID,
		AlmacenID: whUUID,
		Lineas:    lineas,
	}

	dn, err := c.service.ConvertPedidoToAlbaran(r.Context(), empID, input)
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

func (c *SalesController) HandleProcessAlbaran(w http.ResponseWriter, r *http.Request) {
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

	dnUUID, err := uuid.Parse(req.AlbaranID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid delivery_note_id"})
		return
	}

	err = c.service.ProcessAlbaran(r.Context(), empID, dnUUID)
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

func (c *SalesController) HandleConvertAlbaranToFactura(w http.ResponseWriter, r *http.Request) {
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

	dnUUID, err := uuid.Parse(req.AlbaranID)
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

	seriesUUID, err := uuid.Parse(req.SerieFacturacionID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid invoicing_series_id"})
		return
	}

	var lineas []domain.ConversionLineInput
	for _, l := range req.Lineas {
		prodUUID, err := uuid.Parse(l.ProductoID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid producto_id in lineas"})
			return
		}
		lineas = append(lineas, domain.ConversionLineInput{
			ProductoID: prodUUID,
			Cantidad:   l.Cantidad,
		})
	}

	input := domain.ConvertAlbaranInput{
		AlbaranID:          dnUUID,
		TerminalID:         termUUID,
		SerieFacturacionID: seriesUUID,
		Lineas:             lineas,
	}

	invoice, err := c.service.ConvertAlbaranToFactura(r.Context(), empID, input)
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

// HandleGetPresupuestos handles GET /api/v1/sales/quotes
func (c *SalesController) HandleGetPresupuestos(w http.ResponseWriter, r *http.Request) {
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
			filter.ClienteID = &id
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

	quotes, total, err := c.service.ListPresupuestos(r.Context(), empID, filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	resp := salesListResponse[*domain.Presupuesto]{
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

// HandleGetPresupuesto handles GET /api/v1/sales/quotes/{id}
func (c *SalesController) HandleGetPresupuesto(w http.ResponseWriter, r *http.Request) {
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

	quote, err := c.service.GetPresupuesto(r.Context(), quoteID)
	if err != nil {
		if errors.Is(err, domain.ErrPresupuestoNotFound) {
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

// HandleGetPedidos handles GET /api/v1/sales/orders
func (c *SalesController) HandleGetPedidos(w http.ResponseWriter, r *http.Request) {
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

	orders, total, err := c.service.ListPedidos(r.Context(), empID, filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	resp := salesListResponse[*domain.Pedido]{
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

// HandleGetPedido handles GET /api/v1/sales/orders/{id}
func (c *SalesController) HandleGetPedido(w http.ResponseWriter, r *http.Request) {
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

	order, err := c.service.GetPedido(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, domain.ErrPedidoNotFound) {
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

// HandleGetAlbarans handles GET /api/v1/sales/delivery-notes
func (c *SalesController) HandleGetAlbarans(w http.ResponseWriter, r *http.Request) {
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

	notes, total, err := c.service.ListAlbarans(r.Context(), empID, filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	resp := salesListResponse[*domain.Albaran]{
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

// HandleGetAlbaran handles GET /api/v1/sales/delivery-notes/{id}
func (c *SalesController) HandleGetAlbaran(w http.ResponseWriter, r *http.Request) {
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

	note, err := c.service.GetAlbaran(r.Context(), noteID)
	if err != nil {
		if errors.Is(err, domain.ErrAlbaranNotFound) {
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

// HandleGetFacturas handles GET /api/v1/sales/invoices
func (c *SalesController) HandleGetFacturas(w http.ResponseWriter, r *http.Request) {
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

	invoices, total, err := c.service.ListFacturas(r.Context(), empID, filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	resp := salesListResponse[*domain.Factura]{
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

// HandleGetFactura handles GET /api/v1/sales/invoices/{id}
func (c *SalesController) HandleGetFactura(w http.ResponseWriter, r *http.Request) {
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

	invoice, err := c.service.GetFactura(r.Context(), invoiceID)
	if err != nil {
		if errors.Is(err, domain.ErrFacturaNotFound) {
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

// HandlePatchPresupuesto handles PATCH /api/v1/sales/quotes/{id}
func (c *SalesController) HandlePatchPresupuesto(w http.ResponseWriter, r *http.Request) {
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
	var req patchPresupuestoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	input := domain.UpdatePresupuestoInput{ID: id}
	if req.ClienteID != nil {
		clientUUID, err := uuid.Parse(*req.ClienteID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid client_id format"})
			return
		}
		input.ClienteID = &clientUUID
	}
	if req.FechaValidez != nil {
		t, err := time.Parse(time.RFC3339, *req.FechaValidez)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid expires_at format, use RFC3339"})
			return
		}
		input.FechaValidez = &t
	}
	if err := c.service.UpdatePresupuesto(r.Context(), input); err != nil {
		if errors.Is(err, domain.ErrPresupuestoNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	// Fetch and return updated quote
	quote, err := c.service.GetPresupuesto(r.Context(), id)
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

// HandlePatchPedido handles PATCH /api/v1/sales/orders/{id}
func (c *SalesController) HandlePatchPedido(w http.ResponseWriter, r *http.Request) {
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
	input := domain.UpdatePedidoInput{ID: id}
	if err := c.service.UpdatePedido(r.Context(), input); err != nil {
		if errors.Is(err, domain.ErrPedidoNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	// Return the current order
	order, err := c.service.GetPedido(r.Context(), id)
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

// HandlePatchAlbaran handles PATCH /api/v1/sales/delivery-notes/{id}
func (c *SalesController) HandlePatchAlbaran(w http.ResponseWriter, r *http.Request) {
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
	input := domain.UpdateAlbaranInput{ID: id}
	if err := c.service.UpdateAlbaran(r.Context(), input); err != nil {
		if errors.Is(err, domain.ErrAlbaranNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	dn, err := c.service.GetAlbaran(r.Context(), id)
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

// HandleCancelPresupuesto handles POST /api/v1/sales/quotes/{id}/cancel
func (c *SalesController) HandleCancelPresupuesto(w http.ResponseWriter, r *http.Request) {
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

	err = c.service.CancelPresupuesto(r.Context(), empID, id)
	if err != nil {
		if errors.Is(err, domain.ErrPresupuestoNotFound) {
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
	quote, err := c.service.GetPresupuesto(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(quote)
}

// HandleCancelPedido handles POST /api/v1/sales/orders/{id}/cancel
func (c *SalesController) HandleCancelPedido(w http.ResponseWriter, r *http.Request) {
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

	err = c.service.CancelPedido(r.Context(), empID, id)
	if err != nil {
		if errors.Is(err, domain.ErrPedidoNotFound) {
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
	order, err := c.service.GetPedido(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(order)
}

// HandleCancelAlbaran handles POST /api/v1/sales/delivery-notes/{id}/cancel
func (c *SalesController) HandleCancelAlbaran(w http.ResponseWriter, r *http.Request) {
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

	err = c.service.CancelAlbaran(r.Context(), empID, id)
	if err != nil {
		if errors.Is(err, domain.ErrAlbaranNotFound) {
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
	dn, err := c.service.GetAlbaran(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(dn)
}

// HandleCancelFactura handles POST /api/v1/sales/invoices/{id}/cancel
func (c *SalesController) HandleCancelFactura(w http.ResponseWriter, r *http.Request) {
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

	err = c.service.CancelFactura(r.Context(), empID, id)
	if err != nil {
		if errors.Is(err, domain.ErrFacturaNotFound) {
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
	invoice, err := c.service.GetFactura(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(invoice)
}

// FR (factura rectificativa) request types
type CreateFacturaRectificativaRequest struct {
	FacturaID  string                             `json:"factura_id"`
	Motivo     string                             `json:"motivo"`
	Lines      []FacturaRectificativaLineaRequest `json:"lines"`
	TerminalID *string                            `json:"terminal_id,omitempty"`
}

type FacturaRectificativaLineaRequest struct {
	ProductoID     string  `json:"producto_id"`
	Cantidad       float64 `json:"cantidad"`
	PrecioUnitario float64 `json:"precio_unitario"`
}

// HandleCreateFacturaRectificativa handles POST /api/v1/sales/facturas-rectificativas
func (c *SalesController) HandleCreateFacturaRectificativa(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var req CreateFacturaRectificativaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	invUUID, err := uuid.Parse(req.FacturaID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid invoice_id"})
		return
	}

	if req.Motivo == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "reason is required"})
		return
	}

	if len(req.Lines) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "at least one line is required"})
		return
	}

	lines := make([]domain.FacturaRectificativaLineaInput, len(req.Lines))
	for i, l := range req.Lines {
		prodUUID, err := uuid.Parse(l.ProductoID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("invalid producto_id at line %d", i)})
			return
		}
		lines[i] = domain.FacturaRectificativaLineaInput{
			ProductoID:     prodUUID,
			Cantidad:       l.Cantidad,
			PrecioUnitario: l.PrecioUnitario,
		}
	}

	var terminalUUIDPtr *uuid.UUID
	if req.TerminalID != nil && *req.TerminalID != "" {
		tUUID, err := uuid.Parse(*req.TerminalID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid terminal_id"})
			return
		}
		terminalUUIDPtr = &tUUID
	}

	fr, err := c.service.CreateFacturaRectificativa(r.Context(), empID, invUUID, req.Motivo, lines, terminalUUIDPtr)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTenantMismatch):
			w.WriteHeader(http.StatusForbidden)
		case errors.Is(err, domain.ErrFacturaAlreadyRectified),
			errors.Is(err, domain.ErrCannotRectifyCancelled),
			errors.Is(err, domain.ErrFacturaNoAlbaran),
			errors.Is(err, domain.ErrProductNotOnFactura),
			errors.Is(err, domain.ErrQuantityExceedsFactura):
			w.WriteHeader(http.StatusConflict)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(fr)
}

// HandleGetFacturasRectificativas handles GET /api/v1/sales/facturas-rectificativas
func (c *SalesController) HandleGetFacturasRectificativas(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	notes, err := c.service.ListFacturasRectificativas(r.Context(), empID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	resp := salesListResponse[domain.FacturaRectificativa]{
		Data:     notes,
		Total:    len(notes),
		Page:     1,
		PageSize: len(notes),
	}
	if resp.PageSize < 1 {
		resp.PageSize = 20
	}

	json.NewEncoder(w).Encode(resp)
}

// HandleGetFacturaRectificativa handles GET /api/v1/sales/facturas-rectificativas/{id}
func (c *SalesController) HandleGetFacturaRectificativa(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/sales/facturas-rectificativas/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	noteID, err := uuid.Parse(parts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid FR ID format"})
		return
	}

	note, err := c.service.GetFacturaRectificativa(r.Context(), noteID)
	if err != nil {
		if errors.Is(err, domain.ErrFacturaRectificativaNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if note.EmpresaID != empID {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "factura rectificativa not found"})
		return
	}

	json.NewEncoder(w).Encode(note)
}
