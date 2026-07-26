# A10 — P0 Traceability & Evidence-Overlap Map (closure prep for the Principal)

- agent / identity: **Opus48#C** · lane **A10** · task **P0-TRACEABILITY** · pane `w8:p1` (`$HERDR_PANE_ID`)
- output lock (sole mutable file): `.deploy-control/p0/handoffs/A10-traceability.md`
- check-in: `.deploy-control/p0/checkins/Opus48-C__P0-TRACEABILITY__20260721T223641Z.json`
- as-of (UTC): `2026-07-21T23:38Z` · repo HEAD `a6d50986…` · **phase `IMPLEMENTATION_W1_SERIAL`, gate GREEN, `implementation_authorized=true`** (see §8 for the live update) · branch `integration/dev-transition-candidate-20260719`
- role: **preparer only.** I do NOT mark OpenSpec/GSD checkboxes, do NOT rerun tests, do NOT open a QA/reviewer lane, do NOT invent evidence/RouteModel IDs. Overlapping `5.x`/`8.x` proof is deduplicated to one live run per route family. `8.5–8.7` are OmniRoute-external.
- OpenSpec disk state (unchanged; closure is Principal-only): `5.6 [ ]`, `5.7 [ ]`, `5.8 [ ]`, `8.1 [ ]`, `8.2 [ ]`.

## 0. Preflight
git `/usr/bin/git` 2.50.1 · python3 `/usr/bin/python3` 3.9.25 · rg `/usr/local/bin/rg` 15.2.0 · sha256sum `/usr/bin/sha256sum` (coreutils 8.32). p0_control.py check-in OK. Read-only on product/OpenSpec/GSD.

## 1. Current lane states consumed (source: control.json + checkins/*.json + handoffs/* + events.jsonl)

> Refreshed as of `2026-07-21T22:39Z` — A1+A2 and A6 now DONE (handoffs on disk); implementation-design phase has begun (R1/R2-DESIGN, R1-TEST-DESIGN, W1-SOURCE-FREEZE); A8 checked in; A7 still missing.

**Gap-matrix phase lanes (all DONE except A7):**

| Lane | Agent · pane | Task | State | Artifact (on disk?) | One-line result |
|---|---|---|---|---|---|
| A1+A2 | Opus48#A · w6:p1 | P0-CLINE-FOUNDATION | **DONE** (100%, 22:37:53Z) | `handoffs/A1-A2-cline-foundation.md` ✓ | Cline carrier `cline.go` already EQUIV (zero delta); real 5.6/5.7 work = 5 wiring gaps in W1 hotspots (C3–C7); reuse `CLIOpenAICompatible`(exec=cline), no new `CLICline` |
| A3 | Codex56#A · w7:p3 | P0-ROUTE-FREEZE | **DONE** (100%, 22:35:39Z) | `handoffs/A3-route-freeze.md` ✓ | GLM FROZEN `cp/cline-pass/glm-5.2`; NVIDIA fallback FROZEN non-Brain-selectable; Kimi BLOCKED_EXTERNAL; availability BLOCKED_EXTERNAL |
| A4 | Codex56#B · w7:p4 | P0-OPUS48-ROUTE | **DONE** (100%, 22:36:18Z) | `handoffs/A4-opus48.md` ✓ | frontend PROVEN=`claude-code`+`ProfileAnthropicMessages`; zero new Main-Brain code; exact Opus48 AWS RouteModel BLOCKED_EXTERNAL |
| A5 | Opus48#C · w8:p1 | P0-ANTIGRAVITY-EQUIVALENCE | **DONE** (100%, 22:34:50Z) | `evidence/A5-antigravity-equivalence.md` ✓ | **STALE**: route/protocol REUSE (profiles.go/models.go byte-identical), runtime adapter `antigravity.go` changed post-acceptance (GODEBUG inject, `7735bdc`) |
| A6 | Opus48#D · w8:p2 | P0-LIFECYCLE-GAP | **DONE** (100%, 22:39:16Z) | `handoffs/A6-lifecycle-gap-matrix.md` ✓ | Kanban→terminal lifecycle substantially EQUIV (L0–L9); exactly **one** real Main-Brain gap **L3c** (Cline credentialless admit+launch-config), == A1+A2 C5/C7; 5.8 launch lifecycle L3a ready |
| W1+A9 | Opus48#B · w6:p2 | P0-W1-PREFLIGHT | **DONE** (100%, 22:34:00Z) | `handoffs/W1-A9-integration-validation.md` ✓ | serial order Prod-integrity→Core→Cline→Opus48; delta→command matrix D1–D8; live-token prereqs; proposed zero-overlap source locks |
| A7 | Agy-P0-A7 · wB:p1 | P0-PROD-INTEGRITY-FE | **IN_PROGRESS** (0%, hb 22:42:33Z) | `handoffs/A7-production-integrity-frontend.md` (pending) | now checked in — gate denominator met (§6) |
| A8 | Agy-P0-A8 · wB:p2 | P0-PROD-INTEGRITY-BE | **DONE** (100%, 22:55:50Z) | `handoffs/A8-production-integrity-backend.md` ✓ | **NO_REACHABLE_RESIDUAL** — no reachable mocks/fake-success/QA routes/placeholders/demo persistence in backend/config/deploy (outside hotspots) |
| — | Opus48-Kiro · w5:p1 | P0-SUPERVISION | IN_PROGRESS (supervisor) | `kiro-audit.jsonl` | last sweep 22:31:33Z sev=GREEN (management, not a requirement owner) |

