# PROMPT — CODEX#d · RT-SETUP (daemon vivo + task real que dispara rotação)

## 0. ANTI-ALUCINAÇÃO (regras duras)
- NÃO invente comando/flag/campo/rota. Os fatos verificados estão em
  .deploy-control/REALTIME_E2E_RUNBOOK.md (seção WAVE 0) — LEIA antes. Se um comando não
  existir como você espera, rode `--help` do próprio CLI; NÃO adivinhe.
- Toda afirmação de progresso precisa de saída de comando REAL colada. Se algo falhar,
  reporte a saída real e marque BLOCKED — NUNCA fabrique sucesso.
- Sem segredo/token/e-mail em claro no log/check-out (mascare tokens: primeiros 6 chars + ***).

## 1. PAPEL
Você é CODEX#d. Preparar o ambiente realtime e disparar UMA task real que faça o DAEMON
VIVO rotacionar a conta (caminho proativo por ledger). NÃO edita código de produção;
só scripts/staging novos + comandos de operação (CLI/psql/curl).

## 2. FATOS VERIFICADOS (do runbook — não reabrir)
- Stack up: backend `multica-backend-1` :8080 (/metrics em 0.0.0.0:9090 dentro do container),
  Postgres `multica-postgres-1` (rede compose `multica_default`, NÃO publicado no host).
- CLI já buildado: `server/bin/multica`. Rodar com:
  `cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work/server && \
   PATH=$HOME/.local/go/bin:$PATH GOCACHE=/tmp/gocache-host GOMODCACHE=/tmp/gomod-host \
   MULTICA_CONFIG_DIR=/tmp/multica-staging ./bin/multica --profile staging <cmd>`
  (server_url/app_url já configurados p/ localhost:8080 / :3000).
- Auth headless PROVADA: `curl -s -X POST http://localhost:8080/auth/send-code -H 'Content-Type: application/json' -d '{"email":"staging-rotation@example.com"}'`
  então `POST /auth/verify-code {"email":"staging-rotation@example.com","code":"888888"}` → `{token,user}`.
  Cada code é one-shot: capture o token DA MESMA resposta do verify-code.
- Rotas: `/api/me`, `/api/workspaces` (GET/POST), auth por header `Authorization: Bearer <token>`.
- codex no host: `~/.local/bin/codex` (0.142.5) + `~/.codex/auth.json`.
- Pool seedado: accounts codex `stg-codex-a`(UUID ...001, prio 1), `stg-codex-b`(...002, prio 2),
  ambas available; creds em scripts/staging/creds/codex-{a,b}/auth.json.
- Caminho de rotação alvo: LEDGER proativo — conta ativa com TokensUsed>=95% de TokensPerWin
  e WindowStart recente → daemon dispara ReasonQuotaProactive → `rotation_total`++ no /metrics.

## 3. CHECK-IN
- Nome: CODEX-d__RT-SETUP__<START_UTC>.md (date -u +%Y%m%dT%H%M%SZ)
- Front-matter: agent: CODEX#d / stream: RT-SETUP / status: IN_PROGRESS /
  files_locked: [scripts/staging/rt_setup.sh, scripts/staging/set_active_account_ledger.sql] /
  depends_on: [STG-SEED] / ...

## 4. TAREFA (passos; cole a saída real de cada um no check-out)
P1. Autenticar o CLI: obter token via verify-code (curl acima), depois gravar no profile:
    inspecionar `./bin/multica config --help` e o arquivo de config gerado em
    /tmp/multica-staging p/ achar o campo de token (ex.: auth_token) e setá-lo, OU usar
    `./bin/multica login --token <token>` se aceitar JWT. Confirmar com `./bin/multica --profile staging auth status`.
P2. Criar workspace: `./bin/multica --profile staging workspace create` (ver `--help`) OU
    `curl -s -X POST http://localhost:8080/api/workspaces -H "Authorization: Bearer <tok>" ...`.
    Anotar workspace_id.
P3. Iniciar daemon vivo: `./bin/multica --profile staging daemon start` (ver `--help`;
    pode exigir --workspace-id). Confirmar: `./bin/multica --profile staging daemon status`
    mostra rodando e codex detectado; e no DB há runtime registrado.
P4. Criar 1 agente Codex no workspace (`./bin/multica --profile staging agent create ...`,
    provider codex). Anotar agent_id.
P5. Ledger da conta ativa: criar scripts/staging/set_active_account_ledger.sql que faz
    UPDATE em accounts (prioridade 1 / stg-codex-a) setando tokens_per_win>0,
    tokens_used>=95% desse valor, window_start=now() (USAR NOMES REAIS das colunas —
    ler migrations/123_rotation.up.sql; NÃO inventar). Aplicar via
    `docker exec -i multica-postgres-1 psql -U multica -d multica < ...`. Verificar com SELECT.
P6. Dispatch de UMA task real: criar/assinar 1 issue trivial ao agente Codex e disparar
    (`./bin/multica --profile staging issue create ...` + assign, ou o fluxo do CLI). Poll:
    confirmar no DB (agent_task_queue) que o daemon PEGOU e está RUNNING/DONE.
    Empacotar P1-P6 reproduzíveis em scripts/staging/rt_setup.sh (idempotente onde possível).

## 5. VERIFICAÇÃO (antes de DONE)
- `daemon status` = running + codex detectado (colar).
- SELECT provando conta ativa com tokens_used>=95%*tokens_per_win (colar).
- agent_task_queue mostrando a task processada pelo runtime (colar).
- DONE = daemon vivo + task real processada + ledger>=95% armado. A CAPTURA da métrica
  de rotação é do Agente E (RT-OBSERVE); você deixa o gatilho armado e a task rodando.
- BLOCKED (com saída real) se: auth não persistir, daemon não registrar, ou codex não detectado.

## 6. RESUMO
Só scripts/staging + operação; nada de produção; nada inventado; tokens mascarados.
Check-out DONE com as 3 evidências coladas + workspace_id/agent_id/runtime_id anotados.
