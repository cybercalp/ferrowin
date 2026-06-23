package domain_test

import (
	"testing"

	"ferrowin/internal/security/domain"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func TestUser_HasPermission(t *testing.T) {
	// Build reusable domain objects
	roleRead := domain.Role{ID: uuid.New(), Name: "perm-read"}
	roleWrite := domain.Role{ID: uuid.New(), Name: "perm-write"}
	roleDelete := domain.Role{ID: uuid.New(), Name: "perm-delete"}
	roleAdmin := domain.Role{ID: uuid.New(), Name: "perm-admin"}

	readRoleSet := domain.RoleSet{
		ID:    uuid.New(),
		Name:  "Reader",
		Roles: []domain.Role{roleRead},
	}
	writeRoleSet := domain.RoleSet{
		ID:    uuid.New(),
		Name:  "Writer",
		Roles: []domain.Role{roleWrite, roleDelete},
	}
	adminRoleSet := domain.RoleSet{
		ID:    uuid.New(),
		Name:  "Admin",
		Roles: []domain.Role{roleAdmin, roleRead},
	}

	emptyRoleSet := domain.RoleSet{
		ID:    uuid.New(),
		Name:  "Empty",
		Roles: []domain.Role{},
	}

	tests := []struct {
		name       string
		user       *domain.User
		permission string
		want       bool
	}{
		{
			name: "exact permission match in single role set",
			user: &domain.User{
				ID:       uuid.New(),
				Username: "reader",
				Groups: []domain.Group{
					{ID: uuid.New(), Name: "Readers", RoleSets: []domain.RoleSet{readRoleSet}},
				},
			},
			permission: "perm-read",
			want:       true,
		},
		{
			name: "permission denied when role not in any role set",
			user: &domain.User{
				ID:       uuid.New(),
				Username: "reader",
				Groups: []domain.Group{
					{ID: uuid.New(), Name: "Readers", RoleSets: []domain.RoleSet{readRoleSet}},
				},
			},
			permission: "perm-admin",
			want:       false,
		},
		{
			name: "permission from second role set via shared group",
			user: &domain.User{
				ID:       uuid.New(),
				Username: "writer",
				Groups: []domain.Group{
					{ID: uuid.New(), Name: "Writers", RoleSets: []domain.RoleSet{readRoleSet, writeRoleSet}},
				},
			},
			permission: "perm-write",
			want:       true,
		},
		{
			name: "permission from second group",
			user: &domain.User{
				ID:       uuid.New(),
				Username: "multi-group",
				Groups: []domain.Group{
					{ID: uuid.New(), Name: "Readers", RoleSets: []domain.RoleSet{readRoleSet}},
					{ID: uuid.New(), Name: "Admins", RoleSets: []domain.RoleSet{adminRoleSet}},
				},
			},
			permission: "perm-admin",
			want:       true,
		},
		{
			name: "empty groups returns false",
			user: &domain.User{
				ID:       uuid.New(),
				Username: "nobody",
				Groups:   []domain.Group{},
			},
			permission: "perm-read",
			want:       false,
		},
		{
			name: "empty role set returns false",
			user: &domain.User{
				ID:       uuid.New(),
				Username: "empty",
				Groups: []domain.Group{
					{ID: uuid.New(), Name: "EmptyGroup", RoleSets: []domain.RoleSet{emptyRoleSet}},
				},
			},
			permission: "perm-read",
			want:       false,
		},
		{
			name: "permission shared across role sets (read is in both Reader and Admin)",
			user: &domain.User{
				ID:       uuid.New(),
				Username: "multi-role",
				Groups: []domain.Group{
					{ID: uuid.New(), Name: "Admins", RoleSets: []domain.RoleSet{readRoleSet, adminRoleSet}},
				},
			},
			permission: "perm-read",
			want:       true,
		},
		{
			name: "permission denied for empty permission string",
			user: &domain.User{
				ID:       uuid.New(),
				Username: "reader",
				Groups: []domain.Group{
					{ID: uuid.New(), Name: "Readers", RoleSets: []domain.RoleSet{readRoleSet}},
				},
			},
			permission: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.user.HasPermission(tt.permission)
			if got != tt.want {
				t.Errorf("User.HasPermission(%q) = %v, want %v", tt.permission, got, tt.want)
			}
		})
	}
}

func hashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return string(hash)
}

func TestUser_VerifyPassword(t *testing.T) {
	correctPassword := "secret123"
	hash := hashPassword(t, correctPassword)

	user := &domain.User{
		ID:           uuid.New(),
		Username:     "testuser",
		PasswordHash: hash,
	}

	t.Run("correct password returns true", func(t *testing.T) {
		if !user.VerifyPassword(correctPassword) {
			t.Error("expected VerifyPassword to return true for correct password")
		}
	})

	t.Run("wrong password returns false", func(t *testing.T) {
		if user.VerifyPassword("wrongpassword") {
			t.Error("expected VerifyPassword to return false for wrong password")
		}
	})

	t.Run("empty password returns false", func(t *testing.T) {
		if user.VerifyPassword("") {
			t.Error("expected VerifyPassword to return false for empty password")
		}
	})

	t.Run("malformed hash returns false", func(t *testing.T) {
		badUser := &domain.User{
			ID:           uuid.New(),
			Username:     "badhash",
			PasswordHash: "not-a-valid-bcrypt-hash",
		}
		if badUser.VerifyPassword("anything") {
			t.Error("expected VerifyPassword to return false for malformed hash")
		}
	})
}
