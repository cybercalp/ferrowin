package adapters

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"ferrowin/internal/shared/idempotency"
)

type CatalogSyncController struct {
	db           *sql.DB
	isSQLite     bool
	idempTracker *idempotency.Tracker
}

func NewCatalogSyncController(db *sql.DB, isSQLite bool, idempTracker *idempotency.Tracker) *CatalogSyncController {
	return &CatalogSyncController{
		db:           db,
		isSQLite:     isSQLite,
		idempTracker: idempTracker,
	}
}

func (c *CatalogSyncController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1/catalog/sync" {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		c.HandleCatalogSync(w, r)
		return
	}
	if r.URL.Path == "/api/v1/catalog/clients/dossier" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		c.HandleClientDossier(w, r)
		return
	}
	if r.URL.Path == "/api/v1/sync/payments" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		c.HandleSyncPayments(w, r)
		return
	}
	http.NotFound(w, r)
}

type TipoIVAResponse struct {
	ID         string    `json:"id"`
	Nombre     string    `json:"nombre"`
	Porcentaje float64   `json:"porcentaje"`
	UpdatedAt  time.Time `json:"updated_at"`
	Activo     bool      `json:"activo"`
}

type FamiliaResponse struct {
	ID        string    `json:"id"`
	Nombre    string    `json:"nombre"`
	UpdatedAt time.Time `json:"updated_at"`
	Activo    bool      `json:"activo"`
}

type ProductoResponse struct {
	ID          string    `json:"id"`
	Codigo      string    `json:"codigo"`
	Nombre      string    `json:"nombre"`
	PrecioVenta float64   `json:"precio_venta"`
	FamiliaID   *string   `json:"familia_id"`
	TipoIvaID   string    `json:"tipo_iva_id"`
	UpdatedAt   time.Time `json:"updated_at"`
	Activo      bool      `json:"activo"`
}

type ClienteResponse struct {
	ID        string    `json:"id"`
	Nombre    string    `json:"nombre"`
	NIF       *string   `json:"nif"`
	Email     *string   `json:"email"`
	UpdatedAt time.Time `json:"updated_at"`
	Activo    bool      `json:"activo"`
}

type EliminadosResponse struct {
	Productos []string `json:"productos"`
	Clientes  []string `json:"clientes"`
	Familias  []string `json:"familias"`
	TiposIVA  []string `json:"tipos_iva"`
}

type CatalogSyncResponse struct {
	TiposIVA   []TipoIVAResponse   `json:"tipos_iva"`
	Familias   []FamiliaResponse   `json:"familias"`
	Productos  []ProductoResponse  `json:"productos"`
	Clientes   []ClienteResponse   `json:"clientes"`
	Eliminados EliminadosResponse  `json:"eliminados"`
}

func parseTime(val interface{}) time.Time {
	if val == nil {
		return time.Time{}
	}
	switch t := val.(type) {
	case time.Time:
		return t
	case string:
		for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05Z", "2006-01-02T15:04:05-07:00"} {
			if p, err := time.Parse(layout, t); err == nil {
				return p
			}
		}
	}
	return time.Time{}
}

func parseBool(val interface{}) bool {
	if val == nil {
		return false
	}
	switch b := val.(type) {
	case bool:
		return b
	case int64:
		return b != 0
	case int:
		return b != 0
	}
	return false
}

