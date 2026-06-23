## Exploration: ERP Ferrowin Architecture Design

### Current State
This is a greenfield project (`Ferrowin` ERP for hardware stores) with an empty workspace. No technical stack or database structure has been established yet.

### Affected Areas
- `openspec/changes/init-ferrowin-erp/exploration.md` — Technical architecture exploration document detailing domains, stack, and sync strategy.

### Approaches
We compare a Modular Monolith (Clean/Hexagonal Architecture) with a distributed microservices approach for the core backend, and evaluate different client options for the local offline POS.

#### Backend Architecture Options
1. **Modular Monolith (Go or NestJS + TypeScript)** — A single deployable unit with clearly separated modules per domain (Security, Products, Purchases, Sales, Treasury).
   - Pros: Simpler deployment, easier transaction management within central database, shared library types, low infrastructure overhead.
   - Cons: Scaling requires scaling the entire monolith (mitigated by using modular structures).
   - Effort: Low to Medium.

2. **Microservices (Spring Boot / NestJS / Go)** — Each domain deployed as an independent service with its own DB schema.
   - Pros: Independent deployment, technology diversity per domain, independent scaling.
   - Cons: High deployment complexity, network overhead, complex distributed transactions (Saga pattern), synchronization complexity.
   - Effort: High.

#### Offline POS Client Architecture Options
1. **Tauri App (Rust + Web Frontend)** — Desktop POS application that bundles SQLite natively.
   - Pros: Native hardware access (printers, barcode scanners), very low memory footprint, single binary, cross-platform.
   - Cons: Rust learning curve for native plugins (though frontend is TS/JS).
   - Effort: Medium.

2. **Electron App (Node.js + Chromium + Web Frontend)** — Traditional desktop wrapper.
   - Pros: Easy to write, standard Node.js APIs for SQLite.
   - Cons: Heavy memory footprint (~150MB+ idle), large binary size.
   - Effort: Low.

### Recommendation
1. **Backend**: A **Modular Monolith in Go or NestJS (TypeScript)** following **Hexagonal/Clean Architecture**. This keeps the code organized, testable, and ready to split into microservices if needed later, without the upfront operational complexity.
2. **Databases**:
   - **Central**: PostgreSQL (robust, handles JSONB, reliable ACID).
   - **Local POS**: SQLite (zero-config, embedded, reliable file-based DB).
3. **Frontend**: **Tauri (React/TS + Tailwind)** for the POS client to optimize performance on hardware-store terminal hardware (which are often low-spec PCs). A standard responsive web app (Next.js/React) for the central backoffice.
4. **Sync Agent**: Embedded native service in the Tauri client (implemented in Rust or Go) that polls a SQLite sync queue and pushes transactions to PostgreSQL with idempotency keys (UUIDs).

### Risks
- **Network Sync Conflicts**: Off-sync inventory updates can lead to out-of-stock situations. Requires a strict delta-based inventory sync.
- **Hardware Integration**: Directly communicating with old serial/USB ticket printers and scales in Windows environments can be brittle. Tauri native plugins will need thorough testing.
- **Data Integrity**: Local SQLite data could be corrupted by sudden power outages at retail terminals. SQLite WAL mode should be used to protect transactions.

### Ready for Proposal
Yes — The architecture direction is clear. The next phase will draft the functional/non-functional specs and design schema mapping.
