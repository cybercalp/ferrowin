package ports

import (
	"context"
	"ferrowin/internal/billing/domain"

	"github.com/google/uuid"
)

// TerminalRepository defines the contract for terminal persistence.
type TerminalRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Terminal, error)
	List(ctx context.Context) ([]domain.Terminal, error)
	Save(ctx context.Context, terminal *domain.Terminal) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// InvoicingSeriesRepository defines the contract for series persistence and locking.
type InvoicingSeriesRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.InvoicingSeries, error)
	GetByTerminalID(ctx context.Context, terminalID uuid.UUID) (*domain.InvoicingSeries, error)
	List(ctx context.Context) ([]domain.InvoicingSeries, error)
	Save(ctx context.Context, series *domain.InvoicingSeries) error
	Delete(ctx context.Context, id uuid.UUID) error
	// IncrementSequence safely locks the series for the given terminal,
	// increments the sequence, updates the database, and returns the prefix and new sequence number.
	IncrementSequence(ctx context.Context, terminalID uuid.UUID) (string, int, error)
}

// BillingService defines the contract for generating invoice numbers.
type BillingService interface {
	GenerateFacturaNumber(ctx context.Context, terminalID uuid.UUID) (string, int, error)
}
