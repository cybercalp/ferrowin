package adapters

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ferrowin/internal/purchases/domain"
	"github.com/google/uuid"
)

// Patch request types for partial updates
type patchCompanyRequest struct {
	RazonSocial *string `json:"razon_social"`
	NIF         *string `json:"nif"`
}

type patchWarehouseRequest struct {
	Name   *string `json:"name"`
	Active *bool   `json:"active"`
}

type patchSupplierRequest struct {
	RazonSocial *string `json:"razon_social"`
	CIF         *string `json:"cif"`
	Email       *string `json:"email"`
	Telefono    *string `json:"telefono"`
	Activo      *bool   `json:"activo"`
}

type cancelRequest struct {
	Reason string `json:"reason"`
}

type PurchaseController struct {
	service *domain.PurchaseService
}

func NewPurchaseController(service *domain.PurchaseService) *PurchaseController {
	return &PurchaseController{
		service: service,
	}
}

func (c *PurchaseController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := r.URL.Path

	// Exact matches
	if path == "/api/v1/purchases/companies" {
		switch r.Method {
		case http.MethodGet:
			c.HandleGetCompanies(w, r)
		case http.MethodPost:
			c.HandleCreateCompany(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	if strings.HasPrefix(path, "/api/v1/purchases/companies/") {
		if r.Method == http.MethodPatch {
			c.HandlePatchCompany(w, r)
			return
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if path == "/api/v1/purchases/warehouses" {
		switch r.Method {
		case http.MethodGet:
			c.HandleGetWarehouses(w, r)
		case http.MethodPost:
			c.HandleCreateWarehouse(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	if strings.HasPrefix(path, "/api/v1/purchases/warehouses/") {
		if r.Method == http.MethodPatch {
			c.HandlePatchWarehouse(w, r)
			return
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if path == "/api/v1/purchases/suppliers" {
		switch r.Method {
		case http.MethodPost:
			c.HandleCreateSupplier(w, r)
		case http.MethodGet:
			c.HandleGetSuppliers(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	if strings.HasPrefix(path, "/api/v1/purchases/suppliers/") {
		if r.Method == http.MethodPatch {
			c.HandlePatchSupplier(w, r)
			return
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Purchase orders routing
	if path == "/api/v1/purchases/orders" {
		switch r.Method {
		case http.MethodGet:
			c.HandleGetPurchaseOrders(w, r)
		case http.MethodPost:
			c.HandleCreatePurchaseOrder(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	if strings.HasPrefix(path, "/api/v1/purchases/orders/") {
		// POST /api/v1/purchases/orders/approve/{id}
		if strings.HasPrefix(path, "/api/v1/purchases/orders/approve/") {
			if r.Method != http.MethodPost {
				http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
				return
			}
			c.HandleApprovePurchaseOrder(w, r)
			return
		}
		// POST /api/v1/purchases/orders/{id}/cancel
		parts := strings.Split(strings.TrimPrefix(path, "/api/v1/purchases/orders/"), "/")
		if len(parts) >= 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
			c.HandleCancelPurchaseOrder(w, r)
			return
		}
		// GET /api/v1/purchases/orders/{id}
		if r.Method == http.MethodGet {
			c.HandleGetPurchaseOrder(w, r)
			return
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Purchase receipts routing
	if path == "/api/v1/purchases/receipts" {
		switch r.Method {
		case http.MethodGet:
			c.HandleGetPurchaseReceipts(w, r)
		case http.MethodPost:
			c.HandleCreatePurchaseReceipt(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	if strings.HasPrefix(path, "/api/v1/purchases/receipts/") {
		// POST /api/v1/purchases/receipts/process/{id}
		if strings.HasPrefix(path, "/api/v1/purchases/receipts/process/") {
			if r.Method != http.MethodPost {
				http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
				return
			}
			c.HandleProcessPurchaseReceipt(w, r)
			return
		}
		// POST /api/v1/purchases/receipts/{id}/cancel
		parts := strings.Split(strings.TrimPrefix(path, "/api/v1/purchases/receipts/"), "/")
		if len(parts) >= 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
			c.HandleCancelPurchaseReceipt(w, r)
			return
		}
		// GET /api/v1/purchases/receipts/{id}
		if r.Method == http.MethodGet {
			c.HandleGetPurchaseReceipt(w, r)
			return
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	http.NotFound(w, r)
}

// Request and Response structs
type CreateCompanyRequest struct {
	RazonSocial string `json:"razon_social"`
	NIF         string `json:"nif"`
}

type CreateWarehouseRequest struct {
	Name string `json:"name"`
}

type CreateSupplierRequest struct {
	RazonSocial string `json:"razon_social"`
	CIF         string `json:"cif"`
	Email       string `json:"email"`
	Telefono    string `json:"telefono"`
	Direccion   string `json:"direccion"`
}

type CreateOrderRequest struct {
	ProveedorID  string             `json:"proveedor_id"`
	NumeroPedido string             `json:"numero_pedido"`
	Lineas       []OrderLineRequest `json:"lineas"`
}

type OrderLineRequest struct {
	ProductoID     string  `json:"producto_id"`
	Cantidad       float64 `json:"cantidad"`
	PrecioUnitario float64 `json:"precio_unitario"`
}

type CreateReceiptRequest struct {
	ProveedorID    string               `json:"proveedor_id"`
	PedidoCompraID *string              `json:"pedido_compra_id,omitempty"`
	NumeroAlbaran  string               `json:"numero_albaran"`
	WarehouseID    string               `json:"warehouse_id"`
	Lineas         []ReceiptLineRequest `json:"lineas"`
}

type ReceiptLineRequest struct {
	ProductoID     string  `json:"producto_id"`
	Cantidad       float64 `json:"cantidad"`
	PrecioUnitario float64 `json:"precio_unitario"`
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

func (c *PurchaseController) HandleCreateCompany(w http.ResponseWriter, r *http.Request) {
	var req CreateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	comp, err := c.service.CreateCompany(r.Context(), req.RazonSocial, req.NIF)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comp)
}

func (c *PurchaseController) HandleCreateWarehouse(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var req CreateWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	wh, err := c.service.CreateWarehouse(r.Context(), empID, req.Name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(wh)
}

func (c *PurchaseController) HandleCreateSupplier(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var req CreateSupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	prov, err := c.service.CreateSupplier(r.Context(), empID, req.RazonSocial, req.CIF, req.Email, req.Telefono, req.Direccion)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(prov)
}

func (c *PurchaseController) HandleGetSuppliers(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	suppliers, err := c.service.GetSuppliers(r.Context(), empID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(suppliers)
}

func (c *PurchaseController) HandleCreatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
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

	provUUID, err := uuid.Parse(req.ProveedorID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid proveedor_id"})
		return
	}

	lines := make([]domain.PedidoCompraLinea, len(req.Lineas))
	for i, l := range req.Lineas {
		prodUUID, err := uuid.Parse(l.ProductoID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("invalid producto_id at line %d", i)})
			return
		}
		lines[i] = domain.PedidoCompraLinea{
			ProductoID:     prodUUID,
			Cantidad:       l.Cantidad,
			PrecioUnitario: l.PrecioUnitario,
		}
	}

	po, err := c.service.CreatePurchaseOrder(r.Context(), empID, provUUID, req.NumeroPedido, lines)
	if err != nil {
		if errors.Is(err, domain.ErrTenantMismatch) {
			w.WriteHeader(http.StatusForbidden)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(po)
}

func (c *PurchaseController) HandleApprovePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	
	// URL: /api/v1/purchases/orders/approve/{id}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 7 || pathParts[6] == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing order ID in URL"})
		return
	}

	orderIDStr := pathParts[6]
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid order ID format"})
		return
	}

	err = c.service.ApprovePurchaseOrder(r.Context(), empID, orderID)
	if err != nil {
		if errors.Is(err, domain.ErrPurchaseOrderNotFound) {
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

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "purchase order approved"})
}

func (c *PurchaseController) HandleCreatePurchaseReceipt(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var req CreateReceiptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	provUUID, err := uuid.Parse(req.ProveedorID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid proveedor_id"})
		return
	}

	whUUID, err := uuid.Parse(req.WarehouseID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid warehouse_id"})
		return
	}

	var poUUIDPtr *uuid.UUID
	if req.PedidoCompraID != nil && *req.PedidoCompraID != "" {
		poUUID, err := uuid.Parse(*req.PedidoCompraID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid pedido_compra_id"})
			return
		}
		poUUIDPtr = &poUUID
	}

	lines := make([]domain.RecepcionCompraLinea, len(req.Lineas))
	for i, l := range req.Lineas {
		prodUUID, err := uuid.Parse(l.ProductoID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("invalid producto_id at line %d", i)})
			return
		}
		lines[i] = domain.RecepcionCompraLinea{
			ProductoID:     prodUUID,
			Cantidad:       l.Cantidad,
			PrecioUnitario: l.PrecioUnitario,
		}
	}

	rc, err := c.service.CreatePurchaseReceipt(r.Context(), empID, provUUID, poUUIDPtr, req.NumeroAlbaran, whUUID, lines)
	if err != nil {
		if errors.Is(err, domain.ErrTenantMismatch) {
			w.WriteHeader(http.StatusForbidden)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rc)
}

func (c *PurchaseController) HandleProcessPurchaseReceipt(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// URL: /api/v1/purchases/receipts/process/{id}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 7 || pathParts[6] == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing receipt ID in URL"})
		return
	}

	receiptIDStr := pathParts[6]
	receiptID, err := uuid.Parse(receiptIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid receipt ID format"})
		return
	}

	err = c.service.ProcessPurchaseReceipt(r.Context(), empID, receiptID)
	if err != nil {
		if errors.Is(err, domain.ErrPurchaseReceiptNotFound) {
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

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "purchase receipt processed, stock updated"})
}

// List response types
type purchaseOrderListResponse struct {
	Data     []*domain.PedidoCompra `json:"data"`
	Total    int                    `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

type purchaseReceiptListResponse struct {
	Data     []*domain.RecepcionCompra `json:"data"`
	Total    int                       `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}

// HandleGetCompanies handles GET /api/v1/purchases/companies
func (c *PurchaseController) HandleGetCompanies(w http.ResponseWriter, r *http.Request) {
	companies, err := c.service.GetCompanies(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(companies)
}

// HandleGetWarehouses handles GET /api/v1/purchases/warehouses
func (c *PurchaseController) HandleGetWarehouses(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	warehouses, err := c.service.GetAllWarehouses(r.Context(), empID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(warehouses)
}

// HandleGetPurchaseOrders handles GET /api/v1/purchases/orders
func (c *PurchaseController) HandleGetPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	q := r.URL.Query()
	var filter domain.PurchaseOrderFilter

	if v := q.Get("estado"); v != "" {
		filter.Estado = &v
	}
	if v := q.Get("supplier_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			filter.ProveedorID = &id
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

	orders, total, err := c.service.ListPurchaseOrders(r.Context(), empID, filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	resp := purchaseOrderListResponse{
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

// HandleGetPurchaseOrder handles GET /api/v1/purchases/orders/{id}
func (c *PurchaseController) HandleGetPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/purchases/orders/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing order ID"})
		return
	}

	orderID, err := uuid.Parse(parts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid order ID format"})
		return
	}

	po, err := c.service.GetPurchaseOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, domain.ErrPurchaseOrderNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if po.EmpresaID != empID {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "purchase order not found"})
		return
	}

	json.NewEncoder(w).Encode(po)
}

// HandleGetPurchaseReceipts handles GET /api/v1/purchases/receipts
func (c *PurchaseController) HandleGetPurchaseReceipts(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	q := r.URL.Query()
	var filter domain.PurchaseReceiptFilter

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

	receipts, total, err := c.service.ListPurchaseReceipts(r.Context(), empID, filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	resp := purchaseReceiptListResponse{
		Data:     receipts,
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

// HandleGetPurchaseReceipt handles GET /api/v1/purchases/receipts/{id}
func (c *PurchaseController) HandleGetPurchaseReceipt(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/purchases/receipts/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing receipt ID"})
		return
	}

	receiptID, err := uuid.Parse(parts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid receipt ID format"})
		return
	}

	rc, err := c.service.GetPurchaseReceipt(r.Context(), receiptID)
	if err != nil {
		if errors.Is(err, domain.ErrPurchaseReceiptNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if rc.EmpresaID != empID {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "purchase receipt not found"})
		return
	}

	json.NewEncoder(w).Encode(rc)
}

// parseIDFromPath extracts the ID from paths like /api/v1/purchases/companies/{id}
func parseIDFromPath(prefix, path string) (uuid.UUID, error) {
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(parts) == 0 || parts[0] == "" {
		return uuid.Nil, fmt.Errorf("missing ID in URL")
	}
	return uuid.Parse(parts[0])
}

// HandlePatchCompany handles PATCH /api/v1/purchases/companies/{id}
func (c *PurchaseController) HandlePatchCompany(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	id, err := parseIDFromPath("/api/v1/purchases/companies/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid company ID"})
		return
	}
	var req patchCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	input := domain.UpdateCompanyInput{ID: id}
	if req.RazonSocial != nil {
		input.RazonSocial = req.RazonSocial
	}
	if req.NIF != nil {
		input.NIF = req.NIF
	}
	if err := c.service.UpdateCompany(r.Context(), input); err != nil {
		if errors.Is(err, domain.ErrCompanyNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	// Return all companies (the collection includes the updated one)
	companies, err := c.service.GetCompanies(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var updated *domain.Empresa
	for _, comp := range companies {
		if comp.ID == id {
			updated = comp
			break
		}
		_ = empID // tenant check not needed for company (global read)
	}
	if updated == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "company not found"})
		return
	}
	json.NewEncoder(w).Encode(updated)
}

// HandlePatchWarehouse handles PATCH /api/v1/purchases/warehouses/{id}
func (c *PurchaseController) HandlePatchWarehouse(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	id, err := parseIDFromPath("/api/v1/purchases/warehouses/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid warehouse ID"})
		return
	}
	var req patchWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	input := domain.UpdateWarehouseInput{ID: id}
	if req.Name != nil {
		input.Name = req.Name
	}
	if req.Active != nil {
		input.Active = req.Active
	}
	if err := c.service.UpdateWarehouse(r.Context(), input); err != nil {
		if errors.Is(err, domain.ErrWarehouseNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	// Fetch and return updated warehouse
	wh, err := c.service.GetAllWarehouses(r.Context(), empID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var updated *domain.Warehouse
	for _, w := range wh {
		if w.ID == id {
			updated = w
			break
		}
	}
	if updated == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "warehouse not found"})
		return
	}
	json.NewEncoder(w).Encode(updated)
}

// HandlePatchSupplier handles PATCH /api/v1/purchases/suppliers/{id}
func (c *PurchaseController) HandlePatchSupplier(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	id, err := parseIDFromPath("/api/v1/purchases/suppliers/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid supplier ID"})
		return
	}
	var req patchSupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	input := domain.UpdateSupplierInput{ID: id}
	if req.RazonSocial != nil {
		input.RazonSocial = req.RazonSocial
	}
	if req.CIF != nil {
		input.CIF = req.CIF
	}
	if req.Email != nil {
		input.Email = req.Email
	}
	if req.Telefono != nil {
		input.Telefono = req.Telefono
	}
	if req.Activo != nil {
		input.Activo = req.Activo
	}
	if err := c.service.UpdateSupplier(r.Context(), input); err != nil {
		if errors.Is(err, domain.ErrSupplierNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	// Fetch and return updated supplier
	suppliers, err := c.service.GetSuppliers(r.Context(), empID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var updated *domain.Proveedor
	for _, s := range suppliers {
		if s.ID == id {
			updated = s
			break
		}
	}
	if updated == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "supplier not found"})
		return
	}
	json.NewEncoder(w).Encode(updated)
}

// HandleCancelPurchaseOrder handles POST /api/v1/purchases/orders/{id}/cancel
func (c *PurchaseController) HandleCancelPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	// Parse {id} from URL: /api/v1/purchases/orders/{id}/cancel
	prefix := "/api/v1/purchases/orders/"
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, prefix), "/")
	if len(parts) < 1 || parts[0] == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing order ID"})
		return
	}
	orderID, err := uuid.Parse(parts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid order ID format"})
		return
	}
	// Read optional reason
	var cancelReq cancelRequest
	json.NewDecoder(r.Body).Decode(&cancelReq) // ignore decode error, reason is optional

	err = c.service.CancelPurchaseOrder(r.Context(), empID, orderID)
	if err != nil {
		if errors.Is(err, domain.ErrPurchaseOrderNotFound) {
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
	// Return the updated order
	po, err := c.service.GetPurchaseOrder(r.Context(), orderID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(po)
}

// HandleCancelPurchaseReceipt handles POST /api/v1/purchases/receipts/{id}/cancel
func (c *PurchaseController) HandleCancelPurchaseReceipt(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	prefix := "/api/v1/purchases/receipts/"
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, prefix), "/")
	if len(parts) < 1 || parts[0] == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing receipt ID"})
		return
	}
	receiptID, err := uuid.Parse(parts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid receipt ID format"})
		return
	}
	// Read optional reason
	var cancelReq cancelRequest
	json.NewDecoder(r.Body).Decode(&cancelReq)

	err = c.service.CancelPurchaseReceipt(r.Context(), empID, receiptID)
	if err != nil {
		if errors.Is(err, domain.ErrPurchaseReceiptNotFound) {
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
	// Return the updated receipt
	rc, err := c.service.GetPurchaseReceipt(r.Context(), receiptID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(rc)
}
