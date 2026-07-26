# A1+A2 — Cline foundation & mapping gap analysis (READ-ONLY handoff)

- agent: `Opus48#A`
- lane: `A1+A2`
- task: `P0-CLINE-FOUNDATION`
- pane: `w6:p1`
- covers OpenSpec: **5.6** (Cline → GLM-5.2) and **5.7** (Cline → Kimi-K2.7)
- output lock (only mutable file this phase): `.deploy-control/p0/handoffs/A1-A2-cline-foundation.md`
- product source: **READ-ONLY** in this phase. No product code changed.
- created: 2026-07-21T22:31Z (UTC)

> Boundary reminder honored: OmniRoute owns all auth/credential/quota/retry/failover/account
> selection. This handoff touches only CLI/model **intent**, config materialization and mapping.
> No route IDs are invented — canonical `RouteModel` values are owned by **A3** (`A3-route-freeze.md`)
> and are cited here only as `BLOCKED_EXTERNAL` placeholders.

---

## 1. Tool preflight (exact, this pane)

| Tool | Resolved path | Version | Result |
|---|---|---|---|
| git | `/usr/bin/git` | `git version 2.50.1` | ✓ |
| python3 | `/usr/bin/python3` | `Python 3.9.25` | ✓ |
| rg | `/usr/local/bin/rg` | `ripgrep 15.2.0` | ✓ |
| herdr | `/home/ec2-user/.local/bin/herdr` | `herdr 0.7.4` | ✓ |
| go1.26.1 | `/home/ec2-user/goroot/go/bin/go` | `go version go1.26.1 linux/amd64` (GOROOT=`/home/ec2-user/goroot/go`) | ✓ |
| gofmt | `/home/ec2-user/goroot/go/bin/gofmt` | ships with go1.26.1 | ✓ |

**Toolchain path note (RESOLVED — canonical path corrected).** The canonical, verified go1.26.1
toolchain is **`/home/ec2-user/goroot/go/bin/{go,gofmt}`** (`go version go1.26.1 linux/amd64`,
GOROOT=`/home/ec2-user/goroot/go`) and is used for all Go verification. Per Principal steering,
`ASSIGNMENTS.md`/`control.json` were corrected to this path; the prior
`$HOME/.local/toolchains/go1.26.1/...` reference is obsolete (this pane's `$HOME` is credential
slot `slot-115`, which has no `.local/toolchains/`). Credential-slot homes are not searched. No
action outstanding.

---

## 2. Scope inspected (read-only)

| File | Purpose |
|---|---|
| `multica-auth-work/server/internal/daemon/runtimeenv/cline.go` | Cline `providers.json` contract (A1 core) |
| `multica-auth-work/server/internal/daemon/runtimeenv/cline_test.go` | existing focused tests |
| `multica-auth-work/server/internal/daemon/runtimeenv/adapter.go` | `CredentiallessAdapterContract` (CLIKind→protocol) |
| `multica-auth-work/server/internal/daemon/runtimeenv/env.go` | `BuildGatewayEnvironment` / `trustedAdapterEntries` (secret injection) |
| `multica-auth-work/server/internal/daemon/runtimeenv/policy.go` | env deny-list `ClassifyEnvironmentKey` |
| `multica-auth-work/server/internal/daemon/runtimeenv/model.go` | `GatewayModelPolicy.ValidateSelection` |
| `multica-auth-work/server/internal/daemon/runtimeenv/home.go` | `ValidateTaskHomeManifest` |
| `multica-auth-work/server/internal/daemon/brain/identity.go` | `CLIKind` enum, `RouteModel`, `ProtocolFamily` |
| `multica-auth-work/server/internal/daemon/brain/config.go` | env names incl. `ChildEnvOmniRouteAPIKey` |
| `multica-auth-work/server/internal/daemon/gateway/profiles.go` | `TrustedRuntimeProfiles` / `LookupRuntimeProfile` |
| `multica-auth-work/server/internal/daemon/config.go` | `agentBrainBuiltInCLIFor`, `resolveAgentBrainBuiltInEntry`, `AgentBrainIntegrationConfig.Validate` |
| `multica-auth-work/server/internal/daemon/brain_integration.go` | `newAgentBrainRuntime`, `buildLaunch` |
| `multica-auth-work/server/internal/daemon/execenv/codex_home.go` | `WriteCredentiallessCodexConfig` (parallel pattern) |
| `multica-auth-work/server/internal/daemon/execenv/cline_home.go` | legacy per-account `prepareClineHome` (non-gateway) |

