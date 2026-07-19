# EV-G1-MODELMATRIX — G1 per-model route matrix

- Status: G1 gateway-prep evidence complete; route acceptance remains gated
- Prepared by: Codex 2 — OmniRoute Gateway
- Authorization: `OMNIROUTE_ARCHITECT_RESPONSE.md` §7.1, Waves 0–3, tier 20
- Scope: planning evidence only; no gateway implementation, daemon wiring, canary, or cutover

## Sources and exactness rule

This matrix reconciles:

- `openspec/changes/build-omniroute-agent-brain/OMNIROUTE_ARCHITECT_RESPONSE.md`, especially §§1, 2.1–2.3, 5–7;
- `openspec/changes/build-omniroute-agent-brain/omniroute-architecture-acceptance-checklist.md`, especially §§2, 3, 5, 6, 11, and 12; and
- the normative adapter and fail-closed requirements in the change design/specs.

`CONFIRMED` means the architect response explicitly states the value. `EXPECTED` means the acceptance checklist names it but the architect has not supplied proof. `PENDING` means a protocol surface exists but the required per-model conformance evidence is absent. `UNDECLARED` means no exact value was supplied; it must not be guessed or silently defaulted.

The reviewed baseline is OmniRoute `3.8.48` using an unpinned `:latest` image tag. The digest is still required. The architecture decision is conditional GO for build and a tier-20 canary only; it is not route or production-cutover acceptance.

## A. Route identity, API format, and account pool

The host/WSL runtime root is `http://127.0.0.1:20128`. A future container runtime must select an explicitly reachable container endpoint; it must not reuse host loopback blindly.

| Family | Exact `RouteModel` or route alias | Client adapter | Exact northbound API format | Account pool | Evidence status |
|---|---|---|---|---|---|
| Claude Code | `claude_code_kimi_2.7_Code` | Claude Code | Anthropic Messages: `POST /v1/messages`; Anthropic JSON; SSE when streaming. Claude receives the OmniRoute root without `/v1` so it does not construct a duplicated path. | Pool membership/count `UNDECLARED`. The architect identifies this as the currently configured Claude Code combo, but does not identify its resolved upstream model or accounts. | Endpoint family `CONFIRMED`; exact model map and per-model conformance `PENDING`. |
| Codex/OpenAI | `UNDECLARED` — no exact approved model ID supplied | Codex custom provider | OpenAI Responses: `POST /v1/responses`; Responses JSON; HTTP/SSE; `wire_api="responses"`. WebSocket remains disabled unless separately proven. | `UNDECLARED` — no connection IDs, pool size, or eligibility rules supplied. | Endpoint family `CONFIRMED`; route ID, pool, and per-model conformance `PENDING`. |
| Kimi | `kimi-sub` is a combo/route alias; exact model ID `UNDECLARED` | Kimi-compatible OpenAI adapter; native Kimi provider-registry contract is not yet proven | OpenAI Chat Completions: `POST /v1/chat/completions`; OpenAI-compatible JSON/SSE. ACP is not the upstream model HTTP protocol. | `kimi-sub` pool size `UNDECLARED`. A downstream combo, `cline-kimi-k2.7-dedicated`, is declared with 4 subscriptions. | Chat endpoint family `CONFIRMED`; native CLI contract and per-model conformance `PENDING`. |
| GLM | `cp/cline-pass/glm-5.2` | Codex/OpenAI-compatible adapter | OpenAI Chat Completions: `POST /v1/chat/completions`; OpenAI-compatible JSON/SSE. The checklist's earlier “Responses or Chat” ambiguity is resolved to Chat by the architect response for Kimi/GLM/NVIDIA. | 4 clinepass accounts `EXPECTED` by the checklist, not confirmed for this exact route by the architect response. | Exact route is checklist-listed; pool and per-model conformance `PENDING`. |
| NVIDIA | `nvidia/z-ai/glm-5.2` | Codex-compatible Chat adapter or converted generic NIM adapter | OpenAI Chat Completions: `POST /v1/chat/completions`; OpenAI-compatible JSON/SSE. | 3 NVIDIA connections `EXPECTED`. The architect confirms 3 subscriptions for `group2-nvidia-glm5.2-fallback`, but does not prove that this alias is identical to the exact model row. | Exact route is checklist-listed; alias mapping and per-model conformance `PENDING`. |
| NVIDIA | `nvidia/*` other approved models | Codex-compatible Chat adapter or converted generic NIM adapter | OpenAI Chat Completions: `POST /v1/chat/completions`; OpenAI-compatible JSON/SSE. | NVIDIA pool membership/count `UNDECLARED` per exact model. | No additional exact model IDs were supplied; no route may be admitted from this wildcard. |
| Antigravity | `agy/claude-opus-4-6-thinking` | Direct Antigravity endpoint; Claude/Codex fallback only after compatible-protocol proof | Direct: `POST /v1/antigravity`. The direct endpoint's precise request/response schema and header contract remain undocumented in the response. Candidate fallback: Anthropic Messages or Responses, but neither fallback is proven. | 4 `agy` accounts `CONFIRMED`; 4 sequential HTTP 200 results were reported for this exact model. | Direct connectivity/account cycling `CONFIRMED`; protocol capability and fallback conformance `PENDING`. |
| Antigravity | `agy/claude-sonnet-4-6` | Claude Code fallback, or direct only if its contract is proven | Transport selection `UNRESOLVED`: candidate `POST /v1/messages`; candidate direct `POST /v1/antigravity`. | `agy` pool named by checklist; exact membership/count for this model `UNDECLARED`. | Checklist candidate only; no per-model proof supplied. |
| Antigravity | `agy/gemini-3.1-pro-high` | Approved compatible frontend, not yet selected | Transport selection `UNRESOLVED`: candidate direct `POST /v1/antigravity`; compatible Claude/Codex contract not supplied. | `agy` pool named by checklist; exact membership/count for this model `UNDECLARED`. | Checklist candidate only; no per-model proof supplied. |

