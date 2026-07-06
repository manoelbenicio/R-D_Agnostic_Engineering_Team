# Reset-Claim Planning Matrix

Status: DONE-planning / empirical-gated. Empirical execution is later/gated on
real weekly-exhausted account state. Do not run `prodex redeem` from this
document.

Source scope: official prodex redeem implementation/docs at pinned prodex
`0.246.0` commit `7750da9b6a5c91a6d429e18e6a4d422cab4bc144`.

## Official Redeem Facts

- Manual command: `prodex redeem <profile>` sends one explicit reset-credit
  consume request for a named OpenAI/Codex profile. Source:
  `crates/prodex-cli/src/help.rs`.
- Manual near-reset guard: if the 5h or weekly quota window resets within one
  hour, prodex asks before consuming a credit; `--yes` skips that prompt. Source:
  `crates/prodex-cli/src/help.rs`, `crates/prodex-app/src/app_commands/redeem.rs`.
- Manual non-OpenAI guard: non-OpenAI/Codex profiles cannot redeem reset
  credits. Source: `crates/prodex-app/src/app_commands/redeem.rs`.
- Runtime auto-redeem is opt-in through `--auto-redeem` and is described as
  applying when all configured OpenAI/Codex profiles are weekly-exhausted.
  Source: `crates/prodex-cli/src/runtime_args.rs`.
- Auto-redeem only warrants credit when the weekly window is `Exhausted`, has a
  finite reset timestamp, and the natural weekly reset is more than 5 minutes
  away. Source: `crates/prodex-app/src/runtime_proxy/quota/auto_redeem.rs`.
- Auto-redeem requires a positive `rate_limit_reset_credits.available_count`;
  otherwise it returns `NothingToRedeem`. Source:
  `crates/prodex-app/src/runtime_proxy/quota/auto_redeem.rs`,
  `crates/prodex-quota/src/models.rs`.
- Consume calls carry a `redeem_request_id`; auto-redeem generates an
  idempotency-shaped key per process/time/sequence. Source:
  `crates/prodex-app/src/quota_support/auth.rs`,
  `crates/prodex-app/src/runtime_proxy/quota/auto_redeem.rs`.
- Backend outcomes include reset/default reset, nothing-to-reset, no-credit, and
  already-redeemed behavior; auto-redeem maps `NothingToReset` and `NoCredit` to
  `NothingToRedeem`. Source: `crates/prodex-app/src/quota_support/auth.rs`,
  `crates/prodex-app/src/runtime_proxy/quota/auto_redeem.rs`.

## Case Matrix

| Case | Preconditions to stage | Expected prodex behavior | F9 planning verdict | Evidence |
|---|---|---|---|---|
| no-credit | OpenAI/Codex profile; weekly exhausted; `rate_limit_reset_credits.available_count <= 0` or backend returns `NoCredit` | Auto-redeem logs unavailable/no-credit path and returns `NothingToRedeem`; manual backend decides no-credit. | Do not expect recovery. Record no-credit outcome and continue normal fallback/blocked handling. | `auto_redeem.rs`, `auth.rs`, `models.rs` |
| has-credit | OpenAI/Codex profile; weekly exhausted; reset more than 5 minutes away; `available_count > 0` | Auto-redeem sends consume request with idempotency key; if backend returns `Reset` or `AlreadyRedeemed`, prodex refetches usage and only returns `Redeemed` when 5h and weekly are no longer exhausted. | Eligible for gated empirical run. Success requires post-redeem quota refetch to show retry allowed. | `auto_redeem.rs`, `auth.rs` |
| near-reset | 5h or weekly reset is soon | Manual: prompt within 1 hour unless `--yes`; non-terminal without `--yes` fails. Auto: does not warrant credit when weekly reset is within 5 minutes. | Do not redeem automatically near natural reset. Manual test may use `--yes` only with explicit owner approval. | `help.rs`, `redeem.rs`, `auto_redeem.rs` |
| weekly-exhausted | OpenAI/Codex profile; weekly status `Exhausted`; finite weekly reset timestamp | This is the native auto-redeem trigger surface, subject to natural-reset grace, credit availability, provider eligibility, and pool guards. | Primary positive test case, gated. Include before/after quota snapshots. | `runtime_args.rs`, `auto_redeem.rs` |
| 5h-only | 5h exhausted but weekly is not exhausted | Auto-redeem does not warrant credit because weekly is not exhausted. Manual redeem may still send one explicit request, but backend decides whether anything can reset. | Do not run auto-redeem expecting recovery. Manual empirical run is low priority and owner-gated. | `auto_redeem.rs`, `help.rs` |
| all-exhausted | 5h and weekly both exhausted | Auto-redeem may attempt only if weekly criteria and credit availability pass; after consume, prodex refetches and returns `Failed` if either 5h or weekly remains exhausted. | Gated test should assert no false success when post-redeem state is still blocked. | `auto_redeem.rs` |
| non-OpenAI | Provider is not OpenAI/Codex | Manual redeem bails before consume. Auto-redeem profile lookup and pool scanning filter to OpenAI profiles. | Must not call consume. Record rejected/unsupported. | `redeem.rs`, `auto_redeem.rs` |