**Implementation-design phase lanes (design deltas DONE; W1 freeze in flight) — as of 22:42Z:**

| Lane | Agent · pane | Task | State | Artifact | Maps to |
|---|---|---|---|---|---|
| W1 | Opus48#B · w6:p2 | P0-W1-SOURCE-FREEZE | **DONE** (100%, ~22:46Z) | `handoffs/W1-source-lock-freeze.md` ✓ | source-lock freeze F1–F25, pairwise-∅ proof, serial queue for 5.6–5.8 (plan-only; no locks/tokens issued yet) |
| W1 | Opus48#B · w6:p2 | P0-W1-INTEGRATION | **BLOCKED** (0%, hb 22:48Z) | `handoffs/W1-integration-readiness.md` ✓ | serial integration ready; **awaiting source-lock activation** (gate GREEN but `implementation_authorized=false`, phase `SOURCE_LOCK_FREEZE`; F1–F25 locks not yet activated) |
| R1-DESIGN | Codex56#A · w7:p3 | P0-CLINE-ROUTE-SOURCE-DELTA | **DONE** (100%, 22:41:40Z) | `handoffs/R1-cline-route-source-delta.md` ✓ | 5.6/5.7 exact 7-edit patch plan (D1–D7); GLM implementable now, Kimi ID-blocked |
| R1-TEST-DESIGN | Opus48#A · w6:p1 | P0-CLINE-TEST-CONTRACT | **DONE** (100%, 22:42:31Z) | `handoffs/R1-cline-focused-tests.md` + `R1-test-implementation-readiness.md` ✓ | 5.6/5.7/8.1/8.2 focused test contract; Kimi tests HELD |
| R2-DESIGN | Codex56#B · w7:p4 | P0-OPUS48-SOURCE-DELTA | **DONE** (100%, 22:42:03Z) | `handoffs/R2-opus48-source-delta.md` ✓ | 5.8: **zero product-source lines**; config-only (`AGENT_BRAIN_CLI_KIND=claude-code`+`<OPUS48_AWS_ID>`) |
| R3-TEST-DESIGN | Opus48#D · w8:p2 | P0-LIFECYCLE-TEST-CONTRACT | **DONE** (100%, 22:44:56Z) | `handoffs/R3-lifecycle-focused-tests.md` + `R3-test-implementation-readiness.md` ✓ | 8.2 lifecycle test contract (L3c admit+launch / L7b cleanup / cancel-usage-terminal) |
| A7 | Agy-P0-A7 · wB:p1 | P0-PROD-INTEGRITY-FE | **IN_PROGRESS** (45%, hb 22:44:56Z) | (pending) | now checked in → gate denominator met (see §6) |
| A8 | Agy-P0-A8 · wB:p2 | P0-PROD-INTEGRITY-BE | **DONE** (100%, 22:55:50Z) | `handoffs/A8-production-integrity-backend.md` ✓ | NO_REACHABLE_RESIDUAL (integration step-1 backend clear) |

**Validation-phase lanes (all BLOCKED on the gate/tokens — as of 22:47Z):**

