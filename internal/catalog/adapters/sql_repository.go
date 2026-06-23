package adapters

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"ferrowin/internal/catalog/domain"
	"github.com/google/uuid"
)

// SQLCatalogRepository implements domain.CatalogRepository using database/sql.
type SQLCatalogRepository struct {
	db       *sql.DB
	isSQLite bool
}

// NewSQLCatalogRepository creates a new SQLCatalogRepository.
func NewSQLCatalogRepository(db *sql.DB, isSQLite bool) *SQLCatalogRepository {
	return &SQLCatalogRepository{
		db:       db,
		isSQLite: isSQLite,
	}
}

func (r *SQLCatalogRepository) paramPlaceholder(idx int) string {
	if r.isSQLite {
		return "?"
	}
	return fmt.Sprintf("$%d", idx)
}

func (r *SQLCatalogRepository) updateExec(ctx context.Context, table string, sets map[string]interface{}, id uuid.UUID) error {
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
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// --- Tipos IVA ---

func (r *SQLCatalogRepository) CreateTipoIVA(ctx context.Context, t *domain.TipoIVA) error {
	var query string
	if r.isSQLite {
		query = `INSERT INTO tipos_iva (id, nombre, porcentaje, updated_at, activo) VALUES (?, ?, ?, ?, ?)`
	} else {
		query = `INSERT INTO tipos_iva (id, nombre, porcentaje, updated_at, activo) VALUES ($1, $2, $3, $4, $5)`
	}
	_, err := r.db.ExecContext(ctx, query, t.ID.String(), t.Nombre, t.Porcentaje, t.UpdatedAt, t.Activo)
	return err
}

func (r *SQLCatalogRepository) GetTipoIVA(ctx context.Context, id uuid.UUID) (*domain.TipoIVA, error) {
	var query string
	if r.isSQLite {
		query = `SELECT id, nombre, porcentaje, updated_at, activo FROM tipos_iva WHERE id = ?`
	} else {
		query = `SELECT id, nombre, porcentaje, updated_at, activo FROM tipos_iva WHERE id = $1`
	}
	var idStr, nombre string
	var porcentaje float64
	var updatedAt time.Time
	var activo bool
	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(&idStr, &nombre, &porcentaje, &updatedAt, &activo)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrTipoIVANotFound
	} else if err != nil {
		return nil, err
	}
	idUUID, _ := uuid.Parse(idStr)
	return &domain.TipoIVA{
		ID:         idUUID,
		Nombre:     nombre,
		Porcentaje: porcentaje,
		UpdatedAt:  updatedAt,
		Activo:     activo,
	}, nil
}

func (r *SQLCatalogRepository) ListTiposIVA(ctx context.Context) ([]domain.TipoIVA, error) {
	var query string
	if r.isSQLite {
		query = `SELECT id, nombre, porcentaje, updated_at, activo FROM tipos_iva ORDER BY nombre`
	} else {
		query = `SELECT id, nombre, porcentaje, updated_at, activo FROM tipos_iva ORDER BY nombre`
	}
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.TipoIVA
	for rows.Next() {
		var idStr, nombre string
		var porcentaje float64
		var updatedAt time.Time
		var activo bool
		if err := rows.Scan(&idStr, &nombre, &porcentaje, &updatedAt, &activo); err != nil {
			return nil, err
		}
		idUUID, _ := uuid.Parse(idStr)
		items = append(items, domain.TipoIVA{
			ID:         idUUID,
			Nombre:     nombre,
			Porcentaje: porcentaje,
			UpdatedAt:  updatedAt,
			Activo:     activo,
		})
	}
	return items, nil
}

func (r *SQLCatalogRepository) UpdateTipoIVA(ctx context.Context, t *domain.TipoIVA) error {
	sets := map[string]interface{}{
		"nombre":     t.Nombre,
		"porcentaje": t.Porcentaje,
		"updated_at": time.Now(),
	}
	return r.updateExec(ctx, "tipos_iva", sets, t.ID)
}

func (r *SQLCatalogRepository) DeleteTipoIVA(ctx context.Context, id uuid.UUID) error {
	sets := map[string]interface{}{
		"activo":     false,
		"updated_at": time.Now(),
	}
	return r.updateExec(ctx, "tipos_iva", sets, id)
}

// --- Familias ---

