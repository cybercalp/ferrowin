Decision needed before apply: Yes
Chained PRs recommended: Yes
400-line budget risk: High

# Tasks: Initial Architecture and Data Model of Ferrowin ERP

This task list tracks the implementation of core structures, models, databases, and APIs for the Ferrowin ERP. To fit within the 400-line reviewer budget, the work is sliced into 6 independent work units.

## Slice 1: Database Migrations and Environment Setup

- [x] Create PostgreSQL migration for core ERP schemas
  - Define schema tables for RBAC (users, groups, user_groups, role_sets, group_role_sets, roles, role_set_roles), terminals, invoicing_series, quote, order, delivery_note, invoice, and stock_ledger_movements.
  - Run database migration CLI to apply SQL schema and verify tables using database query tool.
- [x] Create SQLite schema setup script for local POS TPV
  - Define local tables for offline_sales and offline_sale_items with proper foreign keys and offline indexes.
  - Initialize SQLite database using the setup script and verify structures using sqlite3 CLI.

## Slice 2: RBAC Security Module

- [x] Implement RBAC Core Domain and Interfaces
  - Define Go domain structures and structures for User, Group, RoleSet, Role, and their respective repository/service interfaces.
  - Run unit tests `go test ./internal/security/domain/...` to verify domain model initialization.
- [x] Implement RBAC Permission Verification Engine
  - Write permission check service that traverses User -> Groups -> Role Sets -> Roles to authorize user operations.
  - Run unit tests validating access allowed and access denied cases based on user roles (REQ-SEC-01).

## Slice 3: Terminals & Billing Series Management

- [x] Implement Billing Series Domain and DB Repository
  - Create Go structures for Terminal and InvoicingSeries models, with SQL operations to retrieve and increment next sequence.
  - Run unit tests `go test ./internal/billing/...` to verify correct property bindings.
- [x] Implement Sequence Generator with Isolation
  - Write billing service that safely increments terminal sequence and formats invoice numbers (e.g. S1-16) under concurrency using database locks.
  - Run concurrent integration tests checking prefix isolation and safe concurrent increments (REQ-BIL-01).

## Slice 4: Sales Document Flows & Traceability

- [x] Implement Document Transition Rules and State Locks
  - Implement Go service for transitioning Quote -> Order -> Delivery Note -> Invoice. Prevent transition if a parent document is already converted or cancelled.
  - Run unit tests verifying document progression rules and locked parent states (REQ-TRC-01).
- [x] Implement Expired Quote Conversion to Order with Options
  - Add logic to check quote expiration. If expired, check if user has `convert-expired-quote` role. If authorized, allow conversion with options to either recalculate prices or accept original quote prices.
  - Run integration tests verifying conversion denial for unauthorized users, conversion success for authorized users, and application of price options.

## Slice 5: Stock Ledger & FIFO Reconciliation

- [x] Implement Shared Stock Ledger
  - Implement repository and services to log ledger movements for RECEIPT, WITHDRAWAL, and SYNC_ADJUSTMENT. Prevent withdrawal if available stock is insufficient.
  - Run integration tests verifying ledger insertion and withdrawal rejection on low stock (REQ-INV-01).
- [x] Implement FIFO Stock Reconciliation
  - Write stock reconciliation service to reconcile negative stock adjustments from offline syncs when a new stock receipt is registered.
  - Run unit tests verifying negative balance correction, e.g., bringing stock from -2 to 8 upon a +10 receipt (REQ-OFF-01).

## Slice 6: Offline POS Sales Sync & API Gateway

- [x] Implement Idempotency Key Verification Utilities
  - Create database utilities to track sync idempotency keys, preventing duplicate sync request handling.
  - Run unit tests verifying exact-match idempotency key rejection.
- [x] Implement Sales Sync API Endpoint
  - Implement POST `/api/v1/sync/sales` API controller. Bypass central stock validation during sync to allow negative central stock registration for offline sales.
  - Run sync integration tests uploading duplicate sync payloads, verifying standard success responses are returned and database changes are not duplicated.

