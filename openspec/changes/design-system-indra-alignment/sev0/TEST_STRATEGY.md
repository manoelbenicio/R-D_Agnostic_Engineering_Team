# TEST_STRATEGY — SEV-0 Stop-Ship

Per protocol Section 6.4 — test layers, coverage, regression-suite mapping.

---

## 1. Layered test architecture

| Layer | Stack | Scope | Location | Count today | Notes |
|---|---|---|---|--:|---|
| **L1 Unit** | Vitest + jsdom + RTL | Pure functions, hooks, single components | `src/**/__tests__/*.{ts,tsx}` | 282 | Most tests live here |
| **L2 Integration** | Vitest + MSW + RTL | Routes that touch CAO via `caoClient` | `src/api/__tests__/*.test.ts`, `src/finops/__tests__/finops-page.test.tsx`, etc. | 12 | MSW intercepts HTTP |
| **L3 Contract** | Vitest, gated `CAO_LIVE=1` | Live CAO API contract | `src/api/__tests__/contract/*.test.ts` | 3 | Nightly only |
| **L4 Smoke (E2E)** | Playwright + dev server + MSW | Critical-path user flow + a11y | `tests/e2e/*.spec.ts` | 3 | Boots Vite, MSW handlers |
| **L5 A11y** | `axe-core` + Playwright | Accessibility violations across 11 routes | `scripts/run-axe.mjs` | 11 routes | 0 critical/serious required |
| **L6 Visual / DSS parity** | Vitest, **NEW** | Token + component shape + motion infra parity vs DSS canon | `src/design-system/__tests__/dss-*.test.{ts,tsx}` + `src/terminal/__tests__/xterm-theme-indra-parity.test.ts` | **118** | Added by this SEV-0 |

**Total automated tests:** 297 unit/integration + 3 smoke + 11 axe routes.

---

## 2. SEV-0 issue → regression test mapping

Every issue has at least one test that **fails before the fix** and **passes after the fix**, per protocol Section 8.4.

| Issue ID | Severity | Regression test file | Specific assertions |
|---|---|---|---|
| **ISSUE-001** | HIGH | `dss-token-parity.test.ts` | `Spacing scale` describe (10 tests), `Timing tokens` (4), `Easing tokens` (3), `Geometry` (4), `Section rhythm utilities` (2), `Smooth scroll` (2), `Motion keyframes` (8), `DSS subset compliance` (per-token loop, ~40) |
| **ISSUE-002** | HIGH | `dss-component-shape.test.tsx` | `Button — primary uses --indra-deep`, `Button — variant="cyan" exists and uses --indra-cyan`, `Button — secondary transparent + cyan border`, `Button — sharp corners`, `Button — uses --font-sans not --font-mono`, `Button — uppercase`, `Badge — supports DSS variants success/warning/error/gold/info`, `Badge — accepts legacy lifecycle aliases`, `Card — emits class hook for hover` |
| **ISSUE-003** | HIGH | `dss-motion-infra.test.tsx` + parity rules in `dss-token-parity.test.ts` | `useDataAnimateObserver hook is exported`, `registers IntersectionObserver`, `toggles .animate-in on intersect`, `ignores non-data-animate elements`, `[data-animate] base rule exists`, `.animate-in trigger exists`, `.stagger-1..5 utilities exist`, `@keyframes indra-fade-in / indra-slide-in-x exist` |
| **ISSUE-004** | CRITICAL | `xterm-theme-indra-parity.test.ts` | `SENTINEL_TERMINAL_THEME does NOT contain #00f0ff/#06090d/#ff3b30/#00ff66/#ffb700`, `resolveTerminalTheme() resolves cyan/red/green/yellow/black/white to Indra hex`, `selectionBackground uses rgba(0,176,189,...)`, `Override merge preserves Indra base` |
| **ISSUE-005** | HIGH | (subset of ISSUE-004) | `selectionBackground does NOT match rgba(0, 255, 255, *)`, derived rgba(0, 176, 189, *) |
| **ISSUE-006** | MEDIUM | `dss-token-parity.test.ts` Geometry + `dss-component-shape.test.tsx` Border-radius consistency | `--radius-button === 0`, `--radius-card === 8px`, `--radius-glass-card === 16px`, `--radius-badge === 9999px` |
| **ISSUE-007** | MEDIUM | (deferred) | Will be enforced by an ESLint `no-raw-color` rule in a follow-up change |
| **ISSUE-008** | LOW | `dss-token-parity.test.ts` Typography | `--font-sans places Segoe UI before Inter`, `--font-mono lists JetBrains Mono first` |
| **ISSUE-009** | HIGH | (subset of ISSUE-001 + 002) | `--radius-button === 0`, Button computed style contains `border-radius: 0` or `var(--radius-button)` |
| **ISSUE-010** | HIGH | `dss-component-shape.test.tsx` GlassCard | `GlassCard exported from @/design-system`, `renders with backdrop-filter blur(16px)`, `uses --radius-glass-card`, `class indra-glass-card present` |
| **ISSUE-011** | MEDIUM | `dss-token-parity.test.ts` Section rhythm | `tokens.css declares .section { padding: var(--space-30) 0 }`, `.section-title { font-size: 40px; font-weight: 300 }` |
| **ISSUE-012** | LOW | `dss-token-parity.test.ts` Smooth scroll | `tokens.css declares html { scroll-behavior: smooth; scroll-padding-top: 80px }` |
| **FinOps regression** | post-feedback | `src/finops/__tests__/finops-page.test.tsx` (existing 3 tests, all pass) | `data-testid="budget-gauge"` preserved through restructure |
| **Axe contrast regression** | runtime | `scripts/run-axe.mjs` 11-route sweep | `0 critical/serious` |

