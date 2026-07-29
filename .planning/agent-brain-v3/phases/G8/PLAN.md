# PLAN — G8: Debrand completo

gate saída: sem dependência runtime Multica/Prodex ativa; documentação final reconciliada.

## Tasks (tasks.md §11)
- 11.1 [Codex1] Inventory todo nome Multica/Prodex por API/CLI/env/path/pkg/type/storage/metric/log/UI
- 11.2 [Codex1] Introduzir final names p/ binário/module/package/config/contracts sem quebrar compat consumidores
- 11.3 [Codex2] Rename gateway-facing legacy IDs/RouterOwner values; read-compat p/ stored historical records
- 11.4 [Codex3] Rename task-home/runtime-brief/CLI-facing paths/variables com migration determinística de local state
- 11.5 [Codex4] Migrar deploy/dashboards/alerts/docs/runbooks/operator procedures aos final names
- 11.6 [Codex1] Remover cada alias só após telemetry zero-use + consumer migration complete + release notes
- 11.7 [Codex1+4] Final sign-off architecture/parity/security/capacity/ops sem Multica/Prodex runtime dependency
Evidence: EV-G8-01..03

Pré-requisitos: G7. STATUS: NOT_STARTED. Nome definitivo (PD-05) é gate.