func (r *SQLCatalogRepository) CreateFamilia(ctx context.Context, f *domain.Familia) error {
	var query string
	if r.isSQLite {
		query = `INSERT INTO familias (id, nombre, updated_at, activo) VALUES (?, ?, ?, ?)`
	} else {
		query = `INSERT INTO familias (id, nombre, updated_at, activo) VALUES ($1, $2, $3, $4)`
	}
	_, err := r.db.ExecContext(ctx, query, f.ID.String(), f.Nombre, f.UpdatedAt, f.Activo)
	return err
}

func (r *SQLCatalogRepository) GetFamilia(ctx context.Context, id uuid.UUID) (*domain.Familia, error) {
	var query string
	if r.isSQLite {
		query = `SELECT id, nombre, updated_at, activo FROM familias WHERE id = ?`
	} else {
		query = `SELECT id, nombre, updated_at, activo FROM familias WHERE id = $1`
	}
	var idStr, nombre string
	var updatedAt time.Time
	var activo bool
	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(&idStr, &nombre, &updatedAt, &activo)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrFamiliaNotFound
	} else if err != nil {
		return nil, err
	}
	idUUID, _ := uuid.Parse(idStr)
	return &domain.Familia{
		ID:        idUUID,
		Nombre:    nombre,
		UpdatedAt: updatedAt,
		Activo:    activo,
	}, nil
}

func (r *SQLCatalogRepository) ListFamilias(ctx context.Context) ([]domain.Familia, error) {
	var query string
	if r.isSQLite {
		query = `SELECT id, nombre, updated_at, activo FROM familias ORDER BY nombre`
	} else {
		query = `SELECT id, nombre, updated_at, activo FROM familias ORDER BY nombre`
	}
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Familia
	for rows.Next() {
		var idStr, nombre string
		var updatedAt time.Time
		var activo bool
		if err := rows.Scan(&idStr, &nombre, &updatedAt, &activo); err != nil {
			return nil, err
		}
		idUUID, _ := uuid.Parse(idStr)
		items = append(items, domain.Familia{
			ID:        idUUID,
			Nombre:    nombre,
			UpdatedAt: updatedAt,
			Activo:    activo,
		})
	}
	return items, nil
}

func (r *SQLCatalogRepository) UpdateFamilia(ctx context.Context, f *domain.Familia) error {
	sets := map[string]interface{}{
		"nombre":     f.Nombre,
		"updated_at": time.Now(),
	}
	return r.updateExec(ctx, "familias", sets, f.ID)
}

func (r *SQLCatalogRepository) DeleteFamilia(ctx context.Context, id uuid.UUID) error {
	sets := map[string]interface{}{
		"activo":     false,
		"updated_at": time.Now(),
	}
	return r.updateExec(ctx, "familias", sets, id)
}

// --- Productos ---

func (r *SQLCatalogRepository) CreateProducto(ctx context.Context, p *domain.Producto) error {
	var query string
	if r.isSQLite {
		query = `INSERT INTO productos (id, codigo, nombre, precio_venta, familia_id, tipo_iva_id, updated_at, activo) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	} else {
		query = `INSERT INTO productos (id, codigo, nombre, precio_venta, familia_id, tipo_iva_id, updated_at, activo) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	}
	var familiaIDVal interface{}
	if p.FamiliaID != nil {
		familiaIDVal = p.FamiliaID.String()
	}
	_, err := r.db.ExecContext(ctx, query, p.ID.String(), p.Codigo, p.Nombre, p.PrecioVenta, familiaIDVal, p.TipoIvaID.String(), p.UpdatedAt, p.Activo)
	return err
}

func (r *SQLCatalogRepository) GetProducto(ctx context.Context, id uuid.UUID) (*domain.Producto, error) {
	var query string
	if r.isSQLite {
		query = `SELECT id, codigo, nombre, precio_venta, familia_id, tipo_iva_id, updated_at, activo FROM productos WHERE id = ?`
	} else {
		query = `SELECT id, codigo, nombre, precio_venta, familia_id, tipo_iva_id, updated_at, activo FROM productos WHERE id = $1`
	}
	var idStr, codigo, nombre string
	var precioVenta float64
	var familiaID sql.NullString
	var tipoIvaIDStr string
	var updatedAt time.Time
	var activo bool
	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(&idStr, &codigo, &nombre, &precioVenta, &familiaID, &tipoIvaIDStr, &updatedAt, &activo)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrProductoNotFound
	} else if err != nil {
		return nil, err
	}
	idUUID, _ := uuid.Parse(idStr)
	tipoIvaUUID, _ := uuid.Parse(tipoIvaIDStr)
	var familiaIDPtr *uuid.UUID
	if familiaID.Valid {
		fid, _ := uuid.Parse(familiaID.String)
		familiaIDPtr = &fid
	}
	return &domain.Producto{
		ID:          idUUID,
		Codigo:      codigo,
		Nombre:      nombre,
		PrecioVenta: precioVenta,
		FamiliaID:   familiaIDPtr,
		TipoIvaID:   tipoIvaUUID,
		UpdatedAt:   updatedAt,
		Activo:      activo,
	}, nil
}

