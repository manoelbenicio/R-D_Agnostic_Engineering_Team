# GTL-15 — Cross-gate execution plan para GTL-01..14

- Autor: Codex56#B, pane `w7:p4`
- Timestamp de corte: `2026-07-27T11:35:39Z`
- Modo: READ-ONLY; nenhum código, board, build, teste, deploy, restart,
  credencial ou issue foi alterado/executado.
- Escritor/integrador único: General-TL `w5:pC`.
- Escopo: todos os artefatos GTL-01..14 existentes no corte, incluindo a
  auditoria independente obrigatória do GTL-12.

## 1. Veredito executivo

**BLOCK para uma integração única imediata.** Há pacotes implementáveis em
paralelo, mas quatro grupos precisam ser separados por locks exclusivos:

1. upload/chat ORQ-26;
2. UI de runtime/squad;
3. lane serial de servidor/SQLC (`task.go`, `daemon.go`, migrations e gerados);
4. operações de host/produção, sempre com autorização escrita própria.

Os bloqueios materiais são:

- GTL-12 declara cobertura completa da ORQ-18, mas os testes reais não cobrem
  wrapper success/toast/close, cancel sem mutation, erro genérico e
  invalidation/refetch: **GTL-12 = BLOCK**;
- deletar runtime apaga tasks e `task_usage` por cascade; reassign do agente
  não preserva essa história;
- GTL-09 transmite referência de slot/path e permite fallback global, em
  conflito com o contrato pseudônimo e fail-closed;
- GTL-13 afirma receipt do OmniRoute sem captura; GTL-03 corretamente o trata
  como não provado;
- GTL-14 usa estados de issue errados e omite
  `waiting_local_directory`, gerando falsos positivos;
- migrations 127/105 foram propostas por pacotes diferentes ou já estão
  ocupadas historicamente; o General-TL precisa reservar a sequência.

Nenhuma das nove issues históricas ORQ-12, 13, 15, 16, 17, 18, 21, 22 e 23
deve ser rerodada. Aceites que exigirem execução usam somente cards sintéticos
novos e autorização específica.

## 2. Inventário e disposição

| GTL | Artefato | Disposição cross-gate |
|---|---|---|
| 01 | `gtl-orq26-contract-review.md` | **APROVAR COM GATES**: patch mínimo server-side; preservar `ApiContractError` fail-closed. |
| 02 | `gtl-ledger-implementation-audit.md` | **BLOCK até corrigir seam/transport/correlação/autopilot**; depois implementar na lane serial. |
| 03 | `gtl-cost-implementation-audit.md` | **APROVAR diagnóstico; BLOCK patch** até ponte de conta, captura de usage e decisão de preço versionado. |
| 04 | `gtl-tmp-umask-remediation-plan.md` | **STOP-AND-WAIT operacional**: rotação e host permissions exigem decisão própria; não pertence a worktree de código. |
| 05 | `gtl-orq19-stale-runtime-audit.md` | **BLOCK delete físico**: recomendação de reassign+delete não preserva histórico. |
| 06 | `gtl-orq20-reasoning-contract.md` | **APROVAR discovery/UI/validação**; custo depende de GTL-03/09/13. Defaults da frota são decisão do owner. |
| 07 | `gtl-orq17-lan-auth-readiness.md` | **BLOCK fail-closed** até HTTPS, auth real, ACL e rollback aprovados. |
| 08 | `gtl-ui-squad-runtime-integration.md` | **SPLIT** em mobile assignee, project lead e runtime delete; runtime delete converge com GTL-05/12. |
| 09 | `gtl-orq21-bridge-redesign.md` | **REJEITAR patch map atual**; manter apenas seleção no claim e validação física. Nunca transmitir path nem cair em global. |
| 10 | `gtl-orq15-integration-dependency-audit.md` | **APROVAR DAG revisado**: ORQ-21 → snapshot ORQ-12 → usage/custo/testes → relatório ORQ-15/20 → cutover ORQ-23. |
| 11 | `gtl-orq23-durable-cutover-plan.md` | **APROVAR alvo green; corrigir execução**: não é ainda rollback de um comando e `docker restart` não promove imagem. |
| 12 | `gtl-orq18-runtime-delete-readiness.md` | **BLOCK** após auditoria independente; ver §4. |
| 13 | `gtl-orq14-token-accounting-audit.md` | **EXPLORATÓRIO**: parser/receipt somente depois de wire evidence sem valores sensíveis. |
| 14 | `gtl-kanban-integrity-monitor.md` | **BLOCK design** até corrigir invariantes e semântica de status; monitor continua passivo. |

