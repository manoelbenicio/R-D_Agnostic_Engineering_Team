# R1 — Test-lock activation & D1 flip map (BLOCKED, read-only)

- agent: `Opus48#A`  ·  lane: `A9-R1`  ·  task: `P0-R1-TEST-LOCK-ACTIVATION`  ·  pane: `w6:p1`
- control lock: `.deploy-control/p0/handoffs/R1-test-lock-activation.md`
- product test locks (exact, disjoint): 5 files in `internal/daemon/runtimeenv/` (see §1)
- status: **CLOSED → transition** — W1 **D1+D2 APPLIED** (adapter.go returns `AdapterReady`+
  `ProtocolOpenAIChat` for `CLIOpenAICompatible`; env.go injects `CLINE_DATA_DIR`+
  `CLINE_OMNIROUTE_API_KEY` trusted-last via `AdapterEnvironment.ClineDataDir`; gofmt clean, package
  build OK per W1 signal). `implementation_authorized=true`; R1 test implementation now authorized
  under the five exact test locks. Superseded by `P0-R1-TEST-IMPLEMENTATION`.
- product tests: **NOT edited** (locked only). No edit, no run, no live run until W1 D1-applied signal.
- supersedes: `R1-test-implementation-readiness.md` (checked out DONE on adjudication).
- created: 2026-07-21T23:11Z (UTC) · updated 23:13Z · gate: `FLEET_SATURATED=GREEN`,
  phase `IMPLEMENTATION_W1_SERIAL`, `implementation_authorized=true` (worker-only: W1).

---

## 1. Locked product test files — existence & zero-overlap (verified 23:11Z)

| Locked file (server-rel `internal/daemon/runtimeenv/`) | Exists? | Lines | Sole lock owner |
|---|---|---|---|
| `adapter_test.go` | **ABSENT — to create** | — | Opus48#A / P0-R1-TEST-LOCK-ACTIVATION |
| `env_test.go` | EXISTS | 373 | Opus48#A / P0-R1-TEST-LOCK-ACTIVATION |
| `model_test.go` | EXISTS | 68 | Opus48#A / P0-R1-TEST-LOCK-ACTIVATION |
| `cline_test.go` | EXISTS | 144 | Opus48#A / P0-R1-TEST-LOCK-ACTIVATION |
| `isolation_g4_test.go` | EXISTS | 402 | Opus48#A / P0-R1-TEST-LOCK-ACTIVATION |

**Zero-overlap proof:** scan of all active (`IN_PROGRESS`/`BLOCKED`) check-in `files_locked` shows
exactly **5** active locks on these targets, **all** held by `Opus48#A / P0-R1-TEST-LOCK-ACTIVATION`.
No other lane locks any of them. (Adjudication note: `W1-source-lock-freeze.md §3.1/§3.3` had
tentatively grouped `env_test.go`/`isolation_g4_test.go` under W1; this Principal adjudication assigns
all five R1 runtimeenv **test** files to A9-R1, so W1 must **not** also lock them — no concurrent edit.)

**Package-compile coupling (unchanged from freeze):** these tests compile in `package runtimeenv`
alongside W1's D1/D2 **source** edits (`adapter.go`, `env.go`). Resolved by **serialization** — W1
applies D1/D2 source first; only then do I edit these locked tests. Never concurrent.

---

## 2. Exact OLD fail-closed assertions that MUST flip with W1 D1

**W1 D1** (`W1-source-lock-freeze.md §3.6`): `runtimeenv/adapter.go` `CredentiallessAdapterContract`
case `CLIOpenAICompatible` (`adapter.go:58-59`): `AdapterFailClosed{GateOpenAICompatibleUnaccepted}`
→ `AdapterReady, Protocol: ProtocolOpenAIChat`. After D1, every assertion that `CLIOpenAICompatible`
is fail-closed becomes **false** and must be updated.

### 2.1 `model_test.go` — `TestCredentialBearingNativeAdaptersFailClosed` (lines 30–51)
- OLD (line 35): table row `{brain.CLIOpenAICompatible, GateOpenAICompatibleUnaccepted}` is asserted
  via `errors.Is(err, ErrAdapterFailClosed)` (:43) and `contract.State != AdapterFailClosed || Gate != gate` (:46).
- FLIP: **remove** the `CLIOpenAICompatible` row from this fail-closed table. Remaining rows
  (`CLIKimi`/`CLINIM`/`CLIAntigravity`) stay fail-closed (regression guard). The positive
  `CLIOpenAICompatible → AdapterReady + ProtocolOpenAIChat` assertion moves to the new
  `adapter_test.go` (T-ADPT-1) per the test contract.
- UNAFFECTED: `TestNativeFallbackIsNeverAutomatic` (:53) uses `CLIAntigravity` only — no change.
- UNAFFECTED: `TestGatewayModelPolicyAcceptsApprovedOmniRouteIDWithoutNativeDiscovery` (:8) —
  Claude/Antigravity route; add a **new** positive `T-MODEL-1` for `CLIOpenAICompatible` + GLM
  `cp/cline-pass/glm-5.2` (does not modify the existing test).

### 2.2 `isolation_g4_test.go` — `TestG4NativeCredentialBearingAdaptersStayFailClosed` (lines 130–166)
- OLD (line 135): table row `{cli: brain.CLIOpenAICompatible, gate: GateOpenAICompatibleUnaccepted}`
  asserted twice: adapter contract fail-closed (:144) **and** `BuildGatewayEnvironment(...)` returns
  `ErrAdapterFailClosed` with `len(environment.Keys()) != 0` == 0 (:162).
