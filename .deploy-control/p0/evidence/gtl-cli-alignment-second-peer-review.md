# GTL-50 - Segunda auditoria independente: alinhamento de versoes de CLI

- auditor: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-27T11:58Z
- auditados: `gtl-cli-version-alignment-plan.md` (GTL-45, Antigravity w8:p2) e
  `gtl-cli-alignment-peer-review.md` (GTL-46, Agy-P0-A8 wB:p2, veredito PASS)
- modo: READ-ONLY. Nada editado, nenhuma instalacao, upgrade, restart ou mudanca de config.

## VEREDITO: **BLOCK** para os dois documentos

O primeiro review esta **topologicamente errado** e aprovou um plano cuja premissa central nao se
sustenta. O achado decisivo, medido: **Claude Code nao e invocado por nenhum executor de task na
frota** - esta explicitamente desabilitado no unico daemon que executa tasks. Toda a Secao A do
plano, e a "trava obrigatoria" que o review mandou expandir para 3 contextos, incide sobre um CLI
fora do caminho de execucao.

## 1. Topologia medida (fonte: comandos read-only, hoje)

| fato | evidencia |
|---|---|
| ORQ1 = `ip-172-31-18-217` / `172.31.18.217` / `100.118.244.61` | `hostname`, `hostname -I` |
| ORQ2 = `ip-172-31-30-9` / `172.31.30.9` / `100.110.178.47` | idem, host local desta auditoria |
| **ORQ1 nao tem daemon nem unit multica** | `systemctl list-units` e `systemctl --user list-units` filtrados por multica = **VAZIO**; `ps -eo pid,args \| grep multica.*daemon` = **VAZIO** |
| ORQ1 roda 4 containers | `docker ps`: `multica-dev-transition-backend-1` (`multica-backend:agy-status-20260727T102815Z`), `multica-dev-transition-frontend-1` (`multica-web:transition-6a2aba3`), `omniroute` (`diegosouzapw/omniroute:latest`), `multica-dev-transition-postgres-1` (`pgvector/pgvector:pg17`) |
| **O daemon executor esta no ORQ2, como user unit** | `systemctl --user`: `multica-daemon-orq2-credential.service` loaded active running - "Multica credential-isolated daemon on ORQ2"; processo `3417665 /home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1 daemon start --foreground --no-auto-update --server-url http://127.0.0.1:18080 --daemon-id orq2-credential-runtime-v1 --device-name ORQ2 Credential Runtime` |
| **O tunel e user unit no ORQ2** | `systemctl --user`: `multica-orq1-backend-tunnel.service` loaded active running - "Multica ORQ2 to ORQ1 backend tunnel". Nao existe no ORQ1. |

## 2. ACHADO DECISIVO - Claude esta DESABILITADO no daemon executor

`systemctl --user cat multica-daemon-orq2-credential.service`, literal:
```
Environment=MULTICA_CODEX_PATH=/home/ec2-user/.nvm/versions/node/v22.23.1/bin/codex
Environment=MULTICA_KIRO_PATH=/home/ec2-user/.local/bin/kiro-cli
Environment=MULTICA_ANTIGRAVITY_PATH=/home/ec2-user/.local/bin/agy
Environment=MULTICA_CLAUDE_PATH=/run/multica-disabled/claude
Environment=MULTICA_OPENCODE_PATH=/run/multica-disabled/opencode
Environment=MULTICA_CLINE_PATH=/run/multica-disabled/cline
```
Habilitados: **codex, kiro, antigravity** - exatamente os tres obrigatorios do owner. Claude aponta
para `/run/multica-disabled/claude`, que e o padrao de desativacao usado para 8 providers.

Confirmacao cruzada: os smokes do Gate 2 sao `ORQ-27 AGY`, `ORQ-28 Kiro`, `ORQ-29 Codex`, todos
`done` no workspace `orq2-dev` (lidos por `GET /api/issues?workspace_slug=orq2-dev`). **Nao existe
smoke de Claude**, porque nao existe runtime Claude.

Segunda confirmacao: **nenhum pane da frota roda Claude.** `herdr pane list` - 10 panes, agentes
`kiro` (w5:p9, w6:p1, w6:p2, w7:p3, w8:p4), `codex` (w5:pC, w7:p4) e `agy` (w8:p2, wB:p1, wB:p2).
Zero `claude`.

**Consequencia**: o risco de custo de `2.1.219` (depth 1->3) **nao pode se materializar** hoje, nem
via daemon nem via pane. A Secao A do plano trata de um CLI que ninguem invoca.

## 3. Que CLI e realmente exercitado, e onde

- **Exercitado**: `codex` no ORQ2, via `MULTICA_CODEX_PATH=~/.nvm/.../bin/codex`, que **ja esta em
  0.145.0** (`codex-cli 0.145.0`, `@openai/codex@0.145.0`). Mais `kiro-cli` e `agy`, fora do escopo
  deste plano.
