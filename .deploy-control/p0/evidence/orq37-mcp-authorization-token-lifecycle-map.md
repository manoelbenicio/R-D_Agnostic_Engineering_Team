# ORQ-37 — MCP Authorization & "Gatekeeper" token lifecycle: mapa factual e contrato (READ-ONLY)

- Card: **ORQ-37** · Executor: Codex56#A (`w7:p3`) · UTC 2026-07-28T16:26Z
- Skill `.agents/skills/aws-secrets-manager/SKILL.md` já lida integralmente nesta sessão; regras em vigor.
- Fase: **read-only**. Nenhum valor de token lido, nenhuma chamada AWS, nenhum `asm-exec`, nenhum acesso
  ao SMA, nenhuma mutação de runtime, board, container ou daemon. Tudo classificado por **nome de chave,
  caminho, prefixo e metadado**.

## VEREDITO DE READINESS: **BLOCK** — o card mistura três planos distintos e nomeia um componente que não existe

Não é BLOCK por defeito de código: é BLOCK de **definição**. Como está, o card não é executável sem
decidir qual dos três planos abaixo é o alvo, e um dos termos do título não corresponde a nada no
produto.

## 1. "Gatekeeper" não existe no produto

```text
grep -rliE "gatekeeper"  (repo, fora de node_modules)
  -> .deploy-control/p0/evidence/{gtl-kanban-security-regularization-mapping,orq32-33-related-secret-cards-triage,
     current-pending-tasks,gtl-todo-and-milestone-cards-inventory}.md
  -> TL_chat_history.md
  -> multica-auth-work/apps/desktop/build/entitlements.mac.plist   (string do macOS, sem relação)
```

Zero ocorrências em código Go, TS, SQL ou YAML de produto. O termo vive **apenas** em documentos de
coordenação e num plist de entitlements do desktop. É o terceiro caso do mesmo padrão que eu já
documentei: "handshake token" (ORQ-32, falso positivo) e "rev token" (ORQ-33, sem emissor nem
consumidor). Recomendação: substituir "Gatekeeper" pelo nome real do plano escolhido no §5.

## 2. Também não existe superfície MCP **de produto**

```text
grep -rniE "mcp" internal/router internal/handler cmd/server  (sem testes/mcp_config)
  -> apenas comentários; nenhuma rota
grep -rn "mcp" cmd/server/router.go -> (vazio)
```

Não há endpoint `/api/mcp*`, não há middleware de autorização MCP e não há servidor MCP dentro do
produto. Logo **não existe "MCP Authorization" a rotacionar ou revogar no backend**.

## 3. Os três planos que o card confunde, e o que cada um realmente é

### Plano A — tokens de API do produto (existe, é código, tem ciclo de vida)

| classe | prefixo | emissor | consumidor (validação) | revogação |
|---|---|---|---|---|
| PAT humano | `mul_` | `GeneratePATToken` (`auth/jwt.go:62`) | `middleware/auth.go` (ramo `mul_`) | tabela de PAT + fluxo de membro |
| token de daemon | `mdt_` | `GenerateDaemonToken` (`:71`) — **0 chamadores em produção** | `middleware/daemon_auth.go` + cache Redis (`auth/daemon_token_cache.go:27`) | `DeleteDaemonTokensByWorkspaceAndDaemons` em `handler/workspace_revoke.go:105`; `DeleteExpired` por query |
| token de task de agente | `mat_` | `GenerateAgentTaskToken` (`:84`) — **1 chamador em produção** | `middleware/auth.go:151-161` (ramo `mat_`) | `DeleteTaskTokensByTask`, `DeleteExpiredTaskTokens` (`queries/task_token.sql`) |
| PAT de nuvem | `mcn_` | externo | `middleware/auth.go:83-84` | fora do produto |
| JWT de sessão | — | `handler/auth.go:213,226` | `middleware/auth.go:295`, `daemon_auth.go:234`, `realtime/hub.go:691` | inexistente por design: chave única com `sync.Once`, sem `kid` |

Fato que já registrei no ORQ-43 e **confirmo hoje**: `GenerateDaemonToken` tem **zero** chamadores de
produção, então a classe `mdt_` tem validador, cache e revogação, mas **nenhum emissor** — o ciclo de
vida está aberto na ponta da emissão. `mat_` tem exatamente um emissor, o que é coerente.

### Plano B — segredos de terceiros guardados pelo produto (existe, e tem fronteira de autorização)

A coluna `agent.mcp_config` guarda configuração de servidores MCP que, nas palavras do próprio código,
"routinely embed third-party API tokens" (`handler/agent.go:601`). A fronteira de leitura é explícita e
correta:

