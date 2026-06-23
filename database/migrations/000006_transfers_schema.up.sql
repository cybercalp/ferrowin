-- Migration for Warehouse Transfers (Traspasos de Almacén)

-- 1. Create transfer header table
CREATE TABLE traspasos_almacen (
    id UUID PRIMARY KEY,
    empresa_id UUID NOT NULL REFERENCES empresas(id),
    origen_id UUID NOT NULL REFERENCES almacenes(id),
    destino_id UUID NOT NULL REFERENCES almacenes(id),
    estado VARCHAR(20) NOT NULL CHECK (estado IN ('Borrador', 'Procesado', 'Cancelado')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    CONSTRAINT chk_diff_almacenes CHECK (origen_id <> destino_id)
);

-- 2. Create transfer lines table (ON DELETE CASCADE)
CREATE TABLE traspaso_almacen_lineas (
    id UUID PRIMARY KEY,
    traspaso_almacen_id UUID NOT NULL REFERENCES traspasos_almacen(id) ON DELETE CASCADE,
    producto_id UUID NOT NULL REFERENCES productos(id),
    cantidad NUMERIC(12,4) NOT NULL CHECK (cantidad > 0)
);

-- 3. Performance indexes
CREATE INDEX idx_traspasos_almacen_empresa ON traspasos_almacen(empresa_id);
CREATE INDEX idx_traspasos_almacen_origen ON traspasos_almacen(origen_id);
CREATE INDEX idx_traspasos_almacen_destino ON traspasos_almacen(destino_id);
CREATE INDEX idx_traspasos_almacen_estado ON traspasos_almacen(estado);
CREATE INDEX idx_traspasos_almacen_created ON traspasos_almacen(created_at);
CREATE INDEX idx_traspaso_almacen_lineas_transfer ON traspaso_almacen_lineas(traspaso_almacen_id);
CREATE INDEX idx_traspaso_almacen_lineas_producto ON traspaso_almacen_lineas(producto_id);

-- 4. Extend stock_ledger_movements CHECK constraint for TRANSFER type
ALTER TABLE stock_ledger_movements DROP CONSTRAINT IF EXISTS stock_ledger_movements_movement_type_check;
ALTER TABLE stock_ledger_movements ADD CONSTRAINT stock_ledger_movements_movement_type_check
    CHECK (movement_type IN ('RECEIPT', 'WITHDRAWAL', 'SYNC_ADJUSTMENT', 'TRANSFER'));
