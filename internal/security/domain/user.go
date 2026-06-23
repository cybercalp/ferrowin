package domain

import "github.com/google/uuid"

// User represents a system user in the domain model.
type User struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
	Groups       []Group
}
