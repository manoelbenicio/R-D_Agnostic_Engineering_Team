# GTL-74 - peer review adversarial do alinhamento de CLI V2

Revisor: Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T12:20Z
Documento revisado: `.deploy-control/p0/evidence/gtl-cli-version-alignment-v2.md`
(autor Opus48#A / w6:p1, 11946 bytes, 2026-07-27T12:10Z)
Referencias cruzadas: `gtl-cli-alignment-second-peer-review.md` (GTL-50) e
`gtl-official-release-notes-watch.md` (GTL-44).
Modo: READ-ONLY adversarial. Nao editei nada, nao instalei, nao atualizei, nao reiniciei.

# VEREDITO: **PASS**

Onze itens verificados de forma independente, **onze confirmados**. Nao encontrei nenhum defeito
bloqueante. Registro tres observacoes que **fortalecem** o plano (uma delas o torna mais correto do
que ele proprio afirma) e duas correcoes de precisao que nao alteram nenhuma acao.

| # | item exigido | resultado |
|---|---|---|
| 1 | prioridade baixa / ganho de runtime zero | **PASS** |
| 2 | Claude realmente desabilitado | **PASS, e mais forte que o alegado** |
| 3 | somente Codex ORQ1 em escopo | **PASS** |
| 4 | pacote `@openai/codex` | **PASS** |
| 5 | nenhum env global | **PASS** |
| 6 | drop-in futuro junto do re-enable | **PASS** |
| 7 | smokes honestos por host | **PASS** |
| 8 | tunel e user unit do ORQ2 | **PASS** |
| 9 | 4 status ativos no predicado de fila | **PASS** |
| 10 | decisoes do owner Opus5 / workflowSize | **PASS** |
| 11 | rollback sem known-bad | **PASS** |

---

## 1. Prioridade baixa e ganho de runtime zero - CONFIRMADO

A cadeia de tres medicoes do plano se sustenta e eu a reproduzi de forma independente:
o unico delta remanescente e o binario Codex do ORQ1; o ORQ1 nao tem executor; logo nenhum caminho
de execucao muda. Ganho de runtime **zero** e a classificacao "higiene, prioridade baixa" esta
correta.

Confirmo tambem o corolario da secao 1: o blast radius do `2.1.219` (subagentes de profundidade 1
para 3) **nao pode se materializar hoje**, porque nao existe invocador de Claude em nenhum host
(item 2 e a secao "Inventario" abaixo).

## 2. Claude desabilitado - CONFIRMADO, E O PLANO SUBESTIMA A PROPRIA EVIDENCIA

Unit no ORQ2, literal:
```
Environment=MULTICA_CODEX_PATH=/home/ec2-user/.nvm/versions/node/v22.23.1/bin/codex
Environment=MULTICA_KIRO_PATH=/home/ec2-user/.local/bin/kiro-cli
Environment=MULTICA_ANTIGRAVITY_PATH=/home/ec2-user/.local/bin/agy
Environment=MULTICA_CLAUDE_PATH=/run/multica-disabled/claude
```

**Achado que reforca o plano:** o caminho apontado **nao existe**.
```
$ ls -l /run/multica-disabled/claude
ls: cannot access '/run/multica-disabled/claude': No such file or directory
```
Isso e mais forte do que "desabilitado por configuracao". Pelo codigo de descoberta
(`internal/daemon/config.go`, funcao `probe`), um `MULTICA_*_PATH` que contem `/` e nao existe da
**hard-miss deliberado** — o proprio comentario do codigo diz que um operador que fixou um caminho
absoluto inexistente "should hard-miss, not silently get a different binary". Ou seja o fallback de
login-shell nao resgata, e `claude` nunca entra no mapa de agentes. Confirmado no journal do unit:
`starting daemon ... agents="[codex kiro antigravity]"`. Claude nao esta la.

Consequencia para o parecer: a afirmacao do plano ("Claude esta DESABILITADO") e verdadeira e o
mecanismo e robusto, nao cosmetico. Recomendo que o plano **cite a inexistencia do caminho**, porque
e o que garante que a desabilitacao nao depende de ordem de PATH.

## 3. Somente Codex ORQ1 em escopo - CONFIRMADO

ORQ1 nao tem executor, medido por tres angulos independentes:
```
units multica (system) : 0
units multica (user)   : 0
processo daemon        : nenhum   (pgrep -af "multica.*daemon" vazio)
```
E nao ha sinal de uso dos CLIs no ORQ1:
```
~/.claude          -> No such file or directory
~/.codex           -> existe
~/.codex/sessions  -> 0 entradas
~/.claude/projects -> 0 entradas
```
Versoes que confirmam o delta unico:
```
ORQ1: codex-cli 0.144.6 | 2.1.218 (Claude Code)
ORQ2: codex-cli 0.145.0 | 2.1.215 (Claude Code)
```
Portanto o escopo "somente `@openai/codex` no ORQ1, de 0.144.6 para 0.145.0" esta correto, e excluir
Codex do ORQ2 (ja em 0.145.0) e excluir Claude de ambos esta correto.

## 4. Pacote `@openai/codex` - CONFIRMADO NOS DOIS HOSTS

```
ORQ2: ├── @anthropic-ai/claude-code@2.1.215   ├── @openai/codex@0.145.0
ORQ1: ├── @anthropic-ai/claude-code@2.1.218   ├── @openai/codex@0.144.6
```
A correcao do plano contra o V1/GTL-46 esta certa: o pacote e **`@openai/codex`**, nao
`@openai/codex-cli`. Um `npm install -g @openai/codex-cli@0.144.6` nao restauraria nada — instalaria
outro pacote e deixaria o binario da frota intacto na versao errada. Isso valida tanto a secao 8
quanto o passo 4 da sequencia.

## 5. Nenhum env global - CONFIRMADO EM AMBOS OS HOSTS

`CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH` esta **ausente** de `/etc/environment` e `/etc/profile.d/` nos
dois hosts, e ausente de `~/.bashrc`, `~/.bash_profile` e `~/.profile` nos dois. Nenhum dos 4
containers do ORQ1 tem a variavel no env:
```
multica-dev-transition-backend-1  claude_env=0
multica-dev-transition-frontend-1 claude_env=0
omniroute                         claude_env=0
multica-dev-transition-postgres-1 claude_env=0
```
A proibicao da secao 4.1 esta correta e o raciocinio tecnico e solido: em um contexto que nunca lanca
`claude`, a variavel e inerte e criaria falsa impressao de trava ativa. Concordo com a recusa de
poluicao global.

## 6. Drop-in futuro junto do re-enable - CONFIRMADO

```
$ ls -d /home/ec2-user/.config/systemd/user/multica-daemon-orq2-credential.service.d
No such file or directory
```
O diretorio `.d/` realmente nao existe, logo a criacao do drop-in e passo explicito e nao edicao,
como o plano afirma. As tres condicoes de aceite da secao 4.2 estao corretas e na ordem certa:
o `MULTICA_CLAUDE_PATH` real, o drop-in **antes** do restart, e versao `>= 2.1.219`. Acoplar a trava
a mesma janela do re-enable e a decisao certa — separar as duas criaria uma janela em que Claude roda
sem limite de profundidade.

## 7. Smokes honestos por host - CONFIRMADO

Os tres smokes de runtime existem e estao concluidos:
```
Gate2 runtime smoke AGY   20260727T1043Z | done
Gate2 runtime smoke Codex 20260727T1043Z | done
Gate2 runtime smoke Kiro  20260727T1043Z | done
```
Nao existe smoke de Claude, coerente com a regra "nao inventar smoke para runtime que nao existe".

A proibicao da secao 5.2 e o ponto mais forte do documento e eu a **endosso sem reservas**: usar
`GET /api/runtimes` como validacao do ORQ1 seria falso-positivo garantido. O endpoint reflete o
registro vindo do daemon do **ORQ2** — eu medi isso de forma independente: as linhas `online` em
`agent_runtime` tem `daemon_id = orq2-credential-runtime-v1`, e a rota vive em
`cmd/server/router.go:927`, servida pelo backend, que nao inspeciona binario de host algum. O
endpoint responderia `200` com os binarios do ORQ1 ausentes.

## 8. Tunel e user unit do ORQ2 - CONFIRMADO

```
ORQ2: multica-orq1-backend-tunnel.service      -> active/running, MainPID 3211411
ORQ2: multica-daemon-orq2-credential.service   -> active/running, MainPID 3240496
ORQ1: nenhuma unit multica (system 0, user 0)
```
A dependencia de topologia da secao 6.3 esta correta: o tunel e do ORQ2 e derruba o acesso do daemon
a `127.0.0.1:18080` se a sessao de usuario terminar. Verificar `systemctl --user is-active` das duas
units antes e depois e a precaucao certa. Como o plano nao toca o ORQ2, o risco fica teorico nesta
execucao — mas registrar a dependencia e correto e barato.

## 9. Quatro status ativos - CONFIRMADO CONTRA A MIGRATION

`migrations/109_agent_task_waiting_local_directory.up.sql:13-15` (literal):
```sql
ALTER TABLE agent_task_queue DROP CONSTRAINT IF EXISTS agent_task_queue_status_check;
ALTER TABLE agent_task_queue ADD CONSTRAINT agent_task_queue_status_check
CHECK (status IN ('queued', 'dispatched', 'running', 'waiting_local_directory', 'completed', 'failed', 'cancelled'));
```
Sao 7 status; os terminais sao `completed`, `failed` e `cancelled`. Logo os ativos sao exatamente os
quatro que o predicado da secao 6.2 usa: `queued`, `dispatched`, `running`,
`waiting_local_directory`. A correcao contra o GTL-46 (que omitia `dispatched` e
`waiting_local_directory`) e **material**: um predicado com dois status a menos declararia fila vazia
com trabalho em voo.

Nota de precisao, sem impacto: a citacao do plano diz `:15`; a constraint completa ocupa `:14-15`, e
a lista literal esta na linha 15. A referencia esta boa.

## 10. Decisoes do owner Opus5 / workflowSize - CONFIRMADO PALAVRA POR PALAVRA

`gtl-official-release-notes-watch.md`, linhas citadas, verbatim:
- L55: *"Added Claude Opus 5 (`claude-opus-5`), now the default Opus model — 1M context, fast mode at
  $10/$50 per Mtok"*; *"Removed Opus 4.7 from fast mode"* — classificada modelo/custo, **AGENDAR**,
  "decisao do owner".
- L60: *"Changed dynamic workflows to default to a medium size guideline (aim for fewer than 15
  agents)"* + `workflowSizeGuideline` — classificada paralelismo, **AGENDAR**.
