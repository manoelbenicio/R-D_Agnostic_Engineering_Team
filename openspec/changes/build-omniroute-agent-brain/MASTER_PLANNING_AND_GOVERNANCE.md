# Agent Brain — Planejamento Mestre, Governança OpenSpec/GSD e ETA

## 1. Estado de autorização

Implementação das Waves 0–3 e do canary tier 20 foi **AUTORIZADA** pelo dono em 2026-07-17 na Seção 7.1 de `OMNIROUTE_ARCHITECT_RESPONSE.md`.

Nenhuma Wave, teste, canary, mudança de produção, quiescência/ativação do cold recovery mode do Prodex ou alteração de configuração é iniciada por este documento.

**Estado de liderança vigente (2026-07-18):** Kiro/Opus-4.8 é planning/adjudication owner e
Codex#56#A é operational co-lead/Herdr transport/independent verifier. O owner encerrou o pane
Claude após G2; a decisão Claude de 2026-07-17 permanece como histórico G0/G1 e foi superseded
por D-V3-13. Handover vigente: `KIRO_OPUS48_CODEX56A_TL_HANDOVER.md`.

**Decisão de ambiente vigente (2026-07-18):** o sistema não está em produção. Referências
históricas a production canary/soak nas autorizações e evidências congeladas são reinterpretadas
como controlled development validation; integração, segurança, failure injection, rollback e
bounded capacity continuam obrigatórios. Isso não autoriza cutover, Prodex removal ou tiers 50/100.

## 2. Hierarquia das fontes de verdade

| Nível | Fonte | Responsabilidade | Regra |
|---|---|---|---|
| 0 | Registro de decisão assinável | Autorizações, waivers, tiers e decisões irreversíveis | Prevalece sobre todos os demais documentos |
| 1 | OpenSpec `build-omniroute-agent-brain` | Por quê, escopo, arquitetura, requisitos normativos, paridade e tarefas | Contrato técnico do produto/change |
| 2 | GSD Agent Brain v3 | Fases de execução, owners, dependências, estado vivo, check-in/out e evidências | Governança operacional da entrega |
| 3 | Evidências | Resultados reproduzíveis de conformidade, segurança, falhas e capacidade | Um requisito só fecha com evidência vinculada |
| 4 | RPP/Prodex e changes anteriores | Histórico, requisitos herdados e evidência de comportamento | Nunca executados como plano concorrente sem disposição explícita |

OpenSpec responde **o que/por quê/como/aceite**. GSD responde **quando/quem/dependências/estado/evidência**. Um não substitui o outro.

## 3. Registro completo da documentação OpenSpec vigente

| Documento | Finalidade | Estado |
|---|---|---|
| `proposal.md` | Escopo e capacidades do Agent Brain | Completo |
| `design.md` | Arquitetura, decisões, adapters, migração, rollback e quatro streams | Completo |
| `architecture.md` | Diagramas AS-IS e TO-BE e fronteira Brain/OmniRoute | Completo |
| `OMNIROUTE_REQUIREMENTS_HANDOFF.md` | Entrada única para o arquiteto OmniRoute | Completo |
| `OMNIROUTE_ARCHITECT_RESPONSE.md` | Resposta técnica e registro assinável | Completo; autorização aberta |
| `omniroute-architecture-acceptance-checklist.md` | Protocolos, rotação, token/quota/429, segurança e capacidade | Completo; evidências pendentes |
| `prodex-omniroute-feature-parity.md` | Paridade de todas as funções Prodex, inclusive SC01–SC10 | Completo; sign-off pendente |
| `specs/agent-brain-runtime/spec.md` | Contrato do daemon neutro | Completo |
| `specs/omniroute-agent-routing/spec.md` | Contrato de routing/hot path OmniRoute | Completo |
| `specs/credentialless-agent-execution/spec.md` | Isolamento e ausência de credenciais de provider | Completo |
| `specs/parallel-agent-capacity/spec.md` | Capacidade 20/50/100 e overload | Completo |
| `specs/brain-cutover-operations/spec.md` | Migração, gates, rollback e remoção legado | Completo |
| `tasks.md` | 85 tarefas rastreáveis, Waves 0–6 | Completo; 34 concluídas após G2 |
| `MASTER_PLANNING_AND_GOVERNANCE.md` | Governança integrada OpenSpec/GSD e ETA total | Este documento |

`openspec status` reconhece proposal, design, specs e tasks como completos. Isso significa “pronto para revisão/aplicação”, não “produto implementado”.

## 4. Disposição dos outros changes OpenSpec

