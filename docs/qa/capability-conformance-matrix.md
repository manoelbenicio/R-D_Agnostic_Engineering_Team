# Capability Conformance Matrix — PER CAPABILITY verification method + pass criteria

> **Stream:** G7 (conformance PER CAPABILITY) — offloaded from GLM#52#A / F6
> **Owner:** GLM#52#CLINE#A
> **Started:** 2026-07-04T20:27:41Z
> **Status:** PLAN + CRITERIA DELIVERED. **LIVE proof is F0-GATED** (`deploy_owner_approved: false`); only DRY-RUN smoke + unit/contract + static evidence exists today.
> **Scope rule:** This doc is the **per-capability index** of concrete verification methods and pass criteria. It is **disjoint from** (and references) the existing execution-detail plans: `docs/qa/runtime-conformance-plan.md`, `docs/qa/smart-context-shadow-canary-plan.md`, `docs/qa/prod-redeem-validation-checklist.md`. It does **not** restate those plans; it maps each capability to a verification method + pass/fail criteria and points to the executor for detail.
> **Contract baseline:** `rpp.l2.v1` (`docs/contracts/l2-runtime-contract.md`); event schema `docs/contracts/runtime-events.schema.json`; Go/No-Go checklist `docs/contracts/f0-readiness-matrix.md`.
> **Read-only provenance:** no product code or deploy was executed to produce this matrix. Sources read: `docs/vendors/vendor-capability-matrix.md`, `docs/vendors/owner-acceptance-request.md`, `docs/contracts/l2-conformance-notes.md`, `docs/contracts/f0-readiness-matrix.md`, `docs/qa/*.md`, `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`, `scripts/smoke/*` (8 scripts), `multica-auth-work/server/internal/l2runtime/client.go`, `multica-auth-work/server/internal/daemon/{l2_runtime,prodex,daemon,config,types}.go`.

---

## 1. How to read this matrix

For **each** of the 7 ADR-001 capabilities this matrix defines:

1. **Behavioral claim** — what conformance actually means (not the marketing label).
2. **Verification method** — the concrete, repeatable procedure (static check / unit test / contract probe / smoke script / live probe) that proves the claim.
3. **Pass criteria** — the exact boolean assertion(s) that must hold. Fail = non-conformant.
4. **Evidence status now** — which verification layer is GREEN today vs GATED.
5. **Live-proof gate** — what owner approval / precondition is required before the LIVE layer can run.

A capability is **CONFORMANT** only when every verification layer required for its enabled configuration is GREEN. A capability whose vendor value is `not_validated` (and not owner-ACCEPTED) is **DISABLED** by the deploy rule and is conformant-by-disablement only — it must not be enabled until live proof is recorded.

---

## 2. Verification layers

Every capability is verified across up to four layers. A layer is either **GREEN** (evidence recorded), **GATED** (procedure defined + harness ready, blocked on owner approval to execute live), or **N/A** (not applicable to that capability).

| Layer | Name | What it proves | Where evidence lives |
|---|---|---|---|
| **L1** | Static / source-of-truth | The capability value is grounded in official vendor docs OR the prodex/contract spec (not inferred-by-default). | `docs/vendors/vendor-capability-matrix.md` classification (`verified`/`inferred`/`not_validated`) |
| **L2** | Unit / contract | Code enforces the capability's invariants at the boundary (loopback, bearer, fail-closed, router-owner, schema). | `multica-auth-work/server/internal/l2runtime/client.go` + `daemon/*_test.go`; `docs/contracts/l2-conformance-notes.md` acceptance tests |
| **L3** | DRY-RUN smoke | The Go→L2 sidecar call for the capability is wired and the smoke harness asserts the pass criteria against a planned loopback request (no live sidecar). | `scripts/smoke/*.sh` + `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md` |
| **L4** | LIVE proof | A real sidecar (loopback, `127.0.0.1:43117`) returns the conformant response and the pass criteria hold against real provider behavior. **F0-GATED.** | `.deploy-control/evidence/` (to be produced on owner approval) |

