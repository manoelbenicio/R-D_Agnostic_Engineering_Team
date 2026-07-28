# ORQ-41 W4 — correção dos 4 bloqueadores de `c047c0b` (novo commit `e0b0155`)

- Card: **ORQ-41** · UUID `666f1ead-7fe9-4051-bab9-5d0a936c4701`
- Worktree: `/home/ec2-user/workspace/worktrees/gtl-orq41-w4-autopilot`, branch `agent/opus48-d/orq41-w4-autopilot`
- Base do meu trabalho: `c047c0b` (revisado como **BLOCK** por Opus48#A em
  `orq41-w4-c047c0b-code-review.md`), que está sobre `0cb8aeb`
- **Novo commit**: `e0b0155` — *"fix(autopilot): keep replay gate off-path and correlate by prior task"*
  — **sem amend**, `c047c0b` preservado no histórico
- Executor: Codex56#A (`w7:p3`) · UTC 2026-07-27T16:20Z
- Não houve push, mutação de board, deploy, migration, `generated`, `queries` ou `router.go`.

## 1. Escopo: 4 arquivos, todos dentro do FILES_LOCKED da W4

```text
internal/service/autopilot.go              +130 / -0 linhas removidas vs base
internal/service/autopilot_replay_test.go  reescrito (7 testes)
internal/handler/daemon_ledger_summary.go   fail-closed + requisitos de W3
internal/handler/daemon_ledger_summary_test.go  renomeado para dizer a verdade
```

Verificação de que **nenhuma linha legada foi removida**:
`git diff 0cb8aeb -- internal/service/autopilot.go | grep -c "^-[^-]"` → **0**.
Toda a lógica de admissão anterior (assignee, leader, `AgentReadiness`, gate de agente privado)
está intacta, byte a byte.

## 2. Bloqueador 1 — desligamento total do Autopilot: **resolvido por design, não por remoção**

Antes: o gate chamava `commitledger.CheckOrAllow(s.ReplayGateHook, …)` e `CheckOrAllow` com hook nil
retorna erro por contrato (`replay_gate.go:200-203`), e **nenhum** construtor injetava o hook →
100% dos dispatches recusados em produção.

Agora:

- `NewAutopilotService` (o construtor usado em `cmd/server/main.go:348` e `handler.go:208`, **não
  alterados**) deixa o gate **desarmado**;
- `replayGateSkipReason` retorna `("", false)` de imediato quando desarmado — é o **único** ponto
  capaz de recusar por replay;
- armar exige `NewAutopilotServiceWithReplayGate(..., gate, correlation)`, que **falha** com
  `ErrAutopilotReplayGateIncomplete` se faltar qualquer um dos dois colaboradores (armar pela metade
  ou desliga a proteção em silêncio ou bloqueia tudo).

O comportamento "ligado" fica **explicitamente diferido atrás de duas interfaces**, porque depende do
store durável (`task_ledger_summary`, LANE-DB) e da busca da task anterior (W2) — nada disso pertence
à W4 no manifesto V3, que dá à W4 apenas `service/autopilot.go` e `handler/daemon_ledger_summary.go`.

## 3. Bloqueador 2 — correlação errada: **corrigido e provado por mutação**

`AutopilotReplayCorrelation.PriorTaskCorrelationID(ctx, ap, agent) (id string, ok bool, err error)`
resolve a **task anterior**. A ordem no gate é: resolver correlação → se `ok == false` (primeira
execução) **permitir sem consultar o gate** → consultar o gate com o **ID da task**. Erro do
resolvedor **falha fechado**, mas só em deployment armado.

O `autopilot_id` nunca é passado como chave de ledger. Prova por **mutação real**, executada e
revertida:

```text
mutação: AllowReplay(ctx, util.UUIDToString(ap.ID))   # o defeito original
--- FAIL: TestReplayGateSkipReason_CorrelatesByPriorTaskNotAutopilotID
    gate must be consulted with the prior task ID "3fb4f173-…", got "979930c4-…"
    gate must never be consulted with the autopilot ID "979930c4-…"
revert: git checkout -- autopilot.go   -> worktree limpo
```

Ou seja o teste **não** espelha a implementação: ele reprova exatamente o comportamento de `c047c0b`.

## 4. Bloqueador 3 — handler mentindo sucesso: **resolvido**

`PutDaemonLedgerSummary` valida forma (URL vazia, JSON, coerência URL×payload, UUID) e então responde
**503** `{"error":"ledger summary persistence is not configured; fail closed"}`. O tipo
`DaemonLedgerSummaryResponse` e a string `"recorded"` **não existem mais**. Também documentei no
próprio arquivo as duas exigências que só a W3 pode cumprir: montar dentro do grupo autenticado de
daemon e validar **pertencimento** do `taskId` ao daemon chamador (o handler valida forma, não posse).

## 5. Bloqueador 4 — reivindicação de HMAC: **removida**

