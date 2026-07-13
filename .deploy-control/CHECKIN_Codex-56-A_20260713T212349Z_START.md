# CHECK-IN START — Codex-56-A

- UTC: 2026-07-13T21:23:49Z
- Slot: A (`~/.codex-slotA`)
- Escopo: daemon/Go do isolamento de credenciais para codex, kiro, antigravity, glm, cline e opencode.
- Change OpenSpec: `agent-credential-isolation`
- Restricao: nao editar `scripts/ops/agent-cred-isolation.sh`.
- Entrega prevista: fail-closed sem conta atribuida, homes por copia (sem symlink), testes de runtime para os seis vendors, evidencia de `go test` verde e commit atomico.
