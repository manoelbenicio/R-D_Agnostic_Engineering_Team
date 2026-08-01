## ADDED Requirements

### Requirement: Binding-scoped child environment
Main Brain SHALL start every child from a sanitized ambient environment. For `omniroute`, it
SHALL remove provider-native credentials, auth homes, and direct-provider endpoints and MAY
inject only opaque OmniRoute transport access from a restricted reference. For
`native_credential_home`, it SHALL inject no OmniRoute endpoint or secret and SHALL supply only
daemon-local isolated-home references required by the native CLI after R3 resolves the pinned
exclusive opaque home. It MUST NOT expose a global HOME, source raw path, account identity, or
credential data through product APIs, events, logs, or evidence.

#### Scenario: OmniRoute parent contains provider credentials
- **WHEN** an `omniroute` launch inherits provider-native secret variables or auth paths
- **THEN** they are absent from the child environment and no native home is resolved

#### Scenario: Native launch is prepared
- **WHEN** a `native_credential_home` launch has a fresh valid R3 assignment
- **THEN** only the isolated daemon-local home references needed by that CLI are supplied
- **AND** OmniRoute is not probed, contacted, or used

### Requirement: Controlled CLI configuration
Per-task CLI configuration SHALL target only the pinned binding. `omniroute` configuration SHALL
target the approved OmniRoute protocol/model with no copied provider auth file. Native
configuration SHALL preserve the pinned exclusive `home_ref`, use only required daemon-local
isolated-home references, and define no OmniRoute endpoint. Neither mode may define the other as
an alternate endpoint or fallback.

#### Scenario: Codex task home is prepared for OmniRoute
- **WHEN** Main Brain prepares an `omniroute` Codex task
- **THEN** the isolated home has controlled gateway configuration and no copied provider
  `auth.json`

#### Scenario: Codex task home is prepared for native execution
- **WHEN** Main Brain prepares a `native_credential_home` Codex task
- **THEN** no source auth file is copied, moved, or rewritten
- **AND** the pinned binding and home cannot be rotated or replaced for that task

### Requirement: Custom settings cannot override trust
Task or user custom environment and arguments MUST NOT override the pinned transport binding,
OmniRoute URL/auth or route identity, native opaque home assignment or isolated-home reference,
correlation, task identity, provider credentials, or direct endpoints.

#### Scenario: Custom environment requests direct routing
- **WHEN** custom settings contain a provider key or direct base URL
- **THEN** pre-launch validation rejects the task before resolving a restricted transport
  reference or creating a child process

### Requirement: Source credential-home preservation
Source credential homes SHALL never be copied, moved, deleted, truncated, sanitized,
overwritten, chmodded, have ownership changed, or be automatically destroyed based on age.
Catalog TTL/retention may retire or tombstone metadata only. Cleanup is permitted only for
task-local non-source material after active-reference checks prove no live task, process,
snapshot, binding, or reconciliation operation references it.

#### Scenario: Source home exceeds a retention deadline
- **WHEN** a source credential home is older than any catalog TTL or retention deadline
- **THEN** only catalog metadata may become retired or tombstoned
- **AND** the source directory and contents remain untouched

#### Scenario: Task-local non-source material is eligible for cleanup
- **WHEN** task-local generated material is beyond retention and active-reference checks are clear
- **THEN** only that non-source material may be cleaned
- **AND** any active or ambiguous reference blocks cleanup

### Requirement: Secret-safe diagnostics
Logs, metrics, traces, health and errors SHALL exclude credentials, authorization values, cookies, prompts, tool payloads, repository content and account identity.

#### Scenario: Transport error contains sensitive fields
- **WHEN** an error crosses either transport boundary
- **THEN** Main Brain emits only a bounded redacted classification and safe correlation metadata