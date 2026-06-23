package domain

import "github.com/google/uuid"

// Terminal represents a billing terminal in the system.
type Terminal struct {
	ID       uuid.UUID
	Name     string
	IsActive bool
}
