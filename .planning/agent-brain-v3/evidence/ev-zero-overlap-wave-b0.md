# EV-ZERO-OVERLAP — Wave B.0 8-lane ownership disjointness proof

- author: Kiro-TL-Orchestrator (planning/adjudication)
- date: 2026-07-19
- authority: Council + Owner approved Wave B.0 (D-V3-18). Lanes MUST NOT be dispatched until
  Codex56-Principal-TL ACCEPTS this evidence.
- scope: READ-ONLY proof over `multica-auth-work/server/` tracked files. No product code changed.
- decision rule: Prodex activation, 9.1, PD-08 credential work, key handling/rotation, cutover and
  production remain HELD.

## Lane ownership globs (paths relative to `multica-auth-work/server/`)

| Lane | Globs |
|---|---|
| W1 | `internal/daemon/{daemon,config,health}.go`, `cmd/multica/cmd_daemon.go`, `go.mod`, `internal/daemon/execenv/**`, `pkg/agent/models.go`, `internal/daemon/{prodex.go,prodex_fs_linux.go,prodex_fs_other.go,prodex_profiles.go,l2_runtime.go}`, `internal/daemon/brain/**` |
| W2 | `internal/daemon/gateway/**` |
| W3 | `internal/daemon/runtimeenv/**`, `pkg/agent/{claude,codex,kimi,nim,antigravity}.go` |
| W4 | `internal/daemon/deploy/**`, `internal/daemon/observability/**` EXCEPT `internal/daemon/observability/e2e/**` |
| W5 | `internal/daemon/observability/e2e/**` (NEW) |
| W6 | `internal/middleware/obs_ingress.go` (NEW), `internal/daemonws/obs_delivery.go` (NEW) |
| W7 | `internal/service/obs_queue.go` (NEW), `internal/service/obs_persist.go` (NEW) |
| W8 | `openspec/changes/**`, `.planning/agent-brain-v3/evidence/**` (docs/evidence only, no product code) |

Shared anchor files (Wave C, W1-serial only; not in W6/W7 static ownership): `internal/metrics/http.go`,
`internal/daemonws/hub.go`, `internal/service/task.go`.

## Reproducible check (run in `multica-auth-work/server/`)

```bash
tmp=$(mktemp -d)
git ls-files 'internal/daemon/daemon.go' 'internal/daemon/config.go' 'internal/daemon/health.go' \
  'cmd/multica/cmd_daemon.go' 'go.mod' 'internal/daemon/execenv/*' 'pkg/agent/models.go' \
  'internal/daemon/prodex.go' 'internal/daemon/prodex_fs_linux.go' 'internal/daemon/prodex_fs_other.go' \
  'internal/daemon/prodex_profiles.go' 'internal/daemon/l2_runtime.go' 'internal/daemon/brain/*' | sort -u > $tmp/W1
git ls-files 'internal/daemon/gateway/*' | sort -u > $tmp/W2
git ls-files 'internal/daemon/runtimeenv/*' 'pkg/agent/claude.go' 'pkg/agent/codex.go' 'pkg/agent/kimi.go' \
  'pkg/agent/nim.go' 'pkg/agent/antigravity.go' | sort -u > $tmp/W3
git ls-files 'internal/daemon/deploy/*' 'internal/daemon/observability/*' \
  | grep -v 'internal/daemon/observability/e2e/' | sort -u > $tmp/W4
for a in W1 W2 W3 W4; do for b in W1 W2 W3 W4; do [ "$a" \< "$b" ] && \
  echo "$a ∩ $b = $(comm -12 $tmp/$a $tmp/$b | wc -l)"; done; done
```

## Result (2026-07-19, recovery SHA da42282 tree)

```
W1: 69 files   W2: 25 files   W3: 19 files   W4: 26 files   (139 tracked, existing lanes)
W1 ∩ W2 = 0
W1 ∩ W3 = 0
W1 ∩ W4 = 0
W2 ∩ W3 = 0
W2 ∩ W4 = 0
W3 ∩ W4 = 0
TOTAL overlaps (W1-W4) = 0

New target paths (must be absent → no collision):
internal/daemon/observability/e2e     absent-OK
internal/middleware/obs_ingress.go    absent-OK
internal/daemonws/obs_delivery.go     absent-OK
internal/service/obs_queue.go         absent-OK
internal/service/obs_persist.go       absent-OK

Shared anchors present (reserved W1-serial, Wave C):
internal/metrics/http.go   present
internal/daemonws/hub.go   present
internal/service/task.go   present
```

