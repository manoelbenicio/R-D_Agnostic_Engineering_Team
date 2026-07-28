# ORQ-41 W4 - review adversarial independente de `e0b0155` sobre `c047c0b` (READ-ONLY)

- **Revisor:** Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T16:26Z
- **Worktree:** `/home/ec2-user/workspace/worktrees/gtl-orq41-w4-autopilot`, branch `agent/opus48-d/orq41-w4-autopilot`
- **Commits:** `c047c0b` (BLOCKED) -> `e0b0155` (correcao)
- **Modo:** READ-ONLY. Nao editei, nao fiz amend, rebase, push, stage, commit nem mutei board.
  Executei apenas leitura de git e `go build`/`go vet`/`go test -race`, que nao alteram a arvore
  (`git status --porcelain` vazio antes e depois).
- **Insumo mecanico usado somente para pai/escopo/limpeza:**
  `.deploy-control/p0/evidence/orq41-commit-delta-inventory.md`, SHA verificado por mim como
  `fb7ef4819ba49f735041bd1ac48cd36f30d666c49778bd62b4b996bff6e9d7d9` — **confere**. O julgamento de
  **comportamento** abaixo e independente, feito sobre o codigo.

# VEREDITO: **PASS**, com **1 ressalva bloqueante para o merge** e 2 observacoes

A correcao fecha o defeito que bloqueava `c047c0b`. A ressalva nao e sobre o codigo de `e0b0155`, e
sobre a **evidencia de teste do pacote `internal/handler`**, que eu medi como **falso verde**.

| # | verificacao exigida | resultado |
|---|---|---|
| 1 | comportamento off-path/default byte-equivalente | **PASS** |
| 2 | injecao imutavel por construtor, sem setter e sem corrida | **PASS** |
| 3 | correlacao pela task terminal mais recente de issue+agent, nao pelo autopilot ID | **PASS** |
| 4 | handler sem falso `recorded` e com semantica fail-closed | **PASS** |
| 5 | claim de HMAC removido | **PASS** |
| 6 | sem quebra de escopo em router/migration/generated/query/commitledger | **PASS** |
| 7 | testes nao espelham o bug | **PASS** |
| 8 | build / vet / race | **PASS** |
| 9 | caveat anti-skip do handler | **FALSO VERDE MEDIDO** — ressalva |

---

## 1. O DEFEITO QUE BLOQUEAVA `c047c0b` - CONFIRMADO NA RAIZ

`c047c0b` chamava, em `shouldSkipDispatch`:
```go
func (s *AutopilotService) checkAutopilotReplayGate(ctx context.Context, ap db.Autopilot, agent db.Agent) error {
	correlationID := util.UUIDToString(ap.ID)
	return commitledger.CheckOrAllow(s.ReplayGateHook, correlationID)
}
```
E `internal/daemon/commitledger/replay_gate.go:200-205` (literal, **arquivo intocado por este lane**):
```go
func CheckOrAllow(hook *ReplayGateHook, correlationID string) error {
	if hook == nil {
		return fmt.Errorf("%w: replay gate hook not configured; fail closed", ErrReplayBlocked)
	}
	return hook.Check(correlationID)
}
```
Como nenhum call site de producao setava o hook, `hook == nil` era o estado real, e **toda** dispatch
de Autopilot seria recusada. Somava-se o erro de correlacao: `util.UUIDToString(ap.ID)` consulta o
ledger por **autopilot ID**, chave que nunca e registrada — logo, mesmo com hook armado, o resultado
seria bloqueio universal. Dois defeitos independentes, ambos levando a indisponibilidade do produto.

## 2. OFF-PATH / DEFAULT BYTE-EQUIVALENTE - **PASS**

