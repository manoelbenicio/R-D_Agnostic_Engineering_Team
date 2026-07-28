# ORQ-12/13/15/16/17/18/21/22/23 — critérios para sair do limbo

**Autor:** Codex56#B (`w7:p4`)  
**UTC:** 2026-07-27T11:01:27Z  
**Modo:** somente leitura/análise. Nenhum rerun, edição de código, build, restart,
mudança de status, credencial ou arquivo operacional foi executado.

## Estado medido

Consulta direta ao Postgres do backend, sem valores de segredo:

| Issue | Issue/task | Erro literal | tool_use/result |
|---|---|---|---:|
| ORQ-12 | `todo` / `failed` | `runtime went offline` | 57 / 1 |
| ORQ-13 | `todo` / `failed` | `runtime went offline` | 83 / 5 |
| ORQ-15 | `todo` / `failed`, sem `started_at` | `unsupported credential path type ... slot-146 ... cli.log` | 0 / 0 |
| ORQ-16 | `todo` / `failed`, sem `started_at` | `unsupported credential path type ... slot-145 ... cli.log` | 0 / 0 |
| ORQ-17 | `todo` / `failed` | `runtime went offline` | 31 / 0 |
| ORQ-18 | `todo` / `failed` | `runtime went offline` | 45 / 1 |
| ORQ-21 | `todo` / `failed` | `runtime went offline` | 57 / 4 |
| ORQ-22 | `done` / task histórica `failed` | `The request was throttled by the service` | 21 / 0 |
| ORQ-23 | `todo` / `failed` | `runtime went offline` | 87 / 3 |

Regra aplicada: `tool_use` prova intenção, não conclusão. Efeito durável só é
afirmado abaixo quando também existe no filesystem, banco ou estado live.

## Critério de saída por issue

| Issue | Efeito preservado / evidência literal | O que falta exatamente | Sinal verificável para sair do limbo |
|---|---|---|---|
| **ORQ-12** | Worktree `agent/opus48-a/orq-12-task-usage-account-id` está limpo (`changes=0`). O schema atual, `server/migrations/032_task_usage.up.sql:1-11`, declara `task_id, provider, model, ...` e `UNIQUE (task_id, provider, model)`, sem conta. `server/pkg/db/queries/task_usage.sql:6-8` repete o mesmo contrato no upsert. | Implementar a Fase 3.1: identidade pseudônima estável da conta/slot em `task_usage`, chave de conflito e rollups; regenerar sqlc; definir backfill/NULL para histórico; provar que nenhum segredo/identidade pessoal é persistido. | Migration up/down + queries/gerado coerentes + testes + duas tasks em contas distintas produzindo duas linhas/agrupamentos de custo, sem valor de credencial. Só então mover para `in_review`; não há trabalho parcial a reaproveitar neste worktree. |
| **ORQ-13** | Worktree isolado `agent/codex-a/orq-13-reasoning-tier-cost` preserva **13 paths** alterados (11 modificados + migrations 127 up/down), incluindo 15 intenções `write_file`. O código live ainda diz em `packages/core/types/agent.ts:547-551`: `cost is computed client-side from a per-model pricing table`; o wire type em `:552-560` não carrega tier. OpenSpec `tasks.md:32-35` mantém 3.2 e 3.4 abertos. | Auditar/rebasear o patch parcial contra o freeze atual; não aceitar o sqlc editado manualmente sem regeneração; definir preço/versionamento por `(provider, model, thinking_level)` e preservar o tier efetivo do daemon ao relatório. | Gates de migration/sqlc/test/build e duas execuções comparáveis, mesmo provider/model e tiers diferentes, com `thinking_level` persistido e custo calculado distintamente. Reaproveitar o worktree; não rerodar cegamente a issue. |
| **ORQ-15** | A task antiga não iniciou: erro literal no `cli.log` symlink e `0/0` mensagens de ferramenta. O Gate 2 corrigiu essa classe e o daemon live expõe `MULTICA_CREDENTIAL_SLOT_ALLOWLIST_ANTIGRAVITY=141,145,146,150`; discovery AGY concluiu com 11 modelos e smoke ORQ-27 completou. `credential_home.go:141-152` usa rendezvous hash e `:242-255` persiste afinidade por `agentID|provider`. | Provar participação das **quatro** contas, não apenas que uma task funciona. Hoje a atribuição financeira impede aceite: `accounts`, `approved_accounts` e `assignments` têm 0 linhas e ORQ-12/21 seguem abertas. | Quatro identidades pseudônimas aprovadas, afinidade determinística documentada, amostra controlada cobrindo os quatro slots e relatório por conta. Depende de ORQ-12 + ORQ-21; a task antiga não precisa ser “continuada”. |
| **ORQ-16** | A task antiga também falhou antes do start pelo symlink (`slot-145`, `0/0`). O live allowlist AGY contém somente `141,145,146,150`; portanto os slots citados como mortos na issue (`139,140,142,143,149,152`) estão excluídos. `credential_home.go:59-85` exige allowlist explícita e `:226-233` falha fechado se qualquer slot listado for inválido. | Registrar na própria issue a decisão por slot: **excluir** 139/140/142/143/149/152; anexar o allowlist live e o discovery final. Não é necessário autenticar os seis nem rerodar a task falha para provar a exclusão. | Evidência durável mostra os seis fora do allowlist, os quatro elegíveis dentro e discovery AGY `completed=11`. Com confirmação do owner de que “excluir” é a decisão final, pode ir diretamente a `in_review`. |
| **ORQ-17** | Não há repo/worktree criado; os 31 tool uses foram diagnóstico. Estado live: `frontend_bind=127.0.0.1:13100`, root 200, e uma requisição anônima a `/api/me` retorna 200. Logo publicar esse serviço na LAN agora ampliaria uma superfície sem autenticação exigida. | Owner escolher e autorizar o limite de confiança (preferencialmente Tailscale/ACL ou autenticação real), URL/bind, healthcheck e rollback. Implementar sem reutilizar `AUTH bypass` como segurança. | De outro host autorizado: URL estável 200 e sessão autenticada; de origem não autorizada: acesso negado; portas/bind e rollback documentados. Até isso, manter loopback e não rerodar. |
| **ORQ-18** | Worktree `agent/codex-b/orq-18-runtime-delete-ui` preserva 2 arquivos modificados. No live, `runtime-list.tsx:479-500` mantém delete em `DropdownMenu`; o botão tem `opacity-0` em `:482-488`. O patch isolado já propõe botão visível em `:454-477`: `Expose it directly instead of hiding it behind a hover-only kebab` e `onClick={() => setDeleteOpen(true)}`. | Revisão independente do diff, rebase sobre a correção clean-base da ORQ-26, executar testes direcionados e browser QA de permissão, confirmação, sucesso/erro e refresh. | Teste unitário passa e browser autenticado vê o botão sem hover, abre confirmação, cancela sem efeito e atualiza a lista após delete autorizado. Reaproveitar os dois arquivos; não rerodar a issue nem integrar antes da ORQ-26. |
| **ORQ-21** | Banco live: `accounts=0`, `approved_accounts=0`, `assignments=0`. Worktree `agent/codex-b/orq-21` preserva 3 arquivos untracked: resolver, teste e seed. O resolver parcial diz em `resolver.go:36-37`: `never reads credential values`; junta as três tabelas em `:52-71`. O seed grava as três em `seed_approved_assignment.sql:97-166`. Porém o live ainda usa arquivo local versionado: `credential_home.go:17-20` define `multica-assignments.v1.json` e `:237-255` lê/persiste esse documento. | Owner aprovar o mapa pseudônimo registry-slot → tenant/account/agent; revisar o seed e o resolver parcial; integrar a ponte Postgres ao caminho live com fail-closed e sem valores de credencial; resolver concorrência/rotação e alimentar ORQ-12. | As três tabelas deixam de estar vazias com somente metadados aprovados; resolver live seleciona a conta aprovada correta; conta revogada/ausente falha fechado; restart preserva assignment; custo é atribuído ao account_id pseudônimo. |
| **ORQ-22** | **FECHADA após esta auditoria.** A evidência do escritor único `legacy-daemon-security-remediation-20260727.md:18-25` registra backups restringidos a `0600`, movidos sem deleção para quarentena `0700`, com binário `0700`; `:31-34` registra 74 arquivos normalizados e `zero arquivo com marcador sensivel` publicamente legível. `:38-45` fixa o legado permanentemente desativado, T2/ORQ2 como runtime vigente e ORQ-22 em `done`. Consulta posterior confirma issue `done`; a task antiga continua `failed` por throttle, preservada como histórico. | Nenhum trabalho operacional falta para encerrar a issue. Há apenas uma decisão futura, separada: owner autorizar ou negar descarte da quarentena após 72 h de retenção forense. Isso não reabre ORQ-22. | **Já atingido:** papel permanente documentado, segundo daemon ausente, quarentena durável e restrita, rollback vigente nomeado, status `done`, nenhum segredo exposto e nenhum arquivo apagado. |
| **ORQ-23** | Worktree `agent/codex-a/orq-23` preserva o patch/OpenSpec T2 e comentários; não é prova de Fase 3 concluída. O checklist literal em `openspec/.../tasks.md:31-35` mantém 3.1–3.4 abertos; `:42` mantém `4.5 GATE F4: rollback testado em 1 comando` aberto. | Fechar ORQ-12, ORQ-13 e ORQ-21; extrair uso real AGY/Kiro; validar custo por conta+tier; executar o rollback de um comando somente sob autorização operacional específica e voltar ao estado green. | Fase 3.1–3.4 e gate 4.5 marcados com evidência real, hashes e healthchecks antes/depois. O worktree é referência preservada, mas deve ser comparado ao freeze live antes de qualquer reaproveitamento. |

