# Spec: TPV Tienda POS (Cash Register)

## Purpose

Define the behavior and data contracts for the cash register POS mode of the Ferrowin TPV desktop client. This spec covers product search, cart management, payment processing, daily operations, offline/sync, receipt printing, void/returns, and terminal management — all operating fully offline with background sync.

This specification SUPERSEDES `openspec/specs/tpv-desktop-client/spec.md` (REQ-CLI-01 through REQ-CLI-04) for POS-mode operations. The old spec's non-POS concerns (theme toggling) are retained there.

---

## Ferretería Workflow

The ferretería (hardware store) workflow is a specialization of the base POS designed for high-throughput counter sales. Its UX is optimized for USB barcode-scanner input (keyboard wedge), family-based browsing, and non-food units of measure.

### UX Concept

- **Barcode-first**: The scanner is the primary input device. Scanning a barcode instantly adds the product to cart with quantity 1. The search input retains permanent keyboard focus (wedge mode) so the next scan is immediately captured.
- **Product list view**: Products appear as a sortable data table (rows, not cards) showing code, name, unit, price, and an inline quantity field. More data per screen compared to a card grid.
- **Family filter tabs**: A horizontal nav bar above the product list shows family tabs (CERROJO, ELECTRICO, GAS, PINTURA, FERRETERIA, plus a "TODO" tab for all). Tapping a tab filters the product list to that family.
- **Customer always visible**: A customer selector is always accessible in the toolbar/header, not just at payment. When a customer is selected, their tariff discount auto-applies to displayed prices.
- **Document types**: Ticket (simplified anonymous receipt) vs Factura (full invoice with customer NIF/address). At payment time, if a customer is selected, Factura is the default.
- **Units of measure**: Products are sold per unit (ud), per meter (m), or per kilo (kg). Quantity input adapts: integer for `ud`, decimal for `m`/`kg`. Prices display per-unit-of-measure.
- **Payment**: Cash (with quick amounts 5/10/20/50€), Card (flat amount), Bizum (flat amount), or split across methods. Change calculation always shown for cash. Customer info and discount reflected on the payment screen.

The requirements in the 100+ series below describe ferretería-specific behaviors. Where these supplement base requirements (001–075), the ferretería-specific behavior takes precedence for this workflow.

---

## Requirements

### REQ-POS-001 through 010: Product Search & Selection

| ID | Title | Description | Prio | Verification |
|----|-------|-------------|------|-------------|
| REQ-POS-001 | Search by code | MUST search products by exact or partial `codigo` via local SQLite query | P0 | Test: code search returns matching product |
| REQ-POS-002 | Search by name (FTS5) | MUST search products by `nombre` using SQLite FTS5 for fuzzy term matching | P0 | Test: partial name returns relevant results within 500ms |
| REQ-POS-003 | Search by barcode | MUST search products by exact barcode match (keyboard wedge scanner) | P0 | Test: scanned barcode selects and adds product |
| REQ-POS-004 | Product grid | MUST display products as a tappable grid showing image, name, price, and cached stock | P0 | Visual: grid renders at 1280×800 with 4+ columns |
| REQ-POS-005 | Quick-add to cart | MUST add 1 unit of selected product to cart on single tap/click | P0 | Test: tap adds product to cart |
| REQ-POS-006 | Barcode scanner | MUST accept keyboard-wedge scanner input without requiring explicit focus | P1 | Manual: wedge scanner adds product from any screen |
| REQ-POS-007 | Filter by family | SHOULD allow filtering product grid by family (categoría) via dropdown | P1 | Test: filter shows only matching products |
| REQ-POS-008 | Stock indicator | MUST show stock availability as color-coded badge (green/yellow/red) per product | P1 | Visual: stock < 5 shows yellow, 0 shows red |
| REQ-POS-009 | Debounced input | SHOULD debounce search input by 300ms before executing FTS5 query | P2 | Test: rapid typing fires single query |
| REQ-POS-010 | Clear search | MUST provide one-tap button to clear search and restore full product grid | P1 | Test: clear resets display to all active products |

#### Scenarios

**Search by code finds exact match**
- GIVEN the catalog contains `codigo = "ART-001"`
- WHEN the operator types "ART-001" in the search field
- THEN the grid MUST show only that product

**FTS5 search by partial name**
- GIVEN the catalog contains "Tornillo hexagonal 8mm"
- WHEN the operator types "torni" in the search field
- THEN "Tornillo hexagonal 8mm" MUST appear in results within 500ms

