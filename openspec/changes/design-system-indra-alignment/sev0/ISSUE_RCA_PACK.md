# ISSUE_RCA_PACK — SEV-0 Stop-Ship

One blameless RCA per issue, per protocol Section 6.2.  
**Format:** Title · Severity · Impact · Timeline · Root cause · Detection gap · Corrective actions · Prevent recurrence.

---

## ISSUE-001 — Token contract drift vs DSS Universal Standard v3.0

| Field | Value |
|---|---|
| **Severity** | HIGH |
| **Impact** | `tokens.css` did not declare DSS-required timing/easing tokens, extended spacing (4–120 px), or sharp button geometry. Consumers had no way to consume DSS-prescribed motion timings. Type scale and weights were mostly correct. |
| **Timeline** | First detected 2026-05-28 17:25 (Step-2 enumeration). Reproduced 18:19 by `dss-token-parity.test.ts`. Fixed 18:23. Re-verified 18:30 (test green). |
| **Root cause** | Earlier `design-system-indra-alignment` change had aligned palette + typography but did not propagate the *full* DSS contract — timing/easing/extended-spacing tokens existed only in the reference `frontend/styles.css` and were never copied to `tokens.css`. |
| **Detection gap** | The 179 pre-existing tests asserted variable *names*, not the DSS-prescribed *values*. There was no test that diffed `tokens.css` against the canonical DSS source-of-truth file. |
| **Corrective actions** | • Code: `src/design-system/tokens.css` rewritten with full DSS contract (commit-equivalent diff). • Tests: `src/design-system/__tests__/dss-token-parity.test.ts` (81 assertions) parses both `frontend/styles.css` and `tokens.css` and asserts every DSS token resolves to the DSS value. Passes today; will fail any future drift. |
| **Prevention** | The new parity test runs in CI on every PR. Future changes that drift from DSS will fail before merge. |

---

## ISSUE-002 — Component shape mismatch (Button / Badge / Card)

| Field | Value |
|---|---|
| **Severity** | HIGH |
| **Impact** | `Button` `primary` variant rendered with cyan background (DSS specifies `--indra-deep` background + white-alpha border, with the cyan CTA being a separate `cyan` variant). `Button` used `--font-mono` (DSS uses `--font-sans`, uppercase). `Badge` had lifecycle-named variants (`idle/processing/completed/...`) instead of DSS semantic names (`success/warning/error/gold/info`). `Card` had no hover treatment. |
| **Timeline** | Detected 17:25, reproduced 18:19, fixed 18:23 by component rewrites. Re-verified 18:30. |
| **Root cause** | `src/design-system/components/Button.tsx` and `Badge.tsx` were built before the DSS standard was finalized. The earlier alignment change touched tokens only, not components. |
| **Detection gap** | No test asserted DSS variant taxonomy or visual treatment per variant. |
| **Corrective actions** | • Code: `Button.tsx` rewritten — variants `primary | cyan | secondary | ghost`; sharp 0 px corners; sans-serif uppercase; 44 px min-height; 12×28 px padding. `Badge.tsx` rewritten — DSS variants + backwards-compat aliases. `Card` got class-driven hover via tokens.css `.sentinel-card:hover { transform: translateY(-3px); box-shadow: 0 12px 28px rgba(0,0,0,0.3) }`. • Tests: `dss-component-shape.test.tsx` (17 assertions). |
| **Prevention** | Component-shape tests run in CI. Backwards-compat aliases mean downstream consumer code never breaks. |

---

## ISSUE-003 — Motion / animation infrastructure missing

