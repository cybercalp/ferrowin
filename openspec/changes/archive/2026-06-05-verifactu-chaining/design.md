# Design: VeriFactu Chaining

## Technical Approach
This design introduces billing record chaining for the TPV client (local SQLite) and Go backend (PostgreSQL), complying with Spanish VeriFactu regulations. Every invoice must cryptographically chain to the previous one using its signature, creating an unalterable sequence.

## Architecture Decisions

| Option | Tradeoff | Decision |
|---|---|---|
| In-memory vs. DB persistence for last signature | In-memory is fast but loses state on app restart, leading to chaining breaks. DB persistence is resilient. | Persist the last signature in SQLite helper table `ultimo_registro_encadenado` to survive POS restarts. |
| Event logging structure | Log as inline unstructured text vs. dedicated event table. | Create a structured `registro_sucesos` table on both local SQLite and central PostgreSQL for audit-ready event logging and sync. |
| Cryptographic Library | External dependency vs. simple standard hashing. | Use standard `sha2` crate in Rust for SHA-256 calculation to keep dependency footprint minimal. |

## Data Flow
The sequence of saving a sale locally and syncing it:
```
[User checkout] 
       │
       ▼
[save_offline_sale] ──(query)──► [ultimo_registro_encadenado]
       │ (generates new hash with hash_anterior)
       ▼
[Insert offline_sales & update ultimo_registro_encadenado] (SQLite Tx)
       │
       ▼
[sync_pending_sales] (Background loop) ──(POST)──► [POST /api/v1/sync/sales]
       │                                                    │
       ▼ (if 2xx)                                           ▼
[delete_synced_sale] (leaves last signature intact)    [Insert invoice (Postgres)]
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `tpv-client/src-tauri/src/db.rs` | Modify | Update SQLite initialization schema with new chaining columns and local tables. Add query and update functions for the new tables. |
| `tpv-client/src-tauri/src/signature.rs` | Create | Define `Firmador` trait and its mock implementation `FirmaSimulada` producing SHA-256 hashes. |
| `tpv-client/src-tauri/src/lib.rs` | Modify | Update `save_offline_sale` command to query/update last signature and generate the cryptographic signature within the transaction context. |
| `tpv-client/src-tauri/src/sync.rs` | Modify | Update `SyncSalePayload` to include chaining fields, update background loop to sync pending events from `registro_sucesos` via `POST /api/v1/sync/events`. |
| `internal/sync/adapters/api_controller.go` | Modify | Update API payloads and route registry for `POST /api/v1/sync/sales` and add `POST /api/v1/sync/events`. |
| `database/migrations/sqlite_init.sql` | Modify | Document schema upgrades for local POS tables (`offline_sales`, `registro_sucesos`, `ultimo_registro_encadenado`). |
| `database/migrations/000001_init_erp_schemas.up.sql` | Modify | Document schema upgrades for Postgres (`invoice`, `registro_sucesos`). |

## Interfaces / Contracts

### Rust: Cryptographic Trait (`tpv-client/src-tauri/src/signature.rs`)
```rust
pub trait Firmador {
    fn firmar_registro(
        &self,
        prefix: &str,
        sequence: i64,
        total: f64,
        created_at: &str,
        hash_anterior: Option<&str>,
    ) -> Result<String, String>;
}
```

### Go API contract updates (`internal/sync/adapters/api_controller.go`)
```go
type SyncSale struct {
	ID                  string     `json:"id"`
	InvoiceNumber       string     `json:"invoice_number"`
	SequenceNumber      int        `json:"sequence_number"`
	CreatedAt           string     `json:"created_at"`
	Total               float64    `json:"total"`
	FirmaRegistro       string     `json:"firma_registro"`
	HashAnterior        string     `json:"hash_anterior"`
	DatosEncadenamiento string     `json:"datos_encadenamiento"`
	Items               []SyncItem `json:"items"`
}

type SyncEvent struct {
	ID                  string `json:"id"`
	FechaHora           string `json:"fecha_hora"`
	TipoEvento          string `json:"tipo_evento"`
	Detalles            string `json:"detalles"`
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `FirmaSimulada` hash generation | Assert expected SHA-256 output format for mock chaining inputs. |
| Integration | SQLite transaction & chaining | Insert sale, check `ultimo_registro_encadenado` gets updated, verify next sale's `hash_anterior` matches previous `firma_registro`. |
| Integration | Event sync worker loop | Mock backend endpoint `/api/v1/sync/events`, verify HTTP POST and subsequent local SQLite deletion of synced events. |

## Migration / Rollout
- Schema migrations run automatically on TPV start via SQLite `db::init_db`.
- DB columns are nullable or have defaults to avoid breaking existing offline/online tables.

## Open Questions
- None.
