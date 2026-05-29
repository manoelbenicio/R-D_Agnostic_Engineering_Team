## ADDED Requirements

### Requirement: KPI Row

The Dashboard SHALL render a KPI row at the top of the page with the four metrics defined in master spec §8.2:

- **Fleet Status** — count of active terminals across all sessions, sourced from `GET /sessions` polled at 5 s.
- **Cost / MTD** — month-to-date cost estimate, computed by `finops-tier1`. Displays the ⚠️ "rough estimate" label per `finops-tier1`.
- **Budget Utilization** — `cost / budget × 100%`, where budget is configured in Settings.
- **Threats** — count of terminals with status `error` across all sessions.

Each KPI SHALL use SENTINEL Card styling and SHALL display in tabular-nums per master spec §3.3.

#### Scenario: Fleet count updates on session change

- **WHEN** a new terminal is added to a session
- **THEN** within one polling interval the Fleet Status KPI increments

#### Scenario: Threats KPI counts errored terminals

- **WHEN** two terminals across two sessions are in status `error`
- **THEN** the Threats KPI displays "2"

### Requirement: Cost-By-Provider Bar Chart

The Dashboard SHALL render a horizontal or vertical bar chart breaking down the current-period cost by provider, sourced from `finops-tier1`. The chart SHALL use SENTINEL color tokens for bars, SHALL include the ⚠️ label by association with `finops-tier1`, and SHALL update when the underlying cost calculation re-runs (typically every 30 s).

#### Scenario: Chart shows providers with non-zero usage

- **WHEN** only Anthropic and Codex have active terminals during the current MTD window
- **THEN** the chart shows two bars (Anthropic, Codex) and zero bars for unused providers

### Requirement: Fleet Status Donut

The Dashboard SHALL render a donut chart breaking down fleet members by status: `active` (idle + processing), `error`, and `offline` (terminals known to past but no longer present). Hover SHALL show counts for each segment.

#### Scenario: Donut sums to total fleet size

- **WHEN** the fleet has 4 active, 1 error, 0 offline terminals
- **THEN** the donut shows three segments summing to 5

### Requirement: Activity Feed

The Dashboard SHALL render an activity feed showing recent inbox messages and session lifecycle events (terminal created, terminal killed, session deleted). The feed SHALL update reactively from `GET /terminals/{id}/inbox/messages` (5 s poll) and from session-list polling. Per master spec v4.2 §12, the feed SHALL NOT impose an application-level retention cap — entries accumulate for the lifetime of the browser session and are bounded only by the browser's memory. Entries SHALL be ordered newest-first. Older entries SHALL remain reachable via in-feed scroll.

The feed SHALL provide a manual "Clear" affordance that empties the in-memory buffer; clearing SHALL NOT affect any CAO-side data.

#### Scenario: New inbox message appears in feed

- **WHEN** an inbox message is delivered between two terminals in any active session
- **THEN** within one polling interval the feed prepends a new entry with sender, receiver, and message preview

#### Scenario: Older entries remain reachable

- **WHEN** the feed has accumulated 500 entries over a long-running session
- **THEN** all 500 entries are present and reachable via scroll
- **AND** no application-level eviction has occurred

#### Scenario: Manual Clear empties the buffer

- **WHEN** the user clicks "Clear" on the activity feed
- **THEN** the feed becomes empty
- **AND** subsequent polling continues to add new entries normally

### Requirement: Terminal Preview Card

The Dashboard SHALL include a preview card showing the most recent output from a user-selectable "watched" terminal in SENTINEL command-output styling. The preview SHALL use a read-only mini-terminal (sharing the WebSocket fan-out from `terminal-grid`) and SHALL allow the user to click to navigate to the full terminal view.

#### Scenario: Click preview navigates to full terminal

- **WHEN** the user clicks the terminal preview card
- **THEN** the route navigates to `/canvas/:id/terminal/:terminalId` for the watched terminal
