-- PostgreSQL migration for updating sales documents and adding lines

-- Step 1: Add empresa_id to primary sales document tables (quote, order, delivery_note, invoice)
ALTER TABLE quote ADD COLUMN empresa_id UUID;
ALTER TABLE "order" ADD COLUMN empresa_id UUID;
ALTER TABLE delivery_note ADD COLUMN empresa_id UUID;
ALTER TABLE invoice ADD COLUMN empresa_id UUID;

-- Step 2: Assign default company ID to any existing records
UPDATE quote SET empresa_id = '00000000-0000-4000-a000-000000000001' WHERE empresa_id IS NULL;
UPDATE "order" SET empresa_id = '00000000-0000-4000-a000-000000000001' WHERE empresa_id IS NULL;
UPDATE delivery_note SET empresa_id = '00000000-0000-4000-a000-000000000001' WHERE empresa_id IS NULL;
UPDATE invoice SET empresa_id = '00000000-0000-4000-a000-000000000001' WHERE empresa_id IS NULL;

-- Step 3: Enforce NOT NULL constraints
ALTER TABLE quote ALTER COLUMN empresa_id SET NOT NULL;
ALTER TABLE "order" ALTER COLUMN empresa_id SET NOT NULL;
ALTER TABLE delivery_note ALTER COLUMN empresa_id SET NOT NULL;
ALTER TABLE invoice ALTER COLUMN empresa_id SET NOT NULL;

-- Step 4: Add foreign key constraints
ALTER TABLE quote ADD CONSTRAINT fk_quote_empresa FOREIGN KEY (empresa_id) REFERENCES empresas(id) ON DELETE CASCADE;
ALTER TABLE "order" ADD CONSTRAINT fk_order_empresa FOREIGN KEY (empresa_id) REFERENCES empresas(id) ON DELETE CASCADE;
ALTER TABLE delivery_note ADD CONSTRAINT fk_delivery_note_empresa FOREIGN KEY (empresa_id) REFERENCES empresas(id) ON DELETE CASCADE;
ALTER TABLE invoice ADD CONSTRAINT fk_invoice_empresa FOREIGN KEY (empresa_id) REFERENCES empresas(id) ON DELETE CASCADE;

-- Step 5: Add warehouse_id to delivery_note
ALTER TABLE delivery_note ADD COLUMN warehouse_id UUID;
UPDATE delivery_note SET warehouse_id = '00000000-0000-4000-a000-000000000002' WHERE warehouse_id IS NULL;
ALTER TABLE delivery_note ALTER COLUMN warehouse_id SET NOT NULL;
ALTER TABLE delivery_note ADD CONSTRAINT fk_delivery_note_warehouse FOREIGN KEY (warehouse_id) REFERENCES warehouses(id) ON DELETE RESTRICT;

-- Step 6: Create lines tables
CREATE TABLE quote_lines (
    id UUID PRIMARY KEY,
    quote_id UUID NOT NULL REFERENCES quote(id) ON DELETE CASCADE,
    producto_id UUID NOT NULL REFERENCES productos(id) ON DELETE RESTRICT,
    cantidad NUMERIC(12,4) NOT NULL,
    precio_unitario NUMERIC(12,2) NOT NULL,
    coste_unitario NUMERIC(12,2) NOT NULL DEFAULT 0.00
);

CREATE TABLE order_lines (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES "order"(id) ON DELETE CASCADE,
    producto_id UUID NOT NULL REFERENCES productos(id) ON DELETE RESTRICT,
    cantidad NUMERIC(12,4) NOT NULL,
    precio_unitario NUMERIC(12,2) NOT NULL
);

CREATE TABLE delivery_note_lineas (
    id UUID PRIMARY KEY,
    delivery_note_id UUID NOT NULL REFERENCES delivery_note(id) ON DELETE CASCADE,
    producto_id UUID NOT NULL REFERENCES productos(id) ON DELETE RESTRICT,
    cantidad NUMERIC(12,4) NOT NULL,
    precio_unitario NUMERIC(12,2) NOT NULL
);

CREATE TABLE invoice_lineas (
    id UUID PRIMARY KEY,
    invoice_id UUID NOT NULL REFERENCES invoice(id) ON DELETE CASCADE,
    producto_id UUID NOT NULL REFERENCES productos(id) ON DELETE RESTRICT,
    cantidad NUMERIC(12,4) NOT NULL,
    precio_unitario NUMERIC(12,2) NOT NULL
);

-- Indexes for performance & isolation
CREATE INDEX idx_quote_empresa ON quote(empresa_id);
CREATE INDEX idx_order_empresa ON "order"(empresa_id);
CREATE INDEX idx_delivery_note_empresa ON delivery_note(empresa_id);
CREATE INDEX idx_invoice_empresa ON invoice(empresa_id);
CREATE INDEX idx_delivery_note_warehouse ON delivery_note(warehouse_id);
