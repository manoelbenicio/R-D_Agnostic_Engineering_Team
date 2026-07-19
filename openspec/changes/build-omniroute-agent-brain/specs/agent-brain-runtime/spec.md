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

### Requirement: OmniRoute readiness gate
The Agent Brain SHALL fail closed for new model-dependent work when OmniRoute authentication, readiness, required protocol, or selected model capability is unavailable. It MUST NOT fall back to direct provider endpoints or provider-native credentials.

#### Scenario: OmniRoute becomes unavailable
- **WHEN** OmniRoute is not ready while a new task is admitted
- **THEN** the Agent Brain queues or rejects the task according to admission policy and reports an actionable gateway-unavailable status without launching a direct-provider path

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

