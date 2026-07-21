# P0 MAIN BRAIN — Plano executivo agentic

updated: 2026-07-21T21:20Z  
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

ETA consolidado após deduplicação: **6–12h nominal; até 16h conservador**. A estimativa cobre somente gaps reais de implementação/integração e uma execução live por comportamento alterado; não soma tarefas sobrepostas nem repete evidência equivalente já aceita.

## 2. Exclusões duras desta execução

Não consumir lanes P0 com: chat; OBS-1..OBS-11; 9.x/capacidade; 10.x/cutover; 1.4/Prodex recovery; debranding; tiers 50/100. OBS e chat são P2. 1.4/10.4 permanecem HOLD. Nenhum agente lê, imprime, copia, rotaciona ou altera segredo.

O ambiente ativo é a validação funcional. Produção-reachable mocks, placeholders, fake-success fallbacks, QA-only routes e dados sintéticos persistidos são proibidos. Fixtures/testes isolados e guardrails sem caller produtivo permanecem. Evidência aceita da mesma versão/digest, configuração, rota e cenário é reutilizada. Só se executa novamente comportamento alterado, sem evidência, com evidência obsoleta/não equivalente ou com risco material distinto ainda não coberto.

## 3. DAG e paralelismo

```text
Implementação paralela em ownership disjunto
  ├─ R1 Cline/Kimi/GLM: 5.6–5.7 + cobertura 8.1/8.2
  ├─ R2 Antigravity/Kiro: 5.8 + cobertura 8.1/8.2
  ├─ B  fronteira Brain: partes Brain-owned de 8.5/8.6
  └─ L  Kanban/lifecycle Brain-owned: parte de 8.7
           ↓
W1 integra uma lane por vez no ambiente ativo
           ↓
UMA execução live do comportamento alterado
           ↓
O mesmo resultado atualiza todas as tasks sobrepostas que ele prova
           ↓
Próxima lane; após a última, P0 termina
```

Implementação paraleliza; integração de hotspots permanece serial. Não existem QA-A/QA-B, regressão ampla separada, segunda live acceptance ou agente dedicado a evidência. Se W1 resolver conflito alterando comportamento, executa novamente somente o cenário afetado. Registro mínimo (task IDs, SHA/config, comando/ação e resultado) reutiliza a própria execução live e não constitui outro teste.

## 4. Roster e lanes

### Lanes autorizadas após o preflight

| Lane | Escopo único | Não repetir |
|---|---|---|
| R1 | Cline→Kimi-K2.7 e Cline→GLM-5.2; ajustes reais de adapter/config | internals de auth/rotação/fallback do OmniRoute |
| R2 | gap Kiro→Opus48 e preservação da rota Antigravity já operacional | reimplementação ou recertificação integral de Antigravity |
| B | fail-closed, propagação de erro e cancelamento pertencentes ao Main Brain | failure injection de provider já coberta pelo mesmo OmniRoute build/config |
| L | squad/project/Kanban→CLI→resultado terminal; cleanup de processo/slot Brain-owned | add/remove/quarantine internos do OmniRoute já comprovados |
| W1 | integração serial de hotspots e uma execução live por lane integrada | broad regression ou segunda acceptance |

O Opus48-Kiro coordena as quatro lanes e reporta ao Principal. O Principal mantém OpenSpec/GSD, ownership e integração. Não há lane QA ou reviewer que reproduza a execução.

### Ownership de produção

| Lane | Ownership permitido | Proibido |
|---|---|---|
| W1 Integrator | `internal/daemon/{daemon,config,health,brain_integration}.go`, config/command hotspots e integração serial | duplicar lógica hot-path do OmniRoute |
| R1/R2 Routes | `internal/daemon/runtimeenv/**`, adapters estritamente necessários | gateway internals, credenciais/provider account selection |
| B Boundary | somente call sites Brain-owned de readiness/error/cancel | refresh, quota, 429 circuit, account retry/fallback OmniRoute-owned |
| L Lifecycle | fluxo Kanban, processo CLI, persistência terminal e cleanup Brain-owned | estado interno de contas OmniRoute |

Arquivo disputado escala para W1 e é serializado. Não existem duas escritas concorrentes no mesmo arquivo ou worktree.

## 5. Prompt contract comum

Todo prompt de produção contém:

