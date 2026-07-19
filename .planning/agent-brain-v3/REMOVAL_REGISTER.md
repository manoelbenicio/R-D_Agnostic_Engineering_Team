# REMOVAL_REGISTER — Agent Brain v3 (retirada precedida por replacement/migration/ev/rollback)

> Toda remoção exige: replacement + migration + evidence + rollback impact + aprovação.
> Nenhuma remoção por rename/cleanup genérico. Nada removido é silencioso.

| Item | Replacement | Migration | Evidence gate | Rollback impact | Quando (gate) |
|---|---|---|---|---|---|
| Prodex Rust sidecar (l2_runtime.go rpp.l2.v1 + sidecar) | OmniRoute hot plane completo (P01-P34) | **RETAIN-AS-RECOVERY (D-V3-16)** — quiesce para cold recovery mode default-OFF; NÃO deletar | matriz paridade assinada (G5) + máquina de estados recovery (AB-REQ-41) | volta a Agent Brain/OmniRoute prev; recovery mode mutuamente exclusivo, NUNCA provider keys | G6 (quiesce, não delete) |
| prodex.go / prodex_fs_*.go / prodex_profiles.go | OmniRoute + neutrality | **RETAIN-AS-RECOVERY (D-V3-16)** — disable gateway-required por default; código presente mas OFF; NÃO deletar | zero-use no hot path | idem | G6 (quiesce, não delete) |
| Legacy Go rotation/retry/account-selection | OmniRoute rotation (AB-REQ-09..13) | disable gateway-required → delete | failure-injection G4 + zero-use | idem | G6 |
| Provider-auth copying in per-task homes | credentialless env (AB-REQ-16/19) | G2C contract complete; active-path disable/removal wired only by Codex1 in G3 | pre-launch assertion (8.3) | idem | G3 (disable) + G6 (delete path) |
| Provider-native NIM key path | OmniRoute key in NIM slot (no overwrite) | convert NIM→gateway adapter | NIM adapter proof | idem | G2C/G4 |
| Multica-branded names (binário/API/env/path/pkg/storage/metrics/UI) | final product names (PD-05) | rename via strangler; aliases tmp | zero-use aliases + consumer migration | aliases retained until rollback-free | G8 |
| Compatibility facade/aliases | neutral contracts | migrate consumers | zero-use telemetry | aliases só removidos quando rollback não depende | G8 |
| `persist-prodex-runtime-integration` change (worktree) | re-escopado para cold recovery mode default-OFF (D-V3-16); garantias fail-closed/readiness/isolamento retidas como recovery | PD-01: preservar, auditar, concluir e testar sem reset | 16 tasks OpenSpec + testes de overwrite/fallback/restart | mantém rollback seguro; recovery mode mutuamente exclusivo, nunca hot | G1–G3; RETIDO (não deletado) — quiesce em G6 |
| R01 Caveman/plugin hooks | disabled (RCE boundary) | keep disabled | sandboxed future design | n/a | G-R (retire by decision) |
| R02 Prodex terminal rendering | CLI/product UI own presentation | confirm no operator workflow lost | review | n/a | G-R |
| R03 Prodex SQLite/shared state | OmniRoute supported state backend (P23) | authoritative state migrate | backup/restore proof (G4) | — | G-R/G7 |
| R04 RPP L2 sidecar contract/binary | OmniRoute contract ingest | **RETAIN-AS-RECOVERY (D-V3-16)** — mantido para recovery mode default-OFF | G5 ev | rollback via Agent Brain/OmniRoute | G-R/G6 (quiesce, não delete) |
| R05 Legacy Go cred rotation/provider homes | credentialless | after gateway-required proven | zero-use | idem | G5/G6 |

STATUS: nenhum item acima está REMOVIDO hoje. Todas as remoções são gateadas. **Prodex é RETIDO como cold platform recovery mode default-OFF, mutuamente exclusivo e operator-gated (D-V3-16) — deleção está explicitamente FORA DE ESCOPO; não há mandato ativo de delete/removal de Prodex.** Legacy Go rotation/credential-homes/aliases permanecem removíveis por gate (G6/G8) com replacement/evidence/rollback.
