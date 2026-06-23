# User Auth RBAC Specification

## Purpose
Define the RBAC security model supporting Users, Groups, Role Sets, and Roles.

## Requirements
| ID | Requirement | Description |
|---|---|---|
| REQ-SEC-01 | RBAC Hierarchy | The system MUST validate permissions using: User belongs to Groups; Groups map to Role Sets; Role Sets aggregate Roles. |

### Scenarios

#### Scenario: Authorize valid user permission
- GIVEN a User in a Group with Role Set containing "read-audit" Role
- WHEN the User requests the audit log
- THEN the system SHALL allow access

#### Scenario: Deny user lacking permission
- GIVEN a User in a Group with Role Set lacking "delete-user" Role
- WHEN the User requests user deletion
- THEN the system MUST return access denied
