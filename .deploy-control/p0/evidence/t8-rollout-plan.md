# Plano de Rollout T8 v4 — Backend (ORQ1) + Daemon (ORQ2) [REVISÃO FINAL v4]

- **Autor:** Agy-P0-A8 (wB:p2)
- **Destinatários:** Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
- **Data UTC:** 2026-07-27T10:20:33Z
- **Modo:** SOMENTE LEITURA / PREPARAR (Nenhum build, deploy, restart ou edição executada)

---

## 1. Separação Estrita: Código Já Live vs Novos Patches T8

| Componente | Estado Já Live em Produção | NOVO Patch T8 a Ser Aplicado |
| :--- | :--- | :--- |
| **Daemon (ORQ2)** | Daemon PID `3240496` na unit de usuário `multica-daemon-orq2-credential.service` | Patch `execenv` token-only (`execenv.go` pass-through) |
| **Backend API (ORQ1)** | Container Docker em execução conectado via túnel porta `18080` | Patch em `task.go` para atualização atômica de status + emissão de eventos de issue |
| **Túnel (ORQ2)** | `multica-orq1-backend-tunnel.service` ativo | Mantido inalterado (`http://127.0.0.1:18080`) |
| **Frontend UI** | Bundle estático de produção ativo | Mantido 100% intocado (Caminho de rollback preservado) |

---

## 2. Runtimes Mandatórios, Allowlist de Slots e Mapeamento

1. **Escopo Estrito:** 3 Runtimes Mandatórios (`antigravity`/agy, `codex`, `kiro`). OpenCode desconsiderado.
2. **Descoberta do AGY:** Descoberta NATIVA via invocação do CLI `antigravity` no `execenv` (NÃO via consulta HTTP ao OmniRoute).
3. **Kiro Allowlist na Unit:** A unit de usuário atual possui allowlist ativa restrita aos 4 slots: **`139`**, **`140`**, **`143`**, **`149`** (`slot-142` simplesmente não está na allowlist da unit atual).
4. **Mapeamento de Subdiretórios:**
   - `antigravity` -> `<slot_path>/home` (em todos os 22 slots).
   - `kiro` -> `<slot_path>/xdg-data` (restrito aos 4 slots da allowlist).
   - `codex` -> `<slot_path>/codex`.

---

## 3. Staging Atômico de Binário e Comandos Systemd de Usuário

### Staging Atômico do Binário no ORQ2
- A atualização do binário não deve fazer renomeação/sobrescrita direta em uso. O processo de staging atômico exige:
  1. Compilar para arquivo de staging temporário: `/home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1.tmp`.
  2. Ajustar permissões `chmod 0755`.
  3. Realizar `mv` atômico substituindo o alvo `/home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1`.

### Gerenciamento da Unit via Escopo `--user`
Como `multica-daemon-orq2-credential.service` é uma unit systemd de usuário, **todos** os comandos devem utilizar `--user`:
- `systemctl --user daemon-reload`
- `systemctl --user restart multica-daemon-orq2-credential.service`
- `systemctl --user status multica-daemon-orq2-credential.service`

---

## 4. Endpoints Reais de Healthcheck & Validação

- **Endpoint Real de Runtimes:** `GET http://127.0.0.1:18080/api/runtimes?workspace_id=<WORKSPACE_ID>`
  - *Critério:* Status `ONLINE` reportado para os 3 runtimes mandatórios (`antigravity`, `codex`, `kiro`).
- **Validação de Tasks Atômicas (Backend ORQ1):** Testar requisições em `task.go` confirmando atualização atômica de status e disparo de eventos de issue no PostgreSQL.

---

## 5. Estratégia de Cutover e Artefatos de Rollback Duráveis

### Cutover Backend (ORQ1)
- **Artefato:** Nova imagem Docker contendo o patch `task.go`.
- **Ação:** Rebuild da imagem Docker e restart do container no ORQ1.
- **Rollback Durável:** Reverter tag/digest da imagem do container Docker no ORQ1 para a versão estável anterior.

### Cutover Daemon (ORQ2)
- **Artefato:** Binário com staging atômico em `/home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1`.
- **Ação:** Substituição atômica e execução de `systemctl --user daemon-reload && systemctl --user restart multica-daemon-orq2-credential.service`.
- **Rollback Durável:** Apontar a unit ou realizar `mv` atômico a partir dos arquivos de binário durável pré-existentes no ORQ2 (identificados por tamanho de arquivo em bytes):
  - `/home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1.pre-agy-fix` (tamanho do arquivo: 21.333.562 bytes)
  - ou `/home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1.previous` (tamanho do arquivo: 21.333.458 bytes)
  - Executar: `systemctl --user daemon-reload && systemctl --user restart multica-daemon-orq2-credential.service`.

---

## 6. Matriz de Decisões Exigidas do Owner (STOP-AND-WAIT §0.1)

| # | Ação Proposta | Host / Alvo | Classificação | Motivo do Bloqueio |
| :--- | :--- | :--- | :--- | :--- |
| **D1** | Compilar binário daemon no ORQ2 e imagem Docker no ORQ1 | ORQ1 / ORQ2 | **STOP-AND-WAIT** | Rebuild de binário/imagem (§0.1) |
| **D2** | Reiniciar container Docker do Backend com patch `task.go` | ORQ1 | **STOP-AND-WAIT** | Alteração/restart de container de produção (§0.1) |
| **D3** | Substituir binário do daemon e executar `systemctl --user restart multica-daemon-orq2-credential.service` | ORQ2 | **STOP-AND-WAIT** | Alteração de serviço/systemd de usuário no ORQ2 (§0.1) |
