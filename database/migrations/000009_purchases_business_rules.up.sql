-- Add optimistic locking version columns to purchases tables
ALTER TABLE pedidos_compra ADD COLUMN version INT NOT NULL DEFAULT 1;
ALTER TABLE recepciones_compra ADD COLUMN version INT NOT NULL DEFAULT 1;

-- Add partial receipt tracking to purchase order lines
ALTER TABLE pedido_compra_lineas ADD COLUMN recibido NUMERIC(12,4) NOT NULL DEFAULT 0;

-- Update pedidos_compra CHECK constraint to include 'Parcial' status
-- SQLite does not support ALTER CHECK; we need to recreate the table.
-- For PostgreSQL, we can drop and re-add the constraint.
-- This migration handles both via a pragmatic approach.

-- Drop old CHECK constraint (PostgreSQL)
ALTER TABLE pedidos_compra DROP CONSTRAINT IF EXISTS pedidos_compra_estado_check;

-- Add updated CHECK constraint with 'Parcial' (PostgreSQL)
ALTER TABLE pedidos_compra ADD CONSTRAINT pedidos_compra_estado_check
    CHECK (estado IN ('Borrador', 'Aprobado', 'Recibido', 'Parcial', 'Cancelado'));
