# Inventário de Autenticação — 4 projetos (auditoria 2026-07-01)

Auditoria completa via SSH no servidor (192.168.15.6). Existem **três camadas
de auth distintas** — não confundir. O "método de auth próprio" do produto é a
Camada A no AOP (device-grant OAuth), em desenvolvimento ativo.

## Camada A — Login de plataforma / humano (o "auth próprio" desenvolvido)
Local autoritativo (mais recente, 2026-06-28), **AOP/agnostic-ai-platform/backend/app**:
- `routers/auth.py` (13KB) — **OAuth 2.0 Device Authorization Grant (RFC 8628)**
  próprio: `/device/authorize`, `/device/token`, `/device/google/authorize`
  (login via **Google ID token**), emissão de JWT, user codes, expiry pruning.
- `auth.py` — middleware JWT (HS256). ⚠️ `SECRET_KEY` **hardcoded**
  ("super_secret_agnostic_key") → mover para env/secret.
- `auth_middleware.py`, `routers/auth_router.py`.
- `frontend/src/components/DeviceLogin.tsx` — fluxo device + QR code.
- Estado: device store **in-memory** (`DEVICE_AUTHORIZATIONS_BY_*`) → precisa
  persistência para produção multi-processo.

Cópia paralela (Multica, em `New_Product/.../multica-src`, 2026-06-23):
`server/internal/handler/auth.go` (22KB), `middleware/auth.go`,
`middleware/daemon_auth.go`, `cmd/multica/cmd_login.go`, `cmd_auth.go`,
`scope_authorizer.go`; frontend `login-page.tsx`, `auth-cookie.ts`,
`auth-initializer.tsx`; desktop `daemon-auth-probe.ts`, `daemon-reauth.ts`;
mobile `auth-store.ts`, `(auth)/login.tsx`. É a auth do produto Multica (Go).

## Camada B — Isolamento de credencial de agente CLI (nosso alvo Fase 1)
**AOP/control-plane**:
- `seats/pool.py` — `Seat(home_dir, config_dir)` + `get_env()` (HOME/SEAT_ID/…);
  `SeatPool` por (tenant, vendor) com lease/ref_count anticolisão.
- `sessions_api/service.py` — `DeviceLoginService.start(seat_id)`,
  `_prepare_isolated_paths` (abs + config dentro do home + mkdir 0700),
  `_vendor_env` (codex→CODEX_HOME, claude→CLAUDE_CONFIG_DIR,
  gemini→GEMINI_CONFIG_DIR, kiro→KIRO_HOME+KIRO_CONFIG_DIR), `provider_commands`
  por vendor, estados degraded. Device-login/OAuth — **sem senha em disco**.
- `seats_api/` (router/repository/schema), web `seats`+`sessions`, `SeatCards.tsx`.
- Testes: `seats_api/tests/test_seats_sessions_api.py`.

## Camada C — Rotação (duas implementações — não confundir)
- **C1 (nosso alvo Fase 2): AOP/control-plane/rotation/** (879 linhas) — rotação
  de **conta CLI** por esgotamento de token. `models.py` (Account, status,
  VENDOR_PRIORITY, janela 5h/cooldown), `detector.py`, `trigger.py`,
  `service.py`, `auth.py` (DeviceLoginAuthenticator logout/login/wait), `pool.py`,
  `assembly.py`. Teste: `rotation/tests/test_rotation_skeleton.py` (skeleton →
  ainda não completo). Spec: `AOP/docs/30-COMPONENTES/36-ROTACAO-CONTAS-TOKEN.md`.
- **C2 (NÃO é o alvo): AOP/agnostic-ai-platform/app/auth/api_key_rotation.py**
  (234 linhas) — ciclo de vida de **API keys de plataforma**, e é **mock**
  (`MOCK_KEY_DB` in-memory, hashing "simulado"). Concern diferente.

## Maturidade (git)
- AOP é um monorepo v1.0 recente; commits relevantes: "API key rotation service"
  (C2), "AOP v1.0 initial", "multi-agent sprint". Camadas B e C1 vieram no bojo
  do initial/sprint; C1 tem só teste-skeleton → **funcional mas não endurecido**.
- `openspec/changes` do AOP só tem `archive` → sem change ativo cobrindo isto.

## Serviços rodando no servidor (ss -tlnp)
Portas ativas: 80, 22, 3000 (grafana), 5000, 5432 (postgres), 6379 (redis),
8005, 8020, 9094/9095/9098/9104/9120/9870 (métricas/observabilidade). Único
processo app claramente identificado: `python /app/webhook_server.py` (remediação).
Não há uvicorn/next do agnostic-ai-platform rodando agora → **Camada A está em
dev, não em produção contínua** neste host.

## Flags de segurança (registrar como itens separados)
1. `SECRET_KEY` JWT hardcoded em `agnostic-ai-platform/backend/app/auth.py`.
2. `api_key_rotation.py`: hashing simulado + DB in-memory (não-produção).
3. Device-auth store in-memory (não sobrevive a restart / multi-worker).
4. HerdMaster control API `token = "admin"` (fraco) — visto no config vivo.

## Persistência de credenciais — SQLite → Postgres (confirmado)

Confirmado o histórico relatado pelo operador: começou em SQLite, deu **lock**
(`database is locked` sob concorrência), migrou para **Postgres**.

Estado atual (control-plane AOP):
- `seats_api/repository.py` e `sessions_api/repository.py` usam **`psycopg`
  (Postgres v3.3.4)** — `SeatsRepository`/`SessionsRepository` recebem uma
  `psycopg.Connection`. **Não há SQLite** nessas camadas.
- `app/database/connection.py`: `DATABASE_URL` (default
  `postgresql+asyncpg://…@localhost:5432/aop`), **connection pool** SQLAlchemy
  (`DB_POOL_SIZE=20`, `max_overflow=10`, `pool_timeout=30`) → resolve o lock.
- Migrations Alembic (`app/database/migrations/versions/001_initial.py`):
  tabelas `tenants, users, roles, agents, adapters, provisioning_requests,
  audit_logs, cost_records, notifications, platform_settings` (UUID PKs). As de
  `seats`/`sessions` têm schema próprio (repos com SQL direto via psycopg).
- Postgres está **rodando** no servidor (porta 5432). `deps`: `psycopg[binary]`
  + `psycopg-pool`.

Contraste — **onde o SQLite legado ainda vive** (a fonte do lock, NÃO migrado):
- **HerdMaster** (`~/.config/herdmaster/herdmaster.db`) e Kiro
  (`~/.local/share/kiro-cli/data.sqlite3`) ainda são SQLite. HerdMaster teve
  mitigação "single-writer" (webhook via HTTP API, não escreve SQLite direto).
- **Implicação de design (R3 revisitado):** o Kiro guarda auth em SQLite por
  conta (data.sqlite3). Isolar por `XDG_DATA_HOME` = **um sqlite por conta** →
  evita o lock naturalmente (sem writers concorrentes no mesmo arquivo). A
  persistência de metadados de seat/session/rotação, essa sim, vive no Postgres.

## Conclusão para o change
- **Fase 1** = portar Camada B (control-plane seats/sessions) para o runtime que
  spawna os agentes; corrigir `_vendor_env` de kiro/gemini vs realidade (§3/§5).
- **Fase 2** = completar/portar Camada C1 (control-plane/rotation).
- **Camada A** e as flags de segurança = fora do escopo deste change, mas
  documentadas aqui para virarem tickets próprios.
