# GTL-Z01 — Master Integration Control

**Autor/auditor:** Codex56-Z (`w8:p4`)
**Autoridade destinatária:** General-Tech-Lead Codex56-TL (`w5:pC`)
**Snapshot estável de status/fingerprints:** 2026-07-27T12:16:22Z
**Repositório:** `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team`
**Convenção de caminhos:** salvo quando começam por `.deploy-control/`, `.github/` ou `.kiro/`, caminhos de fonte são relativos a `multica-auth-work/`.
**Base comum dos patches recentes:** `0cb8aebb5aff79cb430b3740d22fadc53c0116fd`
**Modo:** READ-ONLY sobre código, OpenSpec, board, config, banco e live. A única escrita de conteúdo desta auditoria é este relatório e seu check-out. Não houve build, teste, migration, deploy, restart, chmod, deleção, leitura de valor secreto, task paga ou rerun.

---

## 0. Veredito executivo

**VEREDITO GLOBAL: BLOCK para integração/cutover único agora.** Há material recuperável, mas nenhum “PASS” documental autoriza promoção. O caminho crítico seguro é:

1. congelar ORQ-26 no fingerprint atual e fechar o gate DB-backed; os gates estáticos atuais não substituem S1–S7 e a proposta de CI root permanece BLOCK;
2. registrar de forma durável a sequência proposta de migrations e executar a lane vermelha serial;
3. integrar ledger `127` somente após resolver o segredo HMAC por injeção sem exposição;
4. integrar ORQ-21 + ORQ-12 + a parte de schema/wire da ORQ-13 em uma evolução `128` expand-first, sem os gerados atuais;
5. comprovar usage real e então pricing/ORQ-14/15/20;
6. construir rollback T2 de um comando para o **green atual**, não para backup antigo;
7. somente depois executar gates live com cards sintéticos novos e autorização explícita.

As nove issues históricas **ORQ-12/13/15/16/17/18/21/22/23 permanecem preservadas e não podem ser rerodadas**. `tool_use` prova início, não conclusão; telemetria nunca autoriza replay (`RCA-HANDOVER-20260727.md:211-226`). ORQ-22 estar `done` não torna sua task histórica candidata a replay.

### Achados de maior prioridade

- Há **18 worktrees** registrados. Seis patches recentes têm conteúdo útil ou rejeitado; ORQ-12 está limpo; o root é um checkout misto e sujo, inadequado como staging de integração.
- A migration **127 está disputada por três desenhos**: ORQ-13, ORQ-12/account+tier e ledger. A reserva canônica é definida em §4: ledger=`127`; account+usage dimensions=`128`; nenhum arquivo é renomeado nesta auditoria.
- ORQ-26: patch atual `6ce6a0a4...` tem gofmt/build/vet/compilação estáticos verdes, mas S1–S7 DB-backed ainda não executaram (`gtl-r03-orq26-gate-execution.md:191-220`). O PASS GTL-65 foi **revogado por steering direto**: workflows sob `multica-auth-work/.github/workflows/` são inertes no repositório root. A proposta atual é exatamente 3 arquivos (`.github/workflows/orq26-db-gate.yml` root + 2 server files), `workflow_dispatch`, sem PR/merge, e permanece **BLOCK aguardando security review GTL-69 e resolução da inviabilidade técnica do dispatch fora da default branch**. A alternativa V2 `push`-only existe, mas contradiz o steering atual e não está autorizada.
- ORQ-23: branch antiga **jamais deve ser reaplicada**; os dois helpers `execenv` divergem do root hardened e podem reintroduzir `cli.log`/cópia insegura (`RCA-HANDOVER-20260727.md:462-480`).
- Kanban V4 e V5 estão revogados; V6 também recebeu BLOCK. O artefato mais novo é V7/GTL-75, **DESIGN-ONLY ainda sem peer review**: corrige estado durável/hidratação, mas não é PATCH/GATE PASS (`gtl-kanban-v5-deep-sql-audit.md:1-15`; `gtl-kanban-v6-peer-review.md:1-17,314-326`; `gtl-kanban-integrity-monitor-v7.md:1-9`).
- Skills é **BLOCK / PENDING PROOF**, não “nomes inventados”: steering direto do owner designa SkillsHub LOCAL, derruba o teto artificial de 48 e exige export de `skills-manifest.json` + SHA-256 antes da cópia; GTL-55 registra a mesma correção (`gtl-skills-rollout-second-peer-review.md:11-18,34-44,60-65,90-93`). A mensagem primária deve ser anexada ao pacote de execução.
- Segurança: `backend-recover.sh` tem marcador de chave sensível e modo inseguro, logo requer contenção e decisão de rotação fora de banda; **não foi lido valor e não se afirma que seja segredo ativo** (`gtl-tmp-umask-remediation-plan.md:9-13,117-128`).

---

## 1. Método, comandos e regra de precedência

### 1.1 Comandos read-only executados

```text
git worktree list --porcelain
git -C <cada-worktree> status --short --branch --untracked-files=all
git -C <cada-worktree> diff --name-status --no-renames
git -C <cada-worktree> diff --cached --name-status --no-renames
git -C <cada-worktree> log -1 --format=...
git -C <cada-worktree> diff --stat / --numstat / --binary
git -C <cada-worktree> ls-files --others --exclude-standard
git for-each-ref ... refs/heads refs/remotes
git merge-base --is-ancestor <head> <base>
find <worktree>/multica-auth-work/server/migrations -maxdepth 1 -name '127*'
sha256sum sobre streams de `git diff --binary` e listas de untracked
git diff --check (apenas quando citado por artefato; não usado para auto-PASS)
cmp -s root/<arquivo> ORQ23/<arquivo>
nl -ba <evidência/fonte> | sed/grep ...
du -sh <cache>; wc -l sobre lista untracked
herdr pane list; herdr agent list; herdr pane run w5:pC <checkpoint>
```

Não foi executado `go`, `pnpm`, `sqlc`, `psql`, `docker`, `systemctl`, `chmod`, `rm`, `git add`, `git commit`, `git checkout`, `git rebase`, `git merge` ou qualquer endpoint mutante.

### 1.2 Precedência de evidência

Ordem canônica:

1. **schema/fonte/diff/status medidos no corte**;
2. **revisão independente posterior** ligada ao mesmo fingerprint;
3. evidência live com hash/digest e timestamp;
4. design revisado;
5. plano/relato do autor;
6. status de task/issue isolado.