**Universal pass criteria (apply to every capability's L2/L3/L4):**
- endpoint is loopback only (`ErrNonLoopbackEndpoint` enforced in `client.go`);
- bearer auth present and validated;
- `contract_version == rpp.l2.v1` on every response;
- no secret in any request/response/event (`secrets_present == false`; `ErrSecretEvent`);
- fail-closed: any rejection aborts the session before runtime traffic (no silent fallback to a non-conformant path).

---

## 3. Per-capability conformance

### 3.1 `launch_mode`

**Enum:** `native_cli | codex_provider_bridge | openai_compatible_api | anthropic_compatible_api | editor_extension`

**Behavioral claim:** Multica launches the vendor via the declared launch mode, with per-profile isolation (`CODEX_HOME`/`XDG`/`HOME`) and prodex pinned by version+commit when `launch_mode` is prodex-bridged. The launch does not leak a shared auth store across profiles.

**Verification method:**
- **L1:** confirm the matrix cell classification is `verified` (official vendor doc) for the enabled vendor.
- **L2:** unit — `TestLoadProdexLaunchConfigRequiresVersionAndCommitPins` asserts prodex is NOT launched unless `MULTICA_PRODEX_VERSION` + `MULTICA_PRODEX_COMMIT` are both set (`daemon/prodex_test.go`); `config.go` `loadProdexLaunchConfig` resolves the binary via `exec.LookPath` and fails closed if not found.
- **L3:** DRY-RUN `session-start-stop-smoke.sh` — asserts a session start is attempted against loopback with `router_owner=rust_l2`.
- **L4 (GATED):** real launch of the pinned prodex binary in an isolated `CODEX_HOME`; assert the child process env has the isolated home and NOT the shared `~/.codex`.

**Pass criteria:**
- prodex-bridged launch: `prodexCfg.Enabled == true` ONLY when version+commit pins are present AND `exec.LookPath` resolves the binary; else launch fails closed (no fallback to unpinned binary).
- isolation: child env `CODEX_HOME`/`HOME` equals the per-account managed root, not the shared user store; `isBlockedEnvKey` blocks `MULTICA_*`/`PRODEX_*` from being inherited by the child.
- non-prodex vendors: launched via their native CLI on PATH with per-profile env isolation.

**Evidence status now:** L1 GREEN for `native_cli` (Codex/Kiro/Antigravity/OpenCode) and `editor_extension` (Cline) per vendor matrix. L2 GREEN (unit tests present). L3 GREEN (dry-run). L4 GATED.

---

### 3.2 `auth_mode`

**Enum:** `oauth_profile | api_key | cloud_iam | cli_native_store | google_signin`

**Behavioral claim:** Profile auth is isolated per account; switching to an invalid/missing profile **fails closed** before any runtime traffic; raw auth material is never sent to the L2 sidecar (accounts are registered by reference, not by secret).

**Verification method:**
- **L1:** matrix cell classification `verified` for the enabled vendor's auth mode.
- **L2:** unit — `RegisterAccounts` in `client.go` rejects non-empty `rejected_profiles` and wrong contract versions; daemon `startL2SessionForTask` fails closed before `StartSession` if readiness/register fails (`daemon/l2_runtime.go:60`, `:94`).
- **L3:** DRY-RUN `profile-fail-closed-smoke.sh` — posts `accounts/register` with an invalid `profile_home` outside the managed root; PASS = HTTP >= 400 OR `rejected_profiles` non-empty.
- **L4 (GATED):** register a real approved profile by reference; switch to an invalid profile mid-session; assert the session aborts (no silent reuse of the prior profile) and a `profile_switch_fail_closed` event is emitted.

**Pass criteria:**
- account registration carries profile **reference only** (no OAuth token / `auth.json` / cookie / API key in the payload);
- missing/invalid profile home → request rejected (HTTP >= 400 or `rejected_profiles` non-empty);
- profile switch with invalid auth → session fails before commit, prior profile NOT silently reused;
- secrets scan on the register payload + emitted event = clean.

**Evidence status now:** L1 GREEN (oauth_profile Codex/Kiro, api_key Cline/OpenCode, google_signin Antigravity). L2 GREEN. L3 GREEN (dry-run). L4 GATED.

---

### 3.3 `quota_mode`

**Enum:** `codex_usage | vendor_balance | rate_limit_headers | custom_probe | credit_system | none`

**Behavioral claim:** The system can determine remaining quota for the active profile WITHOUT embedding provider response bodies, and uses that signal only to trigger **pre-commit** rotation (never mid-stream). For L2-owned sessions (`runtime_router_owner == rust_l2`), Go quota detection does NOT drive rotation — the L2 runtime owns it.

**Verification method:**
- **L1:** matrix cell classification for the vendor's quota mode (note `custom_probe`/`none` for Antigravity is `inferred` → borderline cell #8 in owner-acceptance-request).
- **L2:** unit — `TestL2OwnedTaskSuppressesLegacyGoRotationPaths` asserts that for a `rust_l2`-owned task, `rotateTaskOnExhaustion` with an exhaustion-classified result returns no account and calls no rotation service (`daemon/daemon_test.go`).
- **L3:** DRY-RUN `readyz-smoke.sh` + `state-backend-smoke.sh` — assert readiness + shared-state backend (Postgres, not SQLite) pass; quota probe is part of readiness.
- **L4 (GATED):** drive a real profile to a near-limit state; assert (a) for a Go-owned F0 session, rotation triggers pre-commit and not mid-stream; (b) for an L2-owned session, Go no-ops with `rotation_noop_reason == l2_router_owner` and the L2 runtime makes the selection.

**Pass criteria:**
- remaining-quota signal is derived without logging raw provider response bodies (redaction smoke covers this);
- rotation triggered by quota fires **before first committed response** (pre-commit) only;
- for `rust_l2`-owned sessions: `go_rotation_decision_count(session_id) == 0` and `go_rotation_noop_count(session_id, reason=l2_router_owner) >= 1` (one-router acceptance test #7);
- Antigravity `custom_probe`/`none`: if owner does not ACCEPT cell #8, the adapter defaults to no proactive rotation trigger for Antigravity (conformant-by-disablement).

**Evidence status now:** L1 PARTIAL (Codex/Kiro/Cline/OpenCode `verified`; Antigravity `inferred` → owner borderline #8). L2 GREEN. L3 GREEN (dry-run). L4 GATED.

---

### 3.4 `rotation_mode`

**Enum:** `profile_pool | key_pool | gateway_route | unsupported`

**Behavioral claim:** Exactly **one runtime router per session**. For L2-owned sessions the Rust/prodex runtime routes in-flight requests; Go does NOT select a replacement profile, change `credentialAccountHome`, clear `PriorSessionID`/`PriorWorkDir`, or retry on a different account. For F0/prodex-as-is (no recorded `runtime_router_owner`) legacy Go rotation remains allowed.

**Verification method:**
- **L1:** matrix cell classification for the vendor's rotation mode (Codex/Kiro `profile_pool` `inferred`; Cline `gateway_route` `inferred`; Antigravity `unsupported`).
- **L2:** unit — the seven one-router acceptance tests in `docs/contracts/l2-conformance-notes.md` (StartSession persistence, proactive-ledger no-op, proactive-text no-op, reactive-exhaustion no-op, F0 compatibility, event-ingest non-routing, exactly-one-router assertion) — implemented in `daemon/l2_runtime.go` + `daemon_test.go`.
- **L3:** DRY-RUN `session-start-stop-smoke.sh` — asserts `router_owner=rust_l2` on start; `kill-switch-smoke.sh` covers the disable path.
- **L4 (GATED):** run a real L2-owned session; collect the five counters in acceptance test #7 and assert the exactly-one-router invariants hold against real provider traffic.

**Pass criteria (one-router gate — the bar):**
```
persisted_runtime_router_owner(session_id) == rust_l2
go_rotation_decision_count(session_id) == 0
go_rotation_noop_count(session_id, reason=l2_router_owner) >= 1
l2_selection_count(session_id, runtime_request_id) <= 1
l2_fallback_after_committed_count(session_id, runtime_request_id) == 0
```
- F0 compatibility: with no recorded `runtime_router_owner`, existing legacy Go rotation is unchanged (gate only suppresses Go rotation for L2-owned sessions);
- `unsupported` vendors (Antigravity): no rotation claimed; conformant-by-disablement unless owner ACCEPTs cell #3.

**Evidence status now:** L1 PARTIAL (several `inferred`). L2 GREEN (seven acceptance tests DONE per f0-readiness-matrix + open-items). L3 GREEN (dry-run). L4 GATED.

---

### 3.5 `continuation_mode`

**Enum:** `response_id | session_id | cli_thread | none`

**Behavioral claim:** Continuation affinity beats selection heuristics — a continuation (`previous_response_id` / `session_id` / `cli_thread`) stays bound to the profile that produced the prior turn; load-balance does not move a mid-conversation continuation to a different profile. Go records the affinity; the L2 runtime honors it.

**Verification method:**
- **L1:** matrix cell classification for the vendor's continuation mode.
- **L2:** unit — `StartSession` validates `router_owner == rust_l2` and persists `runtime_router_owner` before execution (`client.go:339`, `l2_runtime.go:47/:94`); continuation affinity is a contract field in the `rpp.l2.v1` session request.
- **L3:** DRY-RUN `session-start-stop-smoke.sh` — asserts a session start/stop cycle on loopback; `event-stream-smoke.sh` validates `selection`/`affinity`/`fallback` events with `contract_version=rpp.l2.v1` + `secrets_present=false`.
- **L4 (GATED):** start a multi-turn continuation on profile A; send a continuation with the prior affinity token; assert it remains on profile A (not moved to profile B by load-balance) across 30+ turns (replay coverage from `runtime-conformance-plan.md` §5).

**Pass criteria:**
- continuation request carries the prior affinity token and the L2 runtime returns a `selection` event bound to the same profile;
- no `fallback_after_committed` event for the same `runtime_request_id` (a committed request is not re-routed);
- `none` vendors: no continuation affinity claimed; conformant-by-disablement.

**Evidence status now:** L1 GREEN for `response_id` (Codex), `session_id` (OpenCode); `inferred` for Kiro/Antigravity (`cli_thread`), Cline (provider-dependent). L2 GREEN. L3 GREEN (dry-run). L4 GATED.

---

### 3.6 `smart_context_mode`

**Enum:** `proxy_rewrite | pre_tool_output_filter | disabled_shadow_only`

**Behavioral claim:** Smart Context (a **prodex-only** wrapper capability — no vendor documents it natively, per owner-acceptance-request §“Structural Insight”) preserves protocol/tool-call/continuation integrity and falls back **exactly** (pass-through) on any integrity risk. It rolls out shadow → canary → live, gated at each step, with a working kill switch. Default live config is shadow-only, canary 0%.

**Verification method:**
- **L1:** there is NO `verified` vendor cell for smart_context — it is `not_validated`/`inferred` for every vendor. Conformance therefore depends on prodex wrapper proof, not vendor docs. Borderline cell #7 (Codex `proxy_rewrite` inferred from prodex) requires owner review.
- **L2:** unit — `KillSwitch` client method (`client.go:247/:269`); daemon injects `PRODEX_KILL_SWITCH_DEFAULT_ON` env (`prodex.go:71`); event stream rejects `secrets_present == true` (`client.go:303`).
- **L3:** DRY-RUN `kill-switch-smoke.sh` — asserts a kill-switch apply for `feature=smart_context state=disabled effective_at=next_request`; `event-stream-smoke.sh` validates events; `policy-apply-smoke.sh` asserts policy uses `Smart Context shadow, canary 0, auto-redeem disabled`.
- **L4 (GATED):** execute the `smart-context-shadow-canary-plan.md` sequence — shadow (measure before/after, no rewrite sent upstream) → canary 1% → live — each gated by owner approval + the exact-fallback criteria below.

**Pass criteria (per `smart-context-shadow-canary-plan.md`):**
- **shadow→canary gate:** no protocol-integrity warning, no tool-call-integrity warning, no secret in logs, event stream healthy, owner approval.
- **exact fallback:** malformed artifact reference / protocol-sensitive payload → exact pass-through (no corrupted JSON, tool-call ids preserved, continuation ids preserved);
- **immediate-disable triggers** (any one fires the kill switch): previous response not found after rewrite, invalid tool-call continuation, corrupted JSON, repeated missing-context recovery, secret appears, exact-fallback fails, user-visible regression linked to rewrite;
- kill switch produces a conformant event and the **next** request is exact/pass-through.
- All `not_validated` smart_context cells (Kiro/Antigravity/Cline/OpenCode #2/#4/#5/#6) default to DISABLED until owner ACCEPT + live proof.

**Evidence status now:** L1 GATED (no vendor `verified` cell; owner must ACCEPT borderline #7 + not_validated #2/#4/#5/#6). L2 GREEN. L3 GREEN (dry-run). L4 GATED (shadow/canary/live not executed; `deploy_owner_approved: false`).

---

### 3.7 `reset_claim_mode`

**Enum:** `codex_redeem | unsupported`

**Behavioral claim:** `reset_claim` is **Codex-specific and prodex-implemented**. Manual `prodex redeem <profile>` may be validated in controlled PROD only; `--auto-redeem` stays disabled until the validation checklist is satisfied. Redeem is attempted only under strict guards and never on 5h-only exhaustion or when another eligible profile exists. This is the **lowest-priority / cold** stream (F9), gated on real account state.

**Verification method:**
- **L1:** Codex `reset_claim_mode = codex_redeem` is `not_validated` (cell #1 — no linkable primary-source doc page for the redeem API/workflow; only prodex wrapper + Codex CLI `/usage` observed). All other vendors `unsupported` (`verified`).
- **L2:** unit — the guard conditions + decision matrix in `prod-redeem-validation-checklist.md` §2/§3 are the executable spec; redeem emits an audit event and respects cooldown/idempotency.
- **L3:** DRY-RUN `policy-apply-smoke.sh` — asserts policy carries `auto-redeem disabled`; no redeem command is run in dry-run.
- **L4 (GATED):** controlled-PROD execution of `prod-redeem-validation-checklist.md` matrix on a real Codex profile (no-credit / credit-present / near-reset / weekly-exhausted / 5h-only / all-exhausted / non-OpenAI / invalid-profile), with scrubbed evidence and owner approval.

**Pass criteria (per `prod-redeem-validation-checklist.md`):**
- redeem attempted ONLY when: provider is OpenAI/Codex, profile approved, weekly window exhausted, no other eligible profile has weekly quota, reset not imminent, kill-switch `auto_redeem` not disabled, cooldown allows, audit sink healthy;
- redeem NOT attempted on: 5h-only exhaustion, thin/critical with other profiles available, non-OpenAI provider, invalid profile auth, audit unavailable, no owner approval;
- outcomes match the matrix: `redeem_no_credit` (no retry storm), `redeem_succeeded` (same profile retried), `redeem_rejected` (reason reset imminent), 5h-only rejected, non-OpenAI unsupported, invalid profile fail-closed;
- `--auto-redeem` promotion gate: ≥1 no-credit case + ≥1 rejected guard + ≥1 success/not-available outcome + cooldown/idempotency verified + owner approval.
- evidence carries profile alias only (no raw OAuth token / cookie / `auth.json` / raw backend response).

**Evidence status now:** L1 GATED (Codex cell #1 `not_validated` → owner must ACCEPT or REJECT; planning DONE per open-items F9). L2 GREEN (spec + guards defined). L3 GREEN (dry-run, auto-redeem disabled). L4 GATED (deferred on real account state; lowest priority).

---

## 4. Cross-capability conformance invariants (span all 7)

These are not single-capability checks; they must hold for the **combination** of enabled capabilities and are the hard gates from `docs/contracts/l2-conformance-notes.md` + `f0-readiness-matrix.md`:

| Invariant | Verification | Pass criteria |
|---|---|---|
| One runtime router per session | one-router acceptance tests #1–#7 (unit) + `session-start-stop-smoke.sh` (L3) + L4 counter assertion | `go_rotation_decision_count == 0` AND `l2_selection_count <= 1` AND `l2_fallback_after_committed == 0` for L2-owned sessions |
| Go sends desired state only; Rust routes in-flight | `policy-apply-smoke.sh` (L3) + event stream non-routing test #6 | Go emits `selection`/`affinity`/`fallback` events for observability/ledger only; never re-routes a committed request |
| Rotate only before commit | `quota_mode`/`rotation_mode` L2 + L4 | no mid-stream rotation; any rotation event timestamp precedes first committed response |
| Continuation affinity beats heuristics | `continuation_mode` L4 replay | continuation stays on originating profile across 30+ turns |
| Profile switch fails closed | `auth_mode` L3 `profile-fail-closed-smoke.sh` + L4 | invalid/missing profile → session aborts before traffic; no silent reuse |
| Smart Context exact fallback + kill switch | `smart_context_mode` L2/L3 + shadow-canary-live L4 | any integrity risk → exact pass-through; kill switch → next request exact |
| No secrets in logs/events/evidence | `redaction-smoke.sh` (L3) + redaction audit | `secrets_present == false` on every event; scrubber confirmation on evidence |
| Postgres shared state (no SQLite) | `state-backend-smoke.sh` (L3) + L4 | `shared_state_backend` check passes; `backend_type == postgres` (not sqlite) |

---

## 5. Evidence traceability map (capability → verifier)

| Capability | L2 unit/contract | L3 smoke script | L4 live executor |
|---|---|---|---|
| launch_mode | `prodex_test.go` (pin+LookPath) | `session-start-stop-smoke.sh` | isolated-launch live probe |
| auth_mode | `client.go` RegisterAccounts; `l2_runtime.go` fail-closed | `profile-fail-closed-smoke.sh` | real profile register + invalid-switch |
| quota_mode | `daemon_test.go` L2-owned suppression | `readyz-smoke.sh`, `state-backend-smoke.sh` | near-limit pre-commit rotation (F0) + L2 no-op |
| rotation_mode | 7 one-router acceptance tests (`l2_runtime.go`, `daemon_test.go`) | `session-start-stop-smoke.sh`, `kill-switch-smoke.sh` | L4 one-router counter assertion |
| continuation_mode | `client.go` StartSession router_owner + affinity | `session-start-stop-smoke.sh`, `event-stream-smoke.sh` | 30+ turn replay (`runtime-conformance-plan.md` §5) |
| smart_context_mode | `client.go` KillSwitch + event secret-reject | `kill-switch-smoke.sh`, `policy-apply-smoke.sh`, `event-stream-smoke.sh` | `smart-context-shadow-canary-plan.md` shadow→canary→live |
| reset_claim_mode | `prod-redeem-validation-checklist.md` guard matrix | `policy-apply-smoke.sh` (auto-redeem disabled) | `prod-redeem-validation-checklist.md` controlled-PROD matrix |

**Smoke harness state (from `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`):** all 8 scripts DRY-RUN GREEN against planned loopback `127.0.0.1:43117`. This proves harness correctness + contract wiring, **not** a live pass. LIVE execution is F0-gated.

---

## 6. F0-GATED live proof plan (execution order on owner approval)

Live proof is **not** run now. When the owner sets `deploy_owner_approved: true` (F7) AND resolves the F5 `not_validated` cells, execute in this order, recording scrubbed evidence under `.deploy-control/evidence/`:

1. **Liveness + readiness** (`readyz-smoke.sh` LIVE) → proves `launch_mode`/`auth_mode`/`quota_mode` L4 baseline.
2. **State backend** (`state-backend-smoke.sh` LIVE) → Postgres, no SQLite.
3. **Policy apply** (`policy-apply-smoke.sh` LIVE) → `rotation_mode`/`smart_context_mode`/`reset_claim_mode` desired-state push (shadow, canary 0, auto-redeem disabled).
4. **Account register + profile fail-closed** (`profile-fail-closed-smoke.sh` LIVE) → `auth_mode` L4.
5. **Session start/stop** (`session-start-stop-smoke.sh` LIVE) → `rotation_mode` one-router + `continuation_mode` affinity L4.
6. **Kill switch** (`kill-switch-smoke.sh` LIVE) → `smart_context_mode` disable L4.
7. **Event stream + redaction** (`event-stream-smoke.sh` + `redaction-smoke.sh` LIVE) → cross-capability no-secrets + event conformance.
8. **Single-router controlled session** → collect the 5 counters (acceptance test #7) → `rotation_mode`/`continuation_mode` exactly-one-router L4.
9. **Smart Context shadow** (then canary, then live — each separately gated) → `smart_context_mode` L4.
10. **Redeem validation** (lowest priority, cold, real Codex account) → `reset_claim_mode` L4.

Each step's pass criteria are the per-capability criteria in §3 + the cross-capability invariants in §4. Any RED aborts the sequence and rolls back per `docs/deploy/rollback-runbook.md`.

---

## 7. Owner decision dependencies (blocks before any L4)

| Gate | Blocks | Resolution |
|---|---|---|
| F5 not_validated cell #1 (Codex `reset_claim_mode`) | `reset_claim_mode` L4 | owner ACCEPT (deploy disabled-until-live) or REJECT |
| F5 not_validated cells #2/#4/#5/#6 (smart_context Kiro/Antigravity/Cline/OpenCode) | `smart_context_mode` for those vendors | owner ACCEPT (disabled-until-live) or REJECT |
| F5 borderline #3 (Antigravity `rotation_mode` unsupported) | `rotation_mode` for Antigravity | owner ACCEPT (conformant-by-disablement) or REJECT |
| F5 borderline #7 (Codex `smart_context_mode` inferred) | `smart_context_mode` Codex L4 | owner accept-inference or downgrade to not_validated |
| F5 borderline #8 (Antigravity `quota_mode` custom_probe/none) | `quota_mode` Antigravity L4 | owner accept or disable proactive rotation for Antigravity |
| F7 deploy/runbook approval (`deploy_owner_approved: true`) | ALL L4 | owner approval + runbook review |

Source: `docs/vendors/owner-acceptance-request.md` + `docs/contracts/f0-readiness-matrix.md` Owner Decision Gates.

---

## 8. Disjointness + references

- **Disjoint from** `docs/qa/runtime-conformance-plan.md` (that is the smoke S1–S5 + conformance C1–C6 + replay plan; this matrix is the per-capability index that maps capabilities to those tests).
- **Disjoint from** `docs/qa/smart-context-shadow-canary-plan.md` (that is the smart_context rollout sequence; this matrix references it as the `smart_context_mode` L4 executor).
- **Disjoint from** `docs/qa/prod-redeem-validation-checklist.md` (that is the redeem guard/matrix; this matrix references it as the `reset_claim_mode` L2/L4 executor).
- **Grounded in** `docs/vendors/vendor-capability-matrix.md` (L1 values), `docs/contracts/l2-conformance-notes.md` (L2 acceptance tests), `docs/contracts/f0-readiness-matrix.md` (Go/No-Go), `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md` (L3 state), `scripts/smoke/*.sh` (L3 harness), `multica-auth-work/server/internal/l2runtime/client.go` + `daemon/*.go` (L2 code).

---

## 9. Status

- **Delivered now (per dispatch):** per-capability verification method + pass criteria + evidence-status-now + live-proof-gate for all 7 capabilities (§3), cross-capability invariants (§4), evidence traceability (§5), F0-gated live proof plan (§6), owner decision dependencies (§7).
- **NOT delivered (F0-GATED, by design):** LIVE proof (L4) for any capability. `deploy_owner_approved: false`; only DRY-RUN smoke + unit/contract + static evidence exist.
- **Read-only provenance:** no product code or deploy was executed to produce this matrix. No files outside `docs/qa/capability-conformance-matrix.md` and the G7 check-in were created or edited by GLM#52#CLINE#A.

_Sign-off: GLM#52#CLINE#A — plan + criteria delivered 2026-07-04; live proof deferred to F0 owner gate._