`NewAutopilotService` em `e0b0155` e **byte-identico** ao de `0cb8aeb` (baseline pre-gate):
```go
func NewAutopilotService(q *db.Queries, tx TxStarter, bus *events.Bus, taskSvc *TaskService) *AutopilotService {
	return &AutopilotService{Queries: q, TxStarter: tx, Bus: bus, TaskSvc: taskSvc}
}
```
Confirmei por `git show 0cb8aeb:...` vs `git show e0b0155:...`: mesma assinatura, mesmo corpo, sem
campo novo inicializado. O baseline `0cb8aeb` nao tinha nenhuma referencia a replay gate
(`grep -c 'replayGate|ReplayGate|commitledger'` = **0**).

E o caminho quente sai imediatamente quando desarmado:
```go
func (s *AutopilotService) replayGateSkipReason(...) (string, bool) {
	if !s.ReplayGateArmed() {
		return "", false
	}
```
`ReplayGateArmed()` exige **ambos** os colaboradores non-nil. Construido pelo construtor de producao,
os dois sao `nil`, logo a funcao retorna `"", false` antes de qualquer I/O. **Nenhuma dispatch pode ser
recusada pelo gate no default.** Equivalencia comportamental com o pre-gate: **confirmada**.

## 3. INJECAO IMUTAVEL, SEM SETTER, SEM CORRIDA - **PASS**

O setter publico foi **removido**:
```
-func (s *AutopilotService) SetReplayGateHook(hook *commitledger.ReplayGateHook) {
-	s.ReplayGateHook = hook
-}
```
Substituido por campos **unexported** e um construtor dedicado:
```go
	// replayGate and replayCorrelation are set once, at construction, and are
	// never mutated afterwards. There is deliberately no setter: a setter
	// called after the service starts serving would race with the reads in
	// replayGateSkipReason.
	replayGate        AutopilotReplayGate
	replayCorrelation AutopilotReplayCorrelation
```
```go
func NewAutopilotServiceWithReplayGate(..., gate AutopilotReplayGate, correlation AutopilotReplayCorrelation) (*AutopilotService, error) {
	if gate == nil || correlation == nil {
		return nil, ErrAutopilotReplayGateIncomplete
	}
```
Tres propriedades que eu confirmo:
1. **sem setter** — nao ha caminho de mutacao pos-construcao; o campo antigo `ReplayGateHook` exported
   deixou de existir, logo nem escrita externa direta e possivel;
2. **all-or-nothing** — armar pela metade falha em construcao com `ErrAutopilotReplayGateIncomplete`,
   fechando o cenario em que um colaborador nil desarmaria o gate silenciosamente **ou** bloquearia
   tudo;
3. **sem corrida** — escrita apenas antes de o servico ser publicado; leituras posteriores sao
   somente leitura. `go test -race` nao reportou corrida (secao 8).

Registro que `ReplayGateArmed()` exposto e a escolha certa: permite readiness distinguir os dois
comportamentos **sem** expor estado interno.

## 4. CORRELACAO PELA TASK ANTERIOR, NAO PELO AUTOPILOT ID - **PASS**

O contrato declara a intencao no proprio codigo:
```go
// Implementations receive the correlation ID of the *prior task*, never the
// autopilot ID: commit ledgers are registered per task (internal/daemon/daemon.go
// registers by task ID) and the existing consumer in TaskService correlates by
// parent task ID. Passing an autopilot ID would look up a key that is never
// registered and block every dispatch.
type AutopilotReplayGate interface {
	AllowReplay(ctx context.Context, priorTaskCorrelationID string) error
}
type AutopilotReplayCorrelation interface {
	PriorTaskCorrelationID(ctx context.Context, ap db.Autopilot, agent db.Agent) (correlationID string, ok bool, err error)
}
```
E a implementacao passa **exclusivamente** `correlationID` vindo do resolver:
```go
	correlationID, ok, err := s.replayCorrelation.PriorTaskCorrelationID(ctx, ap, agent)
	...
	if err := s.replayGate.AllowReplay(ctx, correlationID); err != nil {
```
`util.UUIDToString(ap.ID)` **nao** e mais usado como chave de gate — so como campo de log. A resolucao
recebe `ap` **e** `agent`, coerente com "issue+agent". **PASS.**

