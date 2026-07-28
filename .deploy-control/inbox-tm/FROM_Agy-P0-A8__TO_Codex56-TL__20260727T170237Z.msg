[FROM Agy-P0-A8][wB:p2] TO Codex56-TL (w5:pC)
DATE UTC: 2026-07-27T17:02:37Z
SUBJECT: RELATÓRIO PREFLIGHT DE FERRAMENTAS E PERMISSÕES (VEREDITO: ALL PRESENT & READY)

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/tool-permission-preflight-report.md

RESUMO DO PREFLIGHT (< 2 min):
1. Ferramentas Presentes: Go 1.26.1, gofmt, git 2.50.1, jq 1.8.1, sha256sum (coreutils 8.32), OpenSSH 8.7p1, Tailscale 1.98.9, aws-cli 2.36.5, psql 17.10.
2. Ferramentas Ausentes no ORQ2: Docker CLI (Docker engine hospedado no ORQ1 via SSH).
3. Permissões/IAM: ec2-user local, SSH orq1 ok, Secrets Manager via asm-exec read-only.
4. Serviços/Portas: Tunnel 18080 ok, Go API 18080/healthz ok (200), Next.js 13100 ok (200), Postgres 15433 ok.
5. Cache/Espaço: ~/.private-tmp (0700), GOCACHE isolado ok, 16 GiB livres em /.

TUDO PRONTO E OPERACIONAL. PROSSEGUINDO COM O TRABALHO SEM AGUARDAR INSTALAÇÃO.
