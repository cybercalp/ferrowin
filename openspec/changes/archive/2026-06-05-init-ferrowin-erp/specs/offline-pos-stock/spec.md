# Offline POS Stock Specification

## Purpose
Handle sales stock adjustments for reconnected terminals and reconcile balances.

## Requirements
| ID | Requirement | Description |
|---|---|---|
| REQ-OFF-01 | Sync & FIFO Reconciliation | The system MUST accept offline POS sales, allowing negative central stock. The system SHALL run FIFO reconciliation on sync. |

### Scenarios

#### Scenario: Reconnect and sync offline sale
- GIVEN central database with item "I1" stock at 0
- WHEN terminal syncs an offline sale of 1 unit of "I1"
- THEN the ledger MUST register the transaction and central stock drops to -1

#### Scenario: FIFO reconciliation on stock receipt
- GIVEN central stock of "I1" is at -2 units from offline sales
- WHEN a stock receipt of +10 units is registered
- THEN the system MUST clear the -2 units and update available stock to 8
