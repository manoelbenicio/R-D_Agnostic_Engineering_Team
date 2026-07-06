# TL BRIEFING — READ THIS FIRST (context handoff after restart)

> Você é o TECH LEAD. Você reiniciou (modelo trocou p/ Gemini 3.1 Pro) e PERDEU o contexto da sessão
> anterior. Este doc reconstrói TUDO. Leia inteiro antes de agir. Em dúvida → PERGUNTE AO KIRO (§7).

## 1. Missão (main task)
Milestone **v2.1 — Vendor Validation + PROD Deploy** do Rotation-Parity Polyglot. Objetivo final:
provar Smart Context REAL por vendor em PROD + kill-switch + rollback LIVE, com ZERO evidência fabricada.
Ver `.planning/PROJECT.md`, `.planning/ROADMAP.md`, `.planning/MILESTONE_v2.1.md`.

## 2. Onde estamos AGORA (last task)
Fase **P12 — PROD Deploy**, em execução AGÊNTICA. Tudo até P11 fechado (com caveat: números eram
`local_estimate`). O único bloqueio ativo é a task **12.3 (sessão real via gateway)**.

## 3. O BLOQUEIO atual (verificado nos 2 paths — NÃO refaça do zero)
- O token OAuth do Codex roteia CERTO para `chatgpt.com/backend-api` (sem problema de escopo).
- MAS a conta é `chatgpt_plan_type=free` e **bateu o limite de uso do Codex até 3-Ago-2026**.
- `api.openai.com` também não serve (token é login-only, sem créditos API).
- **Conclusão:** só o DONO desbloqueia — token Plus/Pro com quota OU `sk-...` platform com créditos.
- Check-in honesto: `.deploy-control/Codex-5.5-D__P12-AGENTIC__20260706T015121Z.md` = BLOCKED.

## 4. O que fazer quando o dono fornecer o token/key (ETA ~10 min)
Siga `.planning/phases/12-prod-deploy/PLAN.md` (12.1→12.7) + `RESEARCH.md` + `AGENTIC-REAL-SESSION.md`:
deploy binário PINADO → 12.3 sessão real (gateway 200, model REAL, usage real, tokens_saved distinto)
→ 12.4 kill-switch LIVE → 12.5 rollback LIVE → 12.6 scrub → 12.7 gate + backfill P11 + commit.

## 5. Regras que governam TUDO (leia e obedeça)
- `.planning/GOLDEN_RULES.md` — as 12 regras (check-in, ownership disjunto, evidência real, só TL commita…).
- `.planning/CHECKIN_OUT.md` — protocolo MANDATÓRIO de check-in/out (bloco `<mandatory_signin_signout>`).
- `.planning/EVIDENCE_CONTRACT.md` — o que é REAL vs INVALID. Fabricar = rejeitado + revertido.
- `.planning/ORCHESTRATION_v2.1.md` — roster/panes + protocolo Herdr. `EXECUTION_PLAN_v2.1.md` — waves/gates.

## 6. NÃO faça
Não fabrique evidência (localhost, fake-upstream, smoke build, números idênticos, sign-off forjado).
Não rode task fora do PLAN. Não autore docs em `.planning/` (só o Kiro autora). Prefira BLOCKED honesto.

## 7. EM DÚVIDA → PERGUNTE AO KIRO (como me contatar)
Se tiver QUALQUER dúvida ou decisão ambígua, NÃO adivinhe. Faça assim:
1. Escreva sua pergunta em `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.deploy-control/ASK_KIRO.md`
   (adicione um bloco novo no topo com timestamp UTC + a pergunta + contexto + o que você já tentou).
2. No SEU pane, escreva também uma linha começando com `@KIRO:` resumindo a pergunta (eu leio seu pane
   na cadência de 60s via `herdr pane read w3:pW`).
3. Marque a task como BLOCKED se estiver travado, e AGUARDE minha resposta (chega via `herdr pane run w3:pW`).
Eu (Kiro/Principal) autoro os planos, decido, e verifico. Você orquestra e executa. Pergunte cedo, não tarde.
