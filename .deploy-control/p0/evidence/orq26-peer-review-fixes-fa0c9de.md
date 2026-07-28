# ORQ-26 - correcao do BLOCK do peer review de `4246cfa` (commit `fa0c9de`)

- executor/owner exclusivo: **Opus48#A** - ORQ2 w6:p1 - 2026-07-27T18:12Z
- worktree: `/home/ec2-user/workspace/worktrees/ci-orq26-db-gate`, branch `ci/orq26-db-gate`
- **commit novo, sem amend**: `fa0c9de` sobre `4246cfa` sobre `d244718`; worktree limpo
- **PEDIDO: RE-REVIEW.** Sem push ate PASS.

## O revisor estava certo, e o defeito (1) era fatal

**(1) O teste F2 estava quebrado, nao apenas fragil.** Ele guardava o `context.Context` e o inspecionava
**depois** do retorno de `cleanupOrphanObject` - momento em que o `defer cancel()` do helper **ja
disparou**. Logo `Err()` reporta `context.Canceled` **independentemente** de como o contexto foi
construido, e a assercao `Err() == nil` **so podia falhar**. Eu introduzi um teste que reprovaria em CI.
Aceito o BLOCK sem ressalva.

Correcao: amostragem **dentro** do `Delete`, via `deleteSnapshot`, capturando no instante da chamada:
```go
deadline, hasDeadline := ctx.Deadline()
m.snapshots = append(m.snapshots, deleteSnapshot{
    key: key, err: ctx.Err(), deadline: deadline,
    hasDeadline: hasDeadline, value: ctx.Value(cleanupProbeKey{}),
})
```
Acrescentei duas provas que faltavam: o **valor** de escopo de requisicao tem de sobreviver (prova que
e `WithoutCancel` e nao um contexto novo) e a deadline e medida contra um `time.Now()` tomado **antes**
da chamada, em vez de `time.Until` depois.

**(2) O teste F1 nao provava o wiring.** Chamava o helper direto. Adicionei
`TestUploadFile_InsertFailureDeletesOriginalKeyWhenKeyFromURLCollapses`, que exercita o
**`UploadFile` real** com insert falhando e um storage cujo `Upload` devolve
`https://unrecognised.example.net/some/other/prefix/collapsed.txt` e cujo `KeyFromURL` **colapsa** para
o nome nu, reproduzindo o fallback do `S3Storage`. O teste exige que a chave registrada no `Delete`
conserve o prefixo `workspaces/<id>/` e a extensao, **rejeita** `collapsed.txt` e qualquer chave sem
`/`, confirma que o proprio fake colapsaria a URL, e ainda checa `Err()`/deadline no caminho do
handler. Se alguem voltar a deletar por URL, este teste falha.

**(3) Guarda de arquivos endurecida para igualdade de range.** Era `git log -1` + subconjunto, o que
permitia um commit **anterior** contrabandear um quarto arquivo. Agora:
- `fetch-depth: 0` no checkout, para o ref base existir;
- base = `origin/${{ github.event.repository.default_branch }}`, com **parada fail-closed** se o ref nao
  existir - nao degrada em silencio para checagem de um commit;
- uniao de `git log --name-only base..HEAD`, e **igualdade de conjunto** (`comm -23` vazio, `comm -13`
  vazio e `cmp -s`), nao mais containment.

Isso resolve exatamente a limitacao que eu havia registrado no meu proprio parecer. E note por que era
necessario: **este commit nao toca `file.go`**, portanto uma checagem de um unico commit veria 2
arquivos; a uniao do range ve os 3 e a igualdade continua valendo.

## Verificacao executada

```
gofmt -l file.go file_test.go                      -> vazio
go vet ./internal/handler                          -> exit 0
go vet ./...                                       -> exit 0
go build ./...                                     -> exit 0
git diff --check                                   -> limpo
go test -race -c -o /dev/null ./internal/handler   -> COMPILA
yaml.safe_load do workflow                         -> OK, 16 passos
simulacao do passo de freeze (sha256sum -c)         -> file.go OK, file_test.go OK
dry-run da guarda de range (base = d244718^)        -> uniao == allowed, IGUALDADE OK
```
Gate atualizado: `LOCKED_TEST_SHA256` -> `02f9bb0acf50f492fa524deb6e9898dfcfa74375f0a96407168ed574349e442b`
(`LOCKED_FILE_SHA256` inalterado, `70f45ebd…`, porque `file.go` nao mudou); folhas **10 -> 11** em
`run_re`, `expected`, `test -eq 11`, titulo do passo e linha do summary.

## Nao-afirmacoes

- **Sem push, PR, merge, quadro, AWS, segredo ou mutacao de runtime.**
- **Os 11 testes continuam sem execucao local** - nao ha Postgres neste host e `handler_test.go:40-53`
  faz `os.Exit(0)` antes do `m.Run`; `go test` reportaria `ok` com zero testes. Afirmo **compilacao**
  sob `-race`, nao aprovacao. O novo teste de wiring **depende de banco** (usa `testHandler`,
  `testPool` e `withFailingAttachmentInsert`), portanto so roda em CI.
- A guarda de range **nao** foi executada no GitHub: validei a **logica** localmente com base simulada
  `d244718^`. Se o branch default tiver outro nome ou o ref nao estiver disponivel no runner, o passo
  **para** - por desenho, mas isso ainda nao foi observado em execucao real.
- `fetch-depth: 0` aumenta o tempo de checkout; nao medi o impacto.
- Nao alterei `file.go` neste commit, nem `deleteS3Object`, nem o fallback de `KeyFromURL` em
  `internal/storage/s3.go` - este ultimo **permanece** e segue como recomendacao de card proprio.
- Nao toquei `gtl-orq26` nem `gtl-orq26-consumer-tests` (congelados), nem o worktree principal.
