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

### Requirement: Login credential slot retention (24h, login-based)
Per-login credential slot directories (`~/.agent-cred-homes/slots/slot-<N>`) SHALL be retained for at most 24h and then destroyed automatically, so credential material does not accumulate and exhaust disk. Retention is **login-based and time-based**, NOT task-based: a slot is never deleted merely because a task finished (a task may run from minutes to hours), and deletion is driven only by the slot's age.

#### Scenario: Slot exceeds 24h and is idle
- **WHEN** a login credential slot is older than 24h and no live process is using it (no running process has its cwd or HOME inside the slot)
- **THEN** the slot directory is deleted automatically on the next scheduled sweep, with no manual cleanup

#### Scenario: Slot exceeds 24h but is still in use
- **WHEN** a login credential slot is older than 24h but a live process still references it
- **THEN** the slot is preserved (never deleted out from under a running agent) and is eligible for deletion only once it is no longer in use

### Requirement: Secret-safe diagnostics
Logs, metrics, traces, health and errors SHALL exclude credentials, authorization values, cookies, prompts, tool payloads, repository content and account identity.

#### Scenario: Upstream error contains sensitive fields
- **WHEN** an error crosses the gateway boundary
- **THEN** Main Brain emits only a bounded redacted classification and safe correlation metadata