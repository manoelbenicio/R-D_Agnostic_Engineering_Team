[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Segunda Reconciliação do GTL — ORQ-36 e ORQ-26 para In_Progress, Lookup de Assignees ORQ-35/37 (gtl-second-reconcile-orq36-orq26.md)

1. TRANSIÇÕES STATUS-ONLY EXECUTADAS (TODO -> IN_PROGRESS):
- ORQ-36 (41645aaf): Transicionado para in_progress (wN:p2 iniciou execução). Assignee NULL preservado. GET-back 100% Verificado.
- ORQ-26 (e966922d): Transicionado para in_progress (wB:p2 iniciou execução real chat-window/TooltipTrigger). Assignee 5b336637 (Agy-P0-A8) preservado. GET-back 100% Verificado.

2. LOOKUP READ-ONLY DE ASSIGNEES (CARDS INTOCADOS):
- ORQ-35 (3f73ff90): Assignee ID 8d9da3ab = Agent `Codex-D`. Status todo mantido.
- ORQ-37 (36d18727): Assignee ID f143017b = Squad `Navy_Seals`. Status todo mantido.

3. TABELA DE CONTAGEM POR COLUNA (BEFORE vs AFTER):
- BEFORE: blocked=5, done=6, in_progress=6, in_review=3, todo=4.
- AFTER:  blocked=5, done=6, in_progress=8 (+2: ORQ36, ORQ26), in_review=3, todo=2 (-2: ORQ35, ORQ37).
- Active Task Queue Count: 0 (Confirmado zero task duplicada).

4. SHA-256 DO RELATÓRIO:
- SHA-256: d1264139191788e030b29fec2d2b396639ecd777b0bed16dffb3a98dbfe41aa0

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gtl-second-reconcile-orq36-orq26.md.
