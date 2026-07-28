# ORQ-40 - gate de execucao do alinhamento Codex CLI (ORQ1)

- autor: Opus48#A (agente `2c042fdd-9a76-4da1-9b02-c2332a736a86`), rodando no ORQ2
- UTC: 2026-07-27T13:00Z - 13:05Z
- issue: ORQ-40 `d2001a24-af70-4367-8828-2225ed43ad84`
- plano de referencia: `gtl-cli-version-alignment-v2.md` (GTL-71), parecer PASS `gtl-cli-alignment-v2-peer-review.md` (GTL-74)
- modo: READ-ONLY. Nenhuma instalacao, upgrade, restart, config, unit, container ou env alterado.

---

## 1. PIN EXATO declarado (criterio 1) - OK

- alvo ORQ1: `@openai/codex@0.145.0`
- rollback: `@openai/codex@0.144.6`
- comando de execucao (nao executado): `npm install -g @openai/codex@0.145.0`
- comando de rollback (nao executado): `npm install -g @openai/codex@0.144.6`

Ambas as versoes existem no registry e sao resolviveis (leitura de registry, sem instalar):

```
$ npm view @openai/codex@0.145.0 version dist.tarball
version = '0.145.0'
dist.tarball = 'https://registry.npmjs.org/@openai/codex/-/codex-0.145.0.tgz'
$ npm view @openai/codex@0.144.6 version
0.144.6
$ npm view @openai/codex version        # dist-tag latest
0.145.0
```

Isso satisfaz tambem o criterio 5: o alvo de rollback nao e hipotetico, e a versao viva do ORQ1 hoje
e continua publicada.

## 2. Release note OFICIAL do alvo (criterio 2) - OK, verificado na fonte

Fonte: `https://developers.openai.com/codex/changelog?type=codex-cli` (pagina oficial, filtro Codex CLI),
lida nesta execucao. Nao usei blog, agregador nem memoria.

Entrada literal:

```
2026-07-21
### Codex CLI 0.145.0
`$ npm install -g @openai/codex@0.145.0`
...
Full Changelog: rust-v0.144.0..rust-v0.145.0
```

- data confirmada: **2026-07-21**, exatamente como o card afirma.
- entrada anterior de CLI: `Codex CLI 0.144.6`, **2026-07-18** - a versao instalada hoje no ORQ1.
- **nao existe** entrada de Codex CLI posterior a `0.145.0` na pagina. Corrobora a linha 46 do watch
  GTL-44, que ate aqui era a unica base e nao tinha sido reconferida na fonte (o parecer GTL-74
  declara explicitamente: *"Nao verifiquei as paginas oficiais de release"*). Essa lacuna esta fechada.

Dois itens do changelog de `0.145.0` que confirmam o risco declarado no card:
- `#32093` *"Remove the legacy exec policy engine"* (+ `#34271` *"Migrate legacy exec policy allow rules"*)
- `#33109` *"Reject forks of paginated threads"*

Ambos sao comportamento, nao apenas empacotamento. Mantem-se correta a decisao do card de aceitar
`0.145.0` aqui **somente** com verificacao de binario, e de testar comportamento em card proprio.

## 3. Inventario pre-execucao (evidencia "antes" da secao 5.2 do plano) - OK

ORQ1 (`ip-172-31-18-217`), medido via `ssh orq1` em 2026-07-27T13:02:29Z:

```
$ codex --version
codex-cli 0.144.6
$ claude --version
2.1.218 (Claude Code)        # CONTROLE NEGATIVO - fora de escopo, nao alterar (observacao O2 do GTL-74)
$ npm ls -g --depth=0 | grep -E "codex|claude"
├── @anthropic-ai/claude-code@2.1.218
├── @openai/codex@0.144.6
```

ORQ2 (`ip-172-31-30-9`), host onde este agente roda:

```
$ codex --version
codex-cli 0.145.0
$ npm ls -g --depth=0
├── @anthropic-ai/claude-code@2.1.215
├── @openai/codex@0.145.0
...
```

Confirma o delta unico: Codex do ORQ1. ORQ2 ja esta no alvo, nada a fazer lá.

Topologia (pre-condicao 6.3 do plano), medida agora:

```
ORQ2 $ systemctl --user is-active multica-daemon-orq2-credential.service multica-orq1-backend-tunnel.service
active
active
ORQ1: units multica (system) = 0 ; units multica (user) = 0
```

