# REALTIME E2E (Option 2) — Rotação dirigida pelo DAEMON vivo, métrica no /metrics

> Orquestrador/SME: Opus 4.8. Meta: o MAIS PRÓXIMO de PROD — uma rotação REAL fluindo
> pelo daemon vivo, incrementando `rotation_total` no `/metrics` do backend, observável.
> Opus prepara o ambiente (daemon+auth+workspace) e coordena; agentes fazem streams

## ⚠️ ACHADO CRÍTICO DE CONFIG (Codex#d, verificado Opus 2026-07-02) — DAEMON PRECISA DE DATABASE_URL
O daemon vivo SÓ habilita rotação se nascer com `DATABASE_URL` no ambiente — senão
`rotationStore` fica nil e a rotação é SILENCIOSAMENTE desativada (sem erro, só não
rotaciona). Em PROD/staging, garantir que o processo do daemon receba `DATABASE_URL`
apontando ao Postgres. Armadilha de falha silenciosa → documentar no guia de deploy do
daemon. Prova: task 8bed23df completou + rotation_events ...001→...002
`quota_forecast_proactive` só apareceram após reiniciar o daemon COM DATABASE_URL.
> disjuntos. Nada de métrica fabricada — só incremento real dirigido pelo daemon.

## FATOS CONFIRMADOS (Opus, contra o box)
- Stack self-host de pé (backend `multica-backend-1` :8080, Postgres `multica-postgres-1`).
- `/metrics` LIGADO agora: METRICS_ADDR=0.0.0.0:9090 no backend (83 famílias servidas).
  Famílias de rotação REGISTRADAS (sem série até 1a observação): `rotation_total`
  {vendor,reason,result}, `rotation_duration_seconds`{vendor}, `all_accounts_exhausted`
  {vendor}, `exhaustion_detected_total`{vendor,signal}.
- codex no host: `codex-cli 0.142.5` + `~/.codex/auth.json` real (600).
- Daemon roda FORA do Docker (doc self-host). Rodável da fonte: `make daemon`
  (= `multica daemon restart --profile local`), CLI da fonte: `make multica MULTICA_ARGS=...`.
- Login staging sem e-mail: APP_ENV=development + MULTICA_DEV_VERIFICATION_CODE=888888 (.env).
- Migration 123 aplicada (accounts/credentials/assignments/rotation_events).
- Caminho proativo do daemon (daemon.go): banner via maybeProactiveRotateOnText E ledger
  via ProactiveDetector.ShouldRotate (>=95% TokensUsed/TokensPerWin na janela 5h) ->
  rotateTaskWithReason(ReasonQuotaProactive) -> ObserveRotation emite `rotation_total`.

## ESTRATÉGIA REALTIME (honesta — sem forçar exaustão falsa)
Não dá para forçar o Codex a atingir 5h real em minutos. Usamos o caminho PROATIVO POR
LEDGER, que é PROD-legítimo: seed da conta ativa com TokensUsed >= 95% de TokensPerWin →
quando o daemon vivo processa uma task real Codex, o ProactiveDetector dispara a rotação
ENTRE tasks, incrementando `rotation_total` no backend vivo. É rotação real dirigida pelo
daemon, não métrica fabricada.

## ⚠️ ACHADO CRÍTICO DO OPUS (2026-07-02) — onde a métrica de rotação REALMENTE aparece
Investiguei o código antes de deixar os agentes baterem em endpoint errado:
- O backend (`cmd/server`, registry.go:58) cria o SEU PRÓPRIO `credentialMetrics` e o
  expõe no `/metrics` — mas o backend NÃO rotaciona. Esse `rotation_total` fica 0.
- O DAEMON (daemon.go:264) cria OUTRO `credentialMetrics` com `NewCredentialMetrics()`
  SEM registerer, em PROCESSO SEPARADO, e NÃO sobe metrics server. A rotação real do
  daemon incrementa esse contador em memória — que HOJE não é exposto em lugar nenhum.
- CONCLUSÃO HONESTA: "ver `rotation_total` incrementar no /metrics do backend vivo" NÃO
  é alcançável com o código atual (processos/registries distintos). Isso é um GAP de
  arquitetura de observabilidade do daemon, não falha de agente.
- SINAIS REAIS que PROVAM a rotação do daemon vivo (usar ESTES como evidência realtime):
  1. `rotation_events` no Postgres — linha nova com reason=quota_forecast_proactive
     (fonte de verdade persistida; INDISCUTÍVEL).
  2. LOG do processo daemon: "rotation: proactive quota signal detected" (stdout do daemon).
  3. Estado das contas em `accounts`: ativa vira exhausted/cooldown, próxima assume.
- Métrica Prometheus da rotação do daemon = ITEM DE BACKLOG (expor um metrics server no
  daemon OU registrar credentialMetrics do daemon). NÃO tentar forçar agora; registrar.

## PRÉ-REQUISITOS QUE O OPUS GARANTE (WAVE 0 — ambiente)
### WAVE 0 — PROGRESSO VERIFICADO PELO OPUS (2026-07-02)
- [FEITO] Go 1.26.0 instalado no HOST em `~/.local/go` (sem sudo, reversível). Uso:
  `export PATH=$HOME/.local/go/bin:$PATH; export GOCACHE=/tmp/gocache-host GOMODCACHE=/tmp/gomod-host`.
