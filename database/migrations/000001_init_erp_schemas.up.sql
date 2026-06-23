-- PostgreSQL migration for core ERP schemas

-- RBAC Security
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL
);

CREATE TABLE groups (
    id UUID PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL
);

CREATE TABLE user_groups (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    group_id UUID REFERENCES groups(id) ON DELETE CASCADE,
    PRIMARY KEY(user_id, group_id)
);

CREATE TABLE role_sets (
    id UUID PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL
);

CREATE TABLE group_role_sets (
    group_id UUID REFERENCES groups(id) ON DELETE CASCADE,
    role_set_id UUID REFERENCES role_sets(id) ON DELETE CASCADE,
    PRIMARY KEY(group_id, role_set_id)
);

CREATE TABLE roles (
    id UUID PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL
);

CREATE TABLE role_set_roles (
    role_set_id UUID REFERENCES role_sets(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY(role_set_id, role_id)
);

-- Terminals & Billing Series
CREATE TABLE terminals (
    id UUID PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE series_facturacion (
    id UUID PRIMARY KEY,
    terminal_id UUID REFERENCES terminals(id) ON DELETE RESTRICT,
    prefix VARCHAR(10) UNIQUE NOT NULL,
    next_sequence INT NOT NULL DEFAULT 1
);

-- Traceable Sales Documents
CREATE TABLE presupuestos (
    id UUID PRIMARY KEY,
    empresa_id UUID NOT NULL,
    cliente_id UUID NOT NULL,
    total NUMERIC(12,2) NOT NULL,
    estado VARCHAR(20) CHECK (estado IN ('Borrador', 'Aprobado', 'Convertido', 'Anulado')),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE pedidos (
    id UUID PRIMARY KEY,
    empresa_id UUID NOT NULL,
    presupuesto_id UUID REFERENCES presupuestos(id) ON DELETE SET NULL,
    total NUMERIC(12,2) NOT NULL,
    estado VARCHAR(20) CHECK (estado IN ('Borrador', 'Aprobado', 'Convertido', 'Anulado')),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE albaranes (
    id UUID PRIMARY KEY,
    empresa_id UUID NOT NULL,
    pedido_id UUID REFERENCES pedidos(id) ON DELETE SET NULL,
    almacen_id UUID NOT NULL,
    total NUMERIC(12,2) NOT NULL,
    estado VARCHAR(20) CHECK (estado IN ('Borrador', 'Convertido', 'Anulado')),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE facturas (
    id UUID PRIMARY KEY,
    empresa_id UUID NOT NULL,
    albaran_id UUID REFERENCES albaranes(id) ON DELETE SET NULL,
    terminal_id UUID REFERENCES terminals(id) ON DELETE RESTRICT,
    serie_facturacion_id UUID REFERENCES series_facturacion(id) ON DELETE RESTRICT,
    numero_factura VARCHAR(30) UNIQUE NOT NULL,
    numero_secuencia INT NOT NULL,
    total NUMERIC(12,2) NOT NULL,
    estado VARCHAR(20),
    created_at TIMESTAMP DEFAULT NOW(),
    firma_registro VARCHAR(255),
    hash_anterior VARCHAR(255),
    datos_encadenamiento TEXT
);

CREATE TABLE registro_sucesos (
    id UUID PRIMARY KEY,
    fecha_hora TIMESTAMPTZ NOT NULL,
    tipo_evento VARCHAR(50) NOT NULL,
    detalles TEXT NOT NULL,
    estado_sincronizacion VARCHAR(20) NOT NULL DEFAULT 'PENDIENTE'
);

-- Stock Ledger
CREATE TABLE stock_ledger_movements (
    id UUID PRIMARY KEY,
    item_id UUID NOT NULL,
    almacen_id UUID NOT NULL,
    quantity NUMERIC(12,4) NOT NULL,
    movement_type VARCHAR(20) CHECK (movement_type IN ('RECEIPT', 'WITHDRAWAL', 'SYNC_ADJUSTMENT')),
    reference_document_type VARCHAR(20),
    reference_document_id UUID,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Foreign Key & Performance Indexes
CREATE INDEX idx_user_groups_user_id ON user_groups(user_id);
CREATE INDEX idx_user_groups_group_id ON user_groups(group_id);
CREATE INDEX idx_group_role_sets_group_id ON group_role_sets(group_id);
CREATE INDEX idx_group_role_sets_role_set_id ON group_role_sets(role_set_id);
CREATE INDEX idx_role_set_roles_role_set_id ON role_set_roles(role_set_id);
CREATE INDEX idx_role_set_roles_role_id ON role_set_roles(role_id);
CREATE INDEX idx_series_facturacion_terminal_id ON series_facturacion(terminal_id);
CREATE INDEX idx_pedido_presupuesto_id ON pedidos(presupuesto_id);
CREATE INDEX idx_albaranes_pedido_id ON albaranes(pedido_id);
CREATE INDEX idx_facturas_albaran_id ON facturas(albaran_id);
CREATE INDEX idx_facturas_terminal_id ON facturas(terminal_id);
CREATE INDEX idx_facturas_serie_facturacion_id ON facturas(serie_facturacion_id);
CREATE INDEX idx_stock_ledger_movements_item_almacen ON stock_ledger_movements(item_id, almacen_id);

-- Box Closures (TPV sync)
CREATE TABLE box_closures (
    id UUID PRIMARY KEY,
    opened_at TIMESTAMPTZ NOT NULL,
    closed_at TIMESTAMPTZ NOT NULL,
    cash_reported NUMERIC(12,2) NOT NULL,
    card_reported NUMERIC(12,2) NOT NULL,
    sales_total NUMERIC(12,2) NOT NULL,
    terminal_id UUID REFERENCES terminals(id),
    synced_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_box_closures_terminal_id ON box_closures(terminal_id);
