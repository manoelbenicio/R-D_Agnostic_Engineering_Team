[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Peer Review Retificado GTL-R31A (ORQ-31) — Plano de QA de Browser Descartável (gtl-orq31-browser-qa-plan-peer-review.md)

1. RETIFICAÇÃO DAS 4 CONTRADIÇÕES ESTRUTURAIS DE GITHUB ACTIONS:
- Localização do Workflow: O arquivo .yml DEVE residir na raiz em .github/workflows/orq31-browser-qa.yml. O subdiretório multica-auth-work é configurado via defaults.run.working-directory.
- Networking em Container de Job: O container de serviço Postgres é acessado via hostname postgres:5432 (não localhost) dentro da rede do job container.
- Pinning Imutável de Actions: Inclusão de commit SHAs imutáveis para actions/checkout, setup-go, action-setup e setup-node.
- Estratégia de Trigger: workflow_dispatch não funciona em nova branch; ativado via pull_request para main com filtros de path estritos.

2. SEGURANÇA E VERIFICAÇÃO DE FERRAMENTAS:
- Guarda .env: O job FALHA E ABORTA se arquivos .env ou .env.worktree forem detectados no workspace de CI.
- Ferramentas no Container: Inclusão explícita de passos setup-go e setup-node/pnpm para preparar o ambiente antes da compilação e execução dos testes Playwright.

3. VEREDITO FINAL:
- STATUS: BLOCK (MANTIDO BLOCK ATÉ PINNING COMPLETO E INTEGRAÇÃO DO WORKFLOW DO ORQ-31).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gtl-orq31-browser-qa-plan-peer-review.md.
Operação 100% Read-Only.
