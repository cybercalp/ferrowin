package domain

import "github.com/google/uuid"

// RoleSet represents a collection of permissions (roles) mapped to security groups.
type RoleSet struct {
	ID    uuid.UUID
	Name  string
	Roles []Role
}
