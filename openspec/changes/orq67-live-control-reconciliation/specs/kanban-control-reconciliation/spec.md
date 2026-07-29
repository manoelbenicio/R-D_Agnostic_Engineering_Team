# Spec — kanban-control-reconciliation

## ADDED Requirements

### Requirement: REQ-01 Every recorded count MUST carry its measurement instant and method

Control documentation MUST NOT state a board count, completion percentage or card total without
the UTC instant at which it was measured and the command that produced it.

#### Scenario: Recording a board snapshot

- **WHEN** a reconciliation records Kanban status counts
- **THEN** the document MUST state the UTC timestamp of the measurement
- **AND** MUST state the exact command used, for example
  `multica issue list --limit 500 --output json`
- **AND** MUST record the API-reported `total` and whether pagination was exhausted
- **AND** MUST NOT present a count whose measurement instant is unknown

#### Scenario: The board moves during the run

- **WHEN** two measurements taken during the same reconciliation disagree
- **THEN** the document MUST record both instants and the observed drift
- **AND** MUST NOT silently keep only the more convenient figure

### Requirement: REQ-02 The cohort MUST be the whole workspace, never a convenient subset

A reconciliation MUST reconcile every issue in the workspace, including issues with a null
`project_id` or a different `project_id`.

#### Scenario: Issues exist outside the primary project

- **WHEN** the workspace contains issues that do not carry the primary project's `project_id`
- **THEN** those issues MUST be counted in the cohort and listed by identifier
- **AND** the split between the primary project and the remainder MUST be stated
- **AND** the per-status counts MUST sum to the API-reported workspace `total`
- **AND** MUST NOT report a subset total as the workspace total

#### Scenario: A superseded cohort is quoted

- **WHEN** an earlier revision recorded a different cohort size or completion percentage
- **THEN** the current revision MUST mark that figure superseded and state the measured figure
- **AND** MUST NOT allow the superseded percentage to read as current

### Requirement: REQ-03 Digests MUST be measured at full width and never padded

Any SHA-256 recorded for a binary artifact MUST be the complete 64-hex output of a hashing command
executed against the artifact on disk.

#### Scenario: Recording the live daemon artifact

- **WHEN** the reconciliation records the daemon artifact identity
- **THEN** the value MUST be the full 64-hex `sha256sum` output for the artifact path
- **AND** the artifact path, size and mode MUST be recorded alongside it
- **AND** truncated digests MUST NOT be extended with zeros or any other filler
- **AND** if a prior revision recorded a padded digest, the current revision MUST retract it
  explicitly in writing

### Requirement: REQ-04 Commit SHAs and artifact digests MUST be labelled distinctly and verified

The document MUST distinguish a git commit SHA from a content digest, and MUST verify each commit
SHA in the repository before quoting it.

#### Scenario: Quoting a commit

- **WHEN** the reconciliation cites a commit for a delivery, rollback or lineage base
- **THEN** the value MUST be labelled as a commit
- **AND** MUST resolve under `git rev-parse --verify <sha>^{commit}` in this repository
- **AND** MUST be quoted in full 40-hex form
- **AND** MUST NOT be presented as `sha256:` or as an artifact digest

### Requirement: REQ-05 Measured facts MUST be separated from reported facts

The document MUST state, for every material fact, whether it was measured in the current run or
supplied by another party.

#### Scenario: A fact cannot be measured on the current host

- **WHEN** a required verification tool is unavailable, for example no container runtime is
  installed
- **THEN** the affected value MUST be marked as reported and not independently measured
- **AND** the reason it could not be measured MUST be stated
- **AND** the document MUST name what would be required to verify it
- **AND** MUST NOT present the value as a measured fact

### Requirement: REQ-06 Liveness and readiness gates MUST be named separately, and a unit query MUST prove the unit exists in the scope queried

The daemon gate and the backend readiness gate are distinct endpoints and MUST NOT be conflated. A
supervision claim MUST identify the exact unit that runs the process, in the correct systemd scope,
and MUST combine unit state with cgroup membership and a health probe.

#### Scenario: Recording the recovery gates

- **WHEN** the reconciliation records verification of the recovered daemon
- **THEN** the daemon gate MUST be recorded as `http://127.0.0.1:19514/health`
- **AND** backend readiness MUST be recorded separately as `http://127.0.0.1:18080/readyz`
- **AND** each result MUST carry its measured HTTP status

#### Scenario: Querying a systemd unit

- **WHEN** a document cites `ActiveState`, `SubState`, `NRestarts` or `ExecMainStartTimestamp`
- **THEN** the query MUST name the exact unit that runs the process, which for the ORQ2 daemon is
  `multica-daemon-orq2-credential.service`
