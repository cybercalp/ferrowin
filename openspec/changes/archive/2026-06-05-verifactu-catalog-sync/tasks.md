# Tasks for verifactu-catalog-sync

## Phase 3: Tauri Client Sync and Database CRUD Code

- [x] 3.1 Actualizar `tpv-client/src-tauri/src/db.rs` con helpers CRUD.
- [x] 3.2 Crear `tpv-client/src-tauri/src/catalog_sync.rs` para gestionar el flujo de descarga del delta y dossier.
- [x] 3.3 Modificar `tpv-client/src-tauri/src/sync.rs` para incluir la sincronización de cobros offline (`sync_pending_payments`).
- [x] 3.4 Modificar `tpv-client/src-tauri/src/lib.rs` para registrar y exponer los comandos IPC.

## Phase 4: Frontend POS UI and Business Logic in React/TS

- [x] 4.1 Añadir toggle de Configuración de Modo de Aplicación (TPV Tienda vs TPV Ambulante) in 'tpv-client/src/App.tsx'.
- [x] 4.2 Crear panel de Selección de Ruta de Clientes ('tpv-client/src/components/RouteSetup.tsx') para TPV Ambulante.
- [x] 4.3 Implementar pantalla de Ficha de Cliente Offline ('tpv-client/src/components/ClientDossierView.tsx').
- [x] 4.4 Crear modal de Cobro In Situ ('tpv-client/src/components/ClientCollection.tsx').
- [x] 4.5 Crear modal de Compartición de Comprobantes ('tpv-client/src/components/ShareDocumentModal.tsx').

