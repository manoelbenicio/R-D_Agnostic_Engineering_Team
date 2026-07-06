# F0 Readiness Matrix - `rpp.l2.v1` Go/No-Go Checklist

Status: GATED F0 OWNER GO/NO-GO CHECKLIST

Date: 2026-07-04

Scope:

- Contract: `docs/contracts/l2-runtime-contract.md`
- Runtime event schema: `docs/contracts/runtime-events.schema.json`
- Runtime event validation spec: `docs/contracts/runtime-event-validation-spec.md`
- Current Go evidence: `multica-auth-work/server/internal/l2runtime/client.go`, `multica-auth-work/server/internal/daemon/l2_runtime.go`, `multica-auth-work/server/internal/daemon/daemon.go`
- Smoke evidence: `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`

Legend:

- `DONE`: implemented and evidenced enough for this gate.
- `IN-PROGRESS`: implementation, harness, or dry-run evidence exists, but live sidecar evidence is still missing.
- `OWNER-GATED`: depends on owner acceptance or explicit owner approval before F0 can go live.

Go/no-go summary: F0 is **GATED**, not live-go. The previous F0 blockers for `StartSession` persistence, one-router suppression, and Go runtime-event validation are closed. The remaining go-live decision is tied to owner-approved LIVE smoke execution plus owner F5/F7 gates: vendor capability acceptance and deploy/runbook approval. Current dry-run evidence proves smoke harness correctness only; it is not live sidecar proof.

## Contract Endpoint Gates

| Gate | Required proof | Current status | Evidence pointer |
|---|---|---:|---|
| `HealthCheck` (`GET /healthz`) | Go can call L2 liveness over loopback with bearer auth and validate `contract_version == rpp.l2.v1` plus `status == alive`. | `IN-PROGRESS` | Client surface exists in `multica-auth-work/server/internal/l2runtime/client.go`; live liveness smoke remains gated by F7 owner approval. Dry-run harness evidence is recorded in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`. |
| Readiness as part of `HealthCheck` (`GET /readyz`) | Go fails closed unless policy, accounts, shared-state backend, bearer auth, kill switch, and event stream readiness pass. | `IN-PROGRESS` | `startL2SessionForTask` calls readiness and fails closed before `StartSession` in `multica-auth-work/server/internal/daemon/l2_runtime.go:60`; `readyz-smoke.sh` and `state-backend-smoke.sh` dry-run clean in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`. LIVE `readyz` execution is still gated. |
| `ApplyPolicy` (`POST /v1/policy/apply`) | Go pushes desired policy/budgets/feature rollout before traffic and fails closed on rejection. | `IN-PROGRESS` | Client method exists and validates accepted contract response in `multica-auth-work/server/internal/l2runtime/client.go`; `policy-apply-smoke.sh` was authored and dry-run green in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`. LIVE policy apply remains gated. |
| `RegisterAccounts` (`POST /v1/accounts/register`) | Go pushes approved profile metadata without secrets before `StartSession`; rejected profiles fail closed. | `IN-PROGRESS` | Client rejects wrong contract versions and non-empty `rejected_profiles` in `multica-auth-work/server/internal/l2runtime/client.go:304`; `profile-fail-closed-smoke.sh` was authored and dry-run green in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`. LIVE account/profile smoke remains gated. |
| `StartSession` (`POST /v1/session/start`) | Go validates `router_owner == rust_l2`, persists `runtime_router_owner = rust_l2`, and only then sends traffic. | `DONE` | Client validates `router_owner == rust_l2` in `multica-auth-work/server/internal/l2runtime/client.go:339`; daemon readiness/start/persist flow is in `multica-auth-work/server/internal/daemon/l2_runtime.go:47`; persistence fails closed at `multica-auth-work/server/internal/daemon/l2_runtime.go:94`; acceptance test `TestStartL2SessionPersistsRouterOwnerBeforeExecution` starts at `multica-auth-work/server/internal/daemon/daemon_test.go:1322`. |
| `StopSession` (`POST /v1/session/stop`) | Go sends idempotent stop and validates either a contract-bearing response or an explicitly documented `204 No Content`. | `IN-PROGRESS` | Client stop method exists in `multica-auth-work/server/internal/l2runtime/client.go:358`; `session-start-stop-smoke.sh` was authored and dry-run green in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`. LIVE stop semantics remain gated. |
| `RouteDecisionEvent` (`selection` / `affinity` / `fallback`) | Rust emits route decision events; Go ingests them as observability/ledger only and never reroutes in-flight requests from them. | `IN-PROGRESS` | Go non-routing ingest proof is `TestEventIngestNonRoutingDoesNotTriggerGoRotation` at `multica-auth-work/server/internal/daemon/daemon_test.go:1482`; B documented L2 emitter requirements in `docs/prodex/prodex-l2-event-emission.md` and marked exact schema emission as fork/adapter `a validar`. LIVE L2 emission remains gated. |
| `RuntimeEventStream` (`GET /v1/events/stream`) | Go establishes event stream, validates `rpp.l2.v1` events, rejects secret-bearing events, and writes observability/ledger only. | `IN-PROGRESS` | Event validation spec is DONE in `docs/contracts/runtime-event-validation-spec.md`; C implementation validates before handler in `multica-auth-work/server/internal/l2runtime/client.go:426` and `multica-auth-work/server/internal/l2runtime/client.go:475`; tests cover unknown `event_type`, wrong contract version, `secrets_present`, required fields, and valid selection in `multica-auth-work/server/internal/l2runtime/client_test.go:16` and `multica-auth-work/server/internal/l2runtime/client_test.go:67`. LIVE event stream smoke remains gated. |
| `KillSwitch` (`POST /v1/killswitch/apply`) | Go applies tenant/provider/profile/session feature kill switches through L2 and fails closed if not confirmed. | `IN-PROGRESS` | Client applies and validates kill-switch confirmation in `multica-auth-work/server/internal/l2runtime/client.go:385`; `kill-switch-smoke.sh` was authored and dry-run green in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`. LIVE kill-switch smoke remains gated. |

