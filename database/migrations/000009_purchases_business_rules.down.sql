-- Reverse purchases business rules migration
ALTER TABLE pedidos_compra DROP CONSTRAINT IF EXISTS pedidos_compra_estado_check;
ALTER TABLE pedidos_compra ADD CONSTRAINT pedidos_compra_estado_check
    CHECK (estado IN ('Borrador', 'Aprobado', 'Recibido', 'Cancelado'));

ALTER TABLE pedido_compra_lineas DROP COLUMN IF EXISTS recibido;
ALTER TABLE recepciones_compra DROP COLUMN IF EXISTS version;
ALTER TABLE pedidos_compra DROP COLUMN IF EXISTS version;
