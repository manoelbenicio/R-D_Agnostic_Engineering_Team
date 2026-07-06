# ROADMAP — Milestone v2.0 (Rotation-Parity Polyglot)

> Fases com dependências explícitas e REQ-IDs. Ordem de execução respeita as dependências.
> Regra: verde-em-container com evidência antes de DONE; QA nunca bypassado.

## Grafo de dependências

```
P0 Fundação ──┬──> P1 Contrato ──> P3 Integração ──┬──> P6 QA exaustivo ──> P7 Deploy PROD
              │                                     │        ▲
              ├──> P2 Fork-map/invariants ──────────┘        │
              ├──> P4 State/security ───────────────────────┘
              ├──> P5 Vendor matrix ─────────────────────────┘
              └──> P8 Ops/observability (contínuo)
P9 Reset-claim (por último; não bloqueia) ── depende de P3 + estado real de conta
```

## Fases

### P0 — Fundação: runtime prodex + ambiente  `[REQ-01, REQ-02, REQ-03]`  **BLOQUEIA TUDO**
- Mover source `/tmp/prodex-audit-7750da9` → local estável; instalar Rust; `cargo build --release`.
- Verificar pin (v0.246.0 / `7750da9b`) + integridade; wire `MULTICA_PRODEX_PATH/VERSION/COMMIT`.
- Confirmar Postgres/Redis (docker) + migrations reversíveis.
- **Gate:** `prodex --version` responde do binário pinado; Multica resolve o executável (`exec.LookPath`).

### P1 — Contrato Go↔L2  `[REQ-04]`  (dep: P0)
- `rpp.l2.v1` (HealthCheck/ApplyPolicy/RegisterAccounts/Start-StopSession/RouteDecisionEvent/RuntimeEventStream/KillSwitch) + schema de eventos + invariante roteador único.

### P2 — prodex fork-map / invariantes  `[REQ-09]`  (dep: P0) — análise (alvo do fork)
- Mapear crates; isolar runtime proxy/gateway/Smart Context/state/redeem; preservar hard affinity + rotate-before-commit.

### P3 — Integração Go (lançar prodex)  `[REQ-05, REQ-06, REQ-40]`  (dep: P1)
- Lifecycle sidecar, healthcheck, policy push, event ingest, kill-switch. Ingest de eventos não dispara rotação no Go.
- **Early Rotation (REQ-40):** warnbanner.go no loop MessageText do daemon; rotação antecipada por banner; retomada transparente. CRITICIDADE MÁXIMA.

### P4 — State/security  `[REQ-10, REQ-11, REQ-12, REQ-41]`  (dep: P0)
- Postgres/Redis backend; redaction policy; audit taxonomy; secrets boundary. Sem SQLite.
- **POSIX FS validation (REQ-41):** credenciais em ext4/xfs, NUNCA drvfs; `stat 600`; abortar se incompatível.

### P5 — Vendor capability matrix  `[REQ-07, REQ-08]`  (dep: P0)
- Fonte primária por vendor; **decidir OpenCode (arquivado)**; classificar verified/inferred/not_validated.

### P6 — QA/conformance EXAUSTIVO  `[REQ-13..REQ-18]`  (dep: P3, P4, P5) — **SEM BYPASS**
- C1–C6 por capability + replay + fail-closed + Smart Context shadow→canary→live + tripla CODEX_HOME×prodex×Herdr + Herdr smoke. **Todos com evidência em container.**
- Nota: valida em container/sidecar controlado — resolve o "nó circular" antes do PROD (não depende do F0 live).

### P7 — DevOps / Deploy PROD  `[REQ-19, REQ-20, REQ-21, REQ-25]`  (dep: P6 verde)
- **Kill-switch testado** + **rollback 1-cmd testado** + logs scrubbed → **deploy direto em PROD** (sem canary). Runbook + observability.

### P8 — Ops / evidence index  (contínuo, desde P0)
- Status board, evidence index, open items por owner.

### P9 — Reset-claim (empírico)  `[REQ-22]`  (dep: P3; **por último**, não bloqueia)
- Matriz + validação empírica com contas reais quando o estado ocorrer.

## Meta (transversal)  `[REQ-23, REQ-24]`
- Tasks rastreáveis por fase; arquivar `rotation-router`; reconciliar docs/board.

## Definition of Done do milestone v2.0
Todos os REQ verdes com evidência; prodex AS-IS rodando em PROD via Multica; kill-switch/rollback provados; QA exaustivo verde; docs/board reconciliados.
**STATUS: COMPLETE** (commit `6ba9a70`, 78/78 tasks).

---

# Milestone v2.1 — Full Vendor Validation + PROD Deploy

> Continua numeração de fases: P11, P12.

### P11 — Vendor Validation (behavioral)  (dep: v2.0 COMPLETE)

**Goal:** Validar COMPORTAMENTALMENTE todas as cells `not_validated` na vendor-capability-matrix via runtime real /v1/runtime/proxy. Capability entregue PELO prodex, não pelo vendor.

**Tasks:**
- [ ] 11.1 Smart Context per-vendor (4 shapes): Codex/OpenAI Responses API, Kiro/Anthropic Messages API, Antigravity/Gemini API, Cline/OpenRouter — tokens_saved>0 cada
- [ ] 11.2 Rotation per-vendor: profile_pool com 2+ profiles, trigger rotation, verify session continuity
- [ ] 11.3 Reset-claim Codex: exercise prodex redeem/--auto-redeem, prove event emitted
- [ ] 11.4 OpenCode disposition: ARCHIVED → Crush (or documented superseded)
- [ ] 11.5 Update vendor-capability-matrix.md: all not_validated → verified with evidence pointer
- [ ] 11.6 GATE P11: 0 cells not_validated, all evidence-referenced. Commit + push.

**Staffing:**
- Codex#A → Codex/OpenAI (smart_context + reset_claim)
- Codex#C → Kiro (smart_context via proxy)
- Codex#D → Antigravity (smart_context + rotation)
- Codex#E → Cline + OpenCode
- Codex#B → STANDBY hotspot Rust (corrige se falhar)

### P12 — PROD Deploy + Live Test  (dep: P11 GATE verde)

**Goal:** Deploy em PROD com sessão real provider-backed. F7 AUTORIZADO pelo dono.

**Tasks:**
- [ ] 12.1 Pre-deploy checklist: all gates P0-P11 green
- [ ] 12.2 Execute prod-rollout-runbook (docs/deploy/prod-rollout-runbook.md)
- [ ] 12.3 PROD session: real provider-backed session, capture evidence
- [ ] 12.4 PROD kill-switch: apply + verify in PROD
- [ ] 12.5 PROD rollback: test in PROD
- [ ] 12.6 PROD logs scrubbed: verify no secrets
- [ ] 12.7 GATE P12: PROD evidence + kill-switch + rollback + scrubbed. Commit + push.

**Staffing:** Codex#D (or first free) executes sequentially.

## Definition of Done do milestone v2.1
0 cells not_validated (all VERIFIED). PROD session real with evidence. Kill-switch + rollback tested LIVE. All commits pushed.
