# T20 — Tier-20 capacity tests (STANDBY / test plan)

- agent `Opus48#B` · lane `T20-TESTS` · pane `w6:p2` · task `T20-TIER20-TESTS`
- check-in: `.deploy-control/p0/checkins/Opus48-B__T20-TIER20-TESTS__20260723T210904Z.json`
- posture: **READ-ONLY prep / STANDBY** — no tests written yet (target function not present; see gap).

## Current behavior (verified, `internal/daemon/`)

- **Admission-limit function that exists:** `effectiveTaskAdmissionLimit(agentBrainCfg, requested)`
  (`config.go:664`), called at `config.go:514` (`maxConcurrentTasks = effectiveTaskAdmissionLimit(...)`).
  Logic today:
  ```go
  if agentBrainCfg.DevelopmentEnabled && agentBrainCfg.Neutral.Gateway.Required {
      return agentBrainDevelopmentMaxTasks   // == 1; "tier-20 remains schema-only until task 9.2"
  }
  return requested
  ```
  ⇒ In dev + gateway-required it **unconditionally returns 1**, and does **not** consult the tier or the
  capacity gate. Otherwise it returns the requested value.
- **Config inputs parsed but not yet consumed for tier-20 honoring:**
  - `AGENT_BRAIN_TASK_CAPACITY_TIER` (`brain.EnvTaskCapacityTier`) → `tierValue`, default `"20"`,
    validated ∈ {20,50,100} (`config.go:763-777`; `brain/config.go CapacityTier.Validate`).
  - `AGENT_BRAIN_CAPACITY_GATE_ENABLED` → `capacityGateEnabled` bool, default **false**
    (`config.go:784-786`), stored as `AgentBrainIntegrationConfig.CapacityGateEnabled` (`config.go:800`;
    field doc `:128`).
  - Frozen tier schema (`brain/config.go:64-67`): tier-20 = `TierStateCanaryAuthorized`;
    tiers 50/100 = `TierStateEvidenceRequired` (blocked).

## GAP — target function not implemented

`effectiveAgentBrainCapacity` (named in this lane and referenced in comments `config.go:767` and
`config.go:781-783`: *"the gate is only consulted alongside CapacityTier==20 (see
effectiveAgentBrainCapacity), and tiers 50/100 remain blocked"*) is **NOT defined anywhere**
(`grep 'func .*effectiveAgentBrainCapacity'` → none; only the two comments). The tier-20-honoring
contract this lane targets — *honor 20 ONLY when tier==20 AND CapacityGateEnabled; default/dev=1;
50/100 blocked* — is therefore **not yet wired**; `effectiveTaskAdmissionLimit` still hard-returns 1.

**Consequence:** Go tests for `effectiveAgentBrainCapacity` cannot be written now (they would not
compile). This lane is legitimately STANDBY until the function lands.

## Ready-to-write test plan (apply once the function exists + manager go + test-file lock)

Table-driven, in `internal/daemon/config_test.go` (or a new `capacity_tier_test.go`), against the
implemented `effectiveAgentBrainCapacity`/updated `effectiveTaskAdmissionLimit`:

| dev+gateway-required | tier | CapacityGateEnabled | expected limit |
|---|---|---|---|
| true | 20 | true  | **20** (canary-authorized) |
| true | 20 | false | **1** (gate off → default) |
| true | 50 | true  | **1** (tier 50 blocked / evidence-required) |
| true | 100| true  | **1** (tier 100 blocked) |
| true | 50/100 | false | **1** |
| true | (default, no env) | (default false) | **1** (tier defaults 20 but gate off) |
| false (dev off) OR gateway not required | any | any | `requested` (non-gateway path unchanged) |

Assertions: only `(tier==20 && CapacityGateEnabled==true)` yields 20; every other dev+gateway combination
yields 1; tiers 50/100 never yield >1 regardless of gate; no hard-coding of 20 without both conditions.

