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
	ErrDocumentAlreadyConverted = errors.New("cannot convert: document is already in Converted status")
	ErrDocumentAlreadyCancelled = errors.New("cannot convert: document is already in Cancelled status")
	ErrPresupuestoExpired             = errors.New("cannot convert: quote is expired")
	ErrUnauthorized             = errors.New("cannot convert: user is not authorized to convert expired quotes")
	ErrSecurityServiceNil       = errors.New("security service is required but not configured")
	ErrBillingServiceNil        = errors.New("billing service is required but not configured")
	ErrTenantMismatch           = errors.New("tenant company mismatch")
	ErrInvalidStatus            = errors.New("invalid status transition")
	ErrConcurrentModification   = errors.New("concurrent modification detected, please retry")
	ErrPresupuestoNotFound            = errors.New("quote not found")
	ErrPedidoNotFound            = errors.New("order not found")
	ErrAlbaranNotFound     = errors.New("delivery note not found")
	ErrFacturaNotFound          = errors.New("invoice not found")
	ErrFacturaRectificativaNotFound = errors.New("factura rectificativa not found")
	ErrFacturaAlreadyRectified      = errors.New("invoice is already fully rectified")
	ErrCannotRectifyCancelled       = errors.New("cannot rectify a cancelled invoice")
	ErrFacturaNoAlbaran        = errors.New("invoice has no delivery note, cannot determine warehouse")
	ErrTerminalRequired             = errors.New("terminal_id is required for FR number generation")
	ErrProductNotOnFactura          = errors.New("product not found on invoice")
	ErrQuantityExceedsFactura       = errors.New("rectification quantity exceeds invoiced quantity")
)

// SecurityServiceRequired defines the security permission check contract required by this domain service.
type SecurityServiceRequired interface {
	HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error)
}

// BillingServiceRequired defines the sequence generation contract required by this domain service.
type BillingServiceRequired interface {
	GenerateFacturaNumber(ctx context.Context, terminalID uuid.UUID) (string, int, error)
}

// Update input types for partial updates (nil fields = don't update)
type UpdatePresupuestoInput struct {
	ID        uuid.UUID
	ClienteID  *uuid.UUID
	FechaValidez *time.Time
}

type UpdatePedidoInput struct {
	ID uuid.UUID
}

type UpdateAlbaranInput struct {
	ID uuid.UUID
}

// DocumentFilter holds optional filter fields for listing sales documents.
type DocumentFilter struct {
	EmpresaID *uuid.UUID
	Estado    *string
	ClienteID *uuid.UUID
	Desde     *time.Time
	Hasta     *time.Time
	Page      int
	PageSize  int
}

type SalesRepository interface {
	SavePresupuesto(ctx context.Context, q *Presupuesto) error
	GetPresupuesto(ctx context.Context, id uuid.UUID) (*Presupuesto, error)
	ListPresupuestos(ctx context.Context, empresaID uuid.UUID, filter DocumentFilter) ([]*Presupuesto, int, error)
	UpdatePresupuesto(ctx context.Context, input UpdatePresupuestoInput) error
	CancelPresupuesto(ctx context.Context, id uuid.UUID) error

	SavePedido(ctx context.Context, o *Pedido) error
	GetPedido(ctx context.Context, id uuid.UUID) (*Pedido, error)
	ListPedidos(ctx context.Context, empresaID uuid.UUID, filter DocumentFilter) ([]*Pedido, int, error)
	UpdatePedido(ctx context.Context, input UpdatePedidoInput) error
	CancelPedido(ctx context.Context, id uuid.UUID) error

	SaveAlbaran(ctx context.Context, dn *Albaran) error
	GetAlbaran(ctx context.Context, id uuid.UUID) (*Albaran, error)
	ListAlbarans(ctx context.Context, empresaID uuid.UUID, filter DocumentFilter) ([]*Albaran, int, error)
	UpdateAlbaran(ctx context.Context, input UpdateAlbaranInput) error
	CancelAlbaran(ctx context.Context, id uuid.UUID) error

	SaveFactura(ctx context.Context, inv *Factura) error
	GetFactura(ctx context.Context, id uuid.UUID) (*Factura, error)
	ListFacturas(ctx context.Context, empresaID uuid.UUID, filter DocumentFilter) ([]*Factura, int, error)
	CancelFactura(ctx context.Context, id uuid.UUID) error

	CreateFacturaRectificativa(ctx context.Context, fr *FacturaRectificativa) error
	GetFacturaRectificativa(ctx context.Context, id uuid.UUID) (*FacturaRectificativa, error)
	ListFacturasRectificativas(ctx context.Context, empresaID uuid.UUID) ([]FacturaRectificativa, error)
	UpdateFacturaRectifiedTotal(ctx context.Context, invoiceID uuid.UUID, rectifiedTotal float64) error
	GetRectifiedQuantitiesByInvoice(ctx context.Context, invoiceID uuid.UUID) (map[uuid.UUID]float64, error)
}

