package domain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockSalesRepository struct {
	savePresupuestoFunc           func(ctx context.Context, q *Presupuesto) error
	getPresupuestoFunc            func(ctx context.Context, id uuid.UUID) (*Presupuesto, error)
	savePedidoFunc           func(ctx context.Context, o *Pedido) error
	getPedidoFunc            func(ctx context.Context, id uuid.UUID) (*Pedido, error)
	saveDNFunc              func(ctx context.Context, dn *Albaran) error
	getDNFunc               func(ctx context.Context, id uuid.UUID) (*Albaran, error)
	saveInvFunc             func(ctx context.Context, inv *Factura) error
	getInvFunc              func(ctx context.Context, id uuid.UUID) (*Factura, error)
	updatePresupuestoFunc         func(ctx context.Context, input UpdatePresupuestoInput) error
	updatePedidoFunc         func(ctx context.Context, input UpdatePedidoInput) error
	updateDNFunc            func(ctx context.Context, input UpdateAlbaranInput) error
	cancelPresupuestoFunc         func(ctx context.Context, id uuid.UUID) error
	cancelPedidoFunc         func(ctx context.Context, id uuid.UUID) error
	cancelDNFunc            func(ctx context.Context, id uuid.UUID) error
	cancelFacturaFunc       func(ctx context.Context, id uuid.UUID) error
	createFacturaRectificativaFunc    func(ctx context.Context, fr *FacturaRectificativa) error
	getFacturaRectificativaFunc       func(ctx context.Context, id uuid.UUID) (*FacturaRectificativa, error)
	listFacturasRectificativasFunc    func(ctx context.Context, empresaID uuid.UUID) ([]FacturaRectificativa, error)
	updateInvRectTotalFunc            func(ctx context.Context, invoiceID uuid.UUID, rectifiedTotal float64) error
	getRectifiedQuantitiesFunc        func(ctx context.Context, invoiceID uuid.UUID) (map[uuid.UUID]float64, error)
}

func (m *mockSalesRepository) SavePresupuesto(ctx context.Context, q *Presupuesto) error {
	if m.savePresupuestoFunc != nil { return m.savePresupuestoFunc(ctx, q) }
	return nil
}
func (m *mockSalesRepository) GetPresupuesto(ctx context.Context, id uuid.UUID) (*Presupuesto, error) {
	if m.getPresupuestoFunc != nil { return m.getPresupuestoFunc(ctx, id) }
	return nil, nil
}
func (m *mockSalesRepository) SavePedido(ctx context.Context, o *Pedido) error {
	if m.savePedidoFunc != nil { return m.savePedidoFunc(ctx, o) }
	return nil
}
func (m *mockSalesRepository) GetPedido(ctx context.Context, id uuid.UUID) (*Pedido, error) {
	if m.getPedidoFunc != nil { return m.getPedidoFunc(ctx, id) }
	return nil, nil
}
func (m *mockSalesRepository) SaveAlbaran(ctx context.Context, dn *Albaran) error {
	if m.saveDNFunc != nil { return m.saveDNFunc(ctx, dn) }
	return nil
}
func (m *mockSalesRepository) GetAlbaran(ctx context.Context, id uuid.UUID) (*Albaran, error) {
	if m.getDNFunc != nil { return m.getDNFunc(ctx, id) }
	return nil, nil
}
func (m *mockSalesRepository) SaveFactura(ctx context.Context, inv *Factura) error {
	if m.saveInvFunc != nil { return m.saveInvFunc(ctx, inv) }
	return nil
}
func (m *mockSalesRepository) GetFactura(ctx context.Context, id uuid.UUID) (*Factura, error) {
	if m.getInvFunc != nil { return m.getInvFunc(ctx, id) }
	return nil, nil
}

func (m *mockSalesRepository) ListPresupuestos(ctx context.Context, empresaID uuid.UUID, filter DocumentFilter) ([]*Presupuesto, int, error) {
	return nil, 0, nil
}

func (m *mockSalesRepository) ListPedidos(ctx context.Context, empresaID uuid.UUID, filter DocumentFilter) ([]*Pedido, int, error) {
	return nil, 0, nil
}

func (m *mockSalesRepository) ListAlbarans(ctx context.Context, empresaID uuid.UUID, filter DocumentFilter) ([]*Albaran, int, error) {
	return nil, 0, nil
}

