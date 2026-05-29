## ADDED Requirements

### Requirement: Visual Canvas Editor

The Canvas Builder SHALL provide a 2D infinite canvas using a node-graph editor library (`@xyflow/react`) on which the user places agent blocks and draws orchestration edges between them. The canvas SHALL support pan, zoom, multi-select, drag-to-move, and undo/redo for at least the last 20 user actions. All edits SHALL update the in-memory `CanvasDocument` and SHALL be reflected to the persistence layer on save.

#### Scenario: User adds a node and persists

- **WHEN** the user drags a "Supervisor" template from the palette onto the canvas at coordinates (100, 100) and clicks Save
- **THEN** the underlying `CanvasDocument` contains exactly one node with `role: "supervisor"`, `position: { x: 100, y: 100 }`, and `is_entry_point: true`
- **AND** the document is persisted to IndexedDB

#### Scenario: Undo restores prior state

- **WHEN** the user adds a node, then clicks Undo
- **THEN** the canvas returns to the state preceding the add and the in-memory `CanvasDocument` matches that prior state

### Requirement: Agent Palette

The Builder SHALL display a palette listing the four Milestone 1 starter block types: `Supervisor`, `Developer`, `Reviewer`, and `Custom`. Each palette item SHALL render a SENTINEL card with a glyph and a label. Dragging a palette item onto the canvas SHALL create a new `CanvasNode` populated with sensible defaults for that role: a unique `profile_name`, a default `display_name`, a default `system_prompt` from a role-template registry, and `is_entry_point` set to `true` only for `Supervisor` blocks when no entry-point already exists.

#### Scenario: Adding a second supervisor does not auto-claim entry point

- **WHEN** the canvas already contains a Supervisor node with `is_entry_point: true` and the user drags a second Supervisor onto the canvas
- **THEN** the new node is created with `is_entry_point: false`
- **AND** the user can promote it to entry-point via the configuration panel (replacing the prior entry-point)

#### Scenario: Custom block has minimal defaults

- **WHEN** the user drags a Custom block onto the canvas
- **THEN** the new node has empty `system_prompt`, no preselected provider, and a placeholder `display_name` of "Custom Agent"

### Requirement: Edge Drawing With Mode Selection

The Builder SHALL allow the user to draw a directed edge between two nodes by dragging from a node's output handle to another node's input handle. The newly created edge SHALL default to `type: "handoff"`. The user SHALL be able to change the edge type to `assign` or `send_message` via an inline edge label menu, and the visual style SHALL match the master spec §4.3 visual table (solid for handoff, dashed for assign, dotted for send_message).

#### Scenario: Default edge type is handoff

- **WHEN** the user drags from a Supervisor's output handle to a Developer's input handle
- **THEN** a new edge is created with `type: "handoff"` and rendered as a solid arrow

#### Scenario: User changes edge type via label menu

- **WHEN** the user clicks the edge label and selects "Assign"
- **THEN** the edge's `type` becomes `assign` and the visual style transitions to a dashed arrow within 100 ms

### Requirement: Block Configuration Panel

When a node is selected, the Builder SHALL display a configuration panel exposing every editable field defined by `CanvasNode.data`: `display_name`, `role`, `provider`, `model`, `allowedTools`, and `system_prompt`. The `provider` field SHALL be a dropdown gated by the validated providers from `api-key-management`. The `model` dropdown SHALL show every model returned by the provider's validation response (master spec v4.2 §8.10) with NO default selection and NO recommendation — the user explicitly picks the model. The `system_prompt` field SHALL render in a Monaco-style editor with markdown syntax highlighting and a minimum height of 240 px. Edits in the panel SHALL update the canvas in real time without requiring a save.

#### Scenario: Editing display_name updates the node label live

- **WHEN** the user types into `display_name` while the panel is open
- **THEN** the corresponding node label on the canvas updates with each keystroke (debounced no more than 100 ms)

#### Scenario: Provider dropdown reflects validated set

- **WHEN** only Anthropic is validated in `api-key-management`
- **THEN** the provider dropdown shows exactly Anthropic-mapped CAO providers (e.g., `claude_code`)

#### Scenario: Model dropdown has no default and no recommendation

- **WHEN** the user selects a provider for a node
- **THEN** the model dropdown is populated with every model returned by that provider's validation response
- **AND** no model is preselected and no item is highlighted as "recommended" or "default"
- **AND** the user MUST explicitly pick a model before the node passes pre-deploy validation

### Requirement: Templates Picker

The Builder SHALL expose a Templates picker — invokable from the canvas list at `/`, from an empty canvas, and from a "Use Template" toolbar action. Picker entries SHALL come from the `canvas-templates` capability and SHALL display, per template: name, agent count, primary edge type, and the displayed cost-per-hour estimate (with mandatory ⚠️ label per `finops-tier1`). Selecting a template SHALL clone its definition into a new draft canvas owned by the current user, populating nodes/edges/config; the user SHALL then be free to edit the cloned canvas.