| Lane | Agent · pane | Task | State | Blocker (verbatim owner/next) | Reserved evidence target |
|---|---|---|---|---|---|
| A9-R1 | Opus48#A · w6:p1 | (Cline focused tests) | **BLOCKED** | FLEET_SATURATED **RED**; W1 locks not frozen; owner Principal + W1; next = consume W1 freeze, acquire disjoint test-file locks, implement focused Cline tests | (focused unit tests) |
| A9-R3 | Opus48#D · w8:p2 | P0-LIFECYCLE-TEST-IMPLEMENTATION | **BLOCKED** | FLEET RED; target files (`brain_integration_test.go`/`gc_test.go`/`runtimeenv/assert_test.go`) W1-assigned (F7/F8); confirmed `WriteCredentiallessClineConfig` **absent** + adapter.go still `AdapterFailClosed` (L3c unimplemented); readiness in `R3-test-implementation-readiness.md` | (lifecycle unit tests) |
| A9-GLM | Codex56#A · w7:p3 | (GLM live acceptance) | **BLOCKED** | impl not integrated; `live_runs.cline_glm authorized=false`; owner Principal + W1 + OmniRoute registry owner | `evidence/R1-glm-live-acceptance.md` |
| A9-OPUS48 | Codex56#B · w7:p4 | (Opus48 live acceptance) | **BLOCKED** | exact Opus48 RouteModel not certified; `live_runs.opus48 authorized=false`; owner OmniRoute registry owner + Principal | `evidence/R2-opus48-live-acceptance.md` |

## 2. Requirement → implementation candidate → focused validation → evidence/live-token → blocker

### 5.6 — Cline → GLM-5.2 via OmniRoute
- **Impl candidate (A1+A2 DONE):** shared Cline carrier `runtimeenv/cline.go` (`NewClineConfigContract`) is **already EQUIV — zero delta**; the real 5.6/5.7 work is **wiring in W1 hotspots**, reusing `brain.CLIOpenAICompatible` (exec=`cline`) on `ProtocolOpenAIChat` (no new `CLICline`). Exact gaps: **C3** `config.go agentBrainBuiltInCLIFor` add `CLIOpenAICompatible→{cline,cline}`; **C4** `config.go Validate` accept `CLIOpenAICompatible`; **C5** `runtimeenv/adapter.go` flip `GateOpenAICompatibleUnaccepted`→`AdapterReady`/`ProtocolOpenAIChat`; **C6** `env.go trustedAdapterEntries` inject `CLINE_DATA_DIR`+`CLINE_OMNIROUTE_API_KEY` trusted-last (deny-list rejects them as custom → must be trusted-only); **C7** new `execenv.WriteCredentiallessClineConfig` + `brain_integration.go buildLaunch` Cline branch (carrier in separate `CLINE_DATA_DIR`, excluded from task-home manifest). RouteModel `cp/cline-pass/glm-5.2` + hotspot **SUB-1** at `pkg/agent/models.go:562`. Integrated at W1 serial step 3. *(Design DONE: `R1-cline-route-source-delta.md` = exact 7-edit old→new patch plan D1–D7; `R1-cline-focused-tests.md` = focused test contract T-ADPT/ENV/EXEC/CFG/LAUNCH/MODEL; source locks in `W1-source-lock-freeze.md` F1/F3/F14–F20. GLM implementable now; Kimi rides the same code path on ID publish.)*
- **Focused validation (A9, after integration — not now):** W1-A9 **D3** `go test -count=1 ./internal/daemon/runtimeenv/ -run 'Cline|Model'`; **D3b** `…/gateway/ -run 'Projection|Registry|Model'`. Lockstep-update `models_test.go:172` expectation with SUB-1.
- **Evidence/live-token:** `live_runs.cline_glm` (authorized=false). One live Kanban→terminal run closes **5.6 + 8.1 + 8.2** overlap.
- **Blocker:** BLK-AVAIL (enriched OmniRoute registry row for `cp/cline-pass/glm-5.2` unpublished — owner: OmniRoute architect); **BLK-CLINE-ENV** — Cline 3.0.44 `${ENV}` expansion of `settings.apiKey` unproven in-repo; if false and write-time secret substitution is disallowed (it is, per `ValidateClineConfigBytes`), this becomes a real launch blocker (owner: A1/W1, verify vs installed CLI before live run); D-V3-25(B) security-stop for the live run (owner: Owner/key-revocation); Principal source-lock freeze + token. *(A1+A2 base analysis now DELIVERED.)*

