## ADDED Requirements

### Requirement: Single OmniRoute secret
The Agent Brain and its child CLIs SHALL receive at most one scoped OmniRoute inference secret and MUST NOT receive provider-native API keys, OAuth tokens, refresh tokens, cookies, or copied provider authentication homes.

#### Scenario: Prepare an agent task environment
- **WHEN** the Agent Brain prepares a task for any supported CLI
- **THEN** the resulting environment and task home contain only the approved OmniRoute secret plus non-secret adapter configuration

### Requirement: Deny inherited provider credentials
Gateway-required mode SHALL remove provider-native credential variables and direct-provider endpoint overrides from the inherited daemon environment before a child CLI is launched.

#### Scenario: Daemon inherited a provider key
- **WHEN** the parent daemon environment contains an Anthropic, OpenAI, Google, Kimi, NVIDIA, or other provider-native secret
- **THEN** that variable is absent from the child process and a redacted policy event records its removal

### Requirement: Trusted routing configuration wins
User, task, and custom-agent environment settings MUST NOT override OmniRoute authentication, base URLs, provider configuration, transport safety, or request-correlation variables. Trusted adapter values SHALL be applied after untrusted/custom settings.

#### Scenario: Custom environment attempts direct routing
- **WHEN** a task supplies a direct provider base URL or provider key in custom environment settings
- **THEN** validation rejects or removes the setting and the trusted OmniRoute configuration remains effective

### Requirement: Controlled per-CLI configuration
The Agent Brain SHALL generate controlled per-task CLI configuration where environment variables alone are insufficient, including a Codex custom provider using the Responses API. It MUST NOT copy shared auth files or uncontrolled provider definitions into the task home.

#### Scenario: Prepare a Codex task
- **WHEN** Codex is selected
- **THEN** its isolated configuration declares the OmniRoute provider/base URL/key variable and HTTP Responses transport without copying `auth.json` or a direct-provider configuration

### Requirement: Secret source and permissions
The stable OmniRoute secret SHALL be injected through an operating-system-appropriate restricted secret mechanism and MUST NOT be committed, embedded in images, printed, screenshotted, or stored in world-readable task/config files.

#### Scenario: Daemon reads the host secret
- **WHEN** the service starts with the configured OmniRoute secret reference
- **THEN** it validates restricted access, loads the secret without logging its value, and exposes it only to authorized inference child processes

### Requirement: Secret-safe evidence
Logs, metrics, traces, errors, events, and diagnostics from both Agent Brain and OmniRoute SHALL redact stable and provider secrets, authorization headers, cookies, raw prompts, raw tool payloads, repository content, and opaque reasoning unless an explicit audited content policy permits them.

#### Scenario: Upstream error includes an authorization value
- **WHEN** a provider or adapter error contains credential-like fields
- **THEN** all emitted evidence replaces those values with redacted markers while retaining safe correlation and classification data

### Requirement: Fail-closed credential policy
The runtime SHALL NOT offer a direct-provider fallback when OmniRoute authentication fails or no eligible OmniRoute route exists.

#### Scenario: Stable OmniRoute key is invalid
- **WHEN** OmniRoute rejects the stable key
- **THEN** the task fails or waits with a gateway-authentication error and no provider-native login or credential discovery is attempted

