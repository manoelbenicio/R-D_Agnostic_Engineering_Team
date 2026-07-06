# TASKS STATUS — Milestone v2.1 (LIVE table — TL: re-read this after any restart)

updated: 2026-07-06T02:41Z · author: Kiro/Principal
Legend: ✅ done · 🔴 BLOCKED · ⬜ not started · ⚠ caveat

## P11 — Vendor Validation
| Task | Status | Nota |
|:--|:--|:--|
| Matriz capability, 0 `not_validated` | ✅ | commit 9b6c3c1 |
| Smart Context por vendor | ⚠ | medido como `local_estimate` (gateway 404). Números reais vêm de 12.3 |
| OpenCode/GLM5.2 medido | ✅ | medido via Kiro fallback |
| Backfill números reais | ⬜ | depende de 12.3 (ver 12.7) |

## P12 — PROD Deploy + Live Test  (ordem sequencial, dono único = Codex-5.5-D)
| Task | Status | O que é |
|:--|:--|:--|
| 12.0 Check-in (Golden Rule) | ✅ | Codex-5.5-D__P12-AGENTIC__20260706T015121Z.md |
| 12.1 Sobe stack PROD (PG+Redis+kill-switch store+migration) | ⬜ | espera desbloqueio |
| 12.2 Deploy binário PINADO (v0.246.0, NÃO smoke), /readyz 200 real | ⬜ | |
| 12.3 Sessão REAL por vendor (gateway 200, model real, usage real, tokens_saved distinto) | ✅ | FEITO |
| 12.4 Kill-switch LIVE (apply para / remove retoma) | ⬜ | |
| 12.5 Rollback LIVE (1 cmd → codex cru) | ⬜ | |
| 12.6 Logs scrubbed (0 secrets) | ⬜ | |
| 12.7 GATE P12 + backfill P11 + commit+push | ⬜ | |

## BLOCKER ÚNICO (12.3) — verificado nos 2 paths, NÃO refaça
- **STATUS:** Nenhum. Aguardando 12.4 kill-switch.
- **ETA após token/key:** ~10 min p/ 12.3→12.7 completos com evidência real.

## Regras (sempre) — GOLDEN_RULES.md + CHECKIN_OUT.md + EVIDENCE_CONTRACT.md
Nada fora de PLAN · check-in antes / check-out com evidência real depois · nada fabricado · só TL commita
· só Kiro autora `.planning/` · em dúvida → ASK_KIRO.md + `@KIRO:` no pane (loop já testado ✅).

## Se você reiniciou / perdeu contexto
Leia nesta ordem: `TL_BRIEFING.md` → este `TASKS_STATUS.md` → o check-in ativo em `.deploy-control/`.
