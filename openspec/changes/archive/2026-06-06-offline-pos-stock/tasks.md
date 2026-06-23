# Implementation Tasks: Offline POS Stock Synchronization and Cache

## Review Workload Forecast
Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

## Tasks

### Phase 1: Go Backend Foundation (PR 1)
- [x] Define transaction context key and helper `WithTx(ctx, tx)` in `internal/inventory/adapters/sql_repository.go`.
- [x] Define `dbExecutor` interface supporting database execution methods in `sql_repository.go`.
- [x] Add `getExecutor(ctx)` to select transaction executor if present, fallback to standard `*sql.DB`.
- [x] Update `SQLStockLedgerRepository` methods (e.g. `Save`, `GetMovements`) to use executor resolved from context.

### Phase 2: Go Backend API Sync (PR 2)
- [x] Bind transaction to sync context in `HandleSyncSales` of `internal/sync/adapters/api_controller.go` using `WithTx`.
- [x] Replace direct SQL insertions in `api_controller.go` with `InventoryService.RecordSyncAdjustment`.
- [x] Call `InventoryService.ReconcileFIFO` inside transaction loop in `api_controller.go`.

### Phase 3: Tauri Client Stock Caching (PR 3)
- [x] Implement `decrement_stock_cache(conn, item_id, quantity, last_updated_at)` in `tpv-client/src-tauri/src/db.rs` with SQLite upsert ON CONFLICT.
- [x] Update `save_offline_sale_impl` in `tpv-client/src-tauri/src/lib.rs` to loop over sale items and execute cache decrement.

### Phase 4: Testing & Verification (PR 4)
- [x] Create `test_save_offline_sale_decrements_stock_cache` unit test in Tauri client.
- [x] Update `TestSalesSyncController_HandleSyncSales` integration test to verify stock movements via sync controller.
- [x] Add backend integration test validating FIFO reconciliation executes properly during sync process.
