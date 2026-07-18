# PLAN — G6: Cutover + retirada Prodex

gate saída: Prodex removível sem perda funcional/operacional/rollback.

## Tasks (tasks.md §10)
- 10.1 [Codex1] gateway-required default p/ novas tasks (após gates protocolo/segurança/falha/capacidade)
- 10.2 [Codex4] Observar controlled/default development cohorts; confirmar sem direct-provider,
  dual router ou secret-policy violation
- 10.3 [Codex1] Drain/stop legacy tasks + remover flag legado (após rollback não precisar)
- 10.4 [Codex1] Delete Prodex/L2 startup/facade/profile/filesystem + build/runtime dependency (após parity assinada)
- 10.5 [Codex1] Delete legacy Go rotation state/retry/account-selection + provider-auth home prep (zero-use)
- 10.6 [Codex3] Delete cred copy/restore + direct NIM/provider-key paths; reter só credentialless isolation
- 10.7 [Codex4] Reconciliar docs/deploy/threat model/runbooks/evidence; provar rollback só Agent Brain/OmniRoute aceitos
Evidence: EV-G6-01..03

Pré-requisitos: G5 + autorização explícita de cutover/removal do dono. STATUS: NOT_STARTED. Fora do escopo Waves 0-3.
