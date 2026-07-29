# A9-R3 — R3 lifecycle focused-test implementation readiness (BLOCKED)

- agent: Opus48#D · lane: A9-R3 · task: P0-LIFECYCLE-TEST-IMPLEMENTATION
- pane: `w8:p2` (HERDR_ENV=1) · git HEAD: `a6d5098`
- check-in: `.deploy-control/p0/checkins/Opus48-D__P0-LIFECYCLE-TEST-IMPLEMENTATION__20260721T224532Z.json`
- output lock (only mutable file): `.deploy-control/p0/handoffs/R3-test-implementation-readiness.md`
- upstream contract: `.deploy-control/p0/handoffs/R3-lifecycle-focused-tests.md`
- status: **BLOCKED** — no product edits performed. This lane implements the R3 focused *lifecycle*
  tests (distinct from R1's Cline **unit** tests in `runtimeenv/cline_test.go`) once activation completes.
- gate update (2026-07-21T23:03Z): **`FLEET_SATURATED` is now GREEN**, but **W1 source-lock activation
  is not yet complete** and **`implementation_authorized=false`**. Remains BLOCKED pending W1's activation
  manifest + explicit authorization. Still no source edit/test.

## 0. Tool / Go preflight evidence (canonical path)

| Tool | Command | Result |
|---|---|---|
| go | `/home/ec2-user/goroot/go/bin/go version` | `go version go1.26.1 linux/amd64` |
| gofmt | `/home/ec2-user/goroot/go/bin/gofmt` | present |
| git | `git --version` | 2.50.1 (HEAD `a6d5098`) |
| python3 | `python3 --version` | 3.9.25 |
| rg | `rg --version` | 15.2.0 |

Ready to run, once unblocked (targeted, `-count=1`, no `./...`):
```bash
GO=/home/ec2-user/goroot/go/bin/go ; cd multica-auth-work/server
$GO test ./internal/daemon/ -run 'BuildLaunchOpenAICompatible|AgentBrainSyntheticChildCline|AdmitTaskFailsClosedForUnacceptedClineAdapter|AdmitTaskRejectsNVIDIAOwnedClineRoute|GCReclaimsControlledAgentBrainHome|BuildLaunchClineReleasesCapacity' -count=1
$GO test ./internal/daemon/runtimeenv/ -run 'AssertPreLaunch.*Cline|WriteCredentiallessCline' -count=1
```

## 1. Scope of THIS lane (disjoint from R1)

Own implementation of the R3 focused **lifecycle** tests only:
- 2.3 `TestBuildLaunchOpenAICompatibleWiresClineDataDirAndSecret`, `TestBuildLaunchOpenAICompatibleChildIsolation` + `TestAgentBrainSyntheticChildCline` — `internal/daemon/brain_integration_test.go` (pkg `daemon`).
- 2.4 `TestAssertPreLaunchAcceptsControlledClinePlan`, `TestAssertPreLaunchRejectsClineDataDirOutsideOrTraversal`, `TestAssertPreLaunchRejectsClineCredentialArtifact` — `internal/daemon/runtimeenv/assert_test.go` (pkg `runtimeenv`).
- 2.5 `TestAdmitTaskFailsClosedForUnacceptedClineAdapter`, `TestAdmitTaskRejectsNVIDIAOwnedClineRoute` — `brain_integration_test.go`.
- 2.6 `TestGCReclaimsControlledAgentBrainHomeWithEnvRoot` — `internal/daemon/gc_test.go` (pkg `daemon`).
- 2.7 `TestBuildLaunchClineReleasesCapacityOnPreLaunchFailure` — `brain_integration_test.go`; cancel/terminal/usage are **reused as equivalent evidence**, not re-implemented.

**Explicitly NOT this lane:** R1's `runtimeenv/cline_test.go` unit assertions (`NewClineConfigContract`/
`ValidateClineConfigBytes`) and R1's product deltas D1–D7. Those are R1/A1. 2.2 (`WriteCredentiallessClineConfig`
writer test) is a boundary case — see §3 decision.

## 2. Why BLOCKED (verified, not assumed)

1. **`FLEET_SATURATED` is GREEN (2026-07-21T23:03Z), but W1 source-lock activation is not complete
   and `implementation_authorized=false`.** The gate opening is necessary but not sufficient: the
   Principal/W1 must still publish the W1 **activation manifest** (frozen exact locks + edit tokens)
   and set implementation authorization before any lane edits source or tests.
