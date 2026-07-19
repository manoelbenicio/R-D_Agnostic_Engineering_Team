# PLAN — G3: Integração serial (Codex 1, hotspot único)

gate saída: vertical slice sem credencial provider e sem dual router; `RouterOwner=omniroute`.

## Tasks (tasks.md §7)
- 7.1 [Codex1] Review/integrate gateway, runtime/CLI, ops contra contratos; rejeitar cred escondida/dup routing
- 7.2 [Codex1] Fiar config neutral/aliases via central daemon/config/command entrypoints (sole editor)
- 7.3 [Codex1] Substituir credential-account resolution por CLIKind+RouteModel+trusted profile (gateway-required)
- 7.4 [Codex1] Aplicar sanitized env + controlled CLI config após custom settings antes do launch
- 7.5 [Codex1] Gate admission/launch em readiness OmniRoute + auth key + model/protocol capability
- 7.6 [Codex1] RouterOwner=omniroute + correlation via launch/result/error/cancel
- 7.7 [Codex1] Desabilitar Prodex/L2 startup, Go rotation/retry, cred-home prep, account-selection (gateway-required tasks)
- 7.8 [Codex1] Manter legado isolado atrás de flag default-off (drain); nunca dois router owners
- 7.9 [Codex1] Health/readiness/config diagnostics neutros + redacted
- 7.10 [Codex1] Primeiro vertical slice (uma rota Claude ou Codex aprovada) SEM habilitar broad admission
Evidence: EV-G3-WIRE, EV-G3-04/05/06/07

Pré-requisitos: G2 (4 streams). STATUS: COMPLETE 2026-07-18T02:38:30Z — 44/85 overall;
EV-G3-WIRE/04/05/06/07 accepted for default-off synthetic development scope.

Execution constraints: Codex1 is the only central/hotspot editor. Preserve PD-01 without
reset/stash/revert/discard. PD-08 forbids all credential/auth reads or mutations. G3 must
remain default-off and fail-closed, must not dispatch Codex through the current Multica
daemon, and must finish with a development isolation smoke before G4 fan-out.
