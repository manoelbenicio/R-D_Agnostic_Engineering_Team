# PROMPT — CODEX#b · STG-ROTATE (harness de rotação realtime, Postgres real)

## 1. PAPEL E MODO
Você é engenheiro Go sênior (verificação/integração). Tarefa de STAGING: provar, contra
Postgres REAL + pool seedado, que o sinal proativo dispara a rotação fim-a-fim. Você
entrega um TESTE de staging (arquivo NOVO, build tag) — NÃO edita código de produção.

## 2. AMBIENTE JÁ PRONTO (verificado pelo orquestrador)
- Stack up (backend `multica-backend-1`, Postgres `multica-postgres-1`), migrations ok.
- DB de dentro do container: `postgres://multica:multica@postgres:5432/multica?sslmode=disable`;
  do host: `postgres://multica:multica@localhost:5432/multica?sslmode=disable`.
- Caminho proativo JÁ existe em daemon.go (NÃO editar): `maybeProactiveRotateOnText` ->
  `rotateTaskProactively` -> `rotateTaskWithReason(ctx, task, provider, rotation.ReasonQuotaProactive, taskLog)`.
- Parser real: `usage.go`/`warnbanner.go`. Store: `store_pg.go` (Postgres).

## 3. REGRA DE OURO
- SOMENTE em multica-auth-work/. Sem commit. Você CRIA arquivo novo de teste:
  `server/internal/daemon/staging_rotation_smoke_test.go` com `//go:build staging`
  (p/ NÃO entrar no gate normal). NÃO editar daemon.go/rotation/*/execenv/* de produção.
- Sem segredo/e-mail/screen bruto em log/assert. Determinístico.
- DEPENDE do Agente A (pool seedado). Se `SELECT ... accounts WHERE vendor='codex'` < 2 →
  status BLOCKED (não prosseguir sem pool).

## 4. CHECK-IN
- Nome: CODEX-b__STG-ROTATE__<START_UTC>.md
- Front-matter: agent: CODEX#b / stream: STG-ROTATE / status: IN_PROGRESS /
  files_locked: [server/internal/daemon/staging_rotation_smoke_test.go] /
  depends_on: [STG-SEED] / ...

## 5. TAREFA
Escrever um teste taggeado `staging` que, contra o Postgres REAL (via DATABASE_URL) e o
pool seedado:
- monta o daemon com rotationService real (store Postgres) + as 2 contas codex;
- injeta o texto REAL de banner Codex: "less than 10% of your 5h limit left"
  pelo caminho `maybeProactiveRotateOnText` (provider="codex");
- ASSERTA: (a) rotaciona da conta prioridade 1 -> prioridade 2 EXATAMENTE uma vez;
  (b) uma linha nova em `rotation_events` com reason `quota_forecast_proactive`;
  (c) idempotência: reenviar o mesmo banner na mesma task NÃO gera 2ª rotação;
  (d) sem-conta/serviço-nil preserva AS-IS (não quebra).
Se a montagem do daemon com store real exigir helpers não expostos, usar os construtores
públicos existentes; NÃO alterar produção — se faltar ponto de acesso, BLOCKED com nota.

## 6. VERIFICAÇÃO (antes de DONE)
```
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
docker run --rm --add-host=host.docker.internal:host-gateway -v "$PWD":/src -w /src/server \
  -e DATABASE_URL="postgres://multica:multica@host.docker.internal:5432/multica?sslmode=disable" \
  golang:1.26-alpine sh -c "apk add --no-cache git >/dev/null 2>&1; adduser -D t >/dev/null; \
    mkdir -p /tmp/gc /tmp/gm; chown -R t /tmp/gc /tmp/gm; \
    su t -c 'GOCACHE=/tmp/gc GOMODCACHE=/tmp/gm go test -tags staging ./internal/daemon/ -run StagingRotation -v'" 2>&1 | tail -25
```
DONE só com o teste `staging` verde + a linha de `rotation_events` provada. Colar o tail.

## 7. RESUMO
ANTES: check-in; confirmar pool (>=2 codex). DURANTE: só o teste novo taggeado; sem tocar
produção. DEPOIS: check-out DONE + tail verde + evidência da linha proativa em rotation_events.
BLOCKED se pool vazio ou faltar acesso de montagem.
