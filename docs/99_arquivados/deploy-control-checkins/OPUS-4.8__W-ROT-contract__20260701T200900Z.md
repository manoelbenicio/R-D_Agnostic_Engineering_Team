agent: OPUS-4.8
stream: W-ROT-contract
started_at: 2026-07-01T20:09:00Z
finished_at: 2026-07-01T20:11:00Z
status: DONE
files_locked:
  - server/internal/rotation/contract.go
depends_on: []
build_result: >
  GREEN — golang:1.26-alpine: go build ./internal/rotation/... OK; go vet OK.
notes: >
  CONTRATO FASE 2 PUBLICADO em internal/rotation/contract.go (só tipos +
  interfaces, sem logica). Destrava os 3 streams em PARALELO TOTAL, sem
  dependencia entre eles:
  - W-DETECT (CODEX-1): implementa ExhaustionDetector em detector.go
  - W-ROTATE (CODEX-2): implementa RotationService + pool em service.go/pool.go
  - W-PGSTORE (GLM-5.2): implementa Store (Postgres) em store_pg.go + migrations
  Model Account, AccountStatus, RotationReason, ExhaustionSignal, DetectionResult,
  AccountAuthenticator (port), Store, RotationService, ErrNoAccountAvailable — todos
  definidos. NAO editar contract.go (dono unico Opus); so implementar contra ele em
  arquivos NOVOS. Integracao no daemon = Opus depois.
