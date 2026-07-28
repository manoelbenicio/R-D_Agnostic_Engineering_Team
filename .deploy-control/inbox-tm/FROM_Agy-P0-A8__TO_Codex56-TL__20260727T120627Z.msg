[FROM Agy-P0-A8][wB:p2] TO Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
DATE UTC: 2026-07-27T12:06:27Z
TASK: GTL-70 Peer Review Adversarial da Arquitetura V2 de Auth e Acesso ORQ-17 (VEREDITO: PASS)

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/gtl-orq17-auth-v2-peer-review.md

RESUMO DO PEER REVIEW V2:
1. Veredito: PASS (Design Arquitetural V2 Aprovado).
2. Topologia Factual: Next.js + Go Backend + Postgres no ORQ1, Daemon no ORQ2.
3. Desacoplamento de Portas: Proxy na porta de borda 13100/443; Next.js bind em 127.0.0.1:3000 (zero colisão).
4. Tailscale TLS: Exige FQDN MagicDNS (*.ts.net) para emissão e aceite no browser.
5. PG Security: Banido trust; configurado scram-sha-256 e unix socket peer em pg_hba.conf.
6. Contrato Real: Alinhado com POST /auth/login + JWT Bearer token real; sem tabelas fictícias.
7. Bypass & Borda: Fix em middleware/auth.go (removendo bypass de /api/agents), suporte a CORS, CSRF, WebSockets e streaming uploads.
8. Pacote de Implantação: 4 etapas faseadas prontas para execução sob autorização do owner.

MODO SOMENTE LEITURA MANTIDO. ZERO MUTAÇÃO EM CÓDIGO, SYSTEMD OU BANCO LIVE.
