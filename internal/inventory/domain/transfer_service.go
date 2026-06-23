package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Domain errors for warehouse transfers.
var (
	ErrTransferSameWarehouse       = errors.New("origen and destino must be different")
	ErrTransferCrossCompany        = errors.New("warehouses must belong to the same company")
	ErrTransferAlreadyProcessed    = errors.New("transfer is already processed")
	ErrTransferNotEditable         = errors.New("transfer is not editable in current state")
	ErrTransferNotFound            = errors.New("transfer not found")
	ErrTransferNoLines             = errors.New("cannot process transfer with no lines")
)

// WarehouseView is a minimal warehouse projection used to validate transfers
// without importing the purchases package (avoids circular dependency).
type WarehouseView struct {
	ID        uuid.UUID
	EmpresaID uuid.UUID
}

// WarehouseValidator allows the TransferService to validate warehouses
// without depending on the purchases module directly.
type WarehouseValidator interface {
	GetWarehouse(ctx context.Context, id uuid.UUID) (*WarehouseView, error)
}

// TransferFilter holds optional filter fields for listing transfers.
type TransferFilter struct {
	EmpresaID *uuid.UUID
	OrigenID  *uuid.UUID
	DestinoID *uuid.UUID
	Estado    *TraspasoAlmacenEstado
	Desde     *time.Time
	Hasta     *time.Time
	Page      int // 1-indexed, default 1
	PageSize  int // default 20, max 100
}

// TransferRepository defines persistence operations for warehouse transfers.
type TransferRepository interface {
	Save(ctx context.Context, t *TraspasoAlmacen) error
	GetByID(ctx context.Context, id uuid.UUID) (*TraspasoAlmacen, error)
	List(ctx context.Context, filter TransferFilter) ([]*TraspasoAlmacen, int, error)
	AddLine(ctx context.Context, line *TraspasoAlmacenLinea) error
	RemoveLine(ctx context.Context, lineID uuid.UUID) error
	ProcessTransfer(ctx context.Context, t *TraspasoAlmacen, entries []*StockLedgerEntry) error
}

// TransferService provides domain operations for warehouse transfers.
type TransferService struct {
	repo        TransferRepository
	whValidator WarehouseValidator
	Now         func() time.Time
}

// NewTransferService creates a new TransferService.
func NewTransferService(repo TransferRepository, whValidator WarehouseValidator) *TransferService {
	return &TransferService{
		repo:        repo,
		whValidator: whValidator,
	}
}

