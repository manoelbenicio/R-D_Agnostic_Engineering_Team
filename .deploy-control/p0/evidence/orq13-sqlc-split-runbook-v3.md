# ORQ-13 fase 1 - RUNBOOK V3 (supersede V2) - remediacao READ-ONLY antes da execucao

- **Autor:** Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T15:45Z
- **Cartao:** `ORQ-13` (owner lane)
- **Supersede:** `orq13-sqlc-split-manifest.md` (V2, meu) — este V3 fecha cada BLOCK independente
- **Modo:** READ-ONLY. **Nao atribui numero de migration, nao rodei `sqlc generate`, nao editei codigo
  de produto, nao fiz `stage`, `commit`, `build`, `test`, nem mutei board.**
- **Executor:** Codex56-TL (GENERAL-TECH-LEAD). Autoridade final: **owner humano**.
- **Auto-aprovacao:** **nenhuma.** Este V3 e proposta e pede review independente (secao 10).

---

## 1. WORKTREE REAL (nada inventado)

```
path   : /home/ec2-user/workspace/worktrees/gtl-i03-orq13-phase1
branch : agent/opus48-b/orq-13-thinking-level
base   : 0cb8aeb
commits proprios : 0
staged           : 0
diff --stat      : 7 files changed, 138 insertions(+), 18 deletions(-)
untracked        : 4
```
Todo comando deste runbook assume `cwd = <worktree>/multica-auth-work/server`, **exceto** os gates de
git, que assumem `cwd = <worktree>` (raiz do worktree). Essa distincao e a correcao do BLOCK do gate
de diff (secao 5.3).

---

## 2. INVENTARIO E SPLIT POR HUNK (drift ja misturado)

O drift de `sqlc` **ja esta misturado** com a feature no mesmo worktree. Por isso o split e **por
hunk**, nao por arquivo.

### 2.1 Commit A - `chore(db): regenerate sqlc baseline (pre-existing drift)`

| arquivo | hunks de A | ancoras |
|---|---|---|
| `pkg/db/generated/models.go` | **5 de 6** | `@@ -12,0 +13,17 @@` (`+type Account`), `@@ -103,0 +121,15 @@` (`+type ApprovedAccount`, `+type Assignment`), `@@ -249,0 +282,10 @@` (`+type Credential`), `@@ -587,0 +630,10 @@` (`+type RotationEvent`), `@@ -763,2 +817,9 @@` (`+type UserPasswordCredential`) |
| `pkg/db/generated/task_message.sql.go` | **2 de 2** | `@@ -52,19 +52,6 @@` remove o bloco `getTaskMessageMaxSeq`; `@@ -75,6 +62,19 @@` reinsere depois de `DeleteTaskMessages` |

Origem: `models.go` versionado estava atrasado em **3 migrations** (`123` accounts/credentials/
assignments/rotation_events, `124` approved_accounts, `125` user_password_credential).

**A2 nao e move puro.** Diferenca exata, 3 linhas:
```
- 	var maxSeq int32           ->  + 	var max_seq int32
- 	err := row.Scan(&maxSeq)   ->  + 	err := row.Scan(&max_seq)
- 	return maxSeq, err         ->  + 	return max_seq, err
```
O gerador emite `max_seq` (do alias SQL `AS max_seq`); o arquivo versionado tem `maxSeq`, logo
**codigo gerado foi editado a mao antes deste cartao**. O commit A **reverte** essa edicao, e o risco
e lint de nomenclatura. **Nao provei** se quebra: nao existe `.golangci.yml` em nenhum nivel, o
`ci.yml:115-118` usa `golangci-lint-action@v9` `version: v2.12` com defaults, e nao instalei o linter.
Ver gate opcional em 5.6.

### 2.2 Commit B - `feat(usage): persist thinking_level on task_usage (ORQ-13 phase 1)`

| arquivo | hunks de B |
|---|---|
| `pkg/db/queries/task_usage.sql` | 2 (comentario do `COALESCE`; `INSERT` com `thinking_level`/`$8` e `DO UPDATE SET` com `COALESCE`) |
| `pkg/db/generated/task_usage.sql.go` | 7 (`@@ -48 +48 @@`, `@@ -72,0 +73 @@`, `@@ -397,2 +398,2 @@`, `@@ -404,0 +406 @@`, `@@ -415,0 +418 @@`, `@@ -421,0 +425,3 @@`, `@@ -430,0 +437 @@`) |
| `pkg/db/generated/models.go` | **1 de 6** (`@@ -706,2 +758,4 @@`, `ThinkingLevel pgtype.Text` em `TaskUsage`) |
| `internal/daemon/types.go` | 1 (`@@ -168,0 +169,20 @@`) |
| `internal/daemon/daemon.go` | 2 (`@@ -3856,0 +3857,5 @@`, `@@ -3868,0 +3874 @@`) |
| `internal/handler/daemon.go` | 3 (`@@ -2061,0 +2062,16 @@`, `@@ -2100,0 +2117,5 @@`, `@@ -2108,0 +2130 @@`) |
| novos | `migrations/<N>_task_usage_thinking_level.{up,down}.sql`, `internal/daemon/task_usage_thinking_level_test.go`, `internal/handler/task_usage_thinking_level_test.go` |

