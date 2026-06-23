# tpv-background-sync Specification

## Purpose
Coordinate background synchronization of offline records to the central backend.

## Requirements

### Requirement: REQ-SYN-01: Background Sync Worker
The core Rust loop MUST automatically sync pending local SQLite sales and box closures to Go backend endpoints exactly-once using UUID idempotency keys on connection restore.

#### Scenario: Sync offline sales and closure upon reconnection
- GIVEN pending local SQLite sales and closures with UUIDs and no connection
- WHEN connection to the backend is restored
- THEN the core Rust loop MUST automatically transmit pending records to Go backend endpoints
- AND the client MUST supply UUIDs as idempotency keys to guarantee exactly-once processing

### Requirement: REQ-SYN-02: Safe Local Database Cleanup
The sync worker MUST safely delete local SQLite records of sales and box closures ONLY after receiving a successful (2xx) HTTP confirmation from the Go backend.

#### Scenario: Delete local records after successful sync
- GIVEN synced local SQLite records
- WHEN the Go backend returns a successful 2xx HTTP response
- THEN the Rust worker MUST delete only those successfully confirmed records from local SQLite
