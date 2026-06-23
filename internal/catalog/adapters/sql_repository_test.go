package adapters_test

import (
	"context"
	"database/sql"
	"testing"

	catalogadapters "ferrowin/internal/catalog/adapters"
	catalogdomain "ferrowin/internal/catalog/domain"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open test SQLite DB: %v", err)
	}

	queries := []string{
		`CREATE TABLE empresas (
			id TEXT PRIMARY KEY,
			razon_social TEXT NOT NULL,
			nif TEXT UNIQUE NOT NULL,
			activa INTEGER DEFAULT 1
		)`,
		`CREATE TABLE tipos_iva (
			id TEXT PRIMARY KEY,
			nombre TEXT NOT NULL,
			porcentaje REAL NOT NULL,
			updated_at DATETIME,
			activo INTEGER DEFAULT 1
		)`,
		`CREATE TABLE familias (
			id TEXT PRIMARY KEY,
			nombre TEXT NOT NULL,
			updated_at DATETIME,
			activo INTEGER DEFAULT 1
		)`,
		`CREATE TABLE productos (
			id TEXT PRIMARY KEY,
			codigo TEXT UNIQUE NOT NULL,
			nombre TEXT NOT NULL,
			precio_venta REAL NOT NULL,
			familia_id TEXT REFERENCES familias(id),
			tipo_iva_id TEXT REFERENCES tipos_iva(id),
			updated_at DATETIME,
			activo INTEGER DEFAULT 1
		)`,
		`CREATE TABLE entidades (
			id TEXT PRIMARY KEY,
			empresa_id TEXT NOT NULL REFERENCES empresas(id) ON DELETE CASCADE,
			razon_social TEXT NOT NULL,
			nif TEXT NOT NULL,
			email TEXT,
			telefono TEXT,
			activo INTEGER DEFAULT 1,
			roles TEXT NOT NULL,
			updated_at DATETIME,
			UNIQUE(empresa_id, nif)
		)`,
	}

	for _, q := range queries {
		if _, err = db.Exec(q); err != nil {
			db.Close()
			t.Fatalf("failed to run query %q: %v", q, err)
		}
	}

	// Seed a default empresa for clientes tests
	_, err = db.Exec("INSERT INTO empresas (id, razon_social, nif, activa) VALUES ('00000000-0000-4000-a000-000000000001', 'Default Corp', 'DEFAULTNIF', 1)")
	if err != nil {
		db.Close()
		t.Fatalf("failed to seed empresa: %v", err)
	}

	cleanup := func() {
		db.Close()
	}
	return db, cleanup
}

func TestSQLCatalogRepository_TiposIVA(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := catalogadapters.NewSQLCatalogRepository(db, true)
	ctx := context.Background()

	// Create
	tipoID := uuid.New()
	err := repo.CreateTipoIVA(ctx, &catalogdomain.TipoIVA{
		ID:         tipoID,
		Nombre:     "General",
		Porcentaje: 21.00,
		Activo:     true,
	})
	if err != nil {
		t.Fatalf("CreateTipoIVA failed: %v", err)
	}

	// Get
	got, err := repo.GetTipoIVA(ctx, tipoID)
	if err != nil {
		t.Fatalf("GetTipoIVA failed: %v", err)
	}
	if got.Nombre != "General" || got.Porcentaje != 21.00 {
		t.Errorf("unexpected tipo IVA: %+v", got)
	}

	// List
	list, err := repo.ListTiposIVA(ctx)
	if err != nil {
		t.Fatalf("ListTiposIVA failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 tipo IVA, got %d", len(list))
	}

	// Update
	got.Nombre = "General Actualizado"
	got.Porcentaje = 10.00
	if err := repo.UpdateTipoIVA(ctx, got); err != nil {
		t.Fatalf("UpdateTipoIVA failed: %v", err)
	}
	updated, _ := repo.GetTipoIVA(ctx, tipoID)
	if updated.Nombre != "General Actualizado" || updated.Porcentaje != 10.00 {
		t.Errorf("update not applied: %+v", updated)
	}

	// Soft delete
	if err := repo.DeleteTipoIVA(ctx, tipoID); err != nil {
		t.Fatalf("DeleteTipoIVA failed: %v", err)
	}
	deleted, _ := repo.GetTipoIVA(ctx, tipoID)
	if deleted.Activo {
		t.Error("expected tipo IVA to be inactive after soft delete")
	}
}

