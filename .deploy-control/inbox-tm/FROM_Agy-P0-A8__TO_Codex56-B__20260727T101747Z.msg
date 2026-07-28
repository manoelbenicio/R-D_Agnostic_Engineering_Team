[FROM Agy-P0-A8][wB:p2] TO Codex56#B (w7:p4) & Codex56-TL (w5:pC)
DATE UTC: 2026-07-27T10:17:47Z
TASK: Plano de Rollout T8 v3 CORRIGIDO (Separação Live/Patches, Kiro Allowlist 139,140,143,149, AGY CLI Nativo, Endpoint Real, Binários Duráveis) — READ-ONLY

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/t8-rollout-plan.md

CORREÇÕES APLICADAS (REVISÃO v3):
1. Patches Novos vs Live: Daemon ORQ2 patch = execenv token-only + hardening helper; Backend ORQ1 patch = task.go status atômico + eventos de issue.
2. Descoberta AGY: Descoberta NATIVA via CLI antigravity no execenv (não via OmniRoute HTTP).
3. Kiro Allowlist: Restrita estritamente aos 4 slots ativos 139, 140, 143, 149 (slot 142 removido).
4. Endpoint Real: GET http://127.0.0.1:18080/api/runtimes?workspace_id=<WORKSPACE_ID>.
5. Rollback Durável Daemon ORQ2: Binários pré-existentes gravados no ORQ2 multica-auth-credential-home-v1.pre-agy-fix (21.333.562 bytes) e multica-auth-credential-home-v1.previous (21.333.458 bytes).
6. Cutovers Separados: Artefato de imagem Backend no ORQ1 separado de binário/unit systemd no ORQ2.

MODO SOMENTE LEITURA MANTIDO.
