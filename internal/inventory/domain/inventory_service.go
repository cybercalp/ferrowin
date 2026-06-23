package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrInsufficientStock is returned when a withdrawal is requested but available stock is low.
	ErrInsufficientStock = errors.New("insufficient stock for withdrawal")
	// ErrInvalidQuantity is returned when quantity is less than or equal to zero.
	ErrInvalidQuantity = errors.New("quantity must be greater than zero")
)

// StockLedgerRepository defines the contract for stock ledger persistence.
type StockLedgerRepository interface {
	Save(ctx context.Context, entry *StockLedgerEntry) error
	GetMovements(ctx context.Context, itemID, warehouseID uuid.UUID) ([]*StockLedgerEntry, error)
}

// InventoryService provides domain operations for managing stock movements and FIFO reconciliation.
type InventoryService struct {
	repo StockLedgerRepository
	Now  func() time.Time
}

// NewInventoryService creates a new InventoryService instance.
func NewInventoryService(repo StockLedgerRepository) *InventoryService {
	return &InventoryService{
		repo: repo,
	}
}

func (s *InventoryService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// GetAvailableStock calculates the net stock balance for a given item in a specific warehouse.
func (s *InventoryService) GetAvailableStock(ctx context.Context, itemID, warehouseID uuid.UUID) (float64, error) {
	movements, err := s.repo.GetMovements(ctx, itemID, warehouseID)
	if err != nil {
		return 0, err
	}
	var total float64
	for _, m := range movements {
		total += m.Quantity
	}
	return total, nil
}

// RecordReceipt records a stock receipt (addition of stock).
func (s *InventoryService) RecordReceipt(ctx context.Context, itemID, warehouseID uuid.UUID, qty float64, refDocType *string, refDocID *uuid.UUID) (*StockLedgerEntry, error) {
	if qty <= 0 {
		return nil, ErrInvalidQuantity
	}
	entry := &StockLedgerEntry{
		ID:                    uuid.New(),
		ItemID:                itemID,
		WarehouseID:           warehouseID,
		Quantity:              qty,
		MovementType:          MovementTypeReceipt,
		ReferenceDocumentType: refDocType,
		ReferenceDocumentID:   refDocID,
		CreatedAt:             s.now(),
	}
	if err := s.repo.Save(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// RecordWithdrawal records a stock withdrawal (deduction of stock).
// It blocks the transaction and returns ErrInsufficientStock if the available stock is less than the requested quantity.
func (s *InventoryService) RecordWithdrawal(ctx context.Context, itemID, warehouseID uuid.UUID, qty float64, refDocType *string, refDocID *uuid.UUID) (*StockLedgerEntry, error) {
	if qty <= 0 {
		return nil, ErrInvalidQuantity
	}
	available, err := s.GetAvailableStock(ctx, itemID, warehouseID)
	if err != nil {
		return nil, err
	}
	if available < qty {
		return nil, ErrInsufficientStock
	}

	entry := &StockLedgerEntry{
		ID:                    uuid.New(),
		ItemID:                itemID,
		WarehouseID:           warehouseID,
		Quantity:              -qty,
		MovementType:          MovementTypeWithdrawal,
		ReferenceDocumentType: refDocType,
		ReferenceDocumentID:   refDocID,
		CreatedAt:             s.now(),
	}
	if err := s.repo.Save(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// RecordReturn records a stock return (addition of stock from a credit note / returns).
func (s *InventoryService) RecordReturn(ctx context.Context, itemID, warehouseID uuid.UUID, qty float64, refDocType *string, refDocID *uuid.UUID) (*StockLedgerEntry, error) {
	if qty <= 0 {
		return nil, ErrInvalidQuantity
	}
	entry := &StockLedgerEntry{
		ID:                    uuid.New(),
		ItemID:                itemID,
		WarehouseID:           warehouseID,
		Quantity:              qty,
		MovementType:          MovementTypeReturn,
		ReferenceDocumentType: refDocType,
		ReferenceDocumentID:   refDocID,
		CreatedAt:             s.now(),
	}
	if err := s.repo.Save(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// WithdrawalItem represents a single item for batch withdrawal processing.
type WithdrawalItem struct {
	ItemID      uuid.UUID
	WarehouseID uuid.UUID
	Qty         float64
}

// RecordWithdrawals performs multiple stock withdrawals atomically.
// On partial failure, it compensates by recording returns for all successful entries.
// On success, it returns all recorded entries.
func (s *InventoryService) RecordWithdrawals(ctx context.Context, items []WithdrawalItem, refDocType *string, refDocID *uuid.UUID) ([]*StockLedgerEntry, error) {
	var successful []*StockLedgerEntry
	for _, item := range items {
		entry, err := s.RecordWithdrawal(ctx, item.ItemID, item.WarehouseID, item.Qty, refDocType, refDocID)
		if err != nil {
			// Compensate: roll back all successful withdrawals
			for _, e := range successful {
				_, _ = s.RecordReturn(ctx, e.ItemID, e.WarehouseID, -e.Quantity, e.ReferenceDocumentType, e.ReferenceDocumentID)
			}
			return nil, fmt.Errorf("failed to deduct stock for product %s: %w", item.ItemID, err)
		}
		successful = append(successful, entry)
	}
	return successful, nil
}

// RecordSyncAdjustment records an offline TPV sales sync adjustment (deduction of stock).
// It bypasses the stock check, allowing the stock balance to go negative.
func (s *InventoryService) RecordSyncAdjustment(ctx context.Context, itemID, warehouseID uuid.UUID, qty float64, refDocType *string, refDocID *uuid.UUID) (*StockLedgerEntry, error) {
	if qty <= 0 {
		return nil, ErrInvalidQuantity
	}
	entry := &StockLedgerEntry{
		ID:                    uuid.New(),
		ItemID:                itemID,
		WarehouseID:           warehouseID,
		Quantity:              -qty,
		MovementType:          MovementTypeSyncAdjustment,
		ReferenceDocumentType: refDocType,
		ReferenceDocumentID:   refDocID,
		CreatedAt:             s.now(),
	}
	if err := s.repo.Save(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// ReconcileFIFO simulates FIFO allocations for the given item and warehouse, matching demands to receipts.
func (s *InventoryService) ReconcileFIFO(ctx context.Context, itemID, warehouseID uuid.UUID) ([]*FIFOAllocation, error) {
	movements, err := s.repo.GetMovements(ctx, itemID, warehouseID)
	if err != nil {
		return nil, err
	}

	type batch struct {
		id        uuid.UUID
		remaining float64
	}
	type demand struct {
		id        uuid.UUID
		mType     MovementType
		remaining float64
	}

	var batches []*batch
	var demands []*demand

	for _, m := range movements {
		if m.MovementType == MovementTypeReceipt {
			batches = append(batches, &batch{
				id:        m.ID,
				remaining: m.Quantity,
			})
		} else {
			demands = append(demands, &demand{
				id:        m.ID,
				mType:     m.MovementType,
				remaining: -m.Quantity,
			})
		}
	}

	var allocations []*FIFOAllocation
	for _, d := range demands {
		for _, b := range batches {
			if b.remaining <= 0 {
				continue
			}
			if d.remaining <= 0 {
				break
			}

			allocated := d.remaining
			if b.remaining < allocated {
				allocated = b.remaining
			}

			b.remaining -= allocated
			d.remaining -= allocated

			allocations = append(allocations, &FIFOAllocation{
				DemandID:     d.id,
				DemandType:   d.mType,
				ReceiptID:    b.id,
				QtyAllocated: allocated,
			})
		}
	}

	return allocations, nil
}
