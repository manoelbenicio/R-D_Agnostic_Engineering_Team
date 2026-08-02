# Tasks

> Execução: coders. Validação: Kiro. Check-in START/DONE por agente.
> Evidence Reconciliation (ORQ-88 / REC-CHAT-CLAIMS-01): 12 claims classified (DIRECT 3, CORROBORATED 5, CLAIMED 3, MISSING 1, CONTRADICTED 0).

## Bloqueios (dono decide antes)
- [x] 0.1 Squad default: `Workspace Team`, com Kiro/Opus 4.8 como TL/Manager delegation-only, Codex 5.6 Sol high-thinking como coder preferencial/escape hatch direto e os demais coders disponíveis como membros (CORROBORATED — `squad_briefing.go:28`).
- [x] 0.2 Limiar de quando o leader abre OpenSpec explore (CORROBORATED — `squad_briefing.go:44-50`; E9 24 AST assertions PASS).
- [x] 0.3 Roteamento default (chat→TL) por workspace ou por squad (CORROBORATED — `chat.go:120-145`).

## Implementação
- [x] 1.1 Identity/instructions do leader TL/Manager (protocolo esclarecer→openspec→planejar→delegar→sintetizar; marcador `## Squad Operating Protocol`) (CORROBORATED — `squad_briefing.go:26`; E9 24 AST assertions PASS; pending Root Owner attribution waiver for unattributable `[x]` setting / GAP-CHAT-01).
- [ ] 1.2 Squad TL/Manager default no setup do workspace (leader + membros) (CLAIMED / OPEN — `workspace.go` & `agent.go` code present, but test execution skipped in E16 due to PostgreSQL SASL auth error; isolated-PG evidence gap preserved).
- [x] 1.3 Roteamento default do chat: sem destino → squad TL; com `@agente` → direto (escape hatch) (DIRECT — E1 `CHAT-CLOSEOUT.md`, `TestCreateChatSession_Routing`, 23/23 gates rc=0, SHA-256 `chat.go` `a2a71f47...`, `chat_test.go` `23b5341b...`).
- [x] 1.4 Garantir leader delegation-only (não produz; delega + sintetiza) (CORROBORATED — `squad_briefing.go:105-112`; E9 24 AST assertions PASS; pending Root Owner attribution waiver / GAP-CHAT-01).
- [ ] 1.5 Estabelecer Kanban/API como unica superficie de despacho executavel, exatamente uma task por ativacao e Herdr somente para supervisao (CLAIMED / OPEN — present in OpenSpec & code claims, but unverified by sealed execution log).

## Verificação (Kiro valida)
- [ ] 2.1 Smoke: chat sem destino cai no TL; TL pergunta, delega a membro, sintetiza (CLAIMED / OPEN — HTTP routing proven by PG test, but live LLM synthesis & delegation loop is unverified by runtime tests).
- [x] 2.2 Smoke: chat `@codex` vai direto ao agente (escape hatch funciona) (DIRECT — E1 `CHAT-CLOSEOUT.md:38`, `TestCreateChatSession_Routing` subtest 1).
- [x] 2.3 Check-ins DONE + evidência em `.deploy-control/` (DIRECT — E1 `CHAT-CLOSEOUT.md` & `Kiro__CHAT-ORCH-2.x__20260801T132600Z.json` verified).
- [ ] 2.4 Smoke de despacho nativo: ORQ-12 atribuido por Kanban gerou exatamente uma task; follow-up ocorreu somente apos terminal, sem duplicata Herdr (MISSING / OPEN — zero supporting sealed evidence artifact or test log for native ORQ-12 Kanban dispatch).

---

## Evidence Manifest & Provenance Anchors

- **Candidate Baseline**: `a5aa53e8e89d2845cacfbc82ca851fbd18f9a505` (Tree OID: `b3d3f484fa7204308d085c17a99db8fe7b586f6e`)
- **E1 (Sealed PG Evidence)**: `.deploy-control/p0/evidence/CHAT-CLOSEOUT.md` (Inner manifest SHA-256: `5c247789397d3c462a70b19996a96d20e71bf7e1d2866f366a7a1ee8cd0d0476`)
- **E9 (Clean-Room Review)**: `.planning/agent-brain-v3/evidence/chat-orchestration-1.1-1.4-clean-room-atomic-review.md` (SHA-256: `e8d1d1ce27890a2a2c37c75beee812360ec5cf23bf3b74417b6be7d118727d76`)
- **E16 (Skipped DB Evidence)**: `.deploy-control/evidence/chat-orchestration-1.2-1.3.md` (Recorded SASL authentication error for Task 1.2)
- **Source Files & SHA-256**:
  - `multica-auth-work/server/internal/handler/chat.go`: `a2a71f47b4bbb2ee98671ca6232fa63cc4d76ec3ced438faa3220d1f61a7226c`
  - `multica-auth-work/server/internal/handler/chat_test.go`: `23b5341bd6abaa596ec131b6affd07c59e6bec25fe1e7502dde1091377107014`
  - `multica-auth-work/server/internal/handler/squad_briefing.go`: `8820384aca1cac838c5d75d4e5b219a90c0feae7ccc3b50b4d5061e63d45f0e2`
  - `multica-auth-work/server/internal/handler/squad_briefing_test.go`: `4cffe91db5b3ce11ff87b23f7819a74818fab3d09a7d92b5d766d14c2ebd0037`
