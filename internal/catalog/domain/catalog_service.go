package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrTipoIVANotFound  = errors.New("tipo iva not found")
	ErrFamiliaNotFound  = errors.New("familia not found")
	ErrProductoNotFound = errors.New("producto not found")
	ErrClienteNotFound  = errors.New("cliente not found")
	ErrValidation       = errors.New("validation error")
)

// CatalogRepository defines the data access interface for catalog entities.
type CatalogRepository interface {
	// Tipos IVA
	CreateTipoIVA(ctx context.Context, t *TipoIVA) error
	GetTipoIVA(ctx context.Context, id uuid.UUID) (*TipoIVA, error)
	ListTiposIVA(ctx context.Context) ([]TipoIVA, error)
	UpdateTipoIVA(ctx context.Context, t *TipoIVA) error
	DeleteTipoIVA(ctx context.Context, id uuid.UUID) error

	// Familias
	CreateFamilia(ctx context.Context, f *Familia) error
	GetFamilia(ctx context.Context, id uuid.UUID) (*Familia, error)
	ListFamilias(ctx context.Context) ([]Familia, error)
	UpdateFamilia(ctx context.Context, f *Familia) error
	DeleteFamilia(ctx context.Context, id uuid.UUID) error

	// Productos
	CreateProducto(ctx context.Context, p *Producto) error
	GetProducto(ctx context.Context, id uuid.UUID) (*Producto, error)
	ListProductos(ctx context.Context, familiaID, tipoIvaID *uuid.UUID) ([]Producto, error)
	UpdateProducto(ctx context.Context, p *Producto) error
	DeleteProducto(ctx context.Context, id uuid.UUID) error

	// Entidades (only CLIENTE role)
	CreateCliente(ctx context.Context, e *Entidad) error
	GetCliente(ctx context.Context, id uuid.UUID) (*Entidad, error)
	ListClientes(ctx context.Context, empresaID uuid.UUID) ([]Entidad, error)
	UpdateCliente(ctx context.Context, e *Entidad) error
	DeleteCliente(ctx context.Context, id uuid.UUID) error
}

// CatalogService provides business logic and validation for catalog operations.
type CatalogService struct {
	repo CatalogRepository
}

// NewCatalogService creates a new CatalogService.
func NewCatalogService(repo CatalogRepository) *CatalogService {
	return &CatalogService{repo: repo}
}

// --- Tipos IVA ---