#### Scenario: Picker shows all 10 v1 templates with cost estimates

- **WHEN** the user opens the Templates picker
- **THEN** all 10 templates from the `canvas-templates` capability are listed
- **AND** each template shows its agent count, primary edge type, and ⚠️-labeled cost-per-hour estimate

#### Scenario: Selecting a template creates a draft

- **WHEN** the user selects "Code Review Pipeline" from the picker
- **THEN** a new canvas is created with `deploy_state.status = "draft"`, the 3 nodes and 2 handoff edges from the template, and is opened in the Builder

### Requirement: Voice Trigger

The Builder SHALL display a voice-input affordance (microphone button) in the toolbar. Activating it (click or `Ctrl+Shift+V`/`Cmd+Shift+V`) SHALL hand control to the `speech-to-canvas` capability. When `speech-to-canvas` returns a generated canvas through its intent-preview confirm step, the Builder SHALL replace its current draft (or create a new one if not in a draft) with the generated canvas, and SHALL leave it open in editing mode for the user to refine before deploy.

#### Scenario: Voice trigger opens the speech panel

- **WHEN** the user clicks the microphone button
- **THEN** the `speech-to-canvas` panel is presented overlaid on the canvas
- **AND** focus moves into the speech panel for accessibility

#### Scenario: Confirmed voice intent populates the canvas

- **WHEN** `speech-to-canvas` returns a confirmed `CanvasDocument` to the Builder
- **THEN** the Builder loads the document into the editor with auto-layout applied
- **AND** the canvas is in `deploy_state.status = "draft"` until the user invokes Deploy

### Requirement: Save and Load

The Builder SHALL save the current canvas to IndexedDB whenever the user presses Cmd/Ctrl+S or clicks the Save button. Saving SHALL bump `version`, update `updated_at`, and write a new row to `canvas_versions`. The canvas list at `/` SHALL show all canvases ordered by `updated_at` descending and SHALL allow opening any of them.

#### Scenario: Save bumps version

- **WHEN** the user opens an existing canvas with `version: 4` and saves once
- **THEN** the persisted canvas has `version: 5` and a new `canvas_versions` row exists for version 5

#### Scenario: Canvas list orders by recency

- **WHEN** the user has three canvases A, B, C with `updated_at` of 10:00, 11:00, 09:00 respectively
- **THEN** the canvas list at `/` displays them in order B, A, C

### Requirement: Deploy Entry Point

The Builder SHALL display a `Deploy` button in the toolbar. Clicking Deploy SHALL invoke the `canvas-reconciler` capability with the current `CanvasDocument`. The button SHALL be disabled when the canvas fails any pre-deploy validation: zero or multiple `is_entry_point` nodes, any node missing a validated provider, any node whose model is unselected (per master spec v4.2 §8.10 model-dropdown rule), or zero nodes total. Disabled-state hover SHALL show the specific reason and SHALL identify the offending node id when applicable.

The Builder SHALL allow the user to create and edit canvases regardless of provider validation status (read-only browse mode per master spec v4.2 §12). Only Deploy is gated.

#### Scenario: Deploy disabled when no entry point

- **WHEN** the canvas contains nodes but none have `is_entry_point: true`
- **THEN** the Deploy button is disabled and its hover tooltip reads "No entry-point supervisor — promote one node to entry point"

#### Scenario: Deploy disabled when provider not configured

- **WHEN** any node references a provider whose validation status is not `set`
- **THEN** the Deploy button is disabled and its hover tooltip identifies the offending node and the unconfigured provider

#### Scenario: Deploy disabled when model unselected

- **WHEN** any node has a validated provider but no model selected
- **THEN** the Deploy button is disabled and its hover tooltip identifies the offending node and reads "Pick a model for this node"

#### Scenario: Edits allowed without any provider configured

- **WHEN** a user with zero validated providers opens the Canvas Builder
- **THEN** the user can drag nodes, draw edges, edit prompts, and save drafts to IndexedDB
- **AND** Deploy is disabled with the documented reason

### Requirement: Touch Support Deferred

In Milestone 1 the Canvas Builder SHALL be optimized for desktop pointer input (mouse + trackpad). Touch-first drag-and-drop on tablets is explicitly deferred to Milestone 2. The touch-detect heuristic SHALL still render the canvas (read-only) so users on touch devices see their work; edits on touch devices are out of scope for M1.

#### Scenario: Touch device shows read-only canvas

- **WHEN** the user opens the Canvas Builder on a touch-only device
- **THEN** the canvas renders the document content and palette is hidden
- **AND** a banner indicates "Editing on touch devices arrives in Milestone 2"
