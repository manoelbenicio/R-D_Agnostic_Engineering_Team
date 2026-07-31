## ADDED Requirements

### Requirement: Atomic transport cutover
Cutover SHALL enable only one approved pinned binding after its configuration, readiness,
security, lifecycle and terminal-persistence gates pass. `omniroute` cutover requires strict
OmniRoute route readiness and forbids native homes. `native_credential_home` cutover requires a
fresh exclusive R3 opaque assignment for one existing logical runtime/agent and MUST NOT use
OmniRoute.

#### Scenario: A cutover gate fails
- **WHEN** any mandatory gate for the selected binding is negative
- **THEN** new model admission remains closed
- **AND** no other binding, router, global HOME, or replacement native home is enabled

### Requirement: Fail-closed binding-local recovery state machine
Recovery for an admitted session SHALL contain only `NORMAL(<pinned binding>)` and `DEGRADED`
with owner `none`. Transitions SHALL occur only at session boundaries, and restore SHALL require
the same binding's strict readiness.

#### Scenario: OmniRoute outage occurs
- **WHEN** OmniRoute readiness is lost for an `omniroute` session at a session boundary
- **THEN** recovery enters `DEGRADED`, closes new model admission, and cannot promote native mode

#### Scenario: Native authority becomes unavailable
- **WHEN** R3 assignment/catalog readiness is lost for a `native_credential_home` session at a
  session boundary
- **THEN** recovery enters `DEGRADED`, closes new model admission, and cannot promote OmniRoute or
  another native home

#### Scenario: Selected binding becomes ready again
- **WHEN** strict readiness for the same pinned binding passes at a session boundary
- **THEN** recovery returns to `NORMAL` with that binding unchanged

### Requirement: Safe rollback
Rollback SHALL select a previous accepted release/configuration within the same pinned binding
or keep admissions closed. It MUST NOT switch binding, restore ambiguous routing, resolve a
replacement native home, or copy, move, delete, truncate, sanitize, overwrite, chmod, or change
ownership of a source credential home.

#### Scenario: Previous revision is not ready
- **WHEN** rollback selects the prior revision/configuration but strict readiness for the same
  binding fails
- **THEN** admissions remain closed and the incident is escalated
- **AND** the other binding and any replacement native home remain forbidden

### Requirement: Preserve product/control-plane data
Rollout and rollback SHALL preserve Postgres migrations/data, workspaces, projects, squads, issues/tasks, sessions, terminal results and redacted evidence.

#### Scenario: Daemon revision is rolled back
- **WHEN** Main Brain returns to a previous accepted revision
- **THEN** durable Kanban and terminal state remains intact and is not replaced by placeholders

### Requirement: Repository/OpenSpec synchronization gate
Before any future handoff or deployment, both affected OpenSpecs SHALL strict-validate, the
cross-authority scan SHALL pass, and the Git index/worktree SHALL have zero staged, modified, or
untracked owned files. The exact local commit SHALL be published to its configured non-main
remote branch; local HEAD and upstream SHA SHALL match and ahead/behind SHALL be `0/0`.
Deployment evidence SHALL pin that SHA. This requirement documents a future gate only and does
not authorize publishing or deployment.

#### Scenario: Synchronization is incomplete
- **WHEN** validation or the authority scan fails, any owned file is pending, the commit is not
  published to a non-main upstream, SHAs differ, or ahead/behind is not `0/0`
- **THEN** handoff and deployment remain fail closed
- **AND** no deployment evidence may claim an unpinned or different revision