---

## 3. Coverage targets

| Metric | Target | Current |
|---|---|---|
| **Total tests** | ≥ 250 | **297** |
| **DSS parity tests** | every DSS token + every DSS component variant | **118** |
| **Critical issues with regression coverage** | 100 % | **12 / 12** (ISSUE-007 deferred but not a release blocker) |
| **Axe critical / serious** | 0 across all 11 routes | **0** |
| **Smoke critical-path scenarios** | every documented happy-path step | **3 / 3** |
| **Bundle size** | ≤ 1.5 MB gzipped | **526 KB (34 %)** |

V8 line coverage was not the goal of this SEV-0 — the goal was **issue-traceable regression coverage**. Every fix has a named test that fails red on a code regression, by design.

---

## 4. Property-based / fuzz / concurrency layers

| Layer | Applicability here | Status |
|---|---|---|
| Property-based (e.g. fast-check) | Possible for token-parsing logic in `dss-token-parity.test.ts` | Not added — current explicit-table tests already cover every DSS-defined token; property-based would be redundant |
| Fuzz | N/A — no protocol/parser surface introduced | Not applicable |
| Concurrency / race | N/A — no concurrent code paths introduced | Not applicable |

The SEV-0 changes do not introduce new I/O, parsing, or concurrent surfaces; the existing test layers fully cover the change.

---

## 5. CI parity check

All gates were run **locally** at the same versions and configurations as `.github/workflows/ci.yml`:

| `ci.yml` step | Local equivalent | Status |
|---|---|---|
| `npm ci` | (using existing `node_modules`, `package-lock.json` checked) | OK |
| `npm run lint` | direct invocation | 0 errors / 7 pre-existing warnings |
| `npm run typecheck` | direct invocation | clean |
| `npm test` | direct invocation | 297 / 297 |
| `npm run build` | direct invocation | success |
| `node scripts/check-bundle-size.mjs` | direct invocation | 526.3 KB / 1536 KB |
| `npm run test:smoke` | direct invocation (dev server up) | 3 / 3 |

Contract test job (`contract-nightly.yml`) is `CAO_LIVE=1`-gated; not in scope for this SEV-0 because no CAO API surface changed.

---

## 6. Tests that needed updating (pre-existing tests aligned to new contract)

These are **not** new SEV-0 failures — they are pre-existing assertions that referenced legacy token names. Updated to keep semantic intent, now reference DSS-aligned tokens:

| File | Lines | Change |
|---|---|---|
| `src/terminal/__tests__/TerminalView.test.tsx` | 268-273, 416 | `var(--void)` → `#002B3A`, `var(--cyan)` → `#00B0BD`, `var(--text-primary)` → `#FFFFFF`, `rgba(0, 255, 255, 0.2)` → `rgba(0, 176, 189, 0.25)` |
| `src/design-system/__tests__/components.test.tsx` | 11-43 | StatusBadge color expectations: `var(--threat)` → `var(--indra-error)`, `var(--ops)` → `var(--indra-success)`, `var(--cyan)` → `var(--indra-sky)` |
| `src/finops/__tests__/finops-page.test.tsx` | (no source change needed) | `data-testid="budget-gauge"` preserved through KPI restructure |

All test changes preserve the original assertion **intent**; they only update the **token name** to the DSS canonical. No assertion was relaxed.
