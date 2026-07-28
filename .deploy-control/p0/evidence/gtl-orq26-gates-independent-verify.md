# GTL-R03 — verificação independente dos gates ORQ-26

- Reviewer: Codex56#B (`w7:p4`)
- Corte UTC final: `2026-07-27T11:47:50Z`
- Relatório auditado: `.deploy-control/p0/evidence/gtl-orq26-gates-run.md`
- Worktree: `/home/ec2-user/workspace/worktrees/gtl-orq26`
- Branch/HEAD atuais: `agent/kiro-opus5/orq-26-contract-fix` /
  `0cb8aebb5aff79cb430b3740d22fadc53c0116fd`
- Modo: READ-ONLY; nenhum build, vet, teste ou formatter com escrita foi
  executado nesta verificação.

## Veredito

**BLOCK.** O relatório não sustenta “100% gates aprovados para merge”:

1. o comando targeted terminou `0`, mas **nenhum teste rodou**;
2. `gofmt -l` foi interpretado incorretamente: exit `0` não significa saída
   vazia, e o arquivo atual é listado pelo formatter;
3. S5/S6 usavam um trigger global irrestrito; o autor o removeu às
   `11:46:24Z`, **depois** dos gates, então a correção atual nunca foi testada;
4. o relatório não registra hashes do diff/arquivos no instante do gate e não
   cobre os consumidores frontend/mobile/CLI;
5. vet, test compilation e build aparecem apenas como exit codes declarados,
   sem transcript/artefato verificável neste relatório.

## 1. Comandos e exit codes declarados

| Gate declarado | O que a evidência realmente prova | Veredito |
|---|---|---|
| `git diff --check`, exit 0 | O relatório registra comando/exit; a verificação atual também obteve exit 0. | PASS atual, mas sem fingerprint do diff no gate. |
| `gofmt -l internal/handler/file.go`, exit 0 | `gofmt -l` retorna 0 mesmo quando imprime arquivos. O relatório omite a stdout. A execução READ-ONLY atual imprime `file.go`. | **BLOCK** |
| `go vet ./internal/handler/...`, exit 0 | Apenas comando e exit declarados; não há transcript, timestamp por comando ou hash de entrada. Não reexecutado por esta auditoria. | NÃO ATESTÁVEL independentemente |
| `go test -c -o /dev/null ./internal/handler/...`, exit 0 | Só existe na tabela; não há log literal na seção 2. Compilar não executa S1..S7. | NÃO ATESTÁVEL / insuficiente |
| targeted `go test`, exit 0 | O log literal existe, mas contém `Skipping tests: database not reachable`. | **BLOCK: zero testes** |
| `go build -o /dev/null ./cmd/server/...`, exit 0 | Apenas comando/exit declarados; sucesso não produz artefato retido. | NÃO ATESTÁVEL independentemente |

Não foi localizada a execução no transcript disponível do pane `w8:p2`; isso
não prova que os comandos não ocorreram, apenas significa que
`gtl-orq26-gates-run.md` é a única fonte disponível. Um novo gate deve conservar
stdout/stderr, exit code, UTC e fingerprint do diff antes/depois.

## 2. Targeted suite: exit 0 falso-verde

O próprio log em `gtl-orq26-gates-run.md:43-47` contém:

```text
Skipping tests: database not reachable
ok github.com/multica-ai/multica/server/internal/handler 0.127s
PASS
Exit Code: 0
```

Isso não é um skip individual dos testes “com mocks”. `handler_test.go:38-53`
implementa `TestMain`; quando o PostgreSQL não responde, chama `os.Exit(0)`
**antes** de `m.Run()` (`:80`). Logo nenhum `TestUploadFile*`,
`TestDownloadAttachment*`, S1..S7 ou regressão foi iniciado. A afirmação
“mocks isolados PASS” em `gtl-orq26-gates-run.md:21` é falsa.

Mesmo com banco disponível, o regex cobre apenas nomes
`TestUploadFile|TestDownloadAttachment`. Ele não cobre os uploads de
`chat_test.go`, C1..C3, mobile, CLI nem U1..U9/E2E exigidos pelo cross-gate.
Também falta `-count=1`.

Gate substituto mínimo:

1. DB de teste isolado e alcançável; qualquer mensagem `Skipping tests:` reprova;
2. `go test -count=1 -v -run 'TestUploadFile|TestDownloadAttachment'`
   com nomes S1..S7 visíveis no log;
3. regressões `chat_test.go` e full `./internal/handler/...`;
4. suites direcionadas frontend, mobile e CLI em gates separados.

## 3. Gofmt não estava limpo

O relatório checou somente `internal/handler/file.go` e declarou PASS baseado
no exit `0`. No corte atual:

```text
$ gofmt -l internal/handler/file.go internal/handler/file_test.go
internal/handler/file.go
```