| Field | Value |
|---|---|
| **Severity** | HIGH (this is the "behave exactly as architectures-showcase.html" demand) |
| **Impact** | The SPA had no entrance animations, no scroll-triggered reveal, no stagger utilities, no smooth scroll. Felt static next to the showcase reference. |
| **Timeline** | Detected 17:25 (`grep "data-animate|IntersectionObserver|stagger-[0-9]"` returned 0 hits in source). Fixed 18:23. Re-verified 18:30. |
| **Root cause** | Motion was never part of v1 scope — the original SENTINEL design only included scanlines + scan-sweep + KPI glow ambient effects. The DSS standard mandates `[data-animate]` infrastructure with `IntersectionObserver`-driven `.animate-in` triggers + stagger utilities. |
| **Detection gap** | No test asserted motion infra exists. |
| **Corrective actions** | • `tokens.css` adds `[data-animate]` base + `.animate-in` trigger + `.stagger-1..5` + `@keyframes indra-fade-in / indra-slide-in-x` + `html { scroll-behavior: smooth; scroll-padding-top: 80px }`. • New `src/design-system/hooks/use-data-animate-observer.ts` mounts an `IntersectionObserver` once at the layout level. Re-exported from `@/design-system`. • `AppLayout.tsx` mounts the hook. • Tests: `dss-motion-infra.test.tsx` (3 runtime assertions) + parity assertions in `dss-token-parity.test.ts` (keyframes, stagger, smooth scroll, `[data-animate]` rules — 11 assertions). |
| **Prevention** | Motion infra is now a tested, locked contract. Any regression to the keyframes / observer hook fails CI. |

---

## ISSUE-004 — xterm theme uses pre-Indra SENTINEL palette (CRITICAL)

| Field | Value |
|---|---|
| **Severity** | **CRITICAL** — highest-probability root cause of the director's "current color is a mess" complaint. |
| **Impact** | Every xterm pane in the SPA (canvas-builder, terminal-grid, chat-view) rendered with the pre-Indra cyber palette: `#00f0ff` cyan, `#06090d` void, `#ff3b30` red, `#00ff66` green, `#ffb700` yellow. xterm.js does not resolve CSS custom properties; the static theme had `var(--void)` etc. as string literals which xterm displayed as fallback colors. |
| **Timeline** | Detected 17:25 (`xterm-theme.ts:7-23` audit). Reproduced 18:19 (`xterm-theme-indra-parity.test.ts` 17 RED assertions). Fixed 18:23. Re-verified 18:30 GREEN. |
| **Root cause** | Two compounding factors: (a) xterm.js requires raw color strings — passing `var(--void)` resulted in browser fallback, not Indra; (b) the static fallback values themselves were the legacy SENTINEL hex (`#00f0ff` etc.) — never updated when the palette swap happened. |
| **Detection gap** | No test exercised the xterm theme contract. Existing TerminalView tests asserted the *literal string* `'var(--void)'` was passed to xterm — which is itself a bug the assertion concealed. |
| **Corrective actions** | • New `resolveTerminalTheme()` in `xterm-theme.ts` reads `getComputedStyle(document.documentElement).getPropertyValue('--indra-*')` at theme-construction time. SSR/test-safe fallback uses Indra hex literals (`#002B3A`, `#00B0BD`, `#27AE60`, `#FFC107`, `#E91E63`, `#FFFFFF`, `#B3C1DA`, `#B0B4BD`). • Static `SENTINEL_TERMINAL_THEME` export retained but now contains Indra hex (zero pre-Indra relics). • `createTerminalTheme()` legacy alias kept for backwards compatibility. • Tests: `xterm-theme-indra-parity.test.ts` (17 assertions) cover both the static export and the runtime resolver. • Updated 2 pre-existing `TerminalView.test.tsx` assertions to align with the new (correct) contract. |
| **Prevention** | Test asserts every channel matches Indra hex. Regression is impossible without the test going red. |

---

## ISSUE-005 — xterm selection background uses old SENTINEL cyan rgba

| Field | Value |
|---|---|
| **Severity** | HIGH (subset of ISSUE-004 but called out separately because text-selection is visible on every keystroke) |
| **Impact** | Selection highlight rendered with `rgba(0, 255, 255, 0.2)` — that is `#00FFFF`, not Indra cyan `#00B0BD`. |
| **Timeline** | Reproduced + fixed jointly with ISSUE-004. |
| **Root cause** | Same as ISSUE-004 — copy-paste of the legacy SENTINEL theme. |
| **Detection gap** | Same as ISSUE-004. |
| **Corrective actions** | `xterm-theme.ts` selectionBackground now `rgba(0, 176, 189, 0.25)` (Indra cyan @ 25%). Selection contrast slightly tighter than the prior SENTINEL choice but still WCAG-AA legible against `#FFFFFF` foreground. |
| **Prevention** | Asserted in `xterm-theme-indra-parity.test.ts` `selectionBackground` test. |

---

## ISSUE-006 — Border-radius inconsistency across surfaces

