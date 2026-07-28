[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Plano de Ephemeral DB Runner GTL-61 — gtl-orq26-ephemeral-db-runner.md

1. ARQUITETURA DE RUNNER EFÊMERO:
- Imagem: pgvector/pgvector:pg17 rodando no ORQ1 em porta loopback aleatória (127.0.0.1:<random_port>).
- Identificador Único: orq26-db-runner-<random_hex_8> e rede privada orq26-net-<random_hex_8>.
- Zero toque no DB de produção/live do ORQ2.

2. SEGURANÇA DE SEGREDOS (ZERO-SECRET POLICY):
- Credencial PGPASSWORD e DATABASE_URL geradas e resolvidas estritamente dentro da shell do runner (zero stdout, zero argv, zero logs, zero markdown).

3. TRANSFERÊNCIA E TESTES:
- Transferência do código por commit hash (0cb8aebb5aff79cb430b3740d22fadc53c0116fd / gtl-orq26).
- Migrations completas via go run ./cmd/migrate up e validação via schema_migrations.
- Execução de go test -race -count=1 -json com validação estrita das 8 folhas PASS (S1-S7), reprovando qualquer skip.

4. CLEANUP E AUTORIZAÇÃO:
- Cleanup trap incondicional (docker rm -f, docker network rm, rm -rf gate_dir).
- Declaração de Autorização: execução de mutações Docker no ORQ1 depende de autorização por escrito do Owner humano.

5. VEREDITO FINAL:
- STATUS: PASS (PLANO ARQUITETURAL APROVADO).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gtl-orq26-ephemeral-db-runner.md.
Operação 100% Read-Only. Zero containers/mutações executados.
