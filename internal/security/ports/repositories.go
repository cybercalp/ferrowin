package ports

import (
	"context"
	"ferrowin/internal/security/domain"

	"github.com/google/uuid"
)

// UserRepository defines the contract for user persistence.
type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	Save(ctx context.Context, user *domain.User) error
}

// GroupRepository defines the contract for security group persistence.
type GroupRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Group, error)
	Save(ctx context.Context, group *domain.Group) error
	AssignGroupToUser(ctx context.Context, userID uuid.UUID, groupID uuid.UUID) error
}

// RoleSetRepository defines the contract for role set persistence.
type RoleSetRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.RoleSet, error)
	Save(ctx context.Context, roleSet *domain.RoleSet) error
	AssignRoleSetToGroup(ctx context.Context, groupID uuid.UUID, roleSetID uuid.UUID) error
}

// RoleRepository defines the contract for permission role persistence.
type RoleRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error)
	Save(ctx context.Context, role *domain.Role) error
	AssignRoleToRoleSet(ctx context.Context, roleSetID uuid.UUID, roleID uuid.UUID) error
}

// AuthService defines the contract for checking user permissions.
type AuthService interface {
	HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error)
}