`PASS` do autor não prevalece sobre teste que não rodou, diff posterior ao gate, schema real ou revisão independente posterior. Evidência sem fingerprint não se transfere a conteúdo uncommitted que mudou.

### 1.3 Vocabulário de gate obrigatório

| Estado | Significado | O que **não** significa |
|---|---|---|
| **DESIGN PASS** | arquitetura/regras aprovadas em leitura | código existe ou compila |
| **PATCH PASS** | diff estático aprovado e fingerprintado | testes passaram |
| **GATE PASS** | comandos reproduzíveis rodaram no mesmo fingerprint, sem skip/falso verde | live foi promovido |
| **LIVE PASS** | artefato por hash/digest promovido e health/funcional verificados | issue automaticamente aceita |
| **BLOCK** | falta factual ou contradição material | autorização para “tentar assim mesmo” |
| **PENDING PROOF** | premissa plausível, mas não verificável deste host | premissa falsa |

---

## 2. Inventário de worktrees, patches, arquivos e owner

### 2.1 Patches recentes sobre `0cb8aebb...`

Fingerprints abaixo são do corte desta auditoria e **não são gates**; qualquer mudança exige novo fingerprint e novo gate.

| Owner/branch | Worktree | Estado e arquivos tocados | Fingerprint |
|---|---|---|---|
| **Codex56#B** — `agent/codex-b/orq-18-runtime-delete-ui` | `.../91e70c79/workdir/repo` | DIRTY: `packages/views/runtimes/components/runtime-list.tsx`; `runtime-row-menu.test.tsx`. Direção recuperável; integração de produção retida por política de histórico e testes faltantes. | diff `51d9786e...49afc823`; sem untracked |
| **Opus48#A** — `agent/opus48-a/orq-12-task-usage-account-id` | `.../d4f7f744/workdir/repo` | **CLEAN; zero patch**. ORQ-12 só existe como design/auditoria. | diff vazio `e3b0c442...b855` |
| **Codex56#B** — `agent/codex-b/orq-21` | `.../e39199ed/workdir/repo` | 3 untracked: `scripts/staging/seed_approved_assignment.sql`; `server/internal/credentialregistry/resolver.go`; `resolver_integration_test.go`. Não integrável: zero callers, seed reativa/reset state, path não confiável. | tracked vazio; lista untracked `b82b8678...671783` |
| **Codex56#A** — `agent/codex-a/orq-23` | `.../ff121b28/workdir/repo` | 6 tracked: `daemon.go`, `execenv/antigravity_home.go`, `execenv/kiro_home.go`, `pkg/agent/antigravity.go`, `models.go`, `models_test.go`; 9 untracked: `credential_home.go`, teste, 2 units systemd e 5 arquivos OpenSpec. **Quarentena/known-bad como fonte de merge.** | diff `37e3c9c9...ce3b0b`; lista untracked `bd795dc3...ba57f` |
| **Opus48#B** — `agent/opus48-b/orq-13-thinking-level` | `/workspace/worktrees/gtl-i03-orq13-phase1` | DIRTY: 7 tracked — `daemon.go`, `daemon/types.go`, `handler/daemon.go`, `generated/models.go`, `generated/task_message.sql.go`, `generated/task_usage.sql.go`, `queries/task_usage.sql`; 4 untracked — `daemon/task_usage_thinking_level_test.go`, `handler/task_usage_thinking_level_test.go`, migrations `127_task_usage_thinking_level.{up,down}.sql`. A migration/testes são material de salvage; gerados atuais não. | diff `b87bc5ca...d04707ce`; lista untracked `b218c98a...8cf3326f` |
| **Kiro-Opus5** — `agent/kiro-opus5/orq-26-contract-fix` | `/workspace/worktrees/gtl-orq26` | `server/internal/handler/file.go`, `file_test.go`; mais **4.780** arquivos untracked em `.gtl-orq26-gate/gocache/**` no snapshot. Patch estático e gates gofmt/build/vet/compile PASS no mesmo fingerprint; S1–S7 DB-backed BLOCK. | diff `6ce6a0a4...4bc0`; lista untracked `c5a90ff9...be84fc` |
| **Agy-P0-A7** — `agent/agy-a7/orq26-consumer-tests` | `/workspace/worktrees/gtl-orq26-consumer-tests` | `packages/core/api/client.test.ts`, `schema.test.ts`. Gate Vitest/tsc reportado 95/95, mas deve ser repetido após composição com server e fingerprint conjunto. | diff `5ce9cc95...9a20`; sem untracked |

**Observação de corrida:** o status ORQ-13 ganhou arquivos durante a janela read-only. A tabela usa o snapshot de 12:16:22Z; nenhum hash anterior pode ser reutilizado. ORQ-26 preservou o diff `6ce6...`, mas o volume de cache variou e nunca integra.

### 2.2 Checkout principal/root — owner misto, não staging

Branch `integration/dev-transition-candidate-20260719`, HEAD `0cb8aebb...`, ahead 5. Owner real é **misto/não atribuível por branch**. Estado tracked de produto:

```text
packages/views/agents/components/create-agent-dialog.test.tsx
packages/views/agents/components/create-agent-dialog.tsx
packages/views/agents/components/inspector/thinking-picker.tsx
server/cmd/server/runtime_sweeper.go
server/internal/daemon/brain_integration.go
server/internal/daemon/brain_integration_test.go
server/internal/daemon/daemon.go
server/internal/daemon/execenv/antigravity_home.go
server/internal/daemon/execenv/antigravity_home_test.go
server/internal/daemon/execenv/execenv.go
server/internal/daemon/execenv/kiro_home.go
server/internal/daemon/execenv/kiro_home_test.go
server/internal/service/task.go
server/internal/service/task_complete_race_test.go
server/pkg/agent/antigravity.go
server/pkg/agent/models.go
server/pkg/agent/models_test.go
server/pkg/db/generated/issue.sql.go
server/pkg/db/queries/issue.sql
```

Também há `HERDR_COMMS_GUIDE.md` modificado, nove prompts + `bin/prodex` deletados, e grande conjunto untracked em `.deploy-control/**`, `.kiro/**`, `credential_home*`, units, helpers de cópia e OpenSpec. O root contém 739 inserções/101 remoções apenas nos 19 arquivos tracked de produto. **Regra:** não integrar nele; primeiro criar, em execução futura autorizada, worktree limpo no HEAD aprovado e reaplicar somente pacotes fingerprintados.

