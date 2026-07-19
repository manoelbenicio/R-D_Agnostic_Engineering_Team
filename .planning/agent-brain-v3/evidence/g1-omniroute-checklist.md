# EV-G1-OPS-PREP — OmniRoute architecture acceptance checklist

- Prepared by: Codex 4 — Operations/Parity/Evidence
- Scope: G1 documentary classification and operations plan; no live certification or deployment change
- Authorization: architect response §7.1, Waves 0–3, tier 20
- Source checklist: `openspec/changes/build-omniroute-agent-brain/omniroute-architecture-acceptance-checklist.md`
- Supplier response: `openspec/changes/build-omniroute-agent-brain/OMNIROUTE_ARCHITECT_RESPONSE.md`

## Baseline and redacted configuration

| Field | Recorded value | Acceptance |
|---|---|---|
| OmniRoute version | `3.8.48` | Identified |
| Image | `diegosouzapw/omniroute:latest` | Mutable tag; not acceptable for cutover |
| Image digest | `NOT PROVIDED` | Blocker: pin and record immutable digest before protocol/capacity evidence |
| Runtime | Container `omniroute`, port `20128` | Identified |
| State | `better-sqlite3`, single node, persistent volume `omniroute-data` | Single-instance only; horizontal strict rotation not supported as deployed |
| Docker network | User-defined network `multica_default`; in-network service name `omniroute` | Identified; topology-specific endpoint required |
| Inference authentication | API-key requirement enabled; missing credential returned 401 and an authorized request returned 200 | Credential value/header capture intentionally omitted; scope, rotation, and management separation remain unproven |
| Route policy | Strict RR uses one independent logical request as unit; `stickyRoundRobinLimit=1`; Kimi is a documented session-sticky exception with failure-triggered rotation | Concurrency and continuation proofs pending |
| 429/circuit | Account/model/provider/global/overload classifiers, cooldowns, jittered backoff, half-open breaker, persisted rate-limit time identified by code review | Failure-injection certification pending |
| Smart Context | Compression subsystem identified | SC01–SC10 acceptance or signed waiver pending |
| Capacity | Layered global/model/account limits and bounded queue controls identified | Tier 20 is authorized for canary only; no tier report exists |

Status meanings: **Supported** = the response supplies direct documentary or observed evidence for the item. **Partial** = relevant implementation/configuration is identified but required exact-route, live, failure, or operational evidence is missing. **Not-supported** = absent as evidenced, explicitly unsupported in the deployed topology, or still only a future Agent Brain control.

## 1. Ownership boundary

| ID | Checklist item | Status | Evidence / redacted note | Owner and close action |
|---|---|---|---|---|
| AC-1.1 | OmniRoute solely owns provider credentials, health, quota, rotation, retry, and provider fallback | Partial | Target boundary is approved and OmniRoute has account/route state; accepted Agent Brain path has not yet proved legacy owners disabled | Codex 1 + Codex 3: Wave-2/3 no-dual-owner evidence |
| AC-1.2 | Agent Brain/CLIs receive one scoped OmniRoute credential and no provider-native secrets | Not-supported | Runtime sanitizer and controlled task homes are not implemented or inspected at runtime | Codex 3 + Codex 4: environment/task-home/process evidence |
| AC-1.3 | Hot add/remove/disable/quarantine/re-enable without Brain restart | Partial | Account selection/quarantine components identified; active-load demonstration absent | OmniRoute operator + Codex 2/4: account-change test |
| AC-1.4 | Atomic route/model/pool update | Not-supported | No atomic revision/apply proof supplied | OmniRoute architect: versioned atomic policy and rollback proof |
| AC-1.5 | Version/digest/config/state/backup/upgrade/rollback identified | Partial | Version, container, state, and volume known; digest, backup, and runbooks missing | OmniRoute operator + Codex 4: pin digest and produce runbooks |

## 2.1 Common endpoints

