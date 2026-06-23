# Tasks: Verifactu Chaining

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 400-500 lines |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (Unit 1) → PR 2 (Unit 2) → PR 3 (Unit 3) → PR 4 (Unit 4) → PR 5 (Unit 5) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | DB migrations and signature trait | PR 1 | Base branch; local SQLite, Postgres DDL, and signature.rs |
| 2 | SQLite local chaining and cleanup | PR 2 | Stacked on PR 1; updates save_offline_sale and cleanup |
| 3 | Go sync API and events backend | PR 3 | Stacked on PR 2; Postgres updates and /api/v1/sync/events |
| 4 | Background sync loop worker | PR 4 | Stacked on PR 3; sync.rs background worker changes |
| 5 | Tests and verification | PR 5 | Stacked on PR 4; unit & integration tests validation |

## Phase 1: Foundation (Database DDL & Cryptography)

- [x] 1.1 Modify `database/migrations/sqlite_init.sql` to add `ultimo_registro_encadenado` and `registro_sucesos` tables.
- [x] 1.2 Modify `database/migrations/000001_init_erp_schemas.up.sql` to add `registro_sucesos` table and chaining columns to postgres.
- [x] 1.3 Create `tpv-client/src-tauri/src/signature.rs` defining the `Firmador` trait and its mock implementation `FirmaSimulada` using SHA-256.

## Phase 2: Core Local Chaining (SQLite Persistence)

- [x] 2.1 Update `tpv-client/src-tauri/src/db.rs` to include table schemas, updates, and query functions for last signature.
- [x] 2.2 Modify `save_offline_sale` in `tpv-client/src-tauri/src/lib.rs` to query/update last signature in SQLite transactions.
- [x] 2.3 Update cleanup logic in `tpv-client/src-tauri/src/lib.rs` to preserve `ultimo_registro_encadenado` after synchronization.

## Phase 3: Backend Sync APIs (Go Backend Integration)

- [x] 3.1 Update `SyncSale` struct in `internal/sync/adapters/api_controller.go` to include chaining metadata.
- [x] 3.2 Add `SyncEvent` struct and `POST /api/v1/sync/events` route in `internal/sync/adapters/api_controller.go`.
- [x] 3.3 Modify central db persistence in Go to store `SyncSale` chaining data and `SyncEvent` events log in Postgres.

## Phase 4: Client Background Loop (Sync Worker)

- [x] 4.1 Update `SyncSalePayload` in `tpv-client/src-tauri/src/sync.rs` to include chaining fields.
- [x] 4.2 Modify sync worker background loop in `tpv-client/src-tauri/src/sync.rs` to include `registro_sucesos` sync via POST.

## Phase 5: Verification & Testing

- [x] 5.1 Add unit tests for `FirmaSimulada` SHA-256 hash generation logic in `signature.rs`.
- [x] 5.2 Implement local integration test for SQLite transaction integrity during offline chaining.
- [x] 5.3 Write integration test for background worker verifying sales and event sync to the Go mock endpoints.
