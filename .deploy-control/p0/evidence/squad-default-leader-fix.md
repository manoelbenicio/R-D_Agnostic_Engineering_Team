# Fix — squad default sem leader em `CreateWorkspace`

- **Executor:** Opus48#B (ORQ2, w6:p2), owner exclusivo do worktree e do arquivo
- **Gate 0:** delta enviado com scan de ownership e **aceito**; ownership de `workspace.go` e testes confirmado pelo General-TL
- **Worktree dedicado:** `/home/ec2-user/workspace/worktrees/gtl-squad-default-leader`, branch `agent/opus48-b/squad-default-leader`, base **`0cb8aeb`**
- **Commit local:** `11ef7156ad77e838132d4d8e3ca86ffa1a70ba1a`, parent `0cb8aeb`, sem amend, sem push
- **Restricoes respeitadas:** nenhuma mudanca de schema, nenhuma migration, nenhuma reserva de numero, nenhum `sqlc`, nada tocado no worktree do ORQ-13

## 1. Diagnostico do contrato

Tres fontes, em conflito direto:

| fonte | o que afirma |
|---|---|
| `workspace.go:219` (comentario original) | "Leader is deferred until the first eligible agent is created" — squad **sem** leader e esperado |
| `migrations/084_squad.up.sql:7` | `leader_id UUID NOT NULL REFERENCES agent(id) ON DELETE RESTRICT` — squad sem leader e **proibido** |
| `squad.go:217-239` (CreateSquad do usuario) | `leader_id` **obrigatorio** (400 "leader_id is required") e validado como agente **daquele** workspace |

O codigo pretendia leader nulo; o schema o proibe. A query `CreateSquad`
(`pkg/db/queries/squad.sql:1-4`) insere `leader_id` como `$4`, e `workspace.go:220-225` montava
`CreateSquadParams` **sem** `LeaderID`, mandando o zero value de `pgtype.UUID`, que vira **NULL**.

### 1.1 O achado que excede os testes
`CreateWorkspace` **nunca podia funcionar**. Nao ha ramo, flag ou condicao: toda chamada caia em
```
500 failed to create default squad: ERROR: null value in column "leader_id"
    of relation "squad" violates not-null constraint (SQLSTATE 23502)
```
Isto **nao** e ruido de teste — e o endpoint de criacao de workspace inoperante. Os 2 testes eram
apenas quem denunciava.

### 1.2 Contrato escolhido, e por que
Sem mexer no schema, restam duas opcoes reais:

| opcao | avaliacao |
|---|---|
| criar o squad **com** leader | **impossivel** no ponto: nenhum agente existe no workspace recem-criado, e o leader precisa ser agente daquele workspace |
| **deferir** a criacao do squad | viavel, e ja **modelado pelos consumidores** |

Escolhi deferir. Nao e suposicao minha: os consumidores ja tratam a ausencia.
`CreateChatSession` responde `"no default squad found for routing"` (`chat.go:56-60`) e
`"default squad has no leader yet"` (`chat.go:62`). O segundo ramo, inclusive, era **inalcancavel**
enquanto o schema proibia leader nulo — evidencia de que o produto sempre modelou squad sem leader.

A terceira opcao, tornar `leader_id` nullable, e mudanca de schema e exigiria migration nova com
reserva de numero. **Fora desta task por ordem explicita**, e eu nao sou o registrador. Registro como
alternativa de desenho, nao como recomendacao fechada.

## 2. Fix minimo

`internal/handler/workspace.go`, **1 arquivo, +15/-24**. Removi a criacao do squad default e a adicao
do owner a ele, substituindo por um comentario que documenta a constraint, o motivo e onde a ausencia
e tratada. Nenhuma outra funcao alterada; a variavel `squad` era usada **so** pelo `AddSquadMember`
imediatamente seguinte, entao a remocao e autocontida.

## 3. Evidencia — DB efemero, contagem nominal

Infra, padrao ORQ-41: container `squad-gate-pg` `bdfc589e64ee`, imagem
`pgvector/pgvector@sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0` (local,
sem pull), **loopback-only** `{"5432/tcp":[{"HostIp":"127.0.0.1","HostPort":"47474"}]}`, tunel SSH
privado `127.0.0.1:34515`, senha aleatoria gerada no ORQ1 em arquivo `600`, nunca em argv. Migrations
`up` ate **126**, coerente com a base `0cb8aeb`, **so** no banco descartavel.