| ID | Checklist item | Status | Evidence / redacted note | Owner and close action |
|---|---|---|---|---|
| AC-2.1.1 | Authenticated `GET /v1/models` returns every selectable model | Partial | Authenticated connectivity/model data reported; allow-list completeness not captured | Codex 2 + Codex 4: redacted model-list fixture |
| AC-2.1.2 | Model list/registry exposes protocol, context, capabilities, availability | Not-supported | Complete machine-readable capability registry not supplied | OmniRoute architect + Codex 2: versioned registry |
| AC-2.1.3 | Unknown model returns deterministic 4xx without substitution | Not-supported | No negative fixture supplied | Codex 2 + Codex 4: synthetic unknown-model test |
| AC-2.1.4 | JSON/UTF-8/gzip/keep-alive/cancel/large contexts preserve content | Partial | Endpoint implementations exist; large/cancel/transport conformance absent | Codex 2 + Codex 4: protocol fixture suite |

## 2.2 Anthropic Messages for Claude Code

| ID | Checklist item | Status | Evidence / redacted note | Owner and close action |
|---|---|---|---|---|
| AC-2.2.1 | Root base URL resolves exactly to `POST /v1/messages` | Supported | Route and current Claude root configuration identified; root excludes `/v1` | Codex 3: preserve trusted root in adapter |
| AC-2.2.2 | Claude-emitted Bearer authentication is accepted | Partial | API-key enforcement and a current Claude completion are reported; exact redacted header fixture absent | Codex 2/3/4: synthetic-auth fixture |
| AC-2.2.3 | Core Anthropic request fields preserved | Partial | Translators identified; exact-model conformance pending | Codex 2 + Codex 4: request echo/fixture |
| AC-2.2.4 | Text/image/tool/thinking/cache blocks preserved | Partial | Translator code identified; block-by-block proof absent | Codex 2 + Codex 4: block corpus |
| AC-2.2.5 | Tool schemas, choice, parallel calls, IDs/results round-trip | Partial | Family support claimed; tool fixtures absent | Codex 2 + Codex 4: parallel-tool fixture |
| AC-2.2.6 | Non-stream response shape, stop reason, and usage preserved | Partial | Family endpoint exists; exact-route proof absent | Codex 2 + Codex 4: non-stream fixture |
| AC-2.2.7 | Anthropic SSE ordering/deltas/final usage valid | Partial | SSE route exists; per-model event conformance absent | Codex 2 + Codex 4: SSE fixture |
| AC-2.2.8 | Anthropic errors preserve status/code/retry metadata | Partial | Error translation exists by review; injected proof absent | Codex 2 + Codex 4: error fixtures |

## 2.3 OpenAI Responses for Codex

| ID | Checklist item | Status | Evidence / redacted note | Owner and close action |
|---|---|---|---|---|
| AC-2.3.1 | `POST /v1/responses`, stream and non-stream | Partial | Responses route and state policy identified; conformance pending | Codex 2 + Codex 4: both-mode fixtures |
| AC-2.3.2 | Requests preserve inputs, tools, metadata, and reasoning controls | Partial | Route exists; field corpus absent | Codex 2 + Codex 4: request fixture |
| AC-2.3.3 | Function calls/IDs/arguments/outputs remain lossless | Partial | No exact-model delta/round-trip evidence | Codex 2 + Codex 4: multi-call fixture |
| AC-2.3.4 | Reasoning fields and opaque continuation are preserved or rejected | Partial | Capability declarations and exact-model behavior missing | OmniRoute architect + Codex 2: registry and fixture |
| AC-2.3.5 | Responses SSE event families/order are Codex-compatible | Partial | SSE implementation identified; event conformance missing | Codex 2 + Codex 4: Codex SSE fixture |
| AC-2.3.6 | `previous_response_id` behavior is safe | Partial | `auto`, `strip`, `preserve` and affinity-pin code identified; origin pin unproven | Codex 2 + Codex 4: live continuation proof |
| AC-2.3.7 | Session/request/trace correlation is accepted and returned | Not-supported | No end-to-end correlation fixture supplied | Codex 1/2 + Codex 4: correlation contract/test |
| AC-2.3.8 | WebSocket proven or disabled; HTTP/SSE mandatory | Partial | Target Codex profile disables WS and requires HTTP/SSE; controlled profile not implemented | Codex 2/3: provider config fixture |