## 3. Contradições e resolução mandatória

### C1 — runtime reassign não preserva histórico

GTL-05:39-54 prova `agent_task_queue.runtime_id ON DELETE CASCADE` e que
`task_usage` cai em cascata pelas tasks. Porém GTL-05:149 e :163-169 afirma
que reassign do agente, seguido de DELETE, preserva histórico. Isso é
contraditório: `PATCH agent.runtime_id` não reescreve o `runtime_id` das tasks
históricas.

**Resolução:** até decisão do owner, runtime antigo fica offline/tombstoned e
não é fisicamente deletado. Para permitir delete, escolher uma destas
alternativas e migrar com teste de conservação:

- FK histórica `SET NULL` com snapshot imutável de provider/runtime; ou
- entidade runtime tombstone não deletável enquanto houver task/usage; ou
- export/arquivo durável explícito antes do delete.

Não aceitar “migrar task histórica para runtime novo”: isso falsifica
proveniência. GTL-08 e GTL-12 não podem promover um botão destrutivo antes de
essa política.

### C2 — GTL-09 viola identidade pseudônima e fail-closed

GTL-09:28-31 diz corretamente que o daemon não acessa Postgres, mas propõe
enviar `slot-140` no payload; :44-46 permite cair em comportamento global; e
:52-56 chama `internal/daemon/daemon.go` de “Backend Claim”. O código já tem
resolução local em `internal/daemon/credential_home.go` e
`daemon.go:3472`.

**Resolução:** o backend atomiza no claim somente `account_id` pseudônimo e a
versão/generation da assignment. O daemon converte esse ID por mapa local
versionado/allowlisted. Payload nunca contém HOME, path, slot, terminal ID,
e-mail ou segredo. Missing/revoked/mismatch falha fechado; nunca usa HOME
global.

### C3 — migrations e SQLC colidem

- GTL-02 reserva migration 127 para ledger.
- GTL-03 também assume migration 127 para account/tier.
- GTL-08 propõe `105_project_squad_lead`, embora o repositório já esteja em
  126 e 105 não seja uma reserva segura.

**Reserva serial proposta, conferida novamente no branch cut:**

1. `127_task_ledger_summary`;
2. `128_task_account_usage_dimensions`;
3. `129_project_squad_lead`;
4. `130_runtime_history_retention` somente se o owner escolher schema;
5. `131_kanban_integrity_monitor` somente após redraft GTL-14.

`server/pkg/db/generated/**` é uma lane exclusiva. Um pacote SQL só começa
após o predecessor ser integrado e o worktree seguinte ser rebased.

### C4 — tier explícito vs tier no model ID

GTL-03 rejeita inferir `thinking_level` por sufixo; GTL-06 registra que AGY
oferece tiers como parte canônica do ID do modelo. Ambos ficam compatíveis
assim:

- AGY: persistir o model ID completo e `thinking_level=''`; não parsear sufixo;
- Codex/Kiro: persistir `thinking_level` explícito validado;
- pricing usa chave `(provider, model_exato, thinking_level, effective_at)`.

Nunca preencher `thinking_level` adivinhando `-high`, `-thinking` ou outra
string.

### C5 — telemetria não provada

GTL-13:20-25 assume payload/headers de receipt OmniRoute e propõe interceptor.
GTL-03:173-231 demonstra que não há captura que prove a wire AGY/Kiro e que o
daemon não deve ser reiniciado com parser especulativo.

