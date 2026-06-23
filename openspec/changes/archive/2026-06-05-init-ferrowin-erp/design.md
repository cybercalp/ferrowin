# Technical Design: Ferrowin ERP Core

## Technical Approach
We implement a Modular Monolith in Go/TypeScript with a central PostgreSQL database. To support local offline terminals, they run SQLite locally. Offline TPV sales are uploaded via a REST API. To prevent sales blocks due to local database differences, central stock verification is bypassed during sync, allowing negative balances. A FIFO Stock Reconciliation service later reconciles stock balances when receipts are processed.

## Architecture Decisions

| Decision | Rationale | Alternatives |
|---|---|---|
| **Modular Monolith** | Simplifies deployment and transactional consistency while keeping clean boundaries for future microservice extraction. | Microservices (adds premature network/deploy overhead) |
| **PostgreSQL Central DB** | ACID compliance, JSONB support for extensibility, and robust transaction locking for sales document flows. | MySQL / MongoDB |
| **SQLite for Offline POS** | Lightweight, zero-configuration embedded database suited for local hardware store terminal reliability. | Local PostgreSQL (excessive resource usage) |
| **Bypass Stock Validation on Sync** | Ensures local offline sales are successfully received by the backend, allowing negative stock to be resolved later. | Rejecting sync (causes sales data loss) |
| **Idempotency Key Sync** | Guarantees exactly-once processing for sales uploaded over unreliable network connections. | Simple retry without keys (causes duplicate invoices) |

## Data Flow
```
Document Flow:
[Quote (Draft/Approved)] ---> [Order (Draft/Approved)] ---> [Delivery Note] ---> [Invoice]

Sync Flow:
[SQLite Offline Sale] ---> [Sync API Client] --(Idempotency-Key)--> [Go/TS API Gateway] ---> [PostgreSQL DB] ---> [FIFO Reconciliation Worker]
```

## File Changes
* `database/migrations/000001_init_erp_schemas.up.sql` (New PostgreSQL schema migration)
* `database/migrations/sqlite_init.sql` (New SQLite local TPV schema setup)
* `src/core/security/` (New RBAC authorization engine modules)
* `src/core/billing/` (New invoice number generator & series controller)
* `src/core/sales/` (New document transitions and state locks)
* `src/core/inventory/` (New shared stock ledger & FIFO reconciliation)
* `src/api/sync/` (New sales upload and terminal registration sync controllers)

## Interfaces & Contracts

### PostgreSQL Schema
```sql
-- RBAC Security
CREATE TABLE users (id UUID PRIMARY KEY, username VARCHAR(50) UNIQUE NOT NULL, password_hash VARCHAR(255) NOT NULL);
CREATE TABLE groups (id UUID PRIMARY KEY, name VARCHAR(50) UNIQUE NOT NULL);
CREATE TABLE user_groups (user_id UUID REFERENCES users(id) ON DELETE CASCADE, group_id UUID REFERENCES groups(id) ON DELETE CASCADE, PRIMARY KEY(user_id, group_id));
CREATE TABLE role_sets (id UUID PRIMARY KEY, name VARCHAR(50) UNIQUE NOT NULL);
CREATE TABLE group_role_sets (group_id UUID REFERENCES groups(id) ON DELETE CASCADE, role_set_id UUID REFERENCES role_sets(id) ON DELETE CASCADE, PRIMARY KEY(group_id, role_set_id));
CREATE TABLE roles (id UUID PRIMARY KEY, name VARCHAR(50) UNIQUE NOT NULL);
CREATE TABLE role_set_roles (role_set_id UUID REFERENCES role_sets(id) ON DELETE CASCADE, role_id UUID REFERENCES roles(id) ON DELETE CASCADE, PRIMARY KEY(role_set_id, role_id));

-- Terminals & Billing Series
CREATE TABLE terminals (id UUID PRIMARY KEY, name VARCHAR(50) UNIQUE NOT NULL, is_active BOOLEAN DEFAULT TRUE);
CREATE TABLE invoicing_series (id UUID PRIMARY KEY, terminal_id UUID REFERENCES terminals(id) ON DELETE RESTRICT, prefix VARCHAR(10) UNIQUE NOT NULL, next_sequence INT NOT NULL DEFAULT 1);

-- Traceable Sales Documents
CREATE TABLE quote (id UUID PRIMARY KEY, client_id UUID NOT NULL, total NUMERIC(12,2) NOT NULL, status VARCHAR(20) CHECK (status IN ('Draft', 'Approved', 'Converted', 'Cancelled')), created_at TIMESTAMP DEFAULT NOW());
CREATE TABLE "order" (id UUID PRIMARY KEY, quote_id UUID REFERENCES quote(id) ON DELETE SET NULL, total NUMERIC(12,2) NOT NULL, status VARCHAR(20) CHECK (status IN ('Draft', 'Approved', 'Converted', 'Cancelled')), created_at TIMESTAMP DEFAULT NOW());
CREATE TABLE delivery_note (id UUID PRIMARY KEY, order_id UUID REFERENCES "order"(id) ON DELETE SET NULL, total NUMERIC(12,2) NOT NULL, status VARCHAR(20) CHECK (status IN ('Draft', 'Converted', 'Cancelled')), created_at TIMESTAMP DEFAULT NOW());
CREATE TABLE invoice (id UUID PRIMARY KEY, delivery_note_id UUID REFERENCES delivery_note(id) ON DELETE SET NULL, terminal_id UUID REFERENCES terminals(id) ON DELETE RESTRICT, invoicing_series_id UUID REFERENCES invoicing_series(id) ON DELETE RESTRICT, invoice_number VARCHAR(30) UNIQUE NOT NULL, sequence_number INT NOT NULL, total NUMERIC(12,2) NOT NULL, status VARCHAR(20), created_at TIMESTAMP DEFAULT NOW());

-- Stock Ledger
CREATE TABLE stock_ledger_movements (id UUID PRIMARY KEY, item_id UUID NOT NULL, warehouse_id UUID NOT NULL, quantity NUMERIC(12,4) NOT NULL, movement_type VARCHAR(20) CHECK (movement_type IN ('RECEIPT', 'WITHDRAWAL', 'SYNC_ADJUSTMENT')), reference_document_type VARCHAR(20), reference_document_id UUID, created_at TIMESTAMP DEFAULT NOW());
```

