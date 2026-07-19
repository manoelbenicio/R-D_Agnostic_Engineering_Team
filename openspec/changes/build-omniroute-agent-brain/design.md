## Context

The active runtime is a Go daemon running on WSL/host, not the Docker backend container. It executes host-installed coding-agent CLIs and currently mixes product orchestration with provider-account resolution, credential-home preparation, legacy Go rotation, and partial Prodex/L2 integration hooks. OmniRoute already runs in Docker on host port `20128`, has reachable model/account data, and is intended to become the sole owner of credentials, subscriptions, rotation, quota and hot-path routing.

The source inventory shows that the proven orchestration engine is valuable but highly coupled: `daemon.go` is a central integration hotspot; configuration and wire contracts contain many Multica names; agent selection conflates CLI frontend with provider; and `execenv` can copy or inherit provider-native credentials. A blind rewrite/rename would break the current control API, CLI, stored configuration, task briefs, workspace behavior and recovery paths.

The authoritative supplier and parity documents for this design are:

- `architecture.md`: discovered AS-IS and proposed TO-BE diagrams and ownership boundary.
- `omniroute-architecture-acceptance-checklist.md`: detailed OmniRoute protocol, rotation, expiry, quota, 429, security, telemetry and capacity prerequisites.
- `prodex-omniroute-feature-parity.md`: full Prodex responsibility disposition, including Smart Context/token saving and reset/redeem.

## Goals / Non-Goals

**Goals:**

- Deliver a brand-neutral Agent Brain that preserves working cold-plane orchestration while separating it from model routing and provider credentials.
- Make OmniRoute the exclusive hot router and sole provider credential/subscription owner.
- Support Claude Code, Codex, Kimi, GLM/NVIDIA and Antigravity model routes through explicit, protocol-complete adapters.
- Remove direct provider-key inheritance/copying and enforce one stable, scoped OmniRoute secret.
- Preserve or explicitly disposition every Prodex feature before removal.
- Scale through independently configured 20, 50 and 100 simultaneous-task tiers without confusing round-robin policy with concurrency.
- Enable four parallel implementation streams with non-overlapping file ownership and one integrator for shared hotspots.
- Migrate and debrand incrementally with measurable compatibility usage and safe rollback.

**Non-Goals:**

- Rewriting proven repository, workspace, task lifecycle, cancellation, watchdog, local skill, or result-streaming code solely to change naming.
- Keeping Prodex as a second router or moving provider credentials back into Agent Brain.
- Treating `GET /v1/models` success as end-to-end protocol or feature acceptance.
- Claiming native `agy` or Kimi support before the installed CLI/provider configuration proves an endpoint/protocol override.
- Running unbounded load or enabling the 100-task tier before the exact deployed topology meets its acceptance profile.
- Completing final product naming in the critical integration path; `Agent Brain` is a neutral working name until the product name is approved.

## Decisions

### 1. Extract a neutral core with a compatibility shell

Create new neutral interfaces and packages around the existing orchestration behavior, then progressively move implementations behind them. Keep the old daemon/API/config names as explicit adapters while consumers migrate. Record alias use so deletion is evidence-based.

Alternative considered: wholesale rewrite and global rename. Rejected because the existing daemon contains mature lifecycle, recovery, repository and streaming behavior and is tightly coupled to active wire/API/config consumers. It would lengthen the critical path and create unrelated regressions.

Proposed logical boundaries:

- `brain`: coordinator, task executor, runtime registry, admission and result streamer.
- `gateway`: OmniRoute client, readiness/model catalog, trusted runtime profiles and telemetry correlation.
- `runtimeenv`: isolated process environment, workspace context/skills and controlled CLI configuration without provider auth.
- `cli`: executable/parsing adapters keyed by `CLIKind`.
- compatibility adapters: legacy API, env/config and command aliases translating into neutral contracts.

The first phase may use new neutral packages inside the existing Go module. Renaming the Go module, binary and stored paths is a later convergence step after functional cutover.

### 2. Separate `CLIKind`, `RouteModel` and `RouterOwner`

The task contract will identify:

- `CLIKind`: Claude Code, Codex, Kimi, native Agy, or another executable frontend.
- `RouteModel`: the exact OmniRoute model ID, independent of the CLI vendor label.
- `RouterOwner`: `omniroute` in the target path.
- `TaskID`, `SessionID`, `RequestID` and optional continuation identifier for correlation/affinity.

