[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Emenda de Desenho V4 ORQ-39 — Pipeline de QA Descartável e Playwright (orq39-browser-qa-v4-amendment-design.md)

1. RETIFICAÇÃO DOS PEER REVIEWS ANTERIORES:
- Identificador renomeado para ORQ-39.
- Preservados os commit SHAs imutáveis para todas as GitHub Actions e a imagem mcr.microsoft.com/playwright:v1.58.2-noble.
- Workflow configurado na raiz em .github/workflows/orq39-browser-qa.yml com defaults.run.working-directory: multica-auth-work.

2. MIGRATIONS E AMBIENTE DE TESTE:
- Execução de cd server && go run ./cmd/migrate up contra o container postgres:5432.
- Inicialização da tabela verification_code para suporte seguro aos testes de autenticação E2E.
- Variáveis de ambiente isoladas com APP_ENV=test e fail-fast guard para proibir arquivos .env no workspace.

3. LIMITES DE CUSTO E PORTAS DE AUTORIZAÇÃO:
- Bounded timeout de 15 minutos para o job.
- Quatro portas de autorização separadas e explícitas do Owner (push, PR, merge e workflow run).

4. VEREDITO FINAL:
- STATUS: PASS (EMENDA V4 SUBMETIDA PARA PEER REVIEW INDEPENDENTE).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/orq39-browser-qa-v4-amendment-design.md.
Operação 100% Read-Only.
