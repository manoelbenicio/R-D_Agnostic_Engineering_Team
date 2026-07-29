# Evidência de Materialização do Rollback de Credential Account Home (ORQ-23)

> **Data / Timestamp:** 2026-07-29T13:31:00Z  
> **Issue Assignee / Executor:** `agy-p0-a7` (`780104f1-1ff4-4292-8207-44b9ac4f5fca`)  
> **Issue:** ORQ-23 (`6230b5c0-c57a-4b88-a7df-84899057d0c8`)  
> **Status:** Materializado & Validado em Dry-Run (Em Revisão)

---

## 1. Resumo da Implementação

Foi materializado e validado o artefato de rollback em 1 comando para restauração do diretório de credenciais (`credential-account-home`) e referências do daemon Multica.

### Localização dos Artefatos
1. **Repositório (Bounded Script):**  
   `scripts/ops/rollback-cred-account-home.sh`  
   **SHA256:** `e1c04180092cb5a267bd57d3179cd5f4ddd979a3ad2a288acd74bb05f79b7c5e`
2. **Caminho Local Privado Aprovado (Instalação 0700):**  
   `/home/ec2-user/.local/bin/rollback-cred-account-home.sh`  
   **Permissões:** `0700` (`-rwx------`)  
   **SHA256:** `e1c04180092cb5a267bd57d3179cd5f4ddd979a3ad2a288acd74bb05f79b7c5e`
3. **Script de Isolamento Atualizado:**  
   `scripts/ops/agent-cred-isolation.sh` e `/home/ec2-user/.local/lib/agent-credential-isolation/scripts/ops/agent-cred-isolation.sh`  
   **SHA256:** `c67bc5dce5b529d799fa39474545cfa3cc657de6520abdf2cd8d234f6acf6c62`

---

## 2. Validação do Harness de Isolamento

Executado o harness de teste de isolamento de credenciais:
```bash
bash /home/ec2-user/.local/lib/agent-credential-isolation/scripts/ops/tests/agent-cred-isolation-harness.sh
```
**Resultado:** `PASS: Codex no-delete leases, GROK A-D physical profiles/device-login isolation, 6-vendor migration, recompaction, fallback, and flock allocators`

---

## 3. Saída do Dry-Run (Content-Free Verification)

Execução do comando em modo `--dry-run` enquanto tarefas de produto continuavam ativas:

```
[2026-07-29T13:30:54Z] === ONE-COMMAND ROLLBACK DRY-RUN VERIFICATION (ORQ-23) ===
[2026-07-29T13:30:54Z] Mode: DRY-RUN (Content-Free Inspection)
[2026-07-29T13:30:54Z] Active Product Tasks Count: 1
[2026-07-29T13:30:54Z] GTL Cutover Dispatch Flag: 0
[2026-07-29T13:30:54Z] F3 Cost Gate Status: BLOCKED on ORQ-13/14 (not claimed)
[2026-07-29T13:30:54Z] --- Checking Daemon Binary References ---
[2026-07-29T13:30:54Z] Active Daemon Binary: /home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1 (SHA256: 88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8)
[2026-07-29T13:30:54Z] Prior Daemon Binary:  /home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1.previous (SHA256: e0510d7daf73ea9f61b859273c6016d8593780507a8860b3bb1e8e260eff1ba6)
[2026-07-29T13:30:54Z] --- Checking Systemd Service Reference ---
[2026-07-29T13:30:54Z] Systemd Unit File: /home/ec2-user/.config/systemd/user/multica-daemon-orq2-credential.service (SHA256: 065ba8e6e676d02087d8b3c88d4e2eca1369df8477e5112d07dd5c25d76e00ff)
[2026-07-29T13:30:54Z] --- Verifying Required Runtimes (Content-Free) ---
[2026-07-29T13:30:54Z] Runtime check [PASS]: Codex -> path=/home/ec2-user/.nvm/versions/node/v22.23.1/bin/codex (codex-cli 0.145.0)
[2026-07-29T13:30:54Z] Runtime check [PASS]: Antigravity -> path=/home/ec2-user/.local/bin/agy (1.1.8)
[2026-07-29T13:30:54Z] Runtime check [PASS]: Kiro -> path=/home/ec2-user/.local/bin/kiro-cli (kiro-cli 2.13.0)
[2026-07-29T13:30:54Z] --- Checking Credential Homes Root ---
[2026-07-29T13:30:54Z] Credential Homes Root: /home/ec2-user/.agent-cred-homes (Directory Present, Mode: 700)
[2026-07-29T13:30:54Z] === DRY-RUN VERIFICATION RESULT: PASS ===
[2026-07-29T13:30:54Z] SAFEGUARD: Active product tasks (1) are currently running.
[2026-07-29T13:30:54Z] Destructive live rollback MUST NOT be executed until active tasks = 0 and GTL dispatches cutover.
```

---

## 4. Teste da Trava de Segurança (Live Safeguard Test)

Tentativa de execução em modo `--live` enquanto tarefas ativas persistem:
```bash
/home/ec2-user/.local/bin/rollback-cred-account-home.sh --live
```
**Resultado Esperado e Obtido:**
```
[2026-07-29T13:30:58Z] === ONE-COMMAND ROLLBACK LIVE EXECUTION (ORQ-23) ===
[2026-07-29T13:30:58Z] Active Product Tasks Count: 1
[2026-07-29T13:30:58Z] GTL Cutover Dispatch Flag: 0
[2026-07-29T13:30:58Z] ERROR: LIVE ROLLBACK REFUSED: active product tasks (1) remain running. Live rollback requires active task count = 0.
```

---

## 5. Declarações de Escopo e Portões

1. **Rollback Destrutivo / Ao Vivo:** Não executado nesta fase. O rollback ao vivo permanece retido até que a fila de tarefas ativas atinja zero e o GTL despache o cutover final.
2. **Portão F3 / Custo:** Preservado como **BLOQUEADO** em `ORQ-13/14`. Não é alegada conclusão do portão de custo nesta issue.
3. **Conformidade Content-Free:** Nenhuma credencial ou token foi lido ou exposto durante a verificação dos três runtimes (`codex`, `agy`, `kiro-cli`).
