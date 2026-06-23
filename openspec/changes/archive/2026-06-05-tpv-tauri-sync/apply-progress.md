# Apply Progress: tpv-tauri-sync

**Change**: tpv-tauri-sync
**Mode**: Standard
**Workload / PR Boundary**:
- Mode: stacked-to-main
- Current work unit: Unit 4 (React Frontend UI, banner & SQLite query triggers)
- Boundary: Phase 3 tasks 3.1 - 3.4

## Completed Tasks

### Phase 1 (prior batch)
- [x] 1.1 Scaffold Tauri+Vite+React project in `tpv-client/` and add SQLite driver dependency to `Cargo.toml`.
- [x] 1.2 Create SQLite schema file `database/migrations/sqlite_init.sql` with `offline_box_closures` and `stock_cache` tables.
- [x] 1.3 Add Go backend route `/api/v1/sync/closures` stub in `internal/sync/adapters/api_controller.go`.
- [x] 1.4 Test: Verify `tpv-client` cargo project compiles and Vite dev server runs.

### Phase 2 (prior batch)
- [x] 2.1 Implement `tpv-client/src-tauri/src/db.rs` to handle connection, schema migration, and offline CRUD operations.
- [x] 2.2 Implement backend Go closure sync handler `HandleSyncClosures` in `internal/sync/adapters/api_controller.go`.
- [x] 2.3 Add idempotency locks and verification inside backend Go closure sync database transaction.
- [x] 2.4 Implement autonomous background loop in `tpv-client/src-tauri/src/sync.rs` executing every 30 seconds.
- [x] 2.5 Add Rust-side network checks and POST sync logic inside `sync.rs` sending payload to Go API.
- [x] 2.6 Test: Assert unit tests for Rust SQLite CRUD helpers pass.

### Phase 3 (this batch)
- [x] 3.1 Create React UI component `tpv-client/src/components/SyncWarningBanner.tsx` listening to Tauri events.
- [x] 3.2 Add local SQLite query triggers via Tauri Rust IPC commands inside `tpv-client/src-tauri/src/lib.rs`.
- [x] 3.3 Integrate `SyncWarningBanner` in `tpv-client/src/App.tsx` main layout view.
- [x] 3.4 Test: Verify banner UI displays correct state when simulated network/sync status events are emitted.

## Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `tpv-client/src/components/SyncWarningBanner.tsx` | Created | React UI component listening to `sync-status-changed` to show warning banner (offline) or sync notification (online). |
| `tpv-client/src-tauri/src/lib.rs` | Modified | Exposed `save_offline_sale`, `save_offline_closure`, and `get_stock` Tauri IPC commands and managed `DbState`. |
| `tpv-client/src-tauri/Cargo.toml` | Modified | Added `chrono` dependency. |
| `tpv-client/src/App.tsx` | Modified | Rendered `SyncWarningBanner` and created simulator controls, SQLite operations panel, stock query form, and console logs window. |
| `internal/sync/adapters/api_controller.go` | Modified | Added `/api/v1/health` and `/api/v1/inventory/stock/:item_id` route support and `HandleGetStock` handler. |
| `openspec/changes/tpv-tauri-sync/tasks.md` | Modified | Marked tasks 3.1-3.4 as `[x]`. |

## Deviations from Design
None — implementation matches design specifications.

## Issues Found
None.

## Remaining Tasks
- [ ] 4.1 Write integration tests in `tpv-client/src-tauri/` validating sync behavior during offline/online cycles.
- [ ] 4.2 Run end-to-end local test: Trigger offline closure, verify SQLite save, go online, verify Go DB sync.
- [ ] 4.3 Clean up debug logs and verify that feature flag `enable-tauri-sync` behaves correctly.