| verificacao | resultado |
|---|---|
| `gofmt -l internal/handler/workspace.go` | vazio = limpo |
| `go build ./...` | **OK** |
| `go vet ./internal/handler` | **OK** |
| `go test -race ./internal/handler -run '^(TestCreateWorkspaceUsesRequestedSlug\|TestCreateWorkspace_DoesNotMarkOnboarded)$' -count=1 -v` | **2 PASS nominais**, 0 fail, 0 skip, **0 "Skipping tests:"**, **0 DATA RACE** |
| `go test -race ./internal/handler -count=1` | FAIL de pacote **3 -> 1**, 0 "Skipping tests:", 0 DATA RACE |

Logs em `/home/ec2-user/.cache/squad-evidence/`:
`target-tests.log` sha256 `965e85ab4aff5920179d7838730a97cab2948021f18c10f3b8818cb29ebbcebe`;
`regressao.log` sha256 `a811a702b8ed74dc6025e0c738b7c4adda0517011eeb3e83ac46f927cd22655a`.

Lint: **nao executado**, e declaro por que. O diff nao introduz codigo novo, so remove um bloco e
adiciona comentario, e o veredito de lint para este pacote ja foi medido na task anterior com 119
issues pre-existentes e zero novas. Se o revisor quiser o numero para este commit, rebaixo o
v2.12.0 com checksum conferido e reporto.

## 4. ~~A falha que **permanece**, e nao e minha para corrigir aqui~~ — **SUPERSEDED por `67e9a4c`**

> **SUPERSEDED em 2026-07-27T18:3xZ.** Esta secao esta **historicamente correta mas obsoleta**, e o
> fix de UMA linha que ela propunha foi medido como **INSUFICIENTE**. Mantida apenas para rastreio.
>
> Duas retificacoes:
> 1. **A falha nao permanece.** `TestCreateChatSession_Routing` foi corrigido em `67e9a4c`
>    (Adendo 1), sob autorizacao posterior, e o pacote esta **verde**.
> 2. **O fix de uma linha nao bastava.** Eu escrevi abaixo "fix de uma linha, pronto mas nao
>    aplicado". Ao aplica-lo isoladamente, o teste continuou falhando, agora em `chat_test.go:485`
>    com `null value in column "leader_id" ... violates not-null constraint`: a linha resolvia o 400 e
>    **expunha** um segundo defeito atras dele. Foram necessarias **tres** correcoes, (a) (b) (c) do
>    Adendo 1. Prova em `one-line-insufficient.log` sha256
>    `1cee80c1d29f48e1e5bf6ae8c034f38e17633f77409af5e61b724e973df6fbda`.
>
> O diagnostico de causa desta secao (contexto vs header) segue **valido** e virou a correcao (a).

`TestCreateChatSession_Routing` continua falhando, com causa **independente** ja documentada em
`orq13-phase1-split-execution.md` A3.2: **defeito de teste**. `CreateChatSession` le o workspace do
**contexto** (`chat.go:41` -> `middleware.WorkspaceIDFromContext`, `workspace.go:28-31`), e o teste
chama o handler direto passando o workspace apenas como **header**, resultando em
`400 "invalid workspace id"`. O proprio arquivo tem o helper `withChatTestWorkspaceCtx`
(`chat_test.go:23-33`), usado por **9** testes e por este **0** vezes.

Fix de uma linha, **pronto mas nao aplicado** por estar fora do escopo autorizado:
```go
directReq = withChatTestWorkspaceCtx(t, directReq)   // antes de CreateChatSession
```
Recomendo cartao proprio. **Nao apliquei** para nao expandir escopo sem autorizacao.

## 5. Teardown verificado
- container `squad-gate-pg` removido; `docker ps -a | grep -c squad-gate` = **0**
- tunel encerrado; nenhum `ssh -L 127.0.0.1` residual meu
- senha `/home/ec2-user/.squad-gate-pw` removida; cache `squad-gate` removido
- **NAO** rodei `docker prune`; volumes dangling seguem intocados por falta de autorizacao
- **Produto intacto:** backend `Up 2 hours`, frontend `Up 15 hours`, postgres `Up 6 days (healthy)`,
  omniroute `Up 3 days (healthy)`; disco ORQ1 4.5G livres

## 6. Revisao
Preciso de revisor independente. **Nao pode ser eu.** O ponto que quero atacado primeiro: se
**deferir** a criacao do squad e aceitavel para o frontend, ou se algum consumidor de UI assume
"Workspace Team" existente logo apos criar o workspace. Eu verifiquei os consumidores em Go; **nao**
varri o frontend Next.js, e essa e a lacuna consciente desta entrega.

