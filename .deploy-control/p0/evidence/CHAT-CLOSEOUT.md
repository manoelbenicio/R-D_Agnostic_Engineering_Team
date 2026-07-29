# CHAT-CLOSEOUT — chat orchestration routing (Tasks 1.2/1.3) closeout evidence

- agent: `Opus48#A` · lane `CHAT-CLOSEOUT` · task `CHAT-ORCH-2.x` · pane `w6:p1`
- date: 2026-07-22T12:04Z · Go `/home/ec2-user/goroot/go/bin/go` (go1.26.1), GOCACHE/GOTMPDIR=/tmp
- lock: `.deploy-control/p0/evidence/CHAT-CLOSEOUT.md` only (no test-only fix required — see §3)
- status: **BLOCKED** (concrete DB-unreachable blocker; focused test cannot execute)

## 1. Reuse-only inspection (no rebuild, no broad tests)

- Accepted EV `.deploy-control/evidence/chat-orchestration-1.2-1.3.md`: implements default-squad TL
  materialization (workspace.go/agent.go) and chat routing (chat.go); adds `TestCreateChatSession_Routing`.
  That EV itself already recorded the offline test was **not executed** because the DB was unreachable.
- Source `internal/handler/chat.go` `CreateChatSession`: routes an empty `agent_id` to the default
  squad's `LeaderID` (TL), and an explicit `agent_id` directly to that agent (direct escape hatch).
- Existing test `internal/handler/chat_test.go` `TestCreateChatSession_Routing` (REVIEWED by reading)
  covers **both** required cases:
  - **explicit `agent_id` → direct**: posts `{"agent_id": directAgentID}`, asserts
    `directResp.AgentID == directAgentID`.
  - **omitted target → TL**: clears squads, creates a squad, sets `LeaderID=squadTLID`, posts with no
    `agent_id`, asserts `defaultResp.AgentID == squadTLID`.
  No broad tests were added; no source changed.

## 2. Focused smoke — exact commands + exit codes (NO inference)

Command (from `multica-auth-work/server`, GOCACHE/GOTMPDIR=/tmp):
```
/home/ec2-user/goroot/go/bin/go test ./internal/handler/ -run TestCreateChatSession_Routing -count=1
→ ok  github.com/multica-ai/multica/server/internal/handler  0.065s   (exit 0)
```
**This exit 0 is NOT a real PASS.** The verbose run reveals the DB-dependent tests were SKIPPED, so
`TestCreateChatSession_Routing` executed **zero** assertions:
```
/home/ec2-user/goroot/go/bin/go test ./internal/handler/ -run '^TestCreateChatSession_Routing$' -count=1 -v
→ "Skipping tests: database not reachable: failed to connect to `user=multica database=multica`:
   127.0.0.1:5432 (localhost): dial error: dial tcp 127.0.0.1:5432: connect: connection refused"
   (exit 0; no === RUN / --- PASS for the routing test)
```
Per the closeout rule, a zero-executed-test / skipped result is NOT PASS.

## 3. Test-only fix assessment

None needed. The test is correct and covers both routing cases; it is unexecutable only because the
DB is unreachable. `chat_test.go` was therefore NOT modified (kept out of the lock).

## 4. CONCRETE BLOCKER

- **Fact:** the handler test harness (`TestMain`) requires a reachable Postgres at `127.0.0.1:5432`
  (`user=multica database=multica`) and **skips all DB-dependent tests** when it is not; the socket is
  refused (`connect: connection refused` — no DB listening). `TestCreateChatSession_Routing` cannot run.
- **Owner:** local test-environment / infra provisioner (via Manager `w5:p1` / Principal `w5:p9`).
- **Next action:** bring up a reachable local Postgres (`multica/multica` @ 127.0.0.1:5432) or point the
  harness at a reachable test DB, then re-run
  `/home/ec2-user/goroot/go/bin/go test ./internal/handler/ -run TestCreateChatSession_Routing -count=1 -v`
  and confirm `--- PASS: TestCreateChatSession_Routing` with both subpaths executing.
- Credentials were **not** accessed or read; no attempt to start/patch the DB was made.

## 5. Explicit non-claims

- Not claimed: that `TestCreateChatSession_Routing` executed or PASSED — it was **SKIPPED** (DB unreachable).
- Not claimed: any end-to-end/UI/live verification; no inference; no live services accessed.
- Routing coverage in §1 is **REVIEWED by source reading**, not verified by test execution.
- No deploy/restart/credential access/inference; no source or test edits; no OpenSpec checkbox closed.
