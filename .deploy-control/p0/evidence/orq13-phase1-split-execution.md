# ORQ-13 fase 1 — execucao do split A/B (V3.1 com rollback seguro)

- **Executor:** Opus48#B (ORQ2, w6:p2) — **owner exclusivo** do worktree e da reserva
- **Autorizacao:** Gate 0 aceito; ordem de execucao 2026-07-27T17:46Z
- **Worktree:** `/home/ec2-user/workspace/worktrees/gtl-i03-orq13-phase1`, branch `agent/opus48-b/orq-13-thinking-level`
- **Reserva:** RES-ORQ13-001, migration **127**
- **VEREDITO: GO** para o split. Uma pendencia de evidencia isolada e declarada na secao 6.

## 1. Gates PRE-execucao — todos verdes

| gate | medido | resultado |
|---|---|---|
| TTL da reserva | agora `2026-07-27T17:47:01Z` vs expiracao `2026-07-28T15:37:12Z` | **VALIDO, 21 h restantes** |
| base | `HEAD == 0cb8aeb` | OK |
| limpeza | `staged=0`, `commits_ahead=0`, `stash=0` | OK |
| fingerprint do diff | `b87bc5cae3e877af9a26ebe39cce7f0a90fff2a126e075c7f5d12799d04707ce` | **identico ao registrado** |
| migration 127 up | `0ea3005da0ee257618062552cf8792f9c2e6ed478dce3ba174bf08692486cac1` | confere com a reserva |
| migration 127 down | `74354ae28dee526c7dbc6bc6733471a59c2f3dabfe5a7fe609fe20d747e61113` | confere com a reserva |
| colisao de numero | `125`, `126`, `127` — nenhuma outra `127` | OK |
| dependencias | `sqlc` v1.31.1 local, go 1.26.1, gofmt, git 2.50.1 | presentes |

## 2. Rollback seguro — o `git reset --hard` foi ELIMINADO

A V3.1 usava `git reset --hard` como ponto de aborto. **Substituido** por tres redes
independentes, nenhuma destrutiva:

1. **Tag de salvaguarda** `orq13-safeguard-base` -> `0cb8aeb`, o ponto exato de partida.
2. **Snapshot verificado** em `/home/ec2-user/.cache/orq13-safeguard/` (dir `0700`): patch completo
   dos rastreados (`full-tracked.patch`, sha256 comeca `b87bc5cae3e877af`), copia integral dos **11**
   arquivos em `tree/`, e `tree.sha256` com o hash de cada um.
3. **Stash preservado** `stash@{0} = "ORQ13-SAFEGUARD-V31 20260727T174731Z"`, criado com `-u` para
   incluir os 4 nao rastreados. **Nunca foi dropado e continua la.**

Procedimento de aborto, **sem** perder trabalho:
```bash
cd /home/ec2-user/workspace/worktrees/gtl-i03-orq13-phase1
git branch orq13-attempt-1 HEAD                 # preserva os commits A e B por nome
git switch -C agent/opus48-b/orq-13-thinking-level orq13-safeguard-base
git stash apply stash@{0}                       # apply, NAO pop: o stash permanece
```
Nada aqui apaga commit ou arquivo. Os commits seguem alcancaveis pela branch de tentativa, pela tag e
pelo reflog. Para desfazer apenas o stage usei/uso `git restore --staged <path>`, nunca `reset --hard`.

## 3. O split ficou determinista **sem** cirurgia por hunk

Descoberta que simplifica a V3.1: **nao foi preciso** dividir `models.go` por hunk, nem `git add -p`
interativo, nem `filterdiff` (o `patchutils` esta ausente). Com a arvore no baseline limpo, rodar
`sqlc generate` sobre as queries **nao modificadas** reproduz exatamente o drift pre-existente:

```
models.go               5 hunks
task_message.sql.go     2 hunks
```

Isso e **exatamente** o conteudo previsto no manifesto para o commit A. O gerador e a fonte da
verdade, e o split passa a ser reproduzivel por qualquer pessoa com o mesmo `sqlc` v1.31.1, em vez de
depender de selecao manual de hunks.

## 4. Commits — locais, novos, **sem amend**

