# D6 — Pre-launch fail-closed contract adjudication (Cline / `CLIOpenAICompatible`)

- agent: **Opus48#C** · lane **A10-DESIGN** · task **P0-D6-PRELAUNCH-ADJUDICATION** · pane `w8:p1`
- sole mutable lock: `.deploy-control/p0/handoffs/D6-prelaunch-adjudication.md`
- check-in: `.deploy-control/p0/checkins/Opus48-C__P0-D6-PRELAUNCH-ADJUDICATION__20260721T233544Z.json`
- as-of (UTC): `2026-07-21T23:35Z` · working tree HEAD `a6d50986…` (W1 `P0-W1-IMPLEMENTATION` in flight)
- posture: **READ-ONLY source inspection.** No product edits, no tests, no live run. Design adjudication for W1.
- **Sequencing exception (recorded):** the source reads below occurred *before* this task's check-in because the 23:32Z check-in write failed on a full disk (`No space left on device`, then 24K free). Disk later recovered to ~286M and the check-in succeeded at 23:35:44Z. No product edits were made during the pre-check-in reads.
- preflight: git 2.50.1 · python3 3.9.25 · rg 15.2.0 · sha256sum coreutils 8.32 (all present; no compile run — read-only).

## 0. Verdict

**assert.go MUST change.** `AssertPreLaunch` currently **fails closed for `CLIOpenAICompatible`** in **three** places, so the accepted Cline route (5.6/5.7) cannot launch even though adapter/env/execenv/config are already implemented. The minimum fail-closed D6 contract is **4 edits across 2 files** (assert.go ×3, env.go ×1), the pivotal one being a **new `ChildEnvironment.clineDataDir` field** required for independent data-dir containment. This **corrects R1-DESIGN D2**, which claimed no `ChildEnvironment` field was needed. Current fail-closed behavior is *safe* (no insecure launch) but **blocks the GLM/Kimi live runs** until D6 lands.

## 1. Current working-tree state (verified read-only)

| Component | Symbol | State | Note |
|---|---|---|---|
| Adapter | `adapter.go CredentiallessAdapterContract(CLIOpenAICompatible)` | **DONE** | returns `AdapterReady`/`ProtocolOpenAIChat` (gate flipped) |
| Env inject | `env.go trustedAdapterEntries` case `CLIOpenAICompatible` | **DONE** | injects `CLINE_DATA_DIR`(trustedLocal)+`CLINE_OMNIROUTE_API_KEY`(trustedSecret); compose-time `validatePhysicalControlledDirectory(profile.ClineDataDir)` |
| Env field (input) | `env.go AdapterEnvironment.ClineDataDir` | **DONE** | input field present |
| Execenv writer | `execenv/cline_home.go WriteCredentiallessClineConfig` + `prepareCredentiallessClineHome` | **DONE** | 0700 dir, 0600 `settings/providers.json`, ≤64KiB |
| Config bytes | `cline.go ValidateClineConfigBytes` / `NewClineConfigContract` | **DONE** | exact key set, sentinel-only apiKey (no secret), `/v1` base, NVIDIA reject, ≤64KiB |
| Manifest guard | `home.go ValidateTaskHomeManifest` | **DONE** | rejects `providers.json`/`.cline` in task home |
| **Pre-launch gate** | **`assert.go AssertPreLaunch`** | **NOT DONE — fails closed for Cline** | **the D6 gap (below)** |
| **Env field (readback)** | **`env.go ChildEnvironment.clineDataDir`** | **MISSING** | required by D6 containment (below) |

## 2. The three assert.go defects (all currently → `ErrPreLaunchPolicy` for Cline)

