## ADDED Requirements

### Requirement: Provider-credentialless child environment
Main Brain SHALL remove provider-native credentials, auth homes and direct-provider endpoints from the child environment. It MAY inject only opaque OmniRoute transport access from a restricted reference after sanitization.

#### Scenario: Parent contains provider credentials
- **WHEN** the daemon inherited provider-native secret variables or auth paths
- **THEN** they are absent from the child environment and no replacement provider credential is selected

### Requirement: Controlled CLI configuration
Per-task CLI configuration SHALL target the approved OmniRoute protocol/model and SHALL NOT copy shared provider auth files or define an alternate endpoint/fallback.

#### Scenario: Codex task home is prepared
- **WHEN** Main Brain prepares a Codex task
- **THEN** the isolated home has controlled gateway configuration and no copied provider `auth.json`

### Requirement: Custom settings cannot override trust
Task or user custom environment and arguments MUST NOT override gateway URL/auth, route identity, correlation, task identity, provider credentials or direct endpoints.

#### Scenario: Custom environment requests direct routing
- **WHEN** custom settings contain a provider key or direct base URL
- **THEN** pre-launch validation rejects the task before reading the gateway secret or creating a child process

### Requirement: Credential slot cardinality (binding-based, not age-based)
Credential slot directories (`~/.agent-cred-homes/slots/slot-<N>`) SHALL be reconciled to exactly
one directory per stable active binding, with zero historical directories. Retention is
**binding-based**, NOT age-based and NOT task-based: a slot is never deleted because it is old,
and never deleted merely because a task finished. Non-reuse of a retired identity is guaranteed by
persisted tombstone metadata, which is never deleted by a retention deadline. Deletion of physical
directories is a policy-layer operation; the credential catalog performs no physical folder
creation, copy, delete, or history behavior.

#### Scenario: Slot has no stable active binding
- **WHEN** a credential slot directory corresponds to no stable active binding and no live process
  references it
- **THEN** the directory is removed by explicit stale reconciliation, and the corresponding
  tombstone metadata is retained for non-reuse

#### Scenario: Slot is still referenced by a live process
- **WHEN** a credential slot directory is referenced by a live process
- **THEN** the slot is preserved regardless of age, and becomes eligible for reconciliation only
  once it is no longer referenced

#### Scenario: Process audit is unavailable
- **WHEN** production reconciliation cannot perform a privileged all-process `/proc` audit
- **THEN** reconciliation fails closed and removes nothing

### Requirement: Secret-safe diagnostics
Logs, metrics, traces, health and errors SHALL exclude credentials, authorization values, cookies, prompts, tool payloads, repository content and account identity.

#### Scenario: Upstream error contains sensitive fields
- **WHEN** an error crosses the gateway boundary
- **THEN** Main Brain emits only a bounded redacted classification and safe correlation metadata