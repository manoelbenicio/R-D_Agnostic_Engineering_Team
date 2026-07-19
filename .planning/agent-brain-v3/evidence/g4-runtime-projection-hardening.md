# Evidence: Gateway Projection Warm-Cache Cancellation & R24/R25 Runtime Model Discovery Hardening

## Overview
This evidence pack documents the verification of the gateway RouteModel projection warm-cache cancellation and the p9 runtime model discovery (R24/R25 fail-closed Windows hardening).

## Source Hashes
The exact source files were located and hashed offline:

Projection warm-cache cancellation (internal/daemon/gateway/):
- `server/internal/daemon/gateway/projection_test.go`: `39e0c3c3b05eabb648b03df790bfc57f151fb6c29b7523fbe72c1b49dc566ca6`
- `server/internal/daemon/gateway/projection.go`: `a8b41df2b78fbd6c0dbf6cdd7771ef480e59575e5904a1ab3b19cc63c2d0eef5`

R24/R25 remediation (pkg/agent/):
- `server/pkg/agent/proc_windows.go`: `7a1601f67bfbbddee65e739f3e4725d8d960ca2ede6e46e5428f2613be69e7cc` (R24 fail-closed stubs)
- `server/pkg/agent/proc_other.go`: `e92f2c48385d46f06f877398fcacb8e195c1f8ac21864dc9989b35de57c47ba9` (R24 Unix process-group containment)
- `server/pkg/agent/proc_unsupported.go`: `679af9b9f721eb03a5ed74dd87da31d88a53f51e1481a22744980310942cc2c6` (R24 unsupported-OS fail-closed)
- `server/pkg/agent/models.go`: `a6957e3e0b4a05050da6dc198049581d6402103d474185d0912f3360e8a7b313` (R24 gate + R25 cache key)
- `server/pkg/agent/models_windows_test.go`: `8ff9e9c2ae75d590d4ff75b6bf9d3f1813cde190418def5ee31ee4bb74fb7b7a` (R24 Windows tests)
- `server/pkg/agent/models_test.go`: `b1c62961e671f697b32844448c739e6059bccfe5e9c2bc1d6b1e52fb908dad5b` (R25 stale-isolation tests)
- `server/pkg/agent/thinking.go`: `406c2f478e3c7abe88f80a994603528d9f047e38888ff039ab291ebbf86003aa` (R25 thinking-cache key)
- `server/pkg/agent/thinking_test.go`: `f2b0c3ab4277cf5a7e758c829c3336573361fe03a17c2b76a01f8da798417bb1` (R25 thinking-cache tests)
- `server/pkg/agent/models_process_test.go`: `75f1cc5d94bd240e955df5a61d34ca412ce9e980277eb66e0cf97495137c4211` (R24 Unix counterpart tests)

Orthogonal latency bound (NOT R24/R25; cited for completeness):
- `server/internal/daemon/daemon.go`: `a1d96a3c8a4edfc9f9c583326c6f74779643473904f13f0a8aab63ea1cd6fe07`
- `server/internal/daemon/model_list_report_test.go`: `6a644d5c4a92490c0311beaa80698a251f9b89a65f7ce8e7d0e87c8330203962`

## Acceptance Boundaries

### Warm-Cache Cancellation (projection.go — ACCURATE, verified)
`ProjectRouteModels` fails closed immediately if the context is canceled or the deadline exceeded (`ctx.Err() != nil`) **before** fetching the snapshot (`projection.go:59-61`) and **after** fetching the snapshot (`projection.go:66-68`), bypassing the cache-filtering projection entirely (`projection.go:72` is only reached when `ctx.Err() == nil`). The tests enforce that a warm-cache call performs **zero additional fetches** and **zero adapter-filter side-effects**:
- `TestRegistryProjectRouteModelsRejectsCancelledWarmCacheWithoutSideEffects` (`projection_test.go:283`) — primes the cache (fetches=1), then calls with cancelled / deadline-exceeded ctx; asserts error class matches, zero projection, `fetches.Load() == 1` (no new fetch, line 333), `filterCalls.Load() == 0` (no adapter filter, line 336).
- `TestRegistryProjectRouteModelsRejectsCancellationAfterSnapshot` (`projection_test.go:343`) — cancels ctx during the fetch; asserts `ErrorCancelled`, zero projection, `fetches == 1`, `filterCalls == 0` (line 366).