### D6-1 — final CLI switch `default` rejects Cline
```go
switch environment.cli {
case brain.CLIClaudeCode: if plan.CodexConfig != nil { return ErrPreLaunchPolicy }
case brain.CLICodex:      if plan.CodexConfig == nil || plan.CodexConfig.Validate() != nil { return ErrPreLaunchPolicy }
default:                  return ErrPreLaunchPolicy   // ← CLIOpenAICompatible lands here
}
```
Fix — add:
```go
case brain.CLIOpenAICompatible:
    if plan.CodexConfig != nil { return ErrPreLaunchPolicy }
```
Rationale: Cline carries **no** `CodexConfig` in `LaunchPlan`; its `providers.json` carrier is written to `CLINE_DATA_DIR` by `WriteCredentiallessClineConfig` before launch. assert.go must not (and cannot) re-validate the carrier bytes — that is `ValidateClineConfigBytes`' job — preserving assert.go's "never compares/formats/hashes the secret value" content-free invariant.

### D6-2 — `trustedEntryAllowed` has no Cline case (returns `false`) → trusted keys rejected
The denied-key loop rejects any `Denied` key unless `trustedEntryAllowed(cli, canonical, origin)`:
```go
if classification.Denied && !trustedEntryAllowed(environment.cli, canonical, entry.origin) { return ErrPreLaunchPolicy }
```
`CLINE_DATA_DIR` → `DenyCredentialRoot`; `CLINE_OMNIROUTE_API_KEY` → `DenyProviderCredential`. With no `CLIOpenAICompatible` case, `trustedEntryAllowed` returns `false` ⇒ the legitimately-trusted Cline entries are rejected.
Fix — add to `trustedEntryAllowed`:
```go
case brain.CLIOpenAICompatible:
    switch canonical {
    case "HOME":                   return origin == originTrustedLocal
    case "CLINE_DATA_DIR":         return origin == originTrustedLocal
    case ClineOmniRouteAPIKeyEnv:  return origin == originTrustedSecret   // "CLINE_OMNIROUTE_API_KEY"
    }
```
Note: `canonical` is the upper-cased key; `ClineOmniRouteAPIKeyEnv` is already upper-case and must equal `strings.ToUpper(environment.secretKey)` for the single-secret check to pass.

### D6-3 (SECURITY-MATERIAL) — `launchRootsAreControlled` never checks the Cline data-dir containment
```go
func launchRootsAreControlled(executionRoot string, environment ChildEnvironment) bool {
    ... HOME exactPathWithin check ...
    if environment.cli != brain.CLICodex { return environment.codexHome == "" }  // ← Cline: only asserts codexHome=="", NEVER validates CLINE_DATA_DIR
    codexHome, ok := environment.entries["CODEX_HOME"]
    return ok && exactPathWithin(executionRoot, environment.codexHome, codexHome.value)
}
```
For Cline the function returns `environment.codexHome == ""` and **never verifies** that `CLINE_DATA_DIR` is absolute/canonical/physical/symlink-free/within the execution root. `exactPathWithin(root, expected, actual)` deliberately takes **two** sources — the trusted struct copy (`expected`) and the entries-map value (`actual`) — and requires `expected == actual` AND containment AND physical resolution. For Codex the trusted copy is `environment.codexHome`. **Cline has no equivalent trusted copy on `ChildEnvironment`**, so containment cannot be checked without the new field (D6-4).
Fix — restructure to a switch:
```go
switch environment.cli {
case brain.CLICodex:
    codexHome, ok := environment.entries["CODEX_HOME"]
    return ok && exactPathWithin(executionRoot, environment.codexHome, codexHome.value)
case brain.CLIOpenAICompatible:
    if environment.codexHome != "" { return false }
    clineDataDir, ok := environment.entries["CLINE_DATA_DIR"]
    return ok && exactPathWithin(executionRoot, environment.clineDataDir, clineDataDir.value)
default:
    return environment.codexHome == ""
}
```

## 3. D6-4 (env.go) — new `ChildEnvironment.clineDataDir` field (REQUIRED; corrects R1-DESIGN D2)

