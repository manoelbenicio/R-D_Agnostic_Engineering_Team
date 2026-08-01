# Proposal — Credential Account Home Restoration

## Why

Os runtimes **agy**, **codex** e **kiro** sao obrigatorios em todo cenario (decisao do owner).
Hoje eles operam apenas com **uma conta por provider**, usando o HOME global do processo do
daemon no ORQ1. Nao existe isolamento por conta, nao existe rotacao e nao existe atribuicao
de custo por conta.

A causa nao e feature ausente: e **regressao**. O wiring correto existiu no commit `aa62401`
(2026-07-02). O commit `31d50b9` (2026-07-05) condicionou esse wiring ao antigo L2; a remocao
completa do resolver e o `CredentiallessGateway: true` incondicional ocorreram em `9ab80a6`.

## What Changes

- **RESTORED** resolucao de `CredentialAccountHome` por task e por provider no daemon.
- **ADDED** ponte entre o registry de isolamento existente no ORQ2 e a selecao por task no daemon.
- **MODIFIED** o caminho nativo volta a preparar e injetar o ambiente isolado; o caminho
  gateway continua credentialless e nunca recebe `AccountHome`.
- **ADDED** atribuicao deterministica e persistente por identidade estavel
  `AGENT_CRED_ISOLATION_AGENT_ID + AGENT_CRED_ISOLATION_SUBSCRIPTION_FINGERPRINT`, com falha
  fechada quando ausente ou invalida, sem identidade derivada de Herdr, pane, TTY, PID ou UUID
  aleatorio.
- **ADDED** cardinalidade fisica de exatamente uma pasta por binding ativo estavel, zero pastas
  historicas, com tombstones de metadados preservados para nao-reuso e sem criacao de pasta pela
  camada de catalogo.
- **MODIFIED** o discovery de modelos AGY executa somente sob homes validados da allowlist,
  sem herdar o HOME global do daemon.
- **MODIFIED** o task-home AGY recebe somente o arquivo fisico
  `antigravity-oauth-token`; logs, symlinks, caches e bancos do provider nao sao copiados.
- **MODIFIED** o formulario de criar/duplicar agente persiste `thinking_level` quando o
  catalogo estruturado do runtime oferece `thinking.supported_levels`; modelos AGY, cujo tier
  ja faz parte do ID, continuam sem um segundo seletor.
- **ADDED** snapshot imutavel da conta produtora no claim atomico da task e copia server-side
  para `task_usage`, sem aceitar `account_id` do daemon.
- **ADDED** migration canonica `128_task_usage_account_id`, com exclusividade duravel de conta,
  rollback e preservacao de linhas legadas como nao atribuiveis.

## Scope

Vendors: **agy/antigravity, codex e kiro**.
**Non-goals:** `cline`, `opencode` e qualquer outro provider.

## Non-Goals

- Reimplementar o isolamento fisico de credencial no ORQ2. Os scripts de isolamento existem e
  operam. A composicao concreta do Runtime Manager em PostgreSQL **esta implementada e validada na
  fonte compartilhada `spe6`** — idempotencia duravel de criacao e ativacao, locking de parent,
  UUID esperado fail-closed, `apply_class` persistido, router, middleware e startup. O que falta e
  a importacao seletiva com revisao na arvore de release aceita e a verificacao contra banco real.
- Inventar, normalizar ou substituir o catalogo fornecido por cada runtime/provider.
- Alterar o isolamento de slots existente no ORQ2.

## Impact

- Isolamento e afinidade por conta passam a existir no executor T2.
- Exige rebuild Go e relancamento do daemon (classe STOP-AND-WAIT).
- A persistencia financeira foi implementada e passou no gate combinado ORQ-12/ORQ-21 em
  `ea1eee7`; a integracao conjunta e a revisao independente continuam sendo gates de release.

## Implementation update — 2026-07-28

- `785a8ac` mantem linhas legadas `dispatched` com snapshot NULL visiveis ao reclaim, para que
  o gate fail-closed do ORQ-21 possa cancela-las; reclaim nunca resolve ou inventa conta.
- `ea1eee7` promove os arquivos staged para `128_task_usage_account_id.{up,down}.sql`, remove a
  configuracao SQLC temporaria e regenera a saida canonica.
- Gate descartavel: ORQ-12 19/19 em duas passagens, ORQ-21 handler 2/2 e registry 2/2 com
  `orq21db`, integracao 19+2+2, todos sob race, zero skips; build, vet e gofmt limpos.
