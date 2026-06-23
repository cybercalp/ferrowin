# Exploration: tpv-tauri-sync

### Current State
The project is a Go-based backend using hexagonal architecture. It includes adapters for invoice handling, inventory stock ledger tracking, and offline sales synchronization (`POST /api/v1/sync/sales`). 
The Go sync endpoint uses an `Idempotency-Key` header to avoid duplicate syncs and performs a transaction that inserts invoices and records corresponding negative stock ledger movements. 
The local SQLite database has `offline_sales` (with columns `id`, `sync_status`, `idempotency_key`, `invoice_number`, `sequence_number`) and `offline_sale_items` tables already prepared for local point-of-sale operations.

### Affected Areas
No existing Go server files need modifications as the sync endpoint and database schema are already implemented. The new files will be isolated to the client application:
- `tpv-client/` — New subdirectory containing the Tauri desktop client (Rust + React/TypeScript).
- `tpv-client/src-tauri/` — Rust core process for Tauri, containing local SQLite configuration, query commands, and the background sync worker.
- `tpv-client/src/` — React frontend interface for the terminal.

### Approaches

#### 1. Repository Structure for Go Backend & Tauri Client
- **Approach A: Monorepo with dedicated `tpv-client/` subdirectory**
  - Place the complete client application under a `tpv-client/` folder in the root, isolating Vite, React, TS, and Tauri Rust code.
  - **Pros**: Clean separation. Prevents pollution of the root Go environment. Keeps backend and client source under version control together. Easy to package and manage dependencies.
  - **Cons**: Requires stepping into `tpv-client/` to run client commands, but this can be simplified with root-level scripts or Makefiles.
  - **Effort**: Low.

- **Approach B: Mixing Tauri configuration inside a Go directory (e.g. `cmd/tpv/`)**
  - Place client configuration files and Rust/TS sources directly within `cmd/tpv/`.
  - **Pros**: Matches traditional Go command structure.
  - **Cons**: Highly messy. Tauri relies heavily on a standard frontend project structure (`package.json`, `node_modules`, `Cargo.toml`). Mixing cargo workspace/node projects inside a Go cmd path leads to IDE tooling conflicts and build pipeline complexity.
  - **Effort**: High.

#### 2. Sync Worker Execution Runtime
- **Approach A: Background Task in Rust Core (Tauri Main Process)**
  - Run a background thread or tokio task in Rust that periodically queries SQLite, constructs the sync request, calls the Go API via `reqwest`, and updates database records.
  - **Pros**: 
    - **Reliability**: Runs independently of the WebView frontend. If the UI freezes, reloads, or crashes, the sync worker continues running.
    - **Performance**: Direct SQLite access using native Rust drivers avoids serializing large transaction logs across the IPC boundary.
    - **Security**: Hardened network requests; endpoints and keys are compiled into the binary.
  - **Cons**: Requires writing sync and HTTP logic in Rust.
  - **Effort**: Medium.

- **Approach B: TypeScript Background Service in WebView Frontend**
  - Run a JS timer (`setInterval` or Web Worker) inside the React app that calls a Tauri command to fetch unsynced sales, then uses standard browser `fetch` to send them to the backend.
  - **Pros**: Easier to write and maintain for React developers; shares TypeScript interfaces.
  - **Cons**: 
    - **Unreliable**: Timers can drift or halt when the browser tab/view goes idle, reloads, or crashes.
    - **IPC Overhead**: Significant performance cost due to JSON serialization and IPC marshaling (Rust SQLite -> WebView -> JSON serialization -> API).
  - **Effort**: Low.

#### 3. Sync Worker Processing Logic
- **Approach A: Single-Sale Sync (At-Least-Once Delivery)**
  - Sync each pending sale individually using the sale's unique `idempotency_key` as the `Idempotency-Key` header.
  - **Pros**: Simple atomic logic. If one sale has a validation error, it does not block other sales from syncing.
  - **Cons**: Heavy network overhead due to separate HTTP requests.
  - **Effort**: Low.

- **Approach B: Batch Sync with Deterministic Hashed Idempotency Key**
  - Group pending sales (e.g. up to 50 sales) in a single request. Generate a deterministic idempotency key by hashing the sorted IDs of the sales in that batch.
  - **Pros**: Highly efficient. Minimizes network roundtrips.
  - **Cons**: If one sale in the batch is corrupted, the entire batch fails. Requires complex logic to extract and isolate failing records.
  - **Effort**: Medium.

### Recommendation
1. **Repository Structure**: Use **Approach A** (Dedicated `tpv-client/` directory). This is the industry standard for Tauri projects and avoids polluting the root Go workspace.
2. **Sync Worker Runtime**: Use **Approach A** (Rust Core Background Task). POS systems require extreme robustness; sync logic must not depend on the visual UI thread's stability.
3. **Sync Worker Logic**: Start with **Approach A** (Single-Sale Sync) for the initial MVP to establish stable integration with the Go backend's single-key idempotency scheme, then optimize to **Approach B** (Batch Sync) only if synchronization throughput becomes a bottleneck.

### Risks
- **SQLite Concurrency Lock**: If the Rust background sync worker writes/updates sync status in SQLite while the frontend React app is inserting a new sale, SQLite can throw a `database is locked` error (SQLITE_BUSY). We must configure WAL mode (`PRAGMA journal_mode = WAL;`) and set a busy timeout.
- **Go Idempotency Expiry**: If the Go backend deletes tracked idempotency keys after a time window, the client could theoretically re-send an old sale and cause duplication if the key is gone. The server should retain keys or have long retention.
- **Network Failures**: Network latency might cause the sync worker to timeout before receiving confirmation from Go. The client must retry the exact same request with the exact same idempotency key to prevent duplication.

### Ready for Proposal
Yes. The requirements and technical directions are clear. The orchestrator should present these architecture options and recommendation to the user to obtain approval for the design phase.