`ChildEnvironment` currently is `{entries, cli, gatewayRoot, secretKey, taskHome, codexHome}` — **no `clineDataDir`**. `BuildGatewayEnvironment` sets `child.codexHome` only for `CLICodex`. D6-3's containment check needs the **trusted authoritative** data-dir path (not just the mutable entries value) to compare via `exactPathWithin`. Therefore:
- **Add field:** `clineDataDir string` to `ChildEnvironment`.
- **Populate:** in `BuildGatewayEnvironment`, alongside the existing codex block:
```go
if opts.Adapter.CLI == brain.CLICodex { child.codexHome = opts.Adapter.CodexHome }
if opts.Adapter.CLI == brain.CLIOpenAICompatible { child.clineDataDir = opts.Adapter.ClineDataDir }
```
This is the exact analogue of `codexHome`. **R1-DESIGN D2 said "No new `ChildEnvironment` struct field is required … only add `ChildEnvironment.clineDataDir` if a launch consumer must read it back; none identified." D6 IS that consumer** — the pre-launch containment gate must read it back. The field is mandatory for the fail-closed invariant, not optional.

## 4. CLINE_DATA_DIR controlled-root validation — the two required layers

| Layer | Where | Check | Status |
|---|---|---|---|
| Compose-time | `env.go trustedAdapterEntries` | `validatePhysicalControlledDirectory(profile.ClineDataDir, "Cline data dir")` — absolute, canonical, non-root, existing dir, no symlink components | **DONE** |
| Pre-launch (independent) | `assert.go launchRootsAreControlled` (D6-3) | `exactPathWithin(executionRoot, environment.clineDataDir, entries["CLINE_DATA_DIR"].value)` — trusted copy == entries value, within execution root, physical/symlink-free | **MISSING** |

Both are required (defense-in-depth): the compose-time check validates the input; the pre-launch check independently re-verifies from the (immutable, trusted) `ChildEnvironment` right before spawning the child with the real secret — exactly as HOME/CODEX_HOME are double-checked today.

## 5. Config-byte validation — no assert.go change