| commit | sha | parent | conteudo |
|---|---|---|---|
| **A** | `b1f08e3f0615f63d238a51a81a5b821917b433da` | `0cb8aeb` | `chore(sqlc): regenerate baseline output with sqlc v1.31.1` — 2 arquivos gerados, drift puro, inclui o revert `maxSeq` -> `max_seq` |
| **B** | `c0e93a2` | `b1f08e3` | `feat(cost): record thinking_level on task usage end to end` — 10 arquivos: 6 modificados + 2 migrations + 2 testes |

Arvore limpa depois de cada commit (`git status --porcelain` = 0 linhas). Nenhum `--amend`, nenhum
`push`, nenhuma alteracao de board.

## 5. Prova de determinismo — A + B reconstroem o original, por 3 vias

1. **Diff no escopo identico ao original** (os 7 arquivos que ja eram rastreados na base):
   `git diff 0cb8aeb HEAD -- <7 arquivos>` -> **`b87bc5cae3e877af9a26ebe39cce7f0a90fff2a126e075c7f5d12799d04707ce`**, byte a byte igual ao fingerprint registrado.
2. **Os 4 arquivos novos**, hash no `HEAD` vs hash da salvaguarda: 4/4 **OK**.
3. **Somatorio dos 11 arquivos**, `HEAD` vs salvaguarda:
   `018f1b8fd4615193a0b69cecd71425e6f5962d90061b625449726e41c3b7c27a` nos dois lados.

Nota metodologica, porque um leitor cuidadoso vai reparar: o `git diff 0cb8aeb HEAD` **sem escopo** da
`94905f9a12bd08e6…`, diferente do esperado. Isso **nao** e divergencia: o `git diff` original nao
continha os 4 arquivos nao rastreados, entao o unico escopo comparavel e o dos 7 rastreados. As vias
2 e 3 cobrem os 4 restantes. Registro o hash divergente aqui de proposito, para que ninguem o
encontre depois e conclua que houve deriva.

## 6. Evidencia de qualidade — e a pendencia que NAO reivindico

| verificacao | resultado |
|---|---|
| `gofmt -l` nos 7 arquivos Go | **vazio** = limpo |
| `go build ./...` | **OK** |
| `go vet ./internal/daemon ./internal/handler ./pkg/db/generated` | **OK** |
| `go test -race ./internal/daemon -run ThinkingLevel -count=1 -v` | **4 PASS nominais, 0 fail, 0 skip, 0 data race**, exit 0 |
| `go test ./internal/handler -run ThinkingLevel -count=1 -v` | **FALSO-VERDE — NAO aceito como evidencia** |

### 6.1 O falso-verde, medido e nomeado
```
exit=0
PASS nominais: 0 | FAIL: 0 | SKIP: 0
Skipping tests: database not reachable: ... 127.0.0.1:5432 ... connection refused
ok  github.com/multica-ai/multica/server/internal/handler  0.067s
```
O `ok` e mentiroso: `TestMain` (`internal/handler/handler_test.go:38-54`) chama `os.Exit(0)` quando o
banco esta inalcancavel, **antes** de `m.Run()`. Meus 4 testes de handler
(`TestThinkingLevelText`, `TestThinkingLevelTextNeverStoresEmptyString`,
`TestTaskUsagePayloadDecodesThinkingLevel`, `TestTaskUsagePayloadLegacyDaemonYieldsNull`) sao
**herméticos** — exercitam helpers puros e nao tocam banco — e mesmo assim foram engolidos.

**Portanto NAO afirmo** que o lado handler esta testado. Contagem nominal = 0. Fica pendente de um
banco efemero, e nao apliquei migration em banco nenhum, conforme a ordem.

### 6.2 Lint — nao bloqueia, por ordem explicita
`golangci-lint` esta ausente neste host, sem binario e sem fonte no module cache. Conforme a ordem
recebida, **nao bloqueio**: registro que o veredito de lint para o rename `max_seq` do commit A esta
**NAO MEDIDO** e fica diferido para o CI. Lembrete pertinente: os 4 workflows vivem em
`multica-auth-work/.github/workflows/` e nao na raiz do repositorio, portanto estao **inertes** hoje —
"diferir para o CI" so vale depois que existir CI ativo.

## 7. Veredito

**GO** para o split A/B. Os dois commits estao locais, determinismo provado por 3 vias, arvore limpa,
rollback seguro disponivel e sem nenhum passo destrutivo.

