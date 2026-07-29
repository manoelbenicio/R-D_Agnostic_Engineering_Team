# R3 — Credentialless launch lifecycle: focused test contract

- agent: Opus48#D · lane: R3-TEST-DESIGN · task: P0-LIFECYCLE-TEST-CONTRACT
- pane: `w8:p2` (HERDR_ENV=1) · git HEAD: `a6d5098`
- check-in: `.deploy-control/p0/checkins/Opus48-D__P0-LIFECYCLE-TEST-CONTRACT__20260721T224238Z.json`
- upstream: consumes the confirmed gaps in `.deploy-control/p0/handoffs/A6-lifecycle-gap-matrix.md`
  (L3c Cline credentialless admission+launch, L7b controlled-home cleanup, 8.2 cancel/usage/terminal).
- posture: **design only. Product source is READ-ONLY.** This doc is the only mutable artifact.
  It defines exact test names, files, fixtures, assertions and pass/fail semantics for the delta
  owners (A1/R1 author `runtimeenv`/`execenv` tests; W1 authors the `daemon` package `brain_integration`
  test; A9 runs them). **R3 does not author product code or product tests.**
- guardrails honored: no broad regression, no second QA, no live run, no OmniRoute internals,
  no auth/credential/quota/retry/failover tests, no Prodex. Each test is the *minimum* delta directly
  proving one changed behavior.

## 0. Tool preflight evidence (canonical path)

| Tool | Command | Result |
|---|---|---|
| go | `/home/ec2-user/goroot/go/bin/go version` | `go version go1.26.1 linux/amd64` |
| gofmt | `/home/ec2-user/goroot/go/bin/gofmt -h` | present |
| git | `git --version` | 2.50.1 |
| python3 | `python3 --version` | 3.9.25 |
| rg | `rg --version` | 15.2.0 |

Pane `$HOME` is a credential slot; use the canonical `/home/ec2-user/goroot/go/bin/{go,gofmt}`
(go1.26.1). Runner commands below use this absolute path.

## 1. Existing fixtures/helpers to REUSE (do not re-invent)

- `daemon` package (`internal/daemon/brain_integration_test.go`):
  `newSyntheticGateway(t, ready bool)`, `syntheticAgentBrainConfig(t, url)`, `syntheticGatewayTask()`,
  `syntheticCredentialSource{}`, `countingSyntheticCredentialSource{}`, `syntheticReferenceSecret`,
  the re-exec isolation-child pattern (`exec.Command(os.Args[0], "-test.run=...")` + `command.Env =
  launch.Environment.Exec()` + exit-code assertions), `launch.Environment.Keys()`/`.Exec()`,
  `containsString`.
- `runtimeenv` package (`assert_test.go`): `controlledTestDirectories(t)` → `(executionRoot,
  taskHome, codexHome)`, `NewStableSecret(syntheticSecret)`, `BuildGatewayEnvironment(ComposeOptions{
  Adapter: AdapterEnvironment{CLI, GatewayRoot, TaskHome, CodexHome, StableSecret}})`,
  `LaunchPlan{Environment, CodexConfig, ExecutionRoot, TaskHome:[]HomeEntry{{RelativePath}}}`,
  `AssertPreLaunch`, `ErrPreLaunchPolicy`, `createTestDirectorySymlink`.
- `runtimeenv` package (`cline_test.go`): `clineTestRoot="http://127.0.0.1:20128"`,
  `clineTestUpdatedAt`, `clineAgentBrainRoutes` (`cline-kimi-k2.7-dedicated`, `cp/cline-pass/glm-5.2`),
  `NewClineConfigContract`, `ValidateClineConfigBytes`, `ClineSecretReferenceSentinel`,
  `ClineOmniRouteAPIKeyEnv`, `ClineOpenAICompatibleProviderID`, `ClineTokenSourceOmniRoute`.

All route models in tests stay synthetic/loopback; **no real GLM/Kimi ID is invented** — tests are
parameterized over `clineAgentBrainRoutes` so they bind to A3's frozen IDs without change.

## 2. Test contract by gap

Each row: exact `func` name · file (package) · precondition · assertions · pass/fail. New source
symbols the delta owner must add are marked `[needs impl]` — the test compiles only after the owner
adds them; until then the test is **held with the impl** (never committed red).

### 2.1 — L3c-A: adapter acceptance for the Cline family