| Field | Value |
|---|---|
| **Severity** | MEDIUM |
| **Impact** | Buttons, badges, cards, toasts, scrollbars used 4 different radius values (`4px`, `6px`, `8px`, `20px`) instead of the 4 DSS values (`0` button, `8px` card, `16px` glass-card, `9999px` badge). |
| **Timeline** | Detected 17:25, fixed by ISSUE-001 + ISSUE-002 (token + component changes), verified by `dss-token-parity.test.ts` "Geometry" describe block + `dss-component-shape.test.tsx` "Button — sharp corners" assertion. |
| **Root cause** | `--radius-button` was originally `6px` (a guess); component CSS files used inline `4/8/20px` instead of the token. |
| **Detection gap** | No test enforced radius-token consumption. |
| **Corrective actions** | `--radius-button: 0`, `--radius-card: 8px`, `--radius-glass-card: 16px`, `--radius-badge: 9999px`. Component tests assert button uses `var(--radius-button)`. |
| **Prevention** | Radius tokens are tested. Inline `borderRadius: '8px'` etc. in capability files is tracked under FOLLOW-1 (ISSUE-007 sweep). |

---

## ISSUE-007 — Capability CSS files contain inline rgba literals

| Field | Value |
|---|---|
| **Severity** | MEDIUM (deferred — not in this fix) |
| **Impact** | 53 inline `rgba(...)` occurrences across 14 capability files (`canvas-builder.css`, `terminal-grid.css`, `voice/VoicePanel.tsx`, `agent-studio.css`, `health.css`, `memory-viewer.css`, `dashboard.css`, `finops.css`, `chat-view/ChatView.tsx`, etc.). All values are functionally on-spec (mostly `rgba(255,255,255,*)` overlays + `rgba(0,176,189,*)` Indra cyan tints). Not visible drift — but breaks the "no raw color literals" architectural rule. |
| **Timeline** | Detected 17:25 (grep `"rgba\("`). Deferred. |
| **Root cause** | Capability CSS files predate the alpha-token system. Authors used `rgba()` directly instead of `var(--cyan-tint)` etc. |
| **Detection gap** | No lint rule prohibited raw `rgba()` outside `tokens.css`. |
| **Corrective actions for follow-up change** | (a) Add ESLint rule `agentverse/no-raw-color` that blocks raw `#hex` and `rgba(...)` outside `src/design-system/tokens.css`. (b) Sweep all 14 capability files. (c) Re-run gates. |
| **Prevention** | Lint rule will block future drift. |

---

## ISSUE-008 — Sans-serif stack lists Inter before Segoe UI

| Field | Value |
|---|---|
| **Severity** | LOW |
| **Impact** | On Windows, the director's machine rendered Inter (loaded via Google Fonts) instead of Segoe UI. DSS prefers Segoe UI as the corporate-default with Inter as fallback. |
| **Timeline** | Detected 17:25, fixed 18:23, verified 18:30. |
| **Root cause** | Earlier change put Inter first; DSS spec finalized with Segoe UI first. |
| **Detection gap** | No test asserted font order. |
| **Corrective actions** | `--font-sans: 'Segoe UI', 'Inter', -apple-system, sans-serif;` in `tokens.css`. Test: `dss-token-parity.test.ts` "Typography (ISSUE-008)" describe — asserts Segoe UI substring index < Inter substring index. |
| **Prevention** | Test enforces order. |

---

## ISSUE-009 — Sharp button corners (border-radius: 0) not enforced

| Field | Value |
|---|---|
| **Severity** | HIGH |
| **Impact** | Every CTA button rendered with 6 px rounded corners; DSS specifies sharp 0 px corporate corners. Visible on every screen. |
| **Timeline** | Detected 17:25, fixed jointly with ISSUE-001 (token change) and ISSUE-002 (component rewrite). |
| **Root cause** | Token `--radius-button: 6px` was an arbitrary v1 default, not a DSS-derived value. |
| **Corrective actions** | `--radius-button: 0`. Component renders `border-radius: var(--radius-button)`. Test asserts the resolved style contains `0` or `var(--radius-button)`. |
| **Prevention** | Token-parity test. Component-shape test. |

---

## ISSUE-010 — No GlassCard / KPI primitive