**Barcode scanner adds directly to cart**
- GIVEN a USB barcode scanner is connected and active
- WHEN the scanner emits keystrokes for a valid product barcode
- THEN the product MUST be added to cart with quantity 1
- AND the search input MUST clear, ready for next scan

---

### REQ-POS-011 through 020: Cart Management

| ID | Title | Description | Prio | Verification |
|----|-------|-------------|------|-------------|
| REQ-POS-011 | Add to cart | MUST append selected product as a new line item with quantity 1 | P0 | Test: cart contains added item |
| REQ-POS-012 | Remove item | MUST remove a line item from cart on swipe or tap delete | P0 | Test: item removed, total recalculated |
| REQ-POS-013 | Change quantity | MUST allow increment/decrement of quantity per line via +/- buttons | P0 | Test: quantity changes, subtotal updates |
| REQ-POS-014 | Line subtotal | MUST display each line as `qty × price = subtotal` | P0 | Visual: subtotal matches calculation |
| REQ-POS-015 | Line discount | SHOULD apply per-line percentage discount (0-100%) with visual indicator | P1 | Test: 10% discount reduces line total |
| REQ-POS-016 | Tax breakdown | MUST show cart total split by IVA type (taxable base + tax amount per rate) | P0 | Test: IVA amounts sum to total tax |
| REQ-POS-017 | Scroll capacity | MUST display minimum 10 line items without scrolling at 1024×768 | P0 | Visual: 10 items visible in cart panel |
| REQ-POS-018 | Clear cart | MUST provide a "Clear all" action that removes all items after confirmation | P1 | Test: clear empties cart |
| REQ-POS-019 | Cart summary | MUST display unit count and total amount in the cart header at all times | P0 | Visual: summary always visible |
| REQ-POS-020 | Discount input | MUST allow entering a percentage discount (0-100%) per line item | P1 | Test: discount persists and reflects in total |

#### Scenarios

**Add multiple items and adjust quantities**
- GIVEN an empty cart
- WHEN the operator adds Product A (€10, qty 2) and Product B (€5, qty 1)
- THEN the cart MUST show 3 units totaling €25
- AND the IVA breakdown MUST match the sum of each product's tax

**Remove item recalculates total**
- GIVEN a cart with 3 line items
- WHEN the operator removes one item
- THEN the removed item MUST disappear from the list
- AND total MUST reflect the remaining items only

**Line discount applied**
- GIVEN a €100 line item in the cart
- WHEN the operator enters 15% discount for that line
- THEN the line subtotal MUST show €85
- AND the cart total MUST be recalculated

---

### REQ-POS-021 through 030: Payment Processing

| ID | Title | Description | Prio | Verification |
|----|-------|-------------|------|-------------|
| REQ-POS-021 | Cash payment | MUST accept cash tendered amount and calculate change due | P0 | Test: change = tendered - total |
| REQ-POS-022 | Card payment | MUST accept card as flat amount equal to total (no change) | P0 | Test: card completes payment |
| REQ-POS-023 | Split payment | MUST allow cash + card combination where amounts sum to total | P0 | Test: cash + card = total |
| REQ-POS-024 | Exact amount | MUST provide "Exact" button that sets cash tendered = total | P1 | Test: exact yields €0 change |
| REQ-POS-025 | Quick amounts | SHOULD provide quick-tender buttons (€5, €10, €20, €50) for cash | P1 | Test: button sets correct tendered |
| REQ-POS-026 | Print on payment | MUST trigger receipt printing immediately after successful payment | P0 | Manual: receipt prints after confirmation |
| REQ-POS-027 | Payment summary | MUST show methods + amounts before final confirmation | P1 | Visual: summary visible on confirm screen |
| REQ-POS-028 | Insufficient cash | MUST alert and block confirmation when cash tendered < total | P0 | Test: alert shown, not confirmable |
| REQ-POS-029 | Cancel payment | MUST allow returning to cart from payment screen without data loss | P1 | Test: cart state preserved after cancel |
| REQ-POS-030 | Tax proration | MUST prorate IVA proportionally per line when using split payment | P1 | Test: tax per method matches line ratio |

#### Scenarios

**Cash payment with change**
- GIVEN a cart totaling €23.50
- WHEN the operator enters cash tendered €50
- THEN the system MUST display €26.50 change
- AND MUST complete the sale on confirmation

