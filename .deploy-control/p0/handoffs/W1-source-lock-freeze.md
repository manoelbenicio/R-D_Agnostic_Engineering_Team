# W1 — Source-Lock & Integration-Queue Freeze (PREFLIGHT, read-only)

- agent: `Opus48#B`  ·  pane: `w6:p2`  ·  lane: `W1`  ·  task: `P0-W1-SOURCE-FREEZE`
- output lock (only mutable file): `.deploy-control/p0/handoffs/W1-source-lock-freeze.md`
- check-in: `.deploy-control/p0/checkins/Opus48-B__P0-W1-SOURCE-FREEZE__20260721T223659Z.json`
- repo: branch `integration/dev-transition-candidate-20260719`, HEAD `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- Go (corrected canonical): `/home/ec2-user/goroot/go/bin/go` → `go version go1.26.1 linux/amd64`
- phase: `PREFLIGHT_AND_GAP_MATRIX` — **product source READ-ONLY. No edits, no tests, no live run in this doc.**
- purpose: freeze an **exact, disjoint** source-edit ownership map + serial integration queue so that when
  `FLEET_SATURATED=GREEN` the Principal can freeze locks with a machine-auditable zero-overlap proof.

> This is a plan, not an edit. It maps every candidate source file to **one** owner, its blocking
> dependency, the source-edit gate that must be GREEN before the edit, and the focused check that
> validates it. It proves pairwise zero lock overlap and lists every contested file escalated to W1.
> IDs are consumed from A3/A4 (never invented). No auth/credential/quota/failover/Prodex content.

---

## 1. Inputs consumed (artifacts + check-ins, refreshed 2026-07-21T22:41Z)

| Lane | Agent | Task | Status | Own artifact lock | Source contribution |
|---|---|---|---|---|---|
| A1+A2 | Opus48#A | P0-CLINE-FOUNDATION | DONE | `handoffs/A1-A2-cline-foundation.md` | Cline gap matrix C1–C10 + shared-runtimeenv findings |
| A3 | Codex56#A | P0-ROUTE-FREEZE | DONE | `handoffs/A3-route-freeze.md` | RouteModel freeze → substitutions to W1/A1 |
| **R1-DESIGN** | Codex56#A | P0-CLINE-ROUTE-SOURCE-DELTA | DONE | `handoffs/R1-cline-route-source-delta.md` | **exact 7-edit Cline patch plan D1–D7** |
| A4 | Codex56#B | P0-OPUS48-ROUTE | DONE | `handoffs/A4-opus48.md` | zero new source; config-only + watch-item |
| **R2-DESIGN** | Codex56#B | P0-OPUS48-SOURCE-DELTA | DONE | `handoffs/R2-opus48-source-delta.md` | **proven: Opus48 = config-only, ∅ source** |
| A5 | Opus48#C | P0-ANTIGRAVITY-EQUIVALENCE | DONE | `evidence/A5-antigravity-equivalence.md` | evidence-only; no source edit |
| A6 | Opus48#D | P0-LIFECYCLE-GAP | DONE | `handoffs/A6-lifecycle-gap-matrix.md` | one real lifecycle gap **L3c**; rest reuse |
| A10 | Opus48#C | P0-TRACEABILITY | IN_PROGRESS | `handoffs/A10-traceability.md` | docs only |
| A8 | Agy-P0-A8 | P0-PROD-INTEGRITY-BE | IN_PROGRESS | `handoffs/A8-production-integrity-backend.md` | backend/config/deploy residual (handoff pending) |
| W1+A9 | Opus48#B | P0-W1-PREFLIGHT | DONE | `handoffs/W1-A9-integration-validation.md` | this lane's prior deliverable |
| KIRO | Opus48-Kiro | P0-SUPERVISION | IN_PROGRESS | `kiro-audit.jsonl` | supervision only |

**Current-phase lock reality:** every active/done lane locks only its own `.deploy-control/p0/**`
artifact (verified from check-in `files_locked`). **Zero product-source file is locked yet** — this
freeze defines the *next* phase's disjoint source locks.

**R1-DESIGN / R2-DESIGN / A6 now give exact edit sites** — the freeze table §3 and the new §3.6 are
refined from generic hotspots to precise `file:symbol` old→new deltas (D1–D7 / L3c). Still not arrived:
**A7** (prod-integrity frontend); **A8** handoff body (check-in present, artifact pending). Their exact
reachable paths are locked-before-edit per PROTOCOL and are disjoint from the routing files by
construction (web/mobile/desktop + non-hotspot backend); rows fold in on arrival (`PENDING-ARRIVAL`).

---

## 2. Independent source anchor verification (read-only, HEAD `a6d5098`)

Confirmed by direct `rg` with line numbers (not taken on faith from the handoffs):

| Anchor | File:line | Literal / symbol (verified) |
|---|---|---|
| Cline GLM catalog row | `pkg/agent/models.go:562` | `{ID:"cline-pass/glm-5.2", Label:"GLM-5.2", Provider:"cline-pass"}` |
| Cline Kimi catalog row | `pkg/agent/models.go:563` | `{ID:"cline-pass/kimi-k2.7-code", Label:"Kimi K2.7 Code", Provider:"cline-pass"}` |
| NIM-direct GLM default | `pkg/agent/models.go:572` | `{ID:"z-ai/glm-5.2", Provider:"z-ai", Default:true}` |
| NIM default const | `pkg/agent/nim.go:25`, use `:146` | `nimDefaultModel = "z-ai/glm-5.2"` |
| Catalog expectation test | `pkg/agent/models_test.go:172` | expects `cline-pass/kimi-k2.7-code`, `cline-pass/glm-5.2` |
| Kimi thinking test | `internal/handler/agent_thinking_test.go:164` | `model:"cline-pass/kimi-k2.7-code"` |
| Trusted-adapter env switch | `internal/daemon/runtimeenv/env.go:195-222` | `trustedAdapterEntries`: cases `CLIClaudeCode`,`CLICodex`, `default:ErrAdapterFailClosed` (**no Cline branch**) |
| OpenAI-compat adapter gate | `internal/daemon/runtimeenv/adapter.go:58-59` | `CLIOpenAICompatible` → `AdapterFailClosed`, `GateOpenAICompatibleUnaccepted` |
| Env deny-list | `internal/daemon/runtimeenv/policy.go:18-39` | `DenyCredentialRoot`/`DenyProviderCredential` map (provider keys + `*_HOME`) |
| Task-home manifest guard | `internal/daemon/runtimeenv/home.go:48,65,67` | `ValidateTaskHomeManifest` forbids `.cline` dir + `providers.json` |
| Antigravity resolver (already committed) | `pkg/agent/antigravity.go:78,305-337` | `cmd.Env = antigravityResolverEnv(...)`; `GODEBUG=netdns=cgo` non-overriding |

---

## 3. FREEZE TABLE — exact file → single owner → dependency → source-edit gate → focused check

> One owner per file. Go invoked by absolute path `/home/ec2-user/goroot/go/bin/go`; `-count=1` defeats
> cache. Checks run **after** the edit, by the delta owner / A9. `state`: READY (gate satisfiable now),
> BLOCKED-EXT (external owner), PENDING (upstream lane not frozen), DECISION (needs Principal/W1 ruling),
> NO-EDIT (in tree / evidence-only).

### 3.1 W1-exclusive — `pkg/agent` catalog (Codex1/W1 hotspots)

| # | File (server-rel) | Owner | Depends on | Source-edit gate | Focused check | state |
|---|---|---|---|---|---|---|
| F1 | `pkg/agent/models.go:562` (GLM row SUB-1) | **W1** | A3 GLM FROZEN `cp/cline-pass/glm-5.2` | FLEET GREEN + W1 lock; A3 GLM met | `go test -count=1 ./pkg/agent/ -run 'ClineStaticModels\|StaticModels'` | **READY** |
| F2 | `pkg/agent/models.go:563` (Kimi row SUB-2) | **W1** | A3 **BLK-KIMI** (exact Kimi ID) | OmniRoute publishes exact versioned Kimi RouteModel | same as F1 (lockstep) | **BLOCKED-EXT** |
| F3 | `pkg/agent/models_test.go:172` (expectation) | **W1** | must change in lockstep with F1/F2 | tied to F1 (GLM) / F2 (Kimi) | `go test -count=1 ./pkg/agent/ -run 'StaticModels'` | READY(GLM)/BLOCKED-EXT(Kimi) |
| F4 | `pkg/agent/models.go:572` + `pkg/agent/nim.go:25,146` (NIM-direct GLM default, REVIEW-1) | **W1** | A3 REVIEW-1 flag; task 5.7 "remove obsolete direct-provider" | **Principal/W1 scope decision** (is NIM-direct default obsolete?) | `go test -count=1 ./pkg/agent/ -run 'NIM\|Nim'` | **DECISION** |
| F5 | `pkg/agent/nim_test.go` | **W1** | tied to F4 | tied to F4 decision | `go test -count=1 ./pkg/agent/ -run 'NIM\|Nim'` | DECISION |
| F6 | `internal/handler/agent_thinking_test.go:164` (Kimi literal, coupled) | **W1** (escalated — coupled to F2) | A3 BLK-KIMI | tied to F2 (Kimi literal) | `go test -count=1 ./internal/handler/ -run 'Thinking'` | BLOCKED-EXT |

### 3.2 W1-exclusive — daemon hotspots (FILE_OWNERSHIP D-V3-29)

| # | File (server-rel) | Owner | Depends on | Source-edit gate | Focused check | state |
|---|---|---|---|---|---|---|
| F7 | `internal/daemon/brain_integration.go` | **W1** | A6 launch-materialization gap (60%) + A4 `validateThinking` watch-item | A6 gap matrix frozen; A4 thinking axis needs registry `ThinkingLevels` (BLK-EXT if effort required) | `go test -count=1 ./internal/daemon/ -run 'BrainIntegration'` | **PENDING** (A6) |
| F8 | `internal/daemon/daemon.go` | **W1** | A6 `runTask/handleTask` spine gaps | A6 gap matrix frozen | `go test -count=1 ./internal/daemon/ -run '<changed-lifecycle>'` | PENDING (A6) |
| F9 | `internal/daemon/config.go` | **W1** | Cline/OpenAI-compat config wiring (A1) | A1 handoff frozen | `go test -count=1 ./internal/daemon/ -run 'Config'` | PENDING (A1) |
| F10 | `internal/daemon/health.go` | **W1** | readiness for changed routes | A1/A6 frozen | `go test -count=1 ./internal/daemon/ -run 'Health'` | PENDING |
| F11 | `cmd/multica/cmd_daemon.go` | **W1** | entrypoint wiring (only if changed) | integration | `go build ./cmd/multica/` | PENDING |
| F12 | `go.mod` | **W1** | only if a delta adds a dep | dep change proven necessary | `go build ./...` (server) | PENDING (likely NO-EDIT) |
| F13 | `internal/daemon/brain/**` | **W1** | frozen `agent-brain.v1` contract | only if a proven P0 gap requires it | `go test -count=1 ./internal/daemon/brain/...` | PENDING (likely NO-EDIT) |

### 3.3 CONTESTED shared `runtimeenv` — escalated to W1 serial (see §5)

> These are cross-CLI shared files (used by claude-code/codex today). A1 needs Cline changes in them,
> but they are **not** Cline-exclusive, so they are removed from R1's exact-Cline lock and **escalated to
> W1 serial**. A1 supplies the exact edit spec in `A1-A2-cline-foundation.md`; W1 applies during Wave C.

| # | File (server-rel) | Owner | Depends on | Source-edit gate | Focused check | state |
|---|---|---|---|---|---|---|
| F14 | `internal/daemon/runtimeenv/env.go:195-222` (add Cline branch to `trustedAdapterEntries`) | **W1 serial** | A1 FINDING-3 spec | A1 handoff frozen + architecture decision F18 | `go test -count=1 ./internal/daemon/runtimeenv/ -run 'Env\|Adapter'` | PENDING (A1) |
| F15 | `internal/daemon/runtimeenv/adapter.go:58-59` (flip `CLIOpenAICompatible` gate for accepted Cline) | **W1 serial** | A1 FINDING-3 + A3 contract | A1 frozen + decision F18 | `go test -count=1 ./internal/daemon/runtimeenv/ -run 'Adapter'` | PENDING (A1) |
| F16 | `internal/daemon/runtimeenv/policy.go:18-39` (trusted-inject `CLINE_DATA_DIR`/`CLINE_OMNIROUTE_API_KEY` vs deny) | **W1 serial** | A1 FINDING-3 | A1 frozen | `go test -count=1 ./internal/daemon/runtimeenv/ -run 'Policy'` | PENDING (A1) |
| F17 | `internal/daemon/runtimeenv/home.go:48-67` (carrier in separate `CLINE_DATA_DIR`, manifest rule) | **W1 serial** | A1 FINDING-3 | A1 frozen | `go test -count=1 ./internal/daemon/runtimeenv/ -run 'Home\|Manifest'` | PENDING (A1) |

### 3.4 R1/A1-exclusive — Cline-exact `runtimeenv` files

| # | File (server-rel) | Owner | Depends on | Source-edit gate | Focused check | state |
|---|---|---|---|---|---|---|
| F18 | `internal/daemon/runtimeenv/cline.go` | **R1/A1** | A3 RouteModel (ID-agnostic; GLM via param) | A1 handoff + FLEET GREEN + lock | `go test -count=1 ./internal/daemon/runtimeenv/ -run 'Cline'` | READY(GLM)/HOLD(Kimi) |
| F19 | `internal/daemon/runtimeenv/cline_test.go:18-20,79` | **R1/A1** | A3 GLM ok; Kimi alias = BLK-KIMI | tied to F18 | `go test -count=1 ./internal/daemon/runtimeenv/ -run 'Cline'` | READY(GLM)/HOLD(Kimi) |
| F20 | new `internal/daemon/runtimeenv/cline_*.go` (if A1 adds; e.g. carrier writer) | **R1/A1** | A1 design (reuse `CLIOpenAICompatible`) | A1 frozen; created under lock only | `go test -count=1 ./internal/daemon/runtimeenv/` | PENDING (A1) |

### 3.5 NO-EDIT — evidence/config/live-run only (no source lock issued)

| # | Item | Owner | Note | state |
|---|---|---|---|---|
| F21 | `pkg/agent/antigravity.go:78` + `antigravity_test.go` | (none for edit) | **already committed** (`7735bdc`); A5 = STALE → needs **one live scenario**, not a source edit | NO-EDIT / live gated |
| F22 | Antigravity live scenario (route `agy/claude-opus-4-6-thinking`) | W1 R2 live-run token | blocked by **D-V3-25(B)** key-revocation security stop | BLOCKED-EXT |
| F23 | Opus48 route delivery (5.8) | W1 (config wiring) | A4: **zero new source**; set `CLIKind=claude-code` + `<OPUS48_AWS_ID>`; reuse accepted Anthropic-Messages chain | BLOCKED-EXT (RouteModel) |
| F24 | `internal/daemon/gateway/**` (registry/projection/profiles) | W2 (not P0-edit) | generic; consumes OmniRoute snapshot; no hardcoded IDs; `model_projection.go:46` is dev-compat | NO-EDIT |
| F25 | `packages/core/runtimes/models.ts` | (none) | discovery-based; no static GLM/Kimi IDs to freeze | NO-EDIT |

---

## 3.6 Exact edit-site refinement (R1 D1–D7 · A6 L3c · R2 config-only)

R1-DESIGN froze the Cline delta to **7 exact edits (D1–D7)**; A6 confirmed the single real lifecycle
gap is **L3c** (credentialless Cline admit+materialize); R2-DESIGN proved Opus48 needs **zero** source.
This refines the generic rows above to exact `file:symbol` with owner and the **serial order** R1 §8.

| D | Exact site | old→new (summary) | Owner (editor) | maps to F | dep/gate |
|---|---|---|---|---|---|
| D1 | `runtimeenv/adapter.go` `CredentiallessAdapterContract` case `CLIOpenAICompatible` (`:58-59`) | `AdapterFailClosed{GateOpenAICompatibleUnaccepted}` → `AdapterReady, ProtocolOpenAIChat` | **W1 serial** (spec by R1) | F15 | A3 GLM + OmniRoute Chat contract accepted |
| D2 | `runtimeenv/env.go` `trustedAdapterEntries` (`:195-222`) + add field `AdapterEnvironment.ClineDataDir` (`:107-113`) + `BuildGatewayEnvironment` | add `case CLIOpenAICompatible` injecting `CLINE_DATA_DIR`(trusted-local)+`CLINE_OMNIROUTE_API_KEY`(trusted-secret) | **W1** | F14 | D1; A1 spec |
| D3 | `internal/daemon/config.go` `agentBrainBuiltInCLIFor` (`:141`) | add `case CLIOpenAICompatible: {Provider:"cline",Command:"cline"}` | **W1** | F9 (exact) | A1 spec |
| D4 | `internal/daemon/config.go` `AgentBrainIntegrationConfig.Validate` (`:179`, case `:196`) | add `CLIOpenAICompatible` to accepted case + message | **W1** | F9 (exact) | A1 spec |
| D5 | `internal/daemon/execenv/cline_home.go` **add** `WriteCredentiallessClineConfig` + `prepareCredentiallessClineHome` | mirror `codex_home.go:87 WriteCredentiallessCodexConfig`; carrier at `<clineDataDir>/settings/providers.json` 0600 | **W1** | **F-NEW-1** | A1 spec; ⚠ file EXISTS (see note) |
| D6 | `internal/daemon/brain_integration.go` `buildLaunch` (`:277`; Codex branch `:331-338`; adapter literal `:321-323`) | add sibling `else if CLIOpenAICompatible` branch: create task-scoped dir, `NewClineConfigContract`, `WriteCredentiallessClineConfig`, set `Adapter.ClineDataDir`, empty manifest; **W1 decides `updatedAt` fmt** | **W1** | F7 (exact) | A6 L3c; D1–D5 |
| D7 | `pkg/agent/models.go` `clineStaticModels` GLM row (`:562`) + `models_test.go:172` | `cline-pass/glm-5.2` → `cp/cline-pass/glm-5.2` (GLM only; **do not touch `:563` Kimi / `:572` NIM**) | **W1** | F1/F3 | A3 SUB-1 (met) |

**A6 L3c** = D1+D5+D6 + extend `runtimeenv.AssertPreLaunch` to the Cline data-dir layout (folds into
F14/F7). **New test gap A6 L7b:** assert the controlled `agent-brain-home` (0700, under `env.RootDir`)
is reclaimed with the env root — add one `internal/daemon` test (`gc_test.go` or
`brain_integration_test.go`), **owner W1**, → **F-NEW-2**. Focused check: `go test -count=1 ./internal/daemon/ -run 'GC\|BrainIntegration'`.

**R2 confirms F23 = config-only:** `AGENT_BRAIN_CLI_KIND=claude-code` + `AGENT_BRAIN_ROUTE_MODEL=<OPUS48_AWS_ID>`;
verified resolution chain `config.go loadAgentBrainIntegrationConfig(:735)` → `Validate(:179)` →
`daemon.go resolveTaskAgentEntry(~:3255)` → `brain_integration.go admitTask(:152)`. **Do NOT** edit
`LegacyProviderCLIKind`/`agentBrainBuiltInCLIFor`/`resolveTaskAgentEntry` for Kiro, and **do NOT** wire
native `kiro-cli`/`XDG`/`AWS` (forbidden credential owner). Opus48 launch lifecycle (A6 **L3a**) already ready.

> ⚠ **W1 reconciliation (D5):** R1-DESIGN labels `execenv/cline_home.go` a "new file", but it **already
> exists** (verified: legacy `prepareClineHome`/`resolveClineSourceDir`, 2500 B). W1 must **add** the new
> `WriteCredentiallessClineConfig`/`prepareCredentiallessClineHome` functions to the existing file (or a
> clearly-named sibling), not create a duplicate. This is a naming/placement reconciliation, not a new gap.

> **A6 guardrail (do-not-touch):** the live driver is the inline `runTask`/`handleTask` path; the neutral
> `brain.Coordinator`/`LifecycleTaskExecutor` (`brain/coordinator.go`,`brain/executor.go`) is the
> test-only G2 strangler. **Do not re-route the live path through the Coordinator during P0** — it is by
> design, not a gap. Reinforces F13 (`brain/**`) = NO-EDIT unless a proven gap.

**Refined serial order (R1 §8, supersedes §6 step 3 internals):** D1 → D2 (+field) → D5 → D6
(+`updatedAt` decision) → D3+D4 → D7 (+test lockstep) → GLM live run; then on BLK-KIMI clear apply K2/K3
one-line fixtures → Kimi live run.

---

## 4. Pairwise zero-overlap proof

**Claim:** every source file has exactly one owner; owner sets are pairwise disjoint.

Owner → file set (from §3):

- **W1** = { F1/F2/F3 `pkg/agent/models.go`, `pkg/agent/models_test.go`, F4 `pkg/agent/nim.go`,
  F5 `pkg/agent/nim_test.go`, F6 `internal/handler/agent_thinking_test.go`,
  F7 `brain_integration.go`, F8 `daemon.go`, F9 `config.go`, F10 `health.go`, F11 `cmd_daemon.go`,
  F12 `go.mod`, F13 `brain/**`, **+ escalated shared** F14 `runtimeenv/env.go`, F15 `runtimeenv/adapter.go`,
  F16 `runtimeenv/policy.go`, F17 `runtimeenv/home.go`, **+ D5** F-NEW-1 `internal/daemon/execenv/cline_home.go`
  (add `WriteCredentiallessClineConfig`), **+ L7b** F-NEW-2 `internal/daemon/{gc_test.go|brain_integration_test.go}` (controlled-home reclamation test) }
- **R1/A1** = { F18 `runtimeenv/cline.go`, F19 `runtimeenv/cline_test.go`, F20 new `runtimeenv/cline_*.go` }
- **A3, A4, A5, A6, A10** = ∅ source files. (A3/A4 = docs+substitution specs; A5 = evidence; A6 = gap
  matrix whose fixes land in W1's F7/F8; A10 = traceability docs.)
- **A7, A8** = `PENDING-ARRIVAL` (web/mobile/desktop; non-hotspot backend/config/deploy) — disjoint from
  all routing files above by construction; recorded when handoffs land.
- **W2** = `gateway/**` (F24) — not a P0 edit; listed NO-EDIT.

**Intersection checks:**
- `W1 ∩ R1` : W1 holds no `runtimeenv/cline*.go`; R1 holds only `runtimeenv/cline*.go`. The shared
  runtimeenv files (env/adapter/policy/home) were **removed from R1 and assigned solely to W1** → `∅`. ✓
- `W1 ∩ {A3,A4,A5,A6,A10}` = `∅` (those own no source). ✓
- `R1 ∩ {all others}` = `∅`. ✓
- Package-compile coupling (Go): F18/F19/F20 compile in `package runtimeenv` alongside W1's F14–F17.
  This is a **compile-unit coupling**, handled by **serialization** (§5), not co-editing: R1 edits only
  `cline*.go`; W1 edits the shared files; they are never edited concurrently. Same pattern for
  `package agent` (F1–F5 all W1) and `internal/daemon` (F7–F13 all W1) — single owner per package-shared file.

**Result: pairwise intersection = ∅ across all owners with a source lock.** The only couplings are
same-package compile units, resolved by the serial queue (§5), never by concurrent edits.

---

## 5. Contested files escalated to W1 (explicit list + reason)

| Contested item | Lanes that would touch it | Why contested | Resolution |
|---|---|---|---|
| `runtimeenv/env.go` (F14) | A1 (Cline branch) + shared claude/codex | cross-CLI shared switch, not Cline-exclusive | **→ W1 serial**; A1 supplies spec |
| `runtimeenv/adapter.go` (F15) | A1 (flip OpenAI-compat gate) + shared contract | shared adapter contract for all CLIs | **→ W1 serial** |
| `runtimeenv/policy.go` (F16) | A1 (`CLINE_*` trusted) + shared deny-list | global env deny-list | **→ W1 serial** |
| `runtimeenv/home.go` (F17) | A1 (`.cline`/carrier) + shared manifest guard | shared task-home validator | **→ W1 serial** |
| `brain_integration.go` (F7) | A4 (`validateThinking`) + A6 (launch materialization) | two lanes' deltas land in one W1 hotspot | **→ W1**; serialize A6 then A4 |
| `daemon.go` (F8) | A6 spine gaps | W1 hotspot | **→ W1** |
| `pkg/agent/models.go` (F1/F2) | A3 substitutions | W1 hotspot; A3 is docs-only | **→ W1** applies SUB-1/SUB-2 |
| `internal/handler/agent_thinking_test.go` (F6) | coupled to F2 Kimi literal | must change in lockstep with a W1 hotspot | **→ W1** (escalated coupled test) |
| brain_integration↔runtimeenv **launch-materialization SEAM** | A1 (Cline config materialization) vs A6 (launch-stage credentialless-config gap for OpenAI-compat/Cline) | the same behavior spans R1's `cline.go` and W1's `brain_integration.go` | **→ W1 serial**: integrate A1 `cline.go` delta first, then wire the launch-stage materialization in `brain_integration.go` |

**Architecture decision required before F14–F20 (flag to Principal/W1):** A1 recommends **reusing
`CLIOpenAICompatible` (executable = `cline`)** rather than introducing a new `CLIKind`. This choice
determines the exact edits to F14/F15 and whether F1–F3 need a new catalog entry. It is a
**W1/Principal ruling**, not an auto-applied change.

---

## 6. Serial integration queue (order + gate per stage)

Per PROTOCOL Phase C order (Production integrity → Core → Cline → Kiro/Opus48), refined with the frozen files:

1. **Production integrity** (A7/A8, `PENDING-ARRIVAL`) — exact reachable paths, disjoint from routing files.
   Gate: A7/A8 handoffs frozen. Check: FE `pnpm --filter <pkg> test/typecheck`; BE `go build ./... && go vet`.
2. **Main Brain core / lifecycle** (A6 → F7/F8, +F9/F10) — integrate lifecycle gaps at the W1 hotspot seam.
   Gate: A6 gap matrix frozen. Check: `go test -count=1 ./internal/daemon/ -run 'BrainIntegration\|<lifecycle>'`.
3. **Cline routes** (F18/F19 then shared F14–F17, then F1/F3 GLM) — GLM (5.6) first; **Kimi (5.7) HOLD on BLK-KIMI**.
   Gate: A1 handoff + decision §5 + A3 GLM. Check: `go test -count=1 ./internal/daemon/runtimeenv/ -run 'Cline\|Adapter\|Env\|Policy\|Home' && go test -count=1 ./pkg/agent/ -run 'StaticModels'`.
4. **Kiro/Opus48** (F23 config; F21/F22 Antigravity reuse/live) — zero new source; **BLOCKED-EXT on Opus48 RouteModel** and Antigravity live scenario (D-V3-25(B)).
   Gate: A4 exact RouteModel. Check: synthetic `go test -count=1 ./internal/daemon/gateway/... ./internal/daemon/runtimeenv/... -run 'AnthropicMessages\|RouteModel\|Selection'`.
5. **Final integrated compile gate** (once): `/home/ec2-user/goroot/go/bin/go build ./...` (server).

Rules: one lane merged at a time; focused checks green before next; a conflict that does not change
semantics does not spawn a new campaign; one reserved live run per changed/unproven family closes
overlapping 5.x/8.x.

---

## 7. Freeze blockers (external owner + action; not guessed)

| ID | Blocker | Owner | Required action | Blocks |
|---|---|---|---|---|
| BLK-KIMI | exact Cline→Kimi-K2.7 versioned RouteModel undeclared (3 divergent Multica-side literals) | OmniRoute architect (via Principal) | publish exact versioned Kimi model id + protocol + availability | F2, F6, 5.7, Kimi half of F18/F19 |
| BLK-AVAIL | enriched OmniRoute registry rows (protocol/tools/reasoning/pool/available) not published for Cline routes | OmniRoute architect | publish enriched `/v1/models` rows for `cp/cline-pass/glm-5.2` (+ Kimi) | 8.1 admission (not the GLM exact-ID freeze, which is done) |
| BLK-OPUS48 | exact Opus48-on-AWS RouteModel absent | OmniRoute registry owner | publish anthropic-messages Opus48 row + provenance | F23, 5.8 |
| BLK-25B | live-provider tests security-stopped until exposed key revoked | Product/OmniRoute owner | confirm UI key revocation | F22 Antigravity live + all live runs |
| GATE-FLEET | source edits blocked until `FLEET_SATURATED=GREEN` (A7/A8 + all workers working/blocked + zero overlap) | Principal | freeze locks + issue tokens after GREEN | all F* edits |
| DEC-NIM | is NIM-direct GLM default (F4) an obsolete direct-provider alt to retire (5.7)? | Principal/W1 | ruling | F4/F5 |
| DEC-CLIKIND | reuse `CLIOpenAICompatible` (executable=cline) vs new `CLIKind`? | Principal/W1 | ruling | F14/F15/F18/F20 |

---

## 8. Summary

- **~22 candidate source files** mapped to a single owner each: **W1 = 18** (incl. 4 escalated shared
  `runtimeenv`, 1 coupled handler test, new `execenv/cline_home.go` D5, and the L7b reclamation test),
  **R1/A1 = 3** (Cline-exact) [+ new Cline files]; **A3/A4/R1-DESIGN/R2-DESIGN/A5/A6/A10 own zero source**
  (design/evidence handoffs); **A7/A8 = PENDING-ARRIVAL** (disjoint by construction); **5 NO-EDIT/gateway/evidence items**.
- **Cline delta frozen to 7 exact edits (R1 D1–D7)** + A6 L3c (adapter+materialize+buildLaunch) + L7b test;
  **Opus48 proven config-only (R2, ∅ source)**; the one real Main-Brain lifecycle gap is **L3c** only.
- **Pairwise lock overlap = ∅** (§4); only same-package compile couplings remain, resolved by the serial
  queue (§6/§3.6), never concurrent edits. R1's direct-edit lock is Cline-exact (`cline*.go`); all shared
  `runtimeenv` files are edited **solely by W1** during serial integration consuming R1's exact spec.
- **Contested files explicitly escalated to W1** (§5): 4 shared `runtimeenv` files, 2 daemon hotspots,
  `models.go`, the coupled handler test, and the brain_integration↔runtimeenv launch-materialization seam (= L3c/D6).
- **Ready now (once FLEET GREEN + lock):** GLM path — D1–D7 (GLM) end-to-end. **Blocked external:**
  Kimi (BLK-KIMI, one-line fixtures K2/K3), Opus48 (BLK-OPUS48, config-only), availability (BLK-AVAIL),
  all live runs (BLK-25B). **Decisions:** NIM-direct default (DEC-NIM), Cline CLIKind reuse — **R1 recommends
  reuse `CLIOpenAICompatible`/executable `cline`** (DEC-CLIKIND, confirm), `updatedAt` format (D6), D5 file
  placement reconciliation.
- No product source edited, no test executed, no live run. This document is the freeze only.
