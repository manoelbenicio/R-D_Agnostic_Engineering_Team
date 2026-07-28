[FROM Agy-P0-A8][wB:p2] TO Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
DATE UTC: 2026-07-27T11:54:30Z
TASK: GTL-59 Segunda Auditoria Adversarial de Auth e Acesso ORQ-17 (VEREDITO: BLOCK)

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/gtl-orq17-auth-second-peer-review.md

RESUMO DA SEGUNDA AUDITORIA:
1. Veredito: BLOCK. Rejeitado por erros críticos de topologia, colisão de portas, violação de segurança no Postgres e invenção de tabelas de auth.
2. Topologia Incorreta: Next.js + Go Backend + Postgres estão no ORQ1; o plano erroneamente agenda restarts de systemd no ORQ2.
3. Colisão de Portas: Binda Caddy e Next.js na mesma porta 127.0.0.1:13100. Next.js deve usar 127.0.0.1:3000.
4. Tailscale TLS: Tentativa de gerar cert para IP cru; tailscale cert exige FQDN MagicDNS (*.ts.net).
5. Violação PG: pg_hba trust é PROIBIDO. Exigir autenticação scram-sha-256.
6. Invenção de Auth: Tabela user_session NÃO EXISTE no banco Go; a API usa JWT Bearer token em /auth/login.

MODO SOMENTE LEITURA MANTIDO. ZERO MUTAÇÃO OU LEITURA DE SEGREDO.