**Escopo do GO:** ele cobre a estrutura do split e a qualidade do lado daemon. **Nao** cobre a
evidencia nominal do pacote handler (6.1) nem o lint (6.2). Essas duas continuam abertas e explicitas.

Proximo passo que **nao** executo sem autorizacao especifica: aplicar a migration 127 num banco
efemero e re-rodar o pacote handler com `DATABASE_URL` para converter o falso-verde em PASS nominal.
O padrao ja validado no ORQ-41 serve: container efemero, imagem fixada por digest, ligado a loopback,
alcancado por tunel SSH privado, com teardown verificado.

## 8. Recomendacao transversal
O falso-verde de `TestMain` nao e problema do ORQ-13: atinge **todo** o pacote `internal/handler`. Ja
me custou uma retratacao anterior nesta mesma sessao. Merece cartao proprio — segregar os testes
herméticos ou fazer `TestMain` falhar em vez de sair com `0`. **Nao sou o registrador**, entao apenas
recomendo.

---

# ADENDO 1 — gate de banco efemero e lint fechados (2026-07-27T18:0xZ)

Autorizacoes usadas: gate handler com PostgreSQL efemero, e instalacao de ferramenta para fechar lint.
**As duas pendencias da secao 6 estao FECHADAS.** O falso-verde virou PASS nominal e o lint deixou de
ser "nao medido".

## A1. Infra efemera — padrao ORQ-41, duas rodadas

| item | rodada 1 | rodada 2 (pedido do revisor) |
|---|---|---|
| container | `orq13-gate-pg` `1edf9f971bf5` | `orq13-gate-pg2` `37e28e1f7456` |
| imagem | `pgvector/pgvector@sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0` (local, **sem pull**) | idem |
| binding | `{"5432/tcp":[{"HostIp":"127.0.0.1","HostPort":"47472"}]}` | porta `47473`, tambem **loopback-only** |
| acesso do ORQ2 | tunel SSH privado `127.0.0.1:60709` | `127.0.0.1:53555` |
| `DATABASE_URL` | arquivo modo **600**, senha aleatoria de 20 chars gerada no ORQ1 | idem |
| migrations | `up` ate **127**, so no DB descartavel | idem |

Esquema conferido no DB efemero: `thinking_level | text` nullable em `task_usage`, **68 tabelas**.
Nenhuma migration foi aplicada a banco de produto.

## A2. O falso-verde virou PASS nominal

```
go test -race ./internal/handler -run '^(TestThinkingLevelText|...|TestTaskUsagePayloadLegacyDaemonYieldsNull)$' -count=1 -v
exit=0   PASS nominais 4/4   FAIL 0   SKIP 0   "Skipping tests:" 0   DATA RACE 0
  --- PASS: TestThinkingLevelText                          (+5 subtestes PASS)
  --- PASS: TestThinkingLevelTextNeverStoresEmptyString
  --- PASS: TestTaskUsagePayloadDecodesThinkingLevel
  --- PASS: TestTaskUsagePayloadLegacyDaemonYieldsNull
```
E com o regex pedido pelo revisor, `-run 'ThinkingLevel|TaskUsagePayload'`: **8 PASS nominais**,
0 fail, 0 skip, **0 "Skipping tests:"** — os 4 do ORQ-13 mais 4 pre-existentes de `CreateAgent`/
`UpdateAgent` que o mesmo regex captura. Log `reviewer-regex.log`, sha256
`e191a36c67325138fc97ee3e265618eb18680984ca4601040c5a5ddd77da8dd1`.

### A2.1 Correcao ao revisor: dois dos quatro nomes exigidos nao existem
O revisor pediu `TestTaskUsageThinkingLevelNeverStoresEmptyString` e
`TestTaskUsageDecodesThinkingLevel`. **Nenhum dos dois existe no repositorio** —
`grep -rn "func <nome>" --include='*.go' .` devolve **0 ocorrencias** para ambos. Os nomes reais, em
`internal/handler/task_usage_thinking_level_test.go:12,45,59,79`, sao:
`TestThinkingLevelText`, `TestThinkingLevelTextNeverStoresEmptyString`,
`TestTaskUsagePayloadDecodesThinkingLevel`, `TestTaskUsagePayloadLegacyDaemonYieldsNull`.
Executei pelos nomes reais e os 4 passam. Registro a divergencia para que a proxima revisao nao
persiga funcao inexistente.

