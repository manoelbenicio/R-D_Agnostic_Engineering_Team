# W1 — Final Integration Evidence (P0-W1-FINAL-INTEGRATION-EVIDENCE)

- agent: `Opus48#B` · pane `w6:p2` · lane `W1-EVIDENCE` · task `P0-W1-FINAL-INTEGRATION-EVIDENCE`
- lock (only mutable file): `.deploy-control/p0/handoffs/W1-final-integration-evidence.md`
- check-in: `.deploy-control/p0/checkins/Opus48-B__P0-W1-FINAL-INTEGRATION-EVIDENCE__20260721T235921Z.json`
- repo: branch `integration/dev-transition-candidate-20260719`, HEAD `a6d50986…`; Go `/home/ec2-user/goroot/go/bin/go` (go1.26.1)
- posture: **evidence consolidation only — no source edit, no duplicate test, no live run.** Results below are cited, not re-run.
- upstream: `W1-implementation.md` (D1–D5/D3–D4), `W1-implementation-D6.md` (D6+D7 + final evidence), `D6-prelaunch-adjudication.md`.

## 1. D1–D7 exact changed files (W1-authored)

**W1 SOURCE (7):**
| File | Step(s) | Change (one line) |
|---|---|---|
| `internal/daemon/runtimeenv/adapter.go` | D1 | `CredentiallessAdapterContract(CLIOpenAICompatible)` → `AdapterReady`/`ProtocolOpenAIChat` |
| `internal/daemon/runtimeenv/env.go` | D2, D6-4 | `AdapterEnvironment.ClineDataDir` + `trustedAdapterEntries` Cline case; `ChildEnvironment.clineDataDir` + `BuildGatewayEnvironment` populate |
| `internal/daemon/config.go` | D3, D4 | `agentBrainBuiltInCLIFor` → `{cline,cline}`; `Validate` accepts `CLIOpenAICompatible` |
| `internal/daemon/execenv/cline_home.go` | D5 | `WriteCredentiallessClineConfig` + `prepareCredentiallessClineHome` (0700 dir, 0600 `settings/providers.json`) |
| `internal/daemon/brain_integration.go` | D6 | `buildLaunch` Cline branch (controlled data dir, `NewClineConfigContract`→`WriteCredentiallessClineConfig`, RFC3339 `updatedAt`, empty manifest) |
| `internal/daemon/runtimeenv/assert.go` | D6-1/2/3 | Cline pre-launch: switch accept (CodexConfig==nil), `trustedEntryAllowed` Cline case, `launchRootsAreControlled` `CLINE_DATA_DIR` containment |
| `pkg/agent/models.go` | D7 | GLM catalog ID `cline-pass/glm-5.2` → `cp/cline-pass/glm-5.2` (Kimi `:563`/NIM `:572` untouched) |

**W1 TESTS (2):** `internal/daemon/runtimeenv/assert_test.go` (D6 coupled: accept + containment/traversal/symlink/denied-origin/CodexConfig), `pkg/agent/models_test.go` (D7 lockstep expectation).