## Auditoria aprofundada do patch parcial ORQ-21

Os três arquivos são material útil, mas **não são integráveis como estão**:

1. **Sem caminho de execução:** o daemon live ORQ2 não possui `DATABASE_URL`,
   `PGHOST`, `PGUSER` ou outra configuração Postgres. O único processo que cria
   `pgxpool` é o backend. `resolver.go:19-22` exige `QueryRow`; não existe chamada
   a `credentialregistry.NewResolver` no código. Logo adicionar o pacote ao
   daemon não o conecta a nada.
2. **Upsert destrutivo de estado:** `seed_approved_assignment.sql:128-140`
   permite que um `account_id` credentialless preexistente tenha
   `vendor/tenant/home/config` sobrescritos e ainda força `status='available'`,
   zera `tokens_used`, cooldown, janela e erro. Isso pode reativar conta
   exaurida/revogada ou sequestrar um UUID de outro tenant.
3. **Revogação e reassignment sem guarda:** `:153-155` religa
   `allowed=true`; `:157-164` troca a conta do agente sem verificar task ativa,
   sem lock e sem gravar `rotation_events`. Reexecutar o “seed” não pode
   equivaler a nova aprovação do owner.
4. **Path não confiável:** `resolver.go:52-78` devolve `home_dir/config_dir` do
   banco e `:93-95` só verifica string não vazia. Ele aceita path inexistente,
   symlink, traversal ou provider root errado, perdendo as garantias live de
   `credential_home.go:88-138` (path físico, contido no root, artefato regular).
5. **Provider não canônico:** `resolver.go:87-89` compara strings cruas. O live
   canonicaliza `agy`/`antigravity` em `credential_home.go:29-39`; o patch pode
   rejeitar a mesma conta dependendo do nome persistido.
6. **Colisão de contas:** migration `123_rotation.up.sql:36-42` torna
   `agent_id` único, mas `account_id` tem apenas índice não-único. Vários agentes
   podem compartilhar uma conta simultaneamente sem lease owner ou limite; o
   patch não define a política.
7. **Teste pode passar sem testar:** `resolver_integration_test.go:14-17` faz
   `Skip` se `DATABASE_URL` não existir — exatamente o ambiente do daemon e de
   muitos gates. O fixture usa path fictício em `/tmp` (`:43-56`), então não
   detecta a lacuna de validação. Faltam casos de status, path, alias provider,
   cross-tenant, concorrência e revogação persistente.
8. **Identidade ainda arbitrária:** o seed recebe `account_id` e paths do
   operador; não implementa a ponte determinística do registry ORQ2, nem prova
   proveniência/versionamento do slot pseudônimo.

### Split seguro recomendado ao escritor único

1. Manter o resolver Postgres **no backend**, que já possui pool e tenant
   autenticado. No claim (`handler/daemon.go:1195-1259`), resolver a aprovação e
   acrescentar ao wire `credential_account_id` pseudônimo e um slot/home
   declarado; nunca credencial.
2. Espelhar os campos em `daemon/types.go:42-58`. Antes de preparar o ambiente,
   o daemon deve derivar/validar o provider root pelo mesmo root+allowlist+
   artefato de `credential_home.go:88-138`, rejeitando path remoto que não
   corresponda exatamente a um slot permitido.