## 2.4 OpenAI Chat Completions

| ID | Checklist item | Status | Evidence / redacted note | Owner and close action |
|---|---|---|---|---|
| AC-2.4.1 | Chat stream/non-stream for Kimi, GLM, NVIDIA | Partial | Route and operational combos identified; exact-model tests missing | Codex 2 + Codex 4: per-model fixtures |
| AC-2.4.2 | Roles, multipart, tools, structure, reasoning, limits preserved | Partial | Family route exists; field preservation unproven | Codex 2 + Codex 4: field corpus |
| AC-2.4.3 | Stream text/reasoning/tool deltas/errors/usage preserved | Partial | Per-model streaming proof absent | Codex 2 + Codex 4: SSE corpus |
| AC-2.4.4 | Capability differences declared per model | Not-supported | Versioned exact capability map absent | OmniRoute architect + Codex 2: registry |

## 2.5 Kimi, NVIDIA/GLM, and Antigravity

| ID | Checklist item | Status | Evidence / redacted note | Owner and close action |
|---|---|---|---|---|
| AC-2.5.1 | Kimi CLI/provider upstream contract identified; ACP not confused with HTTP | Not-supported | Chat family identified, but installed Kimi provider-registry override is unproven | Codex 3 + OmniRoute architect: prove adapter or approve compatible frontend |
| AC-2.5.2 | NVIDIA/GLM fidelity and stable usage/errors per exact model | Partial | Chat route and combos identified; exact rows lack proof | Codex 2/3/4: per-model conformance |
| AC-2.5.3 | Antigravity exact direct endpoint/auth and all approved models | Partial | `/v1/antigravity` and four-account 200 sequence proven for one model only; schema/all-model proof absent | OmniRoute architect + Codex 2/4 |
| AC-2.5.4 | Same Agy models work through Claude/Codex fallback | Not-supported | Compatible fallback has no proof | Codex 2/3/4: exact-model fallback fixtures |
| AC-2.5.5 | Versioned approved cross-family model map | Not-supported | Model-route evidence records unresolved IDs/capabilities/context | OmniRoute architect + Codex 1/2: approve registry |

## 3. Rotation and account selection

| ID | Checklist item | Status | Evidence / redacted note | Owner and close action |
|---|---|---|---|---|
| AC-3.1 | Rotation unit is one independent logical request | Supported | Explicitly documented; not SSE/retry/tool/token | Codex 2/4: retain in concurrency proof |
| AC-3.2 | Strict routes use next eligible account without hidden preference | Supported | Strict RR and `stickyRoundRobinLimit=1` documented for single instance | Codex 2/4: certify exact routes |
| AC-3.3 | Stickiness disabled for strict routes and exceptions documented | Supported | Strict policy plus session-sticky Kimi exception documented | Codex 1: approve exception set |
| AC-3.4 | Limit-one RR proven under concurrent arrival | Partial | Single-process atomicity reasoned; simultaneous-arrival test absent | Codex 2 + Codex 4: concurrent sequence test |
| AC-3.5 | Rotation remains concurrent at 20/50/100 | Partial | No one-request architectural bottleneck identified; tier evidence absent | Codex 2 + Codex 4: tier tests |
| AC-3.6 | Per-account/model concurrency independently configurable | Supported | `concurrencyPerModel` and `max_concurrent` identified | Codex 4: capture redacted effective config |
| AC-3.7 | At-limit account is skipped without cursor corruption | Partial | Eligibility model exists; boundary demonstration absent | Codex 2 + Codex 4: saturation fixture |
| AC-3.8 | Pool changes are safe for in-flight requests | Partial | Hot state exists; active-load add/remove proof absent | OmniRoute operator + Codex 2/4 |
| AC-3.9 | Dependent continuation affinity overrides only that request | Partial | Policy and pin component identified; live proof missing | Codex 2 + Codex 4: continuation/cache/tool tests |
| AC-3.10 | Rotation telemetry exposes route and pseudonymous connection | Not-supported | No accepted redacted sample supplied | Codex 2 + Codex 4: telemetry fixture |

