# Delta for tpv-background-sync

## MODIFIED Requirements

### Requirement: REQ-SYN-01: Background Sync Worker
El bucle automático de sincronización MUST transmitir tanto las facturas con su metadato de encadenamiento ('firma_registro', 'hash_anterior', y 'datos_encadenamiento') como la tabla 'registro_sucesos' hacia el backend central en Go.
(Previously: The core Rust loop MUST automatically sync pending local SQLite sales and box closures to Go backend.)

#### Scenario: Sincronización de ventas encadenadas y eventos de trazabilidad
- GIVEN ventas offline con metadatos de encadenamiento y eventos en 'registro_sucesos' pendientes de sincronización
- WHEN la conexión de red con el backend central se restablece
- THEN el bucle automático en Rust MUST transmitir las ventas y la tabla 'registro_sucesos' al backend en Go
- AND el cliente MUST suministrar los UUIDs como claves de idempotencia para garantizar un envío exactly-once
