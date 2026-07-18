# Prodex-to-OmniRoute Feature Parity and Disposition

## Acceptance rule

OmniRoute replaces Prodex only after every row below has an approved disposition and evidence. “OmniRoute appears to have a similar feature” is not parity. The evidence must prove the behavior on the exact deployed OmniRoute version and approved model routes.

Statuses used during delivery:

- **BLOCKER — evidence required**: required for cutover and not yet accepted.
- **BRAIN**: product/cold-plane behavior retained or rebuilt in the brand-neutral Agent Brain, not delegated to OmniRoute.
- **OMNIROUTE**: hot-plane behavior OmniRoute must own and prove.
- **RETIRE BY DECISION**: not carried forward unless product/security explicitly approves its retirement and documents the replacement or loss.

## Core hot-path parity matrix

| ID | Prodex feature/source | Required target behavior | Target owner | Acceptance evidence | Initial status |
|---|---|---|---|---|---|
| P01 | Profile identity and isolated profile homes (`prodex-profile-identity`, shared Codex FS) | Provider accounts remain isolated; no token, cookie, auth file, or state can cross accounts. Agent Brain sees none of them. | OMNIROUTE | Storage/config review plus concurrent cross-account isolation demonstration | BLOCKER — evidence required |
| P02 | Profile export and secret store (`prodex-profile-export`, `prodex-secret-store`) | Account onboarding/export/backup is encrypted, scoped, audited, and never reveals secrets through inference/admin APIs or logs. | OMNIROUTE | Encryption/key-management design, redacted backup/restore and access-control proof | BLOCKER — evidence required |
| P03 | Gateway virtual key and authentication | One stable Agent Brain key authorizes only approved inference routes; provider keys remain internal. Key revoke/rotate is independent of provider accounts. | OMNIROUTE | Scoped-key configuration and rotate/revoke demonstration | BLOCKER — evidence required |
| P04 | Pre-commit profile selection (`prodex-runtime-proxy`, policy) | Select an eligible account before dispatch and report the pseudonymous selected connection. Selection is atomic under concurrency. | OMNIROUTE | Selection traces and 20/50/100 concurrency distribution report | BLOCKER — evidence required |
| P05 | Strict rotation | New independent requests follow strict round-robin across eligible accounts. Rotation is not a global serialization or one-request concurrency cap. | OMNIROUTE | Deterministic concurrent sequence/fairness evidence with session stickiness disabled for this route | BLOCKER — evidence required |
| P06 | Hard continuation affinity (`previous_response_id`, turn state, session ID) | Stateful continuations stay with the owning account or are safely materialized statelessly. Affinity overrides fresh rotation only for the continuation. | OMNIROUTE | Responses API/tool-turn/prompt-cache continuation demonstrations | BLOCKER — evidence required |
| P07 | Rotate before commit | Retry/fallback occurs only before accepted unary output, first committed stream output, or non-idempotent tool action. No unsafe mid-stream replay. | OMNIROUTE | Failure injection before and after first output, with deduplication proof | BLOCKER — evidence required |
| P08 | Bounded fallback/retry budget | Attempts and deadlines are bounded, cancellation stops retries, and terminal error retains the correct status/retry metadata. | OMNIROUTE | Config plus timeout/connection/5xx/429 failure-injection results | BLOCKER — evidence required |
| P09 | Quota adapters (`prodex-quota`, `prodex-runtime-quota`) | Track rate/token/subscription/model quota; distinguish account exhaustion from context-window errors; skip/cool down exhausted accounts. | OMNIROUTE | Per-provider quota-source mapping and quota-exhaustion demonstration | BLOCKER — evidence required |
| P10 | Guarded redeem/reset-claim | Where providers support credit/reset consumption, perform it only under explicit policy, before commit, with idempotency, audit, grace windows, and post-action quota recheck. | OMNIROUTE | Supported-provider matrix and no-credit/credit/near-reset/all-exhausted demonstrations | BLOCKER — evidence required |
| P11 | OAuth refresh and token expiry | Refresh proactively with clock skew and single-flight locking; on refresh failure quarantine only the account and safely fall back. | OMNIROUTE | Expired access token and revoked refresh token demonstrations | BLOCKER — evidence required |
| P12 | 401/403 classification | Separate expired, revoked, entitlement, disabled subscription, model access, and policy denial; avoid infinite refresh/rotation loops. | OMNIROUTE | Injected 401/403 cases with resulting account/model state | BLOCKER — evidence required |
| P13 | Rate-limit backoff and circuit breaker | Classify account/model/provider/global 429; honor reset headers; open scoped circuits; half-open probe and recover; expose earliest retry when all accounts are blocked. | OMNIROUTE | Account-scoped and provider-global 429 demonstrations plus breaker telemetry | BLOCKER — evidence required |
| P14 | Provider fallback/adaptive routing | Same-model/account fallback first; cross-model/provider fallback only by approved ordered policy, without silent capability reduction. Adaptive routing must not violate affinity/quota/safety. | OMNIROUTE | Versioned route/fallback config and actual-model response telemetry | BLOCKER — evidence required |
| P15 | Anthropic runtime translation (`prodex-runtime-anthropic`) | Claude Code Anthropic Messages, content blocks, tools, thinking, errors, usage, and SSE remain lossless on approved routes. | OMNIROUTE | Protocol conformance fixtures on every Claude/agy route | BLOCKER — evidence required |
| P16 | OpenAI runtime/gateway | Codex OpenAI Responses API—including streaming lifecycle, function calls, reasoning, continuation and errors—is supported. Chat Completions alone is insufficient. | OMNIROUTE | Codex custom-provider protocol conformance on every approved model | BLOCKER — evidence required |
| P17 | Gemini/Antigravity compatibility (`prodex-runtime-gemini*`) | Exact direct Antigravity contract is documented, or approved `agy/...` models work through Claude/Codex fallback with equivalent tools/reasoning/streaming. | OMNIROUTE | Endpoint/version proof plus per-model capability matrix | BLOCKER — evidence required |
| P18 | Kimi, GLM and NVIDIA adapters | OpenAI-compatible Responses or Chat contracts preserve roles, tools, reasoning, streaming, errors and usage; Kimi ACP boundary is correctly handled by the local CLI adapter. | OMNIROUTE + BRAIN adapter | Per-model protocol proof and CLI configuration contract | BLOCKER — evidence required |
| P19 | Capability discovery (`prodex-runtime-capabilities`, provider core) | Machine-readable model registry reports protocol family, context, tools, reasoning, structured output, streaming and availability. Unsupported fields are rejected, never dropped silently. | OMNIROUTE | Versioned `/v1/models` metadata or equivalent capability registry | BLOCKER — evidence required |
| P20 | Tool/MCP continuation integrity (`prodex-mcp-stdio`) | Tool call IDs, schemas, argument deltas, results, ordering and continuation survive translation/affinity. Unsafe account switching during a tool turn fails closed. | OMNIROUTE for model protocol; BRAIN for local MCP process | Parallel-tool and multi-turn tool conformance demonstrations | BLOCKER — evidence required |
| P21 | Streaming commit integrity | SSE event order, heartbeat, backpressure, usage, cancellation and partial-stream failure semantics match each CLI. | OMNIROUTE | Long reasoning stream, slow client, cancel, broken stream and restart evidence | BLOCKER — evidence required |
| P22 | Hot-path nonblocking I/O | Selection/commit does not require blocking broad disk/database reads or synchronous state saves; state persistence is bounded and merge-safe. | OMNIROUTE | Architecture/code evidence plus selection latency/resource measurements | BLOCKER — evidence required |
| P23 | Runtime state/store (`prodex-runtime-state`, `prodex-state`, runtime-store) | Persist account health, quota, circuits, continuation bindings and route state safely; define single-node vs multi-worker backend and backup/recovery. | OMNIROUTE | State topology, consistency rules, backup/restore and restart recovery | BLOCKER — evidence required |
| P24 | Runtime broker/registry (`prodex-runtime-broker*`) | Registry exposes route/model/account eligibility, health and metrics without credentials; routing does not depend on stale local Agent Brain state. | OMNIROUTE | Health/registry API and account add/remove under load | BLOCKER — evidence required |
| P25 | Runtime policy (`prodex-runtime-policy`) | Versioned, validated, atomic policy covers routes, accounts, concurrency, retries, fallback, context optimization, logging and kill switches; reject stale revisions. | OMNIROUTE; BRAIN selects approved policy ID | Policy schema and apply/rollback/idempotency proof | BLOCKER — evidence required |
| P26 | Kill switches | Tenant/key, route, provider, model, account and feature kill switches take effect before the next new request and are audited. | OMNIROUTE; BRAIN also stops product tasks | Scoped kill-switch demonstrations during load | BLOCKER — evidence required |
| P27 | Health/readiness | Separate liveness/readiness; readiness fails closed for unusable config/state/auth/routing dependencies, while provider degradation is represented precisely. | OMNIROUTE | Health contract and injected dependency failures | BLOCKER — evidence required |
| P28 | Runtime events and route decisions | Emit idempotent, schema-versioned, redacted lifecycle, selection, fallback, quota, circuit, tool and kill-switch evidence with request/task/session correlation. | OMNIROUTE | Event schema/headers, duplicate handling and redaction proof | BLOCKER — evidence required |
| P29 | Metrics, logs and audit (`prodex-runtime-metrics`, runtime-log, audit-log) | Provide bounded structured telemetry for latency, usage, errors, rotation, retries, circuits, account utilization and configuration changes. | OMNIROUTE; BRAIN aggregates product view | Metric catalog, sample redacted records, retention and alert rules | BLOCKER — evidence required |
| P30 | Redaction and PII protection (`prodex-redaction`, `prodex-presidio`) | Keys/tokens/cookies/raw content are never logged by default; define whether PII scanning/redaction of enabled diagnostic content is supported. | OMNIROUTE | Redaction fixtures including nested headers/errors and PII policy | BLOCKER — evidence required |
| P31 | Runtime cookies | If any provider uses cookies, isolate/encrypt them and never relay them northbound. If unsupported, declare affected routes unavailable. | OMNIROUTE | Cookie-route inventory and isolation/redaction proof | BLOCKER — evidence required |
| P32 | Request idempotency | Duplicate client retries cannot duplicate inference billing or tool actions when outcome is ambiguous; configuration operations are idempotent. | OMNIROUTE + BRAIN request IDs | Timeout/retry deduplication and conflict demonstration | BLOCKER — evidence required |
| P33 | Capacity and overload | Sustain declared 20/50/100 task profiles, configurable global/route/model/account concurrency, bounded queues and deterministic overload errors. | OMNIROUTE inference limits; BRAIN task admission | Reproducible tiered load and recovery report | BLOCKER — evidence required |
| P34 | Model catalog/cost and usage (`prodex-provider-core`, app reports) | Return actual model/provider route, normalized usage and cost inputs without revealing account identity; version pricing/source assumptions. | OMNIROUTE provides; BRAIN aggregates | Response headers/events plus reconciliation example | BLOCKER — evidence required |

