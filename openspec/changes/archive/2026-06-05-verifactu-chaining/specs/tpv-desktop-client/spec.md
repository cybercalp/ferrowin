# Delta for tpv-desktop-client

## ADDED Requirements

### Requirement: REQ-CLI-03: Preservación de firma anterior
El sistema SQLite MUST almacenar y preservar permanentemente la firma del último registro facturado en la tabla 'ultimo_registro_encadenado' para poder encadenar facturas futuras en modo offline.

#### Scenario: La limpieza de base de datos tras sincronización borra la venta pero preserva el hash del último registro encadenado
- GIVEN registros sincronizados con éxito en el servidor
- WHEN el worker de sincronización elimina las ventas locales para liberar espacio
- THEN el sistema MUST conservar intacto el registro correspondiente en la tabla 'ultimo_registro_encadenado'

## MODIFIED Requirements

### Requirement: REQ-CLI-01: Offline Storage
El sistema local SQLite MUST guardar las ventas offline con las columnas 'firma_registro', 'hash_anterior' y 'datos_encadenamiento'.
(Previously: The desktop client MUST record sales, sale items, and box closures (arqueo) to local SQLite when offline.)

#### Scenario: Venta offline guardada localmente con encadenamiento de firmas
- GIVEN el cliente de escritorio sin conexión de red
- WHEN se completa una venta
- THEN la venta MUST guardarse en SQLite local incluyendo 'firma_registro', 'hash_anterior' y 'datos_encadenamiento'

#### Scenario: Offline box closure recorded locally
- GIVEN the desktop client is offline
- WHEN a box closure (arqueo) is requested
- THEN the closure details MUST be saved to the local SQLite database
