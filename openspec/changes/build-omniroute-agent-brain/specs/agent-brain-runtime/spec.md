## ADDED Requirements

### Requirement: Main Brain owns product-neutral orchestration
Main Brain SHALL own Kanban task admission, workspace/repository/worktree preparation, process lifecycle, cancellation/watchdogs, stream delivery, session pinning and terminal-result publication. It SHALL NOT own inference account or provider routing.

#### Scenario: Kanban task completes
- **WHEN** an approved task is claimed and its pinned transport binding is ready
- **THEN** Main Brain prepares the workspace, launches the approved CLI, streams progress, persists exactly one terminal result and releases capacity

### Requirement: One transport plan is mandatory
Every model task SHALL have exactly one admitted `TransportBinding` plan, either `omniroute` or
`native_credential_home`, before a child process is created.

#### Scenario: Transport integration is absent or ambiguous
- **WHEN** a task has no complete plan, more than one binding, or an unknown binding
- **THEN** Main Brain rejects it before CLI launch with a bounded fail-closed classification
- **AND** it does not translate or fallback to another binding

#### Scenario: Native plan is admitted
- **WHEN** a task selects `native_credential_home`
- **THEN** R3 resolves one approved exclusive opaque home for one existing logical runtime/agent
- **AND** OmniRoute is not probed, contacted, or used

### Requirement: CLI and transport identities are separate
`CLIKind` SHALL select only the approved frontend executable. For `omniroute`, `RouteModel`
SHALL select only a declared OmniRoute model and SHALL NOT select provider accounts or
credentials. For `native_credential_home`, the pinned opaque R3 assignment SHALL identify the
only native home and no OmniRoute route identity SHALL be used.

#### Scenario: Compatible frontend uses OmniRoute
- **WHEN** a task selects `omniroute`, an approved frontend, and route model
- **THEN** Main Brain builds the matching gateway adapter without provider-account lookup

#### Scenario: Compatible frontend uses native credential home
- **WHEN** a task selects `native_credential_home` and has a valid R3 assignment
- **THEN** Main Brain builds the matching native CLI environment using only daemon-local isolated
  home references
- **AND** it performs no gateway account lookup and exposes no raw path or account identity

### Requirement: Product state remains durable
Projects, squads, Kanban issues/tasks, sessions, terminal results and related control-plane records SHALL remain persisted in Postgres across daemon restarts.

#### Scenario: Daemon restarts after terminal persistence
- **WHEN** Main Brain restarts after a result was persisted
- **THEN** the product state and terminal result remain available and are not recreated as synthetic success

### Requirement: Kanban-only executable dispatch

Every new agent execution SHALL originate from the supported product Kanban assignment or
follow-up path and SHALL create exactly one product task. Terminal supervision tools SHALL NOT
start parallel work outside that queue.

#### Scenario: TL assigns one issue

- **WHEN** the TL assigns a ready issue to one healthy agent through Kanban
- **THEN** exactly one task SHALL be enqueued for that issue/agent activation
- **AND** no Herdr execution SHALL be launched for the same work

#### Scenario: Task terminates without completing acceptance

- **WHEN** a task completes with a diagnostic, review block or infrastructure failure
- **THEN** the issue SHALL remain open in the truthful workflow state
- **AND** it SHALL NOT be marked done solely because the task is terminal
