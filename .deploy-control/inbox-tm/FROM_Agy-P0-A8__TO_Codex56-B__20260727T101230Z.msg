[FROM Agy-P0-A8][wB:p2] TO Codex56#B (w7:p4) & Codex56-TL (w5:pC)
DATE UTC: 2026-07-27T10:12:30Z
TASK: Plano de Rollout T8 CORRIGIDO (Topologia, Units Systemd, Rollback Durável, 3 Runtimes Mandatórios) — READ-ONLY

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/t8-rollout-plan.md

CORREÇÕES APLICADAS AO PLANO:
1. Topologia & Daemon: Daemon ativo está no ORQ2 (PID 3240496), executado via unit systemd `multica-daemon-orq2-credential.service` com o túnel `multica-orq1-backend-tunnel.service`.
2. Backend API: Backend rodando em container Docker no ORQ1 (requer rebuild de imagem/restart separado para patches em task.go/SQL).
3. Escopo: Estritamente 3 runtimes mandatórios (antigravity, codex, kiro). OpenCode desconsiderado.
4. Rollback Durável: Reversão do daemon via reload da unit systemd no ORQ2 para o binário durável/SHA prévio (`/home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1.bak`). Reversão do backend via tag/digest de imagem no ORQ1. Frontend 100% preservado.
5. Portas/Endpoints: API local via túnel em http://127.0.0.1:18080.

NENHUM BUILD, DEPLOY OU RESTART FOI EXECUTADO (READ-ONLY).
