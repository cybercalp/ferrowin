# Design: TPV Tienda POS — Ferretería Workflow

## Technical Approach

This design implements the **ferretería (hardware store) POS workflow** — a specialization of the base POS optimized for high-throughput counter sales using USB barcode-scanner input (keyboard wedge), family-based browsing, and non-food units of measure. The frontend is a clean-slate React implementation at `tpv-client/src/pos/`, reusing the existing Rust Tauri backend's SQLite layer, sync loop, and Veri\*factu chaining.

**Key decision:** We are NOT adding new Rust commands. The backend already has all needed IPC surface:
- `search_products`, `get_product_by_code`, `get_products_by_family` (product lookup)
- `registrar_cobro_pago` (payment recording)
- `create_sale_with_line_items` (sale persistence with signature chain)
- `get_today_sales`, `get_daily_summary` (daily operations)
- `open_drawer`, `generate_receipt_pdf` (peripherals)
- `get_terminal_health` (monitoring)

All changes are scoped to the React frontend at `tpv-client/src/pos/`. The Go backend, Rust middleware, and SQLite schema remain unchanged from the base POS design.

## Architecture Decisions

### Decision: FTS5 for product search with sync triggers

**Choice**: SQLite FTS5 virtual table on `productos(codigo, nombre)` with content-sync triggers
**Alternatives**: LIKE queries, rust fuzzy matchers
**Rationale**: FTS5 gives sub-100ms partial matches on 10k+ catalogs without adding dependencies. Content-sync triggers on `productos` keep the index consistent automatically. A `reindex_productos_fts()` function is called after each catalog sync to rebuild from scratch (handles bulk upserts).

### Decision: Atomic sequence via dedicated SQLite table

**Choice**: `caja_secuencia(prefix TEXT PK, next_val INTEGER)` with `INSERT ... ON CONFLICT DO UPDATE SET next_val = next_val + 1 RETURNING next_val`
**Alternatives**: MAX() query, UUID-only tickets
**Rationale**: Per-terminal sequence counters need atomic increment within SQLite transactions. A dedicated row per prefix avoids table scans and works offline.

### Decision: Void does NOT re-link the signature chain

**Choice**: Voids update the existing sale row to `status='VOIDED'` and record an `ANULACION` event in `registro_sucesos` with chain metadata. No new `offline_sales` row is created. `ultimo_registro_encadenado` is NOT updated.
**Alternatives**: Insert a new offline_sales row with status VOIDED; separate void table
**Rationale**: The signature chain must remain linear — voiding a sale does NOT change the hash pointers of subsequent valid sales. The void entry's `hash_anterior` points to the preceding *valid* sale's `firma_registro`, but `ultimo_registro_encadenado` stays pointing to the last *completed* sale so TPV-0043 retains its original `hash_anterior`. The `ANULACION` event provides full audit trail without breaking chain integrity.

### Decision: PDF receipt printing as default

**Choice**: PDF generation via `genpdf` or `printpdf` Rust crate, opened via `tauri-plugin-dialog` for the OS print dialog. ESC/POS thermal printer support deferred to a follow-up via Tauri plugin or WebUSB.
**Alternatives**: WebUSB ESC/POS, Tauri print plugin
**Rationale**: PDF + system print dialog works on ALL platforms without thermal printer hardware. ESC/POS requires specific drivers and hardware testing not available in this phase. The component interface abstracts the printer so a thermal backend can be swapped in without changing receipt data generation.

## Data Flow - Ferreteria Workflow

```
[USB Wedge Scanner] --keystrokes--> [BarcodeInput] (permanent focus)
         |                                    |
         |  Enter pressed                      |  Typed query / family click
         v                                    v
[Tauri: get_product_by_code]       [Tauri: search_products / get_products_by_family]
         |                                    |
         +------------+-----------------------+
                      v
              [Cart: add item qty 1]
                      |
                      v
          +-----------------------+
          |  [CustomerSelector]   |--> auto-discount on displayed prices
          |  (always in header)   |--> -X% discount badge in cart + receipt
          +-----------------------+
                      |
                      v
         +------------------------------+
         |  [CartPanel]                 |
         |  * items with qty            |
         |  * UoM-aware input (ud: int, |
         |    m/kg: decimal)            |
         |  * line totals               |
         |  * IVA breakdown             |
         +------------------------------+
                      |
              [CobrarButton] (always visible)
                      |
                      v
              [PaymentModal]
          +-------+--------+-------+
          v       v        v       v
      [Cash]  [Card]  [Bizum]  [Split]
          +-------+--------+-------+
                  |        |
            [DocTypeToggle]
         Ticket <--------> Factura
  (default: Ticket if no customer,
   Factura if customer present - overridable)
                  |
                  v
    [Tauri: create_sale_with_line_items]
                  |
         +--------+--------+
         v                  v
   [SQLite offline_sales]  [Signature chain]
                  |
            [Sync loop] --POST--> [Go /api/v1/sync/sales]

  Separate flows (same as base POS design):
    [TodaySalesList] --void--> [void_sale IPC]                  (unchanged)
    [open_drawer]    --open--> [open_drawer IPC]                (unchanged)
    [DailyClosure]   --X/Z-->  [get_daily_summary IPC]          (unchanged)
```
## Schema Changes

