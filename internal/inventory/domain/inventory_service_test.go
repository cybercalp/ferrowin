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

func setupInventoryTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE stock_ledger_movements (
			id TEXT PRIMARY KEY,
			item_id TEXT NOT NULL,
			warehouse_id TEXT NOT NULL,
			quantity REAL NOT NULL,
			movement_type TEXT NOT NULL CHECK (movement_type IN ('RECEIPT', 'WITHDRAWAL', 'SYNC_ADJUSTMENT', 'TRANSFER')),
			reference_document_type TEXT,
			reference_document_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create stock_ledger_movements table: %v", err)
	}

	cleanup := func() {
		db.Close()
	}
	return db, cleanup
}

func TestInventoryService_RecordMovements(t *testing.T) {
	ctx := context.Background()

	t.Run("Scenario: Stock receipt entry", func(t *testing.T) {
		db, cleanup := setupInventoryTestDB(t)
		defer cleanup()

		repo := adapters.NewSQLStockLedgerRepository(db, true)
		svc := domain.NewInventoryService(repo)

		itemID := uuid.New()
		warehouseID := uuid.New()

		// Initial stock is 0
		qty, err := svc.GetAvailableStock(ctx, itemID, warehouseID)
		if err != nil {
			t.Fatalf("failed to get available stock: %v", err)
		}
		if qty != 0 {
			t.Errorf("expected stock to be 0, got %f", qty)
		}

		// Record a receipt of +10
		entry, err := svc.RecordReceipt(ctx, itemID, warehouseID, 10.0, nil, nil)
		if err != nil {
			t.Fatalf("failed to record receipt: %v", err)
		}
		if entry.Quantity != 10.0 {
			t.Errorf("expected recorded quantity to be 10.0, got %f", entry.Quantity)
		}

		// Available stock should be 10
		qty, err = svc.GetAvailableStock(ctx, itemID, warehouseID)
		if err != nil {
			t.Fatalf("failed to get available stock: %v", err)
		}
		if qty != 10.0 {
			t.Errorf("expected available stock to be 10.0, got %f", qty)
		}

		// Record another receipt of +5
		_, err = svc.RecordReceipt(ctx, itemID, warehouseID, 5.0, nil, nil)
		if err != nil {
			t.Fatalf("failed to record receipt: %v", err)
		}

		// Available stock should be 15
		qty, err = svc.GetAvailableStock(ctx, itemID, warehouseID)
		if err != nil {
			t.Fatalf("failed to get available stock: %v", err)
		}
		if qty != 15.0 {
			t.Errorf("expected available stock to be 15.0, got %f", qty)
		}
	})

	t.Run("Scenario: Insufficient stock rejection", func(t *testing.T) {
		db, cleanup := setupInventoryTestDB(t)
		defer cleanup()

		repo := adapters.NewSQLStockLedgerRepository(db, true)
		svc := domain.NewInventoryService(repo)

		itemID := uuid.New()
		warehouseID := uuid.New()

		// GIVEN warehouse W1 with item I1 stock at 2
		_, err := svc.RecordReceipt(ctx, itemID, warehouseID, 2.0, nil, nil)
		if err != nil {
			t.Fatalf("failed to setup initial stock: %v", err)
		}

		// WHEN a stock withdrawal of 3 is requested
		_, err = svc.RecordWithdrawal(ctx, itemID, warehouseID, 3.0, nil, nil)

		// THEN the system MUST block the transaction and return insufficient stock error
		if !errors.Is(err, domain.ErrInsufficientStock) {
			t.Errorf("expected ErrInsufficientStock, got %v", err)
		}

		// Verify available stock remains 2
		qty, err := svc.GetAvailableStock(ctx, itemID, warehouseID)
		if err != nil {
			t.Fatalf("failed to get available stock: %v", err)
		}
		if qty != 2.0 {
			t.Errorf("expected available stock to remain 2.0, got %f", qty)
		}
	})

	t.Run("Scenario: Successful withdrawal within limits", func(t *testing.T) {
		db, cleanup := setupInventoryTestDB(t)
		defer cleanup()

		repo := adapters.NewSQLStockLedgerRepository(db, true)
		svc := domain.NewInventoryService(repo)

		itemID := uuid.New()
		warehouseID := uuid.New()

		_, err := svc.RecordReceipt(ctx, itemID, warehouseID, 10.0, nil, nil)
		if err != nil {
			t.Fatalf("failed to record receipt: %v", err)
		}

		entry, err := svc.RecordWithdrawal(ctx, itemID, warehouseID, 3.0, nil, nil)
		if err != nil {
			t.Fatalf("failed to record withdrawal: %v", err)
		}
		if entry.Quantity != -3.0 {
			t.Errorf("expected recorded quantity for withdrawal to be -3.0, got %f", entry.Quantity)
		}

		qty, err := svc.GetAvailableStock(ctx, itemID, warehouseID)
		if err != nil {
			t.Fatalf("failed to get available stock: %v", err)
		}
		if qty != 7.0 {
			t.Errorf("expected available stock to be 7.0, got %f", qty)
		}
	})

	t.Run("Scenario: Offline POS sync allows negative stock balance", func(t *testing.T) {
		db, cleanup := setupInventoryTestDB(t)
		defer cleanup()

		repo := adapters.NewSQLStockLedgerRepository(db, true)
		svc := domain.NewInventoryService(repo)

		itemID := uuid.New()
		warehouseID := uuid.New()

		// GIVEN central database with item I1 stock at 0
		qty, _ := svc.GetAvailableStock(ctx, itemID, warehouseID)
		if qty != 0 {
			t.Fatalf("expected initial stock to be 0, got %f", qty)
		}

		// WHEN terminal syncs an offline sale of 1 unit of I1
		entry, err := svc.RecordSyncAdjustment(ctx, itemID, warehouseID, 1.0, nil, nil)
		if err != nil {
			t.Fatalf("failed to record sync adjustment: %v", err)
		}
		if entry.Quantity != -1.0 {
			t.Errorf("expected sync adjustment quantity to be stored as -1.0, got %f", entry.Quantity)
		}

		// THEN the ledger MUST register the transaction and central stock drops to -1
		qty, err = svc.GetAvailableStock(ctx, itemID, warehouseID)
		if err != nil {
			t.Fatalf("failed to get available stock: %v", err)
		}
		if qty != -1.0 {
			t.Errorf("expected available stock to be -1.0, got %f", qty)
		}
	})
}