Nenhum change anterior será apagado silenciosamente. Cada um recebe uma disposição formal antes da implementação.

| Change | Disposição proposta | O que será preservado | O que não será executado |
|---|---|---|---|
| `build-omniroute-agent-brain` | **MASTER ATIVO** | Todo o target Agent Brain + OmniRoute | Nada fora das autorizações por Wave |
| `native-runtimes-onboarding` | **DEPENDÊNCIA A INCORPORAR** | CLI adapters NIM/Cline, model discovery e onboarding útil | Credencial NIM/provider nativa e rotação local; devem virar gateway-only |
| `chat-orchestration-standard` | **DEPENDÊNCIA A PRESERVAR** | TL/Manager, delegação, escape hatch e protocolo de orquestração | Qualquer acoplamento com nomes/contratos Multica que impeça o Brain neutro |
| `agent-credential-isolation` | **OBJETIVO DE SEGURANÇA ABSORVIDO** | Isolamento de processo/home e prevenção de overwrite | Per-provider credential homes, login/rotação dentro do Brain; OmniRoute passa a possuir as contas |
| `persist-prodex-runtime-integration` | **DEFERIDO — COLD-RECOVERY-ONLY (D-V3-16)** | Fail-closed, readiness, reconciliação, isolamento e prevenção de overwrite/fallback como recovery mode default-OFF; completar sob lock Codex1 | Nenhuma expansão funcional além das 16 tasks; `MULTICA_PRODEX_REQUIRED` default 0; Prodex nunca hot/per-request/automático; NÃO deletado (quiesce em G6) |
| `rotation-parity-polyglot` | **HISTÓRICO/BASE DE PARIDADE** | Invariantes, Smart Context, affinity, pre-commit, evidence e segurança | Prodex como target hot path |
| `rotation-router` | **SUPERSEDED/VAZIO** | Nenhum artefato ativo; apenas registro histórico | Criar um segundo router concorrente |

Gate obrigatório: `persist-prodex-runtime-integration` só pode ser executado pelo Codex1 sob
lock exclusivo, como requisito transitório de segurança. `rotation-router` não executa como
plano concorrente. Por D-V3-16, os caminhos Prodex NÃO são deletados: são quiesced para um cold recovery mode default-OFF, mutuamente exclusivo e operator-gated (após replacement/evidence/rollback G6).

## 5. Situação do GSD atual

O GSD atual em `.planning/` pertence ao milestone RPP/Prodex v2.1 e não pode ser usado diretamente como plano Agent Brain porque contém contradições com a decisão atual:

- `PROJECT.md`, `ROADMAP.md` e `STATE.md` ainda definem Prodex/Rust L2 como target.
- `REQUIREMENTS.md` exige Prodex, `rust_l2`, profiles OAuth e rollback para Codex cru.
- `DECISIONS.md` afirma que não existe chave de API, enquanto o target atual usa uma chave estável e limitada do OmniRoute.
- `TASKS_STATUS.md` contém estados históricos contraditórios e não representa as 85 tarefas do Agent Brain.
- `GOLDEN_RULES.md` determina que somente Kiro/Principal pode autorar `.planning/`.

Portanto, estes arquivos permanecem históricos até o rebaseline. Eles não serão sobrescritos por um agente Codex sem uma decisão explícita de governança.

## 6. Pacote GSD Agent Brain v3 necessário

Antes da Wave 0, Claude/GLM-5.2, agora designado pelo dono como planning/orchestration owner, deve registrar formalmente a substituição da regra histórica de autoria e preparar os seguintes documentos canônicos sem apagar o histórico RPP/Prodex:

| Documento GSD | Conteúdo obrigatório |
|---|---|
| `PROJECT.md` | Visão Agent Brain, fronteira cold/hot, non-goals e owners |
| `REQUIREMENTS.md` | IDs AB-REQ únicos, derivados integralmente dos cinco specs OpenSpec e das matrizes de aceite/paridade |
| `ROADMAP.md` | Fases G0–G8, dependências, gates e critérios de saída |
| `STATE.md` | Estado vivo por fase, último fato verificado, blocker e próxima ação autorizada |
| `DECISIONS.md` | OmniRoute como owner, chave única, single-node inicial, naming, tiers e waivers |
| `TRACEABILITY.md` | AB-REQ ↔ OpenSpec requirement/scenario ↔ task ↔ fase ↔ owner ↔ evidência |
| `COMPONENT_REGISTER.md` | Todo componente retain/rename/replace/retire com owner e gate de remoção |
| `INTERFACE_REGISTER.md` | APIs, env, secrets, headers, CLI configs, event schemas e compatibilidade |
| `REMOVAL_REGISTER.md` | Prodex/L2, Go rotation, credential homes e nomes Multica; replacement + prova antes de apagar |
| `RISKS.md` | Riscos, probabilidade, impacto, mitigação, trigger e owner |
| `EVIDENCE_CONTRACT.md` | O que conta como conformidade/canary/capacidade real no novo topology |
| `FILE_OWNERSHIP.md` | Locks disjuntos dos quatro agentes e hotspot único do integrador |
| `AGENT_LEDGER.md` | Check-in/out, tarefa, progresso, ETA, arquivos e resultado |
| `EVIDENCE_INDEX.md` | Índice imutável de protocolo, falhas, segurança, capacidade e rollback |
| `phases/G0...G8/PLAN.md` | Task IDs executáveis por fase; nenhuma ação fora de PLAN |

