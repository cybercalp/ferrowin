# Billing Series Management Specification

## Purpose
Configure separate billing series and sequential invoicing counters per terminal.

## Requirements
| ID | Requirement | Description |
|---|---|---|
| REQ-BIL-01 | Isolation & Sequence | Each terminal MUST use a unique prefix series. The system SHALL generate sequential numbers per terminal. |

### Scenarios

#### Scenario: Sequence increment
- GIVEN terminal "T-01" configured with series "S1" at sequence 15
- WHEN a new invoice is generated on "T-01"
- THEN the invoice number MUST be "S1-16" and sequence updates to 16

#### Scenario: Prefix isolation
- GIVEN terminal "T-01" configured with "S1" and "T-02" with "S2"
- WHEN both terminals create invoices concurrently
- THEN the system SHALL assign "S1" prefix to T-01 and "S2" to T-02
