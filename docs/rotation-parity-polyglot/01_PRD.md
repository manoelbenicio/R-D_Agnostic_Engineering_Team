> **PLANO CORRIGIDO (2026-07-04) — leia antes de agir.** O plano anterior assumia o binario prodex instalado (ERRO). Correcoes herdadas:
> - **P0 FUNDACAO e pre-requisito de TUDO**: provisionar/buildar o binario prodex (source -> ~/runtime/prodex-src; instalar Rust; `cargo build --release`; verificar pin v0.246.0/7750da9b + hash; setar `MULTICA_PRODEX_ENABLED/PATH/VERSION/COMMIT` + `PRODEX_HOME`). Ref: Diligencias/00_FUNDACAO_P0.md + openspec specs/prodex-runtime-provisioning.
> - **Kill-switch + rollback: TESTADOS** (nao so documentados) antes do deploy.
> - **QA EXAUSTIVO em container ANTES do deploy** (NUNCA bypassado); deploy direto em PROD depois.
> - **OpenCode: ARQUIVADO** (sucessor Crush) -> disabled/descope (decisao F5).
> - **IPv6 desabilitado** nos builds: `docker run --sysctl net.ipv6.conf.all.disable_ipv6=1 ...`.
> Fonte de verdade: `openspec/changes/rotation-parity-polyglot/` + `.planning/` + `Diligencias/`.

# PRD — Camada de Rotação Multi-Vendor + Otimização de Contexto (Revisão de Engenharia)

| Campo | Valor |
|-------|-------|
| **Título** | Rotation-Parity & Context-Optimization Layer — pedido de avaliação independente |
| **Autor** | Orquestração / Arquitetura de Soluções (Opus 4.8) |
| **Revisor solicitado** | Codex R&D Engineering Team |
| **Data** | 2026-07-04 |
| **Status** | **DECIDIDO (2026-07-04)** — feedback do R&D incorporado |
| **Objetivo** | Colher a leitura independente do time sobre linguagem (Go × Rust) e arquitetura, e receber **3 cenários** que maximizem eficiência + performance mantendo **todas as features do prodex** |

> **DECISÃO TOMADA (registrada em `docs/rotation-parity-polyglot/02_ADR-001-arquitetura.md`):** **agora** — deploy do **`prodex` AS-IS pinado, direto em PROD**, com o **Multica Go (L4)** orquestrando, para obter imediatamente todas as features (rotação, token-saver/Smart Context, modos, reset-claim). **Alvo (próximo marco)** — arquitetura **polyglot** (Go L4 control plane + Rust L2 endurecido via fork-prodex), com **invariante de roteador único por sessão**.
>
> **Parâmetros:** sem fase de teste dedicada (ajusta-se em PROD — decisão do dono para otimizar tempo); guarda-corpos = knobs nativos do prodex (shadow/canary) + kill switch + logs scrubbed. **Vendors:** Codex/Kiro/Antigravity/Cline/OpenCode (**Kimchi removido**). **Reset-claim:** prioridade baixa (frio/aleatório), mas será feito. **State:** Postgres. **ToS jurídico:** não-aplicável (sem Claude Code; Opus via Kiro/AWS). Consequência aceita: reverte "tudo em Go" no quente e **aposenta como runtime** o rotation-router Go (vira control plane).

> **Como este documento foi usado:** a coluna **"Comentários Opus 4.8"** trouxe a análise interna (viés declarado). A coluna **"Feedback Codex R&D Engineering Team"** recebeu os vereditos do time (ver docs de parecer). Este PRD agora é **registro histórico** da decisão.

---

## 1. TL;DR — o que pedimos ao time

| # | Comentários Opus 4.8 | Feedback Codex R&D Engineering Team |
|---|----------------------|-------------------------------------|
| 1 | Leiam o gap (§3), a solução encontrada (§4) e o que já existe (§5). | |
| 2 | Avaliem a análise Go × Rust para o conjunto **completo** de features (§7) e as opções de implementação (§8). | |
| 3 | Devolvam **(a)** o entendimento de vocês do problema e **(b)** os **3 melhores cenários** (arquitetura + linguagem + reuso do prodex) que deem ótima eficiência e performance **e** permitam usufruir de **todas as features do prodex**. | |

---

## 2. Contexto do produto

**Multica** é uma plataforma de orquestração de agentes de IA (camada L4) que despacha trabalho para CLIs de vendors (Codex, Kiro, Antigravity; futuros Cline/OpenCode/Kimchi). O **moat** do produto é rotacionar **contas de assinatura** (flat-rate) *antes* do esgotamento de quota, dando execução sem interrupção — **sem** pagar por token (metered).