---

# ADENDO 1 — commit 2: `TestCreateChatSession_Routing` corrigido, pacote **VERDE**

- **Commit:** `67e9a4c7b643c729603bcb6498c57ed0489d20b7`, parent **`11ef715`**, sem amend, sem push
- **Escopo:** **1 arquivo, so teste** — `internal/handler/chat_test.go`, +4/-8. Zero producao.

## A1.1 As tres correcoes, exatamente as do peer review

| # | correcao | por que |
|---|---|---|
| (a) | `directReq = withChatTestWorkspaceCtx(t, directReq)` | `CreateChatSession` le o workspace do **contexto** (`chat.go:41` -> `middleware.WorkspaceIDFromContext`), nao do header; sem isso, 400 `invalid workspace id` |
| (b) | `CreateSquad` recebe `LeaderID: util.MustParseUUID(squadTLID)`, e o `UpdateSquad` redundante e **removido** | o teste criava squad sem leader e violava o `NOT NULL` de `084_squad.up.sql:7` — **mesma classe** do bug de producao do commit pai. Com o leader na criacao, o `UpdateSquad` seguinte perde funcao |
| (c) | `defaultReq = withChatTestWorkspaceCtx(t, defaultReq)` | mesma omissao de (a), na segunda requisicao do teste |

Consequencia mecanica de (b), que registro para o diff nao surpreender: com o `UpdateSquad` fora, a
variavel `squad` deixou de ser usada e o `go vet` acusou `declared and not used`. Troquei por
`_, err := ...`. Nao e uma quarta correcao, e o efeito direto de remover o bloco.

## A1.2 Descoberta: a correcao (a) **sozinha** nao bastaria

Apliquei primeiro **apenas** a linha do `directReq` e rodei. Resultado, preservado em
`one-line-insufficient.log` (sha256 `1cee80c1d29f48e1e5bf6ae8c034f38e17633f77409af5e61b724e973df6fbda`):
```
exit=1
chat_test.go:485: failed to create default test squad: ERROR: null value in column
"leader_id" of relation "squad" violates not-null constraint (SQLSTATE 23502)
```
A linha resolveu o 400 e **expos** o defeito seguinte, que estava escondido atras dele. Os tres
problemas estavam **empilhados**, nao paralelos, e por isso nenhum fix isolado poderia zerar o teste.
Guardo esse log porque ele e a prova de que as tres correcoes sao necessarias, nao um pacote inflado.

## A1.3 Evidencia — pacote **inteiro** verde

Mesma infra efemera, segundo container: `squad-gate-pg2` `8dd93b786356`, mesmo digest
`sha256:d2ef61f4…3bad0`, **loopback-only** `127.0.0.1:47475`, tunel SSH privado, senha aleatoria em
arquivo `600`, migrations `up` ate **126**.

| verificacao | resultado |
|---|---|
| `gofmt -l internal/handler/chat_test.go` | vazio = limpo |
| `go vet ./internal/handler` | **OK** |
| `go test -race ./internal/handler -run '^TestCreateChatSession_Routing$' -count=1 -v` | **1 PASS nominal**, 0 fail, **0 "Skipping tests:"**, **0 DATA RACE** |
| `go test -race ./internal/handler -count=1` | **exit 0**, **0 FAIL**, **0 "Skipping tests:"**, **0 DATA RACE**, `ok … 20.456s` |

As **3** falhas originais, contadas uma a uma no log final: `TestCreateChatSession_Routing` **0**,
`TestCreateWorkspaceUsesRequestedSlug` **0**, `TestCreateWorkspace_DoesNotMarkOnboarded` **0**.

Logs em `/home/ec2-user/.cache/squad-evidence/`:
`chat-final.log` `5c3d962a971ce15a377bb14507973c6a30a3deb8862cb9e1f959f06adf044524`;
`pacote-final.log` `6e1be823db943100c1f1524dccd0171e524a0b70dfadb0b7aacb99d70eb045b0`;
`one-line-insufficient.log` `1cee80c1d29f48e1e5bf6ae8c034f38e17633f77409af5e61b724e973df6fbda`.

