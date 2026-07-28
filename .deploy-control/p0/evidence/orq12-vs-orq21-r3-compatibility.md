# ORQ-12 x ORQ-21 R3 (`ee87f7b` -> `ec8f945`): compatibilidade verificada, **1 divergencia corrigida, 3 abertas**

- autor: **Opus48#A** - ORQ2 w6:p1 - 2026-07-28T16:55Z
- meu HEAD: **`d33903f`** (novo commit sobre `d9f0ae2`), worktree limpo
- **Nao toquei o worktree do ORQ-21.** Leitura dos commits dele foi via `git show`, read-only.
- Sem push, PR, merge, quadro, AWS, segredo, prod.

## 1. Sem colisao de arquivos - merge deve ser limpo

`ee87f7b` (10 arquivos) e `ec8f945` (2 arquivos) **nao tocam nenhum** dos meus:
`queries/agent.sql`, `queries/task_usage.sql`, `migrations/staging/*`, `sqlc*.yaml`, `dashboard.go`,
`router.go`, `task_usage_account_test.go`. Verificado por `git show --name-only` nos dois.

Observacao estrutural: o pacote `internal/credentialregistry` **nao existe na minha linhagem** (base
`c0e93a2`); ele chega com o R3. Ou seja, apos o merge passam a existir **dois** implementadores da mesma
politica - o resolver Go dele e o meu SQL de claim.

## 2. 🔴 Divergencia **corrigida** em `d33903f`: canonicalizacao assimetrica

`credentialregistry.CanonicalProvider` (`resolver.go:53-60`) e aplicado aos **dois** lados no R3:
```go
if CanonicalProvider(assignment.Vendor) != CanonicalProvider(provider) { ... }
```
O meu SQL canonicalizava **so o lado do runtime**. Consequencia: uma conta com `vendor = 'agy'` seria
**aceita pelo ORQ-21 e recusada por mim** - a task executaria e o gasto ficaria **nao atribuivel**, que e
exatamente a falha que a coluna existe para eliminar. Corrigido: `CASE` nos **dois** lados, com `btrim`
para que padding armazenado nao mude o veredito, mais o **teste espelho** (runtime `antigravity` contra
vendor `" AGY "`).

## 3. Onde as duas implementacoes **concordam** (verificado linha a linha)

| politica | ORQ-21 R3 | meu SQL |
|---|---|---|
| tenant | `ag.workspace_id = a.tenant_id` **e** `aa.tenant_id = a.tenant_id` | `acc.tenant_id = ag.workspace_id` **e** `ap.tenant_id = ag.workspace_id` - equivalente |
| approved | `aa.allowed = true` | `ap.allowed IS TRUE` |
| worktype | Go: `WorktypeScope == nil \|\| != "GENERAL"` -> recusa (`resolver.go:143-146`) | SQL: `ap.worktype_scope = 'GENERAL'`, que **tambem** exclui `NULL` |
| status | Go: `!= available && != leased` -> `ErrAccountUnavailable` | SQL: `IN ('available','leased')` |
| exclusividade | Go: `EXISTS(... other.agent_id <> ass.agent_id)` -> `ErrAccountAlreadyUsed` | indice **unico** staged em `assignments(account_id)` - defesa em profundidade |

## 4. Divergencias que **permanecem abertas** - precisam de ruling, nao de codigo meu

### 4.1 🟡 Escopo de aplicabilidade
`RequiresApprovedAssignment` (`resolver.go:64-71`) limita o contrato a **`antigravity`, `codex`, `kiro`**.
O meu freeze se aplica a **todo** provider. Para um runtime fora da lista (`claude`, `gemini`, ...) o
R3 nao exige assignment, mas eu **congelaria** a conta se por acaso existir assignment aprovada.
Pergunta: o snapshot deve valer para **todos** os providers, ou so para os credential-bearing? Eu
**nao** decidi isso sozinho.

