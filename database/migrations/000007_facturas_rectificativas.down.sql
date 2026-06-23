-- Down migration: Facturas Rectificativas

-- 1. Restore stock_ledger_movements CHECK constraint without RETURN
ALTER TABLE stock_ledger_movements DROP CONSTRAINT IF EXISTS stock_ledger_movements_movement_type_check;
ALTER TABLE stock_ledger_movements ADD CONSTRAINT stock_ledger_movements_movement_type_check
    CHECK (movement_type IN ('RECEIPT', 'WITHDRAWAL', 'SYNC_ADJUSTMENT', 'TRANSFER'));

-- 2. Drop indexes
DROP INDEX IF EXISTS idx_fr_lineas_fr;
DROP INDEX IF EXISTS idx_fr_created;
DROP INDEX IF EXISTS idx_fr_empresa;
DROP INDEX IF EXISTS idx_fr_facturas;

-- 3. Remove rectified_total from facturas
ALTER TABLE facturas DROP COLUMN IF EXISTS rectified_total;

-- 4. Drop tables
DROP TABLE IF EXISTS factura_rectificativa_lineas;
DROP TABLE IF EXISTS facturas_rectificativas;
