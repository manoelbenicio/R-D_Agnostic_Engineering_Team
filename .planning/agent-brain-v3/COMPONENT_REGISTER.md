# COMPONENT_REGISTER — Agent Brain v3 (retain/rename/replace/retire com owner)

> Nenhum componente desaparece sem replacement + migration + evidence + rollback impact + aprovação.

| Componente | Estado target | Owner | Replacement/gate | Evidence | Phase |
|---|---|---|---|---|---|
| daemon.go (central orchestrator) | RETAIN + extract neutral boundaries | Codex1 (sole) | strangler p/ brain pkg; não rewrite | EV-G3-WIRE | G2A/G3 |
| config.go (ProdexConfig/L2RuntimeConfig...) | RETAIN + neutral config names | Codex1 (sole) | neutral gateway config + aliás | EV-G1-02 | G1/G3 |
| health.go (HealthResponse) | RETAIN + neutral + redacted | Codex1 (sole) | neutral readiness; remove prodex fields? gate G6 | EV-G3-04 | G1/G6 |
| execenv/execenv.go + codex_home.go | RETAIN + credentialless sanitizer | Codex3 + Codex1 hotspot | remove provider-auth copy; pre-launch assert | EV-G3-01/03 | G2C |
| pkg/agent/models.go | RETAIN; separa CLIKind×RouteModel | Codex1 (hotspot) | neutral model map | EV-G1-02 | G1 |
| cmd/multica/cmd_daemon.go | RETAIN; neutral entrypoint | Codex1 (hotspot) | wiring gateway-required | EV-G3-WIRE | G3 |
| pkg/agent/{claude,codex,kimi,nim,antigravity}.go | RETAIN/REPLACE adapters credentialless | Codex3 | OmniRoute base URL/key only; NIM→gateway | EV-G4-ADP | G2C/G4 |
| OmniRoute container (diegosouzapw/omniroute:@digest TBD) | RETAIN hot plane; fix digest | Codex4 (ops) | digest pin (PD-02) | EV-G2D-02 | G0/G2D |
| Prodex Rust sidecar (multica-auth-work/prodex-sidecar) | RETIRE BY DECISION after parity | — | OmniRoute parity gate (G5) | EV-G6-03 | G6 |
| l2_runtime.go (rpp.l2.v1 adapter) | RETIRE BY DECISION after parity | Codex1 | disable under gateway-required; delete G6 | EV-G6-03 | G3/G6 |
| prodex.go / prodex_fs_*.go / prodex_profiles.go | RETIRE BY DECISION after zero-use | Codex1 | disable for gateway-required tasks; delete G6 | EV-G6-03 | G6 |
| legacy Go rotation/account-selection | RETIRE BY DECISION after cutover | Codex1 | disable gateway-required; delete G6 | EV-G6-03 | G6 |
| compatibility facade (legacy API/env/config aliases) | RETAIN bounded then RETIRE | Codex1 | delete após zero-use (telemetry) | EV-G2A-03 | G1/G6/G8 |
| brain (neutral pkg) | CREATED; wiring pending | Codex1 | G3 central integration | EV-G2A-01..05 | G2A/G3 |
| gateway (OmniRoute client pkg) | CREATED; wiring pending | Codex2 | G3 central integration by Codex1 | EV-G2B-01..07 | G2B/G3 |
| runtimeenv (credentialless env pkg) | CREATED; wiring pending | Codex3 | G3 central integration by Codex1 | EV-G2C-01..10 | G2C/G3 |
| cli (CLIKind adapters) | CREATE | Codex3 | — | EV-G4-ADP | G2C |

## Observação — worktree não-commitada (PD-01)

Os arquivos `prodex.go (M)`, `prodex_fs_linux.go (??)`, `prodex_fs_other.go (??)`,
`prodex_profiles.go (??)` e modificações em `daemon.go/config.go/health.go/l2_runtime.go`
não-commitadas pertencem ao change **`persist-prodex-runtime-integration`** e foram
adotadas como baseline de segurança ativo pela PD-01.
PD-01 foi resolvida por preservação: esse diff é baseline auditável de segurança, com ownership
exclusivo Codex1 após o lock G1. Nenhum reset/stash/revert é permitido; eventual retirada dos
componentes Prodex continua gateada por replacement, evidence e rollback em G6.
