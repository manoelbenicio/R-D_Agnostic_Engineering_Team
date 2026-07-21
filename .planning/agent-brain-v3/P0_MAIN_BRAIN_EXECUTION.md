# P0 MAIN BRAIN — Plano executivo final

updated: 2026-07-21
owner: Product Owner
source_of_truth: `openspec/changes/build-omniroute-agent-brain/` + `.planning/agent-brain-v3/`

## 1. Objetivo único

Concluir todas as integrações e features pendentes do **Main Brain** necessárias para o fluxo produtivo:

**squad → project → Kanban task → Agent Brain → CLI/model via OmniRoute → resultado terminal**

O Main Brain possui somente o cold/control plane: tarefa, workspace, processo, admissão baseada em readiness opaca, launch/cancel, eventos, persistência terminal e entrega do resultado.

O **OmniRoute é o proprietário exclusivo** de toda autenticação de inferência, credencial, conta, limite de token, janela de uso (inclusive limite de 5 horas), expiração, refresh, revogação, quota, 401/403, 429/5xx, circuit breaker, retry, seleção/rotação de conta e fallback. O Main Brain não implementa, não corrige, não simula e não recertifica esses comportamentos.

Prodex não participa desta fase. Permanece **Phase 3 / HOLD**, default-OFF, para uma decisão futura de recuperação fria; não é fallback atual, não é caminho por request e não consome lane P0.

## 2. Escopo OpenSpec deduplicado

As únicas tasks abertas que consomem execução P0 do Main Brain são:

- `5.6` — Cline → GLM-5.2 pelo contrato OmniRoute aprovado;
- `5.7` — Cline → Kimi-K2.7 pelo contrato OmniRoute aprovado;
- `5.8` — concluir Kiro/Opus48 e apenas reutilizar a evidência válida de Antigravity;
- `8.1` — comprovar protocolo/modelo somente nas rotas alteradas ou ainda sem evidência equivalente;
- `8.2` — comprovar tools, reasoning, cancellation, usage, terminal result e erro determinístico somente nas rotas afetadas.

`5.6–5.8` e `8.1–8.2` não são cinco campanhas independentes. Uma execução real de uma rota após integração pode fechar simultaneamente todos os requisitos sobrepostos que ela provar.

As tasks `8.5–8.7` não pertencem ao Main Brain: são certificação interna do OmniRoute para credenciais, contas, quota, retry/failover e lifecycle do router. Permanecem externas ao P0 do Main Brain e não geram implementação, QA ou live test no Multica.

### 2.1 ETA por task com 6–10 agentes contínuos

Premissas: agentes disponíveis 24x7 eliminam espera de turno, mas não reduzem linearmente trabalho em hotspots. `daemon.go`, config, health, entrypoints, integração `W1` e o ambiente live permanecem seriais. Agent-hours medem esforço agregado; **ETA wall-clock** mede tempo decorrido e já considera paralelismo. Os relógios abaixo começam com os `RouteModel` exatos publicados pelo OmniRoute, CLIs disponíveis e ambiente de integração operacional; ausência de qualquer um é blocker externo, não trabalho adicional do Main Brain.

| Task | Entrega exata | Agent-hours | ETA wall-clock com 6–10 agentes | Dependência/overlap |
|---|---|---:|---:|---|
| `5.6` | Base Cline compartilhada: `CLIKind`/executável, adapter OpenAI Chat, materialização do `providers.json`, reconciliação do ID GLM e uma Kanban task Cline→GLM terminal | 4–6h | **2.5–4h** | Carrega uma única vez a base reutilizada por `5.7`; exige ID GLM exato no registry |
| `5.7` | Delta Kimi sobre a mesma base Cline: ID exato, seleção do modelo e uma Kanban task Cline→Kimi terminal | 1.5–3h | **1–2h incremental** | Preparação do ID/teste pode ocorrer em paralelo; fechamento depende da integração da base de `5.6` |
| `5.8` | Opus48 via frontend Anthropic aceito + ID exato OmniRoute; comparação de build/config/hashes de Antigravity e reutilização sem rerun quando equivalente | 3–5h | **2–4h** | Bloqueia sem ID Opus48 aprovado; Antigravity acrescenta 0 live runs se equivalente |
| `8.1` | Confirmar exact model/protocol/availability no registry atual e ligar a evidência G4 de protocolo às mesmas execuções de `5.6–5.8` | 1–2h | **≤1h incremental** | Não é campanha separada; ocorre durante preflight/integração/registro das três rotas |
| `8.2` | Capturar tools, reasoning, usage, erro/resultado terminal e cancellation/cleanup somente onde a evidência existente não for equivalente | 2–4h | **0.5–1.5h incremental** | Reusa as execuções de `5.6–5.8`; cancellation adicional somente se o lifecycle Brain mudou ou a prova estiver ausente/stale |

