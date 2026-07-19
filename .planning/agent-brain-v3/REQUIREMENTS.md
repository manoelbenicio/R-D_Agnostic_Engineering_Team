# REQUIREMENTS — Agent Brain v3 (IDs AB-REQ, derivados dos 5 specs OpenSpec + matrizes)

> Cada AB-REQ rastreia para: spec + scenario OpenSpec, task(s) OpenSpec, fase GSD, owner,
> evidence ID, decisão de release/removal. Nenhum requisito existe sem essa cadeia.
> Status padrão: **PLANNED** (implementação não autorizada). Evidência obriga Gate G3+.

## Códigos de origem
- ABR = spec `agent-brain-runtime/spec.md`
- ORR = spec `omniroute-agent-routing/spec.md`
- CLE = spec `credentialless-agent-execution/spec.md`
- PAC = spec `parallel-agent-capacity/spec.md`
- BCO = spec `brain-cutover-operations/spec.md`
- PAR = `prodex-omniroute-feature-parity.md` (P01–P34, SC01–SC10, B01–B08, R01–R05)

## AB-REQs — Cold plane (Agent Brain / ABR)

| ID | Requisito (resumido) | Origem | Spec scenario | OpenSpec tasks | Fase | Owner | Evidence |
|---|---|---|---|---|---|---|---|
| AB-REQ-01 | Daemon brand-neutral (sem dependência de nomes Multica/Prodex); cold-plane only | ABR | Start the neutral daemon | 1.1, 3.1 | G1/G2A | Codex1 | EV-G1-01 |
| AB-REQ-02 | Preservar lifecycle/workspace/repo/cancel/watchdog/stream/recovery/context/skills neutros | ABR | Execute a normal task | 3.5 | G2A | Codex1 | EV-G2A-05 |
| AB-REQ-03 | Separar `CLIKind` de `RouteModel`; não inferir credential vendor do nome do CLI | ABR | Claude Code uses an Antigravity model route | 1.1, 3.2 | G1/G2A | Codex1 | EV-G1-02 |
| AB-REQ-04 | OmniRoute readiness gate + fail-closed (sem fallback para provider direto) | ABR | OmniRoute becomes unavailable | 3.4, 7.5 | G2A/G3 | Codex1 | EV-G3-04 |
| AB-REQ-05 | Compatibility facade time-bounded, observável, traduz p/ contrato neutro | ABR | Legacy control plane assigns a task | 2.3, 3.3 | G1/G2A | Codex1 | EV-G2A-03 |
| AB-REQ-06 | `RouterOwner=omniroute` único; sem Prodex/RustL2/Go rotation p/ request OmniRoute | ABR | Task begins after cutover | 7.6, 7.7 | G3 | Codex1 | EV-G3-06 |

## AB-REQs — Hot plane routing (OmniRoute / ORR + PAR)

| ID | Requisito | Origem | Spec scenario | OpenSpec tasks | Fase | Owner | Evidence |
|---|---|---|---|---|---|---|---|
| AB-REQ-07 | Adapters protocol-complete (Messages/Responses/Chat/Antigravity) preservam stream/tools/reasoning/usage/cancel/continuation | ORR | Agent uses tools during a streamed response | 4.4, 5.x, 8.1, 8.2 | G2B/G2C | Codex2/3 | EV-G4-01 |
| AB-REQ-08 | Model capability registry versionada; rejeita capability não declarada (não drop silencioso) | ORR | Task requests an unsupported capability | 4.3, 8.1 | G2B | Codex2 | EV-G2B-03 |
| AB-REQ-09 | Strict round-robin concurrency-safe por request lógico (não SSE/retry/tool/limite) | ORR | Simultaneous independent requests arrive | 4.5, 8.4 | G2B/G4 | Codex2 | EV-G4-04 |
| AB-REQ-10 | Continuation affinity preserva previous_response_id/turn/cache/tool; sobrepõe só p/ continuação | ORR | Stateful continuation follows a rotated first request | 4.5, 8.4 | G2B/G4 | Codex2 | EV-G4-04 |
| AB-REQ-11 | Pre-commit recovery: retry/fallback bounded; só antes de output/tool não-idempotente; same-model primeiro | ORR | Stream fails after partial output | 4.5, 8.5, 8.6 | G2B/G4 | Codex2 | EV-G4-05 |
| AB-REQ-12 | Credential+quota lifecycle: refresh proativo, single-flight, classify 401/403, quarantine, quota/reset | ORR | Selected account token has expired | 4.x, 8.5 | G2B/G4 | Codex2 | EV-G4-05 |
| AB-REQ-13 | 429 + circuit breaker: classify account/model/global/overload, backoff jittered, half-open | ORR | One account repeatedly returns 429 | 4.5, 8.5 | G2B/G4 | Codex2 | EV-G4-05 |
| AB-REQ-14 | Smart Context parity SC01–SC10 (segmentação, validação, shadow, controlled dev cohort, fallback exato, kill switch) | ORR + PAR | Optimized payload fails structural validation | 4.5, 8.8 | G5 | Codex2/4 | EV-G5-SC |
| AB-REQ-15 | Reset/redeem parity (explicit policy, pre-commit, idempotência, grace, audit, recheck) | ORR + PAR | Another account still has quota | 8.8 | G5 | Codex2/4 | EV-G5-RR |

