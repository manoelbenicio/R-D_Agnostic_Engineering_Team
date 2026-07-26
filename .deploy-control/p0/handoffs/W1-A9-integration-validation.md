# W1+A9 — Serial Integration & Focused-Validation Matrix (PREFLIGHT, read-only)

- agent: `Opus48#B`
- pane: `w6:p2`
- lane: `W1+A9`
- task: `P0-W1-PREFLIGHT`
- phase: `PREFLIGHT_AND_GAP_MATRIX` — **product source is READ-ONLY; no source edits, no test-execution campaign in this phase**
- check-in record: `.deploy-control/p0/checkins/Opus48-B__P0-W1-PREFLIGHT__20260721T223103Z.json`
- output lock (only mutable file): `.deploy-control/p0/handoffs/W1-A9-integration-validation.md`
- authored: 2026-07-21 (UTC), on branch `integration/dev-transition-candidate-20260719`
- covers: `5.6`, `5.7`, `5.8`, `8.1`, `8.2` (Main-Brain-owned integration + focused validation only)

> This document is a read-only integration plan and focused-check matrix. It does not implement,
> does not run a validation campaign, and does not authorize any live run. It hands W1 an exact
> serial order and A9 an exact delta→command→evidence table to execute **after** source locks are
> frozen and lanes deliver diffs. RouteModel IDs are owned by A3 (GLM/Kimi) and A4 (Opus48); this
> plan consumes them, it does not invent them.

---

## 1. Tool preflight — exact versions/paths/results

Run on host `ip-172-31-30-9.sa-east-1.compute.internal`, `HERDR_ENV=1`, pane `$HERDR_PANE_ID=w6:p2`.

| Tool | Command | Result | Status |
|---|---|---|---|
| git | `git --version` | `git version 2.50.1` | OK |
| python3 | `python3 --version` | `Python 3.9.25` | OK |
| ripgrep | `rg --version` | `ripgrep 15.2.0 (rev e89fff89ac)` | OK |
| OpenSpec | `openspec --version` | `1.4.1` | OK |
| Node | `node --version` | `v22.23.1` | OK |
| pnpm | `cd multica-auth-work && pnpm --version` | `10.28.2` (via Corepack download of pnpm 10.28.2) | OK — pin resolved |
| Go | see note G-1 | `go version go1.26.1 linux/amd64` at `/home/ec2-user/goroot/go/bin/go` | OK — version pin satisfied |
| gofmt | `/home/ec2-user/goroot/go/bin/gofmt` present, exec bit set | (binary present) | OK |
| Docker | `docker` | absent | Not required this phase (per `ASSIGNMENTS.md`); any later container-only check BLOCKS until provisioned |

### Note G-1 — Go toolchain path (RESOLVED; canonical path confirmed by owner)

- **FINAL canonical path (owner-confirmed 2026-07-21T22:36Z):** `/home/ec2-user/goroot/go/bin/go`
  and `/home/ec2-user/goroot/go/bin/gofmt` → `go version go1.26.1 linux/amd64`. `ASSIGNMENTS.md` and
  `control.json` corrected upstream to this path; the prior `.local` path is superseded and ignored.
- History (for provenance): the earlier documented path
  `/home/ec2-user/.local/toolchains/go1.26.1/bin/go` did not exist on this filesystem (verified by
  direct `ls`, exit 127; `/home/ec2-user/.local/toolchains` absent). This agent's `$HOME` is a
  credential slot (`/home/ec2-user/.agent-cred-homes/slots/slot-107/home`), so any `$HOME/.local/...`
  reference also failed. That discrepancy is now resolved by the owner correction above.
- **Action for W1/A9:** invoke Go by absolute path `/home/ec2-user/goroot/go/bin/go` (do not rely on
  `$PATH`; `go` is not on `$PATH` and `$GOROOT` is unset). No credential-slot/provider homes were
  searched to establish or confirm this.

---

## 2. Git / build topology (read-only inspection)