**Split payment across methods**
- GIVEN a cart totaling €100.00
- WHEN the operator sets cash €40 and card €60
- THEN the system MUST verify the split sums to €100.00
- AND MUST record both payment methods on the CompletedSale

**Insufficient cash rejected**
- GIVEN a cart totaling €23.50
- WHEN the operator enters cash €20
- THEN an alert "Insufficient cash tendered" MUST appear
- AND the confirm button MUST be disabled

---

### REQ-POS-031 through 040: Daily Operations

| ID | Title | Description | Prio | Verification |
|----|-------|-------------|------|-------------|
| REQ-POS-031 | Open register | MUST record register opening with timestamp and initial cash amount | P0 | Test: open event saved locally |
| REQ-POS-032 | Close register | MUST record closure with timestamp, declared cash, and system totals | P0 | Test: closure saved, Z report generated |
| REQ-POS-033 | X report | MUST generate mid-day report showing sales totals without resetting sequence | P0 | Test: X shows correct totals, sequence unchanged |
| REQ-POS-034 | Z report | MUST generate end-of-day report and reset daily sequence counter | P0 | Test: Z resets counter, next sale starts at 1 |
| REQ-POS-035 | Cash in (petty) | MUST record cash additions to drawer (e.g., change fund) | P1 | Test: cash-in appears in movements |
| REQ-POS-036 | Cash out (petty) | MUST record cash removals from drawer (e.g., cash drop) | P1 | Test: cash-out appears in movements |
| REQ-POS-037 | Daily sales list | MUST display today's sales with timestamp, invoice#, and total | P1 | Visual: today's sales listed |
| REQ-POS-038 | Cash declaration | MUST prompt operator to enter declared cash amount on Z report | P1 | Test: variance = declared - system total |
| REQ-POS-039 | Offline closure storage | MUST save closures to `offline_box_closures` for background sync | P0 | Test: closure saved in PENDING status |
| REQ-POS-040 | Print closure report | SHOULD allow printing X or Z report on thermal printer | P2 | Manual: report prints |

#### Scenarios

**Open register starts session**
- GIVEN the terminal is idle
- WHEN the operator selects "Open Register" and enters €200 starting cash
- THEN the system MUST record an open event with timestamp
- AND MUST begin accepting POS sales

**X report preserves sequence**
- GIVEN 5 sales totaling €450 have been completed
- WHEN the operator requests an X report
- THEN the report MUST show 5 transactions, €450 total
- AND the sequence counter MUST remain unchanged

**Z report closes and resets**
- GIVEN 10 sales totaling €1,200 have been completed
- WHEN the operator requests a Z report and declares €1,250 cash
- THEN the report MUST show €1,200 system total vs €1,250 declared (€50 variance)
- AND the daily sequence counter MUST reset to 0

---

### REQ-POS-041 through 050: Offline & Sync

| ID | Title | Description | Prio | Verification |
|----|-------|-------------|------|-------------|
| REQ-POS-041 | Full offline operation | MUST save all completed sales to local SQLite when backend is unreachable | P0 | Test: sale saved with sync_status = PENDING |
| REQ-POS-042 | Offline closures | MUST save open/close/X/Z events to `offline_box_closures` when offline | P0 | Test: closure saved locally while offline |
| REQ-POS-043 | Background sync loop | MUST attempt sync every 30s when connectivity is restored | P0 | Test: sync fires on reconnection |
| REQ-POS-044 | Veri*factu chain locally | MUST maintain full signature chain (`hash_anterior` → `firma_registro`) in SQLite | P0 | Test: chain verified after 3 offline sales |
| REQ-POS-045 | Conflict detection | SHOULD detect sync conflicts (e.g., duplicate sequence on server — HTTP 409) | P1 | Test: 409 response keeps sale local as PENDING |
| REQ-POS-046 | Sync status indicator | MUST show online/offline status indicator in POS toolbar | P0 | Visual: indicator reflects `sync-status-changed` events |
| REQ-POS-047 | Pending count badge | MUST display count of pending records awaiting sync | P1 | Visual: badge shows pending count |
| REQ-POS-048 | Idempotent sync | MUST include `idempotency_key` (UUID) header on every sync request | P0 | Test: same key sent on retry |
| REQ-POS-049 | Safe cleanup | MUST delete local records ONLY after receiving 2xx HTTP from backend | P0 | Test: record persists on 4xx/5xx response |
| REQ-POS-050 | Reconnection auto-sync | MUST auto-sync all pending records within 30s of connection restoration | P1 | Test: pending count drops to 0 within 30s |

