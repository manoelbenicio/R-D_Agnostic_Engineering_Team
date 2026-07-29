# A6 — Main-Brain lifecycle gap matrix (Kanban → terminal)

- agent: Opus48#D  · lane: A6 · task: P0-LIFECYCLE-GAP
- pane: `w8:p2` (HERDR_ENV=1) · git HEAD: `a6d5098`
- check-in: `.deploy-control/p0/checkins/Opus48-D__P0-LIFECYCLE-GAP__20260721T223028Z.json`
- scope: trace the real squad/project/Kanban → admission → workspace → launch → events →
  terminal-persistence/UI → cancel → cleanup path; classify each item into the five plan
  categories; identify exact source candidates + focused tests; cover the Main-Brain-owned
  portion of task `8.2`. **OmniRoute internals are out of scope by contract.**
- posture: **READ-ONLY on product source.** This handoff is the only mutable artifact. No
  product code, no OpenSpec/GSD edits, no checkbox, no live run.

## 0. Preflight (recorded per PROTOCOL)

| Tool | Version / path | Result |
|---|---|---|
| git | 2.50.1 | OK |
| python3 | 3.9.25 | OK |
| rg | ripgrep 15.2.0 | OK |
| go | `/home/ec2-user/goroot/go/bin/go` → `go1.26.1 linux/amd64` | OK (canonical path per corrected ASSIGNMENTS/control.json) |
| gofmt | `/home/ec2-user/goroot/go/bin/gofmt` | present |

Path note: the pane's `$HOME` is a credential slot; the earlier ASSIGNMENTS path
`$HOME/.local/toolchains/go1.26.1/bin/go` and the interim `/home/ec2-user/.local/toolchains/...`
path do **not** resolve. The corrected canonical toolchain is `/home/ec2-user/goroot/go/bin/`
(go1.26.1 verified). This is **non-blocking for A6** — A6 is a read-only trace and compiles
nothing. Recorded for W1/A9, who do compile.

## 1. Ownership boundary applied to this trace

- **Main-Brain-owned (in A6 view):** task admission/readiness gating, launch-identity gating,
  workspace/worktree prep, credentialless launch env + controlled config materialization,
  redacted lifecycle events, terminal-result mapping/reporting, process cancellation, slot/capacity
  reconciliation, and env cleanup/GC.
- **OmniRoute-owned (Category 4, excluded):** authentication, credential lifecycle, token/window
  limits, quota, 401/403/429/5xx circuits, retry, account selection/rotation/quarantine, provider
  fallback (incl. NVIDIA). Main Brain only *reads* pseudonymous telemetry; it implements none of this.
- **Phase 3 / HOLD (Category 5):** Prodex/L2 cold-recovery.
- **Deferred / out of P0 (not a gap):** the eight-hop E2E observability wave `OBS-1..OBS-11` is
  Priority-2 / DEFERRED behind P0 (FILE_OWNERSHIP D-V3-21) and does **not** block Kanban→terminal.

## 2. Architecture facts that frame every classification

1. The gateway-required Agent Brain path is a **default-off development slice**
   (`agentBrainDevelopmentMaxTasks=1`; enabled only when
   `AgentBrain.DevelopmentEnabled && Neutral.Gateway.Required && !Neutral.LegacyExecution` —
   `daemon.go:993 agentBrainGatewayRequired`, `brain_integration.go:enabled`). In default
   production `agentBrainPlan == nil` and the legacy credentialed path runs. Enabling the slice
   by default is Wave-5 cutover `10.1`, **not P0**.
2. The **live execution driver is the inline `runTask`/`handleTask` path**, not the neutral
   `brain.Coordinator`. `brain.Coordinator`/`LifecycleTaskExecutor`/`PreservedLifecycle`/`ResultSink`
   (`brain/coordinator.go`, `brain/executor.go`) are the G2 strangler contracts — fully unit-tested
   but wired only in tests. The inline daemon path **is** the `PreservedLifecycle`. This is by
   design (G2/G3), **not a functional gap** — flagged so nobody "fixes" it by re-routing the live
   path during P0.
3. `CLIKind` (`brain/identity.go`) has **no** `CLICline`/`CLIKiro`. The P0 routes map to existing
   kinds: Cline→GLM (5.6) and Cline→Kimi (5.7) ride `CLIOpenAICompatible`/`CLIKimi`;
   Kiro/Opus48 (5.8) rides `CLIClaudeCode`.
