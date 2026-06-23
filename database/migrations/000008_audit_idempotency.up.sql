-- Audit trail for document actions
CREATE TABLE registro_eventos (
    id UUID PRIMARY KEY,
    documento_tipo VARCHAR(30) NOT NULL,
    documento_id UUID NOT NULL,
    empresa_id UUID NOT NULL,
    accion VARCHAR(50) NOT NULL,
    usuario_id UUID,
    detalles TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_registro_eventos_doc ON registro_eventos(documento_tipo, documento_id);
CREATE INDEX idx_registro_eventos_empresa ON registro_eventos(empresa_id);

-- Idempotency keys for duplicate request prevention
CREATE TABLE idempotency_keys (
    clave VARCHAR(255) PRIMARY KEY,
    respuesta TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