```
 L4  MULTICA (orquestração: tasks/workspaces/dispatch)   ← produto (Go), intacto
 L3  Agente CLI (codex/kiro/antigravity/…)               ← roda o loop agêntico
 L2  ROTAÇÃO DE CONTA + OTIMIZAÇÃO DE CONTEXTO            ← ESTE PRD (o gap)
 L1  Vendors / modelos                                   ← providers
```

Restrição de negócio inegociável: **não** assinar/pagar um gateway metered de terceiro. Usamos **nossas próprias assinaturas** via perfis isolados. Soluções open-source self-hosted são aceitáveis.

| Comentários Opus 4.8 | Feedback Codex R&D Engineering Team |
|----------------------|-------------------------------------|
| O contexto e as 4 camadas estão corretos? A camada L2 é o escopo deste PRD; Multica (L4) permanece intacto em Go. | |

---

## 3. Gap atual

| # | Gap | Comentários Opus 4.8 (situação hoje) | Feedback Codex R&D Engineering Team |
|---|-----|--------------------------------------|-------------------------------------|
| G1 | Rotação "ingênua" (drena conta 1 até esgotar) | parcialmente resolvido (temos rotação proativa + policies) | |
| G2 | **Reset-claim headless** (resgatar crédito de reset antes de rotacionar) | **não resolvido** — estava *gated* (no-op); `/usage` do Codex é TUI-only | |
| G3 | **Otimização de contexto / token-saver** (reduzir custo de token por request) | **não construído — elevado a MANDATÓRIO** (versão completa, paridade prodex) | |
| G4 | Cobertura além de Codex/Kiro/Antigravity (Cline/OpenCode/Kimchi) | não construído (Fase 3) | |
| G5 | Governança multi-tenant (approved-accounts, SCIM, virtual keys) | parcial | |

---

## 4. A solução encontrada — `prodex`

`prodex` (npm `@christiandoxa/prodex`) é um wrapper multi-conta/multi-provider do Codex com roteamento consciente de quota.

**Fatos verificados (fonte primária — npm + repo GitHub, 2026-07-04):**

| Atributo | Valor verificado |
|----------|------------------|
| Licença | **Apache-2.0** (permite fork/modificação/rebrand; exige manter atribuição) |
| Linguagem | **Rust 91,9%** + JS 7,8% (workspace Cargo) |
| Atividade | 1.425 commits, 244 releases, v0.246 (03/jul/2026) — release ~diária |
| Adoção | 31 stars, 3 forks, **0 issues abertas**, 5.149 downloads npm/semana |
| Mantenedor | **1** (bus-factor 1) |
| Higiene supply-chain | `.gitleaks.toml`, `deny.toml` (cargo-deny), CI presente |

**Features (confirmadas em docs/repo; eficácia em produção NÃO testada por nós):** multi-provider (Codex/Gemini/Antigravity/Anthropic/Copilot/Kiro/DeepSeek/local/Bedrock); auto-rotação só pré-commit com afinidade de sessão; **reset-claim** (`prodex redeem` + `--auto-redeem`, guarda por janela); import Kiro de `~/.local/share/kiro-cli/data.sqlite3`; **Smart Context / token-saver** (caminho quente, com canary/shadow/replay/self-check de integridade); gateway OpenAI-compat (policies fallback/RR/least-busy/lowest-cost/lowest-latency/rpm/tpm, ledger/spend, Prometheus, SCIM/tenant scopes, virtual keys, Presidio); MCP read-only (`prodex-inspect`); modos (Caveman/Super).

| Comentários Opus 4.8 | Feedback Codex R&D Engineering Team |
|----------------------|-------------------------------------|
| Também avaliamos **codex-multi-auth** (MIT): eliminado por ser **Codex-only** (1 de ~6 vendors), sem reset-claim, uso pessoal/não multi-tenant. | |
| Risco central: **bus-factor 1** + churn diário + reset-claim **não validado empiricamente** por nós. Mitigável via pin + vendor + fork (Apache-2.0). Concordam? | |

---

## 5. O que JÁ construímos (ponto de partida — em Go, validado)

Módulo `github.com/multica-ai/multica`, work tree `multica-auth-work/server`.

- **Isolamento por conta** via env nativo (`CODEX_HOME` / `XDG_DATA_HOME` kiro / `HOME` antigravity).
- **Rotação core**: `detector`, `service`/`pool` (state machine), `proactive` (banner+ledger), `store_pg` (Postgres), integração no daemon, `usage`, `warnbanner`, `probe_codex`, `credential_metrics`, `token_lifecycle`.
- **rotation-router** Wave 1/2: `policy.go`, `fallback.go` (retry/backoff/jitter/classify), `loadbalance.go` (afinidade via xxhash), `proactive_reset.go` (**reset-claim GATED = no-op**), migration `124_approved_accounts`, dashboards + KPI "Savings".
- **89 testes verdes**, daemon sem regressão, migrations 123/124 aplicam/revertem.
- **Prova empírica real**: 3 contas Codex distintas coexistindo via `CODEX_HOME`; rotação proativa real em staging (`rotation_events` 001→002); stack de observability de pé.

