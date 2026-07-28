# Plano de Alinhamento de Versoes de CLI - V2 (GTL-71)

- autor: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-27T12:10Z
- substitui: `gtl-cli-version-alignment-plan.md` (GTL-45) e o parecer PASS de `gtl-cli-alignment-peer-review.md` (GTL-46)
- base factual: `gtl-cli-alignment-second-peer-review.md` (GTL-50, medicoes read-only) e
  `gtl-official-release-notes-watch.md` (GTL-44, fontes oficiais)
- modo: READ-ONLY. Nenhuma instalacao, upgrade, restart, mudanca de config, unit, container ou binario.
- escritor/integrador: Codex56-TL (GENERAL-TECH-LEAD) w5:pC. Autoridade final: owner humano.

---

## 1. RECLASSIFICACAO - HIGIENE, PRIORIDADE BAIXA

O plano V1 tratava isto como mitigacao de risco de custo. Nao e. As tres medicoes que reclassificam:

1. **Claude esta DESABILITADO no daemon executor.** Unit `multica-daemon-orq2-credential.service`
   (user unit, ORQ2), literal: `Environment=MULTICA_CLAUDE_PATH=/run/multica-disabled/claude`.
   Habilitados: `MULTICA_CODEX_PATH=~/.nvm/versions/node/v22.23.1/bin/codex`,
   `MULTICA_KIRO_PATH=~/.local/bin/kiro-cli`, `MULTICA_ANTIGRAVITY_PATH=~/.local/bin/agy`.
   Nenhum dos 10 panes Herdr roda `claude` (agentes: kiro, codex, agy). Nao existe smoke de Claude -
   `ORQ-27` e AGY, `ORQ-28` e Kiro, `ORQ-29` e Codex, todos `done` em `orq2-dev`.
2. **O Codex exercitado ja esta no alvo.** ORQ2: `codex-cli 0.145.0` / `@openai/codex@0.145.0`, que e
   exatamente o binario apontado pelo unit.
3. **ORQ1 nao tem executor.** `systemctl list-units` e `systemctl --user list-units` filtrados por
   multica: **vazio**; nenhum processo daemon. ORQ1 roda 4 containers (backend, web, omniroute,
   pgvector). E os CLIs de lá nao tem sinal de uso: **nao existe `~/.claude`**, `~/.codex` com mtime
   `2026-07-20 06:25:12`, `~/.claude/projects` e `~/.codex/sessions` com **0 entradas**.

**Conclusao**: o unico delta que sobra e paridade de binario no ORQ1, host sem executor. Ganho de
runtime esperado: **zero**. Portanto: **HIGIENE, PRIORIDADE BAIXA**, sem urgencia, sem janela
dedicada, e explicitamente **nao** e mitigacao de risco de custo.

Corolario que precisa estar escrito: o risco de blast radius do `2.1.219` (subagentes aninhados de
profundidade 1 para 3) **nao pode se materializar hoje**, porque nao ha invocador de Claude.

---

## 2. Inventario medido (a unica tabela de versoes valida)

| host | endereco | Codex CLI | Claude Code | pacote npm |
|---|---|---|---|---|
| ORQ2 (executor) | `ip-172-31-30-9` / `172.31.30.9` | `codex-cli 0.145.0` | `2.1.215 (Claude Code)` | `@openai/codex@0.145.0`, `@anthropic-ai/claude-code@2.1.215` |
| ORQ1 (containers) | `ip-172-31-18-217` / `172.31.18.217` | `codex-cli 0.144.6` | `2.1.218 (Claude Code)` | `@openai/codex@0.144.6`, `@anthropic-ai/claude-code@2.1.218` |

Bate com `gtl-official-release-notes-watch.md:17-18`. Alvos, **somente** por fonte oficial ja
coletada:
- Codex: `0.145.0` (2026-07-21). Watch linha 46: *"Nao encontrei, na pagina oficial, entrada de
  Codex CLI posterior a 0.145.0"*.
- Claude: `2.1.220`, watch linha 53, texto oficial *"Bug fixes and reliability improvements"*.

Nenhuma versao fora dessas duas fontes entra neste plano.

---

## 3. ESCOPO DE EXECUCAO

### 3.1 Em escopo
- **ORQ1**: alinhar `@openai/codex` de `0.144.6` para `0.145.0`. Motivo: paridade de binario e
  reprodutibilidade de comparacao entre hosts. Prioridade baixa.

### 3.2 Fora de escopo, com motivo
- **Claude em qualquer host**: sem invocador. Subir `2.1.215`/`2.1.218` para `2.1.220` nao muda
  comportamento de execucao e arrasta decisoes de custo (secao 6). **Nao fazer** enquanto Claude
  estiver desabilitado no unit.
- **Codex no ORQ2**: ja em `0.145.0`. Nada a fazer.
- **Qualquer alteracao de env global ou de container**: ver secao 4.