Sobreposição root × ORQ-23:

- byte-idênticos: `daemon.go`, `pkg/agent/antigravity.go`, `models.go`, `models_test.go`;
- **diferentes:** `execenv/antigravity_home.go`, `execenv/kiro_home.go`; o root contém hardening posterior. Isto prova por conteúdo que merge mecânico da ORQ-23 regrediria segurança/capacidade.

### 2.3 Worktrees P0 antigos

| Branch/worktree | Relação com base | Arquivos/estado | Disposição |
|---|---|---|---|
| `work/p0-w1-central-20260720` | DIVERGED | commits: `docs/deploy/omniroute-operational-acceptance-procedure.md`, `cmd/multica/cmd_daemon.go`, `credential_file_source.{go,test.go}`, `runtimeenv/adapter_test.go`; dirty local nos dois `credential_file_source*` | STALE/HOLD; revisar hunk a hunk, nunca merge de branch |
| `work/p0-w2-gateway-20260720` | DIVERGED | 31 arquivos sob `server/internal/daemon/gateway/**`: classification/client/dispatch/executor/model_projection/selection/telemetry, testes e testdata `omniroute-8586-injection-v3*` | STALE/HOLD; não é known-bad por si, mas não entra sem rebase/review novo |
| `work/p0-w3-runtime-20260720` | DIVERGED | `server/internal/daemon/runtimeenv/adapter_test.go` | STALE/HOLD |
| `work/p0-w4-ops-20260720` | DIVERGED | `docs/deploy/omniroute-operational-acceptance-procedure.md` | STALE/HOLD |
| `integration/agent-brain-p0` | HEAD `67ab777...` ancestral da base | clean | ABSORVIDO; nenhum patch atual |
| `work/p0-acceptance-20260721` | mesmo HEAD ancestral | clean | ABSORVIDO |
| `work/p0-failures-20260721` | mesmo HEAD ancestral | clean | ABSORVIDO |
| `work/p0-lifecycle-20260721` | mesmo HEAD ancestral | clean | ABSORVIDO |
| `work/p0-retry-20260721` | mesmo HEAD ancestral | clean | ABSORVIDO |
| `work/p0-routes-20260721` | mesmo HEAD ancestral | clean | ABSORVIDO |

Total: 18 worktrees = root + 7 recentes + 10 P0 antigos/absorvidos.

---

## 3. Colisões de arquivo e schema

### 3.1 Migration 127 — resolução canônica

Estado factual: base termina em `126_runtime_profile_protocol_family_native_runtimes`; o único `127*` materializado em worktree é ORQ-13. Porém três documentos reservam semanticamente o mesmo número:

| Candidato | Conteúdo | Estado |
|---|---|---|
| ORQ-13 | `127_task_usage_thinking_level` | patch existe; número não reservado; não integrar/renomear agora |
| ORQ-12/account+tier | `127_task_usage_account_id` ou combinação account+tier | design; não existe no worktree ORQ-12 |
| Ledger V2 | `127_task_ledger_summary` | design PASS condicionado a P0 secret; sem patch |

**PROPOSTA CANÔNICA / PENDING DURABLE RESERVATION — sem renomear nada agora:**

1. **127 — `task_ledger_summary`**: ledger durável, tabela/queries próprias.
2. **128 — `task_account_usage_dimensions`**: única evolução expand-first reunindo ORQ-21/12/13: snapshot imutável da conta na task, `task_usage.account_id` + `thinking_level`, índices/queries/wire. O hunk SQL da ORQ-13 é portado, não cherry-picked com número 127.
3. **129 — `project_squad_lead`**.
4. **130 — `runtime_history_retention`**, somente após decisão de retenção do owner.
5. **131 — reservado para Kanban integrity somente se V7 passar peer review e exigir schema; não criar enquanto V7 estiver design-only.**

A ordem foi escolhida por colisão operacional, não por número de issue: ledger usa a primeira lane SQLC e fecha replay durável; `128` atomiza account/tier para não haver duas migrations financeiras concorrentes; 129/130/131 esperam a lane. **Ainda não é reserva operacional:** este relatório está untracked e o único `127` materializado é ORQ-13. O General-TL deve registrar/fingerprintar a reserva no controle autoritativo antes de qualquer writer. A afirmação V7 de que 127 pertence à ORQ-13 é stale perante esta proposta e não prevalece até ruling durável.

### 3.2 Resolução técnica do conflito de identidade em `task_usage`

Schema atual tem `UNIQUE(task_id,provider,model)` (`server/migrations/032_task_usage.up.sql:1-14`). O patch ORQ-13 atual argumenta corretamente que tier divergente na mesma task não deve criar segunda linha (`127...up.sql:9-16`). Já o design ORQ-12 propõe account+tier na chave. A resolução canônica é:

- `credential_account_id` e `thinking_level` são **snapshot imutável da execução**, não dimensão livre enviada pelo daemon;
- o backend grava snapshot na transição claim/dispatch e é autoridade;
- `task_usage` copia o snapshot e mantém `UNIQUE(task_id,provider,model)`; update com account/tier divergente falha fechado em vez de abrir linha duplicada;
- histórico antigo fica `account_id=NULL`, `thinking_level=NULL`/sentinel explicitamente documentado; nada é inventado;
- preço nunca é materializado nos rollups; relatórios iniciais por conta/tier leem raw usage ou uma nova projeção não destrutiva;
- se escala exigir rollup account/tier, criar projeção nova depois de prova, em vez de reescrever PKs 073/084/101/102 no primeiro cutover.

Isto impede double-count, misattribution após rotação e rollback destrutivo. `assignments` é linha mutável por `agent_id` (`123_rotation.up.sql:36-42`), portanto resolver account no momento tardio de usage é tecnicamente inválido.

### 3.3 SQLC e gerados

ORQ-13 toca `pkg/db/generated/models.go`, `task_usage.sql.go` e **`task_message.sql.go`**, embora task message não pertença ao escopo; o diff também materializa structs de migrations antigas. Disposição:

- reaproveitar somente SQL/query/source reviewado;
- descartar todos os gerados atuais;
- executar `sqlc generate` uma vez, na lane exclusiva, depois de 127 integrado/rebaseado e 128 escrito;
- checkout limpo deve produzir diff somente nos gerados esperados; qualquer reorder de `task_message` sem mudança de query reprova.

### 3.4 Colisões por arquivo

| Arquivo/família | Pacotes em colisão | Regra |
|---|---|---|
| `server/internal/daemon/daemon.go` | root/ORQ-23, ledger, account snapshot, usage probe/pricing wire | **lane vermelha serial**; um owner por wave |
| `server/internal/service/task.go` | root, ledger, account snapshot | serial após fingerprint do root; nunca editar em paralelo |
| `server/internal/handler/daemon.go` | ledger endpoint, ORQ-12/13 usage | serial |
| `server/internal/daemon/client.go` | ledger transport, usage/account | serial |
| `server/pkg/db/generated/**` | ledger 127, account 128, squad 129, runtime 130 | lock global exclusivo |
| migration numbers | todos pacotes SQL | reserva central 127→131 |
| `queries/task_usage.sql` | ORQ-13, ORQ-12, relatórios ORQ-15/20 | 128 primeiro; relatórios somente após unlock |
| `execenv/antigravity_home.go`, `kiro_home.go` | root hardened × ORQ-23 antiga | root/freeze prevalece; branch antiga proibida |
| `runtime_sweeper.go` | root dirty × futuro Kanban monitor | Kanban aguarda peer V7 + rebase; V4/V5/V6 BLOCK |
| ORQ-26 server × consumers | arquivos disjuntos | desenvolvimento paralelo permitido; gate e promoção conjunta |
| ORQ-18 × ORQ-26 | sem overlap textual | ORQ-18 pode fechar testes em paralelo, mas browser/integration após ORQ-26 para isolar causa |

### 3.5 FILES_LOCKED obrigatórios

Nenhum pacote começa sem registro de lock. Prefixos/globs contam como lock real.

- **L-ORQ26:** `handler/file.go`, `file_test.go`, `packages/core/api/{client.test,schema.test}.ts`, hooks/CLI/mobile/E2E adicionados ao gate.
- **L-ORQ18-PREP:** `runtime-list.tsx`, `runtime-row-menu.test.tsx`, `delete-runtime-dialog.test.tsx`, novo teste de mutations. Produção fica HOLD até runtime retention.
- **L-RED-127:** migration 127, query ledger, `generated/**`, commitledger, `daemon.go`, `client.go`, `handler/daemon.go`, `task.go`, `autopilot.go`, `main.go` e testes.
- **L-RED-128:** migration 128, `agent.sql`, `task_usage.sql`, `generated/**`, claim types/handler/service, `credential_home*`, `daemon.go`, client/handler usage e testes.
- **L-PRICE:** `pkg/agent` adapters comprovados, `daemon.go`, client/handler usage, `metrics/pricing.go`, `metrics/business.go` e testes; somente depois de L-RED-128.
- **L-KANBAN-V7 (piso, ainda não aprovado):** migration up/down de índice+estado, `queries/kanban_integrity.sql`, `generated/{kanban_integrity.sql.go,models.go}`, `runtime_sweeper.go`, `main.go`, `protocol/events.go`, service/handler + testes, `packages/core/types/{events,kanban}.ts`, store/hook realtime no nível workspace, `use-kanban-anomalies.ts`, `board-view.tsx`, `board-card.tsx`, hidratação/reconnect, locale/ARIA. Não liberar até peer review V7 PASS e reserva 131.
- **HOST-OPS:** unit systemd, binários, imagens, secrets handles, `/tmp`, proxy e CLI installs não são file locks Git; são locks operacionais exclusivos e STOP-AND-WAIT.

---

## 4. Estado real dos pacotes/gates

| Pacote | Design | Patch | Gate | Live | Ruling |
|---|---|---|---|---|---|
| ORQ-26 server | PASS direcional | PASS estático `6ce6...` | gofmt/build/vet/compile PASS; **DB S1–S7 BLOCK** | FAIL funcional atual (chat/upload) | CI root proposta BLOCK; GTL-65 revogado; aguardar GTL-69 do desenho atual |
| ORQ-26 consumers | PASS | PASS estático | Vitest 95/95 + tsc relatados, mas separado | não promovido | compor com server e repetir |
| ORQ-18 affordance | PASS condicionado | recuperável | incompleto: success/error/cancel/pending/refetch/permissão real | não promovido | preparar testes; produção HOLD por retenção |
| ORQ-21 bridge | direção backend-claim PASS | patch atual **REJECT** | teste pode skip; zero callers | tabelas live vazias | reescrever dentro de 128 |
| ORQ-12 | PASS revisado (snapshot no claim) | ausente | ausente | não existe account no usage | implementar em 128 |
| ORQ-13 phase 1 | migration mínima recuperável; pricing design PASS | misto/gerados contaminados | ausente no fingerprint final | AGY sem custo/usage | portar SQL/wire, descartar gerados |
| ORQ-14 usage | desenho de sonda PASS com premissa stale corrigida | ausente | requer rebuild/restart autorizado | Codex tem usage; AGY/Kiro não comprovados | primeiro medir schema; não inventar parser |
| Pricing versionado | DESIGN PASS | ausente | 13 testes desenhados, não rodados | preço por tier não existe | owner fornece valores/política; código depois de 128 |
| Ledger V2 | DESIGN PASS condicionado | ausente | ausente | hook não wired; secret vazio fail-closes tudo | P0 secret antes de persistência/wiring |
| Kanban V4/V5/V6 | **BLOCK/revogados** | ausente | não aplicável | nenhum monitor | V7/GTL-75 é design-only sem peer; revisar antes de reservar 131 |
| CLI alignment | DESIGN PASS com correções | ausente/operação | fila/trust/contextos ainda a provar | frota desalinhada | janela canary owner-authorized |
| Skills | **BLOCK/PENDING PROOF** | ausente | falta manifest+hash LOCAL | não instalado | GTL-55 prevalece; não limitar a 48 |
| Security /tmp+umask | plano PASS, fatos parcialmente contraditórios | operação ausente | nenhum aceite aplicado | umask permissivo persiste | conter sem ler valores; rotação out-of-band |
| ORQ-23 cutover | plano de alvo/hash útil | branch antiga known-bad | rollback não ensaiado em 1 comando | daemon green `88ca...`; sem rollback T2 seguro | não reaplicar branch; construir wrapper |
| ORQ-15/20 | DAG/design | ausente | depende de 128+usage+price | relatório por conta/tier inexistente | posterior |

