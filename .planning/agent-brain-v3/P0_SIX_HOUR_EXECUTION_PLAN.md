# Main Brain P0 — plano de execução de seis horas (8–10 agentes)

generated_utc: 2026-07-22T03:46:00Z  
window: 2026-07-22T03:46:00Z → 2026-07-22T09:46:00Z  
planning_owner: Principal Orchestrator  
execution_manager: Opus48-Kiro (`w5:p1`)  
active_change: `build-omniroute-agent-brain`  
schema: `spec-driven`  
source_of_truth: OpenSpec ativo + `.deploy-control/p0` + estado Herdr local

## 1. Objetivo e definição de pronto

Objetivo funcional:

```text
squad → project → task Kanban → Main Brain → OmniRoute → CLI/model
      → resultado persistido no backend/Postgres → terminal/logs/status na UI
```

Este pacote coordena engenharia, testes offline, correlação metadata-only e preparação de aceite. Ele **não autoriza nem executa deploy, restart, Docker/systemd, leitura de segredo ou inferência**. As tasks OpenSpec `6.3`, `6.4` e `6.5` permanecem condicionadas aos gates explícitos de capacidade/live-run em `.deploy-control/p0/control.json`.

## 2. Autoridade e reconciliação

Ordem de precedência desta janela:

1. instruções do owner nesta sessão;
2. `openspec/changes/build-omniroute-agent-brain/{proposal,design,specs,tasks}.md` atuais;
3. `.deploy-control/p0/PROTOCOL.md` e `control.json`;
4. este pacote de seis horas;
5. GSD v3 para traceability/evidence/ownership que não contradiga 1–4;
6. históricos anteriores apenas como evidência, nunca como fila ativa.

O dashboard legado que ainda apresenta tarefas antigas não é fonte de dispatch. Este pacote não reabre planos antigos nem cria lanes para eles.

## 3. Princípios oficiais incorporados

Fontes oficiais consultadas:

- Anthropic Engineering, `Building effective agents`: https://www.anthropic.com/engineering/building-effective-agents
- Anthropic, `Prompting best practices`: https://docs.anthropic.com/en/docs/build-with-claude/prompt-engineering/claude-prompting-best-practices
- OpenAI, `Prompt engineering`: https://developers.openai.com/api/docs/guides/prompt-engineering
- OpenAI, `Orchestration and handoffs`: https://developers.openai.com/api/docs/guides/agents/orchestration
- OpenAI, `Working with evals`: https://developers.openai.com/api/docs/guides/evals

Aplicação prática:

- usar um manager estável e especialistas estreitos;
- paralelizar somente subtarefas independentes;
- manter ownership do resultado no manager, não fazer handoffs de autoridade;
- dar a cada lane papel, contexto, arquivos, restrições, etapas e critérios de aceite explícitos;
- usar estado estruturado e check-ins persistidos;
- avaliar por testes/graders objetivos, não por autoafirmação;
- separar produtor, verificador e adjudicador quando o risco justificar;
- integrar arquivos compartilhados serialmente;
- preferir soluções simples, mínimas e verificáveis;
- nunca otimizar apenas para passar testes nem hard-codear fixtures;
- investigar arquivos antes de fazer afirmações sobre o código.

## 4. Capacidade viva verificada

Snapshot `2026-07-22T03:46:32Z`:

| Papel | Pane | Estado no snapshot | Uso nesta janela |
|---|---|---:|---|
| Principal/adjudicador | `w5:p9` | working | management-only |
| Opus48-Kiro manager | `w5:p1` | idle | dispatch, monitor, quality gate |
| Worker | `w6:p1` | idle | lane L1 |
| Worker | `w6:p2` | idle | lane L2 |
| Worker | `w7:p3` | idle | lane L3 |
| Worker | `w7:p4` | idle | lane L4 |
| Worker | `w8:p1` | idle | lane L5 |
| Worker | `w8:p2` | idle | lane L6 |
| Agy worker | `wB:p2` | idle | lane L8 verifier |
| Agy worker | `wB:p1` | blocked/standing by despite detector `working` | lane L7 após checkout/reassignment formal |

Total detectado: 10 agentes. Alocação pretendida: 1 Principal + 1 manager + 8 workers. O manager deve rodar `herdr agent list` imediatamente antes de cada wave e ajustar apenas para panes realmente `idle|done`; IDs não podem ser presumidos após recriação/movimento.

## 5. Restrições globais

1. Nenhum deploy, restart, Docker, systemd, push, commit ou merge.
2. Nenhuma inferência enquanto `live_runs.<family>.authorized=false`.
3. Nenhuma leitura, impressão, cópia, hash ou persistência de credenciais.
4. Nenhum reset, stash, revert, clean ou descarte da árvore suja.
5. Nenhum clone/download/worktree pesado; disco local estava crítico.
6. Nenhum agente edita OpenSpec/GSD; somente Principal/Kiro atualiza documentos autoritativos.
7. Um arquivo tem exatamente um owner durante uma wave.
8. Arquivo compartilhado ou disputado escala para L1 e é integrado serialmente.
9. A lane verificadora não corrige código; devolve finding ao owner.
10. Sem QA duplicada, broad regression redundante ou segunda live acceptance.
11. Todo agente faz check-in antes de editar e checkout somente com evidência.
12. Heartbeat em até 10 minutos e a cada mudança material.

