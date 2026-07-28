# GTL-02 - Auditoria de implementacao do Ledger Duravel (READ-ONLY)

- auditor: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-27T11:34Z
- alvo: `.deploy-control/p0/evidence/ledger-durable-design.md` (Antigravity w8:p2) contra o codigo em HEAD
- escritor exclusivo: Codex56-TL w5:pC. Nada aqui foi aplicado.
- modo: leitura. Nenhum arquivo de codigo editado, nenhum build, deploy, restart, commit ou rerun.
- ESTADO: entrega final do GTL-02. Nenhuma frente nova proposta; aguardo redistribuicao do
  General-Tech-Lead (Codex56-TL, w5:pC). Nota de registro: houve um comunicado de encerramento da
  jornada durante esta auditoria, REVOGADO pelo owner em seguida; a entrega segue completa e o
  escopo nao mudou.

## 0. Veredito

Design **CORRETO no diagnostico e na direcao**, com **2 bloqueadores de implementacao** que ele
declara inexistentes, **1 premissa falsa** e **1 bypass automatico nao mapeado**. Nao e reprovacao:
e o delta que falta para virar patch.

## 1. Ancoras conferidas uma a uma

| Design afirma | Codigo em HEAD | Resultado |
|---|---|---|
| daemon.go:301-302 registry em memoria | `301 d.commitLedgers = commitledger.NewLedgerRegistry()` / `302 d.commitAckHandler = commitledger.NewAckHandler(...)` | EXATO |
| replay_gate.go:95-99 `LedgerRegistry` | struct real em **104-107** | conteudo confere, linha derivou |
| task.go:1666 `CheckOrAllow` | `1666 if err := commitledger.CheckOrAllow(s.ReplayGateHook, parentTaskID); err != nil {` | EXATO |
| ledger.go:111-120 `DurableSummary` | struct em **112-119**, 6 campos exatos | confere, 1 linha de offset |
| "ledger.go:113 - funcao só lê os 4 booleans" | `123 func (s *DurableSummary) BlocksReplay() bool {` / `124 return s.EverHadToolUse \|\| s.EverDefinite \|\| s.EverAmbiguous \|\| s.EverSaturated` | INVARIANTE CONFIRMADA, linha e 123-125 |
| TTL alinhado a `TerminalTTL` | `ledger.go:39 TerminalTTL = 24 * time.Hour` e `ledger.go:481 cutoff := time.Now().Add(-TerminalTTL)` | EXATO |
| `SchemaVersion` | `ledger.go:33 const SchemaVersion = 1` | EXATO |
| ponto 4.1 pre-tool_use daemon.go:4357 | `4357 toolToken, _ = taskLedger.RecordToolUse(msg.CallID, int64(s))` | EXATO |
| ponto 4.2 pos-tool_result daemon.go:4399 | `4399 _ = taskLedger.RecordToolResult(toolToken)` | EXATO |
| ponto 4.3 defer daemon.go:4199 | `4199 taskLedger.MarkAllUnresolvedAmbiguous()` / `4200 taskLedger.Close()` | EXATO |
| fail-closed com hook nil | `replay_gate.go:200-204`, literal `"%w: replay gate hook not configured; fail closed"` | EXATO |

O comentario `replay_gate.go:103` ja diz *"This is the in-process cache; durable state lives
server-side"*, e `task.go:1663-1664` ja diz *"Until the durable server-side store is wired, all
automatic retries for tasks with tool activity are blocked"*. Ou seja o design nao inventa
direcao: ele executa uma intencao que o codigo ja declara. Isso e ponto a favor.

## 2. BLOQUEADOR 1 - `ReplayGateHook` e struct concreta, nao interface

Design secao 8: *"`commitledger/replay_gate.go` | Nenhuma mudança"*. **Falso.**

```go
152 type ReplayGateHook struct {
153 	Registry *LedgerRegistry
154 	Logger   *slog.Logger
155 }
...
200 func CheckOrAllow(hook *ReplayGateHook, correlationID string) error {
```
e no consumidor:
```go
42 	ReplayGateHook *commitledger.ReplayGateHook
```
`DatabaseReplayGateHook` NAO pode ser atribuido a esse campo nem passado a `CheckOrAllow`: o tipo
e ponteiro para struct concreta, com dependencia dura em `*LedgerRegistry`. Para o desenho da
secao 5 compilar e obrigatorio extrair a costura, por exemplo
`type ReplayGateChecker interface { Check(correlationID string) error }`, trocar a assinatura de
`CheckOrAllow` para receber a interface e trocar o tipo do campo em `task.go:42`. Sao 3 edicoes
alem das listadas. Sem isso o patch nao compila.

