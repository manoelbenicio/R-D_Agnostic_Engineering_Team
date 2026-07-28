# GTL-19 - Ledger Duravel V2 - desenho corrigido (READ-ONLY, nao aplicado)

- autor: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-27T11:40Z
- corrige: `ledger-durable-design.md` (Antigravity w8:p2), incorporando o delta de `gtl-ledger-implementation-audit.md`
- escritor exclusivo / integrador: Codex56-TL (GENERAL-TECH-LEAD) w5:pC
- modo: leitura. Nenhum arquivo de codigo editado, nenhum build, deploy, restart, commit ou rerun.

---

## 0. ACHADO QUE PRECEDE TODO O RESTO - hoje TODO ledger nasce fail-closed

Antes de persistir qualquer coisa, isto tem de ser resolvido, senao a V2 piora o sistema.

```go
config.go:115  CommitLedgerHMACSecret  string // stable >=32-byte hex secret for pseudonymous tool tokens; empty disables ledger (fail-closed)
daemon.go:4144 func (d *Daemon) commitLedgerSecret() []byte {
daemon.go:4145 	raw := d.cfg.CommitLedgerHMACSecret
daemon.go:4146 	if raw == "" { return nil }
...
daemon.go:4181 	if len(hmacSecret) >= 32 {
daemon.go:4182 		taskLedger, ledgerErr = commitledger.New(...)
daemon.go:4190 	} else {
daemon.go:4191 		// No valid HMAC secret configured — fail closed
daemon.go:4192 		taskLedger = commitledger.NewFailClosed(taskID)
daemon.go:4193 	}
```
e o que `NewFailClosed` grava:
```go
ledger.go:170 func NewFailClosed(taskID string) *Ledger {
ledger.go:176 		summary:    DurableSummary{EverSaturated: true},
```

**PROVA de que o campo nunca e preenchido**: `grep -rn "CommitLedgerHMACSecret" --include=*.go .`
(sem testes) retorna **somente 2 linhas**: a declaracao em `config.go:115` e a leitura em
`daemon.go:4145`. Nao existe leitura de env, nao existe flag, nao existe atribuicao. E
`grep -rn "COMMIT_LEDGER" --include=*.go .` retorna **vazio**. Confirmado tambem no processo vivo do
ORQ1: nenhuma variavel de ambiente com `COMMIT` ou `LEDGER` no nome (leitura de `/proc/<pid>/environ`
filtrada por NOME; nenhum valor de segredo lido).

**Consequencia**: hoje `commitLedgerSecret()` sempre devolve nil, logo todo ledger e
`NewFailClosed`, logo `EverSaturated=true` para toda task, logo `BlocksReplay()` = true sempre.
Se a V2 persistir esse summary como desenhado, ela grava `ever_saturated=true` para TODAS as tasks
e o bloqueio passa a **sobreviver ao restart** - estritamente pior que hoje, onde o bloqueio ao
menos morre com o processo.

**Ordem obrigatoria**: P0 (secret) antes de P1 (persistencia). Ver secao 8.

---

## 1. CORRELACAO - PROVADA, e o contrato do codigo esta com a wording errada

Cadeia completa, verificada linha a linha:

| passo | codigo | valor |
|---|---|---|
| tipo | `types.go:42-43 type Task struct { ID string \`json:"id"\` ` | task ID e string |
| chamada | `daemon.go:3842 result, tools, err := d.executeAndDrainForTask(ctx, backend, prompt, execOpts, taskLog, task.ID)` | passa `task.ID` |
| assinatura | `daemon.go:4168 func (d *Daemon) executeAndDrainForTask(..., taskID string)` | mesmo valor |
| ledger | `daemon.go:4184 TaskID: taskID,` | mesmo valor |
| registry | `daemon.go:4195 d.commitLedgers.Register(taskID, taskLedger)` | **chave = task.ID cru** |
| gate | `task.go:1665 parentTaskID := util.UUIDToString(parent.ID)` / `1666 CheckOrAllow(s.ReplayGateHook, parentTaskID)` | **mesmo UUID cru** |

**Veredito**: as duas pontas usam o MESMO identificador, o UUID cru da task. Nao ha divergencia
de chave, e a pergunta que deixei aberta no GTL-02 esta respondida: `task_id UUID PRIMARY KEY` e a
PK correta e casa com a chave em memoria.