### New tables in `init_db`

```sql
-- FTS5 virtual table for product search (CRI-1)
CREATE VIRTUAL TABLE IF NOT EXISTS productos_fts USING fts5(
    codigo, nombre,
    content=productos,
    content_rowid=rowid
);

-- Triggers to keep FTS index in sync with productos (CRI-1)
CREATE TRIGGER IF NOT EXISTS productos_ai AFTER INSERT ON productos BEGIN
    INSERT INTO productos_fts(rowid, codigo, nombre) VALUES (new.rowid, new.codigo, new.nombre);
END;

CREATE TRIGGER IF NOT EXISTS productos_ad AFTER DELETE ON productos BEGIN
    INSERT INTO productos_fts(productos_fts, rowid, codigo, nombre) VALUES('delete', old.rowid, old.codigo, old.nombre);
END;

CREATE TRIGGER IF NOT EXISTS productos_au AFTER UPDATE ON productos BEGIN
    INSERT INTO productos_fts(productos_fts, rowid, codigo, nombre) VALUES('delete', old.rowid, old.codigo, old.nombre);
    INSERT INTO productos_fts(rowid, codigo, nombre) VALUES (new.rowid, new.codigo, new.nombre);
END;

-- Per-terminal sequence counter
CREATE TABLE IF NOT EXISTS caja_secuencia (
    prefix TEXT PRIMARY KEY,
    next_val INTEGER NOT NULL DEFAULT 1
);

-- Payment methods per sale
CREATE TABLE IF NOT EXISTS offline_sale_payments (
    id TEXT PRIMARY KEY,
    sale_id TEXT NOT NULL REFERENCES offline_sales(id) ON DELETE CASCADE,
    metodo_pago TEXT NOT NULL,  -- 'cash' | 'card'
    amount REAL NOT NULL
);

-- Cash register openings (WRN-2)
CREATE TABLE IF NOT EXISTS caja_aperturas (
    id TEXT PRIMARY KEY,
    terminal_id TEXT NOT NULL,
    opened_at TEXT NOT NULL,
    initial_cash REAL NOT NULL DEFAULT 0.0,
    sync_status TEXT NOT NULL DEFAULT 'PENDING'
);

-- Petty cash movements (WRN-3)
CREATE TABLE IF NOT EXISTS caja_movimientos (
    id TEXT PRIMARY KEY,
    terminal_id TEXT NOT NULL,
    tipo TEXT NOT NULL,            -- 'INGRESO' | 'RETIRO'
    concepto TEXT NOT NULL,
    amount REAL NOT NULL,
    created_at TEXT NOT NULL,
    sync_status TEXT NOT NULL DEFAULT 'PENDING',
    idempotency_key TEXT UNIQUE NOT NULL
);
```

### ALTER TABLE migrations in `init_db`

```sql
-- CRI-3: Extend offline_sales with financial & lifecycle fields
ALTER TABLE offline_sales ADD COLUMN subtotal REAL NOT NULL DEFAULT 0;
ALTER TABLE offline_sales ADD COLUMN tax_total REAL NOT NULL DEFAULT 0;
ALTER TABLE offline_sales ADD COLUMN discount_total REAL NOT NULL DEFAULT 0;
ALTER TABLE offline_sales ADD COLUMN status TEXT NOT NULL DEFAULT 'COMPLETED';
ALTER TABLE offline_sales ADD COLUMN void_reason TEXT;
ALTER TABLE offline_sales ADD COLUMN voided_at TEXT;

-- WRN-5: Add discount_percent to sale items
ALTER TABLE offline_sale_items ADD COLUMN discount_percent REAL NOT NULL DEFAULT 0.0;

-- WRN-6: Add imagen_url to productos
ALTER TABLE productos ADD COLUMN imagen_url TEXT;
```

### Reindex function (CRI-1)

```rust
pub fn reindex_productos_fts(conn: &Connection) -> Result<()> {
    conn.execute("DELETE FROM productos_fts", [])?;
    conn.execute(
        "INSERT INTO productos_fts(rowid, codigo, nombre)
         SELECT rowid, codigo, nombre FROM productos WHERE activo = 1",
        [],
    )?;
    Ok(())
}
```

## Rust Struct Changes

### OfflineSale (extended — CRI-3)

