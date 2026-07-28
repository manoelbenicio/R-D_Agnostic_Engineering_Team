[FROM Agy-P0-A8][wB:p2] TO Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
DATE UTC: 2026-07-27T11:29:40Z
TASK: GTL-11 Plano de Cutover Durável & Rollback ORQ-23 (Ordem do Owner)

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/gtl-orq23-durable-cutover-plan.md

RESUMO GTL-11:
1. Fonte de Build: Repositório limpo multica-auth-work/server (NÃO usar worktree ff121b28).
2. Rollback Real: Reverter estritamente para o binário do Gate 1 multica-auth-credential-home-v1.gate1-20260727T102815Z (15.053.065 bytes).
3. Binários Proibidos: PROIBIDO usar backups pre-token-only (21.344.766 bytes) ou pre-agy-fix (21.333.562 bytes) pois quebram AGY.
4. Healthchecks: GET http://127.0.0.1:18080/api/runtimes?workspace_id=... para os 3 runtimes mandatórios (antigravity, codex, kiro).
5. Comandos-Gate: D1 (Build staging), D2 (Backend restart Docker), D3 (mv atômico + systemctl --user restart).

MODO SOMENTE LEITURA MANTIDO. ZERO EXECUÇÃO.