## 6. Matriz de lanes e ownership congelado

| Lane | Papel | Ownership exclusivo | OpenSpec/AB-REQ | Entrega Wave A | ETA |
|---|---|---|---|---|---:|
| L1 | Lead Integrator | `server/internal/daemon/{daemon,config,health,brain_integration,types,wakeup}.go`, `brain/**`, `execenv/**`, `server/cmd/multica/cmd_daemon.go`, `server/pkg/agent/models.go`, `server/go.mod` | 5.1–5.3; AB-REQ-01..06,16–22,31 | hotspot central formatado/compilável; fallout resolvido | 75–105 min |
| L2 | Gateway/readiness | `server/internal/daemon/gateway/**` | 5.1–5.2,6.1; AB-REQ-07..13,33 | testes de registry/readiness/protocol; evidência sem inferência | 45–70 min |
| L3 | Runtime environment | `server/internal/daemon/runtimeenv/**` | 5.1–5.2; AB-REQ-16..22 | GLM/Kimi Cline contract, sanitizer e fail-closed tests | 50–75 min |
| L4 | CLI adapters | `server/pkg/agent/{claude,codex,kimi,nim,antigravity}.go` e testes correspondentes; **exclui `models.go`** | 5.1–5.2; AB-REQ-07,19,21 | adapters e argv/log safety; findings para L1 se modelo compartilhado | 50–75 min |
| L5 | E2E correlation core | `server/internal/daemon/observability/e2e/**` | 6.2; AB-REQ-39,40 | schema/version/join/assembler/leak scan API e testes | 60–90 min |
| L6 | Ingress/queue/persist helpers | somente `server/internal/middleware/obs_ingress*.go`, `server/internal/service/obs_queue*.go`, `server/internal/service/obs_persist*.go` | 6.2; AB-REQ-39,40 | helpers metadata-only e testes; sem anchors compartilhados | 55–85 min |
| L7 | WS/UI delivery + Kanban UI | somente `server/internal/daemonws/obs_delivery*.go` e `packages/views/issues/components/{board-view,issue-detail,execution-log-section}*` | 6.2; AB-REQ-02,39,40 | helper WS + testes/contrato UI; sem backend anchor | 55–85 min |
| L8 | Verificador independente | read-only no código; escreve somente handoff/evidence namespaced | 5.1–5.5,6.1 | baseline, zero-overlap, targeted/full build quando toolchain existir, residual scan | 60–100 min |

Paths são relativos a `multica-auth-work/`, exceto quando indicado `server/`. L5 publica o contrato antes de L6/L7 finalizarem. L6/L7 podem fazer preflight e testes de contrato em paralelo, mas não devem inventar a API L5.

## 7. DAG de seis horas

```text
T+00  fleet refresh + check-ins + zero-overlap
       ├─ L1 central
       ├─ L2 gateway
       ├─ L3 runtimeenv
       ├─ L4 adapters
       ├─ L5 correlation API (publica primeiro)
       ├─ L6 helper prep (aguarda contrato L5 para finalizar)
       ├─ L7 helper/UI prep (aguarda contrato L5 para finalizar)
       └─ L8 baseline verifier

T+90  Wave B
       L5 contract frozen
       ├─ L6/L7 finalize against L5
       ├─ L1 receives only bounded handoffs from L2–L7
       └─ L8 runs package checks and returns findings to owners

T+180 Wave C — serial convergence
       L1 integrates shared anchors/file disputes one at a time
       owners fix findings in own files
       L8 reruns targeted + server-wide checks

T+300 Wave D — release-readiness package
       OpenSpec strict + residual scan + traceability
       no deploy/live run
       manager produces acceptance-ready handoff and blocker list

T+360 STOP
       all lanes checked out or concretely blocked
       Principal adjudicates evidence; no overclaim
```

## 8. Cronograma e checkpoints

| Tempo | Gate | Responsável | Saída obrigatória |
|---:|---|---|---|
| T+00–15 | Preflight | manager + L1–L8 | agent list atual, check-ins, exact locks, toolchains |
| T+15–45 | First material result | todas as lanes | heartbeat ≥20%, blocker concreto ou teste inicial |
| T+45–90 | Wave A completion | owners | handoff + focused validation |
| T+90–105 | Reallocation | manager | lanes concluídas recebem finding real ou ficam standby; nunca busywork |
| T+105–180 | Wave B | L1/L5/L6/L7/L8 | API L5 frozen; helpers finalizados; targeted tests |
| T+180–240 | Serial integration | L1 | anchors/disputas integrados sequencialmente |
| T+240–300 | Evaluator loop | L8→owners→L8 | findings corrigidos e reavaliados, máximo 2 loops |
| T+300–345 | Final verification | L8 + manager | build/test/diff/OpenSpec/residual report |
| T+345–360 | Checkout/adjudication | todos + Principal | immutable checkouts e consolidated status |