```rust
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OfflineSale {
    pub id: String,
    pub terminal_id: String,
    pub customer_id: Option<String>,
    pub total: f64,
    pub subtotal: f64,        // NEW
    pub tax_total: f64,       // NEW
    pub discount_total: f64,  // NEW
    pub created_at: String,
    pub sync_status: String,
    pub idempotency_key: String,
    pub invoice_number: String,
    pub sequence_number: i64,
    pub firma_registro: Option<String>,
    pub hash_anterior: Option<String>,
    pub datos_encadenamiento: Option<String>,
    pub status: String,             // NEW — "COMPLETED" | "VOIDED"
    pub void_reason: Option<String>, // NEW
    pub voided_at: Option<String>,   // NEW
}
```

### OfflineSaleItem (extended — WRN-5)

```rust
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OfflineSaleItem {
    pub id: String,
    pub offline_sale_id: String,
    pub item_id: String,
    pub quantity: f64,
    pub unit_price: f64,
    pub discount_percent: f64,  // NEW — 0.0–100.0
}
```

### POSProduct (extended — ferretería)

```rust
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct POSProduct {
    pub id: String,
    pub codigo: String,
    pub nombre: String,
    pub precio_venta: f64,
    pub tipo_iva_id: String,
    pub iva_porcentaje: f64,
    pub familia_id: Option<String>,
    pub familia_nombre: Option<String>,
    pub unidad_medida: String,           // "ud" | "m" | "kg" — controls qty input step/decimals
    pub stock: f64,
    pub imagen_url: Option<String>,
    pub activo: bool,
}
```

## All Tauri Commands

```rust
// Product search via FTS5 (REQ-POS-001/002)
#[tauri::command]
fn search_products(query: String, state: State<'_, DbState>) -> Result<Vec<db::POSProduct>, String>
// JOIN productos_fts ON rowid = p.rowid WHERE productos_fts MATCH ?

// Exact barcode lookup (REQ-POS-003)
#[tauri::command]
fn get_product_by_code(code: String, state: State<'_, DbState>) -> Result<Option<db::POSProduct>, String>
// SELECT ... WHERE codigo = ? LIMIT 1

// Today's sales with VOIDED included (REQ-POS-037)
#[tauri::command]
fn get_today_sales(state: State<'_, DbState>) -> Result<Vec<db::OfflineSale>, String>
// SELECT ... FROM offline_sales WHERE date(created_at) = date('now') ORDER BY sequence_number

// Void a sale — SEPARATE impl, NOT reusing save_offline_sale_impl (CRI-2, CRI-4)
#[tauri::command]
fn void_sale(sale_id: String, reason: String, state: State<'_, DbState>) -> Result<(), String>

// Atomic sequence counter
#[tauri::command]
fn get_next_sequence(prefix: String, state: State<'_, DbState>) -> Result<i64, String>
// INSERT ... ON CONFLICT DO UPDATE ... RETURNING next_val

// Barcode scanner buffer clear (REQ-POS-006)
#[tauri::command]
fn reset_barcode_buffer(state: State<'_, DbState>) -> Result<(), String>

// Register opening (WRN-2)
#[tauri::command]
fn registrar_apertura(amount: f64, terminal_id: String, state: State<'_, DbState>) -> Result<(), String>

// Petty cash in/out (WRN-3)
#[tauri::command]
fn registrar_ingreso_caja(concepto: String, amount: f64, state: State<'_, DbState>) -> Result<(), String>

#[tauri::command]
fn registrar_retiro_caja(concepto: String, amount: f64, state: State<'_, DbState>) -> Result<(), String>
```

## void_sale_impl — detailed design (CRI-2, CRI-4)

```rust
fn void_sale_impl(sale_id: &str, reason: &str, db_path: &str) -> Result<(), String> {
    let conn = rusqlite::Connection::open(db_path).map_err(...)?;
    let tx = conn.transaction().map_err(...)?;

    // 1. Verify sale exists and is COMPLETED
    let sale: db::OfflineSale = db::get_sale_by_id(&tx, sale_id)?;
    if sale.status != "COMPLETED" {
        return Err("Sale is not in COMPLETED status".into());
    }

    // 2. Get items to restore stock
    let items = db::get_sale_items(&tx, sale_id)?;

    // 3. Update sale to VOIDED
    tx.execute(
        "UPDATE offline_sales SET status='VOIDED', void_reason=?1, voided_at=?2 WHERE id=?3",
        params![reason, chrono::Utc::now().to_rfc3339(), sale_id],
    )?;

    // 4. Restore stock for each item
    for item in &items {
        tx.execute(
            "UPDATE stock_cache SET stock = stock + ?1 WHERE item_id = ?2",
            params![item.quantity, item.item_id],
        )?;
    }

    // 5. Record ANULACION in registro_sucesos with chain metadata
    //    hash_anterior = preceding valid sale's firma_registro (from ultimo_registro_encadenado)
    let preceding_signature = db::get_ultimo_registro_encadenado(&tx)?;
    let firmador = FirmaSimulada;
    let void_hash = firmador.firmar_registro(
        "ANULACION", 0, sale.total, &sale.created_at,
        preceding_signature.as_deref(),
    )?;

    let detalles = serde_json::json!({
        "sale_id": sale_id,
        "invoice_number": sale.invoice_number,
        "reason": reason,
        "firma_registro": void_hash,
        "hash_anterior": preceding_signature,
    }).to_string();

    let event_id = uuid::Uuid::new_v4().to_string();
    db::insert_registro_suceso(&tx, &event_id, "ANULACION", &detalles)?;

    // 6. Do NOT update ultimo_registro_encadenado (CRI-4)
    //    The chain head remains the last COMPLETED sale's signature

    tx.commit().map_err(...)?;
    Ok(())
}
```