This permits Claude Code to use `agy/...` or Kimi model routes where the protocol is compatible and prevents the daemon from resolving a provider credential based on a CLI name.

Alternative considered: retain the existing `provider` enum as the primary key. Rejected because it conflates process protocol, model origin and credential/account ownership.

### 3. OmniRoute is the only hot data plane

Agent Brain sends model intent and correlation only. OmniRoute owns provider credentials, account pools, rotation, continuation affinity, token refresh, quota/subscription state, 429/circuit behavior, bounded pre-commit retry/fallback, protocol translation, Smart Context/token saving, reset/redeem where retained, and hot-path evidence.

Legacy Go rotation, credential account selection and provider-home auth copying are disabled for gateway-required tasks and deleted only after drain/parity gates. Prodex/L2 routing is disabled for gateway-required tasks and quiesced to a default-OFF, mutually-exclusive cold recovery mode (retained, not deleted — D-V3-16).

Alternative considered: keep Prodex or Go rotation as fallback. Rejected because dual ownership can select different accounts, reintroduce credential overwrites, duplicate retry, and make failures non-deterministic.

**Addendum (Wave A, 2026-07-19, D-V3-16) — Prodex cold recovery mode.** Prodex is *not* deleted and is *not* an automatic or per-request fallback. It is retained only as a default-OFF, mutually-exclusive, operator-gated **cold platform recovery mode** in the final Kanban lane. There is always exactly one hot router owner: enabling recovery mode requires OmniRoute to be quiesced first, and restoring OmniRoute requires Prodex to be drained first. Transitions occur only at session boundaries, never mid-flight, preserving the `rpp.l2.v1` single-router invariant. OmniRoute unavailability produces a fail-closed DEGRADED state (queue/reject), never an auto-promotion of Prodex. See the platform recovery-mode state machine (AB-REQ-41) and `docs/contract/single-router-invariant.md` recovery-mode addendum.

### 4. Use protocol-specific adapters

The initial adapter contracts are:

- Claude Code: `ANTHROPIC_BASE_URL=http://127.0.0.1:20128` and a trusted `ANTHROPIC_AUTH_TOKEN` derived from the stable OmniRoute secret. The root URL does not include `/v1`.
- Codex: generate a controlled per-task `config.toml` custom provider with base URL `http://127.0.0.1:20128/v1`, an environment key dedicated to OmniRoute, `wire_api="responses"`, no provider-native auth flow, HTTP/SSE transport, and task/session correlation headers.
- OpenAI-compatible Kimi/GLM/NVIDIA: use an explicitly proven Responses or Chat Completions adapter and exact OmniRoute model ID; do not assume that setting `OPENAI_BASE_URL` alone configures every native CLI.
- Kimi: confirm the installed Kimi provider registry/config contract. ACP controls the local agent; it is not the upstream model HTTP contract. Until proven, a Claude/Codex compatible frontend with a Kimi route is the safe fallback.
- Antigravity: confirm a native endpoint override. Until proven, use Claude Code or Codex with approved `agy/...` model IDs; do not depend on the Windows MITM inside Linux/WSL/container runtimes.
- NIM: either convert the native backend into a generic gateway adapter or point its configurable base URL at OmniRoute while ensuring any legacy `NVIDIA_API_KEY` slot contains only the OmniRoute key and is never overwritten by per-account preparation.

The active host runtime uses loopback. If execution later moves to Docker, deployment selects Docker DNS or host-gateway explicitly; endpoint selection is runtime topology, not a hard-coded universal value.

### 5. Build child environments in a trusted order

Environment construction will be:

1. start from a minimal inherited environment;
2. remove all provider-native secrets and direct endpoint variables;
3. add safe task/workspace/context variables;
4. validate and merge allowed custom settings;
5. generate isolated CLI config/home without auth copies;
6. apply trusted OmniRoute base URL, stable key reference and correlation variables last;
7. validate that no denied credential or direct route remains before process start.

The stable key source will be a Linux permission-restricted secret file or equivalent service secret derived operationally from the existing host source. Its value is never placed in committed Compose/config files, logs, screenshots, task events or general daemon diagnostics.

Alternative considered: set gateway variables globally on the daemon only. Rejected because existing per-task homes/config and custom environment can still retain or override direct-provider behavior.

### 6. Strict rotation and continuation affinity are separate policies