3. Fazer import versionado e reconciliável de
   `multica-assignments.v1.json`; depois escolher uma única autoridade. Não
   manter seleção aleatória local e assignment Postgres concorrentes.
4. Trocar o seed por “insert-or-verify”: conflito só aceita valores idênticos;
   nunca reseta status/uso/cooldown, nunca religa aprovação revogada e toda
   troca de assignment exige guarda de task ativa + `rotation_events`.
5. Gerar `account_id` pseudônimo determinístico de `(provider, slot)` sob
   namespace versionado aprovado; registrar somente slot/provider/tenant, sem
   terminal_id, e-mail ou token.
6. Gates: testes unitários sem `Skip`, integração Postgres realmente executada,
   cross-tenant/path/symlink/provider-alias/revocation/concurrency, claim wire
   sem segredo, restart preservando assignment e duas tasks comprovando
   account_id distinto em ORQ-12.

## Ordem que evita colisão e retrabalho

1. **Pode sair primeiro, sem novo código:** ORQ-16, após registrar a decisão de
   exclusão e anexar allowlist/discovery já medidos.
2. **Recuperar trabalho isolado:** ORQ-18 somente depois da clean-base ORQ-26;
   ORQ-13 e ORQ-21 exigem revisão/rebase por escritor único.
3. **Dependência financeira:** ORQ-21 → ORQ-12 → ORQ-13 → ORQ-15 → ORQ-23.
4. **Decisão operacional restante:** ORQ-17 precisa de auth/rede autorizadas.
   ORQ-22 já foi fechada pelo lote de segurança; descarte após 72 h é decisão
   futura separada.

## Veredito

Nenhuma das nove deve ser rerodada automaticamente. ORQ-15/16 têm causa de
falha pre-start já corrigida, mas seus critérios podem ser provados com gates
controlados. ORQ-13/18/21 têm trabalho parcial recuperável em worktrees isolados.
ORQ-12/17 não deixaram alteração de código. ORQ-22 está encerrada com evidência
de quarentena e papel permanente. ORQ-23 é o agregador dos gates financeiros e
de rollback ainda abertos.

## Auditoria aprofundada do patch parcial ORQ-13

Worktree auditado, somente leitura:
`/home/ec2-user/multica_workspaces/20fce817-895d-447b-965a-49f5e279314a/00c87814/workdir/repo`.
O material preserva 11 arquivos modificados e as migrations 127 up/down. O
`git diff --check` passa, mas **o patch não satisfaz ainda a ORQ-13**.

### Bloqueios de aceite

1. **O relatório não usa nem mostra o tier.** O patch acrescenta
   `thinking_level` aos tipos em `packages/core/types/agent.ts:531-609`, mas
   não altera `packages/views/runtimes/utils.ts`. O tipo precificável em
   `utils.ts:451-454` escolhe somente `model` e tokens; `estimateCost` em
   `:456-465` chama `resolvePricing(model, provider)`; e o agrupamento visível
   por modelo em `:901-911` usa apenas `modelGroupingKey(model, provider)`.
   Não existe leitura de `thinking_level` nos componentes de usage. Portanto
   linhas SQL separadas por tier voltam a ser somadas sob o mesmo modelo e o
   owner não consegue ver/comparar o tier efetivo.
2. **A precificação operacional também continua sem tier.**
   `internal/service/task.go:195-200` chama
   `RecordLLMUsage(... provider, model, tokens ...)`; o patch não muda essa
   assinatura. `internal/metrics/business.go:253-275` resolve preço somente
   por `modelAlias`, e as labels de custo em `:278-285` são provider/model/
   token/runtime/source. Logo Prometheus continua colapsando todos os tiers.
3. **Não foi adicionado preço por tier.** A tabela client-side permanece
   `MODEL_PRICING` em `utils.ts:158-263`, chaveada por modelo; o resolvedor em
   `:294-306` não recebe tier. A tabela server-side em
   `internal/metrics/pricing.go:17-39` também é somente por modelo. Isto
   contradiz literalmente a tarefa OpenSpec 3.2, `tasks.md:31-35`:
   `Adicionar preco por tier de reasoning`.
4. **AGY obrigatório continua sem custo no relatório da UI.** A tabela
   client-side `utils.ts:158-263` não contém nenhum `gemini-*`; e o próprio
   resolvedor declara em `:286-288` que não há fallback por prefixo. Assim os
   IDs live `gemini-3.1-*`, `gemini-3.5-*` e `gemini-3.6-*` permanecem
   unmapped/custo zero, mesmo que o banco agora preserve a string do tier.
5. **Há duas autoridades de preço já divergentes.** Exemplo literal:
   `utils.ts:188` precifica `gpt-5.4` cache-write em `2.50`, enquanto
   `internal/metrics/pricing.go:19` usa `0.25`; DeepSeek flash é
   `0.14/0.28` no client (`utils.ts:214`) e `0.56/1.12` no server
   (`pricing.go:32`). Acrescentar tier sem escolher uma fonte versionada torna
   impossível afirmar qual custo está correto e recalcula história quando a
   constante mudar.
6. **O wire de ingestão aceita cardinalidade arbitrária.**
   `internal/handler/daemon.go:2074-2110` decodifica e grava
   `u.ThinkingLevel` sem trim, enum, limite de tamanho ou validação contra
   provider/model. Como a migration cria UNIQUE e rollup por essa string
   (`127...up.sql:3-27`), um daemon divergente pode criar buckets ilimitados.
   O handler deve aceitar somente o tier efetivo canônico aprovado para o
   modelo/execução; AGY deve manter tier no model ID e `thinking_level=''`.
7. **“Efetivo” ainda não é comprovado.** O daemon copia a seleção do agente em
   `daemon.go:3729-3732`, pode mantê-la após falha de catalog lookup em
   `:3747-3755`, e grava essa mesma string em usage em `:3871-3885`. Isso prova
   a intenção enviada ao adapter, não que o CLI/gateway aplicou o tier. Para
   contabilização financeira, usar recibo/correlação do gateway ou confirmação
   do adapter; nunca inventar multiplicador de tier.

### Gates e riscos de integração

1. **Zero teste novo:** nenhum dos 13 paths do diff é arquivo de teste. Faltam
   up/down migration, conservação de totais, rollup idempotente com dois tiers
   no mesmo task/model, canonicalização/rejeição no handler, compatibilidade de
   daemon antigo e apresentação/custo por tier.
2. **O typecheck tem regressões previsíveis:** `thinking_level` virou
   obrigatório em `packages/core/types/agent.ts:531-609`, mas fixtures
   explicitamente tipadas como `RuntimeUsage` em
   `packages/views/runtimes/utils.test.ts:701-711`, `:758-768` e `:900-910`
   não fornecem o campo. O writer deve atualizar/defaultar o contrato e rodar
   o typecheck, não apenas testes Go.
