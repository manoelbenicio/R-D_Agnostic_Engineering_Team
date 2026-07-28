# ORQ-13 fase 1 - manifesto EXATO de split: commit A (drift de regen) x commit B (feature)

Agente: Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T13:50Z
Cartao: **ORQ-13** · fase 1 (thinking_level em task_usage)
Worktree congelado: `/home/ec2-user/workspace/worktrees/gtl-i03-orq13-phase1`
Branch: `agent/opus48-b/orq-13-thinking-level` · base **`0cb8aeb`** · commits proprios: **0**
Modo: READ-ONLY. **Nao editei, nao fiz `git add`/stage, nao commitei, nao rodei migration.**

Estado total do worktree: 7 arquivos modificados (+138/-18) e 4 novos untracked.

---

## 1. RESUMO DO SPLIT

| commit | natureza | arquivos | hunks | por que separado |
|---|---|---|---|---|
| **A** | drift **pre-existente** de `sqlc generate`, nao causado por esta feature | 2 modificados | 7 | reverter/regerar codigo gerado desatualizado e mudanca de baseline, com risco proprio (secao 2.3) |
| **B** | feature `thinking_level` **somente** | 5 modificados + 4 novos | 15 | mudanca de produto, revisavel isoladamente |

Regra de ordem: **A antes de B.** Se B entrar primeiro, o proximo `sqlc generate` de qualquer pessoa
arrasta o drift de A para dentro de um PR de feature alheio - exatamente o problema que este split
existe para evitar.

---

## 2. COMMIT A - `chore(db): regenerate sqlc baseline (pre-existing drift)`

### 2.1 Inventario de arquivo e hunk

**A1. `multica-auth-work/server/pkg/db/generated/models.go`** - 6 hunks no arquivo, **5 pertencem a A**:

| hunk | ancora | conteudo | commit |
|---|---|---|---|
| `@@ -12,0 +13,17 @@` | apos `import (` | `+type Account struct { ... }` | **A** |
| `@@ -103,0 +121,15 @@` | apos `type AgentTaskQueue struct` | `+type ApprovedAccount struct`, `+type Assignment struct` | **A** |
| `@@ -249,0 +282,10 @@` | apos `type ContactSalesInquiry struct` | `+type Credential struct` | **A** |
| `@@ -587,0 +630,10 @@` | apos `type ProjectResource struct` | `+type RotationEvent struct` | **A** |
| `@@ -706,2 +758,4 @@` | dentro de `type TaskUsage struct` | `+// Reasoning tier ...` + `+ThinkingLevel pgtype.Text` | **B** |
| `@@ -763,2 +817,9 @@` | apos `type User struct` | `+type UserPasswordCredential struct` | **A** |

Origem do drift: as migrations **123** (`accounts`, `credentials`, `assignments`,
`rotation_events`), **124** (`approved_accounts`) e **125** (`user_password_credential`) ja existiam
no repo, mas `models.go` versionado nao continha os structs correspondentes. Ou seja: o generated
estava **atrasado em 3 migrations**, independente de mim.

**A2. `multica-auth-work/server/pkg/db/generated/task_message.sql.go`** - 2 hunks, **ambos em A**:

| hunk | conteudo |
|---|---|
| `@@ -52,19 +52,6 @@` | remove o bloco `getTaskMessageMaxSeq` + `func GetTaskMessageMaxSeq` (13 linhas) |
| `@@ -75,6 +62,19 @@` | reinsere o mesmo bloco depois de `DeleteTaskMessages` |

### 2.2 ATENCAO - A2 nao e um "move" puro

Comparei o bloco removido com o inserido, linha por linha. **Nao sao identicos.** Diferenca exata,
3 linhas:
```
- 	var maxSeq int32          ->  + 	var max_seq int32
- 	err := row.Scan(&maxSeq)  ->  + 	err := row.Scan(&max_seq)
- 	return maxSeq, err        ->  + 	return max_seq, err
```
Leitura: o arquivo **versionado** usa `maxSeq` (camelCase) e o `sqlc generate` produz `max_seq`
(snake_case, derivado do alias SQL `AS max_seq`). Como o gerador nunca emitiria `maxSeq`, a conclusao
e que **o arquivo gerado foi editado a mao em algum momento**, provavelmente para satisfazer um
linter de nomenclatura Go.

Consequencia pratica para o commit A: aplica-lo **reverte essa edicao manual**. Risco concreto e o
gate de lint - `max_seq` tem underscore em nome de variavel Go, que e exatamente o padrao que
`revive`/`stylecheck` (ST1003) reclamam.