O que esta errado e a **wording do contrato**, em dois comentarios:
```go
replay_gate.go:106 	ledgers map[string]*Ledger // keyed by pseudonymous task correlation
replay_gate.go:169 // Uses pseudonymous correlation ID (not raw task ID).
```
Ambos sao falsos hoje. A pseudonimizacao existe, mas em outro nivel: e o **token de entrada**, HMAC
sobre o `call_id`:
```go
ledger.go:180 // TokenizeCallID produces a pseudonymous HMAC-SHA256 token from a raw call_id.
ledger.go:182 // This token is safe to log/emit in telemetry.
```
Ou seja: token de tool = pseudonimo; chave de task = UUID cru. **Correcao exigida na V2**: ajustar
os dois comentarios para refletir a verdade, senao um leitor futuro implementa hash na chave e
quebra o casamento provado acima. E `task.go:1668` loga `"parent_task_id", parentTaskID`, ou seja
o UUID cru ja vai para log hoje - se a politica for nao logar UUID cru, isso e um segundo item, e
NAO deve ser resolvido trocando a chave.

---

## 2. INTERFACE `ReplayGateChecker` - mudancas reais, com antes e depois

Estado atual, que impede a V2:
```go
replay_gate.go:152 type ReplayGateHook struct {
replay_gate.go:153 	Registry *LedgerRegistry
replay_gate.go:154 	Logger   *slog.Logger
replay_gate.go:155 }
replay_gate.go:200 func CheckOrAllow(hook *ReplayGateHook, correlationID string) error {
replay_gate.go:201 	if hook == nil {
replay_gate.go:202 		return fmt.Errorf("%w: replay gate hook not configured; fail closed", ErrReplayBlocked)
replay_gate.go:203 	}
replay_gate.go:204 	return hook.Check(correlationID)
task.go:42         	ReplayGateHook *commitledger.ReplayGateHook
```

### 2.1 `commitledger/replay_gate.go` - ADICIONAR interface, ALTERAR `CheckOrAllow`

ADICIONAR antes da linha 149:
```go
+// ReplayGateChecker is the seam the retry path consults. It exists so the
+// backend can supply a durable, database-backed checker while the daemon keeps
+// the in-process registry implementation. Implementations MUST fail closed.
+type ReplayGateChecker interface {
+	Check(correlationID string) error
+}
+
+// compile-time proof that the in-memory hook satisfies the seam.
+var _ ReplayGateChecker = (*ReplayGateHook)(nil)
```
ALTERAR 200-205 (assinatura, e SOMENTE ela):
```go
-func CheckOrAllow(hook *ReplayGateHook, correlationID string) error {
-	if hook == nil {
+func CheckOrAllow(hook ReplayGateChecker, correlationID string) error {
+	if hook == nil {
 		return fmt.Errorf("%w: replay gate hook not configured; fail closed", ErrReplayBlocked)
 	}
 	return hook.Check(correlationID)
 }
```
**ARMADILHA a documentar no proprio codigo**: com parametro de interface, um ponteiro tipado nil
(`var h *ReplayGateHook = nil` passado como interface) NAO satisfaz `hook == nil`. Nesse caso a
execucao cai em `hook.Check`, que ja e nil-safe por `replay_gate.go:174 if h == nil || h.Registry == nil`
e devolve bloqueio. Ou seja o fail-closed continua garantido pelos DOIS caminhos - mas isso precisa
de teste explicito (ver 7.5), porque e exatamente o tipo de regressao silenciosa que passa em review.

`ReplayGate`, `ReplayGateResult`, `LedgerRegistry`, `Check` e `BlocksReplay` ficam **inalterados**.

### 2.2 `service/task.go` - trocar o tipo do campo
```go
-42 	ReplayGateHook *commitledger.ReplayGateHook
+42 	ReplayGateHook commitledger.ReplayGateChecker
```
`task.go:1666` nao muda uma letra. `NewTaskService` (`task.go:127`) tambem nao: hoje ele nao passa
o hook, e continua nao passando - o wiring e em `main.go` (secao 4).