2. **W1's freeze is a PREFLIGHT plan, not issued locks.** `W1-source-lock-freeze.md` states
   "product source READ-ONLY … This is a plan, not an edit"; the Principal has **not** frozen locks
   or issued edit tokens.
3. **The exact test files this lane needs are currently W1-assigned.** Per the freeze table:
   `brain_integration_test.go`/`daemon.go`/`brain_integration.go` → **F7/F8** (W1); the controlled-home
   reclamation test (`gc_test.go`/`brain_integration_test.go`) → **F-NEW-2** (W1); `runtimeenv/assert.go`
   + `env.go`/`adapter.go`/`home.go` → **F14–F17** (W1 serial). No **disjoint** lock is available for
   A9-R3 to acquire without a Principal carve-out.
4. **Product symbols under test do not exist yet.** Verified `WriteCredentiallessClineConfig` is
   **ABSENT** in `multica-auth-work/server`; `adapter.go` still returns `AdapterFailClosed` for
   `CLIOpenAICompatible` (freeze §2, D1 pending). The tests would not compile/pass until W1 lands
   D1/D2/D5/D6 and A3 freezes the Kimi RouteModel (BLK-KIMI) / OmniRoute publishes enriched rows (BLK-AVAIL).

## 3. Ownership decision required from Principal/W1 (before unblock)

R3's daemon-package lifecycle tests share a **Go compile unit** with W1's source edits
(`brain_integration_test.go` compiles in `package daemon` alongside `brain_integration.go`/`daemon.go`;
`assert_test.go` in `package runtimeenv` alongside `assert.go`). Two clean options — Principal/W1 rule:

- **Option A (test carve-out):** Principal grants A9-R3 exclusive locks on the **test files**
  (`brain_integration_test.go`, `gc_test.go`, `runtimeenv/assert_test.go` Cline cases) while W1 keeps the
  **non-test source**. Coupling is resolved by **serialization** (W1 lands source first; A9-R3 then adds
  tests) exactly as the Wave-B.1 test-ownership precedent did — never concurrent.
- **Option B (W1 authors, A9-R3 validates):** W1 authors these tests inline with its source edits;
  A9-R3 runs the §0 targeted commands, deduplicates, and records delta→command→result (pure A9 role,
  no source edit). This matches the A9 charter ("focused checks, not a second QA").

A9-R3 does not self-assign; it awaits the ruling. Either way it authors **no product source** and
runs **no broad regression / no live run**.

## 4. Unblock conditions (all required)

1. `FLEET_SATURATED=GREEN` — **met (2026-07-21T23:03Z)**. Remaining: **W1 source-lock activation
   complete** (activation manifest published) and **`implementation_authorized=true`** from Principal/W1.
2. Principal freezes locks and rules Option A vs B (§3).
3. W1 lands the product deltas the tests bind to: D1 (`adapter.go` → `AdapterReady`), D2
   (`env.go` `ClineDataDir` + trusted inject), D5 (`execenv/cline_home.go WriteCredentiallessClineConfig`),
   D6 (`brain_integration.go buildLaunch` Cline branch), + `AssertPreLaunch` Cline layout.
4. A3 GLM RouteModel is frozen (met: `cp/cline-pass/glm-5.2`); **Kimi remains BLK-KIMI** → Kimi-parameterized
   cases stay HOLD; GLM-parameterized cases can proceed. BLK-AVAIL gates the live admission side only
   (unit tests use the synthetic gateway, unaffected).

On unblock: acquire the disjoint test-file lock(s) per the ruling, implement **only** the §1 tests
(parameterized over `clineAgentBrainRoutes` so GLM proceeds and Kimi attaches on BLK-KIMI clear), run
the §0 targeted commands, `gofmt -l` clean, record results, check out with evidence. No product source
beyond the granted test files; no second live acceptance.

## 5. Blocker record

- blocker: `W1 source-lock activation not yet complete; implementation_authorized=false`
- owner: `Opus48#B/W1 + Principal Orchestrator`
- next action: `consume W1 activation manifest, acquire exact disjoint R3 lifecycle test locks, implement only R3 focused tests after explicit authorization`
- gate: `FLEET_SATURATED=GREEN` (met 2026-07-21T23:03Z); authorization + activation still pending.
- also gating (external, not this lane's work): BLK-KIMI (Kimi RouteModel), BLK-AVAIL (enriched rows) — GLM path unaffected for unit tests.

Updated via `p0_control.py block`. No product edits, no source/test work, while BLOCKED.
