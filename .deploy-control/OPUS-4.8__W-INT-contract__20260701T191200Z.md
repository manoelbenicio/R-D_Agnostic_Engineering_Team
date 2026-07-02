agent: OPUS-4.8
stream: W-INT-contract
started_at: 2026-07-01T19:12:00Z
finished_at: 2026-07-01T19:20:00Z
status: DONE
files_locked:
  - server/internal/daemon/execenv/execenv.go
  - server/internal/daemon/daemon.go
  - server/internal/daemon/execenv/codex_home.go
depends_on: []
build_result: >
  GREEN — golang:1.26-alpine: go build ./internal/daemon/... ./pkg/agent/... OK;
  go test ./internal/daemon/execenv/ OK (inclui testes de isolamento por conta).
notes: >
  CONTRATO PUBLICADO. Campo unico de credencial por conta:
  PrepareParams.CredentialAccountHome (string) e ReuseParams.CredentialAccountHome
  (string). Vazio = comportamento global historico (fallback total). Codex ja
  consome via CodexHomeOptions.AccountHome -> seedAccountAuth (copia auth.json
  isolado). Helper reutilizavel em codex_home.go:
  seedAccountAuth(accountHome, home, logger) copia <accountHome>/auth.json ->
  <home>/auth.json via syncCopiedFile (refresh-on-reuse). Padrao para novos
  vendors: criar arquivo NOVO execenv/<vendor>_home.go com
  prepare<Vendor>HomeWithOpts(home, opts{AccountHome}, logger). NAO editar
  execenv.go / daemon.go (dono unico OPUS-4.8) — apenas publicar a func nova; a
  fiacao no core (call site + injecao de env em daemon.go) eu faco.
