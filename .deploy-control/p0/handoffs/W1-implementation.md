# W1 — Implementation Log (P0-W1-IMPLEMENTATION, GLM-ready D1–D7)

- agent: `Opus48#B` · pane: `w6:p2` · lane: `W1` · task: `P0-W1-IMPLEMENTATION`
- check-in: `.deploy-control/p0/checkins/Opus48-B__P0-W1-IMPLEMENTATION__20260721T230931Z.json`
- authorization: `control.json` `implementation_authorized=true`, phase `IMPLEMENTATION_W1_SERIAL`,
  `authorized_delta="GLM-ready D1-D7 only under activated 16-file lock set"`, `live_runs_authorized=false`
- repo: branch `integration/dev-transition-candidate-20260719`, HEAD `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- Go: `/home/ec2-user/goroot/go/bin/go` → `go1.26.1 linux/amd64`
- scope guard: GLM only; NO Kimi ID, NO Opus48 ID/config, NO auth/credential/quota/fallback, NO live run.

---

## Transition (atomic)

1. `P0-W1-SOURCE-LOCK-ACTIVATION` → checked out DONE (evidence `W1-source-lock-activation.md`).
2. `P0-W1-IMPLEMENTATION` → checked in immediately, same 16 product files + this log
   (`.deploy-control/p0/handoffs/W1-implementation.md`), lane W1.

## Pre-edit source review (read-only; before any edit) — MANDATORY per "block on unplanned file"

Before touching D1, I read the exact edit sites and searched existing tests for assertions the D-edits
would break. **A blocking conflict was found before any source edit was made** (build kept green).

### BLOCKING CONFLICT — D1/D2 break two pre-existing tests that assert the OLD fail-closed contract

D1 flips `runtimeenv/adapter.go` `CredentiallessAdapterContract(CLIOpenAICompatible)` from
`AdapterFailClosed{GateOpenAICompatibleUnaccepted}` to `AdapterReady, ProtocolOpenAIChat`. D2 then lets
`BuildGatewayEnvironment` build a real child env for `CLIOpenAICompatible`. Two existing tests assert the
exact opposite and would FAIL under the D1-focused check `go test ./internal/daemon/runtimeenv/...`:

| Test file (package `runtimeenv`) | Test | Exact assertion broken | In W1 lock? | Handoff owner |
|---|---|---|---|---|
| `internal/daemon/runtimeenv/model_test.go` | `TestCredentialBearingNativeAdaptersFailClosed` | `:35` table row `{CLIOpenAICompatible, GateOpenAICompatibleUnaccepted}`; `:43` `errors.Is(err, ErrAdapterFailClosed)`; `:46` `contract.State != AdapterFailClosed` | **NO** | R1 (`R1-test-implementation-readiness.md`, `Opus48#A`) |
| `internal/daemon/runtimeenv/isolation_g4_test.go` | `TestG4NativeCredentialBearingAdaptersStayFailClosed` | `:135` table row `{CLIOpenAICompatible, GateOpenAICompatibleUnaccepted}`; `:144` asserts fail-closed; `:162` asserts `BuildGatewayEnvironment(CLI:CLIOpenAICompatible)` returns `ErrAdapterFailClosed` with `len(Keys())==0` | **NO** | **UNASSIGNED** (not in R1, R3, or W1 sets) |

Consequences:
- Making the `runtimeenv` package green after D1/D2 **requires editing both test files**, which are
  **outside W1's 16-file lock** → touching them = "unplanned file" (forbidden by this task's directive).
- `model_test.go` is R1's to update (R1 §6.1/§6.7 already plans the new-behavior assertions), but the
  freeze/manifest safeguard says **W1 lands+greens the package source before R1 authors** — that ordering
  cannot hold while the OLD assertions remain and are R1/unassigned-owned.
- `isolation_g4_test.go` is owned by **no active lane** (absent from R1, R3, and W1). It will stay RED
  indefinitely unless an owner is assigned to remove `CLIOpenAICompatible` from its fail-closed tables.

