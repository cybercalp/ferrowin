-- Add foreign keys (empresa_id and almacen_id columns already in 000001)
ALTER TABLE presupuestos ADD CONSTRAINT fk_presupuestos_empresa FOREIGN KEY (empresa_id) REFERENCES empresas(id) ON DELETE CASCADE;
ALTER TABLE pedidos ADD CONSTRAINT fk_pedido_empresa FOREIGN KEY (empresa_id) REFERENCES empresas(id) ON DELETE CASCADE;
ALTER TABLE albaranes ADD CONSTRAINT fk_albaranes_empresa FOREIGN KEY (empresa_id) REFERENCES empresas(id) ON DELETE CASCADE;
ALTER TABLE facturas ADD CONSTRAINT fk_facturas_empresa FOREIGN KEY (empresa_id) REFERENCES empresas(id) ON DELETE CASCADE;
ALTER TABLE albaranes ADD CONSTRAINT fk_albaranes_almacen FOREIGN KEY (almacen_id) REFERENCES almacenes(id) ON DELETE RESTRICT;

-- Line tables
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

-- Indexes
CREATE INDEX idx_presupuestos_empresa ON presupuestos(empresa_id);
CREATE INDEX idx_pedido_empresa ON pedidos(empresa_id);
CREATE INDEX idx_albaranes_empresa ON albaranes(empresa_id);
CREATE INDEX idx_facturas_empresa ON facturas(empresa_id);
CREATE INDEX idx_albaranes_almacen ON albaranes(almacen_id);