## 4. Token, expiry, quota, and subscriptions

| ID | Checklist item | Status | Evidence / redacted note | Owner and close action |
|---|---|---|---|---|
| AC-4.1 | Proactive refresh with clock skew | Partial | Refresh subsystem referenced; timing behavior not demonstrated | OmniRoute architect + Codex 4: expiry test |
| AC-4.2 | Refresh is single-flight per account | Partial | Explicit mandatory gate; proof absent | OmniRoute architect + Codex 2/4: concurrent refresh test |
| AC-4.3 | Expiry between selection/dispatch safely refreshes pre-output | Partial | Intended policy; injected boundary test absent | Codex 2 + Codex 4 |
| AC-4.4 | Failed refresh quarantines only account and falls back safely | Partial | Quarantine/fallback concepts identified; revoked-token proof absent | Codex 2 + Codex 4 |
| AC-4.5 | 401 classified and bounded re-auth attempted once | Partial | Classifier architecture exists; failure injection absent | Codex 2 + Codex 4 |
| AC-4.6 | 403 entitlement/access/policy classes are distinct | Partial | Required classifier behavior documented; injected proof absent | Codex 2 + Codex 4 |
| AC-4.7 | Quota/subscription/reset/entitlement tracked | Partial | Quota exhaustion cooldown and Codex reset component identified; provider map absent | OmniRoute architect + Codex 4 |
| AC-4.8 | Exhausted account rotates and re-enters after reset/probe | Partial | Cooldown/persisted limit time identified; live exhaustion/re-entry absent | Codex 2 + Codex 4 |
| AC-4.9 | Context overflow is not account exhaustion | Not-supported | No deterministic classification fixture supplied | Codex 2 + Codex 4 |
| AC-4.10 | Prompt/output/cache/reasoning usage is consistent | Partial | Protocol routes expose usage surfaces; cross-route reconciliation absent | Codex 2 + Codex 4 |
| AC-4.11 | Quota discovery sources and accuracy limits documented | Not-supported | No per-provider source map supplied | OmniRoute architect: publish mapping |

## 5. 429, overload, retry, circuits, and fallback

| ID | Checklist item | Status | Evidence / redacted note | Owner and close action |
|---|---|---|---|---|
| AC-5.1 | 429 classified by account/model/provider/global/local overload | Supported | Classifier and provider rules identified by code review | Codex 2/4: failure-injection certification |
| AC-5.2 | Retry metadata honored with bounded jitter | Partial | Cooldown/backoff code identified; header-by-header proof absent | Codex 2 + Codex 4 |
| AC-5.3 | Scoped configurable breaker opens/half-opens/recovers | Partial | Breaker/half-open implementation identified; live proof absent | Codex 2 + Codex 4 |
| AC-5.4 | One throttled account falls through safely | Partial | Account fallback identified; injected proof absent | Codex 2 + Codex 4 |
| AC-5.5 | All-throttled returns deterministic 429/earliest retry | Not-supported | No captured all-exhausted behavior | Codex 2 + Codex 4 |
| AC-5.6 | Network/timeout/5xx/malformed/local-overload policies differ | Partial | Error-classifier components exist; full matrix untested | Codex 2 + Codex 4 |
| AC-5.7 | Attempts/deadline bounded; cancellation stops retry/upstream | Partial | Bounded policy intended; configuration and cancellation proof absent | Codex 2 + Codex 4 |
| AC-5.8 | Replay only pre-commit; partial output is not replayed | Partial | Normative policy frozen; before/after-output injection absent | Codex 2 + Codex 4 |
| AC-5.9 | Idempotency prevents duplicate inference/tool actions | Not-supported | No deduplication strategy/proof supplied | OmniRoute architect + Codex 1/2 |
| AC-5.10 | Same-model first; explicit reported cross-model fallback | Partial | Rule documented and Kimi chain reported; capability equivalence/versioned approval absent | Codex 1 + Codex 2/4 |
| AC-5.11 | Fallback never silently reduces capabilities/safety | Not-supported | No equivalence guard or negative fixture supplied | Codex 1/2 + product owner |

