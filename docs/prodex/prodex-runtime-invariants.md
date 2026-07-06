# prodex Runtime Invariants

Status: TARGET-MILESTONE INVARIANTS. Does not block F0 prodex-as-is rollout.

Source scope: pinned official prodex `0.246.0`
`7750da9b6a5c91a6d429e18e6a4d422cab4bc144`, plus local
`openspec/changes/rotation-parity-polyglot/design.md` and ADR-001.

Labels:

- `verified`: directly present in pinned prodex docs/repo or local ADR/OpenSpec.
- `inferred`: required fork behavior derived from verified sources.
- `not-validated`: implementation or outcome requires live/load validation.

## Authority Split

1. `verified`: Go is the cold control plane in local ADR/OpenSpec. It owns
   tenants, approved accounts, policy, budgets, kill switches, orchestration,
   Postgres, and aggregated observability.
2. `verified`: Rust/prodex is the hot runtime plane in local ADR/OpenSpec. It
   owns runtime proxy/gateway, session/profile affinity, pre-commit routing,
   fallback, Smart Context, and guarded redeem.
3. `inferred`: after `StartSession`, Go must not call legacy Go runtime routing
   for that session. Rust emits events for observability/ledger only; events do
   not re-decide a request already in flight.

## Hard Affinity

`verified`: prodex official architecture and state docs require hard affinity:

- `previous_response_id -> profile`;
- `x-codex-turn-state -> profile`;
- `session_id -> profile` for session-scoped unary routes.

`verified`: runtime-state code stores response, session, turn-state, and
session-id continuation bindings, with lifecycle status for warm, verified,
suspect, and dead bindings.

Rules:

1. Continuation affinity beats fresh selection heuristics, transport backoff,
   profile health, and in-flight caps.
2. A fresh request may choose among eligible profiles; a continuation request
   must use the bound owner unless prodex marks that binding unavailable through
   its explicit stale/dead continuation policy.
3. Smart Context must preserve continuation fields exactly.
4. Multica event consumers may group by session/request/profile but must not
   infer a different owner after Rust has selected one.

## Rotate Before Commit

`verified`: prodex official architecture says rotation is allowed only before:

- first accepted unary response;
- first committed stream response;
- returning quota/overload to Codex.

It also says prodex must not rotate mid-stream after model output starts.

Rules:

1. Profile fallback is pre-commit only.
2. After model output or a committed upstream response exists, preserve upstream
   status/body/stream payloads except for failures where no upstream response
   existed.
3. Pre-commit retry/fallback budgets remain bounded per request.
4. Redeem may retry only through guarded pre-commit paths; it must not redeem to
   alter a committed request.

## Hot Path I/O And Persistence

`verified`: official architecture/runtime-policy docs require request and stream
commit paths to avoid broad disk reads, blocking state saves, unbounded thread
spawn, and terminal output while Codex TUI runs.

`verified`: runtime-state scheduling classifies save reasons, debounces hot
continuation saves, and runtime-store merges selected state sections and
continuations instead of replacing stale snapshots wholesale.

Rules:

1. Request/stream commit paths cannot block on shared file/SQLite/Postgres/Redis
   writes.
2. Runtime state persistence is asynchronous or scheduled, and merge-safe for
   active profile, profile metadata, response bindings, session bindings,
   usage snapshots, and backoffs.
3. Runtime logs are diagnostics only. They must never become required source of
   truth for request success or routing correctness.
4. Prodex-owned screens may print outside runtime launch; runtime notices during
   Codex TUI execution go to logs only.

## Smart Context

`verified`: official Smart Context docs and code require:

- exact preservation of control-plane, continuation, protocol, ordering,
  function/tool-call IDs, and explicit exact-mode fields;
- independent segment classification;
- validation of JSON structure, continuation/tool integrity, mandatory
  references, critical-signal recall, nonempty mandatory payloads, duplicated
  appendices, and segment allocations;
- whole-request fallback for explicit exact mode or failures that can affect
  protocol, continuation, or global structural correctness;
- native rollout through `PRODEX_SMART_CONTEXT_SHADOW=1` and
  `PRODEX_SMART_CONTEXT_CANARY_PERCENT=N`;
