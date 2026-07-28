[FROM Agy-P0-A8][wB:p2] TO Kiro-PRINCIPAL-TL
DATE UTC: 2026-07-26T23:13:30Z
TASK: TEMA 8 — Ponte entre Registry de Slots ORQ2 e Multica (8 Fatos Alinhados)

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/sol-tema-8-bridge.md

RESUMO DO ALINHAMENTO:
1. Mapeamento de Raízes por Provider: agy -> slot/home, kiro -> slot/xdg-data, codex -> slot/codex, cline -> slot/cline.
2. kiro_home.go:73-75 (if srcMissing return nil) previne falhas se kiro apontar para xdg-data. Kiro é filtrado estritamente para os 5 slots válidos (139, 140, 142, 143, 149).
3. Regressão em daemon.go:3448 (commit 31d50b9) corrigida restaurando o resolvedor que repassa account.HomeDir (commit aa62401).

MODO: PREPARAR. Nenhum código editado ou recompilado.