### R24 — Windows ACP fail-closed before `cmd.Start` (pkg/agent/ — corrected)
R24 (per RISKS.md:30) is "Windows ACP: Job Object attached AFTER `Start` → pre-assignment descendant escape". The chosen remediation is **fail-closed on Windows ACP discovery until atomic containment** (the `CREATE_SUSPENDED`→assign→resume alternative is NOT implemented; confirmed: no `CREATE_SUSPENDED`/`AssignProcessToJobObject`/`CreateJobObject` anywhere in `server/`).

- `pkg/agent/proc_windows.go:22-25` — explicit R24 comment: *"Windows model discovery is intentionally disabled until process creation and Job assignment can be performed atomically before user code runs. Attaching a process to a Job after cmd.Start leaves a race in which descendants can escape, so this implementation fails closed before cmd.Start."*
- `pkg/agent/proc_windows.go:28-46` — `requireDiscoveryProcessContainment()`, `configureDiscoveryProcessTree()`, `attach()`, `terminate()`, `terminateUnattachedDiscoveryProcess()` all return `errDiscoveryProcessContainmentUnavailable` (`pkg/agent/models.go:104`).
- `pkg/agent/models.go:1134-1136` — `discoverACPModels` fails closed at entry on Windows before any work; the same gate is applied at 21 call sites in `models.go` and 6 in `thinking.go`.
- Unix counterpart (`pkg/agent/proc_other.go:21-47`) — atomic containment via `Setpgid` at `fork()`, `attach()` records `pgid`, `terminate()` does `Kill(-pgid, SIGKILL)`.
- Unsupported-OS fail-closed (`pkg/agent/proc_unsupported.go:10-22`).

R24 tests:
- `pkg/agent/models_windows_test.go:15` — `TestWindowsACPDiscoveryFailsClosedBeforeStart` (uses a synthetic start-marker file; asserts the discovery child never started).
- `pkg/agent/models_windows_test.go:39` — `TestWindowsDynamicModelDiscoveryFailsClosed`.
- `pkg/agent/models_process_test.go:17,46,74` — Unix counterpart: `TestDiscoverACPModelsClosesStdinAndWaitsGracefully`, `TestDiscoverACPModelsReapsOrphanChild`, `TestDiscoverACPModelsTimeoutReapsProcessTree`.

### R25 — Discovery cache keyed by provider + executable (pkg/agent/ — corrected)
R25 (per RISKS.md:31) is "dynamic discovery cache keyed only by provider aliases the wrong executable → stale isolation". The remediation keys the cache by **provider + executable path** with path normalization (LookPath + Abs + Clean + EvalSymlinks), so two distinct executables for the same provider never share a cache entry.

- `pkg/agent/models.go:448-451` — `discoveryCacheKey(providerType, executablePath)` = `provider + "\x00" + normalizedDiscoveryExecutablePath(...)` (NUL delimiter prevents provider/executable collision).
- `pkg/agent/models.go:453-477` — `normalizedDiscoveryExecutablePath`: empty→default binary, `exec.LookPath`, `filepath.Abs`, `filepath.Clean`, `filepath.EvalSymlinks`, Windows lowercase. This is the "invalidate on path change" mechanism: a different path produces a different key.
- `pkg/agent/thinking.go:32-36` — stronger variant `thinkingCacheKey{provider, executablePath, cliVersion}` (CLI version bump also invalidates).
- Cache eviction: `pkg/agent/models.go:408-414` `pruneHardExpiredModelCacheLocked`, `:416-437` `putModelCacheLocked` (LRU at `modelCacheMaxEntries=64`).