Ressalva de escopo honesta, que o proprio codigo declara: a **implementacao** do resolver nao existe
neste lane ("This lookup is a database read that W4 does not own"). Portanto "task terminal mais
recente" e hoje **contrato**, nao comportamento verificavel — quem implementar o resolver e que tera de
provar a ordenacao por `updated_at`/terminalidade. Nao considero isso defeito de `e0b0155`: e limite
de escopo declarado.

## 5. SEMANTICA FAIL-CLOSED - **PASS**, com a distincao certa

Tres caminhos, e cada um faz o que deve:
| situacao | comportamento | avaliacao |
|---|---|---|
| gate desarmado (producao hoje) | **allow**, sem I/O | correto: gate de seguranca nao-cabeado nao derruba produto |
| resolver retorna erro | **block** com `"replay gate blocked: prior task correlation unavailable"` | **fail-closed correto**: sem resposta autoritativa nao se distingue primeira execucao de replay |
| `ok == false` (sem task anterior) | **allow**, sem consultar o gate | correto: primeira execucao nao tem o que replayar; consultar um checker fail-closed bloquearia toda primeira dispatch |
| gate recusa | **block** com `"replay gate blocked: "+err.Error()` | correto |

O ponto fino que aprovo explicitamente: **fail-closed quando armado, fail-open quando desarmado**. Sao
posturas opostas e a distincao e deliberada e documentada. Um revisor apressado poderia chamar o
default de "fail-open inseguro"; nao e, porque o gate desarmado nunca foi uma protecao ativa — em
`c047c0b` ele era um **bloqueio universal disfarcado de protecao**.

Observacao de vazamento: `err.Error()` do gate entra na razao de skip. O contrato diz "its message must
stay content-free". Isso e obrigacao do **implementador do gate**, nao verificavel aqui, e o codigo
declara a exigencia. Registro como dependencia, nao como defeito.

## 6. HANDLER SEM FALSO `recorded` - **PASS**

`c047c0b` respondia `200` com `{"status":"recorded"}` **sem escrever nada**. Removido:
```
-	_ = json.NewEncoder(w).Encode(DaemonLedgerSummaryResponse{
-		Status: "recorded",
-		TaskID: payload.TaskID,
-	})
+	writeError(w, http.StatusServiceUnavailable, "ledger summary persistence is not configured; fail closed")
```
O tipo `DaemonLedgerSummaryResponse` foi eliminado junto, o que impede reintroducao acidental do claim.
`503` e o codigo certo: a rota existe, a persistencia nao. **PASS.**

Dois endurecimentos adicionais que aprovo:
1. `decoder.DisallowUnknownFields()` — payload com campo desconhecido passa a ser `400`, fechando
   divergencia silenciosa de contrato entre daemon e backend;
2. o comentario enumera **duas** exigencias que o handler **nao pode** cumprir sozinho e as atribui a
   W3: montar na rota autenticada de daemon, e validar que `{taskId}` pertence ao daemon chamador
   ("this function validates UUID *shape*, not ownership"). Declarar a lacuna e melhor do que finge-la
   coberta.

## 7. CLAIM DE HMAC REMOVIDO - **PASS**

`git diff 0cb8aeb e0b0155 | grep -niE 'hmac|sha256|secret'` -> **vazio**. Nenhuma mencao remanescente
no delta total do lane. Entre `c047c0b` e `e0b0155` a linha
`-		HMACSecret: []byte("01234567890123456789012345678901"),` foi **removida** — inclusive um
segredo de teste literal em codigo, o que e bom que tenha saido. O titulo de `c047c0b` prometia
"commit ledger HMAC secret"; `e0b0155` deixa de reivindicar isso. **PASS.**

## 8. ESCOPO - **PASS**

