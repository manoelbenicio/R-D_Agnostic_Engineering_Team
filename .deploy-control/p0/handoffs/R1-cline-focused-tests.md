# R1 — Cline GLM/Kimi focused test contract (TEST DESIGN, READ-ONLY)

- agent: `Opus48#A`  ·  lane: `R1-TEST-DESIGN`  ·  task: `P0-CLINE-TEST-CONTRACT`  ·  pane: `w6:p1`
- output lock (only mutable file): `.deploy-control/p0/handoffs/R1-cline-focused-tests.md`
- consumes: `.deploy-control/p0/handoffs/A1-A2-cline-foundation.md` (gap matrix C1–C10) +
  `.deploy-control/p0/handoffs/A3-route-freeze.md` (RouteModel freeze / blockers)
- covers OpenSpec: **5.6**, **5.7**, **8.1**, **8.2** (Main-Brain-owned Cline slice only)
- product source: **READ-ONLY**. No source edited, no test executed, no live run in this phase.
- created: 2026-07-21T22:41Z (UTC)
- toolchain for the OWNING lanes to run these (not run here): `/home/ec2-user/goroot/go/bin/{go,gofmt}` (go1.26.1)

> This is a **test design contract**: exact test names, inputs, and pass/fail expectations that the
> delta owners (A1 for `runtimeenv/cline*.go`; W1 for hotspots) implement and A9 runs **once** each.
> It invents **no** RouteModel. GLM uses the A3-FROZEN id `cp/cline-pass/glm-5.2`; Kimi is
> `BLOCKED_EXTERNAL` (A3 BLK-KIMI) and its exact-model assertions are HELD behind one pending constant.

---

## 1. Inputs consumed (facts, not assumptions)

From **A3 route freeze**:
- **GLM 5.6 — FROZEN:** `RouteModel = cp/cline-pass/glm-5.2`, protocol **OpenAI Chat Completions**
  (`/v1/chat/completions`), Brain-selectable = YES. `runtimeenv/cline_test.go` GLM fixtures already
  canonical. One W1 hotspot substitution SUB-1 (`server/pkg/agent/models.go:562`
  `cline-pass/glm-5.2` → `cp/cline-pass/glm-5.2`) + its lockstep expectation `models_test.go:172`.
- **NVIDIA fallback — FROZEN, NOT Brain-selectable:** `nvidia/z-ai/glm-5.2` (OmniRoute-owned). Must
  stay fail-closed in `cline.go` (`isNVIDIAOwnedRoute`).
- **Kimi 5.7 — BLOCKED_EXTERNAL (BLK-KIMI):** no OmniRoute-declared exact versioned model id. Three
  divergent Multica-side literals exist (`cline-kimi-k2.7-dedicated`, `cline-pass/kimi-k2.7-code`,
  `claude_code_kimi_2.7_Code`); **none may be chosen by plausibility**. `claude_code_kimi_2.7_Code`
  is the **Claude Code** family (Anthropic Messages) and must **not** be used for Cline→Kimi.
- **Availability 8.1 — BLOCKED_EXTERNAL (BLK-AVAIL):** enriched OmniRoute `/v1/models` rows for the
  Cline routes not published; GLM exact-id freeze is nonetheless complete and independent. Live
  acceptance additionally gated by the D-V3-25(B) security stop.

From **A1-A2 foundation** (gap matrix): C1 shared `cline.go` contract already EQUIV (zero delta);
wiring gaps C3 (`config.go` exe map), C4 (`config.go` validate), C5 (`adapter.go` gate flip),
C6 (`env.go` trusted inject), C7 (`execenv` writer + `brain_integration.go` `buildLaunch` branch);
recommended CLIKind = **`brain.CLIOpenAICompatible`** (executable `cline`).

---

## 2. Test taxonomy & what each 5.x/8.x requirement maps to (dedup)

