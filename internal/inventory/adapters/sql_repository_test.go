package adapters_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"ferrowin/internal/inventory/adapters"
	"ferrowin/internal/inventory/domain"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open test SQLite DB: %v", err)
	}

	queries := []string{
		`CREATE TABLE stock_ledger_movements (
			id TEXT PRIMARY KEY,
			item_id TEXT NOT NULL,
			warehouse_id TEXT NOT NULL,
			quantity REAL NOT NULL,
			movement_type TEXT NOT NULL,
			reference_document_type TEXT,
			reference_document_id TEXT,
			created_at TEXT NOT NULL
		)`,
	}

	for _, q := range queries {
		if _, err = db.Exec(q); err != nil {
			db.Close()
			t.Fatalf("failed to run query %q: %v", q, err)
		}
	}

	cleanup := func() {
		db.Close()
	}
	return db, cleanup
}

func TestSQLStockLedgerRepository_Save(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := adapters.NewSQLStockLedgerRepository(db, true)
	ctx := context.Background()

	t.Run("insert receipt movement", func(t *testing.T) {
		entry := &domain.StockLedgerEntry{
			ID:           uuid.New(),
			ItemID:       uuid.New(),
			WarehouseID:  uuid.New(),
			Quantity:     100.0,
			MovementType: domain.MovementTypeReceipt,
			CreatedAt:    time.Now(),
		}

		err := repo.Save(ctx, entry)
		if err != nil {
			t.Fatalf("failed to save receipt movement: %v", err)
		}
	})

	t.Run("insert withdrawal movement", func(t *testing.T) {
		entry := &domain.StockLedgerEntry{
			ID:           uuid.New(),
			ItemID:       uuid.New(),
			WarehouseID:  uuid.New(),
			Quantity:     -50.0,
			MovementType: domain.MovementTypeWithdrawal,
			CreatedAt:    time.Now(),
		}

		err := repo.Save(ctx, entry)
		if err != nil {
			t.Fatalf("failed to save withdrawal movement: %v", err)
		}
	})

	t.Run("insert with reference document", func(t *testing.T) {
		refType := "PURCHASE_ORDER"
		refID := uuid.New()
		entry := &domain.StockLedgerEntry{
			ID:                    uuid.New(),
			ItemID:                uuid.New(),
			WarehouseID:           uuid.New(),
			Quantity:              75.0,
			MovementType:          domain.MovementTypeReceipt,
			ReferenceDocumentType: &refType,
			ReferenceDocumentID:   &refID,
			CreatedAt:             time.Now(),
		}

		err := repo.Save(ctx, entry)
		if err != nil {
			t.Fatalf("failed to save movement with reference: %v", err)
		}

		movements, err := repo.GetMovements(ctx, entry.ItemID, entry.WarehouseID)
		if err != nil {
			t.Fatalf("failed to get movements: %v", err)
		}
		if len(movements) != 1 {
			t.Fatalf("expected 1 movement, got %d", len(movements))
		}
		if movements[0].ReferenceDocumentType == nil {
			t.Fatal("expected reference document type to be non-nil")
		}
		if *movements[0].ReferenceDocumentType != refType {
			t.Errorf("expected ref type %s, got %s", refType, *movements[0].ReferenceDocumentType)
		}
		if movements[0].ReferenceDocumentID == nil {
			t.Fatal("expected reference document ID to be non-nil")
		}
		if *movements[0].ReferenceDocumentID != refID {
			t.Errorf("expected ref ID %s, got %s", refID, *movements[0].ReferenceDocumentID)
		}
	})

	t.Run("insert without references (nil pointers)", func(t *testing.T) {
		entry := &domain.StockLedgerEntry{
			ID:          uuid.New(),
			ItemID:      uuid.New(),
			WarehouseID: uuid.New(),
			Quantity:    30.0,
			MovementType: domain.MovementTypeSyncAdjustment,
			CreatedAt:   time.Now(),
		}

		err := repo.Save(ctx, entry)
		if err != nil {
			t.Fatalf("failed to save movement without references: %v", err)
		}

		movements, err := repo.GetMovements(ctx, entry.ItemID, entry.WarehouseID)
		if err != nil {
			t.Fatalf("failed to get movements: %v", err)
		}
		if len(movements) != 1 {
			t.Fatalf("expected 1 movement, got %d", len(movements))
		}
		if movements[0].ReferenceDocumentType != nil {
			t.Errorf("expected nil ref type, got %s", *movements[0].ReferenceDocumentType)
		}
		if movements[0].ReferenceDocumentID != nil {
			t.Errorf("expected nil ref ID, got %v", movements[0].ReferenceDocumentID)
		}
	})
}

