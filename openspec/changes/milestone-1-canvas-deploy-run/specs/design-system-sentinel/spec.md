## ADDED Requirements

### Requirement: SENTINEL Design Tokens

The system SHALL expose the SENTINEL color, typography, spacing, radius, and shadow values as CSS custom properties (`--*` tokens) defined on `:root`. All UI components in AgentVerse SHALL consume tokens through these CSS variables; SHALL NOT hard-code hex colors or font stacks; and SHALL NOT introduce additional palette colors without a corresponding token.

The minimum token set for v1 includes:

- Color: `--void`, `--panel`, `--card`, `--cyan`, `--amber`, `--threat`, `--ops`, `--text-primary`, `--text-muted`, `--text-dim`, `--border`, `--border-accent`.
- Typography (defaults per master spec §14.7 user-approved override): `--font-display` (JetBrains Mono), `--font-body` (JetBrains Mono), `--font-mono` (JetBrains Mono). Inter SHALL remain available as a built-in alternative selectable through Settings → Appearance.
- Radius: `--radius-card`, `--radius-button`, `--radius-badge`.
- Spacing scale: `--space-1` … `--space-8`.

Color, radius, and spacing token values SHALL match the AgentVerse Master Specification §3.2 and §3.4 exactly. Typography defaults SHALL follow §14.7 (mainframe/tmux aesthetic by default), with §3.3 sizes/weights retained as font-family-agnostic typography rules.

#### Scenario: Component uses a token not a literal

- **WHEN** an audit examines any component in `src/design-system/` or any consumer thereof
- **THEN** there are zero raw hex color literals or `rgb(...)` values in the source — all colors flow through `var(--*)` tokens
- **AND** there are zero font-family declarations referencing a typeface other than via `var(--font-*)`

#### Scenario: Tokens load before any styled element renders

- **WHEN** the SPA mounts in a fresh browser
- **THEN** `:root` exposes every required token as a non-empty CSS custom property
- **AND** there is no flash of unstyled content using browser defaults

### Requirement: Core Component Set

The Design System SHALL provide the following base components, each consuming SENTINEL tokens, with documented prop contracts and accessible defaults:

- `Card` — panel container with SENTINEL border radius and translucent panel background.
- `Button` — primary, secondary, and ghost variants. Always includes a focus ring meeting WCAG 2.1 AA contrast against the page background.
- `Badge` — pill-shaped status indicator. Status variants must include `idle`, `processing`, `completed`, `waiting_user_answer`, `error`, each mapped to its SENTINEL color from §3.5 of the master spec.
- `NavBar` — fixed top navigation. Translucent backdrop-blurred background.
- `StatusBadge` — composite of a Badge and a glyph (`○`, `●`, `✓`, `⚠`, `✕`) per master spec §3.5.
- `FormField` — label + input + helper-text + error-text composition with proper `<label htmlFor>` wiring.

#### Scenario: Buttons are keyboard-focusable with a visible ring

- **WHEN** a user presses Tab onto a `Button`
- **THEN** a focus indicator is visible against the SENTINEL `--void` background with at least a 3:1 contrast ratio
- **AND** activating the button via Enter or Space invokes its handler

#### Scenario: StatusBadge maps each status to the correct color

- **WHEN** `<StatusBadge status="error" />` renders
- **THEN** the rendered element has its border or fill derived from `var(--threat)` and includes the `✕` glyph

### Requirement: Locked-Early Policy

After Milestone 1 Week 1, the design system tokens, base components, and global stylesheet SHALL be considered locked. Any modification SHALL require sign-off from the supervisor agent (master spec §14). Pull requests touching `src/design-system/` from any non-supervisor owner SHALL fail review automatically.

#### Scenario: Unauthorized design-system change is blocked

- **WHEN** a frontend developer agent submits a PR that modifies `src/design-system/tokens.css` without a supervisor approval
- **THEN** CI rejects the PR with a message identifying the locked-files policy

### Requirement: User-Configurable Fonts via Settings

The system SHALL allow users to override the default font for body text, headings, and terminal/code via the Settings → Appearance tab (per master spec §14.7). The Settings store SHALL persist the user's selections and the design system SHALL apply them by overriding the corresponding `--font-*` tokens at `:root`. Built-in font choices SHALL include at minimum: JetBrains Mono (default), Inter, and the system UI default. Users SHALL also be able to provide a custom font-family string, with a documented warning that they are responsible for installing the typeface on their system.

#### Scenario: User changes UI body font

- **WHEN** the user selects "Inter" for "UI Body" in Settings → Appearance and presses Save
- **THEN** within one render cycle every component using `var(--font-body)` displays in Inter
- **AND** the choice persists across page reloads

#### Scenario: Custom font with missing typeface degrades safely

- **WHEN** the user enters a custom font-family value referencing a typeface that is not installed
- **THEN** the browser falls back to the system stack and the UI remains readable
- **AND** Settings shows an inline warning that the typeface could not be loaded

### Requirement: Background and Animation Effects

The system SHALL implement the SENTINEL ambient effects defined in master spec §3.6–3.7 as opt-in CSS classes (not always-on globals): scanline overlay, scan-sweep line, and KPI glow. Animations SHALL respect `prefers-reduced-motion: reduce` and SHALL disable continuous animations when that preference is set.

#### Scenario: Reduced-motion users see no scan sweep

- **WHEN** the operating system or browser advertises `prefers-reduced-motion: reduce`
- **THEN** the scan-sweep line is not rendered or is held at a static position
- **AND** terminal cursor blink is reduced to a non-animated cursor
