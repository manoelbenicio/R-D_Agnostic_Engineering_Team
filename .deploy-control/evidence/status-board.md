# Status Board

Status: PRE-DEPLOY INITIAL

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

- F1 contract acceptance.
- F4 state/security acceptance.
- F7 deploy runbook acceptance.
- F6 conformance plan acceptance.
- Kill switch smoke.
- Redaction smoke.
- Sidecar readiness smoke.


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