## Smart Context/token-saver parity

Smart Context is a substantial Prodex feature, not a synonym for normal prompt truncation. If OmniRoute is to replace all Prodex hot-path features, the following are mandatory or require an explicit product decision to defer cutover.

| ID | Prodex behavior | Required OmniRoute behavior | Acceptance evidence | Initial status |
|---|---|---|---|---|
| SC01 | Segment classification | Classify system/control, continuation, tools/functions, user content, repository/context and appendices independently. | Design and deterministic fixture corpus | BLOCKER — evidence required |
| SC02 | Exact protocol preservation | Preserve roles, ordering, control fields, continuation fields, tool IDs/results, JSON structure, mandatory references and explicit exact-mode fields. | Byte/semantic conformance fixtures | BLOCKER — evidence required |
| SC03 | Structural validation | Reject optimized payloads with invalid JSON, broken tool relationships, empty mandatory content, duplicate appendices, lost critical signals or invalid allocations. | Negative/self-check fixtures | BLOCKER — evidence required |
| SC04 | Whole-request exact fallback | On unsupported input, panic, failed self-check or structural risk, send the original/minified-equivalent request—not a degraded rewrite. | Fault injection proving original-body fallback | BLOCKER — evidence required |
| SC05 | Shadow mode | Compute savings/quality telemetry while always dispatching the original request. | Shadow comparison evidence | BLOCKER — evidence required |
| SC06 | Canary rollout | Deterministically apply optimization only to configured percentage/routes; canary-out requests pass through unchanged. | Rollout distribution and unchanged-body proof | BLOCKER — evidence required |
| SC07 | Live-mode safety | Dispatch a rewritten request only after all capability and structural self-checks pass. | Approved fixture/replay benchmark | BLOCKER — evidence required |
| SC08 | Continuation/cache integrity | Optimization does not invalidate provider continuation state, prompt-cache keys, account affinity or tool turns. | Multi-turn/cache/tool demonstrations | BLOCKER — evidence required |
| SC09 | Savings and quality telemetry | Report original/optimized tokens, reduction, decision reason, fallback reason and validation outcome without logging content. | Redacted event/metric samples | BLOCKER — evidence required |
| SC10 | Immediate kill switch | Disable optimization before the next request without daemon restart; original requests continue. | Live kill-switch demonstration | BLOCKER — evidence required |

