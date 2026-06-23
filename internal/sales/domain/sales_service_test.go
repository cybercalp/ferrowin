package domain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockSalesRepository struct {
	saveQuoteFunc       func(ctx context.Context, q *Quote) error
	getQuoteFunc        func(ctx context.Context, id uuid.UUID) (*Quote, error)
	saveOrderFunc       func(ctx context.Context, o *Order) error
	getOrderFunc        func(ctx context.Context, id uuid.UUID) (*Order, error)
	saveDNFunc          func(ctx context.Context, dn *DeliveryNote) error
	getDNFunc           func(ctx context.Context, id uuid.UUID) (*DeliveryNote, error)
	saveInvFunc         func(ctx context.Context, inv *Invoice) error
	getInvFunc          func(ctx context.Context, id uuid.UUID) (*Invoice, error)
	updateQuoteFunc     func(ctx context.Context, input UpdateQuoteInput) error
	updateOrderFunc     func(ctx context.Context, input UpdateOrderInput) error
	updateDNFunc        func(ctx context.Context, input UpdateDeliveryNoteInput) error
	cancelQuoteFunc     func(ctx context.Context, id uuid.UUID) error
	cancelOrderFunc     func(ctx context.Context, id uuid.UUID) error
	cancelDNFunc        func(ctx context.Context, id uuid.UUID) error
	cancelInvoiceFunc   func(ctx context.Context, id uuid.UUID) error
}

func (m *mockSalesRepository) SaveQuote(ctx context.Context, q *Quote) error {
	if m.saveQuoteFunc != nil { return m.saveQuoteFunc(ctx, q) }
	return nil
}
func (m *mockSalesRepository) GetQuote(ctx context.Context, id uuid.UUID) (*Quote, error) {
	if m.getQuoteFunc != nil { return m.getQuoteFunc(ctx, id) }
	return nil, nil
}
func (m *mockSalesRepository) SaveOrder(ctx context.Context, o *Order) error {
	if m.saveOrderFunc != nil { return m.saveOrderFunc(ctx, o) }
	return nil
}
func (m *mockSalesRepository) GetOrder(ctx context.Context, id uuid.UUID) (*Order, error) {
	if m.getOrderFunc != nil { return m.getOrderFunc(ctx, id) }
	return nil, nil
}
func (m *mockSalesRepository) SaveDeliveryNote(ctx context.Context, dn *DeliveryNote) error {
	if m.saveDNFunc != nil { return m.saveDNFunc(ctx, dn) }
	return nil
}
func (m *mockSalesRepository) GetDeliveryNote(ctx context.Context, id uuid.UUID) (*DeliveryNote, error) {
	if m.getDNFunc != nil { return m.getDNFunc(ctx, id) }
	return nil, nil
}
func (m *mockSalesRepository) SaveInvoice(ctx context.Context, inv *Invoice) error {
	if m.saveInvFunc != nil { return m.saveInvFunc(ctx, inv) }
	return nil
}
func (m *mockSalesRepository) GetInvoice(ctx context.Context, id uuid.UUID) (*Invoice, error) {
	if m.getInvFunc != nil { return m.getInvFunc(ctx, id) }
	return nil, nil
}

func (m *mockSalesRepository) ListQuotes(ctx context.Context, empresaID uuid.UUID, filter DocumentFilter) ([]*Quote, int, error) {
	return nil, 0, nil
}

func (m *mockSalesRepository) ListOrders(ctx context.Context, empresaID uuid.UUID, filter DocumentFilter) ([]*Order, int, error) {
	return nil, 0, nil
}

func (m *mockSalesRepository) ListDeliveryNotes(ctx context.Context, empresaID uuid.UUID, filter DocumentFilter) ([]*DeliveryNote, int, error) {
	return nil, 0, nil
}

func (m *mockSalesRepository) ListInvoices(ctx context.Context, empresaID uuid.UUID, filter DocumentFilter) ([]*Invoice, int, error) {
	return nil, 0, nil
}