O rebaseline deve preservar o GSD RPP v2.1 em histórico e declarar as decisões antigas superseded em vez de apagá-las.

## 7. Roadmap completo da entrega

```text
G0 Governança e rebaseline OpenSpec ↔ GSD
            │
            ▼
G1 Freeze de contratos, IDs, owners e files
            │
            ├───────────────┬───────────────┬───────────────┐
            ▼               ▼               ▼               ▼
G2A Brain core       G2B Gateway       G2C Runtime/CLI   G2D Ops/evidence
            └───────────────┴───────────────┴───────────────┘
                                    │
                                    ▼
G3 Integração serial pelo owner do daemon
                                    │
                                    ▼
G4 Protocolos + falhas + segurança + canary tier 20
                                    │
                      ┌─────────────┴─────────────┐
                      ▼                           ▼
G5 Paridade Prodex/Smart Context       G6 Cutover + Prodex→cold recovery (quiesce, não deletar)
                                                  │
                                      ┌───────────┴───────────┐
                                      ▼                       ▼
                               G7 Tiers 50/100          G8 Debrand completo
```

### G0 — Governança/rebaseline

- Tornar o change Agent Brain o master inequívoco.
- Criar GSD v3 e matriz bidirecional.
- Resolver autoria do `.planning/`, state topology inicial, digest e escopo autorizado.
- Marcar formalmente os changes históricos como dependência, absorbed ou superseded.

**Gate:** nenhum requisito, componente, interface ou task sem ID/owner/disposição.

### G1 — Freeze

- Congelar contratos neutros, `CLIKind`, `RouteModel`, gateway config, secret ref, headers, compatibility facade e acceptance IDs.
- Congelar ownership de arquivos; `daemon.go/config.go/health.go` têm um único integrador.

**Gate:** quatro agentes conseguem trabalhar sem editar o mesmo hotspot.

### G2 — Quatro streams paralelas

- Codex 1: Brain core/facade/config foundation.
- Codex 2: OmniRoute client/models/protocol policies.
- Codex 3: environment sanitizer e CLI adapters credentialless.
- Codex 4: deployment, secrets, observability, evidence e capacity harness.

**Gate:** entregas isoladas contra contratos congelados; nenhuma fiação no daemon por agentes 2–4.

### G3 — Integração

- Somente Codex 1 conecta os módulos ao daemon.
- Ativa gateway-required sob flag, fail-closed e `RouterOwner=omniroute`.
- Desliga paths Prodex/Go rotation/credential copy somente para o canary novo; não remove código.

**Gate:** primeiro vertical slice sem credencial provider e sem dual router.

### G4 — Aceite tier 20

- Conformidade por modelo/protocolo.
- Expiry, revoked, quota, 401/403/429/5xx/timeout/broken stream/cancellation/restart.
- Single-flight refresh, affinity preserve e readiness fail-closed.
- Carga/recuperação com 20 tarefas.

**Gate:** todos os portões da Seção 7.4 aplicáveis ao canary têm evidência ou waiver autorizado.

### G5 — Paridade completa Prodex

- Fechar P01–P34 e SC01–SC10.
- Implementar gaps no OmniRoute ou obter waiver explícito.
- Validar Smart Context shadow→canary→fallback exato→self-check e reset/redeem onde requerido.

**Gate:** matriz de paridade assinada; nenhum feature hot-path sem replacement/disposição.

### G6 — Cutover e Prodex→cold recovery mode (quiesce, não deletar)

- Gateway-required default para novas tarefas.
- Drenar legado, observar; **quiescer Prodex/L2 para cold recovery mode default-OFF (retido, não deletado — D-V3-16)**; remover apenas Go rotation após zero-use.
- Rollback volta para versão anterior Agent Brain/OmniRoute, nunca para credenciais diretas/dual router.

