# verifactu-trazabilidad Specification

## Purpose
Garantizar la trazabilidad del sistema mediante el registro local y centralizado de sucesos relevantes.

## Requirements

### Requirement: REQ-TRA-01: Registro de Sucesos
El sistema MUST guardar un registro de auditoría local (tabla SQLite 'registro_sucesos') y central (Postgres) de sucesos como inicios de sesión, cambios de red, sincronizaciones y errores.

#### Scenario: Registro local de suceso cuando no hay conexión
- GIVEN que el cliente de escritorio no dispone de conexión de red
- WHEN ocurre un suceso registrable en el sistema
- THEN el sistema MUST insertar el registro en la tabla local 'registro_sucesos'

#### Scenario: Sincronización automática de sucesos al restaurarse la red
- GIVEN sucesos pendientes de envío en la tabla SQLite 'registro_sucesos'
- WHEN se detecta que la conexión de red con el servidor ha sido restaurada
- THEN el sistema MUST sincronizar automáticamente dichos registros con la base de datos central Postgres