func (m *mockSalesRepository) ListFacturas(ctx context.Context, empresaID uuid.UUID, filter DocumentFilter) ([]*Factura, int, error) {
	return nil, 0, nil
}

func (m *mockSalesRepository) UpdatePresupuesto(ctx context.Context, input UpdatePresupuestoInput) error {
	if m.updatePresupuestoFunc != nil { return m.updatePresupuestoFunc(ctx, input) }
	return nil
}

func (m *mockSalesRepository) UpdatePedido(ctx context.Context, input UpdatePedidoInput) error {
	if m.updatePedidoFunc != nil { return m.updatePedidoFunc(ctx, input) }
	return nil
}

func (m *mockSalesRepository) UpdateAlbaran(ctx context.Context, input UpdateAlbaranInput) error {
	if m.updateDNFunc != nil { return m.updateDNFunc(ctx, input) }
	return nil
}

func (m *mockSalesRepository) CancelPresupuesto(ctx context.Context, id uuid.UUID) error {
	if m.cancelPresupuestoFunc != nil { return m.cancelPresupuestoFunc(ctx, id) }
	return nil
}

func (m *mockSalesRepository) CancelPedido(ctx context.Context, id uuid.UUID) error {
	if m.cancelPedidoFunc != nil { return m.cancelPedidoFunc(ctx, id) }
	return nil
}

func (m *mockSalesRepository) CancelAlbaran(ctx context.Context, id uuid.UUID) error {
	if m.cancelDNFunc != nil { return m.cancelDNFunc(ctx, id) }
	return nil
}

func (m *mockSalesRepository) CancelFactura(ctx context.Context, id uuid.UUID) error {
	if m.cancelFacturaFunc != nil { return m.cancelFacturaFunc(ctx, id) }
	return nil
}

func (m *mockSalesRepository) CreateFacturaRectificativa(ctx context.Context, fr *FacturaRectificativa) error {
	if m.createFacturaRectificativaFunc != nil { return m.createFacturaRectificativaFunc(ctx, fr) }
	return nil
}

func (m *mockSalesRepository) GetFacturaRectificativa(ctx context.Context, id uuid.UUID) (*FacturaRectificativa, error) {
	if m.getFacturaRectificativaFunc != nil { return m.getFacturaRectificativaFunc(ctx, id) }
	return nil, nil
}

func (m *mockSalesRepository) ListFacturasRectificativas(ctx context.Context, empresaID uuid.UUID) ([]FacturaRectificativa, error) {
	if m.listFacturasRectificativasFunc != nil { return m.listFacturasRectificativasFunc(ctx, empresaID) }
	return nil, nil
}

func (m *mockSalesRepository) UpdateFacturaRectifiedTotal(ctx context.Context, invoiceID uuid.UUID, rectifiedTotal float64) error {
	if m.updateInvRectTotalFunc != nil { return m.updateInvRectTotalFunc(ctx, invoiceID, rectifiedTotal) }
	return nil
}

func (m *mockSalesRepository) GetRectifiedQuantitiesByInvoice(ctx context.Context, invoiceID uuid.UUID) (map[uuid.UUID]float64, error) {
	if m.getRectifiedQuantitiesFunc != nil { return m.getRectifiedQuantitiesFunc(ctx, invoiceID) }
	return nil, nil
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
	generateFacturaNumberFunc func(ctx context.Context, terminalID uuid.UUID) (string, int, error)
}

func (m *mockBillingService) GenerateFacturaNumber(ctx context.Context, terminalID uuid.UUID) (string, int, error) {
	if m.generateFacturaNumberFunc != nil {
		return m.generateFacturaNumberFunc(ctx, terminalID)
	}
	return "S1-100", 100, nil
}

