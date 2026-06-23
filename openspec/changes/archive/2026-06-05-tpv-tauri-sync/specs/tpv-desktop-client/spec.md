# tpv-desktop-client Specification

## Purpose
Manage offline local storage and status warning indications on the TPV desktop client.

## Requirements

### Requirement: REQ-CLI-01: Offline Storage
The desktop client MUST record sales, sale items, and box closures (arqueo) to local SQLite when offline.

#### Scenario: Offline sale recorded locally
- GIVEN the desktop client is offline
- WHEN a sale is completed
- THEN the sale and its items MUST be saved to the local SQLite database

#### Scenario: Offline box closure recorded locally
- GIVEN the desktop client is offline
- WHEN a box closure (arqueo) is requested
- THEN the closure details MUST be saved to the local SQLite database

### Requirement: REQ-CLI-02: Connection Status Warning
The desktop client MUST display a visual warning when connection to the backend is lost, but MUST NOT restrict or block sales/billing.

#### Scenario: Connection loss shows warning banner
- GIVEN the desktop client is online
- WHEN connection to the backend is lost
- THEN a visual connection warning banner MUST be displayed
- AND the cashier MUST still be permitted to perform sales and billing

#### Scenario: Connection restore hides warning banner
- GIVEN the desktop client is offline with a warning banner visible
- WHEN connection to the backend is restored
- THEN the visual warning banner MUST be hidden