#### Scenarios

**Offline sale with full chaining**
- GIVEN the terminal is offline
- WHEN a sale for €50 is completed
- THEN the sale MUST be saved to `offline_sales` with populated `firma_registro`, `hash_anterior`, and `datos_encadenamiento`
- AND local cached stock MUST be decremented immediately

**Sync sends records and cleans up on success**
- GIVEN 3 offline sales are pending (sync_status = "PENDING")
- WHEN connectivity is restored and the sync loop runs
- THEN each sale MUST be POSTed to `/api/v1/sync/sales` with its idempotency key
- AND upon 2xx, each synced sale MUST be deleted from local storage

**Conflict on duplicate sequence**
- GIVEN an offline sale has `sequence_number = 42`
- WHEN the backend responds HTTP 409 for that sequence
- THEN the sale MUST remain locally with sync_status = "PENDING"
- AND a `registro_sucesos` event of type "CONFLICT" MUST be recorded

---

### REQ-POS-051 through 060: Receipt Printing

| ID | Title | Description | Prio | Verification |
|----|-------|-------------|------|-------------|
| REQ-POS-051 | ESC/POS thermal | MUST attempt thermal receipt via Tauri print plugin or WebUSB ESC/POS protocol | P0 | Manual: thermal receipt prints |
| REQ-POS-052 | PDF fallback | MUST generate printable PDF when thermal printing is unavailable | P0 | Manual: PDF opens in system viewer |
| REQ-POS-053 | Receipt preview | MUST show on-screen receipt preview before triggering print | P1 | Visual: preview matches printed output |
| REQ-POS-054 | Re-print last receipt | MUST allow printing the last receipt again without re-entering payment | P1 | Test: re-print regenerates identical receipt |
| REQ-POS-055 | Printer configuration | SHOULD allow configuring printer port/interface in settings view | P2 | Manual: settings persist across restarts |
| REQ-POS-056 | Custom header/footer | SHOULD allow configuring receipt header and footer text (business name, address) | P2 | Manual: text appears on receipts |
| REQ-POS-057 | Business info on receipt | MUST include business name, NIF, and address on every receipt | P0 | Visual: receipt has business details |
| REQ-POS-058 | Itemized lines | MUST list each item with name, quantity, unit price, discount, and line total | P0 | Visual: receipt shows all line details |
| REQ-POS-059 | IVA breakdown on receipt | MUST show each IVA rate with taxable base and tax amount | P0 | Visual: receipt has IVA summary section |
| REQ-POS-060 | Payment details on receipt | MUST show payment method(s), amounts tendered, and change | P0 | Visual: receipt shows payment info |

#### Scenarios

**Thermal receipt after payment**
- GIVEN an ESC/POS printer is configured
- WHEN a payment completes successfully
- THEN the receipt MUST be sent to the thermal printer
- AND it MUST display all line items, IVA breakdown, and payment details

**PDF fallback without thermal printer**
- GIVEN no thermal printer is configured
- WHEN a payment completes
- THEN a PDF MUST be generated and opened in the OS default viewer
- AND the content MUST be identical to the thermal version

**Re-print last receipt**
- GIVEN the last sale was receipt #TPV-0042
- WHEN the operator selects "Re-print receipt"
- THEN the exact same receipt content MUST be generated
- AND MUST be marked "(Reprint)" on the thermal copy

---

### REQ-POS-061 through 065: Void/Returns

| ID | Title | Description | Prio | Verification |
|----|-------|-------------|------|-------------|
| REQ-POS-061 | Void current-day sale | MUST allow voiding any sale from the current day | P0 | Test: voided sale status = "VOIDED" |
| REQ-POS-062 | Reason selection | MUST require a void reason from a predefined list before confirming | P0 | Test: void blocked without reason |
| REQ-POS-063 | Chain marking for voids | MUST create a void entry in the Veri*factu chain; `firma_registro` MUST link to void | P0 | Test: void entry has valid chain linkage |
| REQ-POS-064 | Sync void records | MUST sync voids to backend on reconnection with idempotency key | P0 | Test: void synced and acknowledged |
| REQ-POS-065 | Void visibility | SHOULD show voided sales in a separate list with reason and timestamp | P1 | Visual: voided sales distinguishable from active |

