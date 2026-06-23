package adapters_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"ferrowin/internal/security/adapters"
	"ferrowin/internal/security/domain"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open test SQLite DB: %v", err)
	}

	queries := []string{
		`CREATE TABLE users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL
		)`,
		`CREATE TABLE groups (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL
		)`,
		`CREATE TABLE user_groups (
			user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			group_id TEXT REFERENCES groups(id) ON DELETE CASCADE,
			PRIMARY KEY(user_id, group_id)
		)`,
		`CREATE TABLE role_sets (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL
		)`,
		`CREATE TABLE group_role_sets (
			group_id TEXT REFERENCES groups(id) ON DELETE CASCADE,
			role_set_id TEXT REFERENCES role_sets(id) ON DELETE CASCADE,
			PRIMARY KEY(group_id, role_set_id)
		)`,
		`CREATE TABLE roles (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL
		)`,
		`CREATE TABLE role_set_roles (
			role_set_id TEXT REFERENCES role_sets(id) ON DELETE CASCADE,
			role_id TEXT REFERENCES roles(id) ON DELETE CASCADE,
			PRIMARY KEY(role_set_id, role_id)
		)`,
	}

	for _, q := range queries {
		if _, err = db.Exec(q); err != nil {
			db.Close()
			t.Fatalf("failed to run query %q: %v", q, err)
		}
	}

	cleanup := func() {
		db.Close()
	}
	return db, cleanup
}

