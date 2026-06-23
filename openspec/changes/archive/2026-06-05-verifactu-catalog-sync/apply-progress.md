# Apply Progress: verifactu-catalog-sync

## Phase 3 Implementation Progress

**Change**: verifactu-catalog-sync  
**Mode**: Standard  

### Completed Tasks
- [x] **3.1 Actualizar `tpv-client/src-tauri/src/db.rs` con helpers CRUD**:
  - Defined the domain model Rust structs: `TipoIVA`, `Familia`, `Producto`, `Cliente`, `RecentSaleDossier`, `ClientStatsDossier`, `PendingInvoiceDossier`, `ClientDossier`, `OfflineCobro`.
  - Implemented core database helper functions: `upsert_tipo_iva`, `upsert_familia`, `upsert_producto`, `upsert_cliente`, `deactivate_tipo_iva`, `deactivate_familia`, `deactivate_producto`, `deactivate_cliente`, `get_ultimo_sync_catalogo`, `set_ultimo_sync_catalogo`, `save_cliente_dossier`, `get_cliente_dossier`, `insert_offline_cobro`, `get_pending_cobros`, `delete_synced_cobro`.
  - Added new comprehensive unit tests validating CRUD functionality, client dossier persistence, and immediate balance reduction upon registering offline payments.
- [x] **3.2 Crear `tpv-client/src-tauri/src/catalog_sync.rs`**:
  - Implemented async download and sync of catalog delta (`sync_catalog_delta`) query parameterizing by the last sync timestamp.
  - Implemented async download of client dossiers (`download_client_dossiers`) matching backend payloads.
  - Implemented comprehensive mock server HTTP/JSON integration tests in `catalog_sync.rs` leveraging tokio TcpListener.
- [x] **3.3 Modificar `tpv-client/src-tauri/src/sync.rs`**:
  - Extended the background loop to process `sync_pending_payments` when online.
  - Added a generated unique UUID for `Idempotency-Key` header during post request to backend.
  - Updated `count_pending` to include offline payments in the sync count.
  - Added integration tests verifying background loop payment sync.
- [x] **3.4 Modificar `tpv-client/src-tauri/src/lib.rs`**:
  - Declared `catalog_sync` module.
  - Implemented and exposed Tauri commands: `sync_catalog`, `download_dossiers`, `registrar_cobro`, and `get_cliente_dossier`.
  - Registered commands in the Tauri builder handler.

## Phase 4 Implementation Progress

### Completed Tasks
- [x] **4.1 Añadir toggle de Configuración de Modo de Aplicación**:
  - Added a premium setting toggle in `tpv-client/src/App.tsx` allowing the POS operator to switch between "TPV Tienda (Online)" and "TPV Ambulante (Rutas/Offline)".
- [x] **4.2 Crear panel de Selección de Ruta de Clientes**:
  - Created `tpv-client/src/components/RouteSetup.tsx` with high-fidelity styling, search capabilities, and multi-selection checkboxes.
  - Upon selecting clients and clicking "Preparar Ruta", it invokes Tauri v2 commands `sync_catalog` and `download_dossiers` with selected client IDs, caching catalog and customer data locally.
- [x] **4.3 Implementar pantalla de Ficha de Cliente Offline**:
  - Created `tpv-client/src/components/ClientDossierView.tsx` which fetches client details, stats (credit limit, pending balance, available credit), recent sales history, and outstanding invoices.
- [x] **4.4 Crear modal de Cobro In Situ**:
  - Developed `tpv-client/src/components/ClientCollection.tsx` to handle generic payments on account (`A_CUENTA`) and specific invoice settlements (`DEUDA`).
  - Calls Tauri command `registrar_cobro` to immediately register the payment offline and update local stats/balances.
- [x] **4.5 Crear modal de Compartición de Comprobantes**:
  - Created `tpv-client/src/components/ShareDocumentModal.tsx` allowing operators to preview and copy receipt text to clipboard, or send it directly using WhatsApp Web / App deep linking.

### Files Changed
| File | Action | Description |
|------|--------|-------------|
| `tpv-client/src-tauri/src/db.rs` | Modified | Added `get_clientes` helper function to query active clients from SQLite. |
| `tpv-client/src-tauri/src/lib.rs` | Modified | Registered and exposed `get_clientes` Tauri command. |
| `tpv-client/src/App.tsx` | Modified | Registered and integrated all new Phase 4 React views and state logic. |
| `tpv-client/src/components/RouteSetup.tsx` | Created | Selection page for route clients with delta sync/dossier download triggers. |
| `tpv-client/src/components/ClientDossierView.tsx` | Created | Credit stats, recent sales, and pending invoice details view. |
| `tpv-client/src/components/ClientCollection.tsx` | Created | Payment collection modal interface for offline debt settlement. |
| `tpv-client/src/components/ShareDocumentModal.tsx` | Created | Copy and WhatsApp share modal for invoice receipts. |

### Deviations from Design
None — implementation matches the design specifications.

### Issues Found
None. Frontend TypeScript compiles successfully, and production build succeeded without errors.

### Workload / PR Boundary
- Mode: stacked-to-main
- Current work unit: Unit 4 (Frontend POS UI and Business Logic in React/TS)
- Boundary: Starts at mode toggle switcher, extends through route client selection, credit status dashboard, in-situ collections, and WhatsApp sharing.
