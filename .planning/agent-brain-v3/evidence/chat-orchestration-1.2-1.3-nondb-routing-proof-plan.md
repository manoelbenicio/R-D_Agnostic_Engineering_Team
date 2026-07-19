# chat-orchestration 1.2/1.3 — pure non-DB executable proof plan (advisory)

- Author: Kiro/Opus-4.8, pane **w7:p2**. **Advisory design only; Kiro TL adjudicates.** No implementation, no
  checkbox, no DB/network/services, no source/test/shared-planning/spec/tasks/git/index/ref edits.
- Tasks: **1.2** default TL/Manager squad on workspace setup (leader + members); **1.3** default chat routing
  (no target → squad TL; `@agente` → direct escape hatch). Both currently `[ ]` in tasks.md.

## Check-IN / Check-OUT

- **Check-IN** 2026-07-18T22:20:00Z — read-only study of `internal/handler/{workspace,agent,chat,chat_test}.go`
  + existing evidence; sole deliverable is this artifact.
- **Check-OUT** 2026-07-18T22:34:00Z — DONE. Hashes verified @ HEAD `b6571299`; no drift vs prior manifest.

## Provenance — current source hashes (SHA-256) — no drift vs interim/review manifest

| File | SHA-256 | Git state |
|---|---|---|
| `internal/handler/chat.go` | `52af110d08be90b9faeb65f180211af6f673f1ca2ac1fb29f9623d7124dba77c` | `M` |
| `internal/handler/workspace.go` | `f3c7f66c1685d5c95273f6fbd9234d301c422529b33a8ec341074f808a4e18c3` | `M` |
| `internal/handler/agent.go` | `1339bff805d241451e9271c3684c13c280164c29f9ec6869c718415f584bdbb7` | `M` |
| `internal/handler/chat_test.go` | `623754cd749ab282e7828ce8af3d73581fa2ebbe5238b4a90c59de7b884cdfad` | `M` |
| `openspec/changes/chat-orchestration-standard/tasks.md` | `a7d19efa305fdfd8a9e4b1c8ca0a306f7fb4339b60ceed3d72987ec2841a00dc` | `M` |

All four source hashes equal the `chat-orchestration-1.2-1.3-interim.md` / `-review.md` manifest → the state below
is the same one opencode reviewed.

## Current implementation status (both implemented; runtime unproven)

- **1.3 routing** — `chat.go` `CreateChatSession` (`chat.go:53-71`): `if req.AgentID == "" { squads,_ :=
  ListSquads(ws); if err||len==0 → 400 "no default squad found for routing"; if !squads[0].LeaderID.Valid → 400
  "default squad has no leader yet"; agentID = squads[0].LeaderID } else { parseUUIDOrBadRequest(agent_id) }`, then
  workspace-scoped `GetAgentInWorkspace` + private-agent gate.
- **1.2 default squad** — `workspace.go` `CreateWorkspace` (`workspace.go:~218-241`): inside the create tx, `qtx.CreateSquad`
  name `"Workspace Team"` + `qtx.AddSquadMember` (owner, role `member`); leader deferred.
- **1.2 deferred leader** — `agent.go` `CreateAgent` (`agent.go:~844-855`): `if isFirstAgent { ListSquads;
  for leaderless squad → UpdateSquad{LeaderID: created.ID} }`. Backing query `UpdateSquad` exists
  (`pkg/db/queries/squad.sql:48`, `leader_id = COALESCE(sqlc.narg('leader_id'), leader_id)`).
- **Prior acceptance**: `chat-orchestration-1.2-1.3-review.md` (opencode, independent) graded **1.2 ACCEPT · 1.3
  ACCEPT** for the *implementation contract* via an external AST verifier ×20/race/vet/build; checkboxes remain OPEN;
  runtime DB smoke (2.1–2.3) and behavior explicitly non-claimed.

## Why the smallest proof cannot be a normal in-package test (blockers)

1. **DB-gated TestMain** — `handler_test.go:38-54` calls `os.Exit(0)` before `m.Run()` when Postgres is
   unreachable, gating **every** test in package `handler` (including `chat_test.go:TestCreateChatSession_Routing`,
   which the interim run recorded as skipped/no-op offline).
2. **Concrete dependency** — `Handler.Queries` is a concrete `*db.Queries` (sqlc), not an interface → cannot be
   mocked from a same-package test without a production seam change.
3. Consequence: the routing/leader decisions are inline in DB-coupled HTTP handlers; a **pure, runnable** test needs
   either (A) an external source/AST verifier (no runtime logic), or (B) extraction of the decision into a
   DB-free, non-gated package (a small production refactor).

## Exact seams, table cases, expected assertions

### 1.3 — routing decision (target of extraction)
Proposed pure kernel (behavior-preserving extraction of `chat.go:53-71`):
`func DefaultChatRoute(reqAgentID string, squads []db.Squad) (leader pgtype.UUID, useDefault bool, errMsg string)`