**Gate:** Prodex fora do hot path sem perda funcional/operacional/rollback; retido como recovery mode default-OFF, mutuamente exclusivo.

### G7 — Capacidade 50/100

- Tier 50 após relatório aprovado.
- Decisão single-node vs estado compartilhado antes do tier 100/horizontalidade.
- Tier 100 após load/fairness/recovery e resource SLO.

**Gate:** somente o maior tier comprovado fica habilitado.

### G8 — Debrand completo

- Migrar binário, APIs, env, paths, packages/types, storage, métricas, UI e docs.
- Remover aliases somente após zero-use e rollback independente.

**Gate:** nenhuma dependência runtime Multica/Prodex ativa e documentação final reconciliada.

## 8. Controle para nenhum componente ou integração desaparecer

Cada item deve possuir esta cadeia completa:

```text
COMPONENTE / INTERFACE
  → AB-REQ
  → OpenSpec requirement + scenario
  → OpenSpec task
  → GSD phase/task-ID
  → owner + files_locked
  → acceptance/evidence ID
  → status e decisão de release/removal
```

Regras:

1. Nenhum componente é removido por rename global ou cleanup genérico.
2. Toda remoção exige `replacement`, `migration`, `evidence`, `rollback impact` e aprovação.
3. Toda integração deve aparecer no `INTERFACE_REGISTER`: endpoint, auth, message format, timeout, retry, cancellation, observability e owner.
4. Todo requirement deve ter ao menos um scenario OpenSpec e uma evidência GSD.
5. Toda task deve referenciar IDs de requisito e aceite; trabalho sem task-ID é rejeitado.
6. Ao fim de cada fase roda-se auditoria de órfãos: requirements sem task, tasks sem requirement, componentes sem owner e evidências sem requirement.
7. Mudança de escopo atualiza primeiro OpenSpec; depois GSD e traceability no mesmo gate documental.
8. Changes históricos permanecem read-only até sua disposição formal; não há dois planos masters.
9. Prodex e compatibilidade só desaparecem pelo `REMOVAL_REGISTER`.
10. Sign-off técnico, produto e segurança é separado; nenhum agente forja aceite.

## 9. ETA total de alto nível

As estimativas abaixo são tempo corrido com quatro agentes bem coordenados. Desenvolvimento paraleliza; integração central, failure-injection, soak/capacidade e decisões não reduzem linearmente em 4×.

| Fase | ETA esperado |
|---|---:|
| G0 — Rebaseline OpenSpec/GSD | 2–4 horas |
| G1 — Freeze | 0,5–1 hora |
| G2 — Quatro streams paralelas | 3–6 horas |
| G3 — Integração serial | 2–4 horas |
| G4 — Protocolos/falhas/segurança/tier 20 | 4–8 horas |
| **Primeiro canary 20 aceito** | **1–2 dias úteis** |
| G5 — Paridade/Smart Context | 2–5 dias úteis, conforme gaps |
| G6 — Cutover + Prodex→cold recovery quiesce | 1–2 dias úteis |
| G7 — Tiers 50/100 + state decision | 1–3 dias úteis |
| G8 — Debrand completo | 2–4 dias úteis |

### Cenários totais

- **Melhor caso: 5–7 dias úteis.** OmniRoute já satisfaz SC01–SC10, affinity/refresh e protocolos; single-instance é aceito; debrand encontra poucas dependências externas.
- **Mais provável: 8–14 dias úteis.** Alguns gaps OmniRoute precisam ajuste, os gates são executados integralmente e debrand exige migração controlada.
- **Pior caso controlado: 15–25 dias úteis.** Smart Context requer implementação relevante, tier 100 exige estado distribuído e vários consumidores externos dependem de nomes/contratos Multica.

O ETA de **3–6 horas** citado anteriormente representa apenas o melhor caso para produzir um primeiro vertical slice/canary técnico. Não representa paridade Prodex, cutover, tiers 50/100, debrand ou aceite operacional completo.

## 10. Próximo gate documental

Antes de autorizar implementação:

1. aprovar este plano-mestre e o roadmap G0–G8;
2. formalizar no GSD v3 a decisão já tomada de Claude/GLM-5.2 como planning/orchestration owner deste milestone;
3. criar e revisar o GSD v3 com traceability/component/interface/removal registers;
4. resolver/redigir o prefixo parcial de chave exposto no architect response;
5. somente então marcar `AUTORIZADO` para Waves 0–3.