func TestSQLRepository_SaveAndGet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	roleRepo := adapters.NewSQLRoleRepository(db, true)
	roleSetRepo := adapters.NewSQLRoleSetRepository(db, true)
	groupRepo := adapters.NewSQLGroupRepository(db, true)
	userRepo := adapters.NewSQLUserRepository(db, true)

	t.Run("Save and Get Role", func(t *testing.T) {
		roleID := uuid.New()
		role := &domain.Role{
			ID:   roleID,
			Name: "test-role-1",
		}

		err := roleRepo.Save(ctx, role)
		if err != nil {
			t.Fatalf("failed to save role: %v", err)
		}

		fetched, err := roleRepo.GetByID(ctx, roleID)
		if err != nil {
			t.Fatalf("failed to get role: %v", err)
		}
		if fetched == nil {
			t.Fatal("expected role to be found, got nil")
		}
		if fetched.Name != role.Name || fetched.ID != role.ID {
			t.Errorf("expected role %+v, got %+v", role, fetched)
		}
	})

	t.Run("Save and Get RoleSet", func(t *testing.T) {
		roleSetID := uuid.New()
		roleSet := &domain.RoleSet{
			ID:   roleSetID,
			Name: "test-role-set-1",
		}

		err := roleSetRepo.Save(ctx, roleSet)
		if err != nil {
			t.Fatalf("failed to save role set: %v", err)
		}

		fetched, err := roleSetRepo.GetByID(ctx, roleSetID)
		if err != nil {
			t.Fatalf("failed to get role set: %v", err)
		}
		if fetched == nil {
			t.Fatal("expected role set to be found, got nil")
		}
		if fetched.Name != roleSet.Name || fetched.ID != roleSet.ID {
			t.Errorf("expected role set %+v, got %+v", roleSet, fetched)
		}
	})

	t.Run("Save and Get Group", func(t *testing.T) {
		groupID := uuid.New()
		group := &domain.Group{
			ID:   groupID,
			Name: "test-group-1",
		}

		err := groupRepo.Save(ctx, group)
		if err != nil {
			t.Fatalf("failed to save group: %v", err)
		}

		fetched, err := groupRepo.GetByID(ctx, groupID)
		if err != nil {
			t.Fatalf("failed to get group: %v", err)
		}
		if fetched == nil {
			t.Fatal("expected group to be found, got nil")
		}
		if fetched.Name != group.Name || fetched.ID != group.ID {
			t.Errorf("expected group %+v, got %+v", group, fetched)
		}
	})

	t.Run("Save and Get User", func(t *testing.T) {
		userID := uuid.New()
		user := &domain.User{
			ID:           userID,
			Username:     "test-user-1",
			PasswordHash: "secret-hash",
		}

		err := userRepo.Save(ctx, user)
		if err != nil {
			t.Fatalf("failed to save user: %v", err)
		}

		fetched, err := userRepo.GetByID(ctx, userID)
		if err != nil {
			t.Fatalf("failed to get user: %v", err)
		}
		if fetched == nil {
			t.Fatal("expected user to be found, got nil")
		}
		if fetched.Username != user.Username || fetched.PasswordHash != user.PasswordHash {
			t.Errorf("expected user %+v, got %+v", user, fetched)
		}

		fetchedByUsername, err := userRepo.GetByUsername(ctx, user.Username)
		if err != nil {
			t.Fatalf("failed to get user by username: %v", err)
		}
		if fetchedByUsername == nil {
			t.Fatal("expected user to be found by username, got nil")
		}
		if fetchedByUsername.ID != user.ID {
			t.Errorf("expected user ID %s, got %s", user.ID, fetchedByUsername.ID)
		}
	})

	t.Run("Link Associations", func(t *testing.T) {
		roleID := uuid.New()
		role := &domain.Role{ID: roleID, Name: "permission-1"}
		_ = roleRepo.Save(ctx, role)

		roleSetID := uuid.New()
		roleSet := &domain.RoleSet{ID: roleSetID, Name: "role-set-assoc"}
		_ = roleSetRepo.Save(ctx, roleSet)

		groupID := uuid.New()
		group := &domain.Group{ID: groupID, Name: "group-assoc"}
		_ = groupRepo.Save(ctx, group)

		userID := uuid.New()
		user := &domain.User{ID: userID, Username: "user-assoc", PasswordHash: "hash"}
		_ = userRepo.Save(ctx, user)

		// 1. Assign Role to RoleSet
		err := roleRepo.AssignRoleToRoleSet(ctx, roleSetID, roleID)
		if err != nil {
			t.Fatalf("failed to assign role to roleset: %v", err)
		}

		// Verify role set has role
		fetchedRoleSet, err := roleSetRepo.GetByID(ctx, roleSetID)
		if err != nil || fetchedRoleSet == nil {
			t.Fatalf("failed to fetch roleset: %v", err)
		}
		if len(fetchedRoleSet.Roles) != 1 || fetchedRoleSet.Roles[0].ID != roleID {
			t.Errorf("expected role %s inside roleset, got %v", roleID, fetchedRoleSet.Roles)
		}

		// 2. Assign RoleSet to Group
		err = roleSetRepo.AssignRoleSetToGroup(ctx, groupID, roleSetID)
		if err != nil {
			t.Fatalf("failed to assign roleset to group: %v", err)
		}

		// Verify group has roleset
		fetchedGroup, err := groupRepo.GetByID(ctx, groupID)
		if err != nil || fetchedGroup == nil {
			t.Fatalf("failed to fetch group: %v", err)
		}
		if len(fetchedGroup.RoleSets) != 1 || fetchedGroup.RoleSets[0].ID != roleSetID {
			t.Errorf("expected roleset %s inside group, got %v", roleSetID, fetchedGroup.RoleSets)
		}

		// 3. Assign Group to User
		err = groupRepo.AssignGroupToUser(ctx, userID, groupID)
		if err != nil {
			t.Fatalf("failed to assign group to user: %v", err)
		}

		// Verify user has group
		fetchedUser, err := userRepo.GetByID(ctx, userID)
		if err != nil || fetchedUser == nil {
			t.Fatalf("failed to fetch user: %v", err)
		}
		if len(fetchedUser.Groups) != 1 || fetchedUser.Groups[0].ID != groupID {
			t.Errorf("expected group %s inside user, got %v", groupID, fetchedUser.Groups)
		}
	})
}

