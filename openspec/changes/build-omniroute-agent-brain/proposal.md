## Why

The current daemon mixes Multica-specific branding, provider credential preparation, and agent execution concerns, while OmniRoute already provides the hot-path account, subscription, routing, and rotation capabilities. We need a brand-neutral main brain now so every coding-agent request uses one stable OmniRoute credential and the platform can scale concurrent work without duplicating hot-path logic.

## What Changes

- Introduce a brand-neutral Agent Brain daemon as the cold control plane for tasks, workspaces, sessions, agent processes, policy, and lifecycle management.
- Make OmniRoute the sole hot data plane for provider credentials, account rotation, quota/failover decisions, and model routing.
- Add a credentialless runtime contract: the daemon and its agents receive only one OmniRoute secret and never store or rotate provider-native credentials.
- Add explicit runtime adapters for Claude Code, Codex, Kimi, GLM/NVIDIA, and Antigravity-compatible model routes, with per-task model and session correlation.
- Support configurable capacity tiers of at least 20, 50, and 100 simultaneous tasks; strict round-robin selection policy is independent from concurrency limits.
- Extract reusable orchestration behavior from the current daemon into new brand-neutral packages and commands, retaining a temporary compatibility adapter while consumers migrate.
- Add health gating, telemetry, secure secret injection, deployment, rollback, and migration controls for the host daemon and containerized OmniRoute topology.
- Add a mandatory, blocking end-to-end metadata-only observability stop-gate (G4-OBS, OBS-1..OBS-11) that must pass before any capacity tier or cutover: a single correlated trace across all eight hops (ingress API → DB queue → daemon → CLI → OmniRoute/provider → terminal persistence → WS/UI delivery) with per-hop redaction, dashboards/alerts, continuous synthetic trace assembly and a leak-clean acceptance. (Owner decision D-V3-17.)
- Establish dual OpenSpec/GSD governance with bidirectional requirement, component, interface, task, owner, evidence, and removal traceability before implementation begins.
- Supersede Prodex **as a request-path router**: Prodex is never a per-request or automatic hot fallback and is never simultaneously hot with OmniRoute. Prodex and its Rust sidecar are **retained** — not deleted — solely as a default-OFF, mutually-exclusive, operator-gated **cold platform recovery mode** relocated to the final Kanban lane. (Owner decision D-V3-16.)
- **BREAKING** Remove provider-account assignment and native provider-key injection from the new Agent Brain execution contract.
- **BREAKING** Deprecate Multica-branded daemon configuration, environment variables, and APIs after a bounded compatibility period.

## Capabilities

### New Capabilities

- `agent-brain-runtime`: Brand-neutral daemon lifecycle, task orchestration, workspace/session ownership, and compatibility boundaries.
- `omniroute-agent-routing`: Per-CLI routing, model selection, request/session correlation, and OmniRoute health gating.
- `credentialless-agent-execution`: Single-secret injection, provider-key exclusion, task isolation, and secret-safe observability.
- `parallel-agent-capacity`: Capacity tiers, admission control, concurrency behavior, failure isolation, and measurable performance targets.
- `brain-cutover-operations`: Deployment, migration, compatibility, rollback, and removal of Multica/Prodex runtime dependencies.
- `end-to-end-observability`: Mandatory blocking G4-OBS stop-gate — full eight-hop metadata-only correlation, per-hop redaction, continuous synthetic trace assembly, leak-clean acceptance, and dashboards/alerts across the whole request path.

### Modified Capabilities

No mainline OpenSpec capabilities exist yet. The historical change artifacts remain evidence inputs; `persist-prodex-runtime-integration` is **deferred and re-scoped to cold-recovery-only** (default-OFF, mutually-exclusive, operator-gated) rather than promoted into the target request path or deleted. See D-V3-16.

## Impact

- Affects the current Go daemon under `multica-auth-work/server`, its command/config surface, task execution environment, credential-account preparation, agent backends, and deployment scripts.
- Introduces new brand-neutral daemon and OmniRoute adapter modules, with one integration owner for shared daemon entrypoints to avoid parallel merge conflicts.
- Changes runtime connectivity to the host OmniRoute endpoint (`http://127.0.0.1:20128` for the current host daemon) and requires secure injection of the single OmniRoute key.
- Retains existing task/workspace/session behavior where it is product-neutral, while renaming or retiring Multica-specific types, configuration, logs, metrics, and user-visible contracts.
- Requires capacity validation at 20, 50, and 100 simultaneous tasks, plus provider-path acceptance for Claude, Codex, Kimi, GLM/NVIDIA, and Antigravity models.
- Requires a new Agent Brain GSD baseline; the existing RPP/Prodex v2.1 `.planning` documents remain historical until explicitly superseded and preserved.
