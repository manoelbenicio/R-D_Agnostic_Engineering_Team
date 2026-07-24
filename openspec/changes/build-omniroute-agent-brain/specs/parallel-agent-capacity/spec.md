## ADDED Requirements

### Requirement: Bounded task admission
Main Brain SHALL enforce an explicit active-task limit and bounded overload behavior. It MUST NOT permit unbounded goroutine, process, socket, queue or log growth.

#### Scenario: Admission limit is full
- **WHEN** another Kanban task is claimed at the active-task limit
- **THEN** it waits or receives a deterministic retryable status without starting a CLI

### Requirement: Cancellation releases capacity once
Cancellation SHALL stop the child process and downstream request, persist one terminal cancellation and release task capacity exactly once.

#### Scenario: Active streamed task is cancelled
- **WHEN** cancellation reaches Main Brain
- **THEN** execution stops, counters reconcile, the slot is released once and one cancelled terminal result is published

### Requirement: Evidence-based tiers
Capacity tiers 20, 50 and 100 SHALL remain disabled until the exact deployed Main Brain/OmniRoute topology passes its approved workload and resource thresholds.

#### Scenario: Higher tier lacks evidence
- **WHEN** a requested tier has no accepted report
- **THEN** Main Brain enforces the highest proven lower limit and exposes that effective limit

### Requirement: Failure isolation
One task's timeout, gateway rejection, cancellation or terminal-persistence failure SHALL NOT leak credentials, corrupt another task's environment/session or release another task's slot.

#### Scenario: One parallel task fails readiness
- **WHEN** one task is rejected while others are running
- **THEN** only that task receives the fail-closed terminal disposition and other task lifecycle state remains intact