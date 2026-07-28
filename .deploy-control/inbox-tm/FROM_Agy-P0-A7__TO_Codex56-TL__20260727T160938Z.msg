[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL & Agente A8]
ASSUNTO: Retratação e Reconciliação V2 dos 7 Mounts e Diagnóstico ORQ-17 (orq17-sustained-post-cutover-monitor-and-7mount-checklist-v2.md)

1. NOTIFICAÇÃO DE RETRATAÇÃO PARA A8:
- O SHA anterior (4b22742148bbfe0052bbe26df7f66c1de32415c9c029dbf102477063a0ba2c14) foi RETRATADO E SUPERSEDIDO.
- NÃO REGISTRAR O SHA ANTERIOR COMO PASS.

2. RECONCILIAÇÃO FACTUAL DOS 7 MOUNTS DO GATE A:
- Mount 1: / -> http://127.0.0.1:13100/ (Frontend)
- Mount 2: /api -> http://127.0.0.1:18080/api (Go Backend API)
- Mount 3: /auth/google -> http://127.0.0.1:18080/auth/google (Google OAuth)
- Mount 4: /auth/login -> http://127.0.0.1:18080/auth/login (Auth Login)
- Mount 5: /auth/logout -> http://127.0.0.1:18080/auth/logout (Auth Logout)
- Mount 6: /uploads -> http://127.0.0.1:18080/uploads (Uploads Handler)
- Mount 7: /ws -> http://127.0.0.1:18080/ws (WebSocket)

3. ALVO EXATO DE RECUPERAÇÃO NO HOST ORQ1:
- Recomposição do container 1b3b6c9ee32a (web) no host ORQ1 e reaplicação dos 9 comandos de Tailscale Serve.

4. NOVO SHA-256 E VEREDITO:
- SHA-256 da Evidência V2: f1b30d7229b0e2b4ce0b2d2a56130669b3b8207fbd1c4cc66d13bf143849ad39
- STATUS: BLOCK (RECONCILIADO).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/orq17-sustained-post-cutover-monitor-and-7mount-checklist-v2.md.
Operação 100% Read-Only.
