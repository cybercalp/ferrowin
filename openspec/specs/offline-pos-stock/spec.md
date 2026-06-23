# Offline POS Stock Specification

## Purpose
Handle sales stock adjustments for reconnected terminals and reconcile balances.

## Requirements
| ID | Requirement | Description |
|---|---|---|
| REQ-OFF-01 | Sync & FIFO Reconciliation | The system MUST accept offline POS sales, allowing negative central stock. The system SHALL run FIFO reconciliation on sync. |
| REQ-OFF-02 | Eventual Stock Queries | When online, the TPV MUST perform real-time central stock queries. When offline, the TPV MUST fall back to displaying the last known local stock value cached in SQLite. When sales are saved offline, local SQLite cached stock MUST be decremented immediately upon saving to prevent double-selling before synchronization. |

### Scenarios

#### Scenario: Reconnect and sync offline sale
- GIVEN central database with item "I1" stock at 0
- WHEN terminal syncs an offline sale of 1 unit of "I1"
- THEN the ledger MUST register the transaction and central stock drops to -1

#### Scenario: FIFO reconciliation on stock receipt
- GIVEN central stock of "I1" is at -2 units from offline sales
- WHEN a stock receipt of +10 units is registered
- THEN the system MUST clear the -2 units and update available stock to 8

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
