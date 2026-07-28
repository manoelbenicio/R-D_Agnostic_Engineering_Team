[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL & Agente A8]
ASSUNTO: GATE 0 — Retificação Factual do Relatório de Preflight com Atribuição Discriminada por Host (gate0-tool-permission-preflight-report.md)

1. RETIFICAÇÃO FACTUAL DE HOSTS EM CADA LINHA:
- ORQ2 (Host Local de Orquestração ip-172-31-30-9):
  * Partição Raiz /: 60 GB Total | 44 GB Usado | 16 GB Livre (74% Uso - Estado Pós-Expansão).
  * Partição /tmp: 7.7 GB Total | 353 MB Usado | 7.4 GB Livre (5% Uso).
  * Go 1.26.1, sqlc v1.31.1, Tailscale 1.98.9, Git 2.50.1, Python 3.9.21, Node v22.14.0/pnpm 10.5.2.
- ORQ1 (Nó Remoto de Produção 100.118.244.61 / ip-172-31-18-217):
  * Partição Raiz /: 24 GB Total | 20 GB Usado | 4.9 GB Livre (80% Uso).
  * Partição /tmp: 3.8 GB Total | 291 MB Usado | 3.6 GB Livre (8% Uso).
  * Docker 25.0.14, Tailscale 1.98.9, Go 1.26.1.
  * Backend API (:18080) HTTP 200, Frontend (:13100) HTTP 200.

2. ATENDIMENTO INTEGRAL À REGRA DE GATE 0:
- Nenhuma mutação, build, teste ou deploy iniciado.
- Atribuição direta e sem mistura de dados entre ORQ1 e ORQ2.

3. SHA-256 DO RELATÓRIO RETIFICADO:
- SHA-256: b1ca766158b25bf85eed20574b40fb37e02899de2a2bf58c9d7add942a3b2233

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gate0-tool-permission-preflight-report.md.