**Nao consegui provar se o lint quebra.** Medido: nao existe `.golangci.yml`/`.golangci.yaml` em
nenhum nivel do repo (`find -maxdepth 3` vazio), e o `ci.yml:115-118` invoca
`golangci-lint-action@v9` com `version: v2.12` e `working-directory: server`, portanto **defaults do
golangci-lint v2**. Se ST1003 estiver ativo por default nessa versao, o commit A quebra o lint.
Nao rodei `golangci-lint` porque instala-lo e proibido. **Este e o item de maior risco do commit A e
precisa de verificacao por quem tiver o gate.**

Nota adicional: o `ci.yml` esta em `multica-auth-work/.github/workflows/`, e o GitHub Actions le
apenas `.github/workflows/` da raiz do repositorio, que **nao existe**. Logo esse gate de lint hoje
**nao executa** neste repositorio - o que nao o torna irrelevante, apenas nao automatico.

### 2.3 Gate independente do commit A

Aplicavel **sem** nada do commit B, com `TMPDIR`, `GOCACHE` e `GOTMPDIR` fora de `/tmp` (o `/tmp` do
ORQ2 esta 100% cheio):
```bash
cd multica-auth-work/server
sqlc generate                 # deve produzir ZERO diff apos A: A e exatamente a saida do gerador
git diff --exit-code pkg/db/generated/    # esperado: exit 0
go build ./...                            # esperado: exit 0
go vet ./pkg/db/...                       # esperado: exit 0
gofmt -l pkg/db/generated/                # esperado: vazio
golangci-lint run ./...                   # OBRIGATORIO aqui: valida o risco de 2.2
go test ./internal/handler ./internal/daemon -count=1   # nenhuma mudanca de comportamento esperada
```
Criterio de aceite de A: `sqlc generate` idempotente (segunda execucao sem diff) **e** lint verde.
Se o lint reprovar por `max_seq`, A **nao deve ser forcado**: a correcao certa e configurar exclusao
de `pkg/db/generated/` no linter, que e mudanca de outro dono.

---

## 3. COMMIT B - `feat(usage): persist thinking_level on task_usage (ORQ-13 phase 1)`

### 3.1 Inventario de arquivo e hunk

**B1. `server/migrations/<N>_task_usage_thinking_level.up.sql`** - NOVO, 21 linhas.
Conteudo efetivo: `ALTER TABLE task_usage ADD COLUMN IF NOT EXISTS thinking_level TEXT;` +
`COMMENT ON COLUMN`. Nullable, sem `NOT NULL`, sem `DEFAULT`, sem indice, `UNIQUE` de 032 intocada.

**B2. `server/migrations/<N>_task_usage_thinking_level.down.sql`** - NOVO, 8 linhas.
`ALTER TABLE task_usage DROP COLUMN IF EXISTS thinking_level;`

**B3. `server/pkg/db/queries/task_usage.sql`** - 2 hunks, ambos em B:
- hunk 1: comentario explicando o `COALESCE` no conflito;
- hunk 2: `INSERT` ganha `thinking_level` e `$8`; `DO UPDATE SET` ganha
  `thinking_level = COALESCE(EXCLUDED.thinking_level, task_usage.thinking_level)`.

**B4. `server/pkg/db/generated/task_usage.sql.go`** - 7 hunks, **todos em B**:

| hunk | conteudo |
|---|---|
| `@@ -48 +48 @@` | `getTaskUsage` SELECT passa a listar `thinking_level` |
| `@@ -72,0 +73 @@` | `Scan` ganha `&i.ThinkingLevel` |
| `@@ -397,2 +398,2 @@` | `INSERT` com a coluna e `$8` |
| `@@ -404,0 +406 @@` | `DO UPDATE SET` com `COALESCE` |
| `@@ -415,0 +418 @@` | `UpsertTaskUsageParams` ganha `ThinkingLevel pgtype.Text` |
| `@@ -421,0 +425,3 @@` | comentario propagado do `.sql` |
| `@@ -430,0 +437 @@` | argumento `arg.ThinkingLevel` no `Exec` |

**B5. `server/pkg/db/generated/models.go`** - **1 hunk** de 6 (`@@ -706,2 +758,4 @@`):
`TaskUsage` ganha comentario + `ThinkingLevel pgtype.Text`.

**B6. `server/internal/daemon/types.go`** - 1 hunk `@@ -168,0 +169,20 @@`:
`TaskUsageEntry` ganha `ThinkingLevel string` com `omitempty`, mais a funcao
`usageThinkingLevelFor(task Task) string`.

