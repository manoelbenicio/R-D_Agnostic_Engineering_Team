# REG — gateway-required routing regression (OpenSpec 2.1)

- agent: **Opus48#C** · lane **config-regression** · task **REG-GATEWAY-REQUIRED** · pane `w8:p1`
- sole mutable lock: `.deploy-control/p0/evidence/REG-gateway-required.md` (this file)
- check-in: `.deploy-control/p0/checkins/Opus48-C__REG-GATEWAY-REQUIRED__20260722T215611Z.json`
- as-of (UTC): `2026-07-22T21:57Z` · HEAD `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- posture: **READ-ONLY on product code.** No product/test edit; no install; no OmniRoute internals probed. Verification of existing focused regressions + real PASS evidence.

## 0. Verdict

**VERIFIED (regression present + green).** The daemon gateway-required mode is covered by existing focused regressions proving: (a) approved tasks route via OmniRoute with `GatewayRequired=true` and `RouterOwner=omniroute`; (b) any non-`omniroute` router owner / dual router is **rejected**; (c) native/alternate/direct-provider fallback is refused (fail-closed) on outage or non-readiness. This satisfies **OpenSpec tasks.md 2.1** ("Restrict router identity to `omniroute` and reject non-gateway task contracts") and the `omniroute-agent-routing` spec. **No new test was required and no product code was edited** (verify lane).

## 1. Spec anchors (OpenSpec)

- `tasks.md` **2.1** (checked): *Restrict router identity to `omniroute` and reject non-gateway task contracts.*
- `specs/omniroute-agent-routing/spec.md`:
  - §3–4 "OmniRoute is the only router owner": *SHALL accept only `omniroute` as `RouterOwner`; unknown/historical/alternate values MUST be rejected* (+ scenario: any other owner → rejected).
  - §11 admission *SHALL require OmniRoute liveness, authentication, model-registry, selected-model and selected-protocol readiness*.
  - §18 *Main Brain MUST NOT route directly to a provider, start an alternate router, select an account, or retry through another provider/model path when OmniRoute is unavailable or returns an error* (native-fallback prohibition).
- `specs/agent-brain-runtime/spec.md`:10 "OmniRoute plan is mandatory": *Every model task SHALL have an admitted plan with `RouterOwner=omniroute` before a child process is created.*

## 2. Regression coverage → assertion → spec map

| Requirement (2.1) | Test:symbol | Exact assertion | Result |
|---|---|---|---|
| Accept `RouterOwner=omniroute`, `GatewayRequired=true` | `brain/brain_test.go: TestTaskRequestSeparatesCLIModelAndOwner` | `TaskRequest{RouterOwner: RouterOwnerOmniRoute, GatewayRequired:true}.Validate()==nil` | PASS |
| **Reject** non-omniroute router owner (dual/legacy) | same test | `request.RouterOwner = "alternate_router"; Validate()` returns error (fatal if accepted) | PASS |
| gateway-required task routes via OmniRoute (positive plan) | `daemon/brain_integration_test.go` (`TestAgentBrainSyntheticChild` path, L76) | `plan.Task.Request.RouterOwner == brain.RouterOwnerOmniRoute && plan.Profile.ID == gateway.ProfileAnthropicMessages` | PASS |
| Legacy task translates to omniroute owner | `brain/brain_test.go: TestLegacyTranslationAndTokenRedaction` | `TranslateLegacyTask(...,gatewayRequired=true).Request.RouterOwner == RouterOwnerOmniRoute` | PASS |
| **Reject** dual router at admission (before gateway access) | `daemon/brain_integration_test.go: TestAgentBrainRejectsDualRouterBeforeGatewayAccess` | `task.RuntimeRouterOwner="alternate_router"; admitTask(...)` returns error ("dual router task was admitted" if nil) | PASS |
| Admission fail-closed on non-readiness (no fallback) | `daemon/brain_integration_test.go: TestAgentBrainFailsClosedWhenGatewayNotReady` | unready gateway → `admitTask` error; `snapshot().Readiness != Ready` | PASS |
| Admission fail-closed taxonomy | `brain/g2a_test.go: TestGatewayAdmissionFailsClosed` (subtests: gateway_unavailable, authentication_failed, model_registry_unavailable, model_unavailable, protocol_unavailable) | each fails closed | PASS (5/5) |
| Strict readiness gate | `brain/brain_test.go: TestStrictReadinessFailsClosed` | unauthenticated snapshot rejected | PASS |
| No native/direct fallback on outage (§18) | `brain/recovery_test.go: TestRecoveryModeGatewayOutageFailsClosedWithoutFallback` | outage → `DEGRADED` owner `none` (no alternate router); restore requires ready OmniRoute | PASS |
| No cross-model fallback in initial route | `brain/brain_test.go: TestInitialRouteHasNoCrossModelFallback` | `routes[0].Fallback.CrossModelFallback == 0 && PreCommitOnly` | PASS |
| Authenticated OmniRoute health contract only | `daemon/brain_integration_test.go: TestAgentBrainUsesInstalledOmniRouteHealthContract`, `...DegradedPingFailsClosedBeforeAuthenticatedReadiness` | uses installed `/api/health/ping` + authenticated `/v1/models`; degraded ping fails closed | PASS |

## 3. Evidence — exact commands + exit codes (real, no fake PASS)

Run from `multica-auth-work/server`, `GOROOT=/home/ec2-user/goroot/go`, `GOCACHE=/tmp/l5-gocache`, `GOTMPDIR=/tmp`:

```text
go test ./internal/daemon/brain/ -run 'TestTaskRequestSeparatesCLIModelAndOwner|TestStrictReadinessFailsClosed|TestLegacyTranslationAndTokenRedaction|TestGatewayAdmissionFailsClosed|TestRecoveryModeGatewayOutageFailsClosedWithoutFallback|TestInitialRouteHasNoCrossModelFallback' -count=1 -v
  → PASS (all; TestGatewayAdmissionFailsClosed = 5/5 subtests) ; ok internal/daemon/brain 0.012s ; exit 0