## Runtime Event Validation Gate

| Gate | Required proof | Current status | Evidence pointer |
|---|---|---:|---|
| Contract validation spec | For every schema `event_type`, define required envelope, event-specific fields, nested fields, hard reject rules, and test requirements. | `DONE` | `docs/contracts/runtime-event-validation-spec.md` lists all 19 `event_type` values from `docs/contracts/runtime-events.schema.json` and requires rejection for unknown types, `contract_version != rpp.l2.v1`, and `redaction.secrets_present == true`. |
| Go pre-write validation | Go validates runtime events before ledger/observability handler execution. | `DONE` | `StreamEvents` validates before calling the handler in `multica-auth-work/server/internal/l2runtime/client.go:457`; `validateRuntimeEvent` rejects unknown fields, missing common fields, wrong contract version, unknown event types, invalid producer/redaction, and event-specific required fields in `multica-auth-work/server/internal/l2runtime/client.go:475`; C check-in `.deploy-control/Codex-5.5-C__RUNTIME-EVENT-VALIDATION__20260704T195448Z.md` reports green `go build`, `go vet`, and tests. |
| Secret-bearing event rejection | Go rejects `redaction.secrets_present == true` before writing. | `DONE` | `validateRuntimeEventRedaction` returns `ErrSecretEvent` when `secrets_present` is true in `multica-auth-work/server/internal/l2runtime/client.go:569`; test case `secrets present` is in `multica-auth-work/server/internal/l2runtime/client_test.go:35`. |

## One-Router Acceptance Tests