**Resolução:** antes de código, capturar apenas nomes de campos/shape e
correlation IDs de uma task sintética autorizada por provider; nenhum token,
prompt ou segredo. Se o CLI não atravessar o interceptor, receipt de gateway
não é fonte disponível nesse ponto. Ausência de prova mantém o pacote
bloqueado.

### C6 — reasoning não pode degradar silenciosamente

GTL-06:120 descreve incompatibilidade como warning + fallback. Isso muda
capacidade/custo sem consentimento.

**Resolução proposta:** handler rejeita 400 antes de persistir quando possível;
daemon falha fechado se catálogo mudou depois do dispatch. Fallback só pode
existir como opção explícita do owner e deve constar no receipt.

### C7 — rollback GTL-11 ainda não é de um comando

GTL-11:41-48 lista quatro comandos manuais; :75-77 usa `docker restart` para
backend. Restart não troca imagem e não prova consumo do `env_file`.

**Resolução:** wrapper idempotente de promoção/rollback, com hash esperado,
staging+rename atômico, backend recriado por compose/env_file, backend primeiro,
daemon depois, health automático e restauração do último green em falha. Os
artefatos `.pre-token-only`, `.pre-agy-fix` e `.previous` antigos são
**known-bad**; somente o green hash
`88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8`
ou um sucessor já aprovado pode ser baseline.

### C8 — GTL-14 geraria falsos positivos

- GTL-14:54 e :63 omite `waiting_local_directory` dos estados ativos;
- :67-83 usa issue status `completed`, mas issue usa `done`;
- :87-95 usa pai `completed`, mas issue pai usa `done`;
- :40-50 classifica todo atribuído sem task como anomalia universal, embora
  atribuição manual não implique enqueue automático.

**Resolução:** não implementar as SQLs atuais. O redraft deve usar
`queued/dispatched/running/waiting_local_directory`, `done/cancelled`, e um
indicador explícito de que enqueue era esperado. Evento continua somente
observacional; nunca muda issue, cria task ou reroda.

## 4. Auditoria independente obrigatória GTL-12

**Veredito: BLOCK.** O patch parcial em
`agent/codex-b/orq-18-runtime-delete-ui` é útil e isolado, mas a declaração
“cobertura total” não corresponde aos arquivos reais.

### O que o patch realmente cobre

- `runtime-list.tsx:442-494` troca kebab por botão visível, abre o diálogo e
  conserva o guard `canDelete`.
- `runtime-row-menu.test.tsx:132-145` confirma botão visível e abertura do
  heading; :147-170 cobre online/offline/cloud e ausência sem permissão.
- `delete-runtime-dialog.test.tsx:248-320` cobre os dois conflitos 409 e
  re-confirmação.
- `delete-runtime-dialog.test.tsx:409-427` cobre uma resposta light bem
  sucedida e a chamada do callback `onDeleted`.

### Coberturas declaradas que não existem

1. **Success do wrapper / toast / close:** produção faz
   `setDeleteOpen(false)` e `toast.success` em `runtime-list.tsx:487-490`.
   O teste novo abre o diálogo, mas não clica em confirm. O teste do diálogo
   chama um `onDeleted` fake, portanto não atravessa o wrapper.
2. **Cancel sem mutation:** não há teste que clique Cancel e prove zero
   chamadas. `delete-runtime-dialog.tsx:181-185` é código, não cobertura.
3. **Erro genérico:** `delete-runtime-dialog.tsx:155-175` tem o catch, porém
   nenhum teste injeta 500/503/network, verifica `toast.error` e confirma que o
   diálogo permanece aberto.
4. **Refetch/invalidation:** `packages/core/runtimes/mutations.ts:7-14` e
   :27-42 invalidam caches, mas ambos os testes mockam os hooks
   (`runtime-row-menu.test.tsx:30-37` e
   `delete-runtime-dialog.test.tsx:53-61`). Nenhum teste real espiona
   `QueryClient.invalidateQueries`.
5. **Pending/submitting:** o disable existe no componente, mas não há teste de
   duplo clique/close durante promise pendente.
