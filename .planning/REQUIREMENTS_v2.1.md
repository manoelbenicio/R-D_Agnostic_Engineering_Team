# REQUIREMENTS — Milestone v2.1 (Vendor Validation + PROD Deploy)

> Requisitos rastreáveis (REQ-IDs). Continuação da numeração v2.0 (que foi até REQ-32).
> Cada REQ mapeia para fase(s) no ROADMAP e é governado pelo EVIDENCE_CONTRACT.md.
> Aterrado no estado real (ver STATE.md): P11 = estimate-only; P12 = blocked em creds/host reais.

## Fase 11 — Vendor Validation
- **REQ-33** — Medição REAL de Smart Context por vendor real {OpenAI/Codex, Antigravity, OpenCode/GLM5.2, Kiro/Opus4.8}: gateway_status=200, measurement_source=gateway_usage, tokens_saved>0, números DISTINTOS por vendor. (Nota: fecha de fato em P12.12.3 — round-trip real.)
- **REQ-34** — OpenCode/GLM5.2 medido de verdade (hoje NÃO medido — mediram Cline por engano). Cline não é vendor-alvo.
- **REQ-35** — Matriz de capability sem células `not_validated`, com proveniência por EVIDENCE_CONTRACT §0.
- **REQ-36** — Qualquer número marcado `local_estimate` DEVE ser rotulado como tal até o backfill real (REQ-40).

## Fase 12 — PROD Deploy + Live Test
- **REQ-37** — Deploy do binário PINADO (versão+commit reais, não "smoke"/0.1.0) em host PROD real (não 127.0.0.1); /readyz 200 com probe PG real; /healthz ok.
- **REQ-38** — Sessão live REAL provider-backed por vendor: upstream real (model id real, ≠ fake-upstream-logging), creds reais (≠ placeholder), usage realista (≠ 8/1), runtime_session_id distinto. (EVIDENCE_CONTRACT §1–§2.)
- **REQ-39** — Kill-switch LIVE em PROD: apply → roteamento para observável; remove → retoma. (before/after, não prosa.)
- **REQ-40** — Rollback LIVE em PROD: 1 comando → serviço volta a codex cru, observável.
- **REQ-41** — Logs scrubbed no caminho PROD: grep secrets/tokens → 0 matches (mostrado).
- **REQ-42** — Backfill: substituir os números `local_estimate` de P11 pelos reais de 12.3; OpenCode coberto.

## Governança (transversal)
- **REQ-43** — Nenhuma task executa sem task-ID em PLAN.md + check-in Golden Rule (ORCHESTRATION §protocolo).
- **REQ-44** — Zero evidência fabricada. Violação → INVALID + task BLOCKED + escala ao Kiro (EVIDENCE_CONTRACT §rejeição).
- **REQ-45** — Pré-requisito owner-only: creds de provider real + host PROD real (phases/12-prod-deploy/PREREQUISITES.md). Se indisponível → P12 DEFERIDO honestamente (owner-signed), nunca fingido.

## Rastreabilidade
| REQ | Fase | Doc de execução |
|:--|:--|:--|
| 33–36 | P11 | phases/11-vendor-validation/{SPEC,PLAN}.md |
| 37–42 | P12 | phases/12-prod-deploy/{SPEC,RESEARCH,PLAN}.md |
| 43–45 | ambas | EVIDENCE_CONTRACT.md, ORCHESTRATION_v2.1.md, PREREQUISITES.md |
