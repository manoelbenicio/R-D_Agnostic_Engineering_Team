# ROADMAP — Agent Brain v3 (G0–G8)

> Waves 0–3/tier 20 AUTORIZADOS. Seção 7.1 de OMNIROUTE_ARCHITECT_RESPONSE.md = `AUTORIZADO`.
> G0/G1/G2 concluídos no escopo autorizado. G3 está READY; cutover/remoção Prodex/tiers
> 50–100 permanecem bloqueados.

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
   ├──────────────┴──────────────┐
   ▼                             ▼
G5 Paridade Prodex/Smart Context   G6 Cutover + retirada Prodex
   │  Gate: matriz paridade assinada       │  Gate: Prodex removível sem perda
   ├────────────────────────────┘
   ▼
G7 Tiers 50/100 + state decision
   │  Gate: só o maior tier comprovado habilitado
   ▼
G8 Debrand completo
   Gate: sem dependência runtime Multica/Prodex; docs reconciliados
```

## Fases e gates

| Fase | Objetivo | Gate de saída | ETA | Autorização |
|---|---|---|---|---|
| G0 | Rebaseline OpenSpec↔GSD; registrar TL owner; registros; auditoria de órfãos; disposição de changes; resolver worktree suja | Nenhum req/comp/iface/task órfão; worktree resolvida; GSD v3 pronto | 2–4h | CONCLUÍDO |
| G1 | Freeze contratos neutros, CLIKind/RouteModel/RouterOwner, gateway config, secret ref, headers, facade, IDs aceite, ownership de arquivos | 4 agentes operam sem conflito de hotspot | concluído | COMPLETE |
| G2 | 4 streams paralelas (Brain/Gateway/Runtime-CLI/Ops) | Entregas isoladas contra contratos congelados | concluído | COMPLETE 2026-07-18 |
| G3 | Codex 1 integra módulos no daemon (hotspot único); gateway-required sob flag; fail-closed | Vertical slice sem credencial provider e sem dual router | concluído | COMPLETE 2026-07-18T02:38:30Z |
| G4 | Conformidade por modelo/protocolo; expiry/quota/401/403/429/5xx/timeout/stream/cancel/restart; tier 20 dev | Evidência de desenvolvimento para protocolo, segurança, falha, rollback e capacidade | active | IN_PROGRESS 2026-07-18T03:05:35Z |
| G5 | Fechar P01–P34 + SC01–SC10; implementar gaps no OmniRoute ou waiver | Matriz de paridade assinada | 2–5 dias úteis | PENDENTE |
| G6 | Gateway-required default p/ novas tasks; drenar legado; retirar Prodex/L2/Go rotation (após zero-use) | Prodex removível sem perda func/op/rollback | 1–2 dias úteis | PENDENTE |
| G7 | Tier 50 após relatório; decisão single-node vs compartilhado; tier 100 após load/fairness/recovery | Só o maior tier comprovado habilitado | 1–3 dias úteis | PENDENTE |
| G8 | Migrar binário/APIs/env/paths/packages/storage/métricas/UI/docs; remover aliases após zero-use | Sem dependência runtime Multica/Prodex; docs finais | 2–4 dias úteis | PENDENTE |

## ETA total (não-linear — gates seriais)

- ETA rebaselineado a partir de G3: primeiro vertical slice 2–4h; milestone completo ainda
  depende de gaps G4–G8 e não deve ser inferido do slice.
- **NÃO prometer 4× de redução**: implementação paraleliza; integração, failure-injection,
  bounded-capacity validation e sign-offs são caminhos seriais.

## Primeiro tier autorizado

- Tier **20** como primeiro perfil de validação de desenvolvimento; não é produção.
- Tiers 50/100 e cutover-default e remoção de Prodex **não autorizados** no escopo Waves 0–3.
