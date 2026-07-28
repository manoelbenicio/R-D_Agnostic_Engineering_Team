[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Runbook de Execução Wave 0 SQLC Baseline ORQ-13 (orq13-wave0-sqlc-baseline-runbook.md)

1. ESTRATÉGIA DE SPLIT POR COMMITS:
- Commit 1 (Wave 0): Regeneração idempotente pura do sqlc sobre o schema atual para isolar churn de código gerado em pkg/db/generated/.
- Commit 2 (Wave 1): Migration NEXT_CANONICAL (thinking_level, reasoning_effort, reasoning_budget), atualização de queries sqlc e suporte nos handlers.

2. GOVERNAÇA E ISOLAMENTO DE CACHE:
- Disciplina de Escritor Único (Single Owner) em worktree isolado.
- Isolamento estrito de cache/temp via TMPDIR, GOTMPDIR e GOCACHE privados em $HOME/.private-tmp (modo 0700).

3. VALIDAÇÕES E ESTRUTURA DE GATES:
- Comandos idempotentes: sqlc generate --config sqlc.yaml.
- Lint e formatação: gofmt -l, git diff --check, go vet.
- Sequenciamento em 3 Gates (Gate 1: Baseline -> Gate 2: Feature -> Gate 3: Verification).

4. VEREDITO FINAL:
- STATUS: PASS (RUNBOOK WAVE 0 SQLC BASELINE APROVADO PARA REVIEW).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/orq13-wave0-sqlc-baseline-runbook.md.
Operação 100% Read-Only. Zero edições, zero builds, zero migrations executadas.
