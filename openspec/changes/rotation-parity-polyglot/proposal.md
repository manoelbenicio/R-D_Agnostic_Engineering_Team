# Proposal — Rotation-Parity Polyglot (prodex AS-IS em PROD → alvo Go L4 + Rust L2)

## Why
A rotação de contas em Go cobre o **caminho frio** (decisão por sessão), mas o produto exige agora
features de **caminho quente** — sobretudo **token-saver / Smart Context** (mandatório) — além de
reset-claim headless e cobertura multi-vendor não-uniforme. Reconstruir o caminho quente em Go é
inferior (risco semântico/protocolo + cauda p95) e lento. O `prodex` (Apache-2.0, Rust) já entrega
tudo isso, na linguagem certa para o hot path, mantido ativamente. Decisão do dono (ver ADR-001):
**usar o `prodex` AS-IS em PROD agora** para obter as features imediatamente, e **endurecer via fork
(polyglot Go+Rust) no próximo marco**.

## What Changes
- **Agora:** Multica Go (L4) passa a **orquestrar o `prodex`** (lança `prodex`/`prodex s` no lugar de
  `codex` cru), **pinado por versão/commit**, **direto em PROD**, usando assinaturas próprias. Ativa
  as features: rotação pré-commit, afinidade de sessão, **token-saver/Smart Context**, modos,
  reset-claim.
- **Alvo (próximo marco):** endurecer como **Rust L2 sidecar via fork do `prodex`**, com contrato
  local estrito com o Multica Go e **invariante de roteador único por sessão**.
- **Aposenta como runtime** o `rotation-router` Go (policy/fallback/loadbalance/proactive_reset): a
  autoridade de runtime passa ao `prodex`/Rust L2. O Go retém control plane (cadastro, policy,
  approved-accounts, observability).
- Introduz **matriz de capabilities por provider** (não interface genérica).

## Scope
- **Fundação (pré-requisito, antes do F0):** provisionar o binário prodex (build do source pinado + Rust + verify). Ver capability `prodex-runtime-provisioning`.
- Vendors: **Codex, Kiro, Antigravity, Cline, OpenCode** — **OpenCode a reavaliar** (projeto ARQUIVADO, sucessor Crush): disabled / descopar / migrar (decisão documentada no F5).
- REUSA a fundação fria já verde (isolamento `CODEX_HOME`/`XDG`/`HOME`, detecção, Postgres, observability).
- State compartilhado: **Postgres** (SQLite proibido); **migrations reversíveis**.
- Guarda-corpos em PROD **provados por teste** (não só doc): knobs nativos do prodex (shadow/canary),
  **kill switch testado**, logs scrubbed, **rollback 1-cmd testado**.
- **QA exaustivo em container ANTES do deploy** (nunca bypassado); deploy direto em PROD só após QA verde.

## Non-Goals
- **Kimchi** — removido do escopo.
- **Não** reimplementar Smart Context em Go.
- **Não** migrar o Multica L4 para Rust (L4 é frio e já validado).
- **Não** virar gateway metered de terceiro (usa assinaturas próprias).
- **ToS jurídico** — não-aplicável (sem Claude Code; Opus via Kiro/AWS).

## Impact
- Ganho imediato de todas as features do prodex em PROD, sem custo metered.
- Caminho quente na linguagem certa (Rust) desde o dia 1 (prodex as-is), com endurecimento
  proprietário (fork) planejado.
- Prioridade: **reset-claim é baixa** (caminho frio/aleatório) — será feito, mas por último.
- Runtime-router Go supersedido; documentação e board reconciliados (ver `openspec/changes/rotation-router`).
