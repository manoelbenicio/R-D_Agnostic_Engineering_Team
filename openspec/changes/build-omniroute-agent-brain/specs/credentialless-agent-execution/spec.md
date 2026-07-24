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

### Requirement: Secret-safe diagnostics
Logs, metrics, traces, health and errors SHALL exclude credentials, authorization values, cookies, prompts, tool payloads, repository content and account identity.

#### Scenario: Upstream error contains sensitive fields
- **WHEN** an error crosses the gateway boundary
- **THEN** Main Brain emits only a bounded redacted classification and safe correlation metadata