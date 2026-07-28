# ORQ-43 — Design de ciclo de vida e rotação do daemon token `mdt_` (READ-ONLY)

- Card: **ORQ-43** · UUID `d1149dd3-9da8-4678-a4b7-d98f3eddca14` · number 43 · `todo` / high / sem assignee
- Autor: Codex56#A (`w7:p3`) · UTC 2026-07-27T15:22Z
- Skill carregada integralmente antes de qualquer coisa: `.agents/skills/aws-secrets-manager/SKILL.md`
  (regras 1–3: proibido `get-secret-value`/`batch-get-secret-value`, proibido ler da SMA em
  `localhost:2773`, obrigatório `{{resolve:secretsmanager:...}}` resolvido por `asm-exec` no processo
  filho).
- Modo: **READ-ONLY**. Nenhum valor de segredo lido, nenhum `GetSecretValue`, **nenhum `asm-exec`
  executado** (apenas citado como forma correta), nenhuma mutação de código, board, config, serviço
  ou AWS.
- **Fora de escopo por decisão explícita:** `mat_`/`MULTICA_TOKEN` (ORQ-36) e PAT `mul_`/`mcn_`.

## 1. ACHADO QUE PRECEDE O DESIGN: hoje não existe emissor de `mdt_` no código

| elemento | existe? | evidência |
|---|---|---|
| **Gerador** | sim, mas **sem chamador** | `internal/auth/jwt.go:70-76` `GenerateDaemonToken()` → `"mdt_"+40 hex`. `grep -rn GenerateDaemonToken --include=*.go` (excluindo `generated/`) retorna **apenas a própria definição** |
| **Persistência** | query existe, **sem chamador** | `pkg/db/queries/daemon_token.sql:1` `-- name: CreateDaemonToken :one`; `grep -rn CreateDaemonToken` fora de `pkg/db/generated` → **zero** ocorrências |
| **Validador** | sim, ativo | `internal/middleware/daemon_auth.go:106-107` (ramo `mdt_`), `:122` `GetDaemonTokenByHash`, `:79` `DaemonAuth(...)`; cadeia montada em `cmd/server/router.go:511` |
| **Cache** | sim | `internal/auth/daemon_token_cache.go:31-43` (Redis; `nil` desabilita), `:50` `Get`, `:69-75` `Set` com TTL clampado por `TTLForExpiry`, `:92` `Invalidate` |
| **Revogação em massa** | sim, um chamador | `pkg/db/queries/daemon_token.sql:10` `DeleteDaemonTokensByWorkspaceAndDaemons`, usada em `internal/handler/workspace_revoke.go:105` (retorna `RevokedTokenHashes`) |
| **Expurgo** | query existe, sem chamador visível | `daemon_token.sql:23` `DeleteExpiredDaemonTokens` |
| **Storage client-side** | **nenhum** | `grep` por `daemon_token|DaemonToken|daemonToken` em `cmd/multica` → vazio |

Schema (`migrations/029_daemon_token.up.sql:1-11`):

```sql
CREATE TABLE daemon_token (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  token_hash TEXT NOT NULL,
  workspace_id UUID NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
  daemon_id TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now());
CREATE UNIQUE INDEX idx_daemon_token_hash ON daemon_token(token_hash);
CREATE INDEX idx_daemon_token_workspace_daemon ON daemon_token(workspace_id, daemon_id);
```

**Consequência de escopo:** o plano de dados **suporta rotação com sobreposição nativamente** (índice
único é por `token_hash`, e `(workspace_id, daemon_id)` é índice **não único**, logo N tokens válidos
por daemon), **mas não há caminho de emissão em produção**. Portanto ORQ-43 tem duas metades:

- **43-A (implementação, precede a operação):** expor emissão/renovação (`GenerateDaemonToken` +
  `CreateDaemonToken`), persistência client-side no daemon e o expurgo `DeleteExpiredDaemonTokens`.
  Sem isso, "rotacionar" significaria apenas **revogar** (via `workspace_revoke`) e deixar o daemon
  sem credencial — disrupção total, não rotação.
- **43-B (operação de rotação):** sequência abaixo, executável **somente depois** de 43-A.

Isto é o mesmo padrão de falha do ORQ-32 (runbook assumindo mecanismo inexistente), detectado **antes**
de virar runbook.

## 2. Sequência de rotação com sobreposição (aceite condicional)

**Overlap é aceitável e é a estratégia correta**, porque o schema permite múltiplos tokens válidos por
daemon e o validador resolve por hash, sem estado de "token único". Sequência de disrupção limitada
(não zero-downtime enquanto 43-A não existir):

