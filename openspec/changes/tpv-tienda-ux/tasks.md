# Tasks: TPV Tienda POS (Cash Register)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~3500 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | 6 stacked PRs to main |
| Delivery strategy | auto-chain (stacked-to-main) |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Scope |
|------|-------|
| 1 | Rust data model + core commands |
| 2 | Rust void/payments/closure/print |
| 3 | Frontend core UI (POSProvider + search/cart) |
| 4 | Frontend payment/closure/receipt/settings |
| 5 | Rust sync layer (payload/void/conflict) |
| 6 | Go backend (void endpoint + sync structs) |

---

## Phase 1: Rust Backend — Data Model + Core Commands

- [x] 1.1 FTS5 DDL (`productos_fts`), content-sync triggers, `reindex_productos_fts()` fn in init_db
- [x] 1.2 ALTER TABLE migrations: `offline_sales` (+6 cols), `offline_sale_items` (+discount_percent), `productos` (+imagen_url)
- [x] 1.3 New tables: `caja_secuencia`, `offline_sale_payments`, `caja_aperturas`, `caja_movimientos`
- [x] 1.4 Extend `OfflineSale` struct: subtotal, tax_total, discount_total, status, void_reason, voided_at
- [x] 1.5 Extend `OfflineSaleItem` struct: discount_percent. Add `POSProduct` struct with imagen_url
- [x] 1.6 Commands: `search_products` (FTS5), `get_product_by_code`, `get_next_sequence` (atomic), `get_today_sales`, `reset_barcode_buffer`
- [x] 1.7 Helpers in db.rs: `get_sale_by_id`, `get_sale_items`
- [x] 1.8 Register new commands in lib.rs `run()`; remove old ambulante/CRM command handlers
- [x] 1.9 Tests: FTS5 search, get_today_sales date filter, sequence atomicity, extended fields roundtrip

## Phase 2: Rust Backend — Void, Payments, Closure, Printing

- [x] 2.1 `void_sale_impl`: status→VOIDED, stock restore per item, ANULACION in `registro_sucesos`. **CRI-4**: read voided sale's own `hash_anterior`, NOT `get_ultimo_registro_encadenado()`. Do NOT update `ultimo_registro_encadenado`
- [x] 2.2 Commands: `void_sale`, `registrar_apertura`, `registrar_ingreso_caja`, `registrar_retiro_caja`
- [x] 2.3 `registrar_cobro`: INSERT `offline_sale_payments`, chain-sign sale, mark COMPLETED
- [x] 2.4 `get_terminal_health` returning `TerminalHealth` (db_size, pending counts, terminal_id, version)
- [x] 2.5 `generate_receipt_pdf`: build `ReceiptData`, render PDF via genpdf/printpdf, return base64
- [x] 2.6 Remove obsolete structs: `Cliente`, `Direccion`, `Contacto`, `Nota`, `ClientDossier`, `OfflineCobro`
- [x] 2.7 Remove obsolete tables from init_db: `entidad_direcciones`, `entidad_contactos`, `entidad_notas`, `offline_cobros_recibidos`
- [x] 2.8 Tests: void chain integrity (3-sale chain, void middle), stock restoration, reject non-COMPLETED

## Phase 3: Frontend — POS Core UI

- [x] 3.1 Strip `App.tsx`: remove ambulante/entidades modes, dev simulator sections; render pure POS with `POSProvider`
- [x] 3.2 `POSProvider` (React Context): cart, search, payment, daily state per POSState shape
- [x] 3.3 `POSMainLayout`: 3-column (search/results, cart, quick buttons)
- [x] 3.4 `ProductSearchBar`: debounced (300ms), family filter dropdown, barcode wedge listener
- [x] 3.5 `ProductList` (replaces `ProductResultsGrid`): card grid with image, name, price, stock badge (green/yellow/red)
- [x] 3.6 `CartPanel` + `CartItemRow`: items list with images, IVA breakdown, qty +/-, discount %, collapsible, pay/clear
- [x] 3.7 Keep `SyncWarningBanner` as-is
- [x] 3.8 Remove old: `RouteSetup`, `ClientDossierView`, `ClientCollection`, `EntityManager`, `ShareDocumentModal`
- [x] 3.9 POS layout styles in `pos-ferreteria.css` (replaces `App.css` POS styles)
- [x] 3.10 Tests: cart add/remove/qty/discount math, search debounce, barcode add-to-cart

## Phase 4: Frontend — Payment, Closure, Receipt, Settings

- [x] 4.1 `PaymentModal`: full-screen overlay, cash/card/split, tender math, insufficient cash alert
- [x] 4.2 `CashPayment`: tender input, quick € buttons (5/10/20/50), Exact button, change display
- [x] 4.3 `CardPayment`: flat amount = total, no change, confirm only
- [x] 4.4 `SplitPayment`: cash+card balance, tax proration, must sum to total
- [x] 4.5 `DailyClosurePanel`: X report (no reset), Z report (reset + cash declaration), register open, petty cash in/out
- [x] 4.6 `ReceiptPreview`: on-screen preview, re-print last receipt with "(Reprint)" marker
- [x] 4.7 `TerminalSettings`: terminal ID config, printer config, health/connectivity display
- [x] 4.8 Tests: cash exact/change/split/insufficient payment flows, X/Z totals

## Phase 5: Sync Layer Updates (Rust)

- [x] 5.1 Extend `SyncSalePayload` Rust struct: `payments: Vec<SyncPayment>`, subtotal, tax_total, discount_total, status, void_reason, voided_at. **Show explicit struct definition in code**
- [x] 5.2 `sync_pending_voids`: query `registro_sucesos` ANULACION entries with sync_status=PENDING, POST to `/api/v1/sync/voids`
- [x] 5.3 Include void/event sync in `sync_all` loop alongside `sync_pending_sales`
- [x] 5.4 HTTP 409 handler in `sync_pending_sales`: keep sale PENDING, record CONFLICT event in `registro_sucesos`
- [x] 5.5 **Resolve clientes contradiction**: keep in `CatalogSyncResponse` (forward compat) but do NOT save to local DB
- [x] 5.6 Call `reindex_productos_fts()` after catalog sync upserts
- [x] 5.7 Remove `sync_pending_payments` (old ambulante payment sync)
- [x] 5.8 Tests: HTTP 409 keeps sale PENDING + CONFLICT event (mock HTTP), void payload roundtrip

## Phase 6: Go Backend Updates

- [x] 6.1 Extend `SyncSale` Go struct: `Subtotal`, `TaxTotal`, `DiscountTotal`, `Status`, `VoidReason`, `VoidedAt`, `Payments []SyncSalePayment`
- [x] 6.2 Add `SyncSalePayment` Go struct with `Method` and `Amount` fields (named `SyncSalePayment` to avoid conflict with existing `SyncPayment` in catalog_controller.go)
- [x] 6.3 Add `POST /api/v1/sync/voids` handler accepting `SyncVoidRequest` (sale_id, reason, firma_registro, hash_anterior)
- [x] 6.4 Register void route in `main.go`
- [x] 6.5 Tests: void handler idempotency, duplicate 409, SyncSale roundtrip with financial fields + payments

## Testing Strategy

Each PR must pass its layer's test suite before marking done:

| Phase | Gate |
|-------|------|
| 1–2, 5 | `cargo test` — all Rust unit + integration tests |
| 3–4 | `npm test` — all React component + integration tests |
| 6 | `go test ./internal/...` — Go unit + integration tests |
