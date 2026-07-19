## ADDED Requirements

### Requirement: Eight-hop correlation schema
The system SHALL define a versioned, metadata-only correlation schema that joins a single task's activity across eight hops: ingress control API, DB queue, daemon admission/lifecycle, CLI process, OmniRoute/provider, terminal persistence, WS/UI delivery, and assembled trace. The schema SHALL declare the identifiers `request_id`, `queue_msg_id`, `task_id`, `session_id`, `launch_id`, `proc_id`, `omni_request_id`, `result_id`, and `delivery_id`, their join relationships, their propagation carriers, a `contract_version`, and a `secrets_present=false` invariant.

#### Scenario: A correlation identifier is missing at a hop
- **WHEN** a span is emitted without the join identifier required to link it to its adjacent hops
- **THEN** trace assembly flags the span as an orphan and the observability gate cannot be declared passed

### Requirement: Per-hop metadata-only spans
Each of the eight hops SHALL emit a structured span containing only correlation identifiers, classifications, counters, and latencies — never prompts, tool payloads, repository content, opaque reasoning, secrets, authorization headers, cookies, account emails, or connection strings. CLI argv SHALL be redacted structurally (shape only, never values).

#### Scenario: A hop processes a task carrying sensitive content
- **WHEN** any hop emits its span for a task whose request or result contains credential-like or content fields
- **THEN** the span contains only metadata and correlation identifiers and no sensitive value appears in any label, field, or log

### Requirement: Continuous end-to-end trace assembly
For a synthetic task, the system SHALL assemble one continuous trace that joins all eight hops on the correlation identifiers, with no gaps or orphaned spans. Trace assembly SHALL detect and report any hop that fails to join.

#### Scenario: Every synthetic task produces one continuous trace
- **WHEN** the synthetic observability workload runs a batch of tasks end to end
- **THEN** each task yields exactly one continuous eight-hop trace and any gap or orphan is reported as a gate failure

### Requirement: Structural leakage-clean acceptance
The system SHALL run a structural (not pattern-only) scan across every span, label, and log for all hops and SHALL prove that no secret or content value is present. Any detected leakage SHALL fail the observability gate.

#### Scenario: A leak scan runs across all hops
- **WHEN** the structural leak scan inspects all emitted spans, labels, and logs
- **THEN** it confirms zero secret or content values, or it fails the gate and identifies the offending hop

### Requirement: Observability dashboards and alerts
The system SHALL provide dashboards and alerts covering per-hop latency, per-hop error classification, queue depth/wait, delivery drops, and trace gap/orphan rates, using pseudonymous identifiers only.

#### Scenario: A hop degrades
- **WHEN** a hop's error rate, latency, or trace-gap rate crosses its configured threshold
- **THEN** an alert fires with per-hop attribution and correlation identifiers and without exposing secrets or content

### Requirement: Blocking G4-OBS stop-gate
The end-to-end observability gate SHALL be blocking: it passes only when every hop span (OBS-2..OBS-8), the correlation schema (OBS-1), continuous trace assembly (OBS-9), the structural leak scan (OBS-10), and the dashboards/alerts acceptance bundle (OBS-11) are independently accepted. Capacity tiers and default cutover MUST NOT proceed until the gate passes.

#### Scenario: Capacity or cutover is attempted before the gate passes
- **WHEN** a capacity tier run or default gateway-required cutover is attempted while G4-OBS is not passed
- **THEN** the action is blocked and the responsible stage remains gated until the full OBS acceptance bundle is recorded
