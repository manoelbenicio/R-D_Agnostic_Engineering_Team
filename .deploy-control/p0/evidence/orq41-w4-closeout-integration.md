# ORQ-41 W4 - CLOSEOUT DE INTEGRACAO de `e0b0155` (READ-ONLY)

- **Autor:** Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T16:44Z
- **Cartao:** `ORQ-41` Wave W4
- **Commit avaliado:** `e0b0155d0f0d46a9da3724ecbfc51a5167bd2ebe`
- **Pai:** `c047c0b7ed710f260e76a6ee50257b2a5f96dc70` (o commit BLOQUEADO, preservado, sem amend)
- **Worktree:** `/home/ec2-user/workspace/worktrees/gtl-orq41-w4-autopilot`, branch `agent/opus48-d/orq41-w4-autopilot`
- **Modo:** READ-ONLY. **Nao** fiz amend, push, merge, rebase, stage, commit nem mutei board.

# VEREDITO FINAL: **CODE_PASS / EVIDENCE_PASS**

O `EVIDENCE_BLOCK` da secao 11 da revisao adversarial esta **fechado**. O bloqueio nunca foi sobre o
codigo de `e0b0155`; era sobre a **prova** dos testes de handler, que rodavam contra `ok` vazio.

---

## 1. EVIDENCIA IMUTAVEL DO GATE

Comando exato, conforme retido e autorizado:
```
go test -race ./internal/handler -run 'PutDaemonLedgerSummary' -count=1 -v
```
Saida integral, preservada em `/home/ec2-user/.cache/orq41_gate_evidence.log`
(**sha256 `c7be43e5465a489c362372d2d3c64e6e1688db6aa4695dac005c0fe3b02a2ee1`**):
```
EmailService: disabled — configure SMTP_HOST or RESEND_API_KEY before sending email
=== RUN   TestPutDaemonLedgerSummary_WellFormedPayloadFailsClosedWithoutClaimingPersistence
--- PASS: TestPutDaemonLedgerSummary_WellFormedPayloadFailsClosedWithoutClaimingPersistence (0.00s)
=== RUN   TestPutDaemonLedgerSummary_InvalidUUID
--- PASS: TestPutDaemonLedgerSummary_InvalidUUID (0.00s)
=== RUN   TestPutDaemonLedgerSummary_IDMismatch
--- PASS: TestPutDaemonLedgerSummary_IDMismatch (0.00s)
PASS
ok  	github.com/multica-ai/multica/server/internal/handler	1.821s
```
| criterio de aceite | exigido | medido |
|---|---|---|
| PASS nominais | 3 | **3** |
| `Skipping tests:` | 0 | **0** |
| `WARNING: DATA RACE` | 0 | **0** |
| exit code | 0 | **0** |

As tres linhas `=== RUN` provam **execucao**, nao apenas ausencia de falha. E a ausencia da linha
`Skipping tests:` prova que o `TestMain` **nao** abortou via `os.Exit(0)`.

### 1.1 Contraste com a medicao anterior - o falso verde, agora documentado
```
sem banco : ok  internal/handler  1.841s   |  --- PASS: 0   |  Skipping tests: 1
com banco : ok  internal/handler  1.821s   |  --- PASS: 3   |  Skipping tests: 0
```
Os dois `ok` sao indistinguiveis por exit code. **So a contagem nominal separa prova de aparencia.**
Este par de medicoes fica registrado como o artefato que justifica a regra: neste pacote, `ok` nunca e
evidencia.

## 2. INFRA EFEMERA USADA, E O QUE FALHOU ANTES

**Tentativa 1, descartada:** cluster PostgreSQL 17 privado no proprio ORQ2, em diretorio `0700` fora de
`/tmp`, porta efemera em loopback com socket privado. Falhou na migration `001` com
`extension "pgcrypto" is not available` — `postgresql17-contrib` nao esta instalado e instalar pacote e
proibido. Cluster parado e diretorio removido.

**Tentativa 2, usada:** container efemero no ORQ1 com a imagem **ja presente localmente**, sem
`docker pull`, pinada por digest
`pgvector/pgvector:pg17@sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0`,
nome `orq41-gate-pg`, flag `--rm`, publicada **somente** em `127.0.0.1:47471`, credencial descartavel.
Acesso do ORQ2 por **tunel SSH privado**; nenhuma porta exposta em tailnet.

