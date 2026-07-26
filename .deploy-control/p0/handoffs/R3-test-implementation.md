# A9-R3 — R3 lifecycle focused tests: implementation (DONE)

- agent: Opus48#D · lane: A9-R3 · task: P0-R3-TEST-IMPLEMENTATION
- pane: `w8:p2` (HERDR_ENV=1) · git HEAD: `a6d5098` (+ working-tree D1–D6 by W1)
- check-in: `.deploy-control/p0/checkins/Opus48-D__P0-R3-TEST-IMPLEMENTATION__20260721T234803Z.json`
- authorization: W1 D3–D6 signal (source build-clean; runtimeenv Assert/Cline green); `implementation_authorized=true`.
- locks (edited, only these product files): `execenv/cline_home_test.go`, `config_test.go`, `brain_integration_test.go`.
- posture: **test-only additions**; no product source changed; existing Claude/Codex behavior preserved; **Kimi held** (only GLM `cp/cline-pass/glm-5.2` used; NVIDIA reject exercised).

## 0. Preflight (canonical path)

`/home/ec2-user/goroot/go/bin/go version` → `go version go1.26.1 linux/amd64`; gofmt same dir; git 2.50.1; python3 3.9.25; rg 15.2.0. Disk ~99% — module downloads/build succeeded; `go clean -cache` not required.

## 1. Source under test (W1-applied, verified before writing)

- **D1** `runtimeenv/adapter.go` — `CLIOpenAICompatible` → `AdapterReady, ProtocolOpenAIChat`.
- **D2** `runtimeenv/env.go` — `case CLIOpenAICompatible` injects `CLINE_DATA_DIR` (trusted-local) + `CLINE_OMNIROUTE_API_KEY` (trusted-secret); `AdapterEnvironment.ClineDataDir`.
- **D3** `config.go agentBrainBuiltInCLIFor` — `CLIOpenAICompatible` → `{Provider:"cline", Command:"cline"}`.
- **D4** `config.go AgentBrainIntegrationConfig.Validate` — accepts `CLIClaudeCode, CLICodex, CLIOpenAICompatible`.
- **D5** `execenv/cline_home.go` — `WriteCredentiallessClineConfig(clineDataDir, raw)` writes `<clineDataDir>/settings/providers.json` (dir 0700, file 0600, verbatim; rejects empty/relative/root dir + empty/oversize raw); `prepareCredentiallessClineHome`.
- **D6** `brain_integration.go buildLaunch` — `CLIOpenAICompatible` branch: controlled `<RootDir>/cline-data`, `NewClineConfigContract` + `WriteCredentiallessClineConfig`, `ClineDataDir` wired into `BuildGatewayEnvironment`.

## 2. Tests implemented (additive; reuse existing fixtures)

### `execenv/cline_home_test.go` (D5)
- `TestWriteCredentiallessClineConfigMaterializesCarrier` — verbatim write at `settings/providers.json`, modes 0600/0700/0700, idempotent replace. Schema-agnostic (synthetic non-secret bytes) — matches the writer's caller-validates contract; no `runtimeenv` import (avoids any cycle).
- `TestWriteCredentiallessClineConfigRejectsInvalidInputs` — empty/relative/root dir, empty/oversize (>64KiB) carrier rejected; no carrier left behind.
- Legacy `TestPrepareClineHome*` (credentialed path) left unchanged — regression guard.

### `config_test.go` (D3/D4)  [+`brain` import added]
- `TestAgentBrainBuiltInCLIForAcceptsOpenAICompatibleAndPreservesClaudeCodex` — cline→`{cline,cline}`; claude/codex preserved; unmapped kind (`CLINIM`) fails closed.
- `TestAgentBrainIntegrationConfigValidateAcceptsOpenAICompatibleCline` — Validate accepts Cline; claude/codex still validate; `CLINIM` still fails closed.

### `brain_integration_test.go` (D6)
- `TestBuildLaunchOpenAICompatibleWiresClineDataDirAndSecret` — child env has `CLINE_DATA_DIR`/`CLINE_OMNIROUTE_API_KEY`/`MULTICA_*`; forbidden keys absent (`OPENAI_API_KEY`,`OPENAI_BASE_URL`,`NVIDIA_API_KEY`,`CODEX_HOME`,`ANTHROPIC_AUTH_TOKEN`,`ANTHROPIC_BASE_URL`); `CLINE_DATA_DIR` pinned under the execution root; on-disk `providers.json` carries the reference sentinel and **not** the resolved secret value; re-exec isolation child confirms the actual child process env.
- `TestAgentBrainSyntheticChildCline` — isolation child (exit-code gated) asserting controlled Cline env in the spawned process.
- `TestBuildLaunchOpenAICompatibleRejectsNVIDIAOwnedRoute` — launch-stage fail-closed on an OmniRoute-owned NVIDIA route; no carrier written.
- helpers added: `syntheticClineAgentBrainRuntime`, `syntheticClinePlan` (reuse `syntheticAgentBrainConfig`/`syntheticCredentialSource`/`containsString`).
- Existing Claude/Codex isolation/admission tests unchanged.

## 3. Validation (targeted, `-count=1`; no broad regression, no live run)

```
GO=/home/ec2-user/goroot/go/bin/go ; cd multica-auth-work/server
$GO test ./internal/daemon/execenv/ -run 'WriteCredentiallessCline|PrepareClineHome' -count=1
  → ok  github.com/multica-ai/multica/server/internal/daemon/execenv  0.005s
$GO test ./internal/daemon/ -run 'AgentBrainBuiltInCLIForAcceptsOpenAICompatible|AgentBrainIntegrationConfigValidateAcceptsOpenAICompatibleCline|BuildLaunchOpenAICompatible|AgentBrainSyntheticChildCline' -count=1 -v
  → PASS: TestBuildLaunchOpenAICompatibleWiresClineDataDirAndSecret
  → PASS: TestAgentBrainSyntheticChildCline
  → PASS: TestBuildLaunchOpenAICompatibleRejectsNVIDIAOwnedRoute
  → PASS: TestAgentBrainBuiltInCLIForAcceptsOpenAICompatibleAndPreservesClaudeCodex
  → PASS: TestAgentBrainIntegrationConfigValidateAcceptsOpenAICompatibleCline
  → ok  github.com/multica-ai/multica/server/internal/daemon  0.020s
gofmt -l (3 touched files) → empty
```

The daemon test package compiled in full during the run (all sibling test files), confirming the edited files introduce no compile breakage.

## 4. Scope / limitations

- Only the three locked test files were edited; no product source; no new non-test file created.
- **Kimi held** (BLK-KIMI): no Kimi route model used; GLM path + NVIDIA-exclusion only.
- Cancellation / terminal-result / usage reused as equivalent evidence (existing
  `TestAgentBrainCentralCapacityReconcilesOverloadAndCancellation`,
  `daemon_test.go::TestExecuteAndDrain_ContextCancelled_ReportsCancelled`, `TestMergeUsage`) — not re-tested.
- No broad regression, no live run, no `./...`.
