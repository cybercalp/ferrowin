-- Migration for Facturas Rectificativas (credit notes)

-- 1. Create facturas_rectificativas table
CREATE TABLE facturas_rectificativas (
    id UUID PRIMARY KEY,
    factura_id UUID NOT NULL REFERENCES facturas(id) ON DELETE RESTRICT,
    empresa_id UUID NOT NULL REFERENCES empresas(id) ON DELETE CASCADE,
    terminal_id UUID REFERENCES terminals(id) ON DELETE RESTRICT,
    numero_fr VARCHAR(30) UNIQUE NOT NULL,
    numero_secuencia INT NOT NULL,
    total NUMERIC(12,2) NOT NULL,
    reason TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'Issued',
    created_at TIMESTAMP DEFAULT NOW()
);

-- 2. Create factura_rectificativa_lineas table
CREATE TABLE factura_rectificativa_lineas (
    id UUID PRIMARY KEY,
    factura_rectificativa_id UUID NOT NULL REFERENCES facturas_rectificativas(id) ON DELETE CASCADE,
    producto_id UUID NOT NULL REFERENCES productos(id) ON DELETE RESTRICT,
    cantidad NUMERIC(12,4) NOT NULL,
    precio_unitario NUMERIC(12,2) NOT NULL
);

-- 3. Add rectified_total to facturas (track partial rectifications)
ALTER TABLE facturas ADD COLUMN rectified_total NUMERIC(12,2) NOT NULL DEFAULT 0.00;

-- 4. Extend stock_ledger_movements CHECK constraint for RETURN type
ALTER TABLE stock_ledger_movements DROP CONSTRAINT IF EXISTS stock_ledger_movements_movement_type_check;
ALTER TABLE stock_ledger_movements ADD CONSTRAINT stock_ledger_movements_movement_type_check
    CHECK (movement_type IN ('RECEIPT', 'WITHDRAWAL', 'SYNC_ADJUSTMENT', 'TRANSFER', 'RETURN'));

-- 5. Performance indexes
CREATE INDEX idx_fr_facturas ON facturas_rectificativas(factura_id);
CREATE INDEX idx_fr_empresa ON facturas_rectificativas(empresa_id);
CREATE INDEX idx_fr_created ON facturas_rectificativas(created_at);
CREATE INDEX idx_fr_lineas_fr ON factura_rectificativa_lineas(factura_rectificativa_id);
