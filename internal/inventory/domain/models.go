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
	MovementTypeTransfer       MovementType = "TRANSFER"
	MovementTypeReturn         MovementType = "RETURN"
)

// TraspasoAlmacenEstado defines the state of a warehouse transfer document.
type TraspasoAlmacenEstado string

const (
	TraspasoBorrador  TraspasoAlmacenEstado = "Borrador"
	TraspasoProcesado TraspasoAlmacenEstado = "Procesado"
	TraspasoCancelado TraspasoAlmacenEstado = "Cancelado"
)

// TraspasoAlmacen represents a warehouse transfer document.
type TraspasoAlmacen struct {
	ID          uuid.UUID              `json:"id"`
	EmpresaID   uuid.UUID              `json:"empresa_id"`
	OrigenID    uuid.UUID              `json:"origen_id"`
	DestinoID   uuid.UUID              `json:"destino_id"`
	Estado      TraspasoAlmacenEstado  `json:"estado"`
	CreatedAt   time.Time              `json:"created_at"`
	ProcessedAt *time.Time             `json:"processed_at,omitempty"`
	CancelledAt *time.Time             `json:"cancelled_at,omitempty"`
	Lineas      []TraspasoAlmacenLinea `json:"lineas,omitempty"`
}

// TraspasoAlmacenLinea represents a single line in a warehouse transfer.
type TraspasoAlmacenLinea struct {
	ID                uuid.UUID `json:"id"`
	TraspasoAlmacenID uuid.UUID `json:"traspaso_almacen_id"`
	ProductoID        uuid.UUID `json:"producto_id"`
	Cantidad          float64   `json:"cantidad"`
}

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
