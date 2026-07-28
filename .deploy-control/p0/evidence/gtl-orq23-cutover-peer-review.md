# GTL-37 - peer review independente do plano de cutover ORQ-23

Revisor: Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T11:48Z
Documento revisado: `.deploy-control/p0/evidence/gtl-orq23-durable-cutover-plan.md`
(autor Agy-P0-A8 / wB:p2, 77 linhas, 3955 bytes, 2026-07-27T11:29:40Z)
Modo: READ-ONLY. Nao editei codigo, nao compilei, nao executei nenhum comando do plano, nao
reiniciei nada. Gravei somente este arquivo e o meu check-out.

# VEREDITO: **BLOCK**

Quatro defeitos bloqueantes, sendo o primeiro fatal por si so. Dois deles sao factuais e verificaveis
em segundos. O plano acerta o essencial de seguranca (nao usa `ff121b28`, marca os backups
pre-token-only como proibidos, classifica as mutacoes como STOP-AND-WAIT) mas o **comando de build
produziria um binario que a unit systemd nao consegue executar**.

| # | item auditado | resultado |
|---|---|---|
| 1 | nao usa `ff121b28` | **PASS** |
| 2 | nao usa backups pre-token-only known-bad | **PASS com ressalva** (2 binarios known-bad nao listados) |
| 3 | artefato reproduzivel | **BLOCK** (entrypoint errado + build nao reproduzivel + arvore suja) |
| 4 | rollback para baseline `88ca4f39` | **BLOCK** (o plano nao referencia esse baseline; ref nao resolve no repo) |
| 5 | 3 runtimes obrigatorios | **PASS com ressalva** (H2 e acao live sem gate) |
| 6 | prereqs de account / tier / usage | **BLOCK** (ausentes; so slots foram cobertos) |
| 7 | comandos e gates | **PASS com ressalva** (H2 e a fila nao gateados) |
| 8 | ausencia de live mutation no plano | **PASS** para mutacao; **BLOCK parcial** para acao live (H2) |

---

## BLOQUEANTE 1 - o comando de build gera o binario ERRADO (fatal)

O plano propoe, linha 26-28 (literal):
```bash
cd /home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server
/home/ec2-user/goroot/go/bin/go build -o /home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1.orq23-staging cmd/server/main.go
```

A unit exige a interface de CLI, nao a de servidor. `systemctl --user cat
multica-daemon-orq2-credential.service`, linha 37 (literal):
```
ExecStart=/home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1 daemon start --foreground --no-auto-update --server-url http://127.0.0.1:18080 --daemon-id orq2-credential-runtime-v1 --device-name "ORQ2 Credential Runtime"
```

O subcomando `daemon start` **nao existe** em `cmd/server`. Ele esta em `cmd/multica`:
```
$ grep -rIln 'daemon start|"daemon"' cmd/
cmd/multica/cmd_daemon.go
cmd/multica/cmd_login.go
cmd/multica/help.go
cmd/multica/cmd_daemon_test.go

$ grep -rln 'func main()' cmd/server/ cmd/multica/
cmd/server/main.go
cmd/multica/main.go
```
`cmd/server/main.go` e o **backend HTTP**; `cmd/multica/main.go` e a CLI que hospeda
`daemon start`. Um binario compilado de `cmd/server` nao aceita `daemon start` e a unit falharia no
`ExecStart`. O rollback salvaria a situacao, mas o cutover nao teria como passar.

Defeito somado, independente do primeiro: `go build ... cmd/server/main.go` aponta **um arquivo**, e
o pacote tem 15 arquivos `.go` nao-teste:
```
$ ls cmd/server/*.go | grep -v _test | wc -l
15
```
Compilar um unico arquivo de um pacote multi-arquivo falha com simbolos indefinidos. Ou seja o
comando nem chega a produzir binario. O correto e apontar o **pacote**, nao o arquivo, e o pacote
correto e `./cmd/multica`.

## BLOQUEANTE 2 - rollback nao esta ancorado no baseline `88ca4f39`

O plano define rollback como um **artefato binario**, secao 2, linha 37:
`multica-auth-credential-home-v1.gate1-20260727T102815Z`, 15.053.065 bytes, "ALVO UNICO DE ROLLBACK".
O documento **nao menciona `88ca4f39` em nenhuma linha** (`grep` por `88ca4f39` no plano: zero
ocorrencias).

