# Design — Credential Account Home Restoration

> Regra: nada inventado. Cada afirmacao abaixo foi medida. Onde nao foi, esta marcado
> NAO VERIFICADO.

## 1. Fatos estabelecidos (medidos)

| # | Fato | Evidencia |
|---|---|---|
| F1 | Os isolamentos estao no **ORQ2** | `/home/ec2-user/.agent-cred-homes`, `drwx------`, 22 slots, host `ip-172-31-30-9` |
| F2 | Isolamento **existe e funciona** | `registry.json` 13183 B com `next_slot: 153`; `registry.lock`; backup `pre-orphan-cleanup.20260722T023208Z`; scripts em `~/.local/lib/agent-credential-isolation/scripts` |
| F3 | Slot tem **raizes por provider**, nao e um HOME | `slot-140` contem `cline cline-sandbox codex home xdg-config xdg-data` |
| F4 | 22/22 slots tem agy; **so 5 tem kiro** | slots 139, 140, 142, 143, 149. `slot-145` nao tem kiro |
| F5 | Tabelas de conta do Multica estao **vazias** | Postgres ORQ1: `accounts=0 approved_accounts=0 assignments=0 agent=14` |
| F6 | `daemon.go:3448` tem literal vazio | `credentialAccountHome := ""`, 3 ocorrencias no daemon, nenhuma atribuicao |
| F7 | Regressao datada | `aa62401` criou o resolver; `31d50b9` o condicionou ao L2; `9ab80a6` removeu resolver/rotation store e tornou o gateway incondicional |
| F8 | Preparadores **existem** | `execenv.go:287` codex, `:303` kiro, `:314` antigravity |
| F9 | Modelos exigidos vem do provider | `HOME=<slot-145>/home agy models` retorna `claude-opus-4-6-thinking`, `claude-sonnet-4-6`, `gemini-3.1-pro-high/low`, `gemini-3.5-flash-*`, `gemini-3.6-flash-*` |
| F10 | Gateway **nao** tem gemini 3.6 | `/v1/models` = 327 modelos, zero `gemini-3.6`; unico "3.6" e `oc/qwen3.6-plus-free` |
| F11 | Discovery AGY nao recebia AccountHome | `handleModelList` chamava `agent.ListModels` sem HOME; `agy models` herdava o HOME global do daemon |
| F12 | Backend ja persiste reasoning | `CreateAgentRequest`/`UpdateAgentRequest` e o registro `AgentData` possuem `ThinkingLevel`; PUT seguido de GET preservou `high` |
| F13 | Formulario de criacao omitia reasoning | O inspetor ja usava `ThinkingPicker`, mas `CreateAgentDialog` nao renderizava o campo nem enviava `thinking_level` |
| F14 | Copiar a arvore AGY inteira quebra o task-home | Os slots elegiveis possuem `cli.log` como symlink; `copyCredentialDir` o rejeita. Em mount namespace efemera, HOME contendo somente `antigravity-oauth-token` executou `agy models` com exit 0 e os 10 modelos obrigatorios |

## 2. Mapa de AccountHome por provider

`AccountHome` **nao** e o slot. E uma raiz **dentro** do slot, especifica do vendor.

| Provider | AccountHome | Espera encontrar | Fonte |
|---|---|---|---|
| antigravity (agy) | `<slot>/home` | `.gemini/antigravity-cli/antigravity-oauth-token` | `credential_home.go` e `antigravity_home.go` |
| kiro | `<slot>/xdg-data` | `kiro-cli/data.sqlite3` | `kiro_home.go:11` |
| codex | `<slot>/codex` | `CODEX_HOME` source | `execenv.go:287` |

**Falha silenciosa:** `kiro_home.go:73-75` faz `if srcMissing { return nil }`. Raiz errada
nao gera erro: semeia nada e o kiro roda **sem credencial**. O resolver deve validar a raiz e
o artefato nativo antes de chamar `execenv`; erro de validacao bloqueia a task.

## 3. Lacuna real: modelo de identidade

O registry de isolamento indexa slot por `terminal_id`; a task do Multica traz `AgentID` e
provider. A ponte T2 cruza apenas slots fisicamente existentes e elegiveis, escolhe por
rendezvous-hash de `AgentID + provider` e persiste o slot pseudonimo escolhido. Nenhuma
identidade, email ou valor de credencial entra no arquivo de atribuicao ou nos logs.

