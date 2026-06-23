# Delta for tpv-ui-facelift

## MODIFIED Requirements

### Requirement: REQ-CLI-02: Connection Status Warning
The desktop client MUST display a visual warning when connection to the backend is lost, but MUST NOT restrict or block sales/billing. The warning banner MUST use dynamic color styling and smooth sliding animations matching the active visual theme.

#### Scenario: Connection loss shows warning banner
- GIVEN the desktop client is online
- WHEN connection to the backend is lost
- THEN a visual connection warning banner MUST be displayed
- AND the banner MUST use dynamic color styling and smooth sliding animations matching the active visual theme
- AND the cashier MUST still be permitted to perform sales and billing

#### Scenario: Connection restore hides warning banner
- GIVEN the desktop client is offline with a warning banner visible
- WHEN connection to the backend is restored
- THEN the visual warning banner MUST be hidden

## ADDED Requirements

### Requirement: REQ-CLI-04: Dynamic Theme Toggling
The desktop client MUST support a dynamic dark and light mode theme, allowing the operator to toggle between them at runtime.

#### Scenario: Operator toggles theme from light to dark mode at runtime
- GIVEN the desktop client is running in light mode
- WHEN the operator triggers the theme toggle action
- THEN the client interface MUST immediately transition to dark mode without requiring a restart
- AND all UI components MUST update their styles to match the dark theme

#### Scenario: Operator toggles theme from dark to light mode at runtime
- GIVEN the desktop client is running in dark mode
- WHEN the operator triggers the theme toggle action
- THEN the client interface MUST immediately transition to light mode without requiring a restart
- AND all UI components MUST update their styles to match the light theme