## 6. Streaming, long requests, and cancellation

| ID | Checklist item | Status | Evidence / redacted note | Owner and close action |
|---|---|---|---|---|
| AC-6.1 | SSE heartbeat/proxy timeouts support provider latency | Partial | SSE endpoints exist; long-reasoning/heartbeat evidence absent | Codex 2 + Codex 4 |
| AC-6.2 | Backpressure and response buffers are bounded | Not-supported | Effective buffer/max response configuration not supplied | OmniRoute architect + Codex 4 |
| AC-6.3 | Disconnect/cancel aborts upstream and releases capacity once | Partial | Cancellation is required by route contract; live counter proof absent | Codex 2 + Codex 4 |
| AC-6.4 | TTFB/idle/total deadlines configurable per route/model | Not-supported | Only queue timing is identified; deadline matrix absent | OmniRoute architect + Codex 2 |
| AC-6.5 | Streams drain or interruption is documented on config/restart | Not-supported | No drain/restart behavior supplied | OmniRoute operator + Codex 4 |
| AC-6.6 | Partial output/usage/account/terminal status reconciled | Partial | Required telemetry semantics exist in design; broken-stream proof absent | Codex 2 + Codex 4 |

## 7. Security and secret handling

| ID | Checklist item | Status | Evidence / redacted note | Owner and close action |
|---|---|---|---|---|
| AC-7.1 | Stable credential scoped to routes and independently rotated | Partial | Authentication required; scope/revoke/rotation proof absent | OmniRoute operator + Codex 4 |
| AC-7.2 | Credentials encrypted/redacted and never returned | Partial | AES-256-GCM component identified; full nested redaction/API proof absent | OmniRoute architect + Codex 4 |
| AC-7.3 | Raw prompts/completions/tools/reasoning not logged by default | Not-supported | No accepted content-logging policy or fixture supplied | Security + OmniRoute architect |
| AC-7.4 | Missing/invalid auth fails closed on inference/management | Partial | Missing credential 401 and authorized 200 observed; every inference and privileged endpoint not tested | Codex 2 + Codex 4 |
| AC-7.5 | Management authorization separated from inference | Not-supported | No management/inference authorization map supplied | OmniRoute architect + Security |
| AC-7.6 | TLS boundary documented; plain HTTP local only | Not-supported | Current deployment includes a non-loopback exposure; accepted TLS/local-boundary decision absent | Security + OmniRoute operator |
| AC-7.7 | Linux-restricted secret injection, no broad Windows inheritance | Not-supported | Plan is defined below; implementation/runtime evidence absent | Codex 4 plan; Codex 1/3 implement and verify |
| AC-7.8 | Audit covers credential/account/config/auth failures without values | Not-supported | No redacted audit sample supplied | OmniRoute architect + Codex 4 |

## 8. Health, observability, and diagnostics

| ID | Checklist item | Status | Evidence / redacted note | Owner and close action |
|---|---|---|---|---|
| AC-8.1 | Separate liveness/readiness; readiness reflects inference ability | Not-supported | Readiness fail-closed is an explicit outstanding gate | Codex 2 + Codex 4 |
| AC-8.2 | Protected route health gives safe account-state counts | Not-supported | No accepted route-health response supplied | OmniRoute architect + Codex 2/4 |
| AC-8.3 | Request plus task/session correlation across all evidence | Not-supported | Common correlation is designed but not wired | Codex 1/2 + Codex 4 |
| AC-8.4 | Required latency/error/retry/circuit/usage/utilization metrics | Partial | Components/desired fields identified; catalog/sample missing | OmniRoute architect + Codex 4 |
| AC-8.5 | Required dashboards and alerts | Not-supported | No dashboards/alerts evidence supplied | Codex 4 G2D |
| AC-8.6 | Safe machine-readable diagnostic error codes | Partial | Error classifiers identified; northbound code catalog unproven | Codex 2 + Codex 4 |
| AC-8.7 | Logs/metrics bounded for 100 streams | Not-supported | Retention/bounds and tier-100 test absent | OmniRoute operator + Codex 4 |