### 2.3 `service/replay_gate_db.go` - NOVO arquivo, o checker duravel
```go
+// DatabaseReplayGateChecker consults the durable ledger summary table. It is
+// the backend-side implementation of commitledger.ReplayGateChecker.
+//
+// INVARIANTE: apenas os 4 campos ever_* decidem o gate. TotalToolUseCount e
+// HighestSeqSeen sao auditoria e nunca sao lidos aqui. Telemetria (task_usage)
+// nao tem permissao de escrita nesta tabela.
+type DatabaseReplayGateChecker struct {
+	Queries *db.Queries
+	Logger  *slog.Logger
+}
+
+func (c *DatabaseReplayGateChecker) Check(correlationID string) error {
+	if c == nil || c.Queries == nil {
+		return fmt.Errorf("%w: no durable ledger store; fail closed", commitledger.ErrReplayBlocked)
+	}
+	taskID, err := util.StringToUUID(correlationID)   // conferir nome real do helper
+	if err != nil {
+		return fmt.Errorf("%w: unparseable correlation; fail closed", commitledger.ErrReplayBlocked)
+	}
+	row, err := c.Queries.GetTaskLedgerGate(context.Background(), taskID)
+	switch {
+	case errors.Is(err, pgx.ErrNoRows):
+		return fmt.Errorf("%w: no durable ledger row; fail closed", commitledger.ErrReplayBlocked)
+	case err != nil:
+		return fmt.Errorf("%w: durable ledger unavailable; fail closed", commitledger.ErrReplayBlocked)
+	}
+	if int(row.SchemaVersion) != commitledger.SchemaVersion {
+		return fmt.Errorf("%w: ledger schema mismatch; fail closed", commitledger.ErrReplayBlocked)
+	}
+	summary := commitledger.DurableSummary{
+		EverHadToolUse: row.EverHadToolUse,
+		EverDefinite:   row.EverDefinite,
+		EverAmbiguous:  row.EverAmbiguous,
+		EverSaturated:  row.EverSaturated,
+	}
+	if summary.BlocksReplay() {
+		return fmt.Errorf("%w: durable ledger records tool activity", commitledger.ErrReplayBlocked)
+	}
+	return nil
+}
```
Notas de exatidao: `BlocksReplay` tem **receiver de ponteiro** (`ledger.go:123`), entao a variavel
local `summary` precisa ser addressable - e e, sendo variavel local. `GetTaskLedgerGate` devolve
**somente os 4 booleanos + schema_version**, nunca as colunas de auditoria (secao 3.2).
`Check(correlationID string)` sem `ctx` e imposto pela interface existente; se o integrador quiser
`ctx`, isso muda a assinatura de `Check` no `ReplayGateHook` tambem, e passa a ser mudanca maior -
recomendo manter sem `ctx` nesta rodada e usar `context.Background()` com timeout curto.

---

## 3. PERSISTENCIA daemon -> backend SEM banco no daemon

### 3.1 DECISAO: endpoint dedicado, nao `ReportTaskMessages`

Premissa dura, provada no GTL-02: `grep -rn "DATABASE_URL|pgxpool|DatabaseURL" internal/daemon/*.go`
(sem testes) = **ZERO**. O daemon nao tem DSN nem pool. Toda persistencia vai por HTTP.

Comparei as duas opcoes:

| criterio | endpoint dedicado `PUT /api/daemon/tasks/:id/ledger-summary` | carona no `ReportTaskMessages` (`ack_handler.go:51-53`) |
|---|---|---|
| dispara nos 3 pontos exatos (4357/4399/4199) | SIM, independente do fluxo de mensagens | NAO: so quando houver batch de mensagens a reportar |
| task que falha sem emitir mensagem | grava | **nao grava** - e justamente o caso ambiguo que o gate precisa capturar |
| acoplamento | 1 rota nova | muda contrato de um endpoint quente do caminho de streaming |
| risco de regressao | isolado | atinge streaming de todas as tasks |
| ordenacao | idempotente por `OR`, ordem irrelevante | idem, mas herda o batching |

**DECIDIDO: endpoint dedicado.** O fator eliminatorio e a linha 2: o ponto 4.3
(`daemon.go:4199 MarkAllUnresolvedAmbiguous` + `4200 Close`) roda em `defer`, inclusive quando o
backend morreu antes de produzir mensagem. Carona no report perderia exatamente o sinal de
ambiguidade, que e o unico que nao pode ser perdido. O `AckHandler` permanece como esta, no papel
dele (watermark de output), sem alteracao.