**Chain behavior (REQ-POS-063 scenario):**
- TPV-0041 `firma_registro` = `SIG-0041`, TPV-0042 `firma_registro` = `SIG-0042`, TPV-0043 `firma_registro` = `SIG-0043`
- `ultimo_registro_encadenado` = `SIG-0043`
- After voiding TPV-0042:
  - Void entry `hash_anterior` = `SIG-0041` (preceding VALID sale, NOT the voided `SIG-0042`)
  - `ultimo_registro_encadenado` = `SIG-0043` (UNCHANGED)
  - TPV-0043 retains its original `hash_anterior = SIG-0042`
  - Chain integrity: 0041 → 0042 → 0043 still valid as completed sales; void is side-channel audit trail

## Conflict Detection (WRN-7)

In `sync.rs` `sync_pending_sales()`, when the backend responds with HTTP 409:

```rust
Ok(r) if r.status() == 409 => {
    // Conflict — don't delete, record CONFLICT event
    let event_id = uuid::Uuid::new_v4().to_string();
    let details = format!("SYNC_CONFLICT: sale {} (seq {}) rejected with 409",
        sale.id, sale.sequence_number);
    db::insert_registro_suceso(&conn, &event_id, "CONFLICT", &details)?;
}
// ... existing 200 handling unchanged
```

This satisfies REQ-POS-045: the sale stays `PENDING` locally and a `CONFLICT` event is recorded.

## Receipt Printing Architecture (WRN-1)

```
[Payment Complete]
        │
        ▼
[ReceiptPreview — React component]
        │ generate_receipt_pdf IPC
        ▼
[Rust] generate_receipt_pdf(sale_id, is_reprint) -> Vec<u8>
  │
  ├── genpdf::Document::new() -> build_receipt_content(sale)
  │
  └── return PDF bytes as base64
        │
        ▼
[Frontend] decode base64 -> Blob -> window.open() / tauri-plugin-dialog save
```

The receipt component interface is abstracted so a future ESC/POS thermal backend can be wired in without changing receipt data generation. The `ReceiptData` struct (from the spec) is the shared contract.

## Sync Payload Extension (Go + Rust)

