# Smart Context Shadow / Canary Plan

Status: PLAN + ACCEPTANCE CRITERIA. LIVE EXECUTION F0-GATED.

Source references:

- `docs/prodex/prodex-runtime-invariants.md`: Smart Context stays inside Rust L2;
  Go may only set desired mode and kill switches; native rollout uses
  `PRODEX_SMART_CONTEXT_SHADOW=1` and
  `PRODEX_SMART_CONTEXT_CANARY_PERCENT=N`; exact fallback must preserve
  protocol, continuation, tool-call, and global JSON structure.
- `docs/prodex/prodex-fork-map.md`: Smart Context source areas are
  `crates/prodex-app/src/runtime_proxy/smart_context/*`,
  `prodex-runtime-proxy/src/smart_context/*`, and `prodex-context`; preserve
  replay corpus, telemetry, shadow/canary rollout, and whole-request fallback.
- `docs/prodex/prodex-l2-event-emission.md`: Smart Context runtime events are
  `rewrite_decision`, `spend_savings`, and `guardrail`; exact Multica schema
  emission remains fork/adapter `a validar`.

Live execution note: every step that sends real traffic, changes live canary
percentage, or enables live rewrite is **F0-GATED** and requires owner approval.
This document currently delivers the dry-run plan, test criteria, and evidence
requirements only.

## 1. Default

Initial PROD mode:

```text
PRODEX_SMART_CONTEXT_SHADOW=1
PRODEX_SMART_CONTEXT_CANARY_PERCENT=0
```

No live rewrite without owner approval and evidence.

Default acceptance:

- all Smart Context computation is observe-only;
- upstream receives the original request body;
- Go does not rewrite payloads;
- kill switch can force exact/pass-through before the next request;
- events/evidence are scrubbed and contain no raw prompt, raw tool output,
  bearer token, OAuth material, API key, cookie, or full provider payload.

## 1A. Dry-Run And Static Readiness

Status: executable before F0; no live provider traffic required.

Dry-run checks:

| Check | Method | Pass criteria |
|---|---|---|
| Native knobs present | Inspect launch/policy plan for `PRODEX_SMART_CONTEXT_SHADOW=1` and `PRODEX_SMART_CONTEXT_CANARY_PERCENT=0`. | Shadow is enabled and canary is zero by default. |
| Authority split | Compare launch/policy plan with `docs/prodex/prodex-runtime-invariants.md`. | Go only pushes desired mode/kill switch; Rust/prodex owns rewrite/fallback. |
| Event contract | Validate planned events against `docs/prodex/prodex-l2-event-emission.md` and `docs/contracts/runtime-event-validation-spec.md`. | `rewrite_decision`, `spend_savings`, and `guardrail` fields are schema-valid and `redaction.secrets_present=false`. |
| Replay coverage | Run prodex replay report only in non-live/test environment when artifacts are available. | Replay pass result recorded; live promotion remains F0-GATED regardless. |
| Kill switch dry-run | Use smoke/dry-run harness or static policy evidence. | Smart Context disable request resolves to `effective_at=next_request` or stricter. |

Dry-run evidence must record:

- command or inspection summary;
- target config values;
- event samples or event-field checklist;
- no live traffic assertion;
- owner/reviewer;
- UTC timestamp.

## 2. Shadow Mode

Status: F0-GATED for live traffic. Use dry-run/static evidence before F0.

Purpose: measure before/after and validation decisions while sending original
request upstream.

Required metrics:

- estimated tokens before;
- estimated tokens after;
- rewrite ratio;
- selected segment categories;
- fallback reasons;
- validation result;
- additional turns after shadow if measurable.

Gate to canary:

- no protocol integrity warning;
- no tool-call integrity warning;
- no continuation integrity warning;
- no JSON integrity warning;
- no missing mandatory artifact warning that would affect protocol/global
  structure;
- no secret in logs;
- event stream healthy and schema-valid for `rewrite_decision`,
  `spend_savings`, and `guardrail` when adapter events are enabled;
- kill switch validated in dry-run or controlled test;
- owner approval.

Shadow acceptance criteria:

| Assertion | Required result |
|---|---|
| Upstream body | Exact original request body is sent upstream. |
| `rewrite_decision.mode` | `shadow`. |
| `rewrite_decision.decision` | `shadow_only` or `pass_through`; never `rewrite` for upstream delivery. |
| `rewrite_decision.fallback_exact` | `true` when validation detects protocol, continuation, tool, JSON, mandatory reference, or critical-signal risk. |
| `spend_savings` | Contains only scrubbed byte/token estimates and pressure fields; no source text. |
| `guardrail` | Emits `smart_context_integrity` with `fallback_exact` or `allowed` where applicable. |
| Runtime authority | No Go-side payload rewrite or reroute occurs. |

## 3. Canary Mode

Status: F0-GATED for live traffic.

Initial canary:

```text
PRODEX_SMART_CONTEXT_CANARY_PERCENT=1
```

Canary sequence:

1. Keep `PRODEX_SMART_CONTEXT_SHADOW=1` disabled only if prodex policy requires
   mutually exclusive live/canary mode; otherwise record exact effective mode.
