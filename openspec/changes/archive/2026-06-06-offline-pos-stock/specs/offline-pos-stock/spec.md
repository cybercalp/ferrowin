# Delta for offline-pos-stock

## MODIFIED Requirements

### Requirement: REQ-OFF-02: Eventual Stock Queries
When online, the TPV MUST perform real-time central stock queries. When offline, the TPV MUST fall back to displaying the last known local stock value cached in SQLite. When sales are saved offline, local SQLite cached stock MUST be decremented immediately upon saving to prevent double-selling before synchronization.

#### Scenario: Query stock while online fetches live central data
- GIVEN the TPV client is online
- WHEN a stock query is initiated for an item
- THEN the system MUST retrieve and display the real-time stock value from the central Go backend

#### Scenario: Query stock while offline fetches local cached data
- GIVEN the TPV client is offline
- WHEN a stock query is initiated for an item
- THEN the system MUST display the last known stock value cached in local SQLite

#### Scenario: Decrement local cached stock on offline sale
- GIVEN the TPV client is offline
- AND local SQLite cached stock for item "I1" is 10 units
- WHEN a sale of 2 units of "I1" is saved offline
- THEN the local SQLite cached stock for "I1" MUST be immediately decremented to 8 units
