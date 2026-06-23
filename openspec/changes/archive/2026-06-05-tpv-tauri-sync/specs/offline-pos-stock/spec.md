# Delta for offline-pos-stock

## ADDED Requirements

### Requirement: REQ-OFF-02: Eventual Stock Queries
When online, the TPV MUST perform real-time central stock queries. When offline, the TPV MUST fall back to displaying the last known local stock value cached in SQLite.

#### Scenario: Query stock while online fetches live central data
- GIVEN the TPV client is online
- WHEN a stock query is initiated for an item
- THEN the system MUST retrieve and display the real-time stock value from the central Go backend

#### Scenario: Query stock while offline fetches local cached data
- GIVEN the TPV client is offline
- WHEN a stock query is initiated for an item
- THEN the system MUST display the last known stock value cached in local SQLite