## Status: BLOCKED (STANDBY)
- **Blocker:** `effectiveAgentBrainCapacity` (tier-20-honoring admission logic) not yet implemented;
  `effectiveTaskAdmissionLimit` currently ignores tier/gate (returns 1 in dev+gateway). Writing the tests
  now is impossible (no such symbol) and would be fabricated.
- **Owner:** implementation lane that adds `effectiveAgentBrainCapacity` (config/capacity) + Manager
  (w5:p1) for go-ahead and the exact test-file source lock.
- **Next action:** on the function landing, acquire the test-file lock, add the table above, run
  `go test ./internal/daemon/ -run 'Capacity|AdmissionLimit|Tier' -count=1` with `/tmp` caches, record
  exit + results, and check out.

## Non-claims
Read-only prep only; no test/source written; no fabricated tests for a missing symbol; no run. Advisory
plan pending implementation + manager authorization.

---

## WAVE UPDATE (2026-07-23T21:39Z) — tests added/confirmed + results

Both implementations landed since the standby note: `effectiveAgentBrainCapacity` (config.go:670) and
the strict-readiness `emitPredicates` (gateway/health_models.go:126). Env: `GOCACHE/GOTMPDIR/GOMODCACHE`
under `/tmp` (root full). Go `/home/ec2-user/goroot/go/bin/go`.

### ADDED — emitPredicates instrumentation test (NEW, W1-locked)
`internal/daemon/gateway/predicate_instrumentation_test.go` (same-package; constructs `&ReadinessChecker{}`
+ `SetDiagnosticsLogger` with a capturing `slog.Handler`, calls `emitPredicates` directly):
- `TestEmitPredicatesCapturesAllFieldsOnSuccess` — asserts msg `strict_readiness_predicate` + all fields
  (`route_model`,`protocol`,`live`,`authenticated`,`model_registry_ready`,`selected_model_ready`,
  `selected_protocol_ready`,`registry_version_present`,`ok`) and empty failure fields on success.
- `TestEmitPredicatesCapturesFailingSubcheckFromGatewayError` — `*GatewayError{operationReadiness,
  ErrorRateLimited,429}` → `fail_operation="readiness"`, `fail_error_class="rate_limited"`,
  `fail_status_code=429`, `ok=false`, `authenticated=false`, `registry_version_present=false`.
- `TestEmitPredicatesClassifiesNonGatewayError` — `context.Canceled` → `fail_error_class="non_gateway_error"`,
  `fail_operation=""`, `fail_status_code=0`.
- `TestEmitPredicatesNilDiagnosticsIsNoop` — nil `diag` → no panic (safe no-op).

### CONFIRMED — tier-20 gate tests (pre-existing, not modified)
`internal/daemon/tier20_enablement_test.go`: `TestTier20HonoredOnlyWhenGateEnabledAndTier20` +
`TestAgentBrainTier20SchemaRemainsFailClosedAtDevelopmentLimit` — matches the §matrix (gate off→1;
gate on + tier20→20; gate on + 50/100→blocked 1; non-20→1; dev off→requested).

### Results (exit codes)
| Command | Result | exit |
|---|---|---|
| `gofmt -l …/gateway/predicate_instrumentation_test.go` | clean | 0 |
| `go vet ./internal/daemon/gateway/` | clean | 0 |
| `go test ./internal/daemon/gateway/ -run 'EmitPredicates' -count=1` | 4/4 PASS (ok 0.004s) | 0 |
| `go test ./internal/daemon/ -run 'Tier20' -count=1` | 2/2 PASS (ok 0.020s) | 0 |
| `git diff --check …/predicate_instrumentation_test.go` | clean | 0 |