### 3.3 Se o objetivo real for reabilitar Claude
Isso e **outra decisao**, com gate proprio do owner, e nao um alinhamento de versao. Este plano nao a
propoe nem a autoriza; apenas descreve, na secao 4.2, como a trava deve acompanhar essa mudanca
quando ela for decidida.

---

## 4. `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH` - escopo minimo, sem poluicao

### 4.1 PROIBIDO neste plano
- **NAO** tocar `/etc/environment`.
- **NAO** criar `/etc/profile.d/*claude*`.
- **NAO** adicionar a variavel ao env do container Docker do backend nem ao `docker-compose`.

Razao tecnica, nao estilistica: o container do backend nunca lanca `claude`, e o unit do daemon tem
Claude apontado para `/run/multica-disabled/claude`. Nesses dois contextos a variavel e **inerte** -
seria poluicao global de configuracao sem nenhum efeito, e criaria a falsa impressao de que a trava
esta ativa. Estado atual medido no ORQ1: a variavel esta **ausente** de `/etc/environment` e
`/etc/profile.d/`, e deve continuar ausente.

### 4.2 UNICO local correto, e somente no futuro
Se e quando Claude for reabilitado no daemon, a trava entra **junto**, na mesma mudanca, como drop-in
do unit:

```
~/.config/systemd/user/multica-daemon-orq2-credential.service.d/10-claude-depth.conf
[Service]
Environment=CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH=1
```
Condicoes de aceite dessa mudanca futura, todas na mesma janela:
1. `MULTICA_CLAUDE_PATH` deixa de ser `/run/multica-disabled/claude` e passa a apontar o binario real;
2. o drop-in acima existe **antes** do `systemctl --user restart` da unit;
3. a versao instalada e `>= 2.1.219`, senao a variavel e inerte por inexistencia (ver 7.2).

Medido hoje: o diretorio `.d/` dessa unit **nao existe**, logo a criacao do drop-in e um passo
explicito, nao uma edicao.

### 4.3 Contexto interativo
Se algum humano ou pane passar a rodar `claude` a mao, a trava vai no perfil daquele contexto
especifico. Nao em escopo de maquina.

---

## 5. Validacao por host - o que cada smoke pode provar

### 5.1 ORQ2 - somente quando houver upgrade real
Hoje **nao ha upgrade previsto no ORQ2** (Codex ja em `0.145.0`, Claude fora de escopo). Se e quando
houver upgrade real de um CLI exercitado, reexecutar os smokes de runtime que atravessam o daemon:

| smoke | issue | runtime |
|---|---|---|
| AGY | `ORQ-27` | antigravity / `agy` |
| Kiro | `ORQ-28` | `kiro-cli` |
| Codex | `ORQ-29` | `codex` |

Regra: **nenhum smoke de Claude** enquanto Claude estiver desabilitado. Nao inventar smoke para
runtime que nao existe.

### 5.2 ORQ1 - validacao de binario, nao de plataforma
ORQ1 nao tem daemon; qualquer verificacao de plataforma lá mede outra coisa. Validacao honesta:

```bash
codex --version                 # esperado: codex-cli 0.145.0
claude --version                # esperado: inalterado, 2.1.218 (fora de escopo)
npm ls -g --depth=0 | grep -E "codex|claude"
```
Evidencia a anexar: as tres saidas, antes e depois.

**PROIBIDO como validacao do ORQ1**: `GET /api/runtimes`. Esse endpoint reflete o registro de runtime
vindo do daemon do **ORQ2** e responderia `200` mesmo com os binarios do ORQ1 corrompidos ou
ausentes. Foi o erro do parecer GTL-46 e nao deve reaparecer.

---

## 6. Pre-condicoes do gate

1. **Autorizacao escrita do owner.** Instalar pacote e mutacao; nada roda sem isso.
2. **Fila vazia, MEDIDA e anexada** - nao proposta. Predicado com os quatro estados ativos, conforme
   `109_agent_task_waiting_local_directory.up.sql:15`:
   ```sql
   SELECT count(*) FROM agent_task_queue
   WHERE status IN ('queued', 'dispatched', 'running', 'waiting_local_directory');
   ```
   Anexar a saida real. `count = 0` e pre-condicao; `dispatched` e `waiting_local_directory` **nao**
   podem ser omitidos, como estavam no parecer GTL-46.
3. **Dependencia de topologia do tunel.** `multica-orq1-backend-tunnel.service` e **user unit do
   ORQ2** ("Multica ORQ2 to ORQ1 backend tunnel"). Qualquer acao no ORQ2 que encerre a sessao de
   usuario derruba o tunel e, com ele, o acesso do daemon a `http://127.0.0.1:18080`. Portanto:
   mudanca no ORQ2 nao e "so trocar binario"; exige verificar `systemctl --user is-active` das duas
   units antes e depois. ORQ1 nao tem unit multica alguma.
