# PROMPT — CODEX#e · RT-OBSERVE (evidência realtime da rotação no backend vivo)

## 0. ANTI-ALUCINAÇÃO (regras duras)
- NÃO invente nome de métrica/label/comando. Fatos reais abaixo + em
  .deploy-control/REALTIME_E2E_RUNBOOK.md e docs/project/observability-rotation-staging.md.
- Toda evidência é saída REAL colada. Se a métrica não incrementar, reporte o valor real
  observado e marque BLOCKED — NUNCA fabrique. "NÃO CAPTURADO" é resposta válida.
- Sem segredo/e-mail em logs colados.

## 1. PAPEL
Você é CODEX#e. Capturar a prova REALTIME de que a rotação incrementou a métrica no
backend VIVO, com a task do Agente D rodando e a conta ativa em >=95% ledger. Entrega:
1 doc novo com evidência real. NÃO edita código; read-only + doc.

## 2. FATOS REAIS (não reabrir) — ONDE A ROTAÇÃO REALMENTE APARECE
⚠️ ACHADO DO OPUS (ler REALTIME_E2E_RUNBOOK.md "ACHADO CRÍTICO"): o daemon roda em
PROCESSO SEPARADO com `credentialMetrics` próprio SEM metrics server. Logo o
`rotation_total` da rotação do daemon NÃO aparece no /metrics do backend (que tem outro
registry e fica 0). NÃO perca tempo no /metrics do backend para provar a rotação do daemon.
SINAIS REAIS que provam a rotação do daemon vivo (use ESTES três):
1. `rotation_events` (Postgres): linha nova reason=`quota_forecast_proactive` — verdade persistida.
   `docker exec -i multica-postgres-1 psql -U multica -d multica -c "SELECT from_account_id,to_account_id,reason,at FROM rotation_events WHERE reason='quota_forecast_proactive' ORDER BY at DESC LIMIT 5;"`
2. LOG do processo DAEMON: "rotation: proactive quota signal detected" (stdout do daemon do Agente D).
3. `accounts`: conta ativa (prio 1) vira exhausted/cooldown, prio 2 assume.
   `docker exec -i multica-postgres-1 psql -U multica -d multica -c "SELECT priority,status,tokens_used,tokens_per_win FROM accounts WHERE vendor='codex' ORDER BY priority;"`
Métrica Prometheus da rotação do daemon = BACKLOG (não exposta hoje) — documentar, não forçar.

## 2b. FATOS DE MÉTRICA (contexto, não é a prova)
- Nomes reais (credential_metrics.go): `rotation_total`{vendor,reason,result} etc. Estão no
  backend /metrics mas ficam 0 (backend não rotaciona). Documentar isso como o GAP.

## 3. CHECK-IN
- Nome: CODEX-e__RT-OBSERVE__<START_UTC>.md
- Front-matter: agent: CODEX#e / stream: RT-OBSERVE / status: IN_PROGRESS /
  files_locked: [docs/project/realtime-rotation-evidence.md] / depends_on: [RT-SETUP] / ...

## 4. TAREFA (passos; cole saída real)
P1. Baseline ANTES: contagem em `rotation_events` (reason=quota_forecast_proactive) e
    estado das contas codex (priority/status). Colar.
P2. Garantir que a task do Agente D rodou COM a conta ativa >=95% ledger; aguardar o
    ciclo do daemon (poll do rotation_events / accounts).
P3. Evidência DEPOIS (as 3, coladas — ver seção 2):
    (a) `rotation_events`: nova linha reason=quota_forecast_proactive (from prio1 → to prio2);
    (b) LOG do daemon "rotation: proactive quota signal detected" (stdout do daemon do Agente D);
    (c) `accounts`: prio1 exhausted/cooldown, prio2 assume.
P4. Escrever docs/project/realtime-rotation-evidence.md: descrever que é rotação proativa
    por ledger (PROD-legítima, gatilho seedado a >=95%, honesto vs exaustão natural de 5h),
    os 3 sinais reais colados com o comando de cada um, E registrar o GAP de observabilidade:
    a métrica Prometheus da rotação do daemon não é exposta hoje (backlog), por isso a prova
    é rotation_events + log + accounts, não /metrics.

## 5. VERIFICAÇÃO (antes de DONE)
- Doc contém os 3 sinais reais (rotation_events + log do daemon + estado das contas).
- Colar no build_result a linha de rotation_events (reason=quota_forecast_proactive) + o log.
- DONE só com a rotação do daemon vivo PROVADA por rotation_events (verdade persistida).
- BLOCKED (com saída real) se NÃO houver linha de rotation_events após a task rodar com
  ledger>=95% — reportar p/ Opus (provável: daemon não pegou a task, ou ledger não armado).

## 6. RESUMO
Read-only + 1 doc; nada de produção; nada inventado. Evidência realtime colada ou BLOCKED honesto.