```go
// SyncSale gets new financial fields and payments
type SyncSale struct {
    // ...existing fields...
    Payments      []SyncPayment `json:"payments"`
    Subtotal      float64       `json:"subtotal"`
    TaxTotal      float64       `json:"tax_total"`
    DiscountTotal float64       `json:"discount_total"`
    Status        string        `json:"status"`        // "COMPLETED" | "VOIDED"
    VoidReason    *string       `json:"void_reason,omitempty"`
    VoidedAt      *string       `json:"voided_at,omitempty"`
}

// Void sync — separate endpoint
type SyncVoidRequest struct {
    SaleID      string `json:"sale_id"`
    Reason      string `json:"reason"`
    FirmaRegistro string `json:"firma_registro"`
    HashAnterior  string `json:"hash_anterior"`
}
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| (Backend files unchanged from base design — all Tauri IPC, Rust structs, Go sync endpoints are already implemented) |
| `tpv-client/src/pos/pages/POSPage.tsx` | Create | Root POS page — assembles header, family nav, product panel, cart, action bar |
| `tpv-client/src/pos/components/POSHeader.tsx` | Create | Header bar container: BarcodeInput + CustomerSelector |
| `tpv-client/src/pos/components/BarcodeInput.tsx` | Create | Permanent-focus input capturing USB wedge scanner on Enter |
| `tpv-client/src/pos/components/CustomerSelector.tsx` | Create | Dropdown with search by name/NIF, always visible in header |
| `tpv-client/src/pos/components/DocumentTypeToggle.tsx` | Create | Ticket/Factura radio toggle at payment stage |
| `tpv-client/src/pos/components/FamilyNavBar.tsx` | Create | Horizontal scrollable tab bar (TODO + family names) |
| `tpv-client/src/pos/components/ProductPanel.tsx` | Create | Panel wrapping ProductList with family-filtered IPC data |
| `tpv-client/src/pos/components/ProductList.tsx` | Create | Data table: codigo, nombre, unidad, precio, stock, add button |
| `tpv-client/src/pos/components/CartPanel.tsx` | Modify | UoM-aware qty input, customer discount, deduplication |
| `tpv-client/src/pos/components/CartItemRow.tsx` | Modify | Decimal qty for m/kg, integer for ud, show unidad column |
| `tpv-client/src/pos/components/CartTotals.tsx` | Create | Subtotal, discounts, IVA breakdown, total |
| `tpv-client/src/pos/components/CustomerDiscountBadge.tsx` | Create | Shows -X% badge when customer with discount is selected |
| `tpv-client/src/pos/components/ActionBar.tsx` | Create | Bottom bar with totals + CobrarButton |
| `tpv-client/src/pos/components/CobrarButton.tsx` | Create | Triggers payment modal, shows total, disabled when cart empty |
| `tpv-client/src/pos/components/PaymentModal.tsx` | Modify | Add Bizum, doc type toggle, customer info display |
| `tpv-client/src/pos/components/CashPayment.tsx` | Modify | Add quick-amounts 5/10/20/50, exact, custom tender input |
| `tpv-client/src/pos/components/CardPayment.tsx` | Keep | Flat amount — no changes needed |
| `tpv-client/src/pos/components/BizumPayment.tsx` | Create | Flat amount payment via Bizum (method='bizum') |
| `tpv-client/src/pos/components/SplitPayment.tsx` | Modify | Support cash+card+bizum splits, enforce sum == total |
| `tpv-client/src/pos/components/ChangeCalculation.tsx` | Create | Show change due for cash payments |
| `tpv-client/src/pos/components/CompleteSaleButton.tsx` | Create | Final confirm, triggers create_sale_with_line_items IPC |
| `tpv-client/src/pos/components/ReceiptPreview.tsx` | Keep | No changes needed |
| `tpv-client/src/pos/components/DailyClosurePanel.tsx` | Keep | No changes needed |
| `tpv-client/src/pos/components/TerminalSettings.tsx` | Keep | No changes needed |
| `tpv-client/src/pos/components/PosContext.tsx` | Modify | Update state to FerreteriaState with customer, UoM, families |
| `tpv-client/src/pos/components/types.ts` | Modify | Add unidad_medida, Bizum PaymentMethod, CustomerInfo |
| `tpv-client/src/App.tsx` | Modify | Render POSPage, remove old layout references |
| `tpv-client/src/App.css` | Modify | Add POS table styles, family nav, header layout |
| `tpv-client/src/theme/ThemeProvider.tsx` | Keep | Theme context reused as-is |
| `tpv-client/src/components/SyncWarningBanner.tsx` | Keep | Works as-is, imported by POSPage |

## Component Architecture — Ferretería Workflow

The POS frontend follows a **container-presentational** pattern. State lives in a single React Context (`PosContext`) at the `POSPage` level. All components below receive data and callbacks via context or props.

### Component Tree

```
POSPage (tpv-client/src/pos/pages/POSPage.tsx)
├── POSHeader
│   ├── BarcodeInput (permanent focus, captures scanner input)
│   └── CustomerSelector (always visible, dropdown with search)
├── FamilyNavBar
│   └── (horizontal tabs: TODO, CERROJO, ELECTRICO, GAS, ...)
├── ProductPanel
│   └── ProductList (table: codigo, nombre, unidad, precio, stock, add)
├── CartPanel
│   ├── CartItemRow (name, qty input, unit, line total, remove)
│   ├── CartTotals (subtotal, discounts, IVA, total)
│   └── CustomerDiscountBadge (visible when customer selected)
├── ActionBar
│   └── CobrarButton (always visible, shows total)
└── PaymentModal (overlay)
    ├── AmountDisplay (total with discount, customer name)
    ├── CashPayment (quick amounts 5/10/20/50 + exact + custom)
    ├── CardPayment (flat amount)
    ├── BizumPayment (flat amount)
    ├── SplitPayment (cash+card+bizum, sum must equal total)
    ├── ChangeCalculation (change due display)
    └── CompleteSaleButton (triggers Tauri IPC)
```

### State Management (FerreteriaState)

```typescript
interface FerreteriaState {
  // Barcode & search
  barcodeBuffer: string;
  searchResults: POSProduct[];
  selectedFamily: string | null;     // null = "show all families"

  // Cart
  cart: Cart;

  // Customer
  customer: CustomerInfo | null;
  documentType: "ticket" | "factura";

  // Payment
  payment: PaymentSession;

  // UI flags
  isPaymentOpen: boolean;
  isDailyClosureOpen: boolean;

  // Terminal
  terminalId: string;
  registerOpen: boolean;
  todaySales: CompletedSale[];
  syncStatus: { online: boolean; pending: number };
}

interface Cart {
  items: CartItem[];
  totalUnits: number;
  subtotal: number;
  totalDiscount: number;
  taxTotal: number;
  totalAmount: number;
  ivaBreakdown: IVABreakdownEntry[];
}

interface CartItem {
  product: POSProduct;
  quantity: number;
  unidadMedida: "ud" | "m" | "kg";
  unitPrice: number;            // captured at add time (already discounted)
  discountPercent: number;      // 0-100 per line
  lineTotal: number;            // (unitPrice * qty) * (1 - discountPercent/100)
  ivaAmount: number;
}

