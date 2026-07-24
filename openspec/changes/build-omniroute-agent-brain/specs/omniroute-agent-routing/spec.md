## ADDED Requirements

### Requirement: OmniRoute is the only router owner
The system SHALL accept only `omniroute` as `RouterOwner` for model work. Unknown, historical or alternate values MUST be rejected.

#### Scenario: Task declares another owner
- **WHEN** a task contract declares any router owner other than `omniroute`
- **THEN** validation fails before readiness probing or process launch

### Requirement: Strict selected-route readiness
Admission SHALL require OmniRoute liveness, authentication, model-registry readiness, selected-model readiness and selected-protocol readiness.

#### Scenario: Selected protocol is unavailable
- **WHEN** OmniRoute is live but the selected protocol is not ready
- **THEN** Main Brain rejects the task with a deterministic capability/readiness status and starts no CLI

### Requirement: No direct or alternate fallback
Main Brain MUST NOT route directly to a provider, start an alternate router, select an account, or retry through another provider/model path when OmniRoute is unavailable or returns an error.

#### Scenario: Gateway becomes unavailable
- **WHEN** OmniRoute readiness fails
- **THEN** new model admission closes and remains closed until OmniRoute readiness is restored

### Requirement: Correlated gateway request
Approved adapters SHALL propagate safe task/session/request correlation and SHALL return bounded route/result metadata without credentials or task content.

#### Scenario: Gateway result returns
- **WHEN** an admitted request finishes
- **THEN** Main Brain can correlate the result to its task/session and terminal result without account identity or secret material