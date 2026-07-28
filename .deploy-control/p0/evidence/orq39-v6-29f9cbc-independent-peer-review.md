# ORQ-39 — peer review independente do commit `29f9cbc` (READ-ONLY)

- Revisor: Codex56#A (`w7:p3`) · UTC 2026-07-28T14:33Z
- Alvo: `29f9cbc8b124c825cadf1c5d5d028de55c6bb33d` — *"test: add ORQ-39 browser QA gate"*, autor
  `kiro-lead`, 2026-07-28 13:47:05, sobre `0cb8aeb`; worktree canônico
  `/home/ec2-user/workspace/worktrees/gtl-orq39-v6-browser-qa`, branch `ci/orq39-browser-qa`
- Escopo: **7 arquivos, +1307/-0**, exatamente como declarado — 1 workflow na raiz e 6 specs
- Modo: READ-ONLY. Não editei nada do autor, não commitei, não fiz push, não instalei dependência,
  não executei Playwright, não subi banco, não toquei board.

## VEREDITO: **PASS com 3 amendas** (nenhuma bloqueante)

Este é o V6 e ele **fecha todos os bloqueios que eu havia levantado no V4**. Verifiquei um por um, e
também procurei defeitos novos nas seis specs, no anti-falso-verde e na compatibilidade com o produto.

## 1. Os quatro BLOCKs do V4 estão resolvidos, item por item

| BLOCK do V4 | estado no V6 | evidência |
|---|---|---|
| guarda de segredos com caminho errado | **resolvido** | `:80-93` roda com `working-directory: ${{ github.workspace }}` e itera os quatro caminhos `.env`, `.env.worktree`, `multica-auth-work/.env`, `multica-auth-work/.env.worktree`, mais uma varredura de arquivo credencial; aborta com `FATAL` |
| `services.postgres` sem health check | **resolvido** | `:39-43` `options: --health-cmd "pg_isready -U multica -d multica_test" --health-interval 5s --health-timeout 5s --health-retries 10` |
| serviços em background sem readiness nem propagação | **resolvido** | `wait_ready` com `kill -0` do PID e polling (`:194`, `:230`, `:243`), backend e web supervisionados, `FATAL: $label exited before browser QA completed` (`:218`), traps de INT/TERM |
| tags mutáveis apresentadas como pins imutáveis | **resolvido** | `:31` `mcr.microsoft.com/playwright@sha256:6446946a…`; `:34` `pgvector/pgvector@sha256:d2ef61f4…`; actions por SHA de 40 chars (`:47`, `:97`) |
| gatilho divergente do GTL-69R | **resolvido** | `on: push` restrito a `branches: [ci/orq39-browser-qa]` — exatamente o mecanismo ratificado |
| `setup-node`/`pnpm/action-setup` dentro de `container:` | **resolvido melhor do que eu pedi** | `:102-118` prova o Node imutável da imagem e ativa `corepack prepare pnpm@10.28.2 --activate`, exigindo igualdade de versão; confere com `package.json:30` `"packageManager": "pnpm@10.28.2"` |
| permissões mínimas | **resolvido** | `contents: read`, `actions: none`, `id-token: none` |
| destruição do código de erro fixo | **resolvido** | `redact_log` aplicado aos três logs em falha, com substituição do `JWT_SECRET` por `[REDACTED]` (`:180`) |

## 2. Anti-falso-verde: é a parte mais forte do pacote

O script Node de `:288-353` não confia no exit code do Playwright. Ele exige, sobre o relatório JSON:

- cada um dos **seis** arquivos esperados aparece com ao menos uma spec executada
  (`no executed specs in report`);
- toda spec descoberta tem `tests` não vazio (`discovered spec has no tests`);
- `expectedStatus === "passed"` por teste;
- **exatamente um** resultado por teste (impede que retry mascare instabilidade);
- `results[0].status === "passed"`;
- `stats.expected > 0` e `stats.skipped`, `stats.unexpected`, `stats.flaky` **todos iguais a zero**;
- `report.errors` vazio;
- caminho de spec fora da lista esperada é erro (`unexpected or missing spec path`).

Somado a `--forbid-only` (`:252`) e à verificação `test -f "$spec"` de cada arquivo (`:167`), o gate
não passa por skip, por `.only`, por spec ausente, por spec vazia nem por flaky reclassificado.
Auditei as seis specs em busca de `test.skip`, `test.fixme`, `test.fail` e `expect.soft`: **nenhuma
ocorrência**. Os três `if (!fixture) return;` que aparecem no grep estão dentro de helpers de limpeza
(`removePanelFixture`, `removeChatFixture`, `removeReasoningFixture`), não em corpo de teste, portanto
não são retorno silencioso que produziria verde vazio.

## 3. Comportamento real das seis specs

| spec | testes | expects | fixture de banco | mocks de rota | navega |
|---|---|---|---|---|---|
| `board-kanban` | 1 | 4 | 1 | 0 | via fixture |
| `chat-panel-buttons` | 1 | 4 | 5 | 2 | via fixture |
| `chat-reasoning-level` | 1 | 6 | 5 | 1 | 1 |
| `chat-upload-ui` | 1 | 6 | 4 | 1 | via fixture |
| `delete-flows` | 1 | 6 | 4 | 0 | 1 |
| `squad-model-dropdown` | 1 | 6 | 4 | 1 | 1 |

Todas usam o fixture de autenticação real do repositório (`e2e/fixtures`), criam estado próprio via
`pg.Client` e limpam no teardown, e batem em endpoints reais (`api.*`, 4 a 9 chamadas por spec).
Portanto exercitam produto de verdade, não apenas DOM estático. Os mocks de `page.route` são pontuais
(1 a 2 por spec) e servem para fixar catálogos de modelo/thinking, não para substituir o fluxo sob
teste.