### A2.2 Por que minha primeira rodada mostrou so 3 dos 4
`-run ThinkingLevel` **nao seleciona** `TestTaskUsagePayloadLegacyDaemonYieldsNull`, porque o nome nao
contem a substring. Nao era falha: era filtro. O regex ancorado corrigiu, e e o motivo pelo qual a
precisao do revisor sobre o `-run` estava certa mesmo com os nomes errados.

## A3. Regressao de pacote — 3 FAIL, todos PRE-EXISTENTES

`go test -race ./internal/handler -count=1` -> `exit=1`, **3 FAIL nominais**, 0 "Skipping tests:",
**0 DATA RACE**:
```
--- FAIL: TestCreateChatSession_Routing
--- FAIL: TestCreateWorkspaceUsesRequestedSlug
--- FAIL: TestCreateWorkspace_DoesNotMarkOnboarded
    failed to create default squad: null value in column "leader_id" of relation "squad"
    violates not-null constraint (SQLSTATE 23502)
```
**Nao sao meus, e provei em vez de alegar.** Criei um worktree temporario no baseline `0cb8aeb`, um
banco separado `multica_base` no mesmo container descartavel, migrei com o codigo do baseline (parou
em **126**, como esperado) e rodei os mesmos 3 testes: **os 3 falham identicamente sem os meus
commits**. Log `baseline-3tests.log`, sha256
`29b841897763ed4af57cc1953bd85225d452e329218aef0a9ac95847271928eb`.

Causa raiz do trio, para quem for abrir cartao: `squad.leader_id UUID NOT NULL` vem da migration
**084** (`084_squad.up.sql:7`), e o handler cria o squad default sem leader. Nenhum dos 3 FAIL
menciona `thinking_level` ou `task_usage`.

### A3.1 CORRECAO — `leader_id` explica **2** dos 3 FAIL, nao os 3

A frase acima, como escrita, **estava errada por generalizacao** e a corrijo aqui sem alterar commit
algum. O `leader_id` explica **somente** `TestCreateWorkspaceUsesRequestedSlug` e
`TestCreateWorkspace_DoesNotMarkOnboarded`. O terceiro FAIL tem causa **independente**:

```
chat_test.go:461: CreateChatSession direct explicit: expected 201, got 400: {"error":"invalid workspace id"}
```
Nenhuma mencao a `leader_id`, `squad` ou constraint. Sintoma diferente (400, nao 500), ponto de falha
diferente, causa diferente. Eu agrupei os tres sob um rotulo comum porque falharam juntos e no mesmo
baseline, e isso foi conflacao minha.

### A3.2 RCA independente de `TestCreateChatSession_Routing`

**Causa: defeito de TESTE, nao de produto.** `CreateChatSession` (`internal/handler/chat.go:41`) le o
workspace do **contexto**, nao do header:
```go
workspaceID := ctxWorkspaceID(r.Context())          // chat.go:41
workspaceUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace id")   // chat.go:48
```
E `ctxWorkspaceID` -> `middleware.WorkspaceIDFromContext` (`internal/middleware/workspace.go:28-31`)
le exclusivamente `ctx.Value(ctxKeyWorkspaceID)`.

`TestCreateChatSession_Routing` chama `testHandler.CreateChatSession` **direto**, sem cadeia de
middleware, e injeta o workspace apenas como **header** `X-Workspace-ID`. Logo o contexto esta vazio,
`parseUUIDOrBadRequest` recebe string vazia e devolve **400 "invalid workspace id"**.

Prova por contraste, medida no arquivo: o helper `withChatTestWorkspaceCtx`
(`chat_test.go:23-33`) existe exatamente para isso e faz
`req.WithContext(middleware.SetMemberContext(...))`. Ele e usado **9 vezes** em `chat_test.go`, e
`TestCreateChatSession_Routing` o usa **0 vezes**:
```
ocorrencias de withChatTestWorkspaceCtx dentro de TestCreateChatSession_Routing: 0
ocorrencias no arquivo inteiro:                                                 9
```
O fixture de teste tampouco e vitima do `leader_id`: `setupHandlerTestFixture`
(`handler_test.go:105-110`) insere o workspace por **SQL direto**, sem passar pelo handler que cria o
squad default. Portanto `testWorkspaceID` e valido, e a hipotese de falha em cascata a partir do
`leader_id` esta **descartada**.

