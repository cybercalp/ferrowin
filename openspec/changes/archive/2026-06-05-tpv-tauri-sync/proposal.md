# Proposal: TPV Tauri Synchronization

## Intent
Enable offline-first point-of-sale (TPV) operations in a Tauri desktop client, allowing cashiers to continue billing and perform box closures (arqueo) without internet connectivity, and synchronizing data reliably with the Go backend once connection is restored.

## Scope

### In Scope
- SQLite-backed offline storage for sales and box closures (arqueo).
- Eventual offline stock queries (last known downloaded data) with fallback warning.
- UI warning when offline without blocking active sales.
- Automatic background synchronization when connection is restored.
- Safe local database cleanup of synchronized records (exactly-once sync validation).

### Out of Scope
- Real-time inventory adjustments/queries when offline.
- Conflict resolution for simultaneous box closures on multiple terminals.

## Capabilities

### New Capabilities
- `tpv-desktop-client`: Handles offline-first GUI shell, sqlite-based sales storage, box closures (arqueo) offline, and online/offline warnings.
- `tpv-background-sync`: Automatically detects internet restoration, handles reliable synchronization with Go backend, and performs clean-up of synchronized records.

### Modified Capabilities
- `offline-pos-stock`: Modified to support eventual consistency (last known local stock) when offline, switching to real-time central stock queries only when online.

## Approach
Implement a Tauri desktop client with a local SQLite database for offline storage. The frontend detects offline state, shows a warning banner, and falls back to local SQLite queries (eventual stock) and inserts (sales, box closures). A background loop continuously retries synchronization to the Go backend. On successful sync, SQLite records are purged to conserve disk space.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `src-tauri` | New | Tauri application wrapper and background sync workers |
| `database` | Modified | Add local schema generation/migration scripts if needed |
| `internal/api` | Modified | Endpoints to ingest offline closures and batch sales |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Data loss on app crash | Low | Write sales and box closures to SQLite immediately upon creation |
| Double synchronization | Med | Use unique UUIDs for transactions and implement idempotent backend ingest |

## Rollback Plan
Since this introduces Tauri desktop integration, rollback involves rolling back the Tauri client version or disabling the sync flags on the Go backend API to reject/ignore desktop sync requests, falling back to existing web-only POS.

## Dependencies
- Tauri v1/v2 framework installed.
- Go backend API support for batch sale uploads and arqueo synchronization.

## Success Criteria
- [ ] TPV continues billing and allows box closure when offline without blocking cashiers.
- [ ] Offline warnings are displayed when connection is lost, and cleared on restore.
- [ ] Stock queries show last-known stock when offline, and real-time stock when online.
- [ ] All offline sales and closures sync to Go backend upon reconnection.
- [ ] Synchronized SQLite records are cleaned up post-sync.
