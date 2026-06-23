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
	ErrConcurrentModification   = errors.New("concurrent modification detected, please retry")
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

	SaveEvento(ctx context.Context, evento *RegistroEvento) error
}

type PurchaseService struct {
	repo       PurchaseRepository
	invService *inventorydomain.InventoryService
	Now        func() time.Time
}

func NewPurchaseService(repo PurchaseRepository, invService *inventorydomain.InventoryService) *PurchaseService {
	return &PurchaseService{
		repo:       repo,
		invService: invService,
	}
}

func (s *PurchaseService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// recordEvento is a best-effort audit trail helper.
func (s *PurchaseService) recordEvento(ctx context.Context, empresaID uuid.UUID, docTipo string, docID uuid.UUID, accion string, usuarioID *uuid.UUID, detalles string) {
	evento := &RegistroEvento{
		ID:            uuid.New(),
		DocumentoTipo: docTipo,
		DocumentoID:   docID,
		EmpresaID:     empresaID,
		Accion:        accion,
		UsuarioID:     usuarioID,
		Detalles:      detalles,
		CreatedAt:     s.now(),
	}
	s.repo.SaveEvento(ctx, evento) // best-effort — ignore error
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
	if po.Estado == PurchaseStatusCancelado {
		return fmt.Errorf("%w: purchase order is already cancelled", ErrInvalidStatus)
	}
	if po.Estado == PurchaseStatusRecibido {
		return fmt.Errorf("%w: cannot cancel a received purchase order", ErrInvalidStatus)
	}
	if err := s.repo.CancelPurchaseOrder(ctx, orderID); err != nil {
		return err
	}
	s.recordEvento(ctx, empresaID, "pedido_compra", orderID, "anular", nil, "")
	return nil
}

func (s *PurchaseService) CancelPurchaseReceipt(ctx context.Context, empresaID, receiptID uuid.UUID) error {
	rc, err := s.repo.GetPurchaseReceipt(ctx, receiptID)
	if err != nil {
		return err
	}
	if rc.EmpresaID != empresaID {
		return ErrTenantMismatch
	}
	if rc.Estado == ReceiptStatusCancelado {
		return fmt.Errorf("%w: purchase receipt is already cancelled", ErrInvalidStatus)
	}

	// Reverse stock if receipt was already processed
	if rc.Estado == ReceiptStatusProcesado {
		refDocType := "PURCHASE_RECEIPT"
		for _, l := range rc.Lineas {
			_, err := s.invService.RecordReturn(ctx, l.ProductoID, rc.WarehouseID, l.Cantidad, &refDocType, &rc.ID)
			if err != nil {
				return fmt.Errorf("failed to reverse stock for product %s: %w", l.ProductoID, err)
			}
		}
	}

	if err := s.repo.CancelPurchaseReceipt(ctx, receiptID); err != nil {
		return err
	}
	s.recordEvento(ctx, empresaID, "recepcion_compra", receiptID, "anular", nil, "")
	return nil
}

// Purchase Order
func (s *PurchaseService) CreatePurchaseOrder(ctx context.Context, empresaID, proveedorID uuid.UUID, numeroPedido string, lines []PedidoCompraLinea) (*PedidoCompra, error) {
	if len(lines) == 0 {
		return nil, fmt.Errorf("at least one line is required")
	}
	for i, l := range lines {
		if l.Cantidad <= 0 {
			return nil, fmt.Errorf("line %d: cantidad must be positive", i)
		}
		if l.PrecioUnitario < 0 {
			return nil, fmt.Errorf("line %d: precio_unitario must be non-negative", i)
		}
	}

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
		Fecha:        s.now(),
		Estado:       PurchaseStatusBorrador,
		Total:        total,
		Lineas:       poLines,
	}

	if err := s.repo.SavePurchaseOrder(ctx, po); err != nil {
		return nil, err
	}
	s.recordEvento(ctx, empresaID, "pedido_compra", poID, "crear", nil, "")
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
	if po.Estado != PurchaseStatusBorrador {
		return fmt.Errorf("%w: cannot approve from %s state", ErrInvalidStatus, po.Estado)
	}

	po.Estado = PurchaseStatusAprobado
	if err := s.repo.SavePurchaseOrder(ctx, po); err != nil {
		return err
	}
	s.recordEvento(ctx, empresaID, "pedido_compra", orderID, "aprobar", nil, "")
	return nil
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
	if len(lines) == 0 {
		return nil, fmt.Errorf("at least one line is required")
	}
	for i, l := range lines {
		if l.Cantidad <= 0 {
			return nil, fmt.Errorf("line %d: cantidad must be positive", i)
		}
		if l.PrecioUnitario < 0 {
			return nil, fmt.Errorf("line %d: precio_unitario must be non-negative", i)
		}
	}

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

	// Validate PO status if linked
	if poID != nil {
		po, err := s.repo.GetPurchaseOrder(ctx, *poID)
		if err != nil {
			return nil, fmt.Errorf("purchase order not found: %w", err)
		}
		if po.Estado != PurchaseStatusAprobado {
			return nil, fmt.Errorf("%w: cannot create receipt against a purchase order in %s status", ErrInvalidStatus, po.Estado)
		}
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
		Fecha:          s.now(),
		Estado:         ReceiptStatusBorrador,
		WarehouseID:    warehouseID,
		Lineas:         rcLines,
	}

	if err := s.repo.SavePurchaseReceipt(ctx, rc); err != nil {
		return nil, err
	}
	s.recordEvento(ctx, empresaID, "recepcion_compra", receiptID, "crear", nil, "")
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
	if rc.Estado != ReceiptStatusBorrador {
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

	rc.Estado = ReceiptStatusProcesado
	if err := s.repo.SavePurchaseReceipt(ctx, rc); err != nil {
		return err
	}

	// If linked to a purchase order, update partial receipt tracking
	if rc.PedidoCompraID != nil {
		po, err := s.repo.GetPurchaseOrder(ctx, *rc.PedidoCompraID)
		if err == nil && po != nil {
			// Update each PO line's Recibido quantity
			for _, rcLine := range rc.Lineas {
				for i, poLine := range po.Lineas {
					if poLine.ProductoID == rcLine.ProductoID {
						po.Lineas[i].Recibido += rcLine.Cantidad
						break
					}
				}
			}

			// Determine new PO status based on receipt progress
			allReceived := true
			for _, l := range po.Lineas {
				if l.Recibido < l.Cantidad {
					allReceived = false
					break
				}
			}
			if allReceived {
				po.Estado = PurchaseStatusRecibido
			} else {
				po.Estado = PurchaseStatusParcial
			}

			_ = s.repo.SavePurchaseOrder(ctx, po) // soft error if po update fails
		}
	}

	s.recordEvento(ctx, empresaID, "recepcion_compra", receiptID, "procesar", nil, "")
	return nil
}