| Field | Value |
|---|---|
| **Severity** | HIGH |
| **Impact** | DSS exports a `.glass-card` (backdrop-filter blur 16 px, 16 px radius, padded 32 px). AgentVerse pages (FinOps, Dashboard, Health) rendered KPI cards by hand without backdrop-filter, no shared primitive. |
| **Timeline** | Detected 17:25, new `GlassCard.tsx` component added 18:23, verified 18:30. |
| **Root cause** | The component never existed. |
| **Detection gap** | No test required it to exist. |
| **Corrective actions** | • New `src/design-system/components/GlassCard.tsx`. • Re-exported from `@/design-system`. • Test asserts existence + class hook + backdrop-filter style + radius token. |
| **Prevention** | Component-shape test. |

---

## ISSUE-011 — No section rhythm utilities

| Field | Value |
|---|---|
| **Severity** | MEDIUM |
| **Impact** | DSS specifies `.section { padding: 120px 0 }` and `.section-title { font-size: 40px; weight: 300 }`. Pages used ad-hoc paddings; visual coherence gap between routes. |
| **Timeline** | Detected 17:25, fixed by adding utilities to `tokens.css`, verified 18:30. |
| **Root cause** | Section rhythm was never tokenized. |
| **Corrective actions** | `.section`, `.section--alt`, `.section--light`, `.section-title`, `.section-eyebrow` utilities added to `tokens.css` (compiled into the global stylesheet). Test asserts `tokens.css` declares these rules with the DSS values. |
| **Prevention** | Token-parity test. Adoption per-capability is FOLLOW-2. |

---

## ISSUE-012 — `html { scroll-behavior: smooth }` missing

| Field | Value |
|---|---|
| **Severity** | LOW |
| **Impact** | Anchor jumps and `scrollIntoView()` calls did not smooth-scroll. Below the threshold of director attention but a DSS spec violation. |
| **Timeline** | Detected 17:25, fixed by `tokens.css` adding `html { scroll-behavior: smooth; scroll-padding-top: 80px }`, verified 18:30. |
| **Root cause** | `index.css` styled `body` but never set `html`-level scroll. |
| **Corrective actions** | `tokens.css` adds the `html` rule. Test asserts both declarations are present. |
| **Prevention** | Token-parity test. |

---

## Director-feedback regression — FinOps Budget Utilization KPI rhythm

(Not numbered as ISSUE-013 because it is a director-flagged usability concern post-fix, not a SEV-0 root issue.)

| Field | Value |
|---|---|
| **Severity** | MEDIUM (visual rhythm) |
| **Impact** | After SEV-0 fixes, the FinOps `Budget Utilization` KPI rendered the radial gauge first (148 px tall, dominant) and the `⚠ 0%` value below — out of rhythm with the other 3 KPIs which show `⚠ value` at the top. |
| **Timeline** | Detected 21:13 (director screenshot review). Fixed 21:19. |
| **Root cause** | Original ordering choice in `FinopsPage.tsx:62-77` placed the chart visually before the value. |
| **Corrective actions** | Restructured to put `<CostLabel value={formatPercent(budgetUtil)} />` first, then a 48 px sparkline-style radial arc below. CSS height changed from `148px` to a flex-stack with `48px` spark. FinOps tests re-run GREEN (3/3). |

---

## Axe contrast regression — `.logo-v1` chip

(Caught by axe sweep mid-flight; fixed before final ship.)

| Field | Value |
|---|---|
| **Severity** | HIGH (WCAG AA failure on every route) |
| **Impact** | After Indra alignment, `.logo-v1` chip text used `var(--neon-cyan) → #00B0BD` on `var(--cyan-tint) → rgba(0, 176, 189, 0.10)` background. The two values were too close to meet 4.5:1 minimum contrast. Axe failed across all 11 audited routes. |
| **Timeline** | Detected 19:30 (axe re-run after main fix). Fixed 19:32. Re-verified 19:35: 0/11 violations. |
| **Root cause** | Indra cyan `#00B0BD` is darker than the original SENTINEL `#00F0FF` cyan. The chip's text–background pair worked for SENTINEL but not Indra. |
| **Corrective actions** | `.logo-v1` color changed from `var(--neon-cyan)` to `var(--indra-white)`. White-on-cyan-tint over `--indra-deep` page background passes WCAG AA easily. |
| **Prevention** | Axe gate is part of the release-readiness checklist and runs against every route. |
