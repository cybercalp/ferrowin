# Document Traceability Specification

## Purpose
Track progression and maintain audit links across the sales document lifecycle.

## Requirements
| ID | Requirement | Description |
|---|---|---|
| REQ-TRC-01 | Traceable Lifecycle | The system MUST track progression from Quote -> Order -> Delivery Note -> Invoice, locking converted parent states. |

### Scenarios

#### Scenario: Convert Quote to Order
- GIVEN a Quote in "Approved" state
- WHEN an Order is created referencing the Quote
- THEN the system MUST link them and set Quote state to "Converted"

#### Scenario: Reject conversion of converted document
- GIVEN a Quote in "Converted" state
- WHEN a user attempts to create another Order referencing it
- THEN the system MUST reject the creation with a transition error
