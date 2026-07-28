# CORREÇÃO GTL-11R: Plano de Cutover Durável & Rollback ORQ-23 (READ-ONLY)

- **Autor:** Agy-P0-A8 (wB:p2)
- **Destinatários:** Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
- **Data UTC:** 2026-07-27T11:45:28Z
- **Modo:** SOMENTE LEITURA / PLANO REVISADO GTL-11R (Zero execução, zero mutação)

---

## 1. Inventário Factual Completo dos 6 Binários no ORQ2

Todas as medições de hashes SHA256 e tamanhos foram obtidas diretamente no host ORQ2 (`/home/ec2-user/.local/lib/multica/bin/`):

| Nome do Arquivo | Tamanho (Bytes) | Hash SHA256 Medido | Classificação / Status |
| :--- | :--- | :--- | :--- |
| `multica-auth-credential-home-v1` | **15.053.065 bytes** | `88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8` | **ATIVO LIVE (Comprovadamente Bom)** |
| `multica-auth-credential-home-v1.gate1-20260727T102815Z` | **15.053.065 bytes** | `88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8` | **ALVO ÚNICO DE ROLLBACK (Gate 1 Baseline)** |
| `multica-auth-credential-home-v1.new` | 21.344.766 bytes | `5f9ec49e3c4c37bf06b8001aa57c8961ed13a0961ebbb3c9b34da03d6905562e` | 🚫 **PROIBIDO (Known-bad pre-token-only)** |
| `multica-auth-credential-home-v1.pre-token-only-20260727T102815Z` | 21.344.766 bytes | `5f9ec49e3c4c37bf06b8001aa57c8961ed13a0961ebbb3c9b34da03d6905562e` | 🚫 **PROIBIDO (Known-bad pre-token-only)** |
| `multica-auth-credential-home-v1.pre-agy-fix` | 21.333.562 bytes | `f34630864b40ad5ad2cfee19ba53da9588f0c813b24890845bf3438ddf96ec1a` | 🚫 **PROIBIDO (Known-bad pre-AGY fix)** |
| `multica-auth-credential-home-v1.previous` | 21.333.458 bytes | `e0510d7daf73ea9f61b859273c6016d8593780507a8860b3bb1e8e260eff1ba6` | 🚫 **PROIBIDO (Versão antiga legada)** |

---

## 2. Requisito 1 & 2: Compilação Limpa a partir do Pacote Correto

- **Repositório Limpo:** `multica-auth-work/server` (Commit Base SHA `0cb8aebb5aff79cb430b3740d22fadc53c0116fd`).
- **Pacote Target Correto:** `cmd/multica/main.go` (**NÃO** `cmd/server/main.go`).
- **Comando de Compilação (Requer Autorização Gate D1 do Owner):**
  ```bash
  cd /home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server
  CGO_ENABLED=0 /home/ec2-user/goroot/go/bin/go build -trimpath -ldflags="-s -w" -o /home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1.orq23-staging cmd/multica/main.go
  ```
- **Validação Pós-Build:** Calcular `sha256sum` do novo arquivo estagiado antes de qualquer substituição.

---

## 3. Requisito 3 & 4: Procedimento de Rollback por Hash SHA256

Em caso de anomalia durante o cutover, a reversão é realizada **estritamente comparando o hash SHA256** contra a linha de base do Gate 1 (`88ca4f39...`), e **NUNCA** por tamanho em bytes.

```bash
# 1. Verificar hash da imagem de rollback
sha256sum /home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1.gate1-20260727T102815Z
# Deve retornar obrigatoriamente: 88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8

# 2. Executar substituição atômica e reload no systemd de usuário
cp -a /home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1.gate1-20260727T102815Z /home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1.tmp
mv /home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1.tmp /home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1
systemctl --user daemon-reload
systemctl --user restart multica-daemon-orq2-credential.service
```

---

## 4. Requisito 7: Condição de Fila Vazia Pre-Cutover

Antes de iniciar a substituição do binário, a fila de tarefas do banco PostgreSQL deve ser auditada:

```sql
SELECT count(*) FROM agent_task_queue WHERE status IN ('queued', 'dispatched', 'running');
```

- **Invariante:** O retorno DEVE ser obrigatoriamente `0`. Se houver tarefas ativas ou em fila, o cutover DEVE ser abortado.

---

## 5. Requisito 8: Estratégia de Healthcheck Desacoplada & Esclarecimento AGY

1. **Desacoplamento de Healthchecks:**
   - **H1 (Healthcheck Não-Live):** `GET http://127.0.0.1:18080/api/runtimes` para validar o heartbeat dos deamons em repouso.
   - **H2 (Teste de Execução Live):** A execução de uma task de fumaça ao vivo é realizada em um **Gate separado (D4)** com autorização própria.
2. **Esclarecimento do Slot 142:**
   - Presença física de arquivos no diretório do slot `142` indica **Auth Present**, mas NÃO garante uma sessão ativa (**Session Alive**), pois o slot não pertence à allowlist ativa da unidade em execução.
3. **Esclarecimento do Erro `cli.log` do AGY:**
   - O erro de symlink em `cli.log` é um registro histórico **STALE** (antigo). Ele foi 100% resolvido após o Gate 2 e a fumaça bem-sucedida da ORQ-27, não representando uma falha ativa do sistema.

---

## 6. Matriz de Comandos-Gate (§0.1 STOP-AND-WAIT)

| Gate | Ação Proposta | Escopo / Host | Requisito de Liberação |
| :--- | :--- | :--- | :--- |
| **D1** | Build staging `cmd/multica` | ORQ2 | Aprovação explícita do Owner (§0.1) |
| **D2** | Validação de Fila Vazia (`count == 0`) | ORQ1 DB | Fila em zero tarefas ativas |
| **D3** | Substituição Atômica & Restart | ORQ2 Systemd User | Confirmação do Hash SHA256 |
| **D4** | Teste de Execução Live (H2) | ORQ2 Runtime | Gate de Fumaça Separado |
