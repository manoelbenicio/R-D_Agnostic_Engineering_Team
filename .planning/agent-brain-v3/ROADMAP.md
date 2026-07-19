# ROADMAP — Agent Brain v3 (G0–G8)

> Waves 0–3/tier 20 AUTORIZADOS. Seção 7.1 de OMNIROUTE_ARCHITECT_RESPONSE.md = `AUTORIZADO`.
> G0/G1/G2 concluídos no escopo autorizado. G3 está READY; cutover/tiers 50–100 permanecem
> bloqueados. Prodex NÃO é deletado — quiesced para cold recovery mode default-OFF (D-V3-16).

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
G4 Protocolos + falhas + segurança + dev-validation tier 20  ← IN PROGRESS
   │  Gate: integração/segurança/falhas/rollback/capacidade têm evidência em desenvolvimento
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

## Fases e gates

| Fase | Objetivo | Gate de saída | ETA | Autorização |
|---|---|---|---|---|
| G0 | Rebaseline OpenSpec↔GSD; registrar TL owner; registros; auditoria de órfãos; disposição de changes; resolver worktree suja | Nenhum req/comp/iface/task órfão; worktree resolvida; GSD v3 pronto | 2–4h | CONCLUÍDO |
| G1 | Freeze contratos neutros, CLIKind/RouteModel/RouterOwner, gateway config, secret ref, headers, facade, IDs aceite, ownership de arquivos | 4 agentes operam sem conflito de hotspot | concluído | COMPLETE |
| G2 | 4 streams paralelas (Brain/Gateway/Runtime-CLI/Ops) | Entregas isoladas contra contratos congelados | concluído | COMPLETE 2026-07-18 |
| G3 | Codex 1 integra módulos no daemon (hotspot único); gateway-required sob flag; fail-closed | Vertical slice sem credencial provider e sem dual router | concluído | COMPLETE 2026-07-18T02:38:30Z |
| G4 | Conformidade por modelo/protocolo; expiry/quota/401/403/429/5xx/timeout/stream/cancel/restart; tier 20 dev | Evidência de desenvolvimento para protocolo, segurança, falha, rollback e capacidade | active | IN_PROGRESS 2026-07-18T03:05:35Z |
| G4-OBS | Observabilidade E2E metadata-only nos 8 hops (ingress→queue→daemon→CLI→OmniRoute→persist→WS/UI→trace); OBS-1..OBS-11 | Trace sintético contínuo + leak-clean estrutural + dashboards/alerts aceitos; BLOQUEIA capacidade e cutover | pendente | AUTORIZADO (D-V3-17) — gate bloqueante |
| G5 | Fechar P01–P34 + SC01–SC10; implementar gaps no OmniRoute ou waiver | Matriz de paridade assinada | 2–5 dias úteis | PENDENTE |
| G6 | Gateway-required default p/ novas tasks; drenar legado; **Prodex quiesced para cold recovery mode default-OFF (não deletado, D-V3-16)** | Prodex removível do hot path sem perda; retido como recovery mode mutuamente exclusivo | 1–2 dias úteis | PENDENTE |
| G7 | Tier 50 após relatório; decisão single-node vs compartilhado; tier 100 após load/fairness/recovery | Só o maior tier comprovado habilitado | 1–3 dias úteis | PENDENTE |
| G8 | Migrar binário/APIs/env/paths/packages/storage/métricas/UI/docs; remover aliases após zero-use | Sem dependência runtime HOT Multica/Prodex (Prodex só recovery mode default-OFF, D-V3-16); docs finais | 2–4 dias úteis | PENDENTE |

## ETA total (não-linear — gates seriais)

- ETA rebaselineado a partir de G3: primeiro vertical slice 2–4h; milestone completo ainda
  depende de gaps G4–G8 e não deve ser inferido do slice.
- **NÃO prometer 4× de redução**: implementação paraleliza; integração, failure-injection,
  bounded-capacity validation e sign-offs são caminhos seriais.

## Primeiro tier autorizado

- Tier **20** como primeiro perfil de validação de desenvolvimento; não é produção.
- Tiers 50/100 e cutover-default e a quiesce de Prodex para cold recovery mode **não autorizados** no escopo Waves 0–3. Deleção de Prodex é explicitamente fora de escopo (D-V3-16).
