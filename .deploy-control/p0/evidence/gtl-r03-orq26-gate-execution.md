# GTL-R03 — Execução de gates ORQ-26 (worktree gtl-orq26)

- Executor: Kiro-Opus5 (sem poder de decisão, AA-001 §0.0)
- Worktree: `/home/ec2-user/workspace/worktrees/gtl-orq26`
- Branch: `agent/kiro-opus5/orq-26-contract-fix`
- Timestamp: 2026-07-27T11:54Z
- Sem commit, push, deploy, restart, live ou board. Arquivos de produção alterados:
  somente os dois locked.

## Veredito: BLOCK — não existe Postgres de teste neste host

Gates estáticos passam. Os testes S1-S7 **não foram executados** e não posso executá-los
aqui. Não fabrico PASS.

## 1. Correção do relatório anterior: existe toolchain Go

Meus relatórios anteriores afirmaram "sem toolchain Go neste host". **Isso estava errado.**
O toolchain existe fora do PATH:

```text
/home/ec2-user/goroot/go/bin/go   → go version go1.26.1 linux/amd64
/home/ec2-user/goroot/go/bin/gofmt
```

Minha busca anterior cobriu apenas PATH, `/usr/local/go` e `/opt/go`. Todos os gates
abaixo foram executados com esse toolchain.

## 2. Diretórios privados de gate (addendum GTL-R03)

Criados dentro do worktree, modo 0700, sem tocar `/tmp` nem cache de terceiros:

```text
/home/ec2-user/workspace/worktrees/gtl-orq26/.gtl-orq26-gate            drwx------
/home/ec2-user/workspace/worktrees/gtl-orq26/.gtl-orq26-gate/tmp        drwx------
/home/ec2-user/workspace/worktrees/gtl-orq26/.gtl-orq26-gate/gocache    drwx------
```

Exportados como `TMPDIR`, `GOTMPDIR` e `GOCACHE`. Tamanho atual 394 MB (cache de build
próprio, isolado). Motivo: `/tmp` é tmpfs de 7,7 GB com 99% de uso e 123 MB livres, o que
fazia `go build`/`go vet` falharem com `no space left on device`. **Nada foi limpo,
apagado ou podado**; a raiz `/` tem 17 GB livres.

## 3. Hashes antes e depois

| Arquivo | sha256 antes do gofmt | sha256 final |
|---|---|---|
| `server/internal/handler/file.go` | `ed6457abb386dbd1aafe73cbb3ac565dda7f2323dcde3247cc7f04a387501de8` | `48553c6c48d4423ebfba7d5a05366c0a23ec77d27a463ae4934eda200b557161` |
| `server/internal/handler/file_test.go` | `815b7d1cf12ed9ca18340f9e87813994eebb7e5153aee281125f9e5b3db9e44d` | `815b7d1cf12ed9ca18340f9e87813994eebb7e5153aee281125f9e5b3db9e44d` (inalterado) |

O hash `7f6fcb96...` citado no dispatch não corresponde a nenhum dos dois arquivos no meu
worktree; não sei a que artefato se refere.

`file.go` mudou **apenas por `gofmt -w`**. O que o gofmt reescreveu é um bloco de
comentário **pré-existente** (doc de `buildMarkdownURL`, indentação de lista, linhas
~145-160), não código meu. Prova de que era pré-existente:

```console
$ git show HEAD:multica-auth-work/server/internal/handler/file.go | gofmt -l /dev/stdin
/dev/stdin
```

Ou seja, a versão em HEAD já não estava gofmt-clean sob go1.26. Nenhuma lógica de P1-P3
foi alterada nesta rodada; o código está congelado.

## 4. Comandos, exits e o que realmente rodou

```console
$ gofmt -l internal/handler/file.go internal/handler/file_test.go
            # stdout VAZIA
exit=0

$ go build ./internal/handler/
build_exit=0

$ go vet ./internal/handler/
vet_exit=0

$ go test -c -o "$TMPDIR/handler.test" ./internal/handler/
test_compile_exit=0
# binário de 45.520.855 bytes gerado: o pacote de teste, incluindo o novo
# failingCreateAttachmentDB / errRow / withFailingAttachmentInsert, compila e linka

$ go test ./internal/handler/ -run 'TestUploadFile' -v -count=1
Skipping tests: database not reachable: failed to connect to `user=multica database=multica`:
127.0.0.1:5432 (localhost): dial error: dial tcp 127.0.0.1:5432: connect: connection refused
ok  github.com/multica-ai/multica/server/internal/handler  0.363s
test_run_exit=0

$ go test ./internal/handler/ -run 'TestUploadFile' -v -count=1 | grep -c '^=== RUN'
0

$ git diff --check
exit=0
```

