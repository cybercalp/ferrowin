# Proposal: Offline POS Stock Reconciliation

## Intent

Resolve stock tracking discrepancies for reconnected terminals and offline sales. Currently, the sync backend bypasses the inventory domain service (skipping FIFO reconciliation and negative stock allowance), and the offline POS doesn't update its local stock cache on offline sales.

## Scope

### In Scope
- Integrate `InventoryService` (`RecordSyncAdjustment` and `ReconcileFIFO`) in Go sync backend transactional handlers.
- Propagate database transactions in Go repository via context.
- Decrement local SQLite `stock_cache` in Tauri client on offline sales.

### Out of Scope
- Frontend UI for manual stock conflict resolution.
- Live notifications for negative stock.

## Capabilities

### New Capabilities
- None

### Modified Capabilities
- offline-pos-stock: Enable real-time central stock queries when online and fallback to local SQLite cache when offline. Perform FIFO stock reconciliation on backend Go sync.

## Approach

- **Backend Transaction Propagation**: Update the Go repository to check for an active transaction `*sql.Tx` in the `context.Context`. If present, execute repository operations within it. Update the API controller to wrap sync logic and FIFO reconciliation inside a single database transaction.
- **Client Cache Updates**: Modify `save_offline_sale_impl` in Tauri to run a SQLite update query decrementing cached stock quantities for sold items within the SQLite transaction.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/sync/adapters/api_controller.go` | Modified | Use InventoryService to sync sales and trigger FIFO reconciliation inside a transaction. |
| `internal/inventory/adapters/sql_repository.go` | Modified | Add context-based transaction propagation to execute SQL queries within the sync transaction. |
| `tpv-client/src-tauri/src/lib.rs` | Modified | In save_offline_sale_impl, update sqlite stock_cache by decrementing the sold quantities. |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|-------------|
| Cache Drift | Medium | Automatically resolved when the POS client reconnects and retrieves live central stock. |
| DB Lock Contention | Low | Use optimized indexes and ensure the sync transaction is short-lived. |

## Rollback Plan

Revert code changes in `api_controller.go`, `sql_repository.go`, and `lib.rs` to restore direct sql writing on backend and bypass local cache update on frontend.

## Dependencies

- None

## Success Criteria

- [ ] Syncing offline sales updates central stock (allows negative stock) and triggers FIFO reconciliation.
- [ ] Offline POS displays cached stock and decrements stock locally on sales.
- [ ] Online POS retrieves real-time stock from central backend.