Observacao pertinente para a task do squad, e nao para esta: `CreateChatSession` tem ramo explicito
`"default squad has no leader yet"` (`chat.go:62`). O produto, portanto, **modela** squad sem leader,
enquanto o schema o **proibe** com `NOT NULL`. Essa contradicao e o material da proxima task; nao a
resolvo aqui.

### A3.3 Manifest de evidencia — comando, commit, migration maxima, hash de log

| # | comando | commit | migration maxima aplicada | log | sha256 |
|---|---|---|---|---|---|
| 1 | `go test -race ./internal/handler -count=1` | `c0e93a2` (A+B) | **127** (`up 127_task_usage_thinking_level`) | `regressao-pacote.log` | `8031bdd70ac45dd8eb31c9df32680cf1a640e0b13d855e510c1a140fe42a2c12` |
| 2 | `go test ./internal/handler -run '^(TestCreateChatSession_Routing\|TestCreateWorkspaceUsesRequestedSlug\|TestCreateWorkspace_DoesNotMarkOnboarded)$' -count=1 -v` | `0cb8aeb` (baseline, worktree temporario) | **126** (`up 126_runtime_profile_...`), banco `multica_base` | `baseline-3tests.log` | `29b841897763ed4af57cc1953bd85225d452e329218aef0a9ac95847271928eb` |
| 3 | `go test ./internal/handler -run 'ThinkingLevel\|TaskUsagePayload' -count=1 -v` | `c0e93a2` | **127** | `reviewer-regex.log` | `e191a36c67325138fc97ee3e265618eb18680984ca4601040c5a5ddd77da8dd1` |
| 4 | `golangci-lint v2.12.0 run --timeout 10m ./...` | `c0e93a2` | n/a | `head2.log` | `e5e3b46ba3d23a3400ba3a7c155d803598ad5dbe93df7a8d1a0751794fd122d8` |
| 5 | `golangci-lint v2.12.0 run --timeout 10m ./...` | `0cb8aeb` | n/a | `base2.log` | `e5e3b46ba3d23a3400ba3a7c155d803598ad5dbe93df7a8d1a0751794fd122d8` |

**Atualizacao 2026-07-27T18:3xZ — os 3 FAIL foram RESOLVIDOS fora deste card.** Em branch separada
`agent/opus48-b/squad-default-leader`, sobre a **mesma** base `0cb8aeb`, dois commits fecharam o trio:
`11ef715` (producao: `CreateWorkspace` deixa de criar squad sem leader) e `67e9a4c` (teste:
`TestCreateChatSession_Routing`). Regressao final do pacote: **exit 0, 0 FAIL, 0 "Skipping tests:",
0 DATA RACE**. Evidencia em `squad-default-leader-fix.md`, com o manifest complementar abaixo.

| # | comando | commit | migration maxima | log | sha256 |
|---|---|---|---|---|---|
| 6 | `go test -race ./internal/handler -run '^(TestCreateWorkspaceUsesRequestedSlug\|TestCreateWorkspace_DoesNotMarkOnboarded)$' -count=1 -v` | `11ef715` | **126** | `target-tests.log` | `965e85ab4aff5920179d7838730a97cab2948021f18c10f3b8818cb29ebbcebe` |
| 7 | `go test -race ./internal/handler -count=1` (3 FAIL -> 1) | `11ef715` | **126** | `regressao.log` | `a811a702b8ed74dc6025e0c738b7c4adda0517011eeb3e83ac46f927cd22655a` |
| 8 | `go test ./internal/handler -run '^TestCreateChatSession_Routing$' -count=1 -v` (prova de que **so** a correcao (a) nao basta) | `11ef715` + 1 linha nao commitada | **126** | `one-line-insufficient.log` | `1cee80c1d29f48e1e5bf6ae8c034f38e17633f77409af5e61b724e973df6fbda` |
| 9 | `go test -race ./internal/handler -count=1` (**pacote verde**) | `67e9a4c` | **126** | `pacote-final.log` | `6e1be823db943100c1f1524dccd0171e524a0b70dfadb0b7aacb99d70eb045b0` |

Itens 6-9 em `/home/ec2-user/.cache/squad-evidence/`. **Nada disso alterou `b1f08e3` nem `c0e93a2`**:
sao commits de outra branch, sobre a mesma base, e o worktree do ORQ-13 nao foi tocado.

