# OmniRoute Architecture Acceptance Checklist

## Purpose and response format

This checklist is the prerequisite contract for connecting the Agent Brain to OmniRoute. The OmniRoute architect must answer every item with **Supported**, **Partially supported**, or **Not supported**, and attach the exact version/configuration plus one reproducible proof: API documentation, configuration excerpt with secrets redacted, automated test result, or captured request/response with credentials removed.

An HTTP 200 from `/v1/models` proves connectivity only. It does not prove coding-agent protocol fidelity, safe rotation, failure recovery, or concurrent capacity.

## 1. Ownership boundary

- [ ] OmniRoute is the sole owner of provider OAuth tokens, API keys, subscriptions, refresh tokens, account health, quota state, rotation, retry, and provider fallback.
- [ ] Agent Brain and every launched CLI hold exactly one stable, scoped OmniRoute key; they do not receive Anthropic, OpenAI, Google, Kimi, NVIDIA, or other provider-native secrets.
- [ ] OmniRoute can add, remove, disable, quarantine, or re-enable accounts without restarting Agent Brain or recreating its running tasks.
- [ ] OmniRoute can update route/model/account-pool configuration atomically, without exposing a partially written configuration to new requests.
- [ ] The architect identifies the authoritative OmniRoute version/image digest, configuration source, database/state location, backup method, and supported upgrade/rollback procedure.

## 2. Required northbound protocols and message fidelity

### 2.1 Common endpoints

- [ ] `GET /v1/models` accepts the stable Bearer key and returns every model ID that Agent Brain is allowed to select.
- [ ] Model-list entries expose stable IDs and enough metadata to validate protocol family, context window, capabilities, and availability, or OmniRoute supplies an equivalent machine-readable registry.
- [ ] Unknown model IDs return a deterministic 4xx response and never silently select a different model.
- [ ] JSON request bodies, UTF-8, gzip where used, HTTP keep-alive, request cancellation, and large coding-context payloads are supported without message truncation or field loss.

### 2.2 Anthropic Messages API for Claude Code

- [ ] `ANTHROPIC_BASE_URL=<OmniRoute root>` resolves Claude Code calls to `POST /v1/messages` without appending a duplicated `/v1` segment.
- [ ] OmniRoute accepts `Authorization: Bearer <OmniRoute key>` as emitted when Claude Code uses `ANTHROPIC_AUTH_TOKEN`; document any additional supported Anthropic authentication header.
- [ ] Requests preserve `model`, `system`, `messages`, `max_tokens`, `temperature`, `top_p`, `stop_sequences`, metadata, and supported beta/version headers.
- [ ] Anthropic content blocks are preserved: `text`, `image` where supported, `tool_use`, `tool_result`, `thinking`, `redacted_thinking`, and cache-control blocks.
- [ ] Tool definitions, JSON Schema, `tool_choice`, parallel tool calls, tool IDs, and tool-result correlation round-trip without translation loss.
- [ ] Non-streaming responses follow Anthropic Messages response shape, including stop reason and input/output/cache/reasoning usage when upstream supplies it.
- [ ] Streaming uses valid Anthropic SSE event ordering, including message/content-block start, delta, stop, ping, error, tool-input JSON deltas, thinking deltas, and final usage.
- [ ] Anthropic-format errors remain machine-readable and preserve retryability information, HTTP status, provider code, and `Retry-After` when present.

### 2.3 OpenAI Responses API for Codex

