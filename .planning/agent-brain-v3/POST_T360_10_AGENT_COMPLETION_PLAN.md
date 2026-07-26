# Post-T+360 Main Brain completion plan — 10 workers + Kiro + Principal

Updated: 2026-07-22T11:23Z  
Authority: Principal Orchestrator  
Execution manager: Opus48-Kiro (`w5:p1`)  
Scope: OpenSpec `build-omniroute-agent-brain`, currently 21/28

## 1. Objective and non-negotiable gates

Finish all technically actionable offline work for 5.2, 5.3, 6.1 and 6.2 in the next 90–150 minutes, prepare objective 6.3 evidence, and leave 6.4/6.5 only on explicit external gates. This plan does not authorize deploy, restart, Docker/systemd, provider/model inference, secret access, dependency installation, commit/push, destructive Git operations or fabricated acceptance. `live_runs.*.authorized` remains false.

Manager and Principal do not write product code. Kiro dispatches, monitors, routes findings and reports; Principal adjudicates and closes OpenSpec only from evidence. Producer != evaluator.

## 2. Full pane scan result

| Pane | Actual process/state | Role now |
|---|---|---|
| `w5:p1` | Kiro manager | management-only |
| `w5:p9` | Principal | adjudication-only |
| `w5:pA` | corrected `p0_control monitor --interval 600` | monitor-only |
| `w6:p1` | Kiro worker, R1 done | F1 producer/integrator |
| `w6:p2` | Kiro worker, clean standby | F2 test runner |
| `w7:p3` | Kiro worker, idle | F3 deployed-readiness verifier |
| `w7:p4` | Kiro worker, idle | F4 WS anchor producer |
| `w8:p1` | Kiro worker, idle | F5 ingress anchor producer |
| `w8:p2` | Kiro worker, idle | F6 queue/persist producer |
| `wB:p1` | Agy worker, idle | F7 route anchor producer |
| `wB:p2` | Agy worker/evaluator | F8 sole evaluator |
| `wK:p1` | new Agy worker, idle | F9 capacity harness producer |
| `wK:p2` | new Agy worker, idle | F10 CLI-hop producer |
| `wA:p1`,`wA:p2` | empty bash shells; no agent | reserve, not counted |

Usable worker agents: 10. Total active roles including Kiro and Principal: 12. The dormant GLM shells are activated only if a real disjoint failure lane emerges.

## 3. Current technical baseline

- R1 restored `newTestDaemon`, removed the stale daemon-level NIM isolation test after proving runtimeenv equivalence, `go vet ./internal/daemon` passes, and `go build ./...` passes.
- `go test ./internal/daemon -count=1` runs but has one remaining failure: `TestRunTask_StartTaskCalledAfterWorkdirOnDisk` (`workdir_race_test.go:149`).
- Root disk is 100% with about 189 MB free; `/tmp` is tmpfs with about 7.6 GB free. Full Go validation must use lane-specific `GOCACHE` and `GOTMPDIR` under `/tmp`; no cache/source deletion.
- 6.2 frozen contract and helpers exist: admission, route, ingress, queue, persist, delivery; admission is wired. Missing work is real call-site wiring plus a CLI-hop producer/call site.
- The 20-task synthetic harness exists but explicitly cannot claim capacity acceptance or host-sampled resources.

## 4. Ten disjoint lanes

| Lane/pane | Task | Exclusive mutable ownership | Acceptance | ETA |
|---|---|---|---|---|
| F1 `w6:p1` | Fix remaining daemon ordering failure; later wire CLI helper in `daemon.go` | `server/internal/daemon/daemon.go`, `workdir_race_test.go` | named test PASS, daemon package PASS; after F10, CLI call-site metadata-only and focused tests PASS | 30–60m |
| F2 `w6:p2` | 5.2 matrix + 5.3 server-wide build/test | evidence only: `.deploy-control/p0/evidence/F2-*`; product read-only | exact package matrix, `go build ./...`, then `go test ./... -count=1`; failures routed with file/symbol | 45–90m |
| F3 `w7:p3` | 6.1 deployed immutable revision/readiness proof | evidence only: `.deploy-control/p0/evidence/F3-*`; ORQ1 read-only at `100.118.244.61`, OmniRoute `100.118.244.61:20128`; never ORQ2 loopback or retired LAN | actual deployment revision and selected model/protocol readiness recorded without inference/body/secret; or concrete external blocker | 15–30m |
| F4 `w7:p4` | Wire delivery hop | `server/internal/daemonws/hub.go`, `hub_test.go` | `EmitDelivery` called at bounded delivered/drop/backpressure points; daemonws tests/vet PASS | 30–60m |
| F5 `w8:p1` | Wire ingress hop | `server/internal/middleware/request_logger.go*`, `server/internal/metrics/http.go*` | actual request/task join IDs, no body/header/query content; middleware/metrics tests PASS | 30–60m |
| F6 `w8:p2` | Wire queue + persist hops | `server/internal/service/task.go`, `task_complete_race_test.go`, `task_notify_test.go` | queue enqueue/claim and terminal persist emit exact join IDs; service focused tests/vet PASS | 45–75m |
| F7 `wB:p1` | Wire OmniRoute route hop | `server/internal/daemon/gateway/executor.go`, `executor_test.go`, `obs_span.go`, `obs_span_test.go` | `EmitProviderSpan` used on bounded terminal gateway outcome with sanitized telemetry; gateway tests PASS | 30–60m |
| F8 `wB:p2` | Sole independent evaluator | only `.deploy-control/p0/evidence/F8-*`, `handoffs/F8-*` | zero-overlap, diff, format, strict, focused matrix, seven emitted hops + trace continuity; no product edit | rolling + 30m final |
| F9 `wK:p1` | 6.3 technical harness readiness | only `server/internal/daemon/observability/{harness,synthetic,realtime_process*}.go` and matching tests; excludes `e2e/**` | fix/reproduce current offline realtime harness failure; synthetic 20-task result validates but retains `AcceptanceClaim=false` and modeled-resource non-claim | 45–90m |
| F10 `wK:p2` | Implement metadata-only CLI hop helper | new `server/internal/daemon/cli_observability.go`, `cli_observability_test.go` only | HopCLI carries launch/proc IDs, closed argv shape only, no prompt/args/content; tests/vet PASS; handoff to F1 | 30–60m |