- original-body pass-through in shadow mode, canary-out, invalid/unsupported
  input, panic fallback, or self-check fallback.

Rules:

1. Smart Context stays inside Rust L2.
2. Go may set desired mode and kill switch; Go must not rewrite runtime payloads.
3. Shadow mode must compute telemetry while sending the original request.
4. Canary-out must pass through unchanged.
5. Live mode may send a rewritten request only after validation and self-checks.
6. Exact fallback means no semantic/protocol degradation is allowed as a recovery
   path; if validation cannot prove safety, send the original or minified
   equivalent that preserves protocol structure.

`not-validated`: prodex replay benchmark reports deterministic success on its
checked-in corpus, but this audit did not validate live task quality in Multica
production traffic.

## Gateway And Provider Routing

`verified`: official runtime-policy/deployment docs state `prodex gateway` is an
OpenAI-compatible HTTP gateway with provider presets, virtual keys, admin RBAC,
SCIM/SSO/OIDC options, guardrails, route aliases, observability, and
file/SQLite/Postgres/Redis state.

`verified`: adaptive routing defaults to observational shadow behavior, and
code preserves continuation affinity before adaptive recommendations.

Rules:

1. Gateway adaptive routing remains shadow unless explicit policy enables live
   behavior.
2. Continuation affinity and quota/safety constraints beat adaptive routing.
3. Provider endpoint capability must be explicit: native, translated,
   passthrough, unsupported, partial, or untested.
4. Provider transforms must report lossless, degraded-safe, rejected, or
   unsupported behavior instead of silently dropping parameters.

## State Backends

`verified`: official deployment docs say file and SQLite gateway state are
single-node deployment models. Official runtime-policy docs support
`gateway.state.backend` values `file`, `sqlite`, `postgres`, and `redis`.

Rules:

1. File/SQLite are acceptable only for local or explicitly single-node
   deployment.
2. Forked L2 shared gateway/admin/usage/ledger state must use Postgres and/or
   Redis for multi-worker or multi-host deployment.
3. Profile auth isolation under `$PRODEX_HOME/profiles/<name>` must remain
   stronger than convenience.
4. Shared Codex state remains upstream-compatible and must not become profile
   authority.

## Redeem

`verified`: manual redeem is OpenAI/Codex-only and sends one explicit reset-credit
consume request. Near natural reset, it prompts unless `--yes`.

`verified`: runtime `--auto-redeem` is guarded by weekly exhaustion, natural
reset grace, credit availability, OpenAI profile eligibility, better/remaining
pool alternatives, quota refresh, and post-redeem quota re-check.

Rules:

1. Redeem is low-priority, cold-path, and request-precommit only.
2. Do not redeem when another eligible profile has weekly quota remaining.
3. Do not redeem near a natural reset window.
4. Redeem attempts must emit audit/runtime events without secret material.
5. Use idempotency keys for consume requests.

`not-validated`: this audit did not confirm backend efficacy or outcome mapping
with real accounts. F9 must validate no-credit, with-credit, near-reset,
weekly-exhausted, 5h-only, all-exhausted, and non-OpenAI/provider-disabled
scenarios.

## Kill Switches And Secrets

`verified`: local ADR/OpenSpec require kill switches and scrubbed logs for
prodex-as-is rollout. Official prodex docs/code include redaction helpers,
runtime logs, gateway auth, virtual keys, admin tokens, and audit events.

Rules:

1. Tenant/provider/profile kill switch must override feature enablement.
2. Smart Context, gateway, and auto-redeem kill switches must take effect before
   the next request.
3. Secrets must not appear in logs, traces, evidence, check-ins, or event
   payloads.
4. Runtime events must carry opaque identifiers only.

## Fork Regression Gates

Every fork change touching runtime proxy, Smart Context, provider transforms,
state, or redeem must prove:

- hard affinity remains intact;
- rotate-before-commit remains intact;
- hot commit path does not add blocking persistence or broad disk reads;
- Smart Context exact fallback remains intact;
- provider behavior is declared and fixture-backed;
- file/SQLite is not used for shared multi-node state;
- runtime events are redacted and schema-valid;
- rollback to raw Codex/prodex-as-is remains documented.