- [ ] `POST /v1/responses` is fully supported with `stream=true` and `stream=false`; Chat Completions compatibility alone is not sufficient for Codex.
- [ ] Requests preserve `model`, `instructions`, structured `input`, roles/content parts, tools, tool choice, parallel tool-call control, output limits, temperature where valid, metadata, and reasoning parameters.
- [ ] Function-call names, call IDs, argument JSON, function-call outputs, and multiple calls in one response remain lossless across provider translation.
- [ ] Reasoning fields such as effort/summary and any opaque or encrypted reasoning continuation fields required by the selected Codex model are preserved or explicitly declared unsupported per model.
- [ ] Responses API SSE emits the event families and ordering Codex consumes: response lifecycle, output-item lifecycle, content-part lifecycle, text deltas, function-argument deltas, completion, failure, and final usage.
- [ ] `previous_response_id` and other provider-stateful continuation features have a documented behavior. OmniRoute either pins continuations to the originating account, materializes a stateless continuation, or rejects the feature explicitly; it must not rotate blindly and lose state.
- [ ] OmniRoute accepts `X-Session-Id`, `X-Request-Id`, and trace headers from a Codex custom provider and returns correlation identifiers without using them to override strict rotation unless the configured route says so.
- [ ] WebSocket transport is either proven for all selected OmniRoute model routes or disabled in the Codex provider configuration; the HTTP/SSE path is mandatory.

### 2.4 OpenAI Chat Completions compatibility

- [ ] `POST /v1/chat/completions` supports streaming and non-streaming for OpenAI-style Kimi, GLM, and NVIDIA routes.
- [ ] Requests preserve system/developer/user/assistant/tool roles, multipart content, tool definitions, `tool_choice`, parallel calls, `response_format`/structured output where supported, `reasoning_effort`, token limits, stop conditions, and stream usage options.
- [ ] Streaming preserves text, reasoning when exposed, tool-call argument deltas and indexes, finish reasons, error events, and final usage.
- [ ] Capability differences are declared per model rather than silently dropping unsupported request fields.

### 2.5 Kimi, NVIDIA/GLM, and Antigravity specifics

- [ ] For Kimi, the architect identifies the supported upstream contract used by the Kimi CLI/provider registry. ACP is a local agent-control protocol and must not be confused with the model HTTP protocol between the CLI and OmniRoute.
- [ ] For NVIDIA/GLM, OmniRoute confirms OpenAI-compatible request/response fidelity for every approved `nvidia/...` or `cp/cline-pass/...` model and returns stable usage/error metadata.
- [ ] For Antigravity, OmniRoute documents the exact direct endpoint and authentication contract (currently expected as `/v1/antigravity` where applicable) and proves all approved `agy/...` models.
- [ ] Because the installed native `agy` CLI may not support endpoint override, OmniRoute confirms that the same `agy/...` models work through the supported Claude Code or Codex protocol fallback.
- [ ] The approved model map is versioned and includes at least Claude Opus/Sonnet, Codex/OpenAI, Kimi, GLM, NVIDIA, and Antigravity routes with protocol family and reasoning/tool/streaming capabilities.

## 3. Rotation and account-selection semantics

- [ ] Define the exact rotation unit: a logical API request, not an SSE chunk, HTTP retry, tool block, or token. Identify how multi-turn continuations are treated.
- [ ] A strict round-robin route selects the next eligible account for each new independent request with no hidden preference or randomization.
- [ ] Session stickiness can be disabled for strict per-request round-robin. Confirm `disableSessionStickiness=true` or the equivalent and explain any route where affinity is deliberately retained.
- [ ] `stickyRoundRobinLimit=1` or its equivalent is proven under concurrency; simultaneous arrivals must not race and select the same account contrary to the configured cycle.
- [ ] Rotation is concurrency-safe and does not globally serialize requests. Twenty, fifty, or one hundred requests may be in flight while account selection continues atomically.
- [ ] Per-account and per-model concurrency limits are configurable independently from rotation order and account count.
- [ ] An account at its safe concurrent limit is skipped temporarily without corrupting the round-robin cursor or marking a healthy account permanently failed.
- [ ] Adding/removing/quarantining an account updates the eligible pool safely; in-flight requests complete or drain according to a documented policy.
- [ ] Account affinity required by `previous_response_id`, prompt-cache ownership, or provider conversation state is explicit and takes precedence only for that continuation, not unrelated requests.
- [ ] Rotation telemetry identifies route and pseudonymous account/connection ID without exposing provider credentials or personal account data.

## 4. Token, expiry, quota, and subscription lifecycle