## 9. Capacity and performance

| ID | Checklist item | Status | Evidence / redacted note | Owner and close action |
|---|---|---|---|---|
| AC-9.1 | No architectural one-request-at-a-time bottleneck | Supported | Selection is single-process atomic without global request serialization | Codex 4: retain in load profile |
| AC-9.2 | Effective global/route/model/provider/account limits declared | Partial | Layer types identified; effective redacted values absent | OmniRoute operator + Codex 4 |
| AC-9.3 | 20/50/100 tiers configurable | Partial | Layered controls exist; only tier 20 authorized and none load-accepted | Codex 1/4: enforce highest proven tier |
| AC-9.4 | Reproducible load profile defined | Not-supported | Mix/rate/duration/payload/account profile absent | Codex 4 G2D harness |
| AC-9.5 | Complete 20/50/100 reports supplied | Not-supported | No load report supplied | Codex 4 Wave 3/4 |
| AC-9.6 | Fairness measured/explained | Not-supported | Four sequential Agy successes are not concurrency fairness evidence | Codex 2 + Codex 4 |
| AC-9.7 | Overload bounded with deterministic retryable error | Partial | Queue limit/timeout identified; saturation behavior unproven | Codex 2 + Codex 4 |
| AC-9.8 | Sustained failure/recovery returns to steady state | Not-supported | No sustained recovery run supplied | Codex 2 + Codex 4 |

## 10. Required failure-injection demonstrations

No failure-injection session was supplied. Every row is **Not-supported as accepted**, even where supporting code was identified.

| ID | Demonstration | Status | Owner and close action |
|---|---|---|---|
| AC-10.1 | Disable account during load | Not-supported | Codex 2 + Codex 4: active-load test |
| AC-10.2 | Expire access token; exactly one refresh | Not-supported | Codex 2 + Codex 4: concurrent refresh test |
| AC-10.3 | Revoke refresh token; isolate account | Not-supported | Codex 2 + Codex 4 |
| AC-10.4 | Exhaust quota; rotate and re-enter after reset/probe | Not-supported | Codex 2 + Codex 4 |
| AC-10.5 | Account-scoped 429 and half-open recovery | Not-supported | Codex 2 + Codex 4 |
| AC-10.6 | Provider-global 429 without account thrash | Not-supported | Codex 2 + Codex 4 |
| AC-10.7 | Inject 401/403/timeout/reset/malformed/5xx | Not-supported | Codex 2 + Codex 4 |
| AC-10.8 | Break SSE before and after output | Not-supported | Codex 2 + Codex 4 |
| AC-10.9 | Cancel queued request and active stream | Not-supported | Codex 2 + Codex 4 |
| AC-10.10 | Add/remove account with 20+ requests | Not-supported | Codex 2 + Codex 4 |
| AC-10.11 | Stateful Responses continuation and prompt cache | Not-supported | Codex 2 + Codex 4 |
| AC-10.12 | Restart/roll under load | Not-supported | OmniRoute operator + Codex 2/4 |

## 11. Model-route matrix linkage

The per-model matrix is recorded separately at `.planning/agent-brain-v3/evidence/g1-model-route-matrix.md`. It shows that no route yet has the complete exact model/pool/tools/reasoning/structured-output/context/rotation/fallback proof required for production admission.

## 12. Go/no-go evidence package

| ID | Package item | Status | Evidence / owner action |
|---|---|---|---|
| AC-12.1 | Completed checklist and model matrix with owners/decisions | Supported | This classified checklist plus `g1-model-route-matrix.md`; acceptance gaps remain blocking |
| AC-12.2 | Versioned compatibility statement and exact digest | Partial | Version known; immutable digest absent — OmniRoute operator |
| AC-12.3 | Redacted route/account/concurrency/circuit/timeout config | Partial | High-level redacted notes above; exact effective revision/values absent — OmniRoute operator + Codex 4 |
| AC-12.4 | Automated exact-model protocol conformance | Not-supported | Codex 2 + Codex 4 Wave 3 |
| AC-12.5 | Failure-injection results | Not-supported | Codex 2 + Codex 4 Wave 3 |
| AC-12.6 | Reproducible 20/50/100 report | Not-supported | Codex 4; only tier 20 currently authorized for canary |
| AC-12.7 | Security evidence: storage/scope/redaction/isolation/audit/rotation | Partial | Encryption component and auth check identified; remaining controls unproven — Security + Codex 4 |
| AC-12.8 | Full operational runbooks | Not-supported | Secret/topology plan below is only one subset — OmniRoute operator + Codex 4 |
| AC-12.9 | Named operations/integration owners and escalation | Partial | Role owners assigned in GSD; named human operators/escalation path absent — product owner |

