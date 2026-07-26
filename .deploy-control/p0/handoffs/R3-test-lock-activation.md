# A9-R3 — R3 lifecycle test-lock activation (BLOCKED on W1 D5/D6/D3/D4)

- agent: Opus48#D · lane: A9-R3 · task: P0-R3-TEST-LOCK-ACTIVATION
- pane: `w8:p2` (HERDR_ENV=1) · git HEAD: `a6d5098`
- check-in: `.deploy-control/p0/checkins/Opus48-D__P0-R3-TEST-LOCK-ACTIVATION__20260721T232329Z.json`
- supersedes: `P0-LIFECYCLE-TEST-IMPLEMENTATION` (checked out DONE) ·
  upstream contracts: `R3-lifecycle-focused-tests.md`, `W1-source-lock-freeze.md` (§3.6 D1–D7)
- status: **BLOCKED** — no product or test edits performed.

## 0. Tool / Go preflight (canonical path)

| Tool | Command | Result |
|---|---|---|
| go | `/home/ec2-user/goroot/go/bin/go version` | `go version go1.26.1 linux/amd64` |
| gofmt | `/home/ec2-user/goroot/go/bin/gofmt` | present |
| git | `git --version` | 2.50.1 (HEAD `a6d5098`) |
| python3 / rg | — | 3.9.25 / 15.2.0 |

## 1. Exact test-file locks acquired (this lane)

| Locked file | Maps to | Existence | Zero-overlap |
|---|---|---|---|
| `.deploy-control/p0/handoffs/R3-test-lock-activation.md` | control artifact | created | — |
| `multica-auth-work/server/internal/daemon/execenv/cline_home_test.go` | **D5** | exists (3846 B) | ✓ no active check-in locks it |
| `multica-auth-work/server/internal/daemon/config_test.go` | **D3/D4** | exists (34373 B) | ✓ |
| `multica-auth-work/server/internal/daemon/brain_integration_test.go` | **D6** | exists (32381 B) | ✓ |

Zero-overlap verified two ways: (1) scan of all `checkins/*.json` in `IN_PROGRESS|BLOCKED` → no lock on
these paths; (2) `p0_control.py check-in` accepted the locks (it rejects any intersection). These three
test files are **distinct from R1's Cline unit test** (`runtimeenv/cline_test.go`, R1/A1-owned) and from
W1's non-test source files.

## 2. Current source facts (verified at HEAD `a6d5098`, working tree)

**Applied (confirmed, intentional — do not touch):** D1 `runtimeenv/adapter.go` — `CLIOpenAICompatible`
now returns `AdapterReady, ProtocolOpenAIChat`; D2 `runtimeenv/env.go` — `case CLIOpenAICompatible`
injects `CLINE_DATA_DIR` (trusted-local) + `AdapterEnvironment.ClineDataDir` field. R1's `runtimeenv`
tests are synchronized and green (`env_test.go`, `model_test.go`, `adapter_test.go`). Not in this lane's scope.

