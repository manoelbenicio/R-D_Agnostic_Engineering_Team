# IDEMPOTENCIA DAS 6 AMBIGUAS - ORQ-12, 13, 17, 18, 21, 23

- revisor: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-27T11:02Z
- fonte: API do backend no ORQ1, leitura pura:
  `curl -s http://127.0.0.1:18080/api/issues?workspace_slug=orq2-dev` (workspace `20fce817-895d-447b-965a-49f5e279314a`, prefixo ORQ)
- modo: leitura. Nenhuma issue reexecutada, nenhum codigo ou arquivo do repo editado, nada compilado ou reiniciado.
- ACHADO DE FORMA: nas seis, o campo `acceptance_criteria` da API vem `null`. O criterio de aceite
  esta embutido no corpo de `description`, na linha que comeca por `Critério de aceite:`. Foi essa
  linha que usei para classificar, e nao o titulo.

## Veredito

| Issue | Veredito | Motivo em uma linha |
|---|---|---|
| ORQ-12 | MUTANTE | Criterio exige "persistir identidade pseudônima/conta por execução", ou seja DDL em `task_usage` mais escrita de linhas. |
| ORQ-13 | MUTANTE | Criterio exige "usando execuções comparáveis", ou seja disparar tasks reais que consomem cota e gravam `task_usage`. |
| ORQ-17 | MUTANTE | Criterio exige "URL estável ... com autenticação, exposição de rede e rollback validados", ou seja mudanca de rede e de servico. |
| ORQ-18 | MUTANTE | Criterio exige "botão visível ... coberto por teste de UI", ou seja escrita de codigo no frontend. |
| ORQ-21 | MUTANTE | Criterio exige "contas autorizadas são semeadas", ou seja INSERT em `accounts`, `approved_accounts` e `assignments`. |
| ORQ-23 | MUTANTE | Criterio exige "o rollback do T2 é executado/validado por um comando", e executar rollback e mutacao. |

**Nenhuma das seis e IDEMPOTENTE pelo criterio de aceite. Zero liberaveis para rerun de risco zero.**

## Justificativa por issue, com trecho literal

### ORQ-12 - Contabilizar custo por conta em task_usage - MUTANTE
Literal da issue: *"Critério de aceite: persistir identidade pseudônima/conta por execução e
demonstrar, com dados reais, relatório de custo separado por conta sem expor credenciais."*
`persistir` e `com dados reais` sao mutacao por definicao. Confirmei a premissa no repo:
`migrations/032_task_usage.up.sql:2-10` cria a tabela com `id, task_id, provider, model,
input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, created_at` e **sem
`account_id`**; e `pkg/db/queries/runtime_usage.sql:40` diz literalmente
`-- the runtime-detail "Cost by agent" tab. task_usage only carries task_id,`.
`UpsertTaskUsage` em `pkg/db/queries/task_usage.sql` insere em
`(task_id, provider, model, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, updated_at)`
com `ON CONFLICT (task_id, provider, model)`. Logo: precisa de migration nova, mudanca da chave de
conflito e reescrita do upsert. Rerun cego escreveria schema.

### ORQ-13 - Calcular preço por tier de reasoning - MUTANTE
Literal: *"Critério de aceite: relatório demonstra custo distinto por tier usando execuções
comparáveis e preserva o thinking_level efetivamente escolhido."*
"execuções comparáveis" significa rodar tasks pagas em pelo menos dois tiers: consome cota do
OmniRoute, grava `task_usage` e produz custo real. E mutacao com efeito financeiro.
CORRECAO DE ANCORA, para o escritor unico nao perder tempo: a issue cita `daemon.go:2101`, mas
nessa linha hoje esta apenas o fecha-chaves de `type thinkingLevelWire struct` dentro de
`handleModelList` - a referencia derivou desde que a issue foi escrita. A ancora real de preco e
`internal/metrics/pricing.go` (mais `internal/handler/cloud_billing.go` e
`internal/metrics/business.go`), e `thinking_level` e persistido em `agent`, nao em `task_usage`
(`pkg/db/queries/agent.sql:23,43,50-54`). Ou seja o dado por tier nao existe na tabela de custo.

