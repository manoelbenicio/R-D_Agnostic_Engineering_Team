# P0 MAIN BRAIN — Plano executivo agentic

updated: 2026-07-21T14:11Z  
owner: Product Owner  
principal_orchestrator: Codex#TL#Principal (`w5:p2`)  
co_orchestrator: Opus48-Kiro (`w5:p1`)  
source_of_truth: `openspec/changes/build-omniroute-agent-brain/` + `.planning/agent-brain-v3/`

## 1. Resultado P0 obrigatório

Entregar o app funcional no fluxo **squad → project → Kanban task → Agent Brain → CLI/model via OmniRoute → resultado terminal**, sem depender de chat, observabilidade E2E, certificação de capacidade, cutover default-on, Prodex recovery ou debranding.

O P0 pendente contém exatamente oito checkboxes OpenSpec, agrupados em quatro workstreams:

| Workstream | Tasks | Resultado obrigatório |
|---|---|---|
| Rotas funcionais | 5.6, 5.7, 5.8, 8.1, 8.2 | Squads executam tasks por todas as rotas P0 aprovadas, preservando protocolo, tools, reasoning, usage, cancelamento e erro. |
| Tratamento de falhas | 8.5 | Auth/quota/429/5xx/timeout/malformed produzem estado determinístico, recuperável e sem segredo. |
| Retry, dedup e cancelamento | 8.6 | Retry somente pre-commit; nenhum replay após output/tool; dedup e liberação de slots exatamente uma vez. |
| Lifecycle operacional | 8.7 | Add/remove/quarantine/re-entry e restart/rollback não corrompem tasks ativas. |

ETA consolidado: **24–48h nominal; 72h conservador**, sem somar tarefas sobrepostas.

## 2. Exclusões duras desta execução

Não consumir lanes P0 com: chat; OBS-1..OBS-11; 9.x/capacidade; 10.x/cutover; 1.4/Prodex; debranding; tiers 50/100; produção. OBS e chat são P2. 1.4/10.4 permanecem HOLD. Nenhum agente lê, imprime, copia, rotaciona ou altera segredo. Live non-prod só usa chave nova já injetada diretamente pelo dono no OmniRoute e nunca revela valor.

## 3. DAG e paralelismo

```text
Freeze OpenSpec/GSD + ownership
  ├─ R1 adapters/routes 5.6–5.8
  ├─ R2 protocol acceptance 8.1–8.2
  ├─ F  deterministic failures 8.5
  ├─ D  replay/dedup/cancel 8.6
  └─ L  lifecycle/restart/rollback 8.7
           ↓
W1 serial integration on protected P0 branch
           ↓
Independent review A + security review B
           ↓
Focused regression + live non-prod functional acceptance
           ↓
Evidence hashes/provenance + OpenSpec checkbox closure
```

Rotas, falhas, retry e lifecycle executam em paralelo em worktrees/branches isolados. Somente W1 integra hotspots compartilhados. Reviewer nunca é produtor nem adjudicador.

## 4. Roster e lanes

### Preflight ativo — 2026-07-21T14:11Z

| Pane | Lane read-only | Escopo |
|---|---|---|
| w6:p1 | route/source gap | 5.6/5.7/5.8/8.1 |
| w6:p2 | protocol/tools/live-route | 8.1/8.2 |
| w7:p3 | failure matrix | 8.5 |
| w7:p4 | retry/CommitLedger/no-replay | 8.6 |
| w8:p1 | lifecycle/persistence/active-load | 8.7 |
| w8:p2 | integration/harness | cross-workstream |
| wB:p1 | security/credentialless | 5.6/5.7/5.8/8.2 |
| wB:p2 | independent acceptance matrix | all P0 |

O Opus48-Kiro coordena esses oito workers e reporta somente ao Principal. O Principal mantém OpenSpec/GSD, ownership, adjudicação e integração gates.

### Ownership de produção após o preflight

| Lane | Ownership permitido | Proibido |
|---|---|---|
| W1 Integrator | `internal/daemon/{daemon,config,health,brain_integration}.go`, `internal/daemon/commitledger/**`, config/command hotspots e integração serial | editar módulos owned por W2/W3/W4 durante produção |
| W2 Gateway failures/retry | `internal/daemon/gateway/**` | daemon/config/runtime adapters |
| W3 Routes/runtime adapters | `internal/daemon/runtimeenv/**`, `pkg/agent/{claude,codex,kimi,nim,antigravity}.go` | gateway e hotspots centrais |
| W4 Lifecycle/harness/evidence | harness P0 isolado, runbook funcional, evidência namespaced por task | produto W1/W2/W3, OBS/capacidade |
| QA-A | read-only diff + focused tests | qualquer edição do producer |
| QA-B Security | read-only credentialless/no-secret review | segredo, auth mutation, edição do producer |

