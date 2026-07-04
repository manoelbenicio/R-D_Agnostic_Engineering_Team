# P3 — Diligência: Integração Go (lançar prodex)

## Objetivo
Fazer o Multica Go lançar e orquestrar o prodex como runtime (sidecar), empurrando desired-state e
ingerindo eventos — **sem** rotear request em voo no Go.

## REQ-IDs
REQ-05 (lifecycle/policy/ingest/kill-switch) · REQ-06 (ingest não dispara rotação). Spec: `specs/l2-runtime-contract`.

## Pré-requisitos
- P1 verde (contrato `rpp.l2.v1`). P0 verde (binário).

## Passos
- 3.1 Lifecycle do sidecar prodex (start/stop/health) via `internal/daemon` (`prodex.go`, `l2runtime`).
- 3.2 `ApplyPolicy` + `RegisterAccounts` (Go → L2): policy/budgets/kill-switch + contas aprovadas.
- 3.3 Ingest de `RuntimeEventStream` (L2 → Go) para observabilidade/ledger. Go **não** roteia in-flight.
- 3.4 Teste: ingest de runtime-events **não** dispara rotação no Go (regressão do single-router).

## Verificação / evidência (container)
```
docker run --rm -v "$PWD/multica-auth-work":/src -v gomodcache:/go/pkg/mod \
  -w /src/server golang:1.26-alpine sh -c \
  "apk add --no-cache git && go build ./... && go vet ./internal/... && \
   go test ./internal/daemon ./internal/l2runtime"
```
- build/vet/test **verdes**; testes de rotation + L2 passam.
- Hotspot locks respeitados: `internal/daemon/l2_runtime.go`, `daemon_test.go`, `internal/l2runtime/*`.

## Critério de GATE (DONE)
✅ sidecar sobe/health OK · ✅ policy push + register accounts · ✅ ingest inerte à rotação (teste) · ✅ container verde.

## Riscos
- Hotspot no daemon = dono único serial (evitar edição concorrente).