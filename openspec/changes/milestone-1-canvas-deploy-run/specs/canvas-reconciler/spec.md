## ADDED Requirements

### Requirement: Canvas-to-CAO Translation

The Reconciler SHALL accept a `CanvasDocument` and translate it into a sequence of CAO HTTP calls that materialize the canvas as a running CAO session with one tmux terminal per node. The translation SHALL execute in the following order, and each step SHALL be observable as a distinct phase in the deploy state:

1. Generate one agent profile markdown body per node (YAML frontmatter + `system_prompt` body).
2. Install each profile via `POST /agents/profiles/install`.
3. Create the CAO session with the entry-point profile via `POST /sessions`.
4. Add a terminal per non-entry-point node via `POST /sessions/{name}/terminals`.
5. Record the resulting CAO `session_name` and per-node terminal ids in `CanvasDocument.deploy_state.terminal_map`.

The Reconciler SHALL NOT issue arbitrary CAO calls beyond this sequence in Milestone 1. Validation Proxy registration (master spec §4.7) is explicitly deferred to Milestone 2.

#### Scenario: Three-node canvas deploys in correct order

- **WHEN** a canvas with one Supervisor (entry point) and two Developer nodes is deployed
- **THEN** exactly three profiles are installed, then one session is created using the supervisor profile, then two terminals are added for the developers
- **AND** all five HTTP calls are made in that order, and `terminal_map` records all three node→terminal mappings

#### Scenario: No CAO calls for invalid canvas

- **WHEN** the canvas fails pre-flight validation (e.g., zero entry points)
- **THEN** the Reconciler refuses the deploy and zero CAO HTTP requests are sent

### Requirement: Deploy State Machine

The Reconciler SHALL maintain `CanvasDocument.deploy_state.status` according to the following state machine, with transitions persisted to the document immediately as they occur:

- `draft` → `deploying`: when the user invokes Deploy and validation passes.
- `deploying` → `deployed`: when all CAO calls succeed.
- `deploying` → `degraded`: when one or more CAO calls fail after at least one succeeded (i.e., partial state in CAO).
- `deploying` → `draft`: when zero CAO calls succeeded (full rollback path is acceptable: nothing was created).
- `deployed` → `degraded`: reserved for later milestones (terminal crash detection); not asserted in M1.

#### Scenario: All-success transitions to deployed

- **WHEN** every CAO call in the deploy sequence returns success
- **THEN** the final `deploy_state.status` is `deployed`
- **AND** `deploy_state.last_deployed` records the completion time

#### Scenario: Partial failure transitions to degraded

- **WHEN** profile installs succeed and session creation succeeds but adding the second of two terminals fails
- **THEN** `deploy_state.status` is `degraded`
- **AND** `deploy_state.errors` contains an entry identifying the failing node id and the CAO error

#### Scenario: Pre-session failure rolls back to draft

- **WHEN** the very first CAO call (profile install) fails
- **THEN** no CAO state was created, and `deploy_state.status` returns to `draft` rather than `degraded`
- **AND** the user sees the underlying error

### Requirement: Retry From Degraded

When a canvas is in `degraded` state, the user SHALL be able to invoke Retry Failed, which SHALL re-issue only the CAO calls corresponding to nodes whose ids do not appear in `terminal_map`. Successful previously-created terminals SHALL NOT be touched. The state SHALL transition to `deployed` if all retries succeed, or remain `degraded` if any still fail.

#### Scenario: Retry creates only missing terminals

- **WHEN** a degraded canvas has 2 of 3 worker terminals already created and the user clicks Retry Failed
- **THEN** the Reconciler issues exactly one `POST /sessions/{name}/terminals` for the third worker
- **AND** does not re-install profiles that were already installed

#### Scenario: Retry succeeding all transitions to deployed

- **WHEN** Retry Failed completes with all calls succeeding
- **THEN** `deploy_state.status` becomes `deployed`, `terminal_map` is fully populated, and `deploy_state.errors` is cleared

### Requirement: Deploy Progress UI

While `deploy_state.status === "deploying"`, the Builder SHALL render a non-modal progress panel listing each pending step with its status (`pending`, `in_flight`, `success`, `failed`). The panel SHALL update reactively as each call resolves. Completed deploys SHALL leave the panel visible for at least 3 seconds before auto-dismissing.

#### Scenario: Progress panel reflects per-step status

- **WHEN** a 3-node canvas deploys
- **THEN** the progress panel shows 5 steps and updates each step's status no more than 100 ms after the corresponding CAO response is received

### Requirement: Diff-Based Edit-After-Deploy