3. **Schemas Zod não declaram o campo:** `packages/core/api/schemas.ts:340-362`
   e `:391-432` preservam `thinking_level` apenas acidentalmente por
   `.loose()`. É preciso torná-lo contrato explícito com default `''` e testar
   compatibilidade com backend antigo, como já se faz para provider em
   `schemas.test.ts:282-294`.
4. **Rollback perde a dimensão e reescreve toda a tabela:** a down migration
   soma tiers, executa `DELETE FROM task_usage` e reinsere em
   `127...down.sql:9-33`; apaga hourly/dirty em `:35-36`; depois roda backfill
   `'-infinity'..'infinity'` em `:592`. A perda de tier é inerente ao rollback,
   mas exige backup/aceite explícito e ensaio de conservação; não é rollback
   “sem perda”.
5. **sqlc precisa ser reproduzível:** quatro arquivos em `pkg/db/generated`
   foram alterados junto das queries, porém não há evidência de
   `sqlc generate`. O gate deve regenerar em checkout limpo e exigir diff zero.
6. **Não criar duas migrations financeiras concorrentes.** ORQ-12 precisa
   adicionar `account_id` às mesmas tabelas/chaves/rollups. O writer deve
   combinar o desenho account+tier ou ordenar uma única evolução
   expand/contract; duas cópias independentes das funções de rollup geram
   conflito e drift.

### Critério seguro revisado para ORQ-13

O patch só pode sair de `todo` depois de: (a) fonte de preço única, versionada
ou recibo de custo do gateway; (b) tier canônico validado e vinculado à
execução/account pseudônimo; (c) relatórios UI **e** metrics exibindo/agregando
por account+provider+model+tier; (d) AGY/Codex/Kiro obrigatórios cobertos sem
duplicar tier embutido no ID AGY; (e) migration/sqlc/Go/typecheck/unit/
integração todos verdes; e (f) duas execuções comparáveis autorizadas pelo
owner, em tiers distintos, provando seleção efetiva, tokens e custo. Até lá,
preservar o worktree e não rerodar a task histórica.

## Auditoria aprofundada do patch parcial ORQ-18

Worktree auditado, somente leitura:
`/home/ec2-user/multica_workspaces/20fce817-895d-447b-965a-49f5e279314a/91e70c79/workdir/repo`,
branch `agent/codex-b/orq-18-runtime-delete-ui`. O diff preservado toca somente:

- `runtime-list.tsx`: 23 adições / 43 remoções;
- `runtime-row-menu.test.tsx`: 25 adições / 21 remoções.

O `git diff --check` passa. Não executei teste, build, deploy ou browser.

### Veredito

**Direção correta e recuperável, mas aceite ainda incompleto.** O patch pode ser
reaplicado depois da ORQ-26 sem conflito textual conhecido; antes de integrar,
o escritor único precisa fechar os gates de comportamento abaixo.

### O que está correto

1. **Ação realmente visível:** `runtime-list.tsx:442-481` substitui o kebab
   hover-only pelo botão `Trash2`, com `aria-label`, foco visível e
   `onClick={() => setDeleteOpen(true)}`. Não resta `opacity-0`.
2. **Confirmação continua obrigatória:** `:482-491` reutiliza o único
   `DeleteRuntimeDialog`; o patch não chama a API diretamente pelo ícone.
3. **Autorização não foi ampliada:** `:555-566` continua ocultando pending
   custom runtime e só define `canDelete` para owner/admin ou proprietário do
   runtime. O backend repete a guarda em
   `server/internal/handler/runtime.go:558-568` e `:738-746`.
4. **Lista já possui mecanismo de refresh:** o hook existente
   `packages/core/runtimes/mutations.ts:7-14` invalida `runtimeKeys.all(wsId)`
   em `onSettled`; o cascade invalida runtime, agentes e snapshot em `:27-42`.
   Isso é mecanismo plausível, não prova de aceite.
5. **Compatível com a clean-base:** os blobs dos dois arquivos em
   `6a2aba3550aaf6b0468a37bfdf2f00c7faaae084` são exatamente os mesmos do
   HEAD `0cb8aebb5aff79cb430b3740d22fadc53c0116fd`:
   `runtime-list.tsx=2bd554c680dd47ae835e6a2cef98e3bf9fb3b795` e
   `runtime-row-menu.test.tsx=0aefb031eebe1a78c81b5491e67cff8cbfce1080`.
   A raiz conhecida da ORQ-26 está em `packages/core/api/schema.ts` e nas
   respostas de upload, não nesses arquivos.

### Lacunas que bloqueiam o critério literal

O aceite da issue é: `botão visível com confirmação, estados de sucesso/erro e
atualização imediata da lista, coberto por teste de UI`.

1. **O teste novo cobre só metade:** em
   `runtime-row-menu.test.tsx:132-145` ele comprova botão visível e heading do
   diálogo após o clique. Não confirma a ação, não observa `onDeleted`, toast,
   fechamento nem remoção/refetch da linha.
2. **Erro não está coberto:** o caminho de produção existe em
   `delete-runtime-dialog.tsx:155-177` (`toast.error(message)` e diálogo
   permanece), mas nenhum teste injeta uma falha genérica do DELETE e confirma
   estado recuperável. Os testes existentes cobrem somente os 409 especiais
   `runtime_has_active_agents` e `runtime_delete_plan_changed`.
3. **Refresh imediato não está coberto:** não existe teste de
   `useDeleteRuntime` que espione `invalidateQueries(runtimeKeys.all(wsId))`,
   nem teste do wrapper verificando que a linha some após a resolução. A mera
   existência do `onSettled` não satisfaz o critério observado.
4. **Permissão do componente é injetada no teste:** `makeRow(..., canDelete)`
   em `runtime-row-menu.test.tsx:102-108` e o render direto em `:113-126`
   pulam a derivação real de `RuntimeList`. Assim o teste de “sem permissão”
   apenas repete a prop recebida. Faltam owner/admin, runtime owner,
   non-owner/member e pending-custom renderizados pela lista.
5. **Fonte de verdade duplicada:** o projeto documenta
   `packages/core/permissions/rules.ts:11-19` como regra pura única e já possui
   `canDeleteRuntime` em `:135-149`, mas `RuntimeList` recalcula a permissão em
   `runtime-list.tsx:555-566`. Hoje as expressões coincidem com o backend, mas
   podem divergir. O writer deve consumir a regra central ou travar equivalência
   com teste.
