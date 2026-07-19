# TRACEABILITY — Agent Brain v3 (cadeia sem órfãos)

> Cadeia obrigatória por item:
> componente/interface → AB-REQ → spec requirement/scenario → OpenSpec task →
> fase/tarefa GSD → owner + files_locked → evidence ID → status/decisão release|removal.
> Auditoria de órfãos ao fim de cada fase (task 0.6). Nenhum item sem cadeia completa.

## A. Componente → AB-REQ → Spec → Task → Fase → Owner → Evidence

| Componente/Interface | AB-REQ | Spec req (scenario) | OpenSpec task | GSD fase | Owner | Evidence | Status |
|---|---|---|---|---|---|---|---|
| brain coordinator/task-executor/registry | 01,02,03,31 | ABR(Start/Execute) | 3.1,3.5 | G2A | Codex1 | EV-G2A-01/05 | DONE package; G3 wiring pending |
| CLIKind + RouteModel + RouterOwner types | 03,06 | ABR(Claude uses agy) | 1.1,3.2 | G1/G2A | Codex1 | EV-G1-02/EV-G2A-02 | DONE package |
| gateway-required admission/readiness | 04,22 | ABR(OmniRoute unavailable); CLE | 3.4,7.5 | G2A/G3 | Codex1 | EV-G2A-04/EV-G3-04 | DONE contract; G3 wiring pending |
| compatibility facade | 05 | ABR(Legacy assigns task) | 2.3,3.3 | G1/G2A | Codex1 | EV-G1-04/EV-G2A-03 | DONE package; G3 wiring pending |
| OmniRoute client (auth redacted, base URL, timeouts, cancel) | 07,34 | ORR(tools streamed) | 4.1 | G2B | Codex2 | EV-G2B-01 | DONE package |
| liveness/readiness + /v1/models authenticated | 08 | ORR(capability contract); checklist §1,2 | 4.2 | G2B | Codex2 | EV-G2B-02 | DONE package; live acceptance pending |
| model/capability registry (versioned) | 08,33 | ORR(unsupported capability) | 4.3 | G2B | Codex2 | EV-G2B-03 | DONE package |
| trusted runtime profiles (Messages/Responses/Chat/AG) | 07 | ORR(tools streamed) | 4.4 | G2B | Codex2 | EV-G2B-04 | DONE package |
| route-policy types (RR, affinity, retry, fallback, circuit, SC) | 09-15 | ORR(strict RR, affinity, pre-commit, 429) | 4.5 | G2B | Codex2 | EV-G2B-05 | DONE package; live acceptance pending |
| telemetry parsing (no content/secrets) | 07,21,38 | ORR; CLE; BCO(handover) | 4.6,6.3 | G2B/G2D | Codex2/4 | EV-G2B-06/EV-G2D-03 | DONE package |
| protocol fixtures (synthetic creds/content) | 33 | checklist §12 | 4.7 | G2B | Codex2 | EV-G2B-07 | DONE synthetic fixtures |
| env builder (remove provider keys) | 16,17 | CLE(prepare env) | 5.1,5.2 | G2C | Codex3 | EV-G2C-01/02 | DONE package; G3 application pending |
| trusted-wins env merge | 18 | CLE(custom direct routing) | 5.2 | G2C | Codex3 | EV-G2C-02 | DONE package; G3 application pending |
| controlled per-CLI config (Codex Responses, no auth.json) | 19 | CLE(prepare Codex) | 5.4,5.5 | G2C | Codex3 | EV-G2C-04/05 | DONE contract; G3 wiring pending |
| Claude Code adapter (root URL/token, no marker leak) | 19 | CLE; design §4 | 5.5 | G2C | Codex3 | EV-G2C-05/EV-G3-06 | DONE contract; G3 wiring pending |
| Kimi/GLM/NVIDIA adapter | 07,14 | ORR(P18 row) | 5.6,5.7 | G2C | Codex3 | EV-G2C-06/07/EV-G4-ADP | FAIL-CLOSED CONTRACT; native acceptance pending |
| NIM → gateway (no direct NVIDIA key overwrite) | 16 | CLE; design §4 | 5.6 | G2C | Codex3 | EV-G2C-06/EV-G4-NIM | FAIL-CLOSED CONTRACT; native acceptance pending |
| Agy native/fallback | 07 | ORR(P17 row) | 5.8 | G2C | Codex3 | EV-G2C-08/EV-G4-AGY | FAIL-CLOSED CONTRACT; acceptance pending |
| pre-launch assertion (only OmniRoute secret) | 16,22 | CLE | 5.10 | G2C | Codex3 | EV-G2C-03/10 | DONE package; G3 application pending |
| Linux restricted secret (no copy/log) | 20 | CLE(secret source) | 6.1 | G2D | Codex4 | EV-G2D-01 | DONE reference-only contract |
| endpoint config (host loopback vs container DNS) | 34 | BCO(env endpoint) | 6.2 | G2D | Codex4/2 | EV-G2D-02 | DONE package |
| structured redacted events/metrics | 21,38 | CLE; BCO(handover) | 6.3 | G2D | Codex4 | EV-G2D-03 | DONE package |
| dashboards/alerts | 38 | BCO; checklist §8 | 6.4 | G2D | Codex4 | EV-G2D-04 | DONE specification |
| capacity/failure harness spec (20/50/100) | 23-29 | PAC; checklist §9 | 6.5 | G2D | Codex4 | EV-G2D-05 | DONE non-runnable specification |
| runbooks (backup/restore, hot-change, rotation, upgrade, rollback) | 36,38 | BCO(handover) | 6.6 | G2D | Codex4 | EV-G2D-06 | DONE specification |
| staged flags/controlled cohort/rollback triggers | 35,36 | BCO(atomic cutover/safe rollback) | 6.7 | G2D | Codex4 | EV-G2D-07 | DONE default-off specification |
| central daemon wiring (config/aliases through entrypoints) | 04,06,22 | ABR; ORR; CLE | 7.1-7.9 | G3 | Codex1 | EV-G3-WIRE | PLANNED |
| first vertical slice (one approved route) | 31,04 | BCO(first slice) | 7.10 | G3 | Codex1 | EV-G3-07 | PLANNED |
| protocol/failure acceptance | 33 | BCO(protocol gate) | 8.1-8.7 | G4 | Codex2/3/4 | EV-G4-08 | PLANNED |
| E2E correlation schema + trace assembly | 39 | EOO(correlation/trace) | OBS-1,OBS-9 | G4-OBS | W5 | EV-OBS-01/09 | PLANNED (gate) |
| per-hop spans (ingress/queue/daemon/CLI/route/persist/WS) | 39,40 | EOO(per-hop spans); ORR(correlation); CLE(metadata-only) | OBS-2..OBS-8 | G4-OBS | W6/W7/W1/W3/W2 | EV-OBS-02..08 | PLANNED (gate) |
| structural leak-scan + dashboards + G4-OBS acceptance | 40 | EOO(leak-clean/dashboards/stop-gate) | OBS-10,OBS-11 | G4-OBS | W5/W4 | EV-OBS-10/11 | PLANNED (blocking gate before §9/§10) |
| platform recovery-mode state machine (Prodex retain, default-OFF) | 41 | BCO(cold recovery mode) | 10.4 (retain-as-recovery),7.8 | G6 | Codex1/W1 | EV-REC-MODE | PLANNED |
| capacity tiers 20/50/100 run | 23,28,29 | PAC; checklist §9 | 9.1-9.6 | G4/G7 | Codex4/1 | EV-G4-CAP | PLANNED (após G4-OBS PASS) |
| ops sign-off | 38 | BCO(handover) | 9.7 | G4/G7 | Codex4 | EV-G4-07 | PLANNED |
| default cutover + drain | 35,37 | BCO(legacy removal) | 10.1-10.3 | G6 | Codex1/4 | EV-G6-01 | PLANNED |
| Prodex→recovery quiesce (retain); delete Go rotation/creds only | 37,41 | BCO(removal gate/cold recovery mode) | 10.4 (retain-as-recovery),10.5,10.6 | G6 | Codex1/3 | EV-REC-MODE/EV-G6-03 | PLANNED |
| reconcile docs/rollback after removal | 36 | BCO(safe rollback) | 10.7 | G6 | Codex4 | EV-G6-02 | PLANNED |
| inventory remaining Multica/Prodex names | — | BCO(debrand) | 11.1 | G8 | Codex1 | EV-G8-01 | PLANNED |
| final names (binary/module/config) | — | BCO; design §1 | 11.2-11.6 | G8 | Codex1/2/3/4 | EV-G8-02 | PLANNED |
| final sign-off | — | design §12 | 11.7 | G8 | Codex1/4 | EV-G8-03 | PLANNED |

