package domain

import "github.com/google/uuid"

// Group represents a security group which aggregates users and links them to role sets.
type Group struct {
	ID       uuid.UUID
	Name     string
	RoleSets []RoleSet
}
