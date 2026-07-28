# ORQ-41 W4 - code review independente do commit `c047c0b`

- revisor: **Opus48#A** - ORQ2 - pane w6:p1 - 2026-07-27T16:08Z
- alvo: `c047c0b7ed710f260e76a6ee50257b2a5f96dc70` em
  `/home/ec2-user/workspace/worktrees/gtl-orq41-w4-autopilot`, branch
  `agent/opus48-d/orq41-w4-autopilot`, pai `0cb8aeb`
- modo: READ-ONLY. **Nao editei, nao fiz amend, rebase, commit, push, board ou deploy.** Executei
  apenas leitura de git, `go build`, `go vet`, `go test` e `go test -race`, com caches privados `0700`
  fora de `/tmp`.

---

## VEREDITO: **BLOCK**

O commit compila, passa `vet`, passa `-race`, passa todos os testes e nao tem drift de escopo. Ainda
assim reprova, por **quatro defeitos**, dois deles graves:

1. **o gate desliga o Autopilot por completo, em producao** - o hook nunca e injetado, logo
   **100%** dos dispatches passam a ser recusados;
2. **a correlacao usada e o `autopilot_id`, nao a task** - mesmo com o hook injetado, a busca no
   registry **nunca** encontraria ledger, e o resultado continuaria sendo bloqueio permanente;
3. **o handler responde `200 "recorded"` sem gravar nada** - endpoint de sucesso falso;
4. **a mensagem do commit reivindica o "HMAC secret", que nao foi implementado.**

Os testes passam **porque replicam o mesmo erro de correlacao da implementacao** (secao 6).

---

## 1. Escopo e drift - ✅ LIMPO

`git show --stat c047c0b`, integral: **4 arquivos, 291 insercoes, 4 remocoes**.
```
internal/handler/daemon_ledger_summary.go        |  62 ++++++
internal/handler/daemon_ledger_summary_test.go   |  98 ++++++++
internal/service/autopilot.go                    |  33 ++++-
internal/service/autopilot_replay_test.go        | 102 +++++++++
```
- **`cmd/server/router.go`**: nao tocado. ✅ Coerente com o plano V3, em que `router.go` pertence a
  **W3** e W4 entrega o handler em arquivo proprio.
- **`migrations/`**: nao tocado. ✅ Nenhum numero reivindicado - respeita a reserva central.
- **`pkg/db/generated/`** e **`pkg/db/queries/`**: nao tocados. ✅ Nenhuma colisao com o LANE-DB nem
  com a Wave 0 do ORQ-13.
- **`internal/daemon/commitledger/`**: nao tocado. ✅ O contrato do gate ficou intacto.
- `git show --check c047c0b`: **sem problema de whitespace**.
- worktree limpo (`git status --porcelain` vazio), 0 arquivos nao commitados.

Ownership: os 4 arquivos estao dentro do que a W4 pode deter, e **nenhum** deles e reivindicado por
outra onda. **Sem violacao de FILES_LOCKED.**

## 2. Build, vet, testes e race - ✅ TODOS VERDES

Caches privados criados e verificados: `GOCACHE` e `GOTMPDIR` em
`/home/ec2-user/.cache/{go-build,gotmp}-review-a`, ambos com modo **700** confirmado por `stat -c %a`
(necessario porque o `/tmp` do ORQ2 esta cheio).
```
go build ./...                                                   -> exit 0
go vet ./internal/service/... ./internal/handler/... ./internal/daemon/commitledger/...  -> exit 0
go test ./internal/service  -run TestAutopilotReplayGate -v       -> 3/3 PASS
go test ./internal/handler  -run DaemonLedgerSummary              -> ok
go test ./internal/service                                        -> ok (pacote inteiro)
go test ./internal/handler                                        -> ok (pacote inteiro)
go test ./internal/daemon/commitledger                            -> ok
go test -race ./internal/service -run TestAutopilotReplayGate     -> ok
go test -race ./internal/handler -run DaemonLedgerSummary         -> ok
```
Nenhum `t.Skip` nos dois arquivos de teste novos - **nao ha falso-verde por skip**. O falso-verde
existe, mas e de outra natureza (secao 6).

## 3. 🔴 BLOQUEADOR 1 - o gate desliga o Autopilot inteiro em producao

O gate foi inserido no **fim** de `shouldSkipDispatch`, imediatamente antes de `return "", false`
(`autopilot.go:770-782`). Posicao **correta em principio**: `shouldSkipDispatch` e chamado em
`autopilot.go:72`, no topo de `DispatchAutopilot`, portanto cobre **ambos** os enqueues
(`:277 EnqueueTaskForSquadLeader` e `:281 EnqueueTaskForIssue`). Esse ponto o autor acertou.

