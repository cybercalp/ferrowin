package domain

import "github.com/google/uuid"

// Role represents a fine-grained permission.
type Role struct {
	ID   uuid.UUID
	Name string
}
