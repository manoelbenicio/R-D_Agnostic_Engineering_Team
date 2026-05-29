# design-system-indra-alignment — Design

## Context

AgentVerse v1 shipped with the **SENTINEL** design tokens (cyber-dark aesthetic: `#06090d` voids, `#00f0ff` cyan, JetBrains Mono everywhere). The customer-facing standard is now the **Indra Group** brand defined in `dashboards_templates/template_padrao.html` (corporate teal: `#002B3A` deep, `#00B0BD` cyan, Inter for body, JetBrains Mono for code).

The locked design-system spec at `milestone-1-canvas-deploy-run/specs/design-system-sentinel/spec.md` requires every UI surface to consume tokens via CSS variables (no raw hex literals). 34 files in `src/` reference these tokens by name.

## Decision: token-only swap with preserved variable names

Three implementation strategies were considered:

| Strategy | Pros | Cons |
|----------|------|------|
| **A. Rename tokens** (`--cyan` → `--indra-cyan` everywhere) | Self-documenting | Touches all 34 consumers; high blast radius; breaks 11 design-system components; reopens unit tests |
| **B. Coexist palettes** (`--sentinel-cyan` + `--indra-cyan` side by side, opt-in via class) | Theming preserved, A/B reversible | Doubles token surface; consumers must know which palette they're on; defeats "100% Indra alignment" |
| **C. Remap values, preserve names** ✅ | Zero touch on 34 consumers; same components, same tests, new look; reversible by editing one file | Variable names lie about meaning (`--cyan` no longer SENTINEL `#00f0ff`) |

**Chosen: C.** Justification:
- The locked spec defines tokens by **role** (surface, accent, danger, success), not by literal value. Remapping role-tokens to a new palette preserves spec compliance.
- Single point of change → reversible in seconds.
- 159 unit tests + 6 design-system component tests assert on `var(--name)` and never on hex values; no test rewrites required.
- The `--indra-*` tokens are added as first-class so future components can refer to them explicitly when "Indra" semantics are intended.

## Token mapping (SENTINEL → Indra)

### Surfaces
| SENTINEL alias | Old value | New value | Indra source |
|----------------|----------|-----------|--------------|
| `--void` | `#06090d` | `var(--indra-deep)` | `#002B3A` |
| `--panel` | `rgba(11,15,23,0.7)` | `rgba(0,43,58,0.85)` | derived from indra-deep |
| `--card` | `rgba(11,15,23,0.65)` | `var(--indra-card-surface)` | `rgba(0,62,80,0.45)` |

### Brand accents
| SENTINEL alias | Old value | New value | Indra source |
|----------------|----------|-----------|--------------|
| `--cyan` | `#00f0ff` | `var(--indra-cyan)` | `#00B0BD` |
| `--amber` | `#ffb700` | `var(--indra-gold)` | `#FFC107` |
| `--threat` | `#ff3b30` | `var(--indra-error)` | `#E91E63` |
| `--ops` | `#00ff66` | `var(--indra-success)` | `#27AE60` |

### Text
| SENTINEL alias | Old value | New value | Indra source |
|----------------|----------|-----------|--------------|
| `--text-primary` | `#e2e8f0` | `var(--indra-white)` | `#FFFFFF` |
| `--text-muted` | `#94a3b8` | `var(--indra-blue-gray)` | `#B3C1DA` |
| `--text-dim` | `#64748b` | `var(--indra-warm-gray)` | `#B0B4BD` |

### Borders
| SENTINEL alias | Old value | New value |
|----------------|----------|-----------|
| `--border` | `rgba(255,255,255,0.08)` | `var(--indra-border)` (same value, renamed source) |
| `--border-accent` | `rgba(0,240,255,0.25)` | `rgba(0,176,189,0.25)` (Indra cyan @ 25%) |

### Ambient effects (recolored)
- `.scan-sweep::before` background+shadow: `rgba(0,240,255,*)` → `rgba(0,176,189,*)`
- `.kpi-glow` box-shadow: same recolor
- `.sentinel-btn-sys` focus/hover shadows: same recolor

## Decision: Inter as default sans, JetBrains Mono as code-only

Indra spec uses `Inter` for body/display, `JetBrains Mono` reserved for monospace. AgentVerse D13 in `ARCHITECTURE.md` says "JetBrains Mono everywhere as default". This is a deliberate **deviation from D13**.

Rationale:
- 100% alignment with Indra is a hard requirement from the customer-facing standard.
- D13 was a v1 default, not a load-bearing constraint — the design-system spec only requires that the type tokens exist and resolve, not that any specific family wins.
- User-configurable font overrides remain functional via `applyFontOverrides()` in `src/design-system/utils/font-override.ts`. Users who want JetBrains Mono everywhere can still flip the switch in Settings → Appearance.

Recorded follow-up: ARCHITECTURE.md D13 entry must be amended in a separate change to reflect that Inter is now the primary default and JetBrains Mono is the code/terminal default. Tracked as task 5.1 in this change's `tasks.md`.

## New tokens (additions, not replacements)

These are introduced as first-class for any component that wants to opt into Indra semantics explicitly:

```css
/* Brand */
--indra-deep --indra-dark --indra-primary --indra-secondary
--indra-teal --indra-cyan --indra-light --indra-blue-gray
--indra-sky --indra-warm-gray --indra-off-white --indra-white
--indra-card-surface --indra-border

/* Status */
--indra-success --indra-warning --indra-error --indra-gold

/* Type scale */
--text-xs(11) --text-sm(12) --text-base(14) --text-md(15)
--text-lg(16) --text-xl(18) --text-2xl(24) --text-3xl(32) --text-4xl(48)

/* Weights (Inter 300-900) */
--weight-light --weight-regular --weight-medium --weight-semibold
--weight-bold --weight-extra --weight-black

/* Line heights */
--leading-tight(1.1) --leading-snug(1.3) --leading-normal(1.6)

/* Letter spacing */
--tracking-tight --tracking-normal --tracking-wide
--tracking-wider --tracking-widest
```

## Out-of-band fix: `.env.local` MSW flag

During smoke verification a pre-existing bug surfaced: `.env.local` had `VITE_USE_MSW=1` but `src/main.tsx:27` does `import.meta.env.VITE_USE_MSW !== 'true'`. The string `'1'` was silently disabling MSW, which would have broken the milestone-1 task 22.4 demo against the running dev server. Corrected to `VITE_USE_MSW=true`. This fix is included here because it was discovered while running the gates required to prove this change is safe.

## What this change does NOT do

- Does not port Indra's KPI-card / sidebar / topbar layouts. Component HTML is untouched.
- Does not change any logo or brand asset. No SVGs added.
- Does not amend `ARCHITECTURE.md` D13. That update is task 5.1, deferred as a follow-up.
- Does not re-audit a11y beyond `axe` automated check. Manual contrast review on production data is recommended but not blocking.
- Does not migrate the locked design-system spec to "Indra Design System" — only its color/typography requirements are modified. Spec name `design-system-sentinel` remains for capability-routing continuity.

## Reversibility

Single-file revert: `git checkout HEAD~1 -- src/design-system/tokens.css index.html .env.local` returns to SENTINEL. No other files need touching.
