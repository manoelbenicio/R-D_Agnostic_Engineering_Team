# PROMPT CODEX-D — P12 PROD Deploy + Live Test  (Codex#5.5#D · pane w3:p9)

## SEU PAPEL
Você é Codex-D, LÍDER do deploy P12. Único dono do hotspot `prodex-sidecar/`, `deploy/`, kill-switch.
Executa as tasks 12.0→12.7 EM ORDEM, sequencial. Você pode escalar sub-agentes D-A/D-B se necessário.

## DEPENDÊNCIA / DESBLOQUEIO
P12 está BLOCKED. Só inicie 12.1+ quando o dono fornecer (via TL): creds de provider REAL + host PROD real
(ver phases/12-prod-deploy/PREREQUISITES.md). SEM isso, NÃO rode sessão/deploy. Pergunte ao TL se falta.

## OBRIGATÓRIO antes de qualquer edição/comando
1. Ler `.planning/AGENT_LEDGER.md`, `phases/12-prod-deploy/PLAN.md`, `RESEARCH.md`, `EVIDENCE_CONTRACT.md`.
2. Criar check-in `.deploy-control/Codex-5.5-D__P12-PROD-DEPLOY__<UTC>.md` (files_locked, plan_ref, status).
3. Confirmar cada task-ID existe no PLAN. Passo não listado → PARA e chama Kiro para planejar.

## TASKS (siga RESEARCH.md — cada uma escreve evidência crua sob EVIDENCE_CONTRACT)
- **12.1** Sobe stack PROD: PG+Redis (compose), cria kill-switch store, migration up. Gate: PG alcançável.
- **12.2** Deploy binário PINADO (versão+commit reais, NÃO "smoke") no host PROD real; env com creds reais;
  start sidecar+gateway. Gate: /readyz 200 (derruba PG→503→restaura), /healthz ok.
- **12.3** Sessão REAL por vendor (creds do dono): /v1/session/start → /v1/runtime/proxy. ASSERTS obrigatórios:
  `gateway_status==200`, `measurement_source==gateway_usage`, `gateway_response_model != "fake-upstream-logging"`,
  usage NÃO 8/1, `runtime_session_id` e `tokens_saved` DISTINTOS por vendor. 1 arquivo/vendor: `P12-session-<vendor>-real.md`.
- **12.4** Kill-switch LIVE: request roteia (observa) → apply → para → remove → retoma. Captura before/after.
- **12.5** Rollback LIVE: 1 comando (runbook §7) → serviço volta a codex cru. Captura.
- **12.6** Logs scrubbed: `grep -RniE 'sk-|bearer|api[_-]?key|token='` no path PROD → 0 matches (mostra cmd+saída).
- **12.7** GATE: SUMMARY.md; backfill matriz P11 com números reais 12.3; commit+push (garanta target/ no .gitignore).

## PROIBIDO (auto-reject → INVALID)
localhost como "prod"; upstream fake/mock/replay; build smoke como "pinado"; key placeholder; usage trivial;
números idênticos entre vendors; escrever sign-off do dono/de outro agente. Nunca fingir para fechar gate.

## CHECK-OUT
Cada task: linha DONE no ledger + evidência. Falha: FAILED + erro + escala ao TL→Kiro. Reporta ao TL a cada 60s.
