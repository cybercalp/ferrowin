# Proposal: User Authentication and Role-Based Access Control (RBAC)

## Intent

Implement a real, database-backed security layer for the Go backend. Currently, the system uses a dummy stub (`securityStub`) that grants all permissions to any request, bypassing authorization. This change implements the database adapters for RBAC (Users, Groups, Role Sets, Roles) and wires them to enable real permission validation.

## Scope

### In Scope
- Implement UserRepository, GroupRepository, RoleSetRepository, and RoleRepository in a new SQL repository file.
- Implement a single multi-level JOIN query to fetch the entire User -> Group -> RoleSet -> Role hierarchy efficiently.
- Wire the real domain AuthService and repositories in main.go, replacing the temporary stub.
- Ensure SQL query compatibility for both PostgreSQL and SQLite backends.

### Out of Scope
- Offline user and RBAC tables in the Tauri client.
- Frontend user login interface or authentication UI in Tauri POS.
- Session token management, JWT handling, or user-session lifecycle.

## Capabilities

### New Capabilities
- None

### Modified Capabilities
- None (pure implementation of existing specs/user-auth-rbac/spec.md requirements)

## Approach

We will create a new repository implementation in `internal/security/adapters/sql_repository.go` implementing the interfaces in `internal/security/ports/repositories.go`.
To fetch the user and authorization tree efficiently and avoid multiple round-trips to the DB, `GetByID` and `GetByUsername` will retrieve the entire tree using standard multi-level `LEFT JOIN` statements across `users`, `user_groups`, `groups`, `group_role_sets`, `role_sets`, `role_set_roles`, and `roles`.
The fetched row details will be parsed into the nested domain structures in Go.
In `main.go`, we will initialize the `SQLUserRepository` and instantiate the real `NewAuthService` passing the repository, then pass this service into `salesdomain.NewSalesService`.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/security/adapters/sql_repository.go` | New | Implement SQL-based repositories for User, Group, RoleSet, and Role, including hierarchical JOIN fetches. |
| `main.go` | Modified | Instantiate real security repositories and auth service, replacing `securityStub`. |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Performance of multi-level JOIN queries | Low | Index foreign keys; permissions hierarchy is small, leading to fast index-only scans. |
| Compatibility between Postgres and SQLite SQL syntax | Medium | Use standard SQL JOIN and parameter substitution helpers to ensure compatibility on both backends. |

## Rollback Plan

Revert the codebase to the previous commit (restoring the `securityStub` in `main.go` and removing the new `sql_repository.go` file). Database tables are already created via migrations and do not need to be dropped.

## Dependencies

- None

## Success Criteria

- [ ] `UserRepository.GetByID` and `UserRepository.GetByUsername` retrieve a user with their complete nested group, role-set, and role hierarchy.
- [ ] Domain `AuthService.HasPermission` correctly resolves user access based on database records.
- [ ] Integration tests verify RBAC queries execute successfully on both PostgreSQL and SQLite.
- [ ] `main.go` compiles and runs with the real security adapters injected.