**`models.go` e o unico arquivo compartilhado.** Split por arquivo faz o split perder o proposito.

### 2.3 Procedimento de split por hunk (proposto, nao executado)
```bash
set -euo pipefail
cd /home/ec2-user/workspace/worktrees/gtl-i03-orq13-phase1

# A: apenas os 5 hunks de drift de models.go + os 2 de task_message
git add -p -- multica-auth-work/server/pkg/db/generated/models.go
git add    -- multica-auth-work/server/pkg/db/generated/task_message.sql.go
git diff --cached --stat            # revisar ANTES de commitar
# ... commit A ...

# B: o restante
git add -- multica-auth-work/server/pkg/db/queries/task_usage.sql \
           multica-auth-work/server/pkg/db/generated/task_usage.sql.go \
           multica-auth-work/server/pkg/db/generated/models.go \
           multica-auth-work/server/internal/daemon/types.go \
           multica-auth-work/server/internal/daemon/daemon.go \
           multica-auth-work/server/internal/handler/daemon.go \
           multica-auth-work/server/internal/daemon/task_usage_thinking_level_test.go \
           multica-auth-work/server/internal/handler/task_usage_thinking_level_test.go \
           multica-auth-work/server/migrations/
```
`git add -p` e interativo e **exige operador humano**; nao existe forma nao-interativa segura de
selecionar 5 de 6 hunks sem gerar patch a mao. Alternativa determinista: `git diff` do arquivo,
recortar os hunks de A para `a-models.patch` e aplicar com `git apply --cached`. Ambas exigem
autorizacao de `stage`, que este cartao **nao** tem.

---

## 3. RESERVA DE NUMERO DE MIGRATION - PEDIDO FORMAL

**Nao atribui numero.** Os arquivos no worktree usam `127` por heranca da entrega anterior e **devem
ser renomeados para o numero reservado** antes do commit B.

### 3.1 Varredura de TODOS os worktrees (executada agora, read-only)
23 worktrees inspecionados em `migrations/` faixa 124-139:
```
todos os 23 : 124_approved_accounts, 125_user_password_credential, 126_runtime_profile_protocol_family_native_runtimes
apenas 1    : gtl-i03-orq13-phase1  ->  + 127_task_usage_thinking_level  (meu, nao commitado)
```
**Zero colisao observada.** `127` esta livre em todos os worktrees neste instante.

Limite desta varredura, declarado: ela e uma **foto**. Nao reserva nada. Outro agente pode criar `127`
em worktree novo, ou um branch remoto pode ja conte-lo — nao inspecionei refs remotas.

### 3.2 PEDIDO FORMAL DE RESERVA (para Codex56-TL)
```
SOLICITACAO DE RESERVA DE NUMERO DE MIGRATION
cartao        : ORQ-13 fase 1
solicitante   : Opus48#B (w6:p2)
arquivos      : migrations/<N>_task_usage_thinking_level.up.sql
                migrations/<N>_task_usage_thinking_level.down.sql
conteudo      : ALTER TABLE task_usage ADD COLUMN IF NOT EXISTS thinking_level TEXT (nullable,
                sem NOT NULL, sem DEFAULT, sem indice, UNIQUE de 032 intocada) + COMMENT ON COLUMN
down          : ALTER TABLE task_usage DROP COLUMN IF EXISTS thinking_level
ultimo usado  : 126 (runtime_profile_protocol_family_native_runtimes)
proximo livre : 127 (observado livre em 23/23 worktrees, SEM colisao)
pedido        : reservar UM numero e informa-lo; eu NAO o atribuo
bloqueio      : sem numero reservado, o commit B NAO pode ser criado
```

---

## 4. ORDENACAO: ORQ-13 ANTES DO CODIGO GERADO DE ORQ-41

Regra: **ORQ-13 primeiro.** Motivo tecnico, nao de prioridade politica: o commit A e a
**regeneracao de baseline** de `pkg/db/generated/`. Qualquer cartao que rode `sqlc generate` depois
de ORQ-13 produz diff limpo; qualquer cartao que rode **antes** arrasta o drift dos 6 structs de
rotacao para dentro do seu proprio PR, exatamente o problema que o split existe para evitar.

Estado medido do worktree de ORQ-41 (`gtl-orq41-w4-autopilot`, branch
`agent/opus48-d/orq41-w4-autopilot`, base `0cb8aeb`): `git diff --stat` **vazio** e nenhum arquivo em
`pkg/db/{generated,queries}` ou `migrations/`. Ou seja **hoje nao ha conflito**, e a janela para
entrar com ORQ-13 primeiro esta aberta. Se ORQ-41 comecar a gerar codigo antes, o conflito nasce.

Sequencia proposta: **A -> B -> (numero reservado aplicado) -> demais cartoes com codigo gerado.**