| Comentários Opus 4.8 | Feedback Codex R&D Engineering Team |
|----------------------|-------------------------------------|
| Este código é quase todo **caminho frio** (decisões por sessão/ocasionais), não por-token → é justo a parte que **não** exige Rust. ~60-70% do subset de rotação já está pronto e validado. | |

---

## 6. Requisitos e restrições

| Tipo | Item | Feedback Codex R&D Engineering Team |
|------|------|-------------------------------------|
| Mandatório | Rotação multi-vendor pré-exaustão + afinidade de sessão | |
| Mandatório | **Reset-claim headless funcionando de verdade** (fechar G2) | |
| **Mandatório (elevado)** | **Token-saver / Smart Context COMPLETO** (paridade prodex) — caminho quente | |
| Mandatório | Zero custo metered de terceiro (usar assinaturas próprias) | |
| Mandatório | Multica (L4) permanece **Go**; a camada nova integra a ele | |
| Desejável | Gateway OpenAI-compat, governança multi-tenant, MCP, modos, guardrails | |
| Disciplina | Sign-in/out em disco por agente; propriedade de arquivo disjunta; verde-em-container antes de DONE; caminhos absolutos; sem segredo em log; nada inventado | |

---

## 7. Análise técnica — Go × Rust para o conjunto COMPLETO

Classificação: **quente** = roda por request/token (sensível a performance/segurança de memória); **fria** = por sessão/ocasional.

**Dados de referência (fit por feature):**

| Feature | Caminho | Fit Go | Fit Rust |
|---------|---------|--------|----------|
| Rotação multi-vendor | frio | excelente | bom |
| Isolamento de credencial (env) | frio | excelente | bom |
| Detecção/forecast de quota | frio | bom | bom |
| Rotação proativa + afinidade | frio | bom | bom |
| Reset-claim headless | frio | bom | bom |
| Policies (fallback/LB/latency/cost) | frio | bom | bom |
| Observability/spend/savings | frio | excelente (Prometheus Go-nativo) | ok |
| Multi-tenant/SCIM/approved-accounts | frio | excelente | ok |
| **Token-saver / Smart Context** | **QUENTE** | dá, mas GC/alloc + risco de data-race no mux de stream | **ideal** (zero-copy, sem GC, concorrência segura) |
| Modos (Caveman/Super) | frio | bom | bom |
| MCP server | frio | bom | bom |
| **Gateway OpenAI-compat (proxy)** | **QUENTE** | viável, cauda de latência/memória pior sob carga | **ideal** (proxy alto-throughput) |
| Guardrails/PII (Presidio externo via HTTP) | quente-ish | ok | ok |
| Integração com Multica (L4) | frio | ganha forte (mesma linguagem) | fronteira FFI/sidecar |

**Prós/contras (opinião — desafiem):**

| Tópico | Comentários Opus 4.8 | Feedback Codex R&D Engineering Team |
|--------|----------------------|-------------------------------------|
| Go — prós | vence a maioria fria; mesma linguagem do Multica (1 toolchain/CI); compila em segundos (iteração rápida com fleet); agentes geram Go correto com menos falha; ecossistema cloud-native/observability nativo; concorrência simples p/ subprocessos | |
| Go — contras | Smart Context/gateway (quente) tecnicamente inferior (jitter de GC/alloc); cauda pior sob carga; sem borrow-checker → risco de data-race é nosso; sem reuso direto do blueprint (prodex é Rust) | |
| Rust — prós | domina o caminho quente (latência p99 previsível); segurança de memória/concorrência em compile-time (crítico p/ proxy que multiplexa streams+segredos); reuso direto do blueprint prodex; footprint menor | |
| Rust — contras | perde na maioria fria (grosso do trabalho); 2ª linguagem ao lado do Multica Go (fronteira, 2 toolchains/CIs); compilação lenta (fleet itera devagar); curva íngreme (borrow/lifetimes/async → mais falha de compile dos agentes); pool de contratação menor | |
| Observação central | com o conjunto **completo**, o quente (token-saver/gateway) é justo onde Rust ganha e é a peça mais complexa/arriscada de reconstruir — motivo pelo qual o prodex é Rust. O frio favorece Go. Nenhuma é perfeita p/ o conjunto inteiro. | |