### ORQ-17 - Publicar frontend 13100 com acesso LAN estável - MUTANTE
Literal: *"Critério de aceite: URL estável acessível pelo owner sem túnel SSH, com autenticação,
exposição de rede e rollback explicitamente validados."*
Exige alterar bind/publicacao do servico e a rede. E a issue de MAIOR blast radius das seis:
expor porta hoje em `127.0.0.1:13100` para a LAN e mudanca de superficie de ataque; o proprio
criterio manda validar autenticacao junto, o que confirma que sem isso viraria endpoint sem
autenticacao. Nao e rerun, e mudanca de infraestrutura - STOP-AND-WAIT.

### ORQ-18 - Adicionar botão de exclusão de runtime na UI - MUTANTE
Literal: *"Critério de aceite: botão visível com confirmação, estados de sucesso/erro e
atualização imediata da lista, coberto por teste de UI."*
Escreve componente e teste no frontend. Agrava: colide com ORQ-26, que esta ABERTA justamente por
bundle live sujo (`cf8017e3`, `ApiContractError`/`throw`, `schema.ts:56`). Mexer no frontend antes
do rebuild limpo da ORQ-26 mistura duas causas de falha.

### ORQ-21 - Popular contas aprovadas e assignments no Multica - MUTANTE
Literal: *"Critério de aceite: contas autorizadas são semeadas sem valor de credencial,
assignments são reproduzíveis e o resolver usa os dados persistidos em teste real."*
"são semeadas" e "dados persistidos" = INSERT em `accounts`, `approved_accounts` e `assignments`,
hoje com zero linhas. E a ponte que o FATO 5 descreve. Mutacao de banco por definicao.

### ORQ-23 - Concluir contabilização da Fase 3 e gate de rollback 4.5 - MUTANTE
Literal: *"Critério de aceite: Fase 3 de custo fecha com evidência real e o rollback do T2 é
executado/validado por um comando, sem regressão dos três runtimes obrigatórios."*
Duas metades: a contabilizacao da Fase 3 seria idempotente, mas o gate 4.5 exige **executar** o
rollback do T2 e depois revalidar AGY, Codex e Kiro. Executar rollback e mutacao, e uma que toca
justamente os tres obrigatorios que acabaram de passar nos Gates 1 e 2. Das seis, e a de pior
risco de reverter ganho recem-conquistado.

## O que o owner PODE liberar, se quiser risco zero

Nenhuma das seis inteira. Mas cinco tem um prefixo puramente investigativo que pode ser liberado
como escopo REDUZIDO e explicitamente medido, entregando evidencia sem escrever nada:

- ORQ-12: levantar o desenho da coluna e da nova chave de conflito, sem migration. IDEMPOTENTE.
- ORQ-13: medir a lacuna de dado - onde `thinking_level` vive hoje e por que nao chega ao custo -
  usando `task_usage` existente, sem execucao nova paga. IDEMPOTENTE.
- ORQ-17: desenhar a opcao de exposicao com autenticacao e rollback, sem aplicar. IDEMPOTENTE.
- ORQ-18: especificar o componente e o teste, sem escrever arquivo, e depois da ORQ-26. IDEMPOTENTE.
- ORQ-21: mapear registry do ORQ2 -> `accounts`/`approved_accounts`/`assignments`, sem INSERT.
  IDEMPOTENTE.
- ORQ-23: a metade "Fase 3 fecha com evidencia" e IDEMPOTENTE; o gate 4.5 NAO pode entrar no
  mesmo rerun.

Isso e recomendacao, nao autorizacao: quem decide e o owner, e quem escreve e o Codex56-TL.

## Nada mutado

Nenhuma das 9 issues preservadas foi reexecutada. Nenhum codigo, arquivo de repo, `.env` ou
`/tmp/daemon.env.bak` tocado. Nada compilado, nada reiniciado. Somente `GET` na API do backend e
leitura de fonte. O unico arquivo que criei e este, mais o meu check-out.