- [ ] OAuth access tokens are refreshed proactively before expiry with clock-skew allowance.
- [ ] Concurrent refresh attempts for one account are single-flight/locked so one refresh cannot invalidate another.
- [ ] If a token expires between selection and dispatch, OmniRoute refreshes it and safely retries before any response bytes are delivered.
- [ ] A failed refresh quarantines only the affected account, records a reason/cooldown, and selects another eligible account when policy permits.
- [ ] A provider `401` is classified as expired/revoked/invalid; a refresh or re-auth path is attempted once according to policy, without an infinite retry loop.
- [ ] A provider `403` is classified separately for entitlement, disabled subscription, model access, or policy denial; permanent failures remove that account/model pair from selection until repaired.
- [ ] Account usage quota, subscription allowance, reset time, and model-specific entitlement are tracked when the provider exposes them.
- [ ] When an account reaches a rate/token/subscription quota, OmniRoute selects another eligible account immediately and cools down the exhausted account until the authoritative reset or a bounded probe succeeds.
- [ ] Context-window overflow or a request exceeding a model's maximum tokens is not misclassified as account exhaustion. It returns a deterministic client error unless an explicitly approved compatible-model fallback exists.
- [ ] Prompt, completion, cache, and reasoning token usage is returned consistently and can be aggregated per route/model/account without secret exposure.
- [ ] The architect documents how quotas are discovered: response headers, provider API, OAuth metadata, local counters, or conservative inference, including known accuracy limits.

## 5. 429, overload, retry, circuit breaker, and fallback

- [ ] `429` responses are classified at least as account-scoped, model-scoped, provider/global, or OmniRoute-local overload; the system does not blindly rotate all accounts for a global outage.
- [ ] `Retry-After`, rate-limit reset headers, and provider error codes are honored with bounded jittered backoff.
- [ ] Repeated 429s open a circuit breaker for the appropriate account/model/route scope; thresholds, observation window, cooldown, half-open probes, and recovery criteria are configurable and observable.
- [ ] A single throttled account can fall through to the next healthy account within the same logical route without returning 429 when safe capacity remains.
- [ ] If every account is throttled, OmniRoute returns a deterministic 429 with the earliest safe retry time and route-level diagnostics, not an unbounded internal retry.
- [ ] Network errors, timeouts, provider 5xx, malformed upstream responses, and OmniRoute-local resource exhaustion have separate retry/fallback policies.
- [ ] Retries are bounded by attempts and end-to-end deadline; client cancellation stops queued retries and aborts the upstream request.
- [ ] Automatic replay is allowed only before user-visible output or a non-idempotent tool action. A broken stream after partial output is surfaced with correlation data and is not silently replayed.
- [ ] An idempotency/deduplication strategy prevents duplicate requests and duplicate tool calls when the client retries after an ambiguous timeout.
- [ ] Same-model fallback across accounts is preferred. Cross-model or cross-provider fallback occurs only through an explicit ordered policy and reports the actual model/provider used.
- [ ] Fallback never silently reduces required context size, tool support, structured output, reasoning capability, or safety policy.

## 6. Streaming, long-running requests, and cancellation

- [ ] SSE connections remain open through provider latency with heartbeat behavior compatible with each client and all intermediate proxy timeouts.
- [ ] Backpressure prevents a slow client from causing unbounded memory use; documented buffer and maximum response limits exist.
- [ ] Client disconnect/cancellation promptly aborts the upstream call, releases account concurrency, and closes accounting exactly once.
- [ ] Time-to-first-byte, idle, and total deadlines are configurable per route/model; long reasoning calls are not killed by a generic short timeout.
- [ ] OmniRoute drains in-flight streams during configuration changes, restart, and deployment or explicitly reports the interruption behavior.
- [ ] Partial output, upstream usage, selected account, and terminal status are reconciled even when a stream fails.

## 7. Security and secret handling