Classification legend (per `P0_MAIN_BRAIN_EXECUTION.md §3`): **[EQUIV]** already implemented +
equivalent; **[IMPL/NOEV]** implemented, no equivalent proof; **[GAP]** real Main-Brain gap;
**[OMNI]** OmniRoute-owned/external; **[HOLD]** Phase 3.

---

## 3. Headline result

The **shared Cline config carrier is already implemented** (`cline.go` `ClineConfigContract`) and is
already model-agnostic across the two Cline routes — it is reused by GLM-5.2 and Kimi-K2.7 with no
per-model duplication. **A1's "shared base" objective is essentially already met at the contract
layer.** The real remaining P0 work for 5.6/5.7 is **wiring**, and every wiring edit lands in
**W1-owned hotspots** — not in `runtimeenv/cline.go`. There are **four exact wiring gaps** plus one
**external blocker** (A3 route IDs) and one **behavioral verification** (Cline `${ENV}` expansion).

---

## 4. Current-vs-required matrix (5.6 / 5.7)

| # | Capability | Required for 5.6/5.7 | Current state | Class | Owner |
|---|---|---|---|---|---|
| C1 | Cline `providers.json` materialization contract (schema v1, `openai-compatible` provider, `/v1` base, sentinel apiKey, NVIDIA reject, tamper validation) | shared, one impl for GLM+Kimi | **Present & complete** in `cline.go` (`NewClineConfigContract`, `ValidateClineConfigBytes`) | **[EQUIV]** | A1 (no delta) |
| C2 | Canonical `RouteModel` IDs for GLM-5.2 and Kimi-K2.7 | exact registry IDs | Only **test placeholders** exist (`cp/cline-pass/glm-5.2`, `cline-kimi-k2.7-dedicated`) — not proven canonical | **[OMNI/BLOCKED_EXTERNAL]** | **A3** |
| C3 | `CLIKind` for the Cline frontend + executable resolution (`cline` binary) in gateway mode | must exist & be accepted | `agentBrainBuiltInCLIFor` maps **only** claude-code→`claude`, codex→`codex`; Cline path returns error | **[GAP]** | **W1** (`config.go`) |
| C4 | Config validation accepts the Cline CLIKind in gateway-required mode | must accept | `AgentBrainIntegrationConfig.Validate` switch allows **only** `CLIClaudeCode`/`CLICodex` | **[GAP]** | **W1** (`config.go`) |
| C5 | Adapter contract returns `AdapterReady` + `ProtocolOpenAIChat` for the Cline CLIKind | must be accepted | `CredentiallessAdapterContract(CLIOpenAICompatible)` returns **fail-closed** `GateOpenAICompatibleUnaccepted` | **[GAP]** | **A1/R1** (`adapter.go`) + W1 serialize |
| C6 | Trusted child-env injection for Cline (data dir + OmniRoute secret env, OpenAI base URL) | must inject as trusted-last | `trustedAdapterEntries` switch handles **only** ClaudeCode/Codex; `default: ErrAdapterFailClosed` | **[GAP]** | **W1** (`env.go`, shared) |
| C7 | Launch-path write of `providers.json` into a controlled Cline data dir | must write before spawn | `buildLaunch` writes config **only** for `CLICodex`; no Cline branch, and no `WriteCredentiallessClineConfig` exists | **[GAP]** | **W1** (`brain_integration.go`) + execenv writer |
| C8 | OpenAI-chat runtime profile lists the Cline CLIKind | must be allowed | `ProtocolOpenAIChat` profile **already** allows `CLIKimi, CLIOpenAICompatible, CLINIM` | **[EQUIV]** | — |
| C9 | Gateway model policy validates (CLIKind, model, protocol) | pure, gateway-aware | `GatewayModelPolicy.ValidateSelection` present & correct; depends on C5 adapter accept + C2 IDs | **[EQUIV] (blocked on C2/C5)** | — |
| C10 | One live Kanban→terminal run per family | after W1 integration | not run (reserved single live token) | **[IMPL/NOEV]** | W1 + Principal |