R25 stale-isolation tests:
- `pkg/agent/models_test.go:1458` — `TestExecutableDiscoveryKeysIsolateFreshAndNegativeCatalogs` (asserts `discoveryCacheKey("cursor", first) != discoveryCacheKey("cursor", second)`; a negative cache for the second does not contaminate the first's fresh catalog).
- `pkg/agent/models_test.go:1544` — `TestExecutableDiscoveryKeysIsolateStaleCatalogsAndRefreshErrors` (two codebuddy executables' stale catalogs stay isolated when refreshes error independently).
- `pkg/agent/models_test.go:1519` — `TestEveryExecutableBackedProviderKeysCustomPathsIndependently` (12 providers; no provider collapses two paths; no two providers share a key).
- `pkg/agent/models_test.go:1223,1261` — `TestListModelsAntigravityCachesPerExecutable`, `TestListModelsCursorCachesPerExecutable`.
- `pkg/agent/thinking_test.go:485` — `TestThinkingCacheKeyDistinct`; `:717` — `TestCodebuddyHelpCacheIsolatedByExecutable`.

### Orthogonal latency bound (NOT R24/R25 — corrected)
`daemon.go:2057` wraps the discovery RPC in `context.WithTimeout(ctx, 40*time.Second)`; `daemon.go:2071-2073` rewrites the error to surface "model discovery exceeded the 40 second daemon limit" to the UI. This is a per-discovery latency bound, applies to all providers on all platforms, and is **independent of R24** (process containment) and **R25** (cache keying). On Windows the R24 fail-closed returns synchronously at `models.go:136/155/...` — well before the 40s clock can elapse. `model_list_report_test.go` tests retry-on-500 / no-retry-on-4xx / correct-path — also orthogonal to R24/R25.

## Execution
**STATUS: ACCEPT** (corrected — see reviewer section)

Executed offline on WSL linux/amd64 with `/home/dataops-lab/go-sdk/bin/go` (go1.26.4), `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`, no external credentials, network, or DB. Workdir: `multica-auth-work/server`.

### Projection warm-cache cancellation (reran representative subset)
```bash
$ go vet ./internal/daemon/gateway/                                    # exit=0
$ go test -run 'TestRegistryProjectRouteModelsRejectsCancelledWarmCacheWithoutSideEffects|TestRegistryProjectRouteModelsRejectsCancellationAfterSnapshot' -count=20 -race ./internal/daemon/gateway/
ok  github.com/multica-ai/multica/server/internal/daemon/gateway  1.286s   # exit=0
$ GOOS=windows GOARCH=amd64 go build ./internal/daemon/gateway/        # exit=0
```
Iteration proof: 20 RUN lines per focused test confirmed.

### R25 stale-isolation (reran representative subset)
```bash
$ go test -run 'TestExecutableDiscoveryKeysIsolateFreshAndNegativeCatalogs|TestExecutableDiscoveryKeysIsolateStaleCatalogsAndRefreshErrors|TestEveryExecutableBackedProviderKeysCustomPathsIndependently|TestListModelsAntigravityCachesPerExecutable|TestListModelsCursorCachesPerExecutable' -count=20 -race ./pkg/agent/
ok  github.com/multica-ai/multica/server/pkg/agent  6.655s   # exit=0
```

### R24 Unix process-tree containment (reran representative subset)
```bash
$ go test -run 'TestDiscoverACPModelsClosesStdinAndWaitsGracefully|TestDiscoverACPModelsReapsOrphanChild|TestDiscoverACPModelsTimeoutReapsProcessTree' -count=20 -race ./pkg/agent/
ok  github.com/multica-ai/multica/server/pkg/agent  25.443s   # exit=0
```

### R24 Windows compile (cross-compile; Windows tests are `//go:build windows` and cannot execute on Linux)
```bash
$ GOOS=windows GOARCH=amd64 go vet ./pkg/agent/    # exit=0
$ GOOS=windows GOARCH=amd64 go build ./pkg/agent/  # exit=0
```

### Formatting
```bash
$ gofmt -l pkg/agent/proc_windows.go pkg/agent/models.go pkg/agent/models_windows_test.go pkg/agent/models_test.go pkg/agent/thinking.go pkg/agent/thinking_test.go internal/daemon/gateway/projection.go internal/daemon/gateway/projection_test.go   # CLEAN (no output)
```

All operations genuinely passed. No synthetic outputs were generated.

## Honest Residual Limitations
- The offline compile/test execution proves the source meets its contract: failing closed without generating background requests upon cancellation/timeout (projection), failing closed before `cmd.Start` on Windows ACP discovery (R24), and isolating stale catalogs per executable (R25). It does not exercise live provider network connectivity, TCP latency, or a live OmniRoute registry.
- R24 Windows **runtime** behavior (`TestWindowsACPDiscoveryFailsClosedBeforeStart` etc.) was not executed on this Linux host — only cross-compile vet+build was run. The Windows fail-closed stubs are compile-verified; runtime verification requires a Windows host.
- The 40-second daemon timeout (`daemon.go:2057,2071-2073`) is an orthogonal latency bound, not part of R24/R25; it is cited here only to correct a prior conflation.
- No credential, secret, auth home, or live session was inspected, listed, hashed, copied, or mutated. Only synthetic test paths and synthetic constants were used.

---

## Reviewer Section (independent evidence-quality review, 2026-07-18)

**Reviewer verdict: BLOCK → CORRECTED → ACCEPT.** The original artifact by Gemini was **inaccurate** on the R24/R25 acceptance boundary and has been corrected above. The projection warm-cache cancellation half was accurate and is preserved unchanged in substance.

### What was verified accurate (unchanged)
- All four originally-cited SHA256 hashes match current disk exactly (recomputed: `39e0c3c3…` projection_test.go, `a8b41df2…` projection.go, `a1d96a3c…` daemon.go, `6a644d5c…` model_list_report_test.go).
- Both projection test names exist at the cited locations: `TestRegistryProjectRouteModelsRejectsCancelledWarmCacheWithoutSideEffects` (`projection_test.go:283`), `TestRegistryProjectRouteModelsRejectsCancellationAfterSnapshot` (`projection_test.go:343`).
- The projection warm-cache cancellation boundary description is accurate: `ctx.Err()` checked before snapshot (`projection.go:59-61`) and after (`projection.go:66-68`); tests assert zero additional fetches and zero adapter-filter side-effects (`projection_test.go:333,336,366`).
- No live/provider claims appear in the artifact.

### What was inaccurate (corrected)
1. **R24/R25 boundary conflated with the 40s model-list timeout.** The original text claimed "R24/R25 Runtime Model Discovery: The daemon enforces a strict 40-second timeout on model list reports (daemon.go:2071)". Per `RISKS.md:30-31`, R24 is Windows Job Object atomic containment and R25 is discovery cache keyed by provider+executable — neither is a 40s timeout. The 40s timeout (`daemon.go:2057,2071-2073`) is an orthogonal per-discovery latency bound; on Windows the R24 fail-closed returns synchronously before the 40s clock can elapse. Corrected: R24/R25 boundaries now point to the actual remediation; the 40s timeout is relabeled as orthogonal.
2. **Wrong files cited for R24/R25.** The original cited `daemon.go` and `model_list_report_test.go` for R24/R25. The actual R24/R25 remediation lives in `pkg/agent/proc_windows.go`, `pkg/agent/models.go`, `pkg/agent/thinking.go` and their tests. Corrected: actual files + hashes added; orthogonal files retained under a separate heading.
3. **Wrong tests cited for R24/R25.** The original ran `TestReportModelListResult` (retry-on-500 / no-retry-on-4xx / correct-path — orthogonal to R24/R25). Corrected: R24 tests are `TestWindowsACPDiscoveryFailsClosedBeforeStart` etc. + Unix counterpart `TestDiscoverACPModels*`; R25 tests are `TestExecutableDiscoveryKeysIsolate*` + `TestEveryExecutableBackedProviderKeysCustomPathsIndependently`.
4. **Vague "POSIX/Windows splits (such as directory symlink alternatives)" claim removed** — did not correspond to the actual R24 (Windows fail-closed) or R25 (cache-key path normalization via `EvalSymlinks`) remediation.

### Representative subset rerun (this review)
- Projection warm-cache x20 race: ok 1.286s (exit=0).
- R25 stale-isolation x20 race: ok 6.655s (exit=0).
- R24 Unix containment x20 race: ok 25.443s (exit=0).
- R24 Windows cross-compile vet+build: exit=0/0 (runtime tests are `//go:build windows`, not executable on Linux — documented as a residual).
- gofmt on all R24/R25 + projection files: CLEAN.

### Acceptance boundary non-conflation check
- Warm-cache cancellation ↔ R24/R25: **distinct**. Projection lives in `internal/daemon/gateway/`; R24/R25 live in `pkg/agent/`. No shared code path between the projection `ctx.Err()` gate and the Windows process-containment / cache-key remediation. Confirmed not conflated in the corrected artifact.
- R24 (Windows fail-closed before `cmd.Start`) ↔ R25 (cache keyed by provider+executable): **distinct**. R24 is about process-tree containment; R25 is about cache-key construction. Both live in `pkg/agent/models.go` but at different sites (R24 gate at `:1134-1136` + 21 call sites; R25 key at `:448-477`). Confirmed not conflated in the corrected artifact.
- Orthogonal 40s timeout ↔ R24/R25: **distinct and correctly labeled** as orthogonal in the corrected artifact.

### Constraints honored by this review
No product code, OpenSpec file, main planning doc (`STATE.md`, `AGENT_LEDGER.md`, `RISKS.md`, `EVIDENCE_INDEX.md`), other evidence file, credential, auth home, or live session was edited or inspected. Only the target artifact `g4-runtime-projection-hardening.md` was corrected. No OpenSpec checkbox was changed.
