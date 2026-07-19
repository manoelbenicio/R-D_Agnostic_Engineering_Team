# Resposta do Arquiteto OmniRoute ao Checklist de Aceite

> **Operational interpretation updated 2026-07-18:** this response remains immutable
> architecture/authorization evidence. Because the system is non-production, its tier-20
> production-canary wording is executed as controlled development validation under D-V3-14;
> security, failure, rollback and capacity gates remain, while cutover/Prodex removal/50/100
> remain unauthorized.

> Documento de resposta ao `OMNIROUTE_REQUIREMENTS_HANDOFF.md`.
> Natureza: revisão de arquitetura/código (review-only). Itens marcados como
> "evidência pendente" exigem as execuções reproduzíveis de conformidade,
> failure-injection e capacidade exigidas pelo próprio checklist — uma revisão
> documental não as substitui.
> Nenhum segredo (chave, token, prompt, credencial) é exposto neste documento.

## 0. Baseline avaliado

- OmniRoute `3.8.48`, imagem `diegosouzapw/omniroute:latest` (**ação: fixar digest**), container `omniroute`, porta `20128`.
- Estado: `better-sqlite3` **single-node**, volume `omniroute-data`.
- Rede: anexado a `multica_default` (`omniroute:20128`) e exposto na LAN em `192.168.1.27:20128` (portproxy + firewall `192.168.1.0/24`).
- **Segurança aplicada nesta sessão:** `REQUIRE_API_KEY=true` (override de feature-flag no `key_value/feature_flags`). Sem chave → 401; chave registrada `[REDACTED-PREFIX]` → 200. Verificado. O prefixo anteriormente presente foi redigido em 2026-07-18; rotação permanece PD-07.

## 1. Go/No-Go do arquiteto

**GO condicional** para build + canary de 20 tarefas.
**NO-GO** para remoção do Prodex e tier de 100 até: conformidade de protocolo por modelo, failure-injection, single-flight refresh, readiness fail-closed, Smart Context em shadow/canary (ou waiver) e a decisão estado single-node vs. compartilhado.

## 2. Áreas críticas (resposta focada)

### 2.1 Protocolos

| Cliente | Formato | Rota | Status | Evidência |
|---|---|---|---|---|
| Claude Code | Anthropic Messages + SSE | `/v1/messages` | Suportado (conformidade pendente) | `src/app/api/v1/messages/route.ts` + tradutores Anthropic; combo Kimi 200 ao vivo |
| Codex | OpenAI Responses (não só Chat) | `/v1/responses` | Suportado (conformidade pendente) | `src/app/api/v1/responses/route.ts` + `[...path]`; `open-sse/utils/responsesStatePolicy.ts`; `wire_api=responses` |
| Kimi/GLM/NVIDIA | OpenAI Chat | `/v1/chat/completions` | Suportado (por-modelo pendente) | rota + combos `clinepass`/`nvidia` |
| Antigravity | endpoint direto | `/v1/antigravity` | Suportado (200 provado nas 4 contas `agy`) | testes desta sessão |

Gap: fixtures de conformidade por modelo (SSE, tools, thinking, function-call deltas).

### 2.2 Rotação
- Unidade = **uma requisição lógica independente** (não SSE/retry/tool turn). Evidência: `open-sse/services/combo.ts`, `account-selector`.
- Round-robin estrito, `stickyRoundRobinLimit=1`; atômico dentro do processo (event loop single-thread).
- **Ressalva (P23/Q18):** estado single-node SQLite + cursor em memória → rotação global estrita garantida apenas em instância única. Replicação exige estado compartilhado (Redis/Postgres) + reverificação.

### 2.3 Afinidade de continuação
- `responsesStatePolicy.ts` (`auto|strip|preserve`, default `auto`), `sessionAffinityPin.ts`.
- Parcial: comprovar que `preserve` fixa na conta de origem (teste ao vivo). Afinidade sobrepõe rotação só para a continuação dependente.