func TestSQLRepository_HierarchyLoading(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	roleRepo := adapters.NewSQLRoleRepository(db, true)
	roleSetRepo := adapters.NewSQLRoleSetRepository(db, true)
	groupRepo := adapters.NewSQLGroupRepository(db, true)
	userRepo := adapters.NewSQLUserRepository(db, true)

	// Roles
	r1 := &domain.Role{ID: uuid.New(), Name: "p-read"}
	r2 := &domain.Role{ID: uuid.New(), Name: "p-write"}
	r3 := &domain.Role{ID: uuid.New(), Name: "p-delete"}
	r4 := &domain.Role{ID: uuid.New(), Name: "p-admin"}

	for _, r := range []*domain.Role{r1, r2, r3, r4} {
		_ = roleRepo.Save(ctx, r)
	}

	// RoleSets
	rs1 := &domain.RoleSet{ID: uuid.New(), Name: "Reader"}
	rs2 := &domain.RoleSet{ID: uuid.New(), Name: "Writer"}
	rs3 := &domain.RoleSet{ID: uuid.New(), Name: "Admin"}

	for _, rs := range []*domain.RoleSet{rs1, rs2, rs3} {
		_ = roleSetRepo.Save(ctx, rs)
	}

	// Groups
	g1 := &domain.Group{ID: uuid.New(), Name: "RegularStaff"}
	g2 := &domain.Group{ID: uuid.New(), Name: "Management"}

	for _, g := range []*domain.Group{g1, g2} {
		_ = groupRepo.Save(ctx, g)
	}

	// User
	user := &domain.User{ID: uuid.New(), Username: "john_doe", PasswordHash: "hash"}
	_ = userRepo.Save(ctx, user)

	// Link: rs1 has r1
	_ = roleRepo.AssignRoleToRoleSet(ctx, rs1.ID, r1.ID)
	// Link: rs2 has r2, r3
	_ = roleRepo.AssignRoleToRoleSet(ctx, rs2.ID, r2.ID)
	_ = roleRepo.AssignRoleToRoleSet(ctx, rs2.ID, r3.ID)
	// Link: rs3 has r4, and also r1 (shared role)
	_ = roleRepo.AssignRoleToRoleSet(ctx, rs3.ID, r4.ID)
	_ = roleRepo.AssignRoleToRoleSet(ctx, rs3.ID, r1.ID)

	// Link: g1 has rs1, rs2
	_ = roleSetRepo.AssignRoleSetToGroup(ctx, g1.ID, rs1.ID)
	_ = roleSetRepo.AssignRoleSetToGroup(ctx, g1.ID, rs2.ID)

	// Link: g2 has rs2, rs3 (shared role set rs2!)
	_ = roleSetRepo.AssignRoleSetToGroup(ctx, g2.ID, rs2.ID)
	_ = roleSetRepo.AssignRoleSetToGroup(ctx, g2.ID, rs3.ID)

	// Link: user is in both g1 and g2
	_ = groupRepo.AssignGroupToUser(ctx, user.ID, g1.ID)
	_ = groupRepo.AssignGroupToUser(ctx, user.ID, g2.ID)

	// Now retrieve the user
	fetchedUser, err := userRepo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to fetch user hierarchy: %v", err)
	}
	if fetchedUser == nil {
		t.Fatal("expected user to be loaded")
	}

	// We verify counts and deduplication
	if len(fetchedUser.Groups) != 2 {
		t.Fatalf("expected exactly 2 groups, got %d", len(fetchedUser.Groups))
	}

	// Let's index the groups by ID to check contents
	groupsByID := make(map[uuid.UUID]domain.Group)
	for _, g := range fetchedUser.Groups {
		groupsByID[g.ID] = g
	}

	g1Fetched, exists := groupsByID[g1.ID]
	if !exists {
		t.Fatalf("group 1 missing from fetched user")
	}
	if len(g1Fetched.RoleSets) != 2 {
		t.Errorf("expected 2 role sets in group 1, got %d", len(g1Fetched.RoleSets))
	}

	g2Fetched, exists := groupsByID[g2.ID]
	if !exists {
		t.Fatalf("group 2 missing from fetched user")
	}
	if len(g2Fetched.RoleSets) != 2 {
		t.Errorf("expected 2 role sets in group 2, got %d", len(g2Fetched.RoleSets))
	}

	// Validate role set contents and their roles
	for _, g := range fetchedUser.Groups {
		for _, rs := range g.RoleSets {
			if rs.ID == rs1.ID {
				if len(rs.Roles) != 1 || rs.Roles[0].ID != r1.ID {
					t.Errorf("RoleSet 1 has wrong roles: %+v", rs.Roles)
				}
			} else if rs.ID == rs2.ID {
				if len(rs.Roles) != 2 {
					t.Errorf("RoleSet 2 has wrong role count: %d", len(rs.Roles))
				}
			} else if rs.ID == rs3.ID {
				if len(rs.Roles) != 2 {
					t.Errorf("RoleSet 3 has wrong role count: %d", len(rs.Roles))
				}
			}
		}
	}
}

