# Guia de Teste — Isolamento de Credencial + Rotação (Fase 1 + Fase 2)

Cópia de trabalho: `multica-auth-work/`. Nada aqui altera o source original nem exige commit.
Todos os comandos usam containers (não precisa Go/psql instalados).

---

## A. Testes automatizados (o que já está verde)

### A1. Suite completa dos pacotes tocados (com git + Postgres real)
```bash
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
docker network create testnet 2>/dev/null
docker run -d --rm --name testpg --network testnet \
  -e POSTGRES_PASSWORD=pw -e POSTGRES_USER=aop -e POSTGRES_DB=aop postgres:17-alpine
sleep 6
docker run --rm --network testnet -v "$PWD":/src -w /src/server \
  -e DATABASE_URL="postgres://aop:pw@testpg:5432/aop?sslmode=disable" golang:1.26-alpine \
  sh -c "apk add --no-cache git >/dev/null 2>&1; git config --global user.email t@t; \
         git config --global user.name t; \
         go build ./... && go vet ./internal/... && \
         go test ./internal/daemon/... ./internal/rotation/... ./internal/metrics/..."
docker stop testpg; docker network rm testnet
```
Esperado: tudo `ok`, EXCETO o único teste ambiental tolerado
`TestValidateLocalPath/rejects_a_symlink_pointing_at_the_user_home` (container roda
como root — não é do nosso código).

### A2. Só a rotação (mais rápido)
```bash
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
docker network create rnet 2>/dev/null
docker run -d --rm --name rpg --network rnet -e POSTGRES_PASSWORD=pw -e POSTGRES_USER=aop -e POSTGRES_DB=aop postgres:17-alpine
sleep 6
docker run --rm --network rnet -v "$PWD":/src -w /src/server \
  -e DATABASE_URL="postgres://aop:pw@rpg:5432/aop?sslmode=disable" golang:1.26-alpine \
  sh -c "go test ./internal/rotation/... -v 2>&1 | tail -40"
docker stop rpg; docker network rm rnet
```
Cobre: detector (reativo), proactive (ledger 95%), pool (prioridade/cooldown),
service (OnExhaustion), PGStore (contra Postgres), authenticator, e o E2E (quando entregue).

### A3. Garantia AS-IS (não-regressão)
```bash
docker run --rm -v "$PWD":/src -w /src/server golang:1.26-alpine \
  go test ./internal/daemon/execenv/ -run Fallback -v
```
Prova: com `CredentialAccountHome == ""`, nenhum `CODEX_HOME/XDG_DATA_HOME/HOME` novo
é injetado → produto idêntico ao atual.

---

## B. Teste manual da isolação por conta (sem tocar produção)

Prova que 2 contas do mesmo vendor NÃO se sobrepõem:
```bash
# monta 2 dirs de conta com auth.json distintos
mkdir -p /tmp/acctA /tmp/acctB
echo '{"account":"A"}' > /tmp/acctA/auth.json
echo '{"account":"B"}' > /tmp/acctB/auth.json
# (o teste automatizado TestPrepareCodexHomePerAccountIsolatesAuth já prova isso;
#  este passo manual é para inspeção visual se desejar)
```

---

## C. Deploy real (GATED — só quando você decidir)

> Isto liga a rotação de verdade. Fora do escopo "AS-IS": só ativa quando
> `DATABASE_URL` estiver setado no ambiente do daemon.

### C1. Aplicar a migration da rotação
A migration `server/migrations/123_rotation.up.sql` cria as 4 tabelas
(`rotation_accounts`, `rotation_credentials`, `rotation_assignments`, `rotation_events`).
Aplique com o comando de migrate do projeto (`server/cmd/migrate`) apontando para o
Postgres de produção — o mesmo runner que aplica as demais migrations.

### C2. Ativar a rotação
- Sem `DATABASE_URL` no daemon → rotação DESLIGADA, comportamento atual preservado.
- Com `DATABASE_URL` apontando para o Postgres com o schema aplicado → rotação LIGA.

### C3. Popular contas
Inserir contas em `rotation_accounts` (vendor, tenant_id, priority, home_dir,
config_dir, status='available', tokens_per_win, window_start) e a referência de
credencial em `rotation_credentials` (secret_ref — NUNCA o token em claro).

### C4. Observabilidade (opcional, easy-deploy)
```bash
cd multica-auth-work/deploy/observability && docker compose up -d
```
Sobe Prometheus (9090), Grafana (3000), Alertmanager (9093), postgres-exporter (9187).

---

## D. Critérios de sucesso do teste de hoje
- [ ] A1 verde (só o teste ambiental de symlink falha).
- [ ] A2 rotação toda verde (incl. E2E quando entregue pelo CODEX-1).
- [ ] A3 fallback verde (garantia AS-IS).
- [ ] (Opcional) C4 observabilidade sobe e Grafana abre.