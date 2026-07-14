# DEPLOY PLAN — Isolamento de Credencial (Codex A/B, 24x7)

> Plano de registro. Trilhas paralelas, arquivos disjuntos.
> A = Codex-56-A (`w3:p18`, cargas pesadas, Go/daemon). B = Codex-56-B (`w3:pJ`, ops/shell).
> Kiro (TL) coordena e valida cada DONE na fonte (não produz código).
> Seção LIVE STATUS é atualizada automaticamente a cada ~3 min por monitor de fundo.

## Tasks × agente × ETA

| ID | Task | Agente | ETA | Dep | Status |
|----|------|--------|-----|-----|--------|
| B1 | Âncora estável de terminal (`terminal_id`; fallback tty+uuid) | Codex-56-B | ~15m | — | ✅ |
| B2 | Alocador de slots + `registry.json` (flock, monotônico, preserva 1º login/dia) | Codex-56-B | ~25m | B1 | ✅ |
| B3 | Export env por vendor → slot (codex/kiro/antigravity/glm/cline/opencode) | Codex-56-B | ~20m | B2 | ✅ |
| B4 | Fail-safe default (terminal não mapeado → slot próprio) | Codex-56-B | ~10m | B2 | ✅ |
| B5 | Migração por cópia dos homes atuais (preserva logins) | Codex-56-B | ~25m | B3 | ✅ |
| B6 | Doctor `status` (pane→terminal→slot→vendor→conta→on/off) | Codex-56-B | ~20m | B3 | ✅ |
| B7 | Prova empírica (2 logins mesmo vendor sem sobrescrita + recompactação; cline+agy) | Codex-56-B | ~30m | B4,B5 | ✅ |
| B8 | Instalar no `~/.bashrc` (remover bloco antigo) + evidência no RUNBOOK | Codex-56-B | ~15m | B7 | ✅ |
| A1 | Fail-closed em `daemon.go` (`credentialAccountHomeForTask`/`requiresCredentialIsolation`) | Codex-56-A | ~30m | — | ✅ |
| A2 | Cópia-não-symlink em `execenv/*_home.go` (6 vendors) | Codex-56-A | ~25m | A1 | ✅ |
| A3 | Estender + passar `runtime_isolation_test.go` (6 vendors, verde-em-container) | Codex-56-A | ~35m | A2 | ✅ |
| A4 | build/vet verde + commit atômico + evidência no `_DONE.md` | Codex-56-A | ~10m | A3 | ✅ |
| K1 | Validar cada DONE na fonte (concorrência p/ B; fail-closed + 6-vendor p/ A) | Kiro | ~15m/DONE | B8/A4 | ✅ |
| K2 | Atualizar RUNBOOK com caminho final do script + saída do `status` | Kiro | ~10m | K1 | ✅ |

ETA total (paralelo + validação): ~1h45–2h00, sem novo bloqueio de auth/modelo.

<!-- LIVE_STATUS_START -->
**LIVE STATUS** · atualizado 2026-07-14T04:43:45Z (auto/3min)

_Agentes:_
- Codex-56-A (w3:p18): **idle**
- Codex-56-B (w3:pJ): **idle**

_Check-ins em disco (recentes):_
- CHECKIN_Codex-56-A_20260713T213221Z_DONE.md
- CHECKIN_Codex-56-A_20260713T212349Z_START.md
- CHECKIN_Codex-56-B_20260713T212645Z_DONE.md
- CHECKIN_Codex-56-B_20260713T211559Z_START.md

_Últimos commits:_
- a564651 fix(daemon): enforce credential isolation fail-closed
- e352985 feat(ops): isolate pane credential homes
- ea18ed8 feat(agent): wire NIM and Cline runtimes
- 70fd719 feat(auth): add password login provider
- b47ac95 feat(agent): add native NIM OpenAI-compatible backend
- 7aebb16 docs(openspec): add task 1.7 backend /auth/login (Firebase-ready); mark 1.1/1.4 validated; clarify A5 owns marketing removal
<!-- LIVE_STATUS_END -->