W1 added exactly ONE product file (the new gateway test); no gateway/daemon source edited (other gateway
worktree changes are other lanes'). No fabricated/zero-assertion pass.

---

## WAVE UPDATE 2 (2026-07-23T21:49Z) — FetchModels body/decode/version classification regression

ADDED `internal/daemon/gateway/fetchmodels_classification_test.go` (NEW, W1-locked; deterministic,
no live network — injected `roundTripFunc` transport + synthetic credential):
- `TestFetchModelsBodyReadFailureIsRetryableTransport` — 200 then body `Read` returns a connection error
  (`errorReadCloser`) → `IsErrorClass(err, ErrorTransport)` **and** `GatewayError.Retryable==true`, body closed.
  (matches client.go:184 "read failure after healthy status: retryable transport fault".)
- `TestFetchModelsInvalidJSONIsProtocol` — 200 with truncated/invalid JSON → `ErrorProtocol` (client.go:199).
- `TestFetchModelsRegistryVersionMismatchIsProtocol` — 200, body `registry_version="body-v1"` vs header
  `X-OmniRoute-Registry-Version="header-v2"` → `ErrorProtocol` (client.go:204). Header set via `Header.Set`
  so canonicalization matches the client's `Header.Get`.

### Results (exit codes)
| Command | Result | exit |
|---|---|---|
| `gofmt -l …/fetchmodels_classification_test.go` | clean | 0 |
| `go vet ./internal/daemon/gateway/` | clean | 0 |
| `go test ./internal/daemon/gateway/ -run 'FetchModels(BodyReadFailure|InvalidJSON|RegistryVersionMismatch)' -count=1` | 3/3 PASS (ok 0.002s) | 0 |
| `git diff --check …` | clean | 0 |

Distinction locked in: **transport (read/drain) faults after a healthy 200 are retryable ErrorTransport**;
**decode + registry-version-mismatch are non-retryable ErrorProtocol**. One product file created
(test-only); no gateway source edited by me.

---

## WAVE UPDATE 3 (2026-07-23T22:08Z) — do() context lifecycle (REAL httptest server) — PASS

ADDED `internal/daemon/gateway/do_context_lifecycle_test.go` (NEW, W1-locked). Uses a **real
`httptest.Server`** (injected transports don't honor request-context cancellation, so a real body is
required). Validates the `handedOff` + `cancelOnCloseBody` mechanism in `Client.do` (client.go:217-283).

| Test | Proves | Result |
|---|---|---|
| `TestClientDoDoesNotCancelBeforeLargeBodyRead` | 83 KiB body served in flushed 4 KiB chunks (2ms apart) is read IN FULL by the caller after `do()` returns — no "context canceled" abort; body is the `*cancelOnCloseBody` wrapper | **PASS (0.05s)** |
| `TestClientDoNonSuccessCancelsAndClosesImmediately` | 503 → nil response + `ErrorOverloaded`; error path does not hand off (handedOff=false → deferred `cancel()` fires immediately; body closed in `do`) | **PASS** |
| `TestClientDoBodyCloseIsIdempotent` | closing the wrapped success body twice → no double-close panic, both closes error-free; `cancel()` idempotent (no context leak) | **PASS** |

Commands (exit codes): `gofmt -l`=clean(0); `go vet ./internal/daemon/gateway/`=clean(0, lostcancel-clean);
`go test ./internal/daemon/gateway/ -run 'TestClientDo(...)' -count=1`= **3/3 PASS ok 0.050s (0)**;
`git diff --check`=clean(0).

**Verdict: PASS.** `do()` correctly defers request-context cancellation to `Body.Close()` on success
(full body readable), cancels immediately on error paths, and `Body.Close` is idempotent.

**CONCERN (environment, not code):** `-race` could not be run — `CGO_ENABLED=0` and no C compiler
(`gcc`/`cc`) present, and dependency/toolchain install is forbidden. The no-context-leak claim is
therefore backed by `go vet` lostcancel-clean + the deterministic idempotent close/cancel logic, **not**
by a race-detector run. Re-run with `-race` when a C toolchain is available for full confirmation.
