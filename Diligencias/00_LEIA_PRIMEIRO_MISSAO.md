# 🌍 LEIA PRIMEIRO — Missão & Mundo do Projeto (OBRIGATÓRIO p/ TODOS)

> **Todo agente, o Tech-Lead (Opus 4.8) e o dono leem este documento ANTES de qualquer tarefa.**
> Ele explica o TODO: o que é, por que, como, quando, onde, quem — e as regras do mundo em que
> você vive. Sem entender o todo, você não deve tocar em nada. (Aqui é onde tem água; e não, você
> não pode comer uma bicicleta.)

---

## O QUÊ (What)
Projeto **Rotation-Parity Polyglot**: fazer o **Multica** (plataforma open-source de *managed agents*)
lançar o **prodex** (runtime Rust) no lugar do `codex` cru, para ganhar o **caminho quente**:
rotação de contas pré-commit, afinidade de sessão, **Smart Context/token-saver** (mandatório),
reset-claim e cobertura multi-vendor. Detalhe do produto: `00_CONTEXTO_MULTICA.md`.

## POR QUÊ (Why)
A rotação em Go cobre só o caminho frio. O produto **exige** features de caminho quente (Smart Context
etc.). Reconstruir isso em Go é inferior (risco de protocolo + cauda p95) e lento. O **prodex já entrega**
tudo, na linguagem certa (Rust), mantido ativamente. Decisão do dono (ADR-001): **usar prodex AS-IS
em PROD agora**, endurecer via **fork** depois. Objetivo: features imediatas, sem custo metered.

## COMO (How)
- **Arquitetura polyglot:** L4 **Multica Go** (control plane frio: policy, contas, kill-switch, observability)
  + L2 **prodex Rust** (runtime quente: request em voo). Contrato local `rpp.l2.v1`. **Um roteador por sessão.**
- **AS-IS agora → fork depois.** prodex pinado (v0.246.0 / commit `7750da9b`).
- **Builds em container** (Go: `golang:1.26-alpine`; Rust: `rust:1.85-bookworm`), **IPv6 desabilitado**.
- **QA EXAUSTIVO em container ANTES do deploy** (nunca bypassado); **deploy direto em PROD** só depois,
  atrás de **kill-switch + rollback TESTADOS** + logs scrubbed.

## QUANDO (When) — sequência (não pule fases)
`P0 Fundação (BLOQUEIA TUDO)` → `P1 Contrato` → `P2 Fork-map` / `P4 State` / `P5 Vendors` → `P3 Integração`
→ `P6 QA EXAUSTIVO` → `P7 Deploy`. `P9 Reset-claim` = por último (não bloqueia).
Milestone atual: **v2.0 "Fundação + Deploy Correto"**. Ver `ROADMAP.md` (grafo de dependências).

## ONDE (Where)
- **Multica (produto Go):** repo `multica-ai/multica` → `multica-auth-work/server` (Go 1.26.1).
- **prodex (source):** `github.com/christiandoxa/prodex` @ `7750da9b` → estabilizar em `~/runtime/prodex-src`.
- **Datastores:** Postgres (`pgvector/pgvector:pg17`, :5432) + Redis (:6379), via docker.
- **Board/rastreabilidade:** `.deploy-control/` (check-ins) + `.deploy-control/evidence/`.
- **Planejamento (fonte de verdade):** `openspec/changes/rotation-parity-polyglot/` + `.planning/` + `Diligencias/`.
- **Harness de execução:** Herdr (agentes rodam em panes; só operar Herdr se `HERDR_ENV=1`).

## QUEM (Who)
- **Dono:** decide F5 (vendor sign-off) e F7 (deploy). Só é acionado se grave ou decisão dele.
- **Tech-Lead / POC:** **Opus 4.8** (orquestração/planejamento/validação — NÃO escreve código de produto).
- **Orquestrador (TL operacional):** `opus-4.8-orchestrator` (dirige o fleet, 90–120s).
- **Roster (por fase):** Codex#5.5#A (contrato), #B (fork-map/reset), #C (Fundação P0 + integração), #D (DevOps/deploy),
  GLM#52#A (QA), GLM#52#B (state/security), Gemini#Pro (vendor matrix), Gemini#Flash35 (ops).

## AS REGRAS DO MUNDO (inegociáveis — "não se come bicicleta")
1. **SIGN-IN/OUT em disco** antes/depois de tocar em arquivo (`.deploy-control/<AGENT>__<STREAM>__<UTC>.md`, caminho absoluto).
2. **Propriedade de arquivo disjunta;** hotspots (daemon) = dono único serial.
3. **Verde-em-container com evidência ANTES de DONE.** Não confie no tail; o validador re-roda.
4. **Nada inventado** — só fonte primária. Nunca invente flag do prodex/Herdr/comando.
5. **Sem segredo** em log/trace/evidência/check-in. **SQLite proibido** p/ estado compartilhado (use Postgres).
6. **Invariantes runtime:** roteador único/sessão, hard affinity, rotate-before-commit, troca de perfil fail-closed.
7. **Segurança prodex:** Caveman/hook **DESABILITADO** por padrão (RCE); `ALLOW_UNSAFE_CHILD_ENV=off`.
8. **QA nunca bypassado.** Deploy só com kill-switch + rollback testados.
9. **Se travar ou for ambíguo: PARE e escale ao Opus** — não improvise, não decida sozinho.

## MAPA DE LEITURA (nesta ordem)
1. **Este documento** (missão & mundo). 2. `00_CONTEXTO_MULTICA.md` (produto/repo).
3. `00b_DEPENDENCY_SOURCES.md` (de onde baixar). 4. `00c_PRODEX_CRATE_COVERAGE.md` (44 crates × plano).
5. `00d_CONFIG_ENV_SECURITY.md` (env/segurança/Caveman). 6. Sua fase: `Diligencias/0X_*.md`.
7. Autoridade: `openspec/changes/rotation-parity-polyglot/` (proposal/design/tasks/specs) + ADR-001.

## DEFINITION OF DONE (do milestone)
Todos os REQ verdes com evidência; prodex AS-IS rodando em PROD via Multica; kill-switch/rollback provados;
QA exaustivo verde; Caveman travado; docs/board reconciliados.