`ValidateClineConfigBytes` (cline.go) + the `WriteCredentiallessClineConfig` size guard fully cover byte validation (exact key set, sentinel-only `apiKey`, `/v1` base, NVIDIA reject, ≤64KiB). **assert.go must NOT re-validate bytes** — it stays env/roots/provenance-only and secret-value-free. assert.go also does not verify the carrier file exists (that is `buildLaunch`'s write-before-assert responsibility; a missing carrier fails deterministically at Cline spawn). Recommendation: **do not** add file/content checks to assert.go.

## 6. Ownership & test implications

- **assert.go** (`AssertPreLaunch`, `trustedEntryAllowed`, `launchRootsAreControlled`) and **env.go** (`ChildEnvironment` struct + `BuildGatewayEnvironment`) are **W1-exclusive** (shared `runtimeenv` + brain_integration seam). Per `W1-source-lock-freeze.md`, env.go = **F14** (W1 serial); assert.go needs a **new F-row (propose `F-ASSERT`)**, W1 serial, integrated in Wave-C **step 3 (Cline) before the GLM live run**.
- **Tests (assert_test.go, W1-owned, coupled to the assert.go edit):** extend the existing tables —
  - add `TestAssertPreLaunchAcceptsControlledClinePlan` (mirror the Claude case: `CLIOpenAICompatible`, `ClineDataDir` set, `CodexConfig=nil`, empty/carrier-excluded manifest → `nil`);
  - extend `TestAssertPreLaunchRejectsHomesOutsideOrEscapingExecutionRoot` with `{cli: CLIOpenAICompatible, outsideCline:true}` and a `CLINE_DATA_DIR` traversal case;
  - extend `TestAssertPreLaunchRejectsPhysicalHomeSubstitution` with `CLIOpenAICompatible` (symlinked data dir);
  - add a denied-origin case: `CLINE_DATA_DIR`/`CLINE_OMNIROUTE_API_KEY` with wrong origin → reject; and a `CodexConfig!=nil` for Cline → reject.
  - This aligns with R3-TEST-DESIGN / A9-R3 (`R3-lifecycle-focused-tests.md`): the T-ASSERT group must be expanded to include the containment + trusted-entry cases D6 introduces.

## 7. Security rationale (concise)

`AssertPreLaunch` is the **last independent fail-closed gate** before the child is spawned with the real OmniRoute secret in its environment. It re-verifies that every controlled root is canonical/physical/within the execution root and that no denied env key rides except the exact trusted set. If the **Cline data dir is not independently containment-checked** here, a compose bug or a tampered entries map could place the writable data dir (holding the injected `providers.json` and receiving the trusted secret env) **outside** the controlled sandbox — enabling path traversal/escape or leaking the injected secret env into an uncontrolled location. The `clineDataDir` struct field is the **trusted authoritative** path compared against the mutable entries value (identical protection to `codexHome`). Keeping bytes/secret out of assert.go preserves its content-free/secret-free contract.

## 8. New lock recommendation

Add to `W1-source-lock-freeze.md` (W1-exclusive, serial, Wave-C step 3, before any Cline live run):
- **F-ASSERT** — `internal/daemon/runtimeenv/assert.go` (`AssertPreLaunch` switch, `trustedEntryAllowed`, `launchRootsAreControlled`) + `assert_test.go`.
- **F14 (amend)** — `internal/daemon/runtimeenv/env.go`: add `ChildEnvironment.clineDataDir` field + populate in `BuildGatewayEnvironment`.
Both are already W1-owned (no new lane); pairwise-∅ preserved (same `runtimeenv` package, single W1 editor, serial). No R1/A1 edit — this is integrator-hotspot work.

## 9. Stop conditions (fail-closed; assert.go returns `ErrPreLaunchPolicy` / no spawn)

1. `CLINE_DATA_DIR` not within the execution root, or not canonical/physical/symlink-free → reject.
2. `environment.clineDataDir` (trusted copy) ≠ `entries["CLINE_DATA_DIR"].value` → reject.
3. Any denied key rides other than the exact trusted set `{HOME, CLINE_DATA_DIR, CLINE_OMNIROUTE_API_KEY}` → reject.
4. `secretCount != 1` or the trusted-secret canonical ≠ `CLINE_OMNIROUTE_API_KEY` → reject.
5. `plan.CodexConfig != nil` for Cline → reject.
6. `TaskHome` manifest contains `providers.json`/`.cline` → reject (home.go).
7. adapter contract not `AdapterReady` for `CLIOpenAICompatible` → reject.
8. empty `entries` / empty `secretKey` / empty `gatewayRoot` → reject.

**Consequence for the fleet:** until F-ASSERT + F14(amend) land, `AssertPreLaunch` fail-closes Cline — W1's Cline implementation is **incomplete** and the GLM/Kimi `live_runs` (A9-GLM/A9-R1) **cannot succeed** through the real launch path. This is the concrete pre-live blocker to hand W1 before the reserved GLM run.

## 10. Agent status
- STATUS: DONE (read-only adjudication)
- DELIVERED: 4-edit minimum fail-closed D6 contract (assert.go ×3 + env.go field), CLINE_DATA_DIR two-layer containment, config-byte separation, ownership/test plan, security rationale, lock rows, stop conditions.
- FILES: `.deploy-control/p0/handoffs/D6-prelaunch-adjudication.md` (only mutable file).
- VALIDATION: none run (read-only, no compile). Symbols verified by direct source read at HEAD `a6d5098`: `assert.go` (AssertPreLaunch/launchRootsAreControlled/trustedEntryAllowed), `env.go` (ChildEnvironment/AdapterEnvironment/trustedAdapterEntries/BuildGatewayEnvironment), `adapter.go`, `home.go`, `cline.go`, `execenv/cline_home.go`, `assert_test.go`.
- BLOCKERS/LIMITATIONS: read-only phase; edits are handoff intent for W1. Disk was 100% full at 23:32Z (recovered to ~286M); sequencing exception recorded above.
- W1_HANDOFF: §2 (D6-1/2/3), §3 (D6-4 field), §6 (tests), §8 (locks), §9 (stop conditions). Land before the reserved Cline live run.
