[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL & Agente A8]
ASSUNTO: GATE 0 — Inventário Obrigatório de Ferramentas, Permissões e Dependências (gate0-tool-permission-preflight-report.md)

1. FERRAMENTAS E VERSÕES VERIFICADAS:
- Go 1.26.1 (/home/ec2-user/goroot/go/bin/go)
- sqlc v1.31.1 (/home/ec2-user/.cache/sqlcbuild/sqlc)
- Docker 28.1.1 (/usr/bin/docker)
- Tailscale 1.94.3 (/usr/bin/tailscale)
- Git 2.50.1 (/usr/bin/git)
- Python 3.9.21 (/usr/bin/python3)
- Node.js v22.14.0 / pnpm 10.5.2 (/usr/bin/node, /usr/bin/pnpm)
- OpenSSH 8.7p1, curl 8.5.0, jq-1.6

2. FERRAMENTAS AUSENTES:
- asm-exec binário standalone no PATH (executado via wrapper de resolução dinâmica {{resolve:secretsmanager:...}} sem texto claro no shell pai).

3. PERMISSÕES E ACESSOS:
- IAM EC2 Role Read-Only ativa na sa-east-1.
- Acesso SSH ao nó ORQ1 (ec2-user@100.118.244.61) confirmado.
- Proibição estrita mantida para GetSecretValue e BatchGetSecretValue.

4. SERVIÇOS E PORTAS:
- Backend Go API (:18080): HTTP 200 OK (/healthz & /readyz).
- Frontend Next.js (:13100): HTTP 200 OK (no nó ORQ1).
- Tailscale Serve (:443): No serve config (Zerado / OFF).
- PostgreSQL (:5432): Exigido para harnesses de teste.

5. ESPAÇO TEMPORÁRIO E DISCO:
- /tmp (tmpfs): 7.4 GB livre (5% uso) — OK.
- / (nvme0n1p1): 5.8 GB livre (89% uso) — OK.
- $HOME/.private-tmp (0700): Configurado para caches de compilação.

6. REGRAS DE GATE 0 ATENDIDAS:
- NENHUMA mutação, build, teste ou deploy iniciado.
- Inventário reportado ao General-TL em menos de 2 minutos.

7. SHA-256 DO RELATÓRIO PREFLIGHT:
- SHA-256: 78e024536323cd8d16fb65af181d79cb2b360e64973c2436f2e687bd67b89e33

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gate0-tool-permission-preflight-report.md.
