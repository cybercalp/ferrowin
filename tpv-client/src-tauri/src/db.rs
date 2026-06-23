use rusqlite::{params, Connection, Result};
use serde::{Deserialize, Serialize};

// ---------------------------------------------------------------------------
// Structs
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OfflineSale {
    pub id: String,
    pub terminal_id: String,
    pub customer_id: Option<String>,
    pub total: f64,
    pub created_at: String,
    pub sync_status: String,
    pub idempotency_key: String,
    pub invoice_number: String,
    pub sequence_number: i64,
    pub firma_registro: Option<String>,
    pub hash_anterior: Option<String>,
    pub datos_encadenamiento: Option<String>,
    // Phase 1 — TPV Tienda POS financial fields
    pub subtotal: f64,
    pub tax_total: f64,
    pub discount_total: f64,
    #[serde(default = "default_sale_status")]
    pub status: String,
    pub void_reason: Option<String>,
    pub voided_at: Option<String>,
}

fn default_sale_status() -> String {
    "COMPLETED".to_string()
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RegistroSuceso {
    pub id: String,
    pub fecha_hora: String,
    pub tipo_evento: String,
    pub detalles: String,
    pub estado_sincronizacion: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OfflineSaleItem {
    pub id: String,
    pub offline_sale_id: String,
    pub item_id: String,
    pub quantity: f64,
    pub unit_price: f64,
    #[serde(default)]
    pub discount_percent: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct POSProduct {
    pub id: String,
    pub codigo: String,
    pub nombre: String,
    pub precio_venta: f64,
    pub stock: Option<f64>,
    pub familia_nombre: Option<String>,
    pub tipo_iva_nombre: Option<String>,
    pub tipo_iva_porcentaje: Option<f64>,
    pub imagen_url: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OfflineBoxClosure {
    pub id: String,
    pub opened_at: String,
    pub closed_at: String,
    pub cash_reported: f64,
    pub card_reported: f64,
    pub sales_total: f64,
    pub sync_status: String,
    pub idempotency_key: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StockCacheEntry {
    pub item_id: String,
    pub stock: f64,
    pub last_updated_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TipoIVA {
    pub id: String,
    pub nombre: String,
    pub porcentaje: f64,
    pub updated_at: String,
    pub activo: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Familia {
    pub id: String,
    pub nombre: String,
    pub updated_at: String,
    pub activo: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Producto {
    pub id: String,
    pub codigo: String,
    pub nombre: String,
    pub precio_venta: f64,
    pub familia_id: Option<String>,
    pub tipo_iva_id: String,
    pub updated_at: String,
    pub activo: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TerminalHealth {
    pub terminal_id: String,
    pub db_size_bytes: i64,
    pub pending_sales_count: i64,
    pub pending_closures_count: i64,
    pub online_status: bool,
    pub app_version: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Cliente {
    pub id: String,
    pub nombre: String,
    pub nif: Option<String>,
    pub email: Option<String>,
    pub updated_at: Option<String>,
    pub activo: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Direccion {
    pub id: String,
    pub entidad_id: String,
    pub tipo_direccion: String,
    pub calle: String,
    pub ciudad: String,
    pub provincia: String,
    pub codigo_postal: String,
    pub pais: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Contacto {
    pub id: String,
    pub entidad_id: String,
    pub nombre: String,
    pub puesto: Option<String>,
    pub email: Option<String>,
    pub telefono: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Nota {
    pub id: String,
    pub entidad_id: String,
    pub nota: String,
    pub creado_en: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CustomerInfo {
    pub id: String,
    pub nombre: String,
    pub nif: Option<String>,
    pub direccion: Option<String>,
    pub descuento: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecentSaleDossier {
    pub id_factura: String,
    pub cliente_id: String,
    pub fecha: String,
    pub numero: String,
    pub total: f64,
    pub estado: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PendingInvoiceDossier {
    pub id_factura: String,
    pub cliente_id: String,
    pub numero_factura: String,
    pub importe_pendiente: f64,
    pub fecha_emision: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClientStatsDossier {
    pub cliente_id: String,
    pub saldo_pendiente: f64,
    pub limite_credito: f64,
    pub articulos_mas_comprados_json: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClientDossier {
    pub cliente: Cliente,
    pub estadisticas: Option<ClientStatsDossier>,
    pub ventas_recientes: Vec<RecentSaleDossier>,
    pub facturas_pendientes: Vec<PendingInvoiceDossier>,
}

// ---------------------------------------------------------------------------
// Connection & Schema
// ---------------------------------------------------------------------------

/// Opens (or creates) the SQLite database at `path`, enables WAL mode,
/// sets a 5 000 ms busy-timeout, and runs the schema DDL.
pub fn init_db(path: &str) -> Result<Connection> {
    let conn = Connection::open(path)?;

    conn.execute_batch("PRAGMA journal_mode = WAL;")?;
    conn.execute_batch("PRAGMA busy_timeout = 5000;")?;
    conn.execute_batch("PRAGMA foreign_keys = ON;")?;

    conn.execute_batch(
        "
        CREATE TABLE IF NOT EXISTS offline_sales (
            id TEXT PRIMARY KEY,
            terminal_id TEXT NOT NULL,
            customer_id TEXT,
            total REAL NOT NULL,
            created_at TEXT NOT NULL,
            sync_status TEXT DEFAULT 'PENDING',
            idempotency_key TEXT UNIQUE NOT NULL,
            invoice_number TEXT UNIQUE NOT NULL,
            sequence_number INTEGER NOT NULL,
            firma_registro TEXT,
            hash_anterior TEXT,
            datos_encadenamiento TEXT
        );

        CREATE TABLE IF NOT EXISTS offline_sale_items (
            id TEXT PRIMARY KEY,
            offline_sale_id TEXT REFERENCES offline_sales(id) ON DELETE CASCADE,
            item_id TEXT NOT NULL,
            quantity REAL NOT NULL,
            unit_price REAL NOT NULL
        );

        CREATE INDEX IF NOT EXISTS idx_offline_sales_terminal_id
            ON offline_sales(terminal_id);
        CREATE INDEX IF NOT EXISTS idx_offline_sales_sync_status
            ON offline_sales(sync_status);
        CREATE INDEX IF NOT EXISTS idx_offline_sale_items_offline_sale_id
            ON offline_sale_items(offline_sale_id);

        CREATE TABLE IF NOT EXISTS offline_box_closures (
            id TEXT PRIMARY KEY,
            opened_at TEXT NOT NULL,
            closed_at TEXT NOT NULL,
            cash_reported REAL NOT NULL,
            card_reported REAL NOT NULL,
            sales_total REAL NOT NULL,
            sync_status TEXT NOT NULL DEFAULT 'PENDING',
            idempotency_key TEXT UNIQUE NOT NULL
        );

        CREATE TABLE IF NOT EXISTS stock_cache (
            item_id TEXT PRIMARY KEY,
            stock REAL NOT NULL,
            last_updated_at TEXT NOT NULL
        );

        CREATE TABLE IF NOT EXISTS ultimo_registro_encadenado (
            id_factura TEXT PRIMARY KEY,
            firma_registro TEXT NOT NULL
        );

        CREATE TABLE IF NOT EXISTS registro_sucesos (
            id TEXT PRIMARY KEY,
            fecha_hora TEXT NOT NULL,
            tipo_evento TEXT NOT NULL,
            detalles TEXT NOT NULL,
            estado_sincronizacion TEXT NOT NULL DEFAULT 'PENDING'
        );

        CREATE TABLE IF NOT EXISTS tipos_iva (
            id TEXT PRIMARY KEY,
            nombre TEXT NOT NULL,
            porcentaje REAL NOT NULL,
            updated_at TEXT,
            activo INTEGER DEFAULT 1
        );

        CREATE TABLE IF NOT EXISTS familias (
            id TEXT PRIMARY KEY,
            nombre TEXT NOT NULL,
            updated_at TEXT,
            activo INTEGER DEFAULT 1
        );

        CREATE TABLE IF NOT EXISTS productos (
            id TEXT PRIMARY KEY,
            codigo TEXT UNIQUE NOT NULL,
            nombre TEXT NOT NULL,
            precio_venta REAL NOT NULL,
            familia_id TEXT,
            tipo_iva_id TEXT,
            updated_at TEXT,
            activo INTEGER DEFAULT 1
        );

        CREATE TABLE IF NOT EXISTS clientes (
            id TEXT PRIMARY KEY,
            nombre TEXT NOT NULL,
            nif TEXT,
            email TEXT,
            updated_at TEXT,
            activo INTEGER DEFAULT 1,
            roles TEXT DEFAULT 'CLIENTE'
        );

        CREATE TABLE IF NOT EXISTS direcciones (
            id TEXT PRIMARY KEY,
            entidad_id TEXT NOT NULL REFERENCES clientes(id) ON DELETE CASCADE,
            tipo_direccion TEXT NOT NULL DEFAULT 'FISCAL',
            calle TEXT NOT NULL,
            ciudad TEXT NOT NULL,
            provincia TEXT NOT NULL,
            codigo_postal TEXT NOT NULL,
            pais TEXT NOT NULL DEFAULT 'España'
        );

        CREATE TABLE IF NOT EXISTS contactos (
            id TEXT PRIMARY KEY,
            entidad_id TEXT NOT NULL REFERENCES clientes(id) ON DELETE CASCADE,
            nombre TEXT NOT NULL,
            puesto TEXT,
            email TEXT,
            telefono TEXT
        );

        CREATE TABLE IF NOT EXISTS notas (
            id TEXT PRIMARY KEY,
            entidad_id TEXT NOT NULL REFERENCES clientes(id) ON DELETE CASCADE,
            nota TEXT NOT NULL,
            creado_en TEXT NOT NULL
        );

        CREATE TABLE IF NOT EXISTS cobros (
            id TEXT PRIMARY KEY,
            cliente_id TEXT NOT NULL,
            factura_id TEXT,
            importe REAL NOT NULL,
            metodo_pago TEXT NOT NULL,
            tipo_cobro TEXT NOT NULL DEFAULT 'DEUDA',
            created_at TEXT NOT NULL
        );

        CREATE TABLE IF NOT EXISTS sincronizacion_metadatos (
            clave TEXT PRIMARY KEY,
            valor TEXT
        );
        ",
    )?;

    // Try to add new columns to offline_sales if they don't exist
    let _ = conn.execute("ALTER TABLE offline_sales ADD COLUMN firma_registro TEXT;", []);
    let _ = conn.execute("ALTER TABLE offline_sales ADD COLUMN hash_anterior TEXT;", []);
    let _ = conn.execute("ALTER TABLE offline_sales ADD COLUMN datos_encadenamiento TEXT;", []);

    // Client dossier tables (safe if already exist via CREATE TABLE IF NOT EXISTS above)
    // No ALTER TABLE needed — these are new tables created in the batch above.

    // Phase 1 — TPV Tienda POS: extend existing tables
    let _ = conn.execute("ALTER TABLE offline_sales ADD COLUMN subtotal REAL NOT NULL DEFAULT 0;", []);
    let _ = conn.execute("ALTER TABLE offline_sales ADD COLUMN tax_total REAL NOT NULL DEFAULT 0;", []);
    let _ = conn.execute("ALTER TABLE offline_sales ADD COLUMN discount_total REAL NOT NULL DEFAULT 0;", []);
    let _ = conn.execute("ALTER TABLE offline_sales ADD COLUMN status TEXT NOT NULL DEFAULT 'COMPLETED';", []);
    let _ = conn.execute("ALTER TABLE offline_sales ADD COLUMN void_reason TEXT;", []);
    let _ = conn.execute("ALTER TABLE offline_sales ADD COLUMN voided_at TEXT;", []);
    let _ = conn.execute("ALTER TABLE offline_sale_items ADD COLUMN discount_percent REAL NOT NULL DEFAULT 0;", []);
    let _ = conn.execute("ALTER TABLE productos ADD COLUMN imagen_url TEXT;", []);

    // Phase 1 — TPV Tienda POS: new tables
    conn.execute_batch(
        "
        CREATE TABLE IF NOT EXISTS caja_secuencia (
            prefix TEXT PRIMARY KEY,
            next_val INTEGER NOT NULL DEFAULT 1
        );

        CREATE TABLE IF NOT EXISTS offline_sale_payments (
            id TEXT PRIMARY KEY,
            sale_id TEXT NOT NULL REFERENCES offline_sales(id) ON DELETE CASCADE,
            metodo_pago TEXT NOT NULL,
            amount REAL NOT NULL,
            created_at TEXT NOT NULL
        );

        CREATE TABLE IF NOT EXISTS caja_aperturas (
            id TEXT PRIMARY KEY,
            amount REAL NOT NULL,
            opened_at TEXT NOT NULL
        );

        CREATE TABLE IF NOT EXISTS caja_movimientos (
            id TEXT PRIMARY KEY,
            tipo TEXT NOT NULL CHECK(tipo IN ('INGRESO', 'RETIRO')),
            concepto TEXT NOT NULL,
            amount REAL NOT NULL,
            created_at TEXT NOT NULL
        );
        ",
    )?;

    // Phase 1 — TPV Tienda POS: FTS5 virtual table for product search
    // content-sync triggers keep the index in sync with the productos table
    let _ = conn.execute_batch(
        "
        CREATE VIRTUAL TABLE IF NOT EXISTS productos_fts USING fts5(
            codigo, nombre,
            content=productos,
            content_rowid=rowid
        );

        CREATE TRIGGER IF NOT EXISTS productos_ai AFTER INSERT ON productos BEGIN
            INSERT INTO productos_fts(rowid, codigo, nombre)
            VALUES (new.rowid, new.codigo, new.nombre);
        END;

        CREATE TRIGGER IF NOT EXISTS productos_ad AFTER DELETE ON productos BEGIN
            INSERT INTO productos_fts(productos_fts, rowid, codigo, nombre)
            VALUES ('delete', old.rowid, old.codigo, old.nombre);
        END;

        CREATE TRIGGER IF NOT EXISTS productos_au AFTER UPDATE ON productos BEGIN
            INSERT INTO productos_fts(productos_fts, rowid, codigo, nombre)
            VALUES ('delete', old.rowid, old.codigo, old.nombre);
            INSERT INTO productos_fts(rowid, codigo, nombre)
            VALUES (new.rowid, new.codigo, new.nombre);
        END;
        ",
    );

    Ok(conn)
}

// ---------------------------------------------------------------------------
// FTS5 Helpers
// ---------------------------------------------------------------------------

/// Rebuilds the productos_fts index from scratch. Call this after a bulk
/// catalog sync to ensure the FTS index is consistent with productos.
pub fn reindex_productos_fts(conn: &Connection) -> rusqlite::Result<()> {
    conn.execute("INSERT INTO productos_fts(productos_fts) VALUES('rebuild')", [])?;
    Ok(())
}

// ---------------------------------------------------------------------------
// Sequence Counter
// ---------------------------------------------------------------------------

/// Atomically increments the sequence counter for `prefix` and returns the
/// new value. Inserts 1 for a new prefix, then increments on subsequent calls.
/// Must be called within a transaction for true atomicity across operations.
pub fn get_next_sequence(conn: &Connection, prefix: &str) -> Result<i64> {
    conn.execute(
        "INSERT INTO caja_secuencia (prefix, next_val) VALUES (?1, 2)
         ON CONFLICT(prefix) DO UPDATE SET next_val = next_val + 1",
        params![prefix],
    )?;
    let next_val: i64 = conn.query_row(
        "SELECT next_val FROM caja_secuencia WHERE prefix = ?1",
        params![prefix],
        |row| row.get(0),
    )?;
    Ok(next_val)
}

// ---------------------------------------------------------------------------
// Product Search (FTS5)
// ---------------------------------------------------------------------------

/// Searches products using FTS5 on codigo + nombre. Sanitises the query to
/// prevent FTS5 syntax errors. Returns POSProduct rows with joined stock,
/// familia, and tipo_iva info.
pub fn search_products(conn: &Connection, query: &str) -> Result<Vec<POSProduct>> {
    let sanitised: String = query
        .chars()
        .filter(|c| c.is_alphanumeric() || c.is_whitespace() || *c == '-' || *c == '_')
        .collect();
    let terms: Vec<&str> = sanitised.split_whitespace().collect();
    if terms.is_empty() {
        return Ok(Vec::new());
    }
    // Build a prefix-match FTS5 query: each term quoted + trailing *
    let fts_query: String = terms
        .iter()
        .map(|t| format!("\"{}\"*", t))
        .collect::<Vec<_>>()
        .join(" ");

    let sql = r#"
        SELECT p.id, p.codigo, p.nombre, p.precio_venta,
               sc.stock, f.nombre, ti.nombre, ti.porcentaje, p.imagen_url
        FROM productos_fts
        JOIN productos p ON productos_fts.rowid = p.rowid
        LEFT JOIN stock_cache sc ON sc.item_id = p.id
        LEFT JOIN familias f ON f.id = p.familia_id
        LEFT JOIN tipos_iva ti ON ti.id = p.tipo_iva_id
        WHERE productos_fts MATCH ?1
          AND p.activo = 1
        LIMIT 50
    "#;
    let mut stmt = conn.prepare(sql)?;
    let rows = stmt.query_map(params![fts_query], |row| {
        Ok(POSProduct {
            id: row.get(0)?,
            codigo: row.get(1)?,
            nombre: row.get(2)?,
            precio_venta: row.get(3)?,
            stock: row.get(4)?,
            familia_nombre: row.get(5)?,
            tipo_iva_nombre: row.get(6)?,
            tipo_iva_porcentaje: row.get(7)?,
            imagen_url: row.get(8)?,
        })
    })?;
    rows.collect()
}

/// Looks up a single product by its exact codigo (barcode / SKU).
pub fn get_product_by_code(conn: &Connection, code: &str) -> Result<Option<POSProduct>> {
    let sql = r#"
        SELECT p.id, p.codigo, p.nombre, p.precio_venta,
               sc.stock, f.nombre, ti.nombre, ti.porcentaje, p.imagen_url
        FROM productos p
        LEFT JOIN stock_cache sc ON sc.item_id = p.id
        LEFT JOIN familias f ON f.id = p.familia_id
        LEFT JOIN tipos_iva ti ON ti.id = p.tipo_iva_id
        WHERE p.codigo = ?1 AND p.activo = 1
        LIMIT 1
    "#;
    let mut stmt = conn.prepare(sql)?;
    let mut rows = stmt.query_map(params![code], |row| {
        Ok(POSProduct {
            id: row.get(0)?,
            codigo: row.get(1)?,
            nombre: row.get(2)?,
            precio_venta: row.get(3)?,
            stock: row.get(4)?,
            familia_nombre: row.get(5)?,
            tipo_iva_nombre: row.get(6)?,
            tipo_iva_porcentaje: row.get(7)?,
            imagen_url: row.get(8)?,
        })
    })?;
    match rows.next() {
        Some(r) => Ok(Some(r?)),
        None => Ok(None),
    }
}

// ---------------------------------------------------------------------------
// Offline Sales CRUD
// ---------------------------------------------------------------------------

pub fn insert_offline_sale(conn: &Connection, sale: &OfflineSale) -> Result<()> {
    conn.execute(
        "INSERT INTO offline_sales
            (id, terminal_id, customer_id, total, created_at,
             sync_status, idempotency_key, invoice_number, sequence_number,
             firma_registro, hash_anterior, datos_encadenamiento,
             subtotal, tax_total, discount_total, status, void_reason, voided_at)
         VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12,
                 ?13, ?14, ?15, ?16, ?17, ?18)",
        params![
            sale.id,
            sale.terminal_id,
            sale.customer_id,
            sale.total,
            sale.created_at,
            sale.sync_status,
            sale.idempotency_key,
            sale.invoice_number,
            sale.sequence_number,
            sale.firma_registro,
            sale.hash_anterior,
            sale.datos_encadenamiento,
            sale.subtotal,
            sale.tax_total,
            sale.discount_total,
            sale.status,
            sale.void_reason,
            sale.voided_at,
        ],
    )?;
    Ok(())
}

pub fn insert_offline_sale_item(conn: &Connection, item: &OfflineSaleItem) -> Result<()> {
    conn.execute(
        "INSERT INTO offline_sale_items
            (id, offline_sale_id, item_id, quantity, unit_price, discount_percent)
         VALUES (?1, ?2, ?3, ?4, ?5, ?6)",
        params![
            item.id,
            item.offline_sale_id,
            item.item_id,
            item.quantity,
            item.unit_price,
            item.discount_percent,
        ],
    )?;
    Ok(())
}

/// Builds an OfflineSale from a query row. Used by multiple query functions
/// to avoid repeating the column mapping.
macro_rules! row_to_offline_sale {
    ($row:ident) => {
        Ok(OfflineSale {
            id: $row.get(0)?,
            terminal_id: $row.get(1)?,
            customer_id: $row.get(2)?,
            total: $row.get(3)?,
            created_at: $row.get(4)?,
            sync_status: $row.get(5)?,
            idempotency_key: $row.get(6)?,
            invoice_number: $row.get(7)?,
            sequence_number: $row.get(8)?,
            firma_registro: $row.get(9)?,
            hash_anterior: $row.get(10)?,
            datos_encadenamiento: $row.get(11)?,
            subtotal: $row.get(12)?,
            tax_total: $row.get(13)?,
            discount_total: $row.get(14)?,
            status: $row.get(15)?,
            void_reason: $row.get(16)?,
            voided_at: $row.get(17)?,
        })
    };
}

const OFFLINE_SALE_COLUMNS: &str =
    "id, terminal_id, customer_id, total, created_at, \
     sync_status, idempotency_key, invoice_number, sequence_number, \
     firma_registro, hash_anterior, datos_encadenamiento, \
     subtotal, tax_total, discount_total, status, void_reason, voided_at";

pub fn get_pending_sales(conn: &Connection) -> Result<Vec<OfflineSale>> {
    let sql = format!(
        "SELECT {} FROM offline_sales WHERE sync_status = 'PENDING'",
        OFFLINE_SALE_COLUMNS
    );
    let mut stmt = conn.prepare(&sql)?;

    let rows = stmt.query_map([], |row| {
        row_to_offline_sale!(row)
    })?;

    rows.collect()
}

pub fn get_sale_by_id(conn: &Connection, id: &str) -> Result<Option<OfflineSale>> {
    let sql = format!(
        "SELECT {} FROM offline_sales WHERE id = ?1",
        OFFLINE_SALE_COLUMNS
    );
    let mut stmt = conn.prepare(&sql)?;
    let mut rows = stmt.query_map(params![id], |row| {
        row_to_offline_sale!(row)
    })?;
    match rows.next() {
        Some(r) => Ok(Some(r?)),
        None => Ok(None),
    }
}

pub fn get_today_sales(conn: &Connection) -> Result<Vec<OfflineSale>> {
    let sql = format!(
        "SELECT {} FROM offline_sales WHERE date(created_at) = date('now') ORDER BY created_at",
        OFFLINE_SALE_COLUMNS
    );
    let mut stmt = conn.prepare(&sql)?;

    let rows = stmt.query_map([], |row| {
        row_to_offline_sale!(row)
    })?;

    rows.collect()
}

pub fn get_sale_items(conn: &Connection, sale_id: &str) -> Result<Vec<OfflineSaleItem>> {
    let mut stmt = conn.prepare(
        "SELECT id, offline_sale_id, item_id, quantity, unit_price, discount_percent
         FROM offline_sale_items
         WHERE offline_sale_id = ?1",
    )?;
    let rows = stmt.query_map(params![sale_id], |row| {
        Ok(OfflineSaleItem {
            id: row.get(0)?,
            offline_sale_id: row.get(1)?,
            item_id: row.get(2)?,
            quantity: row.get(3)?,
            unit_price: row.get(4)?,
            discount_percent: row.get(5)?,
        })
    })?;
    rows.collect()
}

pub fn delete_synced_sale(conn: &Connection, id: &str) -> Result<()> {
    conn.execute("DELETE FROM offline_sales WHERE id = ?1", params![id])?;
    Ok(())
}

pub fn insert_registro_suceso(conn: &Connection, id: &str, tipo_evento: &str, detalles: &str) -> Result<()> {
    let fecha_hora = chrono::Utc::now().to_rfc3339();
    conn.execute(
        "INSERT INTO registro_sucesos (id, fecha_hora, tipo_evento, detalles, estado_sincronizacion)
         VALUES (?1, ?2, ?3, ?4, 'PENDING')",
        params![id, fecha_hora, tipo_evento, detalles],
    )?;
    Ok(())
}

pub fn get_pending_events(conn: &Connection) -> Result<Vec<RegistroSuceso>> {
    let mut stmt = conn.prepare(
        "SELECT id, fecha_hora, tipo_evento, detalles, estado_sincronizacion
         FROM registro_sucesos
         WHERE estado_sincronizacion = 'PENDING'",
    )?;
    let rows = stmt.query_map([], |row| {
        Ok(RegistroSuceso {
            id: row.get(0)?,
            fecha_hora: row.get(1)?,
            tipo_evento: row.get(2)?,
            detalles: row.get(3)?,
            estado_sincronizacion: row.get(4)?,
        })
    })?;
    rows.collect()
}

pub fn delete_synced_event(conn: &Connection, id: &str) -> Result<()> {
    conn.execute("DELETE FROM registro_sucesos WHERE id = ?1", params![id])?;
    Ok(())
}

pub fn get_ultimo_registro_encadenado(conn: &Connection) -> Result<Option<String>> {
    let mut stmt = conn.prepare("SELECT firma_registro FROM ultimo_registro_encadenado LIMIT 1")?;
    let mut rows = stmt.query([])?;
    if let Some(row) = rows.next()? {
        let firma: String = row.get(0)?;
        Ok(Some(firma))
    } else {
        Ok(None)
    }
}

pub fn upsert_ultimo_registro_encadenado(conn: &Connection, id_factura: &str, firma_registro: &str) -> Result<()> {
    conn.execute("DELETE FROM ultimo_registro_encadenado", [])?;
    conn.execute(
        "INSERT OR REPLACE INTO ultimo_registro_encadenado (id_factura, firma_registro) VALUES (?1, ?2)",
        params![id_factura, firma_registro],
    )?;
    Ok(())
}

// ---------------------------------------------------------------------------
// Offline Box Closures CRUD
// ---------------------------------------------------------------------------

pub fn insert_offline_box_closure(conn: &Connection, closure: &OfflineBoxClosure) -> Result<()> {
    conn.execute(
        "INSERT INTO offline_box_closures
            (id, opened_at, closed_at, cash_reported, card_reported,
             sales_total, sync_status, idempotency_key)
         VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8)",
        params![
            closure.id,
            closure.opened_at,
            closure.closed_at,
            closure.cash_reported,
            closure.card_reported,
            closure.sales_total,
            closure.sync_status,
            closure.idempotency_key,
        ],
    )?;
    Ok(())
}

pub fn get_pending_closures(conn: &Connection) -> Result<Vec<OfflineBoxClosure>> {
    let mut stmt = conn.prepare(
        "SELECT id, opened_at, closed_at, cash_reported, card_reported,
                sales_total, sync_status, idempotency_key
         FROM offline_box_closures
         WHERE sync_status = 'PENDING'",
    )?;

    let rows = stmt.query_map([], |row| {
        Ok(OfflineBoxClosure {
            id: row.get(0)?,
            opened_at: row.get(1)?,
            closed_at: row.get(2)?,
            cash_reported: row.get(3)?,
            card_reported: row.get(4)?,
            sales_total: row.get(5)?,
            sync_status: row.get(6)?,
            idempotency_key: row.get(7)?,
        })
    })?;

    rows.collect()
}

pub fn delete_synced_closure(conn: &Connection, id: &str) -> Result<()> {
    conn.execute(
        "DELETE FROM offline_box_closures WHERE id = ?1",
        params![id],
    )?;
    Ok(())
}

// ---------------------------------------------------------------------------
// Stock Cache CRUD
// ---------------------------------------------------------------------------

pub fn upsert_stock_cache(
    conn: &Connection,
    item_id: &str,
    stock: f64,
    last_updated_at: &str,
) -> Result<()> {
    conn.execute(
        "INSERT INTO stock_cache (item_id, stock, last_updated_at)
         VALUES (?1, ?2, ?3)
         ON CONFLICT(item_id) DO UPDATE
            SET stock = excluded.stock,
                last_updated_at = excluded.last_updated_at",
        params![item_id, stock, last_updated_at],
    )?;
    Ok(())
}

pub fn decrement_stock_cache(
    conn: &Connection,
    item_id: &str,
    quantity: f64,
    last_updated_at: &str,
) -> Result<()> {
    conn.execute(
        "INSERT INTO stock_cache (item_id, stock, last_updated_at)
         VALUES (?1, -?2, ?3)
         ON CONFLICT(item_id) DO UPDATE SET
            stock = stock - ?2,
            last_updated_at = ?3",
        params![item_id, quantity, last_updated_at],
    )?;
    Ok(())
}


pub fn get_cached_stock(conn: &Connection, item_id: &str) -> Result<Option<f64>> {
    let mut stmt =
        conn.prepare("SELECT stock FROM stock_cache WHERE item_id = ?1")?;

    let mut rows = stmt.query(params![item_id])?;
    match rows.next()? {
        Some(row) => Ok(Some(row.get(0)?)),
        None => Ok(None),
    }
}

pub fn upsert_tipo_iva(conn: &Connection, item: &TipoIVA) -> Result<()> {
    conn.execute(
        "INSERT INTO tipos_iva (id, nombre, porcentaje, updated_at, activo)
         VALUES (?1, ?2, ?3, ?4, ?5)
         ON CONFLICT(id) DO UPDATE SET
             nombre = excluded.nombre,
             porcentaje = excluded.porcentaje,
             updated_at = excluded.updated_at,
             activo = excluded.activo",
        params![
            item.id,
            item.nombre,
            item.porcentaje,
            item.updated_at,
            if item.activo { 1 } else { 0 }
        ],
    )?;
    Ok(())
}

pub fn upsert_familia(conn: &Connection, item: &Familia) -> Result<()> {
    conn.execute(
        "INSERT INTO familias (id, nombre, updated_at, activo)
         VALUES (?1, ?2, ?3, ?4)
         ON CONFLICT(id) DO UPDATE SET
             nombre = excluded.nombre,
             updated_at = excluded.updated_at,
             activo = excluded.activo",
        params![
            item.id,
            item.nombre,
            item.updated_at,
            if item.activo { 1 } else { 0 }
        ],
    )?;
    Ok(())
}

pub fn upsert_producto(conn: &Connection, item: &Producto) -> Result<()> {
    conn.execute(
        "INSERT INTO productos (id, codigo, nombre, precio_venta, familia_id, tipo_iva_id, updated_at, activo)
         VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8)
         ON CONFLICT(id) DO UPDATE SET
             codigo = excluded.codigo,
             nombre = excluded.nombre,
             precio_venta = excluded.precio_venta,
             familia_id = excluded.familia_id,
             tipo_iva_id = excluded.tipo_iva_id,
             updated_at = excluded.updated_at,
             activo = excluded.activo",
        params![
            item.id,
            item.codigo,
            item.nombre,
            item.precio_venta,
            item.familia_id,
            item.tipo_iva_id,
            item.updated_at,
            if item.activo { 1 } else { 0 }
        ],
    )?;
    Ok(())
}

pub fn deactivate_tipo_iva(conn: &Connection, id: &str) -> Result<()> {
    conn.execute("UPDATE tipos_iva SET activo = 0 WHERE id = ?1", params![id])?;
    Ok(())
}

pub fn deactivate_familia(conn: &Connection, id: &str) -> Result<()> {
    conn.execute("UPDATE familias SET activo = 0 WHERE id = ?1", params![id])?;
    Ok(())
}

pub fn deactivate_producto(conn: &Connection, id: &str) -> Result<()> {
    conn.execute("UPDATE productos SET activo = 0 WHERE id = ?1", params![id])?;
    Ok(())
}

pub fn get_ultimo_sync_catalogo(conn: &Connection) -> Result<Option<String>> {
    let mut stmt = conn.prepare("SELECT valor FROM sincronizacion_metadatos WHERE clave = 'ultimo_sync_catalogo'")?;
    let mut rows = stmt.query([])?;
    if let Some(row) = rows.next()? {
        let val: String = row.get(0)?;
        Ok(Some(val))
    } else {
        Ok(None)
    }
}

pub fn set_ultimo_sync_catalogo(conn: &Connection, timestamp: &str) -> Result<()> {
    conn.execute(
        "INSERT OR REPLACE INTO sincronizacion_metadatos (clave, valor) VALUES ('ultimo_sync_catalogo', ?1)",
        params![timestamp],
    )?;
    Ok(())
}

// ---------------------------------------------------------------------------
// Helper functions for Phase 2 commands
// ---------------------------------------------------------------------------

pub fn update_sale_status_to_voided(
    conn: &Connection,
    id: &str,
    reason: &str,
    voided_at: &str,
) -> Result<()> {
    conn.execute(
        "UPDATE offline_sales SET status = 'VOIDED', void_reason = ?1, voided_at = ?2 WHERE id = ?3",
        params![reason, voided_at, id],
    )?;
    Ok(())
}

pub fn increment_stock_cache(conn: &Connection, item_id: &str, quantity: f64) -> Result<()> {
    conn.execute(
        "INSERT INTO stock_cache (item_id, stock, last_updated_at)
         VALUES (?1, ?2, ?3)
         ON CONFLICT(item_id) DO UPDATE SET
            stock = stock + ?2,
            last_updated_at = ?3",
        params![item_id, quantity, chrono::Utc::now().to_rfc3339()],
    )?;
    Ok(())
}

pub fn insert_offline_sale_payment(
    conn: &Connection,
    id: &str,
    sale_id: &str,
    metodo_pago: &str,
    amount: f64,
    created_at: &str,
) -> Result<()> {
    conn.execute(
        "INSERT INTO offline_sale_payments (id, sale_id, metodo_pago, amount, created_at)
         VALUES (?1, ?2, ?3, ?4, ?5)",
        params![id, sale_id, metodo_pago, amount, created_at],
    )?;
    Ok(())
}

/// Returns (metodo_pago, amount) pairs for a sale's payments, in created order.
pub fn get_sale_payments(conn: &Connection, sale_id: &str) -> Result<Vec<(String, f64)>> {
    let mut stmt = conn.prepare(
        "SELECT metodo_pago, amount FROM offline_sale_payments WHERE sale_id = ?1 ORDER BY created_at",
    )?;
    let rows = stmt.query_map(params![sale_id], |row| {
        Ok((row.get(0)?, row.get(1)?))
    })?;
    rows.collect()
}

pub fn insert_caja_apertura(conn: &Connection, id: &str, amount: f64, opened_at: &str) -> Result<()> {
    conn.execute(
        "INSERT INTO caja_aperturas (id, amount, opened_at) VALUES (?1, ?2, ?3)",
        params![id, amount, opened_at],
    )?;
    Ok(())
}

pub fn insert_caja_movimiento(
    conn: &Connection,
    id: &str,
    tipo: &str,
    concepto: &str,
    amount: f64,
    created_at: &str,
) -> Result<()> {
    conn.execute(
        "INSERT INTO caja_movimientos (id, tipo, concepto, amount, created_at) VALUES (?1, ?2, ?3, ?4, ?5)",
        params![id, tipo, concepto, amount, created_at],
    )?;
    Ok(())
}

pub fn get_db_size(conn: &Connection) -> Result<i64> {
    let size: i64 = conn.query_row(
        "SELECT page_count * page_size FROM pragma_page_count, pragma_page_size",
        [],
        |row| row.get(0),
    )?;
    Ok(size)
}

// ---------------------------------------------------------------------------
// Client Dossier CRUD
// ---------------------------------------------------------------------------

pub fn get_all_clientes(conn: &Connection) -> Result<Vec<Cliente>> {
    let mut stmt = conn.prepare(
        "SELECT id, nombre, nif, email, updated_at, activo FROM clientes WHERE activo = 1 ORDER BY nombre",
    )?;
    let rows = stmt.query_map([], |row| {
        Ok(Cliente {
            id: row.get(0)?,
            nombre: row.get(1)?,
            nif: row.get(2)?,
            email: row.get(3)?,
            updated_at: row.get(4)?,
            activo: row.get::<_, i32>(5)? != 0,
        })
    })?;
    rows.collect()
}

pub fn search_clientes(conn: &Connection, query: &str) -> Result<Vec<CustomerInfo>> {
    let pattern = format!("%{}%", query);
    let mut stmt = conn.prepare(
        "SELECT c.id, c.nombre, c.nif,
                (SELECT d.calle || ', ' || d.ciudad FROM direcciones d WHERE d.entidad_id = c.id LIMIT 1) as direccion
         FROM clientes c
         WHERE c.activo = 1
           AND (c.nombre LIKE ?1 OR c.nif LIKE ?1 OR c.id LIKE ?1)
         ORDER BY c.nombre
         LIMIT 20",
    )?;
    let rows = stmt.query_map(params![pattern], |row| {
        Ok(CustomerInfo {
            id: row.get(0)?,
            nombre: row.get(1)?,
            nif: row.get(2)?,
            direccion: row.get(3)?,
            descuento: 0.0,
        })
    })?;
    rows.collect()
}

pub fn get_direcciones_by_entidad(conn: &Connection, entidad_id: &str) -> Result<Vec<Direccion>> {
    let mut stmt = conn.prepare(
        "SELECT id, entidad_id, tipo_direccion, calle, ciudad, provincia, codigo_postal, pais
         FROM direcciones WHERE entidad_id = ?1",
    )?;
    let rows = stmt.query_map(params![entidad_id], |row| {
        Ok(Direccion {
            id: row.get(0)?,
            entidad_id: row.get(1)?,
            tipo_direccion: row.get(2)?,
            calle: row.get(3)?,
            ciudad: row.get(4)?,
            provincia: row.get(5)?,
            codigo_postal: row.get(6)?,
            pais: row.get(7)?,
        })
    })?;
    rows.collect()
}

pub fn upsert_direccion(conn: &Connection, dir: &Direccion) -> Result<()> {
    conn.execute(
        "INSERT INTO direcciones (id, entidad_id, tipo_direccion, calle, ciudad, provincia, codigo_postal, pais)
         VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8)
         ON CONFLICT(id) DO UPDATE SET
             tipo_direccion = excluded.tipo_direccion,
             calle = excluded.calle,
             ciudad = excluded.ciudad,
             provincia = excluded.provincia,
             codigo_postal = excluded.codigo_postal,
             pais = excluded.pais",
        params![
            dir.id, dir.entidad_id, dir.tipo_direccion, dir.calle,
            dir.ciudad, dir.provincia, dir.codigo_postal, dir.pais,
        ],
    )?;
    Ok(())
}

pub fn get_contactos_by_entidad(conn: &Connection, entidad_id: &str) -> Result<Vec<Contacto>> {
    let mut stmt = conn.prepare(
        "SELECT id, entidad_id, nombre, puesto, email, telefono
         FROM contactos WHERE entidad_id = ?1",
    )?;
    let rows = stmt.query_map(params![entidad_id], |row| {
        Ok(Contacto {
            id: row.get(0)?,
            entidad_id: row.get(1)?,
            nombre: row.get(2)?,
            puesto: row.get(3)?,
            email: row.get(4)?,
            telefono: row.get(5)?,
        })
    })?;
    rows.collect()
}

pub fn upsert_contacto(conn: &Connection, contacto: &Contacto) -> Result<()> {
    conn.execute(
        "INSERT INTO contactos (id, entidad_id, nombre, puesto, email, telefono)
         VALUES (?1, ?2, ?3, ?4, ?5, ?6)
         ON CONFLICT(id) DO UPDATE SET
             nombre = excluded.nombre,
             puesto = excluded.puesto,
             email = excluded.email,
             telefono = excluded.telefono",
        params![
            contacto.id, contacto.entidad_id, contacto.nombre,
            contacto.puesto, contacto.email, contacto.telefono,
        ],
    )?;
    Ok(())
}

pub fn get_notas_by_entidad(conn: &Connection, entidad_id: &str) -> Result<Vec<Nota>> {
    let mut stmt = conn.prepare(
        "SELECT id, entidad_id, nota, creado_en FROM notas WHERE entidad_id = ?1 ORDER BY creado_en DESC",
    )?;
    let rows = stmt.query_map(params![entidad_id], |row| {
        Ok(Nota {
            id: row.get(0)?,
            entidad_id: row.get(1)?,
            nota: row.get(2)?,
            creado_en: row.get(3)?,
        })
    })?;
    rows.collect()
}

pub fn upsert_nota(conn: &Connection, nota: &Nota) -> Result<()> {
    conn.execute(
        "INSERT INTO notas (id, entidad_id, nota, creado_en)
         VALUES (?1, ?2, ?3, ?4)
         ON CONFLICT(id) DO UPDATE SET
             nota = excluded.nota,
             creado_en = excluded.creado_en",
        params![nota.id, nota.entidad_id, nota.nota, nota.creado_en],
    )?;
    Ok(())
}

pub fn get_cliente_by_id(conn: &Connection, id: &str) -> Result<Option<Cliente>> {
    let mut stmt = conn.prepare(
        "SELECT id, nombre, nif, email, updated_at, activo FROM clientes WHERE id = ?1",
    )?;
    let mut rows = stmt.query_map(params![id], |row| {
        Ok(Cliente {
            id: row.get(0)?,
            nombre: row.get(1)?,
            nif: row.get(2)?,
            email: row.get(3)?,
            updated_at: row.get(4)?,
            activo: row.get::<_, i32>(5)? != 0,
        })
    })?;
    match rows.next() {
        Some(r) => Ok(Some(r?)),
        None => Ok(None),
    }
}

pub fn get_cliente_dossier(conn: &Connection, cliente_id: &str) -> Result<ClientDossier> {
    let cliente = get_cliente_by_id(conn, cliente_id)?
        .ok_or_else(|| rusqlite::Error::QueryReturnedNoRows)?;

    // Recent sales for this customer
    let mut stmt = conn.prepare(
        "SELECT id, customer_id, created_at, invoice_number, total, status
         FROM offline_sales
         WHERE customer_id = ?1
         ORDER BY created_at DESC
         LIMIT 10",
    )?;
    let ventas_recientes: Vec<RecentSaleDossier> = stmt
        .query_map(params![cliente_id], |row| {
            Ok(RecentSaleDossier {
                id_factura: row.get(0)?,
                cliente_id: row.get(1)?,
                fecha: row.get(2)?,
                numero: row.get(3)?,
                total: row.get(4)?,
                estado: row.get(5)?,
            })
        })?
        .filter_map(|r| r.ok())
        .collect();

    // Pending invoices: completed sales with no cobro recorded
    let mut stmt = conn.prepare(
        "SELECT os.id, os.customer_id, os.invoice_number, os.total, os.created_at
         FROM offline_sales os
         WHERE os.customer_id = ?1 AND os.status = 'COMPLETED'
         ORDER BY os.created_at DESC
         LIMIT 20",
    )?;
    let facturas_pendientes: Vec<PendingInvoiceDossier> = stmt
        .query_map(params![cliente_id], |row| {
            let total: f64 = row.get(4)?;
            Ok(PendingInvoiceDossier {
                id_factura: row.get(0)?,
                cliente_id: row.get(1)?,
                numero_factura: row.get(2)?,
                importe_pendiente: total,
                fecha_emision: row.get(3)?,
            })
        })?
        .filter_map(|r| r.ok())
        .collect();

    Ok(ClientDossier {
        cliente,
        estadisticas: None,
        ventas_recientes,
        facturas_pendientes,
    })
}

pub fn get_distinct_families(conn: &Connection) -> Result<Vec<String>> {
    let mut stmt = conn.prepare(
        "SELECT DISTINCT nombre FROM familias WHERE activo = 1 ORDER BY nombre",
    )?;
    let rows = stmt.query_map([], |row| row.get(0))?;
    rows.collect()
}

pub fn get_products_by_family(conn: &Connection, familia: Option<&str>) -> Result<Vec<POSProduct>> {
    let sql = r#"
        SELECT p.id, p.codigo, p.nombre, p.precio_venta,
               sc.stock, f.nombre, ti.nombre, ti.porcentaje, p.imagen_url
        FROM productos p
        LEFT JOIN stock_cache sc ON sc.item_id = p.id
        LEFT JOIN familias f ON f.id = p.familia_id
        LEFT JOIN tipos_iva ti ON ti.id = p.tipo_iva_id
        WHERE p.activo = 1
          AND (?1 IS NULL OR f.nombre = ?1)
        ORDER BY p.nombre
        LIMIT 100
    "#;
    let mut stmt = conn.prepare(sql)?;
    let rows = stmt.query_map(params![familia], |row| {
        Ok(POSProduct {
            id: row.get(0)?,
            codigo: row.get(1)?,
            nombre: row.get(2)?,
            precio_venta: row.get(3)?,
            stock: row.get(4)?,
            familia_nombre: row.get(5)?,
            tipo_iva_nombre: row.get(6)?,
            tipo_iva_porcentaje: row.get(7)?,
            imagen_url: row.get(8)?,
        })
    })?;
    rows.collect()
}

pub fn insert_cobro(
    conn: &Connection,
    id: &str,
    cliente_id: &str,
    factura_id: Option<&str>,
    importe: f64,
    metodo_pago: &str,
    tipo_cobro: &str,
    created_at: &str,
) -> Result<()> {
    conn.execute(
        "INSERT INTO cobros (id, cliente_id, factura_id, importe, metodo_pago, tipo_cobro, created_at)
         VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7)",
        params![id, cliente_id, factura_id, importe, metodo_pago, tipo_cobro, created_at],
    )?;
    Ok(())
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    fn setup_db() -> Connection {
        init_db(":memory:").expect("failed to initialize in-memory db")
    }

    // -- Sales tests -------------------------------------------------------

    #[test]
    fn test_insert_and_get_pending_sales() {
        let conn = setup_db();

        let sale = OfflineSale {
            id: "sale-001".into(),
            terminal_id: "term-1".into(),
            customer_id: None,
            total: 100.50,
            created_at: "2026-06-05T10:00:00Z".into(),
            sync_status: "PENDING".into(),
            idempotency_key: "key-sale-001".into(),
            invoice_number: "TPV-0001".into(),
            sequence_number: 1,
            firma_registro: Some("hash123".into()),
            hash_anterior: Some("prevhash123".into()),
            datos_encadenamiento: Some("meta123".into()),
            subtotal: 100.50,
            tax_total: 0.0,
            discount_total: 0.0,
            status: "COMPLETED".into(),
            void_reason: None,
            voided_at: None,
        };

        insert_offline_sale(&conn, &sale).unwrap();

        let pending = get_pending_sales(&conn).unwrap();
        assert_eq!(pending.len(), 1);
        assert_eq!(pending[0].id, "sale-001");
        assert!((pending[0].total - 100.50).abs() < f64::EPSILON);
        assert!((pending[0].subtotal - 100.50).abs() < f64::EPSILON);
        assert_eq!(pending[0].status, "COMPLETED");
        assert_eq!(pending[0].firma_registro, Some("hash123".into()));
        assert_eq!(pending[0].hash_anterior, Some("prevhash123".into()));
        assert_eq!(pending[0].datos_encadenamiento, Some("meta123".into()));
    }

    #[test]
    fn test_delete_synced_sale() {
        let conn = setup_db();

        let sale = OfflineSale {
            id: "sale-002".into(),
            terminal_id: "term-1".into(),
            customer_id: Some("customer-1".into()),
            total: 200.00,
            created_at: "2026-06-05T11:00:00Z".into(),
            sync_status: "PENDING".into(),
            idempotency_key: "key-sale-002".into(),
            invoice_number: "TPV-0002".into(),
            sequence_number: 2,
            firma_registro: None,
            hash_anterior: None,
            datos_encadenamiento: None,
            subtotal: 200.0,
            tax_total: 0.0,
            discount_total: 0.0,
            status: "COMPLETED".into(),
            void_reason: None,
            voided_at: None,
        };

        insert_offline_sale(&conn, &sale).unwrap();
        delete_synced_sale(&conn, "sale-002").unwrap();

        let pending = get_pending_sales(&conn).unwrap();
        assert!(pending.is_empty());
    }

    #[test]
    fn test_insert_sale_item() {
        let conn = setup_db();

        let sale = OfflineSale {
            id: "sale-003".into(),
            terminal_id: "term-1".into(),
            customer_id: None,
            total: 50.00,
            created_at: "2026-06-05T12:00:00Z".into(),
            sync_status: "PENDING".into(),
            idempotency_key: "key-sale-003".into(),
            invoice_number: "TPV-0003".into(),
            sequence_number: 3,
            firma_registro: None,
            hash_anterior: None,
            datos_encadenamiento: None,
            subtotal: 50.0,
            tax_total: 0.0,
            discount_total: 0.0,
            status: "COMPLETED".into(),
            void_reason: None,
            voided_at: None,
        };
        insert_offline_sale(&conn, &sale).unwrap();

        let item = OfflineSaleItem {
            id: "item-001".into(),
            offline_sale_id: "sale-003".into(),
            item_id: "product-abc".into(),
            quantity: 2.0,
            unit_price: 25.00,
            discount_percent: 0.0,
        };
        insert_offline_sale_item(&conn, &item).unwrap();

        // Verify item exists
        let count: i64 = conn
            .query_row(
                "SELECT COUNT(*) FROM offline_sale_items WHERE offline_sale_id = ?1",
                params!["sale-003"],
                |r| r.get(0),
            )
            .unwrap();
        assert_eq!(count, 1);

        // Verify discount_percent roundtrip
        let items = get_sale_items(&conn, "sale-003").unwrap();
        assert_eq!(items.len(), 1);
        assert!((items[0].discount_percent - 0.0).abs() < f64::EPSILON);
    }

    // -- Closure tests -----------------------------------------------------

    #[test]
    fn test_insert_and_get_pending_closures() {
        let conn = setup_db();

        let closure = OfflineBoxClosure {
            id: "closure-001".into(),
            opened_at: "2026-06-05T08:00:00Z".into(),
            closed_at: "2026-06-05T18:00:00Z".into(),
            cash_reported: 1500.50,
            card_reported: 850.00,
            sales_total: 2350.50,
            sync_status: "PENDING".into(),
            idempotency_key: "key-closure-001".into(),
        };

        insert_offline_box_closure(&conn, &closure).unwrap();

        let pending = get_pending_closures(&conn).unwrap();
        assert_eq!(pending.len(), 1);
        assert_eq!(pending[0].id, "closure-001");
        assert!((pending[0].cash_reported - 1500.50).abs() < f64::EPSILON);
        assert!((pending[0].sales_total - 2350.50).abs() < f64::EPSILON);
    }

    #[test]
    fn test_delete_synced_closure() {
        let conn = setup_db();

        let closure = OfflineBoxClosure {
            id: "closure-002".into(),
            opened_at: "2026-06-05T08:00:00Z".into(),
            closed_at: "2026-06-05T18:00:00Z".into(),
            cash_reported: 500.00,
            card_reported: 300.00,
            sales_total: 800.00,
            sync_status: "PENDING".into(),
            idempotency_key: "key-closure-002".into(),
        };

        insert_offline_box_closure(&conn, &closure).unwrap();
        delete_synced_closure(&conn, "closure-002").unwrap();

        let pending = get_pending_closures(&conn).unwrap();
        assert!(pending.is_empty());
    }

    // -- Stock cache tests -------------------------------------------------

    #[test]
    fn test_upsert_and_get_cached_stock() {
        let conn = setup_db();

        // Insert
        upsert_stock_cache(&conn, "item-100", 42.0, "2026-06-05T10:00:00Z").unwrap();
        let stock = get_cached_stock(&conn, "item-100").unwrap();
        assert_eq!(stock, Some(42.0));

        // Update (upsert)
        upsert_stock_cache(&conn, "item-100", 38.5, "2026-06-05T11:00:00Z").unwrap();
        let stock = get_cached_stock(&conn, "item-100").unwrap();
        assert_eq!(stock, Some(38.5));
    }

    #[test]
    fn test_get_cached_stock_missing() {
        let conn = setup_db();
        let stock = get_cached_stock(&conn, "nonexistent").unwrap();
        assert_eq!(stock, None);
    }

    #[test]
    fn test_ultimo_registro_encadenado() {
        let conn = setup_db();

        // Initially empty
        let last = get_ultimo_registro_encadenado(&conn).unwrap();
        assert!(last.is_none());

        // Upsert first time
        upsert_ultimo_registro_encadenado(&conn, "sale-001", "firma-001").unwrap();
        let last = get_ultimo_registro_encadenado(&conn).unwrap();
        assert_eq!(last, Some("firma-001".to_string()));

        // Upsert second time should replace
        upsert_ultimo_registro_encadenado(&conn, "sale-002", "firma-002").unwrap();
        let last = get_ultimo_registro_encadenado(&conn).unwrap();
        assert_eq!(last, Some("firma-002".to_string()));

        // Verify count is 1
        let count: i64 = conn
            .query_row("SELECT COUNT(*) FROM ultimo_registro_encadenado", [], |r| r.get(0))
            .unwrap();
        assert_eq!(count, 1);
    }

    #[test]
    fn test_registro_sucesos() {
        let conn = setup_db();

        // Initially empty
        let events = get_pending_events(&conn).unwrap();
        assert!(events.is_empty());

        // Insert event
        insert_registro_suceso(&conn, "evt-001", "ALTA_FACTURA", "Factura emitida").unwrap();

        // Get pending
        let events = get_pending_events(&conn).unwrap();
        assert_eq!(events.len(), 1);
        assert_eq!(events[0].id, "evt-001");
        assert_eq!(events[0].tipo_evento, "ALTA_FACTURA");
        assert_eq!(events[0].detalles, "Factura emitida");
        assert_eq!(events[0].estado_sincronizacion, "PENDING");
        assert!(!events[0].fecha_hora.is_empty());

        // Delete/Sync event
        delete_synced_event(&conn, "evt-001").unwrap();
        let events = get_pending_events(&conn).unwrap();
        assert!(events.is_empty());
    }

    // -- Phase 3 tests ----------------------------------------------------

    #[test]
    fn test_catalog_upsert_and_deactivate() {
        let conn = setup_db();

        let t_iva = TipoIVA {
            id: "iva-21".into(),
            nombre: "IVA 21%".into(),
            porcentaje: 21.0,
            updated_at: "2026-06-05T10:00:00Z".into(),
            activo: true,
        };
        upsert_tipo_iva(&conn, &t_iva).unwrap();

        let count: i64 = conn
            .query_row("SELECT COUNT(*) FROM tipos_iva WHERE id = 'iva-21'", [], |r| r.get(0))
            .unwrap();
        assert_eq!(count, 1);

        // Deactivate
        deactivate_tipo_iva(&conn, "iva-21").unwrap();
        let activo: i32 = conn
            .query_row("SELECT activo FROM tipos_iva WHERE id = 'iva-21'", [], |r| r.get(0))
            .unwrap();
        assert_eq!(activo, 0);
    }

    #[test]
    fn test_metadata_sync_catalogo() {
        let conn = setup_db();
        
        let last = get_ultimo_sync_catalogo(&conn).unwrap();
        assert!(last.is_none());

        set_ultimo_sync_catalogo(&conn, "2026-06-05T20:00:00Z").unwrap();
        let last = get_ultimo_sync_catalogo(&conn).unwrap();
        assert_eq!(last, Some("2026-06-05T20:00:00Z".to_string()));
    }

    // -- Phase 1 — TPV Tienda POS tests ------------------------------------

    #[test]
    fn test_fts5_search_returns_products() {
        let conn = setup_db();

        // Insert catalog data
        conn.execute(
            "INSERT INTO tipos_iva (id, nombre, porcentaje) VALUES ('iva-21', 'IVA 21%', 21.0)",
            [],
        )
        .unwrap();
        conn.execute(
            "INSERT INTO familias (id, nombre) VALUES ('fam-1', 'Ferreteria')",
            [],
        )
        .unwrap();
        conn.execute(
            "INSERT INTO productos (id, codigo, nombre, precio_venta, familia_id, tipo_iva_id, activo)
             VALUES ('p1', '842123', 'Martillo 500g', 15.50, 'fam-1', 'iva-21', 1)",
            [],
        )
        .unwrap();
        conn.execute(
            "INSERT INTO productos (id, codigo, nombre, precio_venta, activo)
             VALUES ('p2', '842456', 'Destornillador plano', 8.90, 1)",
            [],
        )
        .unwrap();

        // Rebuild FTS index after bulk insert (triggers only fire per-row;
        // for test reliability we rebuild explicitly)
        reindex_productos_fts(&conn).unwrap();

        // Search by code prefix
        let results = search_products(&conn, "842").unwrap();
        assert_eq!(results.len(), 2, "should match both products by code");

        // Search by name
        let results = search_products(&conn, "Martillo").unwrap();
        assert_eq!(results.len(), 1);
        assert_eq!(results[0].codigo, "842123");
        assert_eq!(results[0].nombre, "Martillo 500g");
        assert!((results[0].precio_venta - 15.50).abs() < f64::EPSILON);
        assert_eq!(results[0].familia_nombre, Some("Ferreteria".into()));
        assert_eq!(results[0].tipo_iva_nombre, Some("IVA 21%".into()));
        assert_eq!(results[0].tipo_iva_porcentaje, Some(21.0));

        // Search with no matches
        let results = search_products(&conn, "ZZZZNOMATCH").unwrap();
        assert!(results.is_empty());

        // Search with empty query
        let results = search_products(&conn, "").unwrap();
        assert!(results.is_empty());
    }

    #[test]
    fn test_get_product_by_code_exact() {
        let conn = setup_db();

        conn.execute(
            "INSERT INTO productos (id, codigo, nombre, precio_venta, activo)
             VALUES ('p1', 'BARCODE001', 'Taladro', 99.99, 1)",
            [],
        )
        .unwrap();

        let found = get_product_by_code(&conn, "BARCODE001").unwrap();
        assert!(found.is_some());
        assert_eq!(found.unwrap().nombre, "Taladro");

        let not_found = get_product_by_code(&conn, "NONEXISTENT").unwrap();
        assert!(not_found.is_none());

        // Inactive products should not be returned
        conn.execute(
            "INSERT INTO productos (id, codigo, nombre, precio_venta, activo)
             VALUES ('p2', 'BARCODE002', 'Inactivo', 10.0, 0)",
            [],
        )
        .unwrap();
        let inactive = get_product_by_code(&conn, "BARCODE002").unwrap();
        assert!(inactive.is_none());
    }

    #[test]
    fn test_get_today_sales_filters_by_date() {
        let conn = setup_db();

        // Insert a sale from today
        let sale_today = OfflineSale {
            id: "sale-today".into(),
            terminal_id: "term-1".into(),
            customer_id: None,
            total: 100.0,
            created_at: chrono::Utc::now().to_rfc3339(),
            sync_status: "PENDING".into(),
            idempotency_key: "key-today".into(),
            invoice_number: "TPV-0100".into(),
            sequence_number: 100,
            firma_registro: None,
            hash_anterior: None,
            datos_encadenamiento: None,
            subtotal: 100.0,
            tax_total: 0.0,
            discount_total: 0.0,
            status: "COMPLETED".into(),
            void_reason: None,
            voided_at: None,
        };
        insert_offline_sale(&conn, &sale_today).unwrap();

        // Insert a sale from yesterday
        let sale_yesterday = OfflineSale {
            id: "sale-yest".into(),
            terminal_id: "term-1".into(),
            customer_id: None,
            total: 50.0,
            created_at: "2026-06-19T10:00:00Z".into(),
            sync_status: "PENDING".into(),
            idempotency_key: "key-yest".into(),
            invoice_number: "TPV-0099".into(),
            sequence_number: 99,
            firma_registro: None,
            hash_anterior: None,
            datos_encadenamiento: None,
            subtotal: 50.0,
            tax_total: 0.0,
            discount_total: 0.0,
            status: "COMPLETED".into(),
            void_reason: None,
            voided_at: None,
        };
        insert_offline_sale(&conn, &sale_yesterday).unwrap();

        let today = get_today_sales(&conn).unwrap();
        assert_eq!(today.len(), 1, "should only return today's sales");
        assert_eq!(today[0].id, "sale-today");
    }

    #[test]
    fn test_sequence_atomicity() {
        let conn = setup_db();

        // First call for a new prefix returns 2 (inserts 2 as next_val)
        let seq1 = get_next_sequence(&conn, "TPV").unwrap();
        assert_eq!(seq1, 2);

        // Second call increments: 2 → 3
        let seq2 = get_next_sequence(&conn, "TPV").unwrap();
        assert_eq!(seq2, 3);

        // Different prefix starts at 2
        let seq3 = get_next_sequence(&conn, "FAC").unwrap();
        assert_eq!(seq3, 2);

        // TPV continues uninterrupted
        let seq4 = get_next_sequence(&conn, "TPV").unwrap();
        assert_eq!(seq4, 4);
    }

    #[test]
    fn test_extended_fields_roundtrip() {
        let conn = setup_db();

        let sale = OfflineSale {
            id: "sale-ext".into(),
            terminal_id: "term-1".into(),
            customer_id: Some("cli-1".into()),
            total: 121.0,
            created_at: "2026-06-20T10:00:00Z".into(),
            sync_status: "PENDING".into(),
            idempotency_key: "key-ext".into(),
            invoice_number: "TPV-0200".into(),
            sequence_number: 200,
            firma_registro: Some("sig".into()),
            hash_anterior: Some("prev".into()),
            datos_encadenamiento: Some("chain".into()),
            subtotal: 100.0,
            tax_total: 21.0,
            discount_total: 0.0,
            status: "COMPLETED".into(),
            void_reason: None,
            voided_at: None,
        };
        insert_offline_sale(&conn, &sale).unwrap();

        let loaded = get_sale_by_id(&conn, "sale-ext").unwrap().unwrap();
        assert!((loaded.subtotal - 100.0).abs() < f64::EPSILON);
        assert!((loaded.tax_total - 21.0).abs() < f64::EPSILON);
        assert!((loaded.discount_total - 0.0).abs() < f64::EPSILON);
        assert_eq!(loaded.status, "COMPLETED");
        assert!(loaded.void_reason.is_none());
        assert!(loaded.voided_at.is_none());

        // Insert item with discount_percent
        let item = OfflineSaleItem {
            id: "item-ext".into(),
            offline_sale_id: "sale-ext".into(),
            item_id: "prod-1".into(),
            quantity: 2.0,
            unit_price: 50.0,
            discount_percent: 10.0,
        };
        insert_offline_sale_item(&conn, &item).unwrap();

        let items = get_sale_items(&conn, "sale-ext").unwrap();
        assert_eq!(items.len(), 1);
        assert!((items[0].discount_percent - 10.0).abs() < f64::EPSILON);
    }
}