## AB-REQs — Credentialless (CLE)

| ID | Requisito | Origem | Spec scenario | OpenSpec tasks | Fase | Owner | Evidence |
|---|---|---|---|---|---|---|---|
| AB-REQ-16 | Uma única secret OmniRoute; sem provider keys/OAuth/cookies/copied homes | CLE | Prepare an agent task environment | 5.1, 5.3, 5.10 | G2C | Codex3 | EV-G3-03 |
| AB-REQ-17 | Negar credenciais provider herdadas no env child (gateway-required) | CLE | Daemon inherited a provider key | 5.1, 5.2 | G2C | Codex3 | EV-G3-01 |
| AB-REQ-18 | Trusted routing config vence (aplicada por último); custom não sobrescreve | CLE | Custom environment attempts direct routing | 5.2 | G2C | Codex3 | EV-G3-02 |
| AB-REQ-19 | Config por-CLI controlada (Codex Responses custom provider) sem copiar auth files | CLE | Prepare a Codex task | 5.4, 5.5 | G2C | Codex3 | EV-G4-COD |
| AB-REQ-20 | Secret source/permissions: Linux restricted secret, nunca committed/image/log/screenshot | CLE | Daemon reads the host secret | 6.1 | G2D | Codex4 | EV-G2D-01 |
| AB-REQ-21 | Secret-safe evidence: redige secrets/cookies/prompts/tool payloads/repo content/reasoning | CLE | Upstream error includes an authorization value | 6.3, 8.3 | G2D/G4 | Codex4/3 | EV-G4-03 |
| AB-REQ-22 | Fail-closed credential policy: sem fallback provider quando OmniRoute auth falha | CLE | Stable OmniRoute key is invalid | 7.5 | G3 | Codex1 | EV-G3-05 |

## AB-REQs — Capacity (PAC)

| ID | Requisito | Origem | Spec scenario | OpenSpec tasks | Fase | Owner | Evidence |
|---|---|---|---|---|---|---|---|
| AB-REQ-23 | Tiers 20/50/100 configuráveis, sujeitos a limites medidos | PAC | Operator selects the 50-task tier | 2.2, 6.5, 9.x | G2D/G4/G7 | Codex4/1 | EV-G4-CAP |
| AB-REQ-24 | Task admission ≠ inference concurrency; limites independentes; RR ≠ 1-at-time | PAC | Accounts have active requests | 4.5, 9.x | G2B/G4 | Codex2/1 | EV-G4-04 |
| AB-REQ-25 | Bounded admission/overload determinístico; sem crescimento ilimitado de mem/thread/socket/log | PAC | Capacity and queue are full | 6.5, 9.x | G2D/G4 | Codex4 | EV-G4-CAP |
| AB-REQ-26 | Cancel libera capacidade (task/CLI/stream/request) exatamente uma vez | PAC | Operator cancels an active streaming task | 8.6 | G3/G4 | Codex1/2 | EV-G4-06 |
| AB-REQ-27 | Fairness/eligibility evidence: explicar distribuição por eligibility/affinity/quota/circuit | PAC | One account receives fewer requests | 9.1, 8.7 | G4 | Codex4 | EV-G4-04 |
| AB-REQ-28 | Tiered capacity acceptance: evidence reproduzível (mix, sizes, p50/p95/p99, filas, recursos) | PAC | Approve the 100-task tier | 9.1–9.6 | G4/G7 | Codex4 | EV-G4-CAP |
| AB-REQ-29 | Capacity downgrade explícito: enforce tier provado; gap ≠ "limitação RR" | PAC | 100-task tier misses its SLO | 9.4, 9.6 | G4/G7 | Codex1/4 | EV-G4-CAP |

## AB-REQs — Cutover/operations (BCO)

