package adapters

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"ferrowin/internal/purchases/domain"
	"github.com/google/uuid"
)

type SQLPurchaseRepository struct {
	db       *sql.DB
	isSQLite bool
}

func NewSQLPurchaseRepository(db *sql.DB, isSQLite bool) *SQLPurchaseRepository {
	return &SQLPurchaseRepository{
		db:       db,
		isSQLite: isSQLite,
	}
}

func (r *SQLPurchaseRepository) SaveCompany(ctx context.Context, c *domain.Empresa) error {
	var query string
	if r.isSQLite {
		query = `INSERT INTO empresas (id, razon_social, nif, activa) 
                 VALUES (?, ?, ?, ?)
                 ON CONFLICT(id) DO UPDATE SET razon_social=excluded.razon_social, nif=excluded.nif, activa=excluded.activa`
	} else {
		query = `INSERT INTO empresas (id, razon_social, nif, activa) 
                 VALUES ($1, $2, $3, $4)
                 ON CONFLICT(id) DO UPDATE SET razon_social=EXCLUDED.razon_social, nif=EXCLUDED.nif, activa=EXCLUDED.activa`
	}
	_, err := r.db.ExecContext(ctx, query, c.ID.String(), c.RazonSocial, c.NIF, c.Activa)
	return err
}

func (r *SQLPurchaseRepository) SaveWarehouse(ctx context.Context, w *domain.Warehouse) error {
	var query string
	if r.isSQLite {
		query = `INSERT INTO warehouses (id, empresa_id, name, active) 
                 VALUES (?, ?, ?, ?)
                 ON CONFLICT(id) DO UPDATE SET name=excluded.name, active=excluded.active`
	} else {
		query = `INSERT INTO warehouses (id, empresa_id, name, active) 
                 VALUES ($1, $2, $3, $4)
                 ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name, active=EXCLUDED.active`
	}
	_, err := r.db.ExecContext(ctx, query, w.ID.String(), w.EmpresaID.String(), w.Name, w.Active)
	return err
}

func (r *SQLPurchaseRepository) paramPlaceholder(idx int) string {
	if r.isSQLite {
		return "?"
	}
	return fmt.Sprintf("$%d", idx)
}

func (r *SQLPurchaseRepository) GetCompanies(ctx context.Context) ([]*domain.Empresa, error) {
	var query string
	if r.isSQLite {
		query = `SELECT id, razon_social, nif, activa FROM empresas`
	} else {
		query = `SELECT id, razon_social, nif, activa FROM empresas`
	}
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var companies []*domain.Empresa
	for rows.Next() {
		var idStr, razonSocial, nif string
		var activa bool
		if err := rows.Scan(&idStr, &razonSocial, &nif, &activa); err != nil {
			return nil, err
		}
		idUUID, _ := uuid.Parse(idStr)
		companies = append(companies, &domain.Empresa{
			ID:          idUUID,
			RazonSocial: razonSocial,
			NIF:         nif,
			Activa:      activa,
		})
	}
	return companies, nil
}