### 2.4 429 / circuit breaker
- Suportado: classificação account / model / provider-global / overload (`classify429`, `open-sse/config/providerErrorRules.ts`, `errorClassifier.ts`).
- Cooldowns por tipo (rate_limit ~60s, quota_exhausted ~1h), backoff com jitter, breaker com half-open (`accountFallback.ts`, `circuitBreaker`), `rate_limited_until` persistido.
- Gap: sessão de failure-injection para certificar.

### 2.5 Smart Context
- Parcial: subsistema `open-sse/services/compression/*` (modos off/lite/standard/aggressive/ultra/rtk/omniglyph/stacked; engines session-dedup/ccr/headroom/relevance/llmlingua/caveman; `fidelityGate`, `riskGate`, `pipelineGuards`, `cachingAware`, `resultMemo`, `eval/runner`, `/api/compression/preview`).
- Gap: comprovar SC01–SC10 (shadow → canary → fallback exato do request inteiro → self-check) OU waiver assinado.

### 2.6 Capacidade 20/50/100
- Camadas independentes: admissão (Brain) → global → rota/modelo (`concurrencyPerModel`) → por-conta (`max_concurrent`), filas limitadas (`queueTimeoutMs`).
- Regra: `pico_concorrente_por_rota ≤ Σ(contas_elegíveis × concorrência_por_conta)`.
- 20 recomendado para lançamento; 50 após ajuste; 100 após evidência + decisão de estado.
- Gap: relatório de carga reproduzível (mix, streaming, tools, p50/p95/p99, filas, CPU/mem/sockets, fairness).

## 3. As 20 questões arquiteturais (resolução condensada)

11 Suportado · 6 Parcialmente suportado (evidência pendente) · 1 Não suportado como implantado (Q18 rotação horizontal — single-node) · 2 esclarecimentos de capacidade/fronteira. Detalhe item a item no `OMNIROUTE_REQUIREMENTS_HANDOFF.md` e nos specs.

Destaques:
- Q1 unidade de rotação = requisição lógica (SUPORTADO).
- Q2 concorrência atômica single-node (SUPORTADO); multi-node PARCIAL.
- Q4 `previous_response_id` via `auto/strip/preserve` (PARCIAL — pin a comprovar).
- Q9 single-flight refresh (PARCIAL — evidência requerida).
- Q15 Smart Context (PARCIAL). Q16 reset/redeem = Codex only (`src/lib/usage/codexResetCredits.ts`).
- Q18 estado horizontal (NÃO SUPORTADO como implantado).

## 4. Decisões acordadas

1. **Autorização de implementação (4 agentes):** recomendação do arquiteto = prosseguir, limitado ao canary de 20, com os portões da Seção 1. Autorização explícita é do dono do produto.
2. **Nome provisório:** `Agent Brain` até a etapa de debrand (passo 10 da migração).
3. **Capacidade de lançamento:** iniciar em **20 tarefas**; 50 e 100 como etapas seguintes, cada uma liberada só com relatório de carga aprovado.

## 5. Evidência exigida antes do cutover total

1. Digest da imagem fixado + config redigida.
2. Conformidade automatizada: Anthropic Messages, OpenAI Responses, Chat Completions por modelo aprovado.
3. Failure-injection: expiry, revoked, quota, 429 (account/global), 5xx, timeout, stream quebrado, cancelamento, restart.
4. Relatório de capacidade 20/50/100 com fairness e recursos.
5. Segurança: encryption (AES-256-GCM `src/lib/db/encryption.ts`), escopo/rotação de chave, redação, isolamento management/inference, auditoria.
6. Decisão de estado: single-instance (com drain/rollback) ou backend compartilhado + reverificação de rotação.
7. Smart Context SC01–SC10 (conformidade ou waiver).
8. Prova de afinidade `preserve` + single-flight refresh.
9. Runbooks: onboarding/remoção de conta, mudança de model-map, incidente, backup/restore, upgrade, rollback; donos nomeados + escalonamento.

