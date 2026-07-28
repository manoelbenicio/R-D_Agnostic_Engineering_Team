[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Reconciliação de Ownership e Visibilidade do GTL (V1) — ORQ-12 In_Review, Notas EXT-OWNER-V1 Publicadas (gtl-ownership-reconciliation-v1.md)

1. TRANSIÇÃO DE STATUS DO ORQ-12:
- ORQ-12 (8b1419f5...): Transicionado status-only de `in_progress` para `in_review` (GET-back 100% Verificado). Assignee NULL preservado.

2. NOTAS IDEMPOTENTES PUBLICADAS (EXT-OWNER-V1):
- ORQ15 -> `EXT-OWNER-V1:ORQ15:w7:p4` (Codex56-B)
- ORQ23 -> `EXT-OWNER-V1:ORQ23:wN:p1` (AGY-3.6-Flash)
- ORQ34 -> `EXT-OWNER-V1:ORQ34:w8:p2` (Opus48-D)
- ORQ36 -> `EXT-OWNER-V1:ORQ36:wN:p2` (AGY-3.6-Flash)
- ORQ37 -> `EXT-OWNER-V1:ORQ37:w7:p3` (Codex56-A)
- ORQ26 -> Confirmado assignee DB `5b336637` (Agy-P0-A8), sem nota duplicada.

3. TABELA DE CONTAGEM POR COLUNA (BEFORE vs AFTER):
- BEFORE: blocked=5, done=6, in_progress=7, in_review=5, todo=1.
- AFTER:  blocked=5, done=6, in_progress=6 (-1), in_review=6 (+1), todo=1 (ORQ35).
- Active Task Queue Count: 0 (Confirmado zero task duplicada).

4. REGISTRO DA POLÍTICA FUTURA DE DESPACHO:
- Novos despatches utilizarão atribuição explícita de `assignee` via API e exatamente 1 product task (sem status-only + Herdr paralelo).

5. SHA-256 DO RELATÓRIO:
- SHA-256: 39596ab3272c58927786eb8a41dfaf94a00fcc2943da03a5dcecc44d3d98979b

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gtl-ownership-reconciliation-v1.md.
