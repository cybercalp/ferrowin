package domain_test

import (
	"testing"

	"ferrowin/internal/security/domain"

	"github.com/google/uuid"
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
