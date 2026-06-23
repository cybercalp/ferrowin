-- PostgreSQL migration for Unifying Entities (Clientes & Proveedores)

-- 1. Create new unified tables
CREATE TABLE entidades (
    id UUID PRIMARY KEY,
    empresa_id UUID NOT NULL REFERENCES empresas(id) ON DELETE CASCADE,
    razon_social VARCHAR(150) NOT NULL,
    nif VARCHAR(20) NOT NULL,
    email VARCHAR(100),
    telefono VARCHAR(20),
    activo BOOLEAN DEFAULT TRUE,
    roles VARCHAR(150) NOT NULL,
    codigo_interno VARCHAR(50),
    codigos_alternativos TEXT,
    configuracion_contable TEXT,
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(empresa_id, nif)
);

CREATE TABLE entidad_direcciones (
    id UUID PRIMARY KEY,
    entidad_id UUID NOT NULL REFERENCES entidades(id) ON DELETE CASCADE,
    tipo_direccion VARCHAR(50) NOT NULL,
    calle VARCHAR(255) NOT NULL,
    ciudad VARCHAR(100) NOT NULL,
    provincia VARCHAR(100) NOT NULL,
    codigo_postal VARCHAR(20) NOT NULL,
    pais VARCHAR(100) NOT NULL
);

CREATE TABLE entidad_contactos (
    id UUID PRIMARY KEY,
    entidad_id UUID NOT NULL REFERENCES entidades(id) ON DELETE CASCADE,
    nombre VARCHAR(150) NOT NULL,
    puesto VARCHAR(100),
    email VARCHAR(100),
    telefono VARCHAR(20)
);

CREATE TABLE entidad_notas (
    id UUID PRIMARY KEY,
    entidad_id UUID NOT NULL REFERENCES entidades(id) ON DELETE CASCADE,
    nota TEXT NOT NULL,
    creado_en TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. Populate entities with clients (role: CLIENTE)
INSERT INTO entidades (id, empresa_id, razon_social, nif, email, telefono, activo, roles)
SELECT id, '00000000-0000-4000-a000-000000000001', nombre, COALESCE(nif, 'TEMP-CLI-' || id::text), email, NULL, activo, 'CLIENTE'
FROM clientes;

-- 3. Populate entities with suppliers (role: PROVEEDOR)
-- Create a temporary mapping of (old_supplier_uuid -> unified_entity_uuid) for unifications
CREATE TEMP TABLE supplier_mapping (
    old_id UUID,
    new_id UUID
);

-- Insert mappings for suppliers that share NIF with an existing client
INSERT INTO supplier_mapping (old_id, new_id)
SELECT p.id, e.id
FROM proveedores p
JOIN entidades e ON e.nif = p.cif AND e.empresa_id = p.empresa_id;

-- Insert suppliers that DO NOT share NIF with any existing client
INSERT INTO entidades (id, empresa_id, razon_social, nif, email, telefono, activo, roles)
SELECT p.id, p.empresa_id, p.razon_social, p.cif, p.email, p.telefono, p.activo, 'PROVEEDOR'
FROM proveedores p
WHERE p.id NOT IN (SELECT old_id FROM supplier_mapping);

-- Update roles for unified entities (clients that are also suppliers)
UPDATE entidades
SET roles = 'CLIENTE,PROVEEDOR'
WHERE id IN (SELECT new_id FROM supplier_mapping);

-- Migrate supplier directions to address table
INSERT INTO entidad_direcciones (id, entidad_id, tipo_direccion, calle, ciudad, provincia, codigo_postal, pais)
SELECT gen_random_uuid(), COALESCE(m.new_id, p.id), 'Fiscal', p.direccion, 'Ciudad', 'Provincia', 'CP', 'Pais'
FROM proveedores p
LEFT JOIN supplier_mapping m ON m.old_id = p.id
WHERE p.direccion IS NOT NULL AND p.direccion <> '';

-- 4. Update foreign keys referencing old supplier UUIDs to the new unified UUIDs
UPDATE pedidos_compra pc
SET proveedor_id = m.new_id
FROM supplier_mapping m
WHERE pc.proveedor_id = m.old_id;

UPDATE recepciones_compra rc
SET proveedor_id = m.new_id
FROM supplier_mapping m
WHERE rc.proveedor_id = m.old_id;

-- 5. Drop old foreign keys and add new ones referencing entidades
ALTER TABLE cobros_recibidos DROP CONSTRAINT IF EXISTS cobros_recibidos_cliente_id_fkey;
ALTER TABLE cobros_recibidos ADD CONSTRAINT fk_cobros_recibidos_cliente FOREIGN KEY (cliente_id) REFERENCES entidades(id) ON DELETE CASCADE;

ALTER TABLE quote DROP CONSTRAINT IF EXISTS quote_client_id_fkey;
ALTER TABLE quote ADD CONSTRAINT fk_quote_client FOREIGN KEY (client_id) REFERENCES entidades(id) ON DELETE RESTRICT;

ALTER TABLE pedidos_compra DROP CONSTRAINT IF EXISTS pedidos_compra_proveedor_id_fkey;
ALTER TABLE pedidos_compra ADD CONSTRAINT fk_pedidos_compra_proveedor FOREIGN KEY (proveedor_id) REFERENCES entidades(id) ON DELETE RESTRICT;

ALTER TABLE recepciones_compra DROP CONSTRAINT IF EXISTS recepciones_compra_proveedor_id_fkey;
ALTER TABLE recepciones_compra ADD CONSTRAINT fk_recepciones_compra_proveedor FOREIGN KEY (proveedor_id) REFERENCES entidades(id) ON DELETE RESTRICT;

-- 6. Cleanup
DROP TABLE supplier_mapping;
DROP TABLE IF EXISTS clientes;
DROP TABLE IF EXISTS proveedores;