// ConvertPresupuestoOptions specifies options when converting a quote.
type ConvertPresupuestoOptions struct {
	RecalculatePrices bool
}

// ConversionLineInput specifies a single line for partial conversion.
type ConversionLineInput struct {
	ProductoID uuid.UUID
	Cantidad   float64
}

// ConvertPedidoInput holds parameters for pedido-to-albaran conversion.
type ConvertPedidoInput struct {
	PedidoID  uuid.UUID
	AlmacenID uuid.UUID
	Lineas    []ConversionLineInput // nil/empty = convert all remaining
}

// ConvertPresupuestoInput holds parameters for presupuesto-to-pedido conversion.
type ConvertPresupuestoInput struct {
	PresupuestoID     uuid.UUID
	UserID            uuid.UUID
	RecalculatePrices bool
	Lineas            []ConversionLineInput // nil/empty = convert all remaining
}

// ConvertAlbaranInput holds parameters for albaran-to-factura conversion.
type ConvertAlbaranInput struct {
	AlbaranID          uuid.UUID
	TerminalID         uuid.UUID
	SerieFacturacionID uuid.UUID
	Lineas             []ConversionLineInput // nil/empty = convert all remaining
}

// SalesService handles sales document flows, validation, and transitions.
type SalesService struct {
	repo            SalesRepository
	invService      *inventorydomain.InventoryService
	securityService SecurityServiceRequired
	billingService  BillingServiceRequired
	Now             func() time.Time
}

// NewSalesService creates a new instance of SalesService.
func NewSalesService(
	repo SalesRepository,
	invService *inventorydomain.InventoryService,
	security SecurityServiceRequired,
	billing BillingServiceRequired,
) *SalesService {
	return &SalesService{
		repo:            repo,
		invService:      invService,
		securityService: security,
		billingService:  billing,
	}
}

