# STATE — Milestone v2.1 (Full Vendor Validation + PROD Deploy)

> Estado vivo do milestone. Atualizado a cada avanço de fase.

## Posição atual
- **Milestone:** v2.1 "Full Vendor Validation + PROD Deploy"
- **Milestone anterior:** v2.0 "Fundação + Deploy Correto" — COMPLETE (commit `6ba9a70`, 78/78 tasks, Smart Context REAL tokens_saved=4139/16476/65827, D2 ALL PASS, readyz-falsification PASS, commit final `69160af`).
- **Fase atual:** Phase 11 — Vendor Validation (IN_PROGRESS)
- **Próxima fase:** Phase 12 — PROD Deploy (BLOCKED by Phase 11)
- **Config GSD:** mode=yolo, profile=quality, evidence-gated, elite Codex squad.

## Blocker crítico
- **Nenhum** — runtime sidecar UP (43292), gateway UP (43291), Postgres UP, todas capabilities base provadas.

## Staffing v2.1
| Agent | Pane | Task | Status |
|:---|:---|:---|:---|
| Codex#A | w3:pJ | V1: Codex(OpenAI) smart_context + reset_claim | DISPATCHED |
| Codex#B | w3:pK | STANDBY — dono hotspot Rust, corrige se falhar | STANDBY |
| Codex#C | w3:pM | V1: Kiro smart_context via proxy | DISPATCHED |
| Codex#D | w3:p9 | V1: Antigravity smart_context + rotation | DISPATCHED |
| Codex#E | — | V1: Cline smart_context + OpenCode disposition | DISPATCHED |
| Codex#F | — | RESERVA/failover | IDLE |

## Já pronto (v2.0 evidence)
- Smart Context REAL: tokens_saved 4139/16476/65827 em 16/64/256KiB (C5-final-remeasure-3sizes.md)
- Smoke C5 corrigido: tokens_saved=16651 no /v1/runtime/proxy (C5-smoke-fix-reconciliation.md)
- D2 ALL PASS: readyz, state-backend, session, events, policy, profile
- D3: killswitch applied=true, readyz-falsification 503 when PG down
- Readyz: NOT hardcoded (provado empiricamente)
- prodex binary: v0.246.0, commit 7750da9b, cargo build --release OK
- Go integration: daemon.go + l2_runtime.go + prodex.go tested

## Próximo passo
V1 validação paralela por vendor → V2 deploy PROD.
