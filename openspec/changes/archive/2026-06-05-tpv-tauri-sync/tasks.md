# Tasks: TPV Tauri Synchronization

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 800 - 1200 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 -> PR 2 -> PR 3 -> PR 4 |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Tauri + React project bootstrapping & local SQLite setup | PR 1 | Base branch; Cargo setup, SQLite init migrations |
| 2 | Backend Go Sync Closure API & idempotency locks | PR 2 | Backend endpoint implementation and DB schema updates |
| 3 | Tauri core Rust sync worker and HTTP loop | PR 3 | Background async loop, sync post logic in Rust |
| 4 | React Frontend UI, banner & SQLite query triggers | PR 4 | Banner component, state store, Tauri IPC triggers |

## Phase 1: Foundation & Infrastructure

- [x] 1.1 Scaffold Tauri+Vite+React project in `tpv-client/` and add SQLite driver dependency to `Cargo.toml`.
- [x] 1.2 Create SQLite schema file `database/migrations/sqlite_init.sql` with `offline_box_closures` and `stock_cache` tables.
- [x] 1.3 Add Go backend route `/api/v1/sync/closures` stub in `internal/sync/adapters/api_controller.go`.
- [x] 1.4 Test: Verify `tpv-client` cargo project compiles and Vite dev server runs.

## Phase 2: Core Implementation

- [x] 2.1 Implement `tpv-client/src-tauri/src/db.rs` to handle connection, schema migration, and offline CRUD operations.
- [x] 2.2 Implement backend Go closure sync handler `HandleSyncClosures` in `internal/sync/adapters/api_controller.go`.
- [x] 2.3 Add idempotency locks and verification inside backend Go closure sync database transaction.
- [x] 2.4 Implement autonomous background loop in `tpv-client/src-tauri/src/sync.rs` executing every 30 seconds.
- [x] 2.5 Add Rust-side network checks and POST sync logic inside `sync.rs` sending payload to Go API.
- [x] 2.6 Test: Assert unit tests for Rust SQLite CRUD helpers pass.

## Phase 3: Integration & UI

- [x] 3.1 Create React UI component `tpv-client/src/components/SyncWarningBanner.tsx` listening to Tauri events.
- [x] 3.2 Add local SQLite query triggers via Tauri Rust IPC commands inside `tpv-client/src-tauri/src/lib.rs`.
- [x] 3.3 Integrate `SyncWarningBanner` in `tpv-client/src/App.tsx` main layout view.
- [x] 3.4 Test: Verify banner UI displays correct state when simulated network/sync status events are emitted.

## Phase 4: Verification & Cleanup

- [x] 4.1 Write integration tests in `tpv-client/src-tauri/` validating sync behavior during offline/online cycles.
- [x] 4.2 Run end-to-end local test: Trigger offline closure, verify SQLite save, go online, verify Go DB sync.
- [x] 4.3 Clean up debug logs and verify that feature flag `enable-tauri-sync` behaves correctly.
