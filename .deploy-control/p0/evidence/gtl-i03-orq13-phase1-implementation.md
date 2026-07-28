# GTL-I03 - ORQ-13 fase 1: thinking_level nullable em task_usage + wiring E2E

Agente: Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T12:05Z
Modo: IMPLEMENTACAO em worktree isolado. **Nenhum commit, nenhum push, nenhuma migration aplicada,
nenhum restart, nenhuma mutacao de board, nenhum acesso live.** Sem pricing, sem valores monetarios,
sem `account_id`, sem alteracao de rollup.

## 1. WORKTREE

Criado por mim, limpo, a partir do HEAD de integracao. **Nao reutilizei o worktree rejeitado.**
```
$ git worktree add -b agent/opus48-b/orq-13-thinking-level \
    /home/ec2-user/workspace/worktrees/gtl-i03-orq13-phase1 0cb8aeb
HEAD is now at 0cb8aeb chore(p0): lane A check-out - record accidental amend of lane B commit

path   : /home/ec2-user/workspace/worktrees/gtl-i03-orq13-phase1
branch : agent/opus48-b/orq-13-thinking-level
base   : 0cb8aeb  (HEAD de integration/dev-transition-candidate-20260719)
estado inicial: git status --porcelain = 0 linhas (limpo)
```
Registro relevante: `ff121b28` **nao e um commit**, e o nome do diretorio de um worktree existente
(`/home/ec2-user/multica_workspaces/.../ff121b28/workdir/repo`, branch `agent/codex-a/orq-23`).
Isso fecha a duvida que levantei no GTL-37, onde `git log ff121b28` nao resolvia. Nao toquei nele.

## 2. DIFF

```
 internal/daemon/daemon.go                  |  6 +++
 internal/daemon/types.go                   | 20 +++++++
 internal/handler/daemon.go                 | 22 ++++++++
 pkg/db/generated/models.go                 | 61 ++++++++++++++++++++++
 pkg/db/generated/task_message.sql.go       | 26 ++++-----
 pkg/db/generated/task_usage.sql.go         | 13 +++--
 pkg/db/queries/task_usage.sql              |  8 ++-
 7 files changed, 138 insertions(+), 18 deletions(-)

novos (untracked):
 migrations/127_task_usage_thinking_level.up.sql
 migrations/127_task_usage_thinking_level.down.sql
 internal/daemon/task_usage_thinking_level_test.go
 internal/handler/task_usage_thinking_level_test.go
```

### 2.1 Migration 127, aditiva e nao destrutiva

`migrations/127_task_usage_thinking_level.up.sql`:
```sql
ALTER TABLE task_usage
    ADD COLUMN IF NOT EXISTS thinking_level TEXT;

COMMENT ON COLUMN task_usage.thinking_level IS
    'Reasoning tier declared by the agent config for this task (e.g. high, low, thinking). NULL = not declared by the reporting daemon. Never inferred from the model name.';
```
Decisoes deliberadas, todas documentadas no proprio arquivo:
- **nullable, sem `NOT NULL` e sem `DEFAULT`**. Um `DEFAULT ''` faria toda linha historica reivindicar
  o tier base, que e exatamente o falso-sucesso que essa coluna existe para eliminar. `NULL`
  significa "nao declarado", nunca "standard".
- **`UNIQUE (task_id, provider, model)` da migration 032 intocada.** Se o tier entrasse na chave, uma
  task poderia acumular duas linhas para o mesmo modelo e dobrar a contagem de tokens.
- **sem indice**: nada consulta por tier nesta fase.
- **rollups 073/084/101/102 intocados**: custo continua derivado na leitura.

`127_..._down.sql`: `ALTER TABLE task_usage DROP COLUMN IF EXISTS thinking_level;`
Descarta somente o dado que esta migration introduziu; todas as demais colunas e a UNIQUE sobrevivem.
Idempotente nos dois sentidos (`IF NOT EXISTS` / `IF EXISTS`).

### 2.2 Query + sqlc

`pkg/db/queries/task_usage.sql`, `UpsertTaskUsage`:
```sql
INSERT INTO task_usage (task_id, provider, model, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, thinking_level, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
ON CONFLICT (task_id, provider, model)
DO UPDATE SET
    ...
    thinking_level = COALESCE(EXCLUDED.thinking_level, task_usage.thinking_level),
    updated_at = now();
```
`COALESCE` no conflito e a decisao central: um re-report parcial (daemon antigo, ou correcao de
contagem) **nao pode apagar** o tier que produziu os tokens.

