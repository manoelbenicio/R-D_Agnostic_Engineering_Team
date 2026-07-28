# ORQ-12 - `account_id` snapshot em `task_usage` (commit `e7c6a5c`, GTM R2)

- executor/owner: **Opus48#A** - ORQ2 w6:p1 - 2026-07-28T14:20Z
- worktree dedicado: `/home/ec2-user/workspace/worktrees/orq12-account-id`, limpo
- branch novo `agent/opus48-a/orq12-task-usage-account-id`, base **`c0e93a2`** (ORQ-13
  `feat(cost): record thinking_level on task usage end to end`)
- commit **local** `e7c6a5c`. **Sem push, PR, merge, quadro, AWS, segredo, Docker ou alvo real.**
- **PEDIDO: revisao independente** por agente que nao seja eu.

## 1. Numero de migration **nao materializado** - e por que a mecanica funciona

O DDL esta em `migrations/staging/NEXT_CANONICAL_task_usage_account_id.{up,down}.sql`, com o
**placeholder** no nome. Ele so recebe numero quando o registrador central varrer a proxima versao
livre **acima de 127** - `127_task_usage_thinking_level` e a mais alta materializada nesta linhagem
(confirmei: existe no worktree, criada pelo ORQ-13).

O que garante que nada e aplicado antes disso: `internal/migrations.Files` usa
```go
files, err := filepath.Glob(filepath.Join(dir, "*"+suffix))   // migrations.go:59
```
isto e, glob **nao recursivo**. Verifiquei empiricamente que `migrations/*.up.sql` **nao** inclui
`staging/` (164 arquivos, zero de staging). Logo `migrate up` nao pode aplicar o arquivo staged.

O `sqlc`, porem, **precisa** ver a coluna para tipar o codigo gerado - e medi que ele **tambem nao
recursa**: com `schema: "migrations/"` o `models.go` ficou **byte-identico** apos `generate`. Por isso
`sqlc.yaml` passou a listar **dois** roots explicitos (`migrations/` e `migrations/staging/`). As duas
ferramentas discordam **de proposito**: revisado e tipado, mas nao aplicado.

## 2. Resolucao **server-side**, e o chamador nao pode influenciar

`UpsertTaskUsageParams` **nao ganhou campo nenhum** - confirmado no codigo gerado. A atribuicao e
derivada dentro da propria query:
```sql
(
    SELECT asg.account_id
    FROM agent_task_queue q
    JOIN assignments asg ON asg.agent_id = q.agent_id
    WHERE q.id = $1
)
```
Cadeia `agent -> assignment -> account` vinda de `migrations/123_rotation.up.sql`
(`assignments(agent_id PK, account_id NOT NULL REFERENCES accounts)`). Consequencia de seguranca: um
daemon reportando tokens **nao pode** atribuir gasto a outra conta, porque nao existe parametro para
isso.

## 3. Nulavel, aditivo, nunca inventado

- linha legada -> **NULL**: precede a coluna e a atribuicao nao e recuperavel a posteriori;
- agente **sem** assignment -> **NULL**. `NULL` significa "nao atribuivel", nunca "a conta padrao";
- `ON DELETE SET NULL`: apagar conta perde a **atribuicao**, nunca os tokens;
- `ON CONFLICT` usa `COALESCE(EXCLUDED.account_id, task_usage.account_id)`, igual ao `thinking_level`:
  um re-report parcial **preenche** um desconhecido, mas **nunca reescreve** o que ja foi gravado;
- `UNIQUE (task_id, provider, model)` da 032 **intocada** - se a conta entrasse na chave, uma task
  poderia acumular duas linhas do mesmo modelo e **dobrar** a contagem de tokens;
- rollups 073/084/101/102 **intocados**: esta fase apenas registra a dimensao;
- indice **parcial** `WHERE account_id IS NOT NULL`, porque so linhas atribuiveis sao agrupadas por
  conta e a cauda NULL inflaria o indice sem leitor.

## 4. Relatorios que **nao escondem** a cauda

`ListTaskUsageByAccount` e `GetTaskUsageAccountAttribution` expoem o bucket `NULL`
**explicitamente**, com `attributable` booleano e `row_count`. Filtrar o NULL faria os totais por conta
**parecerem completos** enquanto linhas legadas e de agentes sem assignment desapareciam - exatamente a
leitura errada que a coluna existe para impedir. E a atribuicao devolve **tres contadores**
(`total_rows`, `attributed_rows`, `unattributed_rows`) em vez de uma razao, para que denominador zero
nao seja mal lido.

## 5. Testes

`internal/handler/task_usage_account_test.go`, 5 testes, fixando:
1. resolucao pela assignment;
2. **NULL** com agente sem assignment;
3. snapshot **sobrevive a reatribuicao** - o agente rotaciona, a linha e re-reportada, a conta original
   permanece **e** a correcao de tokens (7,9) chega;
4. um **NULL e preenchivel** depois, quando a assignment passa a existir;
5. o relatorio mantem o bucket NULL com seu `row_count`, e os tres contadores **somam**.

**Defeito meu, corrigido antes do commit:** a primeira versao do helper inseria
`accounts (provider, label, status='active')`. O schema real (`123_rotation`) tem `vendor`,
`tenant_id` **NOT NULL** e um `CHECK` que **nao** aceita `active`. Compilava, e falharia **so em CI** -
o mesmo tipo de defeito latente que reprovei em outros. Corrigido para
`(vendor, tenant_id, status='available')`.

## 6. Verificacao executada

```
go build ./...                     -> OK
go vet ./internal/handler ./pkg/db/... -> OK
sqlc generate (v1.31.1)            -> OK, e o diff do models.go e exatamente a coluna nova
go test -c -o /dev/null ./internal/handler -> COMPILA
gofmt -l                           -> vazio
git diff --check                   -> limpo
```

## 7. Nao-afirmacoes

- **Os testes com banco NAO foram executados.** Este host nao tem Postgres alcancavel (5432 fechado,
  `docker` ausente) e o `TestMain` do pacote `handler` faz `os.Exit(0)` antes do `m.Run`, o que o
  `go test` reporta como `ok` com **zero** testes. Afirmo **compilacao**, nao aprovacao.
- Portanto **nao verifiquei em execucao**: se a subquery resolve, se o `COALESCE` protege o snapshot, se
  o indice parcial e criado, nem se os contadores somam. Tudo isso e o que a revisao com DB efemero
  precisa provar.
- **Nao apliquei a migration em lugar nenhum**, nem em banco de teste. O arquivo esta staged e sem
  numero.
- **Nao reivindiquei numero de migration** e nao pedi reserva ao registrador; nao sou o registrador.
- `sqlc.yaml` foi alterado - e uma mudanca de **configuracao compartilhada**: qualquer `sqlc generate`
  futuro passara a ler `staging/`. Isso e intencional e documentado no proprio arquivo, mas o revisor
  deve decidir se aceita esse acoplamento ou prefere um perfil separado.
- Nao medi impacto de performance do indice parcial nem do subselect no upsert (um lookup por PK em
  `agent_task_queue` mais um por PK em `assignments`).
- ORQ-26 segue em `HOLD_EXTERNAL_BILLING`; ORQ-42 `fc77e89` aguarda re-review. Nao toquei nenhum dos
  dois nesta rodada.
