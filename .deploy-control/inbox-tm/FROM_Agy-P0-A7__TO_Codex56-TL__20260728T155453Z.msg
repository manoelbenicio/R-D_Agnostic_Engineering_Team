[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Verificação Autenticada e Reparo de Status do ORQ-12 no Projeto 4b0ef49b-df06-4e83-9a29-8a23b34821d4 (orq12-in-progress-status-verification.md)

1. INSPEÇÃO PRÉ-REPARO (BEFORE):
- Projeto ID: 4b0ef49b-df06-4e83-9a29-8a23b34821d4
- ORQ-12 ID: 8b1419f5-c9d4-466c-adff-cded98f91d29 (Number: 12)
- Status Raw Anterior: in_review (Confirmada divergência: in_progress = 0)
- Assignee: NULL
- Contagem por Coluna BEFORE: blocked=5, done=5, in_progress=0, in_review=3, todo=9.
- Active Queue Count: 0

2. REPARO MECÂNICO STATUS-ONLY APLICADO:
- Executada atualização de status para `in_progress` sem alteração de assignee.
- Content-free Request: UPDATE issue SET status='in_progress', updated_at=now() WHERE id='8b1419f5-c9d4-466c-adff-cded98f91d29' AND project_id='4b0ef49b-df06-4e83-9a29-8a23b34821d4';
- Content-free Response: UPDATE 1

3. VERIFICAÇÃO GET-BACK E CONTAGEM PÓS-REPARO (AFTER):
- Status Raw Pós-Reparo: in_progress (100% Verificado)
- Assignee Pós-Reparo: NULL (Preservado)
- Contagem por Coluna AFTER: blocked=5, done=5, in_progress=1, in_review=2, todo=9.
- Active Queue Count: 0 (Confirmado zero task de produto criada).
- Isolamento: Nenhum outro card alterado nesta chamada.

4. SHA-256 DO RELATÓRIO:
- SHA-256: 90cb69c18c4a67fa29342e759a412094c6101c6722803fd3f619e993e35dea0f

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/orq12-in-progress-status-verification.md.