func (s *CatalogService) CreateTipoIVA(ctx context.Context, nombre string, porcentaje float64) (*TipoIVA, error) {
	if nombre == "" {
		return nil, errors.New("nombre is required")
	}
	if porcentaje <= 0 {
		return nil, errors.New("porcentaje must be greater than 0")
	}
	t := &TipoIVA{
		ID:         uuid.New(),
		Nombre:     nombre,
		Porcentaje: porcentaje,
		UpdatedAt:  time.Now(),
		Activo:     true,
	}
	if err := s.repo.CreateTipoIVA(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *CatalogService) GetTipoIVA(ctx context.Context, id uuid.UUID) (*TipoIVA, error) {
	return s.repo.GetTipoIVA(ctx, id)
}

func (s *CatalogService) ListTiposIVA(ctx context.Context) ([]TipoIVA, error) {
	return s.repo.ListTiposIVA(ctx)
}

func (s *CatalogService) UpdateTipoIVA(ctx context.Context, t *TipoIVA) error {
	if t.Nombre == "" {
		return errors.New("nombre is required")
	}
	if t.Porcentaje <= 0 {
		return errors.New("porcentaje must be greater than 0")
	}
	t.UpdatedAt = time.Now()
	return s.repo.UpdateTipoIVA(ctx, t)
}

func (s *CatalogService) DeleteTipoIVA(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteTipoIVA(ctx, id)
}

// --- Familias ---

func (s *CatalogService) CreateFamilia(ctx context.Context, nombre string) (*Familia, error) {
	if nombre == "" {
		return nil, errors.New("nombre is required")
	}
	f := &Familia{
		ID:        uuid.New(),
		Nombre:    nombre,
		UpdatedAt: time.Now(),
		Activo:    true,
	}
	if err := s.repo.CreateFamilia(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *CatalogService) GetFamilia(ctx context.Context, id uuid.UUID) (*Familia, error) {
	return s.repo.GetFamilia(ctx, id)
}

func (s *CatalogService) ListFamilias(ctx context.Context) ([]Familia, error) {
	return s.repo.ListFamilias(ctx)
}

func (s *CatalogService) UpdateFamilia(ctx context.Context, f *Familia) error {
	if f.Nombre == "" {
		return errors.New("nombre is required")
	}
	f.UpdatedAt = time.Now()
	return s.repo.UpdateFamilia(ctx, f)
}

func (s *CatalogService) DeleteFamilia(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteFamilia(ctx, id)
}

// --- Productos ---

func (s *CatalogService) CreateProducto(ctx context.Context, codigo, nombre string, precioVenta float64, familiaID *uuid.UUID, tipoIvaID uuid.UUID) (*Producto, error) {
	if codigo == "" {
		return nil, errors.New("codigo is required")
	}
	if nombre == "" {
		return nil, errors.New("nombre is required")
	}
	if precioVenta < 0 {
		return nil, errors.New("precio_venta must be >= 0")
	}
	// Validate tipo_iva_id references an existing tipo IVA
	if _, err := s.repo.GetTipoIVA(ctx, tipoIvaID); err != nil {
		return nil, errors.New("tipo_iva_id does not reference an existing tipo IVA")
	}

	p := &Producto{
		ID:          uuid.New(),
		Codigo:      codigo,
		Nombre:      nombre,
		PrecioVenta: precioVenta,
		FamiliaID:   familiaID,
		TipoIvaID:   tipoIvaID,
		UpdatedAt:   time.Now(),
		Activo:      true,
	}
	if err := s.repo.CreateProducto(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *CatalogService) GetProducto(ctx context.Context, id uuid.UUID) (*Producto, error) {
	return s.repo.GetProducto(ctx, id)
}

func (s *CatalogService) ListProductos(ctx context.Context, familiaID, tipoIvaID *uuid.UUID) ([]Producto, error) {
	return s.repo.ListProductos(ctx, familiaID, tipoIvaID)
}

func (s *CatalogService) UpdateProducto(ctx context.Context, p *Producto) error {
	if p.Codigo == "" {
		return errors.New("codigo is required")
	}
	if p.Nombre == "" {
		return errors.New("nombre is required")
	}
	if p.PrecioVenta < 0 {
		return errors.New("precio_venta must be >= 0")
	}
	p.UpdatedAt = time.Now()
	return s.repo.UpdateProducto(ctx, p)
}

func (s *CatalogService) DeleteProducto(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteProducto(ctx, id)
}

// --- Clientes ---

func (s *CatalogService) CreateCliente(ctx context.Context, empresaID uuid.UUID, razonSocial, nif string, email, telefono *string) (*Entidad, error) {
	if razonSocial == "" {
		return nil, errors.New("razon_social is required")
	}
	if nif == "" {
		return nil, errors.New("nif is required")
	}
	e := &Entidad{
		ID:          uuid.New(),
		EmpresaID:   empresaID,
		RazonSocial: razonSocial,
		NIF:         nif,
		Email:       email,
		Telefono:    telefono,
		Activo:      true,
		Roles:       "CLIENTE",
		UpdatedAt:   time.Now(),
	}
	if err := s.repo.CreateCliente(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *CatalogService) GetCliente(ctx context.Context, id uuid.UUID) (*Entidad, error) {
	return s.repo.GetCliente(ctx, id)
}

func (s *CatalogService) ListClientes(ctx context.Context, empresaID uuid.UUID) ([]Entidad, error) {
	return s.repo.ListClientes(ctx, empresaID)
}

func (s *CatalogService) UpdateCliente(ctx context.Context, e *Entidad) error {
	if e.RazonSocial == "" {
		return errors.New("razon_social is required")
	}
	if e.NIF == "" {
		return errors.New("nif is required")
	}
	e.UpdatedAt = time.Now()
	return s.repo.UpdateCliente(ctx, e)
}

func (s *CatalogService) DeleteCliente(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteCliente(ctx, id)
}