**Schema:** `go run ./cmd/migrate up` aplicado ao banco **efemero**, 68 tabelas em `public`. Necessario
porque o `TestMain` do pacote cria fixtures com `INSERT` em `"user"` e `workspace`
(`handler_test.go:73`, `:97-110`). **Nenhuma migration tocou banco de produto.**

**Caches:** `GOCACHE`, `GOTMPDIR` e `TMPDIR` em `/home/ec2-user/.cache/orq41-gate`, criados com
`install -d -m 0700` e **verificados** por `stat -c %a` = `700`, todos fora de `/tmp`.

## 3. TEARDOWN E AUSENCIA DE RESIDUO - VERIFICADO

| recurso | estado verificado |
|---|---|
| worktree | `git status --porcelain` = **0 linhas**; HEAD = `e0b0155d0f0d46a9da3724ecbfc51a5167bd2ebe` |
| cache privado `orq41-gate` | **removido** — `ls` retorna `No such file or directory` |
| cluster PG local (ORQ2) | **parado** — nenhum processo `/usr/bin/postgres -D /home/ec2-user/.cache` |
| tunel SSH | **fechado** — nenhum processo `ssh ... -L ...:47471` |
| container `orq41-gate-pg` (ORQ1) | **removido** — `docker ps -a \| grep -c` = **0** |
| porta `47471` no ORQ1 | **livre** — `ss -ltn \| grep -c 47471` = **0** |
| containers de produto (ORQ1) | **intactos**: backend `75e4416f06e9`, frontend `1b3b6c9ee32a`, omniroute `2fb3fd57e885`, postgres-1 `2a4a84897363` |
| disco | ORQ2 raiz **89% / 5.8G livres**, `/tmp` **5% / 7.4G livres**; ORQ1 raiz **80% / 4.9G livres** — iguais ao estado do sinal de limpeza |

Nota de metodo: a primeira varredura de processos casou o **proprio comando** de varredura. Refiz
excluindo `bash -c` e `grep -E`, e o resultado real e **vazio**. Registro para nao induzir leitura
errada.

Unico artefato deliberadamente **preservado**: `/home/ec2-user/.cache/orq41_gate_evidence.log`, 559
bytes, fora do diretorio removido, com o sha256 da secao 1.

## 4. A FEATURE PERMANECE **DESLIGADA EM RUNTIME**

Isto e o ponto que nao pode se perder no closeout: **`e0b0155` esta aprovado, e o replay gate nao
protege nada hoje.**

Cadeia de fatos, todos verificados na revisao:
1. `NewAutopilotService` — o construtor de producao — e **byte-identico** ao de `0cb8aeb` e **nao**
   inicializa `replayGate` nem `replayCorrelation`;
2. `ReplayGateArmed()` exige **ambos** non-nil; com o construtor de producao, retorna `false`;
3. `replayGateSkipReason` retorna `"", false` imediatamente quando desarmado, **antes de qualquer I/O**;
4. **nao existe call site de producao** para `NewAutopilotServiceWithReplayGate`;
5. o handler `PutDaemonLedgerSummary` responde **`503`** e nao persiste nada.

Portanto o valor de seguranca entregue por W4 em runtime e **zero**. O que W4 entregou e a **costura**:
contratos, construtor seguro, correlacao correta e testes que travam os dois defeitos de `c047c0b`.
Marcar "replay gate entregue" e assumir protecao seria erro de leitura — e o proprio codigo declara isso.

## 5. PRE-REQUISITOS EXATOS DE INTEGRACAO

### 5.1 Para integrar `e0b0155` (merge do commit, sem ligar a feature)
| # | pre-requisito | estado |
|---|---|---|
| I1 | `c047c0b` preservado, sem amend, `e0b0155^` == `c047c0b` | **OK** (verificado) |
| I2 | escopo restrito a 4 arquivos, sem router/migration/generated/query/commitledger | **OK** (verificado) |
| I3 | `go build ./...` e `go vet` exit 0 | **OK** |
| I4 | `go test -race ./internal/service` — 7 PASS, 0 FAIL, 0 race | **OK** |
| I5 | 3 PASS nominais em `./internal/handler` com banco, 0 skip | **OK** (secao 1) |
| I6 | worktree limpo, sem residuo privado | **OK** (secao 3) |
| I7 | autorizacao de merge do owner | **PENDENTE** — nao e minha |
| I8 | revisor de integracao distinto de mim | **PENDENTE** |

