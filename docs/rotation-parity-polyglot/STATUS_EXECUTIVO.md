> **PLANO CORRIGIDO (2026-07-04) — leia antes de agir.** O plano anterior assumia o binario prodex instalado (ERRO). Correcoes herdadas:
> - **P0 FUNDACAO e pre-requisito de TUDO**: provisionar/buildar o binario prodex (source -> ~/runtime/prodex-src; instalar Rust; `cargo build --release`; verificar pin v0.246.0/7750da9b + hash; setar `MULTICA_PRODEX_ENABLED/PATH/VERSION/COMMIT` + `PRODEX_HOME`). Ref: Diligencias/00_FUNDACAO_P0.md + openspec specs/prodex-runtime-provisioning.
> - **Kill-switch + rollback: TESTADOS** (nao so documentados) antes do deploy.
> - **QA EXAUSTIVO em container ANTES do deploy** (NUNCA bypassado); deploy direto em PROD depois.
> - **OpenCode: ARQUIVADO** (sucessor Crush) -> disabled/descope (decisao F5).
> - **IPv6 desabilitado** nos builds: `docker run --sysctl net.ipv6.conf.all.disable_ipv6=1 ...`.
> Fonte de verdade: `openspec/changes/rotation-parity-polyglot/` + `.planning/` + `Diligencias/`.

# STATUS EXECUTIVO (C-LEVEL) — Rotation-Parity Polyglot

> **Gerado:** 2026-07-04 20:26:42Z · **Fonte:** Herdr socket + board (manoelneto-laptop) · **Mantido por:** Tech-Lead (Opus 4.8)

## Panorama: **OVERALL 40%** — FALTAM **12/20**

✅ 8 concluídas · 🔄 0 em curso · ⬜ 12 em espera · ⛔ 0 bloqueadas · ❌ 0 falhadas · ⊘ 0 canceladas

| # | Item | Tarefa | Status | Dono | Prog | ETA | Observação |
|---|------|--------|--------|------|-----:|-----|------------|
| 1 | F0 | Deploy prodex AS-IS em PROD | ⬜ EM ESPERA | Codex#5.5#C/D | 0% | 4h | GATED — aguarda aprovação do dono + smokes verdes |
| 2 | F1 | Contrato Go<->L2 + eventos (+conformance) | ✅ DONE | Codex#5.5#A | 100% | 0m | — |
| 3 | F2 | prodex fork map / runtime invariants | ✅ DONE | Codex#5.5#B | 100% | 0m | — |
| 4 | F3 | Go integration + lancar prodex | ✅ DONE | Codex#5.5#A | 100% | 0m | — |
| 5 | F4 | State/security + redaction/no-SQLite smoke | ⬜ EM ESPERA | GLM#52#B | 0% | 6h | não iniciado |
| 6 | F5 | Vendor capability matrix | ✅ DONE | Gemini#Pro | 100% | 0m | — |
| 7 | F6 | QA/conformance C1-C6 + PROD validation | ⬜ EM ESPERA | GLM#52#A | 0% | 8h | não iniciado |
| 8 | F7 | DevOps runbook + rollback + smoke scripts | ✅ DONE | Codex#5.5#D | 100% | 0m | — |
| 9 | F8 | Ops triage / status board | ✅ DONE | Gemini#Flash35 | 100% | 0m | — |
| 10 | F9 | Reset-claim (empirico) | ✅ DONE | Codex#5.5#B | 100% | 0m | — |
| 11 | G1 | Tripla CODEX_HOME x prodex x Herdr validada | ⬜ EM ESPERA | GLM#52#A | 0% | 3h | não iniciado |
| 12 | G2 | Coordenacao Herdr operacional (smoke) | ⬜ EM ESPERA | opus-4.8-orchestrator | 0% | 2h | não iniciado |
| 13 | G3 | Roteador unico por sessao provado em teste | ⬜ EM ESPERA | Codex#5.5#A/C | 0% | 2h | não iniciado |
| 14 | G4 | Troca de perfil fail-closed | ⬜ EM ESPERA | Codex#5.5#C | 0% | 2h | não iniciado |
| 15 | G5 | Smart Context shadow->canary->live+fallback | ⬜ EM ESPERA | GLM#52#A | 0% | 6h | não iniciado |
| 16 | G6 | Reset-claim matriz empirica c/ evidencia | ⬜ EM ESPERA | Codex#5.5#B | 0% | 4h | não iniciado |
| 17 | G7 | Conformance por capability (nao por rotulo) | ⬜ EM ESPERA | GLM#52#A | 0% | 4h | não iniciado |
| 18 | G8 | Secrets redaction test (logs/traces/audit) | ✅ DONE | GLM#52#CLINE#B | 100% | 0m | — |
| 19 | G9 | Postgres/Redis (sem SQLite) + migrations | ⬜ EM ESPERA | GLM#52#B | 0% | 3h | não iniciado |
| 20 | G10 | Container verde + sidecar + killswitch+rollb | ⬜ EM ESPERA | Codex#5.5#C/D | 0% | 3h | não iniciado |

## Pendências (o que falta até GA)
- **F0 — Deploy prodex AS-IS em PROD** (⬜ EM ESPERA, dono Codex#5.5#C/D, ETA 4h): GATED — aguarda aprovação do dono + smokes verdes
- **F4 — State/security + redaction/no-SQLite smoke** (⬜ EM ESPERA, dono GLM#52#B, ETA 6h): não iniciado
- **F6 — QA/conformance C1-C6 + PROD validation** (⬜ EM ESPERA, dono GLM#52#A, ETA 8h): não iniciado
- **G1 — Tripla CODEX_HOME x prodex x Herdr validada** (⬜ EM ESPERA, dono GLM#52#A, ETA 3h): não iniciado
- **G2 — Coordenacao Herdr operacional (smoke)** (⬜ EM ESPERA, dono opus-4.8-orchestrator, ETA 2h): não iniciado
- **G3 — Roteador unico por sessao provado em teste** (⬜ EM ESPERA, dono Codex#5.5#A/C, ETA 2h): não iniciado
- **G4 — Troca de perfil fail-closed** (⬜ EM ESPERA, dono Codex#5.5#C, ETA 2h): não iniciado
- **G5 — Smart Context shadow->canary->live+fallback** (⬜ EM ESPERA, dono GLM#52#A, ETA 6h): não iniciado
- **G6 — Reset-claim matriz empirica c/ evidencia** (⬜ EM ESPERA, dono Codex#5.5#B, ETA 4h): não iniciado
- **G7 — Conformance por capability (nao por rotulo)** (⬜ EM ESPERA, dono GLM#52#A, ETA 4h): não iniciado
- **G9 — Postgres/Redis (sem SQLite) + migrations** (⬜ EM ESPERA, dono GLM#52#B, ETA 3h): não iniciado
- **G10 — Container verde + sidecar + killswitch+rollbk** (⬜ EM ESPERA, dono Codex#5.5#C/D, ETA 3h): não iniciado

## Decisões pendentes do dono (owner-only)
- **F5 vendor sign-off:** aceitar capabilities `not_validated` como disabled-by-default (ACCEPT recomendado p/ ausências factuais; #7 Smart Context = gate; #6 OpenCode arquivado = descopar).
- **F7 deploy PROD:** **NO-GO** até G5/G10/F4/F6 verdes + runbook reconciliado.

## Próximo marco (gate de GA)
Deploy prodex AS-IS em PROD (F0) **somente após**: G5 Smart Context (shadow→canary), G10 container/kill-switch/rollback, F4 redaction/no-SQLite e F6 conformance — todos **verdes com evidência** — e aprovação do dono.