### 5.7 — Cline → Kimi-K2.7 via OmniRoute
- **Impl candidate:** increment on the **same** 5.6 Cline base (never a 2nd Cline impl). RouteModel undeclared. **REVIEW-1** flagged by A3: NIM-direct GLM default `z-ai/glm-5.2` (`models.go:572`, `nim.go:25`) as candidate "obsolete direct-provider alternative" to retire per 5.7 — **W1/A1+Principal** scope decision, not a freeze/A10 decision.
- **Focused validation:** **D4** = reuse D3/D3b for the same packages after the Kimi ID is wired (no new campaign).
- **Evidence/live-token:** `live_runs.cline_kimi` (authorized=false). One run closes **5.7 + 8.1 + 8.2** overlap.
- **Blocker:** **BLK-KIMI** — exact versioned Kimi `RouteModel` UNDECLARED; 3 divergent Multica-side literals (`cline-kimi-k2.7-dedicated`, `cline-pass/kimi-k2.7-code`, `kimi-sub`), none selectable by plausibility (owner: OmniRoute architect). Plus BLK-AVAIL, D-V3-25(B), Principal, A1 base.

### 5.8 — Antigravity revalidation **and** Kiro/Opus48 delivery (two disjoint halves)
- **Half A — Antigravity (A5 DONE):** decision **STALE**. Route-identity/protocol layer **REUSE** (`gateway/profiles.go`=`4824ef05…`, `pkg/agent/models.go`=`a6957e3e…` byte-identical to accepted `b657` pins). Runtime launch adapter `antigravity.go` changed (`96ee0c98…`→`1196c6f4…`, commit `7735bdc` in-product `GODEBUG=netdns=cgo`) with no accepted pin. → default `reuse_equivalent_evidence` **does NOT apply**; the one minimal scenario A5 named (single non-streaming `agy/claude-opus-4-6-thinking` Kanban→terminal on HEAD `a6d5`) must be **reserved** (not run).
- **Half B — Opus48 (A4 + R2-DESIGN DONE):** frontend PROVEN = `CLIClaudeCode` + `ProfileAnthropicMessages`(`/v1/messages`, SSE, root) + credentialless `AdapterReady`. **Zero product-source lines** — `R2-opus48-source-delta.md` proves the path down to `config.go loadAgentBrainIntegrationConfig`+`admitTask`+`FrozenTier20CanaryPolicy.Validate` (no RouteModel allow-list): delivered purely by config `AGENT_BRAIN_CLI_KIND=claude-code` + `AGENT_BRAIN_ROUTE_MODEL=<OPUS48_AWS_ID>` + a `provider="claude"` Kanban agent, reusing the accepted Claude→Anthropic-Messages chain. Explicitly rejected (non-minimal/forbidden): `kiro` CLIKind/provider mapping, native `kiro-cli`/`XDG`/`AWS` path, inventing a route allow-list. Watch-item: `validateThinking` rejects non-empty thinking — Anthropic-protocol-general, gated by registry `ThinkingLevels` (do not invent effort approvals).
- **Focused validation:** **D5** `…/runtimeenv/ -run 'Assert|Model'` + `…/gateway/ -run 'Projection|Profiles'`; **D6** (Antigravity) = A5's single scenario (because STALE), else none.
- **Evidence/live-token:** `live_runs.opus48` (authorized=false) closes **5.8(Opus48) + 8.1 + 8.2**; `live_runs.antigravity` — STALE ⇒ reserve the single A5 scenario under W1/R2 (closes 5.8(Agy)+8.1+8.2 overlap).
- **Blocker:** exact Opus48 AWS `RouteModel` BLOCKED_EXTERNAL (owner: OmniRoute registry owner); Antigravity scenario + all Opus48 live gated by D-V3-25(B); Principal token.

### 8.1 — model/capability/protocol/availability (changed or unproven routes only)
- **Not a separate campaign.** Rides on the registry/projection checks of the routes above: **D3b** (Cline GLM/Kimi) and **D5** (Opus48); Antigravity protocol **REUSE** (EV-G4-01 synthetic conformance for `agy/claude-opus-4-6-thinking` still valid since `profiles.go`/`models.go` unchanged), availability on candidate build folds into the A5 scenario.
- **Closed by:** the same `cline_glm`/`cline_kimi`/`opus48`/`antigravity` runs (overlap). No extra run.
- **Blocker:** same external RouteModel + BLK-AVAIL + D-V3-25(B).