#### Scenarios

**Void a sale with reason**
- GIVEN sale TPV-0042 exists in today's sales list
- WHEN the operator selects void with reason "Operator error"
- THEN the sale MUST be marked as VOIDED in local storage
- AND a void entry MUST be created in the Veri*factu chain with `firma_registro` and `hash_anterior`
- AND the void MUST have sync_status = "PENDING"

**Void preserves chain linkage**
- GIVEN sales TPV-0041 (signed), TPV-0042 (signed), and TPV-0043 (signed) in chain order
- WHEN TPV-0042 is voided
- THEN the void entry's `hash_anterior` MUST equal TPV-0041's `firma_registro`
- AND TPV-0043 MUST remain valid with its original `hash_anterior` pointing to TPV-0042's original `firma_registro`

---

### REQ-POS-071 through 075: Terminal Management

| ID | Title | Description | Prio | Verification |
|----|-------|-------------|------|-------------|
| REQ-POS-071 | Terminal ID | MUST persist a unique terminal identifier in `sincronizacion_metadatos` | P0 | Test: terminal ID survives restart |
| REQ-POS-072 | Ticket sequence | MUST maintain per-terminal numeric ticket sequence in `caja_secuencia` | P0 | Test: sequence increments per completed sale |
| REQ-POS-073 | Health/connectivity | MUST display terminal health: sync status, DB status, last sync time | P1 | Visual: status screen renders correctly |
| REQ-POS-074 | Database info | SHOULD show local DB size and pending record counts | P2 | Visual: DB info in settings |
| REQ-POS-075 | Version display | MUST display application version from Tauri build metadata | P1 | Visual: version shown in settings |

#### Scenarios

**Terminal ID configured**
- GIVEN the terminal has no configured ID
- WHEN the operator sets terminal ID to "CAJA-01" in settings
- THEN the ID MUST be saved to `sincronizacion_metadatos`
- AND all subsequent sales MUST include `terminal_id = "CAJA-01"`

**Sequence increments per sale**
- GIVEN the current sequence number is 42
- WHEN a sale is completed
- THEN the new sale MUST use `sequence_number = 43`
- AND the `caja_secuencia` table MUST persist the new value

**Z report resets daily sequence**
- GIVEN the daily sequence is at 87
- WHEN a Z report is generated
- THEN the daily counter MUST reset to 0
- AND the next sale of the new day MUST use `sequence_number = 1`

---

### REQ-POS-100 series: Barcode & Scan

| ID | Title | Description | Prio | Verification |
|----|-------|-------------|------|-------------|
| REQ-POS-101 | Auto-add on scan | MUST add product to cart with quantity 1 when a barcode is scanned via USB wedge | P0 | Test: scan adds product and clears input |
| REQ-POS-102 | Permanent focus | MUST keep keyboard focus on the barcode/search input at all times (wedge mode) | P0 | Test: focus returns after any interaction |
| REQ-POS-103 | Manual code entry | MUST allow typing a product code manually and adding it on Enter | P1 | Test: manual code adds product to cart |

#### Scenarios

**Barcode scan adds instantly**
- GIVEN the search input has focus
- WHEN a USB wedge scanner emits keystrokes for a valid product barcode
- THEN the product MUST appear in the cart with quantity 1
- AND the search input MUST clear and regain focus immediately

**Focus retained after cart interaction**
- GIVEN products are in the cart
- WHEN the operator taps a cart button (quantity +/- or remove)
- THEN the search input MUST regain focus automatically

**Manual fallback for unreadable barcodes**
- GIVEN a product's barcode is damaged
- WHEN the operator types the code manually and presses Enter
- THEN the product MUST be added to cart as if scanned

---

### REQ-POS-110 series: Product List View

| ID | Title | Description | Prio | Verification |
|----|-------|-------------|------|-------------|
| REQ-POS-111 | Product list table | MUST display products as a table/rows with columns: code, name, unit, price, stock, quantity | P0 | Visual: table renders with all columns |
| REQ-POS-112 | Tap row to add | MUST add product with quantity 1 when a row is tapped/clicked | P0 | Test: tap adds item to cart |
| REQ-POS-113 | Units of measure | MUST display and respect product unidad_medida (ud/m/kg) — integer qty for ud, decimal for m/kg | P0 | Test: decimal input enabled for m/kg only |

