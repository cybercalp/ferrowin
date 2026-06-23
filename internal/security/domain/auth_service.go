package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ErrInvalidCredentials is returned when login fails due to bad username or password.
// It uses a generic message to avoid revealing which credential was wrong.
var ErrInvalidCredentials = errors.New("Invalid credentials")

// UserRepositoryRequired defines the interface needed by the domain AuthService.
// This decouples the domain layer from the ports package to prevent import cycles.
type UserRepositoryRequired interface {
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
}

// LoginResponse represents the response returned after a successful login.
type LoginResponse struct {
	Token string        `json:"token"`
	User  *UserResponse `json:"user"`
}

// UserResponse represents the public user info returned in login responses.
type UserResponse struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
}

type authService struct {
	userRepo UserRepositoryRequired
	jwtCfg   *JWTConfig
}

// NewAuthService creates a new instance of the authorization service.
func NewAuthService(userRepo UserRepositoryRequired, jwtCfg *JWTConfig) *authService {
	return &authService{
		userRepo: userRepo,
		jwtCfg:   jwtCfg,
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

// Login authenticates a user by username and password, returning a signed JWT token
// on success. It returns ErrInvalidCredentials for all authentication failures,
// without distinguishing between unknown users, wrong passwords, or empty inputs.
func (s *authService) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil || user == nil {
		return nil, ErrInvalidCredentials
	}

	if !user.VerifyPassword(password) {
		return nil, ErrInvalidCredentials
	}

	// Collect role IDs for the token
	var roleIDs []string
	for _, group := range user.Groups {
		for _, roleSet := range group.RoleSets {
			for _, role := range roleSet.Roles {
				roleIDs = append(roleIDs, role.ID.String())
			}
		}
	}

	token, err := s.jwtCfg.GenerateToken(user.ID.String(), user.Username, roleIDs)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	return &LoginResponse{
		Token: token,
		User: &UserResponse{
			ID:       user.ID,
			Username: user.Username,
		},
	}, nil
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
