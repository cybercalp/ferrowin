package domain

import (
	"time"

	"github.com/google/uuid"
)

// TipoIVA represents a VAT type (e.g., General 21%, Reducido 10%).
type TipoIVA struct {
	ID         uuid.UUID `json:"id"`
	Nombre     string    `json:"nombre"`
	Porcentaje float64   `json:"porcentaje"`
	UpdatedAt  time.Time `json:"updated_at"`
	Activo     bool      `json:"activo"`
}

// Familia represents a product family/category.
type Familia struct {
	ID        uuid.UUID `json:"id"`
	Nombre    string    `json:"nombre"`
	UpdatedAt time.Time `json:"updated_at"`
	Activo    bool      `json:"activo"`
}

// Producto represents a product/SKU in the catalog.
type Producto struct {
	ID          uuid.UUID  `json:"id"`
	Codigo      string     `json:"codigo"`
	Nombre      string     `json:"nombre"`
	PrecioVenta float64    `json:"precio_venta"`
	FamiliaID   *uuid.UUID `json:"familia_id,omitempty"`
	TipoIvaID   uuid.UUID  `json:"tipo_iva_id"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Activo      bool       `json:"activo"`
}

// Entidad represents a unified business entity (client, supplier, or both).
type Entidad struct {
	ID          uuid.UUID `json:"id"`
	EmpresaID   uuid.UUID `json:"empresa_id"`
	RazonSocial string    `json:"razon_social"`
	NIF         string    `json:"nif"`
	Email       *string   `json:"email,omitempty"`
	Telefono    *string   `json:"telefono,omitempty"`
	Activo      bool      `json:"activo"`
	Roles       string    `json:"roles"`
	UpdatedAt   time.Time `json:"updated_at"`
}
