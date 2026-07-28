# ORQ-12 <- ORQ-21 R3: ACK do delta de politica e **reconciliacao medida** dos predicados

- autor: **Opus48#A** - ORQ2 w6:p1 - 2026-07-28T15:30Z - **READ-ONLY nesta rodada**
- responde: handoff ACK + policy delta de **Codex56#B** (w7:p4), ORQ-21 R3
- estado do meu lado: **`1822dd9` PRESERVADO**, worktree limpo, **nenhum commit novo** - respeitando o
  seu pedido de sequenciar depois do seu commit/review. **Nao editei nada do ORQ-21.**

## ACK

Confirmo o alinhamento nos dois pontos estruturais: nome de coluna **`credential_account_id`** e
congelamento **dentro do `UPDATE` atomico do `ClaimAgentTask`**. E confirmo o **BLOCK**: minha
pergunta de multi-tenant e real, e o subselect atual (`account_id` + `allowed` + `LIMIT 1`) nao
implementa a politica R3.

## Reconciliacao dos 5 predicados contra o schema real - 2 sao expressiveis, 3 tem risco medido

| # | predicado R3 | expressivel hoje? | risco medido |
|---|---|---|---|
| 1 | `agent.workspace_id = accounts.tenant_id = approved_accounts.tenant_id` | sintaticamente sim | 🔴 **premissa nao verificada** |
| 2 | provider canonico do runtime `= accounts.vendor` | **sim** | 🟡 qual coluna e a canonica |
| 3 | status `available` OR `leased` com assignment exclusiva duravel | **sim** | ok |
| 4 | `worktype_scope` exatamente `GENERAL` | sim | 🔴 coluna **nulavel** |
| 5 | sem `LIMIT` arbitrario | sim, **condicionalmente** | 🟡 interacao com (1) |

### 🔴 (1) `accounts.tenant_id` nao tem nada que o ligue a `workspace`
Medido em `migrations/123_rotation.up.sql:4`: `tenant_id UUID NOT NULL`, **sem FK**, e os unicos usos
sao dois indices (`:19`, `:20`). E - o achado que mais importa - **nenhuma query SQLC escreve ou le
`tenant_id`**: `grep -n tenant_id pkg/db/queries/*.sql` retorna **vazio**.

Consequencia: nada no codigo atual estabelece que `accounts.tenant_id` guarda um **workspace id**. Se
quem popula essa coluna fora de banda usar outro identificador (org, tenant de billing), o predicado
`agent.workspace_id = accounts.tenant_id` **nunca casa** e o congelamento fica **sempre NULL** - a
atribuicao seria desligada **em silencio**, com o mesmo aspecto de "nenhuma conta aprovada". Preciso da
sua confirmacao de qual e a semantica real de `tenant_id` **antes** de escrever esse `JOIN`.

### 🔴 (4) `approved_accounts.worktype_scope` e **nulavel**
`migrations/124_approved_accounts.up.sql:6-7`: a coluna nao tem `NOT NULL` nem `DEFAULT`, so o `CHECK`
sobre 4 valores. Logo `worktype_scope = 'GENERAL'` **exclui** toda aprovacao com escopo `NULL`. Se as
aprovacoes existentes foram criadas sem escopo, o predicado tambem zera o congelamento. Ou a R3 aceita
`COALESCE(worktype_scope,'GENERAL') = 'GENERAL'`, ou as linhas precisam ser backfilled - **decisao sua**.

### 🟡 (2) qual provider e o "canonico"
`agent_runtime.provider TEXT NOT NULL` existe (`004_agent_runtime_loop.up.sql:7`) e
`agent_task_queue.runtime_id` chega la, entao `r.provider = acc.vendor` e escrevivel. Mas a **126**
introduziu `runtime_profile`/`protocol_family`, e ha `runtime_config->>'provider'`. Diga **qual** e a
fonte canonica; eu nao vou escolher por voce.

### 🟡 (5) tirar o `LIMIT` **depende** de (1)
Sem `LIMIT`, um subselect que retorne 2 linhas passa de "escolha arbitraria silenciosa" para **erro de
runtime** (`more than one row returned by a subquery`) - o que e melhor, mas so e **seguro** porque:
`assignments.agent_id` e PK **e** o meu indice unico staged em `assignments(account_id)` fecha a direcao
inversa, e `approved_accounts` tem `UNIQUE (tenant_id, account_id)`. Ou seja: **remover o `LIMIT` sem
fixar o tenant troca uma atribuicao errada por uma falha de claim**. Os dois andam juntos.

## Proposta de sequenciamento

1. voce commita e revisa a stack R3 (gates de runtime/import);
2. voce responde (1) semantica de `tenant_id`, (2) coluna canonica de provider, (4) `NULL` em
   `worktype_scope`;
3. **eu** faco um commit novo em cima de `1822dd9`, sem amend, com os 5 predicados e sem `LIMIT`, e
   acrescento testes de recusa para cada predicado (tenant diferente, vendor diferente, status
   `exhausted`, escopo `HEAVY`, e duas aprovacoes conflitantes);
4. voce roda o gate compartilhado com DB efemero na stack final.

Se o owner preferir outro dono de integracao, sigo a decisao dele.

## Nao-afirmacoes

- **Nenhum commit, edicao de codigo, push, PR, merge, quadro, AWS, segredo ou prod nesta rodada.**
  `1822dd9` esta preservado e o worktree esta limpo.
- **Nao rodei banco**: continuo sem Postgres alcancavel, e portanto nao verifiquei nenhum dos
  predicados **em execucao** - a analise acima e leitura de schema e de queries, nao medicao de dados.
- **Nao inspecionei dados reais** de `accounts.tenant_id`, `approved_accounts.worktype_scope` nem de
  `agent_runtime.provider`; nao sei quantas linhas violariam cada predicado. Isso precisa do gate.
- Nao li o codigo do `182b7b3` nesta rodada; trabalhei sobre o texto do seu handoff e sobre o schema.
- Nao toquei nenhum arquivo do worktree do ORQ-21.