**Falso-verde confirmado exatamente como o GTL-R03 descreveu:** `TestMain`
(`internal/handler/handler_test.go:38-53`) faz `os.Exit(0)` quando o `pgxpool.Ping` falha,
antes de `m.Run()`. Logo `go test` retorna exit 0 com **0 testes executados**. A contagem
de linhas `=== RUN` é `0`. Nenhum S1-S7 rodou; nenhuma regressão rodou.

## 5. Testes que existem e NÃO rodaram

| Item | Nome | Linha |
|---|---|---|
| S1 | `TestUploadFile_ContextlessWithoutEntityRefsReturnsEmptyIDAndStorageLinks` | file_test.go:1309 |
| S2-S4 | `TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload` (subtestes `issue_id`, `comment_id`, `chat_session_id`) | file_test.go:1364 |
| S5 | `TestUploadFile_InsertFailureCleansUpAndReturns500` | file_test.go:1395 |
| S6 | `TestUploadFile_InsertFailureStillFailsWhenCleanupIsNoop` | file_test.go:1436 |
| S7 | `TestUploadFile_SuccessShapesAlwaysCarryContractFields` | file_test.go:1468 |

Regressões que também não rodaram: `TestUploadFileForeignWorkspace`,
`TestUploadFileResolvesWorkspaceViaSlugHeader`,
`TestUploadFileResolvesWorkspaceViaIDHeaderStill`,
`TestUploadFile_AttachesToChatSession`, `TestUploadFile_RejectsForeignChatSession`.

## 6. Correção pedida em file_test.go (findings GTL-R02): feita

O trigger global foi removido. Substituído por falha hermética no nível do `DBTX`:

- `failingCreateAttachmentDB` (file_test.go:1243) encapsula o `DBTX` real e falha **apenas**
  a sentença cujo SQL contém `-- name: CreateAttachment`; todo o resto (membership, actor,
  queries de limpeza) passa para o pool real.
- `errRow` (file_test.go:1266) é um `pgx.Row` cujo `Scan` devolve erro, que é como uma
  query sqlc `:one` propaga erro de banco.
- `withFailingAttachmentInsert` (file_test.go:1273) troca `testHandler.Queries` por
  `db.New(fake)` e **restaura no `t.Cleanup`** — sem DDL, sem mutação de tabela
  compartilhada, sem estado residual se o teste falhar.
- S5 e S6 preservados e reforçados: agora também exigem `fake.calls == 1`, provando que o
  insert foi realmente tentado.
- Precedente no repositório para esse padrão de DBTX falso por texto de SQL:
  `internal/service/task_complete_race_test.go:250`.
- Grep confirma zero resíduo em `internal/handler/file_test.go`: nenhum
  `CREATE TRIGGER`, `DROP TRIGGER` ou `orq26_block_attachment_insert`. As duas
  ocorrências de trigger que restam no pacote estão em
  `autopilot_subscriber_test.go:23,37` e são pré-existentes, não minhas.

Imports adicionados a `file_test.go`: `github.com/jackc/pgx/v5` e
`github.com/jackc/pgx/v5/pgconn`.

## 7. Por que não há DB de teste autorizada aqui

```console
$ ss -ltnp | grep 5432        → NO_LISTENER_5432
$ pgrep -a postgres           → (vazio)
$ command -v docker podman    → NO_CONTAINER_RUNTIME
$ command -v psql pg_isready  → /usr/bin/psql /usr/bin/pg_isready
```

Há cliente `psql`, mas **nenhum servidor** local, nenhum listener em 5432 e nenhum runtime
de contêiner para subir um. As alternativas todas caem em STOP-AND-WAIT (AA-001 §0.1) ou
fora do meu escopo:

