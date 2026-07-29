# PLAN — G0: Governança e rebaseline OpenSpec↔GSD

gate saída: nenhum req/comp/iface/task órfão; worktree resolvida (PD-01); GSD v3 aprovado; §7.1 ainda NÃO necessário p/ G0 (planejamento read-only).

## Tasks (OpenSpec tasks.md §0)
- 0.1 [DONO] Aprovar hierarquia/fontes, G0-G8, ETA, preservar v2.1 — EV-G0-01 ✅
- 0.2 [DONO] Regra histórica de planning owner registrada; transição posterior para
  Kiro/Opus-4.8 + Codex#56#A está em D-V3-13 — EV-G0-02 ✅
- 0.3 [TL] Criar GSD v3 PROJECT/REQUIREMENTS/ROADMAP/STATE/DECISIONS/RISKS + phase plans — EV-G0-03 ✅
- 0.4 [TL] Criar TRACEABILITY/COMPONENT/INTERFACE/REMOVAL/FILE_OWNERSHIP/EVIDENCE_INDEX — EV-G0-03 ✅
- 0.5 [TL] Disposição formal dos 6 changes herdados; bloquear execução concorrente de superseded — EV-G0-05 ✅
- 0.6 [TL + Codex1] Auditar órfãos (req/comp/iface/task/owner/ev); fechar todos antes da Wave 0 — EV-G0-04 ✅ (PD-01 resolvida por preservação auditável)
- 0.7 [DONO] Aprovar GSD v3 baseline; só então marcar AUTORIZADO Waves 0-3 — EV-G0-01 ✅

## Critério de saída
- PD-01 resolvida por preservação auditável; disposição persistida em DECISIONS/REMOVAL_REGISTER/FILE_OWNERSHIP.
- Orphan audit: 85 tasks × specs × P01-P34/SC01-SC10 sem órfãos (exceto PD-01 fechado).
- Dono aprova GSD v3. §7.1 permanece gate para G1+.

STATUS: COMPLETE (dono autorizou Waves 0–3/tier 20; PD-01 resolvida; G1 liberado).
