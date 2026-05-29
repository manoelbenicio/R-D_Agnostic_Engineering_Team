## ADDED Requirements

### Requirement: Single-Terminal PTY Rendering

The system SHALL render a single tmux PTY terminal in the browser using xterm.js. The terminal SHALL load the WebGL addon (`@xterm/addon-webgl`) as MANDATORY for production builds; the WebGL renderer SHALL be initialized before the first byte is written. The xterm.js configuration SHALL match the master spec §6.3 zero-lag profile exactly: `smoothScrollDuration: 0`, `scrollback: 10000`, `fontFamily` of JetBrains Mono / Fira Code / Cascadia Code, `fontSize: 13`, `lineHeight: 1.2`, `cursorBlink: true`, `cursorStyle: "bar"`, `fastScrollSensitivity: 5`, `allowProposedApi: true`, and the SENTINEL theme.

#### Scenario: WebGL addon is active at first write

- **WHEN** the Terminal View mounts and connects to a WebSocket
- **THEN** the xterm.js instance has the WebGL addon attached before the first `terminal.write` call
- **AND** if WebGL initialization fails in production, the user is shown an error explaining that WebGL is required, with no Canvas2D fallback

#### Scenario: SENTINEL theme is applied

- **WHEN** the Terminal View renders
- **THEN** the xterm canvas background matches `--void`, the cursor color matches `--cyan`, and the selection background uses the documented translucent cyan

### Requirement: Binary WebSocket Frame Handling

The Terminal View SHALL connect to `ws://<cao-host>/terminals/{id}/ws` via the helper from `cao-integration`, set `binaryType = "arraybuffer"` before the connection opens, and pipe every binary frame directly to `terminal.write(new Uint8Array(event.data))` with no string conversion in the hot path. JSON text frames SHALL be the ONLY format used for outbound input and resize messages, per master spec §6.2.

#### Scenario: Binary input writes directly without conversion

- **WHEN** the WebSocket receives a binary message
- **THEN** the handler invokes `terminal.write(new Uint8Array(event.data))` directly
- **AND** there are no `String.fromCharCode`, `TextDecoder`, or `Blob` conversions on the binary input path

#### Scenario: Outbound input uses JSON text frame

- **WHEN** the user types `ls -la` and presses Enter in the terminal
- **THEN** the WebSocket sends a single text frame matching `{"type":"input","data":"ls -la\n"}`

### Requirement: Resize Handling

The Terminal View SHALL listen for container size changes (via `ResizeObserver`) and SHALL recalculate cell dimensions using the `FitAddon`. Resize messages to the server SHALL be debounced at 100 ms to avoid flooding during window drag. Initial dimensions SHALL be 220×50 to match CAO's default tmux pane size.

#### Scenario: Resize sends a single message after debounce

- **WHEN** the user drags the window edge for 500 ms continuously
- **THEN** at most one resize WebSocket message is sent for that drag, sent shortly after the drag ends
- **AND** the resize message has the form `{"type":"resize","rows":<n>,"cols":<m>}`

#### Scenario: Initial size matches CAO default

- **WHEN** the terminal first mounts before any layout change
- **THEN** xterm.js reports `cols=220, rows=50` (or the closest the container allows, with no resize message sent if the container exactly fits these defaults)

### Requirement: Connection Lifecycle and Reconnection

The Terminal View SHALL handle the following WebSocket lifecycle events:

- On `open`: render xterm and clear any "connecting…" placeholder.
- On `close` with code 4003: render a permanent error banner explaining the IP allowlist; do NOT reconnect.
- On `close` with code 4004: render an error banner identifying that the terminal no longer exists; do NOT reconnect.
- On `close` with any other code or `error`: schedule a reconnect with exponential backoff starting at 500 ms, capped at 30 s, with at least ±20 % jitter, and continue until the user navigates away or explicitly disables reconnection.

The view SHALL show the connection state in a small status pill above the terminal: `connecting`, `connected`, `reconnecting`, `terminated`.

#### Scenario: Backoff progression on repeated failures

- **WHEN** the WebSocket fails to open three consecutive times
- **THEN** the inter-attempt delays are approximately 0.5 s → 1 s → 2 s (with jitter), bounded by the 30 s cap
- **AND** the status pill shows `reconnecting` between attempts

#### Scenario: 4004 does not trigger reconnect

- **WHEN** the server closes the socket with code 4004
- **THEN** no reconnection is scheduled
- **AND** the status pill shows `terminated`

### Requirement: Required xterm Addons

The Terminal View SHALL load these addons at mount time, in this order: `WebglAddon`, `FitAddon`, `WebLinksAddon`, `SearchAddon`, `Unicode11Addon`. All addons SHALL be installed at the master-spec-pinned versions of the `@xterm/*` packages.

#### Scenario: All required addons are loaded

- **WHEN** the Terminal View mounts and inspection is performed on the xterm instance
- **THEN** each of the five addons is registered exactly once
- **AND** removing any one of them breaks a documented feature (resize, link clicking, in-terminal search, GPU rendering, or Unicode 11 width tables)

### Requirement: Multi-Terminal Coexistence

The Terminal View SHALL be designed as a primitive that can be instantiated multiple times on the same page (consumed by the `terminal-grid` capability). Each instance SHALL own a single WebSocket connection. The implementation SHALL ensure that two instances rendered side-by-side render correctly without sharing xterm.js state, addon registrations, or theme overrides. The `terminal-grid` capability SHALL be responsible for layout and coordination across multiple instances; this capability SHALL provide only the per-instance contract.

#### Scenario: Two instances on one page

- **WHEN** the page renders two `<TerminalView />` instances bound to different terminal ids
- **THEN** each opens its own WebSocket and writes to its own xterm instance
- **AND** typing into one does not appear in the other

#### Scenario: Per-instance theme

- **WHEN** an instance is rendered with a theme override prop
- **THEN** only that instance's xterm uses the override; sibling instances retain the default SENTINEL theme