1. Instalar Postgres ou um runtime de contêiner → instalação de pacote/serviço (§0.1).
2. Subir um contêiner Postgres → não há runtime, e seria mudança de infraestrutura (§0.1).
3. Apontar `DATABASE_URL` para um Postgres remoto (orq1/ORQ2/compose de outro host) →
   `TestMain` **escreve** fixtures (`setupHandlerTestFixture`) e o suite insere e apaga
   linhas; isso é mutação de ambiente compartilhado, não autorizada, e exigiria credencial
   — que eu não busco nem imprimo (AGENTS.md, skill `aws-secrets-manager`).

Nenhum segredo foi lido, buscado, impresso ou passado em argumento. A única string de
conexão que apareceu foi o default de desenvolvimento embutido em
`handler_test.go:40-41`, e mesmo assim redigi o padrão `postgres://...` na saída.

## 8. O que desbloqueia

Uma das opções, com decisão escrita do owner:

- A: autorizar `docker`/`podman` neste host e um Postgres efêmero de teste (§0.1: instalação).
- B: autorizar um Postgres de teste dedicado e descartável, com `DATABASE_URL` entregue via
  `asm-exec`, e confirmar por escrito que escrever fixtures nele é aceitável.
- C: reatribuir a execução dos gates DB-backed a um agente em host que já tenha o Postgres
  de teste do compose self-host de pé.

Recomendação: C primeiro (custo zero, sem instalação), B como alternativa.

## 9. Estado do diff

```text
 multica-auth-work/server/internal/handler/file.go  |  78 +++--
 .../server/internal/handler/file_test.go           | 328 +++++++++++++++++++++
 2 files changed, 382 insertions(+), 24 deletions(-)

git status --porcelain
 M multica-auth-work/server/internal/handler/file.go
 M multica-auth-work/server/internal/handler/file_test.go
?? .gtl-orq26-gate/          (diretórios privados de gate, 0700, não versionados)
```

## Check-out

- Entregue: gofmt limpo com stdout vazia nos dois arquivos, build 0, vet 0, compilação do
  binário de teste 0, remoção do trigger global por seam hermético de DBTX, hashes antes e
  depois, e prova de que 0 testes executaram.
- BLOQUEIO: nenhum Postgres de teste disponível ou autorizado neste host; S1-S7 e as
  regressões seguem não executados. Código congelado, aguardando re-review e decisão.
- GTL-35 (peer review ORQ-17) foi suspenso por esta prioridade; a evidência já gravada em
  `.deploy-control/p0/evidence/gtl-orq17-auth-peer-review.md` está completa (veredito
  BLOCK) e só falta o report formal ao pane.

---

## 10. Addendum GTL-51 — pré-validação do harness e pedido de autorização

Lido: `.deploy-control/p0/evidence/gtl-orq26-db-gate-harness.md`. Concordo com o veredito e
com a matriz de detecção de falso-verde da §7 — ela cobre exatamente a falha que reproduzi
na §4 deste documento.

### 10.1 Tudo o que o harness exige e NÃO depende de banco: verificado agora

| Guarda do harness | Resultado |
|---|---|
| `gofmt -l` nos dois locked com stdout vazia | OK, stdout vazia |
| `git diff --check` | OK, exit 0 |
| Conjunto dinâmico de migrations (não hard-code 163) | `find migrations -name '*.up.sql' \| wc -l` = **163** no corte atual; o gate deve seguir calculando dinamicamente |
| Fingerprint de entrada congelado (`git diff --binary \| sha256sum`) | `6ce6a0a4f403063eaf4506b2eefac44b956e842313c9b1a3e4544cc936ae4bc0` |
| sha256 por arquivo | `file.go` = `48553c6c48d4423ebfba7d5a05366c0a23ec77d27a463ae4934eda200b557161`; `file_test.go` = `815b7d1cf12ed9ca18340f9e87813994eebb7e5153aee281125f9e5b3db9e44d` |
| Diretórios privados 0700 para `TMPDIR`/`GOCACHE`/`GOTMPDIR` | Já criados e usados (§2) |
| `jq`, `psql`, `pg_isready` presentes | Sim, os três |
| Os **8 nomes de folha** batem com o código | Sim, confirmado: `file_test.go:1365` gera os subtests `issue_id`, `comment_id`, `chat_session_id`; `file_test.go:1475-1476` gera `workspace shape` e `contextless shape`, que o Go normaliza para `workspace_shape` e `contextless_shape` — exatamente os nomes esperados na §4 do harness. Nenhum ajuste de nome é necessário |
| Ausência de `pgcrypto` no ORQ2 | Confirmada de forma independente: `find / -name pgcrypto.control` não retorna nada, apesar de `postgres` e `initdb` existirem. Cluster local com `initdb` falharia na migration `001_init`. Não instalei nada |