- [ ] The stable OmniRoute key can be scoped to only the required models/routes and can be rotated/revoked without touching provider accounts.
- [ ] Provider credentials and the stable key are encrypted at rest, redacted from logs/errors/metrics/traces, and never returned by configuration APIs.
- [ ] OmniRoute does not log raw prompts, completions, tool arguments, repository content, or reasoning unless an explicit audited policy enables it.
- [ ] Authentication fails closed; missing/invalid keys cannot use `/v1/messages`, `/v1/responses`, `/v1/chat/completions`, or privileged management endpoints.
- [ ] Management APIs are separated from inference authorization and are not exposed through the Agent Brain key.
- [ ] TLS requirements are documented for any traffic leaving loopback/private Docker networking. Plain HTTP is permitted only on the explicitly accepted local trust boundary.
- [ ] Secret injection uses a Linux permission-restricted secret or equivalent secret manager; the Windows source file is not committed, copied into images, printed, or inherited broadly by unrelated processes.
- [ ] Audit records cover key/account/configuration changes and authentication failures without secret values.

## 8. Health, observability, and diagnostics

- [ ] Separate liveness and readiness signals exist. Readiness reflects database/config availability and the ability to accept inference, not merely that the process is running.
- [ ] A protected or redacted route-health view reports eligible, cooling, quarantined, expired, and unavailable account counts per route/model.
- [ ] Every request has one end-to-end request ID plus Agent Brain task/session correlation; IDs appear in response headers, structured logs, metrics, and errors.
- [ ] Metrics include request count, in-flight count, queue time, selection time, time to first token, total latency, status/error class, retry/fallback count, circuit state, token usage, account utilization, and cancellation.
- [ ] Dashboards/alerts cover no eligible accounts, refresh failures, 401/403 spikes, sustained 429s, circuit-open ratio, 5xx, queue growth, memory pressure, and latency/error SLO breach.
- [ ] Diagnostic responses expose safe machine-readable error codes for auth, unknown model, no eligible account, quota exhaustion, timeout, upstream failure, invalid request, and local overload.
- [ ] Logs and metrics are bounded/retained so 100 concurrent streaming tasks cannot exhaust disk or memory.

## 9. Capacity and performance acceptance

- [ ] OmniRoute has no architectural one-request-at-a-time bottleneck. Strict round-robin ordering is selection policy, not a global concurrency limit.
- [ ] The architect declares configured global, route, model, provider, and per-account concurrency limits and the queue/admission behavior at each boundary.
- [ ] Capacity tiers of 20, 50, and 100 simultaneous Agent Brain tasks are supported through configurable limits. A task may issue several sequential or concurrent model requests.
- [ ] A reproducible load profile defines model mix, streaming ratio, prompt/output sizes, tool-call ratio, request rate, duration, account-pool size, and upstream limits.
- [ ] Evidence is supplied for 20-, 50-, and 100-task runs: accepted/completed/failed requests, p50/p95/p99 selection latency, time to first token, end-to-end latency, 429/5xx/retry/fallback rates, peak queue, CPU, memory, sockets, and account distribution.
- [ ] Selection fairness under concurrency is measured; distribution differences must be explainable by account eligibility, continuation affinity, or configured capacity rather than races.
- [ ] Overload is bounded by admission control/queue limits and returns a deterministic retryable error instead of crashing, leaking goroutines/connections, or growing memory without limit.
- [ ] Sustained-load and recovery tests prove that circuits, token refresh, account re-entry, and resource usage return to steady state after throttling or failure.

## 10. Required failure-injection demonstrations

The acceptance session must demonstrate these cases with secrets and prompt content redacted:

