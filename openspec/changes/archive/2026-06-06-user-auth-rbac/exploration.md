## Exploration: user-auth-rbac

### Current State
- The Go backend simulates permission checking using a temporary `securityStub` in `main.go` that always returns `true, nil`.
- The domain models representing the RBAC structure (`User`, `Group`, `RoleSet`, `Role`) and authorization service logic (`authService.HasPermission`) are fully defined in `internal/security/domain/`.
- The repository ports for security are defined under `internal/security/ports/repositories.go`, but there is no database adapter implementation.
- PostgreSQL migrations for RBAC tables (`users`, `groups`, `user_groups`, `role_sets`, `group_role_sets`, `roles`, `role_set_roles`) already exist in `database/migrations/000001_init_erp_schemas.up.sql`.
- The Tauri desktop client SQLite database contains a `roles` column in the `clientes` table, but it is strictly for customer categorization within catalog sync, not point-of-sale operator authorization. Offline SQLite does not have system user or system RBAC tables, indicating that RBAC validation is only required central-side on the Go backend.

### Affected Areas
- `internal/security/adapters/sql_repository.go` — New file required to implement database repositories for users, groups, role sets, and roles.
- `main.go` — Needs to import security adapters, instantiate the database-backed repositories, instantiate the real authorization service, replace `securityStub`, and inject the service into `salesdomain.NewSalesService`.

### Approaches
1. **Hierarchy SQL Join Loading (Single Join Query)** — Retrieve the complete user RBAC tree (User -> Groups -> Role Sets -> Roles) in a single multi-level SQL JOIN query when fetching the user by ID or username. Parse the nested rows in Go to rebuild the hierarchical structure.
   - Pros: Efficient database access (only a single database roundtrip), perfectly populates the existing domain models, avoids multiple queries.
   - Cons: Scan logic is slightly verbose due to parsing nested structures from flat JOIN rows.
   - Effort: Medium

2. **Split Query Loading** — Fetch the user first, then perform separate sequential queries for groups, role sets, and roles, stitching the slices manually in Go.
   - Pros: Simpler SQL queries.
   - Cons: Multiple database roundtrips for a single permission check, introducing unnecessary latency.
   - Effort: High

### Recommendation
- **Approach 1 (Hierarchy SQL Join Loading)** is recommended because authorization is checked frequently during backend requests (e.g. quote-to-order conversions) and reducing database roundtrips is crucial for API latency. It also maps directly to the existing domain methods without modification.

### Risks
- **N+1 Queries**: Fetching relations sequentially would cause an N+1 query pattern. The JOIN query eliminates this risk.
- **SQLite Compatibility**: Because the dev environment toggle supports SQLite, all SQL repository queries must be written in an ANSI-compliant syntax supported by both PostgreSQL and SQLite.

### Ready for Proposal
Yes — The specifications are clear, the database migration structure is ready, and we confirmed that client-side SQLite does not require RBAC. The orchestrator should proceed with proposing the implementation of `sql_repository.go` and wiring it in `main.go`.