## 4. Bloqueio de topologia

Decisao escrita do owner: **T2**. O daemon substituto roda no ORQ2, onde os slots sao locais,
e alcanca o backend loopback do ORQ1 por tunel SSH. O ORQ1 e encerrado somente depois dos tres
runtimes obrigatorios estarem online no T2; nunca existem dois executores elegiveis durante
uma task.

## 5. Segundo bloqueio, especifico do codex

`daemon.go` passa `CredentiallessGateway: true` incondicionalmente e tambem chama
`buildLaunch` mesmo sem plano Agent Brain. O invariante restaurado e:

- plano gateway: `CredentiallessGateway=true`, nenhum `AccountHome`;
- caminho nativo: `CredentiallessGateway=false`, `AccountHome` obrigatorio e ambiente de
  credencial injetado somente a partir do task-home preparado.

## 6. Prepare/Reuse

As condicoes existem nos caminhos Prepare e Reuse. Ambos recebem a mesma resolucao antes de
qualquer preparacao. Reuse nao converte erro em fallback global.

## 7. Estrategia de atribuicao

`rendezvous-hash` de `AgentID + provider`, com persistencia atomica `0600` do slot escolhido.
Round-robin foi rejeitado: quebra afinidade e atribuicao. Se um slot persistido deixa de ser
elegivel, a task falha; remapeamento exige acao operacional explicita.

## 8. Decisoes travadas

1. T2 no ORQ2; nao ha mount nem copia global para o ORQ1.
2. Isolamento nao sera reimplementado; sera integrado.
3. agy/antigravity, codex e kiro obrigatorios; demais providers fora.
4. Atribuicao por rendezvous-hash, nao round-robin.
5. Ausencia, symlink, raiz errada ou artefato nativo ausente bloqueia a task.
6. Discovery AGY usa somente homes validados da allowlist e tenta o proximo elegivel em falha.
7. O task-home AGY recebe somente `antigravity-oauth-token` como arquivo fisico `0600`;
   artefatos irmaos do provider nunca sao copiados.

## 9. Reasoning por formato de catalogo

O wire existente permanece a autoridade. Kiro e Codex anunciam niveis estruturados em
`thinking.supported_levels`; o formulario mostra um seletor separado e envia o token literal
em `thinking_level`. AGY anuncia o tier dentro do proprio ID do modelo, portanto nao recebe
um segundo seletor nem tem o ID normalizado. Trocar runtime ou modelo limpa um override que
nao exista no novo catalogo.

## 10. Snapshot imutavel da conta produtora

O estado corrente de `assignments` nao e uma fonte historica. A conta que produziu uma task e
congelada server-side em `agent_task_queue.credential_account_id` dentro do mesmo UPDATE
atomico do claim. `task_usage.account_id` copia somente esse snapshot; o daemon nao envia e
nao pode escolher o identificador financeiro.

Os predicados de congelamento exigem tenant/workspace, provider canonico, aprovacao ativa,
status `available|leased`, escopo `GENERAL` e assignment exclusiva. Reclaim preserva snapshot
existente. Uma linha legada `dispatched` com snapshot NULL continua selecionavel, mas chega ao
gate ORQ-21 e e cancelada fail-closed; ela nunca consulta a assignment viva nem recebe conta
retroativa.

Migration `128_task_usage_account_id` adiciona as duas colunas UUID nullable, FKs com
`ON DELETE SET NULL`, indices parciais, preflight/backfill canonico e unicidade de
`assignments(account_id)`. NULL permanece um bucket explicito e honesto para uso legado ou
nao atribuivel.

## 11. Riscos

- Falha silenciosa do kiro por raiz errada (mitigado por validacao obrigatoria).
- Symlinks e artefatos volateis no diretorio AGY (mitigado por copia allowlist do token).
- Migration 128 e o stack ORQ-21 precisam entrar juntos; integrar ORQ-12 sozinho deixaria a
  semantica de recusa das linhas legadas incompleta.
- Binario do daemon roda de `/tmp/multica-auth-fixed`, volatil.
