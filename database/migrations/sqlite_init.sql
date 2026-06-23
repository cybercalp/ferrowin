-- SQLite schema setup script for local POS TPV

PRAGMA foreign_keys = ON;

CREATE TABLE offline_sales (
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

CREATE TABLE ultimo_registro_encadenado (
    id_factura TEXT PRIMARY KEY,
    firma_registro TEXT NOT NULL
);

CREATE TABLE registro_sucesos (
    id TEXT PRIMARY KEY,
    fecha_hora TEXT NOT NULL,
    tipo_evento TEXT NOT NULL,
    detalles TEXT NOT NULL,
    estado_sincronizacion TEXT NOT NULL DEFAULT 'PENDIENTE'
);

CREATE TABLE offline_sale_items (
    id TEXT PRIMARY KEY,
    offline_sale_id TEXT REFERENCES offline_sales(id) ON DELETE CASCADE,
    item_id TEXT NOT NULL,
    quantity REAL NOT NULL,
    unit_price REAL NOT NULL
);

-- Offline Indexes
CREATE INDEX idx_offline_sales_terminal_id ON offline_sales(terminal_id);
CREATE INDEX idx_offline_sales_sync_status ON offline_sales(sync_status);
CREATE INDEX idx_offline_sale_items_offline_sale_id ON offline_sale_items(offline_sale_id);

CREATE TABLE offline_box_closures (
    id TEXT PRIMARY KEY,
    opened_at TEXT NOT NULL,
    closed_at TEXT NOT NULL,
    cash_reported REAL NOT NULL,
    card_reported REAL NOT NULL,
    sales_total REAL NOT NULL,
    sync_status TEXT NOT NULL DEFAULT 'PENDING',
    idempotency_key TEXT UNIQUE NOT NULL
);

CREATE TABLE stock_cache (
    item_id TEXT PRIMARY KEY,
    stock REAL NOT NULL,
    last_updated_at TEXT NOT NULL
);

CREATE TABLE tipos_iva (
    id TEXT PRIMARY KEY,
    nombre TEXT NOT NULL,
    porcentaje REAL NOT NULL,
    updated_at TEXT,
    activo INTEGER DEFAULT 1
);

CREATE TABLE familias (
    id TEXT PRIMARY KEY,
    nombre TEXT NOT NULL,
    updated_at TEXT,
    activo INTEGER DEFAULT 1
);

CREATE TABLE productos (
    id TEXT PRIMARY KEY,
    codigo TEXT UNIQUE NOT NULL,
    nombre TEXT NOT NULL,
    precio_venta REAL NOT NULL,
    familia_id TEXT,
    tipo_iva_id TEXT,
    updated_at TEXT,
    activo INTEGER DEFAULT 1
);

CREATE TABLE clientes (
    id TEXT PRIMARY KEY,
    nombre TEXT NOT NULL,
    nif TEXT,
    email TEXT,
    updated_at TEXT,
    activo INTEGER DEFAULT 1
);

CREATE TABLE cliente_ventas_recientes (
    id_factura TEXT PRIMARY KEY,
    cliente_id TEXT,
    fecha TEXT,
    numero TEXT,
    total REAL,
    estado TEXT
);

CREATE TABLE cliente_estadisticas (
    cliente_id TEXT PRIMARY KEY,
    saldo_pendiente REAL,
    limite_credito REAL,
    articulos_mas_comprados_json TEXT
);

CREATE TABLE cliente_facturas_pendientes (
    id_factura TEXT PRIMARY KEY,
    cliente_id TEXT,
    numero_factura TEXT,
    importe_pendiente REAL,
    fecha_emision TEXT
);

CREATE TABLE offline_cobros_recibidos (
    id TEXT PRIMARY KEY,
    cliente_id TEXT,
    factura_id TEXT,
    importe REAL,
    fecha TEXT,
    metodo_pago TEXT,
    tipo_cobro TEXT,
    sync_status TEXT DEFAULT 'PENDING',
    idempotency_key TEXT UNIQUE NOT NULL
);

CREATE TABLE sincronizacion_metadatos (
    clave TEXT PRIMARY KEY,
    valor TEXT
);


