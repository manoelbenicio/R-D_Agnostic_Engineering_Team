# PROMPT — CODEX#a · STG-SEED (pool de rotação no Postgres de staging)

## 1. PAPEL E MODO
Você é engenheiro de dados/Go sênior. Tarefa de STAGING: semear o pool de rotação
(2 contas Codex) no Postgres REAL da stack Multica que já está DE PÉ neste lab.
Você entrega SQL de seed (arquivo NOVO) + verificação. NÃO escreve código de produção.

## 2. AMBIENTE JÁ PRONTO (verificado pelo orquestrador — não remexer)
- Stack up: `docker compose -f docker-compose.selfhost.yml -f docker-compose.selfhost.build.yml`
  em /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work. Backend `multica-backend-1`,
  Postgres `multica-postgres-1`. `/healthz` = db ok, migrations ok.
- Migration 123 aplicada: tabelas `accounts`, `credentials`, `assignments`, `rotation_events`.
- Acesso ao DB: `docker exec -i multica-postgres-1 psql -U multica -d multica`.

## 3. REGRA DE OURO
- SOMENTE em /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work/. Sem commit.
- Você CRIA arquivo novo: `scripts/staging/seed_rotation_pool.sql` (+ README curto se útil).
- NÃO edita Go de produção, NÃO edita migrations/*, NÃO edita execenv/daemon/rotation.
- LER o schema REAL antes: `server/migrations/123_rotation.up.sql`. Usar SOMENTE colunas
  que existem lá. NÃO inventar coluna/constraint. Se o schema não bater → BLOCKED.
- Credencial por REFERÊNCIA (nunca segredo em claro; nunca logar segredo). Os dirs de
  credencial de simulação são fake e isolados (ex.: 2 pastas distintas).

## 4. CHECK-IN (antes de editar)
- Local: Automonous_Agentic/.deploy-control/ (board na raiz).
- Nome: CODEX-a__STG-SEED__<START_UTC>.md  (START_UTC via: date -u +%Y%m%dT%H%M%SZ)
- Front-matter: agent: CODEX#a / stream: STG-SEED / started_at / finished_at: / status: IN_PROGRESS
  / files_locked: [scripts/staging/seed_rotation_pool.sql] / depends_on: [] / build_result: / notes:

## 5. TAREFA
Criar `scripts/staging/seed_rotation_pool.sql` IDEMPOTENTE (re-executável) que insere:
- 2 rows em `accounts` para vendor `codex`, com prioridades DISTINTAS (ex.: 1 e 2),
  status `available`, e campos de janela/uso realistas (tokens_per_win > 0; a conta de
  prioridade 1 já perto do limite p/ facilitar o teste de rotação do Agente B — ex.:
  tokens_used ~ 96% de tokens_per_win — SE essas colunas existirem no schema real; se
  não existirem, usar as colunas equivalentes reais e anotar em notes).
- Para CADA conta: row correspondente em `credentials` (por referência) + `home_dir`
  apontando p/ 2 dirs isolados de simulação (o orquestrador confirmou: a stack NÃO
  pré-criou dirs de credencial — `accounts` está VAZIA — então VOCÊ os cria).
- Usar UPSERT (ON CONFLICT) p/ ser idempotente. IDs estáveis (ex.: 'stg-codex-a'/'stg-codex-b').

### FONTE DE CREDENCIAL RESOLVIDA PELO ORQUESTRADOR (não reabrir)
- Para Codex, `CredentialAccountHome` = um diretório contendo `auth.json` (confirmado em
  execenv/codex_home.go: o daemon COPIA `AccountHome/auth.json` p/ o home por-task).
- Existe um `auth.json` REAL e válido em `~/.codex/auth.json` (mode 600). Clonar ele p/
  DOIS dirs isolados de simulação:
    scripts/staging/creds/codex-a/auth.json
    scripts/staging/creds/codex-b/auth.json
  (cada um cópia do real, mode 600; NUNCA imprimir/logar o conteúdo do token).
- `accounts.home_dir` da conta A -> caminho absoluto de scripts/staging/creds/codex-a;
  conta B -> .../codex-b. (Se a coluna se chamar diferente no schema real, usar a real e
  anotar em notes; NÃO inventar.)
- Passo do seed: criar os 2 dirs + `cp ~/.codex/auth.json` p/ cada, `chmod 600`, ANTES do
  UPSERT nas tabelas. Idempotente (não sobrescrever se já válido).

## 6. VERIFICAÇÃO (antes de DONE — colar no build_result)
```
docker exec -i multica-postgres-1 psql -U multica -d multica < scripts/staging/seed_rotation_pool.sql
docker exec -i multica-postgres-1 psql -U multica -d multica -c \
  "SELECT account_id, vendor, priority, status FROM accounts WHERE vendor='codex' ORDER BY priority;"
```
DONE só com 2 contas codex `available` retornadas, prioridades distintas. Rodar o seed
DUAS vezes p/ provar idempotência (2ª execução não duplica nem erra).

## 7. RESUMO
ANTES: check-in. DURANTE: só o .sql novo; ler schema real; nada inventado; sem segredo.
DEPOIS: check-out com finished_at + DONE + saída da query colada. BLOCKED se o schema
real divergir do esperado (descrever a coluna faltante em notes).