6. **Comentário ficou contraditório:** `runtime-row-menu.test.tsx:49-51` ainda
   diz que o diálogo “never renders ... open=false throughout”, enquanto o
   teste novo em `:140-144` o abre. Não bloqueia runtime, mas deve ser corrigido
   para não induzir a próxima manutenção ao erro.

### Gates exatos antes de integrar

1. Acrescentar testes de wrapper para: confirmação bem-sucedida + toast/close;
   erro genérico + diálogo preservado; cancel sem mutation; e refetch/remoção
   imediata da linha.
2. Testar a derivação real de permissão e pending custom, não apenas passar
   `canDelete` pronto.
3. Rodar no checkout limpo:
   `pnpm --filter @multica/views test -- runtime-row-menu.test.tsx delete-runtime-dialog.test.tsx`,
   depois
   `pnpm --filter @multica/views typecheck` e o build web. Esses comandos são
   recomendação ao escritor único; não foram executados nesta auditoria.
4. Reaplicar os dois arquivos sobre a clean-base/correção ORQ-26, gerar imagem
   com digest único e executar browser autenticado: botão visível sem hover,
   abrir/cancelar, erro preservando a linha, sucesso removendo-a e console sem
   `ApiContractError`. ORQ-18 não deve compartilhar o primeiro canário de
   correção da ORQ-26, embora o diff seja textualmente compatível.

## Correção crítica à proposta de patch da ORQ-26

Auditoria somente leitura do contrato de upload/download. O diagnóstico de
campo ausente (`download_url`) está correto, mas **não** se deve preencher os
dois fallbacks com `attachmentDownloadPath(id)`.

### Por que esse path retorna 404

1. `attachmentDownloadPath` gera
   `/api/attachments/<id>/download` (`server/internal/handler/file.go:136-138`).
2. Essa rota chama `loadAttachmentForDownload` em `file.go:619-623`.
3. O loader executa `GetAttachmentByIDOnly` em `:567-574` e responde
   `attachment not found`/404 em `:575-579` se não houver linha.
4. Justamente os dois fallbacks não criam linha:
   - DB falhou depois do upload: `file.go:447-461`;
   - branch sem workspace: `file.go:465-476`.

Logo `download_url: attachmentDownloadPath(id)` passaria no Zod, mas entregaria
um link quebrado. Seria falso verde de contrato.

### Há um segundo bug no branch sem workspace

O servidor retorna `id.String()` em `file.go:472-476`, embora não exista linha.
O contrato client-side diz literalmente que o branch sem row deve ter ID vazio:
`use-file-upload.ts:74-80` só cai em `att.url` quando `att.id` é vazio, e o
teste em `use-file-upload.test.ts:75-83` fixa esse comportamento. Com ID não
vazio, o cliente fabrica `/api/attachments/<id>/download`, que inevitavelmente
retorna 404.

Para chat é ainda pior: se `X-Workspace-Slug` estiver ausente ou inválido,
`ResolveWorkspaceIDFromRequest` colapsa para vazio
(`middleware/workspace.go:63-67,81-94`). Como `issue_id`, `comment_id` e
`chat_session_id` só são validados **dentro** do branch com workspace
(`file.go:382-437`), o branch sem workspace ignora o vínculo e envia um upload
órfão como se fosse avatar. `chat-input.tsx:208-218,257-264` pressupõe um ID de
attachment real para vincular a mensagem; não existe degradação segura para
chat com ID vazio/falso.

### Patch semântico recomendado ao escritor único

1. **Falha de `CreateAttachment` em upload com workspace:** tratar a gravação
   como atômica. Após `Storage.Upload` bem-sucedido e insert falho, executar
   cleanup best-effort `Storage.Delete(ctx, key)` e responder 500. Não devolver
   200 com objeto órfão. A interface já oferece `Delete` em
   `internal/storage/storage.go:9-13`.
2. **Referência de entidade sem workspace:** se qualquer `issue_id`,
   `comment_id` ou `chat_session_id` vier preenchido, nunca cair no branch de
   avatar. Ou:
   - resolver o workspace pela própria entidade e reaplicar membership/
     ownership/private-agent gate; `GetChatSession` já existe em
     `pkg/db/queries/chat.sql:6-8`; ou
   - falhar 4xx antes do upload, obrigando o cliente a enviar contexto válido.
   Para o owner começar hoje, a primeira opção evita depender de estado global
   de URL, mas exige testes de cross-workspace/IDOR.
3. **Upload realmente sem workspace e sem entidade (avatar/logo):** manter o
   contrato URL-only com `id: ""`, `url: link`, `download_url: link`,
   `markdown_url: ""` e `filename`. Isso satisfaz o schema e faz
   `pickMarkdownLink` usar o storage URL, como os comentários/testes já exigem.
4. **Não reverter globalmente o fail-closed:** `schema.test.ts:393-404` e vários
   contratos do client testam deliberadamente `ApiContractError`. Corrigir os
   produtores e fixtures inválidos, não voltar a mascarar respostas
   malformadas com objeto vazio.
5. **Corrigir o teste que gerou um dos 86/88:** o propósito de
   `packages/core/api/client.test.ts:728-751` é verificar o FormData de chat,
   mas o mock retorna apenas `{id,url}` em `:730-734`. Ele deve devolver também
   `download_url` e `filename`; um teste separado deve continuar provando que
   a ausência real de `download_url` falha fechado.

### Gates direcionados da ORQ-26

1. Handler: DB insert falho retorna 500, chama cleanup com a key correta e não
   deixa resposta 200/orphan.
2. Handler: no-workspace/sem entidade retorna ID vazio e URL/download URL
   utilizáveis; `pickMarkdownLink` escolhe `att.url`.
3. Handler: chat/issue/comment sem header de workspace é resolvido com gates
   completos ou rejeitado antes de `Storage.Upload`; cross-workspace retorna
   404/403 sem criar objeto.
4. Handler: `/api/attachments/<id>/download` de ID inexistente permanece 404,
   travando a regressão “string válida porém link morto”.
5. Client: fixture FormData válida, schema fail-closed separado, upload de chat
   feliz preserva `chat_session_id`, ID real e attachment binding.
6. Só então core api/chat, build web e browser autenticado com upload real,
   preview, envio e reabertura da mensagem. Nenhum desses gates foi executado
   nesta auditoria; nenhuma mutação live foi feita.

## Auditoria aprofundada do worktree ORQ-23 e do T2 live