### Falsos verdes que devem ser anulados

1. `go test` exit 0 com `Skipping tests: database not reachable` = **zero teste**, não PASS.
2. `gofmt -l` exit 0 com stdout não vazia = FAIL de formato.
3. `completed` em task não aceita issue (`RCA-HANDOVER:384-387`).
4. schema Zod verde com `/api/attachments/<id>/download` apontando para row inexistente = falso verde/404.
5. discovery AGY de primeiro HOME = amostra, não paridade dos quatro slots.
6. H1/HTTP 200 = health passivo, não task-capable/live PASS.
7. peer review de plano de rollback = design PASS, não rollback ensaiado.
8. preço base aplicado a tier desconhecido = sucesso falso; deve registrar `tier_unknown` e não cobrar silenciosamente.
9. ausência de skill em ORQ2 = “não instalada”, não “inventada”; existência continua pending proof até manifesto LOCAL.

---

## 5. Evidências contraditórias/stale e qual prevalece

| Contradição | Evidência prevalente | Motivo/ruling |
|---|---|---|
| ORQ-26 “PASS pronto para merge” vs R03 | `gtl-r03-orq26-gate-execution.md` + fingerprint atual | static gates foram refeitos e passaram; execução DB-backed S1–S7 continua ausente, portanto GATE PASS não existe |
| GTL-65 “CI oficial PASS” | steering direto Z01 ORQ26 CI + estrutura real do repo | workflows em `multica-auth-work/.github/workflows` são inertes. Proposta root de 3 arquivos continua BLOCK; `workflow_dispatch` sem presença na default branch é incompatível com “sem PR/merge” e GTL-69 deve resolver |
| Correção ORQ-26 com `attachmentDownloadPath(id)` | `RCA-HANDOVER:429-449` + patch P1/P2/P3 | rota exige row; fallback não tem row; 404 comprovável por fonte |
| ORQ-23 rollback “PASS” | `RCA-HANDOVER:474-480` + leitura do plano | plano lista múltiplos comandos; não há wrapper de um comando ensaiado. Design de baseline PASS, live rollback BLOCK |
| ORQ-23 branch reaplicável | `RCA-HANDOVER:462-480` + `cmp` root/worktree | helpers antigos divergem do hardening; branch é somente proveniência |
| Kanban V4/V5/V6 “corrigidos” | GTL-54, GTL-60, GTL-67 + schema real | V4 erra nomes; V5 gera falsos positivos; V6 falha estado multi-replica/hidratação. V7 é o mais novo, mas ainda sem review |
| Skills “98 nomes inventados/teto 48” | **GTL-55** | owner definiu SkillsHub LOCAL; falta apenas manifest/hash/proveniência e limites reais |
| Segurança: handshake/rev chamados tokens | `RCA-HANDOVER:161-162` | autor identificou strings de teste de pane; classificação de segredo está stale |
| `arch.txt`: prosa sem valor vs auditoria “atribuição com valor” | **UNRESOLVED sem ler valor** | conter pelo modo inseguro; custodiante decide rotação fora do contexto; não afirmar segredo nem falso positivo |
| `backend-recover.sh` “segredo legítimo” | steering owner + GTL-04 | só se prova marcador sensível e modo inseguro. Classificar remediação/rotação pendente, sem ler valor |
| ORQ-13 patch antigo de 13 paths | status/fingerprint do worktree atual | ref antigo não existe; worktree atual mudou e tem conjunto diferente; artefato antigo serve só para anti-patterns |
| Kiro gateway receipt passivo | `gtl-kiro-gateway-receipt-audit.md:1-20` | resposta HTTP fica no processo Kiro; daemon não vê headers sem proxy/interceptor |
| AGY snake_case “causa” | desenho de sonda GTL-17 | formato não foi medido; parser antes da sonda seria especulativo |
| CLI smoke ORQ1 | `gtl-cli-alignment-peer-review.md:30-45` | ORQ-27/28/29 foram ORQ2; canary ORQ1 exige prova própria |

---

## 6. DAG executável e ondas máximas paralelas

### 6.1 DAG

```text
F0 freeze/fingerprint + worktree limpo + locks + decisões de negócio mínimas
 ├─ A1 ORQ-26 server+consumers ──> A1C CI root/GTL-69 BLOCK ──> A1G DB gates ──> A1L browser/live autorizado
 ├─ A2 ORQ-18 testes/permission prep ───────────────┐
 ├─ A3 Skills manifest+SHA LOCAL (sem import)       │
 ├─ A4 Kanban V7 peer-review-only                 │
 ├─ A5 Security/CLI preflight-only                 │
 └─ R127 Ledger P0 secret -> schema/seam -> transport -> wiring -> gates
                                                   │
R127 integrated/unlocked                            │
 └─ R128 ORQ21 authority + ORQ12 snapshot + ORQ13 usage dimensions -> gates
      ├─ U1 usage schema probe autorizado -> AGY/Kiro/Codex synthetic evidence
      │    └─ U2 parser/receipt apenas conforme shape medido
      ├─ P1 pricing versionado + valores/política owner -> gates
      ├─ T14 integration tests
      └─ R129 project squad (lane SQL seguinte)
           ├─ R130 runtime retention -> libera A2 produção
           └─ Reports ORQ15 -> ORQ20
                └─ K7 Kanban monitor (somente após peer review V7 PASS + reserva durável 131)
                     └─ C23 cutover backend-first -> daemon-second -> live gates -> rollback drill
```

### 6.2 Wave 0 — controle, sem produto

Saídas: snapshot de todos os diffs/untracked; worktree de integração limpo; reserva 127–131; locks publicados; nenhum agente trabalha no root. Isto é técnico, não decisão do owner.

### 6.3 Wave 1 — máximo paralelismo seguro

Podem ocorrer em **seis frentes disjuntas**:

| Pacote | Autor/implementador | Reviewer independente | Locks | Saída |
|---|---|---|---|---|
| W1-A ORQ-26 | Kiro-Opus5 + Agy-P0-A7; integração Codex56-TL | Codex56#B; GTL-69 security separado | L-ORQ26 | server static PASS; root CI de 3 arquivos BLOCK; DB/core/CLI/mobile/E2E depois |
| W1-B ORQ-18 prep | Codex56#B | Agy-P0-A8 | L-ORQ18-PREP | testes completos; sem habilitar delete live |
| W1-C ledger 127 | Codex56-TL com design Opus48#A | Agy-P0-A8 ou Codex56#B que não escreveu o patch | L-RED-127 | schema/seam/transport/wiring; gate restart lógico |
| W1-D Skills proof | custodiante LOCAL produz manifest | Codex56#A valida hashes/proveniência | nenhum arquivo produto; staging externo futuro | `skills-manifest.json`, hashes, lotes dentro de limites |
| W1-E Kanban V7 review | Antigravity é autor V7 | Opus48#A + reviewer security independente | documentação apenas | validar estado durável, tenant, lock, hidratação, 14+ locks; sem código |
| W1-F ops preflight | Codex56#A (security) + Antigravity (CLI) | Agy-P0-A8/Codex56#B | HOST-OPS somente leitura | plano de janela, rollback e fila; nenhuma mutação |

**Integração da wave:** ORQ-26 pode entrar antes do ledger porque não há overlap. ORQ-18 produção continua HOLD. Ledger entra sozinho na lane vermelha.

### 6.4 Wave 2 — lane financeira serial

Após 127 integrado e todos os locks vermelhos liberados/rebaseados:

- **R128** é um único pacote técnico, não três branches: backend authority/claim, snapshot imutável, daemon mapping allowlisted, raw usage account+tier, queries e SQLC.
- Autor/integrador: Codex56-TL; material salvage Opus48#B/Codex56#B; reviewers independentes dos hunks: **Agy-P0-A8 + Antigravity**. Codex56#B pode explicar provenance, mas não revisar como independente o próprio salvage.
- ORQ-21 seed atual e todos os gerados ORQ-13 atuais ficam fora.

### 6.5 Wave 3 — prova de usage, pricing e migrations seguintes

Depois de R128:

- usage schema probe, com rebuild/restart somente sob owner gate e fila vazia;
- pricing em código versionado, sem preço inventado e sem fallback silencioso;
- ORQ-14 testes herméticos podem correr em paralelo com desenho de 129, mas não editar `daemon.go`/gerados simultaneamente;
- 129 e 130 permanecem seriais pela lane migration/SQLC.

### 6.6 Wave 4/5 — relatório, monitor e live

ORQ-15/20 só após account+usage+price. Kanban monitor só após peer review V7 PASS e reserva 131. CLI/security/Skills/LAN/cutover são operações separadas, cada qual com STOP-AND-WAIT. ORQ-23 é terminal; não é branch de código a reaplicar.

---

## 7. Gates de integração, live e rollback

### 7.1 Gate comum de pacote

1. base limpa e HEAD registrado;
2. `FILES_LOCKED` sem interseção com qualquer pacote ativo;
3. `git diff --binary | sha256sum` antes dos testes; mesmo hash depois;
4. `git diff --check` zero;
5. formatter com **stdout vazia**, não apenas exit 0;
6. testes targeted com `-count=1 -v`, nomes visíveis e zero `Skipping tests:` global;
7. full suite afetada, typecheck/lint/build do pacote;
8. nenhum teste usa banco compartilhado sem namespace/DB efêmero; nenhum trigger residual;
9. review independente no mesmo hash;
10. novo hash após qualquer correção invalida todos os gates anteriores;
11. CI ORQ-26: até GTL-69 do desenho atual, zero commit/push/dispatch. O gate deve residir em `.github/workflows/` da raiz, usar permissões mínimas/actions e imagem pinadas, zero secrets/artifacts com DSN, changed-file guard exato e runner efêmero; o mecanismo de trigger precisa ser tecnicamente executável e compatível com o steering.

### 7.2 Gate migration/SQLC

- reserva de número confirmada no HEAD integrado;
- up em DB efêmero com história fixture;
- rollback compatível com política: migrations financeiras expand-first **não executam down destrutivo** no rollback de app;
- conservação exata de input/output/cache-read/cache-write;
- história antiga sem conta/tier inventados;
- `sqlc generate` em clean checkout e segunda geração com diff zero;
- generated diff restrito às queries/schema do pacote;
- concorrência/tenant/provider/revocation testados fail-closed.

### 7.3 Gate ORQ-26 específico

Estado de entrada: os checks estáticos no `6ce6...` passaram; isto não é GATE PASS. GTL-65 está revogado. A proposta `workflow_dispatch`/sem PR/merge permanece BLOCK porque o arquivo novo não existe na default branch; GTL-69 deve revisar a variante exata do steering. Não substituir silenciosamente por `push`-only. Depois do ruling e autorização remota:

- S1–S7 DB-backed realmente executados;
- `file.go` e `file_test.go` gofmt clean;
- contextless: `id=""`, `url==download_url`, sem row fantasma;
- entity ref sem workspace: 4xx antes de storage;
- insert falho: cleanup best-effort + 500;
- C1–C3, chat regressions, CLI ID vazio, mobile response real e E2E 500;
- browser autenticado cobre clipe, drag/drop, feedback, avatar, comentário e quick-create;
- nenhum `ApiContractError` no happy path; malformed response continua fail-closed.

### 7.4 Gate financeiro/usage

- account resolvido no claim e gravado atomically com dispatch;
- payload só UUID pseudônimo + generation; nenhum path/e-mail/terminal/segredo;
- daemon deriva slot local allowlisted e falha fechado;
- rotation posterior não muda snapshot;
- wire account divergente é rejeitado;
- AGY usa model ID exato e tier vazio; Codex/Kiro usam tier explícito validado;
- sonda registra somente nomes/tipos JSON, nunca valores;
- preço tem provider/model/tier/effective_at/version/currency/policy;
- `tier_unknown`/`model_unknown` são visíveis; custo zero silencioso reprova;
- novas tasks sintéticas, nunca as nove preservadas.

**Gate de secrets:** quando HMAC/credenciais forem AWS-managed, carregar a orientação local `aws-secrets-manager` e resolver somente em runtime via `asm-exec`, sem recuperar/imprimir o valor (`.kiro/steering/aws-agent-rules.md:10-12`; `aws-agent-toolkit.md:3-7`). Se não forem AWS-managed, usar mecanismo handle-only equivalente aprovado; não criar dependência AWS por inferência.

