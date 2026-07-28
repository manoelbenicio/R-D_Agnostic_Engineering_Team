# Análise de Lacunas de Infraestrutura — TEMA C

- **Tema:** TEMA C — Dependências de Infraestrutura (Redis, Task Queue e Container Backend vs CLIs)
- **Autor:** Agy-P0-A8 (wB:p2)
- **Data UTC:** 2026-07-26T22:48:10Z
- **Modo:** Somente Leitura e Inspeção de Código/Arquitetura

---

## 1. Mapeamento de Dependências e Lacunas Identificadas

### Lacuna 1: Ausência de Container Redis no ORQ1
- **O que falta:** Instância/container Redis para centralizar estado de stores voláteis e rate-limiters.
- **Evidência no Código:**
  - `multica-auth-work/server/cmd/server/router.go:192-200`: `if rdb != nil { h.ModelListStore = handler.NewRedisModelListStore(rdb)... }`.
  - `multica-auth-work/server/internal/handler/handler.go:211`: `ModelListStore: NewInMemoryModelListStore()`.
- **Análise & Fallback:** Existe fallback 100% funcional em memória (`NewInMemoryModelListStore()`, `NewInMemoryUpdateStore()`, `NewInMemoryLivenessStore()`) quando `rdb == nil`.
- **Impacto:** **DEGRADA (Não Bloqueia).** Em ambiente single-node (ORQ1), o fallback em memória atende totalmente. Em cluster multi-node, o cache não é compartilhado entre réplicas da API.

---

### Lacuna 2: Fila de Tarefas (`agent_task_queue`) e Autopilot
- **O que falta:** Nada em termos de Redis.
- **Evidência no Código:**
  - `multica-auth-work/server/internal/handler/agent_test.go:71`: `INSERT INTO agent_task_queue...`.
  - `multica-auth-work/server/internal/daemon/daemon.go:2842`: Daemon consulta e atualiza `agent_task_queue` diretamente via SQL no PostgreSQL.
- **Análise:** A fila de tarefas, transição de estado e o Autopilot são 100% persistidos no **PostgreSQL**. O Redis não é utilizado nem exigido para o ciclo de vida das tasks.
- **Impacto:** **NENHUM (Operação Normal).** Tarefas executam e progridem normalmente no banco de dados.

---

### Lacuna 3: CLIs Ausentes no Container do Backend (`docker exec`)
- **O que falta:** Binários CLI (`codex`, `kiro-cli`, etc.) não estão presentes dentro da imagem do container `backend`.
- **Evidência no Código:**
  - `multica-auth-work/server/internal/daemon/daemon.go:3445`: O processador de tarefas é o **Daemon do Multica** (processo PID 3244391 que roda nativamente no host ORQ1, fora do container backend).
  - `multica-auth-work/server/internal/daemon/brain_integration.go:395`: O Daemon invoca os CLIs diretamente no ambiente do host (`execenv`).
- **Análise:** Por arquitetura, o container `backend` serve apenas como API HTTP/Control Plane. A execução física de CLIs ocorre no **Daemon host**.
- **Impacto:** **NENHUM (Comportamento por Design).** O container backend não precisa conter os CLIs.

---

### Lacuna 4: Signal Wakeup vs Polling no Daemon
- **O que falta:** Pub/sub via Redis para notificação em tempo real de novas tarefas criadas.
- **Evidência no Código:**
  - `multica-auth-work/server/cmd/server/router.go:186-190`: `h.TaskService.Wakeup = opts.DaemonWakeup`.
- **Análise:** Sem Redis Pub/Sub, o aviso de nova tarefa depende de canal local ou polling periódico.
- **Impacto:** **DEGRADA.** Pequeno aumento de latência na inicialização da tarefa (delay de polling no daemon).

---

## 2. Quadro Resumo de Impacto

| Componente | Dependência | Fallback Existente | Status | Severidade |
| :--- | :--- | :--- | :--- | :--- |
| **ModelListStore** | Redis | `InMemoryModelListStore` | Funcional | Nenhuma (Single-node) |
| **agent_task_queue** | PostgreSQL | N/A (Nativo Postgres) | Funcional | Nenhuma |
| **Autopilot** | PostgreSQL | N/A (Nativo Postgres) | Funcional | Nenhuma |
| **Execução de CLI** | Daemon no Host | N/A (Executa no Host) | Funcional | Nenhuma |
| **Rate Limiting / Wakeup** | Redis | Fallback em memória / Polling | Degrada | Baixa (Degrada latência/rate limit) |
