package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// --- Mock Repository ---

type mockCatalogRepo struct {
	// Stored entities for lookups
	tiposIVA map[uuid.UUID]*TipoIVA
	familias map[uuid.UUID]*Familia
}

func newMockRepo() *mockCatalogRepo {
	return &mockCatalogRepo{
		tiposIVA: make(map[uuid.UUID]*TipoIVA),
		familias: make(map[uuid.UUID]*Familia),
	}
}

func (m *mockCatalogRepo) seedTipoIVA(id uuid.UUID, nombre string, pct float64) {
	m.tiposIVA[id] = &TipoIVA{ID: id, Nombre: nombre, Porcentaje: pct, Activo: true}
}

func (m *mockCatalogRepo) CreateTipoIVA(_ context.Context, t *TipoIVA) error {
	m.tiposIVA[t.ID] = t
	return nil
}

func (m *mockCatalogRepo) GetTipoIVA(_ context.Context, id uuid.UUID) (*TipoIVA, error) {
	t, ok := m.tiposIVA[id]
	if !ok {
		return nil, errors.New("tipo iva not found")
	}
	return t, nil
}

func (m *mockCatalogRepo) ListTiposIVA(_ context.Context) ([]TipoIVA, error) {
	out := make([]TipoIVA, 0, len(m.tiposIVA))
	for _, t := range m.tiposIVA {
		out = append(out, *t)
	}
	return out, nil
}

func (m *mockCatalogRepo) UpdateTipoIVA(_ context.Context, t *TipoIVA) error {
	m.tiposIVA[t.ID] = t
	return nil
}

func (m *mockCatalogRepo) DeleteTipoIVA(_ context.Context, id uuid.UUID) error {
	delete(m.tiposIVA, id)
	return nil
}

func (m *mockCatalogRepo) CreateFamilia(_ context.Context, f *Familia) error {
	m.familias[f.ID] = f
	return nil
}

func (m *mockCatalogRepo) GetFamilia(_ context.Context, id uuid.UUID) (*Familia, error) {
	f, ok := m.familias[id]
	if !ok {
		return nil, errors.New("familia not found")
	}
	return f, nil
}

func (m *mockCatalogRepo) ListFamilias(_ context.Context) ([]Familia, error) {
	out := make([]Familia, 0, len(m.familias))
	for _, f := range m.familias {
		out = append(out, *f)
	}
	return out, nil
}

func (m *mockCatalogRepo) UpdateFamilia(_ context.Context, f *Familia) error {
	m.familias[f.ID] = f
	return nil
}

func (m *mockCatalogRepo) DeleteFamilia(_ context.Context, id uuid.UUID) error {
	delete(m.familias, id)
	return nil
}

func (m *mockCatalogRepo) CreateProducto(_ context.Context, _ *Producto) error {
	return nil
}

func (m *mockCatalogRepo) GetProducto(_ context.Context, _ uuid.UUID) (*Producto, error) {
	return nil, errors.New("not found")
}

func (m *mockCatalogRepo) ListProductos(_ context.Context, _, _ *uuid.UUID) ([]Producto, error) {
	return nil, nil
}

func (m *mockCatalogRepo) UpdateProducto(_ context.Context, _ *Producto) error {
	return nil
}

func (m *mockCatalogRepo) DeleteProducto(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockCatalogRepo) CreateCliente(_ context.Context, _ *Entidad) error {
	return nil
}

func (m *mockCatalogRepo) GetCliente(_ context.Context, _ uuid.UUID) (*Entidad, error) {
	return nil, errors.New("not found")
}

func (m *mockCatalogRepo) ListClientes(_ context.Context, _ uuid.UUID) ([]Entidad, error) {
	return nil, nil
}

func (m *mockCatalogRepo) UpdateCliente(_ context.Context, _ *Entidad) error {
	return nil
}

func (m *mockCatalogRepo) DeleteCliente(_ context.Context, _ uuid.UUID) error {
	return nil
}

// --- Tests ---