Todos em `/home/ec2-user/.cache/orq13-evidence/`. Infra dos itens 1-3: container efemero
`orq13-gate-pg2` `37e28e1f7456`, imagem `pgvector/pgvector@sha256:d2ef61f4…3bad0`, loopback-only
`127.0.0.1:47473`, tunel SSH privado, ja **removido** (A5).

Duas honestidades sobre o manifest, para nao inflar o que tenho:
1. A "migration maxima" vem da **saida do `migrate up`** registrada durante a execucao, nao de uma
   consulta a `schema_migrations` feita depois — os bancos ja foram destruidos e nao posso reconsultar.
2. Os itens 1 e 2 usaram **bancos diferentes** no **mesmo** container: `multica` no 127 e
   `multica_base` no 126. Isso e deliberado, para nao contaminar o baseline com a coluna nova, e nao um
   descuido.

**Nenhum commit foi alterado por esta correcao.** `b1f08e3` e `c0e93a2` permanecem intactos; a
conflacao existia apenas no texto da evidencia e no check-out, e esta corrigida aqui e no check-out
novo.

## A4. Lint — fechado com v2.12.0 pinado, **zero issues novas**

Reproduzi o workflow: `.github/workflows/ci.yml:115-118` usa `golangci/golangci-lint-action@v9` com
`version: v2.12` e `working-directory: server`; **nao ha** `.golangci.yml` no repo, logo defaults.
Linters default ativos: `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`.

Procedencia verificada:
```
origem   https://github.com/golangci/golangci-lint/releases/download/v2.12.0
sha256 oficial   6b89d77b6396f81decee882f20473486bda5ef28b160f9a13b08f24848d9003c  golangci-lint-2.12.0-linux-amd64.tar.gz
verificacao      sha256sum -c  ->  OK
binario          golangci-lint 2.12.0, built with go1.26.2 from 7761527a em 2026-05-01T12:41:27Z
```
Instalado em diretorio **privado da task** (`0700`), **nunca** global.

Resultado, com cache isolado por execucao e caminhos normalizados:

| | issues | errcheck | ineffassign | staticcheck | unused |
|---|---|---|---|---|---|
| baseline `0cb8aeb` | **119** | 50 | 1 | 39 | 29 |
| HEAD (A+B) | **119** | 50 | 1 | 39 | 29 |

**Novas no HEAD: 0. Resolvidas: 0.** Os dois logs sao **byte a byte identicos** — mesmo sha256
`e5e3b46ba3d23a3400ba3a7c155d803598ad5dbe93df7a8d1a0751794fd122d8`.

**Veredito do `max_seq`:** o rename do commit A **nao gera nenhuma issue de lint**. Nao ha ocorrencia
em `pkg/db/generated/*` no relatorio. O risco que estava aberto desde a manha esta **fechado como
NEGATIVO**, medido, nao diferido.

Duas honestidades sobre este resultado:
1. `internal/daemon/daemon.go:3557 ineffectual assignment to startedTask (ineffassign)` aparece nos
   **dois** lados. Nao e meu: meus hunks em `daemon.go` estao em `@@ -3854` e `@@ -3866`, longe da
   3557. Confirmado pela igualdade dos relatorios.
2. `golangci-lint run` termina com **exit 1** por causa das 119 issues pre-existentes. Ou seja, se o
   job de lint fosse ligado hoje, ele **falharia por divida anterior**, nao por ORQ-13. Isso e um
   achado do repositorio e merece cartao proprio.

### A4.1 Erro meu no meio do caminho, corrigido
Na primeira comparacao eu reusei o **mesmo** `GOLANGCI_LINT_CACHE` nas duas execucoes. O cache
devolveu os caminhos da primeira rodada, e o log do baseline saiu referenciando arquivos do **meu**
worktree, produzindo um diff falso de ~85 "novas" e ~85 "resolvidas". Detectei pela incoerencia
(baseline nao podia citar `gtl-i03-orq13-phase1`), descartei o resultado e refiz com
`cache-head`/`cache-base` separados. Registro porque o numero errado ja tinha sido gerado, e um
revisor que veja apenas o primeiro log concluiria regressao inexistente.

## A5. Teardown — verificado