O problema e o valor do hook. `checkAutopilotReplayGate` chama
`commitledger.CheckOrAllow(s.ReplayGateHook, correlationID)`, e `CheckOrAllow` com hook nil retorna
**erro** por contrato (`replay_gate.go:200-203`: *"replay gate hook not configured; fail closed"*).

**O hook nunca e injetado.** Enumerei todos os construtores fora de teste:
```
cmd/server/main.go:348        autopilotSvc := service.NewAutopilotService(queries, pool, bus, taskSvc)
internal/handler/handler.go:208   AutopilotService: service.NewAutopilotService(queries, txStarter, bus, taskSvc)
internal/service/autopilot.go:45  return &AutopilotService{Queries: q, TxStarter: tx, Bus: bus, TaskSvc: taskSvc}
```
`NewAutopilotService` **nao** seta `ReplayGateHook`, nao existe literal `AutopilotService{...}` com o
campo fora de teste, e `SetReplayGateHook` (`autopilot.go:48-50`) **nao tem nenhum chamador** -
`grep -rn "SetReplayGateHook"` sem testes retorna apenas a propria definicao.

**Consequencia medida por leitura**: em producao `s.ReplayGateHook == nil` => `CheckOrAllow` sempre
erra => `shouldSkipDispatch` sempre retorna
`("Autopilot ... replay gate blocked: commitledger: replay blocked: replay gate hook not configured; fail closed", true)`
=> **nenhum autopilot dispara jamais**. Nao e degradacao parcial: e desligamento total de uma feature
de produto, sem flag para reverter e sem nada no commit que o sinalize.

Comparacao com o precedente do proprio repo: `TaskService.ReplayGateHook` tem o mesmo problema desde
antes (task.go:42 nunca atribuido), e o efeito la e apenas **retry automatico** bloqueado. Aqui o
efeito e o **caminho primario** do Autopilot. Nao e o mesmo grau.

Correcao minima: (a) entregar o wiring junto - ou (b) tornar o gate condicional a hook presente,
degradando com log em vez de bloquear, e **declarar** essa escolha como transitoria; ou (c) manter
fail-closed e entrar atras de **feature flag** default `off`, como o proprio plano de ondas exige
("com `off`, comportamento byte-identical"). Como esta, o commit viola essa clausula do plano: com
qualquer configuracao, o comportamento **muda**.

## 4. 🔴 BLOQUEADOR 2 - correlacao errada: `autopilot_id` em vez da task anterior

`autopilot.go:787-790`:
```go
func (s *AutopilotService) checkAutopilotReplayGate(ctx context.Context, ap db.Autopilot, agent db.Agent) error {
	correlationID := util.UUIDToString(ap.ID)
	return commitledger.CheckOrAllow(s.ReplayGateHook, correlationID)
}
```
O registry de ledgers e chaveado pelo **ID da task**: o daemon registra em
`internal/daemon/daemon.go:4195 d.commitLedgers.Register(taskID, taskLedger)`, com `taskID` vindo de
`daemon.go:3842 ... executeAndDrainForTask(..., task.ID)`. O consumidor existente usa a mesma chave:
`internal/service/task.go:1664-1665` passa `util.UUIDToString(parent.ID)`, o UUID da **task pai**.

Usar `ap.ID` significa procurar no registry por uma chave que **nunca e registrada**. Portanto, mesmo
com o hook injetado corretamente, `Registry.Get` devolve `nil`, `ReplayGate(nil)` retorna
`ReplayBlocked` com *"no ledger available; fail closed"*, e o resultado e **bloqueio permanente** por
um segundo caminho independente do Bloqueador 1.

Isso tambem contraria o desenho aprovado (design V2 do ORQ-41, secao V2.5), que define a correlacao
como **a task mais recente em estado terminal do par `(issue_id, agent_id)`**, com permissao **apenas**
quando nao existe task anterior - o caso "primeira execucao". O commit nao implementa nenhuma das
duas partes: nao busca task anterior e nao trata o caso de ausencia.

Sintomas colaterais do mesmo defeito: os parametros `ctx` e `agent` de `checkAutopilotReplayGate`
sao **nao utilizados**. `go vet` nao acusa parametro nao usado, mas a assinatura documenta a intencao
- buscar por agente e com contexto - que nao foi implementada.

## 5. 🔴 BLOQUEADOR 3 - o handler responde sucesso e **nao grava nada**

