# W1 — D6 Implementation (P0-W1-IMPLEMENTATION-D6): Cline pre-launch fail-closed contract

- agent: `Opus48#B` · pane `w6:p2` · lane `W1` · task `P0-W1-IMPLEMENTATION-D6`
- check-in: `.deploy-control/p0/checkins/Opus48-B__P0-W1-IMPLEMENTATION-D6__20260721T234143Z.json`
- adjudication adopted: `.deploy-control/p0/handoffs/D6-prelaunch-adjudication.md` (Opus48#C) — minimum contract EXACTLY
- Go: `/home/ec2-user/goroot/go/bin/go` → `go1.26.1`; repo HEAD `a6d50986…`
- scope: D6 only (assert/env/buildLaunch + coupled assert tests). No config-byte checks in assert.go.
  No Kimi ID, no Opus48, no auth/credential/quota/fallback, no live run. D7 NOT started.

## Locks (transition)
Checked out `P0-W1-IMPLEMENTATION` (DONE; evidence `W1-implementation.md`), checked in
`P0-W1-IMPLEMENTATION-D6` with the 16 product locks **plus** `runtimeenv/assert.go` and
`runtimeenv/assert_test.go` + artifact `W1-implementation-D6.md`. Zero-overlap tool-enforced
(assert.go/assert_test.go were unlocked by all active records).

## Applied edits (adjudication §2–§3, exact)

### assert.go ×3 (`internal/daemon/runtimeenv/assert.go`)
- **D6-1** `AssertPreLaunch` CLI switch: added `case brain.CLIOpenAICompatible: if plan.CodexConfig != nil { return ErrPreLaunchPolicy }`. No carrier-byte validation (stays content-free; `ValidateClineConfigBytes` owns bytes).
- **D6-2** `trustedEntryAllowed`: added `case brain.CLIOpenAICompatible` → `HOME`,`CLINE_DATA_DIR` require `originTrustedLocal`; `ClineOmniRouteAPIKeyEnv` requires `originTrustedSecret`.
- **D6-3** `launchRootsAreControlled`: restructured to a switch; Cline case requires `codexHome==""` and `exactPathWithin(executionRoot, environment.clineDataDir, entries["CLINE_DATA_DIR"].value)`; the non-Codex/non-Cline default now requires both `codexHome==""` and `clineDataDir==""`.

### env.go ×1 (`internal/daemon/runtimeenv/env.go`) — D6-4 (corrects R1-DESIGN D2)
- Added field `ChildEnvironment.clineDataDir string`.
- `BuildGatewayEnvironment` populates `child.clineDataDir = opts.Adapter.ClineDataDir` for `CLIOpenAICompatible` (exact analogue of the codexHome block). This is the trusted authoritative path D6-3 compares against the mutable entries value.

### brain_integration.go (`buildLaunch`) — D6 wiring
- Before `BuildGatewayEnvironment`: for `CLIOpenAICompatible`, create controlled `filepath.Join(env.RootDir, "cline-data")` (0700), then `ValidateExecutionRoot`.
- `AdapterEnvironment` literal now passes `ClineDataDir: clineDataDir`.
- Added `else if plan.Task.Request.CLIKind == brain.CLIOpenAICompatible` branch: `NewClineConfigContract(gatewayBaseURL, RouteModel, time.Now().UTC().Format(time.RFC3339))` → `execenv.WriteCredentiallessClineConfig(clineDataDir, contract.Bytes())`; manifest stays empty (carrier excluded per home.go). `updatedAt`=RFC3339 UTC per R1-DESIGN recommendation (minimal, non-clock; `validHeaderValue`-safe).

### assert_test.go coupled tests (W1-owned) — adjudication §6
Added `clineControlledDirectories`/`buildControlledClineEnvironment` helpers +:
- `TestAssertPreLaunchAcceptsControlledClinePlan` (accept: ClineDataDir in root, CodexConfig nil, carrier-excluded manifest).
- `TestAssertPreLaunchRejectsClineDataDirOutsideExecutionRoot`, `…Traversal`, `…ClinePhysicalDataDirSubstitution` (symlink), `…ClineDeniedOriginAndCodexConfig` (wrong-origin secret; non-nil CodexConfig).

## Verification (focused only; R3 daemon tests NOT run)
- `gofmt -l` clean on assert.go, env.go, brain_integration.go, assert_test.go.
- `go test ./internal/daemon/runtimeenv/ -run 'Assert|Cline' -count=1` → **ok (0.006s)**.
- `go build ./internal/daemon/ ./internal/daemon/execenv/ ./internal/daemon/runtimeenv/ ./pkg/agent/` → **all OK** (source build-clean; `_test.go` excluded).
- My D6 product-source edits: `runtimeenv/adapter.go`, `runtimeenv/env.go`, `runtimeenv/assert.go`,
  `internal/daemon/config.go`, `internal/daemon/execenv/cline_home.go`, `internal/daemon/brain_integration.go`
  (cumulative D1–D6); my test edit: `runtimeenv/assert_test.go`. The other modified `runtimeenv/*_test.go`
  + new `adapter_test.go` are **R1's**, not W1.

## R3 SIGNAL
`brain_integration.go`, `config.go`, `cline_home.go` source is **build-clean** (daemon+execenv+pkg/agent
build OK). **R3 (Opus48#D) is CLEAR to run** its focused daemon/execenv tests
(`brain_integration_test.go`, `config_test.go`, `execenv/cline_home_test.go`). W1 did not run daemon R3
tests (ownership).

## Infra
Disk reached 100% during build/module re-download (build cache + module cache were cleared earlier to
recover from a prior 100% event). Freed via `go clean -cache -modcache` (regenerable only; no project or
credential deletion); ~286M free now.

## Status: BLOCKED (D6 applied + focused-green; awaiting R3 daemon tests + D7 authorization)
- **Blocker:** D3/D4/D5/D6 daemon-side behavior is validated by R3's focused tests
  (`config_test.go`/`brain_integration_test.go`/`execenv/cline_home_test.go`) which W1 must not run;
  and D7 (models.go GLM `cp/` prefix + models_test.go lockstep) is a separate authorized step not yet
  greenlit. Live runs remain `live_runs_authorized=false`.
- **Owner:** Principal Orchestrator (+ A9-R3 for daemon focused tests).
- **Next action:** R3 runs its focused daemon/execenv tests against the build-clean source; on green,
  Principal authorizes D7 (GLM catalog prefix + models_test.go lockstep) and, later, the single reserved
  GLM live run (gated by BLK-AVAIL + BLK-25B). No source edit/test/live run by W1 while blocked.

---

## D7 APPLIED (2026-07-21, after R3 D3–D6 focused tests GREEN)

R3 focused tests green (execenv 0.005s; daemon 5 tests 0.020s; gofmt clean). D7 authorized: GLM catalog ID only.

- **models.go:562** `{ID:"cline-pass/glm-5.2"…}` → `{ID:"cp/cline-pass/glm-5.2"…}` (frozen A3 SUB-1). Kimi row `:563` (`cline-pass/kimi-k2.7-code`) and NIM default `:572`/`nim.go:25` (`z-ai/glm-5.2`) **UNTOUCHED** (verified by rg).
- **models_test.go:172** lockstep expectation `cline-pass/glm-5.2` → `cp/cline-pass/glm-5.2` (Kimi expectation unchanged).
- Verification: `gofmt -l` clean; `go test ./pkg/agent/ -run 'StaticModels|Cline' -count=1` → **ok (0.016s)**.
- No Kimi ID / NIM default / Opus48 / auth / fallback / live-run touched.

---

## FINAL W1 IMPLEMENTATION EVIDENCE — D1–D7 COMPLETE (authorized GLM-ready scope)

### Complete changed-file set (product)

**W1-owned SOURCE (7 files, D1–D7):**
| File | Step | Change |
|---|---|---|
| `internal/daemon/runtimeenv/adapter.go` | D1 | `CredentiallessAdapterContract(CLIOpenAICompatible)` → `AdapterReady`/`ProtocolOpenAIChat` |
| `internal/daemon/runtimeenv/env.go` | D2, D6-4 | `AdapterEnvironment.ClineDataDir` + `trustedAdapterEntries` Cline case; `ChildEnvironment.clineDataDir` + populate |
| `internal/daemon/config.go` | D3, D4 | `agentBrainBuiltInCLIFor` `cline` mapping; `Validate` accepts `CLIOpenAICompatible` |
| `internal/daemon/execenv/cline_home.go` | D5 | `WriteCredentiallessClineConfig` + `prepareCredentiallessClineHome` |
| `internal/daemon/brain_integration.go` | D6 | `buildLaunch` Cline branch (data dir, contract write, RFC3339 updatedAt) |
| `internal/daemon/runtimeenv/assert.go` | D6-1/2/3 | pre-launch Cline accept + trusted-entry + CLINE_DATA_DIR containment |
| `pkg/agent/models.go` | D7 | GLM catalog ID → `cp/cline-pass/glm-5.2` (Kimi/NIM untouched) |

**W1-owned TESTS (2 files):** `runtimeenv/assert_test.go` (D6 coupled), `pkg/agent/models_test.go` (D7 lockstep).

**NOT W1 (consumed green, disjoint owners):** R1 tests — `runtimeenv/{env_test.go,isolation_g4_test.go,model_test.go,adapter_test.go}`; R3 tests — `internal/daemon/{brain_integration_test.go,config_test.go,execenv/cline_home_test.go}`.

Diffstat (product): 15 tracked files +622/−29, plus new `runtimeenv/adapter_test.go` (R1). W1 source diff is small and surgical (models.go/models_test.go ±2 lines each; adapter.go ±14; env.go ±29; assert.go ±33; config.go ±6; cline_home.go +43; brain_integration.go +30).

### Validation (focused; each package validated once, no duplication)
| Suite | Command | Result | Owner |
|---|---|---|---|
| runtimeenv D1/D2 | `go test ./internal/daemon/runtimeenv/…` (R1) | GREEN (reported) | R1 |
| runtimeenv Assert/Cline (D6) | `go test ./internal/daemon/runtimeenv/ -run 'Assert|Cline' -count=1` | **ok 0.006s** | W1 |
| execenv (D5) | R3 focused | GREEN 0.005s (reported) | R3 |
| daemon (D3/D4/D6) | R3 focused, 5 tests | GREEN 0.020s (reported) | R3 |
| pkg/agent (D7) | `go test ./pkg/agent/ -run 'StaticModels|Cline' -count=1` | **ok 0.016s** | W1 |
| build-clean | `go build ./internal/daemon/ ./…/execenv/ ./…/runtimeenv/ ./pkg/agent/` | **all OK** | W1 |
| gofmt | `gofmt -l` on all W1-touched files | clean | W1 |

### Scope compliance
- Applied exactly D1–D7 for the GLM-ready Cline (`CLIOpenAICompatible`) route + Opus48 route config-acceptance.
- **Not touched:** exact Kimi RouteModel (BLK-KIMI), Opus48 ID/config wiring, NIM default (DEC-NIM), any auth/credential/quota/retry/fallback, Prodex, gateway internals. **No live run** (`live_runs_authorized=false`).

### Remaining P0 (NOT W1 source work; externally gated)
- Single reserved **GLM live run** (`live_runs.cline_glm`) — gated by BLK-AVAIL (OmniRoute enriched registry rows) + BLK-25B (key-revocation security stop) + Principal live-run token. Closes 5.6/8.1/8.2 (GLM).
- Kimi (5.7) one-line fixtures on BLK-KIMI clear; Opus48 (5.8) config value on BLK-OPUS48 clear. These are not W1 source edits.

## Status: DONE — W1 authorized source implementation (D1–D7) complete and focused-green.
W1's GLM-ready serial implementation is complete under the 16+2 locks. No further authorized W1 source
work remains; the GLM live run is an externally-gated reserved acceptance, not a W1 source blocker.