func TestSQLStockLedgerRepository_GetMovements(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := adapters.NewSQLStockLedgerRepository(db, true)
	ctx := context.Background()

	t.Run("returns movements for specific item+warehouse pair", func(t *testing.T) {
		itemID := uuid.New()
		warehouseID := uuid.New()
		entry := &domain.StockLedgerEntry{
			ID:           uuid.New(),
			ItemID:       itemID,
			WarehouseID:  warehouseID,
			Quantity:     200.0,
			MovementType: domain.MovementTypeReceipt,
			CreatedAt:    time.Now(),
		}

		err := repo.Save(ctx, entry)
		if err != nil {
			t.Fatalf("failed to save movement: %v", err)
		}

		movements, err := repo.GetMovements(ctx, itemID, warehouseID)
		if err != nil {
			t.Fatalf("failed to get movements: %v", err)
		}
		if len(movements) != 1 {
			t.Fatalf("expected 1 movement, got %d", len(movements))
		}
		if movements[0].ID != entry.ID {
			t.Errorf("expected movement ID %s, got %s", entry.ID, movements[0].ID)
		}
		if movements[0].ItemID != itemID {
			t.Errorf("expected item ID %s, got %s", itemID, movements[0].ItemID)
		}
		if movements[0].WarehouseID != warehouseID {
			t.Errorf("expected warehouse ID %s, got %s", warehouseID, movements[0].WarehouseID)
		}
		if movements[0].MovementType != domain.MovementTypeReceipt {
			t.Errorf("expected movement type RECEIPT, got %s", movements[0].MovementType)
		}
	})

	t.Run("returns empty slice when no movements exist", func(t *testing.T) {
		movements, err := repo.GetMovements(ctx, uuid.New(), uuid.New())
		if err != nil {
			t.Fatalf("failed to get movements: %v", err)
		}
		if len(movements) != 0 {
			t.Errorf("expected empty slice, got %d items", len(movements))
		}
	})

	t.Run("returns movements ordered by created_at ASC", func(t *testing.T) {
		itemID := uuid.New()
		warehouseID := uuid.New()
		now := time.Now()

		t1 := now.Add(-2 * time.Hour)
		t2 := now.Add(-1 * time.Hour)
		t3 := now

		e1 := &domain.StockLedgerEntry{ID: uuid.New(), ItemID: itemID, WarehouseID: warehouseID, Quantity: 10, MovementType: domain.MovementTypeReceipt, CreatedAt: t1}
		e2 := &domain.StockLedgerEntry{ID: uuid.New(), ItemID: itemID, WarehouseID: warehouseID, Quantity: 20, MovementType: domain.MovementTypeReceipt, CreatedAt: t2}
		e3 := &domain.StockLedgerEntry{ID: uuid.New(), ItemID: itemID, WarehouseID: warehouseID, Quantity: 30, MovementType: domain.MovementTypeWithdrawal, CreatedAt: t3}

		for _, e := range []*domain.StockLedgerEntry{e1, e2, e3} {
			if err := repo.Save(ctx, e); err != nil {
				t.Fatalf("failed to save movement: %v", err)
			}
		}

		movements, err := repo.GetMovements(ctx, itemID, warehouseID)
		if err != nil {
			t.Fatalf("failed to get movements: %v", err)
		}
		if len(movements) != 3 {
			t.Fatalf("expected 3 movements, got %d", len(movements))
		}

		if movements[0].ID != e1.ID {
			t.Errorf("expected first movement ID %s, got %s", e1.ID, movements[0].ID)
		}
		if movements[1].ID != e2.ID {
			t.Errorf("expected second movement ID %s, got %s", e2.ID, movements[1].ID)
		}
		if movements[2].ID != e3.ID {
			t.Errorf("expected third movement ID %s, got %s", e3.ID, movements[2].ID)
		}
	})

	t.Run("does not return movements for different item or warehouse", func(t *testing.T) {
		itemA := uuid.New()
		itemB := uuid.New()
		whA := uuid.New()
		whB := uuid.New()

		entryA := &domain.StockLedgerEntry{ID: uuid.New(), ItemID: itemA, WarehouseID: whA, Quantity: 50, MovementType: domain.MovementTypeReceipt, CreatedAt: time.Now()}
		entryB := &domain.StockLedgerEntry{ID: uuid.New(), ItemID: itemB, WarehouseID: whB, Quantity: 60, MovementType: domain.MovementTypeReceipt, CreatedAt: time.Now()}

		if err := repo.Save(ctx, entryA); err != nil {
			t.Fatalf("failed to save movement A: %v", err)
		}
		if err := repo.Save(ctx, entryB); err != nil {
			t.Fatalf("failed to save movement B: %v", err)
		}

		// Query for itemA+whA should return only entryA
		movementsA, err := repo.GetMovements(ctx, itemA, whA)
		if err != nil {
			t.Fatalf("failed to get movements for A: %v", err)
		}
		if len(movementsA) != 1 || movementsA[0].ID != entryA.ID {
			t.Errorf("expected only entryA for pair A, got %d items", len(movementsA))
		}

		// Query for itemA+whB should return nothing
		movementsCross, err := repo.GetMovements(ctx, itemA, whB)
		if err != nil {
			t.Fatalf("failed to get movements for mismatched pair: %v", err)
		}
		if len(movementsCross) != 0 {
			t.Errorf("expected empty for mismatched item+warehouse, got %d items", len(movementsCross))
		}
	})

	t.Run("multiple movements return correct quantities and types", func(t *testing.T) {
		itemID := uuid.New()
		warehouseID := uuid.New()
		now := time.Now()

		entries := []*domain.StockLedgerEntry{
			{ID: uuid.New(), ItemID: itemID, WarehouseID: warehouseID, Quantity: 100, MovementType: domain.MovementTypeReceipt, CreatedAt: now.Add(-3 * time.Hour)},
			{ID: uuid.New(), ItemID: itemID, WarehouseID: warehouseID, Quantity: -20, MovementType: domain.MovementTypeWithdrawal, CreatedAt: now.Add(-2 * time.Hour)},
			{ID: uuid.New(), ItemID: itemID, WarehouseID: warehouseID, Quantity: -30, MovementType: domain.MovementTypeWithdrawal, CreatedAt: now.Add(-1 * time.Hour)},
			{ID: uuid.New(), ItemID: itemID, WarehouseID: warehouseID, Quantity: 15, MovementType: domain.MovementTypeSyncAdjustment, CreatedAt: now},
		}

		for _, e := range entries {
			if err := repo.Save(ctx, e); err != nil {
				t.Fatalf("failed to save movement: %v", err)
			}
		}

		movements, err := repo.GetMovements(ctx, itemID, warehouseID)
		if err != nil {
			t.Fatalf("failed to get movements: %v", err)
		}
		if len(movements) != 4 {
			t.Fatalf("expected 4 movements, got %d", len(movements))
		}

		if movements[0].Quantity != 100 || movements[0].MovementType != domain.MovementTypeReceipt {
			t.Errorf("first movement: expected qty=100 type=RECEIPT, got qty=%f type=%s", movements[0].Quantity, movements[0].MovementType)
		}
		if movements[1].Quantity != -20 || movements[1].MovementType != domain.MovementTypeWithdrawal {
			t.Errorf("second movement: expected qty=-20 type=WITHDRAWAL, got qty=%f type=%s", movements[1].Quantity, movements[1].MovementType)
		}
		if movements[2].Quantity != -30 || movements[2].MovementType != domain.MovementTypeWithdrawal {
			t.Errorf("third movement: expected qty=-30 type=WITHDRAWAL, got qty=%f type=%s", movements[2].Quantity, movements[2].MovementType)
		}
		if movements[3].Quantity != 15 || movements[3].MovementType != domain.MovementTypeSyncAdjustment {
			t.Errorf("fourth movement: expected qty=15 type=SYNC_ADJUSTMENT, got qty=%f type=%s", movements[3].Quantity, movements[3].MovementType)
		}
	})
}