If OmniRoute does not currently implement SC01–SC10, the architect must provide one of two written outcomes:

1. a dated OmniRoute implementation plan that keeps Smart Context in the hot data plane; or
2. an explicit product waiver accepting temporary loss of token-saver functionality.

Smart Context must not be moved casually into the Agent Brain because payload rewriting is a hot-path concern and would recreate split routing ownership.

## Cold-plane and local-runtime features retained in Agent Brain

These Prodex-adjacent responsibilities are not valid reasons to put provider credentials or account rotation back into the daemon.

| ID | Existing capability/source | Target disposition | Required result |
|---|---|---|---|
| B01 | Runtime launch planning (`prodex-runtime-launch`) | BRAIN | Brand-neutral CLI launch plans, isolated environment, working directory, limits and cancellation |
| B02 | CLI/app/config primitives (`prodex-core`, codex-config, shared types, app/CLI) | BRAIN | Neutral configuration/contracts; `CLIKind` separated from `RouteModel`; temporary legacy aliases |
| B03 | Task/session/workspace lifecycle | BRAIN | Retain proven orchestration, watchdog, stream batching, repository/worktree and recovery behavior |
| B04 | Local MCP stdio process management | BRAIN | Launch/manage local MCP servers; OmniRoute still preserves model-side tool protocol |
| B05 | Product policy, budget and task kill switch | BRAIN | Decide allowed route/model/budget and stop tasks; never choose provider accounts |
| B06 | Product evidence/ledger | BRAIN | Aggregate OmniRoute redacted evidence under task/project/tenant IDs |
| B07 | Memory MCP backend (`prodex-memory`) | BRAIN or approved external memory service | Preserve only if product uses it; document privacy, tenancy, deletion and redaction |
| B08 | Terminal UI/update notices/housekeeping/bench support | BRAIN/CLI tooling as needed | Reimplement only user-visible or operational behavior still required by the new product |

