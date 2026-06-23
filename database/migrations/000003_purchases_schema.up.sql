-- Migration for Purchases Cycle (Multi-Company & Multi-Warehouse)

CREATE TABLE empresas (
    id UUID PRIMARY KEY,
    razon_social VARCHAR(150) NOT NULL,
    nif VARCHAR(20) UNIQUE NOT NULL,
    activa BOOLEAN DEFAULT TRUE
);

CREATE TABLE almacenes (
    id UUID PRIMARY KEY,
    empresa_id UUID NOT NULL REFERENCES empresas(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    active BOOLEAN DEFAULT TRUE,
    UNIQUE(empresa_id, name)
);

CREATE TABLE proveedores (
    id UUID PRIMARY KEY,
    empresa_id UUID NOT NULL REFERENCES empresas(id) ON DELETE CASCADE,
    razon_social VARCHAR(150) NOT NULL,
    cif VARCHAR(20) NOT NULL,
    email VARCHAR(100),
    telefono VARCHAR(20),
    direccion TEXT,
    activo BOOLEAN DEFAULT TRUE,
    UNIQUE(empresa_id, cif)
);

CREATE TABLE pedidos_compra (
    id UUID PRIMARY KEY,
    empresa_id UUID NOT NULL REFERENCES empresas(id) ON DELETE CASCADE,
    proveedor_id UUID NOT NULL REFERENCES proveedores(id) ON DELETE RESTRICT,
    numero_pedido VARCHAR(50) NOT NULL,
    fecha TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    estado VARCHAR(20) NOT NULL CHECK (estado IN ('Borrador', 'Aprobado', 'Recibido', 'Cancelado')),
    total NUMERIC(12,2) NOT NULL DEFAULT 0.00,
    UNIQUE(empresa_id, numero_pedido)
);

CREATE TABLE pedido_compra_lineas (
    id UUID PRIMARY KEY,
    pedido_compra_id UUID NOT NULL REFERENCES pedidos_compra(id) ON DELETE CASCADE,
    producto_id UUID NOT NULL REFERENCES productos(id) ON DELETE RESTRICT,
    cantidad NUMERIC(12,4) NOT NULL,
    precio_unitario NUMERIC(12,2) NOT NULL
);

CREATE TABLE recepciones_compra (
    id UUID PRIMARY KEY,
    empresa_id UUID NOT NULL REFERENCES empresas(id) ON DELETE CASCADE,
    pedido_compra_id UUID REFERENCES pedidos_compra(id) ON DELETE SET NULL,
    proveedor_id UUID NOT NULL REFERENCES proveedores(id) ON DELETE RESTRICT,
    numero_albaran VARCHAR(50) NOT NULL,
    fecha TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    estado VARCHAR(20) NOT NULL CHECK (estado IN ('Borrador', 'Procesado', 'Cancelado')),
    almacen_id UUID NOT NULL REFERENCES almacenes(id) ON DELETE RESTRICT,
    UNIQUE(empresa_id, numero_albaran)
);

CREATE TABLE recepcion_compra_lineas (
    id UUID PRIMARY KEY,
    recepcion_compra_id UUID NOT NULL REFERENCES recepciones_compra(id) ON DELETE CASCADE,
    producto_id UUID NOT NULL REFERENCES productos(id) ON DELETE RESTRICT,
    cantidad NUMERIC(12,4) NOT NULL,
    precio_unitario NUMERIC(12,2) NOT NULL
);

-- Seed default company and default almacen
INSERT INTO empresas (id, razon_social, nif, activa)
VALUES ('00000000-0000-4000-a000-000000000001', 'Ferrowin S.L.', 'B12345678', TRUE)
ON CONFLICT (id) DO NOTHING;

INSERT INTO almacenes (id, empresa_id, name, active)
VALUES ('00000000-0000-4000-a000-000000000002', '00000000-0000-4000-a000-000000000001', 'Almacén Central', TRUE)
ON CONFLICT (id) DO NOTHING;

-- Update existing ledger movements to default almacen
UPDATE stock_ledger_movements
SET almacen_id = '00000000-0000-4000-a000-000000000002'
WHERE almacen_id IS NOT NULL;

-- Add foreign key constraint to stock_ledger_movements
ALTER TABLE stock_ledger_movements
ADD CONSTRAINT fk_stock_ledger_movements_almacen
FOREIGN KEY (almacen_id) REFERENCES almacenes(id);