| Requirement | Offline focused-test coverage | Live-run coverage (single reserved run per family) |
|---|---|---|
| **5.6** Cline→GLM path implemented | T-GLM-1..7, T-ENV-*, T-CFG-*, T-EXEC-*, T-LAUNCH-* (GLM id) | one Kanban→terminal GLM run (after BLK-AVAIL clears) |
| **5.7** Cline→Kimi path implemented | structural tests reuse GLM; Kimi exact-id tests **HELD** (BLK-KIMI) | HELD until Kimi id + availability published |
| **8.1** model/protocol/availability on changed routes | T-ADPT-1 (protocol), T-MODEL-1 (policy accept), T-CAP-1 (capability via injected snapshot) | availability proven by the same GLM live run |
| **8.2** tools/reasoning/cancellation/usage/terminal/deterministic errors | deterministic-error half = all fail-closed tests (§5); cancellation/cleanup = T-LAUNCH-3 (unit) | tools/reasoning/usage/terminal-result observed in the same one GLM live run |

**Dedup rule (from `P0_MAIN_BRAIN_EXECUTION.md`):** 8.1 and 8.2 are **gates over the same executions**,
not separate campaigns. The offline unit tests below prove protocol/policy/fail-closed determinism;
the **single** GLM live run proves availability + tools/reasoning/usage/terminal. No QA-A/B/C, no
broad regression, no second live acceptance. Kimi adds **zero** new live runs until BLK-KIMI clears.

---

## 3. Existing tests to REUSE as-is (no rerun beyond the one focused pass)

`internal/daemon/runtimeenv/cline_test.go` (all currently pass; ID-agnostic contract):
- `TestNewClineConfigContractKimiAndGLMRoutes` — iterates `clineAgentBrainRoutes`.
- `TestNewClineConfigContractMatchesInstalledCarrierSchema` — GLM `cp/cline-pass/glm-5.2`.
- `TestNewClineConfigContractEmbedsNoSecretValue` — no-secret invariant.
- `TestNewClineConfigContractRejectsNVIDIAFallbackRoute` — NVIDIA fail-closed.
- `TestNewClineConfigContractRejectsInvalidInputs` — empty inputs.
- `TestValidateClineConfigBytesDetectsTamper` — tamper/extra-key/model-mismatch/nvidia.

**Required test-fixture correction (A1-owned, `cline_test.go`, not a contract change):** the Kimi
fixture `cline-kimi-k2.7-dedicated` (lines 19, 79) is an A3-flagged **combo alias, not canonical**.
To avoid implying it is the exact model, replace both occurrences with a single clearly-named pending
constant, e.g.:
```
// clineKimiRouteModelPENDING is a structural placeholder ONLY. A3 BLK-KIMI:
// the exact OmniRoute Kimi RouteModel is UNDECLARED and must be substituted here
// (one edit) once published. It is NOT asserted to be the canonical Kimi id.
const clineKimiRouteModelPENDING = brain.RouteModel("cline-kimi-k2.7-dedicated")
```
Structural coverage (`EmbedsNoSecretValue`, the route-iteration test) then references
`clineKimiRouteModelPENDING`, and a `// BLK-KIMI` comment documents that the *value* is provisional.
This keeps 5.7 structural coverage green while making the block explicit and one-line-unblockable.

---

## 4. NEW focused tests (grouped by delta; named, with inputs & expectation)

> Naming mirrors existing conventions (`TestBuildGatewayEnvironmentCodexUsesDedicatedKeyName`,
> `TestAssertPreLaunchAcceptsControlledCodexPlan`, `TestG4ClaudeAndCodex...`). Each new test is the
> **minimal** proof of one delta; A9 runs each once.

### 4.1 Adapter gate flip — C5 (`runtimeenv/adapter_test.go`, NEW file, package `runtimeenv`)
- **T-ADPT-1 `TestOpenAICompatibleAdapterReadyForOmniRouteChat`**
  - input: `CredentiallessAdapterContract(brain.CLIOpenAICompatible)`
  - expect: `err == nil`; `State == AdapterReady`; `Protocol == brain.ProtocolOpenAIChat`; empty `Gate`.
  - covers: 8.1 protocol.
