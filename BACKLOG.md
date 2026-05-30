# AgentVerse — Backlog pós-v1

Fonte de verdade: `openspec/changes/`. O change `milestone-1-canvas-deploy-run`
é o v1 já entregue. Os 10 abaixo são o backlog pendente, ordenados por
criticidade (segurança/produção > billing > tech-debt > cosmético).

| # | Criticidade | Change | Entrega | Risco |
|---|---|---|---|---|
| 1 | 🔴 Crítica | `cloud-runtime-deployment` | Runtime cloud CAO (Cloud Run/GKE), Firebase Auth, isolamento por tenant, secrets/networking | Alto — auth + multi-tenant + segredos |
| 2 | 🔴 Alta | `validation-proxy` | Bloqueia `handoff`/`assign`/`send_message` fora da topologia do canvas (R3 em runtime) | Alto — segurança de runtime |
| 3 | 🟠 Média | `finops-tier2-token-parsing` | Custo real por token (OpenAI/Anthropic/Google/AWS), usage por sessão/canvas | Médio — fallback Tier 1 existe |
| 4 | 🟠 Média | `tech-debt-voice-coverage-gap` | Extrai `command-executor.ts`; cobertura `src/voice/` ≥70% | Médio — hot path do VoicePanel |
| 5 | 🟠 Média | `tech-debt-voice-event-bus` | `CanvasCommandBus` remove imports laterais voice→canvas (§14.2) | Médio — muda contrato voice↔canvas |
| 6 | 🟡 Baixa-Média | `tech-debt-smoke-voice-real-flow` | Remove workarounds do smoke; pipeline real de voz no CI | Baixo-Médio — flakiness |
| 7 | 🟡 Baixa | `design-system-indra-alignment` | Swap de tokens SENTINEL→Indra (só tokens) | Baixo — gates verdes |
| 8 | 🟡 Baixa | `tech-debt-keystore-validator-coverage` | Contract tests live gated por `KEYSTORE_LIVE=1` + nightly | Baixo — gated |
| 9 | ⚪ Mínima | `tech-debt-schema-version-shared` | Move `SCHEMA_VERSION` p/ `src/shared/` (D9) | Mínimo — namespace |
| 10 | ⚪ Mínima | `tech-debt-react-refresh-cleanups` | Corrige warnings react-refresh (CanvasBuilder/CanvasList/cost-warning) | Mínimo — cosmético |

## Ordem de execução

Resolução do mais contido/baixo-risco para as features maiores: **9 → 10 → 5 →
4 → 6 → 8 → 7 → 3 → 2 → 1**.

## Status (resolução)

| # | Change | Status |
|---|---|---|
| 10 | `tech-debt-react-refresh-cleanups` | ✅ Concluído — constantes movidas p/ `cost-warning-constants.ts`, 0 warnings react-refresh |
| 9 | `tech-debt-schema-version-shared` | ✅ Concluído — `SCHEMA_VERSION` em `@/shared`, shim de re-export, imports atualizados |
| 5 | `tech-debt-voice-event-bus` | ✅ Concluído (já no código) — `CanvasCommandBus` + adapter, sem imports laterais |
| 4 | `tech-debt-voice-coverage-gap` | ✅ Concluído (já no código) — `src/voice` 88% cobertura |
| 6 | `tech-debt-smoke-voice-real-flow` | ✅ Concluído (já no código) — sem `force:true`/`useVoiceStore`, polyfill STT |
| 8 | `tech-debt-keystore-validator-coverage` | ✅ Concluído (já no código) — 8 contract tests gated, nightly, doc |
| 7 | `design-system-indra-alignment` | ✅ Concluído (já no código) — tokens Indra, gates verdes |
| 3 | `finops-tier2-token-parsing` | ✅ Núcleo concluído — parsing/cost/persistência/UI; captura via CAO **deferida** |
| 2 | `validation-proxy` | 🟡 núcleo concluído — `topology-guard` puro + testes; interceptação CAO-side **deferida** |
| 1 | `cloud-runtime-deployment` | ✅ ~95% — código + login UI + gates (lint/typecheck/test/build); deploy GCP/Docker e per-tenant **deferidos** (precisam de creds/decisões) |

Gates finais: lint 0 erros (3 warnings pré-existentes), typecheck limpo, `vitest`
389 passed / 8 skipped, build OK, bundle 470 KB / 1536 KB. Itens deferidos
exigem servidor CAO, credenciais GCP ou decisões de arquitetura — documentados
nos respectivos `tasks.md`.

## Melhorias futuras (não-mandatórias)

- **validation-proxy (enforcement CAO-side)** — decisão arquitetural **D6/R3**
  da spec original: o v1 mitiga R3 no nível de prompt, e o enforcement
  determinístico é **opcional**, só recomendado para **cloud / multi-tenant**
  (feature de confiança do cliente). Núcleo (`topology-guard.ts`,
  `validation-proxy.ts`) pronto e testado, mas **não plugado** — ligar ao
  roteador MCP do CAO exige plugin (auditar) ou fork (bloquear). **Mantido no
  backlog como melhoria futura.**
- **finops-tier2 (captura real de usage)** — CAO não expõe usage por turn;
  parser pronto, captura adiada. Fallback Tier 1 (wall-clock) cobre o caso atual.
- **cloud-runtime (deploy GCP)** — bloqueado por: cao-server é localhost-only
  (rejeita Host/IP cloud), credenciais headless dos workers, e recursos GCP
  faturáveis. Local 100% funcional.