func (m *mockSalesRepository) UpdateQuote(ctx context.Context, input UpdateQuoteInput) error {
	if m.updateQuoteFunc != nil { return m.updateQuoteFunc(ctx, input) }
	return nil
}

func (m *mockSalesRepository) UpdateOrder(ctx context.Context, input UpdateOrderInput) error {
	if m.updateOrderFunc != nil { return m.updateOrderFunc(ctx, input) }
	return nil
}

func (m *mockSalesRepository) UpdateDeliveryNote(ctx context.Context, input UpdateDeliveryNoteInput) error {
	if m.updateDNFunc != nil { return m.updateDNFunc(ctx, input) }
	return nil
}

func (m *mockSalesRepository) CancelQuote(ctx context.Context, id uuid.UUID) error {
	if m.cancelQuoteFunc != nil { return m.cancelQuoteFunc(ctx, id) }
	return nil
}

func (m *mockSalesRepository) CancelOrder(ctx context.Context, id uuid.UUID) error {
	if m.cancelOrderFunc != nil { return m.cancelOrderFunc(ctx, id) }
	return nil
}

func (m *mockSalesRepository) CancelDeliveryNote(ctx context.Context, id uuid.UUID) error {
	if m.cancelDNFunc != nil { return m.cancelDNFunc(ctx, id) }
	return nil
}

func (m *mockSalesRepository) CancelInvoice(ctx context.Context, id uuid.UUID) error {
	if m.cancelInvoiceFunc != nil { return m.cancelInvoiceFunc(ctx, id) }
	return nil
}

type mockSecurityService struct {
	hasPermissionFunc func(ctx context.Context, userID uuid.UUID, permission string) (bool, error)
}

func (m *mockSecurityService) HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	if m.hasPermissionFunc != nil {
		return m.hasPermissionFunc(ctx, userID, permission)
	}
	return false, nil
}

type mockBillingService struct {
	generateInvoiceNumberFunc func(ctx context.Context, terminalID uuid.UUID) (string, int, error)
}

func (m *mockBillingService) GenerateInvoiceNumber(ctx context.Context, terminalID uuid.UUID) (string, int, error) {
	if m.generateInvoiceNumberFunc != nil {
		return m.generateInvoiceNumberFunc(ctx, terminalID)
	}
	return "S1-100", 100, nil
}

func TestConvertQuoteToOrder(t *testing.T) {
	ctx := context.Background()
	empresaID := uuid.New()
	clientID := uuid.New()
	userID := uuid.New()
	fixedTime := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)

	t.Run("Normal conversion of active quote", func(t *testing.T) {
		quote := &Quote{
			ID:        uuid.New(),
			EmpresaID: empresaID,
			ClientID:  clientID,
			Total:     100.0,
			Status:    StatusApproved,
			ExpiresAt: fixedTime.Add(1 * time.Hour), // Not expired
			CreatedAt: fixedTime.Add(-1 * time.Hour),
		}

		repo := &mockSalesRepository{
			getQuoteFunc: func(ctx context.Context, id uuid.UUID) (*Quote, error) {
				return quote, nil
			},
		}

		sec := &mockSecurityService{
			hasPermissionFunc: func(ctx context.Context, uid uuid.UUID, perm string) (bool, error) {
				t.Fatalf("security service should not be called for non-expired quotes")
				return false, nil
			},
		}

		service := NewSalesService(repo, nil, sec, nil)
		service.Now = func() time.Time { return fixedTime }

		order, err := service.ConvertQuoteToOrder(ctx, empresaID, quote.ID, userID, ConvertQuoteOptions{RecalculatePrices: false})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if order == nil {
			t.Fatal("expected order to be created, got nil")
		}

		if *order.QuoteID != quote.ID {
			t.Errorf("expected quote ID %v, got %v", quote.ID, *order.QuoteID)
		}

		if order.Total != 100.0 {
			t.Errorf("expected total 100.0, got %f", order.Total)
		}

		if quote.Status != StatusConverted {
			t.Errorf("expected quote status Converted, got %s", quote.Status)
		}
	})

	t.Run("Expired quote - unauthorized user", func(t *testing.T) {
		quote := &Quote{
			ID:        uuid.New(),
			EmpresaID: empresaID,
			ClientID:  clientID,
			Total:     100.0,
			Status:    StatusApproved,
			ExpiresAt: fixedTime.Add(-1 * time.Hour), // Expired
		}

		repo := &mockSalesRepository{
			getQuoteFunc: func(ctx context.Context, id uuid.UUID) (*Quote, error) {
				return quote, nil
			},
		}

		sec := &mockSecurityService{
			hasPermissionFunc: func(ctx context.Context, uid uuid.UUID, perm string) (bool, error) {
				return false, nil // Unauthorized
			},
		}

		service := NewSalesService(repo, nil, sec, nil)
		service.Now = func() time.Time { return fixedTime }

		_, err := service.ConvertQuoteToOrder(ctx, empresaID, quote.ID, userID, ConvertQuoteOptions{RecalculatePrices: false})
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})
}

