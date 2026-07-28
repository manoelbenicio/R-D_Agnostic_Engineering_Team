# TEMA 5 — Slots de credencial no ORQ1 via agent-credential-isolation
**Revisor:** Antigravity (Opus48#C / w8:p1)
**Data:** 2026-07-26T23:05Z
**Status:** PROPOSTA — NÃO APLICADA. Requer decisão escrita do owner.

---

## O que a ferramenta faz (evidência literal)

Arquivo: `/home/ec2-user/.local/lib/agent-credential-isolation/scripts/ops/agent-cred-isolation.sh`
Só existe no **ORQ2**. O ORQ1 **não tem** o script nem `~/.agent-cred-homes/`.

Funciona em 3 camadas:

| Função | O que faz |
|---|---|
| `agent_cred_isolation_bootstrap` | Aloca um `slot-NNN` no `registry.json` por terminal_id (Herdr pane ID ou TTY hash). Idempotente. |
| `agent_cred_isolation_bootstrap` (vendor) | Exporta `CODEX_HOME`, `XDG_DATA_HOME`, `XDG_CONFIG_HOME`, `CLINE_DATA_DIR`, `HOME` apontando para `~/.agent-cred-homes/slots/slot-NNN/home`. **NÃO copia credencial, não faz login.** |
| `agent_cred_isolation_grok_device_login <A\|B\|C\|D>` | Faz device-login para Grok apenas. Sem equivalente para agy/kiro/cline/opencode/codex. |

**Conclusão sobre logins:** o script **não provisiona** credenciais de agy, kiro, cline, opencode, codex. Ele apenas **isola homes já logados**. O login de cada CLI tem que ser feito manualmente dentro do slot com `HOME=<slot>/home agy login`, `kiro login`, etc. — e isso é STOP-AND-WAIT (requer decisão do owner, pois envolve autenticação com contas externas).

---

## Estado atual no ORQ1 (evidência de comando)

```
~/.agent-cred-homes/           NÃO EXISTE (zero slots)
~/.local/lib/agent-credential-isolation/   NÃO EXISTE (script ausente)
asm-exec                       NOT FOUND na PATH
```

**O que existe no HOME global do ORQ1 (copiado pelo TL):**
```
~/.gemini/antigravity-cli/antigravity-oauth-token  1439 bytes  Jul 26 22:37  → agy=ON (global)
~/.local/share/kiro-cli/data.sqlite3               69632 bytes Jul 26 22:36  → kiro=ON (global)
~/.codex/auth.json             NÃO EXISTE → codex=off no HOME global
~/.agent-cred-homes/slots/     NÃO EXISTE → nenhum slot isolado
```

---

## Por que "provisionar slots hoje no ORQ1" NÃO é possível sem ação do owner

| Passo | Requerido | Proibido sem owner? |
|---|---|---|
| Copiar o script para o ORQ1 | `scp` ou `rsync` ORQ2→ORQ1 | **SIM** (instalação/cópia de infra) |
| Criar estrutura `~/.agent-cred-homes/` | Bootstrap script no ORQ1 | **SIM** (modifica HOME do processo daemon) |
| Fazer login de `agy` num slot | `HOME=<slot>/home agy login` no ORQ1 | **SIM** (autenticação com conta externa) |
| Fazer login de `kiro` num slot | `HOME=<slot>/home kiro login` no ORQ1 | **SIM** (idem) |
| Fazer login de `cline` num slot | `HOME=<slot>/home cline login` no ORQ1 | **SIM** (idem) |
| Fazer login de `opencode` num slot | `HOME=<slot>/home opencode login` no ORQ1 | **SIM** (idem) |

---

## Estado atual funciona hoje (sem slots)

O TL já tem `agy` e `kiro` logados no **HOME global** do ORQ1. O daemon, quando preencher `CredentialAccountHome` com esse path (`/home/ec2-user`), vai encontrar as credenciais **sem precisar de slots isolados**. Para a fase atual (1 conta por CLI, sem isolamento por agente), o HOME global é suficiente.

**O que a isolação por slot adiciona:** multi-conta — 4 contas agy em slots diferentes = 4 runtimes paralelos com credenciais separadas. Isso é a fase **seguinte**, não a fase atual.

---

## Proposta concreta para o owner

### Opção A — Usar HOME global agora (zero infra nova, funciona hoje)
- O daemon preenche `CredentialAccountHome = /home/ec2-user` para os providers agy e kiro.
- agy já logado: `~/.gemini/antigravity-cli/antigravity-oauth-token` presente.
- kiro já logado: `~/.local/share/kiro-cli/data.sqlite3` presente.
- **Risco:** sem isolamento — todas as tasks usam a mesma conta. Custo vai para uma conta só.
- **Ação necessária:** apenas o patch de `daemon.go:3448` que preenche `credentialAccountHome` (outro pacote).

### Opção B — Provisionar slots no ORQ1 (isolamento completo, não-hoje)
**Sequência exata que o owner precisaria autorizar:**
```bash
# 1. Copiar script ORQ2 → ORQ1 (requer autorização)
scp ~/.local/lib/agent-credential-isolation/scripts/ops/agent-cred-isolation.sh \
    ec2-user@<orq1-ip>:~/.local/lib/agent-credential-isolation/scripts/ops/

# 2. No ORQ1, criar slot para conta agy (a2a7860c dataops.cloud.mbf):
ssh orq1 'AGENT_CRED_ISOLATION_ENABLE_VENDOR_SLOTS=1 \
  source ~/.local/lib/agent-credential-isolation/scripts/ops/agent-cred-isolation.sh && \
  HOME=~/.agent-cred-homes/slots/slot-NNN/home agy login'

# 3. Repetir para cada conta (4 contas agy, 1 kiro, etc.)
# 4. O daemon passaria CredentialAccountHome = ~/.agent-cred-homes/slots/slot-NNN/home
```
- **Pré-requisito bloqueante:** o script não faz login automático — cada conta exige autenticação interativa (device-flow ou browser). Não é automatizável sem credencial pré-existente.

---

## Recomendação ao owner

**Hoje:** Opção A. O HOME global já tem agy e kiro logados. O único código a mudar é `daemon.go:3448` (preencher `credentialAccountHome`). Sem nova infra.  
**Depois (fase multi-conta):** Opção B com autorização explícita, script copiado e logins interativos por conta.