- **T-ADPT-2 `TestNativeKimiNIMAntigravityStayFailClosedAfterClineAccept`** (regression guard)
  - input: `CLIKimi`, `CLINIM`, `CLIAntigravity`
  - expect: each returns `*AdapterGateError` (`GateNativeKimiUnaccepted` / `...NIM...` / `...Antigravity...`),
    `State == AdapterFailClosed`. Proves flipping OpenAI-compatible did **not** open native Kimi.
  - covers: 5.7 boundary (Cline→Kimi ≠ CLIKimi), 8.2 deterministic error.

### 4.2 Trusted env injection — C6 (`runtimeenv/env_test.go`, add to existing file)
- **T-ENV-1 `TestBuildGatewayEnvironmentClineUsesDedicatedKeyName`** (parallels the Codex test)
  - input: `BuildGatewayEnvironment(ComposeOptions{Adapter: AdapterEnvironment{CLI: CLIOpenAICompatible,
    GatewayRoot: "http://127.0.0.1:20128", TaskHome: <controlled dir>, StableSecret: <NewStableSecret("x")>}}）`
    plus a controlled Cline data dir.
  - expect: child env contains `CLINE_OMNIROUTE_API_KEY=<secret>` and `CLINE_DATA_DIR=<dir>` as
    **trusted-last** (origin trusted); `secretKey == "CLINE_OMNIROUTE_API_KEY"`; base root normalized.
  - covers: 5.6 launch env; 8.2 usage plumbing.
- **T-ENV-2 `TestBuildGatewayEnvironmentClineSecretRedactedInDiagnostics`**
  - expect: `ChildEnvironment.String()`/`Keys()` never contain the secret value; only `Exec()` carries it.
  - covers: EVIDENCE_CONTRACT rule 5 (no secret in logs).
- **T-ENV-3 `TestClineTrustedKeysRejectedFromCustomOrLocal`** (deny-list collision, A1-A2 §7)
  - input: `ValidateCustomEnvironment({"CLINE_DATA_DIR": "..."})` and `{"CLINE_OMNIROUTE_API_KEY": "..."}`
  - expect: `*EnvironmentPolicyError` with `DenyCredentialRoot` (CLINE_DATA_DIR) and
    `DenyProviderCredential` (CLINE_*+API_KEY). Proves the keys are legal **only** via trusted path.
  - covers: 8.2 deterministic error; security invariant.
- **T-ENV-4 `TestBuildGatewayEnvironmentRejectsClineWhenAdapterFailClosed`** (ordering guard)
  - precondition variant: if C5 not applied, `BuildGatewayEnvironment(CLIOpenAICompatible)` returns
    `ErrAdapterFailClosed` (documents the C5→C6 dependency). Kept as a guard, not a permanent assert.

### 4.3 Providers.json materialization writer — C7 (`execenv/cline_home_test.go`, add; package `execenv`)
> Existing `TestPrepareClineHomePerAccountIsolatesDataDir` / `...AcceptsDataDirAsAccountHome` cover the
> **legacy per-account** path — leave untouched. New gateway writer is separate.
- **T-EXEC-1 `TestWriteCredentiallessClineConfigWritesCarrier`**
  - input: `WriteCredentiallessClineConfig(<clineDataDir>, contract.Bytes())` where contract =
    `NewClineConfigContract("http://127.0.0.1:20128", "cp/cline-pass/glm-5.2", <ts>)`.
  - expect: file at the FROZEN subpath (A1 to freeze; recommended `<CLINE_DATA_DIR>/settings/providers.json`),
    mode `0600`, parent dir `0700`; bytes byte-equal to `contract.Bytes()`; re-validates via
    `runtimeenv.ValidateClineConfigBytes`.
  - covers: 5.6 materialization.
- **T-EXEC-2 `TestWriteCredentiallessClineConfigRejectsEmptyAndOversized`**
  - input: `len==0` and `len>64<<10`
  - expect: deterministic non-nil error, no file written.
  - covers: 8.2 deterministic error.
- **T-EXEC-3 `TestWriteCredentiallessClineConfigRejectsNonAbsoluteOrRootDataDir`**
  - input: relative path, filesystem root
  - expect: deterministic error (mirrors `WriteCredentiallessCodexConfig` guard).

