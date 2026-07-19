# REMOVAL_REGISTER — Agent Brain v3 (retirada precedida por replacement/migration/ev/rollback)

> Toda remoção exige: replacement + migration + evidence + rollback impact + aprovação.
> Nenhuma remoção por rename/cleanup genérico. Nada removido é silencioso.

| Item | Replacement | Migration | Evidence gate | Rollback impact | Quando (gate) |
|---|---|---|---|---|---|
| Prodex Rust sidecar (l2_runtime.go rpp.l2.v1 + sidecar) | OmniRoute hot plane completo (P01-P34) | drain + delete after zero-use | matriz paridade assinada (G5) | volta a Agent Brain/OmniRoute prev, NUNCA provider keys | G6 (após G5 + cutover) |
| prodex.go / prodex_fs_*.go / prodex_profiles.go | OmniRoute + neutrality | disable gateway-required → delete | zero-use telemetry | idem | G6 |
| Legacy Go rotation/retry/account-selection | OmniRoute rotation (AB-REQ-09..13) | disable gateway-required → delete | failure-injection G4 + zero-use | idem | G6 |
| Provider-auth copying in per-task homes | credentialless env (AB-REQ-16/19) | G2C contract complete; active-path disable/removal wired only by Codex1 in G3 | pre-launch assertion (8.3) | idem | G3 (disable) + G6 (delete path) |
| Provider-native NIM key path | OmniRoute key in NIM slot (no overwrite) | convert NIM→gateway adapter | NIM adapter proof | idem | G2C/G4 |
| Multica-branded names (binário/API/env/path/pkg/storage/metrics/UI) | final product names (PD-05) | rename via strangler; aliases tmp | zero-use aliases + consumer migration | aliases retained until rollback-free | G8 |
| Compatibility facade/aliases | neutral contracts | migrate consumers | zero-use telemetry | aliases só removidos quando rollback não depende | G8 |
| `persist-prodex-runtime-integration` change (worktree) | baseline transitório seguro; garantias absorvidas antes da retirada Prodex | PD-01: preservar, auditar, concluir e testar sem reset | 16 tasks OpenSpec + testes de overwrite/fallback/restart | mantém rollback seguro enquanto Agent Brain não substitui Prodex | G1–G3; eventual retirada somente em G6 |
| R01 Caveman/plugin hooks | disabled (RCE boundary) | keep disabled | sandboxed future design | n/a | G-R (retire by decision) |
| R02 Prodex terminal rendering | CLI/product UI own presentation | confirm no operator workflow lost | review | n/a | G-R |
| R03 Prodex SQLite/shared state | OmniRoute supported state backend (P23) | authoritative state migrate | backup/restore proof (G4) | — | G-R/G7 |
| R04 RPP L2 sidecar contract/binary | OmniRoute contract ingest | after parity evidence | G5 ev | rollback via Agent Brain/OmniRoute | G-R/G6 |
| R05 Legacy Go cred rotation/provider homes | credentialless | after gateway-required proven | zero-use | idem | G5/G6 |

STATUS: nenhum item acima está REMOVIDO hoje. Todas as remoções são gateadas. Prodex permanece vivo (não autorizado remover) até G6.