func (s *SalesService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// List methods
func (s *SalesService) ListPresupuestos(ctx context.Context, empresaID uuid.UUID, filter DocumentFilter) ([]*Presupuesto, int, error) {
	filter.EmpresaID = &empresaID
	return s.repo.ListPresupuestos(ctx, empresaID, filter)
}

func (s *SalesService) ListPedidos(ctx context.Context, empresaID uuid.UUID, filter DocumentFilter) ([]*Pedido, int, error) {
	filter.EmpresaID = &empresaID
	return s.repo.ListPedidos(ctx, empresaID, filter)
}

func (s *SalesService) ListAlbarans(ctx context.Context, empresaID uuid.UUID, filter DocumentFilter) ([]*Albaran, int, error) {
	filter.EmpresaID = &empresaID
	return s.repo.ListAlbarans(ctx, empresaID, filter)
}

func (s *SalesService) ListFacturas(ctx context.Context, empresaID uuid.UUID, filter DocumentFilter) ([]*Factura, int, error) {
	filter.EmpresaID = &empresaID
	return s.repo.ListFacturas(ctx, empresaID, filter)
}

// GetByID methods delegate to repository.
func (s *SalesService) GetPresupuesto(ctx context.Context, id uuid.UUID) (*Presupuesto, error) {
	return s.repo.GetPresupuesto(ctx, id)
}

func (s *SalesService) GetPedido(ctx context.Context, id uuid.UUID) (*Pedido, error) {
	return s.repo.GetPedido(ctx, id)
}

func (s *SalesService) GetAlbaran(ctx context.Context, id uuid.UUID) (*Albaran, error) {
	return s.repo.GetAlbaran(ctx, id)
}

func (s *SalesService) GetFactura(ctx context.Context, id uuid.UUID) (*Factura, error) {
	return s.repo.GetFactura(ctx, id)
}

// Update and Cancel methods
func (s *SalesService) UpdatePresupuesto(ctx context.Context, input UpdatePresupuestoInput) error {
	doc, err := s.repo.GetPresupuesto(ctx, input.ID)
	if err != nil {
		return err
	}
	if doc.Estado == StatusConverted || doc.Estado == StatusCancelled {
		return fmt.Errorf("%w: cannot update a %s document", ErrInvalidStatus, doc.Estado)
	}
	return s.repo.UpdatePresupuesto(ctx, input)
}

func (s *SalesService) UpdatePedido(ctx context.Context, input UpdatePedidoInput) error {
	doc, err := s.repo.GetPedido(ctx, input.ID)
	if err != nil {
		return err
	}
	if doc.Estado == StatusConverted || doc.Estado == StatusCancelled || doc.Estado == StatusParcial {
		return fmt.Errorf("%w: cannot update a %s document", ErrInvalidStatus, doc.Estado)
	}
	return s.repo.UpdatePedido(ctx, input)
}

func (s *SalesService) UpdateAlbaran(ctx context.Context, input UpdateAlbaranInput) error {
	doc, err := s.repo.GetAlbaran(ctx, input.ID)
	if err != nil {
		return err
	}
	if doc.Estado == StatusConverted || doc.Estado == StatusCancelled {
		return fmt.Errorf("%w: cannot update a %s document", ErrInvalidStatus, doc.Estado)
	}
	return s.repo.UpdateAlbaran(ctx, input)
}

func (s *SalesService) CancelPresupuesto(ctx context.Context, empresaID, quoteID uuid.UUID) error {
	q, err := s.repo.GetPresupuesto(ctx, quoteID)
	if err != nil {
		return err
	}
	if q.EmpresaID != empresaID {
		return ErrTenantMismatch
	}
	if q.Estado == StatusCancelled {
		return fmt.Errorf("%w: quote is already cancelled", ErrInvalidStatus)
	}
	if q.Estado == StatusConverted {
		return fmt.Errorf("%w: cannot cancel a converted quote", ErrInvalidStatus)
	}
	return s.repo.CancelPresupuesto(ctx, quoteID)
}

func (s *SalesService) CancelPedido(ctx context.Context, empresaID, orderID uuid.UUID) error {
	o, err := s.repo.GetPedido(ctx, orderID)
	if err != nil {
		return err
	}
	if o.EmpresaID != empresaID {
		return ErrTenantMismatch
	}
	if o.Estado == StatusCancelled {
		return fmt.Errorf("%w: order is already cancelled", ErrInvalidStatus)
	}
	if o.Estado == StatusConverted {
		return fmt.Errorf("%w: cannot cancel a converted order", ErrInvalidStatus)
	}
	return s.repo.CancelPedido(ctx, orderID)
}

func (s *SalesService) CancelAlbaran(ctx context.Context, empresaID, dnID uuid.UUID) error {
	dn, err := s.repo.GetAlbaran(ctx, dnID)
	if err != nil {
		return err
	}
	if dn.EmpresaID != empresaID {
		return ErrTenantMismatch
	}
	if dn.Estado == StatusCancelled {
		return fmt.Errorf("%w: delivery note is already cancelled", ErrInvalidStatus)
	}
	if dn.Estado == StatusConverted {
		return fmt.Errorf("%w: cannot cancel a converted delivery note", ErrInvalidStatus)
	}
	if dn.Estado == StatusProcessed {
		// Reverse stock withdrawals before cancelling
		refDocType := "ALBARAN_CANCEL"
		for _, l := range dn.Lineas {
			_, err := s.invService.RecordReturn(ctx, l.ProductoID, dn.AlmacenID, l.Cantidad, &refDocType, &dn.ID)
			if err != nil {
				return fmt.Errorf("failed to reverse stock for product %s: %w", l.ProductoID, err)
			}
		}
	}
	return s.repo.CancelAlbaran(ctx, dnID)
}

func (s *SalesService) CancelFactura(ctx context.Context, empresaID, invoiceID uuid.UUID) error {
	inv, err := s.repo.GetFactura(ctx, invoiceID)
	if err != nil {
		return err
	}
	if inv.EmpresaID != empresaID {
		return ErrTenantMismatch
	}
	if inv.Estado == StatusCancelled {
		return fmt.Errorf("%w: invoice is already cancelled", ErrInvalidStatus)
	}
	return s.repo.CancelFactura(ctx, invoiceID)
}

// CreatePresupuesto creates a new quote.
func (s *SalesService) CreatePresupuesto(ctx context.Context, empresaID, clienteID uuid.UUID, expiresAt time.Time, lines []PresupuestoLinea) (*Presupuesto, error) {
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
		if l.CosteUnitario < 0 {
			return nil, fmt.Errorf("line %d: coste_unitario must be non-negative", i)
		}
	}

	qID := uuid.New()
	var total float64
	qLines := make([]PresupuestoLinea, len(lines))
	for i, l := range lines {
		total += l.Cantidad * l.PrecioUnitario
		qLines[i] = PresupuestoLinea{
			ID:             uuid.New(),
			PresupuestoID:        qID,
			ProductoID:     l.ProductoID,
			Cantidad:       l.Cantidad,
			PrecioUnitario: l.PrecioUnitario,
			CosteUnitario:  l.CosteUnitario,
		}
	}

	q := &Presupuesto{
		ID:        qID,
		EmpresaID: empresaID,
		ClienteID:  clienteID,
		Total:     total,
		Estado:    StatusDraft,
		FechaValidez: expiresAt,
		CreatedAt: s.now(),
		Lineas:    qLines,
	}

	if err := s.repo.SavePresupuesto(ctx, q); err != nil {
		return nil, err
	}
	return q, nil
}