func TestConvertOrderToDeliveryNote(t *testing.T) {
	ctx := context.Background()
	empresaID := uuid.New()
	whID := uuid.New()
	fixedTime := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)

	t.Run("Normal conversion", func(t *testing.T) {
		order := &Order{
			ID:        uuid.New(),
			EmpresaID: empresaID,
			Total:     150.0,
			Status:    StatusApproved,
		}

		repo := &mockSalesRepository{
			getOrderFunc: func(ctx context.Context, id uuid.UUID) (*Order, error) {
				return order, nil
			},
		}

		service := NewSalesService(repo, nil, nil, nil)
		service.Now = func() time.Time { return fixedTime }

		dn, err := service.ConvertOrderToDeliveryNote(ctx, empresaID, order.ID, whID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if dn == nil {
			t.Fatal("expected delivery note to be created, got nil")
		}

		if *dn.OrderID != order.ID {
			t.Errorf("expected order ID %v, got %v", order.ID, *dn.OrderID)
		}

		if order.Status != StatusConverted {
			t.Errorf("expected order status Converted, got %s", order.Status)
		}
	})
}

func TestConvertDeliveryNoteToInvoice(t *testing.T) {
	ctx := context.Background()
	empresaID := uuid.New()
	fixedTime := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	terminalID := uuid.New()
	seriesID := uuid.New()

	t.Run("Normal conversion calling billing service", func(t *testing.T) {
		dn := &DeliveryNote{
			ID:        uuid.New(),
			EmpresaID: empresaID,
			Total:     300.0,
			Status:    StatusProcessed,
		}

		repo := &mockSalesRepository{
			getDNFunc: func(ctx context.Context, id uuid.UUID) (*DeliveryNote, error) {
				return dn, nil
			},
		}

		bill := &mockBillingService{
			generateInvoiceNumberFunc: func(ctx context.Context, tid uuid.UUID) (string, int, error) {
				return "S1-42", 42, nil
			},
		}

		service := NewSalesService(repo, nil, nil, bill)
		service.Now = func() time.Time { return fixedTime }

		invoice, err := service.ConvertDeliveryNoteToInvoice(ctx, empresaID, dn.ID, terminalID, seriesID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if invoice == nil {
			t.Fatal("expected invoice to be created, got nil")
		}

		if *invoice.DeliveryNoteID != dn.ID {
			t.Errorf("expected delivery note ID %v, got %v", dn.ID, *invoice.DeliveryNoteID)
		}

		if invoice.InvoiceNumber != "S1-42" {
			t.Errorf("expected invoice number S1-42, got %s", invoice.InvoiceNumber)
		}
	})
}