E a referencia nao e resolvivel nesta arvore:
```
$ git log --oneline -1 88ca4f39
fatal: ambiguous argument '88ca4f39': unknown revision or path not in the working tree.
$ git log --oneline -1 ff121b28
fatal: ambiguous argument 'ff121b28': unknown revision or path not in the working tree.
```
Consequencias objetivas:
1. Nao ha rastreabilidade do binario de rollback para um commit. "15.053.065 bytes" e **tamanho**,
   nao identidade: dois binarios diferentes podem ter o mesmo tamanho, e o proprio inventario do
   ORQ2 prova isso (ver ressalva 2 abaixo: `.new` e `.pre-token-only` tem **exatamente** 21.344.766
   bytes cada). O plano nao registra nenhum `sha256`.
2. Como `ff121b28` tambem nao resolve aqui, a propria regra "nao usar a worktree corrompida
   `ff121b28`" nao e verificavel nesta arvore — ela pode se referir a outro clone/worktree. A regra
   e correta em intencao, mas o plano deveria dizer **onde** essa ref existe.
3. Sem ancoragem em commit, nao ha como provar que o binario de rollback corresponde ao baseline que
   o General-TL pediu. O `gate1` pode ser o baseline certo; o plano simplesmente nao demonstra.

Correcao minima exigida: registrar `sha256` de cada binario, e amarrar o binario de rollback ao
commit (por exemplo, gravando o commit no `-ldflags` do build e conferindo com `--version`).

## BLOQUEANTE 3 - artefato nao e reproduzivel, e a arvore nao esta limpa

O plano afirma, linha 20: "**Origem Codigo:** Diretorio principal de integracao ... `/server`" e o
diagrama diz "Repositorio Limpo server/". Medicao agora:
```
$ git status --porcelain -- multica-auth-work/ | grep -cE '\.(go|ts|sql)$'
20
```
Ha **20 arquivos de codigo modificados** e nao commitados na arvore (do escritor unico). Build a
partir dai nao e reproduzivel nem auditavel: o binario resultante nao corresponde a nenhum commit.

Faltam, no comando, todos os elementos que tornariam o artefato reproduzivel: `-trimpath`,
`-ldflags` com commit e data, `CGO_ENABLED` fixado, versao de Go declarada no artefato, e registro
de `sha256` do resultado. Sem isso, "artefato reproduzivel" nao se sustenta — e o requisito estava
explicito no dispatch.

## BLOQUEANTE 4 - prereqs de account, tier e usage estao ausentes

A secao 3 cobre **apenas slots**, e nisso esta correta: o mapeamento por provider
(`antigravity -> <slot>/home`, `kiro -> <slot>/xdg-data`, `codex -> <slot>/codex`) coincide com o que
eu medi e registrei no TEMA 2. Mas o dispatch pede prereqs de **account/tier/usage**, e o plano nao
trata nenhum dos tres:
1. **account**: `task_usage` nao tem `account_id`, e `assignments` esta vazia; o pacote Go
   `internal/daemon/rotation`, unico escritor dessas tabelas, nao existe mais no HEAD. O cutover
   passaria sem nenhuma atribuicao de custo por conta.
2. **tier**: `thinking_level` existe so em `agent` (`migrations/095_agent_thinking_level.up.sql:8`) e
   nao em `task_usage`. O plano cita `claude-opus-4-6-thinking` e `gemini-3.6-flash-high` como
   criterio H2, isto e depende de tier, mas nao declara o prereq que registra o tier.
3. **usage**: medicao do banco no GTL-03 — `task_usage` tem 123 linhas de `claude` e 5 de `codex`, e
   **zero linhas de `antigravity` e `kiro`**. Ou seja: os dois runtimes que o plano vai declarar
   "ONLINE" nao registram usage nenhuma hoje. Um cutover "PASS" com esse estado congela o defeito de
   custo em producao e o declara aprovado.

Nao pedi que o plano resolva isso. Exijo que ele **declare** o estado como prereq conhecido, porque
sem isso o gate de sucesso do cutover e mais fraco do que aparenta.

---

## RESSALVA 1 - allowlist de slots do Kiro esta ERRADA (fato medido)

Plano, secao 3, item 2 (literal): "**Kiro Slot Allowlist:** Estritamente restrito aos 4 slots ativos
`139`, `140`, `143`, `149`."

Medicao direta agora, presenca de `xdg-data/kiro-cli/data.sqlite3` por slot:
```
slot-139 KIRO
slot-140 KIRO
slot-142 KIRO
slot-143 KIRO
slot-149 KIRO
```
São **5**, nao 4. O `slot-142` tem credencial Kiro e foi omitido. Efeito pratico: uma allowlist que
exclui um slot valido reduz a capacidade do Kiro em 20% de forma silenciosa — nao gera erro, so
menos paralelismo. Nao e fatal, mas e um numero errado num documento que sera usado como referencia
operacional, e por isso precisa ser corrigido antes de virar procedimento.

