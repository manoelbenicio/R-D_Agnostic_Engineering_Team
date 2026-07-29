# PLAN — G1: Freeze de contratos, IDs, owners e files

gate saída: 4 agentes operam sem editar mesmo hotspot; contratos neutros congelados.

## Tasks (OpenSpec tasks.md §1-§2)
- [x] 1.1 [Codex1] Freeze terminologia Agent Brain, fronteira cold/hot, contrato CLIKind/RouteModel/RouterOwner, inventário de compatibilidade — EV-G1-01
- [x] 1.2 [Codex1] Auditar baseline preservado `persist-prodex-runtime-integration` + registrar ownership exclusivo dos hotspots (PD-01 resolvida) — EV-G1-05
- [x] 1.6 [Codex1] Aprovar model set inicial, fallback chains, stable-key scope, endpoint topology, capacity target, cutover blockers (com Codex4 inputs 1.3-1.5)
- [x] 2.1 [Codex1] Interfaces/tipos neutros em arquivos novos (não mexer no daemon ativo)
- [x] 2.2 [Codex1] Nomes neutros de config, precedência de aliases, gateway-required, secret-file ref, readiness, schema tier 20/50/100
- [x] 2.3 [Codex1] Compatibility facade p/ API/token/RouterOwner/env/config/CLI/brief legados
- [x] 2.4 [Codex1] Publicar interfaces+ownership congelados p/ Codex 2-4; só Codex1 integra entrypoints

## Pré-requisitos (gate)
- G0 concluído; PD-01 resolvido; §7.1 = AUTORIZADO.

STATUS: IN_PROGRESS (Codex1 scope COMPLETE; Codex2-4 documentary inputs produced; live acceptance remains gated).
