# Proposal: tpv-ui-facelift

## Intent

Refactor the hardcoded inline styling in the TPV desktop client to support a dynamic theme switcher, premium typography, and a modernized visual facelift (glassmorphism), enhancing user experience and readability under different lighting conditions.

## Scope

### In Scope
- Define dark/light theme CSS variables in `App.css`.
- Implement a React `ThemeProvider` in `App.tsx`.
- Refactor application components to use CSS variables and glassmorphism.
- Add Google Fonts (Outfit/Inter) and blocking script to `index.html`.
- Add header theme toggler.

### Out of Scope
- Go backend alterations or animations via third-party libraries (only CSS transitions).

## Capabilities

### New Capabilities
- None

### Modified Capabilities
- tpv-desktop-client: Aesthetic facelift, dynamic dark/light mode toggle, premium Google Fonts, glassmorphic cards, smooth hover transitions, and visual warning banner styling.

## Approach

Define all light/dark colors as CSS variables in `App.css`. Add a lightweight React `ThemeProvider` context in `App.tsx` managing the HTML `data-theme` attribute and persisting selection in `localStorage`. Update inline styles in components to reference these variables. Inject Google Fonts and a script in `index.html` head to prevent flash of default theme on page load.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `tpv-client/index.html` | Modified | Add Google Fonts link (Outfit/Inter) and script to set initial theme preventing flash of default theme. |
| `tpv-client/src/App.css` | Modified | Define CSS variables for colors (light/dark themes under data-theme) and global styling rules. |
| `tpv-client/src/App.tsx` | Modified | Wrap application in dynamic ThemeProvider, provide header theme toggle button. |
| `tpv-client/src/components/*` | Modified | Refactor style properties in SyncWarningBanner, RouteSetup, ClientCollection, ClientDossierView, EntityManager, EntityDetailPanel, and ShareDocumentModal to use CSS variables. |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Theme Flash | Low | Inject a blocking script in `index.html` `<head>` to read preference and set `data-theme` early. |
| Contrast Issues | Medium | Run accessibility audits with high-contrast rules during development. |

## Rollback Plan

Revert the UI commits in git to restore inline styling. No database rollback is required.

## Dependencies

- None

## Success Criteria

- [ ] Web application displays dynamically styled layout with premium Google Fonts (Outfit, Inter) and glassmorphism.
- [ ] Theme toggler switched correctly between light and dark modes, persisting in localStorage.
- [ ] Refactored components look visually integrated with dynamic theme variables.
