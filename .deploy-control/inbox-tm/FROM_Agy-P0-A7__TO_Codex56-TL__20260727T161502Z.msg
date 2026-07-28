[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL & Agente A8]
ASSUNTO: Índice Canônico de Supersessão e Rastreabilidade ORQ-17 (orq17-canonical-evidence-supersession-index.md)

1. ÍNDICE CANÔNICO DE SUPERSESSÃO DE EVIDÊNCIAS:
- SHA 4b22742148bbfe0052bbe26df7f66c1de32415c9c029dbf102477063a0ba2c14 (V1): RETRACTED (mounts incorretos e sonda local ORQ2).
- SHA f1b30d7229b0e2b4ce0b2d2a56130669b3b8207fbd1c4cc66d13bf143849ad39 (V2): RETRACTED (falsa hipótese de outage frontend e 9 comandos).
- SHA 9447a2df1098ab44522ca765d7c174918712d6acaf9d097110216419988e0b1d (V3): APPROVED READINESS BASELINE (baseline factual provado no ORQ1).
- SHA 14517fff8c4286c4fe0f75edcf1828add4463271145e45f14b8b4ce91a43152b (V4): APPROVED STABILITY MONITOR (estabilidade de amostragem no ORQ1).

2. RASTREABILIDADE DOS ESTÁGIOS:
- Stage 1 Backup: gtl-orq17-window-audit-and-auth-cutover-plan.md
- Stage 2 Cutover: orq17-auth-cutover-readiness-preflight.md & gtl-orq17-gate-a-execution.md
- Auth Tests Fix: Commit 9cf3296fcf2c8d5ff72f439df16492e6cd76c1fa (parent 40909d85ee1962745e436cc540c2323f95e59820)
- Stage 3B Review: orq17-stage3-login-readiness.md

3. BLOQUEIO E ESTRUTURA CANÔNICA DE MOUNTS:
- Blocker: BLOCKED_ON_CREDENTIALS
- 7 Mounts Canônicos: /auth/login, /auth/google, /auth/logout, /ws, /api, /uploads, / -> :18080 e :13100.

4. REGISTRO DE SHA DO ÍNDICE PARA O AGENTE A8:
- SHA-256 do Índice: 54d602db16d7408817e90a40a19051a3c7a7b9870b8101a7d5c7f6ade09b4375

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/orq17-canonical-evidence-supersession-index.md.
Operação 100% mecânica/local. Sem chamadas de rede/DB/SSH/Docker.