When a canvas is in `deployed` or `degraded` status and the user edits its content, the Reconciler SHALL compute the diff between the desired state (the edited canvas) and the recorded actual state (`terminal_map` plus the per-node profile snapshots taken at deploy time) and SHALL apply only the differences (per master spec §12 user-approved override). The diff strategy SHALL be:

- **Node added** (new node id present in canvas, absent from `terminal_map`) → install profile, add terminal via `POST /sessions/{name}/terminals`, record in `terminal_map`.
- **Node removed** (node id absent from canvas, present in `terminal_map`) → kill terminal via `DELETE /terminals/{id}`, remove from `terminal_map`.
- **Node profile content changed** (system_prompt, allowedTools, model, or provider differs from the deploy-time snapshot) → install updated profile, kill old terminal, add new terminal, update `terminal_map`.
- **Node display-only change** (display_name, position) → no CAO action; persist canvas only.
- **Edge added or removed** → no CAO action in v1; persist canvas. The system SHALL display a non-blocking advisory: "Edge changes require a Tear Down + redeploy to take effect on the supervisor."
- **Entry-point changed** → blocked. The user SHALL see a dialog requiring Tear Down before changing the entry-point.

The diff SHALL be computed against `terminal_map` and the per-node profile snapshots captured at deploy time, not against a fresh `GET /sessions/...`. This makes partial-state diffs deterministic and replayable.

#### Scenario: Adding a node to a deployed canvas creates one terminal

- **WHEN** a canvas is `deployed` with 3 terminals and the user adds a fourth node, then saves
- **THEN** the Reconciler issues exactly one profile install and one `POST /sessions/{name}/terminals`
- **AND** `terminal_map` gains exactly one entry; the existing 3 terminals are not touched

#### Scenario: Changing a node's system prompt replaces just that terminal

- **WHEN** a deployed canvas has its developer node's system_prompt edited and saved
- **THEN** the Reconciler installs a new profile, kills the old developer terminal, and adds a new developer terminal
- **AND** the supervisor and reviewer terminals are not touched

#### Scenario: Display-only edit avoids CAO calls

- **WHEN** the user moves a node 50 px to the right and saves
- **THEN** zero CAO calls are issued
- **AND** the canvas position is persisted to IndexedDB

#### Scenario: Entry-point change is blocked

- **WHEN** the user attempts to set `is_entry_point: true` on a worker node of a deployed canvas
- **THEN** a dialog appears stating "Changing the entry point requires Tear Down" and blocks the save
- **AND** the user can choose Tear Down (which transitions to `draft` and unlocks edits) or Cancel

### Requirement: Edit Allowed When Deployed or Degraded

The Canvas Builder SHALL allow editing when `deploy_state.status` is `deployed` or `degraded`. Saving an edit SHALL trigger the diff-based reconciler. While the diff is being applied, the canvas SHALL show a "Reconciling…" indicator and SHALL block further edits until the diff completes (success or failure).

#### Scenario: Edit during reconciliation is queued

- **WHEN** a diff is in flight and the user attempts a second edit
- **THEN** the editor disables save and shows "Reconciling…" until the in-flight diff completes
- **AND** after completion, the editor re-enables save

### Requirement: Tear Down

The user SHALL be able to invoke a Tear Down action on a canvas in `deployed` or `degraded` status. Tear Down SHALL issue `DELETE /sessions/{name}` for the recorded session and, on success, SHALL reset `deploy_state` to `{ status: "draft" }`. After Tear Down, the canvas SHALL be fully editable again, including the entry-point.

#### Scenario: Tear Down deletes the CAO session

- **WHEN** the user clicks Tear Down on a deployed canvas
- **THEN** the Reconciler issues `DELETE /sessions/{name}` for the recorded session
- **AND** on success, `deploy_state` is reset to `{ status: "draft" }` and `terminal_map` is cleared

#### Scenario: Tear Down failure surfaces error and preserves state

- **WHEN** the `DELETE /sessions/{name}` call fails
- **THEN** the canvas remains in its prior status with an error toast naming the CAO failure
- **AND** the user can retry Tear Down

### Requirement: Atomic Persistence of Deploy State

Every deploy_state mutation SHALL be persisted to IndexedDB before the corresponding CAO call is initiated, and again after the call resolves. A page reload during a deploy SHALL leave the canvas in either `deploying` (with the user offered Resume/Cancel options on next open) or one of the terminal states (`deployed`, `degraded`, `draft`); it SHALL NOT be possible to lose state about whether a CAO call was attempted.

#### Scenario: Reload mid-deploy preserves status

- **WHEN** the user starts a deploy and reloads the page after the session is created but before any terminals are added
- **THEN** opening the canvas shows status `deploying` with a Resume button that re-runs the Reconciler from the next pending step