interface PaymentSession {
  methods: POSPayment[];
  totalTendered: number;
  change: number;
  error: string | null;
}

interface POSPayment {
  method: "cash" | "card" | "bizum";
  amount: number;
}
```

## Ferretería Workflow State Machine

The POS UI operates as a deterministic state machine optimized for the high-throughput counter workflow. State transitions are driven by user actions (scan, browse, add, pay) and system responses (IPC success/failure).

### States

| State | Entry | Exit Conditions |
|-------|-------|-----------------|
| `IDLE` | App starts, sale completed, cart cleared | First product added via scan or manual |
| `BROWSING` | User taps family tab or types search query | Product added to cart becomes CART_ACTIVE |
| `CART_ACTIVE` | First item added to cart; items >= 1 | Cobrar pressed becomes PAYMENT; last item removed becomes IDLE |
| `PAYMENT` | Cobrar button pressed, modal opens | Confirm becomes COMPLETING; Cancel becomes CART_ACTIVE |
| `COMPLETING` | User confirms payment, IPC in flight | IPC success becomes DONE; IPC error becomes PAYMENT with error |
| `DONE` | Sale persisted in SQLite | Receipt dismissed or printed becomes IDLE |

### Orthogonal State Variables (independent of main state)

- **customer**: `CustomerInfo | null` -- selected customer; affects discount calculation and document type default
- **documentType**: `"ticket" | "factura"` -- defaults based on customer presence, manually overridable at payment
- **registerOpen**: `boolean` -- blocks Cobrar button when false

### Transition Rules

1. **Barcode scan**: Always transitions to CART_ACTIVE (or stays in CART_ACTIVE). If product not found, show error toast and stay in current state.
2. **Cobrar button**: Only enabled when CART_ACTIVE AND total > 0 AND registerOpen = true.
3. **Customer change**: Orthogonal -- recalculates display prices reactively via context. Does not change main state.
4. **Document type**: Orthogonal -- can be changed during PAYMENT. Default: customer present becomes Factura, no customer becomes Ticket.
5. **Register closed**: Blocks Cobrar. Show toast: "Open register first."

### State Diagram

```
                  +---------------------------------------+
                  |                                       |
                  v                                       |
            +----------+    scan / tap row      +-----------+
            |  IDLE /  | ---------------------> |   CART    |
            | BROWSING |                        |  ACTIVE   |
            +----------+ <--------------------- +-----------+---+
                  ^       clear / last removed         |       |
                  |                                    | Cobrar|
                  |                                    v       |
                  |                              +----------+   |
                  |                    IPC error  | PAYMENT  |   |
                  |         +------------------- |          |   |
                  |         v                    +----------+   |
                  |   +----------+               |    |         |
                  |   | PAYMENT  |<--------------+    | Cancel  |
                  |   | (error)  |                     |         |
                  |   +----------+                     v         |
                  |                              +----------+   |
                  |                              |COMPLETING|---+
                  |                              +----------+
                  |                                    |
                  +------------------------------------+
                         receipt done into IDLE
```

## Keyboard/Wedge Scanner Handling

USB barcode scanners present as keyboard devices (HID wedge). They emit keystrokes followed by Enter. The BarcodeInput component handles both scanner and manual input through a unified mechanism.

### BarcodeInput Design

```typescript
// Conceptual design -- BarcodeInput.tsx
function BarcodeInput({ onBarcodeScanned, onSearch }: Props) {
  const inputRef = useRef<HTMLInputElement>(null);

  // Permanent focus: auto-refocus after any interaction
  useEffect(() => {
    const handler = () => {
      // Skip refocus when payment modal is open
      if (!isPaymentOpen) inputRef.current?.focus();
    };
    document.addEventListener("click", handler);
    return () => document.removeEventListener("click", handler);
  }, [isPaymentOpen]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      const code = inputRef.current?.value.trim();
      if (code) {
        // Attempt exact barcode match first, fall back to FTS5 search
        onBarcodeScanned(code);
        inputRef.current.value = "";
      }
    }
  };

  return <input ref={inputRef} onKeyDown={handleKeyDown}
    placeholder="Escanea o escribe codigo..." autoFocus />;
}
```

### Input Strategy

1. **Every keystroke** accumulates in the input buffer (standard controlled input).
2. **On Enter**: call `get_product_by_code` (exact match) via Tauri IPC. If no match, show error toast. Product is added to cart with quantity 1.
3. **Debounced search (300ms)**: After the last keystroke without Enter, fire `search_products` (FTS5 fuzzy) for type-ahead results.
4. **Scanner vs manual**: No special detection needed. Scanner sends code + Enter rapidly; the 300ms debounce means FTS5 never fires for scans (Enter fires first).
5. **Focus restoration**: After any non-payment interaction (cart qty change, family tab click, customer select), the input refocuses via the click handler in useEffect.

### DOM Focus Heuristic

- BarcodeInput is the DEFAULT focus target.
- Payment modal inputs steal focus while open (refocus on BarcodeInput is suppressed).
- CustomerSelector dropdown has its own focus within the dropdown; closing it returns focus to BarcodeInput.
- FamilyNavBar tabs and ProductList rows are click-only (no focus needed).

## Customer Selection Flow

The customer selector is **always visible** in the POSHeader, not hidden until payment. This enables the ferreteria workflow where counter staff can assign a customer at any point during the sale.

### Flow

```
CustomerSelector (always in header)
    |
    +-- Click to open dropdown
    |   +-- Shows: nombre, nif, discount_percent
    |   +-- Search field: filter by nombre or NIF (local filter, no IPC)
    |   +-- Select customer sets customer in FerreteriaState
    |
    +-- Customer selected:
    |   +-- Discount applied: all PRECIO_VENTA values reduced by discount_percent
    |   +-- Badge: "-5%" shown next to customer name
    |   +-- DocumentType defaults to "factura" (overridable at payment)
    |   +-- Cart items: unit_price values are recomputed with discount
    |       (price * (1 - discount/100)) -- prices captured at ADD time
    |
    +-- Customer cleared:
        +-- Discount removed, original prices restored
        +-- DocumentType defaults to "ticket"
        +-- Cart items retain their unit_price at time of add
