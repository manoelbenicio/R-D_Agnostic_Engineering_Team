# L6 — ingress/queue/persist metadata-only helpers: design + L1 anchor map (BLOCKED on gates)

- agent: Opus48#D · lane: L6 · task: 6.2-L6-obs-helpers · pane `w8:p2`
- check-in: `.deploy-control/p0/checkins/CHECKIN__Opus48-D__L6__6.2-L6-obs-helpers__20260722T040450Z.json`
- p0_control check-in: `.deploy-control/p0/checkins/Opus48-D__6.2-L6-obs-helpers__20260722T040450Z.json`
- locks (exact, exclusive, all **NEW**): `server/internal/middleware/obs_ingress.go`,
  `server/internal/service/obs_queue.go`, `server/internal/service/obs_persist.go` (+ their `*_test.go`).
- posture: **investigation/design only.** No product file created; two hard gates open (see §4).

## 0. Preflight
cwd `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team` · git HEAD `a6d5098…` · git status 256 ·
go `go1.26.1` (`/home/ec2-user/goroot/go/bin/go`) · node `v22.23.1` · disk 2.6G free (90% used).

## 1. Ownership + zero-overlap (verified)
Exclusive to L6: the three helper files above (+ tests). Confirmed **absent** at HEAD (all NEW), and
**no active check-in locks them** (scan of `checkins/*.json` in IN_PROGRESS|BLOCKED → none). MUST-NOT-TOUCH
(and not locked): `request_logger`/router chain, `service/task.go`, daemon central, `daemonws`, UI,
`observability/e2e/**` (L5 internals), OpenSpec/GSD.

## 2. L5 API to consume (read-only, from `observability/e2e/contract.go` @HEAD — subject to L5 freeze)
- `ContractVersion = "agent-brain.e2e.v1"`; `SupportedContractVersion(v)`.
- `HopKind`: L6 emits `HopIngress` (1), `HopQueue` (2), `HopPersist` (6).
- `Correlation{RequestID,QueueMsgID,TaskID,SessionID,LaunchID,ProcID,OmniRequestID,ResultID,DeliveryID}`.
- `RequiredIDs(hop)` join keys: ingress→`{request_id,task_id}`; queue→`{queue_msg_id,task_id}`;
  persist→`{task_id,result_id}`.
- Hard invariant (AB-REQ-40/PD-08): metadata-only; `SecretsPresent==false`; no bodies/payloads/results,
  no high-cardinality/content labels; fail-closed if content would be carried.
- **NOTE:** L5 is an active lane this window and must publish its *frozen* span-emit entrypoint
  (constructor/emit signature) before L6 finalizes. L6 will not invent or assume that signature.

## 3. Planned helpers (metadata-only) — to implement only after gates clear
- `obs_ingress.go` (pkg `middleware`): build the hop-1 span from an already-parsed request — method,
  normalized route template, pseudonymous principal class, status, latency; carries `request_id`,`task_id`;
  **no request/response bodies, no headers, no query**. Pure helper: input is caller-supplied metadata; it
  does not read `*http.Request` bodies and does not wrap the router.
- `obs_queue.go` (pkg `service`): hop-2 span — enqueue/dequeue timestamps, queue depth, wait duration;
  carries `queue_msg_id`,`task_id`; **no task payload**. Pure helper over caller-supplied counters.
- `obs_persist.go` (pkg `service`): hop-6 span — persist latency, byte/token counts (numeric), terminal
  status code; carries `task_id`,`result_id`; **no result content**.
- Each helper validates `RequiredIDs` presence, sets `ContractVersion`, asserts `SecretsPresent==false`,
  and returns the L5 span type (never a bespoke struct). Tests: table-driven, assert required IDs enforced,
  content-free, fail-closed on missing IDs — package tests + `-race` + `vet`, no DB/schema/migration.