func setupTxTestDB(t *testing.T) (*sql.DB, func()) {
	// Use a file-based database so multiple connections can read/write
	// independently without blocking (unlike :memory: with cache=shared).
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open test SQLite DB: %v", err)
	}

	if _, err = db.Exec(`CREATE TABLE stock_ledger_movements (
		id TEXT PRIMARY KEY,
		item_id TEXT NOT NULL,
		warehouse_id TEXT NOT NULL,
		quantity REAL NOT NULL,
		movement_type TEXT NOT NULL,
		reference_document_type TEXT,
		reference_document_id TEXT,
		created_at TEXT NOT NULL
	)`); err != nil {
		db.Close()
		t.Fatalf("failed to create table: %v", err)
	}

	cleanup := func() {
		db.Close()
	}
	return db, cleanup
}

func TestSQLStockLedgerRepository_TransactionPropagation(t *testing.T) {
	db, cleanup := setupTxTestDB(t)
	defer cleanup()

	repo := adapters.NewSQLStockLedgerRepository(db, true)
	ctx := context.Background()

	itemID := uuid.New()
	warehouseID := uuid.New()

	t.Run("save visible inside transaction before commit", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to start tx: %v", err)
		}
		defer tx.Rollback()

		txCtx := adapters.WithTx(ctx, tx)

		entry := &domain.StockLedgerEntry{
			ID:           uuid.New(),
			ItemID:       itemID,
			WarehouseID:  warehouseID,
			Quantity:     99.0,
			MovementType: domain.MovementTypeReceipt,
			CreatedAt:    time.Now(),
		}

		err = repo.Save(txCtx, entry)
		if err != nil {
			t.Fatalf("failed to save in tx: %v", err)
		}

		// Read via the same transaction context — must see uncommitted data
		fetched, err := repo.GetMovements(txCtx, itemID, warehouseID)
		if err != nil {
			t.Fatalf("failed to query movements inside tx: %v", err)
		}
		if len(fetched) != 1 {
			t.Fatal("expected movement to be visible inside transaction before commit")
		}
		if fetched[0].ID != entry.ID {
			t.Errorf("expected entry ID %s, got %s", entry.ID, fetched[0].ID)
		}
	})

	t.Run("rollback discards uncommitted data", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to start tx: %v", err)
		}

		txCtx := adapters.WithTx(ctx, tx)

		rollbackItem := uuid.New()
		entry := &domain.StockLedgerEntry{
			ID:           uuid.New(),
			ItemID:       rollbackItem,
			WarehouseID:  warehouseID,
			Quantity:     55.0,
			MovementType: domain.MovementTypeReceipt,
			CreatedAt:    time.Now(),
		}

		err = repo.Save(txCtx, entry)
		if err != nil {
			t.Fatalf("failed to save in tx: %v", err)
		}

		err = tx.Rollback()
		if err != nil {
			t.Fatalf("failed to rollback: %v", err)
		}

		fetched, err := repo.GetMovements(ctx, rollbackItem, warehouseID)
		if err != nil {
			t.Fatalf("failed to query movements after rollback: %v", err)
		}
		if len(fetched) != 0 {
			t.Fatal("expected no movements after rollback")
		}
	})

	t.Run("commit persists data", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to start tx: %v", err)
		}

		txCtx := adapters.WithTx(ctx, tx)

		commitItem := uuid.New()
		entry := &domain.StockLedgerEntry{
			ID:           uuid.New(),
			ItemID:       commitItem,
			WarehouseID:  warehouseID,
			Quantity:     42.0,
			MovementType: domain.MovementTypeWithdrawal,
			CreatedAt:    time.Now(),
		}

		err = repo.Save(txCtx, entry)
		if err != nil {
			t.Fatalf("failed to save in tx: %v", err)
		}

		err = tx.Commit()
		if err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		fetched, err := repo.GetMovements(ctx, commitItem, warehouseID)
		if err != nil {
			t.Fatalf("failed to query movements after commit: %v", err)
		}
		if len(fetched) != 1 {
			t.Fatal("expected movement to exist after commit")
		}
		if fetched[0].ID != entry.ID {
			t.Errorf("expected entry ID %s, got %s", entry.ID, fetched[0].ID)
		}
		if fetched[0].Quantity != 42.0 {
			t.Errorf("expected quantity 42, got %f", fetched[0].Quantity)
		}
	})
}