## Required Guards

| Guard | Native prodex support seen in official source | F9 planning rule |
|---|---|---|
| Idempotency | Consume request includes `redeem_request_id`; auto-redeem creates `prodex-auto-redeem-<pid>-<nanos>-<sequence>`. | Every empirical attempt must persist the request id in the evidence record before execution and reuse it for retry correlation. |
| Cooldown | Native auto-redeem has a 5-minute natural-reset grace. A general per-profile redeem-attempt cooldown was not confirmed in the redeem path. | Add external F9 runner cooldown: one empirical consume attempt per `(profile, quota-window, reset_at)` unless owner explicitly resets the gate. |
| Audit event | Redeem-specific audit event was not confirmed in the official redeem path. Runtime logs do record auto-redeem start/result/failure markers. | Multica/F9 evidence must write a scrubbed audit event before and after any gated empirical attempt: `planned`, `skipped`, `attempted`, `result`, or `failed`. |
| No redeem in thin/critical if another eligible profile | Prodex treats weekly `Ready`, `Thin`, and `Critical` as weekly remaining. Pool deferral can skip redeem when another eligible weekly-remaining profile exists, but only on code paths using `prefer_best_pool_profile=true`. | F9 runner must enforce this guard globally before invoking any manual or auto empirical attempt: if another OpenAI/Codex profile is `Ready`, `Thin`, or `Critical` for weekly quota and route-eligible, skip redeem. |
| Provider eligibility | Manual and auto paths are OpenAI/Codex-only. | Non-OpenAI rows are validation-only; never call consume. |
| Near natural reset | Manual prompts within 1 hour; auto uses 5-minute natural reset grace. | External gate should use the stricter planning rule unless owner approves: skip any profile with 5h or weekly reset within 1 hour. |
| Post-redeem verification | Auto-redeem refetches usage and returns success only if 5h and weekly are not exhausted. | Empirical run must capture scrubbed before/after quota summaries and treat unchanged exhausted state as failed/not-effective. |

## Empirical Run Contract

No empirical run is authorized by this document. When F9 is later approved:

1. Resolve profile and provider from prodex state.
2. Fetch quota snapshot only.
3. Apply provider, near-reset, pool, cooldown, and credit-count guards.
4. Persist scrubbed pre-attempt audit/evidence with the idempotency key.
5. Invoke exactly one approved redeem attempt.
6. Fetch quota snapshot again.
7. Persist scrubbed post-attempt audit/evidence.
8. Classify the row as `validated-success`, `validated-noop`,
   `validated-failed`, or `blocked-by-guard`.

## Gated Empirical Test Procedure

This procedure is ready for the future weekly-exhausted window, but remains
blocked until the owner explicitly opens the empirical gate for one named
profile. The procedure must be copied into the run evidence and filled with
scrubbed values only.

### Gate Preconditions

All preconditions must be true before any redeem attempt is allowed:

| Gate | Required state | Failure action |
|---|---|---|
| Owner approval | Written approval names the tenant, profile, provider, allowed UTC window, and whether manual or auto path is being tested. | `blocked-by-guard`; do not call redeem. |
| Provider | Profile is OpenAI/Codex-backed. | `blocked-by-guard`; non-OpenAI rows are validation-only. |
| Weekly quota | Weekly quota classification is `Exhausted` with a finite reset timestamp. | `blocked-by-guard`; do not spend a reset credit. |
| Credit availability | `rate_limit_reset_credits.available_count > 0` or equivalent scrubbed quota summary indicates a credit is available. | `validated-noop` or `blocked-by-guard`, depending on backend response path; do not retry. |
| Natural reset window | Neither 5h nor weekly reset is within 1 hour unless owner explicitly approves the near-reset override. | `blocked-by-guard`; wait for natural reset. |
| Pool alternative | No other route-eligible OpenAI/Codex profile has weekly quota `Ready`, `Thin`, or `Critical`. | `blocked-by-guard`; use the eligible profile instead. |
| Cooldown | No prior empirical consume attempt exists for `(tenant_id, profile_id, weekly_reset_at)`. | `blocked-by-guard`; do not retry. |
| Kill switch | Reset-claim/redeem kill switch is not active for tenant, provider, profile, or session. | `blocked-by-guard`; fail closed. |
| Evidence sink | Scrubbed pre/post audit record location is available before execution. | `blocked-by-guard`; do not run without evidence. |

### Required Evidence Record

Create the evidence record before execution and update it after execution. It
must contain no secrets, cookies, bearer tokens, OAuth material, raw prompts,
raw tool outputs, or full provider payloads.

Required scrubbed fields:

