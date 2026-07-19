# FILE_OWNERSHIP — Agent Brain v3 (locks disjuntos; sem edição concorrente de hotspot)

> Dois agentes NÃO editam o mesmo arquivo simultaneamente. Check-in/out em AGENT_LEDGER.

Planning/docs owner: Kiro/Opus-4.8. Operational state/transport/verification owner:
Codex#56#A. Product-code ownership below is unchanged. G3 uses Codex1 as the only editor of
central wiring; Codex2–4 may review or produce evidence only in their disjoint scopes.

## Hotspots com owner ÚNICO (não-concorrente)

| Path | Owner único | Observação |
|---|---|---|
| multica-auth-work/server/internal/daemon/daemon.go | Codex1 | integrador-líder |
| multica-auth-work/server/internal/daemon/config.go | Codex1 | |
| multica-auth-work/server/internal/daemon/health.go | Codex1 | |
| multica-auth-work/server/internal/daemon/execenv/execenv.go | Codex1 | Codex3 submits contract-compatible input; no direct edit |
| multica-auth-work/server/internal/daemon/execenv/codex_home.go | Codex1 | Codex3 submits contract-compatible input; no direct edit |
| multica-auth-work/server/pkg/agent/models.go | Codex1 | Codex3 does not edit this hotspot |
| multica-auth-work/server/cmd/multica/cmd_daemon.go | Codex1 | |
| multica-auth-work/server/go.mod | Codex1 (mudanças de dep) | |
| multica-auth-work/server/internal/daemon/brain/** | Codex1 | frozen contract `agent-brain.v1` |
| multica-auth-work/server/internal/daemon/prodex.go | Codex1 | preserved PD-01 baseline |
| multica-auth-work/server/internal/daemon/prodex_fs_linux.go | Codex1 | preserved PD-01 baseline |
| multica-auth-work/server/internal/daemon/prodex_fs_other.go | Codex1 | preserved PD-01 baseline |
| multica-auth-work/server/internal/daemon/prodex_profiles.go | Codex1 | preserved PD-01 baseline |
| multica-auth-work/server/internal/daemon/l2_runtime.go | Codex1 | preserved PD-01 baseline |

Regra: agentes 2,3,4 criam modules novos (`gateway/`, `runtimeenv/`, `cli/`, ops) e NÃO fiam no daemon. Somente Codex1 conecta (G3).

## Ownership por stream (após autorização)

| Stream | Diretório/files | Agente |
|---|---|---|
| Brain core/facade/neutral contracts | internal/daemon/brain/** (new), types/contracts, compatibility facade | Codex1 |
| OmniRoute gateway | internal/daemon/gateway/** (new), protocol fixtures, telemetry parsing types | Codex2 |
| Runtime/CLI security | internal/daemon/runtimeenv/** (new), pkg/agent/{claude,codex,kimi,nim,antigravity}.go (com coordenação), execenv sanitizer | Codex3 |
| Ops/parity/evidence | deploy/**, observability, evidence harness, runbooks, parity matrices Estados, secret ref | Codex4 |

## Código Prodex (baseline preservado — PD-01 resolvida)

`internal/daemon/{prodex.go,l2_runtime.go,prodex_fs_linux.go,prodex_fs_other.go,prodex_profiles.go}`,
`multica-auth-work/prodex-sidecar/` — atualmente modificado/não-commitado e preservado por
decisão do dono. Owner exclusivo: Codex1 (integrator). Até o check-in/lock de G1: READ-ONLY.
Depois do lock, Codex1 audita, testa e reconcilia as 16 tasks de
`persist-prodex-runtime-integration`; nenhum outro agente edita esses hotspots.

## Registro de locks ativos

G1 freeze publicado em `evidence/g1-codex1-contract-freeze.md`; G2 isolated packages are
complete. Shared entrypoints and all hotspots above remain Codex1-only for G3. See
AGENT_LEDGER.md for check-in/out.

## Frontend ownership — active correction round only (vendor/model visibility)

Bounded grant for the vendor/model visibility UI correction; NOT a general frontend grant.

| Path | Owner | Scope note |
|---|---|---|
| multica-auth-work/packages/core/runtimes/models.ts | Codex3 | provider grouping, fallback-only-for-missing, provider search, cache-only provider resolution |
| multica-auth-work/packages/views/agents/components/inspector/model-picker.tsx | Codex3 | provider-visible picker |
| multica-auth-work/packages/views/agents/components/model-dropdown.tsx | Codex3 | provider-visible dropdown |
| (matching `*.test.tsx` for the three files above) | Codex3 | tests |

Boundary: Codex4 remains owner of server-side observability attribution (`deploy/**`, `observability/**`).
Codex3's grant is limited to the bounded files above for this round only; no other frontend files,
no server/daemon/credential/active-path changes. Formal Packet-B acceptance trails EV-G4-03 acceptance.
