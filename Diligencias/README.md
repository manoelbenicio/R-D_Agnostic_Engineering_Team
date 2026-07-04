# Diligências — Rotation-Parity Polyglot (v2.0)

> Documentação de diligência **completa e atualizada**, da fundação até a última fase de QA (P0→P6).
> Fonte de verdade cruzada: OpenSpec (`openspec/changes/rotation-parity-polyglot/`) + GSD (`.planning/`).
> Aterrada no estado REAL verificado nas repos (2026-07-04). Regra: verde-em-container com evidência
> antes de DONE; QA NUNCA bypassado; sem segredo em log.

## Como ler
Cada fase tem um documento de diligência com: **objetivo · REQ-IDs · pré-requisitos · passos concretos ·
verificação/evidência · critério de gate (DONE) · riscos**.

## 🌍 LEIA PRIMEIRO (obrigatório p/ TODOS)
**[00_LEIA_PRIMEIRO_MISSAO.md](00_LEIA_PRIMEIRO_MISSAO.md)** — missão & mundo: o quê/por quê/como/quando/onde/quem + regras do mundo.

## Contexto do produto (leia ANTES de tudo)
**[00_CONTEXTO_MULTICA.md](00_CONTEXTO_MULTICA.md)** — o que é o Multica, o repo, e como o prodex/rotation-parity se encaixa.
- **[00b_DEPENDENCY_SOURCES.md](00b_DEPENDENCY_SOURCES.md)** — de onde baixar cada dependência (pinadas).
- **[00c_PRODEX_CRATE_COVERAGE.md](00c_PRODEX_CRATE_COVERAGE.md)** — os 44 crates do prodex × plano (cobertura verificável).
- **[00d_CONFIG_ENV_SECURITY.md](00d_CONFIG_ENV_SECURITY.md)** — ENV/config + segurança (Caveman/hook, ALLOW_UNSAFE_CHILD_ENV, subcomandos, providers).
- **[00e_COMPLETENESS_REVIEW.md](00e_COMPLETENESS_REVIEW.md)** — revisão de completude (browser/Mem0/CI/deploy + o que falta varrer).

## Índice de fases (até QA)
| # | Fase | Documento | Bloqueia |
|---|------|-----------|----------|
| P0 | **Fundação** — runtime prodex + ambiente | [00_FUNDACAO_P0.md](00_FUNDACAO_P0.md) | **tudo** |
| P0 | **Fontes de dependência** (de onde baixar) | [00b_DEPENDENCY_SOURCES.md](00b_DEPENDENCY_SOURCES.md) | P0 |
| P1 | Contrato Go↔L2 (`rpp.l2.v1`) | [01_CONTRATO_P1.md](01_CONTRATO_P1.md) | P3 |
| P2 | prodex fork-map / invariantes | [02_FORKMAP_P2.md](02_FORKMAP_P2.md) | (alvo fork) |
| P3 | Integração Go — lançar prodex | [03_INTEGRACAO_P3.md](03_INTEGRACAO_P3.md) | P6 |
| P4 | State/security (Postgres, redaction, audit) | [04_STATE_SECURITY_P4.md](04_STATE_SECURITY_P4.md) | P6 |
| P5 | Vendor capability matrix | [05_VENDOR_MATRIX_P5.md](05_VENDOR_MATRIX_P5.md) | P6 |
| P6 | **QA/conformance EXAUSTIVO** (C1–C6) | [06_QA_CONFORMANCE_P6.md](06_QA_CONFORMANCE_P6.md) | Deploy (P7) |

> P7 (Deploy) e P9 (Reset-claim) ficam fora deste conjunto — o escopo desta diligência vai **até o QA**.

## Estado REAL verificado (base de toda diligência)
| Item | Estado | Evidência |
|---|---|---|
| Multica server (Go 1.26.1) | ✅ presente | `multica-auth-work/server`, `github.com/multica-ai/multica/server` |
| Integração `prodex.go` / `l2runtime` | ✅ código existe | `server/internal/daemon/prodex.go` |
| Isolamento por conta (execenv) | ✅ intacto | codex_home/antigravity_home/kiro_home |
| prodex SOURCE | ✅ `/tmp/prodex-audit-7750da9` @`7750da9b` | workspace Cargo `name=prodex` |
| prodex BINÁRIO | ❌ não buildado | `target/release` ausente; Rust ausente |
| Postgres pg17 / Redis | ✅ docker healthy | `:5432` / `:6379` |
| docker | ✅ v29.6.0 | build/QA via container |

## Gate global de build (container)
```
docker run --rm -v "$PWD/multica-auth-work":/src -v gomodcache:/go/pkg/mod \
  -w /src/server golang:1.26-alpine sh -c \
  "apk add --no-cache git && go build ./... && go vet ./internal/... && go test ./internal/..."
```
