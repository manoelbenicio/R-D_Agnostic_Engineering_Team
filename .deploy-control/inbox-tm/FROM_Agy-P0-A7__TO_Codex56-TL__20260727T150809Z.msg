[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Pre-Flight Factual da Toolchain Wave 0 ORQ-13 (orq13-wave0-toolchain-preflight.md)

1. TOOLCHAIN E VERSÕES:
- Go: /home/ec2-user/goroot/go/bin/go (go1.26.1).
- sqlc: /home/ec2-user/.cache/sqlcbuild/sqlc (v1.31.1). Requer apenas export PATH=/home/ec2-user/.cache/sqlcbuild:$PATH.
- golangci-lint: Ausente (binário e .golangci.yml não encontrados). Substituto nativo aprovado: gofmt -l, git diff --check, go vet.

2. CONFIGURAÇÕES E PASTA PRIVADA:
- Config: sqlc.yaml presente em multica-auth-work/server/sqlc.yaml.
- Temp/Cache Privado: /home/ec2-user/.private-tmp existente em modo 0700.

3. VEREDITO FINAL:
- STATUS: PASS (PRÉ-FLIGHT FACTUAL DA TOOLCHAIN CONCLUÍDO).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/orq13-wave0-toolchain-preflight.md.
Operação 100% Read-Only. Zero instalações, zero downloads, zero edições.
