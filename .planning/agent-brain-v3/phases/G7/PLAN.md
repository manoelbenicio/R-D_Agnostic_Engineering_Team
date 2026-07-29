# PLAN — G7: Capacidade 50/100 + state decision

gate saída: só o maior tier comprovado habilitado; decisão single-node vs compartilhado resolvida.

## Tasks (tasks.md §9)
- 9.3 [Codex4] Run 50-task sustained/recovery profile; documentar tuning
- 9.4 [Codex1] Habilitar tier 50 só se evidence passar; senão enforce 20 + remediation
- 9.5 [Codex4] Run 100-task sustained/recovery + bounded overload OOM
- 9.6 [Codex1] Habilitar tier 100 só se evidence passar; senão enforce highest proven
- [Dono] Decisão single-node vs backend compartilhado (PD-04) antes tier 100/horizontal
- 9.7 [Codex4] Dashboards/alerts/backup/restore/secret rotation/incident/upgrade-rollback/owner sign-off
Evidence: EV-G7-50, EV-G7-100, EV-G7-state

Pré-requisitos: G6. STATUS: NOT_STARTED. Não autorizado no escopo inicial.
