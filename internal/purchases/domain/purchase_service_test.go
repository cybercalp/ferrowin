package domain

import (
	"context"
	"errors"
	"testing"

	inventorydomain "ferrowin/internal/inventory/domain"

	"github.com/google/uuid"
)

type mockPurchaseRepository struct {
	saveCompanyFunc         func(ctx context.Context, c *Empresa) error
	saveWarehouseFunc       func(ctx context.Context, w *Warehouse) error
	getWarehouseFunc        func(ctx context.Context, id uuid.UUID) (*Warehouse, error)
	saveSupplierFunc        func(ctx context.Context, s *Proveedor) error
	getSuppliersFunc        func(ctx context.Context, empresaID uuid.UUID) ([]*Proveedor, error)
	getSupplierFunc         func(ctx context.Context, id uuid.UUID) (*Proveedor, error)
	savePurchaseOrderFunc   func(ctx context.Context, o *PedidoCompra) error
	getPurchaseOrderFunc    func(ctx context.Context, id uuid.UUID) (*PedidoCompra, error)
	savePurchaseReceiptFunc func(ctx context.Context, r *RecepcionCompra) error
	getPurchaseReceiptFunc  func(ctx context.Context, id uuid.UUID) (*RecepcionCompra, error)
}

func (m *mockPurchaseRepository) SaveCompany(ctx context.Context, c *Empresa) error {
	if m.saveCompanyFunc != nil {
		return m.saveCompanyFunc(ctx, c)
	}
	return nil
}

func (m *mockPurchaseRepository) SaveWarehouse(ctx context.Context, w *Warehouse) error {
	if m.saveWarehouseFunc != nil {
		return m.saveWarehouseFunc(ctx, w)
	}
	return nil
}

func (m *mockPurchaseRepository) GetWarehouse(ctx context.Context, id uuid.UUID) (*Warehouse, error) {
	if m.getWarehouseFunc != nil {
		return m.getWarehouseFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockPurchaseRepository) SaveSupplier(ctx context.Context, s *Proveedor) error {
	if m.saveSupplierFunc != nil {
		return m.saveSupplierFunc(ctx, s)
	}
	return nil
}

func (m *mockPurchaseRepository) GetSuppliers(ctx context.Context, empresaID uuid.UUID) ([]*Proveedor, error) {
	if m.getSuppliersFunc != nil {
		return m.getSuppliersFunc(ctx, empresaID)
	}
	return nil, nil
}

func (m *mockPurchaseRepository) GetSupplier(ctx context.Context, id uuid.UUID) (*Proveedor, error) {
	if m.getSupplierFunc != nil {
		return m.getSupplierFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockPurchaseRepository) SavePurchaseOrder(ctx context.Context, o *PedidoCompra) error {
	if m.savePurchaseOrderFunc != nil {
		return m.savePurchaseOrderFunc(ctx, o)
	}
	return nil
}

func (m *mockPurchaseRepository) GetPurchaseOrder(ctx context.Context, id uuid.UUID) (*PedidoCompra, error) {
	if m.getPurchaseOrderFunc != nil {
		return m.getPurchaseOrderFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockPurchaseRepository) SavePurchaseReceipt(ctx context.Context, r *RecepcionCompra) error {
	if m.savePurchaseReceiptFunc != nil {
		return m.savePurchaseReceiptFunc(ctx, r)
	}
	return nil
}

func (m *mockPurchaseRepository) GetPurchaseReceipt(ctx context.Context, id uuid.UUID) (*RecepcionCompra, error) {
	if m.getPurchaseReceiptFunc != nil {
		return m.getPurchaseReceiptFunc(ctx, id)
	}
	return nil, nil
}

type mockStockLedgerRepository struct {
	saveFunc             func(ctx context.Context, entry *inventorydomain.StockLedgerEntry) error
	getMovementsFunc     func(ctx context.Context, itemID, warehouseID uuid.UUID) ([]*inventorydomain.StockLedgerEntry, error)
}

func (m *mockStockLedgerRepository) Save(ctx context.Context, entry *inventorydomain.StockLedgerEntry) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, entry)
	}
	return nil
}