### 4.2 🟡 Duas canonicalizacoes, um dono
Depois do merge, `CanonicalProvider` (Go) e o meu `CASE` (SQL) implementam o mesmo mapeamento. Hoje
ambos tratam so `agy -> antigravity`; um alias novo em Go **nao** chega ao SQL, e o efeito seria o claim
parar de congelar **em silencio**. Duas saidas, e continuo preferindo a segunda: **(a)** teste de
paridade que percorra os aliases comparando SQL e `CanonicalProvider`; **(b)** normalizar **na escrita**,
gravando `agent_runtime.provider` e `accounts.vendor` ja canonicos, de modo que o SQL compare igualdade
simples. `agent_runtime.provider` e `TEXT` **sem `CHECK`** (`004:7`), o que hoje permite qualquer string.

### 4.3 🟡 Caminho de **reclaim** nao congela
`ClaimAgentTask` esta no caminho real - `internal/service/task.go:1057`, dentro do fluxo de
`ClaimTaskForRuntime`. Mas `ReclaimStaleDispatchedTaskForRuntime` (`task.go:1125`) devolve uma task **ja
`dispatched`** **sem** passar por ele. Para uma task congelada no claim original isso e inofensivo (o meu
`COALESCE` preserva), mas uma task que chegue a `dispatched` por qualquer outra rota ficaria **NULL para
sempre**. A ADR pede "reclaim preservation" no gate; sugiro provar tambem "reclaim **nao** cria
atribuicao nova".

### 4.4 nota menor
O R3 recusa metadado invalido (`ErrInvalidMetadata`, `HomeDir`/`ConfigDir`). O meu SQL nao olha isso -
uma task pode ficar congelada e **nao** executar. Inofensivo, porque sem execucao nao ha linha de usage;
registro so para nao parecer omissao.

## 5. Verificacao desta rodada

```
sqlc generate (config staged)  -> OK, schema-validado
go vet ./internal/handler ./pkg/db/...  -> OK
go build ./...                 -> OK
go test -c -o /dev/null ./internal/handler -> COMPILA
gofmt -l  meus arquivos        -> vazio
17 funcoes de teste no arquivo
```
**O commit foi gateado**: `go vet` e `go test -c` rodaram **antes** e o commit so aconteceu porque os
dois passaram - a correcao de processo do lapso que gerou `41ae01b`.

## 6. ⚠️ Pressao de disco no ORQ2 - **97%**, e o build falhou por isso

O `go test -c` falhou com `write $WORK/...: no space left on device`. Medido: `/` com **60G**, **58G
usados**, **2.4G livres (97%)**; `/home/ec2-user/.cache` com **7.2G** e `workspace/worktrees` com
**5.6G** - cada worktree e um checkout completo de ~5000 arquivos.

Acao que tomei, restrita ao que e **meu e descartavel**: removi `go-build-det-c`, `go-build-det-d`,
`go-build-review-b` e os quatro diretorios `orq42-bin*` (os hashes de determinismo ja estao registrados
em evidencia e sao reproduziveis). Liberou ~300 MB, e o build passou. **Nao** toquei cache de nenhum
outro agente, nem worktree alheio.

Isso e um risco **compartilhado**: com 2.7G livres, qualquer gate com DB efemero, imagem Docker ou
`-race` pode falhar por disco e **parecer** falha de teste. Recomendo que o General-TL trate a limpeza
como item proprio antes do gate combinado.

## 7. Nao-afirmacoes

- **Os testes com banco continuam sem execucao** - sem Postgres aqui. Afirmo **compilacao**, nao
  aprovacao, para os 17 testes.
- **Nao li o R3 inteiro**: examinei `resolver.go`, o trecho de `handler/daemon.go` e as estatisticas dos
  dois commits. Nao revisei os testes dele nem o `seed_approved_assignment.sql`.
- **Nao fiz rebase nem merge** com o R3; ele esta em re-review e o sequenciamento e do owner.
- A afirmacao "merge deve ser limpo" e sobre **nomes de arquivo**, nao semantica: as tres divergencias da
  secao 4 sao exatamente o que um merge textualmente limpo **nao** resolve.
- Nao apliquei a migration; numero segue **nao materializado**.
