## ADDED Requirements

### Requirement: OmniRoute-exclusive authentication and credential custody
OmniRoute SHALL be the exclusive owner of provider and northbound inference authentication, including credential creation, storage, validation, refresh, rotation, revocation, account binding, and failure recovery. The Agent Brain and Multica MUST NOT implement authentication flows, credential lifecycle, provider login, account selection, or credential failover.

If the accepted OmniRoute northbound contract requires opaque access material, that material SHALL be provisioned and managed by OmniRoute. Agent Brain MAY consume it only as an opaque transport input and MUST NOT inspect, validate, refresh, rotate, persist, or derive routing decisions from it.

#### Scenario: Prepare an agent task environment
- **WHEN** the Agent Brain prepares a task for any supported CLI
- **THEN** it supplies only non-secret route/model/adapter configuration and, only when required by the accepted OmniRoute contract, opaque OmniRoute-managed transport access; it supplies no provider-native credential or authentication state

### Requirement: Deny inherited provider credentials
Gateway-required mode SHALL prevent provider-native credential variables, authentication homes, and direct-provider endpoint overrides from reaching a child CLI. This is an execution-isolation boundary, not a credential-management implementation.

#### Scenario: Daemon inherited a provider key
- **WHEN** the parent daemon environment contains an Anthropic, OpenAI, Google, Kimi, NVIDIA, or other provider-native secret
- **THEN** that variable is absent from the child process and no Multica/Agent Brain code attempts to validate, refresh, select, or replace it

### Requirement: Trusted route intent only
User, task, and custom-agent settings MUST NOT introduce provider credentials, authentication configuration, direct-provider endpoints, account selectors, retry policy, or fallback policy. Agent Brain SHALL pass only approved `CLIKind`, `RouteModel`, correlation, and non-secret transport configuration; OmniRoute owns every authentication and hot-routing decision.

#### Scenario: Custom environment attempts direct routing
- **WHEN** a task supplies a direct provider base URL, provider key, account selector, or fallback setting
- **THEN** the launch is rejected or the setting is removed, and no alternative credential or provider route is selected by Agent Brain

### Requirement: Controlled per-CLI configuration
The Agent Brain SHALL generate only the minimum non-secret per-task CLI configuration required to target the accepted OmniRoute protocol. It MUST NOT copy shared auth files, generate provider credentials, perform login, or define provider/account fallback behavior.

#### Scenario: Prepare a Codex task
- **WHEN** Codex is selected
- **THEN** its isolated configuration declares the accepted OmniRoute Responses transport and model intent without copying `auth.json`, provider credentials, account state, or Brain-owned authentication policy

### Requirement: OmniRoute-managed access lifecycle
Any OmniRoute northbound access material SHALL be created, stored, rotated, revoked, and validated by OmniRoute-owned deployment/runtime mechanisms. Multica and Agent Brain MUST NOT own a secret store, rotation procedure, authentication acceptance test, or recovery path for that material.

#### Scenario: OmniRoute access is rejected
- **WHEN** OmniRoute rejects or cannot establish authentication
- **THEN** Agent Brain receives an opaque gateway-unavailable/error result, does not inspect or repair credentials, and does not attempt a provider-native or alternate-account path

### Requirement: Secret-safe evidence
Logs, metrics, traces, errors, events, and diagnostics from Agent Brain SHALL contain no provider credentials, OmniRoute access material, authorization headers, cookies, raw prompts, raw tool payloads, repository content, or opaque reasoning. OmniRoute owns credential-specific security evidence and lifecycle certification.

#### Scenario: Upstream error contains an authorization value
- **WHEN** an error returned across the OmniRoute boundary contains credential-like fields
- **THEN** Agent Brain records only a redacted, opaque gateway error classification and correlation metadata

### Requirement: No Brain authentication or failover acceptance lane
Multica/Agent Brain acceptance SHALL NOT execute or duplicate tests for token expiry, refresh, revocation, account quarantine, quota-based selection, 401/403 handling, 429 circuits, provider retry, or provider/account failover. Those behaviors are owned and certified by OmniRoute. Brain acceptance is limited to orchestration behavior it owns, such as admission from an opaque readiness state, process lifecycle, cancellation, terminal persistence, and honest propagation of an OmniRoute result.

#### Scenario: OmniRoute supplier evidence covers credential failover
- **WHEN** accepted OmniRoute evidence covers an authentication, credential, account, retry, or failover scenario
- **THEN** Multica/Agent Brain records the external ownership/evidence reference and performs no duplicate implementation or execution

### Requirement: Metadata-only end-to-end spans
Every Agent Brain span SHALL contain only correlation identifiers, classifications, counters, and latencies. It MUST NOT contain provider credentials, OmniRoute access material, authorization headers, cookies, raw prompts, raw tool payloads, repository content, opaque reasoning, account emails, or connection strings. CLI argv MUST be redacted structurally (shape only, never values).

#### Scenario: A hop span is emitted for a task carrying sensitive content
- **WHEN** an Agent Brain hop emits a span for a task whose request or result contains credential-like or content fields
- **THEN** the span records only metadata and correlation identifiers, argv is redacted structurally, and credential lifecycle details remain exclusively inside OmniRoute-owned evidence
