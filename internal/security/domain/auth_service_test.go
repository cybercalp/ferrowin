package domain_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ferrowin/internal/security/domain"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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

func testJWTConfig() *domain.JWTConfig {
	return domain.NewJWTConfig("test-secret-for-auth-service-test", 1*time.Hour)
}

func hashPwd(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return string(hash)
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

	authSvc := domain.NewAuthService(mockRepo, testJWTConfig())

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
		hasPerm, err := authSvc.HasPermission(ctx, validUserID, "delete-user")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasPerm {
			t.Errorf("expected user to NOT have 'delete-user' permission, but got true")
		}
	})

	t.Run("Scenario: Authorize permission from multiple groups", func(t *testing.T) {
		hasAuditPerm, err := authSvc.HasPermission(ctx, multiGroupUserID, "read-audit")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hasAuditPerm {
			t.Errorf("expected multi-group user to have 'read-audit'")
		}

		hasSalesPerm, err := authSvc.HasPermission(ctx, multiGroupUserID, "write-sales")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hasSalesPerm {
			t.Errorf("expected multi-group user to have 'write-sales'")
		}

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

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()
	cfg := testJWTConfig()

	correctPassword := "secure-password-123"
	hash := hashPwd(t, correctPassword)

	userID := uuid.New()
	testUser := &domain.User{
		ID:           userID,
		Username:     "admin",
		PasswordHash: hash,
		Groups:       []domain.Group{},
	}

	mockRepo := &mockUserRepository{
		users: map[uuid.UUID]*domain.User{
			userID: testUser,
		},
	}

	authSvc := domain.NewAuthService(mockRepo, cfg)

	t.Run("valid credentials return token and user info", func(t *testing.T) {
		resp, err := authSvc.Login(ctx, "admin", correctPassword)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Token == "" {
			t.Error("expected non-empty token")
		}
		if resp.User == nil {
			t.Fatal("expected non-nil user")
		}
		if resp.User.ID != userID {
			t.Errorf("expected user ID %v, got %v", userID, resp.User.ID)
		}
		if resp.User.Username != "admin" {
			t.Errorf("expected username 'admin', got %q", resp.User.Username)
		}

		// Validate that the returned token is valid
		claims, err := cfg.ValidateToken(resp.Token)
		if err != nil {
			t.Fatalf("returned token should be valid: %v", err)
		}
		if claims.Username != "admin" {
			t.Errorf("expected claims username 'admin', got %q", claims.Username)
		}
	})

	t.Run("wrong password returns ErrInvalidCredentials", func(t *testing.T) {
		_, err := authSvc.Login(ctx, "admin", "wrong-password")
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("unknown username returns ErrInvalidCredentials", func(t *testing.T) {
		_, err := authSvc.Login(ctx, "nonexistent", correctPassword)
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("empty username returns ErrInvalidCredentials", func(t *testing.T) {
		_, err := authSvc.Login(ctx, "", correctPassword)
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("empty password returns ErrInvalidCredentials", func(t *testing.T) {
		_, err := authSvc.Login(ctx, "admin", "")
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
		}
	})
}
