[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Runbook de Rotação Segura de Handshake Token ORQ-32 (orq32-handshake-token-rotation-runbook.md)

1. CONFORMIDADE COM SKILL AWS-SECRETS-MANAGER:
- Garantido zero acesso/impressão de texto claro (proibição de GetSecretValue).
- Resolução dinâmica exclusiva via asm-exec com {{resolve:secretsmanager:prod/handshake-token:SecretString:token:...}}.

2. ARQUITETURA E TOPOLOGIA DUAL-TOKEN:
- Consumidores mapeados: Go API Server (cmd/server), Daemon Workers (cmd/multica), Multica Fleet Gateway.
- Fases de transição zero-downtime: Stage AWSPENDING -> Enable Dual Server (Primary+Secondary) -> Daemon Fleet Cutover -> Promote AWSCURRENT.

3. GATES E ROLLBACK:
- Queue-Zero Gate: Drenagem completa de tarefas ativas (running tasks = 0).
- Rollback: Reversão imediata dos daemons para AWSPREVIOUS mantendo o servidor em modo dual.

4. VEREDITO FINAL:
- STATUS: PASS (RUNBOOK ORQ-32 APROVADO).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/orq32-handshake-token-rotation-runbook.md.
Operação 100% Read-Only.