Authentication for every route is the single scoped OmniRoute inference credential. Provider-native credentials or direct-provider fallback are forbidden. This evidence records no credential value, prefix, file content, authorization header, or provider account identity.

## B. Per-model capability acceptance

Family-level route code or a successful HTTP status is not capability acceptance. The architect explicitly lists missing per-model fixtures for SSE, tools, thinking/reasoning, and function-call deltas. The response supplies no numeric context limit for any row.

| Exact route/model | Streaming | Tools | Reasoning | Structured output | Context limit | Current disposition |
|---|---|---|---|---|---:|---|
| `claude_code_kimi_2.7_Code` | `PENDING`: Anthropic SSE conformance | `PENDING`: Anthropic tool block/schema/correlation | `PENDING`: thinking and redacted-thinking preservation | `UNDECLARED`: no per-model structured-output contract | `UNDECLARED` | Not admissible until the combo resolves to a versioned exact model row and passes conformance. |
| Codex/OpenAI exact model `UNDECLARED` | `PENDING`: Responses SSE event families/order | `PENDING`: function calls, IDs, argument deltas/results | `PENDING`: effort/summary and opaque continuation fields | `PENDING`: Responses structured-output contract | `UNDECLARED` | Not admissible; exact model ID and pool are missing. |
| `kimi-sub` / exact Kimi model `UNDECLARED` | `PENDING`: Chat SSE and usage | `PENDING`: tool-call deltas/indexes/results | `PENDING`: exposed reasoning and `reasoning_effort` behavior | `PENDING`: `response_format` behavior | `UNDECLARED` | Not admissible; combo alias is not an exact versioned model row. |
| `cp/cline-pass/glm-5.2` | `PENDING`: Chat SSE and usage | `PENDING`: tool-call deltas/indexes/results | `PENDING`: reasoning fields | `PENDING`: `response_format` behavior | `UNDECLARED` | Not admissible pending exact-route conformance and pool proof. |
| `nvidia/z-ai/glm-5.2` | `PENDING`: Chat SSE and usage | `PENDING`: tool-call deltas/indexes/results | `PENDING`: reasoning fields | `PENDING`: `response_format` behavior | `UNDECLARED` | Not admissible pending exact-route conformance and alias/pool proof. |
| `agy/claude-opus-4-6-thinking` | `PENDING`: 200 does not prove streaming | `PENDING`: no tool proof | `PENDING`: model name does not prove reasoning fidelity | `UNDECLARED` | `UNDECLARED` | Connectivity-only canary candidate; not capability-accepted. |
| `agy/claude-sonnet-4-6` | `PENDING` | `PENDING` | `PENDING` | `UNDECLARED` | `UNDECLARED` | Not admissible; adapter/API selection and all capabilities are unproven. |
| `agy/gemini-3.1-pro-high` | `PENDING` | `PENDING` | `PENDING` | `UNDECLARED` | `UNDECLARED` | Not admissible; frontend/API selection and all capabilities are unproven. |

