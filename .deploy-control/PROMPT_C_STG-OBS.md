# PROMPT — CODEX#c · STG-OBS (verificação de observabilidade da rotação)

## 0. ANTI-ALUCINAÇÃO (ler primeiro — regras duras)
- NÃO invente nomes de métrica, label, arquivo ou comando. TODOS os fatos que você
  precisa estão abaixo, já extraídos do código real pelo orquestrador. Se algo não
  estiver aqui, LEIA o arquivo citado — NÃO adivinhe.
- NÃO edite NENHUM código. Sua entrega é UM arquivo de doc novo + evidência real colada.
- NÃO crie containers, proxies, nem publique portas. Use os comandos EXATOS desta página.
- Toda afirmação no doc precisa de evidência (saída de comando colada) OU citação
  arquivo:linha. Sem "provavelmente", sem invenção. Se não conseguir capturar algo,
  escreva "NÃO CAPTURADO" — nunca fabrique saída.

## 1. PAPEL E MODO
Você é CODEX#c, engenheiro de observabilidade. Tarefa de STAGING (read-only + 1 doc):
gerar UMA rotação real, capturar a evidência (log + métrica) e documentar os nomes reais
das métricas para o dashboard. A rotação já foi provada verde; você só a observa e documenta.

## 2. FATOS REAIS (extraídos do código — NÃO reabrir)
Emissores em daemon.go (NÃO editar): `rotateTaskWithReason` chama
`credentialMetrics.ObserveRotation(provider, reason, result, seconds)` (daemon.go:3969 ok /
3962 error) e `SetAllAccountsExhausted(provider, bool)` (daemon.go:3956/3968).
Log estruturado: "rotation: proactive quota signal detected" (rotateTaskProactively).
Métricas Prometheus REAIS (fonte: internal/metrics/credential_metrics.go) — nomes e labels:
- `rotation_total` (CounterVec) labels: vendor, reason, result   ← via ObserveRotation
- `rotation_duration_seconds` (HistogramVec) labels: vendor       ← via ObserveRotation
- `all_accounts_exhausted` (GaugeVec) labels: vendor              ← via SetAllAccountsExhausted
- `exhaustion_detected_total` (CounterVec) labels: vendor, signal ← via ObserveExhaustionDetected
- (contexto) `accounts_available`{vendor}, `account_status`{vendor,account_id,status},
  `account_tokens_used`, `account_window_seconds_remaining`, `credential_restore_total`,
  `cred_env_injection_total`, `credential_prepare_seconds`.
`reason` para rotação proativa = `quota_forecast_proactive` (rotation/contract.go).
Ambiente: backend `multica-backend-1`, Postgres `multica-postgres-1` (NÃO publicado no
host; só na rede compose `multica_default`). Teste staging já existe:
server/internal/daemon/staging_rotation_smoke_test.go (//go:build staging).

## 3. REGRA DE OURO
- SOMENTE em multica-auth-work/ e docs/project/. Sem commit. CRIA arquivo novo:
  `docs/project/observability-rotation-staging.md`. NÃO editar código/metrics/produção.
- Sem segredo/e-mail/token em exemplos ou logs colados.

## 4. CHECK-IN (antes de editar)
- Local: Automonous_Agentic/.deploy-control/ (board na raiz).
- Nome: CODEX-c__STG-OBS__<START_UTC>.md  (START_UTC via: date -u +%Y%m%dT%H%M%SZ)
- Front-matter: agent: CODEX#c / stream: STG-OBS / started_at / finished_at: /
  status: IN_PROGRESS / files_locked: [docs/project/observability-rotation-staging.md] /
  depends_on: [STG-ROTATE] / build_result: / notes:

## 5. TAREFA (passos EXATOS, nesta ordem)
Passo 1 — gerar uma rotação real (roda o teste na rede compose; NÃO usar proxy):
```
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
docker run --rm --network multica_default -v "$PWD":/src -w /src/server \
  -e DATABASE_URL="postgres://multica:multica@postgres:5432/multica?sslmode=disable" \
  golang:1.26-alpine sh -c "apk add --no-cache git >/dev/null 2>&1; adduser -D t >/dev/null; \
    mkdir -p /tmp/gc /tmp/gm; chown -R t /tmp/gc /tmp/gm; \
    su t -c 'GOCACHE=/tmp/gc GOMODCACHE=/tmp/gm go test -tags staging ./internal/daemon/ -run StagingRotation -v'" 2>&1 | tail -8
```
Passo 2 — provar a rotação no banco (evidência):
```
docker exec -i multica-postgres-1 psql -U multica -d multica -c \
  "SELECT reason, count(*) FROM rotation_events WHERE reason='quota_forecast_proactive' GROUP BY reason;"
```
(O teste faz cleanup ao final; se retornar 0, rode o Passo 1 e o Passo 2 na MESMA janela,
ou documente que a contagem é observada durante o teste via log — ver Passo 3. NÃO fabricar.)
Passo 3 — capturar o log real de sinal proativo:
```
docker logs multica-backend-1 2>&1 | grep -i "rotation:" | tail -5
```
Se o backend-1 não tiver a linha (o teste roda o daemon em processo próprio, não no
backend-1), então a evidência de log vem do stdout do Passo 1 (o `-v` do teste imprime
a linha "rotation: proactive quota signal detected"). Use a que existir; declare a fonte.
Passo 4 — métricas: verificar se METRICS_ADDR está ligado no backend:
```
docker exec multica-backend-1 sh -c 'echo METRICS_ADDR=$METRICS_ADDR'
```
Se VAZIO: documentar que /metrics está OFF por padrão e como ligar (env METRICS_ADDR=
127.0.0.1:9090 no .env + restart do backend) — SEM alterar produção agora, apenas instrução.
Se LIGADO: `docker exec multica-backend-1 sh -c 'wget -qO- 127.0.0.1:9090/metrics | grep -E "rotation_total|all_accounts_exhausted|exhaustion_detected_total"'` e colar.

## 6. ENTREGA — docs/project/observability-rotation-staging.md
Conteúdo obrigatório:
- Tabela: métrica | tipo | labels | significado | emissor (arquivo:linha) — usando os
  nomes reais da seção 2.
- Queries/greps prontos p/ o operador acompanhar rotações em tempo real (os da seção 5).
- Evidência REAL colada: saída do Passo 1 (PASS + linha de log), Passo 2 (contagem) e
  Passo 3/4 (log e/ou métricas). Cada bloco rotulado com o comando que o gerou.
- Se algo ficou OFF (ex.: METRICS_ADDR), registrar como "estado atual + como ligar".

## 7. VERIFICAÇÃO (antes de DONE)
- O doc existe, tem a tabela com nomes reais (batendo com credential_metrics.go), e
  pelo menos DUAS evidências reais coladas (o PASS do teste + a linha de log OU a métrica).
- Colar no build_result do check-out o tail do Passo 1 (PASS) e o grep de log.
- DONE só com evidência real. Se não capturar métrica (METRICS_ADDR off), DONE mesmo assim
  desde que documente o estado e o log de rotação esteja provado. BLOCKED só se o teste
  do Passo 1 não passar (aí é regressão — reportar).

## 8. RESUMO
ANTES: check-in. DURANTE: rodar os comandos EXATOS da seção 5; só o doc novo; nada de
produção; nada inventado. DEPOIS: check-out DONE + tail PASS + log colados. Nomes de
métrica vêm da seção 2 / credential_metrics.go — nunca de memória.
