# tpv-desktop-client Specification

## Purpose
Manage offline local storage and status warning indications on the TPV desktop client.

## Requirements

### Requirement: REQ-CLI-01: Offline Storage
El sistema local SQLite MUST guardar las ventas offline con las columnas 'firma_registro', 'hash_anterior' y 'datos_encadenamiento'.

#### Scenario: Venta offline guardada localmente con encadenamiento de firmas
- GIVEN el cliente de escritorio sin conexión de red
- WHEN se completa una venta
- THEN la venta MUST guardarse en SQLite local incluyendo 'firma_registro', 'hash_anterior' y 'datos_encadenamiento'

#### Scenario: Offline box closure recorded locally
- GIVEN the desktop client is offline
- WHEN a box closure (arqueo) is requested
- THEN the closure details MUST be saved to the local SQLite database

### Requirement: REQ-CLI-02: Connection Status Warning
The desktop client MUST display a visual warning when connection to the backend is lost, but MUST NOT restrict or block sales/billing. The warning banner MUST use dynamic color styling and smooth sliding animations matching the active visual theme.

#### Scenario: Connection loss shows warning banner
- GIVEN the desktop client is online
- WHEN connection to the backend is lost
- THEN a visual connection warning banner MUST be displayed
- AND the banner MUST use dynamic color styling and smooth sliding animations matching the active visual theme
- AND the cashier MUST still be permitted to perform sales and billing

#### Scenario: Connection restore hides warning banner
- GIVEN the desktop client is offline with a warning banner visible
- WHEN connection to the backend is restored
- THEN the visual warning banner MUST be hidden

### Requirement: REQ-CLI-03: Preservación de firma anterior
El sistema SQLite MUST almacenar y preservar permanentemente la firma del último registro facturado en la tabla 'ultimo_registro_encadenado' para poder encadenar facturas futuras en modo offline.

#### Scenario: La limpieza de base de datos tras sincronización borra la venta pero preserva el hash del último registro encadenado
- GIVEN registros sincronizados con éxito en el servidor
- WHEN el worker de sincronización elimina las ventas locales para liberar espacio
- THEN el sistema MUST conservar intacto el registro correspondiente en la tabla 'ultimo_registro_encadenado'

### Requirement: REQ-CLI-04: Dynamic Theme Toggling
The desktop client MUST support a dynamic dark and light mode theme, allowing the operator to toggle between them at runtime.

#### Scenario: Operator toggles theme from light to dark mode at runtime
- GIVEN the desktop client is running in light mode
- WHEN the operator triggers the theme toggle action
- THEN the client interface MUST immediately transition to dark mode without requiring a restart
- AND all UI components MUST update their styles to match the dark theme

#### Scenario: Operator toggles theme from dark to light mode at runtime
- GIVEN the desktop client is running in dark mode
- WHEN the operator triggers the theme toggle action
- THEN the client interface MUST immediately transition to light mode without requiring a restart
- AND all UI components MUST update their styles to match the light theme
