# Proposal: TPV Tienda UX (Cash Register POS)

## Intent

Strip all non-POS functionality (ambulante itinerant sales, CRM entity management) and build a real cash register interface for in-person retail sales. The existing offline SQLite, Veri*factu chaining, and background sync infrastructure is reused.

## Scope

### In Scope
- Rust Tauri commands: `search_products`, `get_product_by_code`, `get_today_sales`, `void_sale`, receipt printing
- React POS screens: main (search + cart), payment (cash/card/split), daily closure (X/Z), receipt preview, settings
- Data: `caja_secuencia` table, payment method + subtotal fields on `OfflineSale`, `offline_sale_payments` table
- Removal: ambulante/CRM Rust structs, commands, React components, and UI routes

### Out of Scope
- TPV Comercial (route sales management) — separate change
- Multi-terminal sync conflict resolution
- Cloud backup / remote access

## Capabilities

### New
- `pos-sale-flow`: product search (FTS5), cart, payment, receipt generation, Veri*factu chain
- `pos-daily-closure`: X report (mid-day), Z report (end-of-day), cash verification, sync
- `pos-void-sale`: void with reason, signature chain re-link

### Modified
- `tpv-desktop-client`: extend from offline-indicator-only to full POS UI
- `offline-pos-stock`: extend with `search_products` query over local SQLite FTS5
- `tpv-background-sync`: extend with closure sync and void sync

## Approach

Rewrite Tauri Rust backend — remove ambulante/CRM, add POS query and mutation commands. Rewrite React frontend to a dedicated cash register interface. Reuse existing SQLite layer, sync worker loop, and Veri*factu chaining. Add ESC/POS thermal printing via Tauri plugin; fallback to PDF + OS print dialog. Maintain a local sequence counter (`caja_secuencia`) for ticket numbering.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `tpv-client/src-tauri/src/commands/` | Modified | Add POS commands, remove old |
| `tpv-client/src-tauri/src/models/` | Modified | Remove ambulante structs, extend sale |
| `tpv-client/src-tauri/src/db.rs` | Modified | Add `caja_secuencia`, `offline_sale_payments` |
| `tpv-client/src/` | Modified | Rewrite React to POS screens |
| `internal/api/sync.go` | Modified | Add closures and voids sync endpoints |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| ESC/POS WebUSB unsupported on some terminals | Low | Fallback to PDF + OS print dialog |
| Split payment tax allocation errors | Low | Prorate taxes proportionally per line |
| Offline search perf with large catalogs | Low | FTS5 index on code + nombre |

## Rollback

Git revert of the change branch. No destructive schema migrations — old tables remain, new tables can be dropped. Old ambulante code preserved in a tagged commit for future reuse if needed.

## Dependencies

- Tauri v2 print plugin or WebUSB for ESC/POS
- Go backend sync endpoint extensions

## Success Criteria

- [ ] Full sale flow works offline: search → add → quantity → pay → receipt → chain
- [ ] Daily closure generates correct X/Z from local sales
- [ ] Void updates Veri*factu chain without breaking linkage
- [ ] Sync pushes sales, closures, and voids on reconnection
- [ ] Zero remaining ambulante/CRM code references in Tauri backend
