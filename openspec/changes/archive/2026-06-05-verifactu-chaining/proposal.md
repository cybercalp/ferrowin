# Proposal: Veri*factu Chaining

## Intent
Implement Veri*factu-compliant hash chaining and cryptographic signature generation for offline sales in the Tauri desktop client (SQLite) and Go backend (PostgreSQL), ensuring local data integrity before sync.

## Scope

### In Scope
- Create local tables `registro_sucesos` (fields: `id`, `fecha_hora`, `tipo_evento`, `detalles`, `estado_sincronizacion`) and `ultimo_registro_encadenado` (fields: `id_factura`, `firma_registro`) in SQLite.
- Create central `registro_sucesos` table in PostgreSQL.
- Add `firma_registro`, `hash_anterior`, and `datos_encadenamiento` columns to offline sales (SQLite) and invoices (Postgres).
- Implement Rust mock signer (`FirmaSimulada`) implementing a clean trait interface to generate signatures.
- Implement hash chaining (if no previous invoice, leave `hash_anterior` empty/null).
- Include `registro_sucesos` table in the background sync worker loop to sync event logs to the Go backend.

### Out of Scope
- Integration with real PKCS#12 hardware security modules or production certificates.
- Automated transmission to the Spanish Tax Agency (AEAT) server.

## Capabilities

### New Capabilities
- `verifactu-trazabilidad-client`: Implements local hash chaining and mock signing in the Tauri client.

### Modified Capabilities
- `tpv-desktop-client`: Rust core handles signature generation and Spanish DB columns mapping.
- `tpv-background-sync`: Extends synchronization payload to include chaining metadata and syncs `registro_sucesos`.

## Approach
Create the `registro_sucesos` table in both local SQLite and central Postgres databases. Modify Tauri commands to generate a mock signature (`FirmaSimulada` implementing a `Signer` trait) on invoice creation.
Calculate `hash_anterior` based on the previous signature of the series stored in `ultimo_registro_encadenado`. For the first invoice in a series, leave `hash_anterior` as null (empty).
Persist metadata in Spanish (`firma_registro`, `hash_anterior`, `datos_encadenamiento`). Sync both sales and events log to the Go backend, inserting them into Postgres and verifying integrity.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `tpv-client/src-tauri/src/db.rs` | Modified | SQLite init schemas and DB helpers. |
| `tpv-client/src-tauri/src/lib.rs` | Modified | Tauri commands for sale persistence and chain logic. |
| `tpv-client/src-tauri/src/sync.rs` | Modified | Client background sync loop for sales and event logs. |
| `database/migrations/` | New/Modified | Database schemas (SQLite & Postgres migrations). |
| `internal/sync/adapters/` | Modified | Go API controller to ingest and store Spanish chaining columns. |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Signature Failure | Low | Atomic SQLite transactions rollback both sale and chain status. |
| Clock Drift | Low | Focus verification on hash chain sequence validation, not strict timestamp matches. |

## Rollback Plan
Revert database schema migrations. Roll back Tauri client code and Go API backend changes to their previous git commits. Local SQLite databases will discard the new Spanish columns and log tables upon client re-installation.

## Dependencies
- Go backend API update deployed before the client sync update.

## Success Criteria
- [ ] Invoices generate a deterministic mock signature.
- [ ] Subsequent invoices chain the hash of the preceding invoice's signature.
- [ ] First invoice in a series leaves `hash_anterior` empty (null).
- [ ] Background worker successfully syncs `registro_sucesos` to the Go backend.