2. Set `PRODEX_SMART_CONTEXT_CANARY_PERCENT=1`.
3. Confirm canary-out requests pass through unchanged.
4. Confirm canary-in requests rewrite only after validation/self-checks pass.
5. Exercise exact fallback cases before any percentage increase.
6. Increase only by owner-approved steps: `1 -> 5 -> 10 -> 25 -> 50 -> 100`.
7. Stop and roll back to shadow on any disable condition in section 5.

Promotion requires:

- exact fallback works;
- no continuation failure;
- no tool-call corruption;
- no JSON corruption;
- no missing mandatory artifact;
- p95 rewrite overhead acceptable to owner;
- rollback and kill switch tested.

Canary acceptance criteria:

| Assertion | Required result |
|---|---|
| Canary-out | Original request body passes through unchanged and logs `canary_out` or equivalent rollout reason. |
| Canary-in | Rewritten request is sent only after validation passes. |
| Exact fallback | Protocol-sensitive, malformed artifact, invalid JSON, continuation, or tool-call risk sends original/exact-safe body. |
| Event stream | `rewrite_decision.rollout_mode` is `canary_in` or `canary_out`; canary percent is recorded when available. |
| Quality guard | No previous-response-not-found, invalid tool continuation, corrupted JSON, or repeated missing-context recovery linked to rewrite. |
| Latency guard | p95 rewrite overhead is within owner-approved budget for the current step. |
| Rollback | Setting canary to `0` and/or kill switch produces next-request pass-through. |

## 4. Live Mode

Status: F0-GATED. Do not enable live rewrite from this plan.

Live mode is allowed only after:

- shadow evidence;
- canary evidence;
- QA sign-off;
- owner approval;
- kill switch test.

Live entry criteria:

| Gate | Required evidence |
|---|---|
| Shadow complete | Shadow acceptance criteria passed for the approved sample/window. |
| Canary complete | Canary reached owner-approved percentage with no stop condition. |
| Exact fallback | All exact fallback probes passed during shadow and canary. |
| Event ingest | Go accepts schema-valid Smart Context events and rejects malformed/secret-bearing events before ledger/observability writes. |
| Kill switch | Smart Context disable takes effect before the next request. |
| Rollback | Raw/exact pass-through rollback path is documented and dry-run tested. |
| Owner approval | Explicit F0 approval names window, tenant/profile scope, and max canary/live percentage. |

Live acceptance criteria:

- `rewrite_decision.mode=live` only after all gates pass;
- every live rewrite has `validation_result=passed`;
- any failed validation produces `exact_fallback` or `pass_through`;
- no semantic/protocol degradation is accepted as a recovery path;
- no raw prompt/tool/provider payload content enters events, logs, or evidence;
- live mode can be disabled before the next request.

## 4A. Exact Fallback Probes

Run these probes in dry-run/replay first; live probes are F0-GATED.

| Probe | Input shape | Expected decision |
|---|---|---|
| Protocol-sensitive payload | Payload contains control-plane/protocol fields that must stay exact. | `pass_through` or `exact_fallback`; no rewrite sent. |
| Continuation binding | Request carries `previous_response_id`, turn state, or session binding. | Continuation fields preserved byte-for-byte; bound profile unchanged. |
| Tool-call integrity | Tool/function call ids and ordering are present. | Tool ids/order preserved; fallback exact on mismatch. |
| Malformed artifact reference | Missing/corrupted required artifact hash or mandatory ref. | Dependent segment preserved or whole request exact fallback if global/protocol risk. |
| Invalid JSON candidate | Rewrite candidate cannot parse or changes required structure. | Original body sent; `validation_result=failed_json_integrity`. |
| Critical signal recall risk | Mandatory/critical segment would be dropped or condensed unsafely. | Exact fallback or segment rollback. |

Exact fallback pass criteria:

- upstream receives original or exact-safe equivalent;
- no corrupted JSON;
- continuation ids and tool-call ids are preserved;
- `rewrite_decision.fallback_exact=true` when fallback is used;
- `guardrail.guardrail_type=smart_context_integrity` is emitted when adapter
  events are enabled;
- no Go runtime rewrite or reroute is triggered.

## 5. Immediate Disable Conditions

Disable Smart Context if:

- previous response not found after rewrite;
- invalid tool-call continuation;
- corrupted JSON;
- repeated missing context recovery;
- secret appears;
- fallback exact fails;
- user-visible task regression linked to rewrite.

Disable action:

1. Apply Smart Context kill switch or restore shadow-only/canary-zero config.
2. Require next request to be exact/pass-through.
3. Record `guardrail.guardrail_type=kill_switch` or
   `smart_context_integrity` when adapter events are enabled.
4. Preserve scrubbed evidence and stop promotion.
5. Do not re-enable without owner approval and root-cause note.

## 6. Evidence Package

Each gate must write scrubbed evidence under `.deploy-control/evidence/` or an
owner-approved equivalent:

- mode and effective knobs;
- sample count/window, or `dry-run` when no live traffic;
- `rewrite_decision` summary counts by decision and validation result;
- `spend_savings` aggregate fields only, no source contents;
- guardrail/kill-switch events;
- exact fallback probe results;
- latency summary for canary/live;
- disable/rollback proof;
- owner approval reference for every F0-GATED step.

Evidence is invalid if it contains secrets, raw prompts, raw tool outputs, full
provider bodies, cookies, bearer tokens, or OAuth material.
