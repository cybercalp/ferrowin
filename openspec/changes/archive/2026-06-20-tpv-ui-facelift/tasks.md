# Tasks: TPV UI Facelift

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 300–400 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | auto-chain (stacked-to-main) |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | All 10 files — CSS foundation + hex→var() refactor | Single PR | All modifications, no new files |

## Phase 1: Theme Foundation

- [x] 1.1 `tpv-client/index.html` — Add Google Fonts `<link>`, blocking theme-init `<script>` in `<head>`, update title
- [x] 1.2 `tpv-client/src/App.css` — Rewrite with `:root` + `[data-theme="dark"]` CSS variable definitions, glassmorphism templates, keyframes
- [x] 1.3 `tpv-client/src/App.tsx` — Create `ThemeProvider` wrapping `ThemeContextValue`, add toggle button in header, remove all inline style blocks, replace with `var()` refs

## Phase 2: Component Refactoring (hex → var())

- [x] 2.1 `tpv-client/src/components/SyncWarningBanner.tsx` — Replace amber/green hex colors with `var(--status-warning-*)`, `var(--status-syncing-*)`
- [x] 2.2 `tpv-client/src/components/RouteSetup.tsx` — Replace hex: panel bg, table, buttons, status box
- [x] 2.3 `tpv-client/src/components/ClientCollection.tsx` — Replace hex: overlay, modal, buttons, form elements
- [x] 2.4 `tpv-client/src/components/ClientDossierView.tsx` — Replace hex: cards, stat colors, badges, action buttons
- [x] 2.5 `tpv-client/src/components/EntityManager.tsx` — Replace hex: hero gradient, pills, table; remove inline `@import`
- [x] 2.6 `tpv-client/src/components/EntityDetailPanel.tsx` — Replace hex: backdrop, panel, tabs, badges, buttons, feedback
- [x] 2.7 `tpv-client/src/components/ShareDocumentModal.tsx` — Replace hex: overlay, modal, WhatsApp button, copy button

## Phase 3: Verification

- [x] 3.1 Manual smoke test — toggle light/dark, verify all 7 components render correctly under both themes, no broken layouts
- [x] 3.2 Accessibility audit — verify WCAG AA color contrast in both themes via dev tools or axe