## A1.4 Teardown
Container `squad-gate-pg2` removido, `docker ps -a | grep -c squad-gate` = **0**; tunel encerrado sem
`ssh -L` residual; senha e cache `squad-gate2` removidos; `docker prune` **nao** executado. Produto
intacto: backend `Up 2 hours`, frontend `Up 15 hours`, postgres `Up 6 days (healthy)`, omniroute
`Up 3 days (healthy)`; ORQ1 com 4.5G livres.

## A1.5 Lint
**Nao executado**, por instrucao explicita de nao repetir salvo exigencia do revisor. Se exigir,
rebaixo o v2.12.0 com checksum conferido em diretorio privado e reporto.

## A1.6 Re-review dos DOIS commits
Peco revisao de `11ef715` **e** `67e9a4c`, por alguem que **nao** seja eu. A lacuna consciente segue a
mesma da secao 6: verifiquei os consumidores em **Go**; **nao** varri o frontend Next.js para
confirmar que nada depende do squad default existir imediatamente apos criar o workspace.

## 7. Nao-afirmacoes
- **Nenhuma mudanca de schema, nenhuma migration, nenhuma reserva de numero, nenhum `sqlc`.**
- Nenhum `push`, `merge`, PR, board, comentario, atribuicao. Nenhum `--amend`. Nenhum `git reset --hard`.
- **Nada tocado** no worktree do ORQ-13, em suas migrations ou em seu output de sqlc; sem handoff, sem necessidade.
- Nenhuma migration em banco de produto: apenas o descartavel, removido depois. `multica-dev-transition-postgres-1` segue `Up 6 days (healthy)`.
- **Nao varri o frontend** para confirmar que ninguem depende do squad default imediato (secao 6).
- Nao executei lint neste commit (secao 3), e nao afirmo veredito de lint para ele.
- ~~**Nao corrigi** `TestCreateChatSession_Routing`: fix identificado, deliberadamente nao aplicado (secao 4).~~
  **SUPERSEDED por `67e9a4c`**: corrigido sob autorizacao posterior, com **tres** correcoes, nao uma.
  Ver Adendo 1 e a nota de supersessao na secao 4.
- Nao removi caches de terceiros nem os que o owner mandou preservar; removi so o que criei nesta task.
- Nao li nem usei segredo, token ou credencial de produto.

---

# ADENDO 2 — corrida de evidencia em `-json` (2026-07-27T18:4xZ)

Pedida pela revisao final, que deu **CODE PASS** em `11ef715`+`67e9a4c` e **BLOCK apenas de
evidencia**. Nenhum codigo novo, nenhum amend, nenhum lint.

## A2.1 Manifest da corrida

| campo | valor |
|---|---|
| comando | `go test -race -count=1 -json ./internal/handler` |
| commit sob teste | `67e9a4c7b643c729603bcb6498c57ed0489d20b7`, arvore limpa (`git status --porcelain` = 0) |
| container | `squad-gate-pg3`, id `dfd40bd943d201e40d89ac2c3dcc1397a8a01efe3b81a1647f74ef6a0211ae2e` |
| imagem | `pgvector/pgvector@sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0` (local, sem pull) |
| binding | `{"5432/tcp":[{"HostIp":"127.0.0.1","HostPort":"47476"}]}` — **loopback-only** |
| tunel | SSH privado `127.0.0.1:49585` -> ORQ1 `127.0.0.1:47476` |
| `DATABASE_URL` | arquivo modo **600**, senha aleatoria gerada no ORQ1, nunca em argv |
| servidor | PostgreSQL **17.10** (Debian 17.10-1.pgdg12+1) |
| migration maxima | **`126_runtime_profile_protocol_family_native_runtimes`**, lida de `schema_migrations` |
| idempotencia | segundo `migrate up` -> `Done.` sem novas aplicacoes |
| schema conferido | `squad.leader_id` `is_nullable = NO`; **68** tabelas |
| log | `/home/ec2-user/.cache/squad-evidence/package-json.log` |
| sha256 do log | `9027b7896a5015c01460628133c356ebf5a184ccf59966eefe365988f86d974d` |

## A2.2 Resultado

```
exit = 0
veredito de pacote (evento sem campo Test): github.com/multica-ai/multica/server/internal/handler = "pass"
Action counts: cont 7 · output 2902 · pass 1338 · pause 7 · run 1374 · skip 37 · start 1
fail = 0        DATA RACE = 0
```
**PACKAGE PASS confirmado**, `fail = 0`, `DATA RACE = 0`, **1338** eventos `pass`.

## A2.3 `skip == 0` NAO foi atingido, e o criterio e inalcancavel neste pacote