func TestSQLCatalogRepository_Familias(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := catalogadapters.NewSQLCatalogRepository(db, true)
	ctx := context.Background()

	// Create
	famID := uuid.New()
	err := repo.CreateFamilia(ctx, &catalogdomain.Familia{
		ID:     famID,
		Nombre: "Electrónica",
		Activo: true,
	})
	if err != nil {
		t.Fatalf("CreateFamilia failed: %v", err)
	}

	// Get
	got, err := repo.GetFamilia(ctx, famID)
	if err != nil {
		t.Fatalf("GetFamilia failed: %v", err)
	}
	if got.Nombre != "Electrónica" {
		t.Errorf("unexpected familia: %+v", got)
	}

	// List
	list, err := repo.ListFamilias(ctx)
	if err != nil {
		t.Fatalf("ListFamilias failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 familia, got %d", len(list))
	}

	// Update
	got.Nombre = "Electrónica y Computación"
	if err := repo.UpdateFamilia(ctx, got); err != nil {
		t.Fatalf("UpdateFamilia failed: %v", err)
	}
	updated, _ := repo.GetFamilia(ctx, famID)
	if updated.Nombre != "Electrónica y Computación" {
		t.Errorf("update not applied: %+v", updated)
	}

	// Soft delete
	if err := repo.DeleteFamilia(ctx, famID); err != nil {
		t.Fatalf("DeleteFamilia failed: %v", err)
	}
	deleted, _ := repo.GetFamilia(ctx, famID)
	if deleted.Activo {
		t.Error("expected familia to be inactive after soft delete")
	}
}

func TestSQLCatalogRepository_Productos(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := catalogadapters.NewSQLCatalogRepository(db, true)
	ctx := context.Background()

	// Seed required reference data
	ivaID := uuid.New()
	_, err := db.Exec("INSERT INTO tipos_iva (id, nombre, porcentaje, activo) VALUES (?, 'General', 21.00, 1)", ivaID.String())
	if err != nil {
		t.Fatalf("seed iva failed: %v", err)
	}

	famID := uuid.New()
	_, err = db.Exec("INSERT INTO familias (id, nombre, activo) VALUES (?, 'Electrónica', 1)", famID.String())
	if err != nil {
		t.Fatalf("seed familia failed: %v", err)
	}

	// Create producto with familia
	prodID := uuid.New()
	err = repo.CreateProducto(ctx, &catalogdomain.Producto{
		ID:          prodID,
		Codigo:      "PROD-001",
		Nombre:      "Tornillo M6",
		PrecioVenta: 1.50,
		FamiliaID:   &famID,
		TipoIvaID:   ivaID,
		Activo:      true,
	})
	if err != nil {
		t.Fatalf("CreateProducto failed: %v", err)
	}

	// Get
	got, err := repo.GetProducto(ctx, prodID)
	if err != nil {
		t.Fatalf("GetProducto failed: %v", err)
	}
	if got.Codigo != "PROD-001" || got.Nombre != "Tornillo M6" {
		t.Errorf("unexpected producto: %+v", got)
	}
	if got.FamiliaID == nil || *got.FamiliaID != famID {
		t.Error("expected familia_id to be set")
	}

	// Create producto without familia
	prod2ID := uuid.New()
	err = repo.CreateProducto(ctx, &catalogdomain.Producto{
		ID:          prod2ID,
		Codigo:      "PROD-002",
		Nombre:      "Tuerca M6",
		PrecioVenta: 0.75,
		TipoIvaID:   ivaID,
		Activo:      true,
	})
	if err != nil {
		t.Fatalf("CreateProducto without familia failed: %v", err)
	}

	// List all
	list, err := repo.ListProductos(ctx, nil, nil)
	if err != nil {
		t.Fatalf("ListProductos failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 productos, got %d", len(list))
	}

	// Filter by familia_id
	filtered, err := repo.ListProductos(ctx, &famID, nil)
	if err != nil {
		t.Fatalf("ListProductos with familia filter failed: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 producto filtered by familia, got %d", len(filtered))
	}

	// Filter by tipo_iva_id
	filtered2, err := repo.ListProductos(ctx, nil, &ivaID)
	if err != nil {
		t.Fatalf("ListProductos with iva filter failed: %v", err)
	}
	if len(filtered2) != 2 {
		t.Fatalf("expected 2 productos filtered by iva, got %d", len(filtered2))
	}

	// Update
	got.Nombre = "Tornillo M8"
	got.PrecioVenta = 2.00
	got.FamiliaID = nil
	if err := repo.UpdateProducto(ctx, got); err != nil {
		t.Fatalf("UpdateProducto failed: %v", err)
	}
	updated, _ := repo.GetProducto(ctx, prodID)
	if updated.Nombre != "Tornillo M8" || updated.PrecioVenta != 2.00 {
		t.Errorf("update not applied: %+v", updated)
	}
	if updated.FamiliaID != nil {
		t.Error("expected familia_id to be nil after update")
	}

	// Soft delete
	if err := repo.DeleteProducto(ctx, prodID); err != nil {
		t.Fatalf("DeleteProducto failed: %v", err)
	}
	deleted, _ := repo.GetProducto(ctx, prodID)
	if deleted.Activo {
		t.Error("expected producto to be inactive after soft delete")
	}

	// Verify soft delete still returns it via Get
	stillThere, err := repo.GetProducto(ctx, prodID)
	if err != nil {
		t.Fatalf("GetProducto after soft delete should still return the item: %v", err)
	}
	if stillThere.Activo {
		t.Error("expected producto to be inactive")
	}
}

