[FROM Agy-P0-A8][wB:p2] TO Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
DATE UTC: 2026-07-27T13:38:31Z
TASK: ORQ-33 Runbook de Rotação Segura do Rev Token (VEREDITO: PASS)

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/orq33-rev-token-rotation-runbook.md
- Issue Citada: ORQ-33 (dfeabbdc-33e1-4ab8-9460-27b43df227db)

RESUMO DO RUNBOOK:
1. Secret Safety: 100% aderente à skill aws-secrets-manager (zero chamadas GetSecretValue, zero plaintext em context/logs, uso exclusivo de asm-exec com {{resolve:secretsmanager:...}}).
2. Topologia: Emissores (Go Backend Auth Handler, Session Manager) e Consumidores (Auth Middleware, pat_cache.go, membership_cache.go, Daemons).
3. Rotação Zero-Downtime: Estratégia de dupla chave (Primary/Secondary) com janela de sobreposição e drenagem de 15 minutos (TTL do cache).
4. Gates Fila Zero: active_tasks == 0 em agent_task_queue, resolução IAM via asm-exec dry-run e healthz 200 OK.
5. Rollback & Auditoria: Restauração imediata de Primary via asm-exec em falha >0.01%, com auditoria CloudTrail garantindo zero leitura manual.

MODO LEITURA APENAS. ZERO LEITURA DE SEGREDOS, ZERO MUTAÇÃO EM CÓDIGO OU AMBIENTE VIVO.
