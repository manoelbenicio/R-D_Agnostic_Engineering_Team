# ROADMAP — Agent Brain v3 (G0–G8 & Multica Kanban Wave 3)

> Waves 0–3/tier 20 AUTORIZADOS. Seção 7.1 de OMNIROUTE_ARCHITECT_RESPONSE.md = `AUTORIZADO`.
> G0/G1/G2/G3 concluídos; Wave 3 Full Live Rebaseline efetuada sob **ORQ-59** em 2026-07-29.
> OpenSpec Topic SHA: `7618599f29d43e964a485ab12a9932a9fd037e1f` (accepted-in-review topic content, ORQ-62).
> Ponteiro base: `main` em `b657129`; Overlay de produção ativo: `8227241`.
> Prodex NÃO é deletado — quiesced para cold recovery mode default-OFF (D-V3-16).
> Contagem do Quadro DB (2026-07-29T14:55:30Z): 52 cards no total (24 done, 12 in_review, 4 in_progress, 7 blocked, 1 todo, 2 backlog, 2 cancelled).

```text
G0 Governança/rebaseline  ← CONCLUÍDO
   │  Gate: nenhum req/comp/iface/task sem ID/owner/disposição; worktree suja resolvida
   ▼
G1 Freeze de contratos/IDs/owners/files  ← CONCLUÍDO
   │  Gate: 4 agentes trabalham sem editar mesmo hotspot
   ├──────────┬──────────┬──────────┐
   ▼          ▼          ▼          ▼
G2A Brain  G2B Gateway  G2C Runtime  G2D Ops  ← CONCLUÍDOS
   └──────────┴──────────┴──────────┘
   │  Gate: entregas isoladas contra contratos congelados; sem fiação por agentes 2–4
   ▼
G3 Integração serial (Codex 1, hotspot único)  ← CONCLUÍDO
   │  Gate: vertical slice sem credencial provider e sem dual router
   ▼
G4 Protocolos + falhas + segurança + dev-validation tier 20 + Wave 3 Rebaseline  ← IN PROGRESS (ORQ-59)
   │  Gate: integração/segurança/falhas/rollback/capacidade têm evidência e 52 cards Kanban mapeados
   ▼
G4-OBS Stop-gate de observabilidade E2E (OBS-1..OBS-11)  ← BLOQUEANTE (D-V3-17)
   │  Gate: trace metadata-only contínuo nos 8 hops + leak-clean; obrigatório antes de capacidade/cutover
   ├──────────────┴──────────────┐
   ▼                             ▼
G5 Paridade Prodex/Smart Context   G6 Cutover + Prodex→cold recovery mode (quiesce, não deletar)
   │  Gate: matriz paridade assinada       │  Gate: Prodex fora do hot path, retido default-OFF/mutuamente exclusivo
   ├────────────────────────────┘
   ▼
G7 Tiers 50/100 + state decision
   │  Gate: só o maior tier comprovado habilitado
   ▼
G8 Debrand completo
   Gate: sem dependência runtime HOT Multica/Prodex (Prodex só recovery mode default-OFF); docs reconciliados
```

## Fases, gates e Mapeamento de Cards Multica Kanban (ORQ-11 a ORQ-62 no DB)

| Fase | Objetivo | Gate de saída | Status / Cards Mapeados |
|---|---|---|---|
| G0 | Rebaseline OpenSpec↔GSD; registrar TL owner; registros; auditoria de órfãos | Nenhum req/comp/iface/task órfão; GSD v3 pronto | CONCLUÍDO (ORQ-11) |
| G1 | Freeze contratos neutros, CLIKind/RouteModel/RouterOwner, gateway config | 4 agentes operam sem conflito de hotspot | CONCLUÍDO (ORQ-20) |
| G2 | 4 streams paralelas (Brain/Gateway/Runtime-CLI/Ops) | Entregas isoladas contra contratos congelados | CONCLUÍDO (ORQ-27, ORQ-28, ORQ-29) |
| G3 | Codex 1 integra módulos no daemon (hotspot único); gateway-required sob flag | Vertical slice sem credencial provider e sem dual router | CONCLUÍDO (ORQ-13, ORQ-26) |
| G4 | Full Live Rebaseline (ORQ-59), Credential Isolation (ORQ-13/14/23/36/37), Native Onboarding (ORQ-51/52/53), Chat Escape Hatch (ORQ-54), OpenSpec Integrity (ORQ-61/62), Capacity (ORQ-50) | Evidência de desenvolvimento para protocolo, segurança, falha, rollback e capacidade; 52 cards mapeados no DB | **IN_PROGRESS** (ORQ-23, 41, 57, 59 in_progress; 24 done; 12 in_review; 7 blocked; 1 todo; 2 backlog; 2 cancelled) |
| G4-OBS | Observabilidade E2E metadata-only nos 8 hops (ingress→queue→daemon→CLI→OmniRoute→persist→WS/UI→trace); OBS-1..OBS-11 | Trace sintético contínuo + leak-clean estrutural + dashboards/alerts aceitos; BLOQUEIA capacidade e cutover | AUTORIZADO (D-V3-17) — gate bloqueante |
| G5 | Fechar P01–P34 + SC01–SC10; implementar gaps no OmniRoute ou waiver | Matriz de paridade assinada | PENDENTE |
| G6 | Gateway-required default p/ novas tasks; drenar legado; Prodex quiesced para cold recovery mode default-OFF (D-V3-16) | Prodex removível do hot path sem perda; retido como recovery mode mutuamente exclusivo | PENDENTE (ORQ-58 backlog) |
| G7 | Tier 50 após relatório; decisão single-node vs compartilhado; tier 100 após load/fairness/recovery | Só o maior tier comprovado habilitado | PENDENTE |
| G8 | Migrar binário/APIs/env/paths/packages/storage/métricas/UI/docs; remover aliases após zero-use | Sem dependência runtime HOT Multica/Prodex; docs finais | PENDENTE |

## Fatos de Produção e Mapeamento de Incidências (Derivado do DB em 2026-07-29T14:55:30Z)

- **Base Integration Pointer**: `main` em `b657129`.
- **Production Overlay**: `8227241` ativo nos contêineres Docker/EC2 de produção.
- **OpenSpec Topic Content SHA**: `7618599f29d43e964a485ab12a9932a9fd037e1f` (accepted-in-review, ORQ-62).
- **Incidente Ativo de Chat (ORQ-26)**: Falha de roteamento de squad e materialização do default squad.
- **GitHub Billing Lock**: Bloqueio de cobrança pendente de resolução administrativa.
- **Incidente Helper-Label ORQ-61/59**: Rótulos auxiliares de cards corrigidos como metadados.
- **Kanban Card Coverage**: 52 cards no total (ORQ-11 a ORQ-62) extraídos do DB com status exatos.
