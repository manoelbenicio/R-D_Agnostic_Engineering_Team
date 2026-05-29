## ADDED Requirements

### Requirement: Flow List

The system SHALL render a flow list at `/flows` showing every CAO flow returned by `GET /flows`. Each entry SHALL display: name, schedule (human-readable via `cronstrue` plus the raw cron expression), agent profile, provider, enabled state, last run, and next run. Each row SHALL provide actions: Run Now, Enable/Disable, Edit, Delete. The list SHALL refresh on a 15 s poll per master spec §9.

#### Scenario: List displays human-readable schedule

- **WHEN** a flow has cron `0 9 * * 1-5`
- **THEN** the list row shows "At 09:00 on every day-of-week from Monday through Friday" alongside the raw expression

### Requirement: Flow Create / Edit Form

The system SHALL provide a create/edit form with the fields required by `Flow` (master spec §7.3): `name`, `schedule`, `agent_profile` (selector populated from `GET /agents/profiles`), `provider` (selector gated by `api-key-management`), `prompt_template` (Monaco-style multiline editor), and `enabled` toggle. Submitting SHALL POST to `/flows` (create) or PUT/POST equivalent for edit (per CAO API). Validation errors from CAO SHALL be surfaced verbatim.

The schedule field SHALL provide a quick-pick UI for common patterns (every-N-minutes, hourly, daily-at-time, weekdays-at-time, weekly) and a raw cron input for power users.

#### Scenario: Quick-pick fills the cron expression

- **WHEN** the user picks "Weekdays at 9:00" in the schedule quick-pick
- **THEN** the schedule field is filled with `0 9 * * 1-5` and the human-readable text "At 09:00 on every day-of-week from Monday through Friday" appears below

#### Scenario: Invalid cron is rejected before submit

- **WHEN** the user types `99 * * * *` in the raw cron field
- **THEN** the form shows an inline validation error "Invalid cron expression" and the Save button is disabled

### Requirement: Run Now Action

Each flow row SHALL expose a Run Now button that issues `POST /flows/{name}/run`. The UI SHALL show a non-blocking toast confirming the manual trigger and the next polling cycle SHALL reflect any resulting session in the Dashboard fleet count.

#### Scenario: Run Now confirms via toast

- **WHEN** the user clicks Run Now on an enabled flow
- **THEN** a success toast appears within 1 second of the CAO 200 response
- **AND** if the response is non-200, an error toast surfaces the CAO error verbatim

### Requirement: Enable/Disable Toggle

Each flow row SHALL expose a toggle that issues `POST /flows/{name}/enable` or `POST /flows/{name}/disable`. The toggle SHALL update optimistically in the UI and revert if the CAO call fails.

#### Scenario: Disable persists after refresh

- **WHEN** the user toggles a flow from Enabled to Disabled and the CAO call returns 200
- **THEN** on the next poll cycle the list confirms the flow's `enabled: false` state

### Requirement: Conditional Gating Display

When a flow has a gating script attached (per master spec §7.11), the flow detail SHALL display the gating-script status (last evaluation result if known) and a note "Conditional execution — only runs when gating script returns success."

#### Scenario: Gated flow shows conditional badge

- **WHEN** a flow has a gating script reference in its definition
- **THEN** the flow row displays a "Conditional" badge with hover text describing the gating behavior