### 4.4 Task-home manifest exclusion — C7/§6.4 (`runtimeenv/assert_test.go`, add)
> Existing `TestAssertPreLaunchAcceptsControlledCodexPlan` / `...ClaudePlan` are the parallels.
- **T-ASSERT-1 `TestAssertPreLaunchAcceptsControlledClinePlan`**
  - input: `LaunchPlan{Environment: <cline child env>, TaskHome: <manifest WITHOUT providers.json/.cline>,
    ExecutionRoot: <root>}`
  - expect: `nil` (accepts a Cline plan whose carrier lives in `CLINE_DATA_DIR`, not task home).
- **T-ASSERT-2 `TestAssertPreLaunchRejectsClineCarrierInsideTaskHome`**
  - input: manifest that lists `providers.json` or `.cline/…`
  - expect: `ErrTaskHomeCredential` (confirms `home.go ValidateTaskHomeManifest` still guards leakage).
  - covers: security invariant + 8.2 deterministic error.

### 4.5 Config wiring — C3/C4 (`daemon/config_test.go` + `brain_integration_test.go`)
- **T-CFG-1 `TestAgentBrainBuiltInResolvesClineExecutable`** (config_test.go)
  - input: `agentBrainBuiltInCLIFor(brain.CLIOpenAICompatible)`
  - expect: `{Provider: "cline", Command: "cline"}, nil`.
  - covers: 5.6 executable mapping (C3).
- **T-CFG-2 `TestLoadConfigGatewayAcceptsOpenAICompatibleCLIKind`** (config_test.go)
  - input: `AgentBrainIntegrationConfig{DevelopmentEnabled:true, Neutral.Gateway.Required:true,
    CLIKind: CLIOpenAICompatible, RouteModel: "cp/cline-pass/glm-5.2", CapacityTier:20,...}.Validate()`
  - expect: `nil` (C4). Negative twin: an unlisted CLIKind (e.g. `CLIAntigravity`) still returns the
    "supports only … Claude Code, Codex, or the accepted OpenAI-compatible (Cline)" error.
  - covers: 5.6 config admission; 8.2 deterministic error.
- **T-CFG-3 `TestAgentBrainBuiltInClineResolutionIgnoresCommandPathOverride`** (brain_integration_test.go,
  parallels existing `TestAgentBrainBuiltInResolutionIgnoresCommandPathOverride`)
  - expect: gateway mode ignores `MULTICA_CLINE_PATH`/profile overrides (credentialless posture).

### 4.6 Launch write path — C7 (`daemon/brain_integration_test.go`)
- **T-LAUNCH-1 `TestAgentBrainClineLaunchWritesProvidersJson`**
  - input: enabled gateway runtime with `CLIKind=CLIOpenAICompatible`, `RouteModel=cp/cline-pass/glm-5.2`,
    injected `CredentialSource`/registry snapshot (offline).
  - expect: `buildLaunch` writes a valid `providers.json` under `CLINE_DATA_DIR`; child env carries
    the trusted Cline keys; manifest passed to `AssertPreLaunch` excludes the carrier.
  - covers: 5.6 end-to-end offline; 8.1 protocol/model.
- **T-LAUNCH-2 `TestAgentBrainClineAdmitUsesInjectedRegistryCapability`** (8.1 availability, offline)
  - input: fake `ModelsDocument`/registry snapshot marking `cp/cline-pass/glm-5.2`
    `available=true, protocol=openai-chat, tools=true, streaming=true`.
  - expect: `admitTask` passes `ValidateCapability`/`LookupModelCapability`. Documents that the ONLY
    thing BLK-AVAIL blocks is the *real* enriched row; the code path is provable offline.
  - covers: 8.1 (offline half).
- **T-LAUNCH-3 `TestAgentBrainClineCancellationAndCleanupEmitOnce`** (parallels
  `TestAgentBrainConcurrentDuplicateStartAndCancellationFinishEmitOnce`)
  - expect: cancel → single terminal/cleanup emission; capacity lease released.
  - covers: 8.2 cancellation/cleanup (unit; the reserved live run confirms end-to-end).