All paths above are relative to `multica-auth-work/`. Each worker must use exact file locks, check in before edits, heartbeat every 10 minutes and checkout with command/exit-code evidence.

## 5. DAG and waves

### Wave 1 — T+0 to T+45 (all ten lanes active)

- F1 fixes the one daemon ordering failure.
- F2 runs already-unblocked package tests using tmpfs and records failures; final full run waits for producers.
- F3 obtains deployed proof or a concrete external blocker.
- F4/F5/F6/F7 wire independent anchors concurrently.
- F9 fixes/verifies technical tier-20 harness readiness without claiming acceptance.
- F10 builds the CLI helper.
- F8 evaluates completed deltas as they land; it never fixes.

### Wave 2 — T+45 to T+90

- F1 consumes F10 handoff and wires CLI call site in its exclusive `daemon.go` lock.
- Owners fix their own focused-test findings.
- F2 runs the complete 5.2 matrix and `go build ./...` using `/tmp` caches.
- F8 verifies each hop and cross-hop correlation.

### Wave 3 — T+90 to T+150

- F2 runs final `go test ./... -count=1` under tmpfs.
- F8 performs final independent evaluation and OpenSpec strict.
- Kiro produces evidence→task closure matrix.
- Principal closes only proven tasks.

## 6. Validation environment

From `multica-auth-work/server`:

```bash
export GO=/home/ec2-user/goroot/go/bin/go
export GOFMT=/home/ec2-user/goroot/go/bin/gofmt
export GOCACHE=/tmp/agent-brain-${LANE}-gocache
export GOTMPDIR=/tmp/agent-brain-${LANE}-gotmp
mkdir -p "$GOCACHE" "$GOTMPDIR"
```

No lane deletes caches or repository files to reclaim disk. F2 is the only lane running the final full suite; targeted lane tests may run concurrently in separate tmpfs paths.

## 7. OpenSpec closure mapping

| Task | Closure owner | Gate |
|---|---|---|
| 5.2 | F1/F2 evidence → F8 → Principal | complete targeted matrix green |
| 5.3 | F2 evidence → F8 → Principal | server-wide build and tests green |
| 6.1 | F3 → Principal | actual deployed revision/model/protocol proof, no inference |
| 6.2 | F1,F4,F5,F6,F7,F10 → F8 → Principal | all seven emitted hops + trace continuity, metadata-only |
| 6.3 | F9 technical proof; Principal/owner approval | real non-modeled 20-task measurements still required by contract |
| 6.4 | unstarted until 6.3 accepted | explicit tier-50/100 authorization and evidence |
| 6.5 | owner-controlled | explicit live token, exact route/build provenance and owner approval |

6.4 and 6.5 are not converted into fake prep work. If their external gates remain closed, the final report states the exact owner decision required and no calendar completion is claimed.

## 8. Non-duplication RACI

| Activity | Kiro manager | Principal | Workers / F8 |
|---|---|---|---|
| Fleet inventory, pane reads, check-in/heartbeat monitoring | **Responsible** | Receives consolidated report; samples only exceptions | Workers maintain receipts |
| Dispatch and first-line reassignment | **Responsible** | Adjudicates only ownership/architecture conflicts | Workers execute |
| Routine test/evidence collection | Consolidates | Does not rerun | Producers run; F8 independently evaluates |
| RED/AMBER investigation | First diagnosis and routing | Samples high-impact finding and decides unresolved conflict | Owner fixes; F8 verifies |
| DAG, ownership boundaries, safety gates | Consulted | **Accountable/Responsible** | Follow exact prompt |
| Product implementation | Prohibited | Prohibited while workers eligible | Producers only |
| Independent acceptance | Routes artifacts | Final adjudication | F8 is sole evaluator |
| OpenSpec checkbox closure | Recommends from evidence | **Sole authority** | Never closes |
| External live/deploy authorization | Never self-authorizes | Requires explicit owner decision | Never self-authorizes |

Operational rule: Kiro performs the continuous full-fleet scan and T+ reports. Principal does not repeat them. Principal independently samples only (a) findings that can close an OpenSpec task, (b) RED/AMBER exceptions, (c) shared-anchor/authorization conflicts, and (d) final acceptance. Kiro does not duplicate Principal architecture work, mutate OpenSpec, write product code or decide external gates.

## 9. Kiro control contract

- Re-read `herdr agent list` before dispatch and each 10-minute checkpoint.
- Dispatch only to idle/done workers listed above; never count `wA` shells.
- Verify canonical check-in and mechanical zero-overlap before permitting edits.
- Publish T+10, T+30, T+60, T+90 and final reports even if no new user prompt arrives; the persistent monitor is evidence, not a substitute for manager reporting.
- Reassign a material blocker within 10 minutes when a disjoint owner exists.
- Do not assign duplicate reviewers, broad regressions, documentation-only busywork or repeated live runs.