---

## 5. A1 — shared `providers.json` base (exact symbols)

**File:** `multica-auth-work/server/internal/daemon/runtimeenv/cline.go` (package `runtimeenv`).

Already-frozen, reusable-by-both-routes symbols:
- `NewClineConfigContract(gatewayRoot string, model brain.RouteModel, updatedAt string) (ClineConfigContract, error)`
- `ClineConfigContract.{Bytes(),Model(),BaseURL(),CredentialEnvKey(),Validate()}`
- `ValidateClineConfigBytes(raw []byte, model brain.RouteModel, baseURL, updatedAt string) error`
- constants: `ClineOpenAICompatibleProviderID="openai-compatible"`, `ClineOmniRouteAPIKeyEnv="CLINE_OMNIROUTE_API_KEY"`, `ClineSecretReferenceSentinel="${CLINE_OMNIROUTE_API_KEY}"`, `ClineTokenSourceOmniRoute="omniroute-stable-secret"`, `clineProvidersSchemaVersion=1`, `clineNVIDIARouteNamespace="nvidia"`
- errors: `ErrClineConfigContract`, `ErrClineRouteNotAgentBrainSelectable`
- `isNVIDIAOwnedRoute(model)` (rejects NVIDIA-owned fallback namespace — keeps NVIDIA OmniRoute-only)

**A1 verdict: zero code delta required in `cline.go` for the shared base.** It already:
- writes the exact installed carrier schema `{version,lastUsedProvider,providers}`;
- embeds **no secret** (apiKey = sentinel only), matching EVIDENCE_CONTRACT rule 5;
- targets `gatewayRoot + "/v1"` (consistent with the OpenAI-chat `BaseURLV1` profile + Cline
  appending `/chat/completions`);
- is model-parameterized, so GLM and Kimi share one code path (differ only by `RouteModel`).

**A1 open item (behavioral, not code):** the sentinel design assumes the credential arrives at
runtime via env var `CLINE_OMNIROUTE_API_KEY` and that `providers.json` keeps the literal sentinel
`${CLINE_OMNIROUTE_API_KEY}`. Whether **Cline 3.0.44 expands `${ENV}` inside `settings.apiKey`** is
**not proven in-repo**. Two mutually exclusive integration designs follow (W1 must pick one):
  1. **Env-expansion (preferred, keeps file secret-free):** child env carries
     `CLINE_OMNIROUTE_API_KEY=<resolved secret>`; file keeps the sentinel. **Requires proof Cline
     expands `${ENV}` in apiKey.**
  2. **Write-time substitution:** injector replaces the sentinel with the real secret in the bytes
     before writing. **Rejected unless unavoidable** — it puts the secret on disk, contradicting the
     no-embed invariant and `ValidateClineConfigBytes` (which *requires* the apiKey to equal the
     sentinel and rejects any concrete value).
  → **BLOCKED_EXTERNAL(behavior):** owner A1/W1 must confirm Cline env-expansion behavior against the
    installed CLI before the live run. If (1) is false and (2) is disallowed, this is a real blocker.

---

## 6. A2 — CLIKind / executable / mapping (exact symbols + old/new intent)