Arquivo disputado escala para W1 e é serializado. Não existem duas escritas concorrentes no mesmo arquivo ou worktree.

## 5. Prompt contract comum

Todo prompt de produção contém:

1. task IDs e requisito normativo exato;
2. branch/worktree e lista de arquivos permitidos;
3. lista explícita de must-not-touch;
4. baseline SHA e dependências aceitas;
5. testes focused obrigatórios (`go test`, `-race` quando toolchain permitir, `go vet`, `gofmt`, `git diff --check`);
6. evidência sem segredo, com comandos/saídas/hashes/proveniência;
7. proibição de checkbox, merge, push, live paid call ou broad test sem autorização;
8. check-in antes de editar e check-out com RESULT/FILES/TESTS/COMMIT/BLOCKERS;
9. parada fail-closed diante de design conflict, segredo ou ownership overlap.

## 6. Prompt packets de produção

### W3 — Rotas funcionais 5.6/5.7/5.8

Implementar apenas os gaps confirmados pelo preflight para Cline→Kimi-K2.7, Cline→GLM52 com fallback NVIDIA OmniRoute-owned, e Antigravity já operacional (revalidar, não reimplementar). Preservar `CLIKind`/`RouteModel`; nenhum provider key ou fallback decidido pelo Brain. Entregar testes focused por adapter e diff isolado.

### W2/W4 — Protocolos 8.1/8.2

Exercitar cada rota P0 aprovada em streaming/non-streaming, tools, reasoning, usage, cancellation e deterministic errors. Component evidence não substitui requisito live. Nenhum body/prompt/tool payload em evidência; registrar apenas metadados e hashes seguros.

### W2 — Falhas 8.5

Fechar a matriz: expired access; revoked refresh; quota; 401; 403; account 429; provider-global 429; 5xx; timeout; malformed upstream. Provar classificação, scope/circuit, fallback permitido, terminal state e zero retry indevido. Não fabricar falha com evidência que não atravessa o boundary sob teste.

### W1/W2 — Retry/dedup/cancel 8.6

Fechar durable CommitLedger/replay gate e integração produtiva: retry pre-first-output; no replay post-output/tool; dedup concurrent/completed; cancellation libera task/request/account slots exatamente uma vez; ambiguous output fail-closed. Store e HMAC/config devem estar efetivamente wired, não apenas unit-tested.

### W2/W4 — Lifecycle 8.7

Provar add/remove/quarantine/re-entry e restart/config rollback sob carga funcional ordinária. Estado deve persistir/reconciliar, streams ativas não podem corromper ou duplicar task, e rollback retorna à versão/config aceita. Não chamar isso de capacity certification.

### QA-A — revisão independente

Revisar o diff completo contra OpenSpec/design/AB-REQ; reproduzir focused tests; procurar código morto, fake harness, missing production caller, race, replay, leaks e overclaim. Veredito: ACCEPT ou FAIL com achados severidade/caminho/linha/reprodução.

### QA-B — segurança e credenciais

Provar credentialless child env/home/process tree, trusted config last, nenhum provider-native endpoint/key, nenhum segredo em logs/errors/evidence e fail-closed auth. Nunca acessar valor de segredo.

## 7. Gates de aceite

Uma task só recebe `[x]` quando todos forem verdadeiros:

- implementação está no caminho produtivo, não somente fixture/harness;
- testes focused passam e o reviewer independente reproduz;
- live non-prod exigido pela task foi executado quando autorizado e disponível;
- evidence artifact contém baseline/final SHA, comandos, resultados, hashes e identidades producer/reviewer;
- nenhuma violação de ownership, segredo, dual router ou escopo P2;
- W1 integrou serialmente no protected P0 branch e o conjunto integrado passou regressão;
- Principal adjudicou sem usar apenas a declaração do producer.

## 8. Monitoramento de 60 segundos

- Opus48-Kiro executa `herdr agent list` a cada 60s, lê panes `done|idle|blocked`, coleta resultado e redistribui imediatamente.
- Principal mantém um segundo monitor local independente de 60s.
- `idle|done` sem entrega aceita → nova tarefa P0 de revisão, teste focused ou gap audit.
- `blocked` → classificar em técnico, ownership, segurança ou decisão humana. Somente segurança irreversível, produção, segredo ou mudança arquitetural grave escala ao dono.
- Toda rodada registra timestamp, pane, lane, estado, deliverable e próxima ação; nenhum conteúdo sensível.

## 9. Integração e evidência

A branch protegida é `integration/agent-brain-p0`; `main` não é tocada. Antes de atualizar a branch: dry-run em worktree descartável, resolução arquivo-a-arquivo, sem `ours/theirs` em massa, sem force push, focused tests + race/vet/smoke/provenance. Commits são criados somente por workers autorizados; o Principal não commita nem edita código de produto.

OpenSpec e GSD permanecem abertos até evidência integrada. Checkbox nunca é usado como proxy de progresso.