- [ ] Disable one account during active load; new independent requests move to healthy accounts and in-flight behavior matches policy.
- [ ] Expire an OAuth access token; exactly one refresh occurs and requests recover, or the account is quarantined and fallback succeeds.
- [ ] Revoke a refresh token; the account is removed from eligibility without poisoning the whole route.
- [ ] Exhaust one account's quota; traffic advances to other accounts and the exhausted account returns only after reset/probe success.
- [ ] Inject repeated account-scoped 429s; the scoped circuit opens, other accounts continue, and half-open recovery works.
- [ ] Inject provider-global 429s; OmniRoute avoids wasteful account thrashing and returns an actionable retry time or approved provider fallback.
- [ ] Inject 401, 403, timeout, connection reset, malformed response, and 500/502/503 responses; each follows its documented classification and bounded policy.
- [ ] Break an SSE stream before first output and after partial output; only the safe pre-output case may be replayed automatically.
- [ ] Cancel a queued request and an active stream; both release all slots and stop upstream work.
- [ ] Add and remove an account while 20+ requests are active; selection remains safe and configuration is atomic.
- [ ] Run stateful Responses API continuation and prompt-cache cases; affinity and strict round-robin boundaries behave as documented.
- [ ] Restart or roll OmniRoute during load; demonstrate readiness, draining/recovery, and what Agent Brain receives.

## 11. Model-route acceptance matrix

The architect must complete one row per approved model, not only per provider.

| Route/model ID | Client adapter | API format | Stream | Tools | Reasoning | Structured output | Context limit | Account pool | Rotation/affinity | Fallback chain | Proof |
|---|---|---|---|---|---|---|---:|---|---|---|---|
| `agy/claude-opus-4-6-thinking` | Claude Code fallback / direct if supported | Anthropic or documented Antigravity | TBD | TBD | TBD | TBD | TBD | 4 agy accounts expected | TBD | TBD | TBD |
| `agy/claude-sonnet-4-6` | Claude Code fallback | Anthropic | TBD | TBD | TBD | TBD | TBD | agy pool | TBD | TBD | TBD |
| `agy/gemini-3.1-pro-high` | Approved compatible frontend | Architect to confirm | TBD | TBD | TBD | TBD | TBD | agy pool | TBD | TBD | TBD |
| Codex/OpenAI model(s) | Codex | Responses API | TBD | TBD | TBD | TBD | TBD | Architect to supply | TBD | TBD | TBD |
| `kimi-sub` / approved Kimi model | Kimi-compatible adapter | Architect to confirm | TBD | TBD | TBD | TBD | TBD | Kimi pool | TBD | TBD | TBD |
| `cp/cline-pass/glm-5.2` | Codex/OpenAI-compatible adapter | Responses or Chat | TBD | TBD | TBD | TBD | TBD | 4 clinepass accounts expected | TBD | TBD | TBD |
| `nvidia/z-ai/glm-5.2` | Codex or NIM adapter | OpenAI compatible | TBD | TBD | TBD | TBD | TBD | 3 NVIDIA connections expected | TBD | TBD | TBD |
| Other approved `nvidia/...` | Codex or NIM adapter | OpenAI compatible | TBD | TBD | TBD | TBD | TBD | NVIDIA pool | TBD | TBD | TBD |

## 12. Go/no-go evidence package

OmniRoute is ready for Agent Brain cutover only when the following are delivered:

- [ ] Completed checklist and model-route matrix, with all partial/not-supported items assigned an owner and decision.
- [ ] Versioned API/protocol compatibility statement and exact deployed image digest.
- [ ] Redacted route/account/concurrency/circuit/timeout configuration.
- [ ] Automated protocol conformance results for Anthropic Messages, OpenAI Responses, and Chat Completions on the exact approved models.
- [ ] Failure-injection results for expiry, revoked auth, quota, 429, 5xx, timeout, broken stream, cancellation, and restart.
- [ ] Reproducible 20/50/100 concurrency report with resource and fairness measurements.
- [ ] Security evidence for secret storage, scoping, redaction, management isolation, audit, and key rotation.
- [ ] Operational runbook for account onboarding/removal, model-map change, incident triage, backup/restore, upgrade, and rollback.
- [ ] Named owners for OmniRoute operations and Agent Brain integration, plus the escalation path for provider-wide failures.

Any unsupported item affecting authentication, message fidelity, tool calls, streaming, safe retry, secret isolation, or stateful continuation is a cutover blocker. Capacity gaps can be accepted only with an explicit lower admission limit and a dated remediation plan; they must never be hidden behind strict round-robin wording.