This is a cross-lane ownership/ordering conflict that only the Principal can resolve. Proceeding would
either (a) break the build and leave failing focused tests, or (b) require W1 to edit unplanned files —
both prohibited. **No source was edited.**

### Remediation options (Principal decides; W1 does not guess)

- **Option B (recommended, minimal, mirrors D7 lockstep):** expand W1's lock to add
  `internal/daemon/runtimeenv/model_test.go` and `internal/daemon/runtimeenv/isolation_g4_test.go` as
  **coupled lockstep test edits**, so W1 removes `CLIOpenAICompatible` from both fail-closed tables (and
  adds a positive `AdapterReady`/`ProtocolOpenAIChat` assertion + a real-env `BuildGatewayEnvironment`
  case) in the same serial step as D1/D2 — keeping the package green after the step. R1 then adds only
  its net-new behavioral cases (adapter_test.go/env_test.go) without touching these two.
- **Option A (sequencing):** relax the per-step green gate for `runtimeenv`: W1 lands D1/D2 (source),
  accept a transient RED, and have R1 immediately update `model_test.go` + take ownership of
  `isolation_g4_test.go` to reach green. Requires assigning `isolation_g4_test.go` to R1.
- Either way, **`isolation_g4_test.go` must get an explicit owner** — it is currently unassigned.

Until resolved, D1 (and therefore the D1→D7 chain, since D6 depends on D1/D2/D5) cannot proceed with a
green focused check under the current lock set.

---

## D1+D2 APPLIED (2026-07-21, after R1 locked the 5 runtimeenv test files)

Authorized to apply D1+D2 source only (adapter.go/env.go), gofmt, no runtimeenv test, no D3.

### D1 — `internal/daemon/runtimeenv/adapter.go`
- Symbol `CredentiallessAdapterContract`, case `brain.CLIOpenAICompatible`:
  - OLD: `AdapterContract{State: AdapterFailClosed, Gate: GateOpenAICompatibleUnaccepted}` + `&AdapterGateError{...}`
  - NEW: `AdapterContract{CLI: cli, State: AdapterReady, Protocol: brain.ProtocolOpenAIChat}, nil`
