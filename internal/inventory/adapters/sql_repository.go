package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"ferrowin/internal/inventory/domain"

	"github.com/google/uuid"
)

type txKey struct{}

// WithTx returns a new context containing the transaction.
func WithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

type dbExecutor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func (r *SQLStockLedgerRepository) getExecutor(ctx context.Context) dbExecutor {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return r.db
}

// SQLStockLedgerRepository implements domain.StockLedgerRepository.
type SQLStockLedgerRepository struct {
	db       *sql.DB
	isSQLite bool
}

// NewSQLStockLedgerRepository creates a new SQLStockLedgerRepository.
func NewSQLStockLedgerRepository(db *sql.DB, isSQLite bool) *SQLStockLedgerRepository {
	return &SQLStockLedgerRepository{
		db:       db,
		isSQLite: isSQLite,
	}
}

// Save persists a new StockLedgerEntry.
func (r *SQLStockLedgerRepository) Save(ctx context.Context, entry *domain.StockLedgerEntry) error {
	var query string
	var refDocType interface{}
	if entry.ReferenceDocumentType != nil {
		refDocType = *entry.ReferenceDocumentType
	} else {
		refDocType = nil
	}

	var refDocID interface{}
	if entry.ReferenceDocumentID != nil {
		refDocID = entry.ReferenceDocumentID.String()
	} else {
		refDocID = nil
	}

	if r.isSQLite {
		query = `INSERT INTO stock_ledger_movements (id, item_id, warehouse_id, quantity, movement_type, reference_document_type, reference_document_id, created_at) 
                 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	} else {
		query = `INSERT INTO stock_ledger_movements (id, item_id, warehouse_id, quantity, movement_type, reference_document_type, reference_document_id, created_at) 
                 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	}

	_, err := r.getExecutor(ctx).ExecContext(ctx, query,
		entry.ID.String(),
		entry.ItemID.String(),
		entry.WarehouseID.String(),
		entry.Quantity,
		string(entry.MovementType),
		refDocType,
		refDocID,
		entry.CreatedAt.UTC(),
	)
	return err
}

// GetMovements retrieves all movements for an item in a specific warehouse ordered by creation time ascending.
func (r *SQLStockLedgerRepository) GetMovements(ctx context.Context, itemID, warehouseID uuid.UUID) ([]*domain.StockLedgerEntry, error) {
	var query string
	if r.isSQLite {
		query = `SELECT id, item_id, warehouse_id, quantity, movement_type, reference_document_type, reference_document_id, created_at 
                 FROM stock_ledger_movements 
                 WHERE item_id = ? AND warehouse_id = ? 
                 ORDER BY created_at ASC`
	} else {
		query = `SELECT id, item_id, warehouse_id, quantity, movement_type, reference_document_type, reference_document_id, created_at 
                 FROM stock_ledger_movements 
                 WHERE item_id = $1 AND warehouse_id = $2 
                 ORDER BY created_at ASC`
	}

	rows, err := r.getExecutor(ctx).QueryContext(ctx, query, itemID.String(), warehouseID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*domain.StockLedgerEntry
	for rows.Next() {
		var idStr, itemIDStr, warehouseIDStr, mTypeStr, createdAtStr string
		var qty float64
		var refDocType sql.NullString
		var refDocIDStr sql.NullString

		err := rows.Scan(&idStr, &itemIDStr, &warehouseIDStr, &qty, &mTypeStr, &refDocType, &refDocIDStr, &createdAtStr)
		if err != nil {
			return nil, err
		}

		var createdAt time.Time
		for _, layout := range []string{
			time.RFC3339Nano,
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05.999999999 -0700 MST",
			"2006-01-02 15:04:05 -0700 MST",
		} {
			createdAt, err = time.Parse(layout, createdAtStr)
			if err == nil {
				break
			}
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at %q: %w", createdAtStr, err)
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, err
		}
		itemUUID, err := uuid.Parse(itemIDStr)
		if err != nil {
			return nil, err
		}
		whUUID, err := uuid.Parse(warehouseIDStr)
		if err != nil {
			return nil, err
		}

		var docTypePtr *string
		if refDocType.Valid {
			v := refDocType.String
			docTypePtr = &v
		}

		var docIDPtr *uuid.UUID
		if refDocIDStr.Valid && refDocIDStr.String != "" {
			dUUID, err := uuid.Parse(refDocIDStr.String)
			if err == nil {
				docIDPtr = &dUUID
			}
		}

		entries = append(entries, &domain.StockLedgerEntry{
			ID:                    id,
			ItemID:                itemUUID,
			WarehouseID:           whUUID,
			Quantity:              qty,
			MovementType:          domain.MovementType(mTypeStr),
			ReferenceDocumentType: docTypePtr,
			ReferenceDocumentID:   docIDPtr,
			CreatedAt:             createdAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}
