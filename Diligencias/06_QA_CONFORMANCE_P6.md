# P6 — Diligência: QA / Conformance EXAUSTIVO (C1–C6)  — ÚLTIMA FASE DESTE ESCOPO

> **SEM BYPASS.** Todos os gates rodam exaustivamente com evidência real em container/sidecar.
> Valida ANTES do PROD — quebra o nó circular (não exige o runtime já no ar).

## Objetivo
Provar, com evidência reproduzível, que o runtime (prodex AS-IS) satisfaz todos os critérios de aceite
antes de qualquer deploy.

## REQ-IDs
REQ-13..REQ-18. Spec: `specs/qa-conformance/spec.md`.

## Pré-requisitos
- P3 (integração) + P4 (state/security) + P5 (matriz) verdes.

## Matriz de conformance (cada linha = evidência obrigatória)

| Gate | O que prova | Evidência esperada |
|------|-------------|--------------------|
| **C1** | Conformance por capability (não por rótulo) | por-capability, comportamento observado vs esperado |
| **C2** | Replay: long-session, tool-calls, `previous_response_id` | transcript replay verde |
| **C3** | Replay: compact, SSE, WebSocket | streams íntegros; sem corrupção |
| **C4** | Troca de perfil **fail-closed** | nunca reusa credencial anterior com perfil novo inválido |
| **C5** | Smart Context **shadow→canary→live** | medição antes/depois + **fallback exato automático** em risco estrutural |
| **C6** | Tripla **CODEX_HOME × prodex × Herdr** | isolamento por conta sem clobber (AccountHome mandatório) |
| **+**  | Herdr coordination smoke | agent send/notification/events provados |

## Passos
- 6.1–6.7 executar C1–C6 + Herdr smoke, cada um gravando evidência **scrubbed** em `.deploy-control/evidence/`.
- Reclassificar qualquer "DONE" que seja plan-only/dry-run → **IN_PROGRESS** até rodar de verdade.

## Verificação / evidência
- Um arquivo de evidência por gate (comando + resultado), sem segredo.
- Container verde; sidecar saudável.
- Validador (Opus/Tech-Lead) **re-roda** e confirma (não confia no tail).

## Critério de GATE (DONE) — libera o Deploy (P7)
✅ **TODOS** C1–C6 verdes com evidência empírica (não plano) · ✅ Herdr smoke verde · ✅ fail-closed provado ·
✅ Smart Context com fallback provado · ✅ isolamento tripla sem clobber.

## Regra de ouro do QA
Nada aqui é "verde de tela". Só conta com **evidência reproduzível em container**. QA exaustivo é
pré-condição inegociável do deploy — nenhum gate pode ser pulado.

## Depois do P6
Com P6 verde → **P7 Deploy** (kill-switch + rollback testados → deploy direto em PROD). Fora deste escopo de diligência.