### SQLite Schema (TPV Offline Sales)
```sql
CREATE TABLE offline_sales (id TEXT PRIMARY KEY, terminal_id TEXT NOT NULL, customer_id TEXT, total REAL NOT NULL, created_at TEXT NOT NULL, sync_status TEXT DEFAULT 'PENDING', idempotency_key TEXT UNIQUE NOT NULL, invoice_number TEXT UNIQUE NOT NULL, sequence_number INTEGER NOT NULL);
CREATE TABLE offline_sale_items (id TEXT PRIMARY KEY, offline_sale_id TEXT REFERENCES offline_sales(id) ON DELETE CASCADE, item_id TEXT NOT NULL, quantity REAL NOT NULL, unit_price REAL NOT NULL);
```

### API Contracts

#### 1. Register Terminal (`POST /api/v1/sync/register`)
* **Headers**: `Content-Type: application/json`
* **Request**:
  ```json
  {"terminal_name": "TPV-01", "requested_prefix": "S1"}
  ```
* **Response (201 Created)**:
  ```json
  {"terminal_id": "8a32a6fa-21d9-482a-9ff8-e215456f912e", "assigned_prefix": "S1"}
  ```

#### 2. Sync Sales (`POST /api/v1/sync/sales`)
* **Headers**: `Content-Type: application/json`, `Idempotency-Key: <UUID>`
* **Request**:
  ```json
  {
    "sales": [{
      "id": "7b2d56fa-21d9-482a-9ff8-e215456f12ab",
      "invoice_number": "S1-16",
      "sequence_number": 16,
      "created_at": "2026-06-05T13:00:00Z",
      "total": 150.00,
      "items": [{"item_id": "3a11a2f1-c4d3-48b0-8e10-3331b2345678", "quantity": 2.0, "unit_price": 75.00}]
    }]
  }
  ```
* **Response (200 OK)**:
  ```json
  {"status": "success", "synced_count": 1, "processed_ids": ["7b2d56fa-21d9-482a-9ff8-e215456f12ab"]}
  ```

## Code Structure (Modular Monolith)

### Go Project Directory Structure
```
/cmd
  /api
    main.go
/internal
  /security
    /domain       # Entities, Values, Aggregate Roots (User, Group, Role)
    /ports        # Repository & Service interfaces
    /adapters     # DB repository implementations & auth controllers
  /billing
    /domain       # InvoicingSeries logic
    /ports        # Series service interface
    /adapters     # SQL sequence operations
  /sales
    /domain       # Documents logic (Quote, Order, DeliveryNote, Invoice)
  /inventory
    /domain       # Ledger movement, FIFO reconciliation logic
  /shared         # Common cross-cutting concerns (idempotency utils)
```

### TypeScript Project Directory Structure
```
/src
  /modules
    /security
      /domain     # User, Role model definitions
      /infra      # TypeORM/Prisma schemas, API routes
      /application# Login, assign permissions commands
    /billing
    /sales
    /inventory
    /shared
```

## Testing Strategy
* **Unit Tests**: Validate RBAC hierarchy (User Group -> Role Set -> Role) authorization rules.
* **Integration Tests**: Verify document progression state transitions and ensure converting a locked parent document triggers transition errors.
* **Sync Integration Tests**: Verify sync accepts duplicate requests and returns the same payload without double-inserting records. Verify negative stock creation logic.
* **FIFO Stock Reconciliation Tests**: Verify that processing a stock receipt of 10 items when available stock is -2 corrects the balance to 8 and resolves pending offline adjustments.

## Open Questions
1. Should the system block converting a Quote to an Order if it has passed its expiration date, or are expired quotes allowed to convert with approval flags?
