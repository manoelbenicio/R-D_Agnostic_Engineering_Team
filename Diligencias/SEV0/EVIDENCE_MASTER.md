# EVIDENCE MASTER — prova de tudo (2026-07-04)

> Evidência concreta (comando + saída) de cada afirmação. Re-rodável.

## 1. OpenSpec — válido, 66 tasks, 4 specs
- `openspec validate rotation-parity-polyglot` → **"is valid"**
- `openspec list` → **0/66 tasks, in-progress** (era 0/0 no-tasks)
- specs: `deploy-rollback`, `l2-runtime-contract`, `prodex-runtime-provisioning`, `qa-conformance`

## 2. Prompts — charter#0 + escopo (8/8 streams)
`grep charter/escopo`: A/B/C/D + GLM-A/GLM-B + Gemini-Pro/Flash35 → **charter=1, escopo=1** cada.
(C-GO-INTEGRATE = SUPERSEDED, sem escopo, correto.)

## 3. Dashboard — 66 tasks + QA 49/49
- `plan_dashboard.py --json` → **66 tasks, 11 fases** (data-driven, auto-reflete tasks.md)
- `test_plan_dashboard.py` → **29/29 PASS**
- `test_plan_dashboard_sev0.py` → **20/20 PASS** (fuzz 200x + 12 edge + property + encoding)
- `fleet_dashboard.py` + `tasks.json` → **SUPERSEDED**

## 4. Requisitos — REQ-01 .. REQ-39 (+37b) = 40 REQs
Todos presentes em `.planning/REQUIREMENTS.md`.

## 5. Diligências — 14 docs + SEV0
Charter (00_LEIA_PRIMEIRO), Contexto, 00b deps, 00c crates, 00d env/sec, 00e completude,
fases 00_FUNDACAO..06_QA, README; SEV0/{RISK_REGISTER, EVIDENCE_LOG, DASHBOARD_QA_VERDICT}.

## 6. RCA — 22 erros documentados
`.planning/RCA-2026-07-04-001-orchestrator-errors.md` → **22 ERR-** (causa/correção/prevenção).

## 7. Produto (Go) — build/vet/test verdes (container, IPv4)
`go build ./...`=0, `go vet`=0, **24/24 pacotes de teste OK, 0 FAIL** (execenv 77.5%, daemon 68.3%, rotation 64.7%, events 100%…). Migrations reversíveis: **322 .sql** up/down.

## 8. Bases de conhecimento (pesquisáveis)
`rpp-projeto-diligencias`, `rpp-openspec-plano`, `rpp-gsd-planning` — indexadas.

## 9. Git — commits pushados (rnd/main)
88fb9a4 (dashboard encoding-safe) → 1e9fa08 (SEV-0) → 90e1694 (prompt C P0) → fb7d766 (reconcile) →
53b5dd4 (deps) → 2afc115 (toolchain) → cad2c06 (contexto) → efc7959 (MCP) → 484f209 (44 crates) →
dac8444 (env/segurança/Caveman) → 3d3f3fd (RCA) → 58b2244 (completude/browser/Mem0/CI/deploy) →
21911cd (propagação em todos os prompts/docs/dashboard).

## Como re-verificar (um comando)
```
openspec validate rotation-parity-polyglot && \
python3 scripts/dashboard/test_plan_dashboard.py && \
python3 scripts/dashboard/test_plan_dashboard_sev0.py && \
python3 scripts/dashboard/plan_dashboard.py --once --ascii
```