### 3.2 Contrato do endpoint
```
PUT /api/daemon/tasks/{task_id}/ledger-summary
body: {"schema_version":1,"ever_had_tool_use":true,"ever_definite":false,
        "ever_ambiguous":false,"ever_saturated":false,
        "highest_seq_seen":12,"total_tool_use_count":3}
200 -> {"stored":true}
```
- autenticacao: a MESMA do restante das rotas `/api/daemon/*` (o integrador reusa o middleware
  existente; nao proponho rota sem autenticacao).
- o handler NAO aceita `false` como rebaixamento: a monotonicidade e imposta no SQL (secao 5), nao
  na confianca no cliente.
- corpo sem `task_id` no payload: vem so do path, para nao existir dois caminhos de identidade.

### 3.3 Os 3 pontos de gravacao (inalterados quanto a posicao, corrigidos quanto a semantica)
```go
daemon.go:4357 toolToken, _ = taskLedger.RecordToolUse(msg.CallID, int64(s))   // -> PUT ever_had_tool_use=true
daemon.go:4399 _ = taskLedger.RecordToolResult(toolToken)                      // -> PUT ever_definite=true
daemon.go:4199 taskLedger.MarkAllUnresolvedAmbiguous()                         // -> PUT summary final
daemon.go:4200 taskLedger.Close()
```
- 4357 grava **antes** do tool executar: conservador e correto, igual ao design original.
- o PUT deve ser **assincrono com retry bounded e nao-bloqueante** no ponto 4357, para nao
  introduzir latencia de rede no hot path de tool. Se o PUT falhar em todas as tentativas, o
  correto e **nao** ter linha: ausencia = fail-closed (secao 6), o lado seguro.
- no ponto 4199 o PUT precisa de contexto proprio: o `defer` roda com o ctx da task possivelmente
  ja cancelado. Usar `context.WithTimeout(context.Background(), ...)`, senao o sinal de
  ambiguidade e perdido exatamente nos cancelamentos - que sao a maioria dos casos ambiguos.

---

## 4. WIRING de producao

```go
// cmd/server/main.go, onde o TaskService e construido
+	taskSvc.ReplayGateHook = &service.DatabaseReplayGateChecker{Queries: queries, Logger: logger}
```
E o daemon continua com o registry em memoria (`daemon.go:301-302`) para o caminho ativo. O
comentario `replay_gate.go:103` ja descreve essa divisao: *"This is the in-process cache; durable
state lives server-side"* - a V2 finalmente a realiza.

Fechamento do buraco atual: hoje `grep -rn "ReplayGateHook:" --include=*.go .` = **vazio**, e
`cmd/server` e `internal/handler` nao mencionam `ReplayGateHook`, o que e a causa literal de
`"replay gate hook not configured; fail closed"`.

---

## 5. SQL - monotonicidade imposta pelo banco, nao por convencao

Migration **127** (a maior atual e `126_runtime_profile_protocol_family_native_runtimes`; sem
colisao: `grep -rln "task_ledger" migrations/ pkg/db/` = vazio).

```sql
-- 127_task_ledger_summary.up.sql
CREATE TABLE task_ledger_summary (
    task_id              UUID        PRIMARY KEY REFERENCES agent_task_queue(id) ON DELETE CASCADE,
    schema_version       INT         NOT NULL,
    ever_had_tool_use    BOOLEAN     NOT NULL DEFAULT FALSE,
    ever_definite        BOOLEAN     NOT NULL DEFAULT FALSE,
    ever_ambiguous       BOOLEAN     NOT NULL DEFAULT FALSE,
    ever_saturated       BOOLEAN     NOT NULL DEFAULT FALSE,
    highest_seq_seen     BIGINT      NOT NULL DEFAULT 0,
    total_tool_use_count BIGINT      NOT NULL DEFAULT 0,
    recorded_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at           TIMESTAMPTZ NOT NULL
);
CREATE INDEX task_ledger_summary_expires_at_idx ON task_ledger_summary (expires_at);
```
Duas escolhas que o design V1 nao fez:
1. **FK com `ON DELETE CASCADE` para `agent_task_queue(id)`**: alinha o ciclo de vida e evita linha
   orfa. RESSALVA para o integrador decidir: se uma task for apagada e recriada com o mesmo id (nao
   acontece com UUID), o cascade apagaria o bloqueio; e se o retention de `agent_task_queue` for
   MENOR que 24h, o cascade encurta o TTL efetivo. Nao verifiquei o retention da fila - fica como
   pergunta, nao como fato.
