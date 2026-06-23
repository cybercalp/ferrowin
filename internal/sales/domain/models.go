package domain

import (
	"time"

	"github.com/google/uuid"
)

// Status constants for Quote, Order, DeliveryNote and Invoice
const (
	StatusDraft     = "Draft"
	StatusApproved  = "Approved"
	StatusConverted = "Converted"
	StatusCancelled = "Cancelled"
	StatusIssued    = "Issued"
	StatusProcessed = "Processed"
)

// Quote represents a sales quote.
type Quote struct {
	ID        uuid.UUID   `json:"id"`
	EmpresaID uuid.UUID   `json:"empresa_id"`
	ClientID  uuid.UUID   `json:"client_id"`
	Total     float64     `json:"total"`
	Status    string      `json:"status"` // Draft, Approved, Converted, Cancelled
	ExpiresAt time.Time   `json:"expires_at"`
	CreatedAt time.Time   `json:"created_at"`
	Lineas    []QuoteLine `json:"lineas,omitempty"`
}

type QuoteLine struct {
	ID             uuid.UUID `json:"id"`
	QuoteID        uuid.UUID `json:"quote_id"`
	ProductoID     uuid.UUID `json:"producto_id"`
	Cantidad       float64   `json:"cantidad"`
	PrecioUnitario float64   `json:"precio_unitario"`
	CosteUnitario  float64   `json:"coste_unitario"`
}

// Order represents a sales order.
type Order struct {
	ID        uuid.UUID   `json:"id"`
	EmpresaID uuid.UUID   `json:"empresa_id"`
	QuoteID   *uuid.UUID  `json:"quote_id,omitempty"`
	Total     float64     `json:"total"`
	Status    string      `json:"status"` // Draft, Approved, Converted, Cancelled
	CreatedAt time.Time   `json:"created_at"`
	Lineas    []OrderLine `json:"lineas,omitempty"`
}

type OrderLine struct {
	ID             uuid.UUID `json:"id"`
	OrderID        uuid.UUID `json:"order_id"`
	ProductoID     uuid.UUID `json:"producto_id"`
	Cantidad       float64   `json:"cantidad"`
	PrecioUnitario float64   `json:"precio_unitario"`
}

// DeliveryNote represents a delivery note.
type DeliveryNote struct {
	ID          uuid.UUID           `json:"id"`
	EmpresaID   uuid.UUID           `json:"empresa_id"`
	OrderID     *uuid.UUID          `json:"order_id,omitempty"`
	Total       float64             `json:"total"`
	Status      string              `json:"status"` // Draft, Converted, Cancelled, Processed
	WarehouseID uuid.UUID           `json:"warehouse_id"`
	CreatedAt   time.Time           `json:"created_at"`
	Lineas      []DeliveryNoteLinea `json:"lineas,omitempty"`
}

type DeliveryNoteLinea struct {
	ID             uuid.UUID `json:"id"`
	DeliveryNoteID uuid.UUID `json:"delivery_note_id"`
	ProductoID     uuid.UUID `json:"producto_id"`
	Cantidad       float64   `json:"cantidad"`
	PrecioUnitario float64   `json:"precio_unitario"`
}

// Invoice represents a sales invoice.
type Invoice struct {
	ID                uuid.UUID      `json:"id"`
	EmpresaID         uuid.UUID      `json:"empresa_id"`
	DeliveryNoteID    *uuid.UUID     `json:"delivery_note_id,omitempty"`
	TerminalID        uuid.UUID      `json:"terminal_id"`
	InvoicingSeriesID uuid.UUID      `json:"invoicing_series_id"`
	InvoiceNumber     string         `json:"invoice_number"`
	SequenceNumber    int            `json:"sequence_number"`
	Total             float64        `json:"total"`
	Status            string         `json:"status"` // Issued, Cancelled
	CreatedAt         time.Time      `json:"created_at"`
	Lineas            []InvoiceLinea `json:"lineas,omitempty"`
}

type InvoiceLinea struct {
	ID             uuid.UUID `json:"id"`
	InvoiceID      uuid.UUID `json:"invoice_id"`
	ProductoID     uuid.UUID `json:"producto_id"`
	Cantidad       float64   `json:"cantidad"`
	PrecioUnitario float64   `json:"precio_unitario"`
}