Strict round-robin advances on each new independent logical request. It does not advance per SSE event, internal retry, tool block, or token. It is concurrency-safe and does not serialize in-flight requests.

Stateful continuation affinity takes precedence only when a request depends on provider-owned state such as `previous_response_id`, turn state, prompt cache or an active tool turn. The OmniRoute architect must document whether each protocol pins the continuation or materializes stateless context. Session stickiness is disabled for strict independent-request routes unless a route explicitly requires affinity.

### 7. Preserve rotate-before-commit and tool safety

OmniRoute may automatically retry/fallback only before user-visible output or a potentially non-idempotent tool action. Broken streams after partial output are surfaced, not replayed. Same-model account fallback precedes cross-model/provider fallback. Cross-model fallback is explicit and cannot silently lose context, tool, reasoning, structure or safety capabilities.

### 8. Prodex parity is a signed gate

The feature-parity matrix is part of the architecture contract. In particular, basic account rotation does not replace Smart Context/token saving, reset/redeem, hard affinity, protocol transforms, MCP/tool continuation, redaction, broker/state, policy, audit or metrics.

If OmniRoute lacks a required hot-path feature, the decision is one of:

- implement it in OmniRoute with acceptance evidence;
- approve a time-bounded product/security waiver and operational restriction; or
- defer Prodex removal.

Moving a hot-path feature into Agent Brain is not the default because it would recreate split router ownership.

### 9. Capacity is layered and evidence-based

Agent Brain owns task admission; OmniRoute owns inference/account concurrency. Global, route, model and per-account limits are independent. The launch tier is the highest profile—20, 50 or 100 simultaneous tasks—that passes a reproducible mix of streaming, tools, prompt sizes and provider routes on the deployed resources.

Queues are bounded, overload is deterministic, and cancellation releases all task/request/account capacity. Capacity may be raised without changing strict round-robin semantics.

### 10. Observability uses common correlation without content leakage

Agent Brain creates task/session/request IDs. Adapters pass them through accepted headers/metadata. OmniRoute returns its request ID, actual model/route, pseudonymous account/connection, selection reason, retries/fallback, quota/circuit state and usage where safe. Both layers emit structured metrics/events without keys, cookies, raw prompts, repository content, tool payloads or opaque reasoning.

Readiness gates required model/protocol capability. Health and metrics distinguish authentication, no eligible account, quota, 429/circuit, upstream failures, protocol errors, cancellation and local overload.

**Addendum (Wave A, 2026-07-19, D-V3-17) — full eight-hop E2E correlation and the G4-OBS stop-gate.** The correlation above is extended from Brain↔OmniRoute to a single metadata-only trace spanning eight hops: (1) ingress control API `request_id`; (2) DB queue `queue_msg_id`; (3) daemon admission/lifecycle `task_id`/`session_id`/`launch_id`; (4) CLI process `proc_id`; (5) OmniRoute/provider `omni_request_id` + actual route/model + pseudonymous account/connection; (6) terminal persistence `result_id`; (7) WS/UI delivery `delivery_id`; (8) assembled trace. Every hop carries the join keys and emits labels/counters only — no bodies, prompts, tool payloads, repository content, reasoning, cookies, keys, account emails or connection strings — under a versioned schema with the `secrets_present=false` invariant. A blocking **G4-OBS** stop-gate (tasks OBS-1..OBS-11, capability `end-to-end-observability`) requires a continuous synthetic trace per task and a structural leak-clean scan, and must pass before any capacity tier (§9) or cutover (§10).

### 11. Four-agent implementation topology

Implementation proceeds in dependency-aware waves. Parallel work reduces effort substantially, but the shared daemon integration and live acceptance remain serial critical paths.

| Agent | Exclusive workstream | Primary ownership | Must not edit |
|---|---|---|---|
| Codex 1 — Lead integrator | Neutral contracts, compatibility facade, configuration and final wiring | Shared daemon entrypoint, central config, health, task contract, merge/review | Other agents' new packages during their active changes |
| Codex 2 — OmniRoute gateway | Gateway client, readiness/models, protocol profiles, correlation and route policy types | New gateway package and fixtures | Central daemon/config entrypoints |
| Codex 3 — Runtime/CLI security | Environment sanitizer, controlled task homes/config, Claude/Codex/OpenAI/Kimi/NIM/Agy adapter behavior | `execenv`/runtime environment and CLI adapter packages | Central daemon/config entrypoints |
| Codex 4 — Operations/parity | Deployment/secret handling, dashboards/alerts, capacity/failure harness, migration/rollback and evidence matrices | Deploy/operations/observability/evidence files and tools | Daemon, gateway and runtime adapter implementation |

