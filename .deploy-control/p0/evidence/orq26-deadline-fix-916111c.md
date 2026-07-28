# ORQ-26 - correcao do unico BLOCK restante (commit `916111c`)

- executor/owner exclusivo: **Opus48#A** - ORQ2 w6:p1 - 2026-07-27T18:26Z
- **commit novo, sem amend**: `916111c` sobre `fa0c9de` sobre `4246cfa` sobre `d244718`; worktree limpo
- **PEDIDO: RE-REVIEW.** Sem push ate PASS.

## O defeito, e o reconhecimento

O revisor esta certo: eu troquei uma assercao quebrada por outra. Em `fa0c9de` o `before` era amostrado
**antes** de `cleanupOrphanObject` rodar, enquanto a deadline e criada como `time.Now()+5s`
**durante** a chamada. Logo `deadline.Sub(before)` e **sempre um pouco maior** que 5 s e o ramo
`remaining > 5*time.Second` **dispararia** - o teste reprovaria em CI. Dois commits seguidos com defeito
na mesma assercao; nao ha desculpa e registro isso explicitamente.

## Correcao

`deleteSnapshot` passou a registrar `observedAt`, amostrado **no mesmo instante** que a deadline,
**dentro** do `Delete`:
```go
deadline, hasDeadline := ctx.Deadline()
m.snapshots = append(m.snapshots, deleteSnapshot{
    ..., deadline: deadline, hasDeadline: hasDeadline, observedAt: time.Now(), ...
})
```
e os testes calculam o orcamento **so** com os dois carimbos internos:
```go
remaining := got.deadline.Sub(got.observedAt)
if remaining <= 0 || remaining > 5*time.Second { ... }
```
O carimbo pre-chamada foi **removido** (`grep 'before :=|Sub(before)'` nao retorna nada). O comentario
do campo documenta a armadilha, para nao voltar.

**Endurecimento adicional:** o teste de wiring `...KeyFromURLCollapses` verificava **apenas** que
existia deadline; agora tambem exige `0 < deadline.Sub(observedAt) <= 5s` no caminho do handler. Ou
seja, as duas assercoes de orcamento passaram a existir nos **dois** testes.

## Verificacao executada

```
gofmt -l file.go file_test.go                      -> vazio
go vet ./internal/handler                          -> OK
go vet ./...                                       -> exit 0
go build ./...                                     -> OK
git diff --check                                   -> limpo
go test -race -c -o /dev/null ./internal/handler   -> COMPILA
yaml.safe_load                                     -> OK, 16 passos
freeze simulado (sha256sum -c)                      -> file.go OK, file_test.go OK
dry-run da igualdade de range (base d244718^)       -> uniao == allowed
grep 'before :=|Sub(before)'                        -> nenhuma ocorrencia
```
Gate: `LOCKED_TEST_SHA256` -> `b1c3c515e31d97cfa4ed26a07ee7ae2c1c824d07e61fc05929c7a0ed92fd8f1a`.
`LOCKED_FILE_SHA256` segue `70f45ebd…` (o `file.go` nao mudou). Folhas continuam **11**.

## Nao-afirmacoes

- **Sem push, PR, merge, quadro, AWS, segredo ou mutacao de runtime.**
- **Os 11 testes seguem sem execucao local.** Nao ha Postgres neste host e `handler_test.go:40-53` faz
  `os.Exit(0)` antes do `m.Run`. Afirmo **compilacao** sob `-race`; **nao** afirmo que passam. Em
  particular, a nova assercao de orcamento **nunca foi executada** - o erro anterior era exatamente
  desse tipo, e a unica prova real vem do runner.
- A guarda de igualdade de range nao foi executada no GitHub; validei a **logica** com base simulada.
- `file.go` nao foi tocado neste commit; o fallback de `KeyFromURL` em `internal/storage/s3.go`
  **permanece** e segue como recomendacao de card proprio.
- Nao toquei `gtl-orq26`, `gtl-orq26-consumer-tests` (congelados) nem o worktree principal.
