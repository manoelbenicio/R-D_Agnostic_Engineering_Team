# P0 — nome duplicado de agente: premissa corrigida + GAP A implementado (`4f87b90`)

- Executor único: Codex56#A (`w7:p3`) · UTC 2026-07-27T18:14Z
- Worktree autorizado: `/home/ec2-user/workspace/worktrees/gtl-p0-agent-name-unique`,
  branch `agent/codex56-a/p0-agent-name-unique`, base `0cb8aeb`
- Commit local: **`4f87b90`** — `fix(agent): map unique-name violation to 409 on update`
- Escopo: `internal/handler/agent.go` (+11) e o novo `internal/handler/agent_duplicate_name_test.go`.
  Nenhuma migration, nenhuma query, nenhum `generated`, nenhum dado alterado. Sem push, board ou prod.

## 1. A premissa original era falsa, e isso mudou a task

O pedido inicial era criar validação de unicidade de nome por workspace. Medi antes de escrever e a
proteção **já existe**:

```text
pg_constraint  agent_workspace_name_unique | UNIQUE (workspace_id, name)     (migration 046, aplicada)
pg_indexes     agent_workspace_name_unique | CREATE UNIQUE INDEX ... (workspace_id, name)
```

As supostas duplicatas estão em **workspaces diferentes**:

```text
Codex-A | ws=20fce817-895d-447b-965a-49f5e279314a | arch=no
Codex-A | ws=6733441a-13f6-4a27-b393-d3c66d0d3425 | arch=no
Codex-B | ws=20fce817-895d-447b-965a-49f5e279314a | arch=no
Codex-B | ws=6733441a-13f6-4a27-b393-d3c66d0d3425 | arch=no
```

E agrupando por `(workspace_id, name)` entre não arquivados o resultado é **vazio**. Corrijo meu
relatório anterior do P0 AGY: eu agrupei só por nome e reportei ambiguidade que não existe dentro de um
workspace. A issue que eu havia preparado para o Leader deve ser retirada.

## 2. GAP A — o que estava realmente quebrado

`CreateAgent` (`agent.go:829-831`) e o caminho de template (`agent_template.go:457`) já traduziam a
violação para 409. `UpdateAgent` **não**: o erro caía no ramo genérico e devolvia

```text
500 {"error":"failed to update agent: ERROR: duplicate key value violates unique constraint
     \"agent_workspace_name_unique\" (SQLSTATE 23505)"}
```

Dois problemas num só: reporta conflito de cliente como falha de servidor, e ecoa nome de constraint e
SQLSTATE para o chamador. Agora responde **409** com
`an agent with this name already exists in this workspace` — nomeia campo e escopo, sem id do agente
colidente, sem texto de driver.

## 3. Testes: 5 casos, banco real, zero skip

O pacote `internal/handler` tem `TestMain` que faz `os.Exit(0)` sem Postgres
(`handler_test.go:38-53`), o que produziria falso-verde. Usei o Postgres efêmero autorizado:

```text
ORQ1: docker run pgvector/pgvector:pg17 -p 127.0.0.1:15544:5432   (porta 15433 já é do Postgres de produção — não tocada)
ORQ2: ssh -f -N -L 15544:127.0.0.1:15544                          (túnel local)
      go run ./cmd/migrate up  -> aplica até 126_runtime_profile_protocol_family_native_runtimes
```

```text
--- PASS: TestAgentDuplicateName_CreateAndUpdateReturnConflict
    create with a taken name conflicts
    rename onto a taken name conflicts instead of 500
    renaming an agent to its own name is idempotent
    rename to a free name succeeds
    name freed by a rename becomes available again
ok  internal/handler  0.148s     (nenhum SKIP)
```

**Prova por mutação**, executada e revertida: removendo o novo ramo, o caso de rename falha com
exatamente o defeito original —
`expected 409, got 500: {"error":"failed to update agent: ERROR: duplicate key value violates unique constraint \"agent_workspace_name_unique\" (SQLSTATE 23505)"}`.
O teste detecta o defeito, não repete a implementação. O helper `assertContentFreeConflict` recusa
`sqlstate`, `23505`, o nome da constraint, `duplicate key` e fragmentos de SQL no corpo.

## 4. Gates

```text
gofmt -l (2 arquivos)                         -> vazio
go build ./internal/...                       -> exit 0
go vet ./internal/handler/...                 -> exit 0
go test -run TestAgentDuplicateName -count=1  -> ok, 5/5, zero skip
go test -race -run TestAgentDuplicateName     -> ok 1.915s
go test ./internal/handler -count=1 (completo) -> 3 falhas
git diff --check                              -> exit 0
git status --porcelain                        -> vazio
```

As 3 falhas do pacote completo são **pré-existentes**, não minhas. Rodei o mesmo pacote na base
(repo principal, `handler` intocado) contra o mesmo banco efêmero e o conjunto de falhas é idêntico:

```text
TestCreateChatSession_Routing
TestCreateWorkspaceUsesRequestedSlug
TestCreateWorkspace_DoesNotMarkOnboarded
comm -23 (falhas minhas − falhas da base) -> vazio
```

Elas merecem card próprio; não são objeto desta task.

## 5. Teardown do banco efêmero

```text
docker rm -f codex56a-ephemeral-pg-20260727T181012Z   -> container_removido; 0 restantes
kill do túnel (pid 3973297)                           -> encerrado
psql pós-teardown em 127.0.0.1:15544                  -> "Is the server running on that host..." (recusado)
arquivos temporários de apoio                         -> removidos
```

O Postgres de **produção** em `127.0.0.1:15433` no ORQ1 não foi tocado em nenhum momento.

## 6. Limite explícito, como exigido

O guard entregue é o mapeamento de erro do banco, então a **atomicidade vem da constraint**, não do
handler. Registro o que isto **não** resolve:

1. **Corrida concorrente**: um guard de leitura no handler (`SELECT` antes do `INSERT/UPDATE`) nunca
   elimina a janela entre checagem e escrita. Quem garante a exclusão mútua aqui é o índice único; o
   handler apenas traduz a rejeição. Foi essa a escolha deliberada — não adicionei pré-checagem, que
   daria falsa sensação de segurança.
2. **GAP B, unicidade insensível a caixa e espaços**: `Codex-A`, `codex-a` e `Codex-A ` continuam
   coexistindo no mesmo workspace e permanecem ambíguos para atribuição por nome. Fechar isso exige
   índice único funcional em `lower(btrim(name))` e o saneamento prévio dos dados existentes — é
   **follow-up LANE-DB**, sem implementação agora, conforme decidido.

## 7. Não-alegações

- Não criei, alterei nem arquivei nenhum agente real; as linhas dos testes são `dupname-test-*` no
  banco efêmero, removidas pelo `t.Cleanup` e depois pelo teardown do container.
- Não toquei migrations, `pkg/db/queries`, `pkg/db/generated` nem qualquer arquivo da fila LANE-DB.
- Não fiz push, PR, merge, deploy ou mutação de board.
- Não corrigi as 3 falhas pré-existentes do pacote `handler`.
- Não implementei GAP B nem pré-checagem de nome; ambos declarados na §6.
- Nenhum segredo lido; as credenciais do banco efêmero são literais descartáveis de teste ligados a
  `127.0.0.1` no ORQ1.