func TestCatalogService_CreateTipoIVA(t *testing.T) {
	ctx := context.Background()
	svc := NewCatalogService(newMockRepo())

	t.Run("success", func(t *testing.T) {
		iva, err := svc.CreateTipoIVA(ctx, "IVA General", 21.0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if iva.Nombre != "IVA General" {
			t.Errorf("expected nombre 'IVA General', got %q", iva.Nombre)
		}
		if iva.Porcentaje != 21.0 {
			t.Errorf("expected porcentaje 21.0, got %f", iva.Porcentaje)
		}
		if iva.ID == uuid.Nil {
			t.Error("expected non-nil ID")
		}
		if !iva.Activo {
			t.Error("expected Activo=true")
		}
	})

	t.Run("empty nombre", func(t *testing.T) {
		_, err := svc.CreateTipoIVA(ctx, "", 21.0)
		if err == nil {
			t.Fatal("expected error for empty nombre")
		}
	})

	t.Run("zero porcentaje", func(t *testing.T) {
		_, err := svc.CreateTipoIVA(ctx, "IVA 0", 0)
		if err == nil {
			t.Fatal("expected error for zero porcentaje")
		}
	})

	t.Run("negative porcentaje", func(t *testing.T) {
		_, err := svc.CreateTipoIVA(ctx, "IVA Negativo", -5.0)
		if err == nil {
			t.Fatal("expected error for negative porcentaje")
		}
	})
}

func TestCatalogService_CreateFamilia(t *testing.T) {
	ctx := context.Background()
	svc := NewCatalogService(newMockRepo())

	t.Run("success", func(t *testing.T) {
		f, err := svc.CreateFamilia(ctx, "Herramientas")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if f.Nombre != "Herramientas" {
			t.Errorf("expected nombre 'Herramientas', got %q", f.Nombre)
		}
		if f.ID == uuid.Nil {
			t.Error("expected non-nil ID")
		}
		if !f.Activo {
			t.Error("expected Activo=true")
		}
	})

	t.Run("empty nombre", func(t *testing.T) {
		_, err := svc.CreateFamilia(ctx, "")
		if err == nil {
			t.Fatal("expected error for empty nombre")
		}
	})
}

func TestCatalogService_CreateProducto(t *testing.T) {
	ctx := context.Background()
	mock := newMockRepo()
	ivaID := uuid.New()
	mock.seedTipoIVA(ivaID, "IVA General", 21.0)
	svc := NewCatalogService(mock)

	familiaID := uuid.New()

	t.Run("success", func(t *testing.T) {
		p, err := svc.CreateProducto(ctx, "P001", "Martillo", 15.50, &familiaID, ivaID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if p.Codigo != "P001" {
			t.Errorf("expected codigo 'P001', got %q", p.Codigo)
		}
		if p.Nombre != "Martillo" {
			t.Errorf("expected nombre 'Martillo', got %q", p.Nombre)
		}
		if p.PrecioVenta != 15.50 {
			t.Errorf("expected precio 15.50, got %f", p.PrecioVenta)
		}
		if p.FamiliaID == nil || *p.FamiliaID != familiaID {
			t.Error("familia_id not set correctly")
		}
		if p.TipoIvaID != ivaID {
			t.Error("tipo_iva_id not set correctly")
		}
		if p.ID == uuid.Nil {
			t.Error("expected non-nil ID")
		}
		if !p.Activo {
			t.Error("expected Activo=true")
		}
	})

	t.Run("success without familia", func(t *testing.T) {
		p, err := svc.CreateProducto(ctx, "P002", "Clavos", 3.00, nil, ivaID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if p.FamiliaID != nil {
			t.Error("expected nil familia_id")
		}
	})

	t.Run("empty codigo", func(t *testing.T) {
		_, err := svc.CreateProducto(ctx, "", "Martillo", 15.50, nil, ivaID)
		if err == nil {
			t.Fatal("expected error for empty codigo")
		}
	})

	t.Run("empty nombre", func(t *testing.T) {
		_, err := svc.CreateProducto(ctx, "P003", "", 15.50, nil, ivaID)
		if err == nil {
			t.Fatal("expected error for empty nombre")
		}
	})

	t.Run("negative precio", func(t *testing.T) {
		_, err := svc.CreateProducto(ctx, "P004", "Martillo", -1.00, nil, ivaID)
		if err == nil {
			t.Fatal("expected error for negative precio")
		}
	})

	t.Run("invalid tipo_iva_id", func(t *testing.T) {
		badID := uuid.New()
		_, err := svc.CreateProducto(ctx, "P005", "Martillo", 10.0, nil, badID)
		if err == nil {
			t.Fatal("expected error for non-existent tipo_iva_id")
		}
	})
}

func TestCatalogService_CreateCliente(t *testing.T) {
	ctx := context.Background()
	svc := NewCatalogService(newMockRepo())
	empresaID := uuid.New()

	t.Run("success", func(t *testing.T) {
		email := "cliente@test.com"
		telefono := "123456789"
		c, err := svc.CreateCliente(ctx, empresaID, "Cliente S.A.", "B12345678", &email, &telefono)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if c.RazonSocial != "Cliente S.A." {
			t.Errorf("expected 'Cliente S.A.', got %q", c.RazonSocial)
		}
		if c.NIF != "B12345678" {
			t.Errorf("expected NIF 'B12345678', got %q", c.NIF)
		}
		if c.Roles != "CLIENTE" {
			t.Errorf("expected roles 'CLIENTE', got %q", c.Roles)
		}
		if c.EmpresaID != empresaID {
			t.Error("empresa_id not set correctly")
		}
		if c.ID == uuid.Nil {
			t.Error("expected non-nil ID")
		}
		if !c.Activo {
			t.Error("expected Activo=true")
		}
	})

	t.Run("success without optional fields", func(t *testing.T) {
		c, err := svc.CreateCliente(ctx, empresaID, "Cliente Básico", "B87654321", nil, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if c.Email != nil {
			t.Error("expected nil email")
		}
		if c.Telefono != nil {
			t.Error("expected nil telefono")
		}
	})

	t.Run("empty razon_social", func(t *testing.T) {
		_, err := svc.CreateCliente(ctx, empresaID, "", "B12345678", nil, nil)
		if err == nil {
			t.Fatal("expected error for empty razon_social")
		}
	})

	t.Run("empty nif", func(t *testing.T) {
		_, err := svc.CreateCliente(ctx, empresaID, "Cliente S.A.", "", nil, nil)
		if err == nil {
			t.Fatal("expected error for empty nif")
		}
	})
}

func TestCatalogService_UpdateTipoIVA(t *testing.T) {
	ctx := context.Background()
	mock := newMockRepo()
	svc := NewCatalogService(mock)

	iva, _ := svc.CreateTipoIVA(ctx, "IVA Original", 10.0)

	t.Run("success", func(t *testing.T) {
		iva.Nombre = "IVA Actualizado"
		iva.Porcentaje = 15.0
		err := svc.UpdateTipoIVA(ctx, iva)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		updated, _ := svc.GetTipoIVA(ctx, iva.ID)
		if updated.Nombre != "IVA Actualizado" {
			t.Errorf("expected 'IVA Actualizado', got %q", updated.Nombre)
		}
		if updated.Porcentaje != 15.0 {
			t.Errorf("expected 15.0, got %f", updated.Porcentaje)
		}
	})

	t.Run("empty nombre", func(t *testing.T) {
		err := svc.UpdateTipoIVA(ctx, &TipoIVA{ID: iva.ID, Nombre: "", Porcentaje: 10.0})
		if err == nil {
			t.Fatal("expected error for empty nombre")
		}
	})

	t.Run("zero porcentaje", func(t *testing.T) {
		err := svc.UpdateTipoIVA(ctx, &TipoIVA{ID: iva.ID, Nombre: "IVA", Porcentaje: 0})
		if err == nil {
			t.Fatal("expected error for zero porcentaje")
		}
	})
}

func TestCatalogService_UpdateProducto(t *testing.T) {
	ctx := context.Background()
	mock := newMockRepo()
	ivaID := uuid.New()
	mock.seedTipoIVA(ivaID, "IVA General", 21.0)
	svc := NewCatalogService(mock)

	p, _ := svc.CreateProducto(ctx, "P001", "Martillo", 15.50, nil, ivaID)

	t.Run("success", func(t *testing.T) {
		p.Nombre = "Martillo Actualizado"
		p.PrecioVenta = 18.00
		err := svc.UpdateProducto(ctx, p)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("empty codigo", func(t *testing.T) {
		err := svc.UpdateProducto(ctx, &Producto{ID: p.ID, Codigo: "", Nombre: "Test", PrecioVenta: 10})
		if err == nil {
			t.Fatal("expected error for empty codigo")
		}
	})

	t.Run("empty nombre", func(t *testing.T) {
		err := svc.UpdateProducto(ctx, &Producto{ID: p.ID, Codigo: "P001", Nombre: "", PrecioVenta: 10})
		if err == nil {
			t.Fatal("expected error for empty nombre")
		}
	})

	t.Run("negative precio", func(t *testing.T) {
		err := svc.UpdateProducto(ctx, &Producto{ID: p.ID, Codigo: "P001", Nombre: "Test", PrecioVenta: -1})
		if err == nil {
			t.Fatal("expected error for negative precio")
		}
	})
}

func TestCatalogService_UpdateCliente(t *testing.T) {
	ctx := context.Background()
	mock := newMockRepo()
	svc := NewCatalogService(mock)
	empresaID := uuid.New()

	c, _ := svc.CreateCliente(ctx, empresaID, "Cliente S.A.", "B12345678", nil, nil)

	t.Run("success", func(t *testing.T) {
		c.RazonSocial = "Cliente Actualizado S.A."
		err := svc.UpdateCliente(ctx, c)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("empty razon_social", func(t *testing.T) {
		err := svc.UpdateCliente(ctx, &Entidad{ID: c.ID, RazonSocial: "", NIF: "B12345678"})
		if err == nil {
			t.Fatal("expected error for empty razon_social")
		}
	})

	t.Run("empty nif", func(t *testing.T) {
		err := svc.UpdateCliente(ctx, &Entidad{ID: c.ID, RazonSocial: "Test", NIF: ""})
		if err == nil {
			t.Fatal("expected error for empty nif")
		}
	})
}

func TestCatalogService_DeleteAndGet(t *testing.T) {
	ctx := context.Background()
	mock := newMockRepo()
	svc := NewCatalogService(mock)

	t.Run("get tipo iva after create", func(t *testing.T) {
		created, _ := svc.CreateTipoIVA(ctx, "IVA Reducido", 10.0)
		got, err := svc.GetTipoIVA(ctx, created.ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got.ID != created.ID {
			t.Error("ID mismatch")
		}
	})

	t.Run("list tipos iva", func(t *testing.T) {
		list, err := svc.ListTiposIVA(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(list) == 0 {
			t.Error("expected non-empty list")
		}
	})

	t.Run("delete tipo iva", func(t *testing.T) {
		created, _ := svc.CreateTipoIVA(ctx, "IVA Exento", 0.01)
		err := svc.DeleteTipoIVA(ctx, created.ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