### 8.2 — tools · reasoning · cancellation · usage · terminal result · deterministic error (affected routes only)
- **Impl dependency (A6 DONE):** Kanban→terminal lifecycle is substantially EQUIV/unit-proven (L0–L2, L3a/L3b, L4–L9 all Category-1 reuse). There is exactly **one** real Main-Brain lifecycle gap — **L3c**: credentialless admit+launch-config materialization for the Cline family (`CLIOpenAICompatible`/`CLIKimi`), which is the **same gap** as A1+A2 **C5/C7** (adapter flip + `WriteCredentiallessClineConfig` + `buildLaunch` branch). See §5 convergence. 5.8/Opus48 launch lifecycle **L3a is already ready** (only the A4 RouteModel is missing). Cancellation is re-verified **only if** the Brain launch branch changes (folds into L3c). **Reasoning decision flag:** the slice currently *rejects* non-empty thinking (`validateThinking`→`ErrThinkingNotApproved`); enabling reasoning on a P0 route is a policy/registry-capability change owned by A3/A4+W1, not an A6/lifecycle gap. Category-4 (OmniRoute) legacy rotation/credential is correctly *gated off* when `RouterOwner=omniroute` (task 7.7).
- **Reuse:** Claude→Anthropic-Messages protocol-level tools/cancel/usage/terminal already accepted (route-matrix #2; G4) → Opus48 reuses. Antigravity prior state = synthetic "deterministic fail-closed contracts"; live tools/reasoning/usage only via the A5 scenario.
- **Focused validation:** **D2** `…/internal/daemon/ -run '<changed-lifecycle>'` (BrainIntegration/admission/launch/cancel/cleanup) + the per-route D3/D4/D5.
- **Closed by:** the same per-family live runs (overlap). No second acceptance.
- **Blocker:** A6 gap classification pending (owner: A6→C/W1); same external RouteModel/security blockers.

### 8.5 / 8.6 / 8.7 — **EXTERNAL to Main Brain P0 (OmniRoute owner)**
No Multica/Agent-Brain lane, file lock, QA, or live run. `control.json.external_non_main_brain_tasks=["8.5","8.6","8.7"]`.
- 8.5 = expired/revoked creds, token/window limits, quota, 401/403, 429, 5xx, timeout, malformed upstream — OmniRoute-certified.
- 8.6 = retry/replay/dedup — OmniRoute; Main Brain validates only its own process cancel/slot cleanup **when changed** (that part lives under 8.2/A6, not 8.6).
- 8.7 = account add/remove/quarantine/re-entry + OmniRoute restart/config rollback — OmniRoute.

## 3. Evidence-overlap dedup (one run per family closes many requirements)

| Route family | Live token | Focused checks | Requirements closed by the **same** evidence |
|---|---|---|---|
| Cline → GLM-5.2 | `cline_glm` (A9-GLM, BLOCKED) → `evidence/R1-glm-live-acceptance.md` | D3, D3b | 5.6 · 8.1 · 8.2 |
| Cline → Kimi-K2.7 | `cline_kimi` (reuses cline base) | D4 (=reuse D3/D3b) | 5.7 · 8.1 · 8.2 (only where not already equivalent) |
| Kiro/Opus48 | `opus48` (A9-OPUS48, BLOCKED) → `evidence/R2-opus48-live-acceptance.md` | D5 | 5.8(Opus48) · 8.1 · 8.2 |
| Antigravity | `antigravity` | D6 = A5 single scenario (STALE) | 5.8(Agy) · 8.1 · 8.2 (reuse where equivalent) |

Already-accepted, **not reopened** (context only): 8.3 (EV-G4-03 child-env isolation, synthetic/reference), 8.4 (RR/affinity — sanitized northbound artifact + gateway Selector tests, independent ACCEPT w7:p4). No Main-Brain token for 8.5–8.7.

## 4. Consolidated blocker register (owner + required action)

| ID | Blocks | Owner | Required action |
|---|---|---|---|
| BLK-KIMI | 5.7, 8.1, 8.2 (Kimi) | OmniRoute architect (via Principal) | publish exact versioned Kimi `RouteModel` + protocol(openai-chat) + availability |
| BLK-AVAIL | 5.6/5.7 live, 8.1 | OmniRoute architect | publish enriched `/v1/models` rows for `cp/cline-pass/glm-5.2` (and Kimi ID): protocol/stream/tools/reasoning/context/pool/available |
| BLK-OPUS48-ID | 5.8(Opus48), 8.1, 8.2 | OmniRoute registry owner | publish exact Opus48-on-AWS `RouteModel` row (anthropic-messages capability fields + provenance) |
| BLK-AGY-STALE | 5.8(Agy), 8.1, 8.2 | W1 (R2 live token) + Owner | reserve/execute A5's single scenario on HEAD `a6d5` **after** security stop clears |
| D-V3-25(B) | **all** live runs | Owner | confirm UI invalidation/revocation of exposed key (offline/synthetic work may proceed) |
| FLEET_SATURATED (now GREEN) → lock activation | source-lock activation + integration + tokens | Principal / W1 | gate GREEN + A7/A8 DONE + W1 freeze DONE; Principal to **activate** F1–F25 locks + set `implementation_authorized=true`, authorize W1 serial integration, then issue one token per family |
| DEC-NIM | 5.7 scope | Principal / W1 | ruling: is NIM-direct `z-ai/glm-5.2` default (`models.go:572`,`nim.go:25`) an obsolete direct-provider alternative to retire per 5.7? (A3 REVIEW-1 / W1 F4) |
| DEC-CLIKIND | 5.6/5.7 impl shape | Principal / W1 | ruling: reuse `brain.CLIOpenAICompatible` (exec=`cline`) vs add new `brain.CLICline`? Determines exact edits F14/F15/F18/F20 (A1+A2/R1-DESIGN recommend reuse) |
| Go path discrepancy | preflight note (non-blocking) | toolchain owner | canonical `/home/ec2-user/goroot/go/bin/go` confirmed (go1.26.1); `.local` path superseded |

## 5. Orphan / hygiene check (A10 scope)
- No requirement is orphaned: 5.6/5.7/5.8/8.1/8.2 each map to ≥1 lane artifact above. 8.5–8.7 correctly externalized (no Multica owner — intended).
- No duplicate acceptance: overlap collapsed to 4 tokens (§3); no QA-A/B/C, no broad regression, no 2nd live acceptance.
- **Single-gap convergence (dedup win):** two independent lanes (A1+A2 gaps **C5/C7** and A6 gap **L3c**) identify the **same** one real Main-Brain implementation gap — credentialless Cline (`CLIOpenAICompatible`) adapter flip + `providers.json` launch-materialization + `buildLaunch` branch. It is one W1-integrated delta closing 5.6, 5.7 (increment) and the Cline slice of 8.2. Everything else on the Kanban→terminal path is Category-1 reuse or external. This is the crisp implementation target for the Principal, not five separate builds.
- No fabricated evidence/ID: every route ID is A3/A4-frozen or explicitly BLOCKED_EXTERNAL; Antigravity decision is hash-pinned (A5).
- No checkbox marked here; OpenSpec/GSD closure remains Principal-only after concrete implementation + accepted live/reused evidence.

## 6. FLEET_SATURATED coverage fact (for Principal — not an A10 adjudication)
Eligible workers = 8. **All 8 now have a check-in with preflight** (A7 joined 22:42:33Z): A1+A2, A3, A4, A5→A10, A6, W1+A9, A7, A8 — every worker is `working|blocked` with its own artifact lock, and active file locks show **zero intersection** (confirmed by W1's F1–F25 pairwise-∅ proof in `W1-source-lock-freeze.md §4`). The FLEET_SATURATED gate is now **GREEN** (all 8 workers checked-in with preflight, A7/A8 delivered, zero lock overlap): `control.json` reads `fleet_saturation_gate.status=GREEN`, `assignment_phase=SOURCE_LOCK_FREEZE` (as of 23:03Z). A8 closed **DONE — NO_REACHABLE_RESIDUAL**. **Source edits have NOT yet been authorized/activated:** `implementation_authorized=false` and the F1–F25 disjoint locks are not yet activated, so **product source remains untouched** — as of the last validation-lane readiness scans, `WriteCredentiallessClineConfig` is still **absent** and `adapter.go` still returns `AdapterFailClosed` for `CLIOpenAICompatible` (L3c/C5/C7 unimplemented). `W1 P0-W1-INTEGRATION` is **BLOCKED** awaiting source-lock activation; the four A9 validation lanes remain **BLOCKED** on lock activation (A9-R1/A9-R3) and on `live_runs.*.authorized=false` + external RouteModel (A9-GLM/A9-OPUS48) — matching §4.
- **Remaining before source edits may land (Principal-owned):** (1) **activate** the F1–F25 disjoint source locks and set `implementation_authorized=true` (gate is GREEN and `W1-source-lock-freeze.md` is ready to enact); (2) resolve rulings **DEC-NIM** and **DEC-CLIKIND**; (3) authorize/run W1 serial integration → produces the integrated + focused-test evidence (unblocks A9-R1/A9-R3); (4) issue live-run tokens per family (unblocks A9-GLM/A9-OPUS48). A10 records these facts; the Principal adjudicates and owns lock activation.

## 7. Living-doc note
This map reflects lane state as of `2026-07-21T23:03Z`. **FLEET_SATURATED is GREEN**; A8 closed **DONE — NO_REACHABLE_RESIDUAL**; all design/test-contract deltas (R1/R2/R3) and W1 source-freeze are **DONE**. **Source edits are not yet authorized/activated** (`implementation_authorized=false`, phase `SOURCE_LOCK_FREEZE`); `W1 P0-W1-INTEGRATION` and the four A9 validation lanes are **BLOCKED** awaiting source-lock activation + tokens; product source is confirmed untouched. **Next A10 refresh triggers (state re-read only — no test rerun, no second review):** (1) Principal activates F1–F25 locks + `implementation_authorized=true`; (2) W1 serial integration produces integrated + focused-test evidence (unblocks A9-R1/A9-R3); (3) rulings DEC-NIM / DEC-CLIKIND; (4) `live_run` token issuance + D-V3-25(B) clearance (unblocks A9-GLM/A9-OPUS48 → `R1-glm-live-acceptance.md` / `R2-opus48-live-acceptance.md`); (5) OmniRoute publication of BLK-KIMI / BLK-AVAIL / BLK-OPUS48-ID. On W1 integration/focused-test evidence, A10 appends final refs and checks out. A10 never marks the OpenSpec/GSD checkbox.

## 8. Update `2026-07-21T23:38Z` — implementation-serial phase + D6 adjudication

**Gate flipped to GREEN and source implementation authorized.** `control.json`: `assignment_phase=IMPLEMENTATION_W1_SERIAL`, `implementation_authorized=true`, `fleet_saturation_gate.status=GREEN`. W1 source locks were **activated** (`Opus48#B / W1-LOCKS / P0-W1-SOURCE-LOCK-ACTIVATION` DONE 23:09:31Z). A7 & A8 production-integrity both **DONE** (A8 = NO_REACHABLE_RESIDUAL); their `*-EVIDENCE` (terminal UI/backend) sub-lanes are BLOCKED pending the live run.

### 8.1 D6 pre-launch adjudication delivered (`handoffs/D6-prelaunch-adjudication.md`, sha256 `610a3bb5…`)
`Opus48#C / A10-DESIGN / P0-D6-PRELAUNCH-ADJUDICATION` **DONE** (23:38:26Z). Verdict: **`assert.go` MUST change** — `AssertPreLaunch` fail-closes `CLIOpenAICompatible` (Cline) in 3 places, so the accepted Cline route cannot launch despite adapter/env/execenv/cline.go being implemented. **Minimum fail-closed contract = 4 edits / 2 files:**
- **D6-1** `assert.go AssertPreLaunch` final CLI switch: add `case brain.CLIOpenAICompatible: if plan.CodexConfig != nil { return ErrPreLaunchPolicy }`.
- **D6-2** `assert.go trustedEntryAllowed`: add `CLIOpenAICompatible` case allowing `HOME`(trustedLocal), `CLINE_DATA_DIR`(trustedLocal), `CLINE_OMNIROUTE_API_KEY`(trustedSecret) — else the denied trusted keys are rejected.
- **D6-3 (security-material)** `assert.go launchRootsAreControlled`: add Cline data-dir containment `exactPathWithin(executionRoot, environment.clineDataDir, entries["CLINE_DATA_DIR"].value)`.
- **D6-4** `env.go`: add **`ChildEnvironment.clineDataDir`** field + populate in `BuildGatewayEnvironment` — **corrects R1-DESIGN D2's "no ChildEnvironment field needed"**; the pre-launch containment gate is exactly the readback consumer.
- CLINE_DATA_DIR validation is two-layer (compose-time `validatePhysicalControlledDirectory` already done + new pre-launch `exactPathWithin`); config-byte validation stays in `ValidateClineConfigBytes` (assert.go stays content/secret-free). 8 fail-closed stop conditions enumerated.

### 8.2 Lock expansion (recommended to `W1-source-lock-freeze.md`)
- **F-ASSERT (new)** — `internal/daemon/runtimeenv/assert.go` (`AssertPreLaunch`/`trustedEntryAllowed`/`launchRootsAreControlled`) + `assert_test.go` — **W1-exclusive serial**, Wave-C step 3 (Cline) **before** the reserved GLM/Kimi live run.
- **F14 (amend)** — `internal/daemon/runtimeenv/env.go`: add `ChildEnvironment.clineDataDir` field + populate. W1-exclusive; pairwise-∅ preserved (single W1 editor, same package, serial).

### 8.3 Current implementation/validation lane state
| Lane | Task | State | Note |
|---|---|---|---|
| W1 (Opus48#B) | P0-W1-INTEGRATION | DONE (23:05) | serial integration seam wired |
| W1 (Opus48#B) | P0-W1-SOURCE-LOCK-ACTIVATION | DONE (23:09) | F-locks activated; `implementation_authorized=true` |
| W1 (Opus48#B) | **P0-W1-IMPLEMENTATION** | **BLOCKED (55%, 23:33Z)** | Cline adapter/env/execenv/cline.go landed; **pre-launch `AssertPreLaunch` gap = the D6 contract §8.1** is the outstanding integrator work before the Cline route launches |
| A9-R1 (Opus48#A) | P0-R1-TEST-IMPLEMENTATION | DONE (23:22) | Cline focused tests implemented under activated test locks |
| A9-R1 (Opus48#A) | **P0-R1-FINAL-INTEGRATION-VALIDATION** | **BLOCKED (23:23:58Z)** | awaits W1 integrated build (incl. D6 assert edits) before the one focused validation pass |
| A9-R3 (Opus48#D) | P0-LIFECYCLE-TEST-IMPLEMENTATION | DONE (23:23) | lifecycle focused tests implemented |
| A9-R3 (Opus48#D) | **P0-R3-TEST-LOCK-ACTIVATION** | **BLOCKED (23:25:57Z)** | awaits W1 integration/lock coordination; T-ASSERT set must expand for D6 containment/trusted-entry cases (§6 of D6 handoff) |
| A9-GLM (Codex56#A) | P0-GLM-LIVE-ACCEPTANCE | BLOCKED | `live_runs.cline_glm.authorized=false` + BLK-AVAIL; **also gated by D6** (AssertPreLaunch fail-closes Cline until F-ASSERT/F14 land) → `evidence/R1-glm-live-acceptance.md` |
| A9-OPUS48 (Codex56#B) | P0-OPUS48-LIVE-ACCEPTANCE | BLOCKED | external Opus48 RouteModel + `live_runs.opus48.authorized=false` → `evidence/R2-opus48-live-acceptance.md` |

### 8.4 Critical path now (for the Principal / W1)
1. **W1 lands D6** (F-ASSERT + F14 amend) → unblocks `AssertPreLaunch` for Cline, completing `P0-W1-IMPLEMENTATION`.
2. R1 `P0-R1-FINAL-INTEGRATION-VALIDATION` + R3 `P0-R3-TEST-LOCK-ACTIVATION` run their **one** focused pass on the integrated build (no duplicate validation).
3. Principal issues `live_runs.cline_glm` token → single GLM Kanban→terminal run closes 5.6/8.1/8.2 (Kimi rides on BLK-KIMI clear; Opus48 on BLK-OPUS48-ID + token; Antigravity per A5 STALE scenario under D-V3-25(B)).
Until step 1, the Cline live path is **fail-closed by design** (safe, but blocking) — this is the concrete pre-live blocker.
