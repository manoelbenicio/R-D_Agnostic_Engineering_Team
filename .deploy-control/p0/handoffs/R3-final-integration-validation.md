# A9-R3 — R3 final integration validation (BLOCKED on W1 D7/final)

- agent: Opus48#D · lane: A9-R3 · task: P0-R3-FINAL-INTEGRATION-VALIDATION
- pane: `w8:p2` (HERDR_ENV=1)
- check-in: `.deploy-control/p0/checkins/Opus48-D__P0-R3-FINAL-INTEGRATION-VALIDATION__20260721T235649Z.json`
- lock (only mutable file): `.deploy-control/p0/handoffs/R3-final-integration-validation.md`
- upstream: `R3-test-implementation.md` (7 R3 tests green), `W1-source-lock-freeze.md §3.6` (D7).
- posture: **BLOCKED, no source inspected, no test run.** Scope is at most **one** targeted re-run,
  and only if W1's D7/final diff changes a route fixture my three tests consume.

## 0. Preflight

`/home/ec2-user/goroot/go/bin/go version` → `go version go1.26.1 linux/amd64`. git 2.50.1 · python3 3.9.25 · rg 15.2.0.

## 1. Conditional trigger (what would require a re-run)

My three R3 test files pin the GLM route literal and Cline mapping:
- `config_test.go` and `brain_integration_test.go` use RouteModel `cp/cline-pass/glm-5.2` and the
  `cline`/`cline` built-in mapping (D3/D4/D6 tests);
- `execenv/cline_home_test.go` (D5 writer) is **fixture-independent** (schema-agnostic synthetic bytes),
  so no route literal affects it.

W1 **D7** (`pkg/agent/models.go` GLM row + `models_test.go`, `cline-pass/glm-5.2` → `cp/cline-pass/glm-5.2`)
is already reflected by my tests' literal. A re-run is warranted **only if** the D7/final diff:
1. changes the GLM route literal away from `cp/cline-pass/glm-5.2`, or
2. changes the Cline `CLIKind`/provider/executable mapping (`cline`/`cline`), or
3. changes `CLINE_DATA_DIR`/`CLINE_OMNIROUTE_API_KEY`/carrier path consumed by the D6 assertions.

If the D7/final diff touches none of the above (e.g., Kimi-only, catalog-label-only, or unrelated files),
**no re-run** — the existing green R3 evidence (`R3-test-implementation.md`) stands as equivalent.

## 2. The single targeted validation (run once on signal, only if triggered)

```bash
GO=/home/ec2-user/goroot/go/bin/go ; cd multica-auth-work/server
$GO test ./internal/daemon/execenv/ -run 'WriteCredentiallessCline|PrepareClineHome' -count=1
$GO test ./internal/daemon/ -run 'AgentBrainBuiltInCLIForAcceptsOpenAICompatible|AgentBrainIntegrationConfigValidateAcceptsOpenAICompatibleCline|BuildLaunchOpenAICompatible|AgentBrainSyntheticChildCline' -count=1
```
Plus `gofmt -l` on the three files if any test literal needs updating (test files are already locked to
this lane). No `./...`, no `-race` sweep, no broad regression, no duplicate live run. If a literal changed,
update only the affected test constant in the locked files, re-run the above once, record delta→result.

## 3. Blocker record

- blocker: `W1 final D7/integration heartbeat not received; final route-fixture diff unknown`
- owner: `W1 (Opus48#B)`
- next action: `on W1 final D7 heartbeat, diff the final route fixtures; if it changes the GLM literal
  (cp/cline-pass/glm-5.2), the cline/cline mapping, or the CLINE_DATA_DIR/secret/carrier path consumed by
  the three R3 tests, run the §2 targeted commands once and update only the affected locked-test literal;
  otherwise record NO-REVALIDATION-NEEDED citing R3-test-implementation.md as equivalent evidence`
- also gating: BLK-KIMI (Kimi held) — Kimi fixtures out of scope until its RouteModel is frozen.

Registered via `p0_control.py block`. No source inspected, no test run, no edits while BLOCKED.

## 4. FINAL DECISION (W1 D7 signal received 2026-07-21T23:59Z): NO-REVALIDATION-NEEDED

W1 final D7 = **only** `pkg/agent/models.go` + `pkg/agent/models_test.go` GLM prefix
(`cline-pass/glm-5.2` → `cp/cline-pass/glm-5.2`); Kimi/NIM untouched.

**Impact analysis (grep evidence, HEAD working tree):** the three locked R3 test files do **not**
consume `pkg/agent.clineStaticModels`/catalog and do **not** import `pkg/agent`. They pin the route as a
direct `brain.RouteModel` string literal already equal to the **final** post-D7 value:
- `internal/daemon/config_test.go:929` → `brain.RouteModel("cp/cline-pass/glm-5.2")`
- `internal/daemon/brain_integration_test.go:885` → `config.RouteModel = brain.RouteModel("cp/cline-pass/glm-5.2")`; `:915` → `syntheticClinePlan("cp/cline-pass/glm-5.2")`
- `internal/daemon/execenv/cline_home_test.go` → no route literal (D5 writer is schema-agnostic/fixture-independent)
- `clineStaticModels` occurs only in `pkg/agent/models.go` + `pkg/agent/models_test.go` (the D7 files); **absent** from all three locked files.
- `pkg/agent` import: present in `daemon.go`/`poisoned*.go`/`daemon_test.go`/handlers — **not** in any of the three locked test files.

**Decision:** the D7 GLM-prefix change is confined to the `pkg/agent` catalog and its own test; it does
not touch the GLM literal, the `cline`/`cline` mapping, or the `CLINE_DATA_DIR`/secret/carrier path that
the R3 tests assert. The R3 suite is already green against the **final** `cp/cline-pass/glm-5.2` value.
**No re-run**; the existing evidence in `R3-test-implementation.md` stands as equivalent. Re-running would
duplicate an already-green suite (prohibited). No source edited; no test executed for this decision.