### 7.5 Fila vazia obrigatória

Antes de build live, migration aplicada, recreate, restart, upgrade CLI, sonda ou cutover:

```sql
SELECT count(*)
FROM agent_task_queue
WHERE status IN ('queued','dispatched','running','waiting_local_directory');
```

Resultado deve ser **0 em duas leituras separadas por um intervalo de estabilidade**. A query de ORQ-23 que omite `waiting_local_directory` é insuficiente. Também reprova qualquer task ativa órfã por runtime/daemon.

### 7.6 AGY/Codex/Kiro obrigatórios

Todo gate live/cutover exige, por provider (não por ordem ambígua):

- **AGY:** online, discovery 11, task sintética concluída;
- **Codex:** online, discovery 19, task sintética concluída;
- **Kiro:** online, discovery 11, task sintética concluída;
- daemon/túnel/backend/frontend healthy, zero restart loop;
- usage/correlation conforme capacidade comprovada; ausência declarada, nunca mascarada.

Os smokes históricos ORQ-27/28/29 provam o live anterior no ORQ2, não o artefato futuro nem canary ORQ1.

### 7.7 Rollback por hash/digest

- registrar commit, diff SHA-256, binário SHA-256 e image digest antes da promoção;
- frontend/backend voltam ao digest green anterior, não a tag mutável;
- migrations 127/128 ficam expand-light durante rollback de app; não colapsar/apagar evidência;
- daemon só pode voltar ao green:
  `88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8`
  ou sucessor já aprovado;
- criar wrapper idempotente de um comando com verify-hash → staging/rename → restart → health → auto-restore em falha;
- ensaiar o wrapper sob owner gate; até isso, T2 está **sem rollback seguro de um comando**.

### 7.8 Known-bad / jamais restaurar

| Artefato | Hash/status | Proibição |
|---|---|---|
| `multica-auth-credential-home-v1.new` | `5f9ec49e...562e` | pre-token-only; jamais promover |
| `.pre-token-only-20260727T102815Z` | mesmo hash | reintroduz AGY task-incapaz |
| `.pre-agy-fix` | `f3463086...c1a` | anterior ao fix AGY |
| `.previous` | `e0510d7d...1ba6` | legado |
| branch `agent/codex-a/orq-23` | diff atual `37e3...` | não merge/cherry-pick/rebase mecânico; apenas proveniência |
| patch ORQ-21 atual | 3 untracked | seed/resolver não integráveis |
| antigos hunks ORQ-13 destrutivos descritos em GTL-40/41 | down com DELETE/rollup rewrite | não ressuscitar, mesmo que ref reapareça |
| revert total `d10d09e0` | 21 arquivos | proibido; preserva fail-closed e outros changes |
| edição não ratificada de `HERDR_COMMS_GUIDE.md` | root dirty | não promover como política |

Fonte dos quatro binários e hashes: `gtl-orq23-durable-cutover-plan.md:10-21`. A aparente cópia `gate1` com hash green é baseline válido por conteúdo, mas só vira rollback quando o wrapper e o drill forem aprovados.

---

## 8. Decisões de negócio do owner e autorizações de risco

### 8.1 Somente decisões de negócio/política

1. **Billing/comercial:** valores, moeda, fonte contratual, vigência, tratamento comercial de tier sem preço e orçamento de tasks sintéticas. Engenharia implementa fail-closed/miss explícito.
2. **Retenção:** prazo legal/comercial de retenção e se o produto oferecerá exclusão permanente. Engenharia escolhe tombstone/archival/snapshot e constraints; até o ruling, sem delete físico.
3. **Exposição:** se o produto será publicado, para qual audiência/organização e quais requisitos de compliance. Engenharia define URL/TLS/ACL/auth; até o ruling, loopback e nenhum LAN.
4. **Replay:** se retry manual poderá existir e por quanto tempo o produto deve manter evidência. Engenharia mantém default fail-closed, sem auto-replay e sem telemetria como autorização.
5. **Skills/catalog:** quais categorias organizacionais podem ser habilitadas e quem é o custodiante LOCAL. Engenharia valida manifesto, hashes, proveniência e limites; nomes permanecem pending proof.

### 8.2 STOP-AND-WAIT — autorizações de risco, não escolhas técnicas

Antes de push/branch remota CI, instalação/upgrade, rotação, chmod/umask, migration live, rebuild, restart, deploy, task paga ou rollback drill, a engenharia apresenta **um procedimento técnico fechado** com alvo, blast radius, custo, rollback e evidência. O owner apenas autoriza ou recusa a janela/risco; não escolhe migration numbers, SQLC, trigger CI, package boundaries, locks, estratégia de snapshot, comandos de teste ou SQL Kanban.

---

## 9. Caminho crítico ASAP sem rerun das nove issues

1. **Hoje, sem live:** congelar fingerprints; retirar root da função de staging; registrar locks e números 127–131.
2. **ASAP #1:** manter ORQ-26 server `6ce6...` congelado; compor consumers somente no gate de integração. Corrigir o plano root CI conforme GTL-69, sem executar commit/push/dispatch enquanto BLOCK; não adotar V2 push-only sem ruling que superseda o steering.
3. **Em paralelo:** fechar testes ORQ-18 (sem promover delete), produzir manifest Skills LOCAL, revisar adversarialmente Kanban V7/GTL-75, preparar security/CLI sem aplicar.
4. **Lane vermelha #1:** implementar ledger 127 com secret HMAC injetado de forma durável e sem valor no contexto; seam/transport/checker/autopilot; gates restart lógico. Não wiring antes do P0 secret.
5. **Lane vermelha #2:** R128 único. Descartar seed ORQ-21 e gerados ORQ-13; implementar claim snapshot/backend authority e daemon local map fail-closed.
6. **Prova controlada:** sonda schema usage com fila vazia e três tasks sintéticas novas, uma por runtime obrigatório. Não rerodar ORQ-* preservadas.
7. **Pricing/ORQ-14:** implementar apenas formatos medidos; owner injeta valores/política; provar miss e custo sem zero silencioso.
8. **ORQ-15/20:** relatório por account pseudônimo/model/tier/version; somas por conta = global.
9. **Runtime retention + ORQ-18 produção:** só após ruling de retenção e prova de conservação.
10. **Kanban V7:** só após review PASS, reserva 131 e decisão técnica entre escrita exclusiva na tabela de estado do monitor versus snapshot; continua proibido mutar issue/task ou enqueue.
11. **ORQ-23 terminal:** construir rollback one-command, backend-first/daemon-second, fila vazia, AGY/Codex/Kiro gates, browser, rollback drill; só então `in_review`.

