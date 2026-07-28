[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Triagem de Kanban GTL-K01 — Criação da Card ORQ-31 (gtl-orq-disposable-qa-kanban-triage.md)

1. RESULTADO DA BUSCA POR DUPLICATAS:
- A auditoria dos cards ORQ-01 a ORQ-30 confirmou que nenhuma card existente cobre a infraestrutura descartável de QA via Playwright, congelamento de versão do Chromium/package.json ou execução de banco de teste efêmero.
- ORQ-26 é específica do contrato backend/client file.go/uploadFile e não possui critérios de aceite cobrindo a suíte E2E descartável.

2. NOVA CARD KANBAN CRIADA: ORQ-31
- Título: Ephemeral Browser QA & Playwright Supply-Chain Pipeline
- Decisão: CRIADA nova card (ORQ-31), vinculada à ORQ-26 como dependência de contrato.
- Assignee: Opus48#B (QA/Frontend Lead) | Revisor: Codex56-TL (General-TL).

3. ESPECIFICAÇÃO DA CARD ORQ-31:
- Evidência Medida: gtl-browser-qa-disposable-plan.md (playwright.config.ts desabilita auto-start; e2e/fixtures.ts executa DELETE no banco; package.json ^1.58.2 exige lockfile congelado).
- Critérios de Aceite: Runner CI efêmero com postgres:17 efêmero; pnpm install --frozen-lockfile travando image tag mcr.microsoft.com/playwright:v1.58.2-noble; suíte Playwright Chromium PASS; zero dados de produção (DATABASE_URL efêmero).

4. VEREDITO FINAL:
- STATUS: PASS (CARD ORQ-31 CRIADA E VINCULADA).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gtl-orq-disposable-qa-kanban-triage.md.
Operação 100% Read-Only de Governança.