## 3. BLOQUEADOR 2 - o daemon NAO tem acesso ao Postgres

Design secao 8, alternativa: *"o daemon grava diretamente no Postgres do backend via string de
conexão já disponível em `daemon.go` (o daemon já conhece o DB — verifica em `d.cfg`)"*.
**Premissa falsa.** `grep -rn "DATABASE_URL|pgxpool|DatabaseURL" internal/daemon/*.go` (sem testes)
retorna **ZERO ocorrencias**. O daemon nao conhece DSN nem abre pool. A "alternativa mais simples"
nao existe: a persistencia tem de atravessar o canal HTTP daemon->backend.

## 4. OPORTUNIDADE - o canal daemon->backend JA EXISTE e o design nao usou

O design propoe endpoint novo `PUT /api/daemon/tasks/:id/ledger-summary`. Ja existe um canal de ack
no mesmo eixo, com semantica de watermark:
```go
ack_handler.go:8-10  // AckHandler processes persisted_through_seq acknowledgements from the
                     // server and advances the corresponding ledger output watermarks.
                     // The flow: server (handler) → HTTP response → client → AckHandler → ledger.
ack_handler.go:51-53 type ReportTaskMessagesResponse struct { PersistedThroughSeq int64 `json:"persisted_through_seq"` }
```
O summary poderia subir carona no `ReportTaskMessages` que o daemon ja chama, em vez de um endpoint
novo. Isso e recomendacao ao escritor unico, nao exigencia: endpoint dedicado tem a vantagem de
gravar nos 3 pontos exatos (4357/4399/4199) independente do fluxo de mensagens. Registro a opcao
porque o design nao a considerou.

## 5. BYPASS AUTOMATICO NAO MAPEADO - autopilot enfileira sem passar pelo gate

Design secao 7 trata apenas do rerun manual: *"O rerun manual via `multica issue rerun` bypassa o
gate por design"*. Correto, e e intencional (`task.go:41 // Manual RerunIssue is exempt`).
Porem `EnqueueTaskForIssue` tem **8 call sites** e NENHUM consulta o gate:
```
internal/handler/comment.go:1136
internal/handler/issue.go:2535, 2559, 3035, 3050
internal/handler/onboarding_shim.go:329
internal/service/issue.go:388
internal/service/autopilot.go:275
```
Sete sao acao de usuario ou de comentario, aceitaveis pela mesma isencao. **`autopilot.go:275` nao
e**: autopilot e replay AUTOMATICO, exatamente a classe que o gate existe para conter, e passa por
fora. O unico ponto guardado e `MaybeRetryFailedTask` (`task.go:1666`). Consequencia pratica: mesmo
com o ledger duravel implementado como desenhado, uma task com `ever_had_tool_use=true` continua
podendo ser re-enfileirada pelo autopilot sem consulta. Isso precisa de decisao explicita do owner:
ou o autopilot passa a consultar o gate, ou fica documentado como isencao consciente.

## 6. Migrations - sem colisao, numero e mecanica

- `grep -rln "task_ledger" migrations/ pkg/db/` retorna **vazio**: nao ha colisao de nome.
- Maior migration atual: `126_runtime_profile_protocol_family_native_runtimes`. A nova deve ser
  **127**, com par `.up.sql`/`.down.sql`, seguindo a convencao de todas as anteriores.
- `sqlc.yaml` usa `schema: "migrations/"` e `queries: "pkg/db/queries/"`, saida em
  `pkg/db/generated`, `sql_package: pgx/v5`, `emit_json_tags` e `emit_empty_slices`. Logo a ordem
  obrigatoria e: (1) migration 127, (2) queries em `pkg/db/queries/`, (3) `sqlc generate`. Escrever
  a query antes da migration faz o sqlc nao resolver a tabela.
- Contexto vizinho: `123_rotation` e `124_approved_accounts` ja existem, o que e relevante porque a
  ponte de contas da ORQ-21 vive nessas tabelas. Sem conflito com `task_ledger_summary`.