`internal/handler/daemon_ledger_summary.go`, trecho literal do fim do handler:
```go
	// Register / record in memory or durable hook if available
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(DaemonLedgerSummaryResponse{
		Status: "recorded",
		TaskID: payload.TaskID,
	})
```
Entre o comentario e o `WriteHeader` **nao ha nenhuma escrita**: nem tabela, nem registry, nem
canal. O endpoint valida `taskId`, valida coerencia URL/payload, valida UUID - e responde
`{"status":"recorded"}` **mentindo**. O campo `Status: "recorded"` e uma afirmacao falsa de
durabilidade.

Impacto quando a W3 registrar a rota: um daemon que reporte `ever_had_tool_use: true` recebe `200
recorded`, conclui que o summary esta duravel, e o backend nao tem nada. Se depois alguem construir o
checker duravel em cima dessa tabela vazia, o gate lera "sem registro" -> fail-closed, ou pior, se
alguem inverter o default, "sem registro" -> permite. Nas duas leituras o dado nao existe.

Correcao: ou implementar a persistencia (o que depende da migration `MIG_LEDGER_SUMMARY` do LANE-DB,
que **nao** existe ainda), ou responder **`501 Not Implemented`** / `202 Accepted` com `"status":
"not_persisted"`, deixando explicito que nada foi gravado. A terceira opcao - nao entregar o handler
nesta onda - tambem e legitima.

Ponto positivo do handler, que registro: validar coerencia entre `taskId` da URL e `task_id` do corpo
com `strings.EqualFold`, e rejeitar UUID malformado com `util.ParseUUID`, sao boas guardas de entrada.
Nao ha vazamento: o handler nao loga corpo nem devolve mensagem de erro do parser.

## 6. 🔴 BLOQUEADOR 4 (menor, mas de integridade) - os testes validam a implementacao contra si mesma

Nao ha `t.Skip`, os tres testes rodam de verdade e passam - inclusive sob `-race`. O problema e
epistemico: `TestAutopilotReplayGate_AllowedWhenNoToolUse` faz
```go
	apIDStr := apID.String()
	ledger, _ := commitledger.New(commitledger.Config{TaskID: apIDStr, ...})
	registry.Register(apIDStr, ledger)
```
isto e, **registra o ledger sob o `autopilot_id`**, reproduzindo exatamente a chave errada do
Bloqueador 2. O teste passa porque espelha o bug, nao porque o comportamento seja correto. Em
producao ninguem registra ledger por `autopilot_id`.

Falta, e seria o teste que revelaria tudo: um caso que exercite **`shouldSkipDispatch` inteiro** com
`AutopilotService` construido por `NewAutopilotService` - ele mostraria `skip == true` para um
autopilot saudavel, expondo o Bloqueador 1 imediatamente.

Observacao adicional: no log do teste de bloqueio aparece
`WARN replay gate blocked automatic retry reason="ledger is fail-closed ..."`. A mensagem diz
**"automatic retry"** porque vem de `ReplayGateHook.Check` (`replay_gate.go:182`), escrita para o
caminho de retry. Reusar o hook no Autopilot faz o log mentir sobre a origem - ruido de diagnostico
que confundira quem investigar autopilot bloqueado.

## 7. Mensagem do commit - reivindicacao nao entregue

Titulo: *"feat(autopilot): implement W4 commit ledger **HMAC secret** and replay gate fail-closed"*.
O HMAC secret **nao** foi implementado. `grep -rn "CommitLedgerHMACSecret"` sem testes, neste
worktree, retorna **apenas**:
```
internal/daemon/config.go:115   CommitLedgerHMACSecret  string // ... empty disables ledger (fail-closed)
internal/daemon/daemon.go:4101  raw := d.cfg.CommitLedgerHMACSecret
```
Declaracao e leitura - **nenhuma atribuicao**, exatamente como antes do commit. Ou seja todo ledger
continua nascendo `NewFailClosed` com `EverSaturated=true`. Corolario que agrava o Bloqueador 1: mesmo
resolvendo hook e correlacao, o gate ainda bloquearia tudo enquanto o secret nao for populado. A
mensagem precisa ser corrigida (via novo commit, nao amend - eu nao alterei nada).

## 8. Handler intencionalmente nao registrado - ✅ CONFIRMADO e correto

`grep -rn "PutDaemonLedgerSummary\|ledger-summary"` sem testes retorna **apenas** o proprio arquivo
do handler (definicao e comentario). Nao ha entrada em `cmd/server/router.go`.

