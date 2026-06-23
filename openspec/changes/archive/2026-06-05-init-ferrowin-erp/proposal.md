# Proposal: Initial Architecture and Data Model of Ferrowin ERP

## Intent

Establish the foundational database schemas, models, and API interfaces for the greenfield Ferrowin ERP, addressing security, stock tracking, billing, and document flows.

## Scope

### In Scope
- Define DB schema for RBAC (Users, Groups, Roles, Role Sets).
- Implement configurable billing series per terminal.
- Design shared stock ledger ("Movimientos de Almacén") schema.
- Define DB schema and tracking model for document flow (Quote -> Order -> Delivery Note -> Invoice).
- Support offline POS sales ignoring strict stock validation.

### Out of Scope
- Native desktop POS client implementation.
- Sync engine queue processing implementation.

## Capabilities

> This section is the CONTRACT between proposal and specs phases.
> The sdd-spec agent reads this to know exactly which spec files to create or update.

### New Capabilities
- `user-auth-rbac`: Security model supporting Users, Groups, Roles, and Role Sets.
- `billing-series-management`: Configurable invoice series per terminal.
- `document-traceability`: Quote to invoice progression tracking.
- `inventory-ledger`: Centralized shared stock ledger for inventory moves.
- `offline-pos-stock`: Sales stock adjustment logic for reconnected terminals.

### Modified Capabilities

## Approach

Develop a modular Go/TypeScript backend with PostgreSQL. Create schema migrations for RBAC, sales documents, ledger, and terminal-series configurations. Implement API endpoints for document creation and synchronization.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `database/migrations` | New | Database schemas for RBAC, sales, stock, and terminals. |
| `src/core/security` | New | RBAC authentication and authorization models. |
| `src/core/inventory` | New | Shared stock ledger operations. |
| `src/core/billing` | New | Terminal billing series counters. |
| `src/core/sales` | New | Traceable document flow validation. |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Duplicate bill IDs from offline POS | Low | Restrict each terminal to unique invoicing series prefixes. |
| Offline negative stock discrepancies | Med | Process sync sales via central stock adjustments with FIFO reconciliation. |

## Rollback Plan

Run down database migrations using CLI migration rollback scripts and revert application code to the initial commit.

## Dependencies

- Database driver/migration tool setup.

## Success Criteria

- [ ] Database schemas support all five core capabilities.
- [ ] Integrated document flow can transition from Quote to Invoice.
- [ ] Stock adjustments from offline tickets correctly update the ledger.
