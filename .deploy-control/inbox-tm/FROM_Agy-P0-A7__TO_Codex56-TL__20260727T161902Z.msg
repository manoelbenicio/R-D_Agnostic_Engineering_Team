[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL & Agente A8]
ASSUNTO: Rascunho DRAFT_NOT_POSTED de Proposta de Issue — Correção de Falso-Verde em TestMain (draft-kanban-issue-false-green-testmain.md)

1. SÍNTESE DO INVENTÁRIO DO FALSO-VERDE:
- Pacote Alvo: github.com/multica-ai/multica/server/internal/handler
- Arquivo & Linhas: multica-auth-work/server/internal/handler/handler_test.go:38-54
- Mecanismo: TestMain chama os.Exit(0) antes de m.Run() quando o PostgreSQL não responde, gerando código de saída 0 ("ok") com zero testes executados.

2. CONTEÚDO DO RASCUNHO PREPARADO:
- Conceito do Problema, Impacto e Reprodução Factual.
- Critérios de Aceite: Exigência de fail-closed (Exit non-zero) ou skip por teste auditável, eliminação de os.Exit(0) pré-m.Run(), e verificação nominal de eventos --- PASS: no log do test runner.
- Limites de escopo, não-sobreposição com código de produção e estratégia de rollback.

3. REGRAS DE GOVERNANÇA ATENDIDAS:
- Status: DRAFT_NOT_POSTED (Rascunho Local).
- NENHUM número de card/título atribuído (sem predição de identificadores).
- NENHUMA publicação, comentário ou alteração efetuada no quadro Kanban.
- NENHUMA execução de rede, banco, API ou testes.

4. SHA-256 DO RASCUNHO:
- SHA-256: 54ed73f446a8cd4e60aeabcbdeec45d28350e8a484109bfc55dc7843ffa32a36

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/draft-kanban-issue-false-green-testmain.md.
Operação 100% mecânica/local.
