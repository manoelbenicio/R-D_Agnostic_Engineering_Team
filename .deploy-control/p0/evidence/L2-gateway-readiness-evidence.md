# L2 — Gateway / Readiness evidence (P0-6H-L2-GATEWAY)

- agent: `Opus48#B` · lane `L2` · pane `w6:p2` · task `P0-6H-L2-GATEWAY`
- ownership (exclusive): `server/internal/daemon/gateway/**` only
- must-not-touch: central L1, runtimeenv L3, adapters L4, observability L5–L7, OpenSpec/GSD
- check-in receipt: `.deploy-control/p0/checkins/CHECKIN__Opus48-B__L2__P0-6H-L2-GATEWAY__20260722T040345Z.json`
- plan_ref: `.planning/agent-brain-v3/P0_SIX_HOUR_EXECUTION_PLAN.md`; prompt_ref: `…P0_SIX_HOUR_AGENT_PROMPTS.md#L2`
- OpenSpec: 5.1, 5.2, 6.1 · AB-REQ-07..13,33
- posture: verification + minimal-fix; **no inference, no live run, no secret read, no source edit needed.**

## Preflight
cwd `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team` · git HEAD `a6d50986…` · git status count 255
(shared worktree incl. other lanes) · go `go1.26.1` (`/home/ec2-user/goroot/go/bin/go`) · node `v22.23.1` · disk 2.6G free.

## Commands + exit codes (focused, gateway only)
| Command | Result | exit |
|---|---|---|
| `gofmt -l internal/daemon/gateway/` | empty (clean) | 0 |
| `go vet ./internal/daemon/gateway/` | clean | 0 |
| `go test ./internal/daemon/gateway/ -count=1` | `ok … 0.278s` | 0 |
| `git diff --check -- internal/daemon/gateway/` | clean | 0 |

## Acceptance verification

### A. Readiness distinguishes liveness / catalog / model / protocol as DISTINCT gates — REVIEWED
`gateway/health_models.go` `ReadinessChecker.CheckGatewayReadiness` sets five independent gates in order:
1. `snapshot.Live` ← `client.CheckLiveness` (**unauthenticated** probe, distinct `endpoints.Liveness`).
2. `snapshot.Authenticated` ← `client.CheckReadiness` (**authenticated** probe, distinct `endpoints.Readiness`).
3. `snapshot.ModelRegistryReady` ← `registry.Snapshot(ctx).Version != ""` (catalog gate).
4. `snapshot.SelectedModelReady` ← model present in registry **and** `model.Available`.
5. `snapshot.SelectedProtocolReady` ← `model.Capability.Protocol == request.Protocol`.
Final `policy.Evaluate(snapshot)` (strict, fail-closed) gates admission. Liveness, authenticated-catalog,
model-registry, selected-model and selected-protocol are separate booleans — not collapsed.

### B. `/v1/models` is NOT treated as proof of inference — REVIEWED
`gateway/client.go` issues only GET `liveness` / `readiness` / `/v1/models`. `FetchModels` decodes the
catalog into the registry; a 200 on the readiness/`/v1/models` path sets **only** `Authenticated=true`.
Model + protocol readiness are **separate registry-backed gates** (A.4/A.5). There is **no** chat/
completions/messages/inference request anywhere in the package. Catalog presence ≠ inference capability.

### C. Brain implements no credential/account lifecycle, retry, or provider fallback — REVIEWED
- Secret: `CredentialSource.WithCredential` exposes the value only for the single HTTP call, sets
  `Authorization: Bearer …` then **immediately `request.Header.Del("Authorization")`**; `Client.String()`
  redacts; `GatewayError` carries no value. No secret is read/printed/hashed by L2.
- No account selection/rotation/quota/refresh/revocation logic in the package. `ModelDocument.{Rotation,
  Affinity,Fallback,AccountPool}` are **parsed telemetry/metadata** from OmniRoute's registry, not
  implemented behavior.
- No internal retry/failover: `do()` performs a single `httpClient.Do`. `GatewayError.Retryable` is a
  **classification flag** (timeout/transport) for the caller, not a retry loop. Retry/failover/circuit are
  OmniRoute-owned.

### D. Deterministic fail-closed — VERIFIED (test)
`TestFrozenTier20CanaryPolicyIsFailClosed` (gateway suite, green) asserts a non-omniroute `RouterOwner`
is rejected with `ErrorInvalidConfiguration`; error classification is centralized in `errors.go`. Cancel/
timeout map to `ErrorCancelled`/`ErrorTimeout` deterministically (`classifyTransportError`).

## Pre-existing modification found in owned tree (NOT authored by L2)
`gateway/contracts_test.go:65` is uncommitted-modified in the shared worktree:
`policy.RouterOwner = brain.RouterOwnerLegacyGo` → `policy.RouterOwner = brain.RouterOwner("alternate_router")`.
This is a **test-only, behavior-equivalent** change (both are non-omniroute owners the frozen policy must
reject); the suite passes with it. L2 did **not** author it and did **not** revert it (revert/clean is
prohibited). Flagged for the manager/L8; no functional impact on the fail-closed assertion.

## Verdict
- **VERIFIED:** gofmt/vet/test on `gateway/` all green (exit 0); deterministic fail-closed proven by suite.
- **REVIEWED (static):** distinct readiness gates; `/v1/models`-not-inference; no credential/account
  lifecycle/retry/fallback in the Brain gateway.
- **files_changed by L2: none** — package already correct; no minimal fix warranted (no busywork).

## 6.1 evidence (metadata-only)
Immutable revision `a6d50986…`; readiness gate model above is metadata/structural + green tests. This is
**not** a live readiness probe against a running OmniRoute and involves **no network, inference, or
secret**. Live readiness/tier runs remain externally gated (`control.json` capacity/live-run gates).

## Non-claims
No live run, no inference, no `/v1/models` treated as inference, no secret read/print/hash, no source
edit, no OpenSpec/GSD edit, no deploy/restart/Docker/systemd, no commit/push, no reset/stash/revert/clean.