// CreatePedido creates a new order.
func (s *SalesService) CreatePedido(ctx context.Context, empresaID uuid.UUID, quoteID *uuid.UUID, lines []PedidoLinea) (*Pedido, error) {
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

	oID := uuid.New()
	var total float64
	oLines := make([]PedidoLinea, len(lines))
	for i, l := range lines {
		total += l.Cantidad * l.PrecioUnitario
		oLines[i] = PedidoLinea{
			ID:             uuid.New(),
			PedidoID:        oID,
			ProductoID:     l.ProductoID,
			Cantidad:       l.Cantidad,
			PrecioUnitario: l.PrecioUnitario,
		}
	}

	o := &Pedido{
		ID:        oID,
		EmpresaID: empresaID,
		PresupuestoID:   quoteID,
		Total:     total,
		Estado:    StatusDraft,
		CreatedAt: s.now(),
		Lineas:    oLines,
	}

	if err := s.repo.SavePedido(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}

// ConvertPresupuestoToPedido transitions an Approved or Draft Presupuesto to a Pedido.
// Supports partial conversion: only specified lines (or all remaining if Lineas is nil).
func (s *SalesService) ConvertPresupuestoToPedido(ctx context.Context, empresaID uuid.UUID, input ConvertPresupuestoInput) (*Pedido, error) {
	quote, err := s.repo.GetPresupuesto(ctx, input.PresupuestoID)
	if err != nil {
		return nil, err
	}
	if quote.EmpresaID != empresaID {
		return nil, ErrTenantMismatch
	}

	if quote.Estado == StatusConverted {
		return nil, ErrDocumentAlreadyConverted
	}
	if quote.Estado == StatusCancelled {
		return nil, ErrDocumentAlreadyCancelled
	}

	isExpired := false
	if !quote.FechaValidez.IsZero() && quote.FechaValidez.Before(s.now()) {
		isExpired = true
	}

	if isExpired {
		if s.securityService == nil {
			return nil, ErrSecurityServiceNil
		}
		authorized, err := s.securityService.HasPermission(ctx, input.UserID, "convert-expired-quote")
		if err != nil {
			return nil, err
		}
		if !authorized {
			return nil, ErrUnauthorized
		}
	}

	// Build conversion lines
	var convLines []ConversionLineInput
	if len(input.Lineas) == 0 {
		// Convert all remaining
		for _, l := range quote.Lineas {
			remaining := l.Cantidad - l.Convertido
			if remaining > 0 {
				convLines = append(convLines, ConversionLineInput{
					ProductoID: l.ProductoID,
					Cantidad:   remaining,
				})
			}
		}
	} else {
		// Validate specified lines
		for _, cl := range input.Lineas {
			if cl.Cantidad <= 0 {
				return nil, fmt.Errorf("conversion quantity must be positive for product %s", cl.ProductoID)
			}
			found := false
			for _, l := range quote.Lineas {
				if l.ProductoID == cl.ProductoID {
					found = true
					remaining := l.Cantidad - l.Convertido
					if cl.Cantidad > remaining {
						return nil, fmt.Errorf("requested quantity %f exceeds remaining %f for product %s", cl.Cantidad, remaining, cl.ProductoID)
					}
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("product %s not found on quote", cl.ProductoID)
			}
			convLines = append(convLines, cl)
		}
	}

	if len(convLines) == 0 {
		return nil, fmt.Errorf("no lines to convert: all quantities already converted")
	}

	// Build pedido lines and calculate total
	oID := uuid.New()
	var total float64
	orderLines := make([]PedidoLinea, len(convLines))
	for i, cl := range convLines {
		// Find the quote line for pricing
		var price float64
		for _, l := range quote.Lineas {
			if l.ProductoID == cl.ProductoID {
				price = l.PrecioUnitario
				if isExpired && input.RecalculatePrices {
					price = l.PrecioUnitario * 1.10
				}
				break
			}
		}
		total += cl.Cantidad * price
		orderLines[i] = PedidoLinea{
			ID:             uuid.New(),
			PedidoID:       oID,
			ProductoID:     cl.ProductoID,
			Cantidad:       cl.Cantidad,
			PrecioUnitario: price,
		}
	}

	// Update quote line Convertido values
	for _, cl := range convLines {
		for i, l := range quote.Lineas {
			if l.ProductoID == cl.ProductoID {
				quote.Lineas[i].Convertido += cl.Cantidad
				break
			}
		}
	}

	// Determine new quote status
	allConverted := true
	for _, l := range quote.Lineas {
		if l.Convertido < l.Cantidad {
			allConverted = false
			break
		}
	}
	if allConverted {
		quote.Estado = StatusConverted
	} else {
		quote.Estado = StatusParcial
	}

	if isExpired && input.RecalculatePrices {
		// Recalculate quote total based on remaining
		var newTotal float64
		for _, l := range quote.Lineas {
			newTotal += l.Cantidad * l.PrecioUnitario * 1.10
		}
		quote.Total = newTotal
	}

	if err := s.repo.SavePresupuesto(ctx, quote); err != nil {
		return nil, err
	}

	order := &Pedido{
		ID:            oID,
		EmpresaID:     empresaID,
		PresupuestoID: &quote.ID,
		Total:         total,
		Estado:        StatusDraft,
		CreatedAt:     s.now(),
		Lineas:        orderLines,
	}

	if err := s.repo.SavePedido(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}

// ConvertPedidoToAlbaran transitions a Pedido to an Albaran in Draft status.
// Supports partial conversion: only specified lines (or all remaining if Lineas is nil).
func (s *SalesService) ConvertPedidoToAlbaran(ctx context.Context, empresaID uuid.UUID, input ConvertPedidoInput) (*Albaran, error) {
	order, err := s.repo.GetPedido(ctx, input.PedidoID)
	if err != nil {
		return nil, err
	}
	if order.EmpresaID != empresaID {
		return nil, ErrTenantMismatch
	}

	if order.Estado == StatusConverted {
		return nil, ErrDocumentAlreadyConverted
	}
	if order.Estado == StatusCancelled {
		return nil, ErrDocumentAlreadyCancelled
	}

	// Build conversion lines
	var convLines []ConversionLineInput
	if len(input.Lineas) == 0 {
		// Convert all remaining
		for _, l := range order.Lineas {
			remaining := l.Cantidad - l.Entregado
			if remaining > 0 {
				convLines = append(convLines, ConversionLineInput{
					ProductoID: l.ProductoID,
					Cantidad:   remaining,
				})
			}
		}
	} else {
		// Validate specified lines
		for _, cl := range input.Lineas {
			if cl.Cantidad <= 0 {
				return nil, fmt.Errorf("conversion quantity must be positive for product %s", cl.ProductoID)
			}
			found := false
			for _, l := range order.Lineas {
				if l.ProductoID == cl.ProductoID {
					found = true
					remaining := l.Cantidad - l.Entregado
					if cl.Cantidad > remaining {
						return nil, fmt.Errorf("requested quantity %f exceeds remaining %f for product %s", cl.Cantidad, remaining, cl.ProductoID)
					}
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("product %s not found on order", cl.ProductoID)
			}
			convLines = append(convLines, cl)
		}
	}

	if len(convLines) == 0 {
		return nil, fmt.Errorf("no lines to convert: all quantities already delivered")
	}

	// Build albarán lines and calculate total
	dnID := uuid.New()
	var total float64
	dnLines := make([]AlbaranLinea, len(convLines))
	for i, cl := range convLines {
		var price float64
		for _, l := range order.Lineas {
			if l.ProductoID == cl.ProductoID {
				price = l.PrecioUnitario
				break
			}
		}
		total += cl.Cantidad * price
		dnLines[i] = AlbaranLinea{
			ID:             uuid.New(),
			AlbaranID:      dnID,
			ProductoID:     cl.ProductoID,
			Cantidad:       cl.Cantidad,
			PrecioUnitario: price,
		}
	}

	// Update pedido line Entregado values
	for _, cl := range convLines {
		for i, l := range order.Lineas {
			if l.ProductoID == cl.ProductoID {
				order.Lineas[i].Entregado += cl.Cantidad
				break
			}
		}
	}

	// Determine new pedido status
	allDelivered := true
	for _, l := range order.Lineas {
		if l.Entregado < l.Cantidad {
			allDelivered = false
			break
		}
	}
	if allDelivered {
		order.Estado = StatusConverted
	} else {
		order.Estado = StatusParcial
	}

	if err := s.repo.SavePedido(ctx, order); err != nil {
		return nil, err
	}

	dn := &Albaran{
		ID:        dnID,
		EmpresaID: empresaID,
		PedidoID:  &order.ID,
		Total:     total,
		Estado:    StatusDraft,
		AlmacenID: input.AlmacenID,
		CreatedAt: s.now(),
		Lineas:    dnLines,
	}

	if err := s.repo.SaveAlbaran(ctx, dn); err != nil {
		return nil, err
	}

	return dn, nil
}

// ProcessAlbaran processes a delivery note, deducting stock from the stock ledger.
func (s *SalesService) ProcessAlbaran(ctx context.Context, empresaID, dnID uuid.UUID) error {
	dn, err := s.repo.GetAlbaran(ctx, dnID)
	if err != nil {
		return err
	}
	if dn.EmpresaID != empresaID {
		return ErrTenantMismatch
	}
	if dn.Estado != StatusDraft {
		return fmt.Errorf("%w: cannot process from %s status", ErrInvalidStatus, dn.Estado)
	}

	refDocType := "ALBARAN"
	// Record stock withdrawals atomically — roll back all on partial failure
	items := make([]inventorydomain.WithdrawalItem, len(dn.Lineas))
	for i, l := range dn.Lineas {
		items[i] = inventorydomain.WithdrawalItem{
			ItemID:      l.ProductoID,
			WarehouseID: dn.AlmacenID,
			Qty:         l.Cantidad,
		}
	}
	if _, err := s.invService.RecordWithdrawals(ctx, items, &refDocType, &dn.ID); err != nil {
		return err
	}

	dn.Estado = StatusProcessed
	return s.repo.SaveAlbaran(ctx, dn)
}

// ConvertAlbaranToFactura transitions an Albaran to a Factura.
// Supports partial conversion: only specified lines (or all remaining if Lineas is nil).
func (s *SalesService) ConvertAlbaranToFactura(ctx context.Context, empresaID uuid.UUID, input ConvertAlbaranInput) (*Factura, error) {
	dn, err := s.repo.GetAlbaran(ctx, input.AlbaranID)
	if err != nil {
		return nil, err
	}
	if dn.EmpresaID != empresaID {
		return nil, ErrTenantMismatch
	}

	if dn.Estado == StatusConverted {
		return nil, ErrDocumentAlreadyConverted
	}
	if dn.Estado == StatusCancelled {
		return nil, ErrDocumentAlreadyCancelled
	}

	// For billing, the delivery note must be processed first to assure delivery
	if dn.Estado != StatusProcessed {
		return nil, fmt.Errorf("%w: delivery note must be Processed before invoicing, currently %s", ErrInvalidStatus, dn.Estado)
	}

	if s.billingService == nil {
		return nil, ErrBillingServiceNil
	}

	// Build conversion lines
	var convLines []ConversionLineInput
	if len(input.Lineas) == 0 {
		// Convert all remaining
		for _, l := range dn.Lineas {
			remaining := l.Cantidad - l.Facturado
			if remaining > 0 {
				convLines = append(convLines, ConversionLineInput{
					ProductoID: l.ProductoID,
					Cantidad:   remaining,
				})
			}
		}
	} else {
		// Validate specified lines
		for _, cl := range input.Lineas {
			if cl.Cantidad <= 0 {
				return nil, fmt.Errorf("conversion quantity must be positive for product %s", cl.ProductoID)
			}
			found := false
			for _, l := range dn.Lineas {
				if l.ProductoID == cl.ProductoID {
					found = true
					remaining := l.Cantidad - l.Facturado
					if cl.Cantidad > remaining {
						return nil, fmt.Errorf("requested quantity %f exceeds remaining %f for product %s", cl.Cantidad, remaining, cl.ProductoID)
					}
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("product %s not found on delivery note", cl.ProductoID)
			}
			convLines = append(convLines, cl)
		}
	}

	if len(convLines) == 0 {
		return nil, fmt.Errorf("no lines to convert: all quantities already invoiced")
	}

	invoiceNumber, seq, err := s.billingService.GenerateFacturaNumber(ctx, input.TerminalID)
	if err != nil {
		return nil, err
	}

	// Build invoice lines and calculate total
	invID := uuid.New()
	var total float64
	invLines := make([]FacturaLinea, len(convLines))
	for i, cl := range convLines {
		var price float64
		for _, l := range dn.Lineas {
			if l.ProductoID == cl.ProductoID {
				price = l.PrecioUnitario
				break
			}
		}
		total += cl.Cantidad * price
		invLines[i] = FacturaLinea{
			ID:             uuid.New(),
			FacturaID:      invID,
			ProductoID:     cl.ProductoID,
			Cantidad:       cl.Cantidad,
			PrecioUnitario: price,
		}
	}

	// Update albarán line Facturado values
	for _, cl := range convLines {
		for i, l := range dn.Lineas {
			if l.ProductoID == cl.ProductoID {
				dn.Lineas[i].Facturado += cl.Cantidad
				break
			}
		}
	}

	// Determine new albarán status
	allInvoiced := true
	for _, l := range dn.Lineas {
		if l.Facturado < l.Cantidad {
			allInvoiced = false
			break
		}
	}
	if allInvoiced {
		dn.Estado = StatusConverted
	} else {
		dn.Estado = StatusParcial
	}

	if err := s.repo.SaveAlbaran(ctx, dn); err != nil {
		return nil, err
	}

	invoice := &Factura{
		ID:                 invID,
		EmpresaID:          empresaID,
		AlbaranID:          &dn.ID,
		TerminalID:         input.TerminalID,
		SerieFacturacionID: input.SerieFacturacionID,
		NumeroFactura:      invoiceNumber,
		NumeroSecuencia:    seq,
		Total:              total,
		RectifiedTotal:     0,
		Estado:             StatusIssued,
		CreatedAt:          s.now(),
		Lineas:             invLines,
	}

	if err := s.repo.SaveFactura(ctx, invoice); err != nil {
		return nil, err
	}

	return invoice, nil
}

// CreateFacturaRectificativa creates a rectifying invoice to reverse an invoice and returns stock.
func (s *SalesService) CreateFacturaRectificativa(ctx context.Context, empresaID uuid.UUID, invoiceID uuid.UUID, reason string, lines []FacturaRectificativaLineaInput, terminalID *uuid.UUID) (*FacturaRectificativa, error) {
	// 1. Fetch invoice and validate
	inv, err := s.repo.GetFactura(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv.EmpresaID != empresaID {
		return nil, ErrTenantMismatch
	}
	if inv.Estado == StatusCancelled {
		return nil, ErrCannotRectifyCancelled
	}
	if inv.RectifiedTotal >= inv.Total {
		return nil, ErrFacturaAlreadyRectified
	}

	// 2. Get warehouse from delivery note chain
	if inv.AlbaranID == nil {
		return nil, ErrFacturaNoAlbaran
	}
	dn, err := s.repo.GetAlbaran(ctx, *inv.AlbaranID)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery note for invoice: %w", err)
	}

	// 3. Get already-rectified quantities for this invoice
	rectifiedQtys, err := s.repo.GetRectifiedQuantitiesByInvoice(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rectified quantities: %w", err)
	}

	// 4. Validate lines against invoice
	var total float64
	for _, l := range lines {
		if l.Cantidad <= 0 {
			return nil, fmt.Errorf("rectification quantity must be positive for product %s", l.ProductoID)
		}
		total += l.Cantidad * l.PrecioUnitario

		// Check product exists on invoice
		found := false
		for _, il := range inv.Lineas {
			if il.ProductoID == l.ProductoID {
				found = true
				remaining := il.Cantidad - rectifiedQtys[il.ProductoID]
				if l.Cantidad > remaining {
					return nil, fmt.Errorf("%w: product %s: rectified %f, remaining %f", ErrQuantityExceedsFactura, l.ProductoID, l.Cantidad, remaining)
				}
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("%w: %s", ErrProductNotOnFactura, l.ProductoID)
		}
	}
	if total <= 0 {
		return nil, errors.New("total de factura rectificativa debe ser positivo")
	}

	// 5. Generate FR number (shared series with invoices)
	if s.billingService == nil {
		return nil, ErrBillingServiceNil
	}
	if terminalID == nil {
		return nil, ErrTerminalRequired
	}

	frNumber, seq, err := s.billingService.GenerateFacturaNumber(ctx, *terminalID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate FR number: %w", err)
	}

	// 6. Build rectifying invoice
	frID := uuid.New()
	refDocType := "FACTURA_RECTIFICATIVA"

	frLines := make([]FacturaRectificativaLinea, len(lines))
	for i, l := range lines {
		frLines[i] = FacturaRectificativaLinea{
			ID:              uuid.New(),
			RectificativaID: frID,
			ProductoID:      l.ProductoID,
			Cantidad:        l.Cantidad,
			PrecioUnitario:  l.PrecioUnitario,
		}
	}

	fr := &FacturaRectificativa{
		ID:             frID,
		FacturaID:      invoiceID,
		EmpresaID:      empresaID,
		TerminalID:     terminalID,
		NumeroFR:       frNumber,
		NumeroSecuencia: seq,
		Total:          total,
		Motivo:         reason,
		Estado:         StatusIssued,
		CreatedAt:      s.now(),
		Lines:          frLines,
	}

	// 7. Record stock return movements (before saving document so ref ID is available)
	for _, l := range lines {
		_, err := s.invService.RecordReturn(ctx, l.ProductoID, dn.AlmacenID, l.Cantidad, &refDocType, &frID)
		if err != nil {
			return nil, fmt.Errorf("failed to record stock return for product %s: %w", l.ProductoID, err)
		}
	}

	// 8. Save FR document
	if err := s.repo.CreateFacturaRectificativa(ctx, fr); err != nil {
		return nil, err
	}

	// 9. Update invoice rectified_total
	newRectifiedTotal := inv.RectifiedTotal + total
	if err := s.repo.UpdateFacturaRectifiedTotal(ctx, invoiceID, newRectifiedTotal); err != nil {
		return nil, err
	}

	// 10. If fully rectified, update invoice status
	if newRectifiedTotal >= inv.Total {
		inv.Estado = StatusRectified
		if err := s.repo.SaveFactura(ctx, inv); err != nil {
			return nil, err
		}
	}

	return fr, nil
}

// GetFacturaRectificativa retrieves a rectifying invoice by ID.
func (s *SalesService) GetFacturaRectificativa(ctx context.Context, id uuid.UUID) (*FacturaRectificativa, error) {
	return s.repo.GetFacturaRectificativa(ctx, id)
}

// ListFacturasRectificativas lists rectifying invoices for a given empresa.
func (s *SalesService) ListFacturasRectificativas(ctx context.Context, empresaID uuid.UUID) ([]FacturaRectificativa, error) {
	return s.repo.ListFacturasRectificativas(ctx, empresaID)
}