func (c *CatalogSyncController) HandleCatalogSync(w http.ResponseWriter, r *http.Request) {
	sinceVal := r.URL.Query().Get("since")
	var (
		ivaQuery, familiaQuery, productoQuery, clienteQuery string
		ivaElim, familiaElim, productoElim, clienteElim     string
		args                                                []interface{}
	)

	var activeCond string
	var inactiveCond string

	if c.isSQLite {
		activeCond = "activo = 1"
		inactiveCond = "activo = 0"
	} else {
		activeCond = "activo = true"
		inactiveCond = "activo = false"
	}

	if sinceVal != "" {
		parsedSince, err := time.Parse(time.RFC3339, sinceVal)
		if err != nil {
			http.Error(w, "invalid since parameter layout, must be RFC3339", http.StatusBadRequest)
			return
		}
		var placeholder string
		if c.isSQLite {
			placeholder = "?"
			args = append(args, parsedSince.Format(time.RFC3339))
		} else {
			placeholder = "$1"
			args = append(args, parsedSince)
		}

		ivaQuery = "SELECT id, nombre, porcentaje, updated_at, activo FROM tipos_iva WHERE " + activeCond + " AND updated_at > " + placeholder
		familiaQuery = "SELECT id, nombre, updated_at, activo FROM familias WHERE " + activeCond + " AND updated_at > " + placeholder
		productoQuery = "SELECT id, codigo, nombre, precio_venta, familia_id, tipo_iva_id, updated_at, activo FROM productos WHERE " + activeCond + " AND updated_at > " + placeholder
		clienteQuery = "SELECT id, razon_social AS nombre, nif, email, updated_at, activo FROM entidades WHERE " + activeCond + " AND roles LIKE '%CLIENTE%' AND updated_at > " + placeholder

		ivaElim = "SELECT id FROM tipos_iva WHERE " + inactiveCond + " AND updated_at > " + placeholder
		familiaElim = "SELECT id FROM familias WHERE " + inactiveCond + " AND updated_at > " + placeholder
		productoElim = "SELECT id FROM productos WHERE " + inactiveCond + " AND updated_at > " + placeholder
		clienteElim = "SELECT id FROM entidades WHERE " + inactiveCond + " AND roles LIKE '%CLIENTE%' AND updated_at > " + placeholder
	} else {
		ivaQuery = "SELECT id, nombre, porcentaje, updated_at, activo FROM tipos_iva WHERE " + activeCond
		familiaQuery = "SELECT id, nombre, updated_at, activo FROM familias WHERE " + activeCond
		productoQuery = "SELECT id, codigo, nombre, precio_venta, familia_id, tipo_iva_id, updated_at, activo FROM productos WHERE " + activeCond
		clienteQuery = "SELECT id, razon_social AS nombre, nif, email, updated_at, activo FROM entidades WHERE " + activeCond + " AND roles LIKE '%CLIENTE%'"

		ivaElim = "SELECT id FROM tipos_iva WHERE " + inactiveCond
		familiaElim = "SELECT id FROM familias WHERE " + inactiveCond
		productoElim = "SELECT id FROM productos WHERE " + inactiveCond
		clienteElim = "SELECT id FROM entidades WHERE " + inactiveCond + " AND roles LIKE '%CLIENTE%'"
	}

	// query tipos_iva
	rows, err := c.db.QueryContext(r.Context(), ivaQuery, args...)
	if err != nil {
		http.Error(w, "query tipos_iva error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var tiposIVA []TipoIVAResponse
	for rows.Next() {
		var item TipoIVAResponse
		var updatedAtRaw interface{}
		var activoRaw interface{}
		if err := rows.Scan(&item.ID, &item.Nombre, &item.Porcentaje, &updatedAtRaw, &activoRaw); err != nil {
			rows.Close()
			http.Error(w, "scan tipos_iva error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		item.UpdatedAt = parseTime(updatedAtRaw)
		item.Activo = parseBool(activoRaw)
		tiposIVA = append(tiposIVA, item)
	}
	rows.Close()

	// query familias
	rows, err = c.db.QueryContext(r.Context(), familiaQuery, args...)
	if err != nil {
		http.Error(w, "query familias error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var familias []FamiliaResponse
	for rows.Next() {
		var item FamiliaResponse
		var updatedAtRaw interface{}
		var activoRaw interface{}
		if err := rows.Scan(&item.ID, &item.Nombre, &updatedAtRaw, &activoRaw); err != nil {
			rows.Close()
			http.Error(w, "scan familias error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		item.UpdatedAt = parseTime(updatedAtRaw)
		item.Activo = parseBool(activoRaw)
		familias = append(familias, item)
	}
	rows.Close()

	// query productos
	rows, err = c.db.QueryContext(r.Context(), productoQuery, args...)
	if err != nil {
		http.Error(w, "query productos error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var productos []ProductoResponse
	for rows.Next() {
		var item ProductoResponse
		var updatedAtRaw interface{}
		var activoRaw interface{}
		var famID sql.NullString
		if err := rows.Scan(&item.ID, &item.Codigo, &item.Nombre, &item.PrecioVenta, &famID, &item.TipoIvaID, &updatedAtRaw, &activoRaw); err != nil {
			rows.Close()
			http.Error(w, "scan productos error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if famID.Valid {
			val := famID.String
			item.FamiliaID = &val
		}
		item.UpdatedAt = parseTime(updatedAtRaw)
		item.Activo = parseBool(activoRaw)
		productos = append(productos, item)
	}
	rows.Close()

	// query clientes
	rows, err = c.db.QueryContext(r.Context(), clienteQuery, args...)
	if err != nil {
		http.Error(w, "query clientes error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var clientes []ClienteResponse
	for rows.Next() {
		var item ClienteResponse
		var updatedAtRaw interface{}
		var activoRaw interface{}
		var nifVal, emailVal sql.NullString
		if err := rows.Scan(&item.ID, &item.Nombre, &nifVal, &emailVal, &updatedAtRaw, &activoRaw); err != nil {
			rows.Close()
			http.Error(w, "scan clientes error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if nifVal.Valid {
			val := nifVal.String
			item.NIF = &val
		}
		if emailVal.Valid {
			val := emailVal.String
			item.Email = &val
		}
		item.UpdatedAt = parseTime(updatedAtRaw)
		item.Activo = parseBool(activoRaw)
		clientes = append(clientes, item)
	}
	rows.Close()

	// Deactivated items
	// tipos_iva
	rows, err = c.db.QueryContext(r.Context(), ivaElim, args...)
	if err != nil {
		http.Error(w, "query iva_eliminados error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var ivaElimIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			http.Error(w, "scan iva_eliminados error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		ivaElimIDs = append(ivaElimIDs, id)
	}
	rows.Close()

	// familias
	rows, err = c.db.QueryContext(r.Context(), familiaElim, args...)
	if err != nil {
		http.Error(w, "query familia_eliminados error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var familiaElimIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			http.Error(w, "scan familia_eliminados error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		familiaElimIDs = append(familiaElimIDs, id)
	}
	rows.Close()

	// productos
	rows, err = c.db.QueryContext(r.Context(), productoElim, args...)
	if err != nil {
		http.Error(w, "query producto_eliminados error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var productoElimIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			http.Error(w, "scan producto_eliminados error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		productoElimIDs = append(productoElimIDs, id)
	}
	rows.Close()

	// clientes
	rows, err = c.db.QueryContext(r.Context(), clienteElim, args...)
	if err != nil {
		http.Error(w, "query cliente_eliminados error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var clienteElimIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			http.Error(w, "scan cliente_eliminados error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		clienteElimIDs = append(clienteElimIDs, id)
	}
	rows.Close()

	resp := CatalogSyncResponse{
		TiposIVA:   tiposIVA,
		Familias:   familias,
		Productos:  productos,
		Clientes:   clientes,
		Eliminados: EliminadosResponse{
			Productos: productoElimIDs,
			Clientes:  clienteElimIDs,
			Familias:  familiaElimIDs,
			TiposIVA:  ivaElimIDs,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

type RecentSaleDossier struct {
	IDFactura string    `json:"id_factura"`
	ClienteID string    `json:"cliente_id"`
	Fecha     time.Time `json:"fecha"`
	Numero    string    `json:"numero"`
	Total     float64   `json:"total"`
	Estado    string    `json:"estado"`
}

type ClientStatsDossier struct {
	ClienteID                string  `json:"cliente_id"`
	SaldoPendiente           float64 `json:"saldo_pendiente"`
	LimiteCredito            float64 `json:"limite_credito"`
	ArticulosMasCompradosJSON string  `json:"articulos_mas_comprados_json"`
}

type PendingInvoiceDossier struct {
	IDFactura        string    `json:"id_factura"`
	ClienteID        string    `json:"cliente_id"`
	NumeroFactura    string    `json:"numero_factura"`
	ImportePendiente float64   `json:"importe_pendiente"`
	FechaEmision     time.Time `json:"fecha_emision"`
}

type ClientDossierResponse struct {
	VentasRecientes    []RecentSaleDossier     `json:"ventas_recientes"`
	Estadisticas       []ClientStatsDossier    `json:"estadisticas"`
	FacturasPendientes []PendingInvoiceDossier `json:"facturas_pendientes"`
}

func (c *CatalogSyncController) HandleClientDossier(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClientIDs []string `json:"client_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	var result ClientDossierResponse
	result.VentasRecientes = []RecentSaleDossier{}
	result.Estadisticas = []ClientStatsDossier{}
	result.FacturasPendientes = []PendingInvoiceDossier{}

	for _, clientID := range req.ClientIDs {
		// Query recent sales
		var sales []RecentSaleDossier
		var hasRecentSales bool
		if c.isSQLite {
			rows, err := c.db.QueryContext(r.Context(), "SELECT id_factura, cliente_id, fecha, numero, total, estado FROM cliente_ventas_recientes WHERE cliente_id = ?", clientID)
			if err == nil {
				for rows.Next() {
					var s RecentSaleDossier
					var fechaRaw interface{}
					if err := rows.Scan(&s.IDFactura, &s.ClienteID, &fechaRaw, &s.Numero, &s.Total, &s.Estado); err == nil {
						s.Fecha = parseTime(fechaRaw)
						sales = append(sales, s)
						hasRecentSales = true
					}
				}
				rows.Close()
			}
		}

		if !hasRecentSales {
			var query string
			if c.isSQLite {
				query = `SELECT i.id, i.invoice_number, i.total, COALESCE(i.status, ''), i.created_at
						 FROM invoice i
						 JOIN delivery_note dn ON i.delivery_note_id = dn.id
						 JOIN "order" o ON dn.order_id = o.id
						 JOIN quote q ON o.quote_id = q.id
						 WHERE q.client_id = ?
						 ORDER BY i.created_at DESC`
			} else {
				query = `SELECT i.id, i.invoice_number, i.total, COALESCE(i.status, ''), i.created_at
						 FROM invoice i
						 JOIN delivery_note dn ON i.delivery_note_id = dn.id
						 JOIN "order" o ON dn.order_id = o.id
						 JOIN quote q ON o.quote_id = q.id
						 WHERE q.client_id = $1
						 ORDER BY i.created_at DESC`
			}
			rows, err := c.db.QueryContext(r.Context(), query, clientID)
			if err == nil {
				for rows.Next() {
					var s RecentSaleDossier
					var fechaRaw interface{}
					if err := rows.Scan(&s.IDFactura, &s.Numero, &s.Total, &s.Estado, &fechaRaw); err == nil {
						s.ClienteID = clientID
						s.Fecha = parseTime(fechaRaw)
						sales = append(sales, s)
					}
				}
				rows.Close()
			}
		}
		result.VentasRecientes = append(result.VentasRecientes, sales...)

		// Query Stats
		var stats ClientStatsDossier
		var hasStats bool
		if c.isSQLite {
			err := c.db.QueryRowContext(r.Context(), "SELECT cliente_id, saldo_pendiente, limite_credito, articulos_mas_comprados_json FROM cliente_estadisticas WHERE cliente_id = ?", clientID).
				Scan(&stats.ClienteID, &stats.SaldoPendiente, &stats.LimiteCredito, &stats.ArticulosMasCompradosJSON)
			if err == nil {
				hasStats = true
			}
		}

		if !hasStats {
			var invoiceSumQuery string
			var paymentsSumQuery string
			if c.isSQLite {
				invoiceSumQuery = `SELECT COALESCE(SUM(i.total), 0)
								   FROM invoice i
								   JOIN delivery_note dn ON i.delivery_note_id = dn.id
								   JOIN "order" o ON dn.order_id = o.id
								   JOIN quote q ON o.quote_id = q.id
								   WHERE q.client_id = ?`
				paymentsSumQuery = `SELECT COALESCE(SUM(importe), 0) FROM cobros_recibidos WHERE cliente_id = ?`
			} else {
				invoiceSumQuery = `SELECT COALESCE(SUM(i.total), 0)
								   FROM invoice i
								   JOIN delivery_note dn ON i.delivery_note_id = dn.id
								   JOIN "order" o ON dn.order_id = o.id
								   JOIN quote q ON o.quote_id = q.id
								   WHERE q.client_id = $1`
				paymentsSumQuery = `SELECT COALESCE(SUM(importe), 0) FROM cobros_recibidos WHERE cliente_id = $1`
			}

			var totalInvoices float64
			var totalPayments float64
			_ = c.db.QueryRowContext(r.Context(), invoiceSumQuery, clientID).Scan(&totalInvoices)
			_ = c.db.QueryRowContext(r.Context(), paymentsSumQuery, clientID).Scan(&totalPayments)

			stats.ClienteID = clientID
			stats.SaldoPendiente = totalInvoices - totalPayments
			stats.LimiteCredito = 5000.0
			stats.ArticulosMasCompradosJSON = "[]"
		}
		result.Estadisticas = append(result.Estadisticas, stats)

		// Query pending invoices
		var pendings []PendingInvoiceDossier
		var hasPendings bool
		if c.isSQLite {
			rows, err := c.db.QueryContext(r.Context(), "SELECT id_factura, cliente_id, numero_factura, importe_pendiente, fecha_emision FROM cliente_facturas_pendientes WHERE cliente_id = ?", clientID)
			if err == nil {
				for rows.Next() {
					var p PendingInvoiceDossier
					var fechaRaw interface{}
					if err := rows.Scan(&p.IDFactura, &p.ClienteID, &p.NumeroFactura, &p.ImportePendiente, &fechaRaw); err == nil {
						p.FechaEmision = parseTime(fechaRaw)
						pendings = append(pendings, p)
						hasPendings = true
					}
				}
				rows.Close()
			}
		}

		if !hasPendings {
			var query string
			if c.isSQLite {
				query = `SELECT i.id, i.invoice_number, i.total, i.created_at
						 FROM invoice i
						 JOIN delivery_note dn ON i.delivery_note_id = dn.id
						 JOIN "order" o ON dn.order_id = o.id
						 JOIN quote q ON o.quote_id = q.id
						 WHERE q.client_id = ? AND (i.status IS NULL OR i.status != 'Paid')`
			} else {
				query = `SELECT i.id, i.invoice_number, i.total, i.created_at
						 FROM invoice i
						 JOIN delivery_note dn ON i.delivery_note_id = dn.id
						 JOIN "order" o ON dn.order_id = o.id
						 JOIN quote q ON o.quote_id = q.id
						 WHERE q.client_id = $1 AND (i.status IS NULL OR i.status != 'Paid')`
			}
			rows, err := c.db.QueryContext(r.Context(), query, clientID)
			if err == nil {
				for rows.Next() {
					var p PendingInvoiceDossier
					var fechaRaw interface{}
					if err := rows.Scan(&p.IDFactura, &p.NumeroFactura, &p.ImportePendiente, &fechaRaw); err == nil {
						p.ClienteID = clientID
						p.FechaEmision = parseTime(fechaRaw)
						pendings = append(pendings, p)
					}
				}
				rows.Close()
			}
		}
		result.FacturasPendientes = append(result.FacturasPendientes, pendings...)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

type SyncPayment struct {
	ID             string  `json:"id"`
	ClienteID      string  `json:"cliente_id"`
	FacturaID      *string `json:"factura_id"`
	Importe        float64 `json:"importe"`
	Fecha          string  `json:"fecha"`
	MetodoPago     string  `json:"metodo_pago"`
	TipoCobro      string  `json:"tipo_cobro"`
	IdempotencyKey string  `json:"idempotency_key"`
}

type SyncPaymentsRequest struct {
	Payments []SyncPayment `json:"payments"`
}

type SyncPaymentsResponse struct {
	Status       string   `json:"status"`
	SyncedCount  int      `json:"synced_count"`
	ProcessedIDs []string `json:"processed_ids"`
}

func (c *CatalogSyncController) HandleSyncPayments(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		http.Error(w, "missing Idempotency-Key header", http.StatusBadRequest)
		return
	}

	if !c.idempTracker.IsValidKey(idemKey) {
		http.Error(w, "invalid Idempotency-Key format", http.StatusBadRequest)
		return
	}

	found, savedBody, err := c.idempTracker.GetResponse(r.Context(), idemKey)
	if err != nil {
		http.Error(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if found && savedBody != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(savedBody))
		return
	}

	if !found {
		err = c.idempTracker.ReserveKey(r.Context(), idemKey)
		if err != nil {
			http.Error(w, "duplicate key or reservation failed: "+err.Error(), http.StatusConflict)
			return
		}
	}

	tx, err := c.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var req SyncPaymentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	var processedIDs []string
	var insertPaymentQuery string
	if c.isSQLite {
		insertPaymentQuery = `INSERT INTO cobros_recibidos (id, cliente_id, factura_id, importe, fecha, metodo_pago, tipo_cobro, idempotency_key, synced_at)
							  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	} else {
		insertPaymentQuery = `INSERT INTO cobros_recibidos (id, cliente_id, factura_id, importe, fecha, metodo_pago, tipo_cobro, idempotency_key, synced_at)
							  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	}

	for _, pay := range req.Payments {
		parsedFecha, err := time.Parse(time.RFC3339, pay.Fecha)
		if err != nil {
			parsedFecha, err = time.Parse("2006-01-02 15:04:05", pay.Fecha)
			if err != nil {
				parsedFecha = time.Now()
			}
		}

		var facturaIDVal sql.NullString
		if pay.FacturaID != nil {
			facturaIDVal.String = *pay.FacturaID
			facturaIDVal.Valid = true
		}

		_, err = tx.ExecContext(r.Context(), insertPaymentQuery,
			pay.ID,
			pay.ClienteID,
			facturaIDVal,
			pay.Importe,
			parsedFecha.UTC(),
			pay.MetodoPago,
			pay.TipoCobro,
			pay.IdempotencyKey,
			time.Now().UTC(),
		)
		if err != nil {
			http.Error(w, "failed to record payment: "+err.Error(), http.StatusInternalServerError)
			return
		}

		processedIDs = append(processedIDs, pay.ID)
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := SyncPaymentsResponse{
		Status:       "success",
		SyncedCount:  len(processedIDs),
		ProcessedIDs: processedIDs,
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "failed to marshal response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_ = c.idempTracker.SaveResponse(r.Context(), idemKey, string(respBytes))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}