## RESSALVA 2 - inventario de binarios incompleto, e tamanho nao e identidade

O plano lista 4 binarios. O diretorio tem 6:
```
$ ls -l /home/ec2-user/.local/lib/multica/bin/
15053065 Jul 27 10:37 multica-auth-credential-home-v1
15053065 Jul 27 10:28 multica-auth-credential-home-v1.gate1-20260727T102815Z
21344766 Jul 27 01:18 multica-auth-credential-home-v1.new                       <- NAO listado no plano
21333562 Jul 27 01:18 multica-auth-credential-home-v1.pre-agy-fix
21344766 Jul 27 01:18 multica-auth-credential-home-v1.pre-token-only-20260727T102815Z
21333458 Jul 26 23:52 multica-auth-credential-home-v1.previous                  <- NAO listado no plano
```
Dois pontos:
1. **`.new` tem exatamente o mesmo tamanho do `pre-token-only` known-bad** (21.344.766). O nome
   `.new` sugere "novo/bom" e o tamanho indica que pertence a classe known-bad. Um operador sob
   pressao pode escolher `.new` acreditando ser o candidato correto. O plano **precisa** nomear
   `.new` e `.previous` como proibidos, ou remove-los do escopo explicitamente.
2. O plano usa tamanho em bytes como criterio de classificacao ("Comprovadamente Bom" pelo tamanho).
   Esse par de arquivos com tamanho identico e a prova de que tamanho nao identifica binario.
   Exigir `sha256` nao e formalismo aqui, e a unica forma de o rollback ser deterministico.

## RESSALVA 3 - H2 e acao LIVE e nao esta na matriz de gates

Plano, secao 4 (literal): "**Criterio de Sucesso H2:** Invocacao de modelos via OmniRoute
(`claude-opus-4-6-thinking`, `claude-sonnet-4-6`, `gemini-3.6-flash-high`) com heartbeat atualizado."

Invocar modelo via OmniRoute e **chamada live de provedor**, com consumo de quota e custo real em
conta de terceiro. A matriz de gates da secao 5 cobre apenas D1 (build), D2 (restart de container) e
D3 (mv + restart de unit). H2 fica de fora, ou seja o plano contem uma acao live sem gate.

Isso e exatamente o padrao que o proprio repositorio proibe: o README diz que nao se deve usar
requisicao live de provedor como verificacao basica de saude da stack. H1 (`GET /api/runtimes`) e
suficiente e nao-live para "3 runtimes ONLINE" — e confirmei que a rota existe
(`cmd/server/router.go:927` `r.Route("/api/runtimes", ...)`). H2 deveria ser D4 na matriz, ou ser
substituido por verificacao nao-live.

## RESSALVA 4 - falta a pre-condicao de fila vazia antes do restart

O plano executa `systemctl --user restart` em D3 e no rollback, sem exigir fila vazia. A auditoria
T3 mediu o custo real dessa janela na troca anterior de daemon: **6 tasks com
`failure_reason=runtime_offline`** e **1 com `runtime_recovery`**. No momento do T3 havia zero tasks
`queued` e zero `running`, que e a janela correta. O plano deve exigir, como gate de entrada de D3,
a verificacao de que `agent_task_queue` nao tem linha em `queued` nem `running`.

## RESSALVA 5 - risco AGY nao mencionado, e ele pode invalidar H2

O T3 mediu que o AGY falha **antes** de chegar a executar, com erro literal
`execenv: prepare antigravity-home: seed per-account antigravity token dir: unsupported credential
path type .../slot-145/home/.gemini/antigravity-cli/cli.log` (idem `slot-146`), causado por
`antigravity_home.go` rejeitar symlink. Nao verifiquei se o binario live de 10:37 corrige isso.
Se nao corrigir, H2 nao pode passar para `antigravity` em slots que contenham `cli.log`, e o cutover
travaria por um motivo que o plano nao antecipa. O plano precisa declarar a pre-condicao: rodar H2
em slot cuja preparacao conclua, ou confirmar que o defeito esta corrigido no artefato alvo.

---

## O QUE O PLANO ACERTA (registro explicito, para nao se perder na correcao)

