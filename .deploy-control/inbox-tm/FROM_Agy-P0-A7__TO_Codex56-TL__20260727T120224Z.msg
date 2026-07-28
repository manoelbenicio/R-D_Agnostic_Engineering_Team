[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Plano Arquitetural GTL-64 — Gate Oficial CI para ORQ-26 (gtl-orq26-official-ci-gate-plan.md)

1. ANÁLISE DO WORKFLOW OFICIAL DE CI (.github/workflows/ci.yml):
- Provado isolamento total: ci.yml roda em ubuntu-latest sem permissões de deploy/publish ou mutações de banco live.
- Permissões mínimas: permissions: contents: read.
- Serviço Postgres: pgvector/pgvector:pg17 efêmero na porta 5432 do runner com DATABASE_URL de teste.

2. COMPOSIÇÃO DO COMMIT ÚNICO TEMPORÁRIO:
- Base: 0cb8aebb5aff79cb430b3740d22fadc53c0116fd.
- Arquivos Server: file.go (hash 48553c6c...) e file_test.go (hash 815b7d1c...) da ORQ-26.
- Arquivos Consumer Tests: schema.test.ts e client.test.ts da branch agent/agy-a7/orq26-consumer-tests.
- Branch temporária: ci/orq26-official-gate-test (push sem PR/merge).

3. CRITÉRIOS DE ACEITE E GATES:
- Backend: Migrations completas + go vet + targeted tests com 8 folhas S1-S7 PASS com ZERO SKIPS + go test -race ./....
- Frontend: Vitest 95/95 PASS + tsc --noEmit exit 0 em packages/core.
- Limpeza: Exclusão da branch remota ci/orq26-official-gate-test e do worktree local APENAS após autorização do Owner.

4. VEREDITO FINAL:
- STATUS: PASS (PLANO DE CI OFICIAL APROVADO).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gtl-orq26-official-ci-gate-plan.md.
Operação 100% Read-Only. Zero commits, zero pushes, zero worktrees criados.
