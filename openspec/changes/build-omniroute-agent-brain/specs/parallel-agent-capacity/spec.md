## ADDED Requirements

### Requirement: Configurable task capacity tiers
The system SHALL support operator-selected capacity tiers of at least 20, 50, and 100 simultaneous Agent Brain tasks, subject to measured host and upstream limits.

#### Scenario: Operator selects the 50-task tier
- **WHEN** the configured and validated capacity tier is 50
- **THEN** the Agent Brain admits up to 50 active tasks and applies the documented queue or rejection policy beyond that limit

### Requirement: Separate task and inference concurrency
Agent Brain task admission, OmniRoute global concurrency, route/model concurrency, and per-account concurrency SHALL be independently configurable. Strict round-robin selection MUST NOT imply one active request per route or one global request at a time.

#### Scenario: Accounts have active requests
- **WHEN** an account pool has multiple requests in flight within configured safety limits
- **THEN** OmniRoute continues selecting accounts for new requests without serializing the entire pool

### Requirement: Bounded admission and overload
The Agent Brain and OmniRoute SHALL use bounded queues/admission controls and deterministic retryable overload errors; they MUST NOT permit unbounded memory, goroutine/thread, socket, or log growth.

#### Scenario: Capacity and queue are full
- **WHEN** a new task or inference request arrives after its applicable active and queue limits are exhausted
- **THEN** the responsible layer rejects it with a safe machine-readable overload status and retry guidance

### Requirement: Cancellation releases capacity
Cancelling a queued task, active CLI, queued inference request, or active stream SHALL stop downstream work promptly and release task, route, account, connection, and accounting capacity exactly once.

#### Scenario: Operator cancels an active streaming task
- **WHEN** cancellation reaches the Agent Brain
- **THEN** the CLI and OmniRoute upstream request are aborted, all capacity counters return to the correct value, and one terminal cancellation is recorded

### Requirement: Fairness and eligibility evidence
The system SHALL measure selection distribution under concurrency and SHALL explain imbalance using eligibility, continuation affinity, capacity, quota, cooldown, or circuit state rather than hidden randomness or races.

#### Scenario: One account receives fewer requests
- **WHEN** a concurrency report shows unequal account distribution
- **THEN** telemetry identifies the exact periods and reasons that the account was ineligible or affinitized traffic changed the expected sequence

### Requirement: Tiered capacity acceptance
Each launch tier SHALL have reproducible evidence stating model mix, prompt/output sizes, streaming/tool ratio, request rate, duration, account pools, upstream limits, completions/failures, latency percentiles, retries/fallback, queue, fairness, CPU, memory, and sockets.

#### Scenario: Approve the 100-task tier
- **WHEN** the team proposes enabling 100 simultaneous tasks
- **THEN** the exact deployed versions pass the defined 100-task sustained and recovery profile within approved error, latency, fairness, and resource thresholds

### Requirement: Capacity downgrade is explicit
If a higher capacity tier is not accepted, the system SHALL enforce the highest proven lower tier and expose that limit operationally; it MUST NOT describe the gap as a round-robin limitation.

#### Scenario: 100-task tier misses its SLO
- **WHEN** the 100-task profile fails but the 50-task profile passes
- **THEN** production admission is capped at 50 with a dated remediation plan and no change to the documented rotation semantics

