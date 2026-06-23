## Exploration: offline-pos-stock

### Current State
1. **Go Backend Sync**: In `internal/sync/adapters/api_controller.go`, the synchronization handler `HandleSyncSales` receives offline sales, inserts the invoice, and directly inserts stock movements into the `stock_ledger_movements` database table using a raw SQL insert statement. This bypasses the domain service `internal/inventory/domain/inventory_service.go`, meaning that `RecordSyncAdjustment` (which allows negative stock) and `ReconcileFIFO` (which runs the FIFO reconciliation calculations) are not invoked.
2. **Tauri Client Stock Cache**: In `tpv-client/src-tauri/src/lib.rs`, the Tauri command `get_stock` manages online vs offline queries. It first tests connection status by pinging `/api/v1/health`. If online, it queries `/api/v1/inventory/stock/{item_id}` from the Go API, stores it in the local SQLite table `stock_cache` via `db::upsert_stock_cache`, and returns it. If offline (or the backend request fails), it retrieves the last cached value from the local SQLite `stock_cache` using `db::get_cached_stock`.
3. **React Frontend Stock Query & Connection Status**: The frontend `tpv-client/src/App.tsx` makes stock queries via the `get_stock` Tauri command. Connection status is broadcast from the background sync loop in `tpv-client/src-tauri/src/sync.rs` (which performs a health ping and counts local pending sync records every 30 seconds), emitting `sync-status-changed` events. The React component `SyncWarningBanner.tsx` listens to this event on mount and updates the warning header dynamically.

### Affected Areas
- `internal/sync/adapters/api_controller.go` — Needs to use the `InventoryService` domain methods (`RecordSyncAdjustment` and `ReconcileFIFO`) within the active database transaction instead of direct SQL inserts.
- `internal/inventory/adapters/sql_repository.go` — Needs context-based transaction propagation to execute repository commands within the sync transaction.
- `tpv-client/src-tauri/src/lib.rs` — The `save_offline_sale_impl` helper must update the local `stock_cache` table by decrementing the sold quantities when a sale is recorded offline.

### Approaches
1. **Context-based Transaction Propagation (Go Backend)** — Retrieve the active transaction `*sql.Tx` from the `context.Context` inside the repository.
   - Pros: Keeps `InventoryService` interface clean and decoupled from transaction management; easily propagates the sync handler transaction.
   - Cons: Requires repository mapping to extract the transaction/db executor from the context.
   - Effort: Low

2. **Exposing explicit `WithTx` method on Repository (Go Backend)** — Instantiate a transactional variant of the repository.
   - Pros: Explicit transaction binding.
   - Cons: Couples the controller/service orchestrator to repository-specific details.
   - Effort: Medium

3. **Local Cache Decrement on Offline Sale (Tauri Client)** — Update the local SQLite `stock_cache` table by decrementing the sold quantities inside the SQLite transaction in `save_offline_sale_impl`.
   - Pros: Ensures the cashier sees immediate, accurate stock updates when offline sales occur.
   - Cons: Local cache might drift from central stock, but it self-corrects on reconnect.
   - Effort: Low

### Recommendation
Adopt **Context-based Transaction Propagation** in the Go backend. When synchronizing sales, invoke `RecordSyncAdjustment` and `ReconcileFIFO` on `InventoryService` using a context carrying the active `*sql.Tx`.
In the Tauri client, modify `save_offline_sale_impl` to decrement the quantities sold from the local SQLite `stock_cache` within the SQLite transaction.

### Risks
- **Concurrently Synced Sales Locks**: High transaction volume under Postgres or SQLite could lead to contention during FIFO reconciliation, but short-lived transactions and index optimization minimize lock duration.
- **Cache Drift**: Cashiers may see slightly drifted stock levels if multiple offline clients sell the same items before sync. Online stock fetch and delta catalog sync will self-correct it.

### Ready for Proposal
Yes — Proceed to generate the implementation proposal for offline stock reconciliation.
