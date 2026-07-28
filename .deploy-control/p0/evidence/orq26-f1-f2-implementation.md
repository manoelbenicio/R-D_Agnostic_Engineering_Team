# ORQ-26 - implementacao de F1 e F2 (commit `4246cfa`)

- executor/owner: **Opus48#A** - ORQ2 w6:p1 - 2026-07-27T17:58Z, sob handoff explicito
- worktree isolado: `/home/ec2-user/workspace/worktrees/ci-orq26-db-gate`, branch `ci/orq26-db-gate`
- **commit novo em cima, sem amend**: `4246cfa12dca326d357f7d6854c148013caf2ef9` sobre `d244718`
- escopo: **exatamente 3 arquivos** (nenhum quarto), worktree limpo apos o commit
```
 .github/workflows/orq26-db-gate.yml                | 14 ++--
 multica-auth-work/server/internal/handler/file.go  | 32 +++++++-
 .../server/internal/handler/file_test.go           | 89 ++++++++++++++++++++++
```
- **sem push, sem PR, sem merge, sem quadro, sem AWS, sem segredo, sem mutacao de runtime.**

## F1 - deletar pela chave exata

`h.deleteS3Object(r.Context(), link)` foi substituido por `h.cleanupOrphanObject(r.Context(), key)`.
O novo helper deleta **a chave que o upload usou** e nunca deriva chave de URL. `deleteS3Object` fica
intacto, porque outros chamadores o usam.

Razao registrada no codigo: o fallback final de `Storage.KeyFromURL` e *"everything after the last
`/`"*, portanto uma forma de URL nao reconhecida (`S3Storage` sem `cdnDomain`, `endpointURL` e
`region`) reduz `users/<id>/<file>` a `<file>` e o delete passaria a mirar a **raiz do bucket**.

## F2 - contexto de limpeza desacoplado do cancelamento

```go
cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
defer cancel()
h.Storage.Delete(cleanupCtx, key)
```
Valores (tracing, auth) preservados, cancelamento descartado, prazo proprio de 5 s. Motivo: o
`CreateAttachment` falha, na maioria das vezes, **porque** o cliente desconectou - e nesse caso o
`r.Context()` ja esta `Done` e o `Delete` falharia de imediato, deixando o orfao.

## Testes focados (sem banco)

Adicionados a `file_test.go`, com um fake que registra **chave e contexto** de cada `Delete`
(`mockStorageRecordingDelete`; o `mockStorage` existente descarta os dois):

1. `TestCleanupOrphanObject_DeletesExactKeyNotDerivedFromURL` - exige **uma** chamada, chave
   **integral**, e **rejeita explicitamente** a regressao do nome-de-arquivo nu
   (`store.deleteKeys[0] == last` = falha), mais `fileCount() == 0`.
2. `TestCleanupOrphanObject_RunsWithCanceledRequestContext` - contexto da requisicao **cancelado
   antes** da limpeza; exige que o `Delete` ocorra, que o contexto recebido tenha `Err() == nil`, que
   tenha **deadline propria** e que o restante esteja em `(0, 5s]`.

Nenhum dos dois toca `testPool`, `testHandler` ou banco: usam `&Handler{Storage: store}`.

## Verificacao local executada

```
gofmt -l internal/handler/file.go internal/handler/file_test.go   -> vazio
go vet ./internal/handler                                          -> exit 0
go vet ./...                                                       -> exit 0
go build ./...                                                     -> exit 0
git diff --check                                                   -> limpo
go test -race -c -o /dev/null ./internal/handler                   -> COMPILA
python3 yaml.safe_load do workflow                                 -> YAML OK
simulacao do passo de freeze (sha256sum -c)                         -> file.go OK, file_test.go OK
```
Caches privados `0700` em `~/.cache/{go-build,gotmp}-review-a`.

## O que **nao** foi provado localmente, e a prova de que nao pode ser

A suite com banco **nao** rodou: nao ha Postgres alcancavel neste host (5432 fechado, `docker`
ausente) e `internal/handler/handler_test.go:40-53` faz `os.Exit(0)` antes do `m.Run`. Medi o efeito:
```
$ go test -count=1 -run TestCleanupOrphanObject ./internal/handler
ok   github.com/multica-ai/multica/server/internal/handler  0.079s     # exit 0, ZERO testes
```
Em modo normal o `go test` **esconde** a mensagem de skip - o falso-verde e invisivel. Confirmei
tambem que a guarda do gate **pegaria** esse caso:
```
$ go test -json ... | jq -r 'select(.Action=="output")|.Output'
Skipping tests: database not reachable: ... 127.0.0.1:5432 ... connection refused
guarda 'Skipping tests:|database not reachable|...'  -> DISPARA (exit 91)
eventos com .Test != null                            -> NENHUM  (logo os grep -Fxq das 10 folhas falham)
```
Ou seja: o harness do gate esta correto e a unica prova valida dos 10 testes vem do runner, apos push
autorizado. **Nao apresento o `ok` local como aprovacao.**

## Gate atualizado

- `LOCKED_FILE_SHA256` -> `70f45ebdcd07e24d892657a9add7a624b2c10bb2c657d9af1118af453c0fc50e`
- `LOCKED_TEST_SHA256` -> `768b18ef64ae229e5182724eefa337a8cd0a81ff8c2e5a8ef12945065f2ebe63`
- ambos conferidos contra os arquivos reais pela simulacao do passo de freeze
- `run_re` ganhou os dois nomes novos; `expected` foi de 8 para **10** folhas; `test "$count" -eq 10`;
  titulo do passo e a linha do summary atualizados para 10
- `paths:` do trigger ja cobre os 3 arquivos, e a guarda de arquivos alterados
  (`git log -1 --name-only`) casa exatamente com o novo commit

## Nao-afirmacoes

- **Nao dei push, nao abri PR, nao fiz merge, nao toquei quadro, AWS, segredo ou runtime.**
- **Nao executei os 10 testes** - impossivel neste host, com a prova acima. Nao afirmo que passam;
  afirmo que **compilam** sob `-race` e que a mecanica do gate os exige.
- Nao alterei `deleteS3Object` nem seus outros chamadores; o fallback de `KeyFromURL` em
  `internal/storage/s3.go` **permanece** e continua sendo um risco para quem deleta por URL - fora do
  escopo de 3 arquivos, fica registrado como recomendacao separada.
- Nao toquei `gtl-orq26` nem `gtl-orq26-consumer-tests` (congelados), nem o worktree principal sujo.
- Metodo: edicao por arquivo com verificacao imediata (gofmt/vet/build) antes do `git add`; git e a
  rede de seguranca, e o commit e a promocao atomica dos tres arquivos de uma vez.