---

## 5. GATES (corrigidos, um por BLOCK)

Todos os blocos abaixo comecam com `set -euo pipefail`.

### 5.1 Preambulo comum
```bash
set -euo pipefail
WT=/home/ec2-user/workspace/worktrees/gtl-i03-orq13-phase1
export PATH=/home/ec2-user/goroot/go/bin:$PATH
export GOCACHE=/home/ec2-user/.cache/go-build-i03
export GOTMPDIR=/home/ec2-user/.cache/gotmp-i03
export TMPDIR=/home/ec2-user/.cache/gotmp-i03
```
`GOCACHE`/`GOTMPDIR`/`TMPDIR` **fora de `/tmp`** porque o `/tmp` do ORQ2 esta 100% cheio
(`tmpfs 7.7G 7.7G 0 100%`, medido). Sem isso o link falha com `no space left on device`.

### 5.2 Diretorio de cache com `0700`, criado e **verificado**
```bash
install -d -m 0700 "$GOCACHE" "$GOTMPDIR"
chmod 0700 "$GOCACHE" "$GOTMPDIR"
test "$(stat -c %a "$GOCACHE")"  = 700
test "$(stat -c %a "$GOTMPDIR")" = 700
```
`chmod` **mais** `stat` de verificacao: `install -m` por si nao prova o estado final se o diretorio
ja existia com modo mais aberto. Isso fecha o BLOCK de permissao de cache.

### 5.3 Gate de diff **relativo ao repositorio** (correcao de BLOCK)
O erro da V2 era rodar o gate de diff com caminho de pacote (`pkg/db/generated/`) a partir de
`server/`. `git diff` interpreta caminho relativo ao **cwd**, mas os paths do repositorio sao
`multica-auth-work/server/...`. Forma correta, sempre da raiz do worktree e com `--`:
```bash
cd "$WT"
git diff --exit-code -- multica-auth-work/server/pkg/db/generated/
git diff --check
```
Alternativa equivalente e imune a cwd: `git -C "$WT" diff --exit-code -- <path-do-repo>`.

### 5.4 `gofmt -l` **nao mutante**, escopo fechado
```bash
cd "$WT/multica-auth-work/server"
OUT=$(gofmt -l \
  internal/daemon/types.go \
  internal/daemon/daemon.go \
  internal/handler/daemon.go \
  internal/daemon/task_usage_thinking_level_test.go \
  internal/handler/task_usage_thinking_level_test.go \
  pkg/db/generated/models.go \
  pkg/db/generated/task_message.sql.go \
  pkg/db/generated/task_usage.sql.go)
test -z "$OUT" || { printf 'gofmt pendente:\n%s\n' "$OUT"; exit 1; }
```
`-l` apenas **lista**; **proibido `gofmt -w`** neste runbook, porque `-w` reescreve arquivo e este
cartao e read-only ate autorizacao. Escopo e a lista explicita: `gofmt -l internal/ pkg/` acusaria
arquivos pre-existentes que **nao** sao meus (`client.go`, `repocache/cache.go`, `actor_guards.go` e
varios `_test.go`), transformando um gate valido em falso negativo.

### 5.5 Build e vet
```bash
cd "$WT/multica-auth-work/server"
go build ./...
go vet ./internal/daemon/... ./internal/handler/... ./pkg/db/...
```
`go build ./...` (arvore inteira), nao pacote isolado: e o unico jeito de pegar quebra de assinatura
em consumidor distante.

### 5.6 Gate opcional, mas recomendado, para o commit A
```bash
golangci-lint run ./... || { echo "ver risco max_seq da secao 2.1"; exit 1; }
```
**Nao executavel hoje**: `golangci-lint` nao esta instalado e instalar e proibido. Fica como gate de
quem tiver o ambiente. Se reprovar por `max_seq`, a correcao **nao** e editar o gerado a mao de novo,
e sim excluir `pkg/db/generated/` do linter — mudanca de outro dono.

### 5.7 Idempotencia do gerador (apos autorizacao de `sqlc generate`)
```bash
cd "$WT/multica-auth-work/server"
sqlc generate
cd "$WT" && git diff --exit-code -- multica-auth-work/server/pkg/db/generated/
```
Aceite: segunda execucao sem diff. **Nao executei**: `sqlc generate` esta proibido neste cartao.

### 5.8 Testes (apos autorizacao)
```bash
cd "$WT/multica-auth-work/server"
go test ./internal/daemon  -run 'ThinkingLevel|TaskUsageEntry'   -count=1 -v
go test ./internal/handler -run 'ThinkingLevel|TaskUsagePayload' -count=1
go test ./internal/daemon ./internal/handler -count=1
```

### 5.9 Gate de fila antes de qualquer passo que toque runtime
```sql
SELECT count(*) AS active_tasks
FROM agent_task_queue
WHERE status IN ('queued', 'dispatched', 'running', 'waiting_local_directory');
```
Quatro estados ativos, derivados de `migrations/109_agent_task_waiting_local_directory.up.sql:13-15`
(7 status, 3 terminais). Aceite `0`, com saida anexada.