Wave 0 (30–45 minutes) freezes interfaces, file ownership and acceptance IDs. Wave 1 runs the four streams in parallel. Wave 2 lets the lead integrator wire completed modules through the sole shared hotspot. Wave 3 performs provider/protocol canaries and 20-task acceptance; higher tiers and full parity follow based on OmniRoute evidence. No agent edits the same central file concurrently.

**Addendum (Wave A, 2026-07-19, D-V3-18) — eight-lane zero-overlap topology.** For the remaining program (G4 acceptance, G4-OBS, capacity, recovery-mode disposition, sibling closure) the four streams are expanded to eight lanes with pairwise-disjoint ownership. Owner/co-lead sit above the lanes: Kiro/Opus-4.8 = planning/adjudication; Codex#56#A = transport/independent verification (no product edits).

| Lane | Role | Exclusive ownership (globs) | Must-not-touch |
|---|---|---|---|
| W1 | Lead Integrator (central wiring, recovery-mode state machine, OBS-4) | `internal/daemon/{daemon,config,health,cmd_daemon}.go`, `go.mod`, `execenv/**`, `pkg/agent/models.go`, `prodex*.go`, `l2_runtime.go`, `brain/**` | any other lane's new package |
| W2 | OmniRoute Gateway (8.1/8.4/8.5/8.6/8.7 gateway side, OBS-6) | `internal/daemon/gateway/**` | central hotspots, other packages |
| W3 | Runtime/CLI Security (8.2/8.3, child-env isolation, OBS-5; 5.6–5.8 stay fail-closed) | `internal/daemon/runtimeenv/**`, `pkg/agent/{claude,codex,kimi,nim,antigravity}.go` (coordinated) | central hotspots, gateway |
| W4 | Ops/Capacity/Evidence (8.8, 9.x harness, OBS-11 dashboards/bundle) | `internal/daemon/deploy/**`, `internal/daemon/observability/dashboards/**`, harness specs, runbooks, `EVIDENCE_INDEX.md` | daemon/gateway/runtime impl |
| W5 | E2E Correlation library + leak-scan (OBS-1/OBS-9/OBS-10) | `internal/daemon/observability/e2e/**` (new lib) | callers' own files |
| W6 | Ingress + WS/UI delivery instrumentation (OBS-2, OBS-8) | frozen HTTP ingress middleware file(s) + WS transport file(s) | `squad_briefing*.go`, daemon hotspots |
| W7 | Queue + terminal-persistence instrumentation (OBS-3, OBS-7) | frozen task-queue repo file(s) + terminal-result store file(s) | daemon hotspots, handler |
| W8 | Governance + Prodex cold-recovery disposition + sibling closure drafting | OpenSpec change docs, parity/removal drafts, sibling reopened-task evidence | product-code hotspots; GSD authored by Kiro |

Zero-overlap proof: (1) W1–W5 own pairwise-disjoint package globs (∩ = ∅ by construction); (2) cross-cutting spans are added by each file's owner **calling** the W5 `observability/e2e` library, never co-editing it; (3) W6/W7 own specific frozen files removed from every other glob; (4) any file two lanes would need is escalated to W1 and serialized across waves, never concurrent; (5) before dispatch the planning owner publishes the ownership matrix and Codex#56#A runs a glob-intersection check (each path matches exactly one lane) recorded as `EV-ZERO-OVERLAP`.

### 12. OpenSpec and GSD have separate mandatory roles

OpenSpec is the authoritative product/change contract: scope, architecture, normative requirements, scenarios, parity, and implementation tasks. GSD is the authoritative execution system: milestone phases, owners, file locks, live state, dependencies, decisions, evidence, and release gates.

Before Wave 0, the project will create an Agent Brain v3 GSD baseline with bidirectional traceability from every component and interface through requirement, OpenSpec task, GSD phase/task, owner and evidence. The existing RPP/Prodex v2.1 GSD remains historical and cannot be treated as an active concurrent plan.

