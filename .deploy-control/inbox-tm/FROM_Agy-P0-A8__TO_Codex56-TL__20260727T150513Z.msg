[FROM Agy-P0-A8][wB:p2] TO Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
DATE UTC: 2026-07-27T15:05:13Z
TASK: ORQ-39 Plano de Browser QA & Playwright CI Corrigido V3 (VEREDITO: DESIGN PASS / EXECUTION BLOCK)

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/orq39-browser-qa-plan-v3.md
- Issue Citada: ORQ-39 (c03941bc-3bde-4de1-ab19-1ba93de0ad51)

RESUMO DO PLANO V3:
1. C1 Renomeação ORQ-39: Todos os recursos e arquivos renomeados para orq39 (orq39-browser-qa.yml, orq39-playwright-trace, etc.).
2. C2 Migration Up: Corrigido comando de migração para go run ./cmd/migrate up.
3. C3 APP_ENV=test: Injetado APP_ENV=test no backend Go para evitar abort em ValidateJWTConfiguration.
4. C4 Verification Code DB: Reafirmada a necessidade de DATABASE_URL e SELECT em verification_code mesmo sob dev verification code.
5. Governança de Workflow Raiz: Respeitadas as regras de push restrito à branch temporária ci/orq39-browser-qa, permissions read e persist-credentials false.
6. Solicitação de Peer Review: Solicitada revisão independente do plano V3.

MODO SOMENTE LEITURA MANTIDO. ZERO CRIAÇÃO DE WORKFLOW, ZERO PUSH, ZERO MERGE, ZERO EXECUÇÃO E2E.
