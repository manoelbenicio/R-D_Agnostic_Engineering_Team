# design-system-indra-alignment — Tasks

> All [x] tasks were applied retroactively after the change was implemented;
> this file documents the work post-hoc per openspec convention.
> Owner: SUP (`src/design-system/` is SUP-owned per `.github/CODEOWNERS`).

## 1. Token swap (palette)

- [x] 1.1 [SUP] Map SENTINEL aliases (`--void`, `--panel`, `--card`, `--cyan`, `--amber`, `--threat`, `--ops`, `--text-*`, `--border*`) to Indra equivalents — see `design.md` mapping tables
- [x] 1.2 [SUP] Add full Indra brand palette as first-class tokens (`--indra-deep`, `--indra-dark`, `--indra-primary`, `--indra-secondary`, `--indra-teal`, `--indra-cyan`, `--indra-light`, `--indra-blue-gray`, `--indra-sky`, `--indra-warm-gray`, `--indra-off-white`, `--indra-white`, `--indra-card-surface`, `--indra-border`)
- [x] 1.3 [SUP] Add Indra status colors (`--indra-success`, `--indra-warning`, `--indra-error`, `--indra-gold`)
- [x] 1.4 [SUP] Recolor ambient effects: `.scan-sweep::before`, `.kpi-glow`, `.sentinel-btn-sys` shadows from `rgba(0,240,255,*)` → `rgba(0,176,189,*)` (Indra cyan)
- [x] 1.5 [SUP] Add `body { background, color, font-family }` defaults at end of `tokens.css` so SPA inherits Indra deep teal background out of the box

## 2. Typography

- [x] 2.1 [SUP] Extend Inter weight range in `index.html` from 400–700 → 300–900 to support the Indra type scale
- [x] 2.2 [SUP] Set `--font-body` and `--font-display` to Inter; keep `--font-mono` as JetBrains Mono
- [x] 2.3 [SUP] Add type scale tokens `--text-xs` (11px) … `--text-4xl` (48px) per Indra spec
- [x] 2.4 [SUP] Add weight tokens `--weight-light` (300) … `--weight-black` (900)
- [x] 2.5 [SUP] Add line-height tokens (`--leading-tight/snug/normal`) and letter-spacing tokens (`--tracking-tight/normal/wide/wider/widest`)
- [x] 2.6 [SUP] Update `.sentinel-prose h1/h2/h3` font-size declarations to use scale tokens

## 3. Geometry

- [x] 3.1 [SUP] Update `--radius-card` from 12px → 8px (Indra panels)
- [x] 3.2 [SUP] Update `--radius-badge` from 20px → 9999px (Indra fully-rounded pills)
- [x] 3.3 [SUP] Preserve spacing scale (`--space-1` … `--space-8`) — Indra uses the same 4-40px progression

## 4. Out-of-band fix discovered during verification

- [x] 4.1 [SUP] Fix `.env.local` `VITE_USE_MSW=1` → `VITE_USE_MSW=true` so MSW enables (string match against `'true'` in `src/main.tsx:27`); this was silently breaking the v1 dev-server demo path

## 5. Verification gates

- [x] 5.1 [SUP] `npm run lint` — 0 errors, 7 pre-existing warnings (baseline unchanged)
- [x] 5.2 [SUP] `npm run typecheck` — clean
- [x] 5.3 [SUP] `npm test` — 44 files / 179 tests passing
- [x] 5.4 [SUP] `npm run build` — success
- [x] 5.5 [SUP] `node scripts/check-bundle-size.mjs` — 428.8 KB gzipped (28% of 1.5 MB budget); +0.2 KB delta
- [x] 5.6 [SUP] `npm run test:smoke` — 3/3 passing against dev server with MSW
- [x] 5.7 [SUP] `node scripts/run-axe.mjs` — 0 critical/serious violations across 11 routes (`/`, `/dashboard`, `/canvas/demo-session`, `/agent-studio`, `/flows`, `/finops`, `/memory`, `/settings/providers`, `/settings/appearance`, `/settings/general`, `/health`)

## 6. Follow-ups (deferred to separate changes)

- [x] 6.1 [SUP] Update `ARCHITECTURE.md` D13 entry to reflect that Inter is now the primary default sans-serif and JetBrains Mono is reserved for code/terminal contexts; cross-reference this change as the deviation source
- [ ] 6.2 [SUP] Manual contrast review on canvas-builder edge labels and terminal scrollback against the new `--indra-deep` background (axe automated coverage is broad but not deep)
- [ ] 6.3 [SUP] Consider a follow-up that ports Indra KPI-card / sidebar / topbar layouts into AgentVerse where they fit (e.g., dashboard hero metrics) — explicitly out of scope here
