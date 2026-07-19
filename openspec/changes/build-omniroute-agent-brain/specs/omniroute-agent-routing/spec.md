## ADDED Requirements

### Requirement: Protocol-complete CLI adapters
The system SHALL route Claude Code through Anthropic Messages, Codex through OpenAI Responses, and approved Kimi/GLM/NVIDIA/Antigravity frontends through an explicitly supported OmniRoute protocol. Each adapter MUST preserve streaming, tools, reasoning, usage, errors, cancellation, and required continuation fields.

#### Scenario: Agent uses tools during a streamed response
- **WHEN** an approved CLI sends a streamed model request containing tool schemas and receives one or more tool calls
- **THEN** OmniRoute preserves event order, call IDs, argument JSON, tool results, continuation state, terminal status, and usage in the client-native format

### Requirement: Model capability contract
OmniRoute SHALL expose a versioned model map or capability registry declaring each approved model ID's protocol family, context limit, streaming, tool, reasoning, structured-output, account pool, rotation/affinity, and fallback capabilities. Unsupported fields MUST be rejected explicitly rather than dropped silently.

#### Scenario: Task requests an unsupported capability
- **WHEN** the Agent Brain requests a model/feature combination not declared by the registry
- **THEN** admission fails with a deterministic capability error before the CLI sends provider traffic

### Requirement: Concurrency-safe strict round-robin
For routes configured as strict round-robin, OmniRoute SHALL select the next eligible account for each new independent logical request atomically under concurrency. Rotation order MUST be independent of streaming chunks, internal retries, tool blocks, and task-capacity limits.

#### Scenario: Simultaneous independent requests arrive
- **WHEN** multiple independent requests arrive concurrently on a strict round-robin route
- **THEN** each request receives the next eligible account according to one atomic sequence without global request serialization or selection races

### Requirement: Continuation affinity
OmniRoute SHALL preserve the account ownership required by `previous_response_id`, turn state, prompt caches, provider conversation state, and tool continuation. Affinity SHALL override fresh selection only for the dependent continuation.

#### Scenario: Stateful continuation follows a rotated first request
- **WHEN** a second request references provider state created by an earlier request
- **THEN** OmniRoute routes it to the owning account or uses a documented stateless continuation mechanism while unrelated requests continue round-robin selection

### Requirement: Pre-commit recovery
OmniRoute SHALL bound retries and fallbacks by attempts and end-to-end deadline and SHALL perform automatic replay only before user-visible output or a potentially non-idempotent tool action. Same-model healthy-account fallback SHALL precede cross-model/provider fallback.

#### Scenario: Stream fails after partial output
- **WHEN** an upstream stream fails after output has been delivered
- **THEN** OmniRoute reports the partial-stream failure with correlation data and does not silently replay the request

### Requirement: Credential and quota lifecycle
OmniRoute SHALL proactively refresh expiring tokens, single-flight concurrent refresh, classify 401/403, track provider quota/reset state where available, quarantine invalid accounts, and select another eligible account without exposing credential material.

#### Scenario: Selected account token has expired
- **WHEN** the selected account's access token expires before dispatch
- **THEN** OmniRoute refreshes it once safely or quarantines the account and performs an allowed pre-commit fallback

### Requirement: 429 and circuit-breaker handling
OmniRoute SHALL classify 429 responses by account, model, provider/global, or local-overload scope; honor retry metadata; apply bounded jittered backoff; open and recover scoped circuit breakers; and avoid account thrashing during provider-global throttling.

#### Scenario: One account repeatedly returns 429
- **WHEN** one account crosses the configured account-scoped 429 threshold
- **THEN** its circuit opens, other eligible accounts continue serving requests, and a bounded half-open probe controls re-entry

### Requirement: Smart Context parity
If Prodex is removed without a product waiver, OmniRoute SHALL provide protocol-safe Smart Context/token optimization with segment classification, structural validation, shadow mode, canary rollout, exact whole-request fallback, continuation/tool integrity, redacted savings telemetry, and an immediate kill switch.

#### Scenario: Optimized payload fails structural validation
- **WHEN** Smart Context cannot prove that roles, ordering, continuation fields, tool relationships, mandatory references, and JSON structure remain valid
- **THEN** OmniRoute dispatches the original or exact-equivalent request and records a redacted fallback reason

### Requirement: Reset/redeem parity
For routes where credit/reset consumption is supported and required for Prodex parity, OmniRoute SHALL perform it only through explicit policy, before commit, with idempotency, grace-window protection, account-pool checks, audit evidence, and a post-action quota recheck.

#### Scenario: Another account still has quota
- **WHEN** the selected account is exhausted but another eligible account has quota
- **THEN** OmniRoute rotates to the eligible account and does not consume reset credit

### Requirement: End-to-end request correlation
Each CLI adapter SHALL propagate the platform correlation identifiers (`request_id`, `task_id`, `session_id`, `launch_id`, `proc_id`) on every model request and SHALL surface OmniRoute's `omni_request_id`, actual route/model and pseudonymous account/connection so the request can be joined into a single end-to-end trace. Correlation data MUST be metadata only and MUST NOT carry prompts, tool payloads, repository content, reasoning, secrets, cookies, or account emails.

#### Scenario: Request is traced across hops
- **WHEN** a task's model request flows from the daemon through the CLI adapter to OmniRoute and back
- **THEN** the adapter emits the inbound correlation IDs and the returned `omni_request_id`/route/model/pseudonymous-connection so downstream observability can assemble one continuous trace without any request content