1. **`ff121b28`**: proibido explicitamente na linha 21. **PASS.**
2. **Backups pre-token-only**: `pre-token-only-20260727T102815Z` e `pre-agy-fix` marcados
   `KNOWN-BAD` e `PROIBIDO USAR`, com o motivo correto (incompativel com AGY). **PASS.** Isso e
   coerente com a ressalva do Codex56#B de que reaplicar o estado anterior reintroduziria o AGY
   task-incapaz.
3. **Mecanica de troca atomica**: `cp -a` para `.tmp` seguido de `mv` sobre o destino final e o
   padrao correto — `mv` no mesmo filesystem e atomico, entao nao existe janela com binario parcial.
   **PASS.** (`systemctl --user daemon-reload` e desnecessario para troca de binario, ja que
   nenhuma unit muda; e ruido inofensivo, nao defeito.)
4. **Classificacao STOP-AND-WAIT**: D1, D2 e D3 corretamente marcados como exigindo decisao, com
   citacao de §0.1. **PASS.**
5. **Mapeamento por provider**: `antigravity -> <slot>/home`, `kiro -> <slot>/xdg-data`,
   `codex -> <slot>/codex` bate exatamente com o que medi no TEMA 2. **PASS.**
6. **Ausencia de mutacao executada**: o documento declara zero mutacoes e nao contem evidencia de
   execucao. Confirmo que os binarios em disco tem mtime de 01:18, 10:28, 10:37 e 23:52 — todos
   anteriores ao plano (11:29), portanto o plano nao produziu mudanca. **PASS.**

Pequena imprecisao sem impacto: o plano diz que a descoberta do antigravity e "via CLI nativo
`antigravity`"; o executavel real e `agy`, conforme
`Environment=MULTICA_ANTIGRAVITY_PATH=/home/ec2-user/.local/bin/agy` na unit.

---

## CONDICOES PARA VIRAR PASS

1. Trocar o build para o pacote correto: `go build ... ./cmd/multica` (pacote, nao arquivo), e
   validar o artefato com `<binario> daemon --help` antes de qualquer `mv`.
2. Registrar `sha256` dos 6 binarios e amarrar o alvo de rollback a um commit; se o baseline e
   `88ca4f39`, dizer em qual arvore essa ref resolve.
3. Buildar de arvore limpa (commit ou stash do escritor unico) e adicionar `-trimpath` mais
   `-ldflags` com commit, para o artefato ser reproduzivel.
4. Declarar os prereqs de account, tier e usage como estado conhecido, incluindo que `task_usage`
   tem zero linhas para `antigravity` e `kiro`.
5. Corrigir a allowlist do Kiro para 5 slots: 139, 140, **142**, 143, 149.
6. Nomear `.new` e `.previous` como proibidos.
7. Mover H2 para a matriz de gates como D4, ou substituir por verificacao nao-live.
8. Adicionar gate de entrada em D3: `agent_task_queue` sem `queued` e sem `running`.
9. Declarar a pre-condicao de slot AGY para H2, em razao do defeito `unsupported credential path
   type`.

Itens 1 e 2 sao os unicos estritamente bloqueantes para a mecanica; 3 e 4 sao bloqueantes para o
requisito do dispatch. Os demais sao correcoes de precisao que evitam erro operacional.

## NAO-AFIRMACOES
- READ-ONLY: nao editei codigo, nao compilei, nao executei nenhum comando do plano, nao fiz
  `mv`, `cp`, restart, build, deploy ou rerun. Gravei apenas este arquivo e o meu check-out.
- Nao executei `go build ./cmd/multica` para provar que compila; a conclusao do BLOQUEANTE 1 vem da
  localizacao do subcomando `daemon start` em `cmd/multica` e do `ExecStart` real da unit.
- Nao calculei `sha256` dos binarios: seria leitura de 112 MB e nao muda o veredito. Recomendo que
  quem corrigir o plano o faca.
- Nao verifiquei se o binario live de 10:37 corrige o defeito `unsupported credential path type` do
  AGY; por isso a ressalva 5 e risco declarado, nao defeito confirmado no artefato alvo.
- `88ca4f39` e `ff121b28` nao resolvem nesta arvore; nao afirmo que sejam invalidos, apenas que nao
  sao verificaveis aqui e que o plano nao diz onde resolvem.
- Nao consultei o Postgres neste ciclo: os numeros de `task_usage` e de tasks falhadas vem das minhas
  medicoes anteriores registradas em `gtl-cost-implementation-audit.md` e `T3-runtime-health-audit.md`.
- Nao toquei nas 9 issues preservadas (ORQ-12, 13, 15, 16, 17, 18, 21, 22, 23) e nao disparei rerun.
