## MODIFIED Requirements

### Requirement: SENTINEL Design Tokens

The system SHALL expose color, typography, spacing, radius, and shadow values as CSS custom properties (`--*` tokens) defined on `:root`. All UI components in AgentVerse SHALL consume tokens through these CSS variables; SHALL NOT hard-code hex colors or font stacks; and SHALL NOT introduce additional palette colors without a corresponding token.

The minimum token set for v1 includes:

- **Color (legacy SENTINEL aliases, retained for backwards compatibility, values aligned with Indra brand)**: `--void`, `--panel`, `--card`, `--cyan`, `--amber`, `--threat`, `--ops`, `--text-primary`, `--text-muted`, `--text-dim`, `--border`, `--border-accent`. These tokens resolve to Indra-equivalent values per the mapping in `openspec/changes/design-system-indra-alignment/design.md`.
- **Color (Indra brand, first-class)**: `--indra-deep`, `--indra-dark`, `--indra-primary`, `--indra-secondary`, `--indra-teal`, `--indra-cyan`, `--indra-light`, `--indra-blue-gray`, `--indra-sky`, `--indra-warm-gray`, `--indra-off-white`, `--indra-white`, `--indra-card-surface`, `--indra-border`, `--indra-success`, `--indra-warning`, `--indra-error`, `--indra-gold`.
- **Typography defaults**: `--font-body` and `--font-display` SHALL resolve to **Inter** (deviation from `ARCHITECTURE.md` D13 default of JetBrains Mono, recorded in `design.md`). `--font-mono` SHALL remain JetBrains Mono. All three SHALL remain user-overridable through Settings → Appearance via `applyFontOverrides()`.
- **Type scale**: `--text-xs` (11px), `--text-sm` (12px), `--text-base` (14px), `--text-md` (15px), `--text-lg` (16px), `--text-xl` (18px), `--text-2xl` (24px), `--text-3xl` (32px), `--text-4xl` (48px) — per Indra spec.
- **Font weights**: `--weight-light` (300), `--weight-regular` (400), `--weight-medium` (500), `--weight-semibold` (600), `--weight-bold` (700), `--weight-extra` (800), `--weight-black` (900). Inter SHALL be loaded with all seven weights via `index.html` `<link>` preconnect.
- **Line heights**: `--leading-tight` (1.1), `--leading-snug` (1.3), `--leading-normal` (1.6).
- **Letter spacing**: `--tracking-tight` (-0.02em), `--tracking-normal` (0), `--tracking-wide` (0.05em), `--tracking-wider` (0.08em), `--tracking-widest` (0.1em).
- **Radius**: `--radius-card` (8px), `--radius-button` (6px), `--radius-badge` (9999px — full pill).
- **Spacing scale**: `--space-1` (4px) … `--space-8` (40px).

Color, radius, and spacing token values SHALL match the Indra brand palette as defined in `dashboards_templates/template_padrao.html`. Typography defaults SHALL follow Indra's Inter+JetBrains-Mono pairing.

#### Scenario: Component uses a token not a literal

- **WHEN** an audit examines any component in `src/design-system/` or any consumer thereof
- **THEN** there are zero raw hex color literals or `rgb(...)` values in the source — all colors flow through `var(--*)` tokens
- **AND** there are zero font-family declarations referencing a typeface other than via `var(--font-*)`

#### Scenario: Tokens load before any styled element renders

- **WHEN** the SPA mounts in a fresh browser
- **THEN** `:root` exposes every required token as a non-empty CSS custom property
- **AND** Inter and JetBrains Mono are loaded via the `index.html` `<link>` preconnect before first paint

#### Scenario: SENTINEL alias resolves to Indra-equivalent value

- **WHEN** a component uses `var(--cyan)`
- **THEN** the resolved value is `#00B0BD` (`--indra-cyan`)
- **AND** when a component uses `var(--threat)`, the resolved value is `#E91E63` (`--indra-error`)
- **AND** when a component uses `var(--ops)`, the resolved value is `#27AE60` (`--indra-success`)
- **AND** when a component uses `var(--amber)`, the resolved value is `#FFC107` (`--indra-gold`)
- **AND** when a component uses `var(--void)`, the resolved value is `#002B3A` (`--indra-deep`)

#### Scenario: Indra brand tokens are first-class

- **WHEN** a new component is written that wants explicit Indra semantics
- **THEN** it MAY consume `var(--indra-deep)`, `var(--indra-cyan)`, `var(--indra-success)`, etc. directly without going through SENTINEL aliases
- **AND** existing components that consume SENTINEL aliases continue to work without modification

#### Scenario: User-configurable font override still functions

- **WHEN** a user changes display/body/mono font in Settings → Appearance
- **THEN** `applyFontOverrides()` sets the corresponding `--font-display` / `--font-body` / `--font-mono` on `document.documentElement.style`
- **AND** that override takes precedence over the Inter+JetBrains-Mono defaults set in `tokens.css`
