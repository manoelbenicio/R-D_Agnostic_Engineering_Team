# DECISIONS / CLOSED ISSUES — NÃO RE-ABRIR, NÃO RE-PERGUNTAR

> TL/agentes: consulte ESTE arquivo ANTES de perguntar ou re-diagnosticar qualquer coisa.
> Cada item abaixo está FECHADO com base em OpenSpec/arquitetura/decisão do dono. Reabrir = erro.

## D-001 — AUTENTICAÇÃO É OAUTH, NÃO EXISTE API KEY (FECHADO — CRÍTICO)
- Fonte: `openspec/.../design.md §5` — `auth_mode: oauth_profile | cli_native_store` p/ os vendors reais.
- O sistema INTEIRO usa OAuth (login nativo das CLIs). **NÃO há `sk-`/api_key.** NUNCA peça API key ao dono.
- Rotation-Parity: `rotation_mode: profile_pool`, `reset_claim_mode: codex_redeem`.

## D-002 — QUOTA EXAURIDA NÃO É BLOCKER: É O CASO DE USO (FECHADO)
- Se um profile OAuth bate limite (ex.: Codex free até 3-Ago), a resposta é o PRÓPRIO produto:
  1) ROTACIONAR p/ outro profile OAuth do pool com quota (`profile_pool`), ou
  2) `prodex redeem <profile>` (reset-claim), ou
  3) rotear o turno de um AGENTE com sessão OAuth viva (os agentes fazem chamadas reais AGORA).
- **NUNCA** resolver com API key. Isso é o coração do milestone (Rotation-Parity).

## D-007 — ISOLAMENTO DE CREDENCIAIS POR AGENTE É MANDATÓRIO (FECHADO — CRÍTICO/SEGURANÇA)
- Cada agente tem credential store/CODEX_HOME PRÓPRIO e ISOLADO. **NUNCA** compartilhar/importar/copiar
  a pasta de credenciais de um agente para outro. Há um FIX implementado que exige isolamento total.
- Violar (ex.: `prodex profile import <vendor>` puxando creds de outro agente) → **CRASH DO SISTEMA INTEIRO.**
- Prova de sessão real (12.3): cada agente roda o PRÓPRIO turno com AS PRÓPRIAS creds isoladas, pelo gateway.
  O agente que tem OAuth vivo+quota prova o path na PRÓPRIA env. NUNCA importar credencial p/ dentro de outro agente.
- Relacionado: REQ-17 (CODEX_HOME × prodex × Herdr coexistindo sem clobber — isolamento provado).

## D-003 — 4 VENDORS REAIS (FECHADO)
OpenAI/Codex, Antigravity, OpenCode/GLM5.2, Kiro/Opus4.8. OpenCode ATIVO (não arquivado). Cline não é alvo.

## D-004 — EVIDÊNCIA FAKE P12 (FECHADO)
Sessão localhost/fake-upstream/smoke = INVALID (marcada). EVIDENCE_CONTRACT impede repetição. Não reusar.

## D-005 — P11 local_estimate / gateway 404 (FECHADO)
Esperado localmente (sem upstream real). Números reais vêm de 12.3 via OAuth real. Não é bug a investigar.

## D-006 — GOLDEN RULES + CHECK-IN/OUT MANDATÓRIOS (FECHADO)
Em vigor (`GOLDEN_RULES.md`, `CHECKIN_OUT.md`). Não é opcional, não debater.

## D-008 — O TL ORQUESTRA TODOS OS AGENTES; NÃO EXECUTA NADA (FECHADO — CRÍTICO)
- O TL SÓ orquestra: planeja (a partir dos docs), ATRIBUI tasks aos agentes, valida evidência (manda re-rodar), sign-off. **Só o TL commita.**
- O TL **NUNCA** roda comando/`prodex`/bash, NÃO produz código, NÃO escreve evidência "as <agente> (proxy)".
- Quem EXECUTA são os agentes Codex (A/C/D/B), cada um na PRÓPRIA env ISOLADA, com PRÓPRIO check-in (D-007).
- Violação observada 2026-07-06: TL rodando `prodex`/credential direto + evidência "as Codex-5.5-D (TL proxy)" — CAUSA-RAIZ da evidência fake e do risco de isolamento. Proibido.

---
Regra: em dúvida sobre algo AQUI → é FECHADO, siga a decisão. Dúvida NOVA (não coberta) → ASK_KIRO.md + `@KIRO:`.

## D-007: Strict Agent Isolation & TL Execution Ban
- **Context:** The TL orchestrator (Gemini) directly executed `prodex gateway` and imported profiles from other agents, violating credential boundaries and crashing the system structure.
- **Decision:** The TL MUST NEVER execute terminal commands, start servers, or run code directly. The TL only plans, assigns tasks, and evaluates evidence.
- **Isolation:** Credential sharing between agents is STRICTLY PROHIBITED. Each agent (Codex A/B/C/D, etc.) must run in its own fully isolated environment with its own `CODEX_HOME` and auth store. An agent with a live OAuth session MUST prove paths using its OWN credentials within its OWN environment.
- **Enforcement:** TL delegates execution exclusively to designated agents (e.g., Codex-5.5-D on w3:p9) using herdr commands.

## D-008: TL is STRICTLY ORCHESTRATOR
- **Context:** The TL directly executed commands and generated code, which is forbidden.
- **Decision:** The TL MUST NOT execute any command, modify code, or generate production artifacts directly. The TL exclusively manages planning, documentation, dispatching agents via herdr, and verifying results. All execution is deferred to isolated subagents.