---

## 8. Opções de implementação

| Opção | Comentários Opus 4.8 (prós / contras) | Feedback Codex R&D Engineering Team |
|-------|----------------------------------------|-------------------------------------|
| **A — Deploy prodex as-is** (sidecar Rust; Multica orquestra `prodex`) | **+** todas as features já; linguagem certa p/ o quente; mantido com release diária; fecha G2/G3; ETA ~dias. **−** dependência de projeto solo (bus-factor 1); nossa rotação Go vira redundante; menos propriedade; branding externo. | |
| **B — Reimplementar tudo em Go** (inclui Smart Context completo) | **+** propriedade total; 1 linguagem; integra direto no Multica; reusa ~60-70% já verdes. **−** Smart Context (quente) é a peça mais difícil/arriscada, na linguagem pior p/ isso; risco de corromper continuação/tool-call; paridade em meses; viramos mantenedor solo perseguindo quota volátil do Codex. | |
| **C — Fork do prodex** (Apache-2.0), rebrand, melhorar por cima | **+** baseline funcional imediato + propriedade (fork); de-branding legal (mantendo atribuição); ETA ~2 dias. **−** é Rust → fronteira com Multica Go; herdamos a complexidade; ainda perseguimos upstream. | |
| **D — Polyglot** (Go frio + Rust/fork-prodex quente como sidecar) | **+** cada linguagem no terreno forte; preserva o Go validado; ganha o quente na linguagem certa. **−** fronteira/contrato entre serviços (IPC/HTTP); dois "cérebros" de rotação a reconciliar; complexidade operacional. | |
| **E — Migrar o Go existente para Rust** | **+** unifica em Rust se o quente dominar a estratégia. **−** o Go existente é **frio** (não pede Rust) → custo sem ganho; **zera a validação empírica** já conquistada; risco de re-introduzir bugs; ROI negativo no curto prazo. | |

```
   RESUMO VISUAL (eixo: propriedade × esforço × fit-do-quente)
   as-is (A) ──── fork (C) ──── polyglot (D) ──── Go-tudo (B) ──── migrar (E)
   ↑ menos esforço / menos propriedade            ↑ mais esforço / mais propriedade
   fit-quente: A/C/D altos (Rust) · B baixo (Go) · E alto mas com custo de migração
```

---

## 9. Perguntas explícitas ao time de engenharia

| # | Pergunta (Comentários Opus 4.8) | Feedback Codex R&D Engineering Team |
|---|---------------------------------|-------------------------------------|
| 1 | **Entendimento:** o recorte do gap (§3), a classificação quente/frio (§7) e os fatos do prodex (§4) estão corretos? O que está errado, exagerado ou faltando? | |
| 2 | **Smart Context em Go:** qual o risco/esforço **real** de implementar o token-saver *completo* (com integridade de continuação/tool-call) em Go? Existe um subconjunto seguro com ~80% da economia e ~20% do risco? | |
| 3 | **Reset-claim:** melhor forma de confirmar/implementar o resgate de crédito headless (endpoint/protocolo), dado que `/usage` do Codex é TUI-only? | |
| 4 | **Fronteira polyglot:** se Go(frio)+Rust(quente), qual contrato de fronteira recomendam (sidecar HTTP local? IPC? binário Rust chamado pelo daemon Go?) e como evitar dois cérebros de rotação? | |
| 5 | **Os 3 cenários:** entreguem os **3 melhores cenários** (arquitetura + linguagem + reuso do prodex) com ótima eficiência e performance **mantendo todas as features do prodex**, com trade-offs e ETA aproximado de cada. | |

### Espaço para os 3 cenários do time

| Cenário | Descrição (arquitetura + linguagem + reuso prodex) | Trade-offs | ETA | Recomendado? |
|---------|-----------------------------------------------------|-----------|-----|--------------|
| 1 | | | | |
| 2 | | | | |
| 3 | | | | |

---

## Anexo A — Fonte e disciplina
- Fatos do prodex: npm + repo GitHub `christiandoxa/prodex` (Apache-2.0), acesso 2026-07-04.
- Nada neste documento foi inventado; comportamento de vendor a partir de docs/repo primários; eficácia de reset-claim marcada como **não validada empiricamente**.
- Regras de execução (se avançarmos): sign-in/out por agente em disco, propriedade de arquivo disjunta, verde-em-container antes de DONE, caminhos absolutos, sem segredo em log.

## Anexo B — Estado validado (evidência)
- 3 contas Codex reais coexistindo via `CODEX_HOME`; rotação proativa real (`rotation_events` 001→002); 89 testes de rotação verdes; migrations 123/124; stack de observability de pé.