func TestInventoryService_FIFOReconciliation(t *testing.T) {
	ctx := context.Background()

	t.Run("Scenario: FIFO reconciliation on stock receipt", func(t *testing.T) {
		db, cleanup := setupInventoryTestDB(t)
		defer cleanup()

		repo := adapters.NewSQLStockLedgerRepository(db, true)
		svc := domain.NewInventoryService(repo)

		// Create a mock clock so we have sequential timestamps
		currentTime := time.Now().Add(-1 * time.Hour)
		svc.Now = func() time.Time {
			currentTime = currentTime.Add(1 * time.Minute)
			return currentTime
		}

		itemID := uuid.New()
		warehouseID := uuid.New()

		// GIVEN central stock of I1 is at -2 units from offline sales (2 sync adjustments of 1 unit each)
		syncAdj1, err := svc.RecordSyncAdjustment(ctx, itemID, warehouseID, 1.0, nil, nil)
		if err != nil {
			t.Fatalf("failed to record sync adjustment 1: %v", err)
		}

		syncAdj2, err := svc.RecordSyncAdjustment(ctx, itemID, warehouseID, 1.0, nil, nil)
		if err != nil {
			t.Fatalf("failed to record sync adjustment 2: %v", err)
		}

		stock, err := svc.GetAvailableStock(ctx, itemID, warehouseID)
		if err != nil {
			t.Fatalf("failed to get available stock: %v", err)
		}
		if stock != -2.0 {
			t.Fatalf("expected stock to be -2.0, got %f", stock)
		}

		// Run a FIFO reconciliation before the receipt to verify empty allocations
		allocs, err := svc.ReconcileFIFO(ctx, itemID, warehouseID)
		if err != nil {
			t.Fatalf("failed to run reconciliation: %v", err)
		}
		if len(allocs) != 0 {
			t.Errorf("expected 0 allocations before receipt, got %d", len(allocs))
		}

		// WHEN a stock receipt of +10 units is registered
		receipt, err := svc.RecordReceipt(ctx, itemID, warehouseID, 10.0, nil, nil)
		if err != nil {
			t.Fatalf("failed to record receipt: %v", err)
		}

		// THEN the system MUST clear the -2 units and update available stock to 8
		stock, err = svc.GetAvailableStock(ctx, itemID, warehouseID)
		if err != nil {
			t.Fatalf("failed to get available stock: %v", err)
		}
		if stock != 8.0 {
			t.Errorf("expected stock to update to 8.0, got %f", stock)
		}

		// Running reconciliation should now produce allocations
		allocs, err = svc.ReconcileFIFO(ctx, itemID, warehouseID)
		if err != nil {
			t.Fatalf("failed to run reconciliation: %v", err)
		}

		// Verify FIFO allocations
		if len(allocs) != 2 {
			t.Fatalf("expected 2 allocations, got %d", len(allocs))
		}

		// First allocation should be syncAdj1 matched with receipt for 1.0 unit
		if allocs[0].DemandID != syncAdj1.ID {
			t.Errorf("expected first allocation demand ID to be %s, got %s", syncAdj1.ID, allocs[0].DemandID)
		}
		if allocs[0].ReceiptID != receipt.ID {
			t.Errorf("expected first allocation receipt ID to be %s, got %s", receipt.ID, allocs[0].ReceiptID)
		}
		if allocs[0].QtyAllocated != 1.0 {
			t.Errorf("expected first allocation quantity to be 1.0, got %f", allocs[0].QtyAllocated)
		}

		// Second allocation should be syncAdj2 matched with receipt for 1.0 unit
		if allocs[1].DemandID != syncAdj2.ID {
			t.Errorf("expected second allocation demand ID to be %s, got %s", syncAdj2.ID, allocs[1].DemandID)
		}
		if allocs[1].ReceiptID != receipt.ID {
			t.Errorf("expected second allocation receipt ID to be %s, got %s", receipt.ID, allocs[1].ReceiptID)
		}
		if allocs[1].QtyAllocated != 1.0 {
			t.Errorf("expected second allocation quantity to be 1.0, got %f", allocs[1].QtyAllocated)
		}
	})
}
