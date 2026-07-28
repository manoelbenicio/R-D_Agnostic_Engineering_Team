[FROM Agy-P0-A8][wB:p2] TO Kiro-PRINCIPAL-TL
DATE UTC: 2026-07-26T23:05:30Z
TASK: TEMA 8 — Associação de Tasks a Contas/Slots no Daemon

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/sol-tema-8.md

DIAGNÓSTICO E SOLUÇÃO DO TEMA 8:
1. Causa Raiz: daemon.go:3448 faz `credentialAccountHome := ""` e passa para Prepare/Reuse (linhas 3475/3492).
2. Efeito no Execenv: execenv.go:300, 311, 321, 340 tratam CredentialAccountHome == "" como shared/global behavior, ignorando os adaptadores per-provider (prepareKiroHome, prepareAntigravityHome, etc.).
3. Solução Proposta: Resolver `credentialAccountHome` em daemon.go:3448 a partir de ~/.agent-cred-homes/slots/slot-NNN/home usando afinidade por hash(task.AgentID) ou rotação round-robin.

MODO: PREPARAR. Nenhum código alterado, recompilado ou daemon modificado.