Worktree preservado:
`/home/ec2-user/multica_workspaces/20fce817-895d-447b-965a-49f5e279314a/ff121b28/workdir/repo`,
branch `agent/codex-a/orq-23`, base `0cb8aebb5aff79cb430b3740d22fadc53c0116fd`.
Ele contém seis arquivos modificados e quatro grupos untracked
(`credential_home.go`, teste, duas units e OpenSpec). `git diff --check` passa.
Esta auditoria não executou build, teste, task, restart nem rollback.

### Veredito de integração: não reaplicar o worktree

Grande parte do worktree já foi promovida e é byte-idêntica ao freeze atual:
`daemon.go`, `pkg/agent/{antigravity.go,models.go,models_test.go}`,
`credential_home.go`, seu teste e as duas units. Porém os arquivos restantes
são **anteriores** ao hardening do Gate 2:

1. `execenv/antigravity_home.go:39-49` do worktree ainda copia o diretório
   `.gemini/antigravity-cli` inteiro por `syncCredentialDir`. Isso reintroduz
   literalmente o defeito do `cli.log` symlink que tornava AGY task-incapaz.
2. `execenv/kiro_home.go:63-69,89-101` do worktree ainda faz `os.Stat` seguido
   de `os.Open`, herda `srcInfo.Mode().Perm()` no destino e pode deixar parcial
   no erro. O freeze atual usa `Lstat`, `O_NOFOLLOW`+`fstat`/`SameFile`, cria
   `0600` desde o primeiro byte e remove parcial.
3. O `tasks.md` do worktree nem contém o item atual 1.10 token-only. Portanto
   merge/rebase mecânico da branch pode desfazer o conserto P0. Ela deve ser
   tratada como histórico/proveniência; somente diffs novos contra o freeze
   atual podem ser considerados.

### O que o live prova hoje

Medição read-only em 2026-07-27, sem listar agent IDs ou qualquer credencial:

- daemon `multica-daemon-orq2-credential.service`: `active/running`,
  PID `3417665`, `NRestarts=0`; túnel também `active/running`, PID `3211411`,
  `NRestarts=0`; ambos `enabled`, e `Linger=yes`;
- binário live
  `/home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1`:
  SHA-256 `88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8`,
  modo `0755`, tamanho `15053065`;
- `multica-assignments.v1.json`: versão 1, modo `0600`, oito afinidades:
  AGY tem duas, distribuídas nos slots pseudônimos 145 e 146; Codex tem duas,
  ambas no 152; Kiro tem quatro, distribuídas entre 139, 140 e 149.

Isso comprova que a seleção persistiu múltiplos slots, mas **não** comprova o
Gate 2.3: o documento não registra task, tokens, custo ou resultado. Não há
evidência correlacionando duas execuções concluídas a duas contas distintas.

### Bloqueios ainda abertos, com âncora literal

1. **Fase 3 não existe no wire financeiro.**
   `internal/daemon/types.go:161-169` define `TaskUsageEntry` apenas com
   `provider`, `model` e quatro contadores; `internal/handler/daemon.go:2053-2062`
   repete o payload sem `account_id`, slot ou tier; e
   `pkg/db/queries/task_usage.sql:6-14` ainda faz conflito somente em
   `(task_id, provider, model)`. Assim uma execução não pode ser atribuída à
   conta escolhida nem separada por reasoning tier.
2. **Uso real está apenas parcialmente disponível.**
   Kiro acumula o usage ACP em `pkg/agent/kiro.go:337-378`, mas cai no modelo
   literal `"unknown"` quando a execução não traz `opts.Model`; falta evidência
   live de que task, modelo e contadores chegam ao banco. AGY declara
   explicitamente `Usage: map[string]TokenUsage{}` em
   `pkg/agent/antigravity.go:156-165`, pois o CLI não expõe tokens. Logo os itens
   3.1–3.4 de `tasks.md:32-35` continuam abertos de fato, não só por checklist.
3. **Discovery multi-conta ainda é uma amostra, não uma garantia.**
   `daemon.go:2075-2079` itera homes e interrompe no primeiro `err == nil`.
   O request final com 11 modelos prova o catálogo de um HOME bem-sucedido,
   não a paridade dos quatro slots AGY 141/145/146/150. Como o rendezvous em
   `credential_home.go:141-152` pode escolher qualquer elegível, catálogos
   divergentes podem anunciar um modelo que a conta sorteada não possui, ou
   ocultar um que outra possui. Antes do F2.3, cada home precisa ter catálogo
   medido e igual; se divergir, usar interseção segura ou catálogo por agente,
   nunca união cega.
4. **O checklist de teste superestima a cobertura.**
   `tasks.md:26` marca “traversal/symlink” como concluído, mas
   `credential_home_test.go:29-141` cobre afinidade, credencial Kiro ausente,
   symlink no slot, slot persistido inelegível, provider fora de escopo e
   allowlist de discovery. Não há caso de traversal, symlink na raiz do
   provider/artefato, JSON com segundo valor, concorrência ou crash durante
   persistência. Prepare/Reuse também não possui teste direto de falha do
   wrapper Kiro; produção bloqueia ambiente vazio apenas mais tarde em
   `daemon.go:3680-3686`.
5. **Reboot não foi provado.**
   Enabled+linger e o estado atual são positivos, mas a spec REQ-08 exige
   restauração após reboot. A unit do daemon usa apenas `After/Requires` sobre
   o túnel; a unit SSH é `Type=simple`. Isso ordena o `exec`, não comprova que
   a porta encaminhada esteja pronta antes do daemon. Falta um boot/recovery
   autorizado ou readiness explícita, seguido de discovery e task smoke.
6. **Gate 4.5 ainda não tem rollback T2 executável em um comando.**
   No ORQ2 há backups duráveis, mas nenhum comando/script de rollback do daemon.
   Os scripts `multica-backend-recreate` e
   `multica-backend-env-rollback` existem no ORQ1 e pertencem somente à
   ORQ-30/backend; não restauram o T2. Além disso os binários
   `.pre-token-only`, `.pre-agy-fix` e `.previous` antecedem o fix token-only;
   usá-los como alvo de aceite reintroduziria a incapacidade AGY conhecida.
   O alvo de rollback de futuras promoções deve ser o green atual `88ca4f...`,
   não um artefato conhecido como funcionalmente degradado.

### Critério de saída revisado da ORQ-23

1. Não integrar a branch ORQ-23 antiga. Implementar Fase 3 sobre o freeze atual
   e em uma única evolução coordenada com ORQ-21 → ORQ-12 → ORQ-13.