### 6.1 The mapping decision: reuse `CLIOpenAICompatible` (recommended) vs new `CLICline`

**Recommendation: reuse the existing `brain.CLIOpenAICompatible = "openai-compatible"` as the Cline
frontend CLIKind; do NOT introduce a new enum.** Evidence:
- `gateway/profiles.go` `ProtocolOpenAIChat.AllowedCLI` **already** includes `CLIOpenAICompatible`.
- `runtimeenv/adapter.go` **already** has a `CLIOpenAICompatible` case (currently fail-closed with
  `GateOpenAICompatibleUnaccepted`) — 5.6/5.7 acceptance is exactly the flip of that gate.
- `cline.go` uses provider discriminator `"openai-compatible"` throughout.
- Adds zero surface to the frozen `brain/**` `CLIKind` enum (a Codex1/W1 contract).

**Naming tension to record (W1 decides):** `CLIOpenAICompatible` is generic, but the only
Brain-selectable OpenAI-compatible frontend in P0 is **Cline** (executable `cline`, Cline 3.0.44).
The executable mapping therefore binds `openai-compatible → cline`. If the Principal wants an
explicit `cline` identity, the **alternative** is a new `brain.CLICline = "cline"` added to
`identity.go` (`supportedCLIKinds`), `adapter.go`, `profiles.go` `AllowedCLI`, and the two `config.go`
switches — strictly more W1 surface in the frozen brain contract. Primary path = reuse.

### 6.2 Exact edits required (all in W1-owned hotspots unless noted)

**(C3) `internal/daemon/config.go` — `agentBrainBuiltInCLIFor(kind brain.CLIKind)`** (owner: W1)
- OLD: `switch kind { case CLIClaudeCode: {claude,claude}; case CLICodex: {codex,codex}; default: error }`
- NEW intent: add `case brain.CLIOpenAICompatible: return agentBrainBuiltInCLI{Provider: "cline", Command: "cline"}, nil`
- Note: `resolveAgentBrainBuiltInEntry` already generically `exec.LookPath`s the returned `Command`,
  so no other change there. Gateway mode intentionally ignores `MULTICA_CLINE_PATH`/profile overrides
  (fail-closed), which is the correct credentialless posture — keep it.

**(C4) `internal/daemon/config.go` — `AgentBrainIntegrationConfig.Validate()`** (owner: W1)
- OLD: `switch c.CLIKind { case brain.CLIClaudeCode, brain.CLICodex: default: error "…only…Claude Code or Codex…" }`
- NEW intent: add `brain.CLIOpenAICompatible` to the accepted case; update the error string to include
  the accepted OpenAI-compatible (Cline) frontend.

**(C5) `internal/daemon/runtimeenv/adapter.go` — `CredentiallessAdapterContract`** (owner: A1/R1 within
`runtimeenv/**`; W1 serializes)
- OLD: `case brain.CLIOpenAICompatible: return {AdapterFailClosed, GateOpenAICompatibleUnaccepted}, err`
- NEW intent: `case brain.CLIOpenAICompatible: return AdapterContract{CLI: cli, State: AdapterReady, Protocol: brain.ProtocolOpenAIChat}, nil`
- This is the single gate flip that turns 5.6/5.7 from fail-closed to selectable. Keep `CLIKimi`,
  `CLINIM`, `CLIAntigravity` fail-closed (unchanged) — see collision section.

**(C6) `internal/daemon/runtimeenv/env.go` — `trustedAdapterEntries(profile AdapterEnvironment)`**
(owner: **W1**, shared file)
- OLD: switch handles `CLIClaudeCode` (ANTHROPIC_BASE_URL + ANTHROPIC_AUTH_TOKEN) and `CLICodex`
  (CODEX_HOME + `AGENT_BRAIN_OMNIROUTE_API_KEY`); `default: ErrAdapterFailClosed`.