Unsupported request fields must be rejected deterministically at admission. They must never be silently dropped or mapped to a different model. Context overflow is a client/capability error, not account exhaustion.

## C. Rotation, continuation affinity, and fallback

The default policy contract is:

1. For a new independent logical request, select the next eligible account atomically. Do not advance on SSE chunks, transport retries, tool blocks, or tokens.
2. For dependent continuation (`previous_response_id`, provider turn state, prompt-cache ownership, or active tool continuation), preserve the owning account or use a proven stateless materialization. The architect's Responses `preserve` pin is still awaiting live proof.
3. Retry/fallback only before user-visible output or a potentially non-idempotent tool action, bounded by attempts and end-to-end deadline. Never replay after partial output/tool action.
4. Try a healthy account for the same exact model before any approved cross-model/provider fallback.
5. Cross-model fallback requires a signed ordered policy and proof of equivalent context, tools, structured output, reasoning, and safety. If absent, fail closed. Never fall back to a provider-native credential or direct-provider endpoint.
6. Strict round-robin is confirmed only for the deployed single OmniRoute process. Horizontal strict rotation is not supported by the current in-memory cursor/single-node SQLite topology.

| Exact route/model | Independent-request rotation | Continuation affinity | Ordered fallback chain |
|---|---|---|---|
| `claude_code_kimi_2.7_Code` | Exact combo policy `UNDECLARED`; general strict-RR behavior must not be assumed for this combo. | Required for cache/tool/provider state; per-route mechanism `PENDING`. | No approved chain supplied → same-model eligible account only, then fail closed. |
| Codex/OpenAI exact model `UNDECLARED` | Intended strict RR with `stickyRoundRobinLimit=1` on single node; exact pool/config and concurrent proof `PENDING`. | Responses modes `auto`, `strip`, and `preserve`; originating-account pin for `preserve` `PENDING`. | No approved chain supplied → same-model eligible account only, then fail closed. |
| `kimi-sub` / exact Kimi model `UNDECLARED` | Architect-declared exception: Kimi rotates only on critical failure; it is not strict per-request RR. | Session-sticky; `failoverBeforeRetry`. Exact continuation semantics still require conformance. | Declared current combo order: `kimi-sub` → `cline-kimi-k2.7-dedicated` (4 subscriptions, RR/failover) → `group2-nvidia-glm5.2-fallback` (3 subscriptions). This chain is not production-approved until exact model mapping and capability equivalence are proven. |
| `cp/cline-pass/glm-5.2` | Exact policy `UNDECLARED`; the checklist expects a 4-account pool but does not prove strict RR for this row. | Per-route continuation/cache behavior `UNDECLARED`. | No separately approved chain supplied → same-model eligible account only, then fail closed. Do not infer the Kimi combo chain applies to this exact ID. |
| `nvidia/z-ai/glm-5.2` | Exact policy `UNDECLARED`; do not infer policy from the 3-subscription fallback combo without alias proof. | Per-route continuation/cache behavior `UNDECLARED`. | No separately approved chain supplied → same-model eligible account only, then fail closed. |
| `agy/claude-opus-4-6-thinking` | Strict RR over 4 `agy` accounts reported and 4×200 observed; concurrency atomicity and eligibility-skipping proof `PENDING`. | Stateful continuation policy `UNDECLARED`. | Direct endpoint first. Claude/Codex protocol fallback is required as an accepted alternative but remains unproven; until then, no fallback and fail closed. |
| `agy/claude-sonnet-4-6` | Exact model/pool policy `UNDECLARED`. | `UNDECLARED`. | No accepted direct/fallback transport → fail closed. |
| `agy/gemini-3.1-pro-high` | Exact model/pool policy `UNDECLARED`. | `UNDECLARED`. | No accepted direct/fallback transport → fail closed. |