#### Scenarios

**Tap row adds to cart**
- GIVEN the product list shows 20 products
- WHEN the operator taps "Tornillo 8mm" row
- THEN "Tornillo 8mm" MUST appear in cart with quantity 1

**Decimal quantity for meter product**
- GIVEN a product with `unidad_medida = "m"`
- WHEN the operator taps to add it
- THEN the quantity input in the cart MUST accept decimals (e.g., 1.5)
- AND the line total MUST calculate as 1.5 × unit_price

**Stock column shows availability**
- GIVEN Product A has stock = 3 and Product B has stock = 0
- THEN Product A MUST show "3" in the stock column
- AND Product B MUST show "0" (or "AGOTADO") red-highlighted

---

### REQ-POS-120 series: Customer & Discount

| ID | Title | Description | Prio | Verification |
|----|-------|-------------|------|-------------|
| REQ-POS-121 | Customer header selector | MUST show a customer selector in the toolbar/header at all times, not only at payment | P0 | Visual: selector visible on main POS screen |
| REQ-POS-122 | Auto discount from tariff | MUST apply customer's tariff discount to all prices when customer is selected | P1 | Test: displayed prices reflect discount |
| REQ-POS-123 | Authorized price override | SHOULD allow authorized staff to override a product's unit price per line | P2 | Test: override changes line total |

#### Scenarios

**Customer selected, discount applied**
- GIVEN a customer "Construcciones SL" has a 5% tariff discount
- WHEN the operator selects that customer in the header
- THEN all displayed product prices MUST show the discounted amount
- AND a "-5%" badge MUST appear next to the customer name

**Discount reflects on payment screen**
- GIVEN a cart totaling €100 with customer discount of 5%
- WHEN the operator proceeds to payment
- THEN the payment screen MUST show the discounted total of €95
- AND MUST display the customer name and discount rate

**Price override requires authorization**
- GIVEN a product with precio_venta = €10
- WHEN staff without override permission attempts to change the unit price
- THEN the system MUST block the change and show "Unauthorized"
- AND an event MUST be logged in `registro_sucesos`

---

### REQ-POS-130 series: Document Types

| ID | Title | Description | Prio | Verification |
|----|-------|-------------|------|-------------|
| REQ-POS-131 | Ticket mode | MUST generate a simplified receipt (Ticket) when no customer is selected | P0 | Test: ticket receipt omits customer info |
| REQ-POS-132 | Factura mode | MUST generate a full invoice (Factura) with customer NIF/address when a customer is selected | P0 | Test: factura includes customer data |
| REQ-POS-133 | Auto-detect type | MUST default to Factura when a customer is present; MUST default to Ticket when no customer is selected | P1 | Test: document type matches customer presence |

#### Scenarios

**Anonymous sale generates Ticket**
- GIVEN no customer is selected
- WHEN the operator completes a sale
- THEN the receipt MUST be a Ticket (no NIF/address section)
- AND the document type on the CompletedSale MUST be "ticket"

**Customer present generates Factura**
- GIVEN customer "Construcciones SL" with NIF "B-12345678" is selected
- WHEN the operator completes a sale
- THEN the receipt MUST be a Factura with customer NIF and address
- AND the document type on the CompletedSale MUST be "factura"

**Operator can override document type**
- GIVEN customer "Construcciones SL" is selected (default Factura)
- WHEN the operator manually switches the document type to Ticket at payment
- THEN the receipt MUST be a Ticket (omit customer data)
- AND the operator MUST confirm the override with a warning: "El cliente no aparecerá en el documento"

---

## Data Contracts

### POSProduct (search result)

```typescript
interface POSProduct {
  id: string;
  codigo: string;
  nombre: string;
  precio_venta: number;
  tipo_iva_id: string;
  iva_porcentaje: number;
  familia_id?: string;
  familia_nombre?: string;
  unidad_medida: "ud" | "m" | "kg";   // default "ud"
  stock: number;                // from stock_cache; 0 if unknown
  imagen_url?: string;
  activo: boolean;
}
```

### CartItem / Cart (frontend state)