4. `runtimeenv.CredentiallessAdapterContract` (`adapter.go`) is `AdapterReady` for
   `CLIClaudeCode` (anthropic-messages) and `CLICodex` (openai-responses); it is
   **`AdapterFailClosed`** for `CLIOpenAICompatible` (`GateOpenAICompatibleUnaccepted`) and
   `CLIKimi` (`GateNativeKimiUnaccepted`). ⇒ the 5.6/5.7 Cline routes **cannot be admitted today**.

## 3. Real caller spine (exact symbols)

```
service/autopilot.go, service/task.go  (squad/project/Kanban task create + queue/claim; server-side)
  └─ daemon.handleTask                         daemon.go:2829
       ├─ acquireLocalDirectoryLockIfNeeded    daemon.go:3000  (local_directory serialization)
       ├─ markActiveEnvRoot (GC guard)         daemon.go:2887
       ├─ runCtx,runCancel + watchTaskCancellation → cancelledByPoll   daemon.go:2901-2919
       └─ runner.run = runTask                 daemon.go:3288
            ├─ workspace_id gate                             daemon.go:3295
            ├─ resolveTaskAgentEntry (launch-identity gate)  daemon.go:3248
            │     gateway-required: rejects custom runtime/args; resolves built-in CLIKind
            ├─ agentBrain.admitTask                          daemon.go:3323 → brain_integration.go:admitTask
            │     capacity.TryBegin → CredentiallessAdapterContract → LookupRuntimeProfile
            │     → legacy.TranslateTask → gateway readiness (Admit) → ValidateCapability(Tools)
            ├─ if plan: RouterOwner=omniroute; entry.Model=RouteModel; defer recordTerminal  :3300-3306
            ├─ legacy proactive rotation DISABLED when plan != nil       :3346
            ├─ execenv.Prepare / execenv.Reuse (CredentiallessGateway=plan!=nil)  :3455/3470
            ├─ client.StartTask → running; ReportProgress                :3520/3536
            ├─ InjectRuntimeConfig (runtime brief)                       :3546
            ├─ agentBrain.buildLaunch (controlled home + child env + config + AssertPreLaunch)  :3701
            ├─ agent.New backend                                         :3713
            ├─ agentBrain.validateThinking (reasoning gate)              :3762
            ├─ agentBrain.recordLaunch (capacity.Start; EventRouteSelection)  :3828
            ├─ executeAndDrainForTask (stream/usage/cancellation)        :4416
            └─ TaskResult switch (completed/blocked/timeout/idle_watchdog/cancelled)  :3896
  ← back in handleTask:
       ├─ client.ReportTaskUsage                daemon.go:2925
       ├─ cancelledByPoll → discard result      daemon.go:2936
       ├─ reportTaskResult → client.CompleteTask/FailTask (fail-closed: only "completed"=success)  daemon.go:3112
       ├─ execenv.WriteGCMeta                    daemon.go:2970
       └─ recordTerminal (deferred: capacity.Finish; EventCancellation)  brain_integration.go:recordTerminal
  UI delivery: client(HTTP) → service/task.go → Postgres → daemonws WS hub → web (server-side, unchanged)
  Cleanup:     gc.go GC loop + markActiveEnvRoot guards + GCOrphanTTL + execenv Cleanup*
```

## 4. Gap matrix — five plan categories

Category legend: **1**=implemented + equivalent evidence (reuse, no run); **2**=implemented, no
equivalent evidence (one live run — folds into the single authorized per-family acceptance, run by
W1); **3**=real Main-Brain gap (implement + validate); **4**=OmniRoute (remove from lane);
**5**=Phase 3/HOLD.

