## ADDED Requirements

### Requirement: Typed CAO HTTP Client

The system SHALL provide a typed TypeScript client that wraps the CAO REST API surface required for v1. The client SHALL expose typed request and response models that mirror the CAO data models (`Terminal`, `AgentProfile`, `Session`, `TerminalStatus`, `ProviderType`, `Flow`, `InboxMessage`) from master spec §7.3. No other module in the application SHALL issue raw `fetch` calls to the CAO server — all CAO traffic flows through this client.

The v1 surface SHALL include the following operations, grouped by area (per master spec §7.2):

**Health (1):**
- `getHealth()` → `{ status: "ok" }`

**Profiles (4):**
- `listProfiles()` → `AgentProfile[]`
- `getProfile(name)` → `AgentProfile`
- `installProfile(profileMarkdown)` → `AgentProfile`
- `listProviders()` → `{ name, installed: boolean }[]`

**Sessions (6):**
- `createSession({ profile, working_directory })` → `Session`
- `listSessions()` → `Session[]`
- `getSession(name)` → `Session`
- `deleteSession(name)` → void
- `addTerminalToSession(sessionName, { profile, working_directory })` → `Terminal`
- `listTerminalsInSession(sessionName)` → `Terminal[]`

**Terminals (7):**
- `getTerminal(id)` → `Terminal`
- `getTerminalOutput(id, mode: "full" | "tail" | "visible")` → `string`
- `getTerminalWorkingDirectory(id)` → `string`
- `getTerminalMemoryContext(id)` → `string`
- `sendTerminalInput(id, message)` → void
- `exitTerminal(id)` → void
- `deleteTerminal(id)` → void

**Inbox (2):**
- `sendInboxMessage(terminalId, message)` → `InboxMessage`
- `listInboxMessages(terminalId, { limit?, status? })` → `InboxMessage[]`

**Flows (7):**
- `listFlows()` → `Flow[]`
- `getFlow(name)` → `Flow`
- `createFlow(flow)` → `Flow`
- `deleteFlow(name)` → void
- `enableFlow(name)` → void
- `disableFlow(name)` → void
- `runFlow(name)` → void

**Settings (2):**
- `getAgentDirs()` → `{ dirs: string[] }`
- `setAgentDirs(dirs)` → `{ dirs: string[] }`

**Skills (1):**
- `getSkill(name)` → `string`

The client SHALL expose exactly this set; adding endpoints SHALL require a change proposal so the CAO surface dependency stays intentional.

#### Scenario: Client surfaces typed errors

- **WHEN** the CAO server returns HTTP 500 from `createSession`
- **THEN** the client throws (or rejects with) a structured `CaoApiError` carrying status code, endpoint, and the response body
- **AND** the error is distinguishable from a network failure (`CaoNetworkError`)

#### Scenario: Adding an endpoint requires a change proposal

- **WHEN** a developer attempts to add an undocumented endpoint to the client
- **THEN** code review rejects the PR pending an OpenSpec change that updates this requirement

### Requirement: Configurable CAO Base URL

The CAO base URL SHALL be read from an environment variable `VITE_CAO_BASE_URL`, defaulting to `http://127.0.0.1:9889` when unset. The base URL value SHALL be evaluated once at SPA bootstrap; runtime changes are not supported in Milestone 1.

#### Scenario: Default base URL works for local dev

- **WHEN** `VITE_CAO_BASE_URL` is not set and CAO is running on the default port
- **THEN** `getHealth()` succeeds without any explicit configuration

#### Scenario: Custom base URL routes all traffic

- **WHEN** `VITE_CAO_BASE_URL=http://example.local:9889` is set at build time
- **THEN** every client method sends its request to `http://example.local:9889/...`

### Requirement: Health Polling

The system SHALL poll `GET /health` at a 10-second interval while the SPA is active, expose the latest result as observable application state, and surface a non-blocking banner when health transitions from `ok` to error or unreachable. Polling SHALL pause when the browser tab is hidden (Page Visibility API) and resume on visibility return.

#### Scenario: Health transition from ok to unreachable shows banner

- **WHEN** the app has been showing health=ok and the next poll fails with a network error
- **THEN** a visible banner appears within at most one polling interval indicating CAO is unreachable
- **AND** the banner is dismissable but reappears on the next failed poll

#### Scenario: Polling pauses on hidden tab

- **WHEN** the user switches to a different browser tab for one minute
- **THEN** zero health requests are sent during that minute
- **AND** within one polling interval after the tab regains visibility, polling resumes

### Requirement: WebSocket PTY Endpoint Helper

The client SHALL expose a `connectTerminalSocket(terminalId, { onBinary, onClose, onError })` helper that constructs the WebSocket URL from the configured base URL (with `ws://` or `wss://` chosen by the protocol of the base URL), sets `binaryType = "arraybuffer"`, and surfaces the documented close codes (4003 IP not allowed, 4004 terminal not found) as typed close reasons. Connection lifecycle ownership and reconnection are the responsibility of the `terminal-streaming` capability — this helper provides only the typed primitive.

#### Scenario: Helper produces correct WebSocket URL

- **WHEN** the base URL is `http://127.0.0.1:9889` and the terminal id is `abcd1234`
- **THEN** the helper opens a WebSocket to `ws://127.0.0.1:9889/terminals/abcd1234/ws`
- **AND** `binaryType` is `arraybuffer` immediately after construction

#### Scenario: 4004 close code is surfaced as TerminalNotFound

- **WHEN** the server closes the socket with code 4004
- **THEN** `onClose` is invoked with a typed reason `TerminalNotFound` (not just the numeric code)

### Requirement: CORS Configuration Contract

The system SHALL document, in `docs/cao-cors.md` shipped with this capability, the exact `CAO_CORS_ORIGINS`, `CAO_ALLOWED_HOSTS`, and `CAO_WS_ALLOWED_CLIENTS` values that the CAO server must be configured with to accept requests from local AgentVerse dev. Setup of the CAO server itself remains the operator's responsibility.

#### Scenario: Documentation is present and current

- **WHEN** a developer follows the documented CORS configuration verbatim and starts CAO on the local default port
- **THEN** an AgentVerse dev server running on `http://localhost:5173` can issue REST and WebSocket requests successfully
