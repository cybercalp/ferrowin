-- PostgreSQL migration for catalog schema

CREATE TABLE tipos_iva (
    id UUID PRIMARY KEY,
    nombre VARCHAR(50) NOT NULL,
    porcentaje NUMERIC(5,2) NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW(),
    activo BOOLEAN DEFAULT TRUE
);

CREATE TABLE familias (
    id UUID PRIMARY KEY,
    nombre VARCHAR(100) NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW(),
    activo BOOLEAN DEFAULT TRUE
);

CREATE TABLE productos (
    id UUID PRIMARY KEY,
    codigo VARCHAR(50) UNIQUE NOT NULL,
    nombre VARCHAR(255) NOT NULL,
    precio_venta NUMERIC(12,2) NOT NULL,
    familia_id UUID REFERENCES familias(id) ON DELETE SET NULL,
    tipo_iva_id UUID REFERENCES tipos_iva(id) ON DELETE RESTRICT,
    updated_at TIMESTAMP DEFAULT NOW(),
    activo BOOLEAN DEFAULT TRUE
);

CREATE TABLE clientes (
    id UUID PRIMARY KEY,
    nombre VARCHAR(255) NOT NULL,
    nif VARCHAR(20) UNIQUE,
    email VARCHAR(100),
    updated_at TIMESTAMP DEFAULT NOW(),
    activo BOOLEAN DEFAULT TRUE
);

CREATE TABLE cobros_recibidos (
    id UUID PRIMARY KEY,
    cliente_id UUID REFERENCES clientes(id) ON DELETE CASCADE,
    factura_id UUID REFERENCES facturas(id) ON DELETE SET NULL,
    importe NUMERIC(12,2) NOT NULL,
    fecha TIMESTAMP DEFAULT NOW(),
    metodo_pago VARCHAR(20) NOT NULL,
    tipo_cobro VARCHAR(20) NOT NULL CHECK (tipo_cobro IN ('FACTURA', 'A_CUENTA')),
    idempotency_key UUID UNIQUE NOT NULL,
    synced_at TIMESTAMP DEFAULT NOW()
);
