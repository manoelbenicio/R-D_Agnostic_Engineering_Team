# W1 — Source-Lock Activation Manifest (locks acquired; HELD pending implementation authorization)

- agent: `Opus48#B` · pane: `w6:p2` · lane: `W1-LOCKS` · task: `P0-W1-SOURCE-LOCK-ACTIVATION`
- control lock: `.deploy-control/p0/handoffs/W1-source-lock-activation.md`
- check-in: `.deploy-control/p0/checkins/Opus48-B__P0-W1-SOURCE-LOCK-ACTIVATION__20260721T230609Z.json`
- repo: branch `integration/dev-transition-candidate-20260719`, HEAD `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- Go: `/home/ec2-user/goroot/go/bin/go` → `go1.26.1 linux/amd64` (canonical, control.json-confirmed)
- **posture: FLEET_SATURATED=GREEN, but `implementation_authorized=false`. W1 source locks are ACQUIRED
  and HELD (this record stays active). NO product source edit, NO test, NO live run until the Principal
  authorizes implementation.**

---

## 1. Gate provenance

| Signal | Value | Source |
|---|---|---|
| `FLEET_SATURATED` | **GREEN** | independent Kiro audit `2026-07-21T23:02:30Z` (per Principal) |
| `control.json.fleet_saturation_gate.status` | `GREEN` | `control.json` |
| `implementation_authorized` | **false** | `control.json` |
| gate `go_toolchain` | `/home/ec2-user/goroot/go/bin/go (go1.26.1)` | `control.json` |

Predecessor `P0-W1-INTEGRATION` was closed DONE once the gate turned GREEN (evidence:
`W1-integration-readiness.md`). This activation succeeds it and holds the source locks.

---

## 2. Production-integrity inputs consumed (A7 + A8)

| Lane | Verdict | Source edits | Effect on serial queue |
|---|---|---|---|
| A7 (`Agy-P0-A7`, `A7-production-integrity-frontend.md`) | **NO_REACHABLE_RESIDUAL** | none | step 1 (FE integrity) = evidence-only no-op |
| A8 (`Agy-P0-A8`, `A8-production-integrity-backend.md`) | **NO_REACHABLE_RESIDUAL** | none | step 1 (BE integrity) = evidence-only no-op |

A7 proved web/mobile/desktop clean (`parseWithFallback` fails closed; dev tooling `import.meta.env.DEV`
gated; mocks only in `*.test.*`). A8 proved backend/config/deploy clean (JWT default/dev-verification/
local-auth-bypass/localhost-DB all fail-closed by `APP_ENV`/loopback guards; no debug/QA HTTP routes).
**Neither hands any file to W1** → no production-integrity file enters the W1 lock set, and serial
step 1 carries no edit (evidence reused for the P0 "no mocks/placeholders/fake-success" criterion).

A8 awareness item (no action now): `OMNIROUTE_DEV_MODELS_COMPAT` appears in `brain_integration.go` (W1)
and `gateway/model_projection.go` (W2), default-OFF explicit opt-in. W1 may add a production startup
warning during D6 integration **only if** it proves a P0 gap; otherwise NO-EDIT.

---

## 3. W1-exclusive source locks ACQUIRED (existence verified, this record)

Repo-relative; all confirmed present on disk at HEAD `a6d5098`:

| # | Locked file | Role | D-map |
|---|---|---|---|
| 1 | `multica-auth-work/server/pkg/agent/models.go` | Cline GLM catalog row | D7 |
| 2 | `multica-auth-work/server/pkg/agent/nim.go` | NIM-direct default (REVIEW-1) | DEC-NIM |
| 3 | `multica-auth-work/server/pkg/agent/models_test.go` | GLM expectation lockstep | D7-T (W1 self) |
| 4 | `multica-auth-work/server/pkg/agent/nim_test.go` | NIM test (REVIEW-1) | DEC-NIM |
| 5 | `multica-auth-work/server/internal/daemon/daemon.go` | lifecycle spine | A6 (L-spine) |
| 6 | `multica-auth-work/server/internal/daemon/config.go` | `agentBrainBuiltInCLIFor`:141 + `Validate`:179/196 | D3, D4 |
| 7 | `multica-auth-work/server/internal/daemon/health.go` | readiness diagnostics | — |
| 8 | `multica-auth-work/server/internal/daemon/brain_integration.go` | `buildLaunch`:277 (Cline branch) | D6 (L3c) |
| 9 | `multica-auth-work/server/cmd/multica/cmd_daemon.go` | entrypoint | — |
| 10 | `multica-auth-work/server/go.mod` | dep changes (W1-only; likely none) | — |
| 11 | `multica-auth-work/server/internal/daemon/runtimeenv/env.go` | `trustedAdapterEntries`:195 + `AdapterEnvironment.ClineDataDir` | D2 |
| 12 | `multica-auth-work/server/internal/daemon/runtimeenv/adapter.go` | `CredentiallessAdapterContract`:58 gate flip | D1 |
| 13 | `multica-auth-work/server/internal/daemon/runtimeenv/policy.go` | env deny-list (CLINE trusted-inject) | D2 (support) |
| 14 | `multica-auth-work/server/internal/daemon/runtimeenv/home.go` | task-home manifest guard | D2 (support) |
| 15 | `multica-auth-work/server/internal/daemon/execenv/cline_home.go` | add `WriteCredentiallessClineConfig` (file EXISTS — reconcile) | D5 |
| 16 | `multica-auth-work/server/internal/handler/agent_thinking_test.go` | Kimi literal (coupled) | D7/BLK-KIMI |

Excluded from lock (intentional): `internal/daemon/brain/**` (NO-EDIT unless a proven gap; A6
guardrail — do not re-route the live path through the test-only `brain.Coordinator`).

---

## 4. Pairwise zero-overlap verification against active records

Active records at activation (status IN_PROGRESS/BLOCKED) and their `files_locked` — **all lock only
`.deploy-control/**` artifacts; none locks any product source**:

| Active record | Locked artifact |
|---|---|
| `Agy-P0-A7` / P0-TERMINAL-UI-EVIDENCE | `evidence/A7-terminal-ui-evidence.md` |
| `Agy-P0-A8` / P0-TERMINAL-BACKEND-EVIDENCE | `evidence/A8-terminal-backend-evidence.md` |
| `Codex56#A` / P0-GLM-LIVE-ACCEPTANCE | `evidence/R1-glm-live-acceptance.md` |
| `Codex56#B` / P0-OPUS48-LIVE-ACCEPTANCE | `evidence/R2-opus48-live-acceptance.md` |
| `Opus48#A` / P0-CLINE-TEST-IMPLEMENTATION | `handoffs/R1-test-implementation-readiness.md` |
| `Opus48#D` / P0-LIFECYCLE-TEST-IMPLEMENTATION | `handoffs/R3-test-implementation-readiness.md` |
| `Opus48#C` / P0-TRACEABILITY | `handoffs/A10-traceability.md` |
| `Opus48-Kiro` / P0-SUPERVISION | `kiro-audit.jsonl` |

- W1 product-source lock set (16 files above) ∩ every active record's locks = **∅** (verified: no active
  lock references any `multica-auth-work/**` path). The `p0_control.py check-in` overlap guard **accepted**
  the activation, independently proving zero intersection.
- The **R1 test lane** (`Opus48#A`, artifact `R1-test-implementation-readiness.md`) and **R3 test lane**
  (`Opus48#D`, artifact `R3-test-implementation-readiness.md`) are the disjoint test-authoring lanes for
  §6. Their **future** product-test locks must be disjoint from W1's source locks (they are — see §6).

---

## 5. Serial D1–D7 integration order (W1 source; execute ONLY after authorization)

One edit at a time; run the focused package check (`/home/ec2-user/goroot/go/bin/go test -count=1 <pkg>`
+ `gofmt -l`) after each; never concurrent. Order per freeze §3.6 / R1 §8:

0. **Production integrity** — evidence-only no-op (A7/A8 = NO_REACHABLE_RESIDUAL). No edit.
1. **D1** `runtimeenv/adapter.go` `CredentiallessAdapterContract(CLIOpenAICompatible)`: fail-closed →
   `AdapterReady, ProtocolOpenAIChat`. Keep `CLIKimi`/`CLINIM`/`CLIAntigravity` fail-closed.
2. **D2** `runtimeenv/env.go` `trustedAdapterEntries` add `CLIOpenAICompatible` case (trusted-inject
   `CLINE_DATA_DIR`+`CLINE_OMNIROUTE_API_KEY`) + add field `AdapterEnvironment.ClineDataDir`. (policy.go/
   home.go touched only as needed to keep the carrier out of the task-home manifest.)
3. **D5** `internal/daemon/execenv/cline_home.go` **add** `WriteCredentiallessClineConfig` +
   `prepareCredentiallessClineHome` (mirror `codex_home.go:87`; carrier `<clineDataDir>/settings/providers.json` 0600).
   ⚠ file EXISTS (legacy `prepareClineHome`) — add functions, do not duplicate.
4. **D6** `brain_integration.go` `buildLaunch`:277 add `CLIOpenAICompatible` sibling branch (create
   task-scoped dir, `NewClineConfigContract`, `WriteCredentiallessClineConfig`, set `Adapter.ClineDataDir`,
   empty manifest); decide `updatedAt` format (recommend injectable-clock RFC3339). Extend `AssertPreLaunch`
   to the Cline data-dir layout.
5. **D3+D4** `config.go` `agentBrainBuiltInCLIFor`:141 add `CLIOpenAICompatible→{cline,cline}` +
   `Validate`:196 accept `CLIOpenAICompatible`.
6. **D7** `pkg/agent/models.go:562` `cline-pass/glm-5.2`→`cp/cline-pass/glm-5.2` (GLM only; NOT :563 Kimi,
   NOT :572 NIM) + `models_test.go:172` lockstep (W1 self-authored).
7. **GLM live run** (single reserved token, `control.json live_runs.cline_glm`) → closes 5.6/8.1/8.2 (GLM).
8. On **BLK-KIMI** clear: Kimi one-line fixtures (R1 K2/K3) → single Kimi live run (5.7).

**Opus48 (5.8):** R2-proven **config-only**, ZERO source edit — `AGENT_BRAIN_CLI_KIND=claude-code` +
`AGENT_BRAIN_ROUTE_MODEL=<OPUS48_AWS_ID>`; not in this source queue. **Antigravity:** A5=STALE, one live
scenario only, no source edit.

---

## 6. Disjoint test handoffs — R1 and R3 (authored AFTER the coupled W1 source edit lands)

W1 self-authors only the lockstep catalog tests it locked (`pkg/agent/models_test.go`,
`pkg/agent/nim_test.go`, `internal/handler/agent_thinking_test.go`). All other new/behavioral tests are
handed to two disjoint lanes. **No test file below is locked by W1** (leaving them free for R1/R3).

### R1 test handoff → `Opus48#A` (`R1-test-implementation-readiness.md`), package `runtimeenv`
| Test file | Covers (R1 §6) | Depends on (W1 edit) |
|---|---|---|
| `internal/daemon/runtimeenv/adapter_test.go` | D1-T: `CLIOpenAICompatible`→`AdapterReady`/`ProtocolOpenAIChat`; Kimi/NIM/Agy still fail-closed | D1 landed |
| `internal/daemon/runtimeenv/env_test.go` | D2-T: `CLINE_DATA_DIR`+secret trusted-last; secret absent from `String()`/`Keys()`, present only in `Exec()`; custom/local set rejected | D2 landed |
| `internal/daemon/runtimeenv/model_test.go` | D6.7-T: `ValidateSelection(CLIOpenAICompatible, cp/cline-pass/glm-5.2, "")` accepted | D1/D2 landed |
| `internal/daemon/runtimeenv/cline_test.go` | existing GLM fixtures (green); Kimi fixture K2 on BLK-KIMI clear | (existing) |

### R3 test handoff → `Opus48#D` (`R3-test-implementation-readiness.md`), packages `execenv` + `daemon`
| Test file | Covers | Depends on (W1 edit) |
|---|---|---|
| `internal/daemon/execenv/cline_home_test.go` (**new**) | D5-T: `WriteCredentiallessClineConfig` writes 0600 `settings/providers.json` under 0700 dir; rejects empty/oversized/relative/root | D5 landed |
| `internal/daemon/config_test.go` | D3-T/D4-T: `agentBrainBuiltInCLIFor(CLIOpenAICompatible)`=={cline,cline}; `Validate` accepts it, still rejects unknown | D3/D4 landed |
| `internal/daemon/brain_integration_test.go` | D6-T: `buildLaunch(CLIOpenAICompatible)` writes carrier to `CLINE_DATA_DIR`, empty manifest, `AssertPreLaunch` ok with `CodexConfig==nil`; **L7b**: controlled `agent-brain-home` reclaimed with env root | D6 landed |

### Mandatory package-coupling safeguards (both handoffs)
1. **Serialization:** W1 lands + greens the source edit for a package **before** R1/R3 author that
   package's test. W1 never edits the handed-off test files; R1/R3 never edit W1 source files. Enforced by
   per-file locks (disjoint) + this serial order — no concurrent same-compile-unit edits.
2. **No new `TestMain`** and no side-effecting package `init()` in any handed-off test (packages
   `runtimeenv`, `daemon`, `execenv` currently declare their own single/zero `TestMain`; do not add one).
3. Tests target the **frozen span/helper contract** only; no import cycles; must not force changes to
   `adapter.go`/`env.go`/`config.go`/`brain_integration.go`/`cline_home.go`.
4. `daemon`-package tests (`config_test.go`, `brain_integration_test.go`) compile alongside W1's
   `daemon.go`/`config.go`/`brain_integration.go`; R3 authors them only after those source edits are
   integrated and green (serial), targeting the frozen behavior.

**Overlap proof for tests:** W1-locked tests {`models_test.go`,`nim_test.go`,`agent_thinking_test.go`}
∩ R1 tests ∩ R3 tests = ∅ (distinct files, distinct packages for W1 vs R1; daemon/execenv tests belong
solely to R3, never W1). No two lanes lock the same test file.

---

## 7. Blockers (report; not guessed)

| ID | Blocker | Owner | Required action |
|---|---|---|---|
| BLK-AUTH | `implementation_authorized=false` | **Principal Orchestrator** | after verifying this activation, set authorization; then W1 executes §5 D1–D7 |
| BLK-KIMI | exact Cline→Kimi-K2.7 RouteModel undeclared | OmniRoute architect (via Principal) | publish exact versioned Kimi id → unblocks 5.7 + R1 K2/K3 fixtures |
| BLK-OPUS48 | exact Opus48-on-AWS RouteModel absent | OmniRoute registry owner | publish anthropic-messages Opus48 row → unblocks 5.8 config |
| BLK-AVAIL | enriched OmniRoute registry rows (GLM/Kimi) unpublished | OmniRoute architect | publish enriched `/v1/models` rows → route admissibility |
| BLK-25B | live-provider tests security-stopped until exposed key revoked | Product/OmniRoute owner | confirm key revocation → unblocks all live runs |

GLM source path (D1–D7) is **ready to execute the instant BLK-AUTH clears**; its live run additionally
needs BLK-AVAIL + BLK-25B.

---

## 8. Summary

- FLEET_SATURATED **GREEN**; `implementation_authorized=false`. **16 W1-exclusive source/test locks
  ACQUIRED and HELD** (existence-verified); pairwise overlap with all 8 active records = ∅ (tool-enforced).
- A7/A8 = NO_REACHABLE_RESIDUAL → production-integrity is evidence-only, no edit, no lock.
- Serial **D1–D7** source order frozen (W1); Opus48 = config-only; Antigravity = one live scenario.
- Disjoint test handoffs defined: **R1** (`runtimeenv` behavioral) and **R3** (`execenv`+`daemon`
  materialization/config/integration), with mandatory serialization + no-TestMain safeguards; W1 self-
  authors only the 3 locked catalog/handler tests.
- **W1 holds the locks and takes no further action** (no source edit, no test, no live run) until the
  Principal authorizes implementation (BLK-AUTH). This record remains active (BLOCKED) to retain the locks.
