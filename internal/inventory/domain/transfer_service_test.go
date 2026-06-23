package domain_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"ferrowin/internal/inventory/adapters"
	"ferrowin/internal/inventory/domain"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// fakeWarehouseValidator implements domain.WarehouseValidator for tests.
type fakeWarehouseValidator struct {
	warehouses map[uuid.UUID]*domain.WarehouseView
}

func (v *fakeWarehouseValidator) GetWarehouse(_ context.Context, id uuid.UUID) (*domain.WarehouseView, error) {
	if w, ok := v.warehouses[id]; ok {
		return w, nil
	}
	return nil, errors.New("warehouse not found")
}

func setupTransferTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	queries := []string{
		`CREATE TABLE traspasos_almacen (
			id TEXT PRIMARY KEY,
			empresa_id TEXT NOT NULL,
			origen_id TEXT NOT NULL,
			destino_id TEXT NOT NULL,
			estado TEXT NOT NULL CHECK (estado IN ('Borrador', 'Procesado', 'Cancelado')),
			created_at TEXT NOT NULL,
			processed_at TEXT,
			cancelled_at TEXT
		)`,
		`CREATE TABLE traspaso_almacen_lineas (
			id TEXT PRIMARY KEY,
			traspaso_almacen_id TEXT NOT NULL REFERENCES traspasos_almacen(id) ON DELETE CASCADE,
			producto_id TEXT NOT NULL,
			cantidad REAL NOT NULL CHECK (cantidad > 0)
		)`,
		`CREATE TABLE stock_ledger_movements (
			id TEXT PRIMARY KEY,
			item_id TEXT NOT NULL,
			warehouse_id TEXT NOT NULL,
			quantity REAL NOT NULL,
			movement_type TEXT NOT NULL CHECK (movement_type IN ('RECEIPT', 'WITHDRAWAL', 'SYNC_ADJUSTMENT', 'TRANSFER')),
			reference_document_type TEXT,
			reference_document_id TEXT,
			created_at TEXT NOT NULL
		)`,
	}

	for _, q := range queries {
		if _, err = db.Exec(q); err != nil {
			db.Close()
			t.Fatalf("failed to run query: %v\nSQL: %s", err, q)
		}
	}

	cleanup := func() { db.Close() }
	return db, cleanup
}

func setupTransferService(t *testing.T, now time.Time) (*domain.TransferService, *domain.InventoryService, *fakeWarehouseValidator, func()) {
	db, cleanup := setupTransferTestDB(t)

	ledgerRepo := adapters.NewSQLStockLedgerRepository(db, true)
	invService := domain.NewInventoryService(ledgerRepo)

	transferRepo := adapters.NewSQLTransferRepository(db, true)

	whValidator := &fakeWarehouseValidator{
		warehouses: make(map[uuid.UUID]*domain.WarehouseView),
	}

	svc := domain.NewTransferService(transferRepo, whValidator)
	svc.Now = func() time.Time { return now }

	return svc, invService, whValidator, cleanup
}

