# Solução Tema 8 — Ponte entre Registry do ORQ2 e Multica Daemon (8 Fatos Alinhados)

- **Autor:** Agy-P0-A8 (wB:p2)
- **Data UTC:** 2026-07-26T23:13:30Z
- **Modo:** PREPARAR (Evidência Alinhada com os 8 Fatos Estabelecidos)

---

## 1. Mapeamento Correto de Raízes por Provider dentro dos Slots (`/home/ec2-user/.agent-cred-homes/slots/slot-NNN`)

Para evitar falhas silenciosas de credencial em `kiro_home.go:73-75` (`if srcMissing return nil`), a atribuição de `AccountHome` pelo daemon deve apontar especificamente para o subdiretório do provider dentro do slot:

| Provider | Subdiretório Específico no Slot | Exemplo de Caminho no Slot | Comportamento |
| :--- | :--- | :--- | :--- |
| **AGY / Antigravity** | `slot/home` | `/home/ec2-user/.agent-cred-homes/slots/slot-140/home` | Presente em todos os 22 slots |
| **KIRO** | `slot/xdg-data` | `/home/ec2-user/.agent-cred-homes/slots/slot-140/xdg-data` | **Restrito aos 5 slots:** 139, 140, 142, 143, 149 |
| **CODEX** | `slot/codex` | `/home/ec2-user/.agent-cred-homes/slots/slot-140/codex` | Presente nos slots habilitados |
| **CLINE** | `slot/cline` | `/home/ec2-user/.agent-cred-homes/slots/slot-140/cline` | Presente nos slots habilitados |

---

## 2. Correção da Regressão em `daemon.go:3448`

A substituição em `daemon.go:3448` restaura o encadeamento original (criado no commit `aa62401` e removido como regressão em `31d50b9`):

```go
// De (regressão 31d50b9):
credentialAccountHome := ""

// Para (restauração do wiring de contas/slots com mapeamento por provider):
credentialAccountHome := d.resolveAccountHomeForTask(task, provider)
```

### Regras do Resolvedor (`resolveAccountHomeForTask`):
1. **Identificação de Provider:** Se `provider == "kiro"`, filtrar exclusivamente o conjunto dos 5 slots que possuem Kiro registrado (slots `139`, `140`, `142`, `143`, `149`).
2. **Atribuição de Raiz:** Retornar o subdiretório exato do provider (`/xdg-data` para kiro, `/home` para agy, `/codex` para codex).
3. **Mapeamento:** Resolver `account.HomeDir` do Multica associado ao `terminal_id` do slot no `registry.json`.

---

## 3. Garantia de Conformidade
Nenhum arquivo de código foi editado ou recompilado. A solução estende a infraestrutura existente de slots no ORQ2 e repara a regressão em `daemon.go:3448`.