## 4. FILA ZERO (criterio 3) - **FALHA**, e o predicado como escrito e insatisfazivel

Duas leituras read-only no Postgres do ORQ1 (`docker exec ... psql`, somente `select`):

Leitura 1 - 2026-07-27T13:03:38Z · Leitura 2 - 2026-07-27T13:04:23Z. Resultado identico:

```
active_total
2

id                                   |status |agent_id                             |issue_id                             |created_at
f460ed12-d44d-4634-8032-ad6d6e05264e |running|4069a041-9c68-416a-b0cf-52226c076c6c |c03941bc-3bde-4de1-ab19-1ba93de0ad51 |2026-07-27 12:53:27+00
07cdc53b-e87c-42f1-8b90-593ee545c920 |running|2c042fdd-9a76-4da1-9b02-c2332a736a86 |d2001a24-af70-4367-8828-2225ed43ad84 |2026-07-27 13:00:10+00
```

Predicado usado, literal, com os quatro status ativos exigidos:

```sql
SELECT count(*) FROM agent_task_queue
WHERE status IN ('queued', 'dispatched', 'running', 'waiting_local_directory');
```

`count = 2`, nao `0`. Portanto o criterio 3 **nao esta satisfeito** e nenhuma instalacao pode ocorrer.

### 4.1 Defeito estrutural no criterio 3, que precisa de correcao do card

A segunda linha e **a propria task deste agente** (`agent_id` = Opus48#A, `issue_id` = ORQ-40; o id da
task `07cdc53b` e o mesmo prefixo do workdir desta execucao). Consequencia: **qualquer** agente que
execute este card pela fila vai medir no minimo `count = 1`, porque ele proprio ocupa uma linha
`running` enquanto mede. `count = 0` e insatisfazivel de dentro da fila - so seria observavel por um
operador humano com a frota parada.

Correcao minima proposta (nao aplicada, precisa de decisao do owner/TL), excluindo a task executora:

```sql
SELECT count(*) FROM agent_task_queue
WHERE status IN ('queued', 'dispatched', 'running', 'waiting_local_directory')
  AND id <> '<id da task que executa o card>';
```

Com essa correcao, a medicao de agora daria `count = 1`, **ainda nao zero**: resta a task
`f460ed12` do agente `4069a041` na issue `c03941bc` = **ORQ-39** (Ephemeral Browser QA &
Playwright Supply-Chain Pipeline), `in_progress`. Ou seja: mesmo com o predicado corrigido, a janela
nao esta aberta neste instante.

## 5. Autorizacao do owner (dependencia declarada) - **AUSENTE**

- Descricao do card: *"Janela autorizada pelo owner (instalacao/upgrade e STOP-AND-WAIT)"* e
  *"Estado atual: Nada instalado... Aguardando janela autorizada"*.
- `multica issue comment list d2001a24... --recent 10` retornou `[]` - **nenhum comentario**, logo
  nenhuma autorizacao escrita existe na issue.
- `multica issue metadata list d2001a24...` retornou `{}`.

A atribuicao da issue a um agente nao e a janela autorizada; o proprio card separa as duas coisas.

## 6. Criterio 4 - nao verificavel nesta execucao

`codex --version` = `0.145.0` no ORQ1 depende da instalacao, que esta travada pelos itens 4 e 5.
Nenhuma task foi criada, nenhum smoke pago rodou, nenhum runtime novo foi registrado por causa
deste card.

## 7. Criterio 6 - OK por construcao

Nada tocado em Claude, env var, config global, unit systemd ou container. A unica escrita desta
execucao e este arquivo.

---

## Veredito

**BLOQUEADO**, com dois impedimentos independentes e medidos:

1. fila ativa = 2 (duas leituras), e o predicado do criterio 3 e insatisfazivel de dentro da fila
   sem a correcao proposta em 4.1;
2. nenhuma autorizacao escrita do owner na issue.

Pronto para executar em uma linha assim que os dois caírem:
`npm install -g @openai/codex@0.145.0` no ORQ1, seguido de `codex --version` + `npm ls -g --depth=0`
como unica validacao, sem tocar ORQ2 e sem smoke de plataforma.

## Nada mutado

Nenhuma instalacao, upgrade, restart, config, unit, container, migration, board ou binario alterado.
Consultas SQL exclusivamente `select`.