6. **Linha citada:** GTL-12 aponta :199-204 como success/refetch; no arquivo
   real essas linhas são `beforeEach` e início do teste de render light.

### Gate para transformar BLOCK em PASS

Adicionar, sem ampliar produção:

- wrapper light success: API resolve → `onDeleted` real fecha diálogo, toast
  uma vez;
- wrapper generic error: toast error, diálogo continua aberto, nenhum success;
- Cancel: fecha e deixa os dois mutateAsync em zero;
- pending: confirm desabilitado e backdrop não fecha durante promise;
- mutation hooks reais: strict e cascade invalidam as chaves documentadas;
- manter os testes de 409 e permissão existentes.

Mesmo com esses testes verdes, merge de produção fica bloqueado por C1:
descoberta do botão não pode facilitar delete que apaga história.

## 4B. Segunda auditoria independente GTL-I01 / ORQ-26

**Veredito: BLOCK para integração; PASS apenas para a direção dos três hunks de
produção.** O worktree congelado
`/home/ec2-user/workspace/worktrees/gtl-orq26` contém exatamente
`server/internal/handler/file.go` e `server/internal/handler/file_test.go`.
Nenhum gate foi executado nesta auditoria READ-ONLY.

### Produção

- **P3 PASS:** `file.go:392-398` rejeita `issue_id`, `comment_id` ou
  `chat_session_id` sem workspace antes de `Storage.Upload`, que só aparece em
  `file.go:458`/`:487`.
- **P1 PASS:** `file.go:466-477` faz cleanup best-effort e devolve 500 quando
  `CreateAttachment` falha; não preserva mais o falso sucesso sem linha.
- **P2 PASS:** `file.go:500-506` devolve a forma contextless coerente:
  `id=""` e `url == download_url == markdown_url`.

### Qualidade real de S1..S7

| Caso | Veredito | Evidência real |
|---|---|---|
| S1 | PASS | `file_test.go:1287-1338` prova 200, ID vazio, igualdade `url/download_url`, um objeto e zero linha. Falta apenas pinar também `markdown_url==url`, campo novo de `file.go:504`. |
| S2/S3/S4 | PASS com lacuna pequena | `:1342-1369` cobre os três refs, 400 e zero objeto. O “gate chat não alcançado” é garantido pelo fluxo de `file.go:392-398`, mas não é instrumentado no teste. |
| S5 | PASS funcional / BLOCK de harness | `:1373-1407` prova 500, objeto removido e zero linha; remover o único objeto também prova que a key convertida foi a correta. A injeção de falha usada é global e insegura, abaixo. |
| S6 | PASS semântico / BLOCK de harness | `:1411-1436` prova 500, uma tentativa e objeto remanescente. `storage.Storage.Delete` não retorna erro (`internal/storage/storage.go:9-13`), portanto o no-op é a única falha observável que a interface atual permite. |
| S7 | PASS mínimo | `:1440-1497` cobre ambas as formas e presença de `id/url/download_url/filename`; não valida `markdown_url`, nem `filename` como string não vazia. |

### Bloqueio alto: trigger global em banco compartilhado

`forceAttachmentInsertFailure` cria nomes fixos
`orq26_block_attachment_insert` em `file_test.go:1236-1251`, e o trigger é
`BEFORE INSERT ON attachment FOR EACH ROW`, sem predicado. Assim, durante S5/S6,
**todo INSERT de attachment no mesmo banco falha**, inclusive outro processo ou
worktree. O risco é real porque `handler_test.go:38-45` usa o `DATABASE_URL`
compartilhado (ou o banco `multica` default), e `:26-27` mantém um único
`testPool`. Se o processo morrer, o trigger persiste. O cleanup em
`file_test.go:1254-1257` ainda descarta ambos os erros, podendo declarar o teste
verde e deixar o banco envenenado.

Gate obrigatório: substituir o trigger irrestrito por falha isolada por teste.
Sem ampliar arquivos de produto, a opção mínima é nomes únicos + `WHEN
(NEW.filename = <marcador único>)`, filenames únicos e cleanup que falha o teste
se não remover trigger/função. Advisory lock sozinho não basta, pois outros
consumidores do banco não o adquirem. A opção mais forte é DB/schema efêmero por
processo ou seam de `CreateAttachment` no handler.