func (r *SQLCatalogRepository) ListProductos(ctx context.Context, familiaID, tipoIvaID *uuid.UUID) ([]domain.Producto, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if familiaID != nil {
		where += fmt.Sprintf(" AND familia_id = %s", r.paramPlaceholder(argIdx))
		args = append(args, familiaID.String())
		argIdx++
	}
	if tipoIvaID != nil {
		where += fmt.Sprintf(" AND tipo_iva_id = %s", r.paramPlaceholder(argIdx))
		args = append(args, tipoIvaID.String())
		argIdx++
	}

	var query string
	if r.isSQLite {
		query = fmt.Sprintf("SELECT id, codigo, nombre, precio_venta, familia_id, tipo_iva_id, updated_at, activo FROM productos %s ORDER BY nombre", where)
	} else {
		query = fmt.Sprintf("SELECT id, codigo, nombre, precio_venta, familia_id, tipo_iva_id, updated_at, activo FROM productos %s ORDER BY nombre", where)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Producto
	for rows.Next() {
		var idStr, codigo, nombre string
		var precioVenta float64
		var familiaID sql.NullString
		var tipoIvaIDStr string
		var updatedAt time.Time
		var activo bool
		if err := rows.Scan(&idStr, &codigo, &nombre, &precioVenta, &familiaID, &tipoIvaIDStr, &updatedAt, &activo); err != nil {
			return nil, err
		}
		idUUID, _ := uuid.Parse(idStr)
		tipoIvaUUID, _ := uuid.Parse(tipoIvaIDStr)
		var familiaIDPtr *uuid.UUID
		if familiaID.Valid {
			fid, _ := uuid.Parse(familiaID.String)
			familiaIDPtr = &fid
		}
		items = append(items, domain.Producto{
			ID:          idUUID,
			Codigo:      codigo,
			Nombre:      nombre,
			PrecioVenta: precioVenta,
			FamiliaID:   familiaIDPtr,
			TipoIvaID:   tipoIvaUUID,
			UpdatedAt:   updatedAt,
			Activo:      activo,
		})
	}
	return items, nil
}

func (r *SQLCatalogRepository) UpdateProducto(ctx context.Context, p *domain.Producto) error {
	sets := map[string]interface{}{
		"codigo":       p.Codigo,
		"nombre":       p.Nombre,
		"precio_venta": p.PrecioVenta,
		"tipo_iva_id":  p.TipoIvaID.String(),
		"updated_at":   time.Now(),
	}
	if p.FamiliaID != nil {
		sets["familia_id"] = p.FamiliaID.String()
	} else {
		sets["familia_id"] = nil
	}
	return r.updateExec(ctx, "productos", sets, p.ID)
}

func (r *SQLCatalogRepository) DeleteProducto(ctx context.Context, id uuid.UUID) error {
	sets := map[string]interface{}{
		"activo":     false,
		"updated_at": time.Now(),
	}
	return r.updateExec(ctx, "productos", sets, id)
}

// --- Clientes ---

func (r *SQLCatalogRepository) CreateCliente(ctx context.Context, e *domain.Entidad) error {
	var query string
	if r.isSQLite {
		query = `INSERT INTO entidades (id, empresa_id, razon_social, nif, email, telefono, activo, roles, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	} else {
		query = `INSERT INTO entidades (id, empresa_id, razon_social, nif, email, telefono, activo, roles, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	}
	var emailVal, telefonoVal interface{}
	if e.Email != nil {
		emailVal = *e.Email
	}
	if e.Telefono != nil {
		telefonoVal = *e.Telefono
	}
	_, err := r.db.ExecContext(ctx, query, e.ID.String(), e.EmpresaID.String(), e.RazonSocial, e.NIF, emailVal, telefonoVal, e.Activo, e.Roles, e.UpdatedAt)
	return err
}

func (r *SQLCatalogRepository) GetCliente(ctx context.Context, id uuid.UUID) (*domain.Entidad, error) {
	var query string
	if r.isSQLite {
		query = `SELECT id, empresa_id, razon_social, nif, email, telefono, activo, roles, updated_at FROM entidades WHERE id = ? AND roles LIKE '%CLIENTE%'`
	} else {
		query = `SELECT id, empresa_id, razon_social, nif, email, telefono, activo, roles, updated_at FROM entidades WHERE id = $1 AND roles LIKE '%CLIENTE%'`
	}
	var idStr, empIDStr, razonSocial, nif, roles string
	var email, telefono sql.NullString
	var activo bool
	var updatedAt time.Time
	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(&idStr, &empIDStr, &razonSocial, &nif, &email, &telefono, &activo, &roles, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrClienteNotFound
	} else if err != nil {
		return nil, err
	}
	idUUID, _ := uuid.Parse(idStr)
	empUUID, _ := uuid.Parse(empIDStr)
	var emailPtr, telefonoPtr *string
	if email.Valid {
		emailPtr = &email.String
	}
	if telefono.Valid {
		telefonoPtr = &telefono.String
	}
	return &domain.Entidad{
		ID:          idUUID,
		EmpresaID:   empUUID,
		RazonSocial: razonSocial,
		NIF:         nif,
		Email:       emailPtr,
		Telefono:    telefonoPtr,
		Activo:      activo,
		Roles:       roles,
		UpdatedAt:   updatedAt,
	}, nil
}

func (r *SQLCatalogRepository) ListClientes(ctx context.Context, empresaID uuid.UUID) ([]domain.Entidad, error) {
	var query string
	if r.isSQLite {
		query = `SELECT id, empresa_id, razon_social, nif, email, telefono, activo, roles, updated_at FROM entidades WHERE empresa_id = ? AND roles LIKE '%CLIENTE%' ORDER BY razon_social`
	} else {
		query = `SELECT id, empresa_id, razon_social, nif, email, telefono, activo, roles, updated_at FROM entidades WHERE empresa_id = $1 AND roles LIKE '%CLIENTE%' ORDER BY razon_social`
	}
	rows, err := r.db.QueryContext(ctx, query, empresaID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Entidad
	for rows.Next() {
		var idStr, empIDStr, razonSocial, nif, roles string
		var email, telefono sql.NullString
		var activo bool
		var updatedAt time.Time
		if err := rows.Scan(&idStr, &empIDStr, &razonSocial, &nif, &email, &telefono, &activo, &roles, &updatedAt); err != nil {
			return nil, err
		}
		idUUID, _ := uuid.Parse(idStr)
		empUUID, _ := uuid.Parse(empIDStr)
		var emailPtr, telefonoPtr *string
		if email.Valid {
			emailPtr = &email.String
		}
		if telefono.Valid {
			telefonoPtr = &telefono.String
		}
		items = append(items, domain.Entidad{
			ID:          idUUID,
			EmpresaID:   empUUID,
			RazonSocial: razonSocial,
			NIF:         nif,
			Email:       emailPtr,
			Telefono:    telefonoPtr,
			Activo:      activo,
			Roles:       roles,
			UpdatedAt:   updatedAt,
		})
	}
	return items, nil
}

func (r *SQLCatalogRepository) UpdateCliente(ctx context.Context, e *domain.Entidad) error {
	sets := map[string]interface{}{
		"razon_social": e.RazonSocial,
		"nif":          e.NIF,
		"updated_at":   time.Now(),
	}
	if e.Email != nil {
		sets["email"] = *e.Email
	} else {
		sets["email"] = nil
	}
	if e.Telefono != nil {
		sets["telefono"] = *e.Telefono
	} else {
		sets["telefono"] = nil
	}
	return r.updateExec(ctx, "entidades", sets, e.ID)
}

func (r *SQLCatalogRepository) DeleteCliente(ctx context.Context, id uuid.UUID) error {
	sets := map[string]interface{}{
		"activo":     false,
		"updated_at": time.Now(),
	}
	return r.updateExec(ctx, "entidades", sets, id)
}
