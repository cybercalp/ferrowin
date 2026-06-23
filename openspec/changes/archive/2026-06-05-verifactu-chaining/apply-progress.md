# Apply Progress: verifactu-chaining

**Change**: verifactu-chaining
**Mode**: Standard

### Completed Tasks
- [x] 1.1 Modify `database/migrations/sqlite_init.sql` to add:
  - Columns `firma_registro TEXT`, `hash_anterior TEXT`, `datos_encadenamiento TEXT` to the `offline_sales` table definition.
  - Table `ultimo_registro_encadenado` (columns: `id_factura` TEXT PRIMARY KEY, `firma_registro` TEXT NOT NULL).
  - Table `registro_sucesos` (columns: `id` TEXT PRIMARY KEY, `fecha_hora` TEXT NOT NULL, `tipo_evento` TEXT NOT NULL, `detalles` TEXT NOT NULL, `estado_sincronizacion` TEXT NOT NULL DEFAULT 'PENDING').
- [x] 1.2 Modify `database/migrations/000001_init_erp_schemas.up.sql` to add:
  - Columns `firma_registro VARCHAR(255)`, `hash_anterior VARCHAR(255)`, `datos_encadenamiento TEXT` to the `invoice` table definition.
  - Table `registro_sucesos` (columns: `id` UUID PRIMARY KEY, `fecha_hora` TIMESTAMPTZ NOT NULL, `tipo_evento` VARCHAR(50) NOT NULL, `detalles` TEXT NOT NULL, `estado_sincronizacion` VARCHAR(20) NOT NULL DEFAULT 'PENDING').
- [x] 1.3 Create `tpv-client/src-tauri/src/signature.rs` defining:
  - Trait `Firmador` with a signature method:
    `fn firmar_registro(&self, prefijo: &str, secuencia: i64, total: f64, creado_en: &str, hash_anterior: Option<&str>) -> Result<String, String>;`
  - Struct `FirmaSimulada` implementing the `Firmador` trait. It should generate a SHA-256 hash using the `sha2` crate, hashing a formatted string: `format!("{}:{}:{}:{}:{}", prefijo, secuencia, total, creado_en, hash_anterior.unwrap_or(""))`.
  - Add `sha2 = "0.10"` dependency to `tpv-client/src-tauri/Cargo.toml` under `dependencies`.
  - Add `pub mod signature;` to `tpv-client/src-tauri/src/lib.rs` to expose the signature module.
- [x] 2.1 Update `tpv-client/src-tauri/src/db.rs` to include table schemas, updates, and query/insert functions for the last signature and event logs:
  - Update `init_db` to run the updated SQL schema (add columns `firma_registro`, `hash_anterior`, `datos_encadenamiento` to `offline_sales`; create tables `ultimo_registro_encadenado` and `registro_sucesos` if not exists).
  - Add struct `RegistroSuceso` with fields: `id`, `fecha_hora`, `tipo_evento`, `detalles`, `estado_sincronizacion`.
  - Implement functions: `insert_registro_suceso`, `get_pending_events`, `delete_synced_event`, `get_ultimo_registro_encadenado`, and `upsert_ultimo_registro_encadenado`.
  - Update tests to cover these new functions and table structures in memory.
- [x] 2.2 Modify `save_offline_sale` command in `tpv-client/src-tauri/src/lib.rs` to query/update last signature in SQLite transactions:
  - Query `get_ultimo_registro_encadenado(&tx)` inside the SQLite transaction.
  - If a signature exists, use it as `hash_anterior` (passed to `FirmaSimulada.firmar_registro`). If not, pass `None`.
  - Generate the signature, assign it to `sale.firma_registro` and `sale.hash_anterior`, and serialize metadata in `sale.datos_encadenamiento`.
  - Insert sale and items.
  - Call `upsert_ultimo_registro_encadenado(&tx, &sale.id, &current_signature)`.
  - Record the event in `registro_sucesos` (`tipo_evento = "ALTA_FACTURA"`, `detalles = format!("Factura emitida offline: {}", sale.invoice_number)`).
- [x] 2.3 Update cleanup logic and commands in `tpv-client/src-tauri/src/lib.rs` / `db.rs`:
  - Ensure that deleting synchronized sales (`delete_synced_sale`) only deletes the sale and items, preserving `ultimo_registro_encadenado` intact.