## B. Matriz de paridade P01–P34 / SC01–SC10 → AB-REQ → Owner → Phase

- P01–P03 (identity/secret/vkey) → AB-REQ-16,12 · Owner Codex2/4 · Phase G2B/G2D
- P04–P08 (selection, strict RR, affinity, rotate-before-commit, bounded fallback) → AB-REQ-09/10/11 · Codex2 · G2B/G4
- P09–P13 (quota, reset/redeem, OAuth refresh, 401/403, 429/circuit) → AB-REQ-12/13/15 · Codex2 · G4
- P14 (provider fallback/adaptive) → AB-REQ-11 · Codex2 · G4
- P15–P18 (Anthropic/OpenAI/Gemini-Agy/Kimi-GLM-NVIDIA adapters) → AB-REQ-07 · Codex2/3 · G2C/G4
- P19 (capability discovery) → AB-REQ-08 · Codex2 · G2B
- P20–P22 (MCP continuation, streaming commit, nonblocking I/O) → AB-REQ-07/ir · Codex2/3 · G4
- P23–P26 (state/store, broker, policy, kill switches) → AB-REQ-12/.. · Codex2/4 · G2B/G2D
- P27–P31 (health/readiness, events, metrics/audit, redaction/PII, cookies) → AB-REQ-21/38 · Codex4 · G2D/G4
- P32 (idempotency) → AB-REQ-11 (request IDs Brain+OmniRoute) · Codex1/2 · G3/G4
- P33 (capacity/overload) → AB-REQ-25/28 · Codex4/1 · G4/G7
- P34 (catalog/cost/usage) → AB-REQ-21(aggregate) · Codex2/4 · G4
- SC01–SC10 (Smart Context) → AB-REQ-14 · Codex2/4 · G5 (evidence ou waiver)
- B01–B08 (cold-plane) → AB-REQ-02/03/31 · Codex1 · G2A
- R01–R05 (retire-by-decision) → REMOVAL_REGISTER · gate G6

## C. Auditoria de órfãos (G0)

- [x] Requisitos specs ↔ AB-REQs: 41 AB-REQs cobrem os 6 specs (incl. `end-to-end-observability`) + paridade.
- [x] Tasks OpenSpec (85) ↔ fases GSD: mapa criado em phases/G0../PLAN.md.*;
- [x] Worktree: PD-01 resolved by preservation and exclusive Codex1 ownership.
- [x] G2 phase-end orphan reconciliation completed during TL handover; G3 links remain planned.