Generated regenerado **somente** via `sqlc generate` (v1.31.1). Resultado em `task_usage.sql.go`:
`UpsertTaskUsageParams` ganha `ThinkingLevel pgtype.Text`, o `SELECT *` de `GetTaskUsage` passa a
listar a coluna, e o `Scan` ganha `&i.ThinkingLevel`. Em `models.go`, `TaskUsage` ganha
`ThinkingLevel pgtype.Text`.

### 2.3 Wiring E2E (3 pontos)

`internal/daemon/types.go` — campo no contrato de wire, com `omitempty` para compatibilidade:
```go
	ThinkingLevel string `json:"thinking_level,omitempty"`
}

func usageThinkingLevelFor(task Task) string {
	if task.Agent == nil {
		return ""
	}
	return task.Agent.ThinkingLevel
}
```

`internal/daemon/daemon.go` (`runTask`, sitio de construcao das entradas de usage):
```go
	usageThinkingLevel := usageThinkingLevelFor(task)
	var usageEntries []TaskUsageEntry
	for model, u := range result.Usage {
		...
		usageEntries = append(usageEntries, TaskUsageEntry{
			...
			ThinkingLevel:    usageThinkingLevel,
		})
	}
```

`internal/handler/daemon.go` — payload e mapeamento para NULL:
```go
	ThinkingLevel string `json:"thinking_level"`
}

func thinkingLevelText(level string) pgtype.Text {
	trimmed := strings.TrimSpace(level)
	return pgtype.Text{String: trimmed, Valid: trimmed != ""}
}
```
e no upsert: `ThinkingLevel: thinkingLevelText(thinkingLevel)`.

**Invariante de projeto, e o ponto mais importante do patch:** o tier vem do registro do agente
(`task.Agent.ThinkingLevel`) e **nunca** do id do modelo. Sufixo `-high`/`-thinking` nao e sinal
confiavel — o OmniRoute tambem serve ids sem sufixo algum (`auto/best-coding`), logo inferir por
sufixo misatribuiria custo. Ha teste dedicado travando isso.

## 3. GATES

Todos executados no worktree, com `GOCACHE`/`GOTMPDIR`/`TMPDIR` fora do `/tmp` (ver gap 4).

| gate | comando | resultado |
|---|---|---|
| build | `go build ./...` | **exit 0** |
| vet | `go vet ./internal/daemon/... ./internal/handler/...` | **exit 0** |
| gofmt | `gofmt -l` nos 5 arquivos Go que toquei | **vazio (limpo)** |
| diff-check | `git diff --check` | **exit 0** |
| testes novos, daemon | `go test ./internal/daemon -run 'ThinkingLevel\|TaskUsageEntry' -count=1 -v` | **5 PASS, ok 0.023s** |
| testes novos, handler | `go test ./internal/handler -run 'ThinkingLevel\|TaskUsagePayload' -count=1` | **ok 0.087s** |
| regressao, suite completa | `go test ./internal/daemon ./internal/handler -count=1` | **ok 177.896s / ok 0.104s** |

Testes adicionados (9 casos + subtestes):
- `TestUsageThinkingLevelFor` (4 subtestes): sem agente, agente sem tier, tier verbatim, `thinking`.
- `TestUsageThinkingLevelIgnoresModelSuffix`: modelo `gemini-3.6-flash-high` **nao** produz tier.
- `TestTaskUsageEntryOmitsEmptyThinkingLevel`: tier vazio nao aparece no wire (compat com backend antigo).
- `TestTaskUsageEntryEmitsThinkingLevelWhenSet`.
- `TestTaskUsageEntryDecodesLegacyPayload`: payload pre-ORQ-13 decodifica e preserva os campos.
- `TestThinkingLevelText` (5 subtestes) e `TestThinkingLevelTextNeverStoresEmptyString`: blank -> NULL.
- `TestTaskUsagePayloadDecodesThinkingLevel` e `TestTaskUsagePayloadLegacyDaemonYieldsNull`:
  cobre a janela de versao mista.

## 4. GAPS E DESVIOS (declarados, nao escondidos)

**G1 - `FILES_LOCKED` cita `server/pkg/agent/types.go`, que nao existe.** `TokenUsage` esta em
`pkg/agent/agent.go:115`. Nao editei nenhum dos dois, e a razao e de projeto: o tier e atributo da
**configuracao da task**, nao da contagem de tokens que o CLI devolve. Colocar tier em
`agent.TokenUsage` obrigaria todos os clients de CLI a conhecer algo que eles nao observam. O campo
foi para `internal/daemon/types.go` (`TaskUsageEntry`), que e o struct de wire daemon->backend.
`internal/daemon/types.go` **nao** consta no `FILES_LOCKED`; e um desvio necessario e o unico
caminho para o campo cruzar o HTTP. Se o lock era literal, isso precisa de ruling.

