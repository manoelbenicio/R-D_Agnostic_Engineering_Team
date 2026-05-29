## ADDED Requirements

### Requirement: Canvas Document Schema

The system SHALL define a `CanvasDocument` JSON schema that represents a multi-agent orchestration design as a directed graph of agent nodes and orchestration edges, plus configuration and deploy state. Every Canvas Builder UI operation, every Reconciler input, and every persistence read/write SHALL operate on this schema.

The schema SHALL include the following top-level fields:

- `id` (string, UUID) — stable canvas identity across saves and edits.
- `name` (string) — user-facing display name.
- `version` (integer, monotonic) — schema-content version, bumped on save.
- `created_at`, `updated_at` (string, ISO-8601 datetime).
- `nodes` (array of `CanvasNode`).
- `edges` (array of `CanvasEdge`).
- `config` (object): `working_directory` (string), `session_name` (string, optional), `provider_default` (`ProviderType`), `env_vars` (record of string→string, optional).
- `deploy_state` (object): `status` (one of `draft`, `deploying`, `deployed`, `degraded`), `session_name` (string, optional), `terminal_map` (record of node_id→terminal_id, optional), `last_deployed` (ISO-8601, optional), `errors` (array of `{ node_id, error }`, optional).

#### Scenario: Newly created canvas has draft status

- **WHEN** a user creates a new canvas through the Canvas Builder
- **THEN** the resulting `CanvasDocument` has `deploy_state.status === "draft"` and an empty `nodes`, empty `edges`, and an `id` that is a valid UUIDv4
- **AND** `created_at` and `updated_at` are equal and within 1 second of the current time

#### Scenario: Schema validation rejects malformed documents

- **WHEN** an external source provides a document missing a required field (e.g., no `nodes` array)
- **THEN** the schema validator rejects it with a structured error citing the missing field path
- **AND** the system SHALL NOT persist or render the malformed document

### Requirement: Canvas Node Schema

A `CanvasNode` SHALL represent a single agent block on the canvas. The schema SHALL include:

- `id` (string, UUID) — stable node identity across edits.
- `type` (literal: `"agent"`) — reserved for future node types.
- `position` (object: `x`, `y` as numbers).
- `data` (object):
  - `profile_name` (string) — the CAO agent profile identifier.
  - `display_name` (string) — user-facing label.
  - `role` (string) — one of `supervisor`, `developer`, `reviewer`, or a custom string.
  - `provider` (`ProviderType`, optional) — one of `kiro_cli`, `claude_code`, `codex`, `gemini_cli`, `kimi_cli`, `copilot_cli`, `opencode_cli`, `q_cli`.
  - `model` (string, optional).
  - `system_prompt` (string, markdown body).
  - `allowedTools` (array of string, optional).
  - `is_entry_point` (boolean) — exactly one node per canvas SHALL have `is_entry_point: true` at deploy time.

#### Scenario: Canvas with multiple entry points fails deploy validation

- **WHEN** the Reconciler receives a canvas containing two or more nodes with `is_entry_point: true`
- **THEN** deploy is rejected before any CAO call is made
- **AND** the user is shown an error identifying both offending nodes

#### Scenario: Canvas with zero entry points fails deploy validation

- **WHEN** the Reconciler receives a canvas where no node has `is_entry_point: true`
- **THEN** deploy is rejected with an explanation that a root supervisor is required

### Requirement: Canvas Edge Schema

A `CanvasEdge` SHALL represent an orchestration relationship between two nodes. The schema SHALL include:

- `id` (string, UUID).
- `source` (string) — id of the source node.
- `target` (string) — id of the target node.
- `type` (one of `handoff`, `assign`, `send_message`).
- `label` (string, optional).

The edge schema SHALL forbid edges where `source === target` (self-loops are not orchestration patterns CAO supports in M1).

#### Scenario: Self-loop edge rejected

- **WHEN** the canvas contains an edge whose `source` and `target` are the same node id
- **THEN** schema validation rejects the edge before persistence

#### Scenario: Edge referencing missing node rejected

- **WHEN** an edge's `source` or `target` does not match any node id in the same canvas
- **THEN** schema validation rejects the document

### Requirement: Local Canvas Persistence (IndexedDB)

In Milestone 1, the system SHALL persist Canvas Documents to the browser's IndexedDB. There SHALL be a `canvases` object store keyed by `id` and a `canvas_versions` object store keyed by `(canvas_id, version)`. Cloud persistence (Firestore/Postgres) is explicitly out of scope for this capability in M1.

#### Scenario: Canvas survives page reload

- **WHEN** a user creates and saves a canvas, then reloads the page
- **THEN** the canvas is still listed and its full content is restored byte-identical to what was saved

#### Scenario: Each save creates a new version snapshot

- **WHEN** a user modifies a canvas and saves it three times
- **THEN** the `canvas_versions` store contains three rows for that canvas id, with `version` values 1, 2, 3
- **AND** the `canvases` row reflects the latest version

### Requirement: Schema Migration Strategy

The system SHALL embed a `schema_version` constant in the data layer and SHALL refuse to load a Canvas Document whose `schema_version` exceeds the constant. When the constant exceeds the document's recorded version, the system SHALL apply migrations registered in a migration registry before returning the document to consumers.

#### Scenario: Document from a future schema version is refused

- **WHEN** IndexedDB contains a canvas whose embedded `schema_version` is greater than the running app's constant
- **THEN** the canvas list shows the canvas as "incompatible" and refuses to open it
- **AND** no automatic transformation is attempted