**Verdict: PASS — pairwise-disjoint ownership; no file claimed by two lanes; new W5/W6/W7 files
do not collide with any existing lane; W4↔W5 separated by explicit `e2e/**` carve-out; shared
anchors reserved to W1 serial integration (Wave C).**

## Post-gate 8-lane → 8-pane mapping (PREPARED; dispatch HELD until Codex56-Principal-TL ACCEPTS)

Coordinators (excluded from the 8 workers): `w3:pW` = Kiro-TL-Orchestrator (planning/adjudication);
`w3:p1B` = Codex56-Principal-TL (transport/verification/approval). Eight available worker panes:

| Lane | Worker pane | Agent | Role |
|---|---|---|---|
| W1 Lead Integrator | `w7:p1` | codex | central wiring, recovery-mode scaffold, OBS-4 |
| W2 OmniRoute Gateway | `w7:p2` | codex | 8.1/8.4/8.5/8.6/8.7 gateway, OBS-6 |
| W3 Runtime/CLI Security | `w4:p1` | agy | 8.2/8.3 + child-env isolation, OBS-5 |
| W4 Ops/Capacity/Evidence | `w4:p2` | agy | 8.8, 9.x harness (held), OBS-11 dashboards/bundle |
| W5 E2E Correlation lib | `w6:p1` | kiro | OBS-1/OBS-9/OBS-10 |
| W6 Ingress+WS spans | `w6:p2` | kiro | OBS-2, OBS-8 |
| W7 Queue+persist spans | `w5:p1` | opencode | OBS-3, OBS-7 |
| W8 Governance+sibling | `w5:p2` | opencode | sibling closure drafts, disposition support |

> Pane ids re-read live from `herdr pane list` at dispatch time (ids compact on close). This mapping
> is a PROPOSAL for Codex56-Principal-TL; verbatim lane prompts are in DISPATCH_QUEUE.md (Wave B),
> and remain NOT DISPATCHED until EV-ZERO-OVERLAP is ACCEPTED. Prodex/9.1/PD-08/keys/cutover HELD.

---

## AMENDMENT 2026-07-19 — W4 real observability stack included (re-acceptance pending)

**Trigger:** W4 commit `2c5f4d4` rejected as insufficient for OBS-11. Root cause: the original W4
freeze covered `internal/daemon/deploy/**` + `internal/daemon/observability/**` (server-relative)
but OMITTED the actual tracked Grafana/Prometheus/Alertmanager stack at
`multica-auth-work/deploy/observability/**` (repo-root-relative). This amendment assigns that exact
existing path **exclusively to W4**. Planning-only; no product edits.

**Amended W4 ownership adds (20 tracked files):** `multica-auth-work/deploy/observability/` —
`docker-compose.yml`, `prometheus.yml`, `alertmanager.yml`, `alerts.yml`,
`grafana/dashboards/*.json`, `grafana/provisioning/**`, `pg-exporter-entrypoint.sh`,
`README.md`, `.env.example`, `.gitignore`, `secrets/*.example` (example templates only — NO real
secret values may be introduced; NO-SECRET hold applies).

**Re-run proof (repo-root-relative):**
```
counts: W1=69 W2=25 W3=19 W4=46   (W4 = 26 server + 20 real stack); total = 159
W1 ∩ W2 = 0
W1 ∩ W3 = 0
W1 ∩ W4 = 0
W2 ∩ W3 = 0
W2 ∩ W4 = 0
W3 ∩ W4 = 0
TOTAL overlaps = 0
new stack files added to W4 = 20 (disjoint tree: multica-auth-work/deploy/ is a sibling of server/)
```

**Verdict: PASS (amended)** — the real stack is exclusively W4 and disjoint from all lanes.

**Governance:** the base EV-ZERO-OVERLAP was ACCEPTED @ `4c67ae0`; this amendment RE-OPENS it for
**re-acceptance by Codex56-Principal-TL**. Until the amended proof is accepted, **W4 MUST NOT edit
the real stack `multica-auth-work/deploy/observability/**` and MUST NOT claim OBS-11.** Commit
`2c5f4d4` is recorded **PRODUCED-NOT-ACCEPTED** (not done).
