# Spec: Multica Gateway selection and lifecycle

## ADDED Requirements

### Requirement: GW-SEL-REQ-01 Deterministic independent selection

The preserved Multica Selector MUST select independent requests through one atomic,
bounded rotation order and MUST emit a monotonic per-Selector sequence.

#### Scenario: Concurrent independent requests

- **WHEN** concurrent independent requests select from the same eligible set
- **THEN** each selection MUST occupy one deterministic sequence position
- **AND** only eligible accounts MUST be returned

### Requirement: GW-SEL-REQ-02 Continuation affinity is explicit and fail-closed

Continuation ownership MUST be created only after a successful response identifies the
produced continuation handle. Affinity hits MUST NOT advance independent rotation.

#### Scenario: Continuation owner remains eligible

- **WHEN** a known continuation handle is reused
- **THEN** selection MUST return its bound account
- **AND** the independent cursor MUST remain unchanged

#### Scenario: Owner is unknown or ineligible

- **WHEN** origin-account affinity cannot resolve an eligible owner
- **THEN** selection MUST fail closed
- **AND** MUST NOT silently bind another account

### Requirement: GW-LIFE-REQ-03 Lifecycle state is bounded

Account order, eligibility, and continuation bindings MUST remain bounded. Quarantined,
cooldown, removed, or unknown accounts MUST NOT be selected.

#### Scenario: Account re-enters eligibility

- **WHEN** an existing account changes from an ineligible state to eligible
- **THEN** it MUST re-enter the deterministic eligible rotation without duplication

#### Scenario: Account is removed

- **WHEN** an account is hard-removed
- **THEN** it MUST leave the rotation
- **AND** its continuation bindings MUST be removed

### Requirement: GW-DISPATCH-REQ-04 Replay and deduplication are pre-commit bounded

Retries MUST remain within configured attempts and deadline and MUST NOT replay after
output or a non-idempotent action is committed.

#### Scenario: Duplicate request is already in flight

- **WHEN** a second execution uses the same request ID
- **THEN** it MUST follow the existing execution rather than create another upstream call

#### Scenario: Output or action is committed

- **WHEN** an attempt has committed output or a non-idempotent action
- **THEN** the dispatch Coordinator MUST NOT replay that request

### Requirement: GW-EVIDENCE-REQ-05 Wiring claims require constructor evidence

Implemented Gateway library behavior MUST NOT be described as active production
selection/lifecycle behavior unless a non-test constructor and caller are evidenced at
the same exact revision.

#### Scenario: Candidate has only direct test callers

- **WHEN** Selector and Executor are instantiated only by tests
- **THEN** documentation MUST classify their behavior as implemented but not
  production-wired

#### Scenario: Production client and readiness wiring exists

- **WHEN** production code constructs Gateway client, registry, readiness, profile, policy,
  or credential-source components without constructing Selector or Executor
- **THEN** that evidence MUST be reported as client/readiness wiring only
- **AND** MUST NOT be promoted to selection/lifecycle wiring evidence