- container rodada 1 `orq13-gate-pg` **removido**; rodada 2 `orq13-gate-pg2` **removido**;
  `docker ps -a | grep -c orq13-gate` = **0**
- tuneis SSH encerrados; nenhum `ssh -L 127.0.0.1` residual meu
- arquivos de senha `/home/ec2-user/.orq13-gate-pw` e `.orq13-gate-pw2` **removidos**
- caches privados `orq13-gate`, `orq13-gate2` e `orq13-lint` (binario + 3 caches de lint) **removidos**
- worktree temporario do baseline **removido** via `git worktree remove`; restam **27** entradas
- logs de evidencia preservados em `/home/ec2-user/.cache/orq13-evidence/` (5 arquivos com sha256)
- **NAO** rodei `docker prune`: existem 6 volumes dangling no ORQ1, um deles possivelmente do meu
  container. Prune exige autorizacao e nao a tenho.

**Prova de produto intacto**, identica ao baseline pre-gate:
```
multica-dev-transition-backend-1   75e4416f06e9  Up 2 hours
multica-dev-transition-frontend-1  1b3b6c9ee32a  Up 15 hours
multica-dev-transition-postgres-1  2a4a84897363  Up 6 days (healthy)
omniroute                          2fb3fd57e885  Up 2 days (healthy)
disco ORQ1: 4.7G livres, 81%
```

Worktree do ORQ-13 **inalterado** pelo gate: `status` 0 linhas, `c0e93a2` sobre `b1f08e3`, stash de
salvaguarda ainda presente, tag intacta, e o fingerprint dos 7 rastreados segue `b87bc5ca…4707be`.

## A6. Veredito atualizado

**GO — sem pendencia de evidencia.** O escopo restrito da secao 7 caiu: handler com 4/4 PASS nominais
e zero skip; lint medido com zero issues novas. Continua valendo que **nao** autorizo merge e que
preciso de revisor independente.

## 9. Nao-afirmacoes
**Atualizadas pelo Adendo 1.** Duas nao-afirmacoes da versao original **caducaram** e estao marcadas.

- **NENHUMA migration foi aplicada a banco de PRODUTO.** No Adendo 1 apliquei migrations ate 127
  **somente** em bancos descartaveis, dentro de container efemero removido depois. O banco de produto
  `multica-dev-transition-postgres-1` nao foi tocado e segue `Up 6 days (healthy)`.
- Nenhum `push`, `merge`, PR, mutacao de board, comentario ou atribuicao.
- Nenhum `--amend`: A e B seguem sendo os mesmos commits, `b1f08e3` e `c0e93a2`. O gate e o lint **nao**
  alteraram nem um byte do worktree.
- **NENHUM `git reset --hard`** em nenhum momento.
- Nao dropei o stash de salvaguarda nem removi a tag; ambos seguem disponiveis.
- ~~Nao reivindico teste do pacote `internal/handler`~~ — **CADUCOU**: agora reivindico, com 4/4 PASS
  nominais, 0 skip, 0 "Skipping tests:", 0 data race (A2). A regressao de pacote tem 3 FAIL **provados
  pre-existentes** contra o baseline (A3).
- ~~Nao reivindico veredito de lint~~ — **CADUCOU**: lint medido com v2.12.0 pinado e checksum oficial
  conferido; **zero issues novas**, relatorios byte a byte identicos (A4). Nao diferi para CI.
- Nao removi os caches antigos que o owner mandou preservar (`go-build-i03`, `go-build-orq41`,
  `sqlcbuild`). Removi **apenas** o que eu criei nesta rodada: `orq13-gate`, `orq13-gate2`,
  `orq13-lint` e o worktree temporario do baseline.
- **Nao rodei `docker prune`.** Ha 6 volumes dangling no ORQ1, um possivelmente do meu container.
  Divulgo em vez de agir: prune exige autorizacao.
- Nao toquei `gtl-orq41-w4-autopilot` nem qualquer outro worktree de terceiro. O worktree temporario do
  baseline foi criado por mim, em diretorio privado, e removido.
- Nao li nem usei segredo, token ou credencial de produto. A senha do banco descartavel foi gerada
  aleatoriamente no ORQ1, gravada em arquivo `600`, nunca passou por argv e foi removida no teardown.
- Nao instalei nada global: o `golangci-lint` ficou em diretorio privado `0700` da task e foi removido.
