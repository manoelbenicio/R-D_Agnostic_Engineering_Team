[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Pre-Flight Factual do Cutover de Autenticação ORQ-17 (orq17-auth-cutover-readiness-preflight.md)

1. RESULTADO DOS ACHADOS FACTUAIS NO HOST ORQ1:
- Tailscale Serve: Desconfigurado (retorna "No serve config").
- Containers Loopback: Inativos (backend em :8080 e web em :3000 inacessíveis).
- Credencial do Owner: Ausente no AWS Secrets Manager (segredo prod/owner-credentials não encontrado em sa-east-1).
- Frontend Next.js: Inspeção de runtime-urls.ts confirma que variáveis NEXT_PUBLIC_* são embutidas estaticamente no build do pacote web. Alterar a URL da API exige recompilação do container Web.

2. AÇÕES SEGURA DE DESBLOQUEIO (SAFE NEXT ACTIONS):
- Provisionar segredo de credencial do Owner via Secrets Manager sob autorização (skill aws-secrets-manager, zero texto claro).
- Compilar imagem Web com NEXT_PUBLIC_API_URL=https://orq1.tail96e2c0.ts.net.
- Iniciar containers e aplicar Tailscale Serve.

3. VEREDITO FINAL:
- STATUS: BLOCK (CUTOVER DE AUTENTICAÇÃO REJEITADO POR BLOQUEIOS FACTUAIS).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/orq17-auth-cutover-readiness-preflight.md.
Operação 100% Read-Only.
