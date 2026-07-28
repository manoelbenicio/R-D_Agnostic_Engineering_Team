# RCA e Handover — Recuperação Operacional do Kanban Multica

| Campo | Valor |
|---|---|
| **Autor** | KIRO-PRINCIPAL-TL (pane `wB:p1`, host LOCAL `21LAPGLMVPJ4`) |
| **Co-autoridade técnica** | Codex56-TL (pane `w5:pC`, ORQ2) — escritor único |
| **Frota** | 8 agentes auditores (Opus48#A-D, Codex56#A-B, Agy-P0-A7/A8) |
| **Período** | 2026-07-26T13:00Z → 2026-07-27T11:05Z |
| **Escopo** | Multica L4 (Go) + daemon de runtime + OmniRoute gateway |
| **Autoridade** | `AUTHORITY_AMENDMENT_001` §0.1 STOP-AND-WAIT; owner é decisor único |

---

## 1. Sumário executivo

O kanban estava inoperante: dos 6 runtimes de CLI, **apenas 1 estava online** e o seletor de
modelos vinha vazio, impedindo a criação de agentes e o início de qualquer projeto.

Ao final da janela: **os 3 runtimes obrigatórios (AGY, Codex, Kiro) estão online, executam task
de ponta a ponta com resposta correta, e o catálogo de modelos popula com 11/19/11 modelos**,
incluindo os tiers de reasoning. Três smokes independentes (ORQ-27, ORQ-28, ORQ-29) completaram.

Foram identificadas **quatro causas raiz distintas**, todas confirmadas por evidência em código
ou medição, nenhuma por inferência.

---

## 2. Causas raiz

### CR-1 — Colapso do mapa de agentes (runtimes offline)

**Sintoma:** `multica daemon status` reportava `Agents: claude`, com 5 CLIs instalados e ignorados.

**Causa:** `internal/daemon/config.go:425-431`

```go
if agentBrainCfg.DevelopmentEnabled && Gateway.Required {
    agents = map[string]AgentEntry{provider: entry}   // descarta os 15 detectados
}
```

O daemon rodava com `--agent-brain-development` e `AGENT_BRAIN_GATEWAY_REQUIRED=true`, o que
substituía o mapa completo de CLIs por **um único**, derivado de `AGENT_BRAIN_CLI_KIND=claude-code`.
Não era falha de detecção: era substituição deliberada do mapa.

**Correção:** relançamento do daemon em modo normal, sem a fatia de development. Os 6 runtimes
registraram sozinhos.

**Evidência de refutação de hipótese:** `--agent-brain-cli-kind` é `string` escalar, não
repetível, e aceita apenas `claude-code` ou `codex` (`cmd_daemon.go:136`, `config.go:235`,
`config.go:791`). Passar múltiplos valores nunca resolveria — a flag era a **causa**, não a cura.

### CR-2 — Ausência do resolver de credencial por conta (regressão datada)

**Sintoma:** credenciais de CLI ausentes no host do daemon; discovery falhando por falta de login.

**Causa:** `internal/daemon/daemon.go:3448`

```go
credentialAccountHome := ""   // literal, nunca reatribuído
```

Três ocorrências no daemon inteiro, nenhuma atribuição. O próprio código documenta o resolver
que nunca foi escrito, em `execenv.go:68-72`: *"The daemon resolves this from the
agent-account assignment... Empty is invalid for credential-bearing providers that require
isolation."*

**Natureza: regressão, não feature ausente.**

| Commit | Data | Efeito |
|---|---|---|
| `aa62401` | 2026-07-02 | criou `credentialAccountHomeForTask` → `CurrentAssignment` → `GetAccount` → `account.HomeDir` |
| `31d50b9` | 2026-07-05 | **removeu** e substituiu por literal vazio |

Os adaptadores por vendor **já existiam** e continuam intactos: `execenv.go:287` codex,
`:303` kiro, `:314` antigravity, `:325` cline.

**Agravante de mapeamento:** `AccountHome` não é o slot, é uma raiz **dentro** do slot:

| Provider | AccountHome correto | Artefato esperado |
|---|---|---|
| antigravity | `<slot>/home` | `.gemini/antigravity-cli/` |
| kiro | `<slot>/xdg-data` | `kiro-cli/data.sqlite3` |
| codex | `<slot>/codex` | `CODEX_HOME` source |

**Falha silenciosa associada:** `kiro_home.go:73-75` faz `if srcMissing { return nil }`. Raiz
errada não gera erro — semeia nada e o kiro roda **sem credencial**, sem sinal.

### CR-3 — Symlink `cli.log` impedindo execução de task no AGY

**Sintoma:** AGY online e com catálogo, mas **task-incapaz**.

**Causa:** a preparação de HOME copiava um symlink `cli.log` e falhava. Uma HOME efêmera
contendo **apenas** `antigravity-oauth-token` executou `agy models` com exit 0 e retornou os 10
modelos obrigatórios.

**Correção:** patch token-only + hardening do helper, promovido no Gate 2.

### CR-4 — Contrato de schema falhando fechado no upload de arquivo (chat)

**Sintoma:** "Failed to send message" e nenhum botão funcional no painel de chat.

**Causa:** campo divergente **`download_url`**.

Cliente — `packages/core/api/schemas.ts:100-108`: `AttachmentResponseSchema` declara
`download_url: z.string()` **obrigatório**, com `.loose()`. Logo o defeito nunca foi campo
extra: é **campo obrigatório a menos**.

Servidor — `server/internal/handler/file.go` tem **três saídas** em `UploadFile`:

| Rota | Linha | Comportamento |
|---|---|---|
| correta | `file.go:453` | `attachmentToResponse` preenche `DownloadURL` via `attachmentDownloadPath(id)` |
| **quebrada A** | `file.go:456-460` | `CreateAttachment` falha → `download_url` **ausente** e `id` string vazia |
| **quebrada B** | `file.go:472-476` | sem contexto de workspace → `download_url` **ausente** |

Erro do Zod nas duas: `invalid_type` em `download_url`, expected string, received undefined.

**Por que só agora:** o commit `d10d09e0` (autosave pre-shutdown, 2026-07-21T21:33:16Z, 40
linhas em `schema.ts`) trocou **degradar por falhar fechado** — removeu `return fallback` e
adicionou `throw new ApiContractError(...)`, exatamente `schema.ts:56`. O terceiro parâmetro
passou a `_legacyFallback` deliberadamente ignorado (`schema.ts:31-37`).

**O contrato não mudou. O tratamento mudou.** A divergência já existia e degradava em silêncio.

**Candidato de correção (auditado):** o commit `d10d09e0` toca **21 arquivos** — reverter o
commit inteiro é proibido. Candidato limpo: base `6a2aba3` mais **apenas os 5 arquivos atuais de
reasoning**. Diferença literal: `schema.ts:41-56` (sujo) ignora o fallback e lança;
`schema.ts:38-54` (limpo) retorna o fallback. `client.test.ts:729-751` mais `schemas.ts:100-108`
reproduzem o drift.

**Gatilho da rota B:** `workspaceID` vazio via `h.resolveWorkspaceID` (`handler.go:445-447`);
o header sai de `client.ts:300-301`, e se `getCurrentSlug()` estiver vazio no upload do chat,
cai em B. Isso explica a falha **isolada** — 86 de 88 testes: o caminho feliz cumpre o contrato.

### CR-5 — umask permissivo: exposição sistêmica de artefatos

**Sintoma:** artefatos com segredo nascendo legíveis por qualquer processo local.

**Causa medida:**

| Host | umask | `/tmp` | Arquivos com perm `/077` |
|---|---|---|---|
| ORQ2 | **0002** | `drwxrwxrwt` 1777 | **74.537** |
| ORQ1 | **0002** | 1777 | 127 |
| LOCAL | 0022 | 1777 | — |

**Todo artefato nasce mundo-legível.** Corrigir arquivo por arquivo sem corrigir o `umask`
reintroduz o defeito no próximo artefato.

**Prova da causa raiz — assimetria:** `av.sh`, `raw.sh`, `imp.sh` e `sync.sh` são **0600 no
ORQ1** e **0644 no LOCAL**. Mesmo script, dois modos, porque os `umask` diferem.

**Ordem de correção obrigatória** (para token cru, `chmod 600` não basta se o valor estiver
vivo): **rotacionar → `chmod` → `umask` → remover**.

**Triagem dos itens de severidade alta**, após eu esclarecer conteúdo que só o autor conhecia:

| Arquivo | Classificação inicial | Veredito final |
|---|---|---|
| `handshake-token.txt` 644, 24 B | token cru exposto | **falso positivo** — string `HANDSHAKE-KIROTL-131232` de teste de pane |
| `rev-token.txt` 644, 24 B | token cru exposto | **falso positivo** — `MSG-FROM-CODEX56-135612` |
| `arch.txt` com `OPENAI_API_KEY=` | atribuição | **falso positivo** — prosa de mensagem, sem valor |
| **`backend-recover.sh`** 664, `JWT_SECRET=` + `POSTGRES_PASSWORD` | atribuição | **legítimo, não é do autor — pendente** |

Os itens de severidade média (`sec.txt`, `o30.txt`, `dec.txt`, `f.txt` e similares) são mensagens
de coordenação contendo **nomes de chave em prosa**, ambiguidade que o auditor declarou não poder
resolver sem ler valor — corretamente.

**Correção de premissa:** `/tmp/daemon.env.bak` e `.cmd.bak` **não existem** em nenhum dos três
hosts. O que existe no ORQ1 são `daemon.environ.*.bak` em **0600, corretos** — renomeados pela
quarentena aplicada na ORQ-22.

---

## 3. Ações executadas e verificadas

| # | Ação | Verificação independente |
|---|---|---|
| 1 | Relançamento do daemon em modo normal | 6 runtimes registrados, `Agents:` com os 6 |
| 2 | Cutover T2 — daemon substituto no ORQ2 com systemd `--user` | unit `active`, `NRestarts=0`, symlink em `default.target.wants` |
| 3 | Binário movido para fora do `/tmp` | `~/.local/lib/multica/bin/multica-auth-credential-home-v1` |
| 4 | Gate 1 — build de binário e imagem | SHA-256 `88ca4f39…`, imagem `sha256:60133934…`, 5 hashes coincidem com o freeze |
| 5 | Gate 2 — cutover backend-first | backend 200, `/api/me` 200, frontend 200, túnel 200, zero restarts |
| 6 | Patch AGY token-only promovido | AGY deixou de ser task-incapaz |
| 7 | Smoke dos 3 obrigatórios | ORQ-27 AGY, ORQ-28 Kiro, ORQ-29 Codex — todos `completed` |
| 8 | ORQ-30 — `.env` durável | modo 0600, dir 0700, JWT preservado, **dois recreates** sem perda |
| 9 | Reversão da credencial global do ORQ1 | AGY e Codex ausentes, Kiro byte-idêntico ao backup |
| 10 | Limpeza de cache no ORQ1 | disco de **98% para 75%** |
| 11 | Limpeza de runtimes obsoletos | de 12 para 5; dois recusaram com 409 por vínculo referencial |

---

## 4. Erros do autor deste documento

Registrados porque a credibilidade do RCA depende de não omiti-los.

| # | Erro | Consequência | Correção |
|---|---|---|---|
| 1 | Edição do `HERDR_COMMS_GUIDE.md` sem consentimento do Codex | política MANDATORY alterada sem autoridade | vetado, congelado como proposta não ratificada |
| 2 | Cópia de credencial para o HOME global do ORQ1 | quebrou isolamento por conta e atribuição de custo | revertida e verificada no item 9 |
| 3 | `/tmp/daemon.env.bak` criado em modo **0664** com `DATABASE_URL`, `JWT_SECRET`, `MULTICA_TOKEN`, `POSTGRES_PASSWORD` | exposição a qualquer usuário do host | **aberto** — aguarda `chmod 0600` e quarentena |
| 4 | `DATABASE_URL` impresso no transcript | senha do Postgres no histórico | recomendada rotação |
| 5 | Quatro erros de parser reportando ausência falsa | conclusões erradas sobre providers, conexões e slots | corrigidos por medição |
| 6 | Interpretação de `360 tool_use / 14 tool_result` como perda de resultado | quase autorizou rerun inseguro | refutado por comparação com caso de controle |
| 7 | Oito agentes ociosos enquanto eu aguardava resposta serial | desperdício de capacidade paga | frota redespachada em frentes disjuntas |

**Padrão comum aos erros 5 e 6:** afirmar ausência ou causa a partir de medição malfeita, sem
caso de controle. **Mitigação adotada:** toda afirmação de ausência exige comando reproduzível.

---

## 5. Disciplina de replay — invariante crítico

Estabelecido após três leituras sucessivamente refutadas:

`tool_use` **não** prova conclusão. Revisão direta: `hermes.go:906-924` emite `tool_use` no
**START** quando há `rawInput`; `hermes.go:992-998` **sempre** emite `tool_result`, mesmo vazio.

| Caso | Números | Leitura correta |
|---|---|---|
| 5 tasks Kiro `completed` | 72 / 0 | característica de formato |
| 6 tasks `runtime_offline` | 360 / 14 | atividade **iniciada**, desfecho **ambíguo** |

**Não se pode afirmar nem efeito aplicado nem perda.** Por isso o replay permanece
**fail-closed**, e as 9 issues seguem preservadas sem rerun.

**Invariante:** *telemetria nunca autoriza replay.*

---

## 6. Classificação de idempotência das 6 issues ambíguas

Resultado: **0 idempotentes, 6 mutantes.** Nenhuma pode ir a rerun de risco zero na forma atual.

| Issue | Veredito | Razão |
|---|---|---|
| ORQ-12 | MUTANTE | "persistir identidade por execução e demonstrar com dados reais". `migrations/032_task_usage.up.sql:2-10` cria a tabela **sem** `account_id`; `runtime_usage.sql:40` declara que `task_usage only carries task_id`; `UpsertTaskUsage` tem `ON CONFLICT` em `task_id, provider, model` — exige migration e troca da chave |
| ORQ-13 | MUTANTE | "execuções comparáveis" implica rodar task paga em dois tiers, consumindo cota |
| ORQ-17 | MUTANTE, **maior blast radius** | remover o bind de `127.0.0.1:13100` muda superfície de ataque; o próprio critério exige validar autenticação junto |
| ORQ-18 | MUTANTE | escreve componente e teste no frontend, e **colide com a ORQ-26 aberta** — misturaria duas causas de falha |
| ORQ-21 | MUTANTE | `INSERT` em `accounts`, `approved_accounts` e `assignments`, hoje com zero linhas |
| ORQ-23 | MUTANTE, **pior risco** | o gate 4.5 exige **executar** o rollback do T2 e revalidar os 3 obrigatórios que acabaram de passar |

**Correção de âncora:** a ORQ-13 cita `daemon.go:2101`, mas ali hoje está apenas o fecha-chaves
de `thinkingLevelWire`. A âncora real de preço é `internal/metrics/pricing.go`, mais
`internal/handler/cloud_billing.go` e `internal/metrics/business.go`. E `thinking_level` é
persistido em `agent` (`pkg/db/queries/agent.sql:23,43,50-54`), **não** em `task_usage` — por
isso o dado por tier não existe na tabela de custo.

**Achado de forma:** o campo `acceptance_criteria` da API vem `NULL` nas seis; o critério está
embutido no corpo de `description`.

**Caminho seguro proposto:** cada uma das seis tem um **prefixo investigativo** liberável como
escopo reduzido, entregando evidência sem escrever nada.

---

## 7. Desenho do ledger durável (proposto, não implementado)

**Problema:** `ReplayGateHook` e `LedgerRegistry` mantêm histórico **em RAM**
(`daemon.go:301`), e `cmd/server/main.go` **não wira** o hook — processos separados. Após
restart o gate falha fechado (`replay_gate.go:201`), bloqueando retry. Houve dois restarts.

**Desenho:**

1. tabela `task_ledger_summary` com quatro booleanos `ever_*` e TTL `expires_at = recorded_at + 24h`
2. três pontos de gravação: pré-`tool_use` (`daemon.go:4357`), pós-`tool_result` (`:4399`), defer-close (`:4199`)
3. `DatabaseReplayGateHook` no backend lendo Postgres — elimina dependência de processo cruzado
4. `commitledger/replay_gate.go:152-205` **inalterado** — reutiliza `BlocksReplay()`
5. invariante preservado: `ever_*` e `TotalToolUseCount` em colunas **separadas**; o gate lê apenas `ever_*`

`MaybeRetryFailedTask:1666` passa a bloquear por **dado**, não por `hook == nil`.

---

## 8. Estado atual do ambiente

| Componente | Estado |
|---|---|
| Backend ORQ1 | imagem `60133934`, HTTP 200, restarts 0 |
| Daemon ORQ2 | PID 3417665, unit `multica-daemon-orq2-credential.service` active, hash `88ca4f39…` |
| Runtimes | AGY, Codex, Kiro **online**; discovery 11/19/11 |
| Frontend | 200, imagem `cf8017e3` — **bundle sujo**, contém `ApiContractError` |
| OmniRoute | 327 modelos, 4 contas AGY, 4 ClinePass, 2 Codex, 3 NVIDIA, todas `active` |
| Isolamento | 22 slots no ORQ2, `registry.json` `next_slot: 153`, 4 contas AGY vivas |
| Kanban | projeto ORQ2, 20 issues, 9 preservadas sem rerun |
| Acesso do owner | túnel SSH supervisionado em `/home/dataops-lab/tunnel-multica.sh` |
| Autenticação | **`/api/me` responde 200 anônimo** — impede publicação na LAN sem ACL |
| Browser QA | **nenhum Chrome ou Chromium** no ORQ1 nem no ORQ2 |

---

## 9. Itens abertos

### Decisão do owner (risco, não técnica)

| # | Item | Risco de postergar |
|---|---|---|
| 1 | ~~`chmod 0600` + quarentena dos backups~~ | **RESOLVIDO** — ORQ-22 done, quarentena em 0600/0700, legacy desabilitado |
| 1b | **`umask` 077 nos três hosts** + `chmod` em massa + tratar `backend-recover.sh` | **74.537 arquivos expostos no ORQ2**; sem o `umask` o defeito volta |
| 2 | Browser para QA da ORQ-26: **instalar Chromium** ou **owner valida clicando** | nenhum browser existe nos hosts; sem validação o chat não fecha |
| 3 | Rerun das 6 ambíguas — **nenhuma é idempotente** | 6 de 20 issues paradas |
| 4 | ORQ-15 e ORQ-16 — `agent_error.unknown`, zero eventos | causa desconhecida |
| 5 | Logar 6 slots AGY mortos | capacidade limitada a 1 conta |
| 6 | Publicar 13100 na LAN | dependência de túnel frágil |
| 7 | Rotação do `DATABASE_URL` exposto por mim | credencial comprometida no histórico |

### Execução do escritor único (Codex56-TL)

| # | Item | Base pronta |
|---|---|---|
| 8 | Corrigir ORQ-26 — rotas A e B devolverem `download_url` via `attachmentDownloadPath(id)` | RCA completo, seção 2 CR-4 |
| 9 | Ledger durável | desenho na seção 7 |
| 10 | Reconciliar ORQ-22 e estados falsos do board | backend novo no ar |
| 11 | Custo: `account_id`, preço por tier, token AGY/Kiro | âncoras corrigidas na seção 6 |
| 12 | Squad no seletor e botão de deletar runtime | `assignee-picker.tsx`, `runtime-list.tsx`, `runtimes-page.tsx:794` |
| 13 | Skills curadas — 98 super-skills, limite de 128 arquivos | `skills-rollout-plan.md` |

### Bloqueio estrutural

`project.lead_type` tem `CHECK (lead_type IN ('member','agent'))` — **não aceita `squad`**.
`squad.leader_id` é escalar único. Squad liderando projeto exige migration.

---

## 10. Lições operacionais

1. **Ausência exige comando reproduzível.** Quatro conclusões erradas vieram de parser mal escrito ou caminho errado, não de dado ausente.
2. **Anomalia numérica não é causa até haver caso de controle.** A proporção `tool_use`/`tool_result` foi lida errado três vezes.
3. **Escritor único elimina colisão.** O preflight abortou por overlap detectado antes de escrever — o mecanismo funciona, mas não se deve depender dele.
4. **Auditoria cruzada pega o que o autor não vê.** Gate 1 e Gate 2 foram verificados por agente distinto do executor.
5. **Recusar patch é resultado válido.** Dois agentes se recusaram a escrever código que passaria fail-open disfarçado de fail-closed.
6. **Coordenação serial desperdiça frota.** Aguardar resposta única deixou 8 agentes ociosos.
7. **Fail-closed sob ambiguidade é a postura correta**, mesmo custando 9 issues paradas.

---

## 11. Achados posteriores à primeira versão deste documento

### 11.1 AGY zero tokens — causa identificada, Kiro é limitação de vendor

`hermes.go:1190-1214`, `handleUsageUpdate` tenta **somente camelCase** (`inputTokens`).
Correção proposta: acessor com fallback para snake_case.

**Kiro:** o CLI 2.x **não emite `usage_update`**. É lacuna do fornecedor — sem correção possível
sem mudança do CLI. Deve ser declarado como limitação, não como bug a corrigir.

### 11.2 Âncoras de custo corrigidas

`daemon.go:2101` citado na ORQ-13 contém hoje apenas o fecha-chaves de `thinkingLevelWire`. As
âncoras reais são `internal/metrics/pricing.go`, `internal/handler/cloud_billing.go` e
`internal/metrics/business.go`. E `thinking_level` é persistido em `agent`
(`pkg/db/queries/agent.sql:23,43,50-54`), **não** em `task_usage` — por isso o dado por tier não
existe na tabela de custo.

Desenho proposto: migration 127 com `ADD COLUMN account_id UUID NULLABLE` mais índice, e
`ADD COLUMN thinking_level TEXT`, com wiring via query `GetTaskAccountID` (`assignments` JOIN
`agent_task_queue`).

### 11.3 ORQ-21 — bloqueador de arquitetura, patch invalidado antes de ser escrito

| Severidade | Achado |
|---|---|
| **BLOQUEADOR** | o daemon do ORQ2 **não tem `DATABASE_URL` nem `PG*`**; o patch de `credentialregistry` tem **zero callers**. Resolver deve ficar no **caminho de claim do backend**, enviando ao daemon apenas conta pseudônima e slot |
| ALTA | seed `L128-140` sobrescreve tenant, vendor, home e config; força `available`; zera usage, cooldown e error |
| ALTA | `L153-164` reabilita revogação e reatribui **sem guarda de task ativa, sem lock e sem `rotation_event`** |
| ALTA | resolver `L52-95` confia no path do banco e **perde** a validação física de root, symlink e artefato de `credential_home.go:88-138` |
| MÉDIA | comparação crua de provider quebra `agy`/`antigravity`; política de compartilhamento de conta ausente |
| TESTE | `L14-17` **pula sem banco**, e fixture falsa em `/tmp` não reproduz o bug de path |

O `L153-164` é o mais grave: reatribuir conta sem guarda pode trocar credencial **sob task em
execução**. E o resolver confiando em path do banco reintroduziria a falha silenciosa do kiro.

### 11.4 ORQ-16 pode fechar sem rerun

A allowlist viva do AGY é `141, 145, 146, 150`, que **exclui todos os seis slots mortos**. O
único comentário na issue é a falha antiga do `cli.log` no slot-145, já resolvida. O escritor
registra a decisão de exclusão mais `discovery=11` e move para `in_review`, **sem rerun**.

### 11.5 Board não deve ser mutado por status `completed`

Medição do escritor único: ORQ-14, 19 e 20 estão `blocked` com `latest completed`; ORQ-25 está
`in_progress` com `completed`; ORQ-26 está `in_progress` **sem task**.

Veredito: **`completed` não prova aceite.** ORQ-25 é candidato falso imediato. O board não foi
mutado — decisão correta, porque reconciliar por status produziria aceite fictício.

### 11.6 Trabalho parcial preservado em três issues

ORQ-13, ORQ-18 e ORQ-21 têm trabalho parcial: **13 caminhos, 2 arquivos e 3 arquivos**
respectivamente. Tratamento correto: **revisar e rebasear, não rerodar.**

### 11.7 Bloqueio de autenticação para exposição na LAN

`/api/me` responde **200 anônimo**. Publicar a porta 13100 na LAN hoje exporia a API **sem
autenticação**. Isso valida a condição imposta pelo escritor único de que a ORQ-17 só avança
após endurecimento de auth e ACL.

### 11.8 ORQ-13 — patch de custo REJEITADO

| Defeito | Local |
|---|---|
| precifica **sem tier** | `utils.ts:451-465` |
| agrupa só por model e provider, **colapsando tier** | `utils.ts:901-911` |
| também colapsa tier | `business.go:253-285` |
| **`MODEL_PRICING` não contém `gemini-*`** → AGY reportaria **custo zero** | tabela de preço |
| aceita tier arbitrário sem validação | `daemon.go:2074-2110` |
| zero testes novos; fixtures TS tipadas omitem campo obrigatório | — |
| **down migration colapsa e regrava dados** | migration |

Recomendação: fonte única versionada **ou** recibo do gateway, e migration **conjunta** de
`account_id` e tier. O item do `MODEL_PRICING` é o mais grave para o negócio: custo invisível em
um dos três runtimes obrigatórios é pior que custo errado.

### 11.9 ORQ-18 — aprovado com condição

Primeiro veredito positivo da rodada.

| Aspecto | Resultado |
|---|---|
| Direção | correta e recuperável — botão `Trash2` só para autorizado, reaproveita confirmação existente |
| Conflito com ORQ-26 | **nenhum** — blobs-base idênticos em `6a2aba355` e HEAD `0cb8aeb` |
| **Bloqueio** | teste novo prova apenas botão e diálogo; **não cobre sucesso, erro, close nem refetch** |
| **Bloqueio** | permissão injetada pronta **duplica regra central** |

Integrar **depois** da ORQ-26, com testes de comportamento e browser autenticado.

### 11.10 ORQ-26 — correção da correção: `attachmentDownloadPath` produziria 404

**A solução registrada na seção 2 CR-4 como "preferida e aditiva" está ERRADA.**

`file.go:136-138` gera `/api/attachments/{id}/download`, mas `file.go:567-579` **exige a row no
banco** para servir. Os fallbacks `:447-461` e `:465-476` ocorrem justamente **quando não há row**.

Usar `attachmentDownloadPath(id)` nos fallbacks produziria **schema verde com 404** — pior que o
bug atual, porque falharia em silêncio no clique do usuário em vez de falhar visivelmente.

**Correção correta, por rota:**

| Rota | Ação |
|---|---|
| sem contexto de workspace | `id` vazio, e `url` e `download_url` apontando para **`link`** — o objeto direto, que existe |
| referência de entidade | resolver o workspace com os gates, **ou rejeitar antes do upload** |
| insert falho | `Delete` best-effort mais **HTTP 500** — parar de responder 200 quando o insert falhou |

### 11.11 Placar da auditoria cruzada

| Issue | Veredito |
|---|---|
| ORQ-21 | **rejeitado** — bloqueador de arquitetura, patch com zero callers |
| ORQ-13 | **rejeitado** — `MODEL_PRICING` sem `gemini-*` |
| ORQ-26 | candidato limpo definido; **primeira correção proposta refutada** |
| ORQ-18 | **aprovado com condição** — após ORQ-26 |
| ORQ-16 | liberável sem rerun |

Cinco patches auditados: **dois barrados, um condicionado, um liberável, uma correção de
correção**. Nenhum entrou em código sem revisão independente. Em dois casos a primeira solução
proposta por um agente competente estava errada e foi pega por outro — o que valida o desenho
adversarial em vez de o desmentir.

### 11.12 ORQ-23 — auditoria profunda, e CORREÇÃO de afirmação deste documento

**Branch antiga NÃO deve ser reaplicada:** ainda copia a árvore AGY **com o `cli.log`** e
**precede** o hardening de `0600` e `O_NOFOLLOW`. Reaplicá-la reintroduziria a CR-3.

| Achado | Detalhe |
|---|---|
| Live `88ca4f` | **green**, com **8 afinidades persistidas** |
| F2.3 | **sem** task-slot-custo |
| Wire de usage | **sem account nem tier**; a aba Usage do AGY vem **vazia** |
| Discovery | **para no primeiro HOME** — não prova paridade entre os 4 slots |

**CORREÇÃO CRÍTICA a este documento:** a seção 3 e o handoff afirmam que o rollback do T2 está
preservado e disponível. **Está incompleto.** Não existe rollback de **um comando** para o T2, e
os backups anteriores ao patch token-only **reintroduziriam o AGY task-incapaz**.

Ou seja: hoje o caminho de volta custa o bug que acabamos de corrigir. Rollback do T2 precisa ser
**construído**, não apenas invocado. Enquanto isso, a postura correta é tratar o estado atual como
**sem rollback seguro** para o T2 — o que eleva o cuidado em qualquer mudança futura no AGY.

Isto não afeta o rollback do **frontend**, que segue disponível por digest, nem o do **backend**,
que tem imagem anterior íntegra.