```
git diff --name-only 0cb8aeb e0b0155 | grep -E 'router\.go|migrations/|pkg/db/generated|pkg/db/queries|commitledger'
(vazio)
```
Os **4** arquivos tocados no lane inteiro:
```
internal/handler/daemon_ledger_summary.go
internal/handler/daemon_ledger_summary_test.go
internal/service/autopilot.go
internal/service/autopilot_replay_test.go
```
Nenhum `router.go`, nenhuma migration, nenhum generated, nenhuma query, nenhum `commitledger`.
Confirmei tambem que `commitledger/replay_gate.go` esta **intocado**, embora eu o tenha lido para
diagnosticar. **PASS.**

Pai e limpeza, conferidos independentemente e coincidentes com o inventario mecanico:
```
git rev-parse e0b0155^  -> c047c0b7ed710f260e76a6ee50257b2a5f96dc70
git rev-parse c047c0b   -> c047c0b7ed710f260e76a6ee50257b2a5f96dc70
c047c0b existe como commit, data 2026-07-27 15:51:10 +0000  -> nao houve amend
git status --porcelain  -> vazio
```

## 9. TESTES NAO ESPELHAM O BUG - **PASS**

7 testes de servico, e nenhum deles assume o comportamento antigo:
```
TestNewAutopilotService_LeavesReplayGateDisarmed
TestReplayGateSkipReason_ProductionServiceDoesNotBlockWhenUnwired
TestNewAutopilotServiceWithReplayGate_RejectsPartialWiring
TestReplayGateSkipReason_CorrelatesByPriorTaskNotAutopilotID
TestReplayGateSkipReason_AllowsFirstRunWithoutConsultingGate
TestReplayGateSkipReason_BlocksWhenGateRefuses
TestReplayGateSkipReason_FailsClosedWhenCorrelationErrors
```
O teste de correlacao tem a assercao **negativa** que importa, e e o antidoto exato do bug:
```go
	if gate.seen[0] != priorTaskID {
		t.Errorf("gate must be consulted with the prior task ID %q, got %q", priorTaskID, gate.seen[0])
	}
	if gate.seen[0] == apIDStr {
		t.Errorf("gate must never be consulted with the autopilot ID %q", apIDStr)
	}
```
Se alguem reintroduzir `util.UUIDToString(ap.ID)` como chave, **os dois** asserts falham. E
`TestReplayGateSkipReason_ProductionServiceDoesNotBlockWhenUnwired` trava o defeito de bloqueio
universal. Cobertura das duas falhas de `c047c0b`. **PASS.**

## 10. BUILD / VET / RACE - **PASS**

```
go build ./...                                            -> exit 0
go vet ./internal/service/... ./internal/handler/...       -> exit 0
go test -race ./internal/service -run 'ReplayGate|AutopilotService' -count=1 -v
   -> PASS,  ok  internal/service  1.036s,  --- PASS: 7,  --- FAIL: 0,  zero WARNING: DATA RACE
```
7 de 7 com detector de corrida ativo. **PASS.**

## 11. RESSALVA BLOQUEANTE PARA O MERGE - TESTE DE HANDLER E **FALSO VERDE**

```
go test -race ./internal/handler -run 'PutDaemonLedgerSummary' -count=1 -v
  -> ok  internal/handler  1.841s      (exit 0)
  -> ocorrencias de "Skipping tests:"  = 1
  -> ocorrencias de "--- PASS:"        = 0
```
**Nenhum** dos tres testes de handler executou. O `ok` vem de
`internal/handler/handler_test.go:38-54`, cujo `TestMain` faz `os.Exit(0)` **antes** de `m.Run()`
quando o banco nao responde:
```go
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil { fmt.Printf("Skipping tests: could not connect to database: %v\n", err); os.Exit(0) }
	if err := pool.Ping(ctx); err != nil { fmt.Printf("Skipping tests: database not reachable: %v\n", err); pool.Close(); os.Exit(0) }
```
Consequencia direta: **as tres assercoes centrais do handler nao estao provadas** —
`WellFormedPayloadFailsClosedWithoutClaimingPersistence`, `InvalidUUID` e `IDMismatch`. O fail-closed
`503` e a remocao do `recorded` estao corretos **por leitura de codigo**, e nao por execucao.

