# AUDITORIA MECÂNICA DO LIFECYCLE DIÁRIO DO ORQ2 (PÓS-RCA AGY)

- **Autor:** Agy-P0-A8 (wB:p2)
- **Destinatários:** Codex56-TL (General-TL) & Owner
- **Data UTC:** 2026-07-27T18:05:59Z
- **Modo:** READ-ONLY Estrito (Zero remoções, zero alteração de timers ou scripts, zero mutação no banco).

---

## 1. Mapeamento de Timers, Serviços e Scripts Ativos

| Serviço / Timer | Status / Frequência | Última Execução | Próxima Execução | Script / Binário |
|---|---|---|---|---|
| `orq2-agent-cache-lifecycle.timer` | Ativo (`04:15:00 UTC` diário) | 2026-07-27 16:47:38 UTC | 2026-07-28 05:15:51 UTC | `/usr/local/libexec/orq2-agent-cache-lifecycle --real` |
| `reap-cred-slots.timer` | Ativo (Horário) | 2026-07-27 17:12:47 UTC | +6 minutos | `/home/ec2-user/.local/bin/reap-cred-slots.sh` (TTL 24h = 1440 min) |
| `logrotate.timer` | Ativo (Diário) | 18h atrás | +5h 53min | `/usr/sbin/logrotate` |
| `systemd-tmpfiles-clean.timer` | Ativo (Diário) | 55min atrás | +23h | `systemd-tmpfiles` |

---

## 2. Diagnóstico de Exclusão Efetiva de Slots & Descoberta de Causa Raiz (RCA)

### A. Diagnóstico de Limpeza de Slots (`.agent-cred-homes/slots/`)
- **Limite de Idade Configurado:** 24 horas (`CRED_SLOT_TTL_MIN=1440`).
- **Comportamento Medido:** Foram encontrados 27 diretórios em `.agent-cred-homes/slots/`, incluindo diretórios antigos de 25/07 (`slot-104`, `slot-105`, `slot-120`, etc.).
- **Causa Raiz Medida nos Logs do Journal (`Permission denied`):**
  - O script `reap-cred-slots.sh` tenta remover slots antigos via `rm -rf -- "$s"`.
  - O Go marca os arquivos de módulo baixados em `$GOPATH/pkg/mod` como **somente leitura (`0444` / `-r--r--r--`)**.
  - Quando o `rm -rf` tenta apagar a árvore sem permissão de escrita explícita nos arquivos/diretórios de módulo, o sistema operacional retorna `Permission denied`.
  - **Recomendação Idempotente:** Atualizar a instrução do script para executar `chmod -R +w "$s" 2>/dev/null` antes do `rm -rf` (sem executar agora).

---

## 3. Investigação do Achado sobre o `slot150`

- **Alegação Inicial:** Hipótese de que o `slot150` provavelmente nunca foi materializado.
- **Medição Direta no Nó:**
  - O diretório `/home/ec2-user/.agent-cred-homes/slots/slot-150` **EXISTE e ESTÁ MATERIALIZADO** (`drwx------. 8 ec2-user ec2-user 99 Jul 27 17:58`).
  - O `slot-150` é o slot ativo da sessão de execução do agente `Agy-P0-A8` (`HOME=/home/ec2-user/.agent-cred-homes/slots/slot-150/home`).
  - O script `reap-cred-slots.sh` detectou o processo ativo via `/proc/$pid/environ` e protegeu corretamente o `slot-150`.
- **Conclusão Factual:** A hipótese de não-materialização do `slot150` foi **REFUTADA POR MEDIÇÃO DIRETA**. O slot foi materializado e está em uso ativo protegido.

---

## 4. Estado de Disco (`df -h`) e Limpeza de Artefatos

- **Espaço em Disco:** 13 GiB livres em `/` (48 GiB usados de 60 GiB - 80% utilizado).
- **Worktrees (`workspace/worktrees/`):** 16 worktrees mantidos isoladamente por agente.
- **Evidências (`.deploy-control/p0/evidence/`):** 3.8 MB em arquivos de auditoria preservados.

---

## 5. Checklist Pós-Run de Segurança

- [x] NENHUM arquivo ou diretório foi apagado ou alterado durante a auditoria (100% READ-ONLY).
- [x] NENHUM timer do systemd foi desativado ou modificado.
- [x] Causa raiz do `Permission denied` nos slots antigos identificada no journal (arquivos read-only do Go module cache).
- [x] Status do `slot-150` verificado e confirmado como materializado e ativo.