| # | Lifecycle item | Primary symbols | Cat | Rationale / equivalent evidence | Focused test (minimal) |
|---|---|---|---|---|---|
| L0 | Squad/project/Kanban create + claim; CLIKind/RouteModel/policy intent onto task | `service/autopilot.go`, `service/task.go`; `brain.TranslateTask` (`brain/compatibility.go`); `TaskRequest`/`Correlation` (`brain/contracts.go`) | 1 | Existing production; unchanged by P0. Intent→neutral contract covered. | `daemon/brain_integration_test.go`; `daemon/config_test.go` (existing) |
| L1 | Admission + readiness gate + deterministic error taxonomy (8.2 *errors*) | `brain/admission.go GatewayAdmissionController.Admit`; `brain_integration.go admitTask`; `daemon.go resolveTaskAgentEntry:3248`; `observeAgentBrainAdmission`; `brain/admission_observability.go` | 1 | Fail-closed classes + observed-once span proven. | `brain/g2a_test.go`, `brain/admission_observability_test.go`, `daemon_obs4_test.go`, `brain_integration_test.go` (FailsClosedWhenGatewayNotReady, RejectsDualRouter, UsesInstalledOmniRouteHealthContract) — existing |
| L1b | Live admission of the **ready** 5.8 route against real OmniRoute readiness | same as L1 + registry `ValidateCapability` | 2 | Mechanism done; G4 was synthetic/reference-only. One live admission during the single W1 acceptance. | reuse W1 live run; no new unit test |
| L2 | Workspace/worktree/repo prep; credentialless flag | `execenv.Prepare`/`Reuse`; `PredictRootDir`; `registerTaskRepos`; `markActiveEnvRoot`; `CredentiallessGateway` | 1 | No lifecycle change beyond credentialless flag; isolation proven. | `execenv/execenv_test.go`, `execenv/sidecar_manifest_test.go`, `workdir_race_test.go`, `runtime_isolation_test.go` — existing |
| L3a | Launch of **CLIClaudeCode** (5.8 Kiro/Opus48): env-only trusted root/token, AssertPreLaunch | `adapter.go CredentiallessAdapterContract (AdapterReady)`; `runtimeenv.BuildGatewayEnvironment`; `AssertPreLaunch`; `buildLaunch` (env branch) | 1 | Adapter ready; no file-config needed; synthetic child + pre-launch assertions proven. RouteModel gap is A3/A4, not lifecycle. | `brain_integration_test.go` (SyntheticChild, IsolationSmoke), `runtimeenv/assert_test.go` — existing |
| L3b | Launch of **CLICodex**: `WriteCredentiallessCodexConfig` materialization | `brain_integration.go:338`; `execenv/codex_home.go WriteCredentiallessCodexConfig`; `runtimeenv/codex.go` | 1 | Codex config contract materialized + asserted. | `runtimeenv/codex_test.go`, `assert_test.go` — existing |
| **L3c** | **Launch of CLIOpenAICompatible/CLIKimi (5.6/5.7 Cline): credentialless config NOT materialized; adapter fail-closed** | `adapter.go` (`GateOpenAICompatibleUnaccepted`/`GateNativeKimiUnaccepted`); `cline.go NewClineConfigContract` (contract-only); **absent** `WriteCredentiallessClineConfig`; `buildLaunch` (no Cline branch) | **3** | `providers.json` contract fully built + unit-tested but never (a) accepted by the adapter, (b) written to a controlled Cline data-dir at launch, (c) branched in `buildLaunch` with sentinel→secret injection. This is the one real Main-Brain **lifecycle** gap blocking 5.6/5.7. **See §6 handoff — A6 does not implement.** | new: `runtimeenv` writer test + `brain_integration_test.go` synthetic-child case for `CLIOpenAICompatible` |
| L4 | Redacted lifecycle events (admission/readiness/route-selection/cancellation) | `observability.SafeEvent`; `brain_integration.go emit/recordAdmission/recordLaunch/recordTerminal`; `observability/schema.go` | 1 | The redacted events the P0 flow needs exist + validate; secrets-absent enforced. Full 8-hop E2E (OBS-*) is DEFERRED, out of P0. | `observability/observability_test.go`, `brain/admission_observability_test.go`, `daemon_obs4_test.go` — existing |
| L5 | Terminal-result mapping + fail-closed reporting + usage (8.2 *usage/terminal result*) | `runTask` TaskResult switch `:3896`; `reportTaskResult:3112` → `client.CompleteTask/FailTask`; `ReportTaskUsage`; `mergeUsage`; `recordTerminal` (capacity.Finish) | 1 | Only `"completed"`→success; everything else fail-closed via `FailTask`; usage reported on all paths incl. cancel; terminal correlation preserved. | `daemon_test.go` (MergeUsage, ExecuteAndDrain_ContextCancelled_ReportsCancelled), `client_test.go` (DefaultTerminalRetrySchedule), `service/task_complete_race_test.go`, `brain/g2a_test.go` (CoordinatorPreservesCancellationAndTerminalResult) — existing |
| L5b | Terminal result + usage delivered to UI for the affected family | L5 symbols + `service/task.go` + `daemonws` hub (server-side) | 2 | Delivery mechanism unchanged/proven; confirm once per changed family in the single W1 live run. | reuse W1 live run |
| L6 | Cancel: poll→cancel, drain-cancelled, slot/capacity release (8.2 *cancellation*) | `watchTaskCancellation`; `runCancel`; `cancelledByPoll`; `executeAndDrainForTask` cancelled path; `brain/coordinator.go` (TaskStatusCancelled, publish `WithoutCancel`); `brain/capacity.go`; `recordTerminal` EventCancellation | 1 | CLI-agnostic; cancel + slot release proven. Per 8.2, extra cancel proof only if Brain lifecycle changed — it hasn't for ready routes. | `daemon_test.go` (WatchTaskCancellation_TaskDeleted/StatusCancelled/RunningTaskNotInterrupted, ExecuteAndDrain_ContextCancelled), `brain/capacity_test.go` (reconciles…Cancellation…, LedgerAcceptsPreStartFailureAndCancellation), `gateway/dispatch_test.go` (CoordinatorCancellationReleasesSlot) — existing |
| L7 | Cleanup: env-root GC, sidecar/runtime-config cleanup, GC meta | `gc.go` (GC loop, `shouldCleanTaskDir`, `GCOrphanTTL`); `markActiveEnvRoot`/`unmark`; `WriteGCMeta`/`gcMetaForTask`; `execenv` `CleanupSidecars`/`CleanupRuntimeConfig` | 1 | Env-root GC + active-root guard + sidecar/config cleanup proven. | `gc_test.go` (ActiveEnvRootSkips*, CancelledIssueOverTTL, CleansEmptyWorkspaceDir), `execenv/sidecar_manifest_test.go` (PrepareThenCleanup*), `execenv/runtime_config_test.go` cleanup cases — existing |
| L7b | Controlled task home (`agent-brain-home`, 0700) reclamation with env root | `buildLaunch` (creates `agent-brain-home` under `env.RootDir`); `gc.go` | 2 | Created under `RootDir` so root GC covers it, but **no explicit test** asserts the controlled home is reclaimed. Add one unit test; no live run. | new: `gc_test.go` (or `brain_integration_test.go`) asserting `agent-brain-home` removed with env root |
| L8 | Legacy credential/account/rotation + provider fallback | `credentialAccountHomeForTask`, `maybeProactiveRotateFromLedger`, `rotateTaskOnExhaustion`, `startL2SessionForTaskWithCredentialHome`, `execenv/cline_home.go prepareClineHome` (legacy credentialed), `rotation_detector_*`; gateway `telemetry.go`/`selection.go`/`dispatch.go`; NVIDIA rejection `cline.go isNVIDIAOwnedRoute` | 4 | OmniRoute-owned internals. **Main-Brain-owned part = the gating**: all legacy rotation/credential-home is disabled when `RouterOwner=omniroute` (task 7.7) — that gate is implemented + logged. Internals themselves stay out of the lane. | gating: existing `brain_integration_test.go` (RejectsDualRouter) + `daemon.go:3346` disable log — no new work |
| L9 | Prodex/L2 cold-recovery state machine | `brain/recovery.go RecoveryMode`; `prodex.go`, `l2_runtime.go`, `prodex_*` | 5 | Default-OFF, never per-request, never auto-promoted; task 10.4 HOLD. | `brain/recovery_test.go` (fails-closed, never auto-promotes Prodex) — existing; no work now |

