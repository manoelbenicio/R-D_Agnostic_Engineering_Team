# Auditoria das 42 pendências

**Conclusão principal:** o número **42 não representa o que falta para colocar o Main Brain funcional**. Ele mistura P0 atual, observabilidade P2, capacidade futura, cutover, remoções pós-cutover e debranding.

A decomposição correta é:

| Horizonte | Quantidade |
|---|---:|
| Ativas agora: 8.5–8.7 | **3** |
| Necessárias para concluir o P0 funcional | **5** |
| Observabilidade P2 | **11** |
| HOLD explícito | **2** |
| Pré-cutover/GA, mas não P0 atual | **8** |
| Follow-up futuro/pós-cutover | **13** |
| **Total** | **42** |

## Sobre observabilidade

**Sim: observabilidade deve permanecer P2 neste momento.** Isso está explícito nas decisões D‑V3‑21/22/23:

1. Primeiro: Main Brain P0 funcional, integrado, testado e operando corretamente.
2. Depois: OBS-1–OBS-11.
3. OBS continua obrigatória antes de certificação de capacidade, default-on e cutover.

Portanto, OBS não deve consumir lanes agora e seus 11 itens não deveriam aparecer como “pendência operacional atual”.

## Legenda

- **P0-NOW:** crítico e ativo imediatamente.
- **P0-EXIT:** necessário para declarar Main Brain P0 100% funcional.
- **P2-CUT:** não executar agora; necessário antes de capacidade/cutover.
- **HOLD:** proibido despachar até nova autorização.
- **FOLLOW-UP:** não pertence ao deploy atual.
- **REWRITE:** tarefa válida em essência, mas texto está velho, duplicado ou conflitante.
- Os ETAs são **esforço ativo depois dos pré-requisitos**, não promessa de calendário.

---

## 1. Arquitetura e Main Brain funcional — 10 itens

| ID | O que é / importância | Criticidade e auditoria | ETA |
|---|---|---|---:|
| **1.3** | Checklist OmniRoute: digest, protocolos, falhas, segurança, capacidade. Importante para adjudicação final. | **REWRITE / pré-cutover.** A classificação documental já existe: 116 linhas, sendo 8 suportadas, 84 parciais e 24 não suportadas. Duplica parcialmente 8.8. Deve virar “resolver blockers e pin do digest”, não “obter checklist”. | 4–8h; dependências externas TBD |
| **1.4** | Paridade Prodex/OmniRoute: P01–P34 + SC01–SC10, waivers e assinaturas. | **HOLD + REWRITE.** A matriz de 44 linhas já existe, mas nenhuma está fully supported: 39 parciais e 5 não suportadas. Phase 3 HOLD. Não pertence ao P0 atual. | 12–24h após liberação; waivers TBD |
| **5.6** | Rota Cline→GLM52 e fallback GLM52→NVIDIA, sempre controlado pelo OmniRoute. | **P0-EXIT / REWRITE.** O texto atual fala em adapter NIM/direct NVIDIA e está superseded pela matriz D‑V3‑27. | 12–24h |
| **5.7** | Rota Cline→Kimi-K2.7 credentialless. | **P0-EXIT / REWRITE.** As antigas alternativas “registry ou fallback Claude/Codex” já foram decididas e não devem continuar abertas como opções. | 8–16h |
| **5.8** | Revalidar Antigravity, hashes, provenance e funcionamento live non-prod. | **P0-EXIT / REWRITE.** Antigravity já é considerado funcional; não deve ser reimplementado. O texto “implementar Agy ou fallback” ficou velho. | 4–8h |
| **8.1** | Streaming e non-streaming para cada protocolo e rota aprovada. | **P0-EXIT / REWRITE.** Deve referenciar explicitamente a matriz D‑V3‑27. Sobrepõe 5.6–5.8 e 8.2; ETA não é aditiva. | 16–32h compartilhadas |
| **8.2** | Tools, reasoning, usage, cancelamento e erros em todas as rotas aprovadas. | **P0-EXIT / REWRITE.** A expressão genérica de cinco vendors está velha. Deve cobrir as sete rotas/pathways atuais e o fallback GLM52→NVIDIA. | 16–32h compartilhadas |
| **8.5** | Expiração/revogação, quota, 401/403/429, 5xx, timeout e upstream inválido. | **P0-NOW, máxima.** É failure handling real. Evidência sintética existente não fecha a tarefa. | 12–24h |
| **8.6** | Retry somente antes do primeiro output, no-replay depois de output/tool, dedup e cancelamento. | **P0-NOW, máxima.** Depende de CommitLedger, watermark de primeiro output e integração produtiva fail-closed. | 8–16h após infraestrutura comum |
| **8.7** | Adicionar/remover/quarentenar/reintegrar contas e testar restart/rollback durante carga. | **P0-NOW, máxima.** Evidência atual é parcial/in-memory; falta serviço persistido e active load. | 12–24h |