2. **`schema_version` sem DEFAULT**: forcar o cliente a declarar a versao. Com DEFAULT 1 um cliente
   antigo grava silenciosamente "compativel".

Monotonicidade no upsert, sem trigger:
```sql
-- name: UpsertTaskLedgerSummary :exec
INSERT INTO task_ledger_summary (
    task_id, schema_version, ever_had_tool_use, ever_definite, ever_ambiguous,
    ever_saturated, highest_seq_seen, total_tool_use_count, expires_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now() + INTERVAL '24 hours')
ON CONFLICT (task_id) DO UPDATE SET
    schema_version       = EXCLUDED.schema_version,
    ever_had_tool_use    = task_ledger_summary.ever_had_tool_use OR EXCLUDED.ever_had_tool_use,
    ever_definite        = task_ledger_summary.ever_definite     OR EXCLUDED.ever_definite,
    ever_ambiguous       = task_ledger_summary.ever_ambiguous    OR EXCLUDED.ever_ambiguous,
    ever_saturated       = task_ledger_summary.ever_saturated    OR EXCLUDED.ever_saturated,
    highest_seq_seen     = GREATEST(task_ledger_summary.highest_seq_seen, EXCLUDED.highest_seq_seen),
    total_tool_use_count = GREATEST(task_ledger_summary.total_tool_use_count, EXCLUDED.total_tool_use_count),
    updated_at           = now();
-- expires_at NAO e tocado no UPDATE: o TTL conta do PRIMEIRO registro, nao do ultimo,
-- senao uma task longa renova a janela indefinidamente.
```
Query de gate, deliberadamente estreita:
```sql
-- name: GetTaskLedgerGate :one
-- INVARIANTE: retorna SOMENTE o que decide o gate. Auditoria fica fora por construcao.
SELECT schema_version, ever_had_tool_use, ever_definite, ever_ambiguous, ever_saturated
FROM task_ledger_summary
WHERE task_id = $1 AND expires_at > now();
```
`expires_at > now()` no WHERE torna a expiracao **logica**, nao dependente do sweep: mesmo se o job
atrasar, linha vencida ja nao autoriza nem informa. O sweep passa a ser so limpeza de espaco:
```sql
-- name: DeleteExpiredTaskLedgerSummaries :execrows
DELETE FROM task_ledger_summary WHERE expires_at < now();
```
Ordem obrigatoria (`sqlc.yaml`: `schema: "migrations/"`, `queries: "pkg/db/queries/"`, out
`pkg/db/generated`, `sql_package: pgx/v5`): migration 127 -> queries -> `sqlc generate`. Query antes
da migration faz o sqlc nao resolver a tabela.

---

## 6. TTL - semantica explicita, decisao do owner

Alinhado a `ledger.go:39 TerminalTTL = 24 * time.Hour` e ao cutoff de `ledger.go:481`.

Tabela-verdade do gate depois da V2:

| estado | decisao | correto? |
|---|---|---|
| linha ausente (nunca gravou) | BLOQUEIA | sim - nao sabemos nada |
| linha com `expires_at <= now()` | BLOQUEIA (WHERE nao casa) | sim, por politica |
| `schema_version` divergente | BLOQUEIA | sim |
| erro de query / store nil | BLOQUEIA | sim |
| todos `ever_*` falsos, dentro da janela | PERMITE | sim - nenhum efeito colateral |
| qualquer `ever_*` verdadeiro | BLOQUEIA | sim |

**Efeito permanente que exige aceite explicito do owner**: linha ausente e linha expirada sao
indistinguiveis, logo **nenhuma task com mais de 24h volta a ter auto-retry**. Isso e mais restritivo
que hoje apenas na aparencia - hoje o bloqueio e 100% por falta de wiring. Nao e detalhe de limpeza,
e politica.

