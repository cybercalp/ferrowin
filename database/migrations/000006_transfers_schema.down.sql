-- Down migration: Warehouse Transfers

ALTER TABLE stock_ledger_movements DROP CONSTRAINT IF EXISTS stock_ledger_movements_movement_type_check;
ALTER TABLE stock_ledger_movements ADD CONSTRAINT stock_ledger_movements_movement_type_check
    CHECK (movement_type IN ('RECEIPT', 'WITHDRAWAL', 'SYNC_ADJUSTMENT'));

DROP TABLE IF EXISTS traspaso_almacen_lineas;
DROP TABLE IF EXISTS traspasos_almacen;