The current governance says only Kiro/Principal authors `.planning/`. Kiro must author the new baseline or the product owner must explicitly change that governance. Codex agents do not silently overwrite the existing GSD.

## Risks / Trade-offs

- [Existing daemon is highly coupled] → Extract neutral boundaries first; make one lead the only owner of central daemon/config files.
- [A full rename breaks active consumers] → Keep measured compatibility aliases and debrand internal/new surfaces first; remove aliases after migration telemetry reaches zero.
- [OmniRoute protocol support differs by route] → Require per-model, per-protocol conformance; do not generalize from one successful request.
- [Strict round-robin conflicts with continuation state] → Define independent-request rotation and explicit continuation affinity separately; prove stateful cases.
- [Custom/inherited environment bypasses OmniRoute] → Deny provider variables and inject trusted configuration last with a pre-launch assertion.
- [Stable key leaks through Windows file permissions or child inheritance] → Stage it in a restricted service secret and expose it only to authorized inference children.
- [Smart Context is absent or behaviorally different] → Treat SC01–SC10 as cutover blockers or obtain a signed waiver; do not hide the gap under basic routing.
- [Automatic retry duplicates output/tool actions] → Enforce pre-commit replay only, bounded idempotency and explicit partial-stream failures.
- [100-task goal exceeds account/provider or host limits] → Measure layered capacity, enforce the highest proven tier, and tune/add accounts/resources without changing rotation semantics.
- [OmniRoute becomes a single point of failure] → Readiness/admission gating, bounded queues, versioned state backup, monitored restart, previous-version rollback and no unsafe direct fallback.
- [Four-agent merge contention erases time savings] → Exclusive directories, contract-first freeze, one central integrator and short convergence windows.

## Migration Plan

1. Approve the planning hierarchy and rebaseline GSD for Agent Brain v3, preserving RPP/Prodex v2.1 as history and creating requirement/component/interface/removal/evidence traceability.
2. Obtain OmniRoute architect responses and evidence for the acceptance checklist and feature-parity matrix. Resolve every blocker or approve an explicit waiver.
3. Freeze neutral task/gateway/runtime interfaces, model map, secret source, route policies, correlation schema, compatibility aliases and file ownership.
4. Build the new gateway package, CLI/runtime environment security and operations assets in parallel while the lead creates the neutral core/compatibility shell.
5. Wire gateway-required mode through the lead-owned daemon entrypoints. Disable legacy credential selection/copy, Prodex startup and Go rotation for that mode.
6. Validate readiness and one protocol canary per approved CLI/model family. Do not enable a route that has only `/v1/models` evidence.
7. Run failure cases for expiry, revoked auth, quota, 429 scopes, 5xx, timeout, cancellation, broken stream, account changes and restart.
8. Run the 20-task profile; then 50 and 100. Enable only the highest accepted tier and preserve reports.
9. Make gateway-required mode default for new tasks; drain legacy tasks; monitor direct-route/compatibility usage and parity evidence.
10. Quiesce Prodex/L2 to a default-OFF, mutually-exclusive cold recovery mode (retain, do NOT delete — D-V3-16); remove legacy Go rotation, credential-home auth copying and provider-key configuration after the removal gates pass.
11. Migrate control API, CLI, configuration, paths, metrics and UI to neutral branding; remove compatibility aliases only when usage is zero and rollback no longer depends on them.

Rollback never restores provider keys or dual routing in Agent Brain. It selects the previous accepted Agent Brain/OmniRoute release/config, reduces or stops new admission, and drains/stops affected tasks according to policy.

## Open Questions

- What final product/binary/config prefix replaces the working `Agent Brain` name?
- Which exact OmniRoute version/image digest will be the supported baseline?
- Which OmniRoute Smart Context/token-saver behaviors already exist, and which require implementation?
- Which providers/routes support guarded reset/redeem, and is exact Prodex parity required for launch?
- Does the installed Kimi CLI support the required custom upstream provider, or will the first release use a compatible Claude/Codex frontend?
- Does a supported native Agy CLI version expose a base URL, or is the Claude/Codex `agy/...` fallback the accepted permanent design?
- What are the approved cross-model/provider fallback chains and capability-equivalence rules?
- What latency/error/resource thresholds approve the 20, 50 and 100 capacity tiers?
- Which OmniRoute admin/state/backup topology is required beyond the current single local container?
- How long must legacy API/env/config aliases remain, and who owns each consumer migration?
