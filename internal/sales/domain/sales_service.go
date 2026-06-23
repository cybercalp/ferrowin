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
	ErrQuoteExpired             = errors.New("cannot convert: quote is expired")
	ErrUnauthorized             = errors.New("cannot convert: user is not authorized to convert expired quotes")
	ErrSecurityServiceNil       = errors.New("security service is required but not configured")
	ErrBillingServiceNil        = errors.New("billing service is required but not configured")
	ErrTenantMismatch           = errors.New("tenant company mismatch")
	ErrInvalidStatus            = errors.New("invalid status transition")
	ErrQuoteNotFound            = errors.New("quote not found")
	ErrOrderNotFound            = errors.New("order not found")
	ErrDeliveryNoteNotFound     = errors.New("delivery note not found")
	ErrInvoiceNotFound          = errors.New("invoice not found")
)

// SecurityServiceRequired defines the security permission check contract required by this domain service.
type SecurityServiceRequired interface {
	HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error)
}

// BillingServiceRequired defines the sequence generation contract required by this domain service.
type BillingServiceRequired interface {
	GenerateInvoiceNumber(ctx context.Context, terminalID uuid.UUID) (string, int, error)
}

type SalesRepository interface {
	SaveQuote(ctx context.Context, q *Quote) error
	GetQuote(ctx context.Context, id uuid.UUID) (*Quote, error)

	SaveOrder(ctx context.Context, o *Order) error
	GetOrder(ctx context.Context, id uuid.UUID) (*Order, error)

	SaveDeliveryNote(ctx context.Context, dn *DeliveryNote) error
	GetDeliveryNote(ctx context.Context, id uuid.UUID) (*DeliveryNote, error)

	SaveInvoice(ctx context.Context, inv *Invoice) error
	GetInvoice(ctx context.Context, id uuid.UUID) (*Invoice, error)
}