`skip = 37`, todos **top-level**, **zero** subtestes. Nao vou apresentar isso como aprovado. Taxonomia
completa, medida no log:

| grupo | qtd | guard | pode chegar a zero? |
|---|---|---|---|
| `TestRedis*` (liveness, local skills, model list, update store, webhook rate limiter) + `TestRequireDaemonWorkspaceAccess_*` + `TestMembershipCache_*` | **35** | `newRedisTestClient` -> `if os.Getenv("REDIS_TEST_URL") == "" { t.Skip }` e `t.Skipf` se o `Ping` falhar (`runtime_local_skills_redis_store_test.go:21-36`) | **sim, com Redis efemero** + `REDIS_TEST_URL` |
| `TestFetchFromSkillsSh_AnthropicPptxIntegration` | 1 | `if os.Getenv("MULTICA_RUN_SKILLS_SH_INTEGRATION") == "" { t.Skip }` (`skill_test.go:525`) | so com **rede viva ao GitHub**; e teste de integracao externa |
| `TestWebhookHandler_DBErrorOnTokenLookupReturns500` | 1 | **`t.Skip` incondicional**, sem env algum: e um **marcador de regressao** deliberado (`autopilot_webhook_handler_test.go:807-819`, comentario explica que o ramo 500 nao e alcancavel sem quebrar a conexao) | **NAO. Nunca.** |

Conclusao objetiva: **`skip == 0` e estruturalmente impossivel** para `./internal/handler`, porque um
dos testes esta marcado para pular **incondicionalmente**, por design documentado no proprio arquivo.
Nenhuma infraestrutura muda isso.

Alem disso, a imagem Redis **nao existe** no ORQ1 (`docker images | grep -iE '^redis|valkey'` ->
nenhuma), logo levar os 35 a zero exigiria **pull de imagem nova**, que e infra alem da autorizacao
atual (que cobre PostgreSQL efemero).

### A2.3.1 Distincao que evita leitura errada das minhas evidencias anteriores
Meus relatorios anteriores diziam `"Skipping tests:" = 0`. Isso **continua verdadeiro** e **nao**
contradiz `skip = 37`. Sao metricas diferentes:
- `"Skipping tests:"` e a string que o **`TestMain`** imprime quando aborta o **pacote inteiro** por
  banco inalcancavel — o falso-verde do ORQ-13. Zero significa que o pacote **de fato rodou**.
- `Action == "skip"` conta testes **individuais** que chamaram `t.Skip`, por dependencia externa
  ausente. Sao 37, e nenhum deles toca o codigo de `11ef715` ou `67e9a4c`.

Os **3** testes que motivaram esta task aparecem como `pass`, nao como `skip`.

### A2.3.2 O que proponho, sem decidir
Tres opcoes, e a escolha nao e minha:
1. **Aceitar** `skip = 37` com esta taxonomia, exigindo `package pass`, `fail = 0`, `race = 0` e
   **zero skip entre os testes afetados** — que e o que esta provado.
2. **Autorizar Redis efemero** (imagem nova, pull no ORQ1) para derrubar 35 dos 37. Restariam **2**, e
   um deles e permanentemente inalcancavel.
3. **Cartao proprio** para reavaliar o `t.Skip` incondicional do webhook, que hoje impede qualquer
   metrica de `skip` zerada neste pacote.

## A2.4 Teardown
`squad-gate-pg3` removido, `docker ps -a | grep -c squad-gate` = **0**; tunel encerrado, nenhum
`ssh -L 127.0.0.1` residual; senha `/home/ec2-user/.squad-gate-pw3` e cache `squad-json` removidos;
`docker prune` **nao** executado. Produto intacto: backend `Up 3 hours`, frontend `Up 15 hours`,
postgres `Up 6 days (healthy)`, omniroute `Up 3 days (healthy)`; ORQ1 com 4.4G livres.

## A2.5 Re-review final
Peco a revisao final de `11ef715` + `67e9a4c` **e** deste Adendo 2, por alguem que **nao** seja eu.
Duas questoes abertas, em ordem:
1. **`skip == 0` e inalcancavel** — qual das tres opcoes de A2.3.2 vale como criterio de aceite?
2. A lacuna que declaro desde o inicio: verifiquei os consumidores em **Go**, **nao** varri o frontend
   Next.js para confirmar que nada depende do squad default existir imediatamente apos criar o
   workspace. `11ef715` muda comportamento de produto.