### 4.7 Model policy — C9 (`runtimeenv/model_test.go`, add; parallels
`TestGatewayModelPolicyAcceptsApprovedOmniRouteIDWithoutNativeDiscovery`)
- **T-MODEL-1 `TestGatewayModelPolicyAcceptsClineGLMOverChat`**
  - input: `NewGatewayModelPolicy([{Model:"cp/cline-pass/glm-5.2", Protocol:ProtocolOpenAIChat,
    CLIs:[CLIOpenAICompatible]}])` then `ValidateSelection(CLIOpenAICompatible,"cp/cline-pass/glm-5.2","")`.
  - expect: `nil`. Negative: thinking!="" → `ErrThinkingNotApproved`; wrong CLI → `ErrCLIModelNotApproved`;
    unknown model → `ErrModelNotApproved`.
  - covers: 8.1 protocol/model; 8.2 deterministic error.

---

## 5. Fail-closed / deterministic-error case matrix (8.2 deterministic half)

| Case | Symbol under test | Input | Expected error | Test |
|---|---|---|---|---|
| NVIDIA route not Brain-selectable | `NewClineConfigContract` / `ValidateClineConfigBytes` | `nvidia/z-ai/glm-5.2`, `NVIDIA/…`, `nvidia/other` | `ErrClineRouteNotAgentBrainSelectable` | existing `...RejectsNVIDIAFallbackRoute` / `...DetectsTamper` |
| empty inputs | `NewClineConfigContract` | empty model / root / updatedAt | non-nil (`ErrClineConfigContract` / parse err) | existing `...RejectsInvalidInputs` |
| embedded secret value | `ValidateClineConfigBytes` | apiKey = concrete `sk-…` | `ErrClineConfigContract` | existing `...DetectsTamper` |
| extra top-level key / model mismatch | `ValidateClineConfigBytes` | extra key / wrong model | `ErrClineConfigContract` | existing `...DetectsTamper` |
| non-`/v1` base path | `ValidateClineConfigBytes` | baseURL path ≠ `/v1` | `ErrClineConfigContract` | (add assertion to existing tamper test) |
| oversized / empty config | `WriteCredentiallessClineConfig` | 0 / >64KiB | non-nil, no write | T-EXEC-2 |
| native Kimi/NIM/Agy adapters | `CredentiallessAdapterContract` | `CLIKimi`/`CLINIM`/`CLIAntigravity` | `*AdapterGateError` (fail-closed) | T-ADPT-2 |
| trusted keys via custom/local | `ValidateCustomEnvironment` | `CLINE_DATA_DIR` / `CLINE_OMNIROUTE_API_KEY` | `*EnvironmentPolicyError` | T-ENV-3 |
| unlisted CLIKind in gateway mode | `AgentBrainIntegrationConfig.Validate` | e.g. `CLIAntigravity` | config error | T-CFG-2 (negative) |
| cline executable absent | `resolveAgentBrainBuiltInEntry` | `cline` not on PATH | "not installed as its canonical built-in command" | (add to config_test; env-controlled) |
| carrier inside task home | `AssertPreLaunch`/`ValidateTaskHomeManifest` | `providers.json`/`.cline` in manifest | `ErrTaskHomeCredential` | T-ASSERT-2 |
| **Kimi exact model UNDECLARED** | model selection | any guessed Kimi literal | **must fail-closed / HELD, never silently default** | see §6 |

---

## 6. GLM implementable path vs Kimi external blocker

**GLM (5.6) — implementable & offline-testable now.** The entire chain
`config validate (T-CFG-2) → executable resolve (T-CFG-1) → adapter ready+chat (T-ADPT-1) → trusted
env inject (T-ENV-1..3) → NewClineConfigContract("cp/cline-pass/glm-5.2") + write (T-EXEC-1) →
model policy (T-MODEL-1) → buildLaunch write + manifest (T-LAUNCH-1) → capability via injected
snapshot (T-LAUNCH-2)` is provable with unit tests **without** the enriched registry. BLK-AVAIL only
blocks the *live* admission/run; the code path is offline-provable. The single GLM live run (after
BLK-AVAIL + D-V3-25(B) clear) then closes 8.1 availability + 8.2 tools/reasoning/usage/terminal.