**B7. `server/internal/daemon/daemon.go`** - 2 hunks:
`@@ -3856,0 +3857,5 @@` (comentario + `usageThinkingLevel := usageThinkingLevelFor(task)`) e
`@@ -3868,0 +3874 @@` (`ThinkingLevel: usageThinkingLevel` no literal de `TaskUsageEntry`).

**B8. `server/internal/handler/daemon.go`** - 3 hunks:
`@@ -2061,0 +2062,16 @@` (`TaskUsagePayload.ThinkingLevel` + `thinkingLevelText`),
`@@ -2100,0 +2117,5 @@` (comentario + `thinkingLevel := strings.TrimSpace(...)`),
`@@ -2108,0 +2130 @@` (`ThinkingLevel: thinkingLevelText(thinkingLevel)` no upsert).

**B9. `server/internal/daemon/task_usage_thinking_level_test.go`** - NOVO, 120 linhas, 5 testes.
**B10. `server/internal/handler/task_usage_thinking_level_test.go`** - NOVO, 94 linhas, 4 testes.

### 3.2 Gate independente do commit B

Assume A ja aplicado (B5 e B4 sao saida do gerador **sobre** o baseline de A):
```bash
cd multica-auth-work/server
sqlc generate && git diff --exit-code pkg/db/generated/   # B4/B5 devem ser exatamente a saida
go build ./...                                            # esperado: exit 0
go vet ./internal/daemon/... ./internal/handler/...        # esperado: exit 0
gofmt -l internal/daemon/types.go internal/daemon/daemon.go internal/handler/daemon.go \
        internal/daemon/task_usage_thinking_level_test.go \
        internal/handler/task_usage_thinking_level_test.go   # esperado: vazio
go test ./internal/daemon -run 'ThinkingLevel|TaskUsageEntry' -count=1 -v   # 5 PASS
go test ./internal/handler -run 'ThinkingLevel|TaskUsagePayload' -count=1   # PASS
go test ./internal/daemon ./internal/handler -count=1        # regressao completa
git diff --check                                            # esperado: exit 0
```
Resultado ja medido na entrega anterior (com A e B juntos): build 0, vet 0, gofmt limpo,
diff-check 0, testes novos 5+4 PASS, regressao `ok 177.896s` e `ok 0.104s`.
**Nao reexecutei nesta rodada** - o worktree esta congelado e nada foi alterado.

Gate de banco, **nao executado e nao autorizado**: aplicar a migration e depois provar
`INSERT`/`UPSERT` com tier vazio e com tier preenchido. Enquanto a migration nao rodar, o
`UpsertTaskUsage` de B falha contra banco sem a coluna. Ordem obrigatoria: **migration antes do
binario**.

---

## 4. NUMERO DE MIGRATION - NAO ASSUMIR ATE RESERVA

Medido em `server/migrations/`: existem **124** (`approved_accounts`), **125**
(`user_password_credential`) e **126** (`runtime_profile_protocol_family_native_runtimes`). O proximo
livre e **127**, e os dois arquivos que eu criei ja usam `127`.

**Isso e uma suposicao minha e deve ser tratada como pendente.** Nao reservei o numero e nao ha
mecanismo de reserva visivel no repo. Risco real: outro agente com worktree paralelo tambem escolher
127, e o conflito so aparecer no merge. Portanto os nomes em B1/B2 estao grafados como
`<N>_task_usage_thinking_level.{up,down}.sql` neste manifesto, e o rename para o numero **reservado**
e passo obrigatorio antes do commit B. Nao renomeei nesta rodada por ser edicao.

---

## 5. RECONFIRMACAO DE VAZAMENTO: account_id / pricing / rollup

### 5.1 Commit B esta limpo
`grep -nEi 'account_id|AccountID|price|pricing|rollup|hourly|daily'` nos arquivos de B retorna
**somente comentario explicativo**, nunca codigo ou coluna:
- `internal/handler/..._test.go:9` - "so a **pricing** lookup can tell a missing tier apart";
- `migrations/<N>_..._up.sql:14` - "**Rollups** (073/084/101/102) are intentionally NOT touched";
- `migrations/<N>_..._down.sql:5` - "token counts and **rollups** survive the rollback";
- `pkg/db/queries/task_usage.sql:5` - comentario **pre-existente** sobre o worker de rollup.