### 10.2 Por que eu paro aqui em vez de executar o gate

O gate exige um PostgreSQL 17 **descartável com pgcrypto**, e este host (ORQ2) não tem
`docker`, `podman` nem `nerdctl` — verificado nesta rodada. As opções restantes esbarram em
STOP-AND-WAIT (AA-001 §0.1) e no §0.6:

1. **Instalar runtime de contêiner ou o pacote contrib do Postgres no ORQ2** → instalação de
   pacote/serviço, §0.1. Além disso o próprio GTL-51 proíbe instalar a extensão.
2. **Usar o host Docker do ORQ1** → é o caminho que o addendum admite, mas o §0.6 do
   AUTHORITY_AMENDMENT_001 registra que a raiz do orq1 está **91% cheia, ~2,4 GiB livres**, e
   que as duas imagens OmniRoute quebradas ali são o caminho de rollback documentado, com a
   determinação explícita: "No disk action of any kind without the owner's written decision".
   Puxar `pgvector/pgvector:pg17` (centenas de MB comprimidos, mais volume PGDATA) contra
   2,4 GiB livres é exatamente uma ação de disco com risco de encher a raiz de um host
   compartilhado. Não faço isso sem decisão escrita do owner.
3. **Apontar `DATABASE_URL` para qualquer Postgres existente/live** → proibido pelo GTL-51 e
   inseguro: as fixtures mutam slug e email fixos e o cleanup apaga por esses
   identificadores (`handler_test.go:91-153`).

Portanto: **BLOCK**, conforme o próprio addendum manda ("se isso não for seguramente
possível, pare e peça autorização antes"). Não executei banco, migration, teste com DB nem
qualquer ação de disco em host remoto. Nenhum `.env`, DSN, senha ou valor de credencial foi
lido, buscado ou impresso.

### 10.3 Autorização que peço (uma das três)

| Opção | O que envolve | Risco / reversibilidade | Blast radius |
|---|---|---|---|
| **A (recomendada)** | Reatribuir a execução do gate a um agente em host que **já** tenha runtime de contêiner com folga de disco, ou ao runner CI oficial (`.github/workflows/ci.yml:69-95` já declara `pgvector/pgvector:pg17`) | Nenhuma ação de disco nova em orq1; totalmente reversível | Nulo fora do runner efêmero |
| **B** | Autorizar por escrito o pull de `pgvector/pgvector:pg17` no host Docker do ORQ1 em namespace CI isolado, sem porta publicada, sem tocar o compose live, com `docker rm -fv` e `docker rmi` no cleanup, e com um guard de disco mínimo (abortar se livre < X GiB) | Ação de disco em host 91% cheio; reversível só se o cleanup rodar; risco de pressão de disco durante o run | Host compartilhado orq1, incluindo o caminho de rollback OmniRoute |
| **C** | Autorizar instalação de runtime/contrib no ORQ2 | Instalação de pacote (§0.1); reversível com desinstalação | Host ORQ2 |

### 10.4 Ruling do GTL e correção factual (registrado após a §10.3)

- **Ruling**: opção **A** (CI oficial). B e C rejeitadas como caminho padrão por misturarem
  gate com produção e com host de credenciais. Os dois arquivos locked ficam congelados nos
  hashes desta evidência. Sem commit/push nesta etapa; apenas manifesto do commit CI
  temporário (ver `gtl-orq26-ci-commit-manifest.md`). O GTL pedirá a autorização mínima ao
  owner depois do GTL-63.
- **Correção factual**: a §10.2 acima citou "orq1 91% cheio" a partir do AA-001 §0.6. O
  último estado consolidado informa **ORQ1 em 75%**. A escolha por isolamento (opção A) não
  muda; o argumento de disco deixa de ser o motivo principal e passa a ser secundário
  diante do motivo estrutural: não misturar gate com host de produção/credenciais.


Custo de esperar (mantido do registro original da §10.3): o patch P1-P3 permanece congelado
e não integrável, pois nenhuma das 8 folhas S1-S7 tem prova de execução. O que desbloqueia é
a autorização mínima descrita no manifesto do commit CI temporário.
