# P1 reasoning — prontidão de integração (`6bf0090` → `d9d569d` → `f5660e9`)

- Executor único: Codex56#A (`w7:p3`) · UTC 2026-07-28T13:16Z
- Worktree: `/home/ec2-user/workspace/worktrees/gtl-p1-reasoning-gateway`,
  branch `agent/codex56-a/p1-reasoning-thinking-gateway`, base `0cb8aeb`, worktree **limpo**
- Escopo total dos três commits: **2 arquivos**,
  `internal/daemon/brain_integration.go` e `internal/daemon/brain_integration_thinking_test.go`.
  Nenhuma migration, query, `generated`, `daemon.go`, `types.go`, `router.go` ou dado.
- Sem push, merge, deploy ou board.

## 1. Correção factual entregue (`f5660e9`)

O comentário anterior afirmava que `agy` não expõe flag de effort. **Falso**: `agy --help` em 1.1.7
lista `--effort  Reasoning effort for the current CLI session (low|medium|high)`.

O que falta é **wiring de produto**, e isso está verificado na fonte:

- `pkg/agent/antigravity.go` nunca lê `opts.ThinkingLevel` (zero referências);
- `antigravity` não tem entrada em `providerThinkingEnums` (`pkg/agent/thinking.go:875-935`), logo
  `handler/agent.go:768` já rejeita `thinking_level` para runtimes antigravity com 400.

Consequência de desenho, mantida: `gatewayApprovedThinkingLevels` continua **sem** antigravity, porque
anunciar um nível que o filho nunca recebe seria mentir. **Não alterei comportamento do runtime
Antigravity** — ligar o `--effort` no produto é mudança com escopo próprio e exige aprovação separada.
As asserções dos testes são as mesmas; só as justificativas passaram a estar corretas, incluindo a
distinção CLI × produto.

## 2. Reconciliação do `brain_integration.go` sujo no repo principal — sem tocá-lo

O repo principal tem edição **não commitada** nesse arquivo desde 2026-07-26 23:51. Conteúdo integral:

```diff
+	// Native execution has no Agent Brain launch plan. Its provider-specific
+	// backend validates the persisted thinking level, so the gateway allowlist
+	// must not run (or dereference a nil plan) on this path.
+	if plan == nil {
+		return nil
+	}
```

São 6 linhas, e elas **já estão** dentro do meu `6bf0090`: `grep -c "if plan == nil"` retorna 3 tanto
no meu commit quanto na árvore atual, e o prólogo de `validateThinking` é **idêntico** entre a versão
suja e a minha. Portanto:

- **nada a portar**: a alteração solta é subconjunto do que integro;
- **não a toquei** e não vou tocar — o descarte (`git checkout --` no repo principal) é do dono daquela
  árvore;
- se ela permanecer, o merge do meu branch resolve trivialmente, mas gera conflito bobo de contexto sem
  necessidade. Recomendo descartar antes.

## 3. Matriz: produto atual × depois do deploy

Estado atual medido em produção: imagem `multica-backend:agy-status-20260727T102815Z`, schema com
`max = 126`, `OMNIROUTE_DEV_MODELS_COMPAT=1`, 20 agentes com `thinking_level = NULL`.

| capacidade | produto HOJE | DEPOIS do deploy destes 3 commits |
|---|---|---|
| agente com `thinking_level` não vazio, gateway (compat) | **task morre na admissão** com `thinking_not_approved`; único estado que despacha é NULL | nível **admitido** se estiver na allowlist do provider; a task segue para execução |
| `codex` (`none…xhigh`) | rejeitado | admitido; `applyCodexReasoningEffort` recebe o valor |
| `claude-code` (`low…max`) | rejeitado | admitido; `--effort` recebe o valor |
| `openai-compatible`/cline (`none…xhigh`) | rejeitado | admitido |
| nível desconhecido (`ultra`) | rejeitado (por acidente, junto com os válidos) | rejeitado **por regra**, fail-closed |
| `" "`, tab, `"medium "` | rejeitado | rejeitado, verbatim, sem normalizar |
| `""` (default) | admitido | admitido, idêntico |
| modelo sem reasoning, schema enriquecido (flag ausente) | rejeitado | rejeitado — `Reasoning=false` observado é veto |
| modelo sob projeção compat (`Reasoning=false` sintético) | rejeitado | **não é veto**: gate de modelo declarado indisponível, allowlist de provider decide |
| Antigravity com nível | 400 na API | **inalterado**: 400 na API, e fail-closed no gateway |
| Kiro com nível | native funciona (`--effort`); gateway impossível (sem CLIKind) | **inalterado**, e agora pinado por teste |
| caminho nativo (plan nil) | provider valida | **inalterado**, byte a byte |
| `task_usage.thinking_level` | ausente | **ausente** — é ORQ-13/LANE-DB, fora daqui |
| schema / dados | — | **nenhuma mudança** |

Risco de regressão: o único caminho alterado é `validateThinking`. Com `thinking_level` NULL — que é
100% dos agentes hoje — o comportamento é idêntico ao atual, então o deploy é inerte até alguém
configurar um nível. É o oposto do estado atual, em que configurar um nível derruba a task.