**Kimi (5.7) — external blocker (A3 BLK-KIMI).** No canonical exact model id exists. Therefore:
- Structural Cline coverage for Kimi is obtained by the **shared, ID-agnostic** contract tests using
  the GLM id (the contract does not branch per model), so 5.7's *structural* Brain behavior is proven
  by the same code as 5.6.
- **HELD tests** (author now, keep `t.Skip("BLK-KIMI: exact OmniRoute Kimi RouteModel undeclared")`
  until unblock): `TestGatewayModelPolicyAcceptsClineKimiOverChat`, and a Kimi twin of T-LAUNCH-1/2.
- Single unblock action: publish exact Kimi `RouteModel` → set `clineKimiRouteModelPENDING` (§3) and
  A3 SUB-2 → remove the skips. **No route id guessed here.**
- Explicit anti-conflation assertion (guard): a test may assert that `claude_code_kimi_2.7_Code` is
  **rejected** as a Cline (`ProtocolOpenAIChat`) selection (it is Anthropic Messages / Claude family),
  proving the two families are not interchangeable — `TestClineRejectsClaudeKimiComboAsChatModel`.

---

## 7. Ownership & run plan (who authors, who runs — no execution here)

| Test group | File | Author (delta owner) | Runner |
|---|---|---|---|
| Fixture rename + reuse | `runtimeenv/cline_test.go` | **A1** | A9 (one pass) |
| T-ADPT-* | `runtimeenv/adapter_test.go` (new) | A1/R1 (with C5) | A9 |
| T-ENV-* | `runtimeenv/env_test.go` | **W1** (env.go is shared) | A9 |
| T-EXEC-* | `execenv/cline_home_test.go` | **W1** (execenv hotspot) | A9 |
| T-ASSERT-* | `runtimeenv/assert_test.go` | A1/R1 | A9 |
| T-CFG-*, T-LAUNCH-* | `daemon/config_test.go`, `daemon/brain_integration_test.go` | **W1** (hotspots) | A9 |
| T-MODEL-1, Kimi-HELD | `runtimeenv/model_test.go` | A1/R1 | A9 |

**Run command (owner/A9 only; NOT run in this phase):**
```
cd multica-auth-work/server
/home/ec2-user/goroot/go/bin/go test ./internal/daemon/runtimeenv/... ./internal/daemon/execenv/... ./internal/daemon/...
/home/ec2-user/goroot/go/bin/gofmt -l internal/daemon
```
A9 runs each new/changed test **once**; equivalent green checks are not repeated; no broad regression.

---

## 8. Blockers / limitations

- **BLK-KIMI (A3, external):** exact OmniRoute Kimi `RouteModel` undeclared → 5.7 exact-model tests
  HELD (skipped), structural coverage via GLM. No id invented. Owner: OmniRoute architect via Principal.
- **BLK-AVAIL (A3, external):** enriched registry rows for Cline routes unpublished → live admission
  blocked; offline capability path proven via injected snapshot (T-LAUNCH-2). Owner: OmniRoute.
- **D-V3-25(B) security stop:** live acceptance additionally gated on key-revocation confirmation.
- This phase produced **no source edits, no test execution, no live run** — every test above is a
  design specification for the owning lanes (A1/W1) to implement and A9 to run once.

---

## 9. Agent status block

- STATUS: DONE (test-design contract)
- DELIVERED: dedup 5.6/5.7/8.1/8.2 evidence map (§2); 6 existing tests to reuse + fixture correction
  (§3); 6 new focused test groups with exact names/inputs/expectations (§4); fail-closed matrix (§5);
  GLM implementable path vs Kimi HELD blocker (§6); ownership/run plan (§7); blockers (§8).
- FILES: `.deploy-control/p0/handoffs/R1-cline-focused-tests.md` (only mutable file).
- VALIDATION: none run this phase (design only); owner/A9 run command in §7.
- EVIDENCE: this artifact; grounded on A1-A2 + A3 handoffs and exact existing test names verified via rg.
- BLOCKERS/LIMITATIONS: §8 (BLK-KIMI, BLK-AVAIL, D-V3-25(B)).
- W1_HANDOFF: §7 groups marked W1 (env/execenv/config/brain_integration); A1/R1 groups for runtimeenv.