As demais ocorrencias em `pkg/db/queries/task_usage.sql` (linhas 36-113: `ListDashboardUsageDaily`,
`task_usage_hourly`, etc.) sao **codigo pre-existente do arquivo**, fora dos 2 hunks de B3.
Confirmado: **nenhuma coluna `account_id`, nenhuma tabela de preco, nenhuma alteracao nas 4 tabelas
de rollup**.

### 5.2 Commit A carrega tipos de conta, mas sem uso
A introduz `Account`, `ApprovedAccount`, `Assignment`, `Credential`, `RotationEvent`. Sao **structs
gerados** a partir de migrations que **ja estao no repo**; A nao cria schema.
Verificacao de uso:
```
grep -rIn 'db\.Account\b|db\.Assignment\b|db\.ApprovedAccount\b|db\.Credential\b|db\.RotationEvent\b' --include='*.go' . | grep -v '/generated/'
(vazio)
```
**Zero referencias** fora do proprio generated. Portanto A nao habilita nem exercita nada de
`account_id`; apenas alinha o generated ao schema existente. Ainda assim, por tocar a vizinhanca
semantica que o escopo do ORQ-13 excluiu, A deve ser um commit **separado e rotulado como chore**,
para que o reviewer nao confunda com feature.

---

## 6. FILES_LOCKED por commit

**Commit A** (2 arquivos, exclusivos):
```
multica-auth-work/server/pkg/db/generated/models.go            (5 dos 6 hunks)
multica-auth-work/server/pkg/db/generated/task_message.sql.go  (2 hunks)
```
**Commit B** (5 modificados + 4 novos):
```
multica-auth-work/server/migrations/<N>_task_usage_thinking_level.up.sql    (novo)
multica-auth-work/server/migrations/<N>_task_usage_thinking_level.down.sql  (novo)
multica-auth-work/server/pkg/db/queries/task_usage.sql                      (2 hunks)
multica-auth-work/server/pkg/db/generated/task_usage.sql.go                 (7 hunks)
multica-auth-work/server/pkg/db/generated/models.go                         (1 hunk — compartilhado com A)
multica-auth-work/server/internal/daemon/types.go                           (1 hunk)
multica-auth-work/server/internal/daemon/daemon.go                          (2 hunks)
multica-auth-work/server/internal/handler/daemon.go                         (3 hunks)
multica-auth-work/server/internal/daemon/task_usage_thinking_level_test.go  (novo)
multica-auth-work/server/internal/handler/task_usage_thinking_level_test.go (novo)
```
**`models.go` e o unico arquivo compartilhado** entre A e B, e por isso o split precisa ser por
**hunk**, nao por arquivo. Se A e B forem separados por arquivo, `models.go` inteiro cai em um dos
dois e o split falha no proposito.

Fora do lock, exige outro cartao: `package.json`, `pnpm-lock.yaml`, `playwright.config.ts`,
qualquer arquivo de rollup, qualquer arquivo de pricing, e a configuracao do linter (relevante para
o risco de 2.2).

Non-overlap: nao toco `packages/**`, `apps/**`, nem o plano de browser QA (ORQ-39, congelado).

---

## 7. NAO-AFIRMACOES
- READ-ONLY: **nao editei, nao fiz stage, nao commitei, nao rodei migration, nao rodei `sqlc
  generate` nesta rodada, nao dei push, nao abri PR.** O worktree segue com 0 commits proprios.
- Nao reexecutei os gates nesta rodada; os resultados citados na secao 3.2 sao da entrega anterior,
  com A e B **juntos**. **Os gates independentes de A e de B, separadamente, nunca foram executados**
  - e essa e a lacuna que o executor tem de fechar.
- Nao provei se o commit A quebra o lint por causa de `max_seq`. Nao existe `.golangci` no repo e o
  CI usa defaults do v2.12; nao instalei `golangci-lint`. **Risco aberto.**
- Nao sei por que `task_message.sql.go` foi editado a mao; inferi da direcao do rename
  (`max_seq` e o que o gerador emite a partir do alias SQL), nao de historico de commit.
- Nao reservei o numero de migration e nao renomeei os arquivos: `127` e apenas o proximo livre
  observado, nao um numero reservado.
- Nao verifiquei se algum outro worktree ativo tambem usa `127`.
- Nao apliquei a migration, logo nao provei o comportamento do `COALESCE` contra Postgres real.
- Nao criei, atribui nem comentei issue alguma; o freeze de assignment/comentario permanece
  respeitado integralmente.
- Nao toquei no plano de browser QA congelado (ORQ-39, sha256 `9a4f5141...95b9`).
