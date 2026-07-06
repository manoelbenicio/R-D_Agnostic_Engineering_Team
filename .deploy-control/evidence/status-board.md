# Status Board

Status: P6/P7 LOCAL GATES GREEN

## Gate Classifications
- **GREEN (empirical/local evidence):** F1, F2, F3, F5, F6, F7-local, F7-docs, F8, F9-plan, G1, G2, G5, G7, G8, G9, G10-container/local.
- **DEFERRED/GATED:** F9 empirical reset-claim real-account validation.
- **PROD SCOPE NOTE:** provider-backed PROD session evidence remains separate from the local `rpp.l2.v1` sidecar gate.

*NOTE: Codex-D validated P6/P7 against the local `prodex-sidecar`; see `p6-p7-final-gates-20260705T062826Z.md`.*

## Active Check-ins (stream/owner/status/blocker)

| Stream | Owner | Status | Blocker |
|---|---|---|---|
| F1 Go<->L2 contract | Codex#5.5#A | DONE | None |
| F2 prodex fork map | Codex#5.5#B | DONE | None |
| F3 Go integration skeleton | Codex#5.5#C | DONE | StartSession/one-router VALIDATED GREEN (1 env test failure); wiring in progress |
| F4 State/security | GLM#52#B | WORKING | None |
| F5 Vendor capability matrix | Gemini#Pro | DONE | 8 not_validated cells (req. owner accept) |
| F6 QA/conformance | GLM#52#A | STARTING | None |
| F7 DevOps/PROD runbook | Codex#5.5#D | DONE | Owner approval missing |
| F8 Ops triage | Gemini#Flash35 | WORKING | None |
| F9 Reset-claim | Codex#5.5#B/GLM#52#A | DEFERRED/GATED | planning DONE; gated on real account state |

## Standing Rules & Audits

- **STANDING RULE:** Hotspot files `daemon.go`/`config.go`/`execenv.go` require explicit lock lines.
- **AUDIT:** Ownership audit DONE (NO collisions). Codex#5.5#C files_locked STALE (C amending). Minor outliers: prodex-l2-facade.md (Codex#B), owner-acceptance-request.md (Gemini#Pro) created outside locks but self-attributed.
- **ROSTER ACK:** 9/10 signed; GLM#52#A signing now.

## Current Verdict

- GO: dispatch agents.
- NO-GO: real PROD deploy until F7 runbook is approved by owner.

## Required Owner Approval Record

```text
deploy_owner_approved: false
owner:
timestamp:
artifact_hash:
notes:
```

## Open Items

- [DONE] Rust L2 facade doc.
- F4 state/security acceptance.
- F7 deploy runbook acceptance.
- F6 conformance plan acceptance.
- Kill switch smoke.
- [DONE] Redaction audit (PASS).
- Sidecar readiness smoke.
- [O1] (hygiene) .env.production is git-tracked despite .gitignore rule **/.env.production.


---

## Validação de incorporação — Opus 4.8 (2026-07-04)

**Pacote:** `rotation-parity-critical-predeploy-artifacts_2026-07-04.zip`
**SHA256:** `C4FA1B462DDF90AB73F77C2684CFE7C6D4FB110009DB926612C75C94BE2AD25F` → ✅ MATCH
**Incorporação:** 23/23 artefatos copiados ao repo, **zero conflito** (adição limpa, sem sobrescrever).
**Varredura de segredos:** ✅ limpa (nenhum segredo real no pacote).
**Schema:** `runtime-events.schema.json` → JSON válido ✅.

### Invariantes arquiteturais — CONFIRMADOS (lidos em contract + prodex-runtime-invariants)
- ✅ Go = control plane; Rust/prodex = runtime plane (Authority Model §2).
- ✅ Um roteador runtime por sessão ("one runtime router per session", §6; "prevents competing routers", §1).
- ✅ Rust decide request em voo; Go não re-roteia commitado.
- ✅ Eventos voltam ao Go só p/ observability/ledger, não redecidem request commitado.
- ✅ Fail-closed (Go e Rust, §7) incl. profile switch com auth inválida.
- ✅ SQLite proibido p/ estado compartilhado (readyz + invariants).
- ✅ Afinidade de continuation / previous_response_id; redeem guard; redaction blocking; kill switch override.

### Gates pré-deploy — DEFINIDOS (12/12 docs presentes), NÃO EXECUTADOS
Contratos/planos existem e estão coerentes, mas ainda **sem evidência de execução** (smokes).

### BLOQUEADORES de DEPLOY REAL (ativos)
1. Runbook PROD (F7) **não aprovado pelo dono**.
2. Vendor capabilities **`not_validated`** (OpenCode, adapters Kiro/Antigravity/Cline, `codex_redeem`) — exigem validação por fonte oficial OU aceite explícito do dono.
3. Smokes de execução pendentes: `readyz`, policy apply, session start/stop, **kill switch**, event stream, **redaction**, profile fail-closed.
4. Reset-claim: validação empírica pendente (conta real).
5. Sidecar saudável + Go container verde: a comprovar na execução.

### VEREDITO
- **GO para AGENT DISPATCH** — baseline obrigatório aceito (contratos válidos, invariantes preservados, segredos limpos). Os 8 agentes podem iniciar os streams (com check-in/out em disco).
- **NO-GO para DEPLOY REAL** — travado pelos bloqueadores acima; só após runbook F7 aprovado pelo dono + smokes verdes + vendor caps resolvidas.