---

## 7. PATCH MAP, ARQUIVOS TRAVADOS E TESTES

### 7.1 Patch map, na ordem de aplicacao
| # | arquivo | acao | ancora |
|---|---|---|---|
| P0 | `internal/daemon/config.go` + fonte de env | POPULAR `CommitLedgerHMACSecret` (hoje nunca atribuido) | `config.go:115`, `daemon.go:4145` |
| P1 | `migrations/127_task_ledger_summary.{up,down}.sql` | NOVO | secao 5 |
| P2 | `pkg/db/queries/task_ledger_summary.sql` | NOVO: Upsert, GetTaskLedgerGate, DeleteExpired | secao 5 |
| P3 | `pkg/db/generated/*` | GERADO por `sqlc generate` - nunca a mao | `sqlc.yaml` |
| P4 | `internal/daemon/commitledger/replay_gate.go` | ADICIONAR `ReplayGateChecker` + `var _`; ALTERAR assinatura de `CheckOrAllow`; CORRIGIR comentarios 106 e 169 | 149, 200-205, 106, 169 |
| P5 | `internal/service/task.go` | trocar tipo do campo | `42` |
| P6 | `internal/service/replay_gate_db.go` | NOVO checker duravel | secao 2.3 |
| P7 | `internal/handler/<daemon route>.go` | NOVO `PUT /api/daemon/tasks/:id/ledger-summary`, autenticado como as demais `/api/daemon/*` | secao 3.2 |
| P8 | `internal/daemon/daemon.go` | 3 chamadas assincronas ao endpoint | `4357`, `4399`, `4199`+`4200` |
| P9 | `internal/daemon/client.go` (ou equivalente) | metodo cliente do PUT | - |
| P10 | `cmd/server/main.go` | wiring do checker | secao 4 |
| P11 | `internal/service/autopilot.go` | gate explicito no replay automatico | `275` |
| P12 | sweep do TTL | job horario chamando `DeleteExpiredTaskLedgerSummaries` | secao 5 |

### 7.2 Gate explicito no Autopilot (P11)
Achado do GTL-02: `EnqueueTaskForIssue` tem 8 call sites e nenhum consulta o gate -
`handler/comment.go:1136`, `handler/issue.go:2535/2559/3035/3050`,
`handler/onboarding_shim.go:329`, `service/issue.go:388`, `service/autopilot.go:275`. Sete sao acao
de usuario/comentario e cabem na isencao ja declarada em `task.go:41 // Manual RerunIssue is exempt
from this gate`. **`autopilot.go:275` nao cabe**: e replay automatico.

Proposta minima, no unico ponto automatico:
```go
// service/autopilot.go, imediatamente antes da linha 275
+	// Automatic replay: same safety bar as MaybeRetryFailedTask. Manual user
+	// actions are exempt (task.go:41); autopilot is not a manual action.
+	if err := commitledger.CheckOrAllow(s.TaskSvc.ReplayGateHook, priorTaskID); err != nil {
+		// skip this issue, record the reason, do not enqueue
+	}
 	if _, err := s.TaskSvc.EnqueueTaskForIssue(ctx, issue); err != nil {
```
DEPENDENCIA que preciso declarar: `autopilot.go:578` ja usa `s.Queries.HasActiveTaskForIssue(ctx, task.IssueID)`,
logo existe uma `task` anterior em escopo em parte do arquivo, mas **NAO confirmei** que na linha 275
existe um `priorTaskID` disponivel. Se nao existir, o gate ali exige primeiro resolver qual task
anterior e a referencia - e isso e decisao de desenho do integrador, nao minha. Alternativa aceitavel:
**declarar formalmente a isencao do autopilot** e cobri-la com o teste 7.6, para virar escolha
consciente em vez de buraco. Das duas, recomendo o gate; mas nao afirmo que a variavel existe.

