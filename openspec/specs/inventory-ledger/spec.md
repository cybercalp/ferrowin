# Inventory Ledger Specification

## Purpose
Provide a centralized transaction ledger for all inventory adjustments.

## Requirements
| ID | Requirement | Description |
|---|---|---|
| REQ-INV-01 | Ledger Log & Stock Enforce | Every stock change MUST create a ledger record. The system SHALL block transactions causing negative stock. |

### Scenarios

#### Scenario: Stock receipt entry
- GIVEN warehouse "W1" with item "I1" stock at 10
- WHEN a stock entry of +5 is registered
- THEN a ledger record of +5 MUST be saved and available stock set to 15

#### Scenario: Insufficient stock rejection
- GIVEN warehouse "W1" with item "I1" stock at 2
- WHEN a stock withdrawal of 3 is requested
- THEN the system MUST block the transaction and return insufficient stock error
