# Tasks

> Execução: coders. Validação: Kiro. Check-in START/DONE por agente.

## Bloqueios (dono decide antes)
- [x] 0.1 Squad default: `Workspace Team`, com Kiro/Opus 4.8 como TL/Manager delegation-only, Codex 5.6 Sol high-thinking como coder preferencial/escape hatch direto e os demais coders disponíveis como membros.
- [x] 0.2 Limiar de quando o leader abre OpenSpec explore
- [x] 0.3 Roteamento default (chat→TL) por workspace ou por squad

## Implementação
- [x] 1.1 Identity/instructions do leader TL/Manager (protocolo esclarecer→openspec→planejar→delegar→sintetizar; marcador `## Squad Operating Protocol`)
- [x] 1.2 Squad TL/Manager default no setup do workspace (leader + membros)
- [x] 1.3 Roteamento default do chat: sem destino → squad TL; com `@agente` → direto (escape hatch)
- [x] 1.4 Garantir leader delegation-only (não produz; delega + sintetiza)

## Verificação (Kiro valida)
- [x] 2.1 Smoke: chat sem destino cai no TL; TL pergunta, delega a membro, sintetiza
- [ ] 2.2 Smoke: chat `@codex` vai direto ao agente (escape hatch funciona)
- [ ] 2.3 Check-ins DONE + evidência em `.deploy-control/`
