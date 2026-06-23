package adapters

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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

func (r *SQLSalesRepository) paramPlaceholder(idx int) string {
	if r.isSQLite {
		return "?"
	}
	return fmt.Sprintf("$%d", idx)
}

func (r *SQLSalesRepository) updateExec(ctx context.Context, table string, sets map[string]interface{}, id uuid.UUID, notFoundErr error) error {
	if len(sets) == 0 {
		return nil
	}
	var setClauses []string
	var args []interface{}
	idx := 1
	for col, val := range sets {
		setClauses = append(setClauses, fmt.Sprintf("%s = %s", col, r.paramPlaceholder(idx)))
		args = append(args, val)
		idx++
	}
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = %s", table, strings.Join(setClauses, ", "), r.paramPlaceholder(idx))
	args = append(args, id.String())

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 && notFoundErr != nil {
		return notFoundErr
	}
	return nil
}

func (r *SQLSalesRepository) UpdatePresupuesto(ctx context.Context, input domain.UpdatePresupuestoInput) error {
	sets := make(map[string]interface{})
	if input.ClienteID != nil {
		sets["cliente_id"] = input.ClienteID.String()
	}
	if input.FechaValidez != nil {
		sets["fecha_validez"] = input.FechaValidez.UTC()
	}
	return r.updateExec(ctx, "presupuestos", sets, input.ID, domain.ErrPresupuestoNotFound)
}

func (r *SQLSalesRepository) UpdatePedido(ctx context.Context, input domain.UpdatePedidoInput) error {
	// No updatable fields currently; no-op.
	if input.ID == uuid.Nil {
		return domain.ErrPedidoNotFound
	}
	return nil
}

func (r *SQLSalesRepository) UpdateAlbaran(ctx context.Context, input domain.UpdateAlbaranInput) error {
	if input.ID == uuid.Nil {
		return domain.ErrAlbaranNotFound
	}
	return nil
}

func (r *SQLSalesRepository) CancelPresupuesto(ctx context.Context, id uuid.UUID) error {
	return r.updateExec(ctx, "presupuestos", map[string]interface{}{"estado": domain.StatusCancelled}, id, domain.ErrPresupuestoNotFound)
}

func (r *SQLSalesRepository) CancelPedido(ctx context.Context, id uuid.UUID) error {
	return r.updateExec(ctx, "pedidos", map[string]interface{}{"estado": domain.StatusCancelled}, id, domain.ErrPedidoNotFound)
}

func (r *SQLSalesRepository) CancelAlbaran(ctx context.Context, id uuid.UUID) error {
	return r.updateExec(ctx, "albaranes", map[string]interface{}{"estado": domain.StatusCancelled}, id, domain.ErrAlbaranNotFound)
}

func (r *SQLSalesRepository) CancelFactura(ctx context.Context, id uuid.UUID) error {
	return r.updateExec(ctx, "facturas", map[string]interface{}{"estado": domain.StatusCancelled}, id, domain.ErrFacturaNotFound)
}

