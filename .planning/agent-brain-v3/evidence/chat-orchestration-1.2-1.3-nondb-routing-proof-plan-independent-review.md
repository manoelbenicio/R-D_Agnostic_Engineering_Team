# EV-CHAT-1.2-1.3-NONDB-PLAN — INDEPENDENT REVIEW

Independent review of `chat-orchestration-1.2-1.3-nondb-routing-proof-plan.md`
(SHA-256 `f8a5ebc92332011efd01f3dd82bda4d3ddfea76494dcfb62464a7c049c71d26c`, verified stable).
Reviewer: **Kiro/Opus-4.8 — session `w8:p2`**, distinct pane from the plan's author **Kiro/Opus-4.8 `w7:p2`**.
Adjudicator: **Kiro TL**. **Technical-feasibility review only — no acceptance, no self-acceptance.**

> **Independence caveat.** Reviewer (`w8:p2`) and plan author (`w7:p2`) are the **same model family**
> (Kiro/Opus-4.8), distinct sessions; separation is by session, not identity. All checks below were
> re-derived first-hand from source. Pane labels are self-declared process metadata.

## CHECK-IN 2026-07-18T22:06:00Z
Mode: READ-ONLY review. Sole writable deliverable = this file. Excluded (honored): no implementation; no
shared-planning/spec/tasks/source/test/git/index/ref edit; no credentials/env values; no DB/network/services.

## Provenance verified (this review, HEAD `b6571299`)
- Plan doc `f8a5ebc9…d26c` (mtime 19:07:21, stable across re-hash).
- **All 5 source hashes MATCH the plan's manifest exactly:** `chat.go` `52af110d…dba77c`, `workspace.go`
  `f3c7f66c…4e18c3`, `agent.go` `1339bff8…4bdbb7`, `chat_test.go` `623754cd…884cdfad`,
  `chat-orchestration-standard/tasks.md` `a7d19efa…41a00dc`. No drift.
- Tasks (verbatim): `tasks.md:12` `1.2 Squad TL/Manager default no setup do workspace (leader + membros)`;
  `tasks.md:13` `1.3 Roteamento default do chat: sem destino → squad TL; com \`@agente\` → direto (escape hatch)`.
  Both `[ ]`.

## Claim-by-claim verification (first-hand)

1. **DB TestMain constraint — CONFIRMED.** `internal/handler/handler_test.go` `TestMain` connects to
   `DATABASE_URL` (defaulting to a local DSN) and calls **`os.Exit(0)`** on either connect-error or
   `pool.Ping` failure, **before `m.Run()`** → every test in package `handler` (incl.
   `chat_test.go`) is skipped offline. The plan's blocker #1 is accurate.

2. **Concrete dependency — CONFIRMED.** `handler.go:99` declares `Queries *db.Queries` (concrete sqlc), not
   an interface → cannot be mocked from a same-package test without a production seam. Plan blocker #2 accurate.

3. **1.3 default chat routing — CONFIRMED verbatim** (`chat.go` CreateChatSession):
   `if req.AgentID == "" { squads := ListSquads(ws); if err||len==0 → 400 "no default squad found for
   routing"; if !squads[0].LeaderID.Valid → 400 "default squad has no leader yet"; agentID =
   squads[0].LeaderID } else { parseUUIDOrBadRequest(agent_id) }`. Error strings and the **`squads[0]`
   ordering assumption** match the plan's table cases 1–5 exactly.

4. **1.2 default TL squad — CONFIRMED** (`workspace.go` CreateWorkspace, inside the create tx `qtx`):
   `CreateSquad{Name:"Workspace Team", Description:"Default workspace squad"}` + `AddSquadMember{owner,
   member/member}`; leader deferred. Matches the plan.

5. **1.2 deferred-leader claim — CONFIRMED** (`agent.go` CreateAgent):
   `if isFirstAgent { for sq := range ListSquads(ws) { if !sq.LeaderID.Valid { UpdateSquad{ID:sq.ID,
   LeaderID:created.ID} } } }`. Matches the plan's `LeaderlessSquadIDsToClaim` extraction target.

6. **Explicit `agent_id` vs literal `@mention` — CONFIRMED and correctly flagged.** `grep` for
   `@`/`mention`/`ParseName` in `chat.go` = **0 matches**. The implemented "escape hatch" is the explicit
   `agent_id` request field; there is **no `@agentname` body parsing** anywhere in `CreateChatSession`/
   send. Since `tasks.md:13` literally specifies `@agente`, this is a genuine **spec-vs-implementation
   gap**: the plan's blocker #1 ("`@agentname` parsing is unimplemented; plan assumes explicit-id
   semantics") is correct and material — an owner decision on whether 1.3 requires literal `@name` parsing.

7. **Proposed DB-free seam — FEASIBLE and behavior-preserving.** `DefaultChatRoute(reqAgentID string,
   squads []db.Squad) (pgtype.UUID, bool, string)` and `LeaderlessSquadIDsToClaim(squads []db.Squad)
   []pgtype.UUID` are faithful pure extractions of the exact branches above; inputs are a sqlc slice + a
   string (no DB), so a `internal/handler/chatroute` package with a non-gated TestMain runs offline. The
   table cases and expected error strings are correct. **It is a production refactor** (touches
   `chat.go`/`agent.go` + new package) — the plan correctly flags this as owner-gated (option B) and notes
   option A (external AST verifier) already exists/accepted for the *encoding* contract.

## Verdict — feasibility vs acceptance (kept separate)

- **Technical feasibility of the proof plan: SOUND.** The two blockers (TestMain `os.Exit(0)` gate; concrete
  `*db.Queries`) are real and verified; the proposed `chatroute` extraction is behavior-preserving and would
  yield genuine offline runtime coverage of the routing + leaderless-claim decisions; the table cases map
  1:1 to the verified branches and error strings.
- **Acceptance: NOT GRANTED (and not the plan's role).** This is an advisory **plan**, not evidence — it
  accepts nothing and sets no checkbox (correct). Independent qualifiers for TL:
  1. **Option B requires owner authorization** to refactor `chat.go`/`agent.go` and add `chatroute`; without
     it, only the already-accepted **AST-contract** proof stands (encoding, not runtime).
  2. **Spec-vs-impl `@agente` gap** (item 6) is an open owner decision, independent of this plan.
  3. **`squads[0]` ordering** is a latent fragility (default squad chosen by array index, not an explicit
     marker) — noted by the plan; worth an owner call for hardening.
  4. Pre-existing chat 1.2/1.3 governance gaps remain out of scope here and unresolved (per `AGENT_LEDGER`:
     reviewer-identity "opencode" ≠ "GLM52#B", missing AB-REQ, DB-backed runtime smoke 2.1–2.3 pending);
     this plan does not close them.
- **Recommendation echo (not decided):** if a runtime proof is wanted, the `chatroute` extraction is the
  smallest sound path (owner-gated); otherwise the accepted AST proof suffices for the encoding contract.
  1.2/1.3 stay `[ ]`. Kiro TL adjudicates; reviewer ≠ adjudicator.

## CHECK-OUT 2026-07-18T22:09:00Z — DONE
Only this file created. Plan doc + all source unchanged; no git stage/commit/push; no
credentials/env/network/DB/services. Adjudication pending Kiro TL.