**NOT applied (my locks' dependencies — the block):**
- **D5** — `WriteCredentiallessClineConfig` / `prepareCredentiallessClineHome`: **ABSENT** in
  `internal/daemon/execenv/cline_home.go` (only the legacy credentialed `prepareClineHome` exists).
- **D6** — `internal/daemon/brain_integration.go buildLaunch`: only the `plan.Task.Request.CLIKind ==
  brain.CLICodex` branch (lines 331/338, `WriteCredentiallessCodexConfig`). **No `CLIOpenAICompatible`
  branch**, no Cline config materialization, no `CLINE_OMNIROUTE_API_KEY` injection.
- **D3** — `internal/daemon/config.go agentBrainBuiltInCLIFor` (line 141): cases only `CLIClaudeCode`
  (`claude`/`claude`) and `CLICodex` (`codex`/`codex`); `default` returns an error. **No
  `CLIOpenAICompatible` (`cline`/`cline`) case.**
- **D4** — `AgentBrainIntegrationConfig.Validate` accepted-CLIKind case for `CLIOpenAICompatible`
  (same file, coupled to D3): pending with D3.

## 3. Assertion map — existing tests affected by D5/D6/D3/D4 (in the locked files)

Rule: existing legacy/claude/codex assertions **must stay green** (no rewrite); the D-deltas **add**
new focused cases. Fixtures reused, never re-invented.

### 3.1 `execenv/cline_home_test.go` (D5)
- **Existing (unchanged):** `TestPrepareClineHomePerAccountIsolatesDataDir`,
  `TestPrepareClineHomeAcceptsDataDirAsAccountHome` — cover the **legacy credentialed** path
  (`env.ClineDataDir`, `CredentialEnv("cline")` → `CLINE_DATA_DIR`/`CLINE_SANDBOX`/`CLINE_SANDBOX_DATA_DIR`).
  D5 adds a **separate credentialless** writer; these must remain green (regression guard).
- **To add on signal:** `TestWriteCredentiallessClineConfigMaterializesProvidersJSON` — writer places
  the carrier at the installed relpath, dir `0700` / file `0600`, bytes round-trip
  `runtimeenv.ValidateClineConfigBytes` (import boundary: keep in `execenv` using the raw
  contract bytes, or place in `runtimeenv` if the writer lands there — W1/D5 owner decides placement per
  the freeze §3.6 reconciliation note), sentinel present + **no secret value on disk**, symlink/second-write guards.
- **Reuse helpers:** `writeTestCredential`, `assertFileContent`, `testLogger`.

### 3.2 `config_test.go` (D3/D4)
- **Existing (unchanged):** `TestLoadConfig_ProbesClineAndNIMCredential` (line 246) and the other
  `TestLoadConfig_*` — credential-probe/auto-update behavior; not the gateway CLIKind mapping. Stay green.
- **To add on signal:** `TestAgentBrainBuiltInCLIForOpenAICompatible` — asserts `agentBrainBuiltInCLIFor(
  brain.CLIOpenAICompatible)` returns `{Provider:"cline", Command:"cline"}` and no error (D3); and
  `TestAgentBrainIntegrationConfigValidateAcceptsOpenAICompatible` — asserts `AgentBrainIntegrationConfig
  .Validate` accepts `CLIKind=CLIOpenAICompatible` (D4). Any existing Validate test that enumerates the
  accepted set gets a **row added**, not a semantic change (delta owner confirms the exact existing name at impl).

### 3.3 `brain_integration_test.go` (D6)
- **Existing (unchanged):** `TestAgentBrainDevelopmentIsolationSmoke`, `TestAgentBrainSyntheticChild`,
  `TestAgentBrainRejectsDualRouterBeforeGatewayAccess`, `TestAgentBrainFailsClosedWhenGatewayNotReady`,
  `TestAgentBrainUsesInstalledOmniRouteHealthContract` — all `CLIClaudeCode`; stay green.
- **To add on signal:** `TestBuildLaunchOpenAICompatibleWiresClineDataDirAndSecret` +
  `TestAgentBrainSyntheticChildCline` (re-exec isolation child): child env has `CLINE_DATA_DIR` under the
  execution root, `providers.json` present + sentinel-only, `CLINE_OMNIROUTE_API_KEY` set; forbidden
  keys absent (`OPENAI_API_KEY`,`NVIDIA_API_KEY`,`CODEX_HOME`,`ANTHROPIC_AUTH_TOKEN`,`MULTICA_L2_ENABLED`,
  `MULTICA_PRODEX_ENABLED`); distinct exit codes. `TestBuildLaunchClineReleasesCapacityOnPreLaunchFailure`
  (capacity reconciled on a Cline pre-launch failure).
- **Contract update vs R3 §2.5:** because **D1 is applied** (adapter now `AdapterReady`), the earlier
  `TestAdmitTaskFailsClosedForUnacceptedClineAdapter` is **obsolete** — replace with a **positive** admit
  path for `CLIOpenAICompatible` and keep only the **NVIDIA-exclusion** negative
  (`TestAdmitTaskRejectsNVIDIAOwnedClineRoute`).
- **Reuse helpers:** `newSyntheticGateway`, `syntheticAgentBrainConfig` (with `CLIKind=CLIOpenAICompatible`
  + a `clineAgentBrainRoutes` RouteModel), `syntheticGatewayTask`, `syntheticCredentialSource`,
  `launch.Environment.Keys()`/`.Exec()`, `containsString`.

## 4. Unblock plan (on W1 source signal)

1. Consume W1's activation manifest confirming D5+D6+D3+D4 applied (+ `implementation_authorized=true`).
2. Under the already-held locks, implement **only** the §3 "to add" tests (parameterized over
   `clineAgentBrainRoutes` so GLM `cp/cline-pass/glm-5.2` proceeds; **Kimi stays HOLD on BLK-KIMI**).
3. Run targeted (no `./...`, no broad regression, no live run):
   ```bash
   GO=/home/ec2-user/goroot/go/bin/go ; cd multica-auth-work/server
   $GO test ./internal/daemon/execenv/ -run 'WriteCredentiallessCline|PrepareClineHome' -count=1
   $GO test ./internal/daemon/ -run 'AgentBrainBuiltInCLIForOpenAICompatible|IntegrationConfigValidateAcceptsOpenAICompatible|BuildLaunchOpenAICompatible|AgentBrainSyntheticChildCline|BuildLaunchClineReleasesCapacity|AdmitTaskRejectsNVIDIAOwnedClineRoute' -count=1
   ```
4. `gofmt -l` on touched test files → empty; record delta→command→result; check out with evidence.

## 5. Blocker record

- blocker: `W1 D5/D6/D3/D4 source not applied`
- owner: `W1` (Opus48#B)
- next action: `on W1 source signal, implement only these locked focused tests`
- verified facts: D5 writer absent (`cline_home.go`); no `CLIOpenAICompatible` branch in `buildLaunch`
  (`brain_integration.go`); `agentBrainBuiltInCLIFor` lacks the `CLIOpenAICompatible` case (`config.go:141`);
  D1/D2 intentionally applied (out of this lane's scope).
- also gating (external): BLK-KIMI (Kimi RouteModel) — GLM cases unaffected.

Registered via `p0_control.py block`. No product or test edits while BLOCKED.