### Consumidores e arquivos ausentes

- **Frontend compatível, gate incompleto:** o schema exige
  `id/url/download_url/filename` em `packages/core/api/schemas.ts:100-108`, e
  `use-file-upload.ts:77-80` cai corretamente em `att.url` quando `id==""`.
  C1 já existe em `use-file-upload.test.ts:75-84` e C2 em
  `api/client.test.ts:492-507`.
- **C3 ausente e um teste já contradiz o contrato:** não há teste de
  `ApiClient.uploadFile` aceitando a forma contextless completa. Em vez disso,
  `api/client.test.ts:729-751` responde apenas `{id,url}`; isso viola o schema
  fail-closed de `client.ts:1693-1697` e não constitui um sucesso válido.
- **CLI compatível:** `internal/cli/client.go:519-529` aceita ID vazio e exige
  URL; `client_test.go:360-378` já pina esse fallback. O avatar consome a URL,
  não o ID (`cmd/multica/cmd_agent.go:726-743`). O happy path do comando ainda
  simula ID não vazio em `cmd_agent_test.go:887-894`; deve ganhar variante com
  a forma real contextless.
- **Mobile compatível por schema, sem gate de upload:**
  `apps/mobile/data/schemas.ts:49-65` aceita ID vazio e defaulta os demais
  campos; `apps/mobile/data/api.ts:1263-1277` valida estritamente. Não existe
  teste de `apps/mobile/data/api.ts:1200-1277` para a nova forma contextless.
- **Superfícies ausentes:** o diff congelado não contém C1..C3 adicionais nem
  os gates U1..U9 listados em GTL-01:174-194. Em particular faltam feedback,
  avatar, chat/drag-drop, quick-create, mobile e E2E de 500 sem card fantasma.

### Gate para transformar GTL-I01 em PASS

1. isolar a falha de insert; cleanup verificado e marcador/filename únicos;
2. adicionar `markdown_url==url` a S1/S7 e preservar os regressivos de
   workspace/chat;
3. corrigir `packages/core/api/client.test.ts:729-751` e adicionar C3 com a
   forma contextless completa, mantendo C2 fail-closed;
4. pinar avatar CLI com `id=""` e adicionar teste mobile do response real;
5. executar S1..S7, regressões handler/chat, CLI, C1..C3, mobile e U1..U9/E2E;
6. somente então Go test/vet/build, suites frontend/mobile, diff-check e browser
   autenticado. Qualquer trigger residual no banco reprova o pacote.

## 5. Protocolo de locks

1. Cada pacote começa de um worktree isolado criado a partir do HEAD integrado
   pelo General-TL.
2. `files_locked` é exclusivo durante a vida do pacote. Reuso de um arquivo só
   ocorre em wave posterior, após merge, unlock e rebase.
3. Lane vermelha serial:
   `server/internal/service/task.go`,
   `server/internal/daemon/daemon.go`,
   `server/internal/daemon/client.go`,
   `server/internal/handler/daemon.go`,
   `server/pkg/db/generated/**` e números de migration.
4. Nenhum pacote paralelo gera/commita o mesmo arquivo sqlc. O pacote SQL em
   curso possui o lock completo de `pkg/db/generated`.
5. Testes pertencem ao pacote do código que cobrem; não existe “agente de
   testes” editando o mesmo arquivo em paralelo.
6. O General-TL integra uma wave, executa o gate global e só então libera locks
   da próxima.

## 6. Ordem de implementação e `files_locked`

### Wave 0 — decisões e prova, sem patch