- [x] 3.1 Update `SyncSale` struct in `internal/sync/adapters/api_controller.go` to include chaining metadata fields.
- [x] 3.2 Add `SyncEvent` struct and `POST /api/v1/sync/events` route in `internal/sync/adapters/api_controller.go`:
  - Defined `SyncEvent` and `SyncEventsRequest` structs.
  - Registered route `/api/v1/sync/events` in `ServeHTTP` under POST method.
  - Implemented `HandleSyncEvents` with standard idempotency logic.
- [x] 3.3 Modify central DB persistence in `internal/sync/adapters/api_controller.go` to store chaining metadata and events log in Postgres and SQLite:
  - Updated `HandleSyncSales` insert queries to save `firma_registro`, `hash_anterior`, and `datos_encadenamiento` in SQLite and Postgres.
  - Updated `HandleSyncEvents` to save events to `registro_sucesos` table as `'SINCRONIZADO'`.
  - Updated Go unit tests in `api_controller_test.go` to cover database schemas and verify correct synchronization.
- [x] 4.1 Update `SyncSalePayload` struct in `tpv-client/src-tauri/src/sync.rs` to include the Spanish chaining metadata fields (`firma_registro`, `hash_anterior`, `datos_encadenamiento`) and populate them in `sync_pending_sales` from local database queries.
- [x] 4.2 Modify background sync loop in `sync.rs` to include `registro_sucesos` synchronization via POST to `/api/v1/sync/events`, query pending events, delete synced records on success, update the pending count in `count_pending`, and add `test_sync_events_flow` unit tests.

### Files Changed
| File | Action | What Was Done |
|------|--------|---------------|
| `database/migrations/sqlite_init.sql` | Modified | Added verifactu chaining fields to offline_sales and created sqlite tables `ultimo_registro_encadenado` and `registro_sucesos`. |
| `database/migrations/000001_init_erp_schemas.up.sql` | Modified | Added verifactu chaining fields to invoice table and created postgres table `registro_sucesos`. |
| `tpv-client/src-tauri/src/signature.rs` | Created | Defined the `Firmador` trait and the `FirmaSimulada` SHA-256 implementation. |
| `tpv-client/src-tauri/src/lib.rs` | Modified | Exposed signature module, updated `save_offline_sale` command with signature calculation, transactional database storage, upserting of the last signature, and audit event insertion. |
| `tpv-client/src-tauri/Cargo.toml` | Modified | Added `sha2 = "0.10"` to `[dependencies]`. |
| `tpv-client/src-tauri/src/db.rs` | Modified | Added columns `firma_registro`, `hash_anterior`, and `datos_encadenamiento` to `offline_sales`; created tables `ultimo_registro_encadenado` and `registro_sucesos` in `init_db`. Implemented `RegistroSuceso` and CRUD/query helper functions for events and last signature. Updated existing tests and added new unit tests. |
| `tpv-client/src-tauri/src/sync.rs` | Modified | Updated `SyncSalePayload` struct with Spanish chaining metadata fields, defined `SyncEventPayload` and `SyncEventsRequest` structs, implemented `sync_pending_events` POST sync, updated `count_pending` query, and added `test_sync_events_flow` test suite. |
| `internal/sync/adapters/api_controller.go` | Modified | Added verifactu chaining fields to `SyncSale`, registered `/api/v1/sync/events` route, updated `HandleSyncSales` insert queries, and implemented `HandleSyncEvents` handler. |
| `internal/sync/adapters/api_controller_test.go` | Modified | Updated test DB setup with chaining columns and `registro_sucesos` table, updated sync test to assert chaining fields persistence, and added `TestSalesSyncController_HandleSyncEvents` unit tests. |

### Deviations from Design
None — implementation matches design.

### Issues Found
None.

### Remaining Tasks
- [ ] 5.1 Add unit tests for `FirmaSimulada` SHA-256 hash generation logic in `signature.rs`.
- [ ] 5.2 Implement local integration test for SQLite transaction integrity during offline chaining.
- [ ] 5.3 Write integration test for background worker verifying sales and event sync to the Go mock endpoints.

### Workload / PR Boundary
- Mode: stacked PR slice
- Current work unit: Unit 4 (Background sync loop worker)
- Boundary: Starts from updating Go central sync backend schemas and API controller, ends with implementing event sync and unit testing both endpoints.
- Estimated review budget impact: ~120 changed lines.

### Status
11/14 tasks complete. Ready for next batch.
