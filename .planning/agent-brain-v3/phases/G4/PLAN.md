# PLAN — G4: Protocolo + falhas + segurança + development tier 20

gate saída: protocolo, isolamento, falhas, rollback e bounded-capacity tier 20 comprovados
em desenvolvimento; nenhuma alegação de produção.

## Tasks (tasks.md §8 + §9 parcial)
- 8.1 [Codex2+4] Conformidade authenticated models/capabilities + stream/non-stream por protocol family/model route
- 8.2 [Codex3+4] Paths Claude/Codex/Kimi/GLM/NVIDIA/Agy com tools/reasoning/cancel/usage/errors determinísticos
- 8.3 [Codex3] Child envs/homes/process trees/logs sem cred provider/auth files/direct endpoints
- 8.4 [Codex2+4] Strict concurrent RR (independent requests) + affinity (Responses continuation/cache/tool turns)
- 8.5 [Codex2+4] Expired access/revoked refresh/quota/401/403/429 account/429 global/5xx/timeout/malformed upstream
- 8.6 [Codex2+4] Safe retry pre-output; no replay post-output/tool; dedup; cancel release
- 8.7 [Codex2+4] Account add/remove/quarantine/re-entry + OmniRoute restart/config rollback under load
- 8.8 [Codex4] Evidence contra cada checklist/parity ID; parar cutover p/ blocker sem waiver
- 9.1 [Codex4] Run 20-task profile (mix/stream/tool/latency/fairness/CPU/mem/sockets/retries)
- 9.2 [Codex1] Habilitar tier 20 só se thresholds passar + counters reconcile
Evidence: EV-G4-01..08, EV-G4-CAP, EV-G4-COD/ADP/NIM/AGY

Pré-requisitos: G3 (slice). STATUS: IN_PROGRESS 2026-07-18T03:05:35Z in isolated Herdr
streams `w3:p8/p9/pA`; independent G3 review `w3:pB`. Portões §7.4 continuam obrigatórios
para qualquer claim além de synthetic development validation.
