# TRACEABILITY — Agent Brain v3 (cadeia sem órfãos & Multica Kanban Wave 3)

> Cadeia obrigatória por item:
> componente/interface → AB-REQ → spec requirement/scenario → OpenSpec task →
> fase/tarefa GSD → owner + files_locked → evidence ID → status/decisão release|removal → Kanban Card (ORQ-11 a ORQ-62).
> OpenSpec Topic SHA: `7618599f29d43e964a485ab12a9932a9fd037e1f` (accepted-in-review topic content, ORQ-62).
> Ponteiro de Integração Base: `b657129` (origin/main) | Overlay de Produção: `8227241`.
> Contagem do Quadro DB (2026-07-29T14:55:30Z): 52 cards no total (24 done, 12 in_review, 4 in_progress, 7 blocked, 1 todo, 2 backlog, 2 cancelled).

## A. Mapeamento de Tarefas OpenSpec Não-Verificadas para Kanban Vivo

| Spec / Domínio | Tarefa OpenSpec | Card Kanban Multica | Status no DB | Descrição / Escopo |
|---|---|---|---|---|
| `native-runtimes-onboarding` | 1.6 | **ORQ-52** | `done` | Native Runtimes Onboarding: Design System Parity & Web QA |
| `native-runtimes-onboarding` | 2.4 – 3.4 | **ORQ-53** | `backlog` | Native Runtimes Integration: NIM/Cline Backend Deploy & End-to-End Smoke |
| `credential-account-home-restoration` | 3.2, 3.4 | **ORQ-13** + **ORQ-14** | `in_review` / `blocked` | Canonical Usage Integration (ORQ-13) + Real Token Telemetry (ORQ-14) |
| `credential-account-home-restoration` | 3.3 | **ORQ-14** | `blocked` | Real Token Telemetry — AGY/Kiro real counters or explicit unavailability |
| `credential-account-home-restoration` | 4.5 | **ORQ-23** | `in_progress` | Independent safety review before live gate 4.5 |
| `chat-orchestration-standard` | 2.2 | **ORQ-54** | `in_review` | Chat Escape Hatch — Direct @agent routing and focused acceptance |
| `chat-orchestration-standard` | 2.3 | **ORQ-59** | `in_progress` | P0 GSD Wave 3 — Full live rebaseline and OpenSpec traceability |
| `build-omniroute-agent-brain` | 6.3, 6.4 | **ORQ-50** | `in_review` | Capacity — Prepare 20/50/100 harness and zero-queue load window |

---

## B. Componente → AB-REQ → Spec → Task → Fase → Owner → Evidence → Kanban Card

| Componente/Interface | AB-REQ | Spec req (scenario) | OpenSpec task | GSD fase | Owner | Evidence | Card Kanban | Status no DB |
|---|---|---|---|---|---|---|---|---|
| brain coordinator/task-executor/registry | 01,02,03,31 | ABR(Start/Execute) | 3.1,3.5 | G2A | Codex1 | EV-G2A-01/05 | ORQ-11, ORQ-22 | DONE package |
| CLIKind + RouteModel + RouterOwner types | 03,06 | ABR(Claude uses agy) | 1.1,3.2 | G1/G2A | Codex1 | EV-G1-02/EV-G2A-02 | ORQ-20 | DONE package |
| gateway-required admission/readiness | 04,22 | ABR(OmniRoute unavailable); CLE | 3.4,7.5 | G2A/G3 | Codex1 | EV-G2A-04/EV-G3-04 | ORQ-57 | IN_PROGRESS |
| compatibility facade | 05 | ABR(Legacy assigns task) | 2.3,3.3 | G1/G2A | Codex1 | EV-G1-04/EV-G2A-03 | ORQ-22 | DONE package |
| OmniRoute client | 07,34 | ORR(tools streamed) | 4.1 | G2B | Codex2 | EV-G2B-01 | ORQ-44 | BLOCKED |
| liveness/readiness + /v1/models authenticated | 08 | ORR(capability contract) | 4.2 | G2B | Codex2 | EV-G2B-02 | ORQ-44 | BLOCKED |
| model/capability registry | 08,33 | ORR(unsupported capability) | 4.3 | G2B | Codex2 | EV-G2B-03 | ORQ-20 | DONE package |
| env builder (remove provider keys) | 16,17 | CLE(prepare env) | 5.1,5.2 | G2C | Codex3 | EV-G2C-01/02 | ORQ-36 | DONE package |
| Linux restricted secret (no copy/log) | 20 | CLE(secret source) | 6.1 | G2D | Codex4 | EV-G2D-01 | ORQ-36 | DONE package |
| capacity/failure harness spec (20/50/100) | 23-29 | PAC; checklist §9 | 6.5 | G2D | Codex4 | EV-G2D-05 | ORQ-50 | IN_REVIEW |
| Chat Lifecycle default squad materialization | 05,31 | Chat Orchestration | 1.1,2.1 | G4 | Codex2 | EV-CHAT-26 | ORQ-26 | IN_REVIEW |
| JWT Secret Durability & Rotation Audit | 20,21 | Security Wave B | 8.1 | G4 | agy-p0-a7 | EV-JWT-33 | ORQ-33 | IN_REVIEW |
| PostgreSQL SCRAM Hardening | 20 | Security Wave B | 8.2 | G4 | Codex4 | EV-PG-35 | ORQ-35 | IN_REVIEW |
| MULTICA_TOKEN Secret Governance | 20 | Security Wave B | 8.3 | G4 | agy-p0-a7 | EV-SEC-36 | ORQ-36 | DONE |
| OpenSpec Link Integrity & Rotation Router | 30 | BCO | 0.4 | G4 | agy-p0-a7 | EV-ORQ-61 | ORQ-61 | IN_REVIEW |
| OpenSpec Full Lineage Reconciliation | 30 | BCO | 0.6 | G4 | agy-p0-a7 | EV-ORQ-62 | ORQ-62 | IN_REVIEW |
| GSD Wave 3 Full Live Rebaseline | 30,41 | BCO | 2.3 | G4 | agy-p0-a7 | EV-GSD-WAVE3 | ORQ-59 | IN_PROGRESS |

---

## C. Matriz de Paridade P01–P34 / SC01–SC10 → AB-REQ → Owner → Phase

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
- SC01–SC10 (Smart Context) → AB-REQ-14 · Codex2/4 · G5
- B01–B08 (cold-plane) → AB-REQ-02/03/31 · Codex1 · G2A
- R01–R05 (retire-by-decision) → REMOVAL_REGISTER · gate G6

---

## D. Auditoria de Órfãos (Reconciliado Wave 3 — UTC 2026-07-29T14:55:30Z)

- [x] Requisitos specs ↔ AB-REQs: 41 AB-REQs cobrem os 6 specs (incl. `end-to-end-observability`) + paridade.
- [x] Tarefas OpenSpec ↔ Kanban Vivo: Cada resíduo executável de OpenSpec possui exatamente um card no Multica Kanban (ORQ-11 a ORQ-62 no DB).
- [x] linhagem OpenSpec: SHA accepted-in-review `7618599f29d43e964a485ab12a9932a9fd037e1f` (ORQ-62) reconcilia todos os 5 changes ativos sem órfãos ou links quebrados.