```typescript
interface CartItem {
  product: POSProduct;
  quantity: number;
  unidad_medida: "ud" | "m" | "kg";  // captured at add time from product
  unit_price: number;           // captured at add time
  discount_percent: number;     // 0–100
  line_total: number;           // (unit_price * qty) * (1 - discount/100)
  iva_amount: number;           // line_total * (iva_porcentaje / (100 + iva_porcentaje))
}

interface Cart {
  items: CartItem[];
  total_units: number;
  total_amount: number;
  total_discount: number;
  iva_breakdown: IVABreakdownEntry[];
  customer?: CustomerInfo;
  document_type: "ticket" | "factura";
}
```

### POSPayment

```typescript
type PaymentMethod = "cash" | "card" | "bizum";

interface POSPayment {
  method: PaymentMethod;
  amount: number;
}

interface PaymentSummary {
  payments: POSPayment[];
  total_tendered: number;
  change: number;               // cash tendered − total_amount
  completed: boolean;
}
```

### CompletedSale (persisted + synced)

```typescript
type SyncStatus = "PENDING" | "SYNCED" | "CONFLICT";
type SaleStatus = "COMPLETED" | "VOIDED";

interface CompletedSale {
  id: string;                   // UUID v4
  terminal_id: string;
  invoice_number: string;       // e.g. "TPV-0042"
  sequence_number: number;
  created_at: string;           // ISO 8601
  document_type: "ticket" | "factura";
  customer_id?: string;
  customer_name?: string;
  customer_nif?: string;
  customer_address?: string;
  customer_discount?: number;   // 0–100
  items: CompletedSaleItem[];
  payments: CompletedPayment[];
  total: number;
  iva_breakdown: IVABreakdownEntry[];
  // Veri*factu chain
  firma_registro?: string;
  hash_anterior?: string;
  datos_encadenamiento?: string;
  // Sync
  sync_status: SyncStatus;
  idempotency_key: string;      // UUID v4
  // Lifecycle
  status: SaleStatus;
  void_reason?: string;
  voided_at?: string;
}

interface CompletedSaleItem {
  id: string;
  item_id: string;
  quantity: number;
  unidad_medida: "ud" | "m" | "kg";
  unit_price: number;
  discount_percent: number;
  line_total: number;
}

interface CompletedPayment {
  method: PaymentMethod;
  amount: number;
}
```

### ReceiptData (for printing)

```typescript
interface ReceiptData {
  business_name: string;
  business_nif: string;
  business_address: string;
  terminal_id: string;
  invoice_number: string;
  created_at: string;
  document_type: "ticket" | "factura";
  customer_name?: string;
  customer_nif?: string;
  customer_address?: string;
  customer_discount?: number;
  items: Array<{
    name: string;
    codigo: string;
    quantity: number;
    unidad_medida: "ud" | "m" | "kg";
    unit_price: number;
    discount_percent: number;
    line_total: number;
  }>;
  payments: CompletedPayment[];
  total_tendered: number;
  change: number;
  iva_breakdown: IVABreakdownEntry[];
  total: number;
  is_reprint: boolean;
}

interface IVABreakdownEntry {
  tipo_iva_id: string;
  nombre: string;
  porcentaje: number;
  base: number;                 // taxable amount
  cuota: number;                // tax amount
}
```

### Customer

```typescript
interface CustomerInfo {
  id: string;
  nombre: string;
  nif?: string;
  email?: string;
  address?: string;
  discount_percent: number;     // tariff discount 0–100
}

interface CustomerSearchResult {
  id: string;
  nombre: string;
  nif?: string;
  discount_percent: number;
}
```

---

## Relation to Existing Specs

This specification SUPERSEDES `openspec/specs/tpv-desktop-client/spec.md` (REQ-CLI-01 through REQ-CLI-04) for all POS-mode operations. The old spec served as the initial scaffold for offline storage and connectivity warnings — those concerns are now fully covered by the POS requirements above.

| Old (tpv-desktop-client) | Superseded by | Notes |
|--------------------------|---------------|-------|
| REQ-CLI-01 Offline Storage | REQ-POS-041, 044, 049 | Expanded with full sale flow |
| REQ-CLI-02 Connection Warning | REQ-POS-046 | Now part of toolbar indicator |
| REQ-CLI-03 Signature Preservation | REQ-POS-044 | Now part of chain integrity |
| REQ-CLI-04 Dynamic Theme | — | Retained in old spec; UI concern outside POS scope |

The old `spec.md` is retained at its original path for its non-POS concerns. Future work on TPV Comercial (route sales) will reference this POS spec for shared infrastructure.