1. task IDs e requisito normativo exato;
2. branch/worktree e lista de arquivos permitidos;
3. lista explícita de must-not-touch;
4. baseline SHA e dependências aceitas;
5. validação proporcional ao delta: build/typecheck/lint ou teste focused somente para comportamento novo/alterado ou gap sem evidência; nenhuma broad regression automática;
6. reutilização explícita de evidência válida da mesma versão/config/rota/cenário; o registro da execução live contém somente task IDs, SHA/config segura, ação e resultado;
7. proibição de merge, push, mutação de segredo ou chamada paga fora da execução live autorizada;
8. check-in antes de editar e check-out com RESULT/FILES/VALIDATION/COMMIT/BLOCKERS;
9. parada fail-closed diante de design conflict, segredo ou ownership overlap.

## 6. Prompt packets de produção

### R1/R2 — Rotas funcionais 5.6/5.7/5.8 + 8.1/8.2

Implementar apenas gaps confirmados: Cline→Kimi-K2.7, Cline→GLM52, gap Kiro→Opus48; Antigravity é preservado, não reimplementado. Cada rota alterada recebe uma única execução live após integração W1; essa execução fecha simultaneamente a cobertura sobreposta de 5.x/8.1/8.2 que provar. Rotas já aceitas no mesmo build/config sem mudança reutilizam evidência.

### B — Fronteira Main Brain 8.5/8.6

OmniRoute continua proprietário exclusivo de credenciais provider, refresh, quota, seleção de conta, 429/5xx circuits e retry/fallback pre-commit. Não implementar CommitLedger de inferência ou account retry no Brain. Validar uma vez somente o delta Brain-owned: OmniRoute indisponível falha fechado sem provider direto; erro seguro vira estado correto; cancelamento encerra CLI e libera slot Brain-owned. Failure-injection interna do OmniRoute usa evidência equivalente existente, salvo mudança de build/config ou gap material não coberto.

### L — Lifecycle 8.7

Validar o fluxo real squad/project/Kanban→Brain→CLI→OmniRoute→resultado terminal e somente estado Brain-owned: admissão, workspace, processo, cancelamento, persistência terminal e cleanup. Add/remove/quarantine/re-entry de contas e rollback interno do OmniRoute não são reexecutados quando já comprovados no mesmo build/config.

### W1 — Integração e execução única

Integrar uma lane por vez. Se a integração não altera comportamento além do diff produzido, executar uma única vez o cenário real da lane e registrar o resultado para todos os checkboxes sobrepostos. Se houver conflito com alteração semântica, repetir apenas o cenário afetado. Após a última lane não existe fase adicional de QA, regressão ou acceptance.

## 7. Gates de aceite

Uma task só recebe `[x]` quando todos forem verdadeiros:

- implementação está no caminho produtivo, não somente fixture/harness;
- não há mock, placeholder, fake-success ou QA-only path alcançável no comportamento aceito;
- validação do delta novo/alterado passou, ou evidência aceita equivalente foi explicitamente reutilizada;
- a execução live única pós-integração cobriu o requisito quando o comportamento foi alterado ou ainda não tinha prova;
- registro mínimo contém task IDs, SHA/config segura, ação e resultado sem segredo;
- nenhuma violação de ownership, segredo, dual router ou escopo P2;
- W1 integrou serialmente no protected P0 branch;
- checkbox não exige uma segunda execução, reviewer ou relatório separado.

## 8. Monitoramento de 60 segundos

- Opus48-Kiro executa `herdr agent list` a cada 60s, lê panes `done|idle|blocked`, coleta resultado e redistribui imediatamente.
- Principal mantém um segundo monitor local independente de 60s.
- `idle|done` sem entrega aceita → nova tarefa de implementação/integração P0 não sobreposta; não criar revisão ou repetição para ocupar lane.
- `blocked` → classificar em técnico, ownership, segurança ou decisão humana. Somente segurança irreversível, segredo ou mudança arquitetural grave escala ao dono.
- Toda rodada registra timestamp, pane, lane, estado, deliverable e próxima ação; nenhum conteúdo sensível.

## 9. Integração e registro

A branch protegida é `integration/agent-brain-p0`; `main` não é tocada. W1 resolve arquivo-a-arquivo, sem `ours/theirs` em massa e sem force push. Validação é proporcional ao delta e ocorre uma vez no ambiente ativo após integrar a lane. Não existe broad regression final obrigatória; conflito que altere semântica exige somente o cenário afetado.

OpenSpec/GSD recebem o resultado da mesma execução live (task IDs, SHA/config segura, ação, resultado). Checkbox nunca é proxy de progresso nem motivo para repetir teste.