- [FEITO] CLI `multica` BUILDADA da fonte: `server/bin/multica` (29 MB) — `./bin/multica --help` ok.
- [FEITO] /metrics ligado no backend (METRICS_ADDR=0.0.0.0:9090, 83 famílias; rotação registrada).
- [FEITO] Auth headless PROVADA: `POST /auth/send-code {email}` então
  `POST /auth/verify-code {email,code:"888888"}` → 200 com `{token(JWT), user}`.
  (code fixo 888888 vale só com APP_ENV=development — que é o caso no .env staging.)
  ATENÇÃO: cada code é one-shot; capturar o token da MESMA resposta do verify-code.
- Rotas reais (server/cmd/server/router.go): `/api/me`, `/api/workspaces` (GET list, POST create),
  runtimes/tasks sob rotas de daemon. Autorização: header `Authorization: Bearer <token>`.
- codex no host: `~/.local/bin/codex` (codex-cli 0.142.5) + `~/.codex/auth.json` real.

### WAVE 0 — RESTANTE (executar via CLI, exatamente)
Usar SEMPRE: `cd server && PATH=$HOME/.local/go/bin:$PATH GOCACHE=/tmp/gocache-host \
GOMODCACHE=/tmp/gomod-host ./bin/multica --profile staging <cmd>` (ou o binário direto).
Config já setada: server_url=http://localhost:8080, app_url=http://localhost:3000
(profile staging, MULTICA_CONFIG_DIR=/tmp/multica-staging).
1. Autenticar o CLI: preferir `login --token <PAT>` se houver PAT; senão, como a verify-code
   devolve um JWT de sessão, gravar esse token no config do profile (campo auth_token) —
   confirmar o nome do campo em `./bin/multica config --help` / arquivo de config gerado.
2. Criar workspace (POST /api/workspaces ou `./bin/multica workspace create`). Anotar workspace_id.
3. Registrar runtime + iniciar daemon: `./bin/multica --profile staging daemon start`
   (ou `setup self-host`). Confirmar runtime no DB e codex detectado.
4. Criar 1 agente Codex no workspace (`./bin/multica agent ...`). Anotar agent_id.
5. Pool: conta ativa (stg-codex-a, prioridade 1) com TokensPerWin>0 e TokensUsed>=95% +
   WindowStart recente (reusar scripts/staging; NÃO inventar coluna) → ShouldRotate=true.
> Só depois disso as WAVES 1–2 (agentes) entram.

## WAVES
- WAVE 0 (Opus): itens 1–4 acima. Gate: daemon vivo registrado + conta ativa >=95% ledger.
- WAVE 1 (Agente D — Codex): dispatch de UMA task real ao agente Codex (issue simples) e
  confirmar que o daemon a executa. Arquivo: script/staging novo (dispatch + poll), sem
  tocar produção.
- WAVE 2 (Agente E — Codex): capturar a evidência REALTIME no backend vivo:
  `rotation_total{vendor="codex",reason="quota_forecast_proactive",result="ok"}` > 0 no
  `/metrics`, + linha de log do daemon "rotation: proactive quota signal detected", +
  linha nova em rotation_events. Doc de evidência.
- WAVE 3 (Opus): validação independente (re-scrape /metrics, ler rotation_events), aceite.

## STREAMS
### Agente D · RT-DISPATCH (Codex) — script novo, sem produção
- Cria scripts/staging/dispatch_codex_task.sh: via `make multica` cria/assigna 1 issue
  trivial ao agente Codex do workspace staging e dispara; faz poll do status até o daemon
  pegar a task. Verificação: task aparece RUNNING/DONE no DB (agent_task_queue) e o daemon
  logou execução. Locks: scripts/staging/dispatch_codex_task.sh. NÃO tocar Go de produção.
- BLOCKED se WAVE 0 (daemon/agent) não estiver pronto.

### Agente E · RT-OBSERVE (Codex) — doc novo, read-only
- Após a task rodar com a conta ativa >=95%: capturar do backend VIVO
  `docker exec multica-backend-1 sh -c 'wget -qO- 127.0.0.1:9090/metrics | grep rotation_total'`
  provando série com reason=quota_forecast_proactive > 0; log do daemon; rotation_events.
- Entrega: docs/project/realtime-rotation-evidence.md com os 3 sinais reais colados.
- Locks: doc novo. read-only no resto. BLOCKED se rotação não ocorreu (reportar p/ Opus).

## DEFINIÇÃO DE PRONTO (realtime)
- Daemon vivo registrado no backend staging, Codex detectado.
- Uma task real processada pelo daemon com a conta ativa em >=95% ledger.
- `rotation_total{reason="quota_forecast_proactive"}` INCREMENTADO no /metrics do backend vivo.
- rotation_events com a linha real; log do daemon com o sinal proativo.
- Opus re-valida por scrape independente. Sem métrica fabricada.

## RISCOS / GUARDA
- Rotação por ledger é PROD-legítima (mesmo caminho do proativo real), só o gatilho é
  seedado — documentar isso claramente (não é exaustão de 5h real, é forecast >=95%).
- Nada publicado no host além do necessário; /metrics só na rede do container.
- Agentes tocam só scripts/staging + doc; zero produção. Opus valida por scrape real.