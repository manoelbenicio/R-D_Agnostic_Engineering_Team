## ADDED Requirements

### Requirement: Tab Bar With Status Badges

The system SHALL render a tab bar across the top of the Orchestrator route showing one tab per terminal in the current session per master spec §6.6 and §8.4. Each tab SHALL display the agent name, a `StatusBadge` reflecting the current `TerminalStatus`, and a close affordance. Selecting a tab SHALL focus its terminal in the main view. A trailing `+` tab SHALL allow adding a new terminal (delegating to the Reconciler / Canvas Builder).

The status badge SHALL update within one polling interval (3 s) of a CAO status change per master spec §9.

#### Scenario: Tab badge updates when terminal transitions to processing

- **WHEN** a terminal transitions from `idle` to `processing` on the CAO server
- **THEN** within one polling interval the corresponding tab's badge displays the `processing` style

#### Scenario: Selecting a tab focuses its terminal

- **WHEN** the user clicks a tab in the bar
- **THEN** the corresponding `terminal-streaming` instance is the visible focused view in the main region

### Requirement: Grid View

The system SHALL provide a Grid View that arranges all terminals in the current session as a responsive 2×3 or 3×4 layout per master spec §6.6. Each cell SHALL display: agent name, provider badge, `StatusBadge`, and a mini-terminal preview (40×15 character read-only). Clicking a cell SHALL expand that terminal to the focused tab view. The grid SHALL update reactively when terminals are added or removed.

#### Scenario: Grid renders one cell per terminal

- **WHEN** the session has 5 terminals
- **THEN** the grid renders 5 cells in a responsive layout (2×3 with 1 empty cell or 3×2)

#### Scenario: Cell click expands to focused view

- **WHEN** the user clicks a grid cell
- **THEN** the view transitions to the tab/focused view with that terminal selected

### Requirement: Mini-Terminal Read-Only Preview

The mini-terminal in each grid cell SHALL render a read-only 40×15 character preview of the terminal output using a separate xterm.js instance with WebGL. Mini-terminals SHALL share the same WebSocket subscription as their corresponding focused terminal (one connection per terminal id, multiple consumers via a fan-out helper) to avoid opening duplicate connections.

#### Scenario: One WebSocket per terminal id

- **WHEN** Grid View renders 5 cells and the focused tab also displays one of those terminals
- **THEN** the browser holds 5 WebSocket connections total (not 6 — the focused tab shares the connection with its mini-terminal)

#### Scenario: Mini-terminal is read-only

- **WHEN** the user attempts to type while focus is in a mini-terminal cell
- **THEN** keystrokes are ignored and the input is not forwarded to CAO

### Requirement: Full-Screen Mode

The system SHALL allow a single terminal to occupy the entire viewport via a Full-Screen toggle per master spec §6.6. Full-Screen mode SHALL hide the tab bar, palette, and any overlays and resize xterm.js to the available viewport. Pressing `Escape` SHALL exit Full-Screen back to the prior view (tab or grid).

#### Scenario: Escape exits full-screen

- **WHEN** the user is in Full-Screen mode and presses `Escape`
- **THEN** the prior view (tab bar or grid) is restored and the terminal continues streaming uninterrupted

### Requirement: Per-Terminal Controls

Every terminal surface (focused tab, grid cell, full-screen) SHALL provide per-terminal controls per master spec §8.4: working directory display (read), Send Message input field (writes via `POST /terminals/{id}/input`), Inbox viewer (reads via `GET /terminals/{id}/inbox/messages`), and Kill button (`DELETE /terminals/{id}` with confirmation). The controls SHALL be present on all surfaces but MAY be collapsed into a menu on grid cells and full-screen mode for space.

#### Scenario: Send message via input field

- **WHEN** the user types in the per-terminal input field and presses Enter
- **THEN** the system issues `POST /terminals/{id}/input` with the message
- **AND** the input field clears for the next message

#### Scenario: Kill requires confirmation

- **WHEN** the user clicks the Kill button on a terminal
- **THEN** a confirmation modal appears before any `DELETE /terminals/{id}` is issued
