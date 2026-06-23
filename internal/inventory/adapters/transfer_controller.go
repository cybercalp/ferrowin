package adapters

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ferrowin/internal/inventory/domain"

	"github.com/google/uuid"
)

// TransferController handles HTTP requests for warehouse transfers.
type TransferController struct {
	service *domain.TransferService
}

// NewTransferController creates a new TransferController.
func NewTransferController(service *domain.TransferService) *TransferController {
	return &TransferController{service: service}
}

// ServeHTTP implements http.Handler.
func (c *TransferController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := r.URL.Path

	// Exact match: /api/v1/inventory/transfers
	if path == "/api/v1/inventory/transfers" || path == "/api/v1/inventory/transfers/" {
		switch r.Method {
		case http.MethodGet:
			c.HandleList(w, r)
		case http.MethodPost:
			c.HandleCreate(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	// Sub-path routing: /api/v1/inventory/transfers/{id}/...
	if strings.HasPrefix(path, "/api/v1/inventory/transfers/") {
		parts := strings.Split(strings.TrimPrefix(path, "/api/v1/inventory/transfers/"), "/")

		if len(parts) >= 2 && parts[1] == "lines" {
			if len(parts) == 2 && r.Method == http.MethodPost {
				c.HandleAddLine(w, r) // POST .../{id}/lines
				return
			}
			if len(parts) == 3 && r.Method == http.MethodDelete {
				c.HandleRemoveLine(w, r) // DELETE .../{id}/lines/{lineId}
				return
			}
		}

		if len(parts) == 2 && parts[1] == "process" && r.Method == http.MethodPost {
			c.HandleProcess(w, r) // POST .../{id}/process
			return
		}

		if len(parts) == 1 && r.Method == http.MethodGet {
			c.HandleGetByID(w, r) // GET .../{id}
			return
		}
	}

	http.NotFound(w, r)
}

// request / response types
type createTransferRequest struct {
	EmpresaID string `json:"empresa_id"`
	OrigenID  string `json:"origen_id"`
	DestinoID string `json:"destino_id"`
}

type addLineRequest struct {
	ProductoID string  `json:"producto_id"`
	Cantidad   float64 `json:"cantidad"`
}

type listResponse struct {
	Data     []*domain.TraspasoAlmacen `json:"data"`
	Total    int                       `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}

func (c *TransferController) getTenantID(r *http.Request) (uuid.UUID, error) {
	val := r.Header.Get("X-Empresa-ID")
	if val == "" {
		return uuid.Nil, errors.New("missing X-Empresa-ID header")
	}
	return uuid.Parse(val)
}

func (c *TransferController) writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// HandleCreate processes POST /api/v1/inventory/transfers
func (c *TransferController) HandleCreate(w http.ResponseWriter, r *http.Request) {
	empID, err := c.getTenantID(r)
	if err != nil {
		c.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req createTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	origenID, err := uuid.Parse(req.OrigenID)
	if err != nil {
		c.writeError(w, http.StatusBadRequest, "invalid origen_id format")
		return
	}

	destinoID, err := uuid.Parse(req.DestinoID)
	if err != nil {
		c.writeError(w, http.StatusBadRequest, "invalid destino_id format")
		return
	}

	t, err := c.service.Create(r.Context(), empID, origenID, destinoID)
	if err != nil {
		if errors.Is(err, domain.ErrTransferSameWarehouse) {
			c.writeError(w, http.StatusBadRequest, err.Error())
		} else if errors.Is(err, domain.ErrTransferCrossCompany) {
			c.writeError(w, http.StatusBadRequest, err.Error())
		} else {
			c.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

// HandleGetByID processes GET /api/v1/inventory/transfers/{id}
func (c *TransferController) HandleGetByID(w http.ResponseWriter, r *http.Request) {
	empID, err := c.getTenantID(r)
	if err != nil {
		c.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/inventory/transfers/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		c.writeError(w, http.StatusBadRequest, "missing transfer ID")
		return
	}

	transferID, err := uuid.Parse(parts[0])
	if err != nil {
		c.writeError(w, http.StatusBadRequest, "invalid transfer ID format")
		return
	}

	t, err := c.service.GetByID(r.Context(), transferID)
	if err != nil {
		if errors.Is(err, domain.ErrTransferNotFound) {
			c.writeError(w, http.StatusNotFound, err.Error())
		} else {
			c.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	if t.EmpresaID != empID {
		c.writeError(w, http.StatusNotFound, "transfer not found")
		return
	}

	json.NewEncoder(w).Encode(t)
}

// HandleList processes GET /api/v1/inventory/transfers
func (c *TransferController) HandleList(w http.ResponseWriter, r *http.Request) {
	empID, err := c.getTenantID(r)
	if err != nil {
		c.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	q := r.URL.Query()
	var filter domain.TransferFilter

	if v := q.Get("origen_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			filter.OrigenID = &id
		}
	}
	if v := q.Get("destino_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			filter.DestinoID = &id
		}
	}
	if v := q.Get("estado"); v != "" {
		estado := domain.TraspasoAlmacenEstado(v)
		filter.Estado = &estado
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

	transfers, total, err := c.service.List(r.Context(), empID, filter)
	if err != nil {
		c.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := listResponse{
		Data:     transfers,
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

// HandleAddLine processes POST /api/v1/inventory/transfers/{id}/lines
func (c *TransferController) HandleAddLine(w http.ResponseWriter, r *http.Request) {
	empID, err := c.getTenantID(r)
	if err != nil {
		c.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/inventory/transfers/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		c.writeError(w, http.StatusBadRequest, "missing transfer ID")
		return
	}

	transferID, err := uuid.Parse(parts[0])
	if err != nil {
		c.writeError(w, http.StatusBadRequest, "invalid transfer ID format")
		return
	}

	var req addLineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	productoID, err := uuid.Parse(req.ProductoID)
	if err != nil {
		c.writeError(w, http.StatusBadRequest, "invalid producto_id format")
		return
	}

	t, err := c.service.AddLine(r.Context(), empID, transferID, productoID, req.Cantidad)
	if err != nil {
		if errors.Is(err, domain.ErrTransferNotEditable) {
			c.writeError(w, http.StatusConflict, err.Error())
		} else if errors.Is(err, domain.ErrInvalidQuantity) {
			c.writeError(w, http.StatusBadRequest, err.Error())
		} else if errors.Is(err, domain.ErrTransferNotFound) {
			c.writeError(w, http.StatusNotFound, err.Error())
		} else if errors.Is(err, domain.ErrTransferCrossCompany) {
			c.writeError(w, http.StatusForbidden, err.Error())
		} else {
			c.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	json.NewEncoder(w).Encode(t)
}

// HandleRemoveLine processes DELETE /api/v1/inventory/transfers/{id}/lines/{lineId}
func (c *TransferController) HandleRemoveLine(w http.ResponseWriter, r *http.Request) {
	empID, err := c.getTenantID(r)
	if err != nil {
		c.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/inventory/transfers/"), "/")
	if len(parts) < 3 || parts[0] == "" || parts[2] == "" {
		c.writeError(w, http.StatusBadRequest, "missing transfer ID or line ID")
		return
	}

	transferID, err := uuid.Parse(parts[0])
	if err != nil {
		c.writeError(w, http.StatusBadRequest, "invalid transfer ID format")
		return
	}

	lineID, err := uuid.Parse(parts[2])
	if err != nil {
		c.writeError(w, http.StatusBadRequest, "invalid line ID format")
		return
	}

	t, err := c.service.RemoveLine(r.Context(), empID, transferID, lineID)
	if err != nil {
		if errors.Is(err, domain.ErrTransferNotEditable) {
			c.writeError(w, http.StatusConflict, err.Error())
		} else if errors.Is(err, domain.ErrTransferNotFound) {
			c.writeError(w, http.StatusNotFound, err.Error())
		} else if errors.Is(err, domain.ErrTransferCrossCompany) {
			c.writeError(w, http.StatusForbidden, err.Error())
		} else {
			c.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	json.NewEncoder(w).Encode(t)
}

// HandleProcess processes POST /api/v1/inventory/transfers/{id}/process
func (c *TransferController) HandleProcess(w http.ResponseWriter, r *http.Request) {
	empID, err := c.getTenantID(r)
	if err != nil {
		c.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/inventory/transfers/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		c.writeError(w, http.StatusBadRequest, "missing transfer ID")
		return
	}

	transferID, err := uuid.Parse(parts[0])
	if err != nil {
		c.writeError(w, http.StatusBadRequest, "invalid transfer ID format")
		return
	}

	err = c.service.Process(r.Context(), empID, transferID)
	if err != nil {
		if errors.Is(err, domain.ErrTransferNoLines) {
			c.writeError(w, http.StatusBadRequest, err.Error())
		} else if errors.Is(err, domain.ErrTransferCrossCompany) {
			c.writeError(w, http.StatusForbidden, err.Error())
		} else if errors.Is(err, domain.ErrTransferAlreadyProcessed) {
			c.writeError(w, http.StatusConflict, err.Error())
		} else if errors.Is(err, domain.ErrTransferSameWarehouse) {
			c.writeError(w, http.StatusBadRequest, err.Error())
		} else if errors.Is(err, domain.ErrTransferNotFound) {
			c.writeError(w, http.StatusNotFound, err.Error())
		} else {
			c.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Return updated transfer
	t, err := c.service.GetByID(r.Context(), transferID)
	if err != nil {
		c.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	json.NewEncoder(w).Encode(t)
}