## 5. Main-Brain-owned portion of task 8.2 (explicit)

`8.2` (tools, reasoning, cancellation, usage, terminal result, deterministic errors). Provider-side
tool/reasoning protocol conformance is `[Codex 3 + Codex 4]` / OmniRoute; the **Main-Brain-owned**
slice is:

| 8.2 item | Main-Brain responsibility | Symbol | Cat | Note |
|---|---|---|---|---|
| tools | assert model advertises tools before launch | `registry.ValidateCapability(…, Tools:true)` in `admitTask` | 1 (+2 live confirm) | conformance-on-the-wire is gateway/OmniRoute |
| reasoning | gate reasoning/thinking intent | `validateThinking` → `runtimeenv GatewayModelPolicy.ValidateSelection`; `ErrThinkingNotApproved` | 1 | **decision flag:** slice currently *rejects* non-empty thinking. If a P0 route must enable reasoning, that is a policy/registry-capability change owned by A3/A4 + W1 — not an A6 lifecycle gap. |
| cancellation | process cancel + slot cleanup | see L6 | 1 | re-run only if the family's launch branch changes (folds into L3c) |
| usage | collect + report on every path | see L5 | 1 (+2 live confirm) | |
| terminal result | fail-closed terminal mapping + persistence | see L5 | 1 (+2 live confirm) | |
| deterministic errors | typed fail-closed admission classes | see L1 | 1 | |