- NEW intent: add `case brain.CLIOpenAICompatible:` that injects, as **trusted-last** entries:
  - `HOME = profile.TaskHome` (already set before switch),
  - the Cline data dir env (`CLINE_DATA_DIR` — see §6.3) as `originTrustedLocal`,
  - `CLINE_OMNIROUTE_API_KEY = profile.StableSecret.value` as `originTrustedSecret`,
  - return `secretKey = "CLINE_OMNIROUTE_API_KEY"`, `root` unchanged.
- `AdapterEnvironment`/`ChildEnvironment` likely need a `clineDataDir` field analogous to `codexHome`
  (currently `child.codexHome` is only set for `CLICodex`). W1 decides field vs. reuse of `TaskHome`.

**(C7) launch write** — two parts:
- `internal/daemon/execenv/` — **new** `WriteCredentiallessClineConfig(clineDataDir string, raw []byte) error`
  mirroring `WriteCredentiallessCodexConfig`: validate size (≤64KiB), create controlled dir 0700,
  write `providers.json` at the exact carrier subpath (§6.3) with 0600. (execenv is a Codex1/W1
  hotspot family — W1 authors/serializes.)
- `internal/daemon/brain_integration.go` — `buildLaunch`: OLD writes config only under
  `if plan.Task.Request.CLIKind == brain.CLICodex`. NEW intent: add an
  `else if … == brain.CLIOpenAICompatible` branch that calls `NewClineConfigContract(gatewayRoot,
  RouteModel, updatedAt)`, `Validate()`, then `WriteCredentiallessClineConfig(clineDataDir, contract.Bytes())`,
  and sets the task-home manifest appropriately (see §6.4). (`brain_integration.go` is W1-owned.)

### 6.3 Exact carrier path (materialization) — A1 to freeze, W1 to wire
Installed Cline carrier is `~/.cline/data/settings/providers.json` (per `cline.go` header) and
`execenv/cline_home.go` `resolveClineSourceDir` recognizes both `<root>/data/settings/providers.json`
and `<root>/settings/providers.json`. In gateway mode there is **no account home**; the controlled
data dir is created per-task. **Decision needed (A1+W1):** set `CLINE_DATA_DIR=<taskRoot>/cline-data`
and write `providers.json` at `<CLINE_DATA_DIR>/settings/providers.json`, mirroring the Codex
`agent-brain-home` pattern in `buildLaunch`. Freeze the exact relative subpath so the writer and the
env var agree.

