# REMEDIATION PLAN — Runtime Prodex REAL por trás do rpp.l2.v1

- Aprovado por: Dono (via Kiro/Auditor)
- Data: 2026-07-05
- Escopo: Token-saver/Smart Context MEDIDO, eventos reais, readyz real
- Tempo NÃO é fator; escopo não se reduz.

## PASSO 0 — Investigação (bloqueia 2a vs 2b)
- **Codex#5.5#B (w3:pM)** ~30min
- Investigar: subcomandos NATIVOS do prodex (app-server-broker, gateway) já expõem sessão de runtime REAL mapeável ao rpp.l2.v1?
- SIM → 2a (wire prodex real, leve)
- NÃO → 2b (broker real no prodex-runtime-broker)

## W1 — Paralelo (4 agentes)
| Task | Agente | Pane | ETA | Descrição |
|------|--------|------|-----|-----------|
| B1 | Codex#5.5#B | w3:pM | ~3h | Runtime REAL por trás do contrato (dep PASSO 0) |
| C1 | Codex#5.5#C | w3:pK | ~2h | Daemon Go lança/dirige prodex REAL + single-router |
| A1 | Codex#5.5#A | w3:pJ | ~2h | Harness medição Smart Context (before/after tokens) |
| D1 | Codex#5.5#D | w3:p9 | ~1.5h | readyz REAL sondando PG/Redis, FALHA se PG down |

## W2 — Dependente de W1
| Task | Agente | ETA | Deps | Descrição |
|------|--------|-----|------|-----------|
| A2 | Codex#A | ~3h | B1 | Smart Context REAL + métricas antes/depois + exact-fallback |
| B2 | Codex#B | ~2h | B1 | Eventos reais no /v1/events/stream |
| C2 | Codex#C | ~1.5h | B1 | Integração Go↔broker real, provar ZERO rotação Go em sessão L2 |

## W3 — Dependente de W2
| Task | Agente | ETA | Descrição |
|------|--------|-----|-----------|
| D2 | Codex#D | ~2.5h | Re-run C1-C6 + S1-S5 LIVE contra runtime REAL, readyz-falsification PASSA |
| D3 | Codex#D | ~1h | Kill-switch + rollback re-teste no runtime real |

## W4 — Close-out (TL + Auditor)
- Docs GSD/tasks.md/specs/SUMMARY/VERIFICATION com evidência REAL
- Commit + push origin/main

## REGRAS
- 4 agentes + 1 reserva (failover quota)
- Hotspot Rust = dono único serial
- SEM shim/mock/override
- Evidence-gated
- Check-in/out Golden Rule por agente
- Evidência em .deploy-control/evidence/