### 7.3 ARQUIVOS TRAVADOS (locked) - proposta de ownership
| arquivo | dono unico sugerido | por que |
|---|---|---|
| `commitledger/replay_gate.go` | Codex56-TL | contrato congelado; toca fail-closed |
| `service/task.go` | Codex56-TL | ja e escritor unico dele nesta janela |
| `service/replay_gate_db.go` | 1 executor | arquivo novo, sem conflito |
| `migrations/127_*` + `pkg/db/queries/task_ledger_summary.sql` | 1 executor (o mesmo) | migration e query precisam nascer juntas |
| `pkg/db/generated/*` | NINGUEM edita | saida de `sqlc generate` |
| `daemon/daemon.go` | 1 executor | arquivo enorme e disputado; so as 3 chamadas |
| `handler/<rota daemon>` | 1 executor | rota nova |
| `cmd/server/main.go` | Codex56-TL | wiring de producao |
| `service/autopilot.go` | 1 executor | depende da decisao 7.2 |
Regra: `pkg/db/generated/*` nunca entra em worktree de dois agentes ao mesmo tempo - conflito de
codigo gerado e ruido puro.

### 7.4 a 7.9 - Testes exigidos
- **7.4 restart (o teste que prova o objetivo)**: registry recem-criado e vazio + linha no banco com
  `ever_had_tool_use=true` -> `Check` BLOQUEIA; linha com os 4 `ever_*` falsos e dentro da janela ->
  PERMITE. Hoje isso e impossivel de testar; e a razao de existir da V2.
- **7.5 fail-closed em todas as bocas**: `hook` nil literal; ponteiro tipado nil dentro da interface
  (a armadilha da secao 2.1); `Queries` nil; `pgx.ErrNoRows`; erro generico de query;
  `schema_version` diferente de `commitledger.SchemaVersion` (`ledger.go:33`); `expires_at` no
  passado. Sete casos, todos devendo BLOQUEAR.
- **7.6 ambiguidade**: summary com `ever_ambiguous=true` originado de
  `MarkAllUnresolvedAmbiguous` (`daemon.go:4199`) sobrevive ao restart e bloqueia. Mais o caso do
  `defer` com ctx cancelado: o PUT ainda tem de sair (secao 3.3).
- **7.7 idempotencia e monotonicidade**: upsert repetido nao cria segunda linha; `false` chegando
  depois de `true` mantem `true` nos quatro booleanos; `highest_seq_seen` e `total_tool_use_count`
  nunca decrescem (`GREATEST`); `expires_at` NAO se move no UPDATE.
- **7.8 invariante de escopo**: a query de gate nao expoe `total_tool_use_count` nem
  `highest_seq_seen` - teste de contrato sobre a struct gerada, para que uma edicao futura na query
  quebre o teste em vez de vazar auditoria para a decisao.
- **7.9 bypass**: teste que fixa o comportamento do autopilot - com gate (bloqueia) ou sem gate
  (documenta a isencao). Qualquer das duas, mas escrito, para nao voltar a ser acidente.

---

## 8. Ordem de rollout e o que NAO fazer

1. **P0 primeiro, sozinho**: popular `CommitLedgerHMACSecret`. Sem isso, todo summing e
   `EverSaturated=true` (secao 0) e a V2 grava bloqueio permanente para todas as tasks.
2. **P1-P3** schema e queries, com `sqlc generate`.
3. **P4-P6** seam e checker, sem wiring - inerte, seguro de integrar.
4. **P7-P9** endpoint e gravacao.
5. **P10** wiring: e aqui que o comportamento muda de verdade.
6. **P11-P12** autopilot e sweep.

NAO fazer: habilitar P10 antes de P0. Nao ha rollback barato depois de gravar
`ever_saturated=true` em massa - a linha sobrevive ao restart, que e justamente o objetivo da
feature virado contra ela.

## 9. Fora de escopo

As 9 issues ORQ-12, 13, 15, 16, 17, 18, 21, 22, 23 permanecem PRESERVADAS: este desenho nao autoriza
nem executa rerun. ORQ-26 e escopo separado. Nada aqui depende de rollback do T2 - ressalva do
Codex56#B respeitada.

## 10. Nada mutado

Nenhum arquivo de codigo editado; sem `go build`, `go vet`, `go test`, deploy, restart, commit,
push, chmod, instalacao ou rerun. Leitura de fonte no ORQ2 e leitura de nomes de variaveis de
ambiente no ORQ1 - nenhum valor de segredo lido, impresso ou transcrito. Os unicos arquivos criados
sao este e o meu check-out.
