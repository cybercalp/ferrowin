package domain

import (
	"time"

	"github.com/google/uuid"
)

// Status constants for Presupuesto, Pedido, Albaran and Factura
const (
	StatusDraft     = "Borrador"
	StatusApproved  = "Aprobado"
	StatusConverted = "Convertido"
	StatusParcial   = "Parcial"
	StatusCancelled = "Anulado"
	StatusIssued    = "Emitida"
	StatusProcessed = "Procesado"
	StatusRectified = "Rectificado"
)

// Presupuesto represents a sales quote.
type Presupuesto struct {
	ID         uuid.UUID         `json:"id"`
	EmpresaID  uuid.UUID         `json:"empresa_id"`
	ClienteID  uuid.UUID         `json:"cliente_id"`
	Total      float64           `json:"total"`
	Estado     string            `json:"estado"`
	FechaValidez time.Time       `json:"fecha_validez"`
	CreatedAt  time.Time         `json:"created_at"`
	Version    int               `json:"version"`
	Lineas     []PresupuestoLinea `json:"lineas,omitempty"`
}

type PresupuestoLinea struct {
	ID             uuid.UUID `json:"id"`
	PresupuestoID  uuid.UUID `json:"presupuesto_id"`
	ProductoID     uuid.UUID `json:"producto_id"`
	Cantidad       float64   `json:"cantidad"`
	PrecioUnitario float64   `json:"precio_unitario"`
	CosteUnitario  float64   `json:"coste_unitario"`
	Convertido     float64   `json:"convertido"`
}

// Pedido represents a sales order.
type Pedido struct {
	ID         uuid.UUID       `json:"id"`
	EmpresaID  uuid.UUID       `json:"empresa_id"`
	PresupuestoID *uuid.UUID   `json:"presupuesto_id,omitempty"`
	Total      float64         `json:"total"`
	Estado     string          `json:"estado"`
	CreatedAt  time.Time       `json:"created_at"`
	Version    int             `json:"version"`
	Lineas     []PedidoLinea   `json:"lineas,omitempty"`
}

type PedidoLinea struct {
	ID             uuid.UUID `json:"id"`
	PedidoID       uuid.UUID `json:"pedido_id"`
	ProductoID     uuid.UUID `json:"producto_id"`
	Cantidad       float64   `json:"cantidad"`
	PrecioUnitario float64   `json:"precio_unitario"`
	Entregado      float64   `json:"entregado"`
}

// Albaran represents a delivery note.
type Albaran struct {
	ID         uuid.UUID       `json:"id"`
	EmpresaID  uuid.UUID       `json:"empresa_id"`
	PedidoID   *uuid.UUID      `json:"pedido_id,omitempty"`
	Total      float64         `json:"total"`
	Estado     string          `json:"estado"`
	AlmacenID  uuid.UUID       `json:"almacen_id"`
	CreatedAt  time.Time       `json:"created_at"`
	Version    int             `json:"version"`
	Lineas     []AlbaranLinea  `json:"lineas,omitempty"`
}

type AlbaranLinea struct {
	ID             uuid.UUID `json:"id"`
	AlbaranID      uuid.UUID `json:"albaran_id"`
	ProductoID     uuid.UUID `json:"producto_id"`
	Cantidad       float64   `json:"cantidad"`
	PrecioUnitario float64   `json:"precio_unitario"`
	Facturado      float64   `json:"facturado"`
}

// Factura represents a sales invoice.
type Factura struct {
	ID               uuid.UUID       `json:"id"`
	EmpresaID        uuid.UUID       `json:"empresa_id"`
	AlbaranID        *uuid.UUID      `json:"albaran_id,omitempty"`
	TerminalID       uuid.UUID       `json:"terminal_id"`
	SerieFacturacionID uuid.UUID     `json:"serie_facturacion_id"`
	NumeroFactura    string          `json:"numero_factura"`
	NumeroSecuencia  int             `json:"numero_secuencia"`
	Total            float64         `json:"total"`
	RectifiedTotal   float64         `json:"rectified_total"`
	Estado           string          `json:"estado"`
	CreatedAt        time.Time       `json:"created_at"`
	Version          int             `json:"version"`
	Lineas           []FacturaLinea  `json:"lineas,omitempty"`
}

// FacturaRectificativa represents a rectifying invoice (document that reverses an invoice).
type FacturaRectificativa struct {
	ID             uuid.UUID              `json:"id"`
	FacturaID      uuid.UUID              `json:"factura_id"`
	EmpresaID      uuid.UUID              `json:"empresa_id"`
	TerminalID     *uuid.UUID             `json:"terminal_id,omitempty"`
	NumeroFR       string                 `json:"numero_fr"`
	NumeroSecuencia int                   `json:"numero_secuencia"`
	Total          float64                `json:"total"`
	Motivo         string                 `json:"motivo"`
	Estado         string                 `json:"estado"`
	CreatedAt      time.Time              `json:"created_at"`
	Lines          []FacturaRectificativaLinea `json:"lineas,omitempty"`
}

// FacturaRectificativaLinea represents a single line in a rectifying invoice.
type FacturaRectificativaLinea struct {
	ID                   uuid.UUID `json:"id"`
	RectificativaID      uuid.UUID `json:"factura_rectificativa_id"`
	ProductoID           uuid.UUID `json:"producto_id"`
	Cantidad             float64   `json:"cantidad"`
	PrecioUnitario       float64   `json:"precio_unitario"`
}

// FacturaRectificativaLineaInput is the input type for creating a rectifying invoice line.
type FacturaRectificativaLineaInput struct {
	ProductoID     uuid.UUID
	Cantidad       float64
	PrecioUnitario float64
}

type FacturaLinea struct {
	ID        uuid.UUID `json:"id"`
	FacturaID uuid.UUID `json:"factura_id"`
	ProductoID uuid.UUID `json:"producto_id"`
	Cantidad  float64   `json:"cantidad"`
	PrecioUnitario float64 `json:"precio_unitario"`
}

// RegistroEvento represents an audit trail entry for document actions.
type RegistroEvento struct {
	ID            uuid.UUID  `json:"id"`
	DocumentoTipo string     `json:"documento_tipo"`
	DocumentoID   uuid.UUID  `json:"documento_id"`
	EmpresaID     uuid.UUID  `json:"empresa_id"`
	Accion        string     `json:"accion"`
	UsuarioID     *uuid.UUID `json:"usuario_id,omitempty"`
	Detalles      string     `json:"detalles"`
	CreatedAt     time.Time  `json:"created_at"`
}