- Function doc comment updated to state Claude/Codex/**accepted OpenAI-compatible (Cline)** are ready and
  Kimi/NIM/Antigravity remain fail-closed. Kimi/NIM/Antigravity cases UNCHANGED (still fail-closed).

### D2 — `internal/daemon/runtimeenv/env.go`
- Symbol `AdapterEnvironment` struct: **added field** `ClineDataDir string` (after `CodexHome`).
- Symbol `trustedAdapterEntries`: **added** `case brain.CLIOpenAICompatible` (before `default`):
  validates `profile.ClineDataDir` via `validatePhysicalControlledDirectory`, injects
  `CLINE_DATA_DIR` (`originTrustedLocal`) + `ClineOmniRouteAPIKeyEnv` (`CLINE_OMNIROUTE_API_KEY`,
  `originTrustedSecret`), returns `(entries, root, ClineOmniRouteAPIKeyEnv, nil)`.
- `BuildGatewayEnvironment` unchanged (no `ChildEnvironment` field needed; `CLINE_DATA_DIR` flows via
  `entries`→`Exec()`). `policy.go`/`home.go` UNCHANGED — their existing deny-list (rejects `CLINE_*`
  from untrusted origins) and manifest guard (forbids `providers.json` in task home) are exactly the
  invariants D2 relies on; no edit needed.

### Verification (source-only; NO tests run)
- `gofmt -w` applied; `gofmt -l` → empty (both files formatted).
- `/home/ec2-user/goroot/go/bin/go build ./internal/daemon/runtimeenv/` → **BUILD OK** (compiles;
  build excludes `_test.go`, so the R1-locked test files are neither read nor run).
- `git status` (product source): only `adapter.go` + `env.go` modified. Diffstat: 2 files, +22/-5.
  **No additional affected file** — did not touch policy.go/home.go or any daemon/pkg/agent file.

## Status: BLOCKED (D1+D2 applied; awaiting R1 synchronized test updates + runtimeenv green).

Serial progress: **D1 ✔, D2 ✔ (source only, not tested).** D3–D7 NOT started. runtimeenv package test
intentionally NOT run (R1 owns the synchronized expectation updates in the 5 locked test files).

- **Blocker:** runtimeenv package test would fail until R1's synchronized expectation updates land
  (`model_test.go`/`isolation_g4_test.go` still assert the OLD fail-closed `CLIOpenAICompatible`; R1 now
  owns and will flip them). W1 must not run the runtimeenv test yet.
- **Owner:** Principal + A9-R1 (`Opus48#A`).
- **Next action:** R1 applies the synchronized expectation changes across its 5 locked runtimeenv test
  files (assert `AdapterReady`/`ProtocolOpenAIChat` + real-env `BuildGatewayEnvironment` for
  `CLIOpenAICompatible`; keep Kimi/NIM/Antigravity fail-closed). W1 signals R1 that D1+D2 are applied;
  once R1 confirms, W1 runs the focused `go test ./internal/daemon/runtimeenv/...` for green, then
  proceeds to D5→D6→D3/D4→D7. No source edit/test/live run while blocked.

---

## D3+D4+D5 APPLIED; D6 STOPPED (2026-07-21, after R1 D1+D2 suite GREEN + R3 test locks)

Authorized to apply D5+D6+D3+D4 source; consumed R1's GREEN D1+D2 suite (no rerun); R3 owns/locked
`cline_home_test.go`, `config_test.go`, `brain_integration_test.go` (not run here).

### D5 — `internal/daemon/execenv/cline_home.go` (APPLIED)
- Added import `strings`.
- Added `prepareCredentiallessClineHome(clineDataDir)` (0700 `clineDataDir` + `settings/`) and
  `WriteCredentiallessClineConfig(clineDataDir, raw)` writing `<clineDataDir>/settings/providers.json`
  at 0600 (mirrors `WriteCredentiallessCodexConfig`; rejects empty/oversized/relative/root; carrier is
  the non-secret contract bytes). Legacy `prepareClineHome`/`resolveClineSourceDir` untouched.

### D3 — `internal/daemon/config.go` `agentBrainBuiltInCLIFor` (APPLIED)
- Added `case brain.CLIOpenAICompatible: return {Provider:"cline", Command:"cline"}`.

### D4 — `internal/daemon/config.go` `AgentBrainIntegrationConfig.Validate` (APPLIED)
- Accepted `brain.CLIOpenAICompatible` in the CLIKind switch; message updated to include the
  OpenAI-compatible (Cline) frontend.

### Verification (source-only; NO R3 tests run)
- `gofmt -l` clean on all touched source (adapter.go, env.go, config.go, cline_home.go).
- `go build ./internal/daemon/execenv/` → EXECENV BUILD OK.
- `go build ./internal/daemon/` → DAEMON BUILD OK.
- My product-source edits this session: `runtimeenv/adapter.go`, `runtimeenv/env.go` (D1/D2),
  `internal/daemon/config.go`, `internal/daemon/execenv/cline_home.go` (D3/D4/D5). The modified
  `runtimeenv/{env_test.go,isolation_g4_test.go,model_test.go}` + new `runtimeenv/adapter_test.go` are
  **R1's** synchronized test updates (Opus48#A), consumed as GREEN — not W1 edits.

### D6 — `brain_integration.go buildLaunch` Cline branch — STOPPED (stop condition hit)

D6 cannot be applied cleanly: it requires editing `internal/daemon/runtimeenv/assert.go`, which is
**NOT in W1's 16-file lock and NOT locked by any active record**. Two hard rejections in
`AssertPreLaunch` block a Cline launch:
1. CLI `switch` ends with `default: return ErrPreLaunchPolicy` — `CLIOpenAICompatible` hits default.
2. `trustedEntryAllowed` has no Cline case; `policy.go` `ClassifyEnvironmentKey` **denies** both
   `CLINE_DATA_DIR` (DenyCredentialRoot) and `CLINE_OMNIROUTE_API_KEY` (CLINE_ prefix →
   DenyProviderCredential) — correct and intended — so the denied-key loop in `AssertPreLaunch` rejects
   them unless a Cline branch is added to `trustedEntryAllowed`.

Plus a genuine **design ambiguity** (assert.go, unowned):
- The CLI `switch` Cline case: assert `plan.CodexConfig == nil` (like Claude), or add a `ClineConfig`
  field to `LaunchPlan` for symmetric pre-launch validation of the providers.json contract?
- `launchRootsAreControlled` validates `CODEX_HOME` within the execution root for Codex but only checks
  `HOME` for non-Codex. Should `CLINE_DATA_DIR` be proven within the execution root too? That requires a
  `ChildEnvironment.clineDataDir` field — which R1-DESIGN D2 explicitly said was **not** needed. This is
  a real security/design decision, not a mechanical edit.

`home.go` `ValidateTaskHomeManifest(CLIOpenAICompatible, [])` returns nil for the empty Cline manifest —
no home.go edit needed. `buildLaunch` was **not** edited (kept clean; applying a branch that fails
`AssertPreLaunch` would be knowingly broken and would fail R3's `brain_integration_test.go`).

## Status: BLOCKED (D1✔ D2✔ D3✔ D4✔ D5✔; D6 stopped). No live run. R3 tests not run.

- **Blocker:** D6 requires `runtimeenv/assert.go` (unlocked, outside W1's 16-file set) and a design
  decision on Cline pre-launch validation; W1 must not touch an unowned file or resolve the design
  unilaterally. Also awaiting R3's focused tests (`cline_home_test.go`/`config_test.go`/
  `brain_integration_test.go`) to validate D3/D4/D5.
- **Owner:** Principal Orchestrator (+ freeze owner for assert.go ownership; A9-R3 for the R3 tests).
- **Next action:** Principal grants W1 an explicit lock on `internal/daemon/runtimeenv/assert.go`
  (currently unowned) and rules on the Cline pre-launch design: (i) `AssertPreLaunch` switch Cline case
  (CodexConfig==nil vs new `LaunchPlan.ClineConfig`); (ii) `trustedEntryAllowed` Cline case
  (`CLINE_DATA_DIR`→trusted-local, `CLINE_OMNIROUTE_API_KEY`→trusted-secret); (iii) whether
  `launchRootsAreControlled` must validate `CLINE_DATA_DIR` within the execution root (needs a
  `ChildEnvironment.clineDataDir` field in env.go — in W1's lock — contradicting R1-DESIGN "no field
  needed"). Then W1 applies D6 + the assert.go changes, signals R3, and R3 runs its focused tests.
  D7 not started (comes after D6/R3 green). No source edit/test/live run while blocked.

---

## Adjudication routing + infra note (2026-07-21T23:3x Z)

- **D6 block owner refined:** Principal Orchestrator + **design worker Opus48#C** (owns the minimal
  prelaunch-contract decision: `LaunchPlan.ClineConfig` field vs controlled `CLINE_DATA_DIR`-within-root
  validation).
- **Next action (sequenced):** (1) consume the forthcoming **D6-prelaunch-adjudication** from Opus48#C;
  (2) acquire an explicit lock on `internal/daemon/runtimeenv/assert.go` via a **new check-in** BEFORE any
  edit; only then apply D6 + the adjudicated assert.go changes and signal R3. **Do not touch
  `assert.go`/`brain_integration.go`; do not proceed D7** until the adjudication + lock land.
- **INFRA (resolved):** the workspace filesystem reached 100% (24G/24G) during D3/D4/D5 verification,
  which caused two `p0_control` writes to fail with `Errno 28` (the block did not persist on first try).
  Freed space safely via `go clean -cache -modcache` (regenerable Go build/module caches, ~283M); now
  ~286M free. Block/heartbeat re-recorded successfully afterward. Flagged for Principal/infra: workspace
  disk headroom is thin; heavy `go build`/module downloads can re-fill it.

Applied+green this session: **D1, D2, D3, D4, D5** (source: `runtimeenv/adapter.go`, `runtimeenv/env.go`,
`internal/daemon/config.go`, `internal/daemon/execenv/cline_home.go`). **D6, D7 not started.**