## 7. Risco de correlacao - chave primaria pode nao casar com a chave em memoria

`task.go:1665-1666` chama o gate com o **UUID cru**:
```go
1665 	parentTaskID := util.UUIDToString(parent.ID)
1666 	if err := commitledger.CheckOrAllow(s.ReplayGateHook, parentTaskID); err != nil {
```
mas o contrato do hook declara o oposto: `replay_gate.go:169 // Uses pseudonymous correlation ID
(not raw task ID)`, e o registry e documentado como `replay_gate.go:106 // keyed by pseudonymous
task correlation`. O daemon registra em `daemon.go:4195 d.commitLedgers.Register(taskID, taskLedger)`.
NAO confirmei a derivacao do `taskID` usado ali - nao segui essa variavel. Fica como pergunta
aberta para o escritor unico: se a chave do daemon nao for a mesma string do UUID cru, o design
(que usa `task_id UUID PRIMARY KEY`) fecha a lacuna por acidente no caminho DB e mantem a
divergencia no caminho em memoria. Vale resolver a contradicao de contrato antes de gravar schema.

## 8. TTL e fail-closed - consistentes, com um efeito permanente a aceitar

`expires_at = recorded_at + 24h` casa com `TerminalTTL` (ledger.go:39) e com o cutoff de
`ledger.go:481`. Ponto que o design trata como detalhe e merece decisao explicita: apos o sweep, a
linha ausente e **indistinguivel** de "task nunca teve ledger", e ambos caem em fail-closed. Logo
toda task com mais de 24h fica permanentemente sem auto-retry. Concordo que e o lado seguro, mas e
mudanca de comportamento permanente, nao um detalhe de limpeza.

Sobre a imutabilidade proposta (`ever_x = ever_x OR $new`): correta e necessaria, porque e o que
preserva monotonicidade equivalente a do ledger em memoria. Recomendo materializar isso como
CHECK/regra no proprio SQL, nao apenas como convencao de escrita, senao um upsert futuro pode
retroceder um booleano sem quebrar teste.

## 9. Testes exigidos - o que existe e o que falta

Existe hoje, tudo em memoria: `ledger_test.go` (15465 B), `replay_gate_test.go` (5910 B),
`race_test.go` (5906 B), `ack_handler_test.go` (4527 B). Nenhum cobre durabilidade, porque a tabela
nao existe. Faltam, na ordem do dispatch:

1. **restart**: registry vazio (`NewLedgerRegistry` recem-criado) + linha no banco com
   `ever_had_tool_use=true` -> gate BLOQUEIA; e linha com todos os `ever_*` falsos -> gate PERMITE.
   E o teste que prova o objetivo do dispatch.
2. **ambiguidade**: `MarkAllUnresolvedAmbiguous` (daemon.go:4199) persistido -> `ever_ambiguous=true`
   sobrevive ao restart e bloqueia.
3. **idempotencia**: upsert repetido do mesmo summary nao cria linha nova nem retrocede booleano;
   `false` chegando depois de `true` mantem `true`.
4. **fail-closed**: linha ausente, `schema_version` diferente de `commitledger.SchemaVersion`
   (ledger.go:33) e erro de query -> todos bloqueiam.
5. **seam**: com a interface do bloqueador 1, um fake `Checker` prova que `CheckOrAllow` continua
   bloqueando com nil.
6. **regressao de bypass**: teste que documenta que `autopilot.go:275` nao consulta o gate, para a
   isencao ser deliberada e nao acidental.

## 10. Fora de escopo, confirmado

- O design nao autoriza rerun das 9 issues preservadas, e eu tambem nao. Permanecem PRESERVADAS.
- ORQ-26 e escopo separado, como o proprio design diz.
- Ressalva do Codex56#B respeitada: nao existe rollback de um comando para o T2 e a branch antiga
  reintroduziria o AGY task-incapaz. Nada nesta auditoria depende de rollback.

## 11. Nada mutado

Nenhum arquivo de codigo editado, nenhum `go build`, `go vet`, `go test`, deploy, restart, commit,
push, chmod, instalacao ou rerun. Somente leitura de fonte e de config no worktree do ORQ2. Os
unicos arquivos que criei sao este e o meu check-out.