## 4. L1 anchor-insertion map (HANDOFF — L6 does NOT edit these shared anchors)
The call sites live in shared anchors owned by L1 (serial, Wave C). L6 delivers the exact insertion points;
L1 inserts the calls to the L6 helpers:
| Hop | L6 helper | Shared anchor (L1-owned) | Insertion point |
|---|---|---|---|
| ingress (1) | `middleware.obs_ingress` | `internal/metrics/http.go` (or router chain) / `internal/middleware/request_logger.go` | after request is served, emit span with `request_id→task_id`, status, latency |
| queue (2) | `service.obs_queue` | `internal/service/task.go` enqueue + dequeue sites | at enqueue and at claim/dequeue, emit with `queue_msg_id↔task_id`, depth, wait |
| persist (6) | `service.obs_persist` | `internal/service/task.go` terminal-result persist site | after terminal result stored, emit with `task_id/result_id`, latency, byte/token counts, terminal status |
L6 provides the helper signatures; L1 owns the edits. No L6 call-site insertion into any anchor.

## 5. Gates — BLOCKED (both must clear before any product file is created)
1. **`control.json implementation_authorized=false`** → per SIX_HOUR_CHECKIN_CONTRACT §8, only
   investigation/plan/tests-verification are permitted; creating NEW product source is prohibited.
   Precedence (plan §2) ranks `control.json` (3) above the 6h package (4). **Owner: Principal Orchestrator (`w5:p9`).**
2. **L5 frozen-contract handoff not yet received** for this window (L5 `w8:p1` is an active parallel lane;
   L6 "must not invent L5 API" and "waits for L5 frozen contract before finalizing"). **Owner: L5 (`w8:p1`) via manager.**

`live_runs.*=false` also holds → no inference (not needed for offline metadata-only helpers anyway).

## 6. Next action on unblock
When `implementation_authorized=true` AND L5 publishes its frozen span-emit signature: create the three
NEW helpers + tests against the L5 types (metadata-only, fail-closed), run
`go test ./internal/middleware/ ./internal/service/ -run 'Obs(Ingress|Queue|Persist)' -race -count=1` +
`go vet` + `gofmt -l` + `git diff --check`, deliver this §4 anchor map to L1, and check out with evidence.
No shared-anchor edits, no deploy/inference/secret, no OpenSpec/GSD.

## 7. IMPLEMENTATION + VALIDATION (resumed @04:31Z; implementation_authorized=true, L5 frozen)

Files created (all NEW, within lock; metadata-only against `agent-brain.e2e.v1`):
- `internal/middleware/obs_ingress.go` + `obs_ingress_test.go` — `NewIngressSpan`/`EmitIngress` (hop 1): labels ⊆ {method,route_template,principal_class}; counter `latency_ms`; HTTP status; corr {request_id,task_id}.
- `internal/service/obs_queue.go` + `obs_queue_test.go` — `NewQueueSpan`/`EmitQueue` (hop 2): counters {wait_ms,queue_depth,enqueue_unix_ms,dequeue_unix_ms}; no labels; corr {queue_msg_id,task_id}.
- `internal/service/obs_persist.go` + `obs_persist_test.go` — `NewPersistSpan`/`EmitPersist` (hop 6): counters {persist_latency_ms,byte_count,token_count}; label {terminal_status}; corr {task_id,result_id}.

Each helper uses only the frozen closed label/counter keys, sets `ContractVersion`, keeps `SecretsPresent=false`, and emits via `e2e.Recorder.Emit` (fail-closed). Tests assert validity, correlation carry, metadata-only key sets, `e2e.ScanSpans` Clean, and fail-closed refusal on a missing required ID.

Evidence (exact commands, `GO=/home/ec2-user/goroot/go/bin/go`, cwd `multica-auth-work/server`):
| # | Command | Result | Exit |
|---|---|---|---|
| 1 | `gofmt -l <6 files>` | empty (clean) | 0 |
| 2 | `go vet ./internal/middleware/` | clean | 0 |
| 3 | `go test ./internal/middleware/ -run Ingress -count=1` | `ok … 0.003s` (3 tests PASS) | 0 |
| 4 | `git diff --check -- <6 files>` | clean | 0 |
| 5 | `CGO_ENABLED=1 go test -race ./internal/middleware/ -run Ingress` | **NOT_AVAILABLE** — `gcc` absent, cgo cannot build (env limit, not code) | (env) |
| 6 | `go vet ./internal/service/` | **BLOCKED** — foreign `internal/service/email.go:9: "log/slog" imported and not used` | 1 |
| 7 | `go test ./internal/service/ -run 'ObsQueue|ObsPersist|NewQueueSpan|NewPersistSpan'` | **BLOCKED (build failed)** — same foreign `email.go` break | 1 |
| 8 | `go build ./internal/service/` (attribution check) | fails ONLY on `email.go`; **zero errors attributed to `obs_queue.go`/`obs_persist.go`** | 1 |