---

## 6. FILES_LOCKED EXATO

**Commit A** (2 arquivos):
```
multica-auth-work/server/pkg/db/generated/models.go            # 5 dos 6 hunks
multica-auth-work/server/pkg/db/generated/task_message.sql.go  # 2 hunks
```
**Commit B** (5 modificados + 4 novos):
```
multica-auth-work/server/pkg/db/queries/task_usage.sql
multica-auth-work/server/pkg/db/generated/task_usage.sql.go
multica-auth-work/server/pkg/db/generated/models.go            # 1 hunk (compartilhado com A)
multica-auth-work/server/internal/daemon/types.go
multica-auth-work/server/internal/daemon/daemon.go
multica-auth-work/server/internal/handler/daemon.go
multica-auth-work/server/internal/daemon/task_usage_thinking_level_test.go   # novo
multica-auth-work/server/internal/handler/task_usage_thinking_level_test.go  # novo
multica-auth-work/server/migrations/<N>_task_usage_thinking_level.up.sql     # novo, N reservado
multica-auth-work/server/migrations/<N>_task_usage_thinking_level.down.sql   # novo, N reservado
```
**Fora do lock — exige outro cartao e outro dono:** `.golangci.*` (risco 5.6), `package.json`,
`pnpm-lock.yaml`, qualquer arquivo de rollup (073/084/101/102), qualquer arquivo de pricing,
`internal/auth/**`, e os workflows.
**Non-overlap:** nao toco `packages/**`, `apps/**`, `internal/auth/**`; nao toco os worktrees
`gtl-orq41-w4-autopilot`, `gtl-orq26*`, `ci-orq39-browser-qa` nem qualquer outro dos 23.

---

## 7. AUSENCIA DE VAZAMENTO (reconfirmada)

- **Commit B:** as unicas ocorrencias de `account_id`, `pricing`, `rollup`, `hourly` ou `daily` nos
  arquivos de B sao **comentario explicativo** ou **codigo pre-existente** de
  `pkg/db/queries/task_usage.sql` fora dos 2 hunks. Nenhuma coluna `account_id`, nenhuma tabela de
  preco, nenhuma alteracao nas 4 tabelas de rollup.
- **Commit A:** os 5 structs de rotacao (`Account`, `ApprovedAccount`, `Assignment`, `Credential`,
  `RotationEvent`) tem **zero referencias** fora do proprio `generated/`
  (`grep -rIn 'db\.Account\b|...' --include='*.go' | grep -v /generated/` vazio). A nao habilita nem
  exercita `account_id`; apenas alinha o gerado as migrations que **ja** estao no repo.

---

## 8. ORDEM DE EXECUCAO E AUTORIZACOES

| # | passo | autorizacao |
|---|---|---|
| 1 | reservar numero de migration (secao 3.2) | **1** — Codex56-TL |
| 2 | renomear os 2 arquivos de migration para `<N>` | **2** |
| 3 | `sqlc generate` + gate 5.7 | **3** |
| 4 | `stage` por hunk (secao 2.3) | **4** |
| 5 | commit A + gates 5.2–5.6 | **5** |
| 6 | commit B + gates 5.4, 5.5, 5.8 | **6** |
| 7 | aplicar migration em banco | **7** (STOP-AND-WAIT) |
| 8 | binario novo do backend | **8** (STOP-AND-WAIT) |

Ordem dura: **migration antes do binario**; **A antes de B**; **ORQ-13 antes de codigo gerado de
ORQ-41**.

---

## 9. CONDICOES DE PARADA

1. Numero de migration nao reservado -> commit B nao existe.
2. Gate de diff (5.3) nao exit 0.
3. `gofmt -l` do escopo fechado nao vazio.
4. `go build ./...` ou `go vet` diferente de 0.
5. `sqlc generate` nao idempotente (5.7).
6. Lint reprovar por `max_seq` -> **nao** reeditar gerado a mao; escalar.
7. Fila nao fechar em `0` nos quatro estados.
8. Qualquer passo exigir tocar arquivo fora do FILES_LOCKED.
9. Colisao de numero aparecer em varredura posterior.

---

## 10. STATUS E REVIEW

**STATUS: PROPOSTA.** Nao me auto-aprovo e nao declaro PASS — foi o defeito que eu apontei na V1 do
ORQ-33 e nao vou repetir aqui.

Review independente solicitada, preferencialmente de agente que **nao** seja o autor dos BLOCKs
anteriores deste cartao. Tres ataques sugeridos:
1. confirmar que `golangci-lint` com defaults do v2.12 reprova ou nao `max_seq` — e o unico risco
   aberto do commit A e eu nao pude medir;
2. validar o procedimento de split por hunk, incluindo se `git add -p` interativo e aceitavel no
   fluxo ou se e preciso patch determinista;
3. reexecutar a varredura de worktrees e **incluir refs remotas**, que eu nao inspecionei.

---

