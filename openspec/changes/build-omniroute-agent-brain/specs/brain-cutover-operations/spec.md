## ADDED Requirements

### Requirement: Atomic single-router cutover
Cutover SHALL enable only the Main Brain/OmniRoute path after configuration, readiness, route, security, lifecycle and terminal-persistence gates pass.

#### Scenario: A cutover gate fails
- **WHEN** any mandatory gate is negative
- **THEN** new model admission remains closed and no alternate router is enabled

### Requirement: Fail-closed recovery state machine
Recovery SHALL contain only `NORMAL` with owner `omniroute` and `DEGRADED` with owner `none`. Transitions SHALL occur only at session boundaries, and restore SHALL require ready OmniRoute.

#### Scenario: OmniRoute outage occurs
- **WHEN** readiness is lost at a session boundary
- **THEN** recovery enters `DEGRADED`, closes new model admission and cannot promote another router

#### Scenario: OmniRoute becomes ready again
- **WHEN** strict readiness passes at a session boundary
- **THEN** recovery returns to `NORMAL` with `omniroute` as the sole owner

### Requirement: Safe rollback
Rollback SHALL select a previous accepted Main Brain/OmniRoute release/config or keep admissions closed. It MUST NOT restore provider credentials, direct routing or an alternate runtime.

#### Scenario: Previous revision is not ready
- **WHEN** rollback selects the prior revision but strict readiness fails
- **THEN** admissions remain closed and the incident is escalated

### Requirement: Preserve product/control-plane data
Rollout and rollback SHALL preserve Postgres migrations/data, workspaces, projects, squads, issues/tasks, sessions, terminal results and redacted evidence.

#### Scenario: Daemon revision is rolled back
- **WHEN** Main Brain returns to a previous accepted revision
- **THEN** durable Kanban and terminal state remains intact and is not replaced by placeholders