- branch: `integration/dev-transition-candidate-20260719`
- HEAD: `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- recent commits: `autosave(pre-shutdown)` series (22:00–22:27Z) — no feature commits pending.
- working tree (`git status --porcelain`, 5 entries) — **control-plane only, zero product source dirty**:
  - ` M .deploy-control/p0/control.json`
  - ` M .deploy-control/p0/events.jsonl`
  - `?? .deploy-control/p0/ASSIGNMENTS.md`
  - `?? .deploy-control/p0/checkins/Codex56-A__P0-ROUTE-FREEZE__20260721T223004Z.json`
  - `?? .deploy-control/p0/checkins/Opus48-C__P0-ANTIGRAVITY-EQUIVALENCE__20260721T222949Z.json`
- Go module: `github.com/multica-ai/multica/server` (`multica-auth-work/server/go.mod`, `go 1.26.1`).
- Build cleanliness for integration: the product tree is clean at HEAD, so any diff W1 integrates is
  attributable solely to a lane delta (no pre-existing uncommitted product edits to disentangle).

---

## 3. Exact W1 hotspots (single-editor; frozen by FILE_OWNERSHIP D-V3-29 P0 override)

All present on disk (verified). Paths relative to `multica-auth-work/server/`:

| Hotspot | Present | Role in P0 |
|---|---|---|
| `internal/daemon/daemon.go` | yes (~199 KB) | central lifecycle/admission/launch wiring |
| `internal/daemon/config.go` | yes (~54 KB) | neutral/gateway config + alias precedence |
| `internal/daemon/health.go` | yes (~11 KB) | readiness/liveness diagnostics (gateway-aware) |
| `internal/daemon/brain_integration.go` | yes (~20 KB) | Brain↔daemon integration seam; consumes CLIKind/RouteModel |
| `cmd/multica/cmd_daemon.go` | yes | command/entrypoint |
| `pkg/agent/models.go` | yes | model/CLIKind surface (dep changes W1-only) |
| `go.mod` | yes | dependency changes W1-only |
| escalated shared files | n/a | any file two lanes would edit → escalate to W1, serialize |

Non-W1 (adjacent) ownership referenced by this plan:
- **R1 (Cline)** owns `internal/daemon/runtimeenv/**` Cline paths — notably
  `internal/daemon/runtimeenv/cline.go` (+ `cline_test.go`, `model.go`, `model_test.go`). The
  contract already declares `Cline → Kimi-K2.7` and `Cline → GLM-5.2` as the only Agent-Brain-selectable
  Cline routes and marks the NVIDIA namespace as OmniRoute-owned/never Brain-selectable
  (`ErrClineRouteNotAgentBrainSelectable`). W1 does **not** edit these; it integrates R1's delta.
- **W2 (gateway)** owns `internal/daemon/gateway/**` (registry/projection/model_projection/profiles)
  where RouteModel projection lives. RouteModel IDs are frozen by **A3/A4**, not W1.
- **Brain contracts** `internal/daemon/brain/**` (identity/compatibility/registry) are W1-owned per
  FILE_OWNERSHIP but under a frozen `agent-brain.v1` contract — touch only if a P0 gap proves it.

---

## 4. Serial integration dependency order (W1)

Fixed order (PROTOCOL Phase C). W1 integrates **one lane at a time**, running only the focused checks
for that delta before proceeding. No concurrent merges.

1. **Production integrity** (A7 frontend / A8 backend) — remove reachable mocks/fake-success/QA
   routes/placeholders/demo persistence first, so later route work integrates against a clean base.
   *Rationale:* if a residual synthetic-success path exists, a later Cline/Opus48 live acceptance
   could pass falsely. Integrity precedes routing.
2. **Main Brain core / lifecycle** (A6 gap matrix → C deltas) — Kanban→admission→workspace→launch→
   events→terminal result/cancel/cleanup gaps that are Brain-owned. Integrated at the hotspot seam
   (`daemon.go`, `brain_integration.go`) before any route rides on it.
3. **Cline routes** (A1 base + A2 mapping + A3 IDs) — `Cline → GLM-5.2` (5.6) then the incremental
   `Cline → Kimi-K2.7` (5.7) on the same base. Depends on core lifecycle being integrated and on A3's
   frozen RouteModel IDs.
4. **Kiro/Opus48** (A4 route + A5 Antigravity equivalence) — Opus48 via accepted Anthropic frontend +
   exact registry RouteModel (5.8); Antigravity reused when A5 proves equivalence (no rerun).

Dependency notes:
- 5.7 is an **increment on 5.6's** integrated base (shared Cline contract) — never a second Cline
  implementation.
- 5.8 Opus48 is independent of the Cline base but still sits behind core lifecycle integration.
- Each step's focused checks (Section 5) must be green before the next lane is merged.

---

## 5. Minimal focused command matrix per delta (A9 executes AFTER integration)

Go invoked by absolute path per Note G-1. `-count=1` defeats stale cache so a check reflects the
integrated delta. **No broad regression, no second QA, no repeated green check on the same build.**
Commands are the *smallest set representative of the changed behavior*; A9 deduplicates equivalents.

| # | Delta (owner) | Focused command(s) | Package under test | Closes/validates |
|---|---|---|---|---|
| D1 | Production integrity backend (A8) | `/home/ec2-user/goroot/go/bin/go build ./...` then `go vet ./internal/daemon/...` (server dir) | affected backend pkgs | P0-support integrity |
| D1f | Production integrity frontend (A7) | `cd multica-auth-work && pnpm --filter <changed pkg> test` + `pnpm --filter <changed pkg> typecheck` (exact filter named at freeze) | changed web/mobile/desktop pkg | P0-support integrity |
| D2 | Main Brain lifecycle/core (C) | `go test -count=1 ./internal/daemon/ -run '<changed-lifecycle-tests>'` (e.g. `BrainIntegration`, admission/launch/cancel/cleanup) | `internal/daemon` (`brain_integration_test.go`, `daemon_test.go`) | 8.2 lifecycle where changed |
| D2b | Brain contract (if touched) | `go test -count=1 ./internal/daemon/brain/...` | `internal/daemon/brain` | contract preserved |
| D3 | Cline base + GLM-5.2 (A1/A2/A3, 5.6) | `go test -count=1 ./internal/daemon/runtimeenv/ -run 'Cline|Model'` | `internal/daemon/runtimeenv` (`cline_test.go`, `model_test.go`) | 5.6 config/materialization/fail-closed |
| D3b | RouteModel projection for GLM/Kimi (A3 → W2) | `go test -count=1 ./internal/daemon/gateway/ -run 'Projection|Registry|Model'` | `internal/daemon/gateway` | 8.1 model/protocol on changed route |
| D4 | Cline → Kimi-K2.7 increment (5.7) | reuse D3/D3b run for the same package after Kimi ID wired (no new campaign) | `runtimeenv` + `gateway` | 5.7 (increment on 5.6 base) |
| D5 | Opus48 Anthropic route (A4, 5.8) | `go test -count=1 ./internal/daemon/runtimeenv/ -run 'Assert|Model'` + `go test -count=1 ./internal/daemon/gateway/ -run 'Projection|Profiles'` | `runtimeenv` + `gateway` | 5.8 route selectability/protocol |
| D6 | Antigravity (A5) | **none** if A5 = REUSE (equivalent evidence). Only if STALE: the single minimal scenario A5 names. | n/a / A5-named | 5.8/8.1/8.2 reuse |
| D7 | Entrypoint/config wiring (W1 hotspot merges) | `go test -count=1 ./internal/daemon/ -run 'Config|Health'` + `go build ./cmd/multica/` | `internal/daemon`, `cmd/multica` | integration soundness |
| D8 | Full integrated compile gate (once, end of serial merge) | `/home/ec2-user/goroot/go/bin/go build ./...` (server dir) | whole server module | integrated build green |

A9 rules (from prompt): receive changed behavior + proposed checks from each producer; dedupe
equivalent commands; record build/config/commit + result per row; return failures to the correct
owner. A9 never launches a live run.

---

## 6. Live-run token prerequisites (one run per changed/unproven route family)

Per `PROTOCOL.md` §Live-run token and `control.json.live_runs` (all currently
`authorized=false`). A live Kanban→terminal execution is permitted **only** when ALL hold:

1. `control.json` route-family entry has `live_run.authorized=true` (Principal-issued; not W1/A9).
2. Exact **RouteModel** frozen and provenance-cited: GLM-5.2 & Kimi-K2.7 from **A3**
   (`handoffs/A3-route-freeze.md`); Opus48 from **A4** (`handoffs/A4-opus48.md`). Absent ID = `BLOCKED_EXTERNAL`.
3. Integrated commit/build/config provenance recorded (post-serial-integration HEAD).
4. No prior accepted run for the same family/build/scenario (no duplicate acceptance).
5. Run ID recorded **before** launch; terminal evidence recorded **after**.

Overlap closure: the **same** accepted run closes the overlapping `5.x`, `8.1` and `8.2` requirements
it proves. Families and their gating token keys:
- `cline_glm` → 5.6 (+ 8.1/8.2 overlap)
- `cline_kimi` → 5.7 (+ 8.1/8.2 overlap)
- `opus48` → 5.8 (+ 8.1/8.2 overlap)
- `antigravity` → default `reuse_equivalent_evidence`; **no token** if A5 proves equivalence.

Hard exclusions: no Main Brain token for `8.5–8.7` (OmniRoute-external). No auth/credential/quota/
retry/failover/circuit test on any of these runs — those are OmniRoute-owned.

---

## 7. Proposed zero-overlap source-lock freeze (for Principal to freeze at gate GREEN)

Product source stays read-only until `FLEET_SATURATED=GREEN`. Proposed disjoint source locks for the
implementation phase (pairwise intersection = ∅). Any file two lanes would need → **escalate to W1,
serialize** (never concurrent).

| Lane | Proposed exclusive source lock (server-relative) | Not-touch |
|---|---|---|
| W1 (integrator) | `internal/daemon/{daemon,config,health,brain_integration}.go`, `cmd/multica/cmd_daemon.go`, `go.mod`, `pkg/agent/models.go`, `internal/daemon/brain/**` (only if a proven P0 gap), escalated shared files | other lanes' packages |
| R1 (Cline) | `internal/daemon/runtimeenv/cline.go` (+ `cline_test.go`), new Cline files in same package | hotspots, gateway, catalog |
| R2 (Kiro/Opus48) | exact Anthropic/runtime paths frozen after gap matrix (disjoint from R1); Antigravity evidence-only | hotspots, R1 Cline files |
| A3 (route freeze) | `handoffs/A3-route-freeze.md`; catalog file only on explicit disjoint Principal lock | code/evidence read-only during freeze |
| P (prod integrity) | exact reachable paths locked before edit (A7 web/mobile/desktop; A8 backend/config/deploy outside hotspots) | hotspots, isolated fixtures/guardrails |
| C (lifecycle) | non-hotspot source only after explicit freeze; hotspot gaps escalate to W1 | hotspots directly |
| A9/E (this lane) | `.deploy-control/p0/{handoffs,evidence}/**` only | all product source |

Zero-overlap invariants to verify at freeze (owner: Codex#56#A intersection check):
- W1's hotspot set is disjoint from R1's `runtimeenv/cline*.go` and from gateway (`W2`).
- Shared anchors / entrypoints are W1-serial, never co-edited.
- `pkg/agent/{claude,codex,kimi,nim,antigravity}.go` (R2/coordination) are distinct from
  `pkg/agent/models.go` (W1).

---

## 8. Integration overlap for 5.6–5.8 / 8.1–8.2 (one execution can close many requirements)

These are **not five independent campaigns**. Mapping of a single integrated run/check to the
overlapping requirements it satisfies:

| Route family | Integrated delta | Focused checks (Sec 5) | Live run (Sec 6) | Requirements closed by the same evidence |
|---|---|---|---|---|
| Cline → GLM-5.2 | R1 base + A3 GLM ID, integrated at core seam | D3, D3b | `cline_glm` (1 run) | 5.6; 8.1 (model/protocol on changed route); 8.2 (tools/reasoning/usage/terminal/cancel where changed) |
| Cline → Kimi-K2.7 | Kimi ID increment on same base | D4 (reuse D3/D3b) | `cline_kimi` (1 run) | 5.7; 8.1; 8.2 (only where not already equivalent) |
| Kiro/Opus48 | A4 Anthropic frontend + exact RouteModel | D5 | `opus48` (1 run) | 5.8; 8.1; 8.2 |
| Antigravity | A5 equivalence decision | D6 (none if REUSE) | none if equivalent | 5.8 reuse; 8.1/8.2 reuse of prior accepted evidence |

Overlap rules applied:
- `8.1` = confirm model/protocol/availability only on **changed or unproven** routes; it rides on the
  same integration/registry checks (D3b/D5), not a separate audit.
- `8.2` = tools/reasoning/cancellation/usage/terminal/deterministic-error captured only where evidence
  is not already equivalent; cancellation re-checked only if the Brain lifecycle changed (tie to A6/C).
- `8.3`/`8.4` already accepted (synthetic/reference + northbound telemetry) — not reopened here.
- `8.5–8.7` OmniRoute-external — no Multica lane, QA, or live run.

---

## 9. Blockers / limitations (concrete, with owner + action)

| Item | Type | Owner | Required action |
|---|---|---|---|
| Go toolchain path | RESOLVED (owner-confirmed 2026-07-21T22:36Z) | toolchain owner | canonical path is `/home/ec2-user/goroot/go/bin/go`; ASSIGNMENTS/control.json corrected; prior `.local` path superseded |
| Exact GLM-5.2 & Kimi-K2.7 RouteModel IDs | external blocker for 5.6/5.7 **live**, not for this plan | A3 (`handoffs/A3-route-freeze.md`) | publish one canonical ID + protocol + provenance per route, or `BLOCKED_EXTERNAL` |
| Exact Opus48 AWS RouteModel + accepted Anthropic frontend/CLIKind | external blocker for 5.8 **live** | A4 (`handoffs/A4-opus48.md`) | publish exact registry ID; confirm frontend/CLIKind |
| Antigravity REUSE vs STALE | gating for 5.8 rerun | A5 (`evidence/A5-antigravity-equivalence.md`) | REUSE (no run) or STALE + single minimal scenario |
| Source locks not yet frozen | phase gate | Principal | freeze disjoint locks after `FLEET_SATURATED=GREEN` |
| live_run tokens all `authorized=false` | phase gate | Principal | issue one token per changed/unproven family per Sec 6 |
| Docker absent | environment | infra | provision only if a later check is truly container-only |

No source was edited. No product test was executed. No live run was performed or authorized. This is a
read-only preflight/integration plan only.
