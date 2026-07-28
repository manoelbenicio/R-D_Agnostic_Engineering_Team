# Codex56-TL — board reconciliation + durable ledger pick
- MODE: READ-ONLY; nenhum codigo, build, restart, API write ou status de board alterado neste despacho.
- ORQ-22 ja esta `done`; evidencia literal: `legacy-daemon-security-remediation-20260727.md:38-45` fixa legado desativado e T2 vigente.
- MEDIDO: `ORQ-14 blocked/completed/0`, `ORQ-19 blocked/completed/0`, `ORQ-20 blocked/completed/0`, `ORQ-25 in_progress/completed/0`, `ORQ-26 in_progress/none/0`; ORQ-22 `done/failed/0` e historico intencional.
- CODIGO: `task.go:1881-1883` diz `ReconcileFailedIssue resets an in-progress issue to todo after its final active task fails`; portanto nao reconcilia completed→review/done nem blocked→review.
- RULING: nao mutar 14/19/20 so pelo ultimo `completed` (aceite pode faltar); ORQ-25 e o unico estado falso candidato imediato; ORQ-26 segue trabalho manual ativo.
- LEDGER PICKED: `daemon.go:301` literal `d.commitLedgers = commitledger.NewLedgerRegistry()`; `main.go:333` cria `NewTaskService(...)` sem wiring do hook.
- GATE LITERAL: `task.go:38-42` declara `Nil fails closed (blocks all retries)` e `ReplayGateHook *commitledger.ReplayGateHook`.
- NEXT WRITE (quando liberado): implementar `ledger-durable-design.md`; ORQ-26/UI, custo e skills ficam enfileirados, sem overlap neste despacho.
- CHECK-OUT: DONE, progress=100, build_result=NOT_RUN, Codex56-TL w5:pC @ 2026-07-27T11:04:25Z.