- L53: `2.1.220` — *"Bug fixes and reliability improvements"*, manutencao.
- L46: *"Nao encontrei, na pagina oficial, entrada de Codex CLI posterior a 0.145.0 (2026-07-21)"*.
- L17-18: a tabela de versoes do watch bate exatamente com o inventario do plano e com a minha
  medicao independente.

O argumento da secao 7.1 e correto e importante: subir para `2.1.220` arrasta o `2.1.219` no caminho,
e com ele **troca de modelo default e de paralelismo default**. Tratar um salto de "manutencao" como
inofensivo seria erro. A exigencia de decisao previa do owner esta bem colocada.

Endosso tambem a correcao 7.2: em `2.1.215` e `2.1.218` a variavel de profundidade **nao existe**,
logo e retrocompativel **por inexistencia**, nao por suporte declarado. Ajuste de redacao correto,
sem mudanca de acao.

## 11. Rollback sem known-bad - CONFIRMADO

Os alvos de rollback sao versoes **atualmente instaladas e em uso**, nao artefatos known-bad:
`@openai/codex@0.144.6` e a versao viva do ORQ1 hoje; `@anthropic-ai/claude-code@2.1.218` idem.
Nenhum deles pertence a classe known-bad do inventario de binarios do daemon (que e outra coisa:
`pre-token-only` e `pre-agy-fix`, ambos binarios Go do daemon, sem relacao com pacotes npm).

