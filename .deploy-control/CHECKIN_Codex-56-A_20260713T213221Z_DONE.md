# CHECK-IN DONE — Codex-56-A

- UTC: 2026-07-13T21:32:21Z
- Slot: A (`~/.codex-slotA`)
- Status: DONE
- Change OpenSpec: `agent-credential-isolation`
- Script reservado do slot B: `scripts/ops/agent-cred-isolation.sh` nao editado.

## Entrega

- `credentialAccountHomeForTask` auditado: os seis vendors P0 exigem store, agent ID, assignment, account compativel e `home_dir`; qualquer ausencia retorna erro antes do spawn.
- Injeção de credencial centralizada e fail-closed: Codex, Kiro, Antigravity, GLM, Cline e OpenCode precisam expor todas as env vars nativas preparadas; erro no refresh/reuse nao cai para credencial compartilhada.
- `custom_env` impedido de sobrescrever `XDG_CONFIG_HOME`, `CLINE_DATA_DIR`, `CLINE_SANDBOX` e `CLINE_SANDBOX_DATA_DIR`, alem das chaves ja protegidas.
- Codex passou a semear `auth.json` pelo copiador de credencial, preservando modo e sempre criando arquivo regular; os demais vendors continuam usando copia recursiva/arquivo regular, nunca symlink de credencial.
- `runtime_isolation_test.go` cobre exatamente os seis vendors P0 e prova: duas contas sem sobreposicao, fail-closed, env nativa completa, credencial copiada regular (nao symlink) e ausencia de segredo nos logs.
- Contrato preexistente de NIM preservado fora da matriz P0 de seis vendors.

## Evidencia verde em container

Imagem Go: `golang:1.26-alpine`; banco efemero: `postgres:17-alpine`; `DATABASE_URL` apontou para o Postgres da rede isolada.

1. `go test ./internal/daemon -run "TestCredentialIsolation" -count=1 -v`
   - PASS para `codex`, `kiro`, `antigravity`, `glm`, `cline`, `opencode`.
   - Em cada vendor: `two_accounts_coexist_without_overlap`, `fail_closed_no_assignment` e `no_secret_in_log` PASS.
   - Resultado: `ok github.com/multica-ai/multica/server/internal/daemon 0.159s`.
2. `go test ./internal/daemon/execenv -count=1`
   - Resultado: `ok github.com/multica-ai/multica/server/internal/daemon/execenv 0.285s`.
3. `go test ./internal/daemon -count=1`
   - Resultado: `ok github.com/multica-ai/multica/server/internal/daemon 15.714s`.
4. `git diff --check`
   - PASS, sem diagnosticos.