```text
S0  pré-condição: 43-A entregue; fila ativa ZERO em DUAS leituras (§3)
S1  emitir token NOVO para o mesmo (workspace_id, daemon_id) com expires_at futuro
       -> o token ANTIGO permanece válido: janela de sobreposição aberta
S2  entregar a referência do novo token ao host do daemon SEM plaintext em contexto:
       arquivo 0600 sob diretório 0700, escrito por processo filho, por exemplo
       asm-exec -- sh -c 'umask 077; printf %s "{{resolve:secretsmanager:<id>:SecretString:<key>}}" > <path>'
       (forma correta pela skill; NÃO executada nesta auditoria)
S3  recarregar o daemon para adotar a nova referência (reload/restart da unit de usuário)
S4  provar adoção: daemon online, reregistro concluído, discovery por provider obrigatório,
       ZERO task paga criada (card sem assignee; ver §5 do check-out)
S5  revogar o token ANTIGO: DeleteDaemonTokensByWorkspaceAndDaemons (handler/workspace_revoke.go:105)
       + Invalidate(hash) no cache (daemon_token_cache.go:92) para cada RevokedTokenHashes
S6  provar rejeição do antigo: requisição de daemon com o token antigo -> 401/403
S7  fechar a janela: confirmar que só o token novo consta em daemon_token para o par
```

Ponto crítico de cache: `Set` usa TTL clampado por `TTLForExpiry` (`:69-75`), então um token revogado
**pode sobreviver em cache até o TTL** se `Invalidate` não for chamado. Por isso S5 exige invalidação
explícita por hash — e é isso que torna a revogação verificável em S6.

## 3. Gate de fila zero (4 estados, duas leituras)

```sql
SELECT count(*) FROM agent_task_queue
WHERE status IN ('queued','dispatched','running','waiting_local_directory');
```

Exigido **= 0 em duas leituras** separadas por intervalo de estabilidade, antes de S1 e reconferido
antes de S3 e S5. A omissão de `waiting_local_directory` invalida o gate.

## 4. Reconexão do daemon e prova de saúde

- daemon volta `online` no backend e reregistra runtimes;
- `discovery` por provider obrigatório (AGY, Codex, Kiro) coerente com a capacidade comprovada;
- backend `/health` 200; nenhum restart-loop na unit;
- **nenhuma task sintética paga** é necessária para ORQ-43: a prova é de autenticação e registro, não
  de execução. Se alguém exigir task, ela vira card próprio com orçamento.

## 5. Rollback

| falha em | ação |
|---|---|
| S3/S4 (daemon não adota) | restaurar a **referência anterior** no host e recarregar; o token antigo **ainda é válido** porque S5 não ocorreu ⇒ retorno imediato ao estado bom |
| S5 (revogação incorreta) | reemitir token para o par e reentregar referência; `daemon_token` é reemitível, e nenhuma evidência é apagada |
| qualquer etapa | nunca apagar linhas de auditoria nem `task_message`; a revogação registra `RevokedTokenHashes`, que é o rastro |

Invariante de rollback: **S5 é o ponto de não-retorno barato**; antes dele o rollback é trocar de volta
uma referência de arquivo.

## 6. Auditoria

- registrar, sem valor: `daemon_id`, `workspace_id`, `token_hash` (hash, não segredo), `expires_at`,
  timestamps de S1/S3/S5, e os `RevokedTokenHashes` retornados;
- IAM/CloudTrail conforme a skill ("best-effort defense, not a security boundary": combinar com
  least-privilege e monitoramento);
- nenhum valor em stdout, argv, log, diff, card ou evidência — nem em mensagem de erro.

## 7. Testes propostos (não escritos, não executados)

1. `TestDaemonAuthAcceptsTwoValidTokensForSameDaemon` — prova a sobreposição no validador.
2. `TestDaemonAuthRejectsRevokedTokenAfterCacheInvalidate` — revoga, invalida, espera 401/403.
3. `TestRevokedTokenStillCachedWithoutInvalidateIsAccepted` — **teste de caracterização** do risco de
   `TTLForExpiry`; documenta por que S5 exige `Invalidate`.
4. `TestCreateDaemonTokenPersistsHashOnlyAndUniqueByHash` — nenhum plaintext em coluna; unicidade por
   hash.
5. `TestDeleteExpiredDaemonTokensPurgesOnlyExpired`.
6. `TestDaemonTokenScopedToWorkspaceAndDaemon` — token de outro par não autentica (cross-tenant).
7. Todos herméticos, `-count=1`, sem banco compartilhado, sem skip global.

## 8. Pedido de revisão independente

Solicito **peer review adversarial** por um agente que não seja eu e que não tenha escrito ORQ-42/43/44,
com foco em: (a) se a conclusão "não há emissor em produção" resiste a uma busca própria; (b) se a
sobreposição é realmente segura sob o validador atual; (c) se S5 sem `Invalidate` é o único risco de
cache; (d) se a ordem S1→S7 tem ponto de disrupção não declarado.

## 9. Não-alegações

- Não li valor de segredo, não chamei `GetSecretValue`/`BatchGetSecretValue`, **não executei `asm-exec`**
  e não toquei AWS. Os comandos com `{{resolve:...}}` são forma correta a executar por quem tiver a
  janela — não foram rodados.
- Não editei código, migration, config, board ou serviço; não reiniciei nada.
- Não provei que a rotação funciona: 43-A não existe. O design é condicional a ele.
- Não verifiquei o conteúdo de `daemon_token` no banco vivo (nem contagem por daemon): exigiria leitura
  operacional que não é necessária para o design e ficaria desatualizada.
- Não afirmo que exista um segredo AWS correspondente ao `mdt_` hoje; se a fonte escolhida for Secrets
  Manager, o ID do segredo é decisão do owner.
