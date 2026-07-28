# P0 ORQ2 — Monitoramento de Armazenamento e Ciclo de Vida Pós-Upload SharePoint (READ-ONLY)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data/Hora UTC:** `2026-07-28T13:13:10Z`
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1), Agente `A8`
- **Governança:** Issue `ORQ-2` Storage Pressure & Lifecycle Policy Verification
- **Host Auditado**: ORQ2 Local (`ip-172-31-30-9.sa-east-1.compute.internal`)
- **Skill Carregada:** `.agents/skills/aws-secrets-manager/SKILL.md` (zero texto claro, zero `GetSecretValue`).
- **Modo:** MONITORAMENTO READ-ONLY — ZERO EXCLUSÃO, TRUNCAÇÃO OU MUTATIVIDADE. Medição factual de utilização do disco de 60G, partição `/tmp`, pasta de upload SharePoint, maiores consumidores e timers de ciclo de vida.

---

## 1. Medição Factual de Utilização de Armazenamento (`df -h` e `du -sh`)

| Ponto de Montagem / Recurso | Dispositivo / Caminho | Tamanho Total | Espaço Usado | Espaço Disponível | % Utilização Factual | Estado Operacional |
|---|---|---|---|---|---|---|
| **` / ` (Raiz ORQ2)** | `/dev/nvme0n1p1` | `60 GB` (64,344,797,184 bytes) | `52 GB` (55,033,761,792 bytes) | **`8.7 GB`** (9,311,035,392 bytes) | **`86%`** | **`ESTÁVEL / DENTRO DO LIMITE`** |
| **` /tmp ` (Tmpfs)** | `tmpfs` | `7.7 GB` (8,234,008,576 bytes) | `509 MB` (533,393,408 bytes) | **`7.2 GB`** (7,700,615,168 bytes) | **`7%`** | **`EXCELENTE ALÍVIO`** |
| **SharePoint Upload** | `/home/ec2-user/workspace/sharepoint` | N/A | **`573 MB`** (600,178,667 bytes) | N/A | N/A | **`INTEGRO E PRESERVADO`** |

---

## 2. Top Consumidores de Disco no ORQ2 (Apenas Metadados)

| Tamanho Medido | Diretório Consumidor sob `/home/ec2-user` | Conteúdo Primário / Escopo |
|---|---|---|
| **`14 GB`** | `/home/ec2-user/.agent-cred-homes` | Ambientes e slots de credenciais dos agentes (`slots` = 13 GB). |
| **`6.8 GB`** | `/home/ec2-user/workspace` | Repositórios de código, worktrees (3.3 GB), R-D_Agnostic (3.0 GB), SharePoint (573 MB). |
| **`6.1 GB`** | `/home/ec2-user/.cache` | Cache de build e compiladores Go (`go-build-i03` = 1.5 GB). |
| **`5.0 GB`** | `/home/ec2-user/.private-tmp` | Caches temporários isolados com modo `0700`. |
| **`4.6 GB`** | `/home/ec2-user/.local` | Binários e ferramentas instaladas no perfil do usuário (`.local/state` = 2.7 GB). |

*Nota de Privacidade*: Leitura estrita de metadados via `du -h`. Nenhum conteúdo interno de arquivos de código ou credenciais foi lido.

---

## 3. Estado do Timer de Ciclo de Vida (`orq2-agent-cache-lifecycle.timer`)

- **Unidade do Timer**: `orq2-agent-cache-lifecycle.timer` (Ativa `orq2-agent-cache-lifecycle.service`)
- **Última Execução Registrada**: `Tue 2026-07-28 05:16:46 UTC` (Status: `SUCCESS` — há 7h)
- **Próximo Disparo Agendado**: `Wed 2026-07-29 05:28:12 UTC` (em aproximadamente 16 horas)
- **Timers Complementares Ativos**:
  - `reap-cred-slots.timer`: Execução a cada 1h (Última: `12:20:41 UTC`, Próxima: `13:20:41 UTC`).
  - `systemd-tmpfiles-clean.timer`: Execução diária (Última: `2026-07-27 17:10:41 UTC`, Próxima: `2026-07-28 17:10:41 UTC`).

---

## 4. Confirmação Factual de Preservação de Exclusões Explícitas

Todas as exclusões obrigatórias especificadas foram auditadas e confirmadas como **100% PRESERVADAS E INTOCADAS**:

1. **Worktrees de Desenvolvimento**: `/home/ec2-user/workspace/worktrees/*` e `/mnt/shared/*` (Intocados - 3.3 GB).
2. **Diretório de Evidências e Auditoria**: `.deploy-control/p0/evidence/` (Confirmado e ativo).
3. **Backups do Sistema**: Preservados.
4. **Ambientes de Credenciais dos Agentes**: `/home/ec2-user/.agent-cred-homes/slots/*` (Confirmado e ativo).
5. **Diretório de Uploads SharePoint**: `/home/ec2-user/workspace/sharepoint` (Confirmado: `573 MB` totalmente preservados).

---

## 5. Eficiência da Política Diária e Limiares de Alerta (Alert Thresholds)

- **Efetividade da Política Diária**: A combinação do purge automático diário de caches efêmeros com o direcionamento para `$HOME/.private-tmp` reduziu a utilização do `/tmp` tmpfs de 97% para **7%**, garantindo estabilidade contínua.
- **Limiares de Alerta Recomendados para Monitoramento**:
  - **Status Operacional Nominal**: `/` <= 85% \| `/tmp` <= 50%
  - **Alerta de Atenção (Warning)**: `/` > 88% (Espaço livre < 7.0 GB) \| `/tmp` > 75% (Espaço livre < 2.0 GB)
  - **Alerta Crítico de Ação (Critical)**: `/` > 92% (Espaço livre < 4.5 GB) \| `/tmp` > 90% (Espaço livre < 800 MB)

---

## 6. Declaração de FILES_LOCKED

- `multica-auth-work/server/internal/middleware/auth.go` (LOCKED — ORQ-17 Auth Hardening V2)
- `multica-auth-work/server/internal/middleware/auth_test.go` (LOCKED — ORQ-17 Tests)

---

## 7. Veredito Final
- **STATUS: POST-SHAREPOINT STORAGE & LIFECYCLE MONITOR COMPLETED — STABLE**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq2-storage-lifecycle-post-sharepoint-monitor.md`
- *Operação 100% Read-Only. Nenhuma exclusão ou alteração realizada.*