Verified: ingress (hop 1) helper+tests GREEN. Reviewed + compile-clean + gofmt/diff-clean: queue/persist helpers+tests — but their package test/vet are BLOCKED by a pre-existing build break in `internal/service/email.go`, a file **outside L6 ownership** (must-not-touch; one owner per file).

## 8. FINDING routed to manager (owner ≠ L6)
- finding: `internal/service/email.go:9:2 "log/slog" imported and not used` → breaks the whole `service` package build, blocking L6 queue/persist package tests.
- owner: the lane/agent owning `internal/service/email.go` (uncommitted change predating this session; route via manager `w5:p1`).
- fix: remove the unused `log/slog` import (or use it) — one-line, in that file's owner's lock. L6 does not touch it.
- after fix: re-run test #7 (`go test ./internal/service/ -run 'ObsQueue|ObsPersist' -count=1`) → expected PASS; then L6 checks out DONE.

## 9. L1 anchor-insertion HANDOFF (final — L6 inserted no call sites)
Per §4 map: L1 (Wave C, serial) inserts calls to the L6 helpers at the shared anchors:
- `middleware.EmitIngress(rec, middleware.IngressObservation{...})` at the ingress/router-served site (`internal/metrics/http.go` or router chain / `request_logger.go`).
- `service.EmitQueue(rec, service.QueueObservation{...})` at `internal/service/task.go` enqueue + dequeue sites.
- `service.EmitPersist(rec, service.PersistObservation{...})` at `internal/service/task.go` terminal-result persist site.
L6 owns the helper signatures only; L1 owns the anchor edits. No L6 edit to `task.go`/router/`hub.go`.

## 10. RESUME @11:02Z — R2 gate cleared; ALL L6 TESTS GREEN → DONE

R2 (`REC-SERVICE-EMAIL`, `w6:p2`) checked out DONE; `email.go` unused import removed; `service` package
compiles. L6 resumed and reran (`GO=/home/ec2-user/goroot/go/bin/go`, cwd `multica-auth-work/server`):

| # | Command | Result | Exit |
|---|---|---|---|
| A | `go build ./internal/service/` | clean (gate confirmed) | 0 |
| B | `go test ./internal/service/ -run 'QueueSpan|PersistSpan' -count=1 -v` | **4 tests PASS** (NewQueueSpan{Valid,MissingQueueMsgID}, NewPersistSpan{Valid,MissingResultID}); `ok 0.007s` | 0 |
| C | `go vet ./internal/service/` | clean | 0 |
| D | `gofmt -l <service L6 files>` | empty (clean) | 0 |
| E | `git diff --check <6 L6 files>` | clean | 0 |
| (prior) | `go test ./internal/middleware/ -run Ingress -count=1` | 3 tests PASS; `ok 0.003s` | 0 |
| (prior) | `go vet ./internal/middleware/` | clean | 0 |

Note: the manager-suggested `-run 'ObsQueue|ObsPersist'` matched **no** tests (my funcs are named
`*QueueSpan*`/`*PersistSpan*`); reran with the matching pattern so tests truly execute (zero-tests ≠ PASS).
`-race` remains **NOT_AVAILABLE** (no `gcc`/cgo — environment limit, not a code defect).

**Final L6 status: DONE.** All three metadata-only helpers (ingress hop1, queue hop2, persist hop6) +
7 tests total GREEN; gofmt/vet/diff clean; metadata-only against frozen `agent-brain.e2e.v1`; no
shared-anchor edit (call-site insertion handed to L1 in §4/§9).