4. **Decisoes do owner ANTES de qualquer Claude 2.1.220** (secao 7.1). Sem elas, Claude fica fora.
5. **Ressalva do Codex56#B mantida**: rollback nunca reaplica branch antiga nem reintroduz copia
   bruta de diretorio de credencial - a arvore AGY antiga com `cli.log` reintroduziria o AGY
   task-incapaz e antecede o hardening `0600`/`O_NOFOLLOW`.

---

## 7. Decisoes que precisam do owner antes de Claude

### 7.1 Dois itens de `2.1.219` que viajam no mesmo salto
O plano V1 omitiu. Fonte: watch GTL-44.

| item | texto oficial (watch) | por que e decisao do owner |
|---|---|---|
| `claude-opus-5` default | linha 55: *"Added Claude Opus 5 (`claude-opus-5`), now the default Opus model - 1M context, fast mode at $10/$50 per Mtok"*; *"Removed Opus 4.7 from fast mode"* | troca de modelo default muda custo por conta sem mudanca de prompt |
| `workflowSizeGuideline` | linha 60: *"Changed dynamic workflows to default to a medium size guideline (aim for fewer than 15 agents)"* | muda paralelismo default da frota |

Subir para `2.1.220` sem decidir os dois e trocar default de modelo e de paralelismo por efeito
colateral de patch de manutencao.

### 7.2 Correcao de fundamentacao do V1
O V1 dizia que manter a trava `=1` e "seguro e retrocompativel com versoes legadas". Preciso: em
`2.1.215` e `2.1.218` a variavel **nao existe**, logo e inerte - retrocompativel por inexistencia,
nao por suporte declarado do fornecedor. Nao mudar a acao; mudar a redacao.

---

## 8. Rollback

| CLI | comando correto | por que |
|---|---|---|
| Codex | `npm install -g @openai/codex@0.144.6` | pacote instalado e **`@openai/codex`**, confirmado por `npm ls -g` nos dois hosts. `@openai/codex-cli`, usado no V1 e aprovado no GTL-46, **nao e o pacote da frota** e nao restaura nada |
| Claude | `npm install -g @anthropic-ai/claude-code@2.1.218` | pacote confirmado por `npm ls -g`; aplicavel so se Claude entrar em escopo |

Rollback de env var: nao ha o que reverter, porque este plano nao cria env var em nenhum escopo
global. Se o drop-in de 4.2 tiver sido criado no futuro, o rollback e remover o arquivo e recarregar
a unit - nunca editar `/etc/environment`.

---

## 9. Sequencia proposta (baixa prioridade, um host, um CLI)

1. Owner autoriza por escrito.
2. Medir e anexar `count(*)` da fila (secao 6.2) e `systemctl --user is-active` das duas units do ORQ2.
3. Capturar evidencia pre: `codex --version`, `claude --version`, `npm ls -g --depth=0` no ORQ1.
4. `npm install -g @openai/codex@0.145.0` **no ORQ1**.
5. Capturar evidencia pos: as mesmas tres saidas.
6. Sem smoke de plataforma no ORQ1 (secao 5.2). Sem tocar ORQ2.
7. Registrar na issue correspondente. Fim.

Nao ha fase canary: com um unico host e um unico CLI fora do caminho de execucao, canary nao agrega
sinal. O V1 propunha canary no ORQ1 com a suite `ORQ-27/28/29`, o que era impossivel (sem daemon lá)
e, mesmo corrigido, validaria binario que nenhum executor invoca.

---

## 10. O que este plano NAO faz

- Nao reabilita Claude, nao propoe reabilitar, e nao autoriza.
- Nao toca `/etc/environment`, `/etc/profile.d/`, `docker-compose` nem env de container.
- Nao reinicia daemon, tunel ou container.
- Nao executa rerun: as 9 issues `ORQ-12, 13, 15, 16, 17, 18, 21, 22, 23` permanecem PRESERVADAS.
- Nao altera credencial, permissao ou slot.

## 11. Perguntas em aberto, declaradas sem afirmar

- Por que Claude foi desabilitado no unit: medi o efeito, nao a decisao. Se foi deliberado pelo
  mandato AGY/Codex/Kiro, convem registrar no proprio unit como comentario.
- Uso de `claude` fora do daemon e fora de pane Herdr (cron, script humano): nao varri o sistema.
- Se `codex exec` no ORQ1 consome cota paga: nao verifiquei o modo de auth daquele host. Por isso a
  secao 5.2 nao inclui execucao, so `--version`.

## 12. Nada mutado

Nenhuma instalacao, upgrade, restart, config, unit, container, binario ou arquivo de codigo alterado.
Somente leitura para produzir este plano. Os unicos arquivos criados sao este e o meu check-out.