func (s *TransferService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Create creates a new warehouse transfer in Borrador state.
func (s *TransferService) Create(ctx context.Context, empresaID, origenID, destinoID uuid.UUID) (*TraspasoAlmacen, error) {
	if origenID == destinoID {
		return nil, ErrTransferSameWarehouse
	}

	origen, err := s.whValidator.GetWarehouse(ctx, origenID)
	if err != nil {
		return nil, fmt.Errorf("validate origen: %w", err)
	}
	destino, err := s.whValidator.GetWarehouse(ctx, destinoID)
	if err != nil {
		return nil, fmt.Errorf("validate destino: %w", err)
	}

	if origen.EmpresaID != empresaID || destino.EmpresaID != empresaID {
		return nil, ErrTransferCrossCompany
	}
	if origen.EmpresaID != destino.EmpresaID {
		return nil, ErrTransferCrossCompany
	}

	t := &TraspasoAlmacen{
		ID:        uuid.New(),
		EmpresaID: empresaID,
		OrigenID:  origenID,
		DestinoID: destinoID,
		Estado:    TraspasoBorrador,
		CreatedAt: s.now(),
	}

	if err := s.repo.Save(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// GetByID retrieves a transfer by its ID (including lines).
func (s *TransferService) GetByID(ctx context.Context, id uuid.UUID) (*TraspasoAlmacen, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns paginated transfers matching the given filter, scoped to empresaID.
func (s *TransferService) List(ctx context.Context, empresaID uuid.UUID, filter TransferFilter) ([]*TraspasoAlmacen, int, error) {
	filter.EmpresaID = &empresaID
	return s.repo.List(ctx, filter)
}

// AddLine adds a product line to a transfer in Borrador state.
// NOTE: There is a TOCTOU race between the GetByID (estado check) and AddLine insert.
// This is acceptable for v1 because the window is small and ProcessTransfer already
// uses FOR UPDATE as the authoritative guard.
func (s *TransferService) AddLine(ctx context.Context, empresaID, transferID, productoID uuid.UUID, cantidad float64) (*TraspasoAlmacen, error) {
	if cantidad <= 0 {
		return nil, ErrInvalidQuantity
	}

	t, err := s.repo.GetByID(ctx, transferID)
	if err != nil {
		return nil, err
	}
	if t.EmpresaID != empresaID {
		return nil, ErrTransferCrossCompany
	}
	if t.Estado != TraspasoBorrador {
		return nil, ErrTransferNotEditable
	}

	line := &TraspasoAlmacenLinea{
		ID:                uuid.New(),
		TraspasoAlmacenID: transferID,
		ProductoID:        productoID,
		Cantidad:          cantidad,
	}
	if err := s.repo.AddLine(ctx, line); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, transferID)
}

// RemoveLine removes a line from a transfer in Borrador state.
// NOTE: Same TOCTOU consideration as AddLine — acceptable for v1.
func (s *TransferService) RemoveLine(ctx context.Context, empresaID, transferID, lineID uuid.UUID) (*TraspasoAlmacen, error) {
	t, err := s.repo.GetByID(ctx, transferID)
	if err != nil {
		return nil, err
	}
	if t.EmpresaID != empresaID {
		return nil, ErrTransferCrossCompany
	}
	if t.Estado != TraspasoBorrador {
		return nil, ErrTransferNotEditable
	}

	if err := s.repo.RemoveLine(ctx, lineID); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, transferID)
}

// Process atomically processes a transfer: validates estado, builds stock ledger entries
// (withdrawal at origen, receipt at destino), and commits everything in a single transaction.
func (s *TransferService) Process(ctx context.Context, empresaID, transferID uuid.UUID) error {
	t, err := s.repo.GetByID(ctx, transferID)
	if err != nil {
		return err
	}

	if t.EmpresaID != empresaID {
		return ErrTransferCrossCompany
	}

	if t.Estado != TraspasoBorrador {
		return ErrTransferAlreadyProcessed
	}

	if t.OrigenID == t.DestinoID {
		return ErrTransferSameWarehouse
	}

	if len(t.Lineas) == 0 {
		return ErrTransferNoLines
	}

	now := s.now()
	refDocType := "TRANSFER"

	var entries []*StockLedgerEntry
	for _, line := range t.Lineas {
		// Withdrawal at origen (negative qty, no stock check — per F08)
		entries = append(entries, &StockLedgerEntry{
			ID:                    uuid.New(),
			ItemID:                line.ProductoID,
			WarehouseID:           t.OrigenID,
			Quantity:              -line.Cantidad,
			MovementType:          MovementTypeTransfer,
			ReferenceDocumentType: &refDocType,
			ReferenceDocumentID:   &t.ID,
			CreatedAt:             now,
		})

		// Receipt at destino (positive qty)
		entries = append(entries, &StockLedgerEntry{
			ID:                    uuid.New(),
			ItemID:                line.ProductoID,
			WarehouseID:           t.DestinoID,
			Quantity:              line.Cantidad,
			MovementType:          MovementTypeTransfer,
			ReferenceDocumentType: &refDocType,
			ReferenceDocumentID:   &t.ID,
			CreatedAt:             now,
		})
	}

	nowCopy := now
	t.ProcessedAt = &nowCopy
	t.Estado = TraspasoProcesado

	return s.repo.ProcessTransfer(ctx, t, entries)
}
