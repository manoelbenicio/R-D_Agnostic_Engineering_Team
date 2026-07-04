# PROJECT — Multica (Rotation-Parity Polyglot)

> Documento de fundação do projeto. Criado no milestone v2.0 (GSD new-milestone, 2026-07-04).
> Fonte de verdade para escopo, arquitetura, decisões e estado verificado.

## 1. O que é

Plataforma multi-vendor de rotação de contas para AI-CLIs (Codex, Kiro, Antigravity, Cline, OpenCode).
Arquitetura **polyglot** (ADR-001):

```
  L4  Multica (Go)  — CONTROL PLANE (frio)     |   L2  prodex (Rust) — RUNTIME PLANE (quente)
  cadastro, policy, approved-accounts,          |   request em voo: afinidade, fallback,
  budgets, kill-switch, observability           |   Smart Context/token-saver, reset-claim
```

- **Agora:** Multica Go orquestra o **prodex AS-IS** (pinado v0.246.0 / commit `7750da9b`), direto em PROD.
- **Alvo (marco futuro):** endurecer via **fork** do prodex (Rust L2 sidecar) com contrato local estrito.

## 2. Estado REAL verificado (2026-07-04, via inspeção das repos)

| Item | Estado | Evidência |
|---|---|---|
| Multica server (Go) | ✅ presente, `module github.com/multica-ai/multica/server`, Go 1.26.1 | repo `multica-auth-work`, HEAD 52cdd87, 91 uncommitted |
| Integração Go↔prodex | ✅ código existe | `server/internal/daemon/prodex.go` + `_test.go`, `internal/l2runtime/` |
| Isolamento por conta (produto) | ✅ intacto no fonte | `execenv/` (codex_home/antigravity_home/kiro_home), CODEX_HOME por tarefa + copia auth.json por conta |
| prodex SOURCE | ✅ clonado no commit certo | `/tmp/prodex-audit-7750da9` (7750da9b), workspace Cargo `name=prodex` |
| prodex BINÁRIO | ❌ **não buildado** | `target/release` ausente; Rust/cargo ausente |
| Postgres / Redis | ✅ rodando (docker) | `deploy-postgres-1` pg17 healthy :5432 · `deploy-redis-1` :6379 |
| docker | ✅ v29.6.0 | builds/QA via container |
| Go/Rust/Node nativo | ⚠️ fora do PATH | builds via docker |
| Contrato de launch | ⚙️ via env | `MULTICA_PRODEX_ENABLED/PATH/VERSION/COMMIT`, `PRODEX_HOME`, kill-switch default ON |

## 3. Milestone atual — v2.0 "Fundação + Deploy Correto"

**Goal:** Levar o prodex AS-IS a PROD com **fundação real** (buildar/pinar o binário), **QA exaustivo sem bypass**, e o control plane Go orquestrando o L2 — **tudo documentado e rastreável**, corrigindo os furos do plano anterior (OpenSpec `rotation-parity-polyglot`).

**Por que este milestone existe:** o plano anterior assumia o binário prodex instalado ("instalação verificada") sem tarefa de provisioná-lo, não registrava tasks rastreáveis, e tinha nós circulares (gates F0-gated). Este milestone corrige a fundação.

## 4. Decisões travadas (ADR-001 + sessão)
1. prodex **AS-IS** em PROD agora; fork/polyglot no próximo marco.
2. **Um roteador por sessão** (Go desired-state; Rust runtime).
3. **Postgres** para estado compartilhado (SQLite proibido).
4. Vendors: Codex/Kiro/Antigravity/Cline/OpenCode; **Kimchi fora**. (OpenCode a reavaliar — arquivado, sucessor Crush.)
5. Reset-claim = **baixa prioridade**, por último (eficácia empírica não verificada).
6. **Sem staging dedicado** — deploy direto em PROD, mitigado por kill-switch + rollback **testados** + QA exaustivo em container ANTES.
7. **Isolamento de conta é do prodex/produto** — não construir tooling paralelo (FLM descartado).

## 5. Invariantes inegociáveis
- Roteador único por sessão; hard affinity (`previous_response_id`/turn/session); **rotate-before-commit** (nunca mid-stream).
- Troca de perfil **fail-closed**. Smart Context com **fallback exato** quando integridade estrutural é afetada.
- **Sem segredo** em log/trace/evidência. Postgres (sem SQLite compartilhado); migrations reversíveis.
- Verde **em container** com evidência antes de DONE (não confiar no tail). QA **nunca** bypassado.

## 6. Riscos abertos
- Eficácia real de reset-claim e Smart Context sob carga PROD (validar empírico + evidência scrubbed).
- prodex bus-factor 1; drift do Codex upstream.
- Deploy direto sem staging → depende de kill-switch/rollback provados.
