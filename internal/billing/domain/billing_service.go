package domain

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// InvoicingSeriesRepositoryRequired defines the repository contract required by the domain service.
type InvoicingSeriesRepositoryRequired interface {
	IncrementSequence(ctx context.Context, terminalID uuid.UUID) (string, int, error)
}

type billingService struct {
	repo InvoicingSeriesRepositoryRequired
}

// NewBillingService creates a new instance of the domain billing service.
func NewBillingService(repo InvoicingSeriesRepositoryRequired) *billingService {
	return &billingService{
		repo: repo,
	}
}

// GenerateFacturaNumber increments the terminal sequence and formats the invoice number (e.g. S1-16).
func (s *billingService) GenerateFacturaNumber(ctx context.Context, terminalID uuid.UUID) (string, int, error) {
	prefix, seq, err := s.repo.IncrementSequence(ctx, terminalID)
	if err != nil {
		return "", 0, err
	}
	invoiceNumber := fmt.Sprintf("%s-%d", prefix, seq)
	return invoiceNumber, seq, nil
}