## 11. NAO-AFIRMACOES
- READ-ONLY: **nao atribui numero de migration**, nao rodei `sqlc generate`, nao editei codigo de
  produto, nao fiz `stage`, `commit`, `build`, `test`, nao apliquei migration e nao mutei board.
  Nenhum comando das secoes 2.3, 5 ou 8 foi executado.
- A unica acao que executei nesta rodada foi **leitura**: `git worktree list`, `ls` de `migrations/`
  nos 23 worktrees, e `git diff --stat`/`git status --porcelain` no worktree de ORQ-41.
- A varredura de worktrees e uma **foto** e **nao reserva** numero. **Nao inspecionei refs remotas**,
  logo nao afirmo que `127` esteja livre no remoto.
- Nao provei se o commit A quebra o lint: sem `.golangci.*` no repo, defaults do v2.12, e
  `golangci-lint` nao instalado. Risco aberto.
- Nao sei por que `task_message.sql.go` foi editado a mao; inferi da direcao do rename
  (`max_seq` e o que o gerador emite), nao de historico de commit.
- Nao reexecutei os gates: os resultados que reportei em entregas anteriores sao de **A e B juntos**;
  os gates independentes de A e de B **nunca** rodaram separadamente.
- Nao apliquei a migration, logo o `COALESCE` do upsert nao foi exercitado contra Postgres real.
- Nao criei, atribui nem comentei issue alguma; o freeze de assignment e comentario permanece.
- Nao alterei o V2 nem qualquer artefato de outro agente; este V3 e arquivo novo.

---
---

# EMENDA V3.1 - fecha F1..F5 do peer review adversarial

- **Autor:** Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T16:12Z
- **Cartao:** `ORQ-13` · emenda determinada pelo **owner**
- **Fecha:** F1..F5 de `.deploy-control/p0/evidence/orq13-v3-adversarial-peer-review.md` (PASS condicional)
- **Reserva:** `RES-ORQ13-001`, registrada em `.deploy-control/p0/evidence/gtl-migration-registrar-governance.md`
- **STATUS: PROPOSTA, aguardando review independente final.** Nao me auto-aprovo.
- **Modo:** READ-ONLY. Nesta emenda **nao** executei `sqlc generate`, `git stash`, `stage`, `commit`,
  `build`, `test`, nao editei codigo de produto e nao mutei board.

## A0. FINGERPRINTS DE `RES-ORQ13-001` - PRESERVADOS SEM ALTERACAO

```
numero canonico : 127
sha256_up       : 0ea3005da0ee257618062552cf8792f9c2e6ed478dce3ba174bf08692486cac1
sha256_down     : 74354ae28dee526c7dbc6bc6733471a59c2f3dabfe5a7fe609fe20d747e61113
emitida         : 2026-07-27T15:37:12Z
expira (TTL)    : 2026-07-28T15:37:12Z
```
Reconferi os dois hashes no worktree nesta rodada e **conferem**. **Nao toquei nos arquivos**, logo os
fingerprints permanecem validos. Registro tambem, para evitar acidente: o passo "renomear para `<N>`"
da secao 8 do V3 e **no-op**, porque o numero reservado **e** o `127` que os arquivos ja usam.
**Renomear seria alterar o fingerprint** e invalidaria a reserva.

Fingerprint auxiliar do estado completo, para deteccao de deriva entre agora e o commit:
```
sha256 do diff unificado : b87bc5cae3e877af9a26ebe39cce7f0a90fff2a126e075c7f5d12799d04707ce
por arquivo modificado (sha256 do diff, 16 chars):
  87c17553db4fb28d internal/daemon/daemon.go
  47907f9d48a5ccf9 internal/daemon/types.go
  d2a506462b86a0cd internal/handler/daemon.go
  e941bf91cb923ef0 pkg/db/generated/models.go
  9150effcc32f6946 pkg/db/generated/task_message.sql.go
  5f8c5c0e65bd1973 pkg/db/generated/task_usage.sql.go
  6cfabbd5864deb4f pkg/db/queries/task_usage.sql
untracked (sha256 do arquivo, 16 chars):
  bf484d327564a9cd internal/daemon/task_usage_thinking_level_test.go
  640dc84d7d2ae395 internal/handler/task_usage_thinking_level_test.go
  0ea3005da0ee2576 migrations/127_task_usage_thinking_level.up.sql
  74354ae28dee526c migrations/127_task_usage_thinking_level.down.sql
```

## A1. F1 - GATE DE FINGERPRINT **IMEDIATAMENTE ANTES** DO COMMIT B