Nenhum passo depende de “continuar” ou “rerodar” uma task histórica. Cards sintéticos de aceite recebem IDs novos e orçamento/autorização próprios.

---

## 10. Matriz de risco e blast radius

| Risco | Prob. | Blast radius | Detecção | Mitigação/stop |
|---|---:|---:|---|---|
| merge branch ORQ-23 antiga | alta se mecânico | daemon/3 runtimes; AGY task-incapaz | diff dos helpers/hash | proibir branch; portar somente hunk novo contra freeze |
| migration 127 duplicada | certa sem reserva | deploy DB bloqueado/inconsistente | scan filenames + migration table | reserva 127–131, lane serial |
| account resolvido tardiamente | alta sob rotação | billing histórico incorreto | teste rotation-after-dispatch | snapshot atômico no claim |
| generated SQLC contaminado | alta no patch atual | compile/API drift amplo | regenerate clean + diff | descartar gerados; gerar uma vez |
| ORQ-26 falso verde/CI inerte | já ocorreu | upload/chat/avatar/mobile | 8 leaves run+pass; workflow root | DB efêmero, fingerprint; GTL-69; trigger executável |
| preço/tier fallback silencioso | alta hoje | cobrança errada de todos tiers | `tier_unknown` metric/test | fail-open proibido; versionamento |
| AGY/Kiro usage especulativo | média | custo ausente/duplo | probe schema + request ID | parser só após shape medido |
| Kiro proxy receipt mal desenhado | média | tráfego/segredo/double count | proxy tests/SSE dedupe | não implementar passivamente; `(task,request_id)` único |
| runtime delete cascade | alta se UI liberada | perda de tasks/usage | contagem antes/depois | HOLD UI; política retention primeiro |
| Kanban estado/SQL inválido | alto até review V7 | monitor cego/falso evento/cross-tenant | DB fixture, failover, reconnect, schema | V7 review + 131 + locks completos |
| monitor auto-mutante | baixa se escopo respeitado | board/tasks duplicadas | strict read-only test | evento passivo; zero DML/enqueue |
| cache ORQ-26 variável (4.780 arquivos no snapshot) | média | disco/noise/status | `du`, untracked count | excluir do pacote; limpeza só com autorização separada |
| root dirty como staging | alta | mistura owners/regressões | status/diff | worktree limpo obrigatório |
| Skills sem proveniência | média | prompt/supply-chain de 10 agentes | manifest/hash/allowlist | BLOCK até GTL-55 proof |
| umask/permissões | alta atual | exposição local sistêmica | `stat/find` sem conteúdo | private TMPDIR/UMask gate; contenção/rotação owner |
| rotação baseada só em nome de chave | média | outage desnecessário | custodian out-of-band | conter primeiro; não ler valor; owner decide rotação |
| CLI upgrade simultâneo | média | frota indisponível/custo | canary + version pin | ORQ1 canary, contextos env, rollback exato |
| LAN com bypass anônimo | certa se publicar agora | API exposta | `/api/me` anônimo | STOP; HTTPS+ACL+401 antes |
| cutover sem rollback T2 | alta consequência | perda dos 3 runtimes | wrapper/drill ausente | nenhum daemon promotion antes do drill |

---

## 11. Critérios de parada

Parar imediatamente a wave/pacote se ocorrer qualquer um:

1. overlap de `FILES_LOCKED` ou owner desconhecido escrevendo no mesmo arquivo;
2. base/root sujo usado como staging;
3. número de migration já existe ou predecessor não foi integrado/rebased;
4. diff hash muda após início dos gates;
5. formatter emite caminho, teste skipa globalmente, DB fixture não é isolado ou SQLC segunda geração muda;
6. schema/fonte contradiz design;
7. segredo/valor aparece em stdout, argv, diff, log ou evidência;
8. fila ativa diferente de zero, inclusive `waiting_local_directory`;
9. rollback por hash/digest não existe ou aponta para known-bad;
10. AGY, Codex ou Kiro perde online/discovery/task capability;
11. miss de preço vira `$0` silencioso ou account/tier diverge do snapshot;
12. qualquer uma das nove issues preservadas recebe rerun/mutação automática;
13. monitor Kanban executa DML/enqueue ou usa status/tabela inexistente;
14. `/api/me` continua 200 anônimo em uma origem que será publicada;
15. manifest/hash Skills não casa, excede 1 MiB/file, 8 MiB/bundle, 128 files/bundle ou depth 4;
16. containment/rotação/permissão/restart/deploy é proposto sem autorização explícita do owner.

A parada preserva o último green; não tenta “corrigir live” no mesmo gate.

---

## 12. Check-out do auditor

- [x] RCA lido integralmente.
- [x] GTLs/revisões relevantes lidos até o snapshot, incluindo ORQ-26 R03/CI, Kanban GTL-60/GTL-67/V7-GTL-75, ledger V2, CLI, security e Skills/GTL-55.
- [x] 18 worktrees inventariados; status/diffs/fingerprints read-only.
- [x] colisões de arquivo/schema e precedência resolvidas; 127–131 marcados explicitamente como proposta pending durable reservation.
- [x] DAG, waves, locks, autores/revisores, gates, rollback, queue e runtimes definidos.
- [x] decisões técnicas resolvidas; somente decisões de negócio/risco mantidas para owner.
- [x] nove issues preservadas; zero rerun.
- [x] nenhum valor de segredo lido.
- [x] checkpoints enviados a `w5:pC` via `herdr pane run`; tentativa inicial por `tmux` falhou com `command not found`, depois canal correto foi resolvido por `herdr pane list`.

**Estado final do controle:** documento corrigido após revisão independente adversarial. Seu hash é registrado externamente no check-out para evitar circularidade. Ele não autoriza execução, commit/push CI, merge, build live, migration, deploy ou restart; autoriza somente preparação read-only de locks/worktrees e pedidos de gate.
