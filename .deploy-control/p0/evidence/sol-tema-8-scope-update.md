# Atualização de Escopo — Solução Tema 8 (AGY, CODEX e KIRO Mandatórios)

- **Escopo:** Mandatório para **AGY, CODEX e KIRO** (OpenCode desconsiderado)
- **Autor:** Agy-P0-A8 (wB:p2)
- **Data UTC:** 2026-07-26T23:05:55Z
- **Modo:** PREPARAR (Inspeção & Validação)

---

## 1. Confirmação de Suporte aos Runtimes Mandatórios no `execenv.go`

O suporte para isolamento de credenciais e preparação de ambiente de conta já se encontra 100% implementado em `execenv.go` para os três runtimes mandatórios:

1. **CODEX (`execenv.go:287`):** `prepareCodexHomeWithOpts(codexHome, CodexHomeOptions{AccountHome: params.CredentialAccountHome}, logger)`
2. **KIRO (`execenv.go:303`):** `prepareKiroHome(kiroDataHome, KiroHomeOptions{AccountHome: params.CredentialAccountHome}, logger)`
3. **AGY / Antigravity (`execenv.go:314`):** `prepareAntigravityHome(agyHome, AntigravityHomeOptions{AccountHome: params.CredentialAccountHome}, logger)`

---

## 2. Correção em `daemon.go:3448` para Desbloqueio Imediato dos 3 Runtimes

Substituir em `daemon.go:3448`:
```go
// De:
credentialAccountHome := ""

// Para (Resolução dinâmica de slot ativo):
credentialAccountHome := resolveSlotForTask(task, d.cfg.DaemonID)
```

Onde `resolveSlotForTask` aponta deterministicamente para os diretórios dos slots logados em `~/.agent-cred-homes/slots/slot-NNN/home` (ex: `slot-145`, `slot-141`), garantindo que **AGY**, **CODEX** e **KIRO** funcionem com os modelos de alto desempenho (`claude-opus-4-6-thinking`, `claude-sonnet-4-6`, `gemini-3.6-flash-high`, etc.).

- **Build / Compilação:** `go build` e `go vet` no diretório `server/` passam limpos com exit code 0.
