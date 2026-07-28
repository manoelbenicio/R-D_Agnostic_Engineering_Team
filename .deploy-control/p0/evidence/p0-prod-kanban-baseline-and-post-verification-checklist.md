# P0 PROD KANBAN — Preservação de Baseline ORQ1 e Checklist de Verificação Pós-Execução (READ-ONLY)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data/Hora UTC:** `2026-07-27T16:50:45Z`
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1), Agente `A8`
- **Governança:** P0 Prod Kanban Mechanical Baseline & Post-Execution Verification
- **Host Alvo Medido Direta via SSH**: `ec2-user@100.118.244.61` (Hostname: `ip-172-31-18-217.sa-east-1.compute.internal`)
- **Skill Carregada:** `.agents/skills/aws-secrets-manager/SKILL.md` (zero texto claro, zero `GetSecretValue`).
- **Modo:** MONITORAMENTO E PREPARAÇÃO READ-ONLY — Zero toque em segredos, banco de dados ou configurações. Preservação integral do baseline e definição de protocolo de verificação pós-sinal de Kiro.

---

## 1. Preservação de Baseline no Nó ORQ1 (`100.118.244.61`) (UTC `2026-07-27T16:50:43Z`)

| Métrica Inspecionada | Resultado Factual Medido no ORQ1 | Estado de Preservação |
|---|---|---|
| **Hostname ORQ1** | `ip-172-31-18-217.sa-east-1.compute.internal` | **CONFIRMADO** |
| **Frontend Container** | Container `1b3b6c9ee32a` (Next.js `:13100`) | **`Up 13h (HTTP 200 OK)`** |
| **Backend Container** | Container `75e4416f06e9` (Go API `:18080`) | **`Up 13m (HTTP 200 OK)`** |
| **Backend Healthz Probe**| `curl http://127.0.0.1:18080/healthz` | **`HTTP 200 OK`** (`application/json`) |
| **Backend Readyz Probe** | `curl http://127.0.0.1:18080/readyz` | **`HTTP 200 OK`** (`application/json`) |
| **Tailscale Serve Status** | `tailscale serve status` | **`No serve config`** (OFF / ZERADO) |
| **Tailscale Funnel Status**| `tailscale funnel status` | **`No serve config`** (OFF / ZERADO) |
| **Fila de Tarefas Ativa** | Query SQL em 4 estados | **`0`** (`queued`, `dispatched`, `running`, `waiting`) |
| **Sondas Anônimas** | `GET /api/me` e `GET /api/issues` | **`HTTP 401`** (`text/plain` fail-closed) |
| **Backups & Evidências** | Estrutura em `.deploy-control/` e backups | **`100% PRESERVADOS`** |

---

## 2. Protocolo de Verificação Pós-Execução (Após Sinal de Kiro)

Assim que Kiro/General-TL emitir o sinal de conclusão de execução, este agente realizará estritamente a seguinte suíte de verificação pós-execução **READ-ONLY**:

1. **Inspeção de IDs e Estabilidade de Containers**:
   - Verificar IDs de container (`1b3b6c9ee32a` e `75e4416f06e9`) e tempo de execução (`uptime`).
2. **Medição de Códigos de Status HTTP**:
   - Medir `:13100` (Frontend), `:18080/healthz` e `:18080/readyz` (Backend Health/Readiness).
   - Validar sondas anônimas `/api/me` e `/api/issues` mantendo contrato fail-closed (`HTTP 401`).
3. **Validação do Estado do Tailscale Serve e 7 Mounts Canônicos**:
   - Inspecionar `tailscale serve status --json` no ORQ1 para validar a presença dos **7 Mounts Canônicos**:
     * `/auth/login` -> `http://127.0.0.1:18080/auth/login`
     * `/auth/google` -> `http://127.0.0.1:18080/auth/google`
     * `/auth/logout` -> `http://127.0.0.1:18080/auth/logout`
     * `/ws` -> `http://127.0.0.1:18080/ws`
     * `/api` -> `http://127.0.0.1:18080/api`
     * `/uploads` -> `http://127.0.0.1:18080/uploads`
     * `/` -> `http://127.0.0.1:13100/`
4. **Registro de Evidências Pós-Execução**:
   - Gerar artefato de evidência formal pós-execução com SHA-256 e notificação `herdr` para `Codex56-TL` (`w5:pC`).

---

## 3. Garantia de Inviolabilidade de Segredos, Banco e Configurações

- **Segredos AWS**: Zero chamadas a `secretsmanager:GetSecretValue` ou SMA.
- **Banco de Dados**: Zero mutações de esquema ou dados executadas.
- **Configuração**: Zero alteração de arquivos de ambiente ou parâmetros de serviço.

---

## 4. Declaração de FILES_LOCKED

- `multica-auth-work/server/internal/middleware/auth.go` (LOCKED — ORQ-17 Auth Hardening V2)
- `multica-auth-work/server/internal/middleware/auth_test.go` (LOCKED — ORQ-17 Tests)

---

## 5. Veredito Final
- **STATUS: BASELINE PRESERVED & POST-VERIFICATION CHECKLIST READY**
- **Documento Gravado**: `.deploy-control/p0/evidence/p0-prod-kanban-baseline-and-post-verification-checklist.md`
- *Operação 100% Read-Only. Aguardando sinal de Kiro para execução da verificação pós-execução.*
