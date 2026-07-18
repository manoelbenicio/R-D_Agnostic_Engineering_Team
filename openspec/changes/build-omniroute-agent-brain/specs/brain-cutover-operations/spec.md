## ADDED Requirements

### Requirement: OpenSpec and GSD traceability
Before implementation, the project SHALL maintain an Agent Brain GSD baseline that maps every component and interface to a unique requirement, OpenSpec scenario/task, GSD phase/task, owner, evidence, status, and removal decision. Historical RPP/Prodex GSD artifacts MUST remain identifiable and MUST NOT operate as a concurrent master plan.

#### Scenario: Orphaned component is discovered
- **WHEN** a planning audit finds a component, interface, requirement, task, or evidence record without the complete traceability chain
- **THEN** the affected phase remains blocked until an owner and disposition are recorded in both OpenSpec and GSD

### Requirement: Strangler extraction
The migration SHALL introduce brand-neutral interfaces/modules around the proven daemon behavior and replace the credential/routing cluster behind those boundaries before removing the compatibility facade. A wholesale rename or rewrite MUST NOT be the first cutover step.

#### Scenario: First runnable vertical slice
- **WHEN** the new Agent Brain path is enabled for a canary task
- **THEN** retained lifecycle/workspace behavior runs through neutral interfaces while all model traffic uses the new OmniRoute adapter

### Requirement: Written feature parity gate
The cutover SHALL maintain a signed Prodex-to-OmniRoute parity matrix covering every known hot-path feature and special surface. Required unsupported behavior MUST block cutover unless product and security approve an explicit waiver with owner, restriction, and remediation date.

#### Scenario: Smart Context has no proven replacement
- **WHEN** OmniRoute cannot demonstrate protocol-safe Smart Context parity and no waiver is approved
- **THEN** Prodex removal is blocked even if basic model calls succeed

### Requirement: Protocol and failure acceptance gate
The exact deployed OmniRoute version SHALL pass the written architecture checklist for authentication, protocol fidelity, streaming, tools, continuation, rotation, expiry, quota, 429/circuits, fallback, cancellation, security, observability, and approved capacity.

#### Scenario: Models endpoint succeeds but tool streaming is unproven
- **WHEN** `/v1/models` returns 200 but an approved CLI's tool-streaming contract lacks evidence
- **THEN** that route remains unapproved for production tasks

### Requirement: Environment-specific endpoint
The deployment SHALL configure the OmniRoute base URL for the runtime that actually launches CLIs. The current host/WSL daemon SHALL use `http://127.0.0.1:20128`; a containerized runtime SHALL use reachable Docker DNS or an explicitly configured host gateway.

#### Scenario: Host daemon launches Codex
- **WHEN** the active Agent Brain runs on WSL/host
- **THEN** its controlled Codex provider targets `http://127.0.0.1:20128/v1` rather than the container-only `omniroute` DNS name

### Requirement: Atomic staged cutover
The migration SHALL progress through readiness, protocol canaries, provider/model canaries, capacity tiers, default-on gateway-required mode, legacy drain, and deletion. Each stage SHALL have recorded entry/exit criteria and rollback triggers.

#### Scenario: Canary error threshold is exceeded
- **WHEN** a stage exceeds its approved authentication, protocol, error, latency, or security threshold
- **THEN** new admissions return to the previous safe stage and affected tasks drain or stop according to policy

### Requirement: Safe rollback
Rollback SHALL restore the last accepted Agent Brain/OmniRoute configuration and task admission behavior without restoring direct provider credentials or dual router ownership.

#### Scenario: OmniRoute release must be rolled back
- **WHEN** the current OmniRoute version fails an operational gate
- **THEN** deployment selects the prior accepted OmniRoute version/config and holds new tasks as necessary rather than reactivating Prodex or provider keys in Agent Brain

### Requirement: Legacy removal gate
Prodex/L2 code, legacy Go rotation, credential account homes, provider-auth copying, and Multica-branded compatibility aliases SHALL be removed only after usage telemetry confirms no active consumer and rollback no longer depends on them.

#### Scenario: Compatibility alias is still in use
- **WHEN** telemetry identifies an active legacy API, environment, CLI, or stored-config consumer
- **THEN** the alias remains isolated behind the compatibility facade and receives a migration owner/deadline rather than being deleted blindly

### Requirement: Operational handover
Production cutover SHALL include named owners, dashboards/alerts, account and route operations, incident classification, backup/restore, secret rotation, upgrade/rollback, capacity controls, and escalation procedures.

#### Scenario: Provider-wide throttling occurs
- **WHEN** OmniRoute detects a provider-global 429 incident
- **THEN** operators can identify affected routes, circuit state, fallback policy, queued workload, safe retry time, and escalation owner without inspecting secrets or raw prompts
