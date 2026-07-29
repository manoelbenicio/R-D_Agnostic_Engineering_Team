# PLAN — G6: Cutover + Prodex→cold recovery mode (quiesce, não deletar)

gate saída: Prodex fora do hot path sem perda funcional/operacional/rollback; retido default-OFF como recovery mode (D-V3-16).

## Tasks (tasks.md §10)
- 10.1 [Codex1] gateway-required default p/ novas tasks (após gates protocolo/segurança/falha/capacidade)
- 10.2 [Codex4] Observar controlled/default development cohorts; confirmar sem direct-provider,
  dual router ou secret-policy violation
- 10.3 [Codex1] Drain/stop legacy tasks + remover flag legado (após rollback não precisar)
- 10.4 [Codex1/W1] Quiesce Prodex/L2 para cold recovery mode default-OFF, mutuamente exclusivo, operator-gated (NÃO deletar; wire à máquina de estados AB-REQ-41)
- 10.5 [Codex1] Delete legacy Go rotation state/retry/account-selection + provider-auth home prep (zero-use)
- 10.6 [Codex3] Delete cred copy/restore + direct NIM/provider-key paths; reter só credentialless isolation
- 10.7 [Codex4] Reconciliar docs/deploy/threat model/runbooks/evidence; provar rollback só Agent Brain/OmniRoute aceitos
Evidence: EV-G6-01/02, EV-REC-MODE (10.4)

Pré-requisitos: G5 + autorização explícita de cutover/removal do dono. STATUS: NOT_STARTED. Fora do escopo Waves 0-3.
