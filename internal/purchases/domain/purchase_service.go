package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	inventorydomain "ferrowin/internal/inventory/domain"
	"github.com/google/uuid"
)

var (
	ErrCompanyNotFound          = errors.New("company not found")
	ErrWarehouseNotFound        = errors.New("warehouse not found")
	ErrSupplierNotFound         = errors.New("supplier not found")
	ErrPurchaseOrderNotFound    = errors.New("purchase order not found")
	ErrPurchaseReceiptNotFound  = errors.New("purchase receipt not found")
	ErrTenantMismatch           = errors.New("tenant company mismatch")
	ErrInvalidStatus            = errors.New("invalid status transition")
)

// PurchaseOrderFilter holds optional filter fields for listing purchase orders.
type PurchaseOrderFilter struct {
	EmpresaID   *uuid.UUID
	Estado      *string
	ProveedorID *uuid.UUID
	Desde       *time.Time
	Hasta       *time.Time
	Page        int
	PageSize    int
}

// PurchaseReceiptFilter holds optional filter fields for listing purchase receipts.
type PurchaseReceiptFilter struct {
	EmpresaID *uuid.UUID
	Estado    *string
	Desde     *time.Time
	Hasta     *time.Time
	Page      int
	PageSize  int
}

// Update input types for partial updates (nil fields = don't update)
type UpdateCompanyInput struct {
	ID          uuid.UUID
	RazonSocial *string
	NIF         *string
}

type UpdateWarehouseInput struct {
	ID     uuid.UUID
	Name   *string
	Active *bool
}

type UpdateSupplierInput struct {
	ID          uuid.UUID
	RazonSocial *string
	CIF         *string
	Email       *string
	Telefono    *string
	Activo      *bool
}

type PurchaseRepository interface {
	SaveCompany(ctx context.Context, c *Empresa) error
	GetCompanies(ctx context.Context) ([]*Empresa, error)
	UpdateCompany(ctx context.Context, input UpdateCompanyInput) error

	SaveWarehouse(ctx context.Context, w *Warehouse) error
	GetWarehouse(ctx context.Context, id uuid.UUID) (*Warehouse, error)
	GetAllWarehouses(ctx context.Context, empresaID uuid.UUID) ([]*Warehouse, error)
	UpdateWarehouse(ctx context.Context, input UpdateWarehouseInput) error

	SaveSupplier(ctx context.Context, s *Proveedor) error
	GetSuppliers(ctx context.Context, empresaID uuid.UUID) ([]*Proveedor, error)
	GetSupplier(ctx context.Context, id uuid.UUID) (*Proveedor, error)
	UpdateSupplier(ctx context.Context, input UpdateSupplierInput) error

	SavePurchaseOrder(ctx context.Context, o *PedidoCompra) error
	GetPurchaseOrder(ctx context.Context, id uuid.UUID) (*PedidoCompra, error)
	ListPurchaseOrders(ctx context.Context, empresaID uuid.UUID, filter PurchaseOrderFilter) ([]*PedidoCompra, int, error)
	CancelPurchaseOrder(ctx context.Context, id uuid.UUID) error

	SavePurchaseReceipt(ctx context.Context, r *RecepcionCompra) error
	GetPurchaseReceipt(ctx context.Context, id uuid.UUID) (*RecepcionCompra, error)
	ListPurchaseReceipts(ctx context.Context, empresaID uuid.UUID, filter PurchaseReceiptFilter) ([]*RecepcionCompra, int, error)
	CancelPurchaseReceipt(ctx context.Context, id uuid.UUID) error
}

type PurchaseService struct {
	repo       PurchaseRepository
	invService *inventorydomain.InventoryService
}

func NewPurchaseService(repo PurchaseRepository, invService *inventorydomain.InventoryService) *PurchaseService {
	return &PurchaseService{
		repo:       repo,
		invService: invService,
	}
}