2. Propagar um `account_id` pseudônimo/versionado da seleção até o recibo de
   usage, sem path, terminal ID, e-mail ou segredo; persistir account+provider+
   model+tier e escolher uma fonte versionada de preço/recibo.
3. Fechar AGY usage com dado real do provider/gateway; validar Kiro live sem
   bucket `"unknown"`. Duas tasks autorizadas, já mapeáveis a slots AGY
   distintos 145/146, devem provar isolamento e custo distinto sem expor
   identidade.
4. Provar paridade de catálogo em todos os slots elegíveis e completar os testes
   fail-closed ausentes; nenhum `agy models` adicional foi disparado nesta
   auditoria porque o CLI pode escrever estado/log.
5. Criar um rollback T2 de um comando que restaure o hash green atual, ensaiá-lo
   somente com autorização escrita e validar, antes/depois: unit+túnel,
   health, discovery 11/19/11 e task smoke AGY/Codex/Kiro. Um teste de reboot ou
   readiness equivalente deve fechar REQ-08.
6. Só então marcar 2.3, 3.1–3.4 e 4.5 e mover ORQ-23 a `in_review`. Hoje o
   veredito permanece **PRESERVAR / NÃO RERODAR**.

## Auditoria e contrato seguro para a ORQ-12 (`account_id` por execução)

O worktree `agent/opus48-a/orq-12-task-usage-account-id` continua limpo; não há
patch parcial para reaproveitar. A análise abaixo é somente leitura e corrige
um risco do desenho preliminar que sugeria `GetTaskAccountID` por join da
assignment atual.

### Achado bloqueante: assignment atual não é evidência histórica

`migrations/123_rotation.up.sql:36-42` define `assignments` como uma única linha
mutável por `agent_id`. `rotation_events` registra trocas, mas o pipeline de uso
não faz resolução temporal. Logo consultar `assignments` quando o usage chega
atribui a task à conta **atual**, não necessariamente à usada quando ela
executou. Uma rotação entre `StartTask` e `ReportTaskUsage` corromperia custo
histórico silenciosamente.

A conta deve virar snapshot imutável da execução:

1. O backend resolve/valida a conta aprovada no claim e grava
   `agent_task_queue.credential_account_id` na mesma transação que muda
   `queued → dispatched`.
2. O claim envia ao daemon somente esse UUID pseudônimo/versionado. Não envia
   HOME, path, e-mail, terminal ID nem segredo.
3. O daemon mapeia o UUID para um slot local allowlisted, valida root e artefato
   por `credential_home.go:88-138` e falha fechado se não houver correspondência.
4. `ReportTaskUsage` toma o account do snapshot da task no servidor. Se o wire
   também trouxer account para correlação, ele deve ser apenas conferido por
   igualdade; nunca aceito como autoridade.
5. Reassignment/rotação depois de `dispatched` não altera a task existente.
   Retry filho é nova execução e recebe seu próprio snapshot.

Isso encaixa no ponto atômico real: hoje `ClaimAgentTask` faz o único
`UPDATE ... SET status='dispatched' ... FOR UPDATE SKIP LOCKED` em
`pkg/db/queries/agent.sql:268-304`. O handler constrói a resposta logo depois
em `internal/handler/daemon.go:1194-1259`. Resolver depois da resposta ou apenas
no usage deixa uma janela de divergência.

### Migration única com a ORQ-13

Não criar migrations independentes 127 para account e tier. A evolução deve
ser uma só, expand-first:

1. `agent_task_queue`:
   - `credential_account_id UUID NULL REFERENCES accounts(account_id)
     ON DELETE RESTRICT`;
   - `credential_assignment_version` ou generation imutável;
   - índice por account para auditoria.
   `NULL` representa somente história pre-cutover/providers fora do escopo.
2. `task_usage`:
   - `account_id UUID NULL REFERENCES accounts(account_id) ON DELETE RESTRICT`;
   - `thinking_level TEXT NOT NULL DEFAULT ''`;
   - substituir a chave de `migrations/032_task_usage.up.sql:11` por
     `UNIQUE NULLS NOT DISTINCT
     (task_id, account_id, provider, model, thinking_level)`.
   `ON DELETE SET NULL` não serve: apagaria a atribuição financeira. A conta
   pseudônima deve ser desativada/revogada, não deletada enquanto houver usage.
3. `task_usage_hourly` e `task_usage_hourly_dirty`:
   acrescentar `account_id` e `thinking_level` às duas constraints. Hoje
   `101_task_usage_hourly_schema.up.sql:37-55,114-126` agrega somente
   provider/model. Todas as seleções, joins, GROUP BY, upserts e
   `deleted_empty` de `102_task_usage_hourly_pipeline.up.sql:233-398` precisam
   carregar as novas dimensões.
4. Queries de dashboard/runtime devem devolver e agrupar account+tier; caso
   contrário a granularidade crua volta a ser colapsada na leitura.
5. Histórico permanece `account_id=NULL`, `thinking_level=''`; não inventar
   conta retroativa. O backfill reconstrói hourly preservando exatamente os
   quatro totais de tokens antes/depois.

### Guarda de aprovação no claim

O claim não pode confiar apenas em `assignments.account_id`. A seleção precisa
validar atomicamente:

- `accounts.tenant_id` igual ao workspace da task;
- vendor canônico igual ao provider do runtime (`agy` = `antigravity`);
- `approved_accounts.allowed=true` e `worktype_scope` compatível;
- status disponível, ou leased explicitamente para o mesmo agente;
- assignment estável e nenhuma troca enquanto houver task
  `dispatched/running/waiting_local_directory`.

Ausente, revogada, cross-tenant, provider divergente, cooldown/exhausted ou
ambígua deve produzir erro explícito e nenhum start. Hoje as tabelas
`accounts`, `approved_accounts` e `assignments` não têm query de produção; só
existem nas migrations 123/124. Portanto adicionar coluna sem integrar esse
claim seria uma aparência de atribuição, não a ponte ORQ-21.

### Identidade pseudônima e autoridade única

O backend deve ser a autoridade de assignment. Para o daemon localizar o slot
sem receber path:

- importar um mapa versionado `account_id → provider/slot_ref` aprovado pelo
  owner; ou
- derivar UUID determinístico sob namespace da instalação a partir de
  `(provider canônico, slot_ref)`.

Nunca derivar de e-mail/identidade da conta. Backend escolhe `account_id`; o
daemon compara esse ID contra os slots físicos da allowlist e deriva o HOME
local. O arquivo `multica-assignments.v1.json` deve ser importado/reconciliado e
depois deixar de competir como segunda autoridade; o live já contém oito
afinidades que precisam ser preservadas.