### 6.4 Task-home manifest invariant (do not self-reject)
`runtimeenv/home.go` `ValidateTaskHomeManifest` **forbids** `providers.json`, `.cline`, and `.cline/`
in the task-home manifest (`ErrTaskHomeCredential`). The gateway Cline `providers.json` therefore
**must live in the separate `CLINE_DATA_DIR`, not in the task home**, and `buildLaunch` must NOT list
it in the `[]HomeEntry` manifest passed to `AssertPreLaunch`. (Codex lists `config.toml/sessions/skills`;
Cline's manifest is different and excludes the carrier.)

---

## 7. Collision risk — Kimi (explicitly required by A2 prompt)

- `brain.CLIKimi = "kimi"` is a **distinct executable frontend** (the native Kimi CLI), probed in
  `config.go` via `MULTICA_KIMI_PATH`/`kimi`, and is **fail-closed** in `adapter.go`
  (`GateNativeKimiUnaccepted`).
- Route **5.7 "Cline → Kimi-K2.7"**: the **frontend is Cline** (`CLIOpenAICompatible`, executable
  `cline`); **Kimi-K2.7 is the model** (`RouteModel`, owned by A3). It **must NOT** be mapped to
  `CLIKimi`.
- **Hard invariant for W1/A3:** both 5.6 and 5.7 use the **same** CLIKind (`CLIOpenAICompatible`),
  the **same** profile (`ProtocolOpenAIChat`), the **same** carrier code (`cline.go`); they differ
  **only** by the A3-frozen `RouteModel`. Do not add a Kimi-specific frontend branch; do not flip
  `CLIKimi` to ready. A model whose *name* contains "kimi" is not the `CLIKimi` frontend.
- Secondary collision (env deny-list, `runtimeenv/policy.go`): `CLINE_DATA_DIR` →
  `DenyCredentialRoot`, and `CLINE_` prefix + `API_KEY` fragment ⇒ `CLINE_OMNIROUTE_API_KEY` →
  `DenyProviderCredential`. Both keys are rejected by `ClassifyEnvironmentKey`/`ValidateCustomEnvironment`.
  They are legal **only** because trusted entries are merged **after** validation in
  `BuildGatewayEnvironment`. **Invariant:** these two keys must be injected **exclusively** through
  `trustedAdapterEntries` (C6), never via inherited/local/custom env, or the launch fails closed.

---

## 8. Test delta (focused, minimal)

Already green (no change): `runtimeenv/cline_test.go` — `TestNewClineConfigContract{KimiAndGLMRoutes,
MatchesInstalledCarrierSchema,EmbedsNoSecretValue,RejectsNVIDIAFallbackRoute,RejectsInvalidInputs}`,
`TestValidateClineConfigBytesDetectsTamper`. These already exercise both routes and the no-secret /
NVIDIA-reject / tamper invariants. NOTE: they use **placeholder** RouteModels
(`cp/cline-pass/glm-5.2`, `cline-kimi-k2.7-dedicated`) that A3 must reconcile; if A3's canonical IDs
differ, update the test fixtures (owner A3/A1, not a semantic change to the contract).

New focused tests required by the wiring gaps (author with each owning edit; run only the affected
package — see §10 commands):
1. `runtimeenv/adapter_test.go` — `CLIOpenAICompatible` now returns `AdapterReady` + `ProtocolOpenAIChat` (C5).
2. `runtimeenv/env_test.go` (or existing gateway env test) — `trustedAdapterEntries`/`BuildGatewayEnvironment`
   for `CLIOpenAICompatible` injects `CLINE_DATA_DIR` + `CLINE_OMNIROUTE_API_KEY` as trusted-last,
   secret redacted in `String()`/`Keys()`, value only in `Exec()` (C6).
3. `execenv` — `WriteCredentiallessClineConfig` writes 0600 `providers.json` at the frozen subpath,
   rejects empty/oversized, creates 0700 dir (C7).
4. `daemon` `config_test.go` — `agentBrainBuiltInCLIFor(CLIOpenAICompatible)` → `{cline,cline}` (C3);
   `AgentBrainIntegrationConfig.Validate()` accepts `CLIOpenAICompatible` in gateway mode (C4).
5. `daemon` `brain_integration_test.go` — `buildLaunch` for the Cline CLIKind writes providers.json to
   `CLINE_DATA_DIR` and passes a manifest that excludes the carrier (C7/§6.4).
6. `runtimeenv/model_test.go` — `GatewayModelPolicy.ValidateSelection(CLIOpenAICompatible, <A3 GLM ID>,
   "")` and `(…, <A3 Kimi ID>, "")` pass once C2/C5 land (C9).

No broad regression, no second QA, no live-run duplication (per protocol).

---

## 9. W1 handoff — integration order, ownership, invariants

**Ownership map (per `FILE_OWNERSHIP.md` P0 override + hotspot table):**
- `runtimeenv/cline.go` — **no change** (A1 confirms EQUIV).
- `runtimeenv/adapter.go`, `runtimeenv/env.go` (shared), `runtimeenv/home.go`, `runtimeenv/policy.go`
  — `runtimeenv/**` is R1/Cline scope, but `env.go`'s `trustedAdapterEntries` is a **shared switch**;
  escalate to **W1-serial**.
- `internal/daemon/config.go`, `internal/daemon/brain_integration.go`, `internal/daemon/execenv/**`,
  `internal/daemon/brain/**` — **W1-only** hotspots (Codex1/W1). All of C3, C4, C6, C7 land here.

**Recommended serial order (W1):**
1. **A3 delivers canonical `RouteModel` IDs** for GLM-5.2 and Kimi-K2.7 → unblocks C2/C9/live run.
   Until then, C5 flip can be prepared but selection stays unproven (`BLOCKED_EXTERNAL`).
2. C5 adapter flip (`adapter.go`) — smallest change, unblocks env + policy.
3. C6 trusted env injection (`env.go`) — must land with the `CLINE_DATA_DIR`/`CLINE_OMNIROUTE_API_KEY`
   trusted-only invariant (§7) and the `ChildEnvironment` data-dir field decision.
4. C7 `WriteCredentiallessClineConfig` (execenv) + `buildLaunch` Cline branch + manifest exclusion (§6.4).
5. C3 + C4 config wiring so gateway-required mode resolves the `cline` executable and passes validation.
6. Resolve the Cline `${ENV}`-expansion behavior (§5 open item) before the single reserved live run.

**Non-negotiable invariants W1 must preserve:**
- No secret embedded in `providers.json` (apiKey stays the sentinel); `ValidateClineConfigBytes`
  already enforces this.
- `CLINE_DATA_DIR` and `CLINE_OMNIROUTE_API_KEY` injected **only** as trusted entries.
- Carrier lives in `CLINE_DATA_DIR`, excluded from the task-home manifest.
- Keep `CLIKimi`/`CLINIM`/`CLIAntigravity` fail-closed; NVIDIA stays OmniRoute-only.
- Both Cline routes share one CLIKind/profile/contract; only `RouteModel` differs.
- No auth/credential/quota/retry/failover logic added to the Brain (OmniRoute-owned).

---

## 10. Verification commands (focused; run against `/home/ec2-user/goroot/go/bin/go`)

Not executed in this read-only phase (no product code changed). For the owning edits:
```
cd multica-auth-work/server
/home/ec2-user/goroot/go/bin/go test ./internal/daemon/runtimeenv/...
/home/ec2-user/goroot/go/bin/go test ./internal/daemon/execenv/...
/home/ec2-user/goroot/go/bin/go test ./internal/daemon/...        # daemon package (config, brain_integration)
/home/ec2-user/goroot/go/bin/gofmt -l internal/daemon/runtimeenv internal/daemon internal/daemon/execenv
```

---

## 11. Blockers / limitations

- **BLOCKED_EXTERNAL (A3):** canonical `RouteModel` IDs for GLM-5.2 and Kimi-K2.7 are not frozen;
  `cline_test.go` values are placeholders. Owner: **A3** (`A3-route-freeze.md`). No ID invented here.
- **BLOCKED_EXTERNAL (behavior):** Cline 3.0.44 `${ENV}` expansion in `settings.apiKey` unproven
  in-repo (§5). Owner: A1/W1 — verify against installed CLI before live run.
- **Env note (resolved):** go1.26.1 used from the corrected canonical path
  `/home/ec2-user/goroot/go` (§1); no action outstanding.
- This phase is READ-ONLY; all §6/§9 edits are **handoff intent** for W1/R1, not applied changes.

---

## 12. Agent status block

- STATUS: DONE (read-only analysis + handoff)
- DELIVERED: preflight; current-vs-required matrix (C1–C10); exact A1 symbols (zero delta) and A2
  CLIKind/executable/mapping edits; Kimi + deny-list collision analysis; test delta; W1 handoff with
  ordering/invariants; blockers.
- FILES: `.deploy-control/p0/handoffs/A1-A2-cline-foundation.md` (only mutable file).
- VALIDATION: none run (no product code changed this phase); focused commands specified in §10.
- EVIDENCE: this artifact; source citations by exact symbol/file in §5–§9.
- BLOCKERS/LIMITATIONS: §11.
- W1_HANDOFF: §6, §9 (C3–C7 in W1 hotspots; C5 in runtimeenv/adapter.go serialized by W1).