```text
handler/agent.go:571     mcp_config segue a regra always-redact do workspace + gate owner/admin por linha
handler/agent.go:601-606 ator do tipo "agent" NUNCA vê mcp_config, mesmo com PAT do host que satisfaria
                         o papel — fecha o mesmo vetor de movimento lateral que MUL-2600 fechou para custom_env
```

Ou seja: aqui há **autorização MCP de verdade**, mas ela é sobre *quem pode ler a configuração*, não
sobre autenticar chamadas MCP. Nada a rotacionar no produto; o que existe é um gate de exposição, e ele
está implementado.

### Plano C — cabeçalhos MCP do *tooling* nos hosts (fora do produto)

`/tmp/mh` e `/tmp/mcp_h` no ORQ1, ambos hoje `600 ec2-user:ec2-user` (verificado por mim no ORQ-31),
contêm cabeçalho `Authorization`/`Bearer` para ferramenta MCP local — é o item **B7** do plano da Wave B.
O MCP configurado neste repo é o `aws-mcp`, em `.kiro/settings/mcp.json`, cujo servidor declara apenas
`command`, `args` e `timeout`, **sem bloco `env`** e portanto sem credencial embutida no arquivo de
configuração.

## 4. Contrato factual de ciclo de vida (o que é executável, por plano)

**A. `mdt_` — fechar a ponta de emissão** (é o único item com defeito real de ciclo de vida)
1. decidir se a classe é para ser emitida: se sim, implementar o emissor e o fluxo de entrega ao daemon;
   se não, remover `GenerateDaemonToken` ou marcá-la explicitamente como reservada;
2. enquanto não houver emissor, **não** existe rotação a fazer: rotacionar zero tokens é no-op;
3. revogação já é convergente e transacional (`workspace_revoke.go:105`), com hashes retornados — manter;
4. o cache Redis precisa de invalidação comprovada junto da revogação, senão um `mdt_` revogado continua
   aceito até expirar o TTL. **Isto é o único risco de segurança concreto que encontrei neste card** e
   merece verificação dedicada (não a fiz: exigiria runtime).

**B. `mcp_config` — manter e provar o gate**
5. o gate de redação é a fronteira; qualquer novo consumidor de `mcp_config` deve passar por
   `canViewAgentSecrets` + `alwaysRedact` + a regra "ator agente nunca vê";
6. contrato de teste: um ator do tipo agente com PAT de dono **não** deve receber `mcp_config` — hoje
   isso está no código, e vale um teste de regressão explícito se ainda não existir.

**C. cabeçalhos MCP do tooling — Wave B / B7, fora do produto**
7. rotação, se houver, é na origem do token da ferramenta, não no produto;
8. contenção de arquivo já aplicada (`0600`); a causa raiz (`umask 002`) segue aberta e é da Wave B.

**Rollback, para qualquer item acima**
9. nenhum dos itens exige mudança de schema, então rollback é reversão de código ou de arquivo, sem
   migração;
10. para `mdt_`, se um emissor for adicionado e precisar ser removido, a revogação em massa já existe e
    é o caminho de rollback: `DeleteDaemonTokensByWorkspaceAndDaemons` mais invalidação do cache.

## 5. O que o GTL precisa decidir para desbloquear o card

1. **qual plano é o alvo**: A (`mdt_`/`mat_` lifecycle), B (fronteira de leitura de `mcp_config`) ou C
   (cabeçalhos do tooling)? Os três têm donos e riscos diferentes e não devem viver num card único;
2. **renomear o card**, removendo "Gatekeeper" e, se o alvo não for C, removendo também "MCP
   Authorization", que não existe como superfície de produto;
3. se o alvo for A, autorizar a verificação de **invalidação de cache na revogação** (item 4 acima), que
   é a única questão de segurança acionável que este mapeamento encontrou.

## 6. Não-alegações

- Não li valor de token, cabeçalho, chave ou configuração MCP; classifiquei por prefixo, nome de coluna
  e caminho.
- Não chamei AWS, não usei `asm-exec`, não acessei o SMA, não toquei runtime, container, daemon, fila ou
  board.
- **Não verifiquei** se a revogação de `mdt_` invalida o cache Redis: isso exige execução e está marcado
  como pendente, não como defeito confirmado.
- Não inventariei consumidores externos de `mul_`/`mcn_` fora do backend.
- Não afirmo que os cabeçalhos em `/tmp/mh` e `/tmp/mcp_h` são inválidos ou válidos: apenas que estão
  contidos em `0600` e que pertencem ao tooling, não ao produto.
