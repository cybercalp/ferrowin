package adapters

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"ferrowin/internal/billing/domain"
	"ferrowin/internal/billing/ports"

	"github.com/google/uuid"
)

// Patch request types for partial updates
type patchTerminalRequest struct {
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
}

type patchInvoicingSeriesRequest struct {
	TerminalID   *string `json:"terminal_id"`
	Prefix       *string `json:"prefix"`
	NextSequence *int    `json:"next_sequence"`
}

// BillingController handles HTTP requests for billing CRUD operations.
type BillingController struct {
	terminalRepo ports.TerminalRepository
	seriesRepo   ports.InvoicingSeriesRepository
}

// NewBillingController creates a new BillingController.
func NewBillingController(terminalRepo ports.TerminalRepository, seriesRepo ports.InvoicingSeriesRepository) *BillingController {
	return &BillingController{
		terminalRepo: terminalRepo,
		seriesRepo:   seriesRepo,
	}
}

func (c *BillingController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := r.URL.Path

	// --- Terminals ---
	if path == "/api/v1/billing/terminals" {
		switch r.Method {
		case http.MethodGet:
			c.handleListTerminals(w, r)
		case http.MethodPost:
			c.handleCreateTerminal(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}
	if strings.HasPrefix(path, "/api/v1/billing/terminals/") {
		id, err := parseIDFromPath("/api/v1/billing/terminals/", path)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid ID")
			return
		}
		switch r.Method {
		case http.MethodGet:
			c.handleGetTerminal(w, r, id)
		case http.MethodPatch:
			c.handlePatchTerminal(w, r, id)
		case http.MethodDelete:
			c.handleDeleteTerminal(w, r, id)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	// --- Invoicing Series ---
	if path == "/api/v1/billing/series" {
		switch r.Method {
		case http.MethodGet:
			c.handleListSeries(w, r)
		case http.MethodPost:
			c.handleCreateSeries(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}
	if strings.HasPrefix(path, "/api/v1/billing/series/by-terminal/") {
		terminalID, err := parseIDFromPath("/api/v1/billing/series/by-terminal/", path)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid terminal ID")
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		c.handleGetSeriesByTerminal(w, r, terminalID)
		return
	}
	if strings.HasPrefix(path, "/api/v1/billing/series/") {
		id, err := parseIDFromPath("/api/v1/billing/series/", path)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid ID")
			return
		}
		switch r.Method {
		case http.MethodGet:
			c.handleGetSeries(w, r, id)
		case http.MethodPatch:
			c.handlePatchSeries(w, r, id)
		case http.MethodDelete:
			c.handleDeleteSeries(w, r, id)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	http.NotFound(w, r)
}

// --- helper types ---

type billingListResponse struct {
	Data  interface{} `json:"data"`
	Total int         `json:"total"`
}

// --- helpers ---

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

// --- Terminal handlers ---

func (c *BillingController) handleListTerminals(w http.ResponseWriter, r *http.Request) {
	terminals, err := c.terminalRepo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, billingListResponse{Data: terminals, Total: len(terminals)})
}

func (c *BillingController) handleCreateTerminal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	terminal := &domain.Terminal{
		ID:       uuid.New(),
		Name:     req.Name,
		IsActive: true,
	}
	if err := c.terminalRepo.Save(r.Context(), terminal); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, terminal)
}

func (c *BillingController) handleGetTerminal(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	terminal, err := c.terminalRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if terminal == nil {
		writeError(w, http.StatusNotFound, "terminal not found")
		return
	}
	writeJSON(w, http.StatusOK, terminal)
}

func (c *BillingController) handlePatchTerminal(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	current, err := c.terminalRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if current == nil {
		writeError(w, http.StatusNotFound, "terminal not found")
		return
	}

	var req patchTerminalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name != nil {
		current.Name = *req.Name
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}

	if err := c.terminalRepo.Save(r.Context(), current); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, current)
}

func (c *BillingController) handleDeleteTerminal(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	if err := c.terminalRepo.Delete(r.Context(), id); err != nil {
		if err.Error() == "terminal not found" {
			writeError(w, http.StatusNotFound, "terminal not found")
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Invoicing Series handlers ---

func (c *BillingController) handleListSeries(w http.ResponseWriter, r *http.Request) {
	series, err := c.seriesRepo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, billingListResponse{Data: series, Total: len(series)})
}

func (c *BillingController) handleCreateSeries(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TerminalID string `json:"terminal_id"`
		Prefix     string `json:"prefix"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Prefix == "" {
		writeError(w, http.StatusBadRequest, "prefix is required")
		return
	}
	terminalID, err := uuid.Parse(req.TerminalID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid terminal_id format")
		return
	}

	// Verify terminal exists
	terminal, err := c.terminalRepo.GetByID(r.Context(), terminalID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if terminal == nil {
		writeError(w, http.StatusBadRequest, "terminal not found")
		return
	}

	series := &domain.InvoicingSeries{
		ID:           uuid.New(),
		TerminalID:   terminalID,
		Prefix:       req.Prefix,
		NextSequence: 1,
	}
	if err := c.seriesRepo.Save(r.Context(), series); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, series)
}

func (c *BillingController) handleGetSeries(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	series, err := c.seriesRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if series == nil {
		writeError(w, http.StatusNotFound, "invoicing series not found")
		return
	}
	writeJSON(w, http.StatusOK, series)
}

func (c *BillingController) handleGetSeriesByTerminal(w http.ResponseWriter, r *http.Request, terminalID uuid.UUID) {
	series, err := c.seriesRepo.GetByTerminalID(r.Context(), terminalID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if series == nil {
		writeError(w, http.StatusNotFound, "invoicing series not found for terminal")
		return
	}
	writeJSON(w, http.StatusOK, series)
}

func (c *BillingController) handlePatchSeries(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	current, err := c.seriesRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if current == nil {
		writeError(w, http.StatusNotFound, "invoicing series not found")
		return
	}

	var req patchInvoicingSeriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TerminalID != nil {
		terminalID, err := uuid.Parse(*req.TerminalID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid terminal_id format")
			return
		}
		current.TerminalID = terminalID
	}
	if req.Prefix != nil {
		current.Prefix = *req.Prefix
	}
	if req.NextSequence != nil {
		current.NextSequence = *req.NextSequence
	}

	if err := c.seriesRepo.Save(r.Context(), current); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, current)
}

func (c *BillingController) handleDeleteSeries(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	if err := c.seriesRepo.Delete(r.Context(), id); err != nil {
		if err.Error() == "invoicing series not found" {
			writeError(w, http.StatusNotFound, "invoicing series not found")
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