| # | Input | Expected |
|---|---|---|
| 1 | `reqAgentID="…uuid…"`, any squads | `useDefault=false, errMsg=""` (handler parses explicit id → **@agent escape hatch**) |
| 2 | `reqAgentID=""`, `squads=nil`/empty | `errMsg="no default squad found for routing"` (400) |
| 3 | `reqAgentID=""`, `squads[0].LeaderID.Valid=false` | `errMsg="default squad has no leader yet"` (400) |
| 4 | `reqAgentID=""`, `squads[0].LeaderID.Valid=true` | `leader=squads[0].LeaderID, useDefault=true, errMsg=""` |
| 5 | `reqAgentID=""`, multiple squads, only `squads[1]` has leader | routes to `squads[0]` (documents the **`squads[0]` ordering assumption**) |

Assertions: exact errMsg strings; leader equals `squads[0].LeaderID`; explicit id path never consults squads.

### 1.2 — deferred-leader kernel (target of extraction)
Proposed pure kernel (extraction of `agent.go:844-855` loop):
`func LeaderlessSquadIDsToClaim(squads []db.Squad) []pgtype.UUID` (returns squads with `!LeaderID.Valid`).

| # | Input | Expected |
|---|---|---|
| 1 | squads all leaderless | returns all their IDs (each gets `UpdateSquad{LeaderID: firstAgent}`) |
| 2 | mix leaderless/led | returns only leaderless IDs (idempotent: led squads untouched) |
| 3 | empty / all led | returns empty (no UpdateSquad calls) |

Assertions: only leaderless squads selected; deterministic ordering; no mutation of led squads. (The `isFirstAgent`
gate and `ListSquads`/`UpdateSquad` DB effects remain in the handler and are non-claimed runtime.)

## Is a production refactor necessary?

- **For a pure runtime-logic executable proof: YES (smallest = extract two kernels into a DB-free package).** Because
  of the TestMain gate + concrete `*db.Queries`, the decisions cannot be unit-tested in place. Smallest
  behavior-preserving refactor: create `internal/handler/chatroute/` (no DB-gated TestMain) exporting
  `DefaultChatRoute` and `LeaderlessSquadIDsToClaim`; `CreateChatSession`/`CreateAgent` call them. Then normal
  table tests in `chatroute` run offline (no DB, no mocks).
- **For a pure structural-contract proof: NO refactor** — reproduce opencode's external AST verifier (parses real
  source, asserts the branch/error/assignment/scope contract). This already exists and is ACCEPTED
  (`chat-orchestration-1.2-1.3-review.md`), so it need not be redone; it proves *encoding*, not runtime.

Recommendation: since the AST-contract proof is already accepted, the highest-value *incremental* pure proof is the
**extract-to-`chatroute` refactor** (option B) for genuine runtime-logic coverage of the routing + leaderless-claim
decisions. If no source change is authorized, the existing AST proof stands and no new proof is needed.

## Blockers / open decisions (owner/Kiro)

1. **@agent semantics**: server-side escape hatch = explicit `agent_id` param; there is **no** `@name` text parsing
   in `CreateChatSession`/`SendChatMessage`. If 1.3 intends `@agentname` parsing from message body, that is
   **unimplemented** — a production feature + separate decision. Proof plan assumes the implemented (explicit-id) semantics.
2. **`squads[0]` ordering**: default routing trusts `ListSquads` array index 0 = the default squad (created first).
   A hardening would select the default squad by an explicit marker; noted, out of 1.2/1.3-as-written.
3. **Refactor authorization**: option B needs owner sign-off to touch `chat.go`/`agent.go` + add `chatroute` pkg.
4. **Runtime smoke 2.1–2.3** remains a separate DB-backed Kiro lane; not covered by any pure proof.

## Recommended atomic file set

- **Option B (recommended, runtime proof):** NEW `internal/handler/chatroute/route.go` (pure `DefaultChatRoute` +
  `LeaderlessSquadIDsToClaim`) + NEW `internal/handler/chatroute/route_test.go` (table tests, no DB); minimal
  behavior-preserving edits to `chat.go` (call `DefaultChatRoute`) and `agent.go` (call `LeaderlessSquadIDsToClaim`).
  Owner: handler owner. Gofmt/vet/`go test ./internal/handler/chatroute -count=20 -race` offline.
- **Option A (no refactor, contract only):** external AST verifier module (as in the accepted opencode EV); no repo
  file added. Already satisfied by `chat-orchestration-1.2-1.3-review.md`.

## Non-claims
- Advisory design only. No code/test/spec/tasks/checkbox/shared-planning/git/index change; no DB/network/services;
  no credentials/env. Hashes verified @ HEAD `b6571299`; both tasks confirmed implemented in source but runtime
  behavior is non-claimed here. Kiro TL adjudicates.
