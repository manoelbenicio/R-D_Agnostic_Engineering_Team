# PROJECT — Agent Brain v3 (OpenSpec/GSD Wave 3 Rebaseline)

Planning/adjudication key: Kiro `claude-opus-4.8` — Technical Lead/Manager.
planning_owner: Kiro/Opus-4.8 · operational_co_lead: Codex#56#A · rebaseline_agent: agy-p0-a7 (ORQ-59)
active_milestone: Agent Brain v3 — Wave 3 Full Live Rebaseline & OpenSpec Traceability (ORQ-59)
openspec_topic_sha: 7618599f29d43e964a485ab12a9932a9fd037e1f (accepted-in-review topic content, produced by ORQ-62 on branch agent/agy-p0-a7/eb2c0b57)
base_integration_pointer: b657129 (origin/main)
production_overlay_commit: 8227241 (Docker / EC2 active container deployment)
supersedes_planning: .planning/ (RPP/Prodex v2.1) — preservado como histórico, NÃO executado como plano ativo
active_openspec_change: build-omniroute-agent-brain (MASTER ATIVO)

## O que é este projeto

Agent Brain é o **cold control plane** brand-neutral: orquestra tasks, workspaces,
repositories, lifecycle de processos, launch/cancelamento de CLIs, streaming de
resultados, admission control (tiers 20/50/100) e a política `CLIKind` + `RouteModel`.
Não possui credenciais de provider. Guarda apenas **uma** chave OmniRoute estável e
limitada.

OmniRoute é o **hot data plane exclusivo**: credenciais/subscriptions de provider,
account pools, strict round-robin, continuation affinity, token refresh/expiry,
quota/reset/redeem, 401/403/429, circuit breakers, bounded retry/fallback,
protocol translation, streaming/tools, Smart Context (SC01–SC10) e telemetria/auditoria.

## Fronteira cold / hot

| Responsabilidade | Agent Brain (cold) | OmniRoute (hot) |
|---|---|---|
| Tasks, workspaces, repos, processos, launch/cancel/stream | Owns | — |
| Admission 20/50/100 | Task-level admission | Inference/account limits |
| Credenciais/subscriptions de provider | **Nunca** | Exclusive owner |
| Account selection + strict round-robin | Não duplica | Exclusive owner |
| Continuation affinity | Fornece IDs opacos | Enforce |
| Refresh/expiry/quota/reset/redeem | Observa status seguro | Exclusive owner |
| 429/5xx/retry/circuit/fallback | Define deadline/policy | Executa bounded pre-commit |
| Protocol translation / streaming / tools | Configura adapter | Preserva ou rejeita |
| Smart Context/token saving | Kill switch only | Computa e executa |
| Secrets observability | Redige chave+conteúdo | Redige chave/secrets+conteúdo |

## Fatos de Produção & Infraestrutura (Reconciliados Wave 3 em 2026-07-29T14:55:30Z)

1. **Ponteiro de Integração Base (`main`)**: Commit `b657129` no repositório `R-D_Agnostic_Engineering_Team` mantido como referência de ponteiro de integração.
2. **Overlay de Produção Ativo**: Imagem Docker de produção e contêineres rodam com overlay commit `8227241`.
3. **Outages e Incidentes Ativos**:
   - **ORQ-26 Outage**: Falha na materialização do default squad e roteamento de Chat Lifecycle.
   - **GitHub Billing Lock**: Bloqueio de cobrança no GitHub afetando workflows automatizados de CI/CD.
   - **Incidente de Helper-Label ORQ-61/59**: Incidente de rótulos auxiliares registrado e corrigido como metadata-only.
4. **OpenSpec SHA Reconciliada**:
   - A SHA remota `7618599f29d43e964a485ab12a9932a9fd037e1f` (produzida no ORQ-62 na branch `agent/agy-p0-a7/eb2c0b57`) representa conteúdo **accepted-in-review / topic content** para a linhagem OpenSpec, e não canonical-main ou integrated.
5. **Superação de Afirmações Obsoletas**:
   - Afirmações anteriores de "não-produtivo" ou "production canary removido" foram **superadas explicitamente** em favor dos fatos de produção reais (overlay `8227241`, canários Docker em EC2, incidente ORQ-26). Histórico preservado sem apagamento.
   - Contagens antigas 51/96 superadas pelo inventário vivo de **52 cards** do Multica Kanban (projetados diretamente do DB em `2026-07-29T14:55:30Z`: 24 done, 12 in_review, 4 in_progress, 7 blocked, 1 todo, 2 backlog, 2 cancelled).
   - Instruções de STOP/PD-08 superadas pela postura de segurança atual (SCRAM hardening ORQ-35/60, JWT rotation ORQ-30/33/42, credential isolation ativo).
6. **Separação de Gates (Readiness vs Acceptance)**:
   - **Readiness Gates**: Validação em harness sintético local (ex: harness de carga local de 20 tasks em ORQ-50).
   - **Acceptance Gates**: Aceitação em ambiente de produção com canários reais, persistência em PostgreSQL e telemetria de produção.

## Non-goals (deste plano, e do target)

- Não deletar Prodex nesta fase (retido como cold recovery mode default-OFF por D-V3-16).
- Não ativar tiers 50/100 sem relatório de capacidade aprovado (gate G7).
- Não ativar gateway-required como default antes do gate G6.
- Não permitir dual router (Prodex + OmniRoute ativos simultaneamente como owners).
- Não pôr credenciais provider dentro do Agent Brain.
- Não ler, imprimir, copiar ou registrar a chave OmniRoute.
- Não reescrever cegamente lifecycle/workspace/repo/cancel/streaming funcionais.

## Owners

- **Planning/adjudication owner:** Kiro/Opus-4.8.
- **Operational co-lead:** Codex#56#A (Herdr transport, independent verification, state/docs, execution control).
- **Integrator-líder:** Codex 1 — único owner de `daemon.go`, `config.go`, `health.go`, `execenv/execenv.go`, `execenv/codex_home.go`, `pkg/agent/models.go`, `cmd/multica/cmd_daemon.go`.
- **Streams paralelos:** Codex 2 (gateway), Codex 3 (runtime/CLI), Codex 4 (ops/paridade/evidência).
- **Sign-off:** arquiteto OmniRoute, dono do produto, segurança.

## Mapeamento de histórico

- RPP/Prodex v2.1 (`.planning/`) → SUPERSEDED como plano ativo; preservado como histórico.
- Decisões legadas D-007 (isolamento credenciais), D-008 (TL delegation-only) e EVIDENCE_CONTRACT → absorvidos como governança permanente.
- OpenSpec Lineage Defect (fc98e218 sem `rotation-parity-polyglot`) → RECONCILIADO sob a SHA accepted-in-review `7618599f29d43e964a485ab12a9932a9fd037e1f` (ORQ-62).

## Índice de artefatos (.planning/agent-brain-v3/)

PROJECT.md · REQUIREMENTS.md · ROADMAP.md · STATE.md · DECISIONS.md · TRACEABILITY.md ·
COMPONENT_REGISTER.md · INTERFACE_REGISTER.md · REMOVAL_REGISTER.md · RISKS.md ·
EVIDENCE_CONTRACT.md · FILE_OWNERSHIP.md · AGENT_LEDGER.md · EVIDENCE_INDEX.md ·
HERDR_TRANSPORT.md · DISPATCH_QUEUE.md · G4_ACCELERATED_PACKET.md ·
evidence/gsd-wave3-rebaseline-reconciliation.md · phases/G0..G8/PLAN.md