As linhas não são somadas como cinco projetos. A base Cline é implementada uma vez; `8.1` e `8.2` são gates sobrepostos. Esforço de suporte executado em paralelo: production-integrity residual **4–8 agent-hours / 2–4h wall**, gap/fix de lifecycle Main Brain **3–8 agent-hours / 2–5h wall** conforme a gap matrix, e integração `W1` + ambiente live **3–5h wall serial**, iniciando assim que a primeira lane estiver pronta.

### 2.2 Alocação útil dos 6–10 agentes

| Agente/lane | Responsabilidade disjunta |
|---|---|
| A1 | contrato/config/home Cline compartilhado |
| A2 | `CLIKind`, executável e mapping Cline; entrega mudanças de hotspot para W1 |
| A3 | IDs exatos e catálogo/registry GLM + Kimi; resolve aliases sem inventar model ID |
| A4 | rota Opus48 via frontend Anthropic aceito |
| A5 | equivalência de evidência Antigravity, sem rerun por padrão |
| A6 | trace/gaps Brain-owned Kanban→launch→terminal/cancel/cleanup |
| A7 | production integrity frontend/mobile/desktop residual |
| A8 | production integrity backend/config/deploy residual |
| A9 | testes focused e checks de build somente dos deltas produzidos |
| A10 | OpenSpec/evidence mapping e preparação de integração, sem segunda QA |

Com 6 agentes, A7–A10 são combinados por prioridade; com 7–10, permanecem separados. Agente livre não cria regression, reviewer ou live run duplicado: ajuda uma lane disjunta ou fica disponível para blocker real. W1 continua único editor/integrador dos hotspots compartilhados.

### 2.3 Critical path e ETA total

```text
T0–1h      congelar IDs/ownership + gap matrix curta
T0.5–4h    Cline, Opus48, lifecycle e production integrity em paralelo
T2.5–7h    W1 integra lanes prontas uma por vez e executa checks focused
T4–10h     uma execução live por rota alterada + fechamento sobreposto 5.x/8.x
```

**ETA total:** 6–10 horas wall-clock nominal; 12–16 horas conservadoras se surgirem gaps reais no lifecycle/production integrity. Não há ETA interno enquanto faltar `RouteModel` aprovado ou disponibilidade do OmniRoute/CLI: isso é blocker externo explícito. Operação 24x7 remove pausas de calendário, mas não elimina o critical path serial nem autoriza validação duplicada.

## 3. O que será feito

### Fase A — Gap matrix real, sem retestar

Comparar implementação atual, TO-BE, OpenSpec e evidência existente e classificar cada item em apenas uma categoria:

1. **já implementado + evidência equivalente** — fechar/reutilizar, sem execução;
2. **implementado sem evidência equivalente** — executar uma vez;
3. **gap real do Main Brain** — implementar e validar;
4. **responsabilidade OmniRoute** — retirar da lane e referenciar ownership externo;
5. **Phase 3/HOLD** — Prodex/recovery, sem trabalho agora.

A análise deve cobrir o caminho real Kanban→terminal e as três famílias de rota pendentes; não deve abrir uma nova auditoria de autenticação, credenciais ou failover.

### Fase B — Implementação paralela com ownership disjunto