| Field | Value |
|---|---|
| `TestCredentiallessAdapterAcceptsOpenAICompatibleClineRoute` | `runtimeenv/adapter_test.go` (pkg `runtimeenv`) |
| precondition | `[needs impl]` A1/R1 flips `CredentiallessAdapterContract(CLIOpenAICompatible)` from `AdapterFailClosed` to `AdapterReady` once OmniRoute Chat-Completions is accepted (A3). |
| assertions | `c,err := CredentiallessAdapterContract(brain.CLIOpenAICompatible)`; `err==nil`; `c.State==AdapterReady`; `c.Protocol==brain.ProtocolOpenAIChat`; `c.Gate==""`; no `FallbackFrontends` auto-selected. |
| negative (keep) | `CredentiallessAdapterContract(brain.CLIKimi)` and any unknown kind still return an `*AdapterGateError` (fail-closed) — Kimi rides the OpenAI-compatible Cline carrier, not a native Kimi adapter, unless A3 accepts it explicitly. |
| pass/fail | pass = ready contract + correct protocol; fail-closed default preserved for unaccepted kinds. |

### 2.2 — L3c-B: credentialless Cline config writer (materialization)

| Field | Value |
|---|---|
| `TestWriteCredentiallessClineConfigMaterializesProvidersJSON` | `runtimeenv/cline_test.go` or `execenv/cline_home_test.go` (owner's package) |
| new symbol | `[needs impl]` `WriteCredentiallessClineConfig(clineDataDir string, raw []byte) error` (mirror of `execenv.WriteCredentiallessCodexConfig`), writing `data/settings/providers.json` under a controlled per-task Cline data dir. |
| precondition | `dir := t.TempDir()`; `contract,_ := NewClineConfigContract(clineTestRoot, brain.RouteModel(model), clineTestUpdatedAt)` for each `model` in `clineAgentBrainRoutes`. |
| assertions | writer returns nil; target file exists at the exact installed carrier relpath (`data/settings/providers.json`); file mode `0600`, containing dir mode `0700`; bytes round-trip `ValidateClineConfigBytes(read, contract.Model(), contract.BaseURL(), clineTestUpdatedAt)==nil`; `strings.Contains(read, ClineSecretReferenceSentinel)` **and** `!strings.Contains(read, syntheticSecret)` (no secret value on disk). |
| idempotency | second write over an existing file replaces content, never appends, never follows a symlink (pre-remove `Lstat` symlink guard → error). |
| pass/fail | pass = exact-path + mode + sentinel-only + valid schema; fail if any secret value, wrong mode, symlink, or wrong relpath. |

### 2.3 — L3c-C: `buildLaunch` Cline branch (env wiring + secret injection)

| Field | Value |
|---|---|
| `TestBuildLaunchOpenAICompatibleWiresClineDataDirAndSecret` | `internal/daemon/brain_integration_test.go` (pkg `daemon`, **W1-authored** — this file is in a W1 hotspot) |
| new behavior | `[needs impl]` `buildLaunch` adds a `plan.Task.Request.CLIKind == brain.CLIOpenAICompatible` branch: creates a controlled Cline data dir under `taskHome` (`filepath.Join(env.RootDir,"agent-brain-home",...)`), calls the L3c-B writer, sets child env `CLINE_DATA_DIR`/`CLINE_SANDBOX_DATA_DIR` inside the execution root and `CLINE_OMNIROUTE_API_KEY` resolved from the single OmniRoute secret via `CredentialSource.WithCredential` (sentinel→value), appends the Cline manifest entries to `AssertPreLaunch`. |
| fixture | reuse `newSyntheticGateway(t,true)`, a `syntheticAgentBrainConfig` variant with `CLIKind=CLIOpenAICompatible` + a `clineAgentBrainRoutes` RouteModel; `syntheticCredentialSource{}`; `prepared := &execenv.Environment{RootDir:t.TempDir(), WorkDir:...}`. |
| assertions | `launch,err := runtime.buildLaunch(ctx, plan, prepared, local, custom)`; `err==nil`; `keys := launch.Environment.Keys()`; **present:** `CLINE_DATA_DIR`, `CLINE_OMNIROUTE_API_KEY`, `MULTICA_SESSION_ID`, `MULTICA_REQUEST_ID`, `MULTICA_ROUTER_OWNER`; **absent (forbidden):** `OPENAI_API_KEY`, `OPENAI_BASE_URL`, `NVIDIA_API_KEY`, `NIM_BASE_URL`, `CODEX_HOME`, `ANTHROPIC_AUTH_TOKEN`; `CLINE_DATA_DIR` value is `strings.HasPrefix(v, prepared.RootDir)`; providers.json exists under it and validates via `ValidateClineConfigBytes`. |
| pass/fail | pass = correct controlled data dir + injected stable secret + no provider/native creds; fail if any direct-provider var, data dir outside root, or missing providers.json. |
| `TestBuildLaunchOpenAICompatibleChildIsolation` + `TestAgentBrainSyntheticChildCline` | same file — re-exec isolation child (mirror of `TestAgentBrainSyntheticChild`): child asserts `CLINE_DATA_DIR` under `HOME`/root, `providers.json` present, `CLINE_OMNIROUTE_API_KEY` set, and forbidden vars (`OPENAI_API_KEY`,`NVIDIA_API_KEY`,`CODEX_HOME`,`ANTHROPIC_AUTH_TOKEN`,`MULTICA_L2_ENABLED`,`MULTICA_PRODEX_ENABLED`) absent; distinct exit codes (e.g. 51–56) per violation. |

### 2.4 — L3c-D: `AssertPreLaunch` extended for the Cline data-dir layout

| Field | Value |
|---|---|
| `TestAssertPreLaunchAcceptsControlledClinePlan` | `runtimeenv/assert_test.go` (pkg `runtimeenv`) |
| precondition | `executionRoot,taskHome,_ := controlledTestDirectories(t)`; `BuildGatewayEnvironment` with `AdapterEnvironment{CLI:brain.CLIOpenAICompatible, GatewayRoot, TaskHome, StableSecret}` `[needs impl in adapter env for the Cline data dir]`. |
| assertions | `AssertPreLaunch(LaunchPlan{Environment, ExecutionRoot, TaskHome:[]HomeEntry{{RelativePath:"cline-data/data/settings/providers.json"}}})==nil`. |
| `TestAssertPreLaunchRejectsClineDataDirOutsideOrTraversal` | same file — Cline data dir outside `ExecutionRoot`, or a `..` traversal in `CLINE_DATA_DIR`, or a symlinked data dir (`createTestDirectorySymlink`) → `errors.Is(err, ErrPreLaunchPolicy)`. |
| `TestAssertPreLaunchRejectsClineCredentialArtifact` | same file — a `data/settings/providers.json` whose bytes fail the no-secret contract, or an `auth.json`/API-key file under the Cline home → `ErrPreLaunchPolicy`. |
| pass/fail | pass = controlled in-root data dir accepted, every escape/secret rejected. |

### 2.5 — L3c-E: admission fail-closed regression guard (documents current state; NVIDIA exclusion)

| Field | Value |
|---|---|
| `TestAdmitTaskFailsClosedForUnacceptedClineAdapter` | `internal/daemon/brain_integration_test.go` (pkg `daemon`) |
| purpose | Until L3c-A lands, prove the Cline family is **rejected**, not silently launched: `admitTask` with `CLIKind=CLIOpenAICompatible` returns an `*agentBrainAdmissionError` whose `class=="adapter_fail_closed"`; snapshot readiness never `Ready`. Delete/flip this guard **in the same change** that flips the adapter (2.1), so the suite never asserts two contradictory states. |
| `TestAdmitTaskRejectsNVIDIAOwnedClineRoute` | same file — a `RouteModel` in the `nvidia/...` namespace for a Cline task is rejected at admission (Brain never selects the OmniRoute-owned NVIDIA fallback), aligning with `runtimeenv` `TestNewClineConfigContractRejectsNVIDIAFallbackRoute`. |
| pass/fail | pass = deterministic fail-closed class before acceptance / NVIDIA never Brain-selected. |

### 2.6 — L7b: controlled task home + Cline data dir reclamation

| Field | Value |
|---|---|
| `TestGCReclaimsControlledAgentBrainHomeWithEnvRoot` | `internal/daemon/gc_test.go` (pkg `daemon`) |
| precondition | build an env root with a child `agent-brain-home/` (0700) and a nested Cline `data/settings/providers.json`; write the same `GCMeta` `runTask` writes (reuse `gcMetaForTask` + `execenv.WriteGCMeta`); set the parent record to a cleanup-eligible state (mirror `TestShouldCleanTaskDir_*`). |
| assertions | after the GC pass, the env root (incl. `agent-brain-home` and the Cline data dir) is removed; while the active-root guard is held (`markActiveEnvRoot`) the controlled home is **skipped** (mirror `TestShouldCleanTaskDir_ActiveEnvRootSkipsFullCleanup`). |
| pass/fail | pass = reclaimed only when eligible and unguarded; never while active. No new secret ever lingers. |

### 2.7 — 8.2 Main-Brain semantics: cancel / terminal / usage for the Cline family

These reuse proven patterns; add a Cline-family case only where the launch branch changed (fold into
the single authorized W1 live acceptance for the wire path — **unit** here, never a second live run).

| Field | Value |
|---|---|
| `TestBuildLaunchClineReleasesCapacityOnPreLaunchFailure` | `brain_integration_test.go` (pkg `daemon`) — inject a writer/AssertPreLaunch failure in the Cline branch; assert `buildLaunch` returns a typed error, no child spawned, and the capacity reservation is released (reuse `brain/capacity_test.go` reconciliation expectation; verify `runtime.snapshot().Capacity` counters return to baseline, mirroring `TestLifecycleCapacityRejectedAdmissionReleasesReservation`). |
| cancellation | **reuse existing** `daemon_test.go::TestExecuteAndDrain_ContextCancelled_ReportsCancelled` and `brain/g2a_test.go::TestCoordinatorPreservesCancellationAndTerminalResult` — CLI-agnostic; **no new cancel test** unless the Cline branch changes the cancel path (it does not). Cite as equivalent evidence. |
| terminal result | **reuse existing** fail-closed mapping (`reportTaskResult`) + `client_test.go::TestDefaultTerminalRetrySchedule_MatchesAgreedPlan` + `service/task_complete_race_test.go`. No new terminal test; the Cline family produces the same `TaskResult` shape. |
| usage | **reuse existing** `daemon_test.go::TestMergeUsage` + `handleTask` `ReportTaskUsage` path. No new usage test; usage collection is provider-agnostic. |
| slot cleanup | `recordTerminal` `capacity.Finish` path already covered by `brain/capacity_test.go::TestLedgerCountersAcceptsPreStartFailureAndCancellation` — cite as equivalent. |

## 3. Runner commands (A9 executes; package-scoped, not broad)

```bash
GO=/home/ec2-user/goroot/go/bin/go
cd multica-auth-work/server
# L3c-A/B/D (runtimeenv writer/adapter/assert) — after A1/R1 impl:
$GO test ./internal/daemon/runtimeenv/ -run 'Cline|ClineConfig|AdapterAcceptsOpenAICompatible|AssertPreLaunch.*Cline' -count=1
# L3c-C/E + L7b + 8.2 capacity (daemon pkg) — after W1 impl:
$GO test ./internal/daemon/ -run 'BuildLaunchOpenAICompatible|AgentBrainSyntheticChildCline|AdmitTaskFailsClosedForUnacceptedClineAdapter|AdmitTaskRejectsNVIDIAOwnedClineRoute|GCReclaimsControlledAgentBrainHome|BuildLaunchClineReleasesCapacity' -count=1
# cited equivalent evidence (already green; do NOT re-run as new QA):
#   ./internal/daemon/ -run 'ExecuteAndDrain_ContextCancelled|MergeUsage|ShouldCleanTaskDir'
#   ./internal/daemon/brain/ -run 'Coordinator|Capacity'
```

`gofmt -l` on any touched `_test.go` must return empty. `-count=1` disables cache; targeted `-run`
keeps scope to the delta — no `./...`, no broad regression, no `-race` sweep required for this contract.

## 4. Ownership & sequencing (disjoint; W1 serial for the hotspot)

1. A1/R1 land 2.1 (adapter), 2.2 (writer), 2.4 (AssertPreLaunch) in `runtimeenv`/`execenv` — plus
   the exact GLM/Kimi RouteModel from **A3** (`A3-route-freeze.md`). Blocked-external until A3 freezes IDs.
2. W1 lands 2.3, 2.5, 2.6, 2.7-capacity in the `daemon` package hotspot (`brain_integration.go` +
   `brain_integration_test.go`, `gc_test.go`), consuming the A1/R1 symbols. Serial, single editor.
3. A9 runs the §3 targeted commands once per delta; records delta→command→result. One live Kanban→
   terminal acceptance for the Cline family (W1) closes the wire-path side of 5.6/5.7/8.1/8.2 — the
   unit contract here is its fail-closed safety net, not a duplicate.

## 5. Blockers / limitations

- **BLOCKED_EXTERNAL:** exact GLM (5.6) / Kimi (5.7) `RouteModel` IDs owned by A3; tests are
  parameterized over `clineAgentBrainRoutes` so they attach without edits once frozen.
- R3 wrote **no** product code and **no** product tests, and ran no Go build/tests (design lane, lock
  only on this doc). Symbol names (`WriteCredentiallessClineConfig`, adapter/env fields) are proposals
  the owners finalize; the assertions/semantics are the binding part.
- No auth/credential/quota/retry/failover, NVIDIA-fallback, Prodex, OBS, capacity-tier or live-run
  tests are specified (out of scope by contract).

## 6. Summary

Converts the one real Main-Brain lifecycle gap (L3c) and the minor cleanup gap (L7b) into a bounded,
named focused-test contract: adapter acceptance, credentialless `providers.json` materialization,
`buildLaunch` env/secret wiring + isolation child, `AssertPreLaunch` Cline layout, a fail-closed
admission regression guard (incl. NVIDIA exclusion), controlled-home GC reclamation, and capacity
release on pre-launch failure — with cancel/terminal/usage explicitly **reused as equivalent
evidence** rather than re-tested. Package-scoped runner commands, W1-serial ownership, zero broad
regression, zero live run.