## 9. Acceptance por lane

### L1

- `gofmt`/equivalente limpo nos arquivos owned;
- nenhum símbolo órfão de packages removidos;
- admission fail-closed antes de CLI sem plano válido;
- lifecycle/workspace/cancel/terminal preservados;
- nenhum fallback/router alternativo;
- focused tests + build package quando toolchain disponível.

### L2

- readiness distingue liveness/catalog/model/protocol;
- selected-model capability rejeita unsupported fields;
- `/v1/models` não é tratado como prova de inference;
- nenhum credential/account lifecycle implementado no Brain;
- testes gateway determinísticos verdes.

### L3

- Cline OpenAI-compatible aceita GLM/Kimi apenas por contrato explícito;
- native unsupported continua fail-closed;
- provider credentials/direct endpoints removidos do child env;
- trusted gateway config aplicada por último;
- testes runtimeenv verdes.

### L4

- argv/log shape redigido estruturalmente;
- configs Claude/Codex/Cline preservam o protocolo aprovado;
- nenhum auth file ou provider key copiado;
- nenhuma edição em `models.go` sem handoff para L1;
- testes adapters verdes.

### L5

- schema versionado com IDs dos oito hops;
- join relationships determinísticos;
- assembler detecta gaps/orphans;
- leak scan estrutural fail-closed;
- conteúdo, prompts, repo/tool payloads e secrets impossíveis no tipo público.

### L6

- helper ingress sem bodies;
- queue/persist sem task/result content;
- IDs determinísticos e tipos L5;
- não edita shared anchors (`task.go`, router/metrics chain).

### L7

- helper WS sem delivered payload;
- UI preserva status/log/result sem inventar fallback;
- não edita backend/daemon anchors;
- evidencia estados loading/error/terminal quando tests existentes permitirem.

### L8

- prova mecânica de interseção zero;
- comandos, exit codes e toolchain exatos;
- `git diff --check`, OpenSpec strict e residual scans;
- targeted/full checks sem corrigir código;
- findings atribuídos ao owner correto;
- classificação `verified|blocked`, nunca `DONE` sem prova.

## 10. Qualidade: evaluator–optimizer

L8 é evaluator; L1–L7 são producers. Fluxo por finding:

1. L8 publica finding com arquivo, linha/símbolo, comando e resultado esperado.
2. Manager roteia somente ao owner do arquivo.
3. Owner corrige minimalmente e atualiza heartbeat.
4. L8 reexecuta o critério específico.
5. Máximo de dois loops. Persistindo falha, status `BLOCKED` com causa e owner — nunca workaround/hard-code.
6. Principal adjudica somente a partir de evidência de disco.

## 11. Mapeamento das dez tasks OpenSpec abertas

| OpenSpec | Lane(s) | Resultado possível nesta janela | ETA crítica |
|---|---|---|---:|
| 5.1 gofmt | L1–L7; L8 verifica | fechável offline | 20–40 min paralelos |
| 5.2 targeted tests | L1–L4; L8 verifica | fechável se toolchain disponível | 45–75 min |
| 5.3 server-wide build/test | L1 + L8 | fechável se toolchain atual validar árvore | 45–90 min após Wave A |
| 5.4 OpenSpec strict | L8 | já observado PASS; registrar prova atual | 5–10 min |
| 5.5 residual scan | L8 | fechável offline | 15–25 min |
| 6.1 immutable revision/readiness | L2 + L8 | evidence-only, sem inference | 20–35 min |
| 6.2 eight-hop correlation | L5–L7 + L1 serial | implementação/testes offline possíveis | 2h30–4h |
| 6.3 tier 20 | nenhuma lane executa sem gate | preparar checklist; run permanece bloqueado | externo |
| 6.4 tier 50/100 | não iniciar antes de 6.3 | bloqueado por design | externo |
| 6.5 Kanban live acceptance | evidence lanes retomam somente com token | bloqueado até autorização explícita | externo |

## 12. Estado final esperado em seis horas

Entregável mínimo:

- 8 lanes ocupadas com trabalho real e ownership disjunto;
- source convergence e tests/build executados onde houver toolchain;
- todo failure concreto atribuído e reavaliado;
- OpenSpec strict/residual evidence atualizada;
- correlação E2E implementada ou bloqueada com gap preciso;
- pacote pronto para o operador executar deploy/aceite depois;
- nenhum segredo, deploy, inference ou claim não provado;
- todos os agentes com check-in, heartbeats e checkout/block em disco.

Se uma tarefa serial exceder a janela, o manager não cria trabalho artificial: preserva a fila, registra o critical path e mantém workers concluídos em standby.
