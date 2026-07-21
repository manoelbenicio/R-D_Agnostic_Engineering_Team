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

ETA após remoção das lanes duplicadas de auth/failover: **6–12 horas nominais; até 16 horas conservadoras**, condicionado apenas a gaps reais encontrados e disponibilidade das rotas OmniRoute necessárias para a execução terminal.