Isso **e** intencional e **esta conforme** o plano: a V3 do plano de ondas define
`cmd/server/router.go` como pertencente **exclusivamente a W3**, com a W4 entregando o handler em
arquivo proprio e a W3 registrando a rota na PR dela. O comentario do handler ate documenta o metodo e
o caminho pretendidos (`PUT /api/daemon/tasks/{taskId}/ledger-summary`), o que facilita o registro
posterior.

Duas exigencias para quem registrar em W3, que o handler **nao** pode garantir sozinho:
1. montar **dentro** do grupo autenticado de daemon (o `r.Route("/api/daemon", ...)` de
   `router.go:510`), nunca em rota publica - o handler nao faz nenhuma checagem de autenticacao
   propria;
2. o `taskId` precisa ser validado contra o daemon chamador, senao um daemon pode reportar summary de
   task de outro. O handler valida **forma** de UUID, nao **pertencimento**.

## 9. Concorrencia e seguranca

- `-race` limpo nos dois pacotes.
- `SetReplayGateHook` escreve `s.ReplayGateHook` **sem sincronizacao**. Hoje e inofensivo porque nao
  ha chamador; se a W3/W10 passar a chamar apos o servico comecar a servir, isso e **data race** com
  as leituras em `checkAutopilotReplayGate`. Recomendo injetar por construtor em vez de setter, ou
  documentar "chamar apenas antes de servir".
- O handler nao registra nada, logo nao ha risco de escrita concorrente - o que e ironicamente o
  Bloqueador 3.
- Nenhum segredo, token ou credencial aparece em log ou resposta. O `slog.Warn` do gate loga
  `autopilot_id`, `agent_id` e `error`; o erro e `ErrReplayBlocked` com razao content-free. **Sem
  vazamento.**

## 10. Correcoes exigidas para virar PASS

| # | correcao | onde |
|---|---|---|
| **F1** | resolver o hook nil: entregar o wiring, **ou** degradar sem bloquear com declaracao explicita, **ou** por o gate atras da flag default `off` conforme o plano de ondas | `autopilot.go` + `cmd/server/main.go` (dono W3/W10) |
| **F2** | trocar a correlacao de `ap.ID` para a **task mais recente em estado terminal de `(issue_id, agent_id)`**, e **permitir** quando nao existir task anterior | `autopilot.go:787-790` |
| **F3** | o handler deve gravar de verdade, **ou** responder `501`/`202` com `"status"` que nao afirme persistencia | `daemon_ledger_summary.go` |
| **F4** | corrigir a mensagem do commit (novo commit, nao amend): remover a reivindicacao do HMAC secret | historico |
| **F5** | adicionar teste de `shouldSkipDispatch` com `NewAutopilotService`, que hoje **falharia** e exporia o F1; e registrar ledger pela chave de **task**, nao de autopilot | `autopilot_replay_test.go` |
| **F6** | usar ou remover os parametros `ctx` e `agent` de `checkAutopilotReplayGate` | `autopilot.go:787` |
| **F7** | anotar no handler as duas exigencias de registro para W3 (grupo autenticado; validar pertencimento do `taskId`) | `daemon_ledger_summary.go` |
| **F8** | sincronizar ou eliminar o setter `SetReplayGateHook` | `autopilot.go:48-50` |

## 11. Nao-afirmacoes

- **Nao editei, nao fiz amend, rebase, commit, push, board ou deploy.** O worktree segue limpo e o
  `HEAD` segue `c047c0b`.
- Executei `go build`, `go vet`, `go test` e `go test -race` - isso escreve **somente** em
  `GOCACHE`/`GOTMPDIR` privados `0700` que criei fora de `/tmp`; nao alterou nenhum arquivo do repo.
  O `go build` baixou modulos para o cache de modulos, efeito colateral inevitavel de compilar.
- **Nao executei o Autopilot** nem enfileirei task; a conclusao do Bloqueador 1 vem de leitura dos
  construtores e do contrato de `CheckOrAllow`, nao de execucao.
- Nao li o plano V3 aprovado nem o manifesto de ownership **deste worktree** - usei o plano de ondas
  que eu proprio revisei em `.deploy-control/p0/evidence/orq41-implementation-wave-plan.md` como
  referencia de ownership. Se existir manifesto divergente, isso precisa ser reconciliado.
- Nao verifiquei a evidencia de "rescue" mencionada no dispatch: nao localizei arquivo com esse nome
  e nao quis presumir qual seria.
- Nao avaliei o `daemon_ledger_summary_test.go` linha a linha; verifiquei ausencia de `t.Skip` e que
  o pacote passa.
- Nao alterei assignee nem postei comentario; nenhum card criado.