```

### Discount Application Rules

1. When a customer is SELECTED, the `discount_percent` from their tariff is applied to `precio_venta` for ALL products:
   - Displayed price in ProductList = `precio_venta * (1 - customer.discount_percent / 100)`
   - Displayed discount badge = "-customer.discount_percent%"
2. When a product is ADDED to cart, the unit_price captured is the DISCOUNTED price (if customer was selected at add time).
3. If customer selection CHANGES (or is removed) after items are in cart:
   - Existing items retain their original unit_price (captured at add time)
   - Only NEW items added after the change use the new discount
4. The customer discount badge shows the current customer's discount rate.
5. Total calculations: sum of line totals, unaffected by customer changes to existing items.

### Customer Search

- Local search over already-loaded customer list (loaded from Tauri at app init)
- Filter by `nombre` (case-insensitive contains) or `nif` (exact prefix)
- No additional IPC calls needed for search
- Customers without `nif` are still selectable (Ticket mode stays available)

## Document Type Decision Flow

The document type (Ticket vs Factura) is decided at payment time based on customer presence, with manual override available.

### Decision Logic

```
Payment Modal Opens
    |
    +-- Customer selected?
    |   +-- YES: default = Factura
    |   |   +-- Operator can toggle to: Ticket
    |   |       (shows warning: "El cliente no aparecera en el documento")
    |   |
    |   +-- NO: default = Ticket
    |       +-- Operator can toggle to: Factura
    |           (Factura without NIF allowed -- some businesses bill to "Consumidor Final")
    |
    v
DocumentTypeToggle component
    +-- Radio/button group: [Ticket] [Factura]
    +-- Active choice highlighted
    +-- Override is PER-SALE only (does not persist)
    +-- Default recalculates each time payment modal opens
