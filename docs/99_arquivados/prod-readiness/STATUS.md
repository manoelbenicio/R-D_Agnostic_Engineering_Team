# STATUS — Fonte única de verdade (Opus 4.8, orquestrador)

Atualizado: 2026-07-01T20:13Z. Reflete o que foi VERIFICADO por mim no
container (`golang:1.26-alpine`), não o que os check-ins afirmam. Cada DONE
abaixo eu re-rodei ou li o artefato em disco — não confiado no tail.

## Fase 1 — Isolamento de credencial por vendor
| Stream | Agente | Status | Verificação Opus |
|--------|--------|--------|------------------|
| W-INT-contract | Opus 4.8 | ✅ DONE | contrato de env publicado |
| W-VENDORS (kiro_home/antigravity_home) | CODEX-1 | ✅ DONE | pacote execenv verde |
| W-METRICS (credential_metrics) | CODEX-2 | ✅ DONE | pacote metrics verde |
| W-OBS (observability stack) | GLM-52 | ✅ DONE | stack revisada |
| W-VERIFY (não-regressão) | CODEX | ✅ DONE | ver W-INTEGRATE baseline |

## Fase 2 — Rotação automática de conta
| Stream | Agente | Arquivos (disco real) | Status | Verificação Opus |
|--------|--------|-----------------------|--------|------------------|
| W-ROT-contract | Opus 4.8 | contract.go | ✅ DONE | build+vet verde |
| W-DETECT (reativo) | CODEX-1 | detector.go | ✅ DONE | testes verdes |
| W-ROTATE (state machine) | CODEX-2 | service.go, pool.go | ✅ DONE | testes verdes |
| W-PGSTORE | **CODEX-2 (redo)** | store_pg.go, migration 123 (`accounts`/`credentials`/`assignments`/`rotation_events`) | ✅ DONE | consumido pelo E2E verde |
| W-PROACTIVE (ledger) | CODEX-2 | proactive.go | ✅ DONE | testes verdes |
| W-WARNBANNER (Codex passivo) | CODEX-2 | warnbanner.go | ✅ DONE | `-run Warn` verde |
| W-USAGE (4 vendors) | CODEX-2 | usage.go | ✅ DONE | `-run Usage` verde |
| W-INTEGRATE (fiação reativa no daemon) | CODEX | daemon.go, auth_authenticator.go | ✅ DONE | baseline daemon verde (só symlink root-only falha, ambiental) |
| W-E2E (Postgres real) | CODEX-1 | rotation_e2e_test.go | ✅ DONE | **re-rodei vs Postgres 17 → PASS** |

## PENDENTE — o que falta para "zero interrupção" de verdade
| Stream | Agente | Escopo | Status |
|--------|--------|--------|--------|
| **W-PROACTIVE-INT** | **CODEX (a disparar)** | Gatilho PROATIVO no daemon: banner passivo (Codex) via MessageText + ledger entre tasks; reusa OnExhaustion(ReasonQuotaProactive). Lock EXCLUSIVO daemon.go. | ⏳ PROMPT PRONTO (PROMPT_CODEX_W-PROACTIVE-INT.md) |
| W-PROBE (futuro) | a definir | Probe ativo `/usage` p/ Kiro/Antigravity entre turnos. Fora do v1 (decisão de arquitetura). | 🔮 BACKLOG |

## Reconciliações de higiene (2026-07-01T20:13Z)
- GLM52__W-PGSTORE → **SUPERSEDED** por CODEX-2 (mesmos arquivos, redo posterior).
  Nomes `rotation_*` daquele check-in NÃO existem em disco; nomes reais são sem prefixo.
- CODEX__W-INTEGRATE → tail com FAIL é AMBIENTAL (imagem sem git + root), não regressão.

## Gate de validação OFICIAL (confirmado com Codex, reproduzido pelo Opus → verde)
O pacote `daemon` exige `git` + identidade git; o container roda como root, então o
subteste symlink→/root é fail AMBIENTAL conhecido e é pulado com `-skip`.
Comando canônico (fonte: Codex; Opus reproduziu 2026-07-01T20:24Z → REAL_EXIT=0,
daemon/execenv/repocache/metrics todos `ok`):
```
docker run --rm -v "$PWD":/src -w /src/server golang:1.26-alpine sh -c '
  apk add --no-cache git >/dev/null 2>&1 &&
  git config --global user.email t@t &&
  git config --global user.name t &&
  go build ./... &&
  go vet ./internal/daemon/... ./internal/metrics/... &&
  go test ./internal/daemon/... ./internal/metrics/... \
    -skip "TestValidateLocalPath/rejects_a_symlink_pointing_at_the_user_home"
'
```
CAVEAT (Codex): NÃO usar `go test ... | tail` sem pipefail — o pipe mascara o
exit code. Se precisar do tail: `sh -o pipefail -c '... 2>&1 | tail -40'`, ou
não usar pipe no gate real.

E2E de rotação (Postgres real; migration no repo em migrations/123_rotation.up.sql):
```
DATABASE_URL=... go test ./internal/rotation/ -run E2E -v
```