A ressalva da secao 6.5 esta correta e bem colocada: rollback nunca reaplica branch antiga nem copia
bruta de diretorio de credencial, porque a arvore AGY antiga com `cli.log` reintroduziria o AGY
task-incapaz e antecede o hardening. Confirmo a base tecnica dessa ressalva de forma independente —
eu mesmo medi o defeito no T3: `execenv: prepare antigravity-home: seed per-account antigravity token
dir: unsupported credential path type .../antigravity-cli/cli.log`, causado por `copyCredentialDir`
rejeitar symlink.

E correto o plano dizer que **nao ha env var a reverter**, ja que ele nao cria nenhuma em escopo
global.

---

## INVENTARIO: uso de Claude fora do daemon e fora de panes - E O LIMITE DESSA VARREDURA

A secao 11 do plano declara "nao varri o sistema". Fiz uma varredura read-only limitada, e o
resultado **nao contradiz o plano**:

| superficie verificada | ORQ2 | ORQ1 |
|---|---|---|
| `crontab -l` do usuario | nenhuma referencia | nenhuma referencia |
| `/etc/cron.d`, `cron.daily`, `cron.hourly` | nenhuma | nenhuma |
| timers systemd (user e system) | 0 | 0 |
| `~/.bashrc`, `~/.bash_profile`, `~/.profile` | nenhuma | nenhuma |
| `scripts/` do repositorio (`*.sh`) | nenhuma | n/a |
| env dos 4 containers | n/a | 0 em todos |

