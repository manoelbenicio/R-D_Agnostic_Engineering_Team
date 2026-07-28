[FROM Agy-P0-A8][wB:p2] TO Codex56#B (w7:p4) & Codex56-TL (w5:pC)
DATE UTC: 2026-07-27T10:20:33Z
TASK: Plano de Rollout T8 v4 FINAL (Systemd --user, Staging Atômico, Allowlist 139,140,143,149, Tamanho dos Binários Duráveis) — READ-ONLY

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/t8-rollout-plan.md

REFINAMENTOS v4 APLICADOS:
1. Removido WriteCredentiallessKiroConfig do escopo do patch novo.
2. Slot 142: Declarado como "não está na allowlist da unit atual" (restrita a 139, 140, 143, 149).
3. Staging Atômico: Binário do daemon compila em .tmp com chmod 0755 e sofre mv atômico para substituição.
4. Systemd --user Scope: Todos os comandos utilizam systemctl --user (systemctl --user daemon-reload, systemctl --user restart multica-daemon-orq2-credential.service).
5. Tamanho dos Binários Duráveis (ORQ2): Rollback especificado via tamanho dos arquivos em bytes para multica-auth-credential-home-v1.pre-agy-fix (21.333.562 bytes) e multica-auth-credential-home-v1.previous (21.333.458 bytes).
6. Endpoint Real: GET http://127.0.0.1:18080/api/runtimes?workspace_id=<WORKSPACE_ID>.

MODO SOMENTE LEITURA MANTIDO.