// Company & Warehouse
func (s *PurchaseService) CreateCompany(ctx context.Context, razonSocial, nif string) (*Empresa, error) {
	c := &Empresa{
		ID:          uuid.New(),
		RazonSocial: razonSocial,
		NIF:         nif,
		Activa:      true,
	}
	if err := s.repo.SaveCompany(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *PurchaseService) CreateWarehouse(ctx context.Context, empresaID uuid.UUID, name string) (*Warehouse, error) {
	w := &Warehouse{
		ID:        uuid.New(),
		EmpresaID: empresaID,
		Name:      name,
		Active:    true,
	}
	if err := s.repo.SaveWarehouse(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

// Company & Warehouse
func (s *PurchaseService) GetCompanies(ctx context.Context) ([]*Empresa, error) {
	return s.repo.GetCompanies(ctx)
}

func (s *PurchaseService) GetAllWarehouses(ctx context.Context, empresaID uuid.UUID) ([]*Warehouse, error) {
	return s.repo.GetAllWarehouses(ctx, empresaID)
}

// Supplier
func (s *PurchaseService) CreateSupplier(ctx context.Context, empresaID uuid.UUID, razonSocial, cif, email, telefono, direccion string) (*Proveedor, error) {
	prov := &Proveedor{
		ID:          uuid.New(),
		EmpresaID:   empresaID,
		RazonSocial: razonSocial,
		CIF:         cif,
		Email:       email,
		Telefono:    telefono,
		Direccion:   direccion,
		Activo:      true,
	}
	if err := s.repo.SaveSupplier(ctx, prov); err != nil {
		return nil, err
	}
	return prov, nil
}

func (s *PurchaseService) GetSuppliers(ctx context.Context, empresaID uuid.UUID) ([]*Proveedor, error) {
	return s.repo.GetSuppliers(ctx, empresaID)
}

// Update methods delegate to repository.
func (s *PurchaseService) UpdateCompany(ctx context.Context, input UpdateCompanyInput) error {
	return s.repo.UpdateCompany(ctx, input)
}
func (s *PurchaseService) UpdateWarehouse(ctx context.Context, input UpdateWarehouseInput) error {
	return s.repo.UpdateWarehouse(ctx, input)
}
func (s *PurchaseService) UpdateSupplier(ctx context.Context, input UpdateSupplierInput) error {
	return s.repo.UpdateSupplier(ctx, input)
}

func (s *PurchaseService) CancelPurchaseOrder(ctx context.Context, empresaID, orderID uuid.UUID) error {
	po, err := s.repo.GetPurchaseOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if po.EmpresaID != empresaID {
		return ErrTenantMismatch
	}
	if po.Estado == "Cancelado" {
		return fmt.Errorf("%w: purchase order is already cancelled", ErrInvalidStatus)
	}
	if po.Estado == "Recibido" {
		return fmt.Errorf("%w: cannot cancel a received purchase order", ErrInvalidStatus)
	}
	return s.repo.CancelPurchaseOrder(ctx, orderID)
}

func (s *PurchaseService) CancelPurchaseReceipt(ctx context.Context, empresaID, receiptID uuid.UUID) error {
	rc, err := s.repo.GetPurchaseReceipt(ctx, receiptID)
	if err != nil {
		return err
	}
	if rc.EmpresaID != empresaID {
		return ErrTenantMismatch
	}
	if rc.Estado == "Cancelado" {
		return fmt.Errorf("%w: purchase receipt is already cancelled", ErrInvalidStatus)
	}
	if rc.Estado == "Procesado" {
		return fmt.Errorf("%w: cannot cancel a processed purchase receipt", ErrInvalidStatus)
	}
	return s.repo.CancelPurchaseReceipt(ctx, receiptID)
}

// Purchase Order
func (s *PurchaseService) CreatePurchaseOrder(ctx context.Context, empresaID, proveedorID uuid.UUID, numeroPedido string, lines []PedidoCompraLinea) (*PedidoCompra, error) {
	// Validate supplier belongs to the same company
	prov, err := s.repo.GetSupplier(ctx, proveedorID)
	if err != nil {
		return nil, err
	}
	if prov.EmpresaID != empresaID {
		return nil, ErrTenantMismatch
	}

	var total float64
	poLines := make([]PedidoCompraLinea, len(lines))
	poID := uuid.New()
	for i, l := range lines {
		total += l.Cantidad * l.PrecioUnitario
		poLines[i] = PedidoCompraLinea{
			ID:             uuid.New(),
			PedidoCompraID: poID,
			ProductoID:     l.ProductoID,
			Cantidad:       l.Cantidad,
			PrecioUnitario: l.PrecioUnitario,
		}
	}

	po := &PedidoCompra{
		ID:           poID,
		EmpresaID:    empresaID,
		ProveedorID:  proveedorID,
		NumeroPedido: numeroPedido,
		Fecha:        time.Now(),
		Estado:       "Borrador",
		Total:        total,
		Lineas:       poLines,
	}

	if err := s.repo.SavePurchaseOrder(ctx, po); err != nil {
		return nil, err
	}
	return po, nil
}

func (s *PurchaseService) GetPurchaseOrder(ctx context.Context, id uuid.UUID) (*PedidoCompra, error) {
	return s.repo.GetPurchaseOrder(ctx, id)
}

func (s *PurchaseService) GetPurchaseReceipt(ctx context.Context, id uuid.UUID) (*RecepcionCompra, error) {
	return s.repo.GetPurchaseReceipt(ctx, id)
}

func (s *PurchaseService) ApprovePurchaseOrder(ctx context.Context, empresaID, orderID uuid.UUID) error {
	po, err := s.repo.GetPurchaseOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if po.EmpresaID != empresaID {
		return ErrTenantMismatch
	}
	if po.Estado != "Borrador" {
		return fmt.Errorf("%w: cannot approve from %s state", ErrInvalidStatus, po.Estado)
	}

	po.Estado = "Aprobado"
	return s.repo.SavePurchaseOrder(ctx, po)
}

func (s *PurchaseService) ListPurchaseOrders(ctx context.Context, empresaID uuid.UUID, filter PurchaseOrderFilter) ([]*PedidoCompra, int, error) {
	filter.EmpresaID = &empresaID
	return s.repo.ListPurchaseOrders(ctx, empresaID, filter)
}

func (s *PurchaseService) ListPurchaseReceipts(ctx context.Context, empresaID uuid.UUID, filter PurchaseReceiptFilter) ([]*RecepcionCompra, int, error) {
	filter.EmpresaID = &empresaID
	return s.repo.ListPurchaseReceipts(ctx, empresaID, filter)
}

// Purchase Receipt
func (s *PurchaseService) CreatePurchaseReceipt(ctx context.Context, empresaID, supplierID uuid.UUID, poID *uuid.UUID, numeroAlbaran string, warehouseID uuid.UUID, lines []RecepcionCompraLinea) (*RecepcionCompra, error) {
	// Validate supplier
	prov, err := s.repo.GetSupplier(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	if prov.EmpresaID != empresaID {
		return nil, ErrTenantMismatch
	}

	// Validate warehouse
	wh, err := s.repo.GetWarehouse(ctx, warehouseID)
	if err != nil {
		return nil, err
	}
	if wh.EmpresaID != empresaID {
		return nil, ErrTenantMismatch
	}

	receiptID := uuid.New()
	rcLines := make([]RecepcionCompraLinea, len(lines))
	for i, l := range lines {
		rcLines[i] = RecepcionCompraLinea{
			ID:                uuid.New(),
			RecepcionCompraID: receiptID,
			ProductoID:        l.ProductoID,
			Cantidad:          l.Cantidad,
			PrecioUnitario:    l.PrecioUnitario,
		}
	}

	rc := &RecepcionCompra{
		ID:             receiptID,
		EmpresaID:      empresaID,
		PedidoCompraID: poID,
		ProveedorID:    supplierID,
		NumeroAlbaran:  numeroAlbaran,
		Fecha:          time.Now(),
		Estado:         "Borrador",
		WarehouseID:    warehouseID,
		Lineas:         rcLines,
	}

	if err := s.repo.SavePurchaseReceipt(ctx, rc); err != nil {
		return nil, err
	}
	return rc, nil
}

func (s *PurchaseService) ProcessPurchaseReceipt(ctx context.Context, empresaID, receiptID uuid.UUID) error {
	rc, err := s.repo.GetPurchaseReceipt(ctx, receiptID)
	if err != nil {
		return err
	}
	if rc.EmpresaID != empresaID {
		return ErrTenantMismatch
	}
	if rc.Estado != "Borrador" {
		return fmt.Errorf("%w: cannot process receipt from %s state", ErrInvalidStatus, rc.Estado)
	}

	// Process stock additions
	refDocType := "PURCHASE_RECEIPT"
	for _, line := range rc.Lineas {
		_, err := s.invService.RecordReceipt(ctx, line.ProductoID, rc.WarehouseID, line.Cantidad, &refDocType, &rc.ID)
		if err != nil {
			return fmt.Errorf("failed to record stock movement for product %s: %w", line.ProductoID, err)
		}
	}

	rc.Estado = "Procesado"
	if err := s.repo.SavePurchaseReceipt(ctx, rc); err != nil {
		return err
	}

	// If linked to a purchase order, transition the purchase order status to 'Recibido'
	if rc.PedidoCompraID != nil {
		po, err := s.repo.GetPurchaseOrder(ctx, *rc.PedidoCompraID)
		if err == nil && po != nil {
			po.Estado = "Recibido"
			_ = s.repo.SavePurchaseOrder(ctx, po) // soft error if po update fails
		}
	}

	return nil
}
