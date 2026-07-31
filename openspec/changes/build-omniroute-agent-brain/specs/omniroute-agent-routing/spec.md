## ADDED Requirements

### Requirement: Transport binding authority is exclusive
The system SHALL accept exactly one `TransportBinding` for model work: `omniroute` or
`native_credential_home`. Unknown, historical, missing, or multiple values MUST be rejected.

#### Scenario: OmniRoute is selected
- **WHEN** a task declares `omniroute`
- **THEN** OmniRoute is the sole inference router and account/credential owner
- **AND** no native home is selected, resolved, or supplied

#### Scenario: Native credential home is selected
- **WHEN** a task declares `native_credential_home`
- **THEN** R3 resolves one approved exclusive opaque home for one existing logical runtime/agent
- **AND** OmniRoute is not probed, contacted, or used

#### Scenario: Task declares invalid authority
- **WHEN** a task declares neither exact binding, both bindings, or an unknown value
- **THEN** validation fails before readiness probing or process launch

### Requirement: Strict OmniRoute selected-route readiness
Admission for an `omniroute` binding SHALL require OmniRoute liveness, authentication,
model-registry readiness, selected-model readiness and selected-protocol readiness. This probe
MUST NOT run for `native_credential_home`.

#### Scenario: Selected protocol is unavailable
- **WHEN** OmniRoute is live but the selected protocol is not ready
- **THEN** Main Brain rejects the task with a deterministic capability/readiness status and starts no CLI

### Requirement: No direct, alternate, or cross-binding fallback
For an `omniroute` binding, Main Brain MUST NOT route directly to a provider, start another
router, select an account, or retry through another provider/model path when OmniRoute is
unavailable or returns an error. For `native_credential_home`, it MUST NOT use OmniRoute or
select/rotate to another home. Neither binding may fallback, translate, retry, or remap to the
other.

#### Scenario: Gateway becomes unavailable
- **WHEN** OmniRoute readiness fails for an `omniroute` task
- **THEN** new admission for that task closes and remains closed until the same OmniRoute route
  is ready
- **AND** native execution is not attempted

#### Scenario: Native assignment becomes invalid
- **WHEN** R3 assignment or native catalog health is stale, missing, ambiguous, or conflicting
- **THEN** native admission closes
- **AND** OmniRoute or another credential home is not attempted

### Requirement: Correlated transport request
Approved adapters SHALL propagate safe task/session/request correlation and SHALL return bounded
transport/result metadata without credentials, raw paths, account identity, or task content.

#### Scenario: Transport result returns
- **WHEN** an admitted OmniRoute or native request finishes
- **THEN** Main Brain can correlate the result to its task/session and terminal result without
  account identity, source path, or secret material