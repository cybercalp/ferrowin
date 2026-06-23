# Implementation Tasks: user-auth-rbac

## Review Workload Forecast
Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

## Tasks

### Phase 1: SQL Repository Implementation (PR 1)
- [x] 1.1 Implement `getExecutor` context helper in `internal/security/adapters/sql_repository.go` for transaction propagation.
- [x] 1.2 Define standard SQL queries for users, groups, role sets, and roles mapping compatible with SQLite and Postgres.
- [x] 1.3 Implement `SQLUserRepository` with single-JOIN nested hierarchy loading mapping to the domain user model.
- [x] 1.4 Implement `SQLGroupRepository`, `SQLRoleSetRepository`, and `SQLRoleRepository` in the sql_repository file.

### Phase 2: Go Backend Wiring (PR 2)
- [x] 2.1 Update imports in `main.go` to reference the newly implemented SQL adapters and domain auth service.
- [x] 2.2 Instantiate `SQLUserRepository` using the global db connection and placeholder settings in `main.go`.
- [x] 2.3 Wire the real `AuthService` (replacing the temporary `securityStub`) and inject it into `salesdomain.NewSalesService`.

### Phase 3: Testing & Verification (PR 3)
- [x] 3.1 Create `internal/security/adapters/sql_repository_test.go` with schema migration setup using a temporary SQLite db.
- [x] 3.2 Write integration tests asserting that users, groups, role sets, and roles are correctly saved and linked.
- [x] 3.3 Add hierarchy loading tests verifying that nested domain models map properly without duplicate records.
- [x] 3.4 Write integration tests combining SQL repository database records and `AuthService` permission evaluations.