func (r *SQLPurchaseRepository) GetAllWarehouses(ctx context.Context, empresaID uuid.UUID) ([]*domain.Warehouse, error) {
	var query string
	if r.isSQLite {
		query = `SELECT id, empresa_id, name, active FROM warehouses WHERE empresa_id = ?`
	} else {
		query = `SELECT id, empresa_id, name, active FROM warehouses WHERE empresa_id = $1`
	}
	rows, err := r.db.QueryContext(ctx, query, empresaID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var warehouses []*domain.Warehouse
	for rows.Next() {
		var idStr, empIDStr, name string
		var active bool
		if err := rows.Scan(&idStr, &empIDStr, &name, &active); err != nil {
			return nil, err
		}
		whUUID, _ := uuid.Parse(idStr)
		empUUID, _ := uuid.Parse(empIDStr)
		warehouses = append(warehouses, &domain.Warehouse{
			ID:        whUUID,
			EmpresaID: empUUID,
			Name:      name,
			Active:    active,
		})
	}
	return warehouses, nil
}

func (r *SQLPurchaseRepository) GetWarehouse(ctx context.Context, id uuid.UUID) (*domain.Warehouse, error) {
	var query string
	if r.isSQLite {
		query = `SELECT id, empresa_id, name, active FROM warehouses WHERE id = ?`
	} else {
		query = `SELECT id, empresa_id, name, active FROM warehouses WHERE id = $1`
	}
	var idStr, empIDStr, name string
	var active bool
	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(&idStr, &empIDStr, &name, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrWarehouseNotFound
	} else if err != nil {
		return nil, err
	}
	whUUID, _ := uuid.Parse(idStr)
	empUUID, _ := uuid.Parse(empIDStr)
	return &domain.Warehouse{
		ID:        whUUID,
		EmpresaID: empUUID,
		Name:      name,
		Active:    active,
	}, nil
}

func (r *SQLPurchaseRepository) SaveSupplier(ctx context.Context, s *domain.Proveedor) error {
	var query string
	if r.isSQLite {
		query = `INSERT INTO entidades (id, empresa_id, razon_social, nif, email, telefono, roles, activo) 
                 VALUES (?, ?, ?, ?, ?, ?, 'PROVEEDOR', ?)
                 ON CONFLICT(id) DO UPDATE SET razon_social=excluded.razon_social, nif=excluded.nif, email=excluded.email, telefono=excluded.telefono, activo=excluded.activo`
	} else {
		query = `INSERT INTO entidades (id, empresa_id, razon_social, nif, email, telefono, roles, activo) 
                 VALUES ($1, $2, $3, $4, $5, $6, 'PROVEEDOR', $7)
                 ON CONFLICT(id) DO UPDATE SET razon_social=EXCLUDED.razon_social, nif=EXCLUDED.nif, email=EXCLUDED.email, telefono=EXCLUDED.telefono, activo=EXCLUDED.activo`
	}
	_, err := r.db.ExecContext(ctx, query, s.ID.String(), s.EmpresaID.String(), s.RazonSocial, s.CIF, s.Email, s.Telefono, s.Activo)
	return err
}

func (r *SQLPurchaseRepository) GetSuppliers(ctx context.Context, empresaID uuid.UUID) ([]*domain.Proveedor, error) {
	var query string
	if r.isSQLite {
		query = `SELECT id, empresa_id, razon_social, nif, email, telefono, activo FROM entidades WHERE empresa_id = ? AND roles LIKE '%PROVEEDOR%'`
	} else {
		query = `SELECT id, empresa_id, razon_social, nif, email, telefono, activo FROM entidades WHERE empresa_id = $1 AND roles LIKE '%PROVEEDOR%'`
	}
	rows, err := r.db.QueryContext(ctx, query, empresaID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suppliers []*domain.Proveedor
	for rows.Next() {
		var idStr, empIDStr, razonSocial, cif string
		var email, telefono sql.NullString
		var activo bool
		if err := rows.Scan(&idStr, &empIDStr, &razonSocial, &cif, &email, &telefono, &activo); err != nil {
			return nil, err
		}
		idUUID, _ := uuid.Parse(idStr)
		empUUID, _ := uuid.Parse(empIDStr)
		suppliers = append(suppliers, &domain.Proveedor{
			ID:          idUUID,
			EmpresaID:   empUUID,
			RazonSocial: razonSocial,
			CIF:         cif,
			Email:       email.String,
			Telefono:    telefono.String,
			Direccion:   "",
			Activo:      activo,
		})
	}
	return suppliers, nil
}

func (r *SQLPurchaseRepository) GetSupplier(ctx context.Context, id uuid.UUID) (*domain.Proveedor, error) {
	var query string
	if r.isSQLite {
		query = `SELECT id, empresa_id, razon_social, nif, email, telefono, activo FROM entidades WHERE id = ? AND roles LIKE '%PROVEEDOR%'`
	} else {
		query = `SELECT id, empresa_id, razon_social, nif, email, telefono, activo FROM entidades WHERE id = $1 AND roles LIKE '%PROVEEDOR%'`
	}
	var idStr, empIDStr, razonSocial, cif string
	var email, telefono sql.NullString
	var activo bool
	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(&idStr, &empIDStr, &razonSocial, &cif, &email, &telefono, &activo)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrSupplierNotFound
	} else if err != nil {
		return nil, err
	}
	idUUID, _ := uuid.Parse(idStr)
	empUUID, _ := uuid.Parse(empIDStr)
	return &domain.Proveedor{
		ID:          idUUID,
		EmpresaID:   empUUID,
		RazonSocial: razonSocial,
		CIF:         cif,
		Email:       email.String,
		Telefono:    telefono.String,
		Direccion:   "",
		Activo:      activo,
	}, nil
}

func (r *SQLPurchaseRepository) updateExec(ctx context.Context, table string, sets map[string]interface{}, id uuid.UUID, notFoundErr error) error {
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

func (r *SQLPurchaseRepository) UpdateCompany(ctx context.Context, input domain.UpdateCompanyInput) error {
	sets := make(map[string]interface{})
	if input.RazonSocial != nil {
		sets["razon_social"] = *input.RazonSocial
	}
	if input.NIF != nil {
		sets["nif"] = *input.NIF
	}
	return r.updateExec(ctx, "empresas", sets, input.ID, domain.ErrCompanyNotFound)
}

func (r *SQLPurchaseRepository) UpdateWarehouse(ctx context.Context, input domain.UpdateWarehouseInput) error {
	sets := make(map[string]interface{})
	if input.Name != nil {
		sets["name"] = *input.Name
	}
	if input.Active != nil {
		sets["active"] = *input.Active
	}
	return r.updateExec(ctx, "warehouses", sets, input.ID, domain.ErrWarehouseNotFound)
}

func (r *SQLPurchaseRepository) UpdateSupplier(ctx context.Context, input domain.UpdateSupplierInput) error {
	sets := make(map[string]interface{})
	if input.RazonSocial != nil {
		sets["razon_social"] = *input.RazonSocial
	}
	if input.CIF != nil {
		sets["nif"] = *input.CIF
	}
	if input.Email != nil {
		sets["email"] = *input.Email
	}
	if input.Telefono != nil {
		sets["telefono"] = *input.Telefono
	}
	if input.Activo != nil {
		sets["activo"] = *input.Activo
	}
	return r.updateExec(ctx, "entidades", sets, input.ID, domain.ErrSupplierNotFound)
}

func (r *SQLPurchaseRepository) CancelPurchaseOrder(ctx context.Context, id uuid.UUID) error {
	return r.updateExec(ctx, "pedidos_compra", map[string]interface{}{"estado": "Cancelado"}, id, domain.ErrPurchaseOrderNotFound)
}

func (r *SQLPurchaseRepository) CancelPurchaseReceipt(ctx context.Context, id uuid.UUID) error {
	return r.updateExec(ctx, "recepciones_compra", map[string]interface{}{"estado": "Cancelado"}, id, domain.ErrPurchaseReceiptNotFound)
}

func (r *SQLPurchaseRepository) SavePurchaseOrder(ctx context.Context, o *domain.PedidoCompra) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var qOrder string
	if r.isSQLite {
		qOrder = `INSERT INTO pedidos_compra (id, empresa_id, proveedor_id, numero_pedido, fecha, estado, total) 
                  VALUES (?, ?, ?, ?, ?, ?, ?)
                  ON CONFLICT(id) DO UPDATE SET estado=excluded.estado, total=excluded.total`
	} else {
		qOrder = `INSERT INTO pedidos_compra (id, empresa_id, proveedor_id, numero_pedido, fecha, estado, total) 
                  VALUES ($1, $2, $3, $4, $5, $6, $7)
                  ON CONFLICT(id) DO UPDATE SET estado=EXCLUDED.estado, total=EXCLUDED.total`
	}

	_, err = tx.ExecContext(ctx, qOrder, o.ID.String(), o.EmpresaID.String(), o.ProveedorID.String(), o.NumeroPedido, o.Fecha.UTC(), o.Estado, o.Total)
	if err != nil {
		return err
	}

	// Delete existing lines if updating
	var qDelete string
	if r.isSQLite {
		qDelete = `DELETE FROM pedido_compra_lineas WHERE pedido_compra_id = ?`
	} else {
		qDelete = `DELETE FROM pedido_compra_lineas WHERE pedido_compra_id = $1`
	}
	_, err = tx.ExecContext(ctx, qDelete, o.ID.String())
	if err != nil {
		return err
	}

	// Insert lines
	var qLine string
	if r.isSQLite {
		qLine = `INSERT INTO pedido_compra_lineas (id, pedido_compra_id, producto_id, cantidad, precio_unitario) 
                 VALUES (?, ?, ?, ?, ?)`
	} else {
		qLine = `INSERT INTO pedido_compra_lineas (id, pedido_compra_id, producto_id, cantidad, precio_unitario) 
                 VALUES ($1, $2, $3, $4, $5)`
	}
	for _, l := range o.Lineas {
		_, err = tx.ExecContext(ctx, qLine, l.ID.String(), l.PedidoCompraID.String(), l.ProductoID.String(), l.Cantidad, l.PrecioUnitario)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SQLPurchaseRepository) GetPurchaseOrder(ctx context.Context, id uuid.UUID) (*domain.PedidoCompra, error) {
	var qOrder string
	if r.isSQLite {
		qOrder = `SELECT id, empresa_id, proveedor_id, numero_pedido, fecha, estado, total FROM pedidos_compra WHERE id = ?`
	} else {
		qOrder = `SELECT id, empresa_id, proveedor_id, numero_pedido, fecha, estado, total FROM pedidos_compra WHERE id = $1`
	}

	var idStr, empIDStr, provIDStr, numeroPedido, estado string
	var fecha time.Time
	var total float64

	err := r.db.QueryRowContext(ctx, qOrder, id.String()).Scan(&idStr, &empIDStr, &provIDStr, &numeroPedido, &fecha, &estado, &total)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPurchaseOrderNotFound
	} else if err != nil {
		return nil, err
	}

	// Fetch lines
	var qLines string
	if r.isSQLite {
		qLines = `SELECT id, pedido_compra_id, producto_id, cantidad, precio_unitario FROM pedido_compra_lineas WHERE pedido_compra_id = ?`
	} else {
		qLines = `SELECT id, pedido_compra_id, producto_id, cantidad, precio_unitario FROM pedido_compra_lineas WHERE pedido_compra_id = $1`
	}
	rows, err := r.db.QueryContext(ctx, qLines, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []domain.PedidoCompraLinea
	for rows.Next() {
		var lIDStr, poIDStr, prodIDStr string
		var qty, price float64
		if err := rows.Scan(&lIDStr, &poIDStr, &prodIDStr, &qty, &price); err != nil {
			return nil, err
		}
		lUUID, _ := uuid.Parse(lIDStr)
		poUUID, _ := uuid.Parse(poIDStr)
		prodUUID, _ := uuid.Parse(prodIDStr)
		lines = append(lines, domain.PedidoCompraLinea{
			ID:             lUUID,
			PedidoCompraID: poUUID,
			ProductoID:     prodUUID,
			Cantidad:       qty,
			PrecioUnitario: price,
		})
	}

	poUUID, _ := uuid.Parse(idStr)
	empUUID, _ := uuid.Parse(empIDStr)
	provUUID, _ := uuid.Parse(provIDStr)

	return &domain.PedidoCompra{
		ID:           poUUID,
		EmpresaID:    empUUID,
		ProveedorID:  provUUID,
		NumeroPedido: numeroPedido,
		Fecha:        fecha,
		Estado:       estado,
		Total:        total,
		Lineas:       lines,
	}, nil
}

func (r *SQLPurchaseRepository) ListPurchaseOrders(ctx context.Context, empresaID uuid.UUID, filter domain.PurchaseOrderFilter) ([]*domain.PedidoCompra, int, error) {
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
	if filter.ProveedorID != nil {
		where += fmt.Sprintf(" AND proveedor_id = %s", r.paramPlaceholder(argIdx))
		args = append(args, filter.ProveedorID.String())
		argIdx++
	}
	if filter.Desde != nil {
		where += fmt.Sprintf(" AND fecha >= %s", r.paramPlaceholder(argIdx))
		args = append(args, filter.Desde.UTC())
		argIdx++
	}
	if filter.Hasta != nil {
		where += fmt.Sprintf(" AND fecha <= %s", r.paramPlaceholder(argIdx))
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
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM pedidos_compra %s", where)
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Data query
	var dataQuery string
	if r.isSQLite {
		dataQuery = fmt.Sprintf("SELECT id, empresa_id, proveedor_id, numero_pedido, fecha, estado, total FROM pedidos_compra %s ORDER BY fecha DESC LIMIT ? OFFSET ?", where)
	} else {
		dataQuery = fmt.Sprintf("SELECT id, empresa_id, proveedor_id, numero_pedido, fecha, estado, total FROM pedidos_compra %s ORDER BY fecha DESC LIMIT $%d OFFSET $%d", where, argIdx, argIdx+1)
	}
	dataArgs := append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []*domain.PedidoCompra
	for rows.Next() {
		var idStr, empIDStr, provIDStr, numeroPedido, estado string
		var fecha time.Time
		var total float64
		if err := rows.Scan(&idStr, &empIDStr, &provIDStr, &numeroPedido, &fecha, &estado, &total); err != nil {
			return nil, 0, err
		}
		poUUID, _ := uuid.Parse(idStr)
		empUUID, _ := uuid.Parse(empIDStr)
		provUUID, _ := uuid.Parse(provIDStr)
		orders = append(orders, &domain.PedidoCompra{
			ID:           poUUID,
			EmpresaID:    empUUID,
			ProveedorID:  provUUID,
			NumeroPedido: numeroPedido,
			Fecha:        fecha,
			Estado:       estado,
			Total:        total,
		})
	}
	return orders, total, nil
}

func (r *SQLPurchaseRepository) SavePurchaseReceipt(ctx context.Context, receipt *domain.RecepcionCompra) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var qReceipt string
	var poIDVal interface{}
	if receipt.PedidoCompraID != nil {
		poIDVal = receipt.PedidoCompraID.String()
	} else {
		poIDVal = nil
	}

	if r.isSQLite {
		qReceipt = `INSERT INTO recepciones_compra (id, empresa_id, pedido_compra_id, proveedor_id, numero_albaran, fecha, estado, warehouse_id) 
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                    ON CONFLICT(id) DO UPDATE SET estado=excluded.estado`
	} else {
		qReceipt = `INSERT INTO recepciones_compra (id, empresa_id, pedido_compra_id, proveedor_id, numero_albaran, fecha, estado, warehouse_id) 
                    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
                    ON CONFLICT(id) DO UPDATE SET estado=EXCLUDED.estado`
	}

	_, err = tx.ExecContext(ctx, qReceipt,
		receipt.ID.String(),
		receipt.EmpresaID.String(),
		poIDVal,
		receipt.ProveedorID.String(),
		receipt.NumeroAlbaran,
		receipt.Fecha.UTC(),
		receipt.Estado,
		receipt.WarehouseID.String(),
	)
	if err != nil {
		return err
	}

	// Delete existing lines if updating
	var qDelete string
	if r.isSQLite {
		qDelete = `DELETE FROM recepcion_compra_lineas WHERE recepcion_compra_id = ?`
	} else {
		qDelete = `DELETE FROM recepcion_compra_lineas WHERE recepcion_compra_id = $1`
	}
	_, err = tx.ExecContext(ctx, qDelete, receipt.ID.String())
	if err != nil {
		return err
	}

	// Insert lines
	var qLine string
	if r.isSQLite {
		qLine = `INSERT INTO recepcion_compra_lineas (id, recepcion_compra_id, producto_id, cantidad, precio_unitario) 
                 VALUES (?, ?, ?, ?, ?)`
	} else {
		qLine = `INSERT INTO recepcion_compra_lineas (id, recepcion_compra_id, producto_id, cantidad, precio_unitario) 
                 VALUES ($1, $2, $3, $4, $5)`
	}
	for _, l := range receipt.Lineas {
		_, err = tx.ExecContext(ctx, qLine, l.ID.String(), l.RecepcionCompraID.String(), l.ProductoID.String(), l.Cantidad, l.PrecioUnitario)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SQLPurchaseRepository) GetPurchaseReceipt(ctx context.Context, id uuid.UUID) (*domain.RecepcionCompra, error) {
	var qReceipt string
	if r.isSQLite {
		qReceipt = `SELECT id, empresa_id, pedido_compra_id, proveedor_id, numero_albaran, fecha, estado, warehouse_id FROM recepciones_compra WHERE id = ?`
	} else {
		qReceipt = `SELECT id, empresa_id, pedido_compra_id, proveedor_id, numero_albaran, fecha, estado, warehouse_id FROM recepciones_compra WHERE id = $1`
	}

	var idStr, empIDStr, provIDStr, numeroAlbaran, estado, whIDStr string
	var poIDStr sql.NullString
	var fecha time.Time

	err := r.db.QueryRowContext(ctx, qReceipt, id.String()).Scan(&idStr, &empIDStr, &poIDStr, &provIDStr, &numeroAlbaran, &fecha, &estado, &whIDStr)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPurchaseReceiptNotFound
	} else if err != nil {
		return nil, err
	}

	// Fetch lines
	var qLines string
	if r.isSQLite {
		qLines = `SELECT id, recepcion_compra_id, producto_id, cantidad, precio_unitario FROM recepcion_compra_lineas WHERE recepcion_compra_id = ?`
	} else {
		qLines = `SELECT id, recepcion_compra_id, producto_id, cantidad, precio_unitario FROM recepcion_compra_lineas WHERE recepcion_compra_id = $1`
	}
	rows, err := r.db.QueryContext(ctx, qLines, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []domain.RecepcionCompraLinea
	for rows.Next() {
		var lIDStr, rcIDStr, prodIDStr string
		var qty, price float64
		if err := rows.Scan(&lIDStr, &rcIDStr, &prodIDStr, &qty, &price); err != nil {
			return nil, err
		}
		lUUID, _ := uuid.Parse(lIDStr)
		rcUUID, _ := uuid.Parse(rcIDStr)
		prodUUID, _ := uuid.Parse(prodIDStr)
		lines = append(lines, domain.RecepcionCompraLinea{
			ID:                lUUID,
			RecepcionCompraID: rcUUID,
			ProductoID:        prodUUID,
			Cantidad:          qty,
			PrecioUnitario:    price,
		})
	}

	var poUUIDPtr *uuid.UUID
	if poIDStr.Valid && poIDStr.String != "" {
		pUUID, _ := uuid.Parse(poIDStr.String)
		poUUIDPtr = &pUUID
	}

	rcUUID, _ := uuid.Parse(idStr)
	empUUID, _ := uuid.Parse(empIDStr)
	provUUID, _ := uuid.Parse(provIDStr)
	whUUID, _ := uuid.Parse(whIDStr)

	return &domain.RecepcionCompra{
		ID:             rcUUID,
		EmpresaID:      empUUID,
		PedidoCompraID: poUUIDPtr,
		ProveedorID:    provUUID,
		NumeroAlbaran:  numeroAlbaran,
		Fecha:          fecha,
		Estado:         estado,
		WarehouseID:    whUUID,
		Lineas:         lines,
	}, nil
}

func (r *SQLPurchaseRepository) ListPurchaseReceipts(ctx context.Context, empresaID uuid.UUID, filter domain.PurchaseReceiptFilter) ([]*domain.RecepcionCompra, int, error) {
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
		where += fmt.Sprintf(" AND fecha >= %s", r.paramPlaceholder(argIdx))
		args = append(args, filter.Desde.UTC())
		argIdx++
	}
	if filter.Hasta != nil {
		where += fmt.Sprintf(" AND fecha <= %s", r.paramPlaceholder(argIdx))
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
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM recepciones_compra %s", where)
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	var dataQuery string
	if r.isSQLite {
		dataQuery = fmt.Sprintf("SELECT id, empresa_id, pedido_compra_id, proveedor_id, numero_albaran, fecha, estado, warehouse_id FROM recepciones_compra %s ORDER BY fecha DESC LIMIT ? OFFSET ?", where)
	} else {
		dataQuery = fmt.Sprintf("SELECT id, empresa_id, pedido_compra_id, proveedor_id, numero_albaran, fecha, estado, warehouse_id FROM recepciones_compra %s ORDER BY fecha DESC LIMIT $%d OFFSET $%d", where, argIdx, argIdx+1)
	}
	dataArgs := append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var receipts []*domain.RecepcionCompra
	for rows.Next() {
		var idStr, empIDStr, provIDStr, numeroAlbaran, estado, whIDStr string
		var poIDStr sql.NullString
		var fecha time.Time
		if err := rows.Scan(&idStr, &empIDStr, &poIDStr, &provIDStr, &numeroAlbaran, &fecha, &estado, &whIDStr); err != nil {
			return nil, 0, err
		}
		rcUUID, _ := uuid.Parse(idStr)
		empUUID, _ := uuid.Parse(empIDStr)
		provUUID, _ := uuid.Parse(provIDStr)
		whUUID, _ := uuid.Parse(whIDStr)

		var poUUIDPtr *uuid.UUID
		if poIDStr.Valid && poIDStr.String != "" {
			pUUID, _ := uuid.Parse(poIDStr.String)
			poUUIDPtr = &pUUID
		}

		receipts = append(receipts, &domain.RecepcionCompra{
			ID:             rcUUID,
			EmpresaID:      empUUID,
			PedidoCompraID: poUUIDPtr,
			ProveedorID:    provUUID,
			NumeroAlbaran:  numeroAlbaran,
			Fecha:          fecha,
			Estado:         estado,
			WarehouseID:    whUUID,
		})
	}
	return receipts, total, nil
}