// ConvertQuoteOptions specifies options when converting a quote.
type ConvertQuoteOptions struct {
	RecalculatePrices bool
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

// CreateQuote creates a new quote.
func (s *SalesService) CreateQuote(ctx context.Context, empresaID, clientID uuid.UUID, expiresAt time.Time, lines []QuoteLine) (*Quote, error) {
	qID := uuid.New()
	var total float64
	qLines := make([]QuoteLine, len(lines))
	for i, l := range lines {
		total += l.Cantidad * l.PrecioUnitario
		qLines[i] = QuoteLine{
			ID:             uuid.New(),
			QuoteID:        qID,
			ProductoID:     l.ProductoID,
			Cantidad:       l.Cantidad,
			PrecioUnitario: l.PrecioUnitario,
			CosteUnitario:  l.CosteUnitario,
		}
	}

	q := &Quote{
		ID:        qID,
		EmpresaID: empresaID,
		ClientID:  clientID,
		Total:     total,
		Status:    StatusDraft,
		ExpiresAt: expiresAt,
		CreatedAt: s.now(),
		Lineas:    qLines,
	}

	if err := s.repo.SaveQuote(ctx, q); err != nil {
		return nil, err
	}
	return q, nil
}

// CreateOrder creates a new order.
func (s *SalesService) CreateOrder(ctx context.Context, empresaID uuid.UUID, quoteID *uuid.UUID, lines []OrderLine) (*Order, error) {
	oID := uuid.New()
	var total float64
	oLines := make([]OrderLine, len(lines))
	for i, l := range lines {
		total += l.Cantidad * l.PrecioUnitario
		oLines[i] = OrderLine{
			ID:             uuid.New(),
			OrderID:        oID,
			ProductoID:     l.ProductoID,
			Cantidad:       l.Cantidad,
			PrecioUnitario: l.PrecioUnitario,
		}
	}

	o := &Order{
		ID:        oID,
		EmpresaID: empresaID,
		QuoteID:   quoteID,
		Total:     total,
		Status:    StatusDraft,
		CreatedAt: s.now(),
		Lineas:    oLines,
	}

	if err := s.repo.SaveOrder(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}

// ConvertQuoteToOrder transitions an Approved or Draft Quote to an Order.
func (s *SalesService) ConvertQuoteToOrder(ctx context.Context, empresaID, quoteID, userID uuid.UUID, opt ConvertQuoteOptions) (*Order, error) {
	quote, err := s.repo.GetQuote(ctx, quoteID)
	if err != nil {
		return nil, err
	}
	if quote.EmpresaID != empresaID {
		return nil, ErrTenantMismatch
	}

	if quote.Status == StatusConverted {
		return nil, ErrDocumentAlreadyConverted
	}
	if quote.Status == StatusCancelled {
		return nil, ErrDocumentAlreadyCancelled
	}

	isExpired := false
	if !quote.ExpiresAt.IsZero() && quote.ExpiresAt.Before(s.now()) {
		isExpired = true
	}

	if isExpired {
		if s.securityService == nil {
			return nil, ErrSecurityServiceNil
		}
		authorized, err := s.securityService.HasPermission(ctx, userID, "convert-expired-quote")
		if err != nil {
			return nil, err
		}
		if !authorized {
			return nil, ErrUnauthorized
		}
	}

	total := quote.Total
	if isExpired && opt.RecalculatePrices {
		total = quote.Total * 1.10
	}

	quote.Status = StatusConverted
	if err := s.repo.SaveQuote(ctx, quote); err != nil {
		return nil, err
	}

	// Create Order
	oID := uuid.New()
	orderLines := make([]OrderLine, len(quote.Lineas))
	for i, l := range quote.Lineas {
		price := l.PrecioUnitario
		if isExpired && opt.RecalculatePrices {
			price = l.PrecioUnitario * 1.10
		}
		orderLines[i] = OrderLine{
			ID:             uuid.New(),
			OrderID:        oID,
			ProductoID:     l.ProductoID,
			Cantidad:       l.Cantidad,
			PrecioUnitario: price,
		}
	}

	order := &Order{
		ID:        oID,
		EmpresaID: empresaID,
		QuoteID:   &quote.ID,
		Total:     total,
		Status:    StatusDraft,
		CreatedAt: s.now(),
		Lineas:    orderLines,
	}

	if err := s.repo.SaveOrder(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}

// ConvertOrderToDeliveryNote transitions an Order to a DeliveryNote in Draft status.
func (s *SalesService) ConvertOrderToDeliveryNote(ctx context.Context, empresaID, orderID, warehouseID uuid.UUID) (*DeliveryNote, error) {
	order, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.EmpresaID != empresaID {
		return nil, ErrTenantMismatch
	}

	if order.Status == StatusConverted {
		return nil, ErrDocumentAlreadyConverted
	}
	if order.Status == StatusCancelled {
		return nil, ErrDocumentAlreadyCancelled
	}

	order.Status = StatusConverted
	if err := s.repo.SaveOrder(ctx, order); err != nil {
		return nil, err
	}

	dnID := uuid.New()
	dnLines := make([]DeliveryNoteLinea, len(order.Lineas))
	for i, l := range order.Lineas {
		dnLines[i] = DeliveryNoteLinea{
			ID:             uuid.New(),
			DeliveryNoteID: dnID,
			ProductoID:     l.ProductoID,
			Cantidad:       l.Cantidad,
			PrecioUnitario: l.PrecioUnitario,
		}
	}

	dn := &DeliveryNote{
		ID:          dnID,
		EmpresaID:   empresaID,
		OrderID:     &order.ID,
		Total:       order.Total,
		Status:      StatusDraft,
		WarehouseID: warehouseID,
		CreatedAt:   s.now(),
		Lineas:      dnLines,
	}

	if err := s.repo.SaveDeliveryNote(ctx, dn); err != nil {
		return nil, err
	}

	return dn, nil
}

// ProcessDeliveryNote processes a delivery note, deducting stock from the stock ledger.
func (s *SalesService) ProcessDeliveryNote(ctx context.Context, empresaID, dnID uuid.UUID) error {
	dn, err := s.repo.GetDeliveryNote(ctx, dnID)
	if err != nil {
		return err
	}
	if dn.EmpresaID != empresaID {
		return ErrTenantMismatch
	}
	if dn.Status != StatusDraft {
		return fmt.Errorf("%w: cannot process from %s status", ErrInvalidStatus, dn.Status)
	}

	refDocType := "DELIVERY_NOTE"
	// Record stock withdrawals
	for _, l := range dn.Lineas {
		_, err := s.invService.RecordWithdrawal(ctx, l.ProductoID, dn.WarehouseID, l.Cantidad, &refDocType, &dn.ID)
		if err != nil {
			return fmt.Errorf("failed to deduct stock for product %s: %w", l.ProductoID, err)
		}
	}

	dn.Status = StatusProcessed
	return s.repo.SaveDeliveryNote(ctx, dn)
}

// ConvertDeliveryNoteToInvoice transitions a DeliveryNote to an Invoice.
func (s *SalesService) ConvertDeliveryNoteToInvoice(ctx context.Context, empresaID, dnID, terminalID, invoicingSeriesID uuid.UUID) (*Invoice, error) {
	dn, err := s.repo.GetDeliveryNote(ctx, dnID)
	if err != nil {
		return nil, err
	}
	if dn.EmpresaID != empresaID {
		return nil, ErrTenantMismatch
	}

	if dn.Status == StatusConverted {
		return nil, ErrDocumentAlreadyConverted
	}
	if dn.Status == StatusCancelled {
		return nil, ErrDocumentAlreadyCancelled
	}

	// For billing, the delivery note must be processed first to assure delivery
	if dn.Status != StatusProcessed {
		return nil, fmt.Errorf("%w: delivery note must be Processed before invoicing, currently %s", ErrInvalidStatus, dn.Status)
	}

	if s.billingService == nil {
		return nil, ErrBillingServiceNil
	}

	invoiceNumber, seq, err := s.billingService.GenerateInvoiceNumber(ctx, terminalID)
	if err != nil {
		return nil, err
	}

	dn.Status = StatusConverted
	if err := s.repo.SaveDeliveryNote(ctx, dn); err != nil {
		return nil, err
	}

	invID := uuid.New()
	invLines := make([]InvoiceLinea, len(dn.Lineas))
	for i, l := range dn.Lineas {
		invLines[i] = InvoiceLinea{
			ID:             uuid.New(),
			InvoiceID:      invID,
			ProductoID:     l.ProductoID,
			Cantidad:       l.Cantidad,
			PrecioUnitario: l.PrecioUnitario,
		}
	}

	invoice := &Invoice{
		ID:                invID,
		EmpresaID:         empresaID,
		DeliveryNoteID:    &dn.ID,
		TerminalID:        terminalID,
		InvoicingSeriesID: invoicingSeriesID,
		InvoiceNumber:     invoiceNumber,
		SequenceNumber:    seq,
		Total:             dn.Total,
		Status:            StatusIssued,
		CreatedAt:         s.now(),
		Lineas:            invLines,
	}

	if err := s.repo.SaveInvoice(ctx, invoice); err != nil {
		return nil, err
	}

	return invoice, nil
}