## 6. Status operacional atual (fora do escopo do build, já aplicado)

- Claude Code global reapontado para OmniRoute + combo `claude_code_kimi_2.7_Code`.
- 4 contas `agy` (Antigravity) em round-robin estrito, provado 4×200 em `agy/claude-opus-4-6-thinking`.
- Kimi/GLM: combos `kimi-sub` (priority) → `cline-kimi-k2.7-dedicated` (4 subs, round-robin, failover) → `group2-nvidia-glm5.2-fallback` (3 subs). Rotação de Kimi só em falha crítica (session-sticky + `failoverBeforeRetry`).
- `REQUIRE_API_KEY=true` ativo; exposição LAN protegida por chave + firewall de subnet.

## 7. Registro de decisão (assinável)

> Preencher e assinar. Este bloco é o registro oficial das decisões relacionadas às
> respostas das Seções 1–6. Enquanto o item 1 estiver "AGUARDAR", nenhuma implementação
> é iniciada (o caminho de produção atual permanece intocado).

### 7.1 Autorização de implementação (topologia de 4 agentes)

- [x] **AUTORIZADO** — iniciar Wave 0 (congelar contratos, ownership de arquivos e IDs de aceite), limitado ao canary de 20 tarefas, com os portões da Seção 1.
- [ ] **AGUARDAR** — permanecer em revisão; não iniciar implementação.

Escopo autorizado (quando AUTORIZADO):
- Wave 0: contratos neutros, fachada de compatibilidade, ownership de arquivos.
- Wave 1 (paralela): Codex 1 integrador-líder · Codex 2 gateway · Codex 3 runtime/segurança CLI · Codex 4 operações/paridade.
- Wave 2: fiação via daemon (hotspot único do integrador).
- Wave 3: canaries de protocolo/modelo + aceite de 20 tarefas.
- **Fora de escopo até novo aceite:** remoção do Prodex, tiers 50/100, cutover default gateway-required.

### 7.2 Nome provisório
- [x] **Agent Brain** (nome neutro provisório; definitivo só na etapa de debrand — passo 10 da migração).

### 7.3 Capacidade de lançamento
- [x] **20 tarefas** como primeiro tier de produção.
- [ ] 50 tarefas — liberar apenas com relatório de carga aprovado.
- [ ] 100 tarefas — liberar apenas com relatório de carga aprovado + decisão de estado (single-node vs. compartilhado) resolvida.

### 7.4 Portões obrigatórios antes do cutover total (todos precisam de evidência ou waiver)
- [ ] Conformidade de protocolo por modelo (Anthropic Messages, OpenAI Responses, Chat Completions).
- [ ] Failure-injection (expiry, revoked, quota, 429 account/global, 5xx, timeout, stream quebrado, cancelamento, restart).
- [ ] Single-flight token refresh comprovado.
- [ ] Readiness fail-closed comprovado.
- [ ] Smart Context SC01–SC10 (shadow→canary→fallback exato→self-check) OU waiver assinado de produto+segurança.
- [ ] Prova de afinidade `preserve` (continuação fixa na conta de origem).
- [ ] Decisão de estado: single-instance (com drain/rollback) OU backend compartilhado + reverificação de rotação.
- [ ] Digest da imagem OmniRoute fixado (substituir `:latest`).

### 7.5 Assinaturas
- Arquiteto OmniRoute: __________________________  Data: ____________
- Dono do produto / integração Agent Brain: __________________________  Data: ____________
- Segurança (para waivers, se houver): __________________________  Data: ____________

### 7.6 Histórico
- 2026-07-17 — Resposta do arquiteto registrada (Seções 0–6). Bloco de decisão adicionado. Aguardando marcação do item 7.1.
- 2026-07-17 — Dono autorizou início da implementação e delegação integral pelo TL Claude/GLM-5.2. Item 7.1 marcado `AUTORIZADO`; escopo inicial permanece limitado às Waves 0–3 e ao tier 20.
