# STATE — Milestone v2.0 (Rotation-Parity Polyglot)

> Estado vivo do milestone. Atualizado a cada avanço de fase.

## Posição atual
- **Milestone:** v2.0 "Fundação + Deploy Correto" — planejamento **CONCLUÍDO** (PROJECT/REQUIREMENTS/ROADMAP criados 2026-07-04).
- **Próxima fase:** **P0 — Fundação** (bloqueia tudo). Ainda não iniciada.

## Blocker crítico (raiz)
- **prodex BINÁRIO não existe** — source presente (`/tmp/prodex-audit-7750da9` @7750da9b) mas **não buildado** (Rust/cargo ausente). → P0/REQ-01. **Nada de deploy até isso.**

## Já pronto (verificado, reaproveitável)
- Multica Go server + integração `prodex.go`/`l2runtime` (código existe).
- Isolamento por conta no produto (execenv) — intacto.
- Postgres/Redis (docker) up. docker v29.
- Contrato/vendor-matrix/redaction-audit produzidos em sessão anterior (revalidar como evidência sob os novos REQs).

## Pendências de processo
- `rotation-parity-polyglot` (OpenSpec) tinha 0 tasks e furos → substituído por este planejamento GSD documentado.
- Arquivar `rotation-router` (SUPERSEDED). [REQ-24]
- Reconciliar "deploy direto × QA exaustivo". [REQ-25]

## Correções de rota registradas (aprendizado desta sessão)
- "1 conta Codex" foi erro (contas colapsadas por clobber de `~/.codex`; homes isolados resolveram).
- FLM descartado (redundante com prodex).
- Plano anterior assumia binário instalado — corrigido com P0 Fundação.

## Ambiente do fleet (harness)
- Workers Codex isolados por `CODEX_HOME` (~/.codex-a/b/c/d), 4 contas distintas — resolvido; isolamento é responsabilidade do prodex/produto, não de tooling paralelo.

## Próximo passo
Iniciar **P0 (Fundação)** — provisionar/buildar o binário prodex e confirmar ambiente. Só então P1→P7.