| Field | Required content |
|---|---|
| `run_id` | Operator-generated id, length 8-128. |
| `tenant_id` | Opaque tenant id. |
| `profile_id` | Opaque prodex profile id or name approved for the test. |
| `provider` | Provider id, expected OpenAI/Codex. |
| `quota_window` | `weekly`. |
| `weekly_reset_at` | Scrubbed UTC timestamp from the pre-attempt quota snapshot. |
| `five_hour_reset_at` | Scrubbed UTC timestamp when available. |
| `pre_weekly_classification` | Expected `weekly_exhausted`. |
| `pre_five_hour_classification` | `available`, `thin`, `critical`, `exhausted`, or `unknown`. |
| `available_reset_credits` | Integer count or `unknown` if not exposed. |
| `redeem_request_id` | Idempotency key persisted before the consume attempt. |
| `cooldown_key` | `${tenant_id}:${profile_id}:${weekly_reset_at}`. |
| `guard_decision` | `planned`, `skipped`, `attempted`, `result`, or `failed`. |
| `operator` | Human or agent identity approving/recording the run. |
| `started_at` / `finished_at` | UTC ISO8601 timestamps. |
| `post_weekly_classification` | Post-attempt weekly classification or `not_observed`. |
| `post_five_hour_classification` | Post-attempt 5h classification or `not_observed`. |
| `classification` | `validated-success`, `validated-noop`, `validated-failed`, or `blocked-by-guard`. |
| `safe_detail` | Scrubbed reason, max 512 chars. |

### Idempotency Guard

Use exactly one `redeem_request_id` per approved empirical attempt and persist it
before invoking any consume path. If the attempt is interrupted after the
request id is persisted, the same evidence record must be updated; do not start
a second run for the same cooldown key.

Rules:

1. The idempotency key must be unique for the approved run.
2. The evidence record must include the key before execution.
3. A backend `AlreadyRedeemed` result is not automatically success. Success
   still requires post-redeem quota refetch showing the request can proceed.
4. Any unknown post-state is `validated-failed` unless owner explicitly marks
   the run inconclusive and keeps the cooldown in force.

### Cooldown Guard

Cooldown scope is `(tenant_id, profile_id, weekly_reset_at)`.

Rules:

1. Allow at most one empirical consume attempt for the cooldown key.
2. Keep the cooldown even when the result is `validated-noop`,
   `validated-failed`, interrupted, or inconclusive.
3. Clear the cooldown only when the weekly reset timestamp changes naturally or
   the owner explicitly opens a new gate in writing.
4. Never bypass cooldown with `--yes`; `--yes` only affects prodex's native
   manual prompt behavior and does not override F9 gates.

### Audit Guard

Write scrubbed audit/evidence states in this order:

1. `planned`: profile, provider, quota window, cooldown key, owner approval,
   and evidence location recorded.
2. `skipped`: if any preflight guard fails, record the guard and stop.
3. `attempted`: record the persisted `redeem_request_id` immediately before
   the single approved attempt.
4. `result`: record backend-safe outcome and post-attempt quota summary.
5. `failed`: record local execution, validation, timeout, or evidence failure.

If the evidence sink cannot write `planned`, `skipped`, or `attempted`, the run
must stop before redeem.

### Execution Steps For Future Approved Window

1. Confirm owner approval names the exact profile, path under test, UTC window,
   and near-reset override status.
2. Capture a scrubbed pre-attempt quota snapshot for the named profile.
3. Evaluate every gate in the Gate Preconditions table.
4. If any gate fails, write `skipped` and classify `blocked-by-guard`.
5. Generate or capture the single `redeem_request_id` and write `planned`.
6. Write `attempted` with the idempotency key and cooldown key.
7. Invoke exactly one approved redeem attempt for the named profile.
8. Capture a scrubbed post-attempt quota snapshot.
9. Write `result` or `failed`.
10. Classify the run:
    - `validated-success`: post-refetch shows weekly and 5h quota no longer
      block the retried request.
    - `validated-noop`: no credit, nothing-to-reset, already-redeemed with no
      usable post-state change, or backend says no action was taken.
    - `validated-failed`: consume path errors, post-state remains exhausted, or
      post-state cannot be verified.
    - `blocked-by-guard`: any preflight guard prevents execution.

### Stop Conditions

Stop immediately and do not retry when any of these occurs:

- evidence write fails before or after the attempt;
- a second route-eligible OpenAI/Codex profile has weekly quota remaining;
- reset is inside the 1-hour near-natural-reset window without owner override;
- provider is not OpenAI/Codex;
- cooldown key already exists;
- reset-claim kill switch is active;
- post-attempt quota cannot be fetched.

## Not Validated Here

- Real backend behavior for no-credit, has-credit, near-reset, weekly-exhausted,
  5h-only, all-exhausted, and non-OpenAI accounts.
- Whether backend `AlreadyRedeemed` always corresponds to a usable post-redeem
  quota state.
- Whether prodex runtime logs are sufficient as product audit evidence.
- Whether all auto-redeem call sites consistently apply the pool deferral guard.
