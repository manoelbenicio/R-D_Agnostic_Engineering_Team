## ADDED Requirements

### Requirement: Health Page

The system SHALL provide a Health page at `/health` showing the status of every checked component. The page SHALL include three sections per master spec §8.9 plus the v4.1 browser checks:

**Server health:**
- CAO Server — `GET /health` returning `status === "ok"`
- tmux Server — `GET /sessions` returning a non-error response
- Each provider (CAO-managed) — `GET /agents/providers` showing `installed === true`

**Provider validations** (BYOK):
- Each provider configured in `api-key-management` and its current validation status (`set`, `unset`, `invalid`)

**Browser capabilities:**
- WebGL2 availability (required for production terminal rendering per `terminal-streaming`)
- IndexedDB availability (required for persistence)
- Microphone permission status (required for `speech-to-canvas`; user-grantable)

Each row SHALL show its component name, status badge (`ok`, `warning`, `error`), and a one-line explanation. Failing checks SHALL include a "Fix" affordance pointing to the relevant settings page or external docs.

#### Scenario: Health page reflects CAO outage

- **WHEN** the CAO server is unreachable
- **THEN** the CAO Server row shows status `error` with the explanation "Cannot reach CAO at <CAO_BASE_URL>"
- **AND** the Fix affordance opens Settings → General to allow editing the CAO base URL

#### Scenario: WebGL2 unavailable shows error

- **WHEN** the browser does not expose WebGL2
- **THEN** the WebGL2 row shows status `error` with the explanation "WebGL2 is required for the Terminal View"
- **AND** the Fix affordance links to docs explaining how to enable hardware acceleration

### Requirement: First-Run Wizard

On first visit (detected via the absence of any saved canvas, validated provider, or persisted settings), the system SHALL present a first-run wizard before routing to `/`. The wizard SHALL guide the user through three steps:

1. **Verify CAO** — calls `getHealth()`. Pass = green; fail = display the configured CAO base URL with an Edit affordance and a "Skip and continue" link.
2. **Configure at least one provider** — embedded mini Settings panel that asks for an API key, validates it, and persists it.
3. **Pick a starting point** — Templates picker (10 entries from `canvas-templates`) plus a "Start Blank" link.

The wizard SHALL be skippable at any step. Skipping at step 2 SHALL still let the user proceed but with the Canvas Builder in the no-provider empty state. Skipping at step 3 SHALL route to `/`.

#### Scenario: First visit triggers wizard

- **WHEN** a user opens AgentVerse with empty IndexedDB and no settings
- **THEN** the wizard appears before the user reaches `/`

#### Scenario: Subsequent visits skip wizard

- **WHEN** a user has at least one validated provider and at least one canvas in IndexedDB
- **THEN** the wizard does not appear on subsequent visits — the user goes directly to `/`

#### Scenario: Provider validation step

- **WHEN** the user enters an Anthropic key in step 2 of the wizard
- **THEN** the same validation logic from `api-key-management` runs and a green check appears on success

### Requirement: Health-Driven NavBar Indicator

The NavBar's CAO health pill (defined in `agentverse-shell`) SHALL be driven by the same health-poll store that the Health page consumes. A click on the NavBar pill SHALL navigate to `/health`.

#### Scenario: Clicking NavBar pill navigates to Health

- **WHEN** the user clicks the NavBar's CAO health pill
- **THEN** the route changes to `/health` and the corresponding row scrolls into view

### Requirement: Microphone Permission Prompt

The Health page SHALL include a "Test Microphone" affordance that requests `getUserMedia` and reports the result. This SHALL allow the user to proactively grant or deny microphone permission outside the voice-input flow.

#### Scenario: Test Microphone updates permission row

- **WHEN** the user clicks "Test Microphone" and grants permission
- **THEN** the Microphone Permission row updates to status `ok` within 1 second