| Lane | Trabalho permitido | Proibido |
|---|---|---|
| `C` Main Brain core | gaps reais em task admission, workspace, launch, cancel, event/result lifecycle, terminal persistence e cleanup | autenticação, credencial, account state, quota, retry/failover |
| `R1` Cline routes | Cline→GLM-5.2 e Cline→Kimi-K2.7; protocolo/model intent e integração CLI necessária | provider login, key management, account selection, NVIDIA/Kimi fallback logic |
| `R2` Kiro/Agy | gap Kiro→Opus48; preservar Antigravity já comprovado | recertificar Antigravity inteiro ou criar novo credential owner |
| `P` Production integrity | remover mocks, fake success, QA routes, placeholders e persistência demo ainda alcançáveis | remover guardrails ou fixtures/testes isolados sem caller produtivo |
| `W1` Integrator | integrar hotspots serialmente e resolver conflito sem duplicar lógica | implementar router, auth ou failover no Brain |

Arquivos compartilhados de daemon/config/health/entrypoints pertencem exclusivamente a `W1`. Nenhuma lane escreve concorrentemente no mesmo arquivo.

### Fase C — Integração serial

`W1` integra uma lane por vez:

1. Production integrity;
2. Main Brain core;
3. Cline routes;
4. Kiro/Agy.

Depois de cada integração, roda somente build/typecheck/test focused exigido pelo delta. Se um conflito não muda semântica, não cria nova campanha. Se muda, repete apenas o cenário diretamente afetado.

### Fase D — Uma aceitação real, sem QA duplicada

A aceitação é o próprio uso real pós-integração:

1. criar/usar squad e project;
2. criar Kanban task;
3. atribuir agente/rota afetada;
4. observar launch pelo Agent Brain;
5. quando aplicável, cancelar e confirmar encerramento/cleanup;
6. confirmar resultado terminal persistido e entregue à UI.

Executar uma vez por família de rota **somente se houve alteração ou falta evidência equivalente**. O mesmo resultado fecha `5.x`, `8.1` e `8.2` sobrepostos. Não existem QA-A, QA-B, QA-C, broad regression final, segunda live acceptance ou failure-injection de OmniRoute pelo time Multica.

## 4. Critérios de conclusão

P0 termina quando:

- o fluxo squad/project/Kanban→Agent Brain→OmniRoute→terminal funciona nas rotas realmente pendentes;
- gaps reais de lifecycle Main Brain estão implementados;
- produção não expõe mocks, placeholders, fake-success, QA routes ou demo persistence nos caminhos alterados;
- cada delta passou no teste/build/typecheck mínimo relevante;
- cada rota alterada ou sem prova recebeu no máximo uma execução real de aceitação;
- evidência equivalente foi reutilizada para comportamento não alterado;
- nenhuma lógica de autenticação, credencial, quota, account selection, retry ou failover foi criada/testada no Main Brain;
- Prodex permaneceu intocado em Phase 3/HOLD;
- OpenSpec/GSD refletem o estado real e os checkboxes só são fechados por implementação/evidência concreta.

## 5. Exclusões duras

Não executar nesta fase:

- auth/credential/token/quota/refresh/revocation/401/403/429/5xx/failover tests;
- account add/remove/quarantine/re-entry do OmniRoute;
- circuit breaker ou retry interno do OmniRoute;
- Prodex integration, recovery ou parity;
- OBS-1..OBS-11, capacity tiers 20/50/100, default cutover ou debranding;
- chat, salvo se a gap matrix provar que ele bloqueia diretamente o fluxo Kanban→terminal;
- qualquer repetição criada apenas para preencher lane ou renomear a mesma validação.

## 6. Ordem imediata e ETA

1. congelar ownership e corrigir OpenSpec/GSD;
2. produzir a gap matrix curta do caminho Main Brain;
3. concluir production-integrity residual;
4. implementar somente gaps confirmados em `C`, `R1` e `R2`;
5. integrar serialmente em `W1`;
6. executar a aceitação real mínima das rotas afetadas;
7. atualizar evidência/checklists e encerrar P0.

ETA consolidado para 6–10 agentes contínuos: **6–10 horas wall-clock nominais; 12–16 horas conservadoras**, conforme a tabela da seção 2.1 e condicionado à disponibilidade dos `RouteModel` exatos, OmniRoute e CLIs.
