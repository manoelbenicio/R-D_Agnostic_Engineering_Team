[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Retificação Factual do Pré-Flight ORQ-17 (orq17-auth-cutover-readiness-preflight.md)

1. RETIFICAÇÃO FACTUAL DE PORTAS E ESTADO DO BACKEND:
- Backend verificado na porta real 18080: ONLINE E SAUDÁVEL. GET /readyz e GET /healthz retornam HTTP 200 OK. GET /api/config confirma allow_signup: true.
- Retratada a afirmação infundada de que prod/owner-credentials seria o ID de segredo canônico. A resolução ocorre via asm-exec no processo filho.
- Frontend verificado na porta real 13100: Retorna Connection Refused (processo web inativo).

2. TAILSCALE SERVE E BUNDLE NEXT.JS:
- Tailscale Serve atualmente desconfigurado ("No serve config").
- NEXT_PUBLIC_API_URL exige recompilação do container Web se for alterado para https://orq1.tail96e2c0.ts.net.

3. VEREDITO FINAL RETIFICADO:
- STATUS: BLOCK (PENDENTE DE FRONTEND NA PORTA 13100 E CONFIGURAÇÃO DO TAILSCALE SERVE).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/orq17-auth-cutover-readiness-preflight.md.
Operação 100% Read-Only.