## Linux restricted secret-file reference plan

This is a reference design only. It does not create, inspect, copy, or print a secret.

| Control | Planned contract |
|---|---|
| Reference | Neutral daemon configuration stores only the path `/etc/agent-brain/secrets/omniroute-inference-key`; final configuration field name is frozen by Codex 1 |
| Directory | `/etc/agent-brain/secrets`, owned by `root:agent-brain`, mode `0750` |
| File | Regular file only, owned by `root:agent-brain`, mode `0440`; daemon service user is the only intended group reader |
| Read safety | Open without following symlinks; reject non-regular files, unexpected owner/group, group-write, any world access, empty/oversized content, or read failure; never include content in an error |
| Injection | Parent daemon reads once into memory and applies the scoped credential only to authorized inference children after inherited/custom environment sanitization; never mounts/copies it into task homes |
| Provisioning | An authorized operator stages the value outside the repository/image and atomically installs it in the restricted directory; Windows source location/value is never referenced in committed configuration or logs |
| Rotation | Stage a restricted sibling file, validate metadata, atomically rename, then controlled reload/restart; retain only an operator-controlled previous version for bounded rollback |
| Logging/evidence | Emit path class, metadata validation result, rotation generation/time, and success/failure code only; redact authorization headers and never hash/fingerprint the credential into general telemetry |
| Revocation/failure | Authentication/read failure makes OmniRoute readiness false and blocks new model-dependent admission; no provider-native or Prodex fallback |
| Backup | Do not include the plaintext file in ordinary repository/config backups; use the approved secret-management backup/escrow process with audited restore |

Preferred future hardening is a service-manager credential facility or secret manager that materializes the same restricted reference at runtime. That change must preserve the no-task-home and no-broad-inheritance rules.

## Endpoint topology plan

| Runtime that launches the CLI | OmniRoute location | Required base | Adapter path rule | Decision |
|---|---|---|---|---|
| Current Agent Brain on host/WSL | Container published on host port 20128 | `http://127.0.0.1:20128` | Claude uses root; Codex/OpenAI-compatible profiles use root plus `/v1`; direct Agy uses the documented exact path only after proof | Approved local topology for Waves 0–3/tier 20 |
| Future Agent Brain container on same Docker network | OmniRoute container | `http://omniroute:20128` | Same protocol path rules; never use container loopback for another container | Planned; requires compose/network health proof |
| Agent Brain container with OmniRoute reachable only through host | Host gateway explicitly configured for that deployment | Explicit configured host-gateway URL | No implicit Docker DNS/loopback assumption | Exception topology; requires reachability and trust-boundary acceptance |
| Any nonlocal/private-network hop | Remote gateway | Deployment-specific HTTPS URL | TLS and authenticated inference required; management authorization remains separate | Not accepted by current local-only evidence |

The current non-loopback/LAN exposure is not selected for Agent Brain inference. Plain HTTP is accepted only on the explicit host-loopback or private same-network container boundary. Firewall presence is defense in depth, not a substitute for endpoint authentication or TLS outside that boundary.

## G1 disposition

This checklist is complete as a 116-item documentary classification. It does not certify production cutover. Required blockers remain: immutable digest, exact per-model registry and protocol conformance, fail-closed readiness, secret/runtime isolation, single-flight refresh, continuation affinity, failure injection, tier-20 load evidence, Smart Context or waiver, state-topology decision, security evidence, and operational runbooks.