## Features requiring explicit retirement approval

| ID | Feature | Default disposition | Required decision |
|---|---|---|---|
| R01 | Prodex Caveman/plugin hook assets | RETIRE BY DECISION | Remain disabled because arbitrary hooks create an RCE boundary; any future plugin system needs a separate sandboxed design |
| R02 | Prodex-owned terminal rendering | RETIRE BY DECISION | Coding-agent CLIs and the product UI own presentation; confirm no required operator workflow is lost |
| R03 | Prodex-specific file/SQLite state | RETIRE BY DECISION | OmniRoute’s supported state backend becomes authoritative; no shared Prodex state survives |
| R04 | Prodex/RPP L2 sidecar contract and binary | RETIRE BY DECISION after parity | Remove only after OmniRoute contract, evidence ingestion and rollback are operational |
| R05 | Legacy Go credential rotation/provider homes | RETIRE BY DECISION after cutover | Delete after gateway-required mode proves no direct-provider fallback and legacy tasks are drained |

## Crate/source coverage cross-check

The parity rows cover the known 44-crate audit groups:

- Foundation/config/app: `prodex-core`, `prodex-shared-types`, `prodex-codex-config`, `prodex-cli`, `prodex-app`, reports, housekeeping, update notice and bench support → B02/B08.
- Identity/secrets/state: shared Codex FS, profile identity/export, secret store, session store, runtime state/state/runtime-store → P01–P03, P06, P23.
- Routing/runtime: runtime proxy/config/policy/launch/tuning/broker/log/quota/provider core/capabilities → P04–P14, P19, P22–P27, P33–P34, B01.
- Protocol providers: Anthropic, Gemini/CLI compatibility, Claude, MCP stdio → P15–P21, B04.
- Context: `prodex-context` → SC01–SC10.
- Evidence/security: audit log, runtime metrics/log, redaction, Presidio, cookies → P28–P31.
- Optional/special surfaces: memory, Caveman assets, terminal UI → B07, R01, R02.

No source group is considered complete because it appears in this cross-check. Completion requires the row-level evidence and approved target disposition.

## Final parity sign-off

The two architects must sign a final copy with:

- OmniRoute version/image digest and deployment configuration revision;
- Agent Brain change/version and adapter versions;
- status and evidence link for every P, SC, B and R row;
- approved exceptions, owner, remediation date and operational restriction;
- capacity tier approved for launch;
- rollback trigger and maximum recovery time;
- confirmation that direct provider credentials, Prodex routing and legacy Go rotation are disabled in the accepted request path.