A mensagem de `e0b0155` declara explicitamente que **nenhum** trabalho de HMAC é reivindicado:
`CommitLedgerHMACSecret` continua declarado em `internal/daemon/config.go:115` e lido em
`daemon.go`, **sem nenhuma atribuição**, e `daemon/config.go` **não** está no FILES_LOCKED da W4 no
manifesto V3. Não posso corrigir a mensagem de `c047c0b` sem amend, que está proibido; o registro da
divergência fica aqui e no check-out. **`orq41-w4-autopilot-rescue-implementation.md` continua
afirmando "HMAC secret" — é evidência de outro agente, fora do meu conjunto de arquivos; sinalizo para
o GTL corrigir na fonte.**

## 6. Corolário do review que também foi tratado

| item do review | tratamento |
|---|---|
| **F6** `ctx`/`agent` não usados | agora ambos são usados: `ctx` vai ao resolvedor e ao gate; `agent` vai ao resolvedor e ao log |
| **F8** setter sem sincronização | `SetReplayGateHook` **removido**; injeção só por construtor, campos unexported e imutáveis → não há escrita após o serviço começar a servir |
| log dizendo "automatic retry" | não reusamos mais `ReplayGateHook`; as mensagens são do Autopilot (`"autopilot admission: replay gate blocked"`, `"replay correlation unavailable"`) |
| vazamento em razão de admissão | teste garante que a razão **não** contém o ID de correlação |

## 7. Gates executados (Go 1.26.1, caches privados `0700` fora de `/tmp`)

```text
/home/ec2-user/.private-tmp/orq41-w4{,/gocache,/gotmp}   modo 700 confirmado

gofmt -l (4 arquivos)                                   -> vazio
go build ./...                                          -> exit 0
go vet ./internal/service/... ./internal/handler/... ./internal/daemon/commitledger/...  -> exit 0

go test ./internal/service            -count=1          -> ok 0.013s
go test ./internal/handler            -count=1          -> ok 0.072s  (ver §8: SKIP silencioso)
go test ./internal/daemon/commitledger -count=1         -> ok 0.138s
go test ./internal/service            -race -count=1    -> ok 1.044s
go test ./internal/handler            -race -count=1    -> ok 1.873s  (idem §8)
go test ./internal/daemon/commitledger -race -count=1   -> ok 1.195s

go test ./internal/service -run ReplayGate -v            -> 7/7 PASS
git diff --check                                         -> exit 0
git status --porcelain                                   -> vazio (0 tracked, 0 untracked)
```

Os 7 testes: gate desarmado no construtor de produção; **serviço de produção não bloqueia enquanto
desarmado**; recusa de armar pela metade; **correlação por task e nunca por autopilot**; primeira
execução permitida sem consultar o gate; bloqueio quando o gate recusa (sem vazar correlação);
fail-closed quando o resolvedor erra.

## 8. Achado novo e importante: os testes do pacote `handler` **não executam** aqui

`internal/handler/handler_test.go:38-53` — `TestMain` faz `os.Exit(0)` quando o Postgres não responde:

```text
Skipping tests: database not reachable: failed to connect to `user=multica database=multica`:
127.0.0.1:5432 … connection refused
ok  github.com/multica-ai/multica/server/internal/handler  0.066s
```

Isto é **falso-verde de pacote**: o `ok` não significa que um único teste rodou. Vale para os testes de
`c047c0b` também — o PASS de handler citado no review anterior não foi execução real. No ORQ2 não há
listener em 5432 nem Docker, então não há como executar sem mudar infraestrutura, o que eu não faço sem
autorização.

Para não deixar o contrato do handler sem execução, rodei um **harness temporário** (`go run`, criado e
**apagado** em seguida; `git status` limpo depois, incluindo untracked):

```text
well_formed_fails_closed        status=503 want=503 body={"error":"ledger summary persistence is not configured; fail closed"}
invalid_uuid_rejected           status=400 want=400
url_payload_mismatch_rejected   status=400 want=400
unknown_field_rejected          status=400 want=400
malformed_json_rejected         status=400 want=400
ALL_HANDLER_CHECKS_PASS
```

Nenhuma resposta contém `"recorded"`. Recomendo card próprio para o `TestMain` do pacote `handler`
falhar (ou marcar `t.Skip` por teste) em vez de sair 0 — hoje qualquer regressão em handler passa
silenciosamente em CI sem Postgres.

## 9. Não-alegações

- Não fiz amend, rebase, push, PR, merge, deploy, nem mutação de board; `c047c0b` intacto e o novo
  commit é `e0b0155`.
- Não toquei `migrations/`, `pkg/db/generated/`, `pkg/db/queries/`, `cmd/server/router.go`,
  `internal/service/task.go`, `internal/daemon/**` nem `commitledger/`.
- **Não implementei o comportamento ligado**: não existe implementação de `AutopilotReplayGate` nem de
  `AutopilotReplayCorrelation` em produção, e nenhum call site as injeta. Isso é deliberado e
  declarado.
- Não executei o Autopilot de verdade nem enfileirei task; as conclusões de admissão vêm de teste
  unitário e leitura.
- Os testes do pacote `handler` **não** foram executados pelo `go test` (§8); a verificação do handler
  vem do harness temporário e de leitura.
- Não corrigi a mensagem de `c047c0b` (exigiria amend) nem editei a evidência de outro agente que
  reivindica HMAC.
- Nenhum segredo lido ou impresso; nenhum valor de credencial em log, teste ou resposta.