## 4. Compatibilidade com o produto: verificada, e corrijo meu próprio primeiro resultado

Extraí todos os nomes acessíveis usados pelas specs e conferi no produto. **Importante**: minha
primeira busca por literais em `packages/views` e `apps/web` retornou zero para quase todos, o que
sugeriria specs incompatíveis. Isso estava **errado** — a UI é i18n e as strings vivem no catálogo. Na
checagem correta, todos existem em `packages/views/locales/en/`:

```text
"Thinking · {{value}}"  agents.json:197      (thinking_tooltip)   -> "Thinking · Follow CLI config", "Thinking · High"
"Follow CLI config"     agents.json:196      (thinking_default)
"Row actions"           agents.json:82       (actions_aria)
"New Squad"             squads.json:4        (new_button)
"Create Squad"          modals.json:57       (title)
"Delete runtime"        runtimes.json:88     (delete_button)
"Delete issue"          issues.json:430      (delete_issue)
"Attach file"           ui.json:2            (attach_file)
"Chat history"          chat.json:82         (history_group)
"Create Agent"          agents.json:227      (title_create)
"Download"              editor.json:32       (download)
```

E o `aria-label` do gatilho de thinking é `triggerTitle = t($.pickers.thinking_tooltip, {value})`
(`thinking-picker.tsx:54,85`), que compõe exatamente o rótulo esperado pela spec. Compatibilidade:
**confirmada**.

## 5. Amendas (não bloqueiam)

**A1 — locale não está pinado, e as seis specs dependem de inglês.** `playwright.config.ts` não
declara `locale` em `use:` e o workflow não define `LANG`/locale. Hoje o Chromium do container roda
en-US e passa, mas qualquer mudança de detecção de idioma no produto (Accept-Language, cookie,
preferência default) quebra o gate por motivo linguístico, não por regressão. Fix de uma linha:
`use: { locale: "en-US" }` no projeto do gate, ou variável explícita no job.

**A2 — nome da spec de reasoning não corresponde à tela exercitada.** `chat-reasoning-level.spec.ts`
navega para `/{workspace}/agents/{agentId}` e valida o picker do **inspetor de agente**, não um
controle de reasoning dentro do chat. O comportamento testado é válido e existe; só o nome engana quem
for ler o relatório. Renomear para `agent-reasoning-level` (ou documentar) evita conclusão errada sobre
cobertura de chat.

**A3 — o mock de `/api/runtimes/{id}/models` limita o alcance.** Ao fixar `thinking.supported_levels`
no cliente, a spec prova a UI mas **não** cobre o contrato real do catálogo do servidor. Vale declarar
isso no card, para ninguém tratar este gate como validação do endpoint de modelos. Sugiro adicionar, em
onda futura, uma asserção mínima contra o endpoint real (por exemplo forma da resposta), separada da
spec de UI.

## 6. Observações menores

1. Existe um arquivo **não rastreado** no worktree do autor durante a minha revisão:
   `.deploy-control/p0/evidence/orq39-browser-qa-v6-correction.md`. Não é parte do commit revisado e eu
   não o toquei; vale commitar ou remover para o worktree ficar limpo antes da integração.
2. O commit contém **só código de gate** — nenhuma evidência/check-out misturada. É o padrão correto,
   e o oposto do que apontei no pacote ORQ-21 R2.
3. `JWT_SECRET: ephemeral-orq39-ci-only-secret-32-bytes` é literal de CI, não segredo real, e é
   redigido nos logs (`:180`). Correto, mas vale um comentário no YAML dizendo explicitamente que é
   descartável, para nenhum leitor futuro tentar "protegê-lo" via secrets do GitHub.
4. O `/healthz` usado pelo `wait_ready` **existe** no produto (`cmd/server/router.go:445`,
   `health.readyHandler`), distinto do `/health` liveness de `:443`. Conferi porque um nome errado
   tornaria o gate sempre vermelho.

## 7. Validações locais que executei, sem instalar nada

```text
python3 -c "yaml.safe_load(...)"                     -> YAML_OK, job único browser-qa
on:/permissions:                                     -> push em ci/orq39-browser-qa; contents:read, actions:none, id-token:none
git show 29f9cbc --check                             -> exit 0 (sem erro de whitespace)
git show --stat                                      -> 7 arquivos, +1307/-0
grep de test.skip|fixme|fail|expect.soft nas 6 specs -> nenhuma ocorrência
grep de rótulos × packages/views/locales/en/*.json   -> 11/11 presentes
router.go                                            -> /healthz existe (readyHandler)
package.json                                         -> packageManager pnpm@10.28.2 == versão exigida no job
```

Não executei `pnpm install`, `playwright test`, `go run ./cmd/migrate` nem subi Postgres: exigiria
instalar dependências e navegador, o que o dispatch proibiu.

## 8. Não-alegações

- Não executei o workflow nem as specs. A avaliação de comportamento vem de leitura do código das
  specs, do YAML e do produto, mais as verificações locais da §7.
- Não posso afirmar que o gate passa em execução real: dependências de rede, imagem e Postgres não
  foram exercitadas por mim.
- Não verifiquei os digests `sha256` contra o registry (exigiria acesso a registry); apenas confirmei
  que são digests e não tags.
- Não avaliei os specs pré-existentes do diretório `e2e/` fora dos seis do commit.
- Não editei, commitei, pushei ou toquei board; nenhum arquivo do autor foi alterado.
