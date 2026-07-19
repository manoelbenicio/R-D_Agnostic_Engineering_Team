# W8 independent-review draft — chat orchestration 1.2/1.3

Status date: 2026-07-19. Reviewer: W8 / Codex (`/root`), evidence-only and
read-only on product code. Original producer: Gemini. Prior evidence reviewer:
`opencode`. Adjudicator: Kiro TL. These are distinct recorded roles; W8 does not
relabel any historical identity and does not adjudicate its own draft.

## Classification

| Task | Implemented | Reviewed | Verified | Accepted |
|---|---|---|---|---|
| 1.2 default TL squad | candidate implementation present | yes, source/evidence review | structural contract only; DB/runtime smoke absent | **NO — remains open** |
| 1.3 default chat route | candidate implementation present | yes, source/evidence review | `agent_id` escape hatch structurally shown; literal `@agente` behavior unproved | **NO — remains open** |

The prior AST review calls both implementation contracts ACCEPT, but its verifier
and transcripts lived outside the repository and are not reproducible from this
checkout. More importantly, OpenSpec says `@agente`; current evidence proves an
explicit `agent_id` request field, not parsing a literal mention. The DB-backed
runtime smokes 2.1–2.3 are also unexecuted. Those are acceptance gaps, not wording
details that W8 may silently waive.

## Exact frozen source manifest

```text
f3c7f66c1685d5c95273f6fbd9234d301c422529b33a8ec341074f808a4e18c3  multica-auth-work/server/internal/handler/workspace.go
1339bff805d241451e9271c3684c13c280164c29f9ec6869c718415f584bdbb7  multica-auth-work/server/internal/handler/agent.go
52af110d08be90b9faeb65f180211af6f673f1ca2ac1fb29f9623d7124dba77c  multica-auth-work/server/internal/handler/chat.go
623754cd749ab282e7828ce8af3d73581fa2ebbe5238b4a90c59de7b884cdfad  multica-auth-work/server/internal/handler/chat_test.go
```

## Exact evidence-input manifest

```text
a7c1fc5488eec2fc7a4b1a3d1a4246ebd6d95246fb95a2e0c8691a24f5a0b375  .planning/agent-brain-v3/evidence/chat-orchestration-1.2-1.3-review.md
83ffc4cffbe005bd4e762011824e495f61d7289e83f8a0b5600c6ac7c718dc6c  .planning/agent-brain-v3/evidence/chat-orchestration-1.2-1.3-nondb-routing-proof-plan-independent-review.md
```

## Required next evidence

1. Owner decision: literal `@name` parsing is required, or `agent_id` is the
   accepted escape-hatch contract and the spec is amended explicitly.
2. A repository-resident, non-zero executable proof or DB-backed runtime smoke
   for default-squad creation, deferred leader assignment, untargeted routing,
   and direct routing.
3. A distinct reviewer reproduces that proof; Kiro TL adjudicates afterward.

No task checkbox, product file, GSD file, secret, service, DB, or network state
was changed or accessed by this draft.
