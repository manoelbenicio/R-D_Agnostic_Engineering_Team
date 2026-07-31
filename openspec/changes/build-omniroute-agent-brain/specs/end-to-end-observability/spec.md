## ADDED Requirements

### Requirement: Metadata-only correlation schema
The system SHALL define versioned safe identifiers joining ingress, DB queue, daemon
admission/lifecycle, CLI process, the pinned transport hop (`omniroute` or
`native_credential_home`), terminal persistence and WS/UI delivery.

#### Scenario: Required join identifier is missing
- **WHEN** a hop cannot be joined to its task/session
- **THEN** trace assembly reports an orphan and observability acceptance fails

### Requirement: Content and secret exclusion
Spans, labels, metrics and logs SHALL contain only safe identifiers, classifications, counters
and latency. They MUST NOT contain credentials, authorization headers, cookies, prompts, tool
payloads, repository content, reasoning, account emails/identity, raw source paths, isolated-home
paths, global HOME values, or connection strings.

#### Scenario: Task contains sensitive content
- **WHEN** a hop emits observability for the task
- **THEN** emitted fields contain no task content and structural leakage scanning remains clean

### Requirement: Lifecycle-to-terminal trace
Each accepted synthetic control-plane task SHALL produce one trace from task admission through exactly one persisted terminal result and delivery disposition.

#### Scenario: Terminal result is duplicated or absent
- **WHEN** trace assembly finds zero or multiple terminal persistence events for one task
- **THEN** acceptance fails with the affected correlation identifiers

### Requirement: Operational dashboards and alerts
Dashboards and alerts SHALL cover readiness/admission, active capacity, cancellation, process exit, terminal persistence latency/failure and trace gap rates without sensitive values.

#### Scenario: Terminal persistence degrades
- **WHEN** persistence latency or failure rate crosses its threshold
- **THEN** an alert identifies the failing hop and safe task correlations

### Requirement: Observability gates capacity and cutover
Higher capacity and production cutover MUST NOT proceed until required spans, trace assembly, structural leak scan and dashboards/alerts have accepted evidence.

#### Scenario: Capacity run is proposed without accepted observability
- **WHEN** the observability bundle is incomplete
- **THEN** tier promotion remains blocked