func TestSQLCatalogRepository_Clientes(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := catalogadapters.NewSQLCatalogRepository(db, true)
	ctx := context.Background()

	empresaID := uuid.MustParse("00000000-0000-4000-a000-000000000001")

	email := "cliente@test.com"
	telefono := "555-0100"
	cliID := uuid.New()

	// Create
	err := repo.CreateCliente(ctx, &catalogdomain.Entidad{
		ID:          cliID,
		EmpresaID:   empresaID,
		RazonSocial: "Cliente de Prueba S.L.",
		NIF:         "B12345678",
		Email:       &email,
		Telefono:    &telefono,
		Activo:      true,
		Roles:       "CLIENTE",
	})
	if err != nil {
		t.Fatalf("CreateCliente failed: %v", err)
	}

	// Get
	got, err := repo.GetCliente(ctx, cliID)
	if err != nil {
		t.Fatalf("GetCliente failed: %v", err)
	}
	if got.RazonSocial != "Cliente de Prueba S.L." || got.NIF != "B12345678" {
		t.Errorf("unexpected cliente: %+v", got)
	}
	if got.Email == nil || *got.Email != "cliente@test.com" {
		t.Error("email not set correctly")
	}
	if got.Roles != "CLIENTE" {
		t.Errorf("expected roles=CLIENTE, got %s", got.Roles)
	}

	// List
	list, err := repo.ListClientes(ctx, empresaID)
	if err != nil {
		t.Fatalf("ListClientes failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 cliente, got %d", len(list))
	}

	// Create a supplier that should NOT appear in clientes list
	_, err = db.Exec("INSERT INTO entidades (id, empresa_id, razon_social, nif, roles, activo) VALUES (?, ?, ?, ?, 'PROVEEDOR', 1)",
		uuid.New().String(), empresaID.String(), "Proveedor Solo", "C87654321")
	if err != nil {
		t.Fatalf("seed supplier failed: %v", err)
	}

	// Verify clientes list still returns only 1 (the supplier is excluded)
	list, err = repo.ListClientes(ctx, empresaID)
	if err != nil {
		t.Fatalf("ListClientes after adding supplier failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 cliente after adding supplier, got %d", len(list))
	}

	// Update
	newEmail := "nuevo@test.com"
	got.Email = &newEmail
	got.Telefono = nil
	if err := repo.UpdateCliente(ctx, got); err != nil {
		t.Fatalf("UpdateCliente failed: %v", err)
	}
	updated, _ := repo.GetCliente(ctx, cliID)
	if updated.Email == nil || *updated.Email != "nuevo@test.com" {
		t.Error("email update not applied")
	}
	if updated.Telefono != nil {
		t.Error("expected telefono to be nil after update")
	}

	// Soft delete
	if err := repo.DeleteCliente(ctx, cliID); err != nil {
		t.Fatalf("DeleteCliente failed: %v", err)
	}
	deleted, _ := repo.GetCliente(ctx, cliID)
	if deleted.Activo {
		t.Error("expected cliente to be inactive after soft delete")
	}

	// Verify supplier-only entity cannot be fetched via GetCliente
	supplierOnlyID := uuid.New()
	_, err = db.Exec("INSERT INTO entidades (id, empresa_id, razon_social, nif, roles, activo) VALUES (?, ?, ?, ?, 'PROVEEDOR', 1)",
		supplierOnlyID.String(), empresaID.String(), "Supplier Only", "D99999999")
	if err != nil {
		t.Fatalf("seed supplier-only failed: %v", err)
	}
	_, err = repo.GetCliente(ctx, supplierOnlyID)
	if err == nil {
		t.Error("expected error when fetching supplier-only entity via GetCliente")
	}
}
