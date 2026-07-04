## ADDED Requirements

### Requirement: Go↔L2 Runtime Contract (rpp.l2.v1)
The system SHALL define a versioned local contract between Multica Go (control plane) and the prodex/Rust runtime (L2) with operations HealthCheck, ApplyPolicy, RegisterAccounts, StartSession, StopSession, RouteDecisionEvent, RuntimeEventStream and KillSwitch. Go MUST NOT route in-flight requests.

#### Scenario: Contract schema compiles
- **WHEN** the contract and event schema are defined
- **THEN** the event schema SHALL compile (JSON Schema Draft 2020-12) and be versioned as `rpp.l2.v1`

#### Scenario: Control plane pushes desired state
- **WHEN** Multica has policy/budgets/kill-switch and approved accounts
- **THEN** Go SHALL push them via ApplyPolicy/RegisterAccounts and the runtime SHALL own in-flight routing

### Requirement: Single Router Per Session
The system SHALL guarantee exactly one router per session: Go holds desired-state, the Rust runtime performs runtime routing. Event ingest MUST NOT trigger Go-side rotation.

#### Scenario: One router proven
- **WHEN** a session runs end to end
- **THEN** a test SHALL prove only the runtime routes the in-flight request and Go never rotates mid-flight

#### Scenario: Event ingest is inert to routing
- **WHEN** runtime events are ingested by Go
- **THEN** ingest SHALL be validated to not trigger any Go-side rotation
