# Solução Tema 8 — Associação de Tasks a Contas/Slots no Daemon

- **Tema:** TEMA 8 — Associação, Rotação e Afinidade de Contas/Slots por Task
- **Autor:** Agy-P0-A8 (wB:p2)
- **Data UTC:** 2026-07-26T23:05:30Z
- **Modo:** PREPARAR (Inspeção de código Go no repositório, sem edições ou recompilação)

---

## 1. Como o Daemon Associa uma Task a uma Conta Hoje (Causa Raiz)

Hoje o daemon **NÃO faz associação nem rotação de contas**. Em `daemon.go:3448`, a variável `credentialAccountHome` é hardcodada como string vazia `""`:

- **`daemon.go:3448`:** `credentialAccountHome := ""`
- **`daemon.go:3475`:** `CredentialAccountHome: credentialAccountHome,` (em `execenv.Reuse`)
- **`daemon.go:3492`:** `CredentialAccountHome: credentialAccountHome,` (em `execenv.PrepareParams`)

No pacote `execenv`, o comportamento para `CredentialAccountHome == ""` é tratar como comportamento global/compartilhado (shared/global behavior):

- **`execenv.go:300-301` (kiro):** `// Empty CredentialAccountHome = shared/global behavior` -> `if params.Provider == "kiro" && params.CredentialAccountHome != ""`
- **`execenv.go:311-312` (antigravity):** `// Empty CredentialAccountHome = shared/global behavior` -> `if params.Provider == "antigravity" && params.CredentialAccountHome != ""`
- **`execenv.go:321-322` (cline):** `// CredentialAccountHome = shared/global behavior` -> `if params.Provider == "cline" && params.CredentialAccountHome != ""`
- **`execenv.go:340-341` (opencode/glm):** `// Empty CredentialAccountHome = shared/global behavior` -> `if (params.Provider == "opencode" || params.Provider == "glm") && params.CredentialAccountHome != ""`

---

## 2. Proposta de Seleção, Afinidade e Rotação de Slots (`~/.agent-cred-homes/slots/slot-NNN/home`)

Para que cada task utilize um slot isolado ativo, o valor de `credentialAccountHome` em `daemon.go:3448` deve ser populado dinamicamente:

### Mecanismo de Resolução de Slot
1. **Verificação de Slots no Host:** O daemon lê os diretórios válidos em `~/.agent-cred-homes/slots/slot-*/home`.
2. **Afinidade por Agente / Task (`AgentID` / `WorkspaceID`):**
   - Mapeamento por HASH determinístico: `slotIndex = hash(task.AgentID) % totalSlots`.
   - Garante que a mesma task ou agente mantenha afinidade com a mesma conta/slot (evitando trocas de contexto/sessão desnecessárias).
3. **Fallback por Rotação (Round-Robin):**
   - Caso `AgentID` não possua afinidade fixada, seleciona o próximo slot disponível em `~/.agent-cred-homes/slots/slot-NNN/home` através de contador atômico por provider.
4. **Preenchimento em `PrepareParams` / `ReuseParams`:**
   - Atribui o caminho do slot resolvido a `credentialAccountHome`, acionando automaticamente `prepareKiroHome`, `prepareAntigravityHome`, `prepareClineHome` e `prepareOpenCodeHome` no `execenv.go`.

---

## 3. Garantia de Inalterabilidade
Nenhum arquivo de código Go foi editado, nenhum binário foi compilado e o daemon (PID 3244391) não foi alterado nesta rodada.
