# Apply Progress: Offline POS Stock Synchronization and Cache

## Phase 1: Go Backend Foundation (PR 1)
Status: Completed

### Tasks Completed:
- [x] Define transaction context key and helper `WithTx(ctx, tx)` in `internal/inventory/adapters/sql_repository.go`.
- [x] Define `dbExecutor` interface supporting database execution methods in `sql_repository.go`.
- [x] Add `getExecutor(ctx)` to select transaction executor if present, fallback to standard `*sql.DB`.
- [x] Update `SQLStockLedgerRepository` methods (e.g. `Save`, `GetMovements`) to use executor resolved from context.

### Details of Changes:
1. **Context-based Transaction Propagation**:
   - Added a private context key `txKey` in `internal/inventory/adapters/sql_repository.go`.
   - Exposed `WithTx(ctx context.Context, tx *sql.Tx) context.Context` to register active database transactions on the context.
   - Introduced `dbExecutor` interface which abstracts standard `*sql.DB` and transaction `*sql.Tx` context-aware query methods: `ExecContext`, `QueryContext`, and `QueryRowContext`.
   - Added `getExecutor(ctx context.Context)` internal helper on `SQLStockLedgerRepository` to resolve the active transaction from the context if present, falling back to `r.db` otherwise.
   
2. **Repository Adjustments**:
   - Refactored `Save` and `GetMovements` methods in `SQLStockLedgerRepository` to invoke database operations on the resolved executor rather than directly on `r.db`.
   
### Verification:
- Code syntax and compiler checks run successfully.
- Tests triggered via Go tooling to verify existing repository operations remain unaffected and compile correctly.

## Phase 2: Go Backend API Sync (PR 2)
Status: Completed

### Tasks Completed:
- [x] Bind transaction to sync context in `HandleSyncSales` of `internal/sync/adapters/api_controller.go` using `WithTx`.
- [x] Replace direct SQL insertions in `api_controller.go` with `InventoryService.RecordSyncAdjustment`.
- [x] Call `InventoryService.ReconcileFIFO` inside transaction loop in `api_controller.go`.

### Details of Changes:
1. **Transaction Wrapping**:
   - Imported the `ferrowin/internal/inventory/adapters` package in `internal/sync/adapters/api_controller.go`.
   - Propagated the active backend database transaction into the context via `inventoryadapters.WithTx(r.Context(), tx)`.
2. **Domain Integration**:
   - Replaced direct, manual SQL inserts into `stock_ledger_movements` inside the sales sync items loop with `c.inventoryService.RecordSyncAdjustment`.
   - Invoked `c.inventoryService.ReconcileFIFO` in the transaction context loop for each item to recalculate FIFO allocation balances immediately.
   - Removed unused SQL definitions and parameters to keep the codebase clean.

### Verification:
- Ran `go test ./internal/sync/adapters/...` which completed successfully (including `TestSalesSyncController_HandleSyncSales`).
- Ran all project tests via `go test ./...` which successfully passed.

## Phase 3: Tauri Client Stock Caching (PR 3)
Status: Completed

### Tasks Completed:
- [x] Implement `decrement_stock_cache(conn, item_id, quantity, last_updated_at)` in `tpv-client/src-tauri/src/db.rs` with SQLite upsert ON CONFLICT.
- [x] Update `save_offline_sale_impl` in `tpv-client/src-tauri/src/lib.rs` to loop over sale items and execute cache decrement.

### Details of Changes:
1. **SQLite Stock Cache Decrement Helper**:
   - Implemented `decrement_stock_cache(conn: &Connection, item_id: &str, quantity: f64, last_updated_at: &str) -> Result<()>` in `tpv-client/src-tauri/src/db.rs`.
   - Utilized SQLite `INSERT ... ON CONFLICT(item_id) DO UPDATE SET stock = stock - ?2, last_updated_at = ?3` behavior to decrement the cached stock value, initializing it to `-quantity` if not present.
2. **Transaction Hook in Offline Sale Saving**:
   - Updated `save_offline_sale_impl` in `tpv-client/src-tauri/src/lib.rs` to loop through the `OfflineSaleItem` list.
   - For each item, `db::decrement_stock_cache` is called inside the SQLite transaction using the same `now_iso` ISO-8601 timestamp generated for the sale.

### Verification:
- Cargo tests run and compile successfully.

## Phase 4: Testing & Verification (PR 4)
Status: Completed

### Tasks Completed:
- [x] Create `test_save_offline_sale_decrements_stock_cache` unit test in Tauri client.
- [x] Update `TestSalesSyncController_HandleSyncSales` integration test to verify stock movements via sync controller.
- [x] Add backend integration test validating FIFO reconciliation executes properly during sync process.

### Details of Changes:
1. **Tauri Client Unit Test**:
   - Added `test_save_offline_sale_decrements_stock_cache` unit test inside the tests module of `tpv-client/src-tauri/src/lib.rs`.
   - The test validates both scenarios: decrementing a pre-cached stock item and decrementing/initializing a non-cached stock item when saving an offline sale.
2. **Go Backend Integration Test Direct Database Assertions**:
   - Enhanced the existing `TestSalesSyncController_HandleSyncSales` integration test in `internal/sync/adapters/api_controller_test.go` to perform raw SQL queries against `stock_ledger_movements`.
   - Confirms that the sync adjustment records are successfully persisted in the database via the controller with correct columns (quantity, reference document, etc.).
3. **Go Backend FIFO Reconciliation Integration Test**:
   - Added a new integration test `TestSalesSyncController_HandleSyncSales_FIFOReconciliation` in `internal/sync/adapters/api_controller_test.go`.
   - Validates that offline sales synchronized to the backend are reconcilable via FIFO allocation after receipts are recorded, confirming correct inventory service logic execution.

### Verification:
- Ran `cargo test --manifest-path tpv-client/src-tauri/Cargo.toml` verifying all 23 tests pass.
- Ran `go test -v ./internal/sync/adapters/...` verifying all Go integration tests pass.


