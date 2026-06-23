package domain

import (
	"time"

	"github.com/google/uuid"
)

// MovementType defines the type of stock movement in the ledger.
type MovementType string

const (
	MovementTypeReceipt        MovementType = "RECEIPT"
	MovementTypeWithdrawal     MovementType = "WITHDRAWAL"
	MovementTypeSyncAdjustment MovementType = "SYNC_ADJUSTMENT"
)

// StockLedgerEntry represents a record in the stock ledger.
type StockLedgerEntry struct {
	ID                    uuid.UUID
	ItemID                uuid.UUID
	WarehouseID           uuid.UUID
	Quantity              float64 // Stores positive for receipt, negative for withdrawals/sync adjustments
	MovementType          MovementType
	ReferenceDocumentType *string
	ReferenceDocumentID   *uuid.UUID
	CreatedAt             time.Time
}

// FIFOAllocation represents the association of a demand movement (Withdrawal or SyncAdjustment)
// to a receipt movement (Receipt) under FIFO reconciliation rules.
type FIFOAllocation struct {
	DemandID     uuid.UUID
	DemandType   MovementType
	ReceiptID    uuid.UUID
	QtyAllocated float64
}