func (m *mockStockLedgerRepository) GetMovements(ctx context.Context, itemID, warehouseID uuid.UUID) ([]*inventorydomain.StockLedgerEntry, error) {
	if m.getMovementsFunc != nil {
		return m.getMovementsFunc(ctx, itemID, warehouseID)
	}
	return nil, nil
}

func newTestInventoryService(repo *mockStockLedgerRepository) *inventorydomain.InventoryService {
	return inventorydomain.NewInventoryService(repo)
}

func TestCreateCompany(t *testing.T) {
	ctx := context.Background()

	t.Run("creates company with correct fields", func(t *testing.T) {
		var saved *Empresa
		repo := &mockPurchaseRepository{
			saveCompanyFunc: func(ctx context.Context, c *Empresa) error {
				saved = c
				return nil
			},
		}
		svc := NewPurchaseService(repo, nil)

		c, err := svc.CreateCompany(ctx, "Test SA", "B12345678")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if c.RazonSocial != "Test SA" {
			t.Errorf("expected RazonSocial 'Test SA', got %s", c.RazonSocial)
		}
		if c.NIF != "B12345678" {
			t.Errorf("expected NIF 'B12345678', got %s", c.NIF)
		}
		if !c.Activa {
			t.Error("expected Activa to be true")
		}
		if c.ID == uuid.Nil {
			t.Error("expected non-zero ID")
		}
		if saved != c {
			t.Error("expected saved company to be the returned pointer")
		}
	})

	t.Run("repository error propagates", func(t *testing.T) {
		repo := &mockPurchaseRepository{
			saveCompanyFunc: func(ctx context.Context, c *Empresa) error {
				return errors.New("db error")
			},
		}
		svc := NewPurchaseService(repo, nil)

		_, err := svc.CreateCompany(ctx, "Test SA", "B12345678")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestCreateWarehouse(t *testing.T) {
	ctx := context.Background()
	empresaID := uuid.New()

	t.Run("creates warehouse linked to empresa", func(t *testing.T) {
		var saved *Warehouse
		repo := &mockPurchaseRepository{
			saveWarehouseFunc: func(ctx context.Context, w *Warehouse) error {
				saved = w
				return nil
			},
		}
		svc := NewPurchaseService(repo, nil)

		w, err := svc.CreateWarehouse(ctx, empresaID, "Main Warehouse")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if w.EmpresaID != empresaID {
			t.Errorf("expected EmpresaID %v, got %v", empresaID, w.EmpresaID)
		}
		if w.Name != "Main Warehouse" {
			t.Errorf("expected Name 'Main Warehouse', got %s", w.Name)
		}
		if !w.Active {
			t.Error("expected Active to be true")
		}
		if w.ID == uuid.Nil {
			t.Error("expected non-zero ID")
		}
		if saved != w {
			t.Error("expected saved warehouse to be the returned pointer")
		}
	})

	t.Run("repository error propagates", func(t *testing.T) {
		repo := &mockPurchaseRepository{
			saveWarehouseFunc: func(ctx context.Context, w *Warehouse) error {
				return errors.New("db error")
			},
		}
		svc := NewPurchaseService(repo, nil)

		_, err := svc.CreateWarehouse(ctx, empresaID, "Main Warehouse")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestCreateSupplier(t *testing.T) {
	ctx := context.Background()
	empresaID := uuid.New()

	t.Run("creates supplier with all fields", func(t *testing.T) {
		var saved *Proveedor
		repo := &mockPurchaseRepository{
			saveSupplierFunc: func(ctx context.Context, s *Proveedor) error {
				saved = s
				return nil
			},
		}
		svc := NewPurchaseService(repo, nil)

		s, err := svc.CreateSupplier(ctx, empresaID, "Proveedor SA", "A12345678", "info@prov.com", "555-0100", "Calle Falsa 123")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.EmpresaID != empresaID {
			t.Errorf("expected EmpresaID %v, got %v", empresaID, s.EmpresaID)
		}
		if s.RazonSocial != "Proveedor SA" {
			t.Errorf("expected RazonSocial 'Proveedor SA', got %s", s.RazonSocial)
		}
		if s.CIF != "A12345678" {
			t.Errorf("expected CIF 'A12345678', got %s", s.CIF)
		}
		if s.Email != "info@prov.com" {
			t.Errorf("expected Email 'info@prov.com', got %s", s.Email)
		}
		if s.Telefono != "555-0100" {
			t.Errorf("expected Telefono '555-0100', got %s", s.Telefono)
		}
		if s.Direccion != "Calle Falsa 123" {
			t.Errorf("expected Direccion 'Calle Falsa 123', got %s", s.Direccion)
		}
		if !s.Activo {
			t.Error("expected Activo to be true")
		}
		if s.ID == uuid.Nil {
			t.Error("expected non-zero ID")
		}
		if saved != s {
			t.Error("expected saved supplier to be the returned pointer")
		}
	})

	t.Run("repository error propagates", func(t *testing.T) {
		repo := &mockPurchaseRepository{
			saveSupplierFunc: func(ctx context.Context, s *Proveedor) error {
				return errors.New("db error")
			},
		}
		svc := NewPurchaseService(repo, nil)

		_, err := svc.CreateSupplier(ctx, empresaID, "Proveedor SA", "A12345678", "info@prov.com", "555-0100", "Calle Falsa 123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetSuppliers(t *testing.T) {
	ctx := context.Background()
	empresaID := uuid.New()

	t.Run("returns suppliers from repo", func(t *testing.T) {
		expected := []*Proveedor{
			{ID: uuid.New(), EmpresaID: empresaID, RazonSocial: "Prov 1"},
			{ID: uuid.New(), EmpresaID: empresaID, RazonSocial: "Prov 2"},
		}
		repo := &mockPurchaseRepository{
			getSuppliersFunc: func(ctx context.Context, eid uuid.UUID) ([]*Proveedor, error) {
				return expected, nil
			},
		}
		svc := NewPurchaseService(repo, nil)

		result, err := svc.GetSuppliers(ctx, empresaID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(result) != len(expected) {
			t.Fatalf("expected %d suppliers, got %d", len(expected), len(result))
		}
		for i := range expected {
			if result[i].ID != expected[i].ID {
				t.Errorf("index %d: expected ID %v, got %v", i, expected[i].ID, result[i].ID)
			}
		}
	})

	t.Run("returns empty slice when none exist", func(t *testing.T) {
		repo := &mockPurchaseRepository{
			getSuppliersFunc: func(ctx context.Context, eid uuid.UUID) ([]*Proveedor, error) {
				return []*Proveedor{}, nil
			},
		}
		svc := NewPurchaseService(repo, nil)

		result, err := svc.GetSuppliers(ctx, empresaID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil empty slice, got nil")
		}
		if len(result) != 0 {
			t.Errorf("expected 0 suppliers, got %d", len(result))
		}
	})
}

func TestCreatePurchaseOrder(t *testing.T) {
	ctx := context.Background()
	empresaID := uuid.New()
	proveedorID := uuid.New()

	t.Run("creates order with correct totals and lines", func(t *testing.T) {
		prov := &Proveedor{ID: proveedorID, EmpresaID: empresaID}
		repo := &mockPurchaseRepository{
			getSupplierFunc: func(ctx context.Context, id uuid.UUID) (*Proveedor, error) {
				return prov, nil
			},
			savePurchaseOrderFunc: func(ctx context.Context, o *PedidoCompra) error {
				return nil
			},
		}
		svc := NewPurchaseService(repo, nil)

		lines := []PedidoCompraLinea{
			{ProductoID: uuid.New(), Cantidad: 2, PrecioUnitario: 10.0},
			{ProductoID: uuid.New(), Cantidad: 3, PrecioUnitario: 5.5},
		}

		po, err := svc.CreatePurchaseOrder(ctx, empresaID, proveedorID, "PO-001", lines)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if po.EmpresaID != empresaID {
			t.Errorf("expected EmpresaID %v, got %v", empresaID, po.EmpresaID)
		}
		if po.ProveedorID != proveedorID {
			t.Errorf("expected ProveedorID %v, got %v", proveedorID, po.ProveedorID)
		}
		if po.NumeroPedido != "PO-001" {
			t.Errorf("expected NumeroPedido 'PO-001', got %s", po.NumeroPedido)
		}
		if po.Estado != "Borrador" {
			t.Errorf("expected Estado 'Borrador', got %s", po.Estado)
		}
		if po.ID == uuid.Nil {
			t.Error("expected non-zero ID")
		}
		if po.Fecha.IsZero() {
			t.Error("expected non-zero Fecha")
		}

		expectedTotal := 2*10.0 + 3*5.5
		if po.Total != expectedTotal {
			t.Errorf("expected Total %.2f, got %.2f", expectedTotal, po.Total)
		}
		if len(po.Lineas) != len(lines) {
			t.Fatalf("expected %d lines, got %d", len(lines), len(po.Lineas))
		}
		for i := range lines {
			if po.Lineas[i].ProductoID != lines[i].ProductoID {
				t.Errorf("line %d: expected ProductoID %v, got %v", i, lines[i].ProductoID, po.Lineas[i].ProductoID)
			}
			if po.Lineas[i].Cantidad != lines[i].Cantidad {
				t.Errorf("line %d: expected Cantidad %.2f, got %.2f", i, lines[i].Cantidad, po.Lineas[i].Cantidad)
			}
			if po.Lineas[i].PrecioUnitario != lines[i].PrecioUnitario {
				t.Errorf("line %d: expected PrecioUnitario %.2f, got %.2f", i, lines[i].PrecioUnitario, po.Lineas[i].PrecioUnitario)
			}
			if po.Lineas[i].PedidoCompraID != po.ID {
				t.Errorf("line %d: expected PedidoCompraID %v, got %v", i, po.ID, po.Lineas[i].PedidoCompraID)
			}
			if po.Lineas[i].ID == uuid.Nil {
				t.Errorf("line %d: expected non-zero ID", i)
			}
		}
	})

	t.Run("rejects if supplier belongs to different empresa", func(t *testing.T) {
		otherEmpresaID := uuid.New()
		prov := &Proveedor{ID: proveedorID, EmpresaID: otherEmpresaID}
		repo := &mockPurchaseRepository{
			getSupplierFunc: func(ctx context.Context, id uuid.UUID) (*Proveedor, error) {
				return prov, nil
			},
		}
		svc := NewPurchaseService(repo, nil)

		_, err := svc.CreatePurchaseOrder(ctx, empresaID, proveedorID, "PO-002", nil)
		if !errors.Is(err, ErrTenantMismatch) {
			t.Fatalf("expected ErrTenantMismatch, got %v", err)
		}
	})
}

func TestApprovePurchaseOrder(t *testing.T) {
	ctx := context.Background()
	empresaID := uuid.New()
	orderID := uuid.New()

	t.Run("approves from Borrador status", func(t *testing.T) {
		po := &PedidoCompra{
			ID:        orderID,
			EmpresaID: empresaID,
			Estado:    "Borrador",
		}
		repo := &mockPurchaseRepository{
			getPurchaseOrderFunc: func(ctx context.Context, id uuid.UUID) (*PedidoCompra, error) {
				return po, nil
			},
			savePurchaseOrderFunc: func(ctx context.Context, o *PedidoCompra) error {
				return nil
			},
		}
		svc := NewPurchaseService(repo, nil)

		err := svc.ApprovePurchaseOrder(ctx, empresaID, orderID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if po.Estado != "Aprobado" {
			t.Errorf("expected Estado 'Aprobado', got %s", po.Estado)
		}
	})

	t.Run("rejects if already approved", func(t *testing.T) {
		po := &PedidoCompra{
			ID:        orderID,
			EmpresaID: empresaID,
			Estado:    "Aprobado",
		}
		repo := &mockPurchaseRepository{
			getPurchaseOrderFunc: func(ctx context.Context, id uuid.UUID) (*PedidoCompra, error) {
				return po, nil
			},
		}
		svc := NewPurchaseService(repo, nil)

		err := svc.ApprovePurchaseOrder(ctx, empresaID, orderID)
		if !errors.Is(err, ErrInvalidStatus) {
			t.Fatalf("expected ErrInvalidStatus, got %v", err)
		}
	})

	t.Run("rejects if tenant mismatch", func(t *testing.T) {
		po := &PedidoCompra{
			ID:        orderID,
			EmpresaID: uuid.New(),
			Estado:    "Borrador",
		}
		repo := &mockPurchaseRepository{
			getPurchaseOrderFunc: func(ctx context.Context, id uuid.UUID) (*PedidoCompra, error) {
				return po, nil
			},
		}
		svc := NewPurchaseService(repo, nil)

		err := svc.ApprovePurchaseOrder(ctx, empresaID, orderID)
		if !errors.Is(err, ErrTenantMismatch) {
			t.Fatalf("expected ErrTenantMismatch, got %v", err)
		}
	})
}

func TestCreatePurchaseReceipt(t *testing.T) {
	ctx := context.Background()
	empresaID := uuid.New()
	supplierID := uuid.New()
	warehouseID := uuid.New()

	t.Run("creates receipt linked to supplier and warehouse", func(t *testing.T) {
		prov := &Proveedor{ID: supplierID, EmpresaID: empresaID}
		wh := &Warehouse{ID: warehouseID, EmpresaID: empresaID}

		repo := &mockPurchaseRepository{
			getSupplierFunc: func(ctx context.Context, id uuid.UUID) (*Proveedor, error) {
				return prov, nil
			},
			getWarehouseFunc: func(ctx context.Context, id uuid.UUID) (*Warehouse, error) {
				return wh, nil
			},
			savePurchaseReceiptFunc: func(ctx context.Context, r *RecepcionCompra) error {
				return nil
			},
		}
		svc := NewPurchaseService(repo, nil)

		lines := []RecepcionCompraLinea{
			{ProductoID: uuid.New(), Cantidad: 10, PrecioUnitario: 5.0},
		}
		poID := uuid.New()

		rc, err := svc.CreatePurchaseReceipt(ctx, empresaID, supplierID, &poID, "ALB-001", warehouseID, lines)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if rc.EmpresaID != empresaID {
			t.Errorf("expected EmpresaID %v, got %v", empresaID, rc.EmpresaID)
		}
		if rc.ProveedorID != supplierID {
			t.Errorf("expected ProveedorID %v, got %v", supplierID, rc.ProveedorID)
		}
		if rc.WarehouseID != warehouseID {
			t.Errorf("expected WarehouseID %v, got %v", warehouseID, rc.WarehouseID)
		}
		if rc.NumeroAlbaran != "ALB-001" {
			t.Errorf("expected NumeroAlbaran 'ALB-001', got %s", rc.NumeroAlbaran)
		}
		if rc.Estado != "Borrador" {
			t.Errorf("expected Estado 'Borrador', got %s", rc.Estado)
		}
		if rc.PedidoCompraID == nil || *rc.PedidoCompraID != poID {
			t.Errorf("expected PedidoCompraID %v, got %v", poID, rc.PedidoCompraID)
		}
		if rc.ID == uuid.Nil {
			t.Error("expected non-zero ID")
		}
		if rc.Fecha.IsZero() {
			t.Error("expected non-zero Fecha")
		}
		if len(rc.Lineas) != len(lines) {
			t.Fatalf("expected %d lines, got %d", len(lines), len(rc.Lineas))
		}
		for i := range lines {
			if rc.Lineas[i].ProductoID != lines[i].ProductoID {
				t.Errorf("line %d: expected ProductoID %v, got %v", i, lines[i].ProductoID, rc.Lineas[i].ProductoID)
			}
			if rc.Lineas[i].Cantidad != lines[i].Cantidad {
				t.Errorf("line %d: expected Cantidad %.2f, got %.2f", i, lines[i].Cantidad, rc.Lineas[i].Cantidad)
			}
			if rc.Lineas[i].PrecioUnitario != lines[i].PrecioUnitario {
				t.Errorf("line %d: expected PrecioUnitario %.2f, got %.2f", i, lines[i].PrecioUnitario, rc.Lineas[i].PrecioUnitario)
			}
			if rc.Lineas[i].RecepcionCompraID != rc.ID {
				t.Errorf("line %d: expected RecepcionCompraID %v, got %v", i, rc.ID, rc.Lineas[i].RecepcionCompraID)
			}
			if rc.Lineas[i].ID == uuid.Nil {
				t.Errorf("line %d: expected non-zero ID", i)
			}
		}
	})

	t.Run("rejects if supplier empresa mismatch", func(t *testing.T) {
		prov := &Proveedor{ID: supplierID, EmpresaID: uuid.New()}
		repo := &mockPurchaseRepository{
			getSupplierFunc: func(ctx context.Context, id uuid.UUID) (*Proveedor, error) {
				return prov, nil
			},
		}
		svc := NewPurchaseService(repo, nil)

		_, err := svc.CreatePurchaseReceipt(ctx, empresaID, supplierID, nil, "ALB-002", warehouseID, nil)
		if !errors.Is(err, ErrTenantMismatch) {
			t.Fatalf("expected ErrTenantMismatch, got %v", err)
		}
	})

	t.Run("rejects if warehouse empresa mismatch", func(t *testing.T) {
		prov := &Proveedor{ID: supplierID, EmpresaID: empresaID}
		wh := &Warehouse{ID: warehouseID, EmpresaID: uuid.New()}
		repo := &mockPurchaseRepository{
			getSupplierFunc: func(ctx context.Context, id uuid.UUID) (*Proveedor, error) {
				return prov, nil
			},
			getWarehouseFunc: func(ctx context.Context, id uuid.UUID) (*Warehouse, error) {
				return wh, nil
			},
		}
		svc := NewPurchaseService(repo, nil)

		_, err := svc.CreatePurchaseReceipt(ctx, empresaID, supplierID, nil, "ALB-003", warehouseID, nil)
		if !errors.Is(err, ErrTenantMismatch) {
			t.Fatalf("expected ErrTenantMismatch, got %v", err)
		}
	})
}

func TestProcessPurchaseReceipt(t *testing.T) {
	ctx := context.Background()
	empresaID := uuid.New()
	receiptID := uuid.New()

	t.Run("processes receipt, calls invService.RecordReceipt for each line", func(t *testing.T) {
		productIDs := []uuid.UUID{uuid.New(), uuid.New()}
		warehouseID := uuid.New()

		rc := &RecepcionCompra{
			ID:          receiptID,
			EmpresaID:   empresaID,
			WarehouseID: warehouseID,
			Estado:      "Borrador",
			Lineas: []RecepcionCompraLinea{
				{ProductoID: productIDs[0], Cantidad: 5, PrecioUnitario: 10.0},
				{ProductoID: productIDs[1], Cantidad: 3, PrecioUnitario: 20.0},
			},
		}

		var savedEntries []*inventorydomain.StockLedgerEntry
		stockRepo := &mockStockLedgerRepository{
			saveFunc: func(ctx context.Context, entry *inventorydomain.StockLedgerEntry) error {
				savedEntries = append(savedEntries, entry)
				return nil
			},
		}
		invSvc := newTestInventoryService(stockRepo)

		repo := &mockPurchaseRepository{
			getPurchaseReceiptFunc: func(ctx context.Context, id uuid.UUID) (*RecepcionCompra, error) {
				return rc, nil
			},
			savePurchaseReceiptFunc: func(ctx context.Context, r *RecepcionCompra) error {
				return nil
			},
		}
		svc := NewPurchaseService(repo, invSvc)

		err := svc.ProcessPurchaseReceipt(ctx, empresaID, receiptID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if rc.Estado != "Procesado" {
			t.Errorf("expected Estado 'Procesado', got %s", rc.Estado)
		}

		if len(savedEntries) != len(rc.Lineas) {
			t.Fatalf("expected %d stock entries, got %d", len(rc.Lineas), len(savedEntries))
		}
		for i, pID := range productIDs {
			e := savedEntries[i]
			if e.ItemID != pID {
				t.Errorf("entry %d: expected ItemID %v, got %v", i, pID, e.ItemID)
			}
			if e.WarehouseID != warehouseID {
				t.Errorf("entry %d: expected WarehouseID %v, got %v", i, warehouseID, e.WarehouseID)
			}
			if e.Quantity != rc.Lineas[i].Cantidad {
				t.Errorf("entry %d: expected Quantity %.2f, got %.2f", i, rc.Lineas[i].Cantidad, e.Quantity)
			}
			if e.ReferenceDocumentType == nil || *e.ReferenceDocumentType != "PURCHASE_RECEIPT" {
				t.Errorf("entry %d: expected ReferenceDocumentType 'PURCHASE_RECEIPT', got %v", i, e.ReferenceDocumentType)
			}
			if e.ReferenceDocumentID == nil || *e.ReferenceDocumentID != receiptID {
				t.Errorf("entry %d: expected ReferenceDocumentID %v, got %v", i, receiptID, e.ReferenceDocumentID)
			}
			if e.MovementType != inventorydomain.MovementTypeReceipt {
				t.Errorf("entry %d: expected MovementType RECEIPT, got %s", i, e.MovementType)
			}
		}
	})

	t.Run("rejects if already processed", func(t *testing.T) {
		rc := &RecepcionCompra{
			ID:        receiptID,
			EmpresaID: empresaID,
			Estado:    "Procesado",
		}
		repo := &mockPurchaseRepository{
			getPurchaseReceiptFunc: func(ctx context.Context, id uuid.UUID) (*RecepcionCompra, error) {
				return rc, nil
			},
		}
		svc := NewPurchaseService(repo, nil)

		err := svc.ProcessPurchaseReceipt(ctx, empresaID, receiptID)
		if !errors.Is(err, ErrInvalidStatus) {
			t.Fatalf("expected ErrInvalidStatus, got %v", err)
		}
	})

	t.Run("rejects if tenant mismatch", func(t *testing.T) {
		rc := &RecepcionCompra{
			ID:        receiptID,
			EmpresaID: uuid.New(),
			Estado:    "Borrador",
		}
		repo := &mockPurchaseRepository{
			getPurchaseReceiptFunc: func(ctx context.Context, id uuid.UUID) (*RecepcionCompra, error) {
				return rc, nil
			},
		}
		svc := NewPurchaseService(repo, nil)

		err := svc.ProcessPurchaseReceipt(ctx, empresaID, receiptID)
		if !errors.Is(err, ErrTenantMismatch) {
			t.Fatalf("expected ErrTenantMismatch, got %v", err)
		}
	})
}
