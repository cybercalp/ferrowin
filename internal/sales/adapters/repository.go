package adapters

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"ferrowin/internal/sales/domain"
	"github.com/google/uuid"
)

type SQLSalesRepository struct {
	db       *sql.DB
	isSQLite bool
}

func NewSQLSalesRepository(db *sql.DB, isSQLite bool) *SQLSalesRepository {
	return &SQLSalesRepository{
		db:       db,
		isSQLite: isSQLite,
	}
}

func (r *SQLSalesRepository) SaveQuote(ctx context.Context, q *domain.Quote) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var qQuote string
	if r.isSQLite {
		qQuote = `INSERT INTO quote (id, empresa_id, client_id, total, status, expires_at, created_at) 
                  VALUES (?, ?, ?, ?, ?, ?, ?)
                  ON CONFLICT(id) DO UPDATE SET status=excluded.status, total=excluded.total`
	} else {
		qQuote = `INSERT INTO quote (id, empresa_id, client_id, total, status, expires_at, created_at) 
                  VALUES ($1, $2, $3, $4, $5, $6, $7)
                  ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status, total=EXCLUDED.total`
	}

	_, err = tx.ExecContext(ctx, qQuote, q.ID.String(), q.EmpresaID.String(), q.ClientID.String(), q.Total, q.Status, q.ExpiresAt.UTC(), q.CreatedAt.UTC())
	if err != nil {
		return err
	}

	// Delete existing lines if updating
	var qDelete string
	if r.isSQLite {
		qDelete = `DELETE FROM quote_lines WHERE quote_id = ?`
	} else {
		qDelete = `DELETE FROM quote_lines WHERE quote_id = $1`
	}
	_, err = tx.ExecContext(ctx, qDelete, q.ID.String())
	if err != nil {
		return err
	}

	// Insert lines
	var qLine string
	if r.isSQLite {
		qLine = `INSERT INTO quote_lines (id, quote_id, producto_id, cantidad, precio_unitario, coste_unitario) 
                 VALUES (?, ?, ?, ?, ?, ?)`
	} else {
		qLine = `INSERT INTO quote_lines (id, quote_id, producto_id, cantidad, precio_unitario, coste_unitario) 
                 VALUES ($1, $2, $3, $4, $5, $6)`
	}
	for _, l := range q.Lineas {
		_, err = tx.ExecContext(ctx, qLine, l.ID.String(), l.QuoteID.String(), l.ProductoID.String(), l.Cantidad, l.PrecioUnitario, l.CosteUnitario)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SQLSalesRepository) GetQuote(ctx context.Context, id uuid.UUID) (*domain.Quote, error) {
	var qQuote string
	if r.isSQLite {
		qQuote = `SELECT id, empresa_id, client_id, total, status, expires_at, created_at FROM quote WHERE id = ?`
	} else {
		qQuote = `SELECT id, empresa_id, client_id, total, status, expires_at, created_at FROM quote WHERE id = $1`
	}

	var idStr, empIDStr, clientIDStr, status string
	var expiresAt, createdAt time.Time
	var total float64

	err := r.db.QueryRowContext(ctx, qQuote, id.String()).Scan(&idStr, &empIDStr, &clientIDStr, &total, &status, &expiresAt, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrQuoteNotFound
	} else if err != nil {
		return nil, err
	}

	// Fetch lines
	var qLines string
	if r.isSQLite {
		qLines = `SELECT id, quote_id, producto_id, cantidad, precio_unitario, coste_unitario FROM quote_lines WHERE quote_id = ?`
	} else {
		qLines = `SELECT id, quote_id, producto_id, cantidad, precio_unitario, coste_unitario FROM quote_lines WHERE quote_id = $1`
	}
	rows, err := r.db.QueryContext(ctx, qLines, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []domain.QuoteLine
	for rows.Next() {
		var lIDStr, qIDStr, prodIDStr string
		var qty, price, cost float64
		if err := rows.Scan(&lIDStr, &qIDStr, &prodIDStr, &qty, &price, &cost); err != nil {
			return nil, err
		}
		lUUID, _ := uuid.Parse(lIDStr)
		qUUID, _ := uuid.Parse(qIDStr)
		prodUUID, _ := uuid.Parse(prodIDStr)
		lines = append(lines, domain.QuoteLine{
			ID:             lUUID,
			QuoteID:        qUUID,
			ProductoID:     prodUUID,
			Cantidad:       qty,
			PrecioUnitario: price,
			CosteUnitario:  cost,
		})
	}

	qUUID, _ := uuid.Parse(idStr)
	empUUID, _ := uuid.Parse(empIDStr)
	clientUUID, _ := uuid.Parse(clientIDStr)

	return &domain.Quote{
		ID:        qUUID,
		EmpresaID: empUUID,
		ClientID:  clientUUID,
		Total:     total,
		Status:    status,
		ExpiresAt: expiresAt,
		CreatedAt: createdAt,
		Lineas:    lines,
	}, nil
}

func (r *SQLSalesRepository) SaveOrder(ctx context.Context, o *domain.Order) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var qOrder string
	var quoteIDVal interface{}
	if o.QuoteID != nil {
		quoteIDVal = o.QuoteID.String()
	} else {
		quoteIDVal = nil
	}

	if r.isSQLite {
		qOrder = `INSERT INTO "order" (id, empresa_id, quote_id, total, status, created_at) 
                  VALUES (?, ?, ?, ?, ?, ?)
                  ON CONFLICT(id) DO UPDATE SET status=excluded.status, total=excluded.total`
	} else {
		qOrder = `INSERT INTO "order" (id, empresa_id, quote_id, total, status, created_at) 
                  VALUES ($1, $2, $3, $4, $5, $6)
                  ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status, total=EXCLUDED.total`
	}

	_, err = tx.ExecContext(ctx, qOrder, o.ID.String(), o.EmpresaID.String(), quoteIDVal, o.Total, o.Status, o.CreatedAt.UTC())
	if err != nil {
		return err
	}

	// Delete existing lines if updating
	var qDelete string
	if r.isSQLite {
		qDelete = `DELETE FROM order_lines WHERE order_id = ?`
	} else {
		qDelete = `DELETE FROM order_lines WHERE order_id = $1`
	}
	_, err = tx.ExecContext(ctx, qDelete, o.ID.String())
	if err != nil {
		return err
	}

	// Insert lines
	var qLine string
	if r.isSQLite {
		qLine = `INSERT INTO order_lines (id, order_id, producto_id, cantidad, precio_unitario) 
                 VALUES (?, ?, ?, ?, ?)`
	} else {
		qLine = `INSERT INTO order_lines (id, order_id, producto_id, cantidad, precio_unitario) 
                 VALUES ($1, $2, $3, $4, $5)`
	}
	for _, l := range o.Lineas {
		_, err = tx.ExecContext(ctx, qLine, l.ID.String(), l.OrderID.String(), l.ProductoID.String(), l.Cantidad, l.PrecioUnitario)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SQLSalesRepository) GetOrder(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	var qOrder string
	if r.isSQLite {
		qOrder = `SELECT id, empresa_id, quote_id, total, status, created_at FROM "order" WHERE id = ?`
	} else {
		qOrder = `SELECT id, empresa_id, quote_id, total, status, created_at FROM "order" WHERE id = $1`
	}

	var idStr, empIDStr, status string
	var quoteIDStr sql.NullString
	var createdAt time.Time
	var total float64

	err := r.db.QueryRowContext(ctx, qOrder, id.String()).Scan(&idStr, &empIDStr, &quoteIDStr, &total, &status, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrOrderNotFound
	} else if err != nil {
		return nil, err
	}

	// Fetch lines
	var qLines string
	if r.isSQLite {
		qLines = `SELECT id, order_id, producto_id, cantidad, precio_unitario FROM order_lines WHERE order_id = ?`
	} else {
		qLines = `SELECT id, order_id, producto_id, cantidad, precio_unitario FROM order_lines WHERE order_id = $1`
	}
	rows, err := r.db.QueryContext(ctx, qLines, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []domain.OrderLine
	for rows.Next() {
		var lIDStr, oIDStr, prodIDStr string
		var qty, price float64
		if err := rows.Scan(&lIDStr, &oIDStr, &prodIDStr, &qty, &price); err != nil {
			return nil, err
		}
		lUUID, _ := uuid.Parse(lIDStr)
		oUUID, _ := uuid.Parse(oIDStr)
		prodUUID, _ := uuid.Parse(prodIDStr)
		lines = append(lines, domain.OrderLine{
			ID:             lUUID,
			OrderID:        oUUID,
			ProductoID:     prodUUID,
			Cantidad:       qty,
			PrecioUnitario: price,
		})
	}

	var quoteUUIDPtr *uuid.UUID
	if quoteIDStr.Valid && quoteIDStr.String != "" {
		qUUID, _ := uuid.Parse(quoteIDStr.String)
		quoteUUIDPtr = &qUUID
	}

	oUUID, _ := uuid.Parse(idStr)
	empUUID, _ := uuid.Parse(empIDStr)

	return &domain.Order{
		ID:        oUUID,
		EmpresaID: empUUID,
		QuoteID:   quoteUUIDPtr,
		Total:     total,
		Status:    status,
		CreatedAt: createdAt,
		Lineas:    lines,
	}, nil
}

func (r *SQLSalesRepository) SaveDeliveryNote(ctx context.Context, dn *domain.DeliveryNote) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var qDN string
	var orderIDVal interface{}
	if dn.OrderID != nil {
		orderIDVal = dn.OrderID.String()
	} else {
		orderIDVal = nil
	}

	if r.isSQLite {
		qDN = `INSERT INTO delivery_note (id, empresa_id, order_id, total, status, warehouse_id, created_at) 
               VALUES (?, ?, ?, ?, ?, ?, ?)
               ON CONFLICT(id) DO UPDATE SET status=excluded.status, total=excluded.total`
	} else {
		qDN = `INSERT INTO delivery_note (id, empresa_id, order_id, total, status, warehouse_id, created_at) 
               VALUES ($1, $2, $3, $4, $5, $6, $7)
               ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status, total=EXCLUDED.total`
	}

	_, err = tx.ExecContext(ctx, qDN, dn.ID.String(), dn.EmpresaID.String(), orderIDVal, dn.Total, dn.Status, dn.WarehouseID.String(), dn.CreatedAt.UTC())
	if err != nil {
		return err
	}

	// Delete existing lines if updating
	var qDelete string
	if r.isSQLite {
		qDelete = `DELETE FROM delivery_note_lineas WHERE delivery_note_id = ?`
	} else {
		qDelete = `DELETE FROM delivery_note_lineas WHERE delivery_note_id = $1`
	}
	_, err = tx.ExecContext(ctx, qDelete, dn.ID.String())
	if err != nil {
		return err
	}

	// Insert lines
	var qLine string
	if r.isSQLite {
		qLine = `INSERT INTO delivery_note_lineas (id, delivery_note_id, producto_id, cantidad, precio_unitario) 
                 VALUES (?, ?, ?, ?, ?)`
	} else {
		qLine = `INSERT INTO delivery_note_lineas (id, delivery_note_id, producto_id, cantidad, precio_unitario) 
                 VALUES ($1, $2, $3, $4, $5)`
	}
	for _, l := range dn.Lineas {
		_, err = tx.ExecContext(ctx, qLine, l.ID.String(), l.DeliveryNoteID.String(), l.ProductoID.String(), l.Cantidad, l.PrecioUnitario)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SQLSalesRepository) GetDeliveryNote(ctx context.Context, id uuid.UUID) (*domain.DeliveryNote, error) {
	var qDN string
	if r.isSQLite {
		qDN = `SELECT id, empresa_id, order_id, total, status, warehouse_id, created_at FROM delivery_note WHERE id = ?`
	} else {
		qDN = `SELECT id, empresa_id, order_id, total, status, warehouse_id, created_at FROM delivery_note WHERE id = $1`
	}

	var idStr, empIDStr, status, whIDStr string
	var orderIDStr sql.NullString
	var createdAt time.Time
	var total float64

	err := r.db.QueryRowContext(ctx, qDN, id.String()).Scan(&idStr, &empIDStr, &orderIDStr, &total, &status, &whIDStr, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrDeliveryNoteNotFound
	} else if err != nil {
		return nil, err
	}

	// Fetch lines
	var qLines string
	if r.isSQLite {
		qLines = `SELECT id, delivery_note_id, producto_id, cantidad, precio_unitario FROM delivery_note_lineas WHERE delivery_note_id = ?`
	} else {
		qLines = `SELECT id, delivery_note_id, producto_id, cantidad, precio_unitario FROM delivery_note_lineas WHERE delivery_note_id = $1`
	}
	rows, err := r.db.QueryContext(ctx, qLines, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []domain.DeliveryNoteLinea
	for rows.Next() {
		var lIDStr, dnIDStr, prodIDStr string
		var qty, price float64
		if err := rows.Scan(&lIDStr, &dnIDStr, &prodIDStr, &qty, &price); err != nil {
			return nil, err
		}
		lUUID, _ := uuid.Parse(lIDStr)
		dnUUID, _ := uuid.Parse(dnIDStr)
		prodUUID, _ := uuid.Parse(prodIDStr)
		lines = append(lines, domain.DeliveryNoteLinea{
			ID:             lUUID,
			DeliveryNoteID: dnUUID,
			ProductoID:     prodUUID,
			Cantidad:       qty,
			PrecioUnitario: price,
		})
	}

	var orderUUIDPtr *uuid.UUID
	if orderIDStr.Valid && orderIDStr.String != "" {
		oUUID, _ := uuid.Parse(orderIDStr.String)
		orderUUIDPtr = &oUUID
	}

	dnUUID, _ := uuid.Parse(idStr)
	empUUID, _ := uuid.Parse(empIDStr)
	whUUID, _ := uuid.Parse(whIDStr)

	return &domain.DeliveryNote{
		ID:          dnUUID,
		EmpresaID:   empUUID,
		OrderID:     orderUUIDPtr,
		Total:       total,
		Status:      status,
		WarehouseID: whUUID,
		CreatedAt:   createdAt,
		Lineas:      lines,
	}, nil
}

func (r *SQLSalesRepository) SaveInvoice(ctx context.Context, inv *domain.Invoice) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var qInvoice string
	var dnIDVal interface{}
	if inv.DeliveryNoteID != nil {
		dnIDVal = inv.DeliveryNoteID.String()
	} else {
		dnIDVal = nil
	}

	if r.isSQLite {
		qInvoice = `INSERT INTO invoice (id, empresa_id, delivery_note_id, terminal_id, invoicing_series_id, invoice_number, sequence_number, total, status, created_at) 
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                    ON CONFLICT(id) DO UPDATE SET status=excluded.status, total=excluded.total`
	} else {
		qInvoice = `INSERT INTO invoice (id, empresa_id, delivery_note_id, terminal_id, invoicing_series_id, invoice_number, sequence_number, total, status, created_at) 
                    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
                    ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status, total=EXCLUDED.total`
	}

	_, err = tx.ExecContext(ctx, qInvoice,
		inv.ID.String(),
		inv.EmpresaID.String(),
		dnIDVal,
		inv.TerminalID.String(),
		inv.InvoicingSeriesID.String(),
		inv.InvoiceNumber,
		inv.SequenceNumber,
		inv.Total,
		inv.Status,
		inv.CreatedAt.UTC(),
	)
	if err != nil {
		return err
	}

	// Delete existing lines if updating
	var qDelete string
	if r.isSQLite {
		qDelete = `DELETE FROM invoice_lineas WHERE invoice_id = ?`
	} else {
		qDelete = `DELETE FROM invoice_lineas WHERE invoice_id = $1`
	}
	_, err = tx.ExecContext(ctx, qDelete, inv.ID.String())
	if err != nil {
		return err
	}

	// Insert lines
	var qLine string
	if r.isSQLite {
		qLine = `INSERT INTO invoice_lineas (id, invoice_id, producto_id, cantidad, precio_unitario) 
                 VALUES (?, ?, ?, ?, ?)`
	} else {
		qLine = `INSERT INTO invoice_lineas (id, invoice_id, producto_id, cantidad, precio_unitario) 
                 VALUES ($1, $2, $3, $4, $5)`
	}
	for _, l := range inv.Lineas {
		_, err = tx.ExecContext(ctx, qLine, l.ID.String(), l.InvoiceID.String(), l.ProductoID.String(), l.Cantidad, l.PrecioUnitario)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SQLSalesRepository) GetInvoice(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	var qInvoice string
	if r.isSQLite {
		qInvoice = `SELECT id, empresa_id, delivery_note_id, terminal_id, invoicing_series_id, invoice_number, sequence_number, total, status, created_at FROM invoice WHERE id = ?`
	} else {
		qInvoice = `SELECT id, empresa_id, delivery_note_id, terminal_id, invoicing_series_id, invoice_number, sequence_number, total, status, created_at FROM invoice WHERE id = $1`
	}

	var idStr, empIDStr, termIDStr, seriesIDStr, invoiceNumber, status string
	var dnIDStr sql.NullString
	var seq int
	var createdAt time.Time
	var total float64

	err := r.db.QueryRowContext(ctx, qInvoice, id.String()).Scan(&idStr, &empIDStr, &dnIDStr, &termIDStr, &seriesIDStr, &invoiceNumber, &seq, &total, &status, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrInvoiceNotFound
	} else if err != nil {
		return nil, err
	}

	// Fetch lines
	var qLines string
	if r.isSQLite {
		qLines = `SELECT id, invoice_id, producto_id, cantidad, precio_unitario FROM invoice_lineas WHERE invoice_id = ?`
	} else {
		qLines = `SELECT id, invoice_id, producto_id, cantidad, precio_unitario FROM invoice_lineas WHERE invoice_id = $1`
	}
	rows, err := r.db.QueryContext(ctx, qLines, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []domain.InvoiceLinea
	for rows.Next() {
		var lIDStr, invIDStr, prodIDStr string
		var qty, price float64
		if err := rows.Scan(&lIDStr, &invIDStr, &prodIDStr, &qty, &price); err != nil {
			return nil, err
		}
		lUUID, _ := uuid.Parse(lIDStr)
		invUUID, _ := uuid.Parse(invIDStr)
		prodUUID, _ := uuid.Parse(prodIDStr)
		lines = append(lines, domain.InvoiceLinea{
			ID:             lUUID,
			InvoiceID:      invUUID,
			ProductoID:     prodUUID,
			Cantidad:       qty,
			PrecioUnitario: price,
		})
	}

	var dnUUIDPtr *uuid.UUID
	if dnIDStr.Valid && dnIDStr.String != "" {
		dnUUID, _ := uuid.Parse(dnIDStr.String)
		dnUUIDPtr = &dnUUID
	}

	invUUID, _ := uuid.Parse(idStr)
	empUUID, _ := uuid.Parse(empIDStr)
	termUUID, _ := uuid.Parse(termIDStr)
	seriesUUID, _ := uuid.Parse(seriesIDStr)

	return &domain.Invoice{
		ID:                invUUID,
		EmpresaID:         empUUID,
		DeliveryNoteID:    dnUUIDPtr,
		TerminalID:        termUUID,
		InvoicingSeriesID: seriesUUID,
		InvoiceNumber:     invoiceNumber,
		SequenceNumber:    seq,
		Total:             total,
		Status:            status,
		CreatedAt:         createdAt,
		Lineas:            lines,
	}, nil
}
