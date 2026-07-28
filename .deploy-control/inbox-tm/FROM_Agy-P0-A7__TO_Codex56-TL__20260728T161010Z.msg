[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Reconciliação de Ações de Status do GTL — ORQ-31 Done, ORQ-39 In_Review e Tabela Before/After (gtl-status-actions-reconciliation.md)

1. AÇÕES DE STATUS EXECUTADAS (STATUS-ONLY):
- ORQ-31 (f7e13350...): Movido para `done` (GET-back 100% Verificado).
- ORQ-39 (c03941bc...): Movido para `in_review` (GET-back 100% Verificado).
- ORQ-18 (b3cec211...) e ORQ-12 (8b1419f5...): Mantidos em `in_progress`.
- Zero alteração de assignee.

2. EXCEÇÕES DOS NOVOS DISPATCHES (ORQ15, 23, 26, 33, 34, 36, 37):
- Mantidos em `todo` devido a limitação de provedor/capacidade ou pendência de confirmação de execução ativa.

3. TABELA DE CONTAGEM POR COLUNA (BEFORE vs AFTER):
- BEFORE: blocked=5, done=5, in_progress=4, in_review=2, todo=8.
- AFTER:  blocked=5, done=6 (+1), in_progress=2 (-2), in_review=3 (+1), todo=8.
- Active Queue Count: 0 (Confirmado zero task duplicada).

4. SHA-256 DO RELATÓRIO:
- SHA-256: 5a1210aca5ceb4c031714a18ac3ffb30828590e0c5d0eaf4bc953289138a7868

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gtl-status-actions-reconciliation.md.