I1..I6 estao fechados. **I7 e I8 nao sao meus** e permanecem abertos.

### 5.2 Para **armar** a feature em runtime (fora do escopo de W4)
Nenhum destes existe hoje; todos sao pre-condicoes, e a ordem importa:
1. **LANE-DB**: migration `task_ledger_summary` **reservada** pelo registrar e aplicada, com o
   `upsert` gerado por `sqlc`. Sem store duravel, o handler nao pode deixar de responder `503`.
2. **W2**: implementacao de `AutopilotReplayCorrelation.PriorTaskCorrelationID`, que precisa resolver a
   **task terminal mais recente** para o par issue+agent. W4 declarou o contrato e **nao** o
   implementou; a ordenacao por terminalidade tem de ser provada por quem implementar.
3. **Checker durável** de `AutopilotReplayGate.AllowReplay`, com a exigencia explicita de mensagem de
   erro **content-free** — o contrato pede, e o cumprimento e do implementador.
4. **W3**: registrar a rota `PUT /api/daemon/tasks/{taskId}/ledger-summary` em `cmd/server/router.go`
   **dentro do grupo autenticado de daemon**, e validar que `{taskId}` **pertence** ao daemon chamador.
   O handler valida **shape** de UUID, nao posse — esta escrito no proprio arquivo.
5. **Call site**: trocar `NewAutopilotService` por `NewAutopilotServiceWithReplayGate` no ponto de
   composicao, o que e **decisao de deploy** e nao refactor: passa a existir caminho que **recusa**
   dispatch.
6. **Gate de fila** antes de ligar: `active_tasks = 0` nos quatro estados de
   `migrations/109:13-15`, com saida anexada.
7. **Plano de reversao** que nao dependa de reverter o commit: como armar exige os dois colaboradores,
   desarmar e voltar ao construtor de producao — e isso precisa estar escrito antes de ligar.
8. **Primeira execucao**: validar explicitamente que a primeira dispatch de um autopilot **novo** nao e
   bloqueada (`ok == false` -> allow), que e o caso que `c047c0b` quebrava.

### 5.3 Dependencia transversal, nao do ORQ-41
`internal/handler/handler_test.go:38-54` faz `os.Exit(0)` **antes** de `m.Run()` quando o banco nao
responde. Isso produz **falso verde para qualquer lane** que teste esse pacote sem banco, nao apenas
para W4. Merece cartao proprio: fazer o `TestMain` **falhar** em vez de sair `0`, ou segregar os testes
hermeticos em pacote sem `TestMain` de banco. E o mesmo padrao que eu registrei no ORQ-13 como F5.

## 6. NAO-AFIRMACOES
- READ-ONLY: **nao** fiz amend, push, merge, rebase, stage, commit; **nao** mutei board.
- Nao toquei banco de **producao** em nenhum momento: schema e testes foram contra o banco efemero.
- Nao criei, recriei, parei nem removi container de produto; criei e removi apenas `orq41-gate-pg`.
- Nao instalei pacote: a falta de `pgcrypto` no ORQ2 foi contornada trocando de host, nao instalando.
- Nao fiz `docker pull`: a imagem ja existia no ORQ1 e foi referenciada por digest.
- Nao expus porta em tailnet: publicacao apenas em `127.0.0.1` do ORQ1, acesso por tunel SSH.
- Removi **apenas** o cache privado novo. Os tres caches anteriores — `go-build-i03` 817M,
  `go-build-orq41` 812M, `sqlcbuild` 76M, ~1.7G — **seguem no disco** aguardando decisao do GTL.
- Nao rodei a suite completa de `internal/handler` nem de `internal/service`; apenas os filtros do gate.
- **Nao verifiquei** o comportamento do resolver "task terminal mais recente": ele nao existe neste
  lane. Avaliei contrato, nao implementacao.
- Nao avaliei `c047c0b` como entrega isolada; o veredito e sobre o par, com foco na correcao.
- Nao autorizo merge nem me declaro revisor de integracao: I7 e I8 seguem abertos.
- Nao criei, atribui nem comentei issue alguma.