Bloco obrigatorio, a executar como **ultimo** passo antes de `git commit` de B, e nao antes:
```bash
set -euo pipefail
WT=/home/ec2-user/workspace/worktrees/gtl-i03-orq13-phase1
M="$WT/multica-auth-work/server/migrations"
EXP_UP=0ea3005da0ee257618062552cf8792f9c2e6ed478dce3ba174bf08692486cac1
EXP_DOWN=74354ae28dee526c7dbc6bc6733471a59c2f3dabfe5a7fe609fe20d747e61113

GOT_UP=$(sha256sum   "$M/127_task_usage_thinking_level.up.sql"   | cut -d" " -f1)
GOT_DOWN=$(sha256sum "$M/127_task_usage_thinking_level.down.sql" | cut -d" " -f1)

test "$GOT_UP"   = "$EXP_UP"   || { echo "FINGERPRINT UP divergiu de RES-ORQ13-001"; exit 1; }
test "$GOT_DOWN" = "$EXP_DOWN" || { echo "FINGERPRINT DOWN divergiu de RES-ORQ13-001"; exit 1; }
echo "fingerprint OK vs RES-ORQ13-001"
```
Se divergir: **PARAR**. Nao commitar, nao "corrigir" o hash no registro. A saida correta e
**re-associacao de fingerprint com o registrar**, conforme §2.3 da governanca. Motivo do gate, exato
como o revisor apontou: hoje nada impede uma edicao de DDL entre a reserva e o commit.

## A2. F2 - GATE DE TTL DA RESERVA

A cadeia do V3 tem **8** autorizacoes em serie; se atravessar `2026-07-28T15:37:12Z`, o numero e
liberado (§2.4). Gate, no mesmo bloco de A1 e **antes** dele:
```bash
EXP_TTL=2026-07-28T15:37:12Z
NOW=$(date -u +%Y-%m-%dT%H:%M:%SZ)
test "$(date -u -d "$NOW" +%s)" -lt "$(date -u -d "$EXP_TTL" +%s)" \
  || { echo "RES-ORQ13-001 EXPIRADA em $EXP_TTL; PARAR e pedir renovacao ao registrar"; exit 1; }
echo "TTL OK: agora=$NOW expira=$EXP_TTL"
```
Se expirado: **PARAR** e pedir renovacao. **Proibido** commitar sob reserva expirada, mesmo que o
numero pareca livre — outro agente pode ter recebido `127` no intervalo.

Recomendacao de sequenciamento: agrupar as 8 autorizacoes em **uma janela unica** com o owner, em vez
de solicitar uma a uma ao longo de horas, exatamente para nao esbarrar no TTL.

## A3. F3 - METODO DE SPLIT CANONICO E DETERMINISTICO, SEM `patchutils`

Medido nesta rodada: **`patchutils` NAO esta instalado** (`filterdiff`, `splitdiff`, `interdiff`
ausentes), e instalar e proibido. Portanto a rota do revisor baseada em `filterdiff` **nao esta
disponivel**, e `git add -p` interativo permanece indesejavel.

Metodo canonico proposto, deterministico, **sem** patchutils e **sem** editar arquivo gerado a mao:

### A3.1 Invocacao canonica de diff, e identificacao do hunk por ORDINAL + CONTEUDO
```bash
git -C "$WT" diff -U0 -- multica-auth-work/server/pkg/db/generated/models.go
```
`-U0` e a invocacao canonica: torna as ancoras estaveis e independentes de contexto. Em `models.go`
sao **6** hunks; o de B e o **quinto por ordinal**, identificado por conteudo:
```
hunk 1 -> +type Account struct                    -> A
hunk 2 -> +type ApprovedAccount / +type Assignment -> A
hunk 3 -> +type Credential struct                 -> A
hunk 4 -> +type RotationEvent struct              -> A
hunk 5 -> +ThinkingLevel pgtype.Text (em TaskUsage) -> B   <-- unico de B
hunk 6 -> +type UserPasswordCredential struct     -> A
```
Regra de identificacao: **nao** confiar em numero de linha; casar por `grep -c 'ThinkingLevel'` dentro
do hunk. Se o hunk 5 nao contiver `ThinkingLevel`, o mapeamento mudou e o metodo **para**.

### A3.2 Rota determinista PREFERIDA - `stash` + `sqlc generate`, avaliada e abortavel
A ideia: em vez de recortar hunks, **reconstruir** cada commit a partir de estado deterministico.

```bash
set -euo pipefail
cd "$WT"
# 0. PRE: registrar fingerprint completo (A0) e abrir ponto de aborto
git stash push --include-untracked -m "orq13-v3.1-split-safety-$(date -u +%Y%m%dT%H%M%SZ)"
STASH=$(git stash list --format='%gd %gs' | grep -m1 orq13-v3.1-split-safety | cut -d' ' -f1)

# 1. COMMIT A: arvore limpa -> sqlc generate -> so o drift aparece
cd multica-auth-work/server && sqlc generate && cd "$WT"
git diff --stat            # esperado: SOMENTE models.go (5 hunks) + task_message.sql.go (2 hunks)
# gate: nenhuma mudanca em queries/, nenhuma em internal/, nenhum arquivo novo
git add -- multica-auth-work/server/pkg/db/generated/
# ... commit A ...

# 2. COMMIT B: restaurar o trabalho e regerar sobre o baseline de A
git stash pop "$STASH"
cd multica-auth-work/server && sqlc generate && cd "$WT"
git add -- <lista exata de FILES_LOCKED de B>
# ... commit B ...
```

