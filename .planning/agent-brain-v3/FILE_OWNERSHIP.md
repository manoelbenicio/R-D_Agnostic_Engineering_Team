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

## Expansão 8 lanes (Wave A, D-V3-18) — zero-overlap

> Substitui a topologia de 4 streams para o restante do programa (G4 acceptance, G4-OBS,
> capacidade, disposição recovery-mode, sibling closure). Kiro/Opus-4.8 = planning/adjudication;
> Codex#56#A = transport/verification (não edita produto). W1 permanece o único editor dos
> hotspots centrais.

| Lane | Papel | Ownership exclusivo (globs) | Não-tocar |
|---|---|---|---|
| W1 | Lead Integrator (wiring central, máquina de estados recovery, OBS-4) | `internal/daemon/{daemon,config,health,cmd_daemon}.go`, `go.mod`, `execenv/**`, `pkg/agent/models.go`, `prodex*.go`, `l2_runtime.go`, `brain/**` | pacotes novos de outras lanes |
| W2 | OmniRoute Gateway (8.1/8.4/8.5/8.6/8.7 gateway, OBS-6) | `internal/daemon/gateway/**` | hotspots centrais, outros pacotes |
| W3 | Runtime/CLI Security (8.2/8.3, isolamento env, OBS-5; 5.6–5.8 fail-closed) | `internal/daemon/runtimeenv/**`, `pkg/agent/{claude,codex,kimi,nim,antigravity}.go` (coordenado) | hotspots centrais, gateway |
| W4 | Ops/Capacity/Evidence (8.8, 9.x harness, OBS-11) | `internal/daemon/deploy/**`, `internal/daemon/observability/dashboards/**`, harness specs, runbooks, `EVIDENCE_INDEX.md` | daemon/gateway/runtime impl |
| W5 | Lib de correlação E2E + leak-scan (OBS-1/OBS-9/OBS-10) | `internal/daemon/observability/e2e/**` (nova lib) | arquivos dos chamadores |
| W6 | Instrumentação ingress + WS/UI delivery (OBS-2, OBS-8) | arquivo(s) de middleware HTTP de ingress + transporte WS (nomeados no freeze) | `squad_briefing*.go`, hotspots daemon |
| W7 | Instrumentação queue + terminal persistence (OBS-3, OBS-7) | arquivo(s) de repo da task-queue + store de resultado terminal (nomeados no freeze) | hotspots daemon, handler |
| W8 | Governança + disposição Prodex cold-recovery + sibling closure (drafts) | docs de change OpenSpec, drafts parity/removal, evidence de sibling reopened | hotspots de produto; GSD autorado por Kiro |

Regra de arbitragem de arquivo compartilhado: qualquer arquivo que duas lanes precisariam editar
é escalado a W1 e serializado entre waves — nunca concorrente. Spans cross-cutting são adicionados
pelo dono do arquivo **chamando** a lib W5 `observability/e2e`, nunca co-editando-a.

Prova de zero-overlap (registrar como `EV-ZERO-OVERLAP` antes do dispatch da Wave B): (1) W1–W5
possuem globs de pacote disjuntos par-a-par (∩ = ∅ por construção); (2) spans cross-cutting via
chamada à lib W5; (3) W6/W7 possuem arquivos específicos congelados, removidos de todo outro glob;
(4) arquivo disputado → escala a W1 + serializa; (5) Codex#56#A roda a checagem de interseção de
globs (cada path casa exatamente uma lane) e registra a prova. Locks em `AGENT_LEDGER.md`.

## Wave B.0 FROZEN — exact instrumentation paths (Council+Owner approved 2026-07-19; D-V3-18)

> Frozen by Kiro-TL; EV-ZERO-OVERLAP recorded (`evidence/ev-zero-overlap-wave-b0.md`).
> Lanes must NOT be dispatched until Codex56-Principal-TL ACCEPTS EV-ZERO-OVERLAP.
> Paths are relative to `multica-auth-work/server/`.

| Lane | Exclusive paths (FROZEN) | New? |
|---|---|---|
| W1 | `internal/daemon/{daemon,config,health}.go`, `cmd/multica/cmd_daemon.go`, `go.mod`, `internal/daemon/execenv/**`, `pkg/agent/models.go`, `internal/daemon/{prodex.go,prodex_fs_linux.go,prodex_fs_other.go,prodex_profiles.go,l2_runtime.go}`, `internal/daemon/brain/**` | existing |
| W2 | `internal/daemon/gateway/**` | existing |
| W3 | `internal/daemon/runtimeenv/**`, `pkg/agent/{claude,codex,kimi,nim,antigravity}.go` | existing |
| W4 | `internal/daemon/deploy/**`, `internal/daemon/observability/**` **EXCEPT `internal/daemon/observability/e2e/**`**, **PLUS the real stack `multica-auth-work/deploy/observability/**` (repo-root-relative — Grafana/Prometheus/Alertmanager: docker-compose.yml, prometheus.yml, alertmanager.yml, alerts.yml, grafana/**)** | existing (carve-out; amended 2026-07-19) | 
| W5 | `internal/daemon/observability/e2e/**` | **NEW** |
| W6 | `internal/middleware/obs_ingress.go` (OBS-2 ingress span), `internal/daemonws/obs_delivery.go` (OBS-8 WS/UI delivery span) | **NEW** |
| W7 | `internal/service/obs_queue.go` (OBS-3 DB-queue span), `internal/service/obs_persist.go` (OBS-7 terminal-persistence span) | **NEW** |
| W8 | `openspec/changes/**` docs + `.planning/agent-brain-v3/evidence/**` sibling evidence (no product code) | docs only |

> **Evidence-path clarification (2026-07-19, from W3↔W8 overlap adjudication):** each lane writes ONLY
> its own EV-id-namespaced artifact under `.planning/agent-brain-v3/evidence/` (e.g., W3 → `EV-OBS-05`
> file, W4 → `EV-OBS-11` file). **W8's evidence scope is limited to sibling-change reopened-task
> evidence** (chat-orchestration / agent-credential-isolation / native-runtimes-onboarding) — it does
> NOT own other lanes' OBS evidence. No lane writes into another lane's EV artifact.

**Shared anchor files — Wave C, W1-serial only (NOT in W6/W7 static ownership):** the call-site
insertions that invoke the W6/W7 span helpers live in shared files and are inserted exclusively by
W1 during Wave C serial integration: `internal/metrics/http.go` (or router chain) → registers
`obs_ingress`; `internal/daemonws/hub.go` broadcast → calls `obs_delivery`; `internal/service/task.go`
enqueue/dequeue + terminal-result sites → call `obs_queue`/`obs_persist`. W6/W7 own only their new
span-emitter files and call the W5 `observability/e2e` library; they do NOT edit the shared anchors.

**Zero-overlap result (see EV-ZERO-OVERLAP + 2026-07-19 amendment):** existing lanes W1–W4 =
**159 tracked files after amendment** (W1 69, W2 25, W3 19, W4 46 = 26 server + 20 real stack),
all pairwise intersections = 0; W5/W6/W7 target paths absent; shared anchors reserved to W1-serial.
No file is claimed by two lanes. **OBS-11 (dashboards/alerts) acceptance requires the real stack
`multica-auth-work/deploy/observability/**`, now exclusively W4.** Note: all lane globs are
server-relative (`multica-auth-work/server/…`) EXCEPT W4's stack path, which is repo-root-relative.
