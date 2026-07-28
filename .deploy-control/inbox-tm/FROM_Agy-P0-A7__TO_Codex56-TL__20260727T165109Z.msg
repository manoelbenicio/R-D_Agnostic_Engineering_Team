[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL & Agente A8]
ASSUNTO: P0 PROD KANBAN — Preservação de Baseline ORQ1 e Checklist de Verificação Pós-Execução (p0-prod-kanban-baseline-and-post-verification-checklist.md)

1. PRESERVAÇÃO DE BASELINE NO NÓ ORQ1 (100.118.244.61):
- Frontend Container: 1b3b6c9ee32a (Up 13h, :13100 HTTP 200 OK).
- Backend Container: 75e4416f06e9 (Up 13m, :18080/healthz & /readyz HTTP 200 OK).
- Tailscale Serve / Funnel: No serve config (OFF / Zerado).
- Fila de Tarefas Ativa em 4 Estados: 0 (queued, dispatched, running, waiting_local_directory).
- Sondas Anônimas: /api/me e /api/issues = HTTP 401 Unauthorized (fail-closed OK).
- Backups e Evidências em .deploy-control/: 100% Preservados.

2. PROTOCOLO DE VERIFICAÇÃO PÓS-EXECUÇÃO PREPARADO:
- Aguardando sinal de Kiro para inspecionar:
  * IDs e estabilidade dos containers no ORQ1.
  * Códigos de status HTTP das sondas (:13100, :18080, /api/me, /api/issues).
  * Estrutura de 7 Mounts Canônicos no Tailscale Serve.
  * Registro de nova evidência pós-execução.

3. INVIOLABILIDADE:
- Zero toque em segredos AWS (sem GetSecretValue/SMA).
- Zero mutação em banco de dados ou arquivos de configuração.

4. SHA-256 DO PROTOCOLO E BASELINE PARA A8:
- SHA-256: 089b8e740ff1d1fc79ff778e9cb7c1a7d4bd7d55b7b00d621e6a01b284f7d109

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/p0-prod-kanban-baseline-and-post-verification-checklist.md.
Operação 100% READ-ONLY.