func TestSQLRepository_AuthServiceIntegration(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	roleRepo := adapters.NewSQLRoleRepository(db, true)
	roleSetRepo := adapters.NewSQLRoleSetRepository(db, true)
	groupRepo := adapters.NewSQLGroupRepository(db, true)
	userRepo := adapters.NewSQLUserRepository(db, true)

	// JWT config for auth service — not used in permission tests but required by constructor
	testJWT := domain.NewJWTConfig("test-integration-secret", 1*time.Hour)
	authSvc := domain.NewAuthService(userRepo, testJWT)

	// Roles
	r1 := &domain.Role{ID: uuid.New(), Name: "perm-create-invoice"}
	r2 := &domain.Role{ID: uuid.New(), Name: "perm-void-invoice"}
	_ = roleRepo.Save(ctx, r1)
	_ = roleRepo.Save(ctx, r2)

	// RoleSets
	rs := &domain.RoleSet{ID: uuid.New(), Name: "InvoiceManager"}
	_ = roleSetRepo.Save(ctx, rs)
	_ = roleRepo.AssignRoleToRoleSet(ctx, rs.ID, r1.ID)

	// Groups
	g := &domain.Group{ID: uuid.New(), Name: "BillingTeam"}
	_ = groupRepo.Save(ctx, g)
	_ = roleSetRepo.AssignRoleSetToGroup(ctx, g.ID, rs.ID)

	// User
	user := &domain.User{ID: uuid.New(), Username: "billing_agent", PasswordHash: "hash"}
	_ = userRepo.Save(ctx, user)
	_ = groupRepo.AssignGroupToUser(ctx, user.ID, g.ID)

	// Now assert permission check via AuthService
	hasCreateInvoice, err := authSvc.HasPermission(ctx, user.ID, "perm-create-invoice")
	if err != nil {
		t.Fatalf("unexpected error checking permission: %v", err)
	}
	if !hasCreateInvoice {
		t.Errorf("expected user to have permission 'perm-create-invoice'")
	}

	hasVoidInvoice, err := authSvc.HasPermission(ctx, user.ID, "perm-void-invoice")
	if err != nil {
		t.Fatalf("unexpected error checking permission: %v", err)
	}
	if hasVoidInvoice {
		t.Errorf("expected user to NOT have permission 'perm-void-invoice'")
	}

	hasUnknown, err := authSvc.HasPermission(ctx, user.ID, "some-random-perm")
	if err != nil {
		t.Fatalf("unexpected error checking permission: %v", err)
	}
	if hasUnknown {
		t.Errorf("expected user to NOT have unknown permission")
	}
}

func TestSQLRepository_TransactionPropagation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	roleRepo := adapters.NewSQLRoleRepository(db, true)

	roleID := uuid.New()
	role := &domain.Role{ID: roleID, Name: "tx-role"}

	t.Run("Rollback transaction", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to start tx: %v", err)
		}

		txCtx := adapters.WithTx(ctx, tx)

		// Save within transaction context
		err = roleRepo.Save(txCtx, role)
		if err != nil {
			t.Fatalf("failed to save in tx: %v", err)
		}

		// Verify the role is visible within the transaction
		fetched, err := roleRepo.GetByID(txCtx, roleID)
		if err != nil {
			t.Fatalf("failed to query role within tx: %v", err)
		}
		if fetched == nil {
			t.Fatal("expected role to be visible inside the transaction")
		}

		// Rollback transaction
		err = tx.Rollback()
		if err != nil {
			t.Fatalf("failed to rollback: %v", err)
		}

		// Get after rollback should return nil
		fetched, err = roleRepo.GetByID(ctx, roleID)
		if err != nil {
			t.Fatalf("failed to query role after rollback: %v", err)
		}
		if fetched != nil {
			t.Fatal("expected role to not exist after rollback")
		}
	})

	t.Run("Commit transaction", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to start tx: %v", err)
		}

		txCtx := adapters.WithTx(ctx, tx)

		// Save within transaction context
		err = roleRepo.Save(txCtx, role)
		if err != nil {
			t.Fatalf("failed to save in tx: %v", err)
		}

		// Commit transaction
		err = tx.Commit()
		if err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Get after commit should return the role
		fetched, err := roleRepo.GetByID(ctx, roleID)
		if err != nil {
			t.Fatalf("failed to query role after commit: %v", err)
		}
		if fetched == nil {
			t.Fatal("expected role to exist after commit")
		}
		if fetched.Name != role.Name {
			t.Errorf("expected role name %s, got %s", role.Name, fetched.Name)
		}
	})
}
