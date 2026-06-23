# Design: TPV UI Facelift

## Technical Approach

Replace all hardcoded `React.CSSProperties` hex values with `var(--name)` references that resolve to CSS custom properties under `:root` (light) and `[data-theme="dark"]` (dark). A lightweight React context (`ThemeProvider`) manages the `data-theme` attribute on `<html>` and persists the choice in `localStorage`. Google Fonts (Outfit/Inter) load synchronously via `<link>` in `index.html` with a blocking script to prevent FOUC. Applies to REQ-CLI-02 (dynamic banner colors) and REQ-CLI-04 (runtime theme toggle).

## Architecture Decisions

| Option | Alternatives | Decision |
|--------|-------------|----------|
| **CSS Variables + inline mapping** — replace hex → `var(--name)` in existing style objects | Tailwind migration, CSS Modules | **CSS Variables**: zero layout risk, preserves existing component structure, no build tooling changes |
| **`data-theme` attribute** on `<html>` | CSS `prefers-color-scheme`, CSS `:has()` toggle | **data-theme**: highest CSS specificity override over inline `var()` refs, matches React state one-to-one |
| **Inline `var()` references** inside `React.CSSProperties` | Extract all styles to CSS classes | **Inline var()**: minimal diff per file, no refactoring of layout logic. CSS variable changes cascade automatically |
| **Google Fonts `<link>` in `<head>`** | CSS `@import`, `font-display: swap` | **`<link>` + blocking**: avoids FOIT, consolidates EntityManager's inline `@import` |

## Data Flow

```
┌─────────────────────────────────────────────────────────┐
│  index.html (blocking <script>)                         │
│  → reads localStorage → sets data-theme on <html>       │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│  App.tsx                                                │
│  ├── <ThemeProvider> (creates context, syncs attr)      │
│  ├── <ThemeToggle> button                               │
│  └── wraps all child components                         │
└─────────────────────────────────────────────────────────┘
        │                                      │
        ▼                                      ▼
┌───────────────────┐          ┌──────────────────────────┐
│  App.css           │          │  Component style objects  │
│  :root { ... }      │          │  backgroundColor:         │
│  [data-theme=dark]  │          │    "var(--bg-card)"       │
│  { ... }            │          │  color: "var(--text-pri)" │
└───────────────────┘          └──────────────────────────┘
        │                                      │
        └────────── CSS cascade ──────────────┘
                           │
                           ▼
                   Rendered DOM
              (data-theme="light|dark")
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `tpv-client/index.html` | Modify | Add Google Fonts `<link>`, blocking theme-init `<script>`, remove Vite default title |
| `tpv-client/src/App.css` | Rewrite | Replace all rules with `:root` + `[data-theme="dark"]` CSS variable definitions, glassmorphism templates, keyframes |
| `tpv-client/src/App.tsx` | Modify | Import `ThemeProvider`, wrap app, add theme toggle button to header, remove all `const xxxStyle` blocks, inline `var()` references |
| `tpv-client/src/components/SyncWarningBanner.tsx` | Modify | Replace hex colors: amber/green banner → `var(--status-warning-*)`, `var(--status-syncing-*)` |
| `tpv-client/src/components/RouteSetup.tsx` | Modify | Replace hex colors: panel bg, table, buttons, status box |
| `tpv-client/src/components/ClientCollection.tsx` | Modify | Replace hex colors: overlay, modal, buttons, form elements |
| `tpv-client/src/components/ClientDossierView.tsx` | Modify | Replace hex colors: cards, stat colors, badges, action buttons |
| `tpv-client/src/components/EntityManager.tsx` | Modify | Replace hex colors: hero gradient, pills, table; remove inline `@import` |
| `tpv-client/src/components/EntityDetailPanel.tsx` | Modify | Replace hex colors: backdrop, panel, tabs, badges, buttons, feedback |
| `tpv-client/src/components/ShareDocumentModal.tsx` | Modify | Replace hex colors: overlay, modal, WhatsApp button, copy button |
| `tpv-client/src/main.tsx` | No change | ThemeProvider wraps inside App.tsx, no structural change needed |

## Interfaces / Contracts

```typescript
// Types for ThemeProvider context
type Theme = "light" | "dark";

interface ThemeContextValue {
  theme: Theme;
  toggleTheme: () => void;
}
```

**CSS Variable Structure** (representative subset — full set in `App.css`):

```
:root {
  /* Typography */
  --font-heading: 'Outfit', sans-serif;
  --font-body: 'Inter', system-ui, sans-serif;

  /* Surfaces */
  --bg-page: #f8fafc;
  --bg-card: #ffffff;
  --bg-elevated: #ffffff;

  /* Text */
  --text-primary: #0f172a;
  --text-secondary: #334155;
  --text-muted: #64748b;
  --text-placeholder: #94a3b8;

  /* Borders */
  --border-default: #e2e8f0;
  --border-input: #cbd5e1;

  /* Accent */
  --accent-default: #2563eb;
  --accent-ring: #3b82f6;

  /* Statuses */
  --status-warning-bg: #fffbeb;
  --status-warning-text: #92400e;
  --status-success-bg: #dcfce7;
  --status-success-text: #166534;
  --status-error-bg: #fee2e2;
  --status-error-text: #991b1b;

  /* Glass */
  --glass-bg: rgba(255, 255, 255, 0.8);
  --glass-border: rgba(255, 255, 255, 0.18);
}

[data-theme="dark"] {
  --bg-page: #0f172a;
  --bg-card: #1e293b;
  --text-primary: #f1f5f9;
  --text-secondary: #cbd5e1;
  --text-muted: #94a3b8;
  --border-default: #334155;
  --accent-default: #3b82f6;
  --status-warning-bg: #78350f;
  --status-warning-text: #fde68a;
  --status-success-bg: #064e3b;
  --status-success-text: #6ee7b7;
  --status-error-bg: #7f1d1d;
  --status-error-text: #fca5a5;
  --glass-bg: rgba(30, 41, 59, 0.8);
  --glass-border: rgba(255, 255, 255, 0.08);
}
```

**Blocking init script** (injected in `index.html` `<head>`):

```html
<script>
  (function() {
    var theme = localStorage.getItem('tpv-theme');
    if (!theme) {
      theme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }
    document.documentElement.setAttribute('data-theme', theme);
  })();
</script>
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | ThemeProvider toggle, localStorage persistence, initial theme detection | Render provider, assert `data-theme` attribute changes, verify localStorage read/write |
| Visual | Every component renders with both themes | Manual smoke test: toggle light/dark, verify contrast, no broken layouts |
| E2E | REQ-CLI-04 scenarios (toggle runtime) | Playwright: toggle theme, assert CSS variable resolves to correct color on a known element |
| A11y | Color contrast in both themes | Axe DevTools or `@axe-core/react` — check WCAG AA compliance |

## Migration / Rollout

No migration required. Theme toggle defaults to system preference (`prefers-color-scheme`). Existing users see the same light theme initially; they can toggle to dark at any time. Preference persists in `localStorage`.

## Open Questions

- [ ] Confirm that the WhatsApp green (`#25d366`) in ShareDocumentModal should remain fixed (brand color) or also theme-adapt.
- [ ] Confirm dark mode hero gradient colors for EntityManager (currently `#0f172a → #1e3a5f → #1d4ed8` — may need a darker variant).
