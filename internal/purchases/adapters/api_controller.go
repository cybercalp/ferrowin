package adapters

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"ferrowin/internal/purchases/domain"
	"github.com/google/uuid"
)

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

	if r.URL.Path == "/api/v1/purchases/companies" {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleCreateCompany(w, r)
		return
	}

	if r.URL.Path == "/api/v1/purchases/warehouses" {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleCreateWarehouse(w, r)
		return
	}

	if r.URL.Path == "/api/v1/purchases/suppliers" {
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

	if r.URL.Path == "/api/v1/purchases/orders" {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleCreatePurchaseOrder(w, r)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/api/v1/purchases/orders/approve/") {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleApprovePurchaseOrder(w, r)
		return
	}

	if r.URL.Path == "/api/v1/purchases/receipts" {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleCreatePurchaseReceipt(w, r)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/api/v1/purchases/receipts/process/") {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.HandleProcessPurchaseReceipt(w, r)
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
