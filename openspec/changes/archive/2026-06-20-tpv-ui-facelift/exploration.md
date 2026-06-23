## Exploration: tpv-ui-facelift

### Current State
Today, the `tpv-client` styling relies on basic, hardcoded light theme colors declared as `React.CSSProperties` constants inside the React component files themselves (e.g., at the bottom of `App.tsx`, `ClientDossierView.tsx`, `EntityManager.tsx`). The default `App.css` only handles simple element styling and has a basic prefers-color-scheme media query that does not affect layout styling due to inline style precedence. This design prevents cohesive theme control and blocks runtime dark/light mode toggling.

### Affected Areas
- `tpv-client/index.html` — Integrate modern typography (`Outfit` and `Inter`) and inject a theme synchronization head script to prevent flash-on-load.
- `tpv-client/src/App.css` — Standardize `:root` and `[data-theme='dark']` CSS custom properties, define premium typography rules, glassmorphism templates, hover transitions, and status micro-animations.
- `tpv-client/src/App.tsx` — Integrate a custom `ThemeProvider` context and wrapper, mount the top header theme switch control, and refactor layout containers to reference CSS variables.
- `tpv-client/src/components/SyncWarningBanner.tsx` — Map warning/success states to dynamic colors and apply slide-down and rotation keyframe micro-animations.
- `tpv-client/src/components/RouteSetup.tsx` — Convert customer listing tables, checkboxes, search, and action buttons to themed attributes.
- `tpv-client/src/components/ClientCollection.tsx` — Refactor the Payment Collection modal overlay, form fields, and dropdowns to dynamic variables.
- `tpv-client/src/components/ClientDossierView.tsx` — Transition credit status cards (using custom left border indicators) and transaction history tables.
- `tpv-client/src/components/EntityManager.tsx` — Redesign the gradient hero container, tab selectors, and active state pill chips to adapt smoothly to dark backgrounds.
- `tpv-client/src/components/EntityDetailPanel.tsx` — Adapt slide-over details panel, tab navigation, address lists, and form inputs.
- `tpv-client/src/components/ShareDocumentModal.tsx` — Refactor textarea preview cards, copy triggers, and WhatsApp share buttons.

### Approaches
1. **Approach 1: CSS Variables + Inline Styles Mapping (Recommended)**
   - Define all light/dark system colors as CSS custom properties in `App.css`. Wrap the React app in a lightweight context `ThemeProvider` that controls the `data-theme` attribute on the HTML element. In individual components, retain the existing `React.CSSProperties` structure but replace hardcoded hex values with CSS variables (e.g., `backgroundColor: "var(--bg-card)"`).
   - Pros: Near-zero regression risk for existing desktop layouts; fast implementation; preserves simple structure without introducing third-party library dependencies.
   - Cons: Inline styles remain in components, meaning styles are not fully centralized.
   - Effort: Low-Medium

2. **Approach 2: Full Tailwind CSS Migration**
   - Install Tailwind CSS, configure the Vite compiler, remove all inline style objects, and rewrite every component with Tailwind classes and utility prefixes (`dark:`).
   - Pros: Highly scalable and clean design system for long-term codebase evolution.
   - Cons: Massive regression risk on fine-grained flex layouts and absolute modal position elements; takes significantly more time and tokens.
   - Effort: High

### Recommendation
We recommend **Approach 1 (CSS Variables + Inline Styles Mapping)**. It offers the fastest route to a premium dark/light mode facelift while ensuring absolute layout safety in the desktop Tauri client, without introducing additional tooling complexity.

### Risks
- **Contrast Ratios**: Some secondary items (like timestamps, credit limits, or disabled inputs) may fall below accessible contrast limits in dark mode if gray scales are not carefully selected.
- **Tauri Native Flash**: Desktop clients can show a brief white background flash when loading before React is hydrated.
  - *Mitigation*: Inject a tiny blocking script in `index.html`'s `<head>` to read and set the `data-theme` immediately.

### Ready for Proposal
Yes. The next step is for the orchestrator to request approval to generate the formal OpenSpec Proposal and Design documents outlining exact CSS property names and component mappings.