**DISJOINT lane tests (NOT W1; consumed green):**
- R1 (Opus48#A, `A9-R1`): `runtimeenv/{adapter_test.go(new),env_test.go,model_test.go,cline_test.go,isolation_g4_test.go}` (D1/D2 synchronized).
- R3 (Opus48#D, `A9-R3`): `internal/daemon/{brain_integration_test.go,config_test.go,execenv/cline_home_test.go}` (D3/D4/D5/D6).

Diffstat (product, all lanes): 15 tracked +622/−29 + new `adapter_test.go`. W1 source is surgical (models.go/models_test.go ±1 line each; adapter.go ±14; env.go +29; assert.go +33; config.go +6; cline_home.go +43; brain_integration.go +30).

## 2. Focused command results (cited; each package validated once, no duplication)

| Delta | Command | Result | Owner |
|---|---|---|---|
| D1/D2 | `go test ./internal/daemon/runtimeenv/ -run 'Adapter|Env|Model|Cline|G4'` (R1 suite) | GREEN | R1 |
| D6 (assert/Cline) | `go test ./internal/daemon/runtimeenv/ -run 'Assert|Cline' -count=1` | **ok 0.006s** | W1 |
| D5 (execenv writer) | `go test ./internal/daemon/execenv/ …` | GREEN 0.005s | R3 |
| D3/D4/D6 (daemon) | `go test ./internal/daemon/ …` (5 tests) | GREEN 0.020s | R3 |
| D7 (catalog) | `go test ./pkg/agent/ -run 'StaticModels|Cline' -count=1` | **ok 0.016s** | W1 |
| build-clean | `go build ./internal/daemon/ ./…/execenv/ ./…/runtimeenv/ ./pkg/agent/` | **all OK** | W1 |
| format | `gofmt -l` (all W1-touched files) | clean | W1 |

## 3. R1 / R3 final decisions (DONE — consumed, not re-run)

Both lanes have completed their final validation against the W1 D1–D7 integrated source:

- **R1 (`A9-R1`, Opus48#A) — DONE, PASS.** Final targeted runtimeenv validation
  `go test ./internal/daemon/runtimeenv/ -run 'Adapter|Env|Model|Cline|G4' -count=1` → **PASS 0.017s, no
  edits required.** R1 inspected the final W1 diff (`runtimeenv/adapter.go`/`assert.go`/`env.go`) and
  confirmed the Cline contract its tests consume is unchanged and `ChildEnvironment.clineDataDir` is
  additive — no contract change to the five R1 tests.
- **R3 (`A9-R3`, Opus48#D) — DONE, NO-REVALIDATION-NEEDED.** Impact grep confirmed its three locked test
  files consume the route as the direct literal `cp/cline-pass/glm-5.2` (e.g. `config_test.go:929`,
  `brain_integration_test.go`) with no `pkg/agent` dependency; W1 D7 (GLM prefix only, Kimi/NIM untouched)
  has **zero impact**. The 7 green R3 tests (`R3-test-implementation.md`) stand as equivalent evidence.

⇒ All W1-owned + R1 + R3 validation is **complete and green**. No W1 pending work remains.

## 4. Remaining external blockers (mapped; none are W1 source work)

| ID | Blocker | Owner | Unblocks |
|---|---|---|---|
| BLK-AVAIL | OmniRoute enriched `/v1/models` rows for `cp/cline-pass/glm-5.2` not published | OmniRoute architect | GLM route admissibility (8.1) + GLM live run |
| BLK-25B | live-provider tests security-stopped until exposed key revoked | Product/OmniRoute owner | all live acceptance runs |
| LIVE-AUTH | `control.json live_runs.cline_glm.authorized=false` (all families false) | Principal Orchestrator | issues the single reserved GLM live-run token |
| BLK-KIMI | exact Cline→Kimi-K2.7 RouteModel undeclared | OmniRoute architect | 5.7 + R1 Kimi one-line fixtures |
| BLK-OPUS48 | exact Opus48-on-AWS RouteModel absent | OmniRoute registry owner | 5.8 (config-only; zero source) |

`8.5–8.7` remain OmniRoute-external (no Multica lane). Antigravity = A5 STALE → one live scenario (gated BLK-25B), no source.

## 5. Status: DONE — W1 final integration evidence COMPLETE

- **W1 delivered:** D1–D7 authorized GLM-ready source implementation — focused-green, build-clean,
  gofmt-clean. R1 final PASS (0.017s, no edits); R3 final NO-REVALIDATION-NEEDED. **All W1 + R1 + R3
  validation is green; no W1 pending work remains.**
- **Only remaining P0 item (NOT W1 work):** the single reserved **GLM live run**
  (`live_runs.cline_glm.authorized=false`; all families false), externally gated by **BLK-AVAIL**
  (OmniRoute enriched `/v1/models` rows) + **BLK-25B** (key-revocation security stop) + a Principal
  live-run token. It closes 5.6/8.1/8.2 (GLM) in one Kanban→terminal acceptance.
- **Owner of the remainder:** Principal Orchestrator (live-run authorization + final acceptance) with
  OmniRoute architect/owner (BLK-AVAIL, BLK-25B). Kimi (5.7) and Opus48 (5.8) stay externally blocked
  on their RouteModel IDs (BLK-KIMI, BLK-OPUS48); neither is W1 source work.
- **Next action:** once BLK-AVAIL + BLK-25B clear, Principal issues the single `cline_glm` live-run token
  and runs the one GLM Kanban→terminal acceptance. No further W1 source edit, duplicate test, or live run.

W1's serial integration and GLM-ready implementation are complete; this evidence hands the sole
remaining step (the externally-gated GLM live run) to the Principal. `live_runs` = all `authorized=false`.

---

## 6. Final validation record refs (exact; completed — not pending)

| Lane | Result | Check-in record | Handoff |
|---|---|---|---|
| R1 (`A9-R1`, Opus48#A) | **DONE — PASS 0.017s, no edits** (5 runtimeenv test contracts pass against final integrated source) | `.deploy-control/p0/checkins/Opus48-A__P0-R1-FINAL-INTEGRATION-VALIDATION__20260721T232302Z.json` | `.deploy-control/p0/handoffs/R1-final-integration-validation.md` |
| R3 (`A9-R3`, Opus48#D) | **DONE — NO-REVALIDATION-NEEDED** (three locked tests use `cp/cline-pass/glm-5.2` literal directly, no `pkg/agent` dependency; 7 green R3 tests equivalent) | `.deploy-control/p0/checkins/Opus48-D__P0-R3-FINAL-INTEGRATION-VALIDATION__20260721T235649Z.json` | `.deploy-control/p0/handoffs/R3-final-integration-validation.md` |

R1/R3 are **completed, not pending**. The earlier "pending R1/R3" wording is superseded by this appendix.

## 7. Terminal state — BLOCKED solely on external GLM live prerequisites

W1 source (D1–D7) + R1 + R3 validations are all green; no W1 or R1/R3 work remains. The single open P0
item is the reserved GLM live run, blocked **only** by external prerequisites:

- `control.json live_runs.cline_glm.authorized = false`
- **BLK-AVAIL** — OmniRoute enriched `/v1/models` rows for `cp/cline-pass/glm-5.2` not published
- **BLK-25B** — live-provider tests security-stopped until the exposed key is revoked

**Owner:** OmniRoute / Product security (BLK-AVAIL, BLK-25B) + Principal Orchestrator (live-run token).
**Next action:** after BLK-AVAIL + BLK-25B clear and the Principal issues the `cline_glm` token, run the
**one** reserved Kanban→terminal GLM acceptance (closes 5.6/8.1/8.2). No W1 source or test action.