func TestTransferService_Create(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("Scenario: Create valid transfer", func(t *testing.T) {
		svc, _, validator, cleanup := setupTransferService(t, now)
		defer cleanup()

		empresaID := uuid.New()
		origenID := uuid.New()
		destinoID := uuid.New()

		validator.warehouses[origenID] = &domain.WarehouseView{ID: origenID, EmpresaID: empresaID}
		validator.warehouses[destinoID] = &domain.WarehouseView{ID: destinoID, EmpresaID: empresaID}

		tran, err := svc.Create(ctx, empresaID, origenID, destinoID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tran.Estado != domain.TraspasoBorrador {
			t.Errorf("expected estado Borrador, got %s", tran.Estado)
		}
		if tran.OrigenID != origenID {
			t.Errorf("expected origen %s, got %s", origenID, tran.OrigenID)
		}
		if tran.DestinoID != destinoID {
			t.Errorf("expected destino %s, got %s", destinoID, tran.DestinoID)
		}
		if len(tran.Lineas) != 0 {
			t.Errorf("expected 0 lineas, got %d", len(tran.Lineas))
		}
	})

	t.Run("Scenario: Create with same warehouse returns error", func(t *testing.T) {
		svc, _, _, cleanup := setupTransferService(t, now)
		defer cleanup()

		whID := uuid.New()
		_, err := svc.Create(ctx, uuid.New(), whID, whID)
		if !errors.Is(err, domain.ErrTransferSameWarehouse) {
			t.Errorf("expected ErrTransferSameWarehouse, got %v", err)
		}
	})

	t.Run("Scenario: Create with cross-company warehouses returns error", func(t *testing.T) {
		svc, _, validator, cleanup := setupTransferService(t, now)
		defer cleanup()

		empresaID := uuid.New()
		origenID := uuid.New()
		destinoID := uuid.New()

		validator.warehouses[origenID] = &domain.WarehouseView{ID: origenID, EmpresaID: empresaID}
		validator.warehouses[destinoID] = &domain.WarehouseView{ID: destinoID, EmpresaID: uuid.New()}

		_, err := svc.Create(ctx, empresaID, origenID, destinoID)
		if !errors.Is(err, domain.ErrTransferCrossCompany) {
			t.Errorf("expected ErrTransferCrossCompany, got %v", err)
		}
	})

	t.Run("Scenario: Create with nonexistent warehouse returns error", func(t *testing.T) {
		svc, _, _, cleanup := setupTransferService(t, now)
		defer cleanup()

		_, err := svc.Create(ctx, uuid.New(), uuid.New(), uuid.New())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestTransferService_AddLine(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("Scenario: Add line to Borrador transfer", func(t *testing.T) {
		svc, _, validator, cleanup := setupTransferService(t, now)
		defer cleanup()

		empresaID := uuid.New()
		origenID := uuid.New()
		destinoID := uuid.New()
		validator.warehouses[origenID] = &domain.WarehouseView{ID: origenID, EmpresaID: empresaID}
		validator.warehouses[destinoID] = &domain.WarehouseView{ID: destinoID, EmpresaID: empresaID}

		tran, err := svc.Create(ctx, empresaID, origenID, destinoID)
		if err != nil {
			t.Fatalf("failed to create: %v", err)
		}

		productoID := uuid.New()
		updated, err := svc.AddLine(ctx, empresaID, tran.ID, productoID, 10.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(updated.Lineas) != 1 {
			t.Fatalf("expected 1 linea, got %d", len(updated.Lineas))
		}
		if updated.Lineas[0].ProductoID != productoID {
			t.Errorf("expected producto %s, got %s", productoID, updated.Lineas[0].ProductoID)
		}
		if updated.Lineas[0].Cantidad != 10.0 {
			t.Errorf("expected cantidad 10.0, got %f", updated.Lineas[0].Cantidad)
		}
	})

	t.Run("Scenario: Add line to processed transfer returns error", func(t *testing.T) {
		svc, _, validator, cleanup := setupTransferService(t, now)
		defer cleanup()

		empresaID := uuid.New()
		origenID := uuid.New()
		destinoID := uuid.New()
		validator.warehouses[origenID] = &domain.WarehouseView{ID: origenID, EmpresaID: empresaID}
		validator.warehouses[destinoID] = &domain.WarehouseView{ID: destinoID, EmpresaID: empresaID}

		tran, err := svc.Create(ctx, empresaID, origenID, destinoID)
		if err != nil {
			t.Fatalf("failed to create: %v", err)
		}

		productoID := uuid.New()
		_, err = svc.AddLine(ctx, empresaID, tran.ID, productoID, 10.0)
		if err != nil {
			t.Fatalf("failed to add line: %v", err)
		}

		// Process it
		err = svc.Process(ctx, empresaID, tran.ID)
		if err != nil {
			t.Fatalf("failed to process: %v", err)
		}

		// Try adding line after processing
		_, err = svc.AddLine(ctx, empresaID, tran.ID, uuid.New(), 5.0)
		if !errors.Is(err, domain.ErrTransferNotEditable) {
			t.Errorf("expected ErrTransferNotEditable, got %v", err)
		}
	})

	t.Run("Scenario: Add line with zero quantity returns error", func(t *testing.T) {
		svc, _, validator, cleanup := setupTransferService(t, now)
		defer cleanup()

		empresaID := uuid.New()
		origenID := uuid.New()
		destinoID := uuid.New()
		validator.warehouses[origenID] = &domain.WarehouseView{ID: origenID, EmpresaID: empresaID}
		validator.warehouses[destinoID] = &domain.WarehouseView{ID: destinoID, EmpresaID: empresaID}

		tran, err := svc.Create(ctx, empresaID, origenID, destinoID)
		if err != nil {
			t.Fatalf("failed to create: %v", err)
		}

		_, err = svc.AddLine(ctx, empresaID, tran.ID, uuid.New(), 0)
		if !errors.Is(err, domain.ErrInvalidQuantity) {
			t.Errorf("expected ErrInvalidQuantity, got %v", err)
		}
	})
}

func TestTransferService_RemoveLine(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("Scenario: Remove line from Borrador transfer", func(t *testing.T) {
		svc, _, validator, cleanup := setupTransferService(t, now)
		defer cleanup()

		empresaID := uuid.New()
		origenID := uuid.New()
		destinoID := uuid.New()
		validator.warehouses[origenID] = &domain.WarehouseView{ID: origenID, EmpresaID: empresaID}
		validator.warehouses[destinoID] = &domain.WarehouseView{ID: destinoID, EmpresaID: empresaID}

		tran, err := svc.Create(ctx, empresaID, origenID, destinoID)
		if err != nil {
			t.Fatalf("failed to create: %v", err)
		}

		updated, err := svc.AddLine(ctx, empresaID, tran.ID, uuid.New(), 10.0)
		if err != nil {
			t.Fatalf("failed to add line: %v", err)
		}
		if len(updated.Lineas) != 1 {
			t.Fatalf("expected 1 line, got %d", len(updated.Lineas))
		}

		lineID := updated.Lineas[0].ID
		updated, err = svc.RemoveLine(ctx, empresaID, tran.ID, lineID)
		if err != nil {
			t.Fatalf("failed to remove line: %v", err)
		}
		if len(updated.Lineas) != 0 {
			t.Errorf("expected 0 lines after removal, got %d", len(updated.Lineas))
		}
	})
}

func TestTransferService_Process(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("Scenario: Full valid lifecycle", func(t *testing.T) {
		svc, invService, validator, cleanup := setupTransferService(t, now)
		defer cleanup()

		empresaID := uuid.New()
		origenID := uuid.New()
		destinoID := uuid.New()
		validator.warehouses[origenID] = &domain.WarehouseView{ID: origenID, EmpresaID: empresaID}
		validator.warehouses[destinoID] = &domain.WarehouseView{ID: destinoID, EmpresaID: empresaID}

		tran, err := svc.Create(ctx, empresaID, origenID, destinoID)
		if err != nil {
			t.Fatalf("failed to create: %v", err)
		}

		productoID := uuid.New()
		_, err = svc.AddLine(ctx, empresaID, tran.ID, productoID, 10.0)
		if err != nil {
			t.Fatalf("failed to add line: %v", err)
		}

		err = svc.Process(ctx, empresaID, tran.ID)
		if err != nil {
			t.Fatalf("failed to process: %v", err)
		}

		// Verify estado
		updated, err := svc.GetByID(ctx, tran.ID)
		if err != nil {
			t.Fatalf("failed to get by ID: %v", err)
		}
		if updated.Estado != domain.TraspasoProcesado {
			t.Errorf("expected estado Procesado, got %s", updated.Estado)
		}
		if updated.ProcessedAt == nil {
			t.Fatal("expected ProcessedAt to be set")
		}

		// Verify stock at origen: quantity should be -10
		stockOrigen, err := invService.GetAvailableStock(ctx, productoID, origenID)
		if err != nil {
			t.Fatalf("failed to get stock at origen: %v", err)
		}
		if stockOrigen != -10.0 {
			t.Errorf("expected stock at origen to be -10.0, got %f", stockOrigen)
		}

		// Verify stock at destino: quantity should be +10
		stockDestino, err := invService.GetAvailableStock(ctx, productoID, destinoID)
		if err != nil {
			t.Fatalf("failed to get stock at destino: %v", err)
		}
		if stockDestino != 10.0 {
			t.Errorf("expected stock at destino to be 10.0, got %f", stockDestino)
		}
	})

	t.Run("Scenario: Process twice returns error", func(t *testing.T) {
		svc, _, validator, cleanup := setupTransferService(t, now)
		defer cleanup()

		empresaID := uuid.New()
		origenID := uuid.New()
		destinoID := uuid.New()
		validator.warehouses[origenID] = &domain.WarehouseView{ID: origenID, EmpresaID: empresaID}
		validator.warehouses[destinoID] = &domain.WarehouseView{ID: destinoID, EmpresaID: empresaID}

		tran, err := svc.Create(ctx, empresaID, origenID, destinoID)
		if err != nil {
			t.Fatalf("failed to create: %v", err)
		}
		_, err = svc.AddLine(ctx, empresaID, tran.ID, uuid.New(), 5.0)
		if err != nil {
			t.Fatalf("failed to add line: %v", err)
		}

		err = svc.Process(ctx, empresaID, tran.ID)
		if err != nil {
			t.Fatalf("failed to process first time: %v", err)
		}

		err = svc.Process(ctx, empresaID, tran.ID)
		if !errors.Is(err, domain.ErrTransferAlreadyProcessed) {
			t.Errorf("expected ErrTransferAlreadyProcessed, got %v", err)
		}
	})

	t.Run("Scenario: Process transfer with no lines returns error", func(t *testing.T) {
		svc, _, validator, cleanup := setupTransferService(t, now)
		defer cleanup()

		empresaID := uuid.New()
		origenID := uuid.New()
		destinoID := uuid.New()
		validator.warehouses[origenID] = &domain.WarehouseView{ID: origenID, EmpresaID: empresaID}
		validator.warehouses[destinoID] = &domain.WarehouseView{ID: destinoID, EmpresaID: empresaID}

		tran, err := svc.Create(ctx, empresaID, origenID, destinoID)
		if err != nil {
			t.Fatalf("failed to create: %v", err)
		}

		err = svc.Process(ctx, empresaID, tran.ID)
		if !errors.Is(err, domain.ErrTransferNoLines) {
			t.Errorf("expected ErrTransferNoLines, got %v", err)
		}
	})

	t.Run("Scenario: Process cross-company returns error", func(t *testing.T) {
		svc, _, validator, cleanup := setupTransferService(t, now)
		defer cleanup()

		empresaID := uuid.New()
		origenID := uuid.New()
		destinoID := uuid.New()
		validator.warehouses[origenID] = &domain.WarehouseView{ID: origenID, EmpresaID: empresaID}
		validator.warehouses[destinoID] = &domain.WarehouseView{ID: destinoID, EmpresaID: empresaID}

		tran, err := svc.Create(ctx, empresaID, origenID, destinoID)
		if err != nil {
			t.Fatalf("failed to create: %v", err)
		}
		_, err = svc.AddLine(ctx, empresaID, tran.ID, uuid.New(), 5.0)
		if err != nil {
			t.Fatalf("failed to add line: %v", err)
		}

		// Process with different empresaID
		err = svc.Process(ctx, uuid.New(), tran.ID)
		if !errors.Is(err, domain.ErrTransferCrossCompany) {
			t.Errorf("expected ErrTransferCrossCompany, got %v", err)
		}
	})
}

func TestTransferService_GetByID(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("Scenario: Get existing transfer", func(t *testing.T) {
		svc, _, validator, cleanup := setupTransferService(t, now)
		defer cleanup()

		empresaID := uuid.New()
		origenID := uuid.New()
		destinoID := uuid.New()
		validator.warehouses[origenID] = &domain.WarehouseView{ID: origenID, EmpresaID: empresaID}
		validator.warehouses[destinoID] = &domain.WarehouseView{ID: destinoID, EmpresaID: empresaID}

		created, err := svc.Create(ctx, empresaID, origenID, destinoID)
		if err != nil {
			t.Fatalf("failed to create: %v", err)
		}

		fetched, err := svc.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("failed to get by ID: %v", err)
		}
		if fetched.ID != created.ID {
			t.Errorf("expected ID %s, got %s", created.ID, fetched.ID)
		}
	})

	t.Run("Scenario: Get nonexistent transfer returns error", func(t *testing.T) {
		svc, _, _, cleanup := setupTransferService(t, now)
		defer cleanup()

		_, err := svc.GetByID(ctx, uuid.New())
		if !errors.Is(err, domain.ErrTransferNotFound) {
			t.Errorf("expected ErrTransferNotFound, got %v", err)
		}
	})
}

func TestTransferService_List(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("Scenario: Filter by estado", func(t *testing.T) {
		svc, _, validator, cleanup := setupTransferService(t, now)
		defer cleanup()

		empresaID := uuid.New()
		origenID := uuid.New()
		destinoID := uuid.New()
		destino2ID := uuid.New()
		validator.warehouses[origenID] = &domain.WarehouseView{ID: origenID, EmpresaID: empresaID}
		validator.warehouses[destinoID] = &domain.WarehouseView{ID: destinoID, EmpresaID: empresaID}
		validator.warehouses[destino2ID] = &domain.WarehouseView{ID: destino2ID, EmpresaID: empresaID}

		// Create 2 transfers
		t1, err := svc.Create(ctx, empresaID, origenID, destinoID)
		if err != nil {
			t.Fatalf("failed to create transfer 1: %v", err)
		}
		_, err = svc.AddLine(ctx, empresaID, t1.ID, uuid.New(), 10.0)
		if err != nil {
			t.Fatalf("failed to add line: %v", err)
		}

		t2, err := svc.Create(ctx, empresaID, origenID, destino2ID)
		if err != nil {
			t.Fatalf("failed to create transfer 2: %v", err)
		}
		_, err = svc.AddLine(ctx, empresaID, t2.ID, uuid.New(), 5.0)
		if err != nil {
			t.Fatalf("failed to add line: %v", err)
		}

		// Process t1 so it becomes Procesado
		err = svc.Process(ctx, empresaID, t1.ID)
		if err != nil {
			t.Fatalf("failed to process t1: %v", err)
		}

		// Filter by Borrador
		borrador := domain.TraspasoBorrador
		results, total, err := svc.List(ctx, empresaID, domain.TransferFilter{Estado: &borrador})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}
		if total != 1 {
			t.Errorf("expected total 1, got %d", total)
		}
		if len(results) != 1 || results[0].ID != t2.ID {
			t.Errorf("expected only transfer 2 (Borrador), got %d results", len(results))
		}

		// Filter by Procesado
		procesado := domain.TraspasoProcesado
		results, total, err = svc.List(ctx, empresaID, domain.TransferFilter{Estado: &procesado})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}
		if total != 1 {
			t.Errorf("expected total 1 for Procesado, got %d", total)
		}
		if len(results) != 1 || results[0].ID != t1.ID {
			t.Errorf("expected only transfer 1 (Procesado), got %d results", len(results))
		}
	})

	t.Run("Scenario: Pagination", func(t *testing.T) {
		svc, _, validator, cleanup := setupTransferService(t, now)
		defer cleanup()

		empresaID := uuid.New()
		origenID := uuid.New()
		destinoID := uuid.New()
		validator.warehouses[origenID] = &domain.WarehouseView{ID: origenID, EmpresaID: empresaID}
		validator.warehouses[destinoID] = &domain.WarehouseView{ID: destinoID, EmpresaID: empresaID}

		// Create 3 transfers
		var ids []uuid.UUID
		for i := 0; i < 3; i++ {
			tr, err := svc.Create(ctx, empresaID, origenID, destinoID)
			if err != nil {
				t.Fatalf("failed to create transfer %d: %v", i, err)
			}
			ids = append(ids, tr.ID)
		}

		// List with page_size=2, page=1
		results, total, err := svc.List(ctx, empresaID, domain.TransferFilter{Page: 1, PageSize: 2})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}
		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 results on page 1, got %d", len(results))
		}

		// List with page_size=2, page=2
		results, total, err = svc.List(ctx, empresaID, domain.TransferFilter{Page: 2, PageSize: 2})
		if err != nil {
			t.Fatalf("failed to list page 2: %v", err)
		}
		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
		if len(results) != 1 {
			t.Errorf("expected 1 result on page 2, got %d", len(results))
		}
	})
}
