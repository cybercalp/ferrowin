package domain

import "github.com/google/uuid"

// InvoicingSeries represents a billing series with its prefix and sequence counter.
type InvoicingSeries struct {
	ID           uuid.UUID
	TerminalID   uuid.UUID
	Prefix       string
	NextSequence int
}