### O que isso significa para o Main Brain

Os oito checkboxes P0 (`5.6–5.8`, `8.1`, `8.2`, `8.5–8.7`) são, na prática, apenas **quatro workstreams**:

1. **Matriz de rotas live:** 5.6–5.8 + 8.1 + 8.2.
2. **Failure handling:** 8.5.
3. **Replay/dedup/cancel:** 8.6.
4. **Lifecycle/restart/rollback:** 8.7.

**ETA funcional oficial já registrada:** **24–48h nominal, 72h conservador**, excluindo OBS, após re-dispatch e disponibilidade da chave nova injetada pelo dono diretamente no OmniRoute.

---

## 2. Observabilidade — 11 itens P2

Todos são legítimos, mas **nenhum deve ser executado agora**. Permanecem como gate futuro antes de capacidade/cutover.

| ID | O que é / importância | Criticidade atual | ETA |
|---|---|---|---:|
| **OBS-1** | Contrato de IDs e correlação metadata-only dos oito hops. | **P2-CUT**; fundação OBS. | 4–8h |
| **OBS-2** | Span do ingresso da API. | **P2-CUT**. | 4–8h |
| **OBS-3** | Span da fila DB, enqueue/dequeue e espera. | **P2-CUT**. | 6–12h |
| **OBS-4** | Span de admission/lifecycle no daemon. | **P2-CUT**. | 4–8h |
| **OBS-5** | Span do processo CLI: launch, exit, cancel e argv estruturalmente redigido. | **P2-CUT**. | 8–16h |
| **OBS-6** | Span OmniRoute/provider usando apenas telemetria segura. | **P2-CUT**. | 4–8h |
| **OBS-7** | Span de persistência terminal, status e contadores sem conteúdo. | **P2-CUT**. | 6–12h |
| **OBS-8** | Span de entrega WS/UI, backpressure, drops e reconnect. | **P2-CUT**. | 4–8h |
| **OBS-9** | Montagem do trace contínuo com detecção de gaps/orphans. | **P2-CUT, crítico na fase OBS.** Depende dos hops anteriores. | 8–16h |
| **OBS-10** | Scan estrutural de vazamento de conteúdo/segredo. | **P2-CUT, STOP gate.** | 8–16h |
| **OBS-11** | Exporter, dashboards, alertas, bundle final e revisão independente. | **P2-CUT, fechamento OBS.** | 12–24h |

Como os hops podem ser implementados em paralelo, não se deve somar todos os ETAs. Caminho crítico preliminar após o P0: aproximadamente **32–64 horas ativas**, sujeito à reautorização P2.

---

## 3. Capacidade, cutover e limpeza — 14 itens

| ID | O que é / importância | Criticidade e auditoria | ETA |
|---|---|---|---:|
| **9.1** | Executar perfil sustentado de 20 tarefas e medir recursos/fairness/recovery. | **P2-CUT.** Obrigatório para certificar tier 20, mas bloqueado por OBS. | 16–32h + duração da execução TBD |
| **9.2** | Habilitar tier 20 somente se os thresholds passarem. | **P2-CUT.** Primeiro tier autorizado. | 4–8h |
| **9.3** | Executar perfil tier 50. | **FOLLOW-UP.** Não autorizado e não bloqueia o deploy tier 20. | 16–32h após autorização |
| **9.4** | Habilitar tier 50 ou manter tier 20. | **FOLLOW-UP.** Gate G7 futuro. | 4–8h |
| **9.5** | Executar perfil tier 100 e overload bounded. | **FOLLOW-UP.** Pós-tier 50 e não autorizado agora. | 24–40h |
| **9.6** | Habilitar tier 100 ou fixar o maior tier comprovado. | **FOLLOW-UP.** | 4–8h |
| **9.7** | Exercitar runbooks, alertas, backup/restore, rotação, incidentes e obter sign-off operacional. | **P2-CUT / REWRITE.** Duplica artefatos 6.4, 6.6 e OBS-11; deve validar/exercitar, não recriar. | 8–16h + sign-off TBD |
| **10.1** | Tornar gateway-required default para novas tarefas. | **CUT / REWRITE.** É o ato de cutover. O texto deve citar explicitamente OBS, tier 20 e paridade assinada. | 8–16h após todos os gates |
| **10.2** | Observar cohort controlada e provar ausência de tráfego direto, dual-router e vazamento. | **P2-CUT.** Não é production soak. | 4–8h ativas + janela TBD |
| **10.3** | Drenar legado e remover flag temporária quando rollback não depender mais dela. | **Pós-cutover imediato.** Depende de cohort e telemetria zero-use. | 8–16h + janela TBD |
| **10.4** | Manter Prodex como cold recovery default-OFF, mutuamente exclusivo e operator-gated. | **HOLD explícito.** Válido, mas Phase 3; não executar agora. | 16–32h após liberação |
| **10.5** | Remover rotation/retry/account-selection Go depois de zero-use. | **FOLLOW-UP.** Destrutivo e desnecessário para o cutover inicial. | 16–32h + zero-use TBD |
| **10.6** | Remover cópia de credenciais e caminhos provider-key antigos. | **FOLLOW-UP / REWRITE.** Deve ser atualizado para a matriz D‑V3‑27. | 8–16h + zero-use TBD |
| **10.7** | Reconciliar documentação, threat model, runbooks e rollback final. | **P2-CUT / REWRITE.** O texto atual pode conflitar com Prodex cold recovery; rollback não pode fingir que Prodex foi deletado. | 8–16h |