## D. Acceptance gaps, owner, and decision

| Gate | Affected rows | Owner(s) | Required decision/evidence |
|---|---|---|---|
| Pin exact image digest and publish versioned model/capability registry | All | OmniRoute architect/operator; Codex 2 validates registry; Codex 4 records evidence | Supply exact digest and one machine-readable row per approved model with protocol, pool, context, stream, tools, reasoning, structure, and availability. |
| Supply missing exact model IDs and account-pool mappings | Claude combo, Codex/OpenAI, Kimi, GLM, NVIDIA, Sonnet/Gemini Agy | OmniRoute architect/operator | Resolve every alias to exact model ID and pseudonymous pool configuration; wildcard or provider-family admission is forbidden. |
| Protocol/model conformance | All | Codex 2 + Codex 4 (OpenSpec 8.1); Codex 3 + Codex 4 for CLI paths (8.2) | Synthetic non-streaming/streaming, tools, reasoning, structure, usage, cancellation, and deterministic error evidence for every exact row. |
| Responses continuation affinity | Codex/OpenAI and any Responses fallback | OmniRoute architect/operator; Codex 2 + Codex 4 | Live proof that `preserve` pins the origin account, plus strict-RR proof for unrelated requests. |
| Kimi native adapter contract | Kimi | Codex 3; OmniRoute architect | Prove installed provider-registry upstream override, or approve Claude/Codex-compatible frontend; ACP cannot satisfy this gate. |
| Antigravity adapter/API selection | All `agy/...` | Codex 3; OmniRoute architect; Codex 1 approves supported set | Prove native endpoint override and exact direct schema/auth, or prove the exact model through Claude/Codex protocol fallback. |
| Ordered cross-model fallback policy | Every row | Codex 1 / product owner, using Codex 2/4 conformance evidence | Approve only capability-equivalent chains. Until approval, same-model account fallback then fail closed. |
| Rotation/failure/capacity proof | All approved rows | Codex 2 + Codex 4 | Concurrent RR/affinity, eligibility changes, pre-commit retry, partial-stream no-replay, 429/circuit, cancellation, and tier-20 report. |

## G1 conclusion

The available documents define four northbound protocol profiles—Anthropic Messages, OpenAI Responses, OpenAI Chat Completions, and direct Antigravity—but do not yet define a complete accepted per-model registry. The only exact route with reported account cycling is `agy/claude-opus-4-6-thinking`, and its 4×200 result proves connectivity/selection only. No row currently satisfies the checklist's full combination of exact model, exact pool, streaming, tools, reasoning, structured output, numeric context limit, rotation/affinity, fallback, and reproducible proof.

Therefore EV-G1-MODELMATRIX is complete as a freeze/input artifact, with all unknowns and owners explicit. G2B may implement registry/profile types against these contracts, but admission, Wave-3 canaries, and production approval must remain fail-closed until the row-level gates above are satisfied.
