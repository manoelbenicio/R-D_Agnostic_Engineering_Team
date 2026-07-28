[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Peer Review GTL-R31 (ORQ-31) — Plano de QA de Browser Descartável (gtl-orq31-browser-qa-plan-peer-review.md)

1. AVALIAÇÃO DA STACK EFÊMERA E AMBIENTE:
- Provedor DB: pgvector/pgvector:pg17 em serviço no runner GitHub Actions (sem acesso a bancos ORQ1/ORQ2 ou segredos de produção).
- Redis: Não é pré-requisito bloqueante (o backend funciona em modo single-node in-memory sem Redis).
- Variáveis de ambiente isoladas: DATABASE_URL local, JWT_SECRET temporário de teste, NEXT_PUBLIC_API_URL em :8080.

2. WORKFLOWS E SUPPLY-CHAIN LOCKING:
- Workflows ativos: ci.yml e mobile-verify.yml. Workflows inertes: desktop-smoke.yml e release.yml.
- Supply-chain lock: Exigência de pnpm install --frozen-lockfile travando a imagem oficial mcr.microsoft.com/playwright:v1.58.2-noble.
- Proteção contra vazamento: Guarda para ignorar .env local no e2e/env.ts durante o job CI.

3. AUDITORIA DAS SPECS E ANÁLISE DE RISCOS:
- Auditadas as 7 specs E2E existentes (auth, chat-attachments, comments, issues, navigation, onboarding-v2-smoke, settings).
- Desafios aos 6 novos testes: Testes visuais (toHaveScreenshot) exigem rodar dentro da imagem oficial v1.58.2-noble para evitar flaky font diffs; artefatos de trace devem expurgar cookies/tokens JWT.

4. VEREDITO FINAL:
- STATUS: PASS (PLANO DE QA DESCARTÁVEL APROVADO COM PORTAS DE EXECUÇÃO).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gtl-orq31-browser-qa-plan-peer-review.md.
Operação 100% Read-Only.