**G2 - churn pre-existente no generated, nao causado por mim.** `sqlc generate` produziu, alem do meu
`task_usage`, seis structs novos em `models.go` (`Account`, `ApprovedAccount`, `Assignment`,
`Credential`, `RotationEvent`, `UserPasswordCredential`) e reordenou funcoes em
`task_message.sql.go` (26 linhas, puro reposicionamento de `GetTaskMessageMaxSeq`). Isso e **drift
pre-existente** entre o generated versionado e a saida do sqlc para o schema atual (migrations 123 e
124 ja existiam). Nao editei generated a mao — o dispatch exige `sqlc generate` e a saida do gerador
e o que esta no diff. Duas consequencias que voce precisa decidir:
1. os structs de `accounts`/`assignments` aparecem no meu diff, o que **toca a vizinhanca de
   `account_id`** que o escopo excluiu. Eu nao os usei em nenhuma linha de codigo.
2. o ideal seria um commit separado "regen sqlc baseline" antes do meu. Nao commitei nada, entao a
   divisao ainda e possivel.

**G3 - `sqlc` nao estava instalado.** `which sqlc` falhou. Instalar pacote e proibido, entao compilei
o `sqlc` v1.31.1 que **ja existia no module cache local**
(`/home/ec2-user/go/pkg/mod/github.com/sqlc-dev/sqlc@v1.31.1`), offline, com
`GOPROXY=file:///home/ec2-user/go/pkg/mod/cache/download`, para
`/home/ec2-user/.cache/sqlcbuild/sqlc`. **Nada foi instalado no sistema, nenhum PATH alterado,
nenhuma dependencia baixada da rede.** Se preferir, o binario pode ser descartado: e um arquivo em
diretorio de cache.

**G4 - `/tmp` esta 100% cheio no ORQ2** (`tmpfs 7.7G 7.7G 0 100%`). Isso quebrou build e link ate eu
apontar `GOCACHE`, `GOTMPDIR` e `TMPDIR` para `/home/ec2-user/.cache`. Nao e do meu escopo e nao
mexi, mas afeta qualquer agente que compile no ORQ2 e merece atencao — inclusive porque check-ins
antigos mostram lanes usando `GOCACHE` sob `/tmp`.

**G5 - migration nao aplicada.** Nao rodei `migrate`. Portanto o wiring esta correto no codigo mas o
`UpsertTaskUsage` falharia em runtime contra um banco sem a coluna. Aplicar a 127 e gate do owner, e
a ordem obrigatoria e **migration antes do binario novo**.

**G6 - `gofmt` do repo tem falhas pre-existentes.** `gofmt -l internal/daemon internal/handler pkg/db`
lista arquivos que eu nao toquei (`client.go`, `repocache/cache.go`, `actor_guards.go`, vários
`_test.go`). Confirmei que `internal/handler/daemon.go` **estava limpo no HEAD** e o deixei limpo:
o `gofmt` queria reescrever um `''` do meu comentario para caractere unicode, e eu reescrevi o
comentario em vez de aceitar o caractere.

**G7 - nao validei ponta a ponta com banco real.** Nenhum teste toca Postgres; a prova do
`COALESCE` e do NULL e por unidade e por leitura do SQL gerado, nao por execucao. Um teste de
integracao com banco fica para quem tiver gate de DB.

## 5. NAO-AFIRMACOES
- Nao commitei, nao dei push, nao criei PR, nao apliquei migration, nao reiniciei daemon ou
  container, nao toquei board nem issue, nao fiz chamada live de provedor.
- Nao toquei em pricing, valores monetarios, `account_id` como codigo, nem em nenhum dos rollups
  073/084/101/102.
- Nao reutilizei o worktree rejeitado `ff121b28` nem qualquer outro worktree existente.
- Nao editei arquivo gerado a mao: `pkg/db/generated/*` vem exclusivamente de `sqlc generate`.
- Nao instalei pacote: o `sqlc` foi compilado do module cache local, offline, para um diretorio de cache.
- Nao apliquei nem testei a coluna contra banco real; o `COALESCE` nao foi exercitado em Postgres.
- Nao alterei `pkg/agent/agent.go` nem criei `pkg/agent/types.go`: ver G1.
- Os arquivos modificados na arvore principal de integracao nao sao meus; trabalhei somente no
  worktree isolado.
