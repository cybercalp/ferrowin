package domain

import (
	"time"

	"github.com/google/uuid"
)

// Status constants for PedidoCompra and RecepcionCompra
const (
	PurchaseStatusBorrador  = "Borrador"
	PurchaseStatusAprobado  = "Aprobado"
	PurchaseStatusRecibido  = "Recibido"
	PurchaseStatusParcial   = "Parcial"
	PurchaseStatusCancelado = "Cancelado"
	ReceiptStatusBorrador   = "Borrador"
	ReceiptStatusProcesado  = "Procesado"
	ReceiptStatusCancelado  = "Cancelado"
)

type Empresa struct {
	ID          uuid.UUID `json:"id"`
	RazonSocial string    `json:"razon_social"`
	NIF         string    `json:"nif"`
	Activa      bool      `json:"activa"`
}

type Warehouse struct {
	ID        uuid.UUID `json:"id"`
	EmpresaID uuid.UUID `json:"empresa_id"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
}

type Proveedor struct {
	ID          uuid.UUID `json:"id"`
	EmpresaID   uuid.UUID `json:"empresa_id"`
	RazonSocial string    `json:"razon_social"`
	CIF         string    `json:"cif"`
	Email       string    `json:"email"`
	Telefono    string    `json:"telefono"`
	Direccion   string    `json:"direccion"`
	Activo      bool      `json:"activo"`
}

type PedidoCompra struct {
	ID           uuid.UUID           `json:"id"`
	EmpresaID    uuid.UUID           `json:"empresa_id"`
	ProveedorID  uuid.UUID           `json:"proveedor_id"`
	NumeroPedido string              `json:"numero_pedido"`
	Fecha        time.Time           `json:"fecha"`
	Estado       string              `json:"estado"` // Borrador, Aprobado, Recibido, Parcial, Cancelado
	Total        float64             `json:"total"`
	Version      int                 `json:"version"`
	Lineas       []PedidoCompraLinea `json:"lineas,omitempty"`
}

type PedidoCompraLinea struct {
	ID             uuid.UUID `json:"id"`
	PedidoCompraID uuid.UUID `json:"pedido_compra_id"`
	ProductoID     uuid.UUID `json:"producto_id"`
	Cantidad       float64   `json:"cantidad"`
	PrecioUnitario float64   `json:"precio_unitario"`
	Recibido       float64   `json:"recibido"`
}

type RecepcionCompra struct {
	ID             uuid.UUID              `json:"id"`
	EmpresaID      uuid.UUID              `json:"empresa_id"`
	PedidoCompraID *uuid.UUID             `json:"pedido_compra_id,omitempty"`
	ProveedorID    uuid.UUID              `json:"proveedor_id"`
	NumeroAlbaran  string                 `json:"numero_albaran"`
	Fecha          time.Time              `json:"fecha"`
	Estado         string                 `json:"estado"` // Borrador, Procesado, Cancelado
	WarehouseID    uuid.UUID              `json:"warehouse_id"`
	Version        int                    `json:"version"`
	Lineas         []RecepcionCompraLinea `json:"lineas,omitempty"`
}

type RecepcionCompraLinea struct {
	ID                uuid.UUID `json:"id"`
	RecepcionCompraID uuid.UUID `json:"recepcion_compra_id"`
	ProductoID        uuid.UUID `json:"producto_id"`
	Cantidad          float64   `json:"cantidad"`
	PrecioUnitario    float64   `json:"precio_unitario"`
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
