package adapters

import (
	"context"
	"database/sql"
	"errors"
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
// SQLTransferRepository implements domain.TransferRepository.
type SQLTransferRepository struct {
	db       *sql.DB
	isSQLite bool
}

// NewSQLTransferRepository creates a new SQLTransferRepository.
func NewSQLTransferRepository(db *sql.DB, isSQLite bool) *SQLTransferRepository {
	return &SQLTransferRepository{
		db:       db,
		isSQLite: isSQLite,
	}
}

func (r *SQLTransferRepository) adaptSQL(sqlite, postgres string) string {
	if r.isSQLite {
		return sqlite
	}
	return postgres
}

// Save persists a transfer header (upsert).
func (r *SQLTransferRepository) Save(ctx context.Context, t *domain.TraspasoAlmacen) error {
	var processedAt, cancelledAt interface{}
	if t.ProcessedAt != nil {
		processedAt = t.ProcessedAt.UTC()
	}
	if t.CancelledAt != nil {
		cancelledAt = t.CancelledAt.UTC()
	}

	query := r.adaptSQL(
		`INSERT INTO traspasos_almacen (id, empresa_id, origen_id, destino_id, estado, created_at, processed_at, cancelled_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET estado=excluded.estado, processed_at=excluded.processed_at, cancelled_at=excluded.cancelled_at`,
		`INSERT INTO traspasos_almacen (id, empresa_id, origen_id, destino_id, estado, created_at, processed_at, cancelled_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT(id) DO UPDATE SET estado=EXCLUDED.estado, processed_at=EXCLUDED.processed_at, cancelled_at=EXCLUDED.cancelled_at`,
	)

	_, err := r.db.ExecContext(ctx, query,
		t.ID.String(),
		t.EmpresaID.String(),
		t.OrigenID.String(),
		t.DestinoID.String(),
		string(t.Estado),
		t.CreatedAt.UTC(),
		processedAt,
		cancelledAt,
	)
	return err
}

// GetByID retrieves a transfer header with its lines.
func (r *SQLTransferRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.TraspasoAlmacen, error) {
	qHeader := r.adaptSQL(
		`SELECT id, empresa_id, origen_id, destino_id, estado, created_at, processed_at, cancelled_at FROM traspasos_almacen WHERE id = ?`,
		`SELECT id, empresa_id, origen_id, destino_id, estado, created_at, processed_at, cancelled_at FROM traspasos_almacen WHERE id = $1`,
	)

	var idStr, empIDStr, origenIDStr, destinoIDStr, estadoStr, createdAtStr string
	var processedAtStr, cancelledAtStr sql.NullString

	err := r.db.QueryRowContext(ctx, qHeader, id.String()).Scan(
		&idStr, &empIDStr, &origenIDStr, &destinoIDStr, &estadoStr, &createdAtStr,
		&processedAtStr, &cancelledAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTransferNotFound
		}
		return nil, err
	}

	createdAt, err := parseTransferTime(createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at %q: %w", createdAtStr, err)
	}

	var processedAt *time.Time
	if processedAtStr.Valid && processedAtStr.String != "" {
		t, err := parseTransferTime(processedAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse processed_at %q: %w", processedAtStr.String, err)
		}
		processedAt = &t
	}

	var cancelledAt *time.Time
	if cancelledAtStr.Valid && cancelledAtStr.String != "" {
		t, err := parseTransferTime(cancelledAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cancelled_at %q: %w", cancelledAtStr.String, err)
		}
		cancelledAt = &t
	}

	// Fetch lines
	qLines := r.adaptSQL(
		`SELECT id, traspaso_almacen_id, producto_id, cantidad FROM traspaso_almacen_lineas WHERE traspaso_almacen_id = ?`,
		`SELECT id, traspaso_almacen_id, producto_id, cantidad FROM traspaso_almacen_lineas WHERE traspaso_almacen_id = $1`,
	)

	rows, err := r.db.QueryContext(ctx, qLines, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lineas []domain.TraspasoAlmacenLinea
	for rows.Next() {
		var lIDStr, tIDStr, prodIDStr string
		var qty float64
		if err := rows.Scan(&lIDStr, &tIDStr, &prodIDStr, &qty); err != nil {
			return nil, err
		}
		lUUID, _ := uuid.Parse(lIDStr)
		tUUID, _ := uuid.Parse(tIDStr)
		pUUID, _ := uuid.Parse(prodIDStr)
		lineas = append(lineas, domain.TraspasoAlmacenLinea{
			ID:                lUUID,
			TraspasoAlmacenID: tUUID,
			ProductoID:        pUUID,
			Cantidad:          qty,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tUUID, _ := uuid.Parse(idStr)
	empUUID, _ := uuid.Parse(empIDStr)
	origenUUID, _ := uuid.Parse(origenIDStr)
	destinoUUID, _ := uuid.Parse(destinoIDStr)

	return &domain.TraspasoAlmacen{
		ID:          tUUID,
		EmpresaID:   empUUID,
		OrigenID:    origenUUID,
		DestinoID:   destinoUUID,
		Estado:      domain.TraspasoAlmacenEstado(estadoStr),
		CreatedAt:   createdAt,
		ProcessedAt: processedAt,
		CancelledAt: cancelledAt,
		Lineas:      lineas,
	}, nil
}

// List returns paginated transfers matching the filter.
func (r *SQLTransferRepository) List(ctx context.Context, filter domain.TransferFilter) ([]*domain.TraspasoAlmacen, int, error) {
	// Build WHERE clause dynamically
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if filter.EmpresaID != nil {
		where += fmt.Sprintf(" AND empresa_id = %s", r.paramPlaceholder(argIdx))
		args = append(args, filter.EmpresaID.String())
		argIdx++
	}
	if filter.OrigenID != nil {
		where += fmt.Sprintf(" AND origen_id = %s", r.paramPlaceholder(argIdx))
		args = append(args, filter.OrigenID.String())
		argIdx++
	}
	if filter.DestinoID != nil {
		where += fmt.Sprintf(" AND destino_id = %s", r.paramPlaceholder(argIdx))
		args = append(args, filter.DestinoID.String())
		argIdx++
	}
	if filter.Estado != nil {
		where += fmt.Sprintf(" AND estado = %s", r.paramPlaceholder(argIdx))
		args = append(args, string(*filter.Estado))
		argIdx++
	}
	if filter.Desde != nil {
		where += fmt.Sprintf(" AND created_at >= %s", r.paramPlaceholder(argIdx))
		args = append(args, filter.Desde.UTC())
		argIdx++
	}
	if filter.Hasta != nil {
		where += fmt.Sprintf(" AND created_at <= %s", r.paramPlaceholder(argIdx))
		args = append(args, filter.Hasta.UTC())
		argIdx++
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	} else if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	// Count query
	countQuery := "SELECT COUNT(*) FROM traspasos_almacen " + where
	var total int
	err := r.db.QueryRowContext(ctx, r.adaptSQL(countQuery, countQuery), args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Data query
	dataQuery := "SELECT id, empresa_id, origen_id, destino_id, estado, created_at, processed_at, cancelled_at FROM traspasos_almacen " +
		where + " ORDER BY created_at DESC"

	if r.isSQLite {
		dataQuery += fmt.Sprintf(" LIMIT ? OFFSET ?")
	} else {
		dataQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	}
	dataArgs := append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, r.adaptSQL(dataQuery, dataQuery), dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var transfers []*domain.TraspasoAlmacen
	for rows.Next() {
		var idStr, empIDStr, origenIDStr, destinoIDStr, estadoStr, createdAtStr string
		var processedAtStr, cancelledAtStr sql.NullString

		err := rows.Scan(&idStr, &empIDStr, &origenIDStr, &destinoIDStr, &estadoStr, &createdAtStr,
			&processedAtStr, &cancelledAtStr)
		if err != nil {
			return nil, 0, err
		}

		createdAt, err := parseTransferTime(createdAtStr)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to parse created_at %q: %w", createdAtStr, err)
		}

		var processedAt *time.Time
		if processedAtStr.Valid && processedAtStr.String != "" {
			t, err := parseTransferTime(processedAtStr.String)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to parse processed_at %q: %w", processedAtStr.String, err)
			}
			processedAt = &t
		}

		var cancelledAt *time.Time
		if cancelledAtStr.Valid && cancelledAtStr.String != "" {
			t, err := parseTransferTime(cancelledAtStr.String)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to parse cancelled_at %q: %w", cancelledAtStr.String, err)
			}
			cancelledAt = &t
		}

		tUUID, _ := uuid.Parse(idStr)
		empUUID, _ := uuid.Parse(empIDStr)
		origenUUID, _ := uuid.Parse(origenIDStr)
		destinoUUID, _ := uuid.Parse(destinoIDStr)

		transfers = append(transfers, &domain.TraspasoAlmacen{
			ID:          tUUID,
			EmpresaID:   empUUID,
			OrigenID:    origenUUID,
			DestinoID:   destinoUUID,
			Estado:      domain.TraspasoAlmacenEstado(estadoStr),
			CreatedAt:   createdAt,
			ProcessedAt: processedAt,
			CancelledAt: cancelledAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return transfers, total, nil
}

func (r *SQLTransferRepository) paramPlaceholder(idx int) string {
	if r.isSQLite {
		return "?"
	}
	return fmt.Sprintf("$%d", idx)
}

// AddLine inserts a new transfer line.
func (r *SQLTransferRepository) AddLine(ctx context.Context, line *domain.TraspasoAlmacenLinea) error {
	query := r.adaptSQL(
		`INSERT INTO traspaso_almacen_lineas (id, traspaso_almacen_id, producto_id, cantidad) VALUES (?, ?, ?, ?)`,
		`INSERT INTO traspaso_almacen_lineas (id, traspaso_almacen_id, producto_id, cantidad) VALUES ($1, $2, $3, $4)`,
	)
	_, err := r.db.ExecContext(ctx, query,
		line.ID.String(),
		line.TraspasoAlmacenID.String(),
		line.ProductoID.String(),
		line.Cantidad,
	)
	return err
}

// RemoveLine deletes a transfer line by its ID.
func (r *SQLTransferRepository) RemoveLine(ctx context.Context, lineID uuid.UUID) error {
	query := r.adaptSQL(
		`DELETE FROM traspaso_almacen_lineas WHERE id = ?`,
		`DELETE FROM traspaso_almacen_lineas WHERE id = $1`,
	)
	_, err := r.db.ExecContext(ctx, query, lineID.String())
	return err
}

// ProcessTransfer atomically updates the transfer estado and inserts stock ledger entries
// in a single database transaction.
func (r *SQLTransferRepository) ProcessTransfer(ctx context.Context, t *domain.TraspasoAlmacen, entries []*domain.StockLedgerEntry) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Idempotency guard WITH row lock
	var currentEstado string
	err = tx.QueryRowContext(ctx, r.adaptSQL(
		`SELECT estado FROM traspasos_almacen WHERE id = ?`,
		`SELECT estado FROM traspasos_almacen WHERE id = $1 FOR UPDATE`,
	), t.ID.String()).Scan(&currentEstado)
	if err != nil {
		return err
	}
	if currentEstado != "Borrador" {
		return domain.ErrTransferAlreadyProcessed
	}

	// 2. Update estado + timestamps
	var processedAt interface{}
	if t.ProcessedAt != nil {
		processedAt = t.ProcessedAt.UTC()
	}
	_, err = tx.ExecContext(ctx, r.adaptSQL(
		`UPDATE traspasos_almacen SET estado = ?, processed_at = ? WHERE id = ?`,
		`UPDATE traspasos_almacen SET estado = $1, processed_at = $2 WHERE id = $3`,
	), string(t.Estado), processedAt, t.ID.String())
	if err != nil {
		return fmt.Errorf("failed to update transfer estado: %w", err)
	}

	// 3. Insert stock ledger entries (same tx, direct SQL)
	for _, e := range entries {
		var refDocType, refDocID interface{}
		if e.ReferenceDocumentType != nil {
			refDocType = *e.ReferenceDocumentType
		}
		if e.ReferenceDocumentID != nil {
			refDocID = e.ReferenceDocumentID.String()
		}

		_, err = tx.ExecContext(ctx, r.adaptSQL(
			`INSERT INTO stock_ledger_movements (id, item_id, warehouse_id, quantity, movement_type, reference_document_type, reference_document_id, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			`INSERT INTO stock_ledger_movements (id, item_id, warehouse_id, quantity, movement_type, reference_document_type, reference_document_id, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		), e.ID.String(), e.ItemID.String(), e.WarehouseID.String(),
			e.Quantity, string(e.MovementType), refDocType, refDocID, e.CreatedAt.UTC())
		if err != nil {
			return fmt.Errorf("failed to save ledger entry for product %s: %w", e.ItemID, err)
		}
	}

	return tx.Commit()
}

// parseTransferTime attempts to parse a timestamp string using multiple layouts,
// matching the format produced by both SQLite and PostgreSQL drivers.
func parseTransferTime(s string) (time.Time, error) {
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05 -0700 MST",
	} {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("failed to parse timestamp %q", s)
}

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