### Rollout e rollback sem perda

1. Com fila ativa em zero, aplicar schema expandido e o import
   insert-or-verify da ORQ-21.
2. Promover backend primeiro: claim grava snapshot e inclui account ID.
   Daemon antigo pode ignorar o campo somente enquanto nenhuma task for
   liberada nessa janela.
3. Promover daemon: resolve o ID recebido para slot local e ecoa correlação.
4. Habilitar fail-closed para os três providers obrigatórios: task nova com
   account nulo não executa.
5. Rollback de aplicação mantém as colunas novas. Não executar um down que
   apaga/colapsa account+tier; isso destrói evidência financeira. Contract/drop
   é fase posterior com retenção e decisão do owner.

### Gates verificáveis da ORQ-12

1. Migration sobre fixture com história: raw/hourly antes e depois conservam
   input/output/cache-read/cache-write; linhas antigas ficam em um único bucket
   NULL, sem duplicata.
2. Claim: approved correto grava snapshot; missing/revoked/cross-tenant/
   provider-mismatch falham fechado; duas claims concorrentes não trocam conta.
3. Imutabilidade: rotacionar assignment depois do start não muda
   `agent_task_queue.credential_account_id` nem o usage daquela task.
4. Handler: account enviado pelo daemon divergente é rejeitado; o servidor
   persiste somente o snapshot da task.
5. Rollup: duas tasks no mesmo provider/model/tier e contas distintas geram
   duas linhas raw, duas hourly e dois agrupamentos de relatório; a soma global
   permanece idêntica.
6. Compatibilidade: old history continua legível; novo AGY/Codex/Kiro sem
   account falha fechado após o cutover; sqlc regenerado produz diff zero.
7. Somente depois, duas tasks novas e autorizadas em contas AGY distintas
   fecham o aceite real. Nenhuma das tasks históricas ORQ-12/13/15/16/17/18/
   21/22/23 deve ser rerodada para esse gate.

## ORQ-17 — fechamento da auditoria de bind, proxy e autenticação

**Veredito:** permanece `PRESERVADA / OWNER`. Não há caminho seguro para
simplesmente trocar o bind do frontend de loopback para `0.0.0.0`. O estado
medido e consolidado em `RCA-HANDOVER-20260727.md:394-398` ainda é:
`/api/me` responde 200 anônimo, portanto publicar 13100 agora ampliaria uma
superfície sem autenticação. Esta auditoria foi somente estática/read-only; não
alterou bind, proxy, ACL, cookie, credencial, serviço ou board.

### Evidência literal

1. O bypass local é explicitamente global para as APIs do browser:
   `server/internal/middleware/auth.go:107-110` diz
   `"every browser API request runs as the configured operator account"` e
   entra nesse ramo antes de extrair qualquer token.
2. A proteção fail-closed já existe, mas depende do valor efetivo de
   `FRONTEND_ORIGIN`: `auth.go:43-52` retorna vazio se a origem não for
   `localhost`/loopback. O teste `auth_test.go:80-92` exige literalmente
   `"public origin must disable local bypass"`.
3. A própria configuração self-host proíbe exposição crua:
   `docker-compose.selfhost.yml:3-8` manda usar reverse proxy com TLS e diz
   `"Do NOT change these bindings to 0.0.0.0"`; frontend e backend continuam
   declarados em loopback nas linhas `50-52` e `128-130`.
4. Só o frontend precisa ser publicado. O bundle faz proxy same-origin:
   `apps/web/next.config.ts:50-66` reescreve `/api`, `/ws`, `/auth` e
   `/uploads` para `REMOTE_API_URL`; o backend pode e deve permanecer
   loopback.
5. HTTPS precisa preceder o novo login. `server/internal/auth/cookie.go:
   114-127` deriva `Secure` do scheme de `FRONTEND_ORIGIN`; as linhas
   `148-181` fixam cookies de auth/CSRF como `SameSite=Strict` e usam esse
   `Secure`. Uma URL HTTP externa desativa `Secure`.
6. Autenticação real está implementada: `cmd/server/router.go:164-173`
   inicializa o store/provider de senha; `router.go:487` publica
   `POST /auth/login`; `internal/handler/auth_provider.go:76-103` lê e grava
   somente hash bcrypt em `user_password_credential`. Isso prova capacidade de
   código, não prova que o owner live já tenha credencial de senha provisionada.

### O que falta exatamente para sair do limbo

1. **Decisão e autorização do owner:** escolher a URL HTTPS estável e a
   fronteira de confiança. Caminho de menor superfície: manter 13100/18080 em
   `127.0.0.1`, publicar somente o frontend por reverse proxy/Tailscale Serve,
   e restringir o ingresso ao tailnet/ACL do owner.
2. **Gate de autenticação antes da exposição:** confirmar, sem revelar hash ou
   senha, que o owner tem `user_password_credential`; testar login real e
   sessão numa URL HTTPS de staging. A configuração externa deve usar
   `FRONTEND_ORIGIN=https://<url>` e bypass efetivamente inativo. O teste
   negativo obrigatório é GET `/api/me` sem cookie/token retornar 401.
3. **Gate de proxy:** `/`, `/login`, `/api/me`, `/ws`, `/auth/*` e
   `/uploads/*` funcionam pela mesma origem; backend e Postgres continuam sem
   listener LAN; TLS, WebSocket upgrade, limite de upload e timeouts são
   validados.
4. **Gate de acesso:** host autorizado obtém 200 após login; origem fora da ACL
   não conecta; logout invalida sessão; request mutante sem CSRF retorna 403.
5. **Rollback antes do cutover:** um comando desativa somente o proxy/publicação
   e restaura a URL loopback, sem recriar backend, gerar JWT ou mexer em
   credenciais. Health local 13100/18080 deve permanecer disponível durante e
   após o rollback.
6. **Aceite de QA:** browser autorizado valida Kanban e console na URL final;
   isso é separado da autorização pendente de browser/Chromium da ORQ-26.
7. **Nenhum rerun histórico:** esses gates usam probes autenticados e, quando o
   owner autorizar, um card novo de evidência; ORQ-17 e as outras oito issues
   preservadas não são rerodadas.

### Ressalva obrigatória de encerramento

Os backups pre-token-only (`.pre-token-only`, `.pre-agy-fix`, `.previous`) são
**known-bad**: restaurá-los reintroduz AGY task-incapaz e precede o hardening
0600/O_NOFOLLOW. Eles não são rollback aceitável. O baseline green para uma
futura rede de segurança é o artefato live hash `88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8`;
nenhum backup, serviço ou artefato foi tocado nesta análise.
