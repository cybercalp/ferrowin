package adapters

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"ferrowin/internal/catalog/domain"
	"github.com/google/uuid"
)

// Patch request types for partial updates
type patchTipoIVARequest struct {
	Nombre     *string  `json:"nombre"`
	Porcentaje *float64 `json:"porcentaje"`
}

type patchFamiliaRequest struct {
	Nombre *string `json:"nombre"`
}

type patchProductoRequest struct {
	Codigo      *string  `json:"codigo"`
	Nombre      *string  `json:"nombre"`
	PrecioVenta *float64 `json:"precio_venta"`
	FamiliaID   *string  `json:"familia_id"`
	TipoIvaID   *string  `json:"tipo_iva_id"`
}

type patchClienteRequest struct {
	RazonSocial *string `json:"razon_social"`
	NIF         *string `json:"nif"`
	Email       *string `json:"email"`
	Telefono    *string `json:"telefono"`
}

// catalogController handles HTTP requests for catalog CRUD operations.
type CatalogController struct {
	service *domain.CatalogService
}

// NewCatalogController creates a new CatalogController.
func NewCatalogController(service *domain.CatalogService) *CatalogController {
	return &CatalogController{service: service}
}

func (c *CatalogController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := r.URL.Path

	// --- Tipos IVA ---
	if path == "/api/v1/catalog/tipos-iva" {
		switch r.Method {
		case http.MethodGet:
			c.handleListTiposIVA(w, r)
		case http.MethodPost:
			c.handleCreateTipoIVA(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}
	if strings.HasPrefix(path, "/api/v1/catalog/tipos-iva/") {
		id, err := parseIDFromPath("/api/v1/catalog/tipos-iva/", path)
		if err != nil {
			http.Error(w, `{"error":"invalid ID"}`, http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			c.handleGetTipoIVA(w, r, id)
		case http.MethodPatch:
			c.handlePatchTipoIVA(w, r, id)
		case http.MethodDelete:
			c.handleDeleteTipoIVA(w, r, id)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	// --- Familias ---
	if path == "/api/v1/catalog/familias" {
		switch r.Method {
		case http.MethodGet:
			c.handleListFamilias(w, r)
		case http.MethodPost:
			c.handleCreateFamilia(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}
	if strings.HasPrefix(path, "/api/v1/catalog/familias/") {
		id, err := parseIDFromPath("/api/v1/catalog/familias/", path)
		if err != nil {
			http.Error(w, `{"error":"invalid ID"}`, http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			c.handleGetFamilia(w, r, id)
		case http.MethodPatch:
			c.handlePatchFamilia(w, r, id)
		case http.MethodDelete:
			c.handleDeleteFamilia(w, r, id)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	// --- Productos ---
	if path == "/api/v1/catalog/productos" {
		switch r.Method {
		case http.MethodGet:
			c.handleListProductos(w, r)
		case http.MethodPost:
			c.handleCreateProducto(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}
	if strings.HasPrefix(path, "/api/v1/catalog/productos/") {
		id, err := parseIDFromPath("/api/v1/catalog/productos/", path)
		if err != nil {
			http.Error(w, `{"error":"invalid ID"}`, http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			c.handleGetProducto(w, r, id)
		case http.MethodPatch:
			c.handlePatchProducto(w, r, id)
		case http.MethodDelete:
			c.handleDeleteProducto(w, r, id)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	// --- Clientes ---
	if path == "/api/v1/catalog/clientes" {
		switch r.Method {
		case http.MethodGet:
			c.handleListClientes(w, r)
		case http.MethodPost:
			c.handleCreateCliente(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}
	if strings.HasPrefix(path, "/api/v1/catalog/clientes/") {
		id, err := parseIDFromPath("/api/v1/catalog/clientes/", path)
		if err != nil {
			http.Error(w, `{"error":"invalid ID"}`, http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			c.handleGetCliente(w, r, id)
		case http.MethodPatch:
			c.handlePatchCliente(w, r, id)
		case http.MethodDelete:
			c.handleDeleteCliente(w, r, id)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	http.NotFound(w, r)
}

// --- helper types ---

type catalogListResponse struct {
	Data  interface{} `json:"data"`
	Total int         `json:"total"`
}

// --- helpers ---

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

func parseIDFromPath(prefix, path string) (uuid.UUID, error) {
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(parts) == 0 || parts[0] == "" {
		return uuid.Nil, errors.New("missing ID in URL")
	}
	return uuid.Parse(parts[0])
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// --- Tipos IVA handlers ---

func (c *CatalogController) handleListTiposIVA(w http.ResponseWriter, r *http.Request) {
	items, err := c.service.ListTiposIVA(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, catalogListResponse{Data: items, Total: len(items)})
}

func (c *CatalogController) handleCreateTipoIVA(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Nombre     string  `json:"nombre"`
		Porcentaje float64 `json:"porcentaje"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	item, err := c.service.CreateTipoIVA(r.Context(), req.Nombre, req.Porcentaje)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (c *CatalogController) handleGetTipoIVA(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	item, err := c.service.GetTipoIVA(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTipoIVANotFound) {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (c *CatalogController) handlePatchTipoIVA(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	current, err := c.service.GetTipoIVA(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTipoIVANotFound) {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	var req patchTipoIVARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Nombre != nil {
		current.Nombre = *req.Nombre
	}
	if req.Porcentaje != nil {
		current.Porcentaje = *req.Porcentaje
	}

	if err := c.service.UpdateTipoIVA(r.Context(), current); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, current)
}

func (c *CatalogController) handleDeleteTipoIVA(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	if err := c.service.DeleteTipoIVA(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrTipoIVANotFound) || errors.Is(err, domain.ErrValidation) {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Familias handlers ---

func (c *CatalogController) handleListFamilias(w http.ResponseWriter, r *http.Request) {
	items, err := c.service.ListFamilias(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, catalogListResponse{Data: items, Total: len(items)})
}

func (c *CatalogController) handleCreateFamilia(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Nombre string `json:"nombre"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	item, err := c.service.CreateFamilia(r.Context(), req.Nombre)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (c *CatalogController) handleGetFamilia(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	item, err := c.service.GetFamilia(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrFamiliaNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (c *CatalogController) handlePatchFamilia(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	current, err := c.service.GetFamilia(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrFamiliaNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	var req patchFamiliaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Nombre != nil {
		current.Nombre = *req.Nombre
	}

	if err := c.service.UpdateFamilia(r.Context(), current); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, current)
}

func (c *CatalogController) handleDeleteFamilia(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	if err := c.service.DeleteFamilia(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrFamiliaNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Productos handlers ---

func (c *CatalogController) handleListProductos(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var familiaID, tipoIvaID *uuid.UUID
	if v := q.Get("familia_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			familiaID = &id
		}
	}
	if v := q.Get("tipo_iva_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			tipoIvaID = &id
		}
	}

	items, err := c.service.ListProductos(r.Context(), familiaID, tipoIvaID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, catalogListResponse{Data: items, Total: len(items)})
}

func (c *CatalogController) handleCreateProducto(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Codigo      string  `json:"codigo"`
		Nombre      string  `json:"nombre"`
		PrecioVenta float64 `json:"precio_venta"`
		FamiliaID   *string `json:"familia_id"`
		TipoIvaID   string  `json:"tipo_iva_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tipoIvaUUID, err := uuid.Parse(req.TipoIvaID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tipo_iva_id format")
		return
	}

	var familiaIDPtr *uuid.UUID
	if req.FamiliaID != nil && *req.FamiliaID != "" {
		fid, err := uuid.Parse(*req.FamiliaID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid familia_id format")
			return
		}
		familiaIDPtr = &fid
	}

	item, err := c.service.CreateProducto(r.Context(), req.Codigo, req.Nombre, req.PrecioVenta, familiaIDPtr, tipoIvaUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (c *CatalogController) handleGetProducto(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	item, err := c.service.GetProducto(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrProductoNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (c *CatalogController) handlePatchProducto(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	current, err := c.service.GetProducto(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrProductoNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	var req patchProductoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Codigo != nil {
		current.Codigo = *req.Codigo
	}
	if req.Nombre != nil {
		current.Nombre = *req.Nombre
	}
	if req.PrecioVenta != nil {
		current.PrecioVenta = *req.PrecioVenta
	}
	if req.FamiliaID != nil {
		if *req.FamiliaID == "" {
			current.FamiliaID = nil
		} else {
			fid, err := uuid.Parse(*req.FamiliaID)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid familia_id format")
				return
			}
			current.FamiliaID = &fid
		}
	}
	if req.TipoIvaID != nil {
		tid, err := uuid.Parse(*req.TipoIvaID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid tipo_iva_id format")
			return
		}
		current.TipoIvaID = tid
	}

	if err := c.service.UpdateProducto(r.Context(), current); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, current)
}

func (c *CatalogController) handleDeleteProducto(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	if err := c.service.DeleteProducto(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrProductoNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Clientes handlers ---

func (c *CatalogController) handleListClientes(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	items, err := c.service.ListClientes(r.Context(), empID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, catalogListResponse{Data: items, Total: len(items)})
}

func (c *CatalogController) handleCreateCliente(w http.ResponseWriter, r *http.Request) {
	empID, err := getTenantID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req struct {
		RazonSocial string  `json:"razon_social"`
		NIF         string  `json:"nif"`
		Email       *string `json:"email"`
		Telefono    *string `json:"telefono"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := c.service.CreateCliente(r.Context(), empID, req.RazonSocial, req.NIF, req.Email, req.Telefono)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (c *CatalogController) handleGetCliente(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	empID, err := getTenantID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	item, err := c.service.GetCliente(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrClienteNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Tenant isolation check
	if item.EmpresaID != empID {
		writeError(w, http.StatusNotFound, "cliente not found")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (c *CatalogController) handlePatchCliente(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	empID, err := getTenantID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	current, err := c.service.GetCliente(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrClienteNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Tenant isolation check
	if current.EmpresaID != empID {
		writeError(w, http.StatusNotFound, "cliente not found")
		return
	}

	var req patchClienteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.RazonSocial != nil {
		current.RazonSocial = *req.RazonSocial
	}
	if req.NIF != nil {
		current.NIF = *req.NIF
	}
	if req.Email != nil {
		current.Email = req.Email
	}
	if req.Telefono != nil {
		current.Telefono = req.Telefono
	}

	if err := c.service.UpdateCliente(r.Context(), current); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, current)
}

func (c *CatalogController) handleDeleteCliente(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	empID, err := getTenantID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Fetch first to verify tenant isolation
	current, err := c.service.GetCliente(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrClienteNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	if current.EmpresaID != empID {
		writeError(w, http.StatusNotFound, "cliente not found")
		return
	}

	if err := c.service.DeleteCliente(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