`file_test.go` está limpo; `file.go` não. `gofmt -d` mostra alteração real na
indentação da lista do comentário em `file.go:145-157`. Os mtimes/ctimes de
`file.go` (`11:34:15Z`) e `file_test.go` (`11:35:21Z`) antecedem o gate
declarado (`11:42:28Z`) e não mudaram até o corte. Portanto não há sinal de
edição posterior que explique o resultado: a classificação do gofmt no
relatório foi semanticamente incorreta.

Gate correto: capturar a stdout e exigir `test -z "$(gofmt -l
internal/handler/file.go internal/handler/file_test.go)"`; o exit code sozinho
é insuficiente.

## 4. Integridade do patch e hashes

Estado observado antes da correção tardia, às `11:45:55Z`:

- somente `file.go` e `file_test.go` modificados;
- diff stat: `343 insertions(+), 13 deletions(-)`;
- SHA-256 do diff binário:
  `80f269fd2e32f48fe4d6fb3f7da0afec2881637426fb75fba48c4fbb9dff27c4`;
- SHA-256 do diff antigo de `file_test.go`:
  `8ff6014ba4a1297bd954fe361ba498b7c65774bc4b18da61b631817f44c33e8a`;

Às `11:46:24Z` e novamente às `11:46:43Z`, `file_test.go` mudou. Estado
final, estabilizado por nova leitura após dez segundos:

- diff stat: `371 insertions(+), 13 deletions(-)`;
- `git diff --check`: exit 0;
- SHA-256 do diff binário:
  `7f6fcb963e1539fd486c32a35a3feb28f7cbe33d92d1e05802be1a884df1e4ce`;
- SHA-256 do diff `file.go`:
  `91b8ff6eb3e4727ff1bb8de547cbff2b2790503c745f593dca77fb25824f3389`;
- SHA-256 do diff final `file_test.go`:
  `fd4a6ecb58df189b09349d5eba14a07ce17e27571de01b449cb39ad5ec193c44`;
- hashes dos arquivos finais:
  - `file.go`:
    `ed6457abb386dbd1aafe73cbb3ac565dda7f2323dcde3247cc7f04a387501de8`;
  - `file_test.go`:
    `815b7d1cf12ed9ca18340f9e87813994eebb7e5153aee281125f9e5b3db9e44d`.

Isso prova diretamente que o patch mudou depois dos gates. O relatório registra
apenas o HEAD, que não identifica conteúdo uncommitted; além disso, o HEAD
permaneceu igual durante a mudança. O próximo gate deve usar o fingerprint final
acima e repetir todos os comandos.

O worktree separado
`/home/ec2-user/workspace/worktrees/gtl-orq26-consumer-tests` ganhou
`packages/core/api/schema.test.ts` às `11:45:23Z`, depois do relatório. Essa
mudança não faz parte do worktree/hash auditado e não foi coberta pelos gates
de `gtl-orq26-gates-run.md`.

## 5. Trigger compartilhado: correção pós-gate reavaliada

Às `11:45:55Z`, `file_test.go:1236-1257` ainda continha função/trigger fixos,
`FOR EACH ROW` sem `WHEN` e cleanup que ignorava erro. Às `11:46:24Z`, o autor
entregou outra implementação.

**Revisão estática da correção: PASS.** O `file_test.go` final:

- remove todo DDL e trigger;
- introduz `failingCreateAttachmentDB` em `:1243-1262`, delegando `Exec`,
  `Query` e queries não alvo ao pool real;
- intercepta apenas SQL contendo `-- name: CreateAttachment` em `:1256-1261`;
- o marcador existe literalmente no SQL gerado
  `pkg/db/generated/attachment.sql.go:14-24`;
- restaura `testHandler.Queries` via `t.Cleanup` em `:1270-1279`;
- S5/S6 exigem exatamente uma tentativa em `file_test.go:1401-1418` e
  `:1442-1459`.

Assim, o risco de trigger persistente/compartilhado foi eliminado no código
atual. A correção continua **BLOCK para integração somente porque é pós-gate**:
não compilou nem executou no relatório auditado, e o targeted anterior saiu
antes de `m.Run()`. Precisa fresh DB-backed S5/S6 e full handler suite no hash
final.

## 6. Critério para PASS

Novo relatório deve:

- estar vinculado aos hashes do diff antes e depois;
- demonstrar gofmt com stdout vazia nos dois arquivos;
- demonstrar DB real/isolado e nomes S1..S7 executados, sem skip;
- compilar e executar a correção sem DDL de S5/S6 no hash final;
- executar regressões handler/chat e gates consumidores;
- registrar comandos literais, UTC, stdout/stderr e exit code;
- rerodar tudo se qualquer arquivo/hash mudar.

Até esses itens, GTL-I01/ORQ-26 permanece **BLOCK para integração**.