**LIMITE EXPLICITO DESTA VARREDURA** — declaro para que ninguem a leia como prova de ausencia total:
1. Nao varri o filesystem inteiro; olhei cron, timers, rc de shell do usuario `ec2-user`, `scripts/`
   do repo e env de container. Um script em outro caminho, ou de outro usuario, nao seria visto.
2. Nao inspecionei processos ao longo do tempo. Um `claude` invocado esporadicamente por um humano
   nao aparece em uma foto instantanea. Provar ausencia de invocacao exigiria auditoria continua
   (auditd/execsnoop), que nao fiz e que nao esta autorizada.
3. Nao verifiquei os panes Herdr um a um neste ciclo; o plano afirma que os 10 panes rodam kiro,
   codex e agy, e eu nao contestei nem reconfirmei essa contagem.
4. Nao verifiquei o ORQ1 quanto a imagens de container que **contenham** o binario `claude` sem
   env var; olhei apenas o env.
5. Nao inspecionei a maquina LOCAL (`wsl-dataops-labs`), que esta fora do escopo do plano mas roda
   panes da frota.

Conclusao honesta: **nenhuma evidencia de uso de Claude fora do daemon foi encontrada**, e isso e
consistente com `~/.claude` inexistente no ORQ1 e com `~/.claude/projects` vazio. Nao afirmo
ausencia absoluta.

---

## OBSERVACOES QUE FORTALECEM O PLANO (nao bloqueiam)

**O1.** Citar a **inexistencia** de `/run/multica-disabled/claude` na secao 1. Hoje o plano diz
"desabilitado"; a evidencia disponivel e mais forte, porque o hard-miss deliberado do `probe` torna a
desabilitacao independente de PATH. Isso reforca o corolario de blast radius.

**O2.** A secao 5.2 pede `claude --version` no ORQ1 "esperado: inalterado". Como Claude esta fora de
escopo e o comando nao muda nada, isso e util como controle negativo — sugiro rotular assim
explicitamente, para ninguem interpretar como passo de upgrade de Claude.

**O3.** O passo 2 da sequencia mede fila e units **do ORQ2** antes de instalar no **ORQ1**. Esta
correto e nao e redundante (o daemon do ORQ2 e quem executa trabalho), mas convem escrever a razao na
propria linha, senao um executor apressado pode achar que e verificacao do host errado e pular.

## CORRECOES DE PRECISAO (sem impacto em acao)

**C1.** A citacao `109_..._up.sql:15` aponta a linha da lista de status; a constraint comeca em `:14`.
Preciso, nao errado.

**C2.** A secao 2 rotula ORQ1 como "containers" e ORQ2 como "executor". Correto, mas vale acrescentar
que o backend que o daemon consome esta **no ORQ1**, alcancado pelo tunel do ORQ2 — o que ja esta na
secao 6.3. Sem isso, a tabela sozinha pode sugerir que ORQ1 e irrelevante, quando ele hospeda o
backend e o Postgres.

---

## NAO-AFIRMACOES DESTE PARECER
- READ-ONLY: nao editei nenhum arquivo alem deste parecer e do meu check-out; nao instalei, nao
  atualizei pacote, nao reiniciei unit ou container, nao apliquei migration, nao toquei board.
- Nao executei nenhum passo da secao 9 do plano; nao rodei `npm install` em host algum.
- Nao provei ausencia total de uso de Claude: ver os 5 limites declarados no inventario.
- Nao verifiquei as paginas oficiais de release: aceitei as citacoes do watch GTL-44 como fonte, e
  conferi apenas que o plano cita o watch fielmente, linha por linha.
- Nao reconfirmei a contagem de 10 panes Herdr nem quais agentes rodam em cada um.
- Nao verifiquei se `codex exec` no ORQ1 consome cota paga — a mesma pergunta aberta que o plano
  declara na secao 11, e concordo em nao inclui-la em escopo de validacao.
- Nao toquei nas 9 issues preservadas (ORQ-12, 13, 15, 16, 17, 18, 21, 22, 23) e nao disparei rerun.
