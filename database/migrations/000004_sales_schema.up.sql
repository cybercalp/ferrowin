-- PostgreSQL migration for updating sales documents and adding lines

-- Step 1: Add empresa_id to primary sales document tables (presupuestos, pedidos, albaranes, facturas)
ALTER TABLE presupuestos ADD COLUMN empresa_id UUID;
ALTER TABLE pedidos ADD COLUMN empresa_id UUID;
ALTER TABLE albaranes ADD COLUMN empresa_id UUID;
ALTER TABLE facturas ADD COLUMN empresa_id UUID;

-- Step 2: Assign default company ID to any existing records
UPDATE presupuestos SET empresa_id = '00000000-0000-4000-a000-000000000001' WHERE empresa_id IS NULL;
UPDATE pedidos SET empresa_id = '00000000-0000-4000-a000-000000000001' WHERE empresa_id IS NULL;
UPDATE albaranes SET empresa_id = '00000000-0000-4000-a000-000000000001' WHERE empresa_id IS NULL;
UPDATE facturas SET empresa_id = '00000000-0000-4000-a000-000000000001' WHERE empresa_id IS NULL;

-- Step 3: Enforce NOT NULL constraints
ALTER TABLE presupuestos ALTER COLUMN empresa_id SET NOT NULL;
ALTER TABLE pedidos ALTER COLUMN empresa_id SET NOT NULL;
ALTER TABLE albaranes ALTER COLUMN empresa_id SET NOT NULL;
ALTER TABLE facturas ALTER COLUMN empresa_id SET NOT NULL;

-- Step 4: Add foreign key constraints
ALTER TABLE presupuestos ADD CONSTRAINT fk_presupuestos_empresa FOREIGN KEY (empresa_id) REFERENCES empresas(id) ON DELETE CASCADE;
ALTER TABLE pedidos ADD CONSTRAINT fk_pedido_empresa FOREIGN KEY (empresa_id) REFERENCES empresas(id) ON DELETE CASCADE;
ALTER TABLE albaranes ADD CONSTRAINT fk_albaranes_empresa FOREIGN KEY (empresa_id) REFERENCES empresas(id) ON DELETE CASCADE;
ALTER TABLE facturas ADD CONSTRAINT fk_facturas_empresa FOREIGN KEY (empresa_id) REFERENCES empresas(id) ON DELETE CASCADE;

-- Step 5: Add almacen_id to albaranes
ALTER TABLE albaranes ADD COLUMN almacen_id UUID;
UPDATE albaranes SET almacen_id = '00000000-0000-4000-a000-000000000002' WHERE almacen_id IS NULL;
ALTER TABLE albaranes ALTER COLUMN almacen_id SET NOT NULL;
ALTER TABLE albaranes ADD CONSTRAINT fk_albaranes_almacen FOREIGN KEY (almacen_id) REFERENCES almacenes(id) ON DELETE RESTRICT;

-- Step 6: Create lines tables
CREATE TABLE presupuesto_lineas (
    id UUID PRIMARY KEY,
    presupuesto_id UUID NOT NULL REFERENCES presupuestos(id) ON DELETE CASCADE,
    producto_id UUID NOT NULL REFERENCES productos(id) ON DELETE RESTRICT,
    cantidad NUMERIC(12,4) NOT NULL,
    precio_unitario NUMERIC(12,2) NOT NULL,
    coste_unitario NUMERIC(12,2) NOT NULL DEFAULT 0.00
);

CREATE TABLE pedido_lineas (
    id UUID PRIMARY KEY,
    pedido_id UUID NOT NULL REFERENCES pedidos(id) ON DELETE CASCADE,
    producto_id UUID NOT NULL REFERENCES productos(id) ON DELETE RESTRICT,
    cantidad NUMERIC(12,4) NOT NULL,
    precio_unitario NUMERIC(12,2) NOT NULL
);

CREATE TABLE albaran_lineas (
    id UUID PRIMARY KEY,
    albaran_id UUID NOT NULL REFERENCES albaranes(id) ON DELETE CASCADE,
    producto_id UUID NOT NULL REFERENCES productos(id) ON DELETE RESTRICT,
    cantidad NUMERIC(12,4) NOT NULL,
    precio_unitario NUMERIC(12,2) NOT NULL
);

CREATE TABLE factura_lineas (
    id UUID PRIMARY KEY,
    factura_id UUID NOT NULL REFERENCES facturas(id) ON DELETE CASCADE,
    producto_id UUID NOT NULL REFERENCES productos(id) ON DELETE RESTRICT,
    cantidad NUMERIC(12,4) NOT NULL,
    precio_unitario NUMERIC(12,2) NOT NULL
);

-- Indexes for performance & isolation
CREATE INDEX idx_presupuestos_empresa ON presupuestos(empresa_id);
CREATE INDEX idx_pedido_empresa ON pedidos(empresa_id);
CREATE INDEX idx_albaranes_empresa ON albaranes(empresa_id);
CREATE INDEX idx_facturas_empresa ON facturas(empresa_id);
CREATE INDEX idx_albaranes_almacen ON albaranes(almacen_id);