## 4. Plano de integração limpo

| # | passo | dono | pré-condição / prova |
|---|---|---|---|
| 1 | descartar a edição suja de `brain_integration.go` no repo principal | dono da árvore principal | §2: conteúdo já contido em `6bf0090` |
| 2 | peer review independente dos 3 commits | revisor ≠ eu | diff de 2 arquivos; foco em `validateThinking` e nos 7 testes |
| 3 | rebase do branch sobre a `integration/dev-transition-candidate-20260719` corrente | eu, sob autorização | hoje base é `0cb8aeb`; se a integração avançou, rebase e re-rodar gates |
| 4 | gates de integração | eu | `gofmt`, `go build ./internal/...`, `go vet ./internal/daemon/...`, `go test ./internal/daemon` completo, `-race` focado, `git diff --check` |
| 5 | merge na branch de integração | GTL | após 2 e 4 |
| 6 | build da imagem + restart do backend/daemon | owner de deploy | fora do meu escopo; sem isso o código não vale em produção |
| 7 | verificação pós-deploy | eu, read-only | configurar 1 agente codex com `high`, observar a task admitir e o filho receber o effort; reverter para NULL |
| 8 | follow-ups registrados | GTL | (a) exportar enum imutável em `pkg/agent` (handoff pedido); (b) projeção compat derivar `Reasoning` de verdade, ou OmniRoute servir o schema enriquecido; (c) `task_usage.thinking_level` em ORQ-13; (d) wiring de `--effort` no Antigravity, com aprovação própria |

Gates já verdes nesta árvore, medidos agora com caches privados 0700 fora de `/tmp`:

```text
gofmt -l (2 arquivos)                    -> vazio
go build ./internal/...                  -> exit 0
go vet ./internal/daemon/...             -> exit 0
go test ./internal/daemon -run 'ValidateThinking|GatewayThinkingLevels'        -> ok, 7/7
go test ./internal/daemon (pacote completo)                                   -> ok 46.937s
go test ./internal/daemon -run 'ValidateThinking|GatewayThinkingLevels' -race -> ok 1.039s
git diff --check                         -> exit 0     worktree limpo
```

## 5. ETA

| etapa | estimativa | observação |
|---|---|---|
| descarte do dirty (passo 1) | **1 min** | um comando do dono da árvore |
| peer review (2) | **20–30 min** | 2 arquivos, +15/-6 no último commit; os 3 somam ~200 linhas com testes |
| rebase + gates (3-4) | **15 min** se a base não mudou; **35 min** se houver conflito em `brain_integration.go` | o pacote `internal/daemon` completo leva ~47 s por execução |
| merge (5) | **5 min** | decisão do GTL |
| build + restart (6) | **não estimo** | depende do pipeline de imagem e da janela do owner |
| verificação pós-deploy (7) | **10 min** | 1 agente, 1 task, reversão |

Total sob meu controle: **~50 min** no caminho sem conflito, excluindo a janela de deploy.

## 6. Note para o Kanban (idempotente, sem disparar agente)

Marcador: `REASONING-GATEWAY-NOTE-V1:f5660e9`

```text
/note [REASONING-GATEWAY-NOTE-V1:f5660e9] Agent reasoning level is no longer refused wholesale under
gateway-required admission. Before, any agent configured with a reasoning level failed task admission
with thinking_not_approved, so the only dispatchable configuration was no level at all. The persisted
level is now validated against the per-provider allowlist for the accepted gateway frontends, and only
after the model advertises reasoning when that capability signal is authoritative. Unknown levels,
values that differ only by surrounding whitespace, providers without a reasoning contract and CLIs
without a gateway mapping all fail closed. Empty level keeps meaning "runtime default" and behaves
exactly as today, so deployments with no configured level see no change. Antigravity behaviour is
unchanged: the CLI accepts an effort flag but the product does not forward one, so the API keeps
rejecting the field; wiring it is a separate scoped change. Local commits 6bf0090, d9d569d and f5660e9,
two files, tests included, no schema or data change; not pushed, not merged, not deployed. Follow-ups:
export the provider level enum from pkg/agent, make the dev models projection derive reasoning
truthfully, and per-task reasoning telemetry stays with the database lane. Idempotency: skip if a
comment already contains the marker above.
```

## 7. Não-alegações

- Não toquei o arquivo sujo do repo principal, não fiz push, merge, PR, deploy ou mutação de board.
- Não alterei comportamento do runtime Antigravity nem `pkg/agent`; só corrigi a **justificativa**
  escrita no meu arquivo e nos meus testes.
- Não rebasei ainda: a base segue `0cb8aeb`. Se a integração avançou, o passo 3 é obrigatório antes do
  merge.
- Não verifiquei em produção que o efeito descrito na matriz ocorre: o deploy não aconteceu, e a
  coluna "depois" é derivada do código e dos testes, não de execução em produção.
- `task_usage.thinking_level` continua ausente e fora deste pacote.
- Nenhum segredo lido; nenhuma credencial tocada.