| ID | Requisito | Origem | Spec scenario | OpenSpec tasks | Fase | Owner | Evidence |
|---|---|---|---|---|---|---|---|
| AB-REQ-30 | OpenSpec↔GSD traceability sem órfãos; histórico não opera como plano concorrente | BCO | Orphaned component is discovered | 0.4, 0.6 | G0 | TL | EV-G0-04 |
| AB-REQ-31 | Strangler extraction: neutral interfaces em torno do daemon provado, não rewrite global | BCO | First runnable vertical slice | 3.1, 7.10 | G2A/G3 | Codex1 | EV-G3-07 |
| AB-REQ-32 | Feature parity gate assinado (P01–P34, SC01–SC10) com waiver explícito onde faltar | BCO | Smart Context has no proven replacement | 1.4, 8.8 | G5 | Codex4 | EV-G5-PAR |
| AB-REQ-33 | Protocol+failure acceptance gate: checklist exato implantado (auth, protocolo, streaming, tools, rotation, expiry, quota, 429, fallback, cancel, security, obs, capacity) | BCO | Models endpoint succeeds but tool streaming is unproven | 1.3, 8.x | G4 | Codex4/2 | EV-G4-08 |
| AB-REQ-34 | Environment-specific endpoint: host/WSL usa `127.0.0.1:20128`; containeriza usa DNS/gateway host | BCO | Host daemon launches Codex | 4.1, 6.2 | G2B/G2D | Codex2/4 | EV-G2D-02 |
| AB-REQ-35 | Atomic staged cutover: readiness→protocol→model→capacity→default-on→drain→delete com gates/triggers | BCO | Canary error threshold is exceeded | 10.x, 6.7 | G6 | Codex1/4 | EV-G6-01 |
| AB-REQ-36 | Safe rollback: restaura Agent Brain/OmniRoute aceito, **nunca** provider keys/dual router | BCO | OmniRoute release must be rolled back | 6.6, 10.7 | G6 | Codex4 | EV-G6-02 |
| AB-REQ-37 | Legacy removal gate: Go rotation/credential homes/aliases só após zero-use e rollback independente. **Prodex/L2 EXENTO de deleção — retido como cold recovery mode default-OFF (D-V3-16)** | BCO | Compatibility alias is still in use | 10.4 (retain-as-recovery), 10.5, 10.6, 11.x | G6 | Codex1/3 | EV-G6-03 |
| AB-REQ-38 | Operational handover: owners nomeados, dashboards/alerts, backup/restore, rotation, upgrade/rollback, escalo | BCO | Provider-wide throttling occurs | 6.3, 6.4, 6.6, 9.7 | G2D/G4 | Codex4 | EV-G4-07 |

## AB-REQs — Observabilidade E2E & recovery mode (EOO/BCO)

| ID | Requisito | Origem | Spec scenario | OpenSpec tasks | Fase | Owner | Evidence |
|---|---|---|---|---|---|---|---|
| AB-REQ-39 | Correlação E2E metadata-only nos 8 hops (request_id/queue_msg_id/task_id/session_id/launch_id/proc_id/omni_request_id/result_id/delivery_id); schema versionado; secrets_present=false | EOO + ORR | Eight-hop correlation schema / Request is traced across hops | OBS-1..OBS-9 | G4-OBS | W5/W6/W7/W1/W2/W3 | EV-OBS-01..09 |
| AB-REQ-40 | Spans per-hop metadata-only + redação estrutural de argv; scan estrutural leak-clean; dashboards/alerts; gate G4-OBS bloqueante antes de capacidade/cutover | EOO + CLE | Per-hop metadata-only spans / Structural leakage-clean acceptance / Blocking G4-OBS stop-gate | OBS-2..OBS-11 | G4-OBS | W4/W5 | EV-OBS-02..11 |
| AB-REQ-41 | Máquina de estados de recovery da plataforma: NORMAL/DEGRADED/RECOVERY; Prodex default-OFF, mutuamente exclusivo, operator-gated; um único router owner; transições só em session boundary; DEGRADED fail-closed (nunca auto-promove Prodex) | BCO | Cold platform recovery mode / OmniRoute is unavailable and an operator considers recovery | 10.4 (retain-as-recovery), 7.8 | G6 | Codex1/W1 | EV-REC-MODE |

## Notas de reconciliação

- P01–P34 (34 itens) e SC01–SC10 (10 itens) da matriz de paridade = 44 IDs. Eles são
  absorvidos por AB-REQ-07..AB-REQ-15 (hot) e AB-REQ-32 (gate). Mapeamento detalhado item
  a item fica em TRACEABILITY.md; REMOVAL_REGISTER cobre o status RETIRE BY DECISION (R01–R05).
- B01–B08 (cold-plane) → AB-REQ-02/03/31 (Brain). R01–R05 → REMOVAL_REGISTER com gate.
- Conflito conhecido: o GSD v2.1 (`REQUIREMENTS.md` REQ-09, REQ-10) exigia Smart Context
  via Prodex e proibia SQLite. No target, Smart Context é OmniRoute (AB-REQ-14) e o
  estado single-node SQLite do OmniRoute é aceito para o tier 20 com decisão de estado
  (P23/Q18) — ver DECISIONS D-V3-06 e D-V3-08.
