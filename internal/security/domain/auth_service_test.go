package domain_test

import (
	"context"
	"testing"

	"ferrowin/internal/security/domain"

	"github.com/google/uuid"
)

// mockUserRepository implements ports.UserRepository for testing.
type mockUserRepository struct {
	users map[uuid.UUID]*domain.User
}

func (m *mockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if u, exists := m.users[id]; exists {
		return u, nil
	}
	return nil, nil
}

func (m *mockUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepository) Save(ctx context.Context, user *domain.User) error {
	m.users[user.ID] = user
	return nil
}

func TestAuthService_HasPermission(t *testing.T) {
	ctx := context.Background()

	// 1. Prepare Roles
	readAuditRole := domain.Role{ID: uuid.New(), Name: "read-audit"}
	writeSalesRole := domain.Role{ID: uuid.New(), Name: "write-sales"}
	deleteUserRole := domain.Role{ID: uuid.New(), Name: "delete-user"}

	// 2. Prepare Role Sets
	auditorRoleSet := domain.RoleSet{
		ID:    uuid.New(),
		Name:  "Auditor",
		Roles: []domain.Role{readAuditRole},
	}
	salesRoleSet := domain.RoleSet{
		ID:    uuid.New(),
		Name:  "Sales Agent",
		Roles: []domain.Role{writeSalesRole},
	}

	// 3. Prepare Groups
	auditGroup := domain.Group{
		ID:       uuid.New(),
		Name:     "Audit Dept",
		RoleSets: []domain.RoleSet{auditorRoleSet},
	}
	salesGroup := domain.Group{
		ID:       uuid.New(),
		Name:     "Sales Dept",
		RoleSets: []domain.RoleSet{salesRoleSet},
	}

	// 4. Prepare Users
	validUserID := uuid.New()
	validUser := &domain.User{
		ID:           validUserID,
		Username:     "auditor1",
		PasswordHash: "hashed_pwd",
		Groups:       []domain.Group{auditGroup},
	}

	multiGroupUserID := uuid.New()
	multiGroupUser := &domain.User{
		ID:           multiGroupUserID,
		Username:     "manager1",
		PasswordHash: "hashed_pwd",
		Groups:       []domain.Group{auditGroup, salesGroup},
	}

	mockRepo := &mockUserRepository{
		users: map[uuid.UUID]*domain.User{
			validUserID:      validUser,
			multiGroupUserID: multiGroupUser,
		},
	}

	authSvc := domain.NewAuthService(mockRepo)

	t.Run("Scenario: Authorize valid user permission (single group)", func(t *testing.T) {
		// GIVEN a User in a Group with Role Set containing "read-audit" Role
		// WHEN the User requests the audit log
		// THEN the system SHALL allow access
		hasPerm, err := authSvc.HasPermission(ctx, validUserID, "read-audit")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hasPerm {
			t.Errorf("expected user to have 'read-audit' permission, but got false")
		}
	})

	t.Run("Scenario: Deny user lacking permission (single group)", func(t *testing.T) {
		// GIVEN a User in a Group with Role Set lacking "delete-user" Role
		// WHEN the User requests user deletion
		// THEN the system MUST return access denied
		hasPerm, err := authSvc.HasPermission(ctx, validUserID, "delete-user")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasPerm {
			t.Errorf("expected user to NOT have 'delete-user' permission, but got true")
		}
	})

	t.Run("Scenario: Authorize permission from multiple groups", func(t *testing.T) {
		// User is in both audit and sales groups.
		// Check read-audit (from auditGroup) -> allowed
		hasAuditPerm, err := authSvc.HasPermission(ctx, multiGroupUserID, "read-audit")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hasAuditPerm {
			t.Errorf("expected multi-group user to have 'read-audit'")
		}

		// Check write-sales (from salesGroup) -> allowed
		hasSalesPerm, err := authSvc.HasPermission(ctx, multiGroupUserID, "write-sales")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hasSalesPerm {
			t.Errorf("expected multi-group user to have 'write-sales'")
		}

		// Check delete-user (not in any group) -> denied
		hasDeletePerm, err := authSvc.HasPermission(ctx, multiGroupUserID, deleteUserRole.Name)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasDeletePerm {
			t.Errorf("expected multi-group user to NOT have 'delete-user'")
		}
	})

	t.Run("Scenario: Deny non-existent user", func(t *testing.T) {
		nonExistentID := uuid.New()
		hasPerm, err := authSvc.HasPermission(ctx, nonExistentID, "read-audit")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasPerm {
			t.Errorf("expected non-existent user to NOT have permission")
		}
	})
}
