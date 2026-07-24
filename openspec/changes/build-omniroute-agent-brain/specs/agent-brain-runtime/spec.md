## ADDED Requirements

### Requirement: Main Brain owns product-neutral orchestration
Main Brain SHALL own Kanban task admission, workspace/repository/worktree preparation, process lifecycle, cancellation/watchdogs, stream delivery, session pinning and terminal-result publication. It SHALL NOT own inference account or provider routing.

#### Scenario: Kanban task completes
- **WHEN** an approved task is claimed and OmniRoute is ready
- **THEN** Main Brain prepares the workspace, launches the approved CLI, streams progress, persists exactly one terminal result and releases capacity

### Requirement: OmniRoute plan is mandatory
Every model task SHALL have an admitted plan with `RouterOwner=omniroute` before a child process is created.

#### Scenario: Integration is disabled or incomplete
- **WHEN** a task has no admitted OmniRoute plan
- **THEN** Main Brain rejects it before CLI launch with a bounded fail-closed classification

### Requirement: CLI and route identities are separate
`CLIKind` SHALL select only the approved frontend executable and `RouteModel` SHALL select only a declared OmniRoute model. Neither field SHALL select provider credentials or accounts.

#### Scenario: Compatible frontend uses a routed model
- **WHEN** a task selects an approved frontend and route model
- **THEN** Main Brain builds the matching gateway adapter without provider-account lookup

### Requirement: Product state remains durable
Projects, squads, Kanban issues/tasks, sessions, terminal results and related control-plane records SHALL remain persisted in Postgres across daemon restarts.

#### Scenario: Daemon restarts after terminal persistence
- **WHEN** Main Brain restarts after a result was persisted
- **THEN** the product state and terminal result remain available and are not recreated as synthetic success