- **Nao exercitado**: todos os CLIs do ORQ1. Sem daemon, sem unit, e sem sinal de uso interativo:
  no ORQ1 **nao existe `~/.claude`** (so `~/.codex`, mtime `2026-07-20 06:25:12`), e
  `~/.claude/projects` e `~/.codex/sessions` tem **0 entradas**.
- **Nao exercitado**: `claude` no ORQ2 (desabilitado no unit, nenhum pane).

Logo o unico delta com efeito real no caminho de execucao seria subir codex **no ORQ2** - e ele ja
esta no alvo. O plano, como escrito, produz **zero ganho de runtime**: e higiene de paridade no
ORQ1. Isso nao o invalida, mas muda a justificativa, a urgencia e o gate.

## 4. Erros especificos do primeiro review (GTL-46)

| item | o que o review afirmou | medido |
|---|---|---|
| 2.1 | "O host ORQ1 executa a imagem do Backend em Docker **e o tunel SSH** (`multica-orq1-backend-tunnel.service`)" | **ERRADO**: a unit e user unit **no ORQ2**, descrita como "ORQ2 to ORQ1 backend tunnel". ORQ1 nao tem unit multica alguma. |
| 2.1 | correcao proposta: validar canary do ORQ1 via `GET /api/runtimes` no backend do ORQ1 | **TOPOLOGICAMENTE INVALIDO**: `/api/runtimes` reflete registro de runtime vindo do daemon do **ORQ2**. Responderia 200 mesmo que os binarios do ORQ1 estivessem corrompidos. Nao valida upgrade de CLI no ORQ1. |
| 2.2 | expandir a env var para 3 contextos: `/etc/environment` + `/etc/profile.d/`, drop-in do unit e **env do container Docker do backend** | **ERRADO e ineficaz**: o container do backend nunca lanca `claude`; o unit tem Claude desabilitado. E poluicao global sem efeito. |
| 2.5 | rollback "cumpre integralmente" via `npm install -g @openai/codex-cli@0.144.6` | **ERRADO**: o pacote instalado e **`@openai/codex`** (`npm ls -g`: `@openai/codex@0.145.0` no ORQ2, `@openai/codex@0.144.6` no ORQ1). `@openai/codex-cli` nao e o pacote da frota; o comando de rollback como escrito nao restaura nada. |
| 2.6 | "Invariante de fila vazia ... **confirmadas**" | **NAO VERIFICAVEL como confirmacao**: e um SQL proposto, nao uma medicao. Nao ha evidencia anexada de `count=0` em nenhum momento. Alem disso o SQL omite `dispatched` e `waiting_local_directory`, que sao estados ativos conforme `109_agent_task_waiting_local_directory.up.sql:15`. |
| veredito | PASS com 2 correcoes | **BLOCK**: 4 erros factuais, 1 correcao proposta invalida e 1 confirmacao sem evidencia. |

## 5. Erros do plano (GTL-45) que o review nao pegou

1. **Canary invertido em valor de sinal.** Fase 3 do diagrama: "Executar Smoke Tests de Validação no
   ORQ1" com a suite `ORQ-27/28/29`. Essas issues sao smokes do daemon do ORQ2 (secao 2). Rodar no
   ORQ1 e impossivel: nao ha daemon lá. E, mesmo corrigido, um smoke no ORQ1 valida um binario que
   nenhum executor invoca.
2. **`/etc/environment` como local da trava.** A secao 2.1 diz "e OBRIGATORIO definir e exportar a
   variavel no perfil global da frota `/etc/environment` ou systemd env". Hoje a variavel esta
   **ausente** em `/etc/environment` e `/etc/profile.d/` no ORQ1 (medido) - e deve continuar ausente:
   ver secao 6 para o escopo correto.
3. **Tabela de versoes: CORRETA.** Unico bloco do plano que passa integralmente. Medido:
   ORQ2 `claude 2.1.215 (Claude Code)` / `codex-cli 0.145.0`; ORQ1 `claude 2.1.218 (Claude Code)` /
   `codex-cli 0.144.6`. Bate exatamente com o plano e com `gtl-official-release-notes-watch.md:17-18`.
4. **Alvos por fonte oficial: substanciados, com uma ressalva.** O watch GTL-44 registra
   `0.145.0 (2026-07-21)` como ultima do Codex (linha 46: "Nao encontrei ... entrada posterior") e
   `2.1.220` como "Bug fixes and reliability improvements" (linha 53). A frase do plano em 2.1 -
   "Na versao 2.1.219, a Anthropic alterou o limite padrao ... de 1 para 3" - **esta correta e
   citada**: watch linha 54, texto oficial "Subagents can now spawn nested subagents up to depth 3 by
   default (was 1); set `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH=1` to disable nesting". Nao ha invencao
   de versao aqui. Ressalva: o plano **omite** dois itens de 2.1.219 que o watch classifica como
   decisao do owner e que viajam no mesmo salto - `claude-opus-5` como novo default de Opus a
   $10/$50 por Mtok (watch:55) e `workflowSizeGuideline` (watch:60). Subir para 2.1.220 sem decidir
   esses dois e trocar default de modelo e de paralelismo sem gate.
5. **Rollback de env var mal fundamentado.** Secao 5.1: "Manter `...=1` (e seguro e retrocompativel
   com versoes legadas)". Em 2.1.215/2.1.218 a variavel nao existe, logo e inerte - "retrocompativel"
   por inexistencia, nao por suporte. Nao e defeito grave, mas a redacao sugere garantia que o
   fornecedor nao deu.

## 6. Correcoes exatas exigidas

**C1 - Escopo da env var (contra a "expansao para 3 contextos").**
Aplicar `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH=1` **somente** no contexto que efetivamente lanca
`claude`. Hoje esse contexto **nao existe**. Portanto:
- NAO tocar `/etc/environment`, `/etc/profile.d/` nem env do container Docker do backend.
- Se e quando Claude for reabilitado no daemon, o local correto e um drop-in do unit:
  `~/.config/systemd/user/multica-daemon-orq2-credential.service.d/10-claude-depth.conf` com
  `Environment=CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH=1`, na **mesma mudanca** que troca
  `MULTICA_CLAUDE_PATH` de `/run/multica-disabled/claude` para o binario real. Hoje nao existe o
  diretorio `.d/` desse unit (medido: "sem drop-in dir").
- Se algum humano/pane passar a usar `claude` interativamente, a trava vai no perfil daquele
  contexto, nao em escopo global de maquina.

**C2 - Smoke que valida upgrade de CLI, por host.**
Substituir "rodar ORQ-27/28/29 no ORQ1" por:
- ORQ2 (host com executor): apos qualquer upgrade de `codex`, re-executar os smokes de runtime que
  ja existem - `ORQ-29` para Codex, `ORQ-28` Kiro, `ORQ-27` AGY - porque eles atravessam o daemon
  real. Sem smoke de Claude enquanto Claude estiver desabilitado.
- ORQ1 (host sem executor): a unica validacao honesta e local e de binario, nao de plataforma:
  `codex --version` e `claude --version` conferindo o alvo, mais `npm ls -g --depth=0`, mais um
  `codex exec` minimo em diretorio descartavel se o owner autorizar consumo. **Nao** usar
  `/api/runtimes`, que mede o daemon do ORQ2.

**C3 - Corrigir o pacote npm do rollback.** `@openai/codex@0.144.6` (nao `@openai/codex-cli`).
Para Claude, `@anthropic-ai/claude-code@2.1.218` esta correto - e o pacote confirmado por `npm ls -g`.

**C4 - Corrigir a atribuicao do tunel.** `multica-orq1-backend-tunnel.service` e user unit do
**ORQ2**. Qualquer janela de upgrade que reinicie sessao de usuario no ORQ2 derruba o tunel e, com
ele, o acesso do daemon a `127.0.0.1:18080`. Isso precisa entrar no plano como dependencia: o
upgrade no ORQ2 nao e "so trocar binario".

**C5 - Reescrever a justificativa e o alvo do plano.** Declarar que (a) o caminho de execucao usa
codex/kiro/agy no ORQ2, (b) o codex exercitado ja esta em 0.145.0, (c) Claude nao tem invocador, e
portanto (d) este plano e **alinhamento de higiene no ORQ1**, com prioridade baixa, e nao mitigacao
de risco de custo. Se o objetivo real for reabilitar Claude, isso e outra decisao, com gate proprio.

**C6 - Fila vazia: medir, nao propor.** Incluir `dispatched` e `waiting_local_directory` no predicado
e anexar a saida real de `count(*)` como evidencia antes da janela.

**C7 - Decidir os dois itens omitidos de 2.1.219** antes de subir para 2.1.220: default
`claude-opus-5` ($10/$50 por Mtok) e `workflowSizeGuideline`. Decisao do owner, por custo.

## 7. Perguntas em aberto, declaradas sem afirmar

- Por que `claude` foi desabilitado no unit: nao investiguei a decisao, apenas medi o efeito. Se foi
  deliberado pelo mandato AGY/Codex/Kiro, o plano deveria dizer isso.
- Se ha uso de `claude` fora de daemon e fora de pane Herdr (cron, script humano): nao varri o
  sistema por completo.
- Se `codex exec` no ORQ1 consome cota paga: nao verifiquei o modo de auth daquele host.

## 8. Nada mutado

Nenhum plano, config, unit, container, binario ou arquivo de codigo alterado. Sem instalacao,
upgrade, restart ou rerun. Somente leitura: `hostname`, `systemctl list-units/cat`, `ps`,
`docker ps`, `--version`, `npm ls -g`, `stat`, `GET /api/issues` e `herdr pane list`. Nenhum valor de
segredo lido. As 9 issues preservadas seguem intactas.