| Acceptance test | Required proof | Current status | Evidence pointer |
|---|---|---:|---|
| 1. `StartSession` persistence test | A successful L2 `StartSession` response persists `runtime_router_owner == rust_l2` before backend execution; failure to persist fails closed. | `DONE` | `TestStartL2SessionPersistsRouterOwnerBeforeExecution` covers success and persisted owner at `multica-auth-work/server/internal/daemon/daemon_test.go:1322`; `TestStartL2SessionPersistenceFailureFailsClosed` covers fail-closed persistence at `multica-auth-work/server/internal/daemon/daemon_test.go:1398`. |
| 2. Proactive ledger no-op test | For L2-owned sessions, proactive ledger rotation returns no account and does not call Go rotation service/store mutation. | `DONE` | `TestL2OwnedTaskSuppressesLegacyGoRotationPaths` calls `maybeProactiveRotateFromLedger` and asserts no rotation service calls at `multica-auth-work/server/internal/daemon/daemon_test.go:1430`. |
| 3. Proactive stream-text no-op test | For L2-owned sessions, quota-warning text cannot trigger Go account/profile rotation. | `DONE` | `TestL2OwnedTaskSuppressesLegacyGoRotationPaths` calls `maybeProactiveRotateOnText` and asserts no rotation service calls at `multica-auth-work/server/internal/daemon/daemon_test.go:1461`. |
| 4. Reactive exhaustion no-op test | For L2-owned sessions, Go exhaustion detector cannot trigger retry-after-rotation or account mutation. | `DONE` | `TestL2OwnedTaskSuppressesLegacyGoRotationPaths` covers `rotateTaskOnExhaustion` and direct `rotateTaskWithReason` no-op at `multica-auth-work/server/internal/daemon/daemon_test.go:1464` and `multica-auth-work/server/internal/daemon/daemon_test.go:1470`. |
| 5. F0 compatibility test | With no `runtime_router_owner`, existing F0/prodex-as-is behavior remains unchanged. | `DONE` | `TestF0NoRouterOwnerAllowsLegacyGoRotation` proves legacy Go rotation remains allowed when no owner is recorded at `multica-auth-work/server/internal/daemon/daemon_test.go:1421`. |
| 6. Event-ingest non-routing test | Valid L2 runtime events write ledger/observability only and cannot call Go rotation. | `DONE` | `ingestL2RuntimeEvent` logs observability-only data in `multica-auth-work/server/internal/daemon/l2_runtime.go:139`; `TestEventIngestNonRoutingDoesNotTriggerGoRotation` asserts zero legacy rotation calls and unchanged router owner at `multica-auth-work/server/internal/daemon/daemon_test.go:1482`. |
| 7. Exactly-one-router assertion | For a test session, `runtime_router_owner == rust_l2`, Go rotation decision count is zero, and runtime events cannot cause Go fallback/reroute. | `DONE` | Combined proof: owner persistence in `TestStartL2SessionPersistsRouterOwnerBeforeExecution` at `multica-auth-work/server/internal/daemon/daemon_test.go:1322`; all legacy rotation paths suppressed in `TestL2OwnedTaskSuppressesLegacyGoRotationPaths` at `multica-auth-work/server/internal/daemon/daemon_test.go:1430`; event ingest non-routing in `TestEventIngestNonRoutingDoesNotTriggerGoRotation` at `multica-auth-work/server/internal/daemon/daemon_test.go:1482`. |

## Pre-Deploy Smoke Gates