| Gate | Dependência | Saída obrigatória |
|---|---|---|
| D0-ledger | Owner define se autopilot passa pelo replay gate e confirma a chave canônica task UUID/correlation. | Contrato assinado; ausência continua fail-closed. |
| D0-runtime | Owner escolhe tombstone/SET NULL/export para história e reassign/archive dos agentes. | Nenhum DELETE live antes da decisão. |
| D0-price | Owner aprova fonte/versionamento/moeda/effective date de preço. | Tabela Go única ou store escolhido; misses observáveis. |
| D0-telemetry | Nova task sintética autorizada por AGY/Kiro mede somente wire shape/correlação. | Evidência de onde tokens/receipt realmente existem. |
| D0-security | Owner autoriza cada rotação/chmod/drop-in/restart separadamente. | Lista exata de recursos e rollback; nenhum valor em contexto. |
| D0-network | Owner escolhe URL HTTPS e ACL/trust boundary. | Bypass local proibido na origem externa. |

### Wave 1 — quatro worktrees paralelos, locks disjuntos

#### P1 — ORQ-26 upload contract (GTL-01)

`files_locked`:

- `server/internal/handler/file.go`
- `server/internal/handler/file_test.go`
- `server/internal/cli/client_test.go`
- `server/cmd/multica/cmd_agent_test.go`
- `packages/core/hooks/use-file-upload.test.ts`
- `packages/core/api/client.test.ts`
- `apps/mobile/data/api.test.ts` (novo)
- `e2e/chat-attachments.spec.ts`

Gate de pacote:

1. S1..S7 com injeção de falha isolada e cleanup verificável;
2. C1..C3 e U1..U9 do GTL-01, incluindo CLI/mobile;
3. regressões de workspace/chat verdes;
4. `ApiContractError` permanece fail-closed;
5. Go test/vet/build e testes frontend/mobile direcionados;
6. diff-check, ausência de trigger residual e full suites pelo General-TL.

Aceite Kanban/UI:

- clipe, drag/drop, paste, feedback e avatar não lançam
  `ApiContractError`;
- contextless retorna `id=""`, `url==download_url`;
- falha DB limpa storage best-effort e retorna 500;
- browser autenticado confirma quick-create, comentários e chat.

#### P2 — ORQ-18 affordance + testes (GTL-12)

`files_locked`:

- `packages/views/runtimes/components/runtime-list.tsx`
- `packages/views/runtimes/components/runtime-row-menu.test.tsx`
- `packages/views/runtimes/components/delete-runtime-dialog.test.tsx`
- `packages/core/runtimes/mutations.test.tsx` (novo)

Gate: fechar todos os itens de §4. O código pode ser preparado, mas não
integrado em produção antes de P7-runtime-retention.

Aceite Kanban/UI: owner/admin vê botão sem hover; não autorizado não vê;
confirm/cancel/success/error/refetch têm prova. Nenhum runtime live é deletado
como teste.

#### P3 — squad assignee mobile (parte independente GTL-08)

`files_locked`:

- `apps/mobile/components/issue/pickers/assignee-picker-body.tsx`
- teste novo adjacente do picker

Gate: squad aparece, seleção emite `assignee_type=squad`, member/agent não
regredem.

Aceite Kanban: em card sintético, squad pode ser assignee no mobile e o web
renderiza o mesmo assignee. Sem auto-enqueue como efeito colateral.

#### P4 — ledger durável (GTL-02; lane vermelha)

`files_locked`:

- `server/migrations/127_task_ledger_summary.{up,down}.sql`
- `server/pkg/db/queries/task_ledger_summary.sql`
- `server/pkg/db/generated/**`
- `server/internal/daemon/commitledger/replay_gate.go`
- `server/internal/daemon/commitledger/*_test.go`
- `server/internal/daemon/daemon.go`
- `server/internal/daemon/client.go`
- `server/internal/handler/daemon.go`
- `server/internal/service/task.go`
- `server/internal/service/autopilot.go`
- testes direcionados de handler/service/autopilot

Gate:

1. interface `ReplayGateChecker`, não ponteiro para struct concreta;
2. persistência via HTTP backend, nunca DSN no daemon;
3. correlation canônica igual nos dois lados;
4. booleans monotônicos, schema version e TTL 24 h;
5. restart com registry vazio ainda bloqueia atividade ambígua;
6. missing/query error/version mismatch fail closed;
7. autopilot obedece a decisão D0-ledger;
8. migration/sqlc, Go full gates.