- Os commits nao devem ser integrados isoladamente: ORQ-12 depende do cancelamento fail-closed
  do stack ORQ-21 compativel.

## Estado atual — 2026-08-01

Release: **HOLD / NOT READY**. Nenhum push, deploy ou restart foi executado ou autorizado.

- `go build ./...` na arvore aceita **passa**, apos companions aditivos exatos em
  `internal/daemon/config.go`, `execenv/codex_home.go`, `execenv/cline_home.go`,
  `internal/service/task.go`, `internal/middleware/request_logger.go` e o leitor de senha limitado
  em `cmd/multica/cmd_user.go`. O bloqueio original `ExactEnv`/`deliveryObs` esta resolvido.
- Suites de pacote verdes: `./pkg/redact`; `./pkg/agent` (7.969s); `./internal/realtime` completo (0.743s);
  `./internal/realtime` (0.743s); `./internal/credentialregistry`;
  `./internal/daemon/credentialcatalog` com e sem `-race`.
- **Suites verdes nao implicam release.** A composicao concreta do Runtime Manager em PostgreSQL
  esta implementada e validada na fonte compartilhada `spe6`: idempotencia duravel de criacao e
  ativacao, locking de parent, UUID esperado fail-closed, `apply_class` persistido, e mount real
  sob router com middleware no startup. O que falta e a importacao seletiva com revisao na arvore
  de release aceita e a verificacao contra banco real; ate lá, nada disso esta alcancavel na arvore
  aceita.
- Os grupos de companions I, J, K e L foram executados e congelados na arvore aceita; os bloqueadores
  de compilacao chegaram a zero. `go build ./...` e a varredura `go test -run ^$ ./...` passam, e
  `go test ./cmd/multica` deixou de falhar. Suites completas dos pacotes afetados: `passwordtest`
  0.244s, `auth` 0.997s, `middleware` 0.558s. **`internal/handler` nao conta como suite executada:**
  a invocacao completa do pacote e compilacao mais skip do `TestMain` quando PostgreSQL esta ausente,
  por isso a duracao proxima de zero; ela nao evidencia testes executados. A varredura `-run ^$` prova
  compilacao, nao execucao de testes; nao houve execucao completa de testes de todo o repositorio.
- Trabalho sistemico de companions na arvore aceita esta **pendente**; a importacao simbolo a
  simbolo foi encerrada em favor de inventario e classificacao sistematicos.
- O gate de banco real **passou** em PostgreSQL descartavel: migracao limpa ate 135; linhas
  protegidas inalteradas no down/up reversivel 134, 133, 132; down de 131 recusado com SQLSTATE 55000
  e a mensagem de politica exata; `TestRuntimeManagerReservationPrimitives` 0.165s; e
  `go test -count=1 ./...` com `SPE6_RUNTIME_MANAGER_DATABASE_URL` apontando para o banco limpo
  aprovando **todos** os pacotes, incluindo `pkg/db/generated` 0.160s e `daemon` 57.673s. Container
  auto-limpo; nenhum estado persistente criado.
- Falhas de redacao de stderr e de cleanup de arvore de processos em `pkg/agent` eram
  **preexistentes**, nao regressoes desta integracao, e foram reparadas.
- Nomes externos canonicos do Runtime Manager: `version` como discriminador de documento,
  `configuration_digest` e `capability_digest` como digests, sempre hex minusculo de 64
  caracteres sem prefixo. Nao existem `active_digest` nem
  `effective_configuration_digest`. O `schema_version` interno do preimage de digest **permanece
  inalterado**, pois renomea-lo mudaria todo digest ja produzido.
- A correcao de identidade e cardinalidade e **somente de codigo-fonte**. O alocador instalado no
  ORQ2 mantem o defeito de origem, portanto o crescimento ilimitado de pastas continua alcancavel
  ate um rebuild e restart nao autorizados.
- A arvore raiz destacada do ORQ2 **nao** e alvo de release. A fonte de integracao C2/C3 e
  `worktrees/spe6-runtime-schema`; a importacao final e seletiva, sem merge de branch inteira.
- Migration 128 verificada inalterada: up `4b0894920069336efc36db645b05fae775a3b130`, down
  `a2b2ea2b56d10f57fe0144bc6998a88277039643`.
