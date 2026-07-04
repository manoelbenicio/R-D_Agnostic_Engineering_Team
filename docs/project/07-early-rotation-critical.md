# FEATURE CRÍTICA — Rotação Antecipada Zero-Interrupção (banner de pré-esgotamento)

**Criticidade: MÁXIMA (corporativa).** Sem esta feature, o produto continua
semi-manual: quando a conta esgota (5h ou tokens), um humano precisa renovar a
sessão à mão. Isso quebra contratos e invalida TODO o trabalho anterior. Atenção,
dedicação e rigor de teste = iguais ou maiores que o resto. NÃO é feature "simples".

## Objetivo (definição de pronto = zero intervenção humana)
Quando o vendor sinaliza aproximação do limite (banner "<10% left" ao vivo, na
saída do agente), o sistema deve **rotacionar de conta ANTES do hard-stop** e
**retomar o trabalho sozinho** — o humano nunca toca.

## Loop completo (o que a feature exige)
1. **Captura ao vivo** do banner: o daemon JÁ lê a saída do agente via
   `session.Messages` → `case agent.MessageText` (daemon.go ~4124). É AQUI que o
   banner deve ser inspecionado, enquanto o agente trabalha (não só no fim da task).
2. **Detecção do pré-aviso**: `warnbanner.go` parseia banner + % restante + reset time,
   por vendor. DISTINTO de "limit reached" (reativo, já existe).
3. **Gatilho de rotação antecipada**: ao detectar o banner, chamar rotação com razão
   PROATIVA antes da task falhar (reusar rotationService.OnExhaustion / caminho novo).
4. **Retomada transparente**: trocar conta (CredentialAccountHome novo) e continuar
   sem intervenção. Preservar backward-compat: sem rotação configurada, comportamento AS-IS.
5. **Cobertura por vendor**: codex/kiro/antigravity — cada texto de banner CONFIRMADO
   contra a tela real (não inventar strings).

## Divisão de streams (disjuntas, sem colisão)
- **W-WARNBANNER (CODEX-2)** — arquivo NOVO `warnbanner.go` (+test): só o parser do
  banner por vendor. Não toca daemon/contract. Entrega isolada e testável.
- **W-EARLYROT (Opus, serial, hotspot)** — integrar o parser no loop
  `session.Messages` do daemon (~4124): ao `MessageText`, rodar o WarningDetector; se
  approaching, disparar rotação antecipada + re-dispatch. Dono único: Opus (hotspot).

## Ponto de integração confirmado (código real)
- `server/pkg/agent/agent.go`: `Session.Messages <-chan Message`; `MessageText`.
- `server/internal/daemon/daemon.go` ~4038: `case msg, ok := <-session.Messages:`;
  ~4124: `case agent.MessageText:` (msg.Content = texto ao vivo do agente).
- Rotação já existe: `d.rotationService.OnExhaustion(...)`, `d.rotateTaskOnExhaustion(...)`.

## Rigor de teste (nível corporativo)
- Unit: warnbanner por vendor (banner dispara; "limit reached" NÃO; texto normal NÃO;
  parse de % e reset time; vendor desconhecido).
- Integração: simular MessageText com banner no loop → prova que dispara rotação
  antecipada e re-despacha, SEM esperar falha da task.
- E2E: com Postgres real, banner → troca de conta → retomada, tudo automático.
- Backward-compat: sem rotação configurada, banner não quebra nada (AS-IS).