go test ./internal/daemon/ -run 'AgentBrain' -count=1 -v
  → PASS (incl. RejectsDualRouterBeforeGatewayAccess, FailsClosedWhenGatewayNotReady, UsesInstalledOmniRouteHealthContract, DegradedPingFailsClosed, SyntheticChild positive plan) ; ok internal/daemon 0.016s ; exit 0
```

## 4. Scope / non-claims
- READ-ONLY: no product or test file edited; only this evidence file written. `git diff --check` on this file: PASS.
- `-race` not run (cgo/gcc absent in env); coverage is deterministic unit/contract level.
- No OmniRoute internals probed/mapped/tested (provider/account/credential/session/rotation/model-mapping untouched); synthetic gateway (`newSyntheticGateway`) is an in-process httptest stub, not a live OmniRoute.
- No install; `go.mod`/`go.sum` unchanged.
- OpenSpec 2.1 checkbox already `[x]`; this artifact records reproducible regression evidence and is NOT a checkbox mutation.
- **Untargeted-chat live acceptance** (campaign header) is OUT OF this read-only config-regression lane: it needs a live-run token (`live_runs.*=false`), the daemon/auth/OmniRoute live pipeline, and Kiro-TL delegation orchestration — not executed here; no fake PASS asserted for it.

## 5. Status
- STATUS: DONE (VERIFIED).
- DELIVERED: regression→assertion→spec map (§2) + real PASS evidence (§3) for gateway-required=true, router_owner=omniroute, and native/dual/direct fallback rejection (OpenSpec 2.1).
- FILES: created this evidence file only. No code changed.
- HANDOFF: none required — coverage exists and is green. If the Principal wants a single consolidated named regression, existing tests already assert every clause; adding a duplicate is unnecessary (no duplicate review).