## 6. W1 handoff — the single real lifecycle gap (L3c)

**Do not let A6 implement this.** It requires an A1/R1 contract + a W1 hotspot edit; serialized by W1.

- **Requirement:** admit and launch the OmniRoute-selectable Cline family (5.6 `Cline→GLM-5.2`,
  5.7 `Cline→Kimi-K2.7`) credentiallessly, materializing `providers.json` into a controlled Cline
  data-dir before process spawn, with the stable OmniRoute secret injected at launch (never embedded).
- **Exact integration points:**
  1. `runtimeenv/adapter.go` — `CredentiallessAdapterContract` must return `AdapterReady`
     (protocol `openai-chat-completions`) for `CLIOpenAICompatible` once A3 freezes the exact GLM/Kimi
     RouteModel and the OmniRoute Chat-Completions contract is accepted. **Owner: A1/R1 (+A3 for IDs).**
  2. `runtimeenv/cline.go` / `execenv` — add a `WriteCredentiallessClineConfig` writer + a controlled
     per-task Cline data-dir path (analogous to `env.CodexHome` + `WriteCredentiallessCodexConfig`),
     honoring the `${CLINE_OMNIROUTE_API_KEY}` sentinel. **Owner: A1/R1.**
  3. `brain_integration.go buildLaunch` (~line 338, currently a Codex-only branch) — add the
     `CLIOpenAICompatible` branch that writes the Cline config and wires `CLINE_DATA_DIR` + the
     sentinel→secret substitution. **Owner: W1 (this file is a W1 P0 hotspot).**
  4. `runtimeenv.AssertPreLaunch` — extend the manifest/assertion set to the Cline data-dir layout.
- **Focused tests (delta only):** `runtimeenv` writer/round-trip test; `brain_integration_test.go`
  synthetic-child case for `CLIOpenAICompatible`; one cancel-path re-run for the family folded into
  the single authorized W1 live acceptance. No broad regression, no second live run.
- **Blocked-external dependency:** exact GLM (5.6) and Kimi (5.7) `RouteModel` IDs are **A3**
  (`A3-route-freeze.md`); until frozen, L3c stays `BLOCKED_EXTERNAL` for admission acceptance.
  Kiro/Opus48 (5.8) `RouteModel` is **A4** — its launch lifecycle (L3a) is already ready.

## 7. Blockers / limitations

- **BLOCKED_EXTERNAL (not an A6 blocker):** 5.6/5.7 admission acceptance depends on A3's exact
  GLM/Kimi RouteModel + OmniRoute Chat-Completions acceptance; 5.8 depends on A4's Opus48 RouteModel.
  A6 records the lifecycle integration points; it does not invent IDs or flip the adapter.
- A6 ran **no** Go build/tests (read-only lane, no lock on source). Every "existing test" cited is
  named from source inspection; **W1/A9** own execution of the delta tests above.
- No auth/credential/quota/retry/failover analysis was performed or is owed (Category 4, OmniRoute).
- `OBS-1..OBS-11`, capacity tiers, cutover, debrand: out of P0 scope, not gaps.

## 8. Summary

The Kanban→terminal Main-Brain lifecycle is **substantially implemented and unit-proven**; most
stages are Category 1 (reuse equivalent evidence) with a handful of Category-2 confirmations that
**fold into the single authorized per-family live acceptance run by W1**. There is exactly **one
real Main-Brain lifecycle gap (L3c)**: credentialless Cline (`CLIOpenAICompatible`/`CLIKimi`)
admission + launch-config materialization for 5.6/5.7, owned by A1/R1 (+A3 IDs) and wired by W1.
5.8 (Kiro/Opus48) launch lifecycle is ready; its only gap is the A4 RouteModel. No OmniRoute
internals, no Prodex, no duplicate QA, no broad regression.