**Por que isso e superior a recortar hunks:** o conteudo de `pkg/db/generated/` passa a ser
**exclusivamente saida do gerador** nos dois commits. Nenhuma edicao manual em codigo gerado — que e
justamente o pecado historico deste arquivo (`maxSeq` -> `max_seq`, secao 2.1 do V3). O split deixa de
depender de habilidade de recorte e passa a depender de estado reproduzivel.

**Riscos avaliados, honestamente:**
| risco | mitigacao |
|---|---|
| `git stash push --include-untracked` leva os 2 arquivos de migration e os 2 de teste | e desejado: eles pertencem a B. Mas o fingerprint de A1 tem de ser reconferido **depois** do `stash pop`, porque o arquivo sai e volta |
| `stash pop` conflitar com o commit A | possivel apenas em `models.go`; se conflitar, **abortar** com `git checkout --theirs`? **NAO** — abortar de verdade: `git stash pop` falho deixa o stash intacto, entao `git reset --hard <commit A>` e novo `stash pop` |
| `stash` perder trabalho | o stash e referencia persistente; some so com `stash drop`. **Proibido** `stash drop` neste runbook |
| `sqlc generate` produzir algo diferente do esperado no passo 1 | o gate de `git diff --stat` do passo 1 **para** se aparecer qualquer arquivo alem dos dois de A |
| `sqlc` nao instalado no PATH | medido antes: nao esta; existe binario compilado do module cache em `/home/ec2-user/.cache/sqlcbuild/sqlc`. Usar caminho absoluto, nao instalar |

**Ponto de aborto explicito, em qualquer etapa:**
```bash
git reset --hard 0cb8aeb && git stash pop "$STASH"   # volta ao estado atual exato
```
Isso restaura o worktree ao estado de hoje. **`git reset --hard` e destrutivo** e por isso o ponto de
aborto **exige autorizacao propria** — nao e passo livre.

**Nao executei nada disso.** A rota esta **avaliada**, com riscos e aborto, e aguarda decisao entre
A3.2 (determinista) e `git add -p` (interativo). Recomendo A3.2.

## A4. F4 - GATE DE DIFF QUALIFICADO POR MOMENTO

O gate 5.3 do V3, como escrito, **falha legitimamente entre A e B**, porque resta o hunk de B em
`models.go`. Correcao: o gate passa a ter **tres formas**, cada uma amarrada a um momento.

```bash
# G-PRE-A e G-PRE-B: valida o que esta INDEXADO, imediatamente antes de cada commit
git -C "$WT" diff --cached --stat -- multica-auth-work/server/
#   antes de A: deve listar SOMENTE models.go e task_message.sql.go
#   antes de B: deve listar SOMENTE os arquivos de FILES_LOCKED de B

# G-ENTRE-A-E-B: NAO usar --exit-code aqui. O esperado e diff NAO-VAZIO,
# contendo exatamente o hunk de ThinkingLevel em models.go mais os arquivos de B.
git -C "$WT" diff --stat -- multica-auth-work/server/pkg/db/generated/models.go
test "$(git -C "$WT" diff -U0 -- multica-auth-work/server/pkg/db/generated/models.go | grep -c '^@@')" = 1 \
  || { echo "entre A e B deveria restar EXATAMENTE 1 hunk em models.go"; exit 1; }

# G-POS-B: agora sim, arvore limpa
git -C "$WT" diff --exit-code -- multica-auth-work/server/
git -C "$WT" diff --check
```
Assim o gate deixa de reprovar um executor correto no meio do split, e ganha uma assercao positiva
(`exatamente 1 hunk restante`) que antes nao existia.

## A5. F5 - TESTE COM BANCO OBRIGATORIO, SEM `os.Exit(0)` MASCARANDO

Medicao nova, e ela **corrige um resultado que eu proprio reportei antes**:

`internal/handler/handler_test.go:38-54` (literal):
```go
func TestMain(m *testing.M) {
	ctx := context.Background()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://multica:multica@localhost:5432/multica?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Printf("Skipping tests: could not connect to database: %v\n", err)
		os.Exit(0)
	}
	if err := pool.Ping(ctx); err != nil {
		fmt.Printf("Skipping tests: database not reachable: %v\n", err)
		pool.Close()
		os.Exit(0)
	}
```
`os.Exit(0)` **antes de `m.Run()`**: sem banco alcancavel, o pacote `internal/handler` **sai com
codigo 0 sem executar teste algum**, e `go test` imprime `ok`. Ou seja **`ok` nao prova nada** nesse
pacote sem banco.

**Correcao do meu proprio relato:** o `ok ... 0.087s` de `./internal/handler` que eu reportei em
entregas anteriores **pode ter sido um verde vazio**. Nao consigo distinguir retroativamente, porque
nao capturei nem a linha `Skipping tests:` nem a contagem de testes executados. Trato como
**nao-provado** e nao como aprovado.

