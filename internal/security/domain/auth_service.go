package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserRepositoryRequired defines the interface needed by the domain AuthService.
// This decouples the domain layer from the ports package to prevent import cycles.
type UserRepositoryRequired interface {
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}

type authService struct {
	userRepo UserRepositoryRequired
}

// NewAuthService creates a new instance of the authorization service.
func NewAuthService(userRepo UserRepositoryRequired) *authService {
	return &authService{
		userRepo: userRepo,
	}
}

// HasPermission verifies if a user with the given ID has the specified permission.
func (s *authService) HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, nil
	}

	return user.HasPermission(permission), nil
}

// HasPermission checks if the User has a specific permission by traversing the RBAC hierarchy:
// User -> Groups -> Role Sets -> Roles
func (u *User) HasPermission(permission string) bool {
	for _, group := range u.Groups {
		for _, roleSet := range group.RoleSets {
			for _, role := range roleSet.Roles {
				if role.Name == permission {
					return true
				}
			}
		}
	}
	return false
}
