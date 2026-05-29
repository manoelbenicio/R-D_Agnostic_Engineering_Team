## ADDED Requirements

### Requirement: Runtime Command Vocabulary

The system SHALL recognize the runtime command vocabulary in master spec §5.8 in both pt-BR and en-US. The minimum required commands SHALL be:

- `kill` — `DELETE /terminals/{id}` (with target by terminal id, agent name, or role)
- `pause` — `POST /terminals/{id}/input` with a pause sentinel
- `focus` — client-side navigation to the named terminal
- `status` — read all agent statuses via `GET /sessions/{name}/terminals`
- `deploy` — invoke the Reconciler on the current canvas
- `stop_all` — `DELETE /sessions/{name}` for the current session
- `cost` — navigate to FinOps and show the current session's cost summary
- `add_node` — client-side canvas edit (delegated to Canvas Builder)
- `connect` — client-side canvas edit (add edge; delegated to Canvas Builder)

Commands not in the recognized set SHALL be passed through to `speech-to-canvas` for NLU intent extraction (canvas authoring), not silently dropped.

#### Scenario: "Matar o revisor" deletes the reviewer terminal

- **WHEN** the user is on a deployed canvas and says "Matar o revisor"
- **THEN** the matcher returns `{ action: "kill", target: { type: "role", value: "reviewer" } }`
- **AND** the system issues `DELETE /terminals/{id}` for the reviewer's terminal id from `terminal_map`

#### Scenario: Unrecognized command falls through to NLU

- **WHEN** the user says "Cria um novo canvas com 3 desenvolvedores" while on a deployed canvas
- **THEN** the runtime matcher returns null and the transcript is forwarded to `speech-to-canvas` for canvas-authoring intent extraction

### Requirement: Command Matcher Implementation

The matcher SHALL be implemented as a regex + keyword detection pipeline (master spec §5.8) — no LLM call is required for runtime commands, in order to keep latency below 100 ms. The matcher SHALL be locale-aware: pt-BR and en-US patterns SHALL be evaluated independently and SHALL share the same return type (`RuntimeCommand`).

#### Scenario: Matcher latency under 100 ms

- **WHEN** the matcher is invoked with a 50-character transcript
- **THEN** the function returns within 100 ms (excluding the underlying CAO HTTP call)

### Requirement: Confirm Before Destructive Actions

The system SHALL require a confirmation step for destructive commands (`kill`, `stop_all`) before issuing the CAO call. Confirmation SHALL be a small modal that displays the action, the target, and two buttons: **Confirm** and **Cancel**. The modal SHALL auto-focus Cancel for safety.

#### Scenario: "Stop everything" requires confirmation

- **WHEN** the user says "Parar tudo" on a deployed canvas with 4 active terminals
- **THEN** a modal appears reading "Confirm stop all? This will kill 4 terminals."
- **AND** Cancel has keyboard focus by default; only Confirm issues the `DELETE /sessions/{name}` call

### Requirement: Voice-Activated Deploy

The `deploy` runtime command SHALL invoke the Reconciler on the currently open canvas if (and only if) the canvas is in `draft` status with all pre-deploy validation passing. If validation fails, the system SHALL surface the same disabled-state reason that the Deploy button shows in `canvas-builder`.

#### Scenario: Voice deploy on a valid draft

- **WHEN** the user says "Deploy" while a valid draft canvas is open
- **THEN** the Reconciler is invoked exactly as if the user had clicked the Deploy button

#### Scenario: Voice deploy on an invalid canvas

- **WHEN** the user says "Deploy" while a canvas missing an entry-point is open
- **THEN** a toast appears with the disabled-state reason and no Reconciler call is made
