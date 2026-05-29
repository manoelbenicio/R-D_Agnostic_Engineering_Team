# design-system-indra-alignment

## Why

The customer-facing standard is now the Indra Group palette and typography defined in `dashboards_templates/template_padrao.html`. AgentVerse v1 originally shipped with the SENTINEL palette (cyber-dark: `#06090d`, `#00f0ff`, JetBrains-Mono-everywhere). To present a consistent corporate visual identity across the AgentVerse SPA and Indra dashboards, the design tokens must align 100% with the Indra spec.

## What Changes

**Strategy: token-only swap, names preserved, components untouched.**

1. `src/design-system/tokens.css` — full rewrite:
   - **Indra brand palette** introduced as first-class tokens (`--indra-deep`, `--indra-dark`, `--indra-primary`, `--indra-secondary`, `--indra-teal`, `--indra-cyan`, `--indra-light`, `--indra-blue-gray`, `--indra-sky`, `--indra-warm-gray`, `--indra-off-white`, `--indra-white`, `--indra-card-surface`, `--indra-success`, `--indra-warning`, `--indra-error`, `--indra-gold`).
   - **SENTINEL aliases retained**, values remapped: `--void → --indra-deep`, `--cyan → --indra-cyan`, `--amber → --indra-gold`, `--threat → --indra-error`, `--ops → --indra-success`, `--text-primary → --indra-white`, `--text-muted → --indra-blue-gray`, `--text-dim → --indra-warm-gray`, `--card → --indra-card-surface`, `--border → --indra-border`.
   - **Typography**: default `--font-body` and `--font-display` now resolve to **Inter** (was JetBrains Mono). `--font-mono` stays JetBrains Mono.
   - **Type scale**: 11/12/14/15/16/18/24/32/48 px tokens (`--text-xs` through `--text-4xl`) per Indra spec.
   - **Font weights**: 300–900 tokens (`--weight-light` through `--weight-black`).
   - **Line heights**: tight/snug/normal (1.1/1.3/1.6).
   - **Letter spacing**: tight/normal/wide/wider/widest.
   - **Radius**: `--radius-card` 12 → 8 px; `--radius-badge` 20 → 9999 px (full pill).
   - **Ambient effects** (scanlines, scan-sweep, kpi-glow): rgba values recolored from SENTINEL cyan `rgba(0, 240, 255, ...)` to Indra cyan `rgba(0, 176, 189, ...)`.
   - **Body default** (`background`, `color`, `font-family`) added at end of file so the SPA shell inherits Indra deep teal background out of the box.

2. `index.html` — extended Inter weight range from 400–700 to **300–900** (Indra uses all 7 weights).

3. `.env.local` corrected from `VITE_USE_MSW=1` to `VITE_USE_MSW=true` (string match in `src/main.tsx:27` requires the literal string `'true'` — `'1'` was silently disabling MSW). This is unrelated to the token swap but was discovered during smoke verification and would have broken the 22.4 demo.

## Impact

### Affected
- `index.html` (font preconnect)
- `src/design-system/tokens.css` (full rewrite)
- `.env.local` (env value normalization)

### Not affected (by design)
- 34 token consumers across `src/` (all use `var(--cyan)`, `var(--threat)`, etc. by name — values flow through CSS variables).
- 11 design-system component files (`Badge`, `Button`, `Card`, `CostLabel`, `FormField`, `Modal`, `NavBar`, `Prose`, `StatusBadge`, `Toast`, `index.ts`).
- `src/design-system/utils/font-override.ts` (still works; user-configurable fonts in Settings remain).
- All 159 unit tests + the design-system test suite (`__tests__/components.test.tsx`) — they assert variable names, not values.

### Quality gates re-verified after swap
| Gate | Before | After |
|------|-------:|------:|
| `npm run lint` | 0 errors / 7 warnings | **0 / 7** |
| `npm run typecheck` | clean | **clean** |
| `npm test` (44 files) | 179 passed | **179 passed** |
| `npm run build` | success | **success** |
| Bundle gzipped | 428.6 KB | **428.8 KB** (+0.2 KB, 28% of 1.5 MB budget) |
| `npm run test:smoke` | 3/3 | **3/3** |

### Architectural deviation logged
**D13 ("JetBrains Mono everywhere as default")** is now overridden by this alignment. Default `--font-display` and `--font-body` are Inter; mono usage continues for code/terminal via `--font-mono`. User-configurable font overrides in Settings → Appearance still function via `applyFontOverrides()`. ARCHITECTURE.md should be amended in a follow-up to reflect that Inter (sans) is now the primary default and JetBrains Mono is reserved for code/terminal contexts.

### Backwards compatibility
- All component imports unchanged.
- All existing CSS class hooks (`.sentinel-prose`, `.sentinel-btn-sys`, `.scanlines`, `.scan-sweep`, `.kpi-glow`) preserved.
- No public API change.

## Out of scope

- Component-level redesign (KPI card layouts, sidebar, topbar from `template_padrao.html` are NOT being ported into AgentVerse — only tokens).
- A11y re-audit beyond what `npm test` covers; the contrast on the new palette is expected to remain WCAG AA but a re-run of `scripts/run-axe.mjs` is recommended in a follow-up.
- Logo / brand assets (no logo SVG was changed).
- Update of ARCHITECTURE.md D13 entry — separate change.
