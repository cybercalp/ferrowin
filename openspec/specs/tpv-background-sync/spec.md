# tpv-background-sync Specification

## Purpose
Coordinate background synchronization of offline records to the central backend.

## Requirements

### Requirement: REQ-SYN-01: Background Sync Worker
El bucle automático de sincronización MUST transmitir tanto las facturas con su metadato de encadenamiento ('firma_registro', 'hash_anterior', y 'datos_encadenamiento') como la tabla 'registro_sucesos' hacia el backend central en Go.

#### Scenario: Sincronización de ventas encadenadas y eventos de trazabilidad
- GIVEN ventas offline con metadatos de encadenamiento y eventos en 'registro_sucesos' pendientes de sincronización
- WHEN la conexión de red con el backend central se restablece
- THEN el bucle automático en Rust MUST transmitir las ventas y la tabla 'registro_sucesos' al backend en Go
- AND el cliente MUST suministrar los UUIDs como claves de idempotencia para garantizar un envío exactly-once

### Requirement: REQ-SYN-02: Safe Local Database Cleanup
The sync worker MUST safely delete local SQLite records of sales and box closures ONLY after receiving a successful (2xx) HTTP confirmation from the Go backend.

#### Scenario: Delete local records after successful sync
- GIVEN synced local SQLite records
- WHEN the Go backend returns a successful 2xx HTTP response
- THEN the Rust worker MUST delete only those successfully confirmed records from local SQLite