func TestConvertPresupuestoToPedido(t *testing.T) {
	ctx := context.Background()
	empresaID := uuid.New()
	clientID := uuid.New()
	userID := uuid.New()
	fixedTime := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)

	t.Run("Normal conversion of active quote", func(t *testing.T) {
		quote := &Presupuesto{
			ID:        uuid.New(),
			EmpresaID: empresaID,
			ClienteID:  clientID,
			Total:     100.0,
			Estado:    StatusApproved,
			FechaValidez: fixedTime.Add(1 * time.Hour), // Not expired
			CreatedAt: fixedTime.Add(-1 * time.Hour),
		}

		repo := &mockSalesRepository{
			getPresupuestoFunc: func(ctx context.Context, id uuid.UUID) (*Presupuesto, error) {
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

		order, err := service.ConvertPresupuestoToPedido(ctx, empresaID, quote.ID, userID, ConvertPresupuestoOptions{RecalculatePrices: false})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if order == nil {
			t.Fatal("expected order to be created, got nil")
		}

		if *order.PresupuestoID != quote.ID {
			t.Errorf("expected quote ID %v, got %v", quote.ID, *order.PresupuestoID)
		}

		if order.Total != 100.0 {
			t.Errorf("expected total 100.0, got %f", order.Total)
		}

		if quote.Estado != StatusConverted {
			t.Errorf("expected quote status Converted, got %s", quote.Estado)
		}
	})

	t.Run("Expired quote - unauthorized user", func(t *testing.T) {
		quote := &Presupuesto{
			ID:        uuid.New(),
			EmpresaID: empresaID,
			ClienteID:  clientID,
			Total:     100.0,
			Estado:    StatusApproved,
			FechaValidez: fixedTime.Add(-1 * time.Hour), // Expired
		}

		repo := &mockSalesRepository{
			getPresupuestoFunc: func(ctx context.Context, id uuid.UUID) (*Presupuesto, error) {
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

		_, err := service.ConvertPresupuestoToPedido(ctx, empresaID, quote.ID, userID, ConvertPresupuestoOptions{RecalculatePrices: false})
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})
}

func TestConvertPedidoToAlbaran(t *testing.T) {
	ctx := context.Background()
	empresaID := uuid.New()
	whID := uuid.New()
	fixedTime := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)

	t.Run("Normal conversion", func(t *testing.T) {
		order := &Pedido{
			ID:        uuid.New(),
			EmpresaID: empresaID,
			Total:     150.0,
			Estado:    StatusApproved,
		}

		repo := &mockSalesRepository{
			getPedidoFunc: func(ctx context.Context, id uuid.UUID) (*Pedido, error) {
				return order, nil
			},
		}

		service := NewSalesService(repo, nil, nil, nil)
		service.Now = func() time.Time { return fixedTime }

		dn, err := service.ConvertPedidoToAlbaran(ctx, empresaID, order.ID, whID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if dn == nil {
			t.Fatal("expected delivery note to be created, got nil")
		}

		if *dn.PedidoID != order.ID {
			t.Errorf("expected order ID %v, got %v", order.ID, *dn.PedidoID)
		}

		if order.Estado != StatusConverted {
			t.Errorf("expected order status Converted, got %s", order.Estado)
		}
	})
}

func TestConvertAlbaranToFactura(t *testing.T) {
	ctx := context.Background()
	empresaID := uuid.New()
	fixedTime := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	terminalID := uuid.New()
	seriesID := uuid.New()

	t.Run("Normal conversion calling billing service", func(t *testing.T) {
		dn := &Albaran{
			ID:        uuid.New(),
			EmpresaID: empresaID,
			Total:     300.0,
			Estado:    StatusProcessed,
		}

		repo := &mockSalesRepository{
			getDNFunc: func(ctx context.Context, id uuid.UUID) (*Albaran, error) {
				return dn, nil
			},
		}

		bill := &mockBillingService{
			generateFacturaNumberFunc: func(ctx context.Context, tid uuid.UUID) (string, int, error) {
				return "S1-42", 42, nil
			},
		}

		service := NewSalesService(repo, nil, nil, bill)
		service.Now = func() time.Time { return fixedTime }

		invoice, err := service.ConvertAlbaranToFactura(ctx, empresaID, dn.ID, terminalID, seriesID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if invoice == nil {
			t.Fatal("expected invoice to be created, got nil")
		}

		if *invoice.AlbaranID != dn.ID {
			t.Errorf("expected delivery note ID %v, got %v", dn.ID, *invoice.AlbaranID)
		}

		if invoice.NumeroFactura != "S1-42" {
			t.Errorf("expected invoice number S1-42, got %s", invoice.NumeroFactura)
		}
	})
}