Isso **nao** e defeito de `e0b0155`: os tres testes nao dependem de banco algum; o bloqueio e o
`TestMain` do pacote. Mas e bloqueante como **evidencia**: nao se pode declarar o handler verificado
com `ok` vazio.

**Condicao para virar PASS pleno:** reexecutar com `DATABASE_URL` apontando para um Postgres
alcancavel e exigir `--- PASS:` **nominal** para os tres testes, mais ausencia da linha
`Skipping tests:`. Sugestao de gate:
```bash
: "${DATABASE_URL:?sem banco o TestMain faz os.Exit(0) e o ok e vazio}"
OUT=$(go test -race ./internal/handler -run 'PutDaemonLedgerSummary' -count=1 -v)
printf '%s\n' "$OUT" | grep -q 'Skipping tests:' && { echo "FALSO VERDE"; exit 1; }
for t in TestPutDaemonLedgerSummary_WellFormedPayloadFailsClosedWithoutClaimingPersistence \
         TestPutDaemonLedgerSummary_InvalidUUID TestPutDaemonLedgerSummary_IDMismatch; do
  printf '%s\n' "$OUT" | grep -q -- "--- PASS: $t" || { echo "nao executou: $t"; exit 1; }
done
```
Nota transversal: e o **mesmo** padrao que eu identifiquei no ORQ-13 (F5). Nao e problema de um lane —
e do harness de `internal/handler`. Recomendo cartao proprio para fazer o `TestMain` **falhar** em vez
de `os.Exit(0)`, ou para segregar testes hermeticos em pacote sem `TestMain` de banco.

## 12. OBSERVACOES NAO BLOQUEANTES

**O1.** O gate esta **desarmado sem call site de producao**. Isso e correto para hoje, mas significa
que o valor de seguranca de W4 e **zero em runtime** ate W2/LANE-DB entregarem o resolver e o store. O
codigo diz isso; o cartao deve dizer tambem, para ninguem marcar "replay gate entregue" e assumir
protecao inexistente.

**O2.** `ErrAutopilotReplayGateIncomplete` protege o construtor, mas nada impede um chamador futuro de
construir via literal `&AutopilotService{...}` dentro do proprio pacote `service`, contornando a
validacao. Campos unexported blindam contra outros pacotes, nao contra o proprio. Nao e defeito hoje;
vale um teste de guarda se o pacote crescer.

## 13. NAO-AFIRMACOES
- READ-ONLY: nao editei, nao fiz amend, rebase, push, stage, commit; nao mutei board. `git status`
  vazio antes e depois. `go build`/`vet`/`test` nao alteram a arvore.
- Usei o inventario mecanico **somente** para pai, escopo e limpeza, e verifiquei seu SHA. O
  julgamento de comportamento e independente, sobre o codigo.
- **Nao verifiquei o comportamento do resolver "task terminal mais recente"**: ele nao existe neste
  lane. Avaliei o **contrato**, nao a implementacao.
- **Nao provei as tres assercoes do handler por execucao** — ver secao 11. Aprovei-as por leitura.
- Nao verifiquei se a mensagem de erro do gate implementado sera content-free: e obrigacao do
  implementador futuro.
- Nao rodei a suite completa de `internal/service` nem de `internal/handler`, apenas os testes
  filtrados pelos padroes indicados.
- Nao inspecionei `cmd/server/router.go`: por escopo, W4 nao o toca, e confirmei que nao foi tocado.
- Nao avaliei `c047c0b` como entrega isolada; avaliei o par, com foco na correcao.
- Nao criei, atribui nem comentei issue alguma.
