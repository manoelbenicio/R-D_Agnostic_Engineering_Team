## ADDED Requirements

### Requirement: Chat View as Alternate Surface

The system SHALL provide a Chat View as an alternate surface to the Terminal Grid for any deployed canvas, per master spec §2.2 and §6.6. The Chat View SHALL render parsed agent output as chat bubbles (one bubble per agent message or output chunk) — no terminal rendering. Users SHALL switch between Terminal Grid and Chat View via a toggle in the Orchestrator toolbar; the choice SHALL persist per canvas in the user's settings.

The Chat View SHALL be the default surface on viewports narrower than 1024 px (mobile and small tablets), where the Terminal Grid is not usable.

#### Scenario: Mobile viewport defaults to Chat View

- **WHEN** the user opens a deployed canvas on a viewport ≤ 768 px wide
- **THEN** the default surface is Chat View, not Terminal Grid

#### Scenario: Toggle persists per canvas

- **WHEN** the user toggles to Chat View on canvas A and reloads the page
- **THEN** canvas A reopens in Chat View; other canvases retain their previously chosen surface

### Requirement: Output Parsing to Chat Bubbles

The system SHALL parse PTY output from the WebSocket stream into chat-style chunks. Parsing SHALL strip ANSI/VT100 escape sequences, group lines into messages by detecting agent-prompt boundaries and tool-call markers, and attribute each chunk to its terminal id. Each parsed message SHALL be rendered as a SENTINEL `Card`-styled bubble with: agent display_name, agent provider badge, timestamp, and content.

The parser SHALL handle partial messages — incoming bytes that do not yet form a complete message SHALL be buffered and rendered as a "typing" state on the most recent bubble until the buffer flushes.

#### Scenario: ANSI escapes are stripped

- **WHEN** PTY output contains the bytes `\x1b[31mERROR\x1b[0m`
- **THEN** the rendered bubble shows the text "ERROR" in the SENTINEL `--threat` color, not the raw escape sequence

#### Scenario: Partial output renders as typing

- **WHEN** the WebSocket has delivered the first half of an agent message
- **THEN** a bubble is rendered with the partial content and a typing indicator
- **AND** when the message completes the bubble updates and the typing indicator clears

### Requirement: Inline Send-Message Composer

The Chat View SHALL include an inline composer at the bottom of the conversation per active terminal. The composer SHALL post user input via `POST /terminals/{id}/input` (master spec §8.4). Submission SHALL be triggered by Enter (Shift+Enter for newline). The composer SHALL display the active terminal's display_name and provider badge above the input.

#### Scenario: Composer posts input to the active terminal

- **WHEN** the user types into the composer and presses Enter
- **THEN** `POST /terminals/{active_terminal_id}/input` is issued with the message
- **AND** the composer clears

### Requirement: Touch and Mobile Affordances

On touch devices the Chat View SHALL: (a) disable hover-only interactions, (b) provide a swipe-left gesture on a bubble to reveal per-message actions (Copy, Reply), and (c) ensure the composer remains visible above the on-screen keyboard via `100dvh` sizing rather than `100vh`.

#### Scenario: Composer above keyboard on iOS

- **WHEN** the user focuses the composer on iOS Safari
- **THEN** the composer remains visible above the on-screen keyboard with no layout overflow