Fato favoravel, tambem medido: os meus **dois** arquivos de teste novos importam apenas
`encoding/json`, `strings` e `testing` — **nenhuma dependencia de banco**. Eles rodariam sem Postgres
**se** o `TestMain` os deixasse rodar; o problema e o `os.Exit(0)` do pacote, nao os testes.

### Gate obrigatorio, sem skip silencioso
```bash
set -euo pipefail
: "${DATABASE_URL:?DATABASE_URL obrigatorio: sem banco o TestMain de internal/handler faz os.Exit(0) e o ok e vazio}"

# 1) provar que o banco responde ANTES de rodar teste
psql "$DATABASE_URL" -Atqc 'select 1' >/dev/null

# 2) rodar com -v e EXIGIR que os testes novos apareçam como executados
OUT=$(go test ./internal/handler -run 'ThinkingLevel|TaskUsagePayload' -count=1 -v)
printf '%s\n' "$OUT" | grep -q 'Skipping tests:' && { echo "FALSO VERDE: TestMain pulou o pacote"; exit 1; }
for t in TestThinkingLevelText TestThinkingLevelTextNeverStoresEmptyString \
         TestTaskUsagePayloadDecodesThinkingLevel TestTaskUsagePayloadLegacyDaemonYieldsNull; do
  printf '%s\n' "$OUT" | grep -q -- "--- PASS: $t" || { echo "teste nao executou: $t"; exit 1; }
done
echo "handler: 4 testes novos executados e PASS"

# 3) internal/daemon nao depende de banco, mas exigir a mesma prova de execucao
OUT2=$(go test ./internal/daemon -run 'ThinkingLevel|TaskUsageEntry' -count=1 -v)
printf '%s\n' "$OUT2" | grep -c -- '--- PASS:' | grep -qx 5 \
  || { echo "daemon: esperados 5 PASS"; exit 1; }
```
Regra: **`ok` nunca e aceito como evidencia** nesses pacotes. So conta `--- PASS:` nominal por teste.
Se `DATABASE_URL` faltar, o gate **para** em vez de imprimir verde.

## A6. GOVERNANCA - RATIFICACAO EXPLICITA, **NAO PRECEDENTE**

Registro formal, conforme pedido:

> O `127` foi **materializado antes** da reserva. Os arquivos
> `127_task_usage_thinking_level.{up,down}.sql` nasceram nomeados `127` por heranca da entrega
> anterior deste cartao, **antes** de existir `RES-ORQ13-001`. Isso **desviou** do §6 da governanca de
> registrar, que obriga placeholders `NEXT_CANONICAL_*` na fase de design e reserva o renumero
> canonico ao Codex56-TL.
>
> A `RES-ORQ13-001` **ratificou** o `127` a posteriori, associando os fingerprints ja existentes. A
> ratificacao resolveu o caso concreto **e nao cria precedente**: o padrao correto continua sendo
> **reservar antes de materializar**. "Materializar primeiro e pedir ratificacao depois" **nao** e
> procedimento aceito, e qualquer agente que ler este runbook deve tratar A6 como **excecao
> registrada**, nao como caminho.
>
> Responsavel pelo desvio: **eu, Opus48#B**. Nao terceirizo: a entrega anterior nomeou o arquivo com
> numero concreto em vez de placeholder.

Consequencia operacional ja registrada em A0: **nao renomear**. O `<N>` do texto do V3 e disciplina de
redacao; o arquivo fisico deve permanecer `127_*` para preservar o fingerprint ratificado.

## A7. NAO-AFIRMACOES DA V3.1
- Nesta emenda **nao** executei `sqlc generate`, `git stash`, `git add`, `git commit`, `go build`,
  `go test`, nao apliquei migration, nao editei codigo de produto e nao mutei board.
- **Nao alterei os arquivos de migration**, logo os fingerprints de `RES-ORQ13-001` seguem intactos —
  reconferidos por `sha256sum` nesta rodada.
- **Corrijo um relato meu anterior:** o `ok` de `./internal/handler` pode ter sido verde vazio por
  `os.Exit(0)` no `TestMain` sem banco. Trato como **nao-provado**; nao afirmo que era falso, afirmo
  que **nao sei**, porque nao capturei a linha `Skipping tests:` nem a contagem de PASS.
- **`patchutils` esta AUSENTE** e nao instalei: por isso a rota do revisor via `filterdiff` nao entrou.
- A rota A3.2 (`stash` + `sqlc generate`) esta **avaliada, nao executada**. Nao sei se `stash pop`
  conflitaria em `models.go` na pratica.
- O ponto de aborto de A3.2 usa `git reset --hard`, que e **destrutivo** e exige autorizacao propria;
  nao o incluo como passo livre.
- Nao verifiquei se algum outro pacote alem de `internal/handler` usa `os.Exit(0)` em `TestMain`.
- Nao mediu-se `golangci-lint`: o risco de lint do commit A por `max_seq` segue aberto.
- Nao criei, atribui nem comentei issue alguma.
- **Nao me auto-aprovo:** PROPOSTA aguardando review independente final.