Aceite Kanban: tentativa automática ambígua não cria task duplicada; UI pode
mostrar “replay blocked”, mas não muda issue. Rerun manual continua ação
explícita do owner.

### Wave 2 — após merge/rebase da lane vermelha

#### P5 — account snapshot + usage schema (GTL-03/09/10)

`files_locked`:

- `server/migrations/128_task_account_usage_dimensions.{up,down}.sql`
- `server/pkg/db/queries/agent.sql`
- `server/pkg/db/queries/task_usage.sql`
- query nova de account/assignment
- `server/pkg/db/generated/**`
- `server/internal/service/task.go`
- tipos de claim/usage em handler/client
- `server/internal/daemon/credential_home.go`
- `server/internal/daemon/credential_home_test.go`
- `server/internal/daemon/daemon.go`

Gate:

1. claim atomiza assignment e snapshot na task;
2. valida tenant/provider/approved/scope/status/active lock;
3. payload contém somente UUID pseudônimo + generation;
4. daemon resolve ID localmente, valida root/allowlist e falha fechado;
5. rotation posterior não muda snapshot da task;
6. `task_usage` e rollups carregam account+thinking sem inventar histórico;
7. migração conserva todos os quatro totais de tokens;
8. duas claims concorrentes não trocam conta.

Aceite Kanban: card sintético mostra conta pseudônima atribuída, nunca
path/e-mail; revogação impede start e deixa erro explícito. Nenhuma issue
histórica é reexecutada.

#### P6 — project squad lead (parte independente GTL-08)

`files_locked`:

- `server/migrations/129_project_squad_lead.{up,down}.sql`
- `server/internal/handler/project.go`
- testes de project handler
- `packages/core/types/project.ts`
- testes/formulários web e mobile de project lead

Gate: create/update aceitam squad válido, rejeitam cross-workspace/ID
inexistente, member/agent permanecem compatíveis.

Aceite Kanban/projeto: criar e editar projeto com squad lead; cards do projeto
continuam renderizando e filtrando corretamente.

### Wave 3 — depois de D0-telemetry, D0-price e P5

#### P7 — token ingestion + pricing versionado (GTL-03/06/13)

`files_locked`:

- adapters comprovadamente envolvidos em `server/pkg/agent/`
- `server/internal/daemon/daemon.go`
- `server/internal/daemon/client.go`
- `server/internal/handler/daemon.go`
- `server/internal/metrics/pricing.go`
- `server/internal/metrics/business.go`
- testes correspondentes

Gate:

1. parser implementa somente shapes medidos;
2. AGY guarda model exato e tier vazio; Codex/Kiro guardam tier explícito;
3. receipt/correlation casa com task+account snapshot;
4. preço inclui effective date/version/moeda;
5. AGY obrigatório tem preço para todos os modelos aprovados;
6. miss incrementa métrica/erro, nunca custo zero silencioso;
7. Codex atual não regride;
8. unitários + integração com fixture, sem prompt real no gate de código.

Aceite Kanban/custo: três cards sintéticos novos, um por runtime obrigatório,
geram usage atribuído; AGY/Kiro não somem do relatório; preço ausente fica
visível como erro, não `$0`.

#### P8 — runtime history retention + UI final (GTL-05/08/12)

Só inicia depois de D0-runtime. `files_locked` dependem da escolha:

- `server/internal/handler/runtime.go`
- runtime queries/migration 130 se necessária
- `packages/views/runtimes/components/delete-runtime-dialog.tsx`
- arquivos já travados por P2, após unlock/rebase

Gate: contador de task/usage antes/depois idêntico; concorrência 409;
reassign preserva provenance; archive/delete explicita impacto irreversível;
nenhum cascade invisível.

Aceite Kanban/UI: diálogo oferece apenas ações semanticamente seguras, recarrega
plano em 409 e nunca promete “history preserved” sem prova de conservação.

### Wave 4 — relatórios e monitor

#### P9 — ORQ-15/20 relatório por conta