- **AND** the query MUST use the correct scope, which for that unit is `systemctl --user`
- **AND** `LoadState` MUST be recorded and MUST be `loaded` before any other property is treated as a
  measurement
- **AND** a `LoadState=not-found` result MUST be reported as "unit absent in this scope" and MUST NOT
  be reported as the daemon being inactive, unsupervised or unmanaged, because systemd returns
  `ActiveState=inactive` and `MainPID=0` as defaults for a unit that does not exist
- **AND** the unit's `MainPID` MUST be cross-checked against `/proc/<pid>/cgroup`, which MUST name the
  same unit

#### Scenario: A daemon restart or recovery is certified

- **WHEN** a daemon recovery, rollback or restart is declared verified
- **THEN** all five signals MUST hold together: `ActiveState=active` with `SubState=running`,
  `NRestarts` at its expected value, an `ExecMainStartTimestamp` matching the intended start instant,
  cgroup membership of the running PID in that unit, and HTTP 200 on the daemon health endpoint
- **AND** `NRestarts` alone MUST NOT be presented as evidence of stability
- **AND** a `--foreground` argument in `ExecStart` MUST NOT be read as evidence against systemd
  supervision, because it is the required form for a `Type=simple` unit

### Requirement: REQ-11 A loopback endpoint MUST be attributed to the host that actually serves it

Where a local port is a forward to another host, the document MUST say so, because the remediation
for a forward failure differs from the remediation for a service failure.

#### Scenario: A loopback port is an SSH forward

- **WHEN** a reconciliation records a `127.0.0.1` endpoint result
- **THEN** it MUST establish whether a local process serves that port or whether a tunnel forwards it
- **AND** if a tunnel serves it, the tunnel unit MUST be named, together with its `ActiveState`
- **AND** the endpoint result MUST be attributed to the remote host that answers
- **AND** the document MUST state that a failure on that endpoint is ambiguous between a remote
  service fault and a broken forward until the tunnel unit is checked

### Requirement: REQ-07 User-facing and internal origins MUST be distinguished by host

The canonical user-facing URL and any loopback origin MUST be recorded with the host that actually
serves them.

#### Scenario: Recording endpoint topology

- **WHEN** the reconciliation records the canonical Kanban URL
- **THEN** `https://orq1.tail96e2c0.ts.net` MUST be recorded as the canonical user-facing URL with
  its measured HTTP status
- **AND** a loopback origin MUST be attributed to the host on which it listens
- **AND** an origin that does not listen on the measuring host MUST be recorded as not listening
  there rather than as an internal origin of that host

### Requirement: REQ-08 Incident records MUST be content-free for credentials

A security-incident record MUST convey cause and remediation without exposing or narrowing any
secret.

#### Scenario: Recording a credential exposure

- **WHEN** the reconciliation records a credential-exposure incident
- **THEN** it MUST record the mechanism, the affected provider role and the remediation
  preconditions
- **AND** MUST NOT record the secret value, any fragment of it, its length, or its storage location
- **AND** the card MUST remain blocked until the owner re-authenticates

### Requirement: REQ-09 Human-blocked cards MUST stay visible and MUST NOT be reported as agent work

Cards whose blocker is an owner action MUST be preserved in the canonical table and identified as
requiring a human.

#### Scenario: Reconciling blocked cards

- **WHEN** the reconciliation lists pending work
- **THEN** ORQ-42, ORQ-43, ORQ-44 and ORQ-64 MUST be present and marked as blocked on a human
- **AND** the specific owner action MUST be named for each
- **AND** they MUST NOT be presented as available for agent dispatch
- **AND** they MUST NOT be dropped by a cohort filter

### Requirement: REQ-10 Reconciliation MUST NOT mutate state it only observes

A documentation reconciliation MUST propose status transitions for approval rather than applying
them, and MUST NOT edit artifacts owned by other work items.

#### Scenario: A card's status disagrees with its evidence

- **WHEN** the reconciliation finds a card whose recorded status conflicts with its own metadata
- **THEN** the conflict MUST be recorded and a transition MUST be proposed for approval
- **AND** the transition MUST NOT be applied by the reconciliation
- **AND** artifacts owned by another work item, such as the ORQ-48 recorder ledger, MUST NOT be
  modified

#### Scenario: Preserving superseded history

- **WHEN** a revision supersedes an earlier cohort or narrative
- **THEN** the earlier content MUST be preserved verbatim and labelled superseded
- **AND** MUST NOT be deleted, because the executions and evidence digests it records occurred