---

## 4. Debranding — 7 itens que não pertencem ao deploy atual

A decisão D‑V3‑03 diz explicitamente para **não amarrar nome/debranding ao caminho crítico**. Todos estes itens devem ir para um change de follow-up G8.

| ID | O que é | Criticidade atual | ETA |
|---|---|---|---:|
| **11.1** | Inventariar todos os nomes Multica/Prodex. | **FOLLOW-UP**, não crítico. | 8–16h |
| **11.2** | Introduzir nomes definitivos e contratos compatíveis. | **FOLLOW-UP**; depende da decisão do nome. | 16–32h |
| **11.3** | Renomear IDs e superfícies do gateway. | **FOLLOW-UP**. | 8–16h |
| **11.4** | Migrar task homes, runtime brief, paths e variáveis CLI. | **FOLLOW-UP**, risco de migração. | 12–24h |
| **11.5** | Renomear deployment, dashboards, docs e runbooks. | **FOLLOW-UP**; sobrepõe 9.7/10.7. | 12–24h |
| **11.6** | Remover aliases após telemetria zero-use. | **FOLLOW-UP**. | 8–16h + zero-use TBD |
| **11.7** | Sign-off final completo. | **FOLLOW-UP / REWRITE.** “Sem dependência Prodex” deve significar sem dependência **hot**; cold recovery permanece. | 8–16h + sign-off TBD |

---

# Limpeza recomendada do OpenSpec

## Manter no board P0 atual

- `8.5`, `8.6`, `8.7`
- Reescrever e manter como gate de saída P0: `5.6`, `5.7`, `5.8`, `8.1`, `8.2`

Isso dá **8 checkboxes**, mas apenas **4 workstreams reais**.

## Marcar claramente como P2/deferred

- `OBS-1` a `OBS-11`
- `9.1`, `9.2`
- `9.7`, `10.1`, `10.2`, `10.3`, `10.7`

## Manter em HOLD

- `1.4`
- `10.4`

## Mover para changes futuros

- Tiers 50/100: `9.3–9.6`
- Remoções pós-zero-use: `10.5–10.6`
- Debranding completo: `11.1–11.7`

Total a retirar da contagem do deploy atual: **13 tarefas**.

## Reescrever porque o texto está velho/duplicado

- `1.3`, `1.4`
- `5.6–5.8`
- `8.1`, `8.2`
- `9.7`
- `10.1`, `10.7`

Nenhuma deve simplesmente ser marcada como concluída: existe evidência parcial, mas não closure-grade.

## Número que deveria aparecer no dashboard

Em vez de “42 pendências do deploy”:

```text
Main Brain P0:
  3 tarefas ativas agora
  5 tarefas de saída P0
  = 8 checkboxes / 4 workstreams

P2 observabilidade:
  11 deferred

HOLD:
  2

Pré-cutover futuro:
  8

Follow-up/pós-cutover:
  13
```

Assim, o número correto para o foco atual é **8**, não 42 — e o caminho crítico operacional imediato continua sendo `8.5–8.7`.

[STEERING steer-38d8d982bcfa4483bbf701bc6a1edd46: confirmei a orientação do usuário contra D‑V3‑21/22/23 e classifiquei OBS-1..OBS-11 como P2/deferred, fora das lanes atuais; preservei apenas seu papel futuro como gate antes de capacidade e cutover.]