Dependências: P5 + P7. GTL-10 define o DAG, mas não enumera arquivos. Portanto
**nenhum lock é emitido ainda**: o General-TL deve primeiro registrar query,
handler e view exatos. Isso evita colisão oculta com `task_usage.sql`,
`business.go` e dashboard.

Gate/aceite: quatro slots AGY aprovados aparecem individualizados por
account_id pseudônimo; soma por conta = soma global; model/tier/preço/version
são visíveis; nenhum card histórico é rerodado.

#### P10 — monitor Kanban passivo (GTL-14)

Só após redraft C8. Locks previstos, ainda não liberados:

- query nova `kanban_integrity.sql`
- serviço novo isolado
- evento protocolado `kanban:integrity_anomaly`
- badge do card e testes

Gate: fixtures para os status reais, zero update/insert/delete, zero enqueue,
dedupe de evento e autorização por workspace.

Aceite Kanban: badge explica anomalia sem mover card nem criar task; fechar o
badge não altera o estado. Falso positivo em todo manual ou
`waiting_local_directory` reprova o pacote.

### Wave 5 — operações com gates próprios

#### O1 — segurança `/tmp`/umask (GTL-04)

Sem worktree. `host_resources_locked`: arquivos inventariados, secret handles,
TMPDIR privado e drop-ins systemd. Rotação antes de descarte; nenhum valor em
stdout/argv/evidência. Não aplicar chmod em massa nem restart sem autorização
específica. Build e Docker precisam continuar verdes depois.

#### O2 — LAN/auth (GTL-07)

Depois de P1 e browser QA local. Manter backend/Postgres loopback; publicar
somente frontend por HTTPS+ACL/proxy. Gate: anônimo `/api/me=401`, owner
autenticado 200, origem fora da ACL negada, CSRF/WS/uploads verdes e rollback
de um comando para loopback. Nunca usar bind cru `0.0.0.0`.

#### O3 — cutover/rollback final (GTL-11)

Depois de todos os pacotes aprovados. Backend ORQ1 primeiro por recreate com
`env_file`; daemon ORQ2 depois por staging atômico. Owner autoriza build,
recreate e restart separadamente.

Gate final:

- hashes/tags registrados;
- rollback green executável em um comando e ensaiado;
- backend/tunnel/frontend health 200;
- unit daemon ativa, zero restart loop;
- AGY/Codex/Kiro online, discovery 11/11/19 conforme contrato vigente;
- queue sem active órfão;
- nove issues históricas inalteradas;
- browser Kanban passa create/edit/drag/upload/runtime affordance;
- falha em qualquer health executa rollback ao green, nunca a backup
  pre-token-only.

## 7. Regra de promoção no Kanban

1. `todo → in_progress`: somente após owner autorizar o pacote e o General-TL
   registrar worktree + `files_locked`.
2. `in_progress → in_review`: diff integrado, gates de pacote verdes e
   evidência anexada.
3. `in_review → done`: aceite funcional correspondente acima, em card sintético
   quando execução for necessária, e assinatura do owner/General-TL.
4. `blocked`: usar para decisão externa concreta; não esconder falha de teste.
5. Status de task `completed` não promove issue automaticamente.
6. Monitor, replay gate e reconciler nunca tomam decisão de rerun ou aceite.

## 8. Ordem final resumida

```text
Decisões D0
  ├─ Wave 1 paralela: P1 upload | P2 ORQ18 tests | P3 mobile squad | P4 ledger
  └─ P4 merge/unlock
       ├─ P5 account snapshot/usage schema
       └─ P6 project squad lead (após lane SQL)
            ├─ P7 telemetry/pricing
            ├─ P8 runtime retention/UI
            └─ P9 reports → P10 passive monitor
                 ├─ O1 security (janela própria)
                 ├─ O2 LAN/auth (janela própria)
                 └─ O3 backend-first/daemon-second cutover
```

Esta ordem maximiza paralelismo onde os locks são disjuntos e serializa
deliberadamente os arquivos críticos. Qualquer artefato posterior ao corte
precisa passar por collision check antes de receber lock.