func (r *SQLSalesRepository) SavePresupuesto(ctx context.Context, q *domain.Presupuesto) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var qPresupuesto string
	if r.isSQLite {
		qPresupuesto = `INSERT INTO presupuestos (id, empresa_id, cliente_id, total, estado, fecha_validez, created_at, version) 
                  VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                  ON CONFLICT(id) DO UPDATE SET estado=excluded.estado, total=excluded.total, version=excluded.version + 1
                  WHERE version = excluded.version`
	} else {
		qPresupuesto = `INSERT INTO presupuestos (id, empresa_id, cliente_id, total, estado, fecha_validez, created_at, version) 
                  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
                  ON CONFLICT(id) DO UPDATE SET estado=EXCLUDED.estado, total=EXCLUDED.total, version=EXCLUDED.version + 1
                  WHERE presupuestos.version = EXCLUDED.version`
	}

	result, err := tx.ExecContext(ctx, qPresupuesto, q.ID.String(), q.EmpresaID.String(), q.ClienteID.String(), q.Total, q.Estado, q.FechaValidez.UTC(), q.CreatedAt.UTC(), q.Version)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrConcurrentModification
	}

	// Delete existing lines if updating
	var qDelete string
	if r.isSQLite {
		qDelete = `DELETE FROM presupuesto_lineas WHERE presupuesto_id = ?`
	} else {
		qDelete = `DELETE FROM presupuesto_lineas WHERE presupuesto_id = $1`
	}
	_, err = tx.ExecContext(ctx, qDelete, q.ID.String())
	if err != nil {
		return err
	}

	// Insert lines
	var qLine string
	if r.isSQLite {
		qLine = `INSERT INTO presupuesto_lineas (id, presupuesto_id, producto_id, cantidad, precio_unitario, coste_unitario) 
                 VALUES (?, ?, ?, ?, ?, ?)`
	} else {
		qLine = `INSERT INTO presupuesto_lineas (id, presupuesto_id, producto_id, cantidad, precio_unitario, coste_unitario) 
                 VALUES ($1, $2, $3, $4, $5, $6)`
	}
	for _, l := range q.Lineas {
		_, err = tx.ExecContext(ctx, qLine, l.ID.String(), l.PresupuestoID.String(), l.ProductoID.String(), l.Cantidad, l.PrecioUnitario, l.CosteUnitario)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SQLSalesRepository) GetPresupuesto(ctx context.Context, id uuid.UUID) (*domain.Presupuesto, error) {
	var qPresupuesto string
	if r.isSQLite {
		qPresupuesto = `SELECT id, empresa_id, cliente_id, total, estado, fecha_validez, created_at, version FROM presupuestos WHERE id = ?`
	} else {
		qPresupuesto = `SELECT id, empresa_id, cliente_id, total, estado, fecha_validez, created_at, version FROM presupuestos WHERE id = $1`
	}

	var idStr, empIDStr, clientIDStr, status string
	var expiresAt, createdAt time.Time
	var total float64
	var version int

	err := r.db.QueryRowContext(ctx, qPresupuesto, id.String()).Scan(&idStr, &empIDStr, &clientIDStr, &total, &status, &expiresAt, &createdAt, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPresupuestoNotFound
	} else if err != nil {
		return nil, err
	}

	// Fetch lines
	var qLines string
	if r.isSQLite {
		qLines = `SELECT id, presupuesto_id, producto_id, cantidad, precio_unitario, coste_unitario FROM presupuesto_lineas WHERE presupuesto_id = ?`
	} else {
		qLines = `SELECT id, presupuesto_id, producto_id, cantidad, precio_unitario, coste_unitario FROM presupuesto_lineas WHERE presupuesto_id = $1`
	}
	rows, err := r.db.QueryContext(ctx, qLines, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []domain.PresupuestoLinea
	for rows.Next() {
		var lIDStr, qIDStr, prodIDStr string
		var qty, price, cost float64
		if err := rows.Scan(&lIDStr, &qIDStr, &prodIDStr, &qty, &price, &cost); err != nil {
			return nil, err
		}
		lUUID, _ := uuid.Parse(lIDStr)
		qUUID, _ := uuid.Parse(qIDStr)
		prodUUID, _ := uuid.Parse(prodIDStr)
		lines = append(lines, domain.PresupuestoLinea{
			ID:             lUUID,
			PresupuestoID:        qUUID,
			ProductoID:     prodUUID,
			Cantidad:       qty,
			PrecioUnitario: price,
			CosteUnitario:  cost,
		})
	}

	qUUID, _ := uuid.Parse(idStr)
	empUUID, _ := uuid.Parse(empIDStr)
	clientUUID, _ := uuid.Parse(clientIDStr)

	return &domain.Presupuesto{
		ID:        qUUID,
		EmpresaID: empUUID,
		ClienteID:  clientUUID,
		Total:     total,
		Estado:    status,
		FechaValidez: expiresAt,
		CreatedAt: createdAt,
		Version:   version,
		Lineas:    lines,
	}, nil
}

func (r *SQLSalesRepository) ListPresupuestos(ctx context.Context, empresaID uuid.UUID, filter domain.DocumentFilter) ([]*domain.Presupuesto, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	where += fmt.Sprintf(" AND empresa_id = %s", r.paramPlaceholder(argIdx))
	args = append(args, empresaID.String())
	argIdx++

	if filter.Estado != nil {
		where += fmt.Sprintf(" AND estado = %s", r.paramPlaceholder(argIdx))
		args = append(args, *filter.Estado)
		argIdx++
	}
	if filter.ClienteID != nil {
		where += fmt.Sprintf(" AND cliente_id = %s", r.paramPlaceholder(argIdx))
		args = append(args, filter.ClienteID.String())
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

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM presupuestos %s", where)
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	var dataQuery string
	if r.isSQLite {
		dataQuery = fmt.Sprintf("SELECT id, empresa_id, cliente_id, total, estado, fecha_validez, created_at, version FROM presupuestos %s ORDER BY created_at DESC LIMIT ? OFFSET ?", where)
	} else {
		dataQuery = fmt.Sprintf("SELECT id, empresa_id, cliente_id, total, estado, fecha_validez, created_at, version FROM presupuestos %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d", where, argIdx, argIdx+1)
	}
	dataArgs := append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var quotes []*domain.Presupuesto
	for rows.Next() {
		var idStr, empIDStr, clientIDStr, status string
		var expiresAt, createdAt time.Time
		var total float64
		var version int
		if err := rows.Scan(&idStr, &empIDStr, &clientIDStr, &total, &status, &expiresAt, &createdAt, &version); err != nil {
			return nil, 0, err
		}
		qUUID, _ := uuid.Parse(idStr)
		empUUID, _ := uuid.Parse(empIDStr)
		clientUUID, _ := uuid.Parse(clientIDStr)
		quotes = append(quotes, &domain.Presupuesto{
			ID:        qUUID,
			EmpresaID: empUUID,
			ClienteID:  clientUUID,
			Total:     total,
			Estado:    status,
			FechaValidez: expiresAt,
			CreatedAt: createdAt,
			Version:   version,
		})
	}
	return quotes, total, nil
}

func (r *SQLSalesRepository) SavePedido(ctx context.Context, o *domain.Pedido) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var qPedido string
	var quoteIDVal interface{}
	if o.PresupuestoID != nil {
		quoteIDVal = o.PresupuestoID.String()
	} else {
		quoteIDVal = nil
	}

	if r.isSQLite {
		qPedido = `INSERT INTO pedidos (id, empresa_id, presupuesto_id, total, estado, created_at, version) 
                  VALUES (?, ?, ?, ?, ?, ?, ?)
                  ON CONFLICT(id) DO UPDATE SET estado=excluded.estado, total=excluded.total, version=excluded.version + 1
                  WHERE version = excluded.version`
	} else {
		qPedido = `INSERT INTO pedidos (id, empresa_id, presupuesto_id, total, estado, created_at, version) 
                  VALUES ($1, $2, $3, $4, $5, $6, $7)
                  ON CONFLICT(id) DO UPDATE SET estado=EXCLUDED.estado, total=EXCLUDED.total, version=EXCLUDED.version + 1
                  WHERE pedidos.version = EXCLUDED.version`
	}

	result, err := tx.ExecContext(ctx, qPedido, o.ID.String(), o.EmpresaID.String(), quoteIDVal, o.Total, o.Estado, o.CreatedAt.UTC(), o.Version)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrConcurrentModification
	}

	// Delete existing lines if updating
	var qDelete string
	if r.isSQLite {
		qDelete = `DELETE FROM pedido_lineas WHERE pedido_id = ?`
	} else {
		qDelete = `DELETE FROM pedido_lineas WHERE pedido_id = $1`
	}
	_, err = tx.ExecContext(ctx, qDelete, o.ID.String())
	if err != nil {
		return err
	}

	// Insert lines
	var qLine string
	if r.isSQLite {
		qLine = `INSERT INTO pedido_lineas (id, pedido_id, producto_id, cantidad, precio_unitario) 
                 VALUES (?, ?, ?, ?, ?)`
	} else {
		qLine = `INSERT INTO pedido_lineas (id, pedido_id, producto_id, cantidad, precio_unitario) 
                 VALUES ($1, $2, $3, $4, $5)`
	}
	for _, l := range o.Lineas {
		_, err = tx.ExecContext(ctx, qLine, l.ID.String(), l.PedidoID.String(), l.ProductoID.String(), l.Cantidad, l.PrecioUnitario)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SQLSalesRepository) GetPedido(ctx context.Context, id uuid.UUID) (*domain.Pedido, error) {
	var qPedido string
	if r.isSQLite {
		qPedido = `SELECT id, empresa_id, presupuesto_id, total, estado, created_at, version FROM pedidos WHERE id = ?`
	} else {
		qPedido = `SELECT id, empresa_id, presupuesto_id, total, estado, created_at, version FROM pedidos WHERE id = $1`
	}

	var idStr, empIDStr, status string
	var quoteIDStr sql.NullString
	var createdAt time.Time
	var total float64
	var version int

	err := r.db.QueryRowContext(ctx, qPedido, id.String()).Scan(&idStr, &empIDStr, &quoteIDStr, &total, &status, &createdAt, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPedidoNotFound
	} else if err != nil {
		return nil, err
	}

	// Fetch lines
	var qLines string
	if r.isSQLite {
		qLines = `SELECT id, pedido_id, producto_id, cantidad, precio_unitario FROM pedido_lineas WHERE pedido_id = ?`
	} else {
		qLines = `SELECT id, pedido_id, producto_id, cantidad, precio_unitario FROM pedido_lineas WHERE pedido_id = $1`
	}
	rows, err := r.db.QueryContext(ctx, qLines, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []domain.PedidoLinea
	for rows.Next() {
		var lIDStr, oIDStr, prodIDStr string
		var qty, price float64
		if err := rows.Scan(&lIDStr, &oIDStr, &prodIDStr, &qty, &price); err != nil {
			return nil, err
		}
		lUUID, _ := uuid.Parse(lIDStr)
		oUUID, _ := uuid.Parse(oIDStr)
		prodUUID, _ := uuid.Parse(prodIDStr)
		lines = append(lines, domain.PedidoLinea{
			ID:             lUUID,
			PedidoID:        oUUID,
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

	return &domain.Pedido{
		ID:        oUUID,
		EmpresaID: empUUID,
		PresupuestoID:   quoteUUIDPtr,
		Total:     total,
		Estado:    status,
		CreatedAt: createdAt,
		Version:   version,
		Lineas:    lines,
	}, nil
}

func (r *SQLSalesRepository) ListPedidos(ctx context.Context, empresaID uuid.UUID, filter domain.DocumentFilter) ([]*domain.Pedido, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	where += fmt.Sprintf(" AND empresa_id = %s", r.paramPlaceholder(argIdx))
	args = append(args, empresaID.String())
	argIdx++

	if filter.Estado != nil {
		where += fmt.Sprintf(" AND estado = %s", r.paramPlaceholder(argIdx))
		args = append(args, *filter.Estado)
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

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM pedidos %s", where)
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	var dataQuery string
	if r.isSQLite {
		dataQuery = fmt.Sprintf("SELECT id, empresa_id, presupuesto_id, total, estado, created_at, version FROM pedidos %s ORDER BY created_at DESC LIMIT ? OFFSET ?", where)
	} else {
		dataQuery = fmt.Sprintf("SELECT id, empresa_id, presupuesto_id, total, estado, created_at, version FROM pedidos %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d", where, argIdx, argIdx+1)
	}
	dataArgs := append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []*domain.Pedido
	for rows.Next() {
		var idStr, empIDStr, status string
		var quoteIDStr sql.NullString
		var createdAt time.Time
		var total float64
		var version int
		if err := rows.Scan(&idStr, &empIDStr, &quoteIDStr, &total, &status, &createdAt, &version); err != nil {
			return nil, 0, err
		}
		oUUID, _ := uuid.Parse(idStr)
		empUUID, _ := uuid.Parse(empIDStr)

		var quoteUUIDPtr *uuid.UUID
		if quoteIDStr.Valid && quoteIDStr.String != "" {
			qUUID, _ := uuid.Parse(quoteIDStr.String)
			quoteUUIDPtr = &qUUID
		}

		orders = append(orders, &domain.Pedido{
			ID:        oUUID,
			EmpresaID: empUUID,
			PresupuestoID:   quoteUUIDPtr,
			Total:     total,
			Estado:    status,
			CreatedAt: createdAt,
			Version:   version,
		})
	}
	return orders, total, nil
}

func (r *SQLSalesRepository) SaveAlbaran(ctx context.Context, dn *domain.Albaran) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var qDN string
	var orderIDVal interface{}
	if dn.PedidoID != nil {
		orderIDVal = dn.PedidoID.String()
	} else {
		orderIDVal = nil
	}

	if r.isSQLite {
		qDN = `INSERT INTO albaranes (id, empresa_id, pedido_id, total, estado, almacen_id, created_at, version) 
               VALUES (?, ?, ?, ?, ?, ?, ?, ?)
               ON CONFLICT(id) DO UPDATE SET estado=excluded.estado, total=excluded.total, version=excluded.version + 1
               WHERE version = excluded.version`
	} else {
		qDN = `INSERT INTO albaranes (id, empresa_id, pedido_id, total, estado, almacen_id, created_at, version) 
               VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
               ON CONFLICT(id) DO UPDATE SET estado=EXCLUDED.estado, total=EXCLUDED.total, version=EXCLUDED.version + 1
               WHERE albaranes.version = EXCLUDED.version`
	}

	result, err := tx.ExecContext(ctx, qDN, dn.ID.String(), dn.EmpresaID.String(), orderIDVal, dn.Total, dn.Estado, dn.AlmacenID.String(), dn.CreatedAt.UTC(), dn.Version)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrConcurrentModification
	}

	// Delete existing lines if updating
	var qDelete string
	if r.isSQLite {
		qDelete = `DELETE FROM albaran_lineas WHERE albaran_id = ?`
	} else {
		qDelete = `DELETE FROM albaran_lineas WHERE albaran_id = $1`
	}
	_, err = tx.ExecContext(ctx, qDelete, dn.ID.String())
	if err != nil {
		return err
	}

	// Insert lines
	var qLine string
	if r.isSQLite {
		qLine = `INSERT INTO albaran_lineas (id, albaran_id, producto_id, cantidad, precio_unitario) 
                 VALUES (?, ?, ?, ?, ?)`
	} else {
		qLine = `INSERT INTO albaran_lineas (id, albaran_id, producto_id, cantidad, precio_unitario) 
                 VALUES ($1, $2, $3, $4, $5)`
	}
	for _, l := range dn.Lineas {
		_, err = tx.ExecContext(ctx, qLine, l.ID.String(), l.AlbaranID.String(), l.ProductoID.String(), l.Cantidad, l.PrecioUnitario)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SQLSalesRepository) GetAlbaran(ctx context.Context, id uuid.UUID) (*domain.Albaran, error) {
	var qDN string
	if r.isSQLite {
		qDN = `SELECT id, empresa_id, pedido_id, total, estado, almacen_id, created_at, version FROM albaranes WHERE id = ?`
	} else {
		qDN = `SELECT id, empresa_id, pedido_id, total, estado, almacen_id, created_at, version FROM albaranes WHERE id = $1`
	}

	var idStr, empIDStr, status, whIDStr string
	var orderIDStr sql.NullString
	var createdAt time.Time
	var total float64
	var version int

	err := r.db.QueryRowContext(ctx, qDN, id.String()).Scan(&idStr, &empIDStr, &orderIDStr, &total, &status, &whIDStr, &createdAt, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrAlbaranNotFound
	} else if err != nil {
		return nil, err
	}

	// Fetch lines
	var qLines string
	if r.isSQLite {
		qLines = `SELECT id, albaran_id, producto_id, cantidad, precio_unitario FROM albaran_lineas WHERE albaran_id = ?`
	} else {
		qLines = `SELECT id, albaran_id, producto_id, cantidad, precio_unitario FROM albaran_lineas WHERE albaran_id = $1`
	}
	rows, err := r.db.QueryContext(ctx, qLines, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []domain.AlbaranLinea
	for rows.Next() {
		var lIDStr, dnIDStr, prodIDStr string
		var qty, price float64
		if err := rows.Scan(&lIDStr, &dnIDStr, &prodIDStr, &qty, &price); err != nil {
			return nil, err
		}
		lUUID, _ := uuid.Parse(lIDStr)
		dnUUID, _ := uuid.Parse(dnIDStr)
		prodUUID, _ := uuid.Parse(prodIDStr)
		lines = append(lines, domain.AlbaranLinea{
			ID:             lUUID,
			AlbaranID: dnUUID,
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

	return &domain.Albaran{
		ID:          dnUUID,
		EmpresaID:   empUUID,
		PedidoID:     orderUUIDPtr,
		Total:       total,
		Estado:      status,
		AlmacenID: whUUID,
		CreatedAt:   createdAt,
		Version:     version,
		Lineas:      lines,
	}, nil
}

func (r *SQLSalesRepository) ListAlbarans(ctx context.Context, empresaID uuid.UUID, filter domain.DocumentFilter) ([]*domain.Albaran, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	where += fmt.Sprintf(" AND empresa_id = %s", r.paramPlaceholder(argIdx))
	args = append(args, empresaID.String())
	argIdx++

	if filter.Estado != nil {
		where += fmt.Sprintf(" AND estado = %s", r.paramPlaceholder(argIdx))
		args = append(args, *filter.Estado)
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

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM albaranes %s", where)
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	var dataQuery string
	if r.isSQLite {
		dataQuery = fmt.Sprintf("SELECT id, empresa_id, pedido_id, total, estado, almacen_id, created_at, version FROM albaranes %s ORDER BY created_at DESC LIMIT ? OFFSET ?", where)
	} else {
		dataQuery = fmt.Sprintf("SELECT id, empresa_id, pedido_id, total, estado, almacen_id, created_at, version FROM albaranes %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d", where, argIdx, argIdx+1)
	}
	dataArgs := append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var notes []*domain.Albaran
	for rows.Next() {
		var idStr, empIDStr, status, whIDStr string
		var orderIDStr sql.NullString
		var createdAt time.Time
		var total float64
		var version int
		if err := rows.Scan(&idStr, &empIDStr, &orderIDStr, &total, &status, &whIDStr, &createdAt, &version); err != nil {
			return nil, 0, err
		}
		dnUUID, _ := uuid.Parse(idStr)
		empUUID, _ := uuid.Parse(empIDStr)
		whUUID, _ := uuid.Parse(whIDStr)

		var orderUUIDPtr *uuid.UUID
		if orderIDStr.Valid && orderIDStr.String != "" {
			oUUID, _ := uuid.Parse(orderIDStr.String)
			orderUUIDPtr = &oUUID
		}

		notes = append(notes, &domain.Albaran{
			ID:          dnUUID,
			EmpresaID:   empUUID,
			PedidoID:     orderUUIDPtr,
			Total:       total,
			Estado:      status,
			AlmacenID: whUUID,
			CreatedAt:   createdAt,
			Version:     version,
		})
	}
	return notes, total, nil
}

func (r *SQLSalesRepository) SaveFactura(ctx context.Context, inv *domain.Factura) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var qFactura string
	var dnIDVal interface{}
	if inv.AlbaranID != nil {
		dnIDVal = inv.AlbaranID.String()
	} else {
		dnIDVal = nil
	}

	if r.isSQLite {
		qFactura = `INSERT INTO facturas (id, empresa_id, albaran_id, terminal_id, serie_facturacion_id, numero_factura, numero_secuencia, total, total_rectificado, estado, created_at, version) 
                     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                     ON CONFLICT(id) DO UPDATE SET estado=excluded.estado, total=excluded.total, total_rectificado=excluded.total_rectificado, version=excluded.version + 1
                     WHERE version = excluded.version`
	} else {
		qFactura = `INSERT INTO facturas (id, empresa_id, albaran_id, terminal_id, serie_facturacion_id, numero_factura, numero_secuencia, total, total_rectificado, estado, created_at, version) 
                     VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
                     ON CONFLICT(id) DO UPDATE SET estado=EXCLUDED.estado, total=EXCLUDED.total, total_rectificado=EXCLUDED.total_rectificado, version=EXCLUDED.version + 1
                     WHERE facturas.version = EXCLUDED.version`
	}

	result, err := tx.ExecContext(ctx, qFactura,
		inv.ID.String(),
		inv.EmpresaID.String(),
		dnIDVal,
		inv.TerminalID.String(),
		inv.SerieFacturacionID.String(),
		inv.NumeroFactura,
		inv.NumeroSecuencia,
		inv.Total,
		inv.RectifiedTotal,
		inv.Estado,
		inv.CreatedAt.UTC(),
		inv.Version,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrConcurrentModification
	}

	// Delete existing lines if updating
	var qDelete string
	if r.isSQLite {
		qDelete = `DELETE FROM factura_lineas WHERE factura_id = ?`
	} else {
		qDelete = `DELETE FROM factura_lineas WHERE factura_id = $1`
	}
	_, err = tx.ExecContext(ctx, qDelete, inv.ID.String())
	if err != nil {
		return err
	}

	// Insert lines
	var qLine string
	if r.isSQLite {
		qLine = `INSERT INTO factura_lineas (id, factura_id, producto_id, cantidad, precio_unitario) 
                 VALUES (?, ?, ?, ?, ?)`
	} else {
		qLine = `INSERT INTO factura_lineas (id, factura_id, producto_id, cantidad, precio_unitario) 
                 VALUES ($1, $2, $3, $4, $5)`
	}
	for _, l := range inv.Lineas {
		_, err = tx.ExecContext(ctx, qLine, l.ID.String(), l.FacturaID.String(), l.ProductoID.String(), l.Cantidad, l.PrecioUnitario)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SQLSalesRepository) GetFactura(ctx context.Context, id uuid.UUID) (*domain.Factura, error) {
	var qFactura string
	if r.isSQLite {
		qFactura = `SELECT id, empresa_id, albaran_id, terminal_id, serie_facturacion_id, numero_factura, numero_secuencia, total, total_rectificado, estado, created_at, version FROM facturas WHERE id = ?`
	} else {
		qFactura = `SELECT id, empresa_id, albaran_id, terminal_id, serie_facturacion_id, numero_factura, numero_secuencia, total, total_rectificado, estado, created_at, version FROM facturas WHERE id = $1`
	}

	var idStr, empIDStr, termIDStr, seriesIDStr, invoiceNumber, status string
	var dnIDStr sql.NullString
	var seq int
	var createdAt time.Time
	var total, creditedTotal float64
	var version int

	err := r.db.QueryRowContext(ctx, qFactura, id.String()).Scan(&idStr, &empIDStr, &dnIDStr, &termIDStr, &seriesIDStr, &invoiceNumber, &seq, &total, &creditedTotal, &status, &createdAt, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrFacturaNotFound
	} else if err != nil {
		return nil, err
	}

	// Fetch lines
	var qLines string
	if r.isSQLite {
		qLines = `SELECT id, factura_id, producto_id, cantidad, precio_unitario FROM factura_lineas WHERE factura_id = ?`
	} else {
		qLines = `SELECT id, factura_id, producto_id, cantidad, precio_unitario FROM factura_lineas WHERE factura_id = $1`
	}
	rows, err := r.db.QueryContext(ctx, qLines, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []domain.FacturaLinea
	for rows.Next() {
		var lIDStr, invIDStr, prodIDStr string
		var qty, price float64
		if err := rows.Scan(&lIDStr, &invIDStr, &prodIDStr, &qty, &price); err != nil {
			return nil, err
		}
		lUUID, _ := uuid.Parse(lIDStr)
		invUUID, _ := uuid.Parse(invIDStr)
		prodUUID, _ := uuid.Parse(prodIDStr)
		lines = append(lines, domain.FacturaLinea{
			ID:             lUUID,
			FacturaID:      invUUID,
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

	return &domain.Factura{
		ID:                invUUID,
		EmpresaID:         empUUID,
		AlbaranID:    dnUUIDPtr,
		TerminalID:        termUUID,
		SerieFacturacionID: seriesUUID,
		NumeroFactura:     invoiceNumber,
		NumeroSecuencia:    seq,
		Total:             total,
		RectifiedTotal:     creditedTotal,
		Estado:            status,
		CreatedAt:         createdAt,
		Version:           version,
		Lineas:            lines,
	}, nil
}

func (r *SQLSalesRepository) ListFacturas(ctx context.Context, empresaID uuid.UUID, filter domain.DocumentFilter) ([]*domain.Factura, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	where += fmt.Sprintf(" AND empresa_id = %s", r.paramPlaceholder(argIdx))
	args = append(args, empresaID.String())
	argIdx++

	if filter.Estado != nil {
		where += fmt.Sprintf(" AND estado = %s", r.paramPlaceholder(argIdx))
		args = append(args, *filter.Estado)
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

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM facturas %s", where)
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	var dataQuery string
	if r.isSQLite {
		dataQuery = fmt.Sprintf(`SELECT id, empresa_id, albaran_id, terminal_id, serie_facturacion_id, numero_factura, numero_secuencia, total, total_rectificado, estado, created_at, version FROM facturas %s ORDER BY created_at DESC LIMIT ? OFFSET ?`, where)
	} else {
		dataQuery = fmt.Sprintf(`SELECT id, empresa_id, albaran_id, terminal_id, serie_facturacion_id, numero_factura, numero_secuencia, total, total_rectificado, estado, created_at, version FROM facturas %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	}
	dataArgs := append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var invoices []*domain.Factura
	for rows.Next() {
		var idStr, empIDStr, termIDStr, seriesIDStr, invoiceNumber, status string
		var dnIDStr sql.NullString
		var seq int
		var createdAt time.Time
		var total, creditedTotal float64
		var version int
		if err := rows.Scan(&idStr, &empIDStr, &dnIDStr, &termIDStr, &seriesIDStr, &invoiceNumber, &seq, &total, &creditedTotal, &status, &createdAt, &version); err != nil {
			return nil, 0, err
		}
		invUUID, _ := uuid.Parse(idStr)
		empUUID, _ := uuid.Parse(empIDStr)
		termUUID, _ := uuid.Parse(termIDStr)
		seriesUUID, _ := uuid.Parse(seriesIDStr)

		var dnUUIDPtr *uuid.UUID
		if dnIDStr.Valid && dnIDStr.String != "" {
			dUUID, _ := uuid.Parse(dnIDStr.String)
			dnUUIDPtr = &dUUID
		}

		invoices = append(invoices, &domain.Factura{
			ID:                invUUID,
			EmpresaID:         empUUID,
			AlbaranID:    dnUUIDPtr,
			TerminalID:        termUUID,
			SerieFacturacionID: seriesUUID,
			NumeroFactura:     invoiceNumber,
			NumeroSecuencia:    seq,
			Total:             total,
			RectifiedTotal:     creditedTotal,
			Estado:            status,
			CreatedAt:         createdAt,
			Version:           version,
		})
	}
	return invoices, total, nil
}

func (r *SQLSalesRepository) CreateFacturaRectificativa(ctx context.Context, fr *domain.FacturaRectificativa) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var terminalIDVal interface{}
	if fr.TerminalID != nil {
		terminalIDVal = fr.TerminalID.String()
	} else {
		terminalIDVal = nil
	}

	var qFR string
	if r.isSQLite {
		qFR = `INSERT INTO facturas_rectificativas (id, invoice_id, empresa_id, terminal_id, numero_fr, sequence_number, total, reason, status, created_at) 
               VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	} else {
		qFR = `INSERT INTO facturas_rectificativas (id, invoice_id, empresa_id, terminal_id, numero_fr, sequence_number, total, reason, status, created_at) 
               VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	}

	_, err = tx.ExecContext(ctx, qFR,
		fr.ID.String(),
		fr.FacturaID.String(),
		fr.EmpresaID.String(),
		terminalIDVal,
		fr.NumeroFR,
		fr.NumeroSecuencia,
		fr.Total,
		fr.Motivo,
		fr.Estado,
		fr.CreatedAt.UTC(),
	)
	if err != nil {
		return err
	}

	// Insert lines
	var qLine string
	if r.isSQLite {
		qLine = `INSERT INTO factura_rectificativa_lineas (id, factura_rectificativa_id, producto_id, cantidad, precio_unitario) 
                 VALUES (?, ?, ?, ?, ?)`
	} else {
		qLine = `INSERT INTO factura_rectificativa_lineas (id, factura_rectificativa_id, producto_id, cantidad, precio_unitario) 
                 VALUES ($1, $2, $3, $4, $5)`
	}
	for _, l := range fr.Lines {
		_, err = tx.ExecContext(ctx, qLine, l.ID.String(), l.RectificativaID.String(), l.ProductoID.String(), l.Cantidad, l.PrecioUnitario)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SQLSalesRepository) GetFacturaRectificativa(ctx context.Context, id uuid.UUID) (*domain.FacturaRectificativa, error) {
	var qFR string
	if r.isSQLite {
		qFR = `SELECT id, invoice_id, empresa_id, terminal_id, numero_fr, sequence_number, total, reason, status, created_at FROM facturas_rectificativas WHERE id = ?`
	} else {
		qFR = `SELECT id, invoice_id, empresa_id, terminal_id, numero_fr, sequence_number, total, reason, status, created_at FROM facturas_rectificativas WHERE id = $1`
	}

	var idStr, invIDStr, empIDStr, frNumber, reason, status string
	var termIDStr sql.NullString
	var seq int
	var createdAt time.Time
	var total float64

	err := r.db.QueryRowContext(ctx, qFR, id.String()).Scan(&idStr, &invIDStr, &empIDStr, &termIDStr, &frNumber, &seq, &total, &reason, &status, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrFacturaRectificativaNotFound
	} else if err != nil {
		return nil, err
	}

	// Fetch lines
	var qLines string
	if r.isSQLite {
		qLines = `SELECT id, factura_rectificativa_id, producto_id, cantidad, precio_unitario FROM factura_rectificativa_lineas WHERE factura_rectificativa_id = ?`
	} else {
		qLines = `SELECT id, factura_rectificativa_id, producto_id, cantidad, precio_unitario FROM factura_rectificativa_lineas WHERE factura_rectificativa_id = $1`
	}
	rows, err := r.db.QueryContext(ctx, qLines, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []domain.FacturaRectificativaLinea
	for rows.Next() {
		var lIDStr, frIDStr, prodIDStr string
		var qty, price float64
		if err := rows.Scan(&lIDStr, &frIDStr, &prodIDStr, &qty, &price); err != nil {
			return nil, err
		}
		lUUID, _ := uuid.Parse(lIDStr)
		frUUID, _ := uuid.Parse(frIDStr)
		prodUUID, _ := uuid.Parse(prodIDStr)
		lines = append(lines, domain.FacturaRectificativaLinea{
			ID:              lUUID,
			RectificativaID: frUUID,
			ProductoID:      prodUUID,
			Cantidad:        qty,
			PrecioUnitario:  price,
		})
	}

	var termUUIDPtr *uuid.UUID
	if termIDStr.Valid && termIDStr.String != "" {
		tUUID, _ := uuid.Parse(termIDStr.String)
		termUUIDPtr = &tUUID
	}

	frUUID, _ := uuid.Parse(idStr)
	invUUID, _ := uuid.Parse(invIDStr)
	empUUID, _ := uuid.Parse(empIDStr)

	return &domain.FacturaRectificativa{
		ID:             frUUID,
		FacturaID:      invUUID,
		EmpresaID:      empUUID,
		TerminalID:     termUUIDPtr,
		NumeroFR:       frNumber,
		NumeroSecuencia: seq,
		Total:          total,
		Motivo:         reason,
		Estado:         status,
		CreatedAt:      createdAt,
		Lines:          lines,
	}, nil
}

func (r *SQLSalesRepository) ListFacturasRectificativas(ctx context.Context, empresaID uuid.UUID) ([]domain.FacturaRectificativa, error) {
	var qFR string
	if r.isSQLite {
		qFR = `SELECT id, invoice_id, empresa_id, terminal_id, numero_fr, sequence_number, total, reason, status, created_at 
               FROM facturas_rectificativas WHERE empresa_id = ? ORDER BY created_at DESC`
	} else {
		qFR = `SELECT id, invoice_id, empresa_id, terminal_id, numero_fr, sequence_number, total, reason, status, created_at 
               FROM facturas_rectificativas WHERE empresa_id = $1 ORDER BY created_at DESC`
	}

	rows, err := r.db.QueryContext(ctx, qFR, empresaID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []domain.FacturaRectificativa
	for rows.Next() {
		var idStr, invIDStr, empIDStr, frNumber, reason, status string
		var termIDStr sql.NullString
		var seq int
		var createdAt time.Time
		var total float64
		if err := rows.Scan(&idStr, &invIDStr, &empIDStr, &termIDStr, &frNumber, &seq, &total, &reason, &status, &createdAt); err != nil {
			return nil, err
		}
		frUUID, _ := uuid.Parse(idStr)
		invUUID, _ := uuid.Parse(invIDStr)
		empUUID, _ := uuid.Parse(empIDStr)

		var termUUIDPtr *uuid.UUID
		if termIDStr.Valid && termIDStr.String != "" {
			tUUID, _ := uuid.Parse(termIDStr.String)
			termUUIDPtr = &tUUID
		}

		notes = append(notes, domain.FacturaRectificativa{
			ID:             frUUID,
			FacturaID:      invUUID,
			EmpresaID:      empUUID,
			TerminalID:     termUUIDPtr,
			NumeroFR:       frNumber,
			NumeroSecuencia: seq,
			Total:          total,
			Motivo:         reason,
			Estado:         status,
			CreatedAt:      createdAt,
		})
	}
	return notes, nil
}

func (r *SQLSalesRepository) UpdateFacturaRectifiedTotal(ctx context.Context, invoiceID uuid.UUID, rectifiedTotal float64) error {
	return r.updateExec(ctx, "facturas", map[string]interface{}{"total_rectificado": rectifiedTotal}, invoiceID, domain.ErrFacturaNotFound)
}

func (r *SQLSalesRepository) GetRectifiedQuantitiesByInvoice(ctx context.Context, invoiceID uuid.UUID) (map[uuid.UUID]float64, error) {
	var query string
	if r.isSQLite {
		query = `SELECT frl.producto_id, SUM(frl.cantidad)
				 FROM factura_rectificativa_lineas frl
				 JOIN facturas_rectificativas fr ON frl.factura_rectificativa_id = fr.id
				 WHERE fr.invoice_id = ?
				 GROUP BY frl.producto_id`
	} else {
		query = `SELECT frl.producto_id, SUM(frl.cantidad)
				 FROM factura_rectificativa_lineas frl
				 JOIN facturas_rectificativas fr ON frl.factura_rectificativa_id = fr.id
				 WHERE fr.invoice_id = $1
				 GROUP BY frl.producto_id`
	}

	rows, err := r.db.QueryContext(ctx, query, invoiceID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID]float64)
	for rows.Next() {
		var prodIDStr string
		var totalQty float64
		if err := rows.Scan(&prodIDStr, &totalQty); err != nil {
			return nil, err
		}
		prodUUID, _ := uuid.Parse(prodIDStr)
		result[prodUUID] = totalQty
	}
	return result, nil
}