| Smoke gate | Required proof | Current status | Evidence pointer |
|---|---|---:|---|
| `readyz` smoke | L2 readiness endpoint passes and Go refuses traffic on any failed readiness check. | `IN-PROGRESS` | `scripts/smoke/readyz-smoke.sh` was authored; DRY-RUN passed in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`. LIVE execution is owner-gated. |
| Policy apply smoke | Go applies policy and receives accepted `rpp.l2.v1` response before session start. | `IN-PROGRESS` | `scripts/smoke/policy-apply-smoke.sh` was authored; DRY-RUN passed in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`. LIVE execution is owner-gated. |
| Account/profile registration smoke | Go registers approved profiles with no secret payload and fails closed on rejected profiles. | `IN-PROGRESS` | Covered by `scripts/smoke/profile-fail-closed-smoke.sh`; DRY-RUN passed in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`. LIVE execution is owner-gated. |
| Session start smoke | Go starts L2 session, validates `router_owner == rust_l2`, and persists owner before traffic. | `IN-PROGRESS` | `scripts/smoke/session-start-stop-smoke.sh` was authored; DRY-RUN passed in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`. StartSession unit acceptance is DONE, but LIVE sidecar smoke is still gated. |
| Session stop smoke | Go stops L2 session idempotently and validates/documented success semantics. | `IN-PROGRESS` | `scripts/smoke/session-start-stop-smoke.sh` was authored; DRY-RUN passed in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`. LIVE sidecar stop smoke is still gated. |
| Kill switch smoke | Go applies L2 kill switch by scope and confirms `effective_at`; failure blocks rollout. | `IN-PROGRESS` | `scripts/smoke/kill-switch-smoke.sh` was authored; DRY-RUN passed in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`. LIVE execution is owner-gated. |
| Event stream smoke | Go establishes event stream, rejects wrong contract version and secret-bearing events, and validates required event families. | `IN-PROGRESS` | `scripts/smoke/event-stream-smoke.sh` was authored; DRY-RUN passed in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`. Go validation tests are DONE; LIVE stream execution remains owner-gated. |
| Redaction smoke | No secrets appear in logs/traces/events/evidence; Go independently rejects secret-bearing runtime events. | `IN-PROGRESS` | `scripts/smoke/redaction-smoke.sh` was authored by the F7 security/state lane and dry-run passed in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`; static redaction audit notes runtime capture is still planned, not executed. LIVE redaction smoke remains gated. |
| Profile fail-closed smoke | Missing/invalid profile, profile-home escape, or non-isolated auth fails closed before runtime traffic. | `IN-PROGRESS` | `scripts/smoke/profile-fail-closed-smoke.sh` was authored and dry-run passed in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`. LIVE sidecar execution remains owner-gated. |
| State backend smoke | Shared runtime state backend is Postgres/Redis and fails closed on shared SQLite. | `IN-PROGRESS` | `scripts/smoke/state-backend-smoke.sh` was authored by the F7 security/state lane and dry-run passed in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`. LIVE backend verification remains owner-gated. |
| Single-router smoke | L2-owned session suppresses all legacy Go rotation and proves exactly one router per session. | `IN-PROGRESS` | Unit acceptance is DONE across the seven one-router checks above; `scripts/smoke/session-start-stop-smoke.sh` dry-run passed in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`. LIVE controlled session proof remains owner-gated. |

## Owner Decision Gates

| Gate | Required owner decision | Current status | Evidence pointer |
|---|---|---:|---|
| F5 vendor capability acceptance | Owner accepts or resolves the `not_validated`/borderline vendor capability cells before enabling dependent providers/features. | `OWNER-GATED` | Gemini#Pro F5 check-in `.deploy-control/Gemini-Pro__RPP-VENDORMATRIX__20260704T181523Z.md` reports owner sign-off required; `.deploy-control/evidence/open-items.md` records F5 as a blocker for deploy gate. |
| F7 deploy/runbook approval | Owner approves real PROD/live smoke execution by setting `deploy_owner_approved: true` and reviewing the runbook package. | `OWNER-GATED` | `docs/deploy/prod-rollout-runbook.md` and `docs/deploy/l2-sidecar-deploy-plan.md` state current real deploy is NO-GO until owner approval; `.deploy-control/evidence/status-board.md` records `deploy_owner_approved: false`. |
| LIVE smoke execution | Owner-approved live sidecar smoke suite must pass and produce scrubbed evidence. | `OWNER-GATED` | Eight smoke scripts exist and DRY-RUN evidence is green in `.deploy-control/evidence/smoke-dry-run-20260704T201249Z.md`; no LIVE evidence has been authorized or recorded. |
| Treat F0 prodex-as-is launch as separate from `rpp.l2.v1` sidecar readiness | Owner accepts that current prodex-as-is launch can proceed only under ADR guardrails and does not satisfy the sidecar contract gates without the validated adapter/fork evidence above. | `OWNER-GATED` | ADR/OpenSpec separate near-term prodex-as-is rollout from target sidecar facade; B's `docs/prodex/prodex-l2-event-emission.md` marks exact schema emission as fork/adapter `a validar`. |

## Go/No-Go Conclusion

Current owner checklist result: **GATED F0 / NO LIVE GO YET**.

Closed since the prior matrix:

1. `StartSession` persistence is DONE.
2. One-router behavior is DONE across the seven acceptance checks.
3. Runtime event validation spec is DONE.
4. Go runtime-event validation implementation and pre-handler rejection are DONE.
5. Eight smoke scripts are authored and DRY-RUN green.

Remaining gates before F0 live go:

1. Owner F5 acceptance for unresolved vendor capability risk.
2. Owner F7 approval for runbook/deploy/live-smoke execution.
3. LIVE sidecar smoke execution with scrubbed evidence for readiness, policy, profile fail-closed, session start/stop, kill switch, event stream, redaction, state backend, and single-router behavior.
