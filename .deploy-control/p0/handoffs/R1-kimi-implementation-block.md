# R1 — Kimi implementation block (BLOCKED_EXTERNAL, read-only)

- agent: `Opus48#A`  ·  lane: `R1-KIMI`  ·  task: `P0-KIMI-IMPLEMENTATION`  ·  pane: `w6:p1`
- control lock (only mutable file): `.deploy-control/p0/handoffs/R1-kimi-implementation-block.md`
- status: **BLOCKED_EXTERNAL** — exact versioned Cline→Kimi-K2.7 `RouteModel` undeclared (BLK-KIMI).
- product source/tests: **READ-ONLY / no edits now**. No ID invention, no source edit, no live run.
- created: 2026-07-22T00:03Z (UTC) · covers OpenSpec `5.7` (+ overlapping `8.1`/`8.2` for Kimi only).
- toolchain: `/home/ec2-user/goroot/go/bin/{go,gofmt}` (go1.26.1).

---

## 1. Preflight (exact, this pane, 00:03Z)

git `2.50.1` · python3 `3.9.25` · rg `15.2.0` · herdr `0.7.4` (HERDR_ENV=1, pane `w6:p1`) ·
go `go1.26.1` (`/home/ec2-user/goroot/go/bin/go`) · gofmt present. All green.

## 2. Completed GLM reuse path (what Kimi reuses — already integrated & green)

The shared Cline carrier is **fully integrated and green** for GLM (W1 D1–D7 + R1 tests validated):
- `runtimeenv/adapter.go` — `CLIOpenAICompatible` → `AdapterReady` + `ProtocolOpenAIChat` (D1).
- `runtimeenv/env.go` — `AdapterEnvironment.ClineDataDir`; trusted-last `CLINE_DATA_DIR` +
  `CLINE_OMNIROUTE_API_KEY` injection (D2).
- `internal/daemon/config.go` — `agentBrainBuiltInCLIFor`/`Validate` accept `CLIOpenAICompatible`→`cline` (D3/D4).
- `execenv/cline_home.go` — `WriteCredentiallessClineConfig` carrier writer (D5).
- `brain_integration.go` — `buildLaunch` Cline branch: `NewClineConfigContract` + `ClineDataDir` (D6).
- `pkg/agent/models.go:562` — GLM `cp/cline-pass/glm-5.2` (D7 / A3 SUB-1).
- `runtimeenv/cline.go` — `NewClineConfigContract` is **ID-agnostic** (parameterized by `brain.RouteModel`).
- R1 focused tests green: `go test ./internal/daemon/runtimeenv/ -run 'Adapter|Env|Model|Cline|G4' -count=1` → `ok` (per `R1-test-implementation.md` / `R1-final-integration-validation.md`).

**Consequence:** Kimi (5.7) needs **no new base implementation**. The frontend (`CLIOpenAICompatible`/
`cline`), protocol (OpenAI Chat), env-injection, carrier writer and launch branch are the same code
already proven for GLM. The **only** missing input is the exact versioned Kimi `RouteModel` string
(what goes into `settings.model` / the `brain.RouteModel` passed to `NewClineConfigContract`), plus a
Kimi catalog row and the enriched-registry availability.

## 3. Blocker — verified (not assumed)

**Blocker (BLK-KIMI):** the exact versioned Cline→Kimi-K2.7 `RouteModel` is **UNDECLARED**. Verified
read-only at HEAD (00:03Z):
- `A3-route-freeze.md`: `Cline → Kimi-K2.7 (5.7)` = **UNDECLARED / BLOCKED_EXTERNAL**; three divergent
  Multica-side literals exist (`cline-kimi-k2.7-dedicated` [combo alias], `cline-pass/kimi-k2.7-code`
  [not OmniRoute-attested], `claude_code_kimi_2.7_Code` [Claude Code family — must not conflate]);
  checklist `:172` = "Architect to confirm". **None may be chosen by plausibility.**
- `control.json` `live_runs.cline_kimi` = `{authorized:false, accepted_run:null}`.
- `pkg/agent/models.go:563` still `cline-pass/kimi-k2.7-code` (A3 SUB-2 = **HOLD**, not applied).
- No certified exact Kimi id present in any P0 handoff.

**Owner:** OmniRoute registry / architect (via Principal Orchestrator).

**Next action (on certified exact Kimi id):**
1. Consume the certified exact versioned Kimi `RouteModel` (+ protocol=OpenAI Chat + enriched
   availability) from the OmniRoute registry — **do not invent or default it**.
2. Acquire **only Kimi-specific fixture/catalog locks via W1** (the shared base is untouched):
   - `pkg/agent/models.go:563` + lockstep `pkg/agent/models_test.go:172` (A3 SUB-2) — **W1 hotspot**;
   - `internal/handler/agent_thinking_test.go:164` (coupled Kimi literal) — **W1**;
   - `runtimeenv/cline_test.go` Kimi fixture (`clineKimiRouteModelPENDING` → certified id) — **R1**;
   - `runtimeenv/model_test.go` `TestGatewayModelPolicyClineKimiHeldPendingExactID` — remove `t.Skip`,
     assert the certified id over OpenAI Chat via `CLIOpenAICompatible` — **R1**.
3. Run **focused tests only**: `go test ./internal/daemon/runtimeenv/ -run 'Adapter|Env|Model|Cline|G4' -count=1`
   and the lockstep `go test ./pkg/agent/ -run 'StaticModels'` (W1) — once each.
4. Then **one authorized Kimi live run** (after BLK-AVAIL enriched row + D-V3-25(B) clear and the
   Principal issues the `cline_kimi` live-run token) — closes 5.7 + overlapping 8.1/8.2.

**Do NOT now:** invent/guess/default any Kimi id, edit product source or tests, or launch a live run.

## 4. Agent status block

- STATUS: BLOCKED_EXTERNAL (readiness recorded; awaiting certified exact Kimi RouteModel)
- DELIVERED: preflight; completed GLM reuse-path inventory; verified BLK-KIMI evidence; exact unblock
  plan (Kimi-only fixture/catalog locks + focused tests + one authorized run).
- FILES: `.deploy-control/p0/handoffs/R1-kimi-implementation-block.md` (only mutable file).
- VALIDATION: none now (blocked; no edit/run/live).
- EVIDENCE: this artifact; A3 freeze + control.json + models.go citations (§3).
- BLOCKERS: BLK-KIMI (exact Kimi RouteModel undeclared) — owner OmniRoute registry/architect.
  Related downstream: BLK-AVAIL (enriched registry), D-V3-25(B) (live-run security stop).
