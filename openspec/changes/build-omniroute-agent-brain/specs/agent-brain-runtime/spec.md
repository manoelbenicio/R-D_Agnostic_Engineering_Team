## ADDED Requirements

### Requirement: Brand-neutral control-plane runtime
The system SHALL provide an Agent Brain daemon whose primary packages, command, configuration, logs, metrics, and new API contracts do not depend on Multica or Prodex product names. The daemon SHALL own only cold-plane orchestration responsibilities.

#### Scenario: Start the neutral daemon
- **WHEN** an operator starts the new daemon with its neutral configuration
- **THEN** it registers with the control plane, reports readiness, and can accept tasks without starting Prodex or a provider-account router

### Requirement: Preserve proven orchestration behavior
The Agent Brain SHALL preserve task lifecycle, workspace/repository handling, process launch, cancellation, watchdog, result streaming, recovery, context, and local skill behavior that is independent of provider credential routing.

#### Scenario: Execute a normal task
- **WHEN** the control plane assigns an approved task to a ready Agent Brain
- **THEN** the daemon prepares the workspace, launches the selected CLI, streams events, records the terminal result, and releases resources using the retained lifecycle semantics

### Requirement: Separate CLI selection from model routing
The Agent Brain SHALL model the executable frontend as `CLIKind` and the OmniRoute destination as `RouteModel`; it MUST NOT infer credential vendor or account ownership solely from the CLI name.

#### Scenario: Claude Code uses an Antigravity model route
- **WHEN** a task specifies `CLIKind=claude-code` and `RouteModel=agy/claude-opus-4-6-thinking`
- **THEN** the Agent Brain launches Claude Code with the approved OmniRoute adapter without attempting to resolve an Anthropic or Antigravity provider account

### Requirement: Opaque OmniRoute readiness gate
The Agent Brain SHALL consume only OmniRoute's readiness/capability result for new model-dependent work. It MUST NOT authenticate providers, inspect or repair credentials, classify token/account lifecycle, select accounts, or implement retry/failover. Authentication and every credential/account decision remain exclusively inside OmniRoute.

#### Scenario: OmniRoute becomes unavailable
- **WHEN** OmniRoute reports not-ready or rejects a new model request
- **THEN** the Agent Brain queues or rejects the task according to admission policy, propagates an opaque actionable gateway status, and performs no credential repair, alternate-account selection, direct-provider launch, or router fallback

### Requirement: Bounded compatibility facade
The system SHALL provide explicit, observable, time-bounded compatibility aliases for required legacy daemon API, environment, stored configuration, and CLI consumers while new neutral consumers migrate.

#### Scenario: Legacy control plane assigns a task during migration
- **WHEN** a supported legacy request reaches the compatibility facade
- **THEN** the system translates it into the neutral internal contract, records compatibility usage, and applies the same credentialless OmniRoute enforcement

### Requirement: Single hot router owner
Every active model request SHALL have `omniroute` as the only hot router owner. The Agent Brain MUST NOT run Prodex, Rust L2, or legacy Go account-selection logic for a request owned by OmniRoute.

#### Scenario: Task begins after cutover
- **WHEN** the Agent Brain launches an agent task in gateway-required mode
- **THEN** runtime evidence identifies OmniRoute as router owner and no legacy rotation, account-home selection, or Prodex sidecar is invoked


### Requirement: Production integrity without duplicate validation
Production artifacts and runtime paths SHALL NOT expose QA-only routes, mock services, placeholder entities, synthetic success responses, fake credentials, demo seed data, or in-memory control acknowledgements presented as durable success. Missing configuration or contract drift MUST fail closed with an actionable error. Test-only fixtures and safety guardrails MAY remain when they are excluded from production artifacts and have no production caller.

Acceptance SHALL reuse existing evidence when it identifies the same implementation, version/digest, configuration, route and scenario. A scenario SHALL be rerun only when affected implementation or integration changed, prior evidence is missing/stale/non-equivalent, or a distinct material risk is not covered. One live execution during serial integration MAY simultaneously satisfy implementation, integration and acceptance for every overlapping task it proves.

#### Scenario: API response violates its production schema
- **WHEN** a production API response fails schema validation
- **THEN** the client surfaces a contract error and MUST NOT fabricate an empty entity, successful mutation, zero balance, pending checkout or other fallback record

#### Scenario: Existing equivalent evidence is available
- **WHEN** accepted evidence covers the same deployed build/configuration and scenario and no relevant code or integration changed
- **THEN** the evidence is reused without a duplicate QA or live execution

#### Scenario: Production configuration is incomplete
- **WHEN** a required secret, upstream or durable control backend is absent
- **THEN** startup/readiness or the affected operation fails closed and MUST NOT substitute a known placeholder, fake key, discard-port upstream or volatile success acknowledgement