```

### Impact on Receipt and Persisted Data

| Field | Ticket | Factura |
|-------|--------|---------|
| `document_type` | `"ticket"` | `"factura"` |
| Customer info on receipt | Omitted | Included (name, NIF, address) |
| IVA breakdown | Required | Required |
| Invoice number | Same sequence (no separate series) | Same sequence |

### Implementation Notes

- DocumentType is stored in `FerreteriaState.documentType`
- Default logic runs on each payment modal open: `customer ? "factura" : "ticket"`
- Manual override does not change customer -- the customer may still be selected for discount even with Ticket
- The CompletedSale record includes both `customer_id` and `document_type` -- if Ticket, customer data is omitted from receipt but still persisted

## Units of Measure Handling

Products in a ferreteria are sold per unit (ud), per meter (m), or per kilo (kg). The UI must adapt input and display based on the product's `unidad_medida` field.

### Quantity Input Behavior

| unidad_medida | Input Type | Step | Min | Max | Example |
|---------------|------------|------|-----|-----|---------|
| `"ud"` | integer | 1 | 1 | 9999 | 5 unidades |
| `"m"` | decimal (2 places) | 0.1 | 0.1 | 9999.99 | 1.50 m |
| `"kg"` | decimal (2 places) | 0.1 | 0.1 | 9999.99 | 2.75 kg |

### Cart Row Display

```
+----------+-----------+------+--------+-----------+----------+
| Producto | Cantidad  | Ud   | Precio | Descuento | Total    |
+----------+-----------+------+--------+-----------+----------+
| Tornillo |     4     | ud   | 0.25   |   0%      |  1.00    |
| Cable    |    1.5    | m    | 2.00   |   0%      |  3.00    |
| Arena    |    2.75   | kg   | 0.80   |   0%      |  2.20    |
+----------+-----------+------+--------+-----------+----------+
```

### Implementation Rules

1. The `unidad_medida` is captured from the product at add time and stored in `CartItem.unidadMedida`.
2. Quantity input in CartItemRow:
   - For `"ud"`: `<input type="number" step="1" min="1">` -- keyboard up/down increments by 1
   - For `"m"` / `"kg"`: `<input type="number" step="0.1" min="0.1">` -- decimal input, arrow keys increment by 0.1
3. The product list table includes a column showing the unit of measure abbreviation.
4. Line total calculation:
   - `lineTotal = unitPrice * quantity * (1 - discountPercent / 100)`
   - Quantity is not rounded for m/kg -- preserves precision
5. Quantity validation:
   - Negative quantities are blocked
   - Zero quantity removes the item from cart
   - For `"ud"`: decimals are rounded to nearest integer with a warning
6. The stock check is unit-agnostic -- stock value is in the same unit as `unidad_medida`

### Price Display

Unit prices always display with 2 decimal places and the appropriate unit label:
- "0.25 /ud", "2.00 /m", "0.80 /kg"

This matches the ferreteria convention where price labels show per-unit-of-measure.

## Terminal Health Display (WRN-4)

In `TerminalSettings`, display:
- **Sync status**: online/offline indicator + last sync timestamp
- **Pending count**: pending sales, closures, events
- **DB info**: SQLite file size, record counts
- **Version**: `tauri::Config` version from build metadata
- **Terminal ID**: current configured ID with edit capability

Data sourced from existing `sync-status-changed` events + new `get_terminal_health` command:

```rust
#[tauri::command]
fn get_terminal_health(state: State<'_, DbState>) -> Result<TerminalHealth, String>

struct TerminalHealth {
    db_size_bytes: u64,
    pending_sales: i64,
    pending_closures: i64,
    pending_events: i64,
    terminal_id: Option<String>,
    app_version: String,
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Rust unit | `search_products` FTS5 query with reindex | In-memory SQLite, insert productos, reindex, assert MATCH works |
| Rust unit | `get_today_sales` DATE filter | Insert sales on different days, assert TODAY filter |
| Rust unit | `void_sale_impl` chain integrity | Init 3 offline sales, void middle one, verify: status=VOIDED, stock restored, registro_sucesos ANULACION exists, ultimo_registro_encadenado unchanged |
| Rust unit | `void_sale_impl` stock restoration | Insert sale with items, void, assert stock_cache incremented correctly |
| Rust unit | `offline_sale` extended fields save/load | Insert with subtotal/tax/discount/status/void fields, roundtrip verify |
| Rust unit | `registrar_apertura`, `registrar_ingreso_caja`, `registrar_retiro_caja` | Insert, assert table contents |
| Rust integration | Sync HTTP 409 conflict detection | Mock server returns 409, assert sale stays PENDING, CONFLICT event recorded |
| React unit | Cart add/remove/qty/discount math | Vitest + render hooks |
| React integration | Payment flow (cash exact, split, insufficient) | Vitest + mocked Tauri IPC |
| Go unit | Void sync handler, idempotency | `httptest` with mock DB |
| Go integration | Payment + financial fields in sale sync payload | Mock server + DB assertions |

## Migration / Rollout

1. **Phase 1 — Schema**: Add FTS5 DDL, new columns via `init_db` ALTER, new tables (`caja_secuencia`, `offline_sale_payments`, `caja_aperturas`, `caja_movimientos`). Call `reindex_productos_fts()` in migration.
2. **Phase 2 — Backend**: Add Rust commands + `void_sale_impl` + remove old code in same commit. Keep `catalog_sync` reading `clientes` from server but not saving locally — backend still sends them, client ignores.
3. **Phase 3 — Go backend**: Add void endpoint. Extend sale struct with payments and financial fields. Add conflict response handling.
4. **Phase 4 — Frontend**: Rewrite App.tsx to pure POS. Old components are deleted. Rollback = `git revert` of the entire change branch.

## Risks and Mitigations

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| FTS5 index stale after catalog sync | Low | `reindex_productos_fts()` called after each catalog sync; triggers cover incremental changes |
| FTS5 content-sync triggers conflict with bulk insert | Low | Triggers are AFTER INSERT/UPDATE/DELETE per row; bulk inserts within a transaction fire triggers correctly. Reindex after bulk sync as safety net |
| ESC/POS WebUSB unsupported on ARM tablets | Low | PDF fallback via `tauri-plugin-dialog` + `print`; printer abstraction layer |
| Split payment tax proration rounding | Low | Distribute cents round-robin by line amount (banker's rounding) |
| `caja_secuencia` gap on failed sales | None by design | Sequence increments on successful insert only (atomic) |
| Void chain breaks linkage of subsequent sales | None | `ultimo_registro_encadenado` is never updated on void; existing sale signatures never mutate |

## Open Questions

- None resolved. All design decisions documented above.