- FLIP: **remove** the `CLIOpenAICompatible` row from this "stay fail-closed" table. After D1 (adapter
  ready) + D2 (`env.go` Cline trusted-inject branch), `CLIOpenAICompatible` will (a) return
  `AdapterReady` and (b) **produce** a controlled child environment — so both :144 and :162 no longer
  hold for it. Keep `CLIKimi`/`CLINIM`/`CLIAntigravity` rows (still fail-closed). Positive Cline env
  behavior is covered by the new `env_test.go` T-ENV-* (added under lock), not here.

### 2.3 `env_test.go` — NO existing flip (net-new tests only)
- `rg` shows **no** `CLIOpenAICompatible`/`GateOpenAICompatible`/`FailClosed` reference in `env_test.go`.
  Existing Claude/Codex trusted tests (`...ClaudeAppliesTrustedValuesLast`,
  `...CodexUsesDedicatedKeyName`, `...RejectsNoncanonicalTrustedHomes`) are **unaffected**. The Cline
  env coverage (T-ENV-1..4) is **added** after D2, not a flip.

### 2.4 `cline_test.go` — NO D1 flip (fixture correction only)
- References are to `ClineOpenAICompatibleProviderID` (the `providers.json` provider discriminator
  constant), **not** the adapter gate — unrelated to D1. The only change here is the test-contract
  fixture correction: the Kimi placeholder → `clineKimiRouteModelPENDING` (BLK-KIMI), GLM fixtures
  stay canonical `cp/cline-pass/glm-5.2`.

### 2.5 `adapter_test.go` — NEW file
- Create with **T-ADPT-1** (`CLIOpenAICompatible` → `AdapterReady` + `ProtocolOpenAIChat`, empty Gate)
  and **T-ADPT-2** (`CLIKimi`/`CLINIM`/`CLIAntigravity` stay `*AdapterGateError` — regression guard
  proving D1 did not open native Kimi).

---

## 3. Blocker (verified) & next action

**Blocker:** W1 D1+D2 source not yet landed/greened. `implementation_authorized=true` but the
authorization is **worker-only for Opus48#B/W1** (per `control.json` authorization_note + activation
manifest `W1-source-lock-activation.md`, 16 W1 locks); `Opus48#A/R1 … remain blocked until W1 lands
and greens each coupled source package, then must acquire their disjoint test locks.` At HEAD,
`adapter.go` `CLIOpenAICompatible` still returns `AdapterFailClosed{GateOpenAICompatibleUnaccepted}`
(D1 not applied). Editing these tests now would break compile/pass against the un-flipped source and
violate the serial coupling.

**Owner:** W1 (Opus48#B).

**Next action (on W1 `D1+D2-applied & greened` heartbeat):** update **only** the five locked test
files per §2 — remove the two `CLIOpenAICompatible` fail-closed rows (2.1, 2.2), add `adapter_test.go`
(2.5), add Cline env tests to `env_test.go` (T-ENV-*), add `T-MODEL-1` + Kimi-`t.Skip("BLK-KIMI")` to
`model_test.go`, apply the `cline_test.go` fixture correction — then run **only** the focused package:
`/home/ec2-user/goroot/go/bin/go test -count=1 ./internal/daemon/runtimeenv/ -run 'Adapter|Env|Model|Cline|G4'`
and `gofmt -l` those files. No broad regression, no other package, no live run. Kimi exact-ID edits
remain separately unauthorized (BLK-KIMI).

**Do NOT edit product tests before the W1 D1+D2-applied signal.**

---

## 4. Preflight (unchanged, canonical toolchain)

git `2.50.1` · python3 `3.9.25` · rg `15.2.0` · herdr `0.7.4` (HERDR_ENV=1, `w6:p1`) ·
go `/home/ec2-user/goroot/go/bin/go` = `go1.26.1 linux/amd64` · gofmt present. All green.

---

## 5. Agent status block

- STATUS: CONDITIONALLY BLOCKED (locks held; auth is worker-only=W1; awaiting W1 D1+D2 land+green)
- DELIVERED: 5 exact disjoint R1 test-file locks (zero-overlap verified); D1 flip map with exact
  file:line assertions to change; new-test plan; focused run command.
- FILES: `.deploy-control/p0/handoffs/R1-test-lock-activation.md` (control) + 5 locked product test
  files (locked, **not edited**).
- VALIDATION: none (blocked; no test edited/run/live).
- EVIDENCE: this artifact; assertions cited by file:line (model_test.go:30-51, isolation_g4_test.go:130-166);
  zero-overlap scan (§1); control.json (`implementation_authorized=true`, phase `IMPLEMENTATION_W1_SERIAL`,
  worker-only auth for W1).
- BLOCKERS: W1 D1+D2 source not yet landed/greened; auth is worker-only (W1). Owner W1 (Opus48#B).
  Next = on W1 D1+D2-applied & greened heartbeat, update only the locked tests and run focused runtimeenv tests.
- W1_HANDOFF: my 5 test locks are disjoint from W1's D1/D2 **source** locks (`adapter.go`,`env.go`,…);
  serialize D1/D2 source → then these tests.
