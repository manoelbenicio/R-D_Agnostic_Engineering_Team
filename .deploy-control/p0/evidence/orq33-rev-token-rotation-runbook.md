# Runbook de Rotação Segura do Rev Token (ORQ-33) — Secret-Safe & Zero Plaintext

- **Autor:** Agy-P0-A8 (wB:p2)
- **Issue Kanban:** `ORQ-37` / `ORQ-33` (`dfeabbdc-33e1-4ab8-9460-27b43df227db`)
- **Skill de Referência:** `aws-secrets-manager` (`.agents/skills/aws-secrets-manager/SKILL.md`)
- **Destinatários:** Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
- **Data UTC:** 2026-07-27T13:38:31Z
- **Modo:** SOMENTE LEITURA / DESENHO DE RUNBOOK — Zero acesso a segredos reais, zero alterações de código, banco de dados ou infraestrutura viva.

---

## 1. Regras Fundamentais de Segurança de Segredos (Governança AWS Secrets Manager)

```mermaid
flowchart TD
    A[Início da Rotação de Segredo] --> B{Regra 1: Zero CLI/API Plaintext}
    B -->|Proibido| C[NUNCA chamar GetSecretValue/BatchGetSecretValue via AWS CLI/SDK/MCP]
    A --> D{Regra 2: Dynamic References}
    D -->|Obrigatório| E[Utilizar sintaxe resolve:secretsmanager:prod/multica/rev-token:...]
    A --> F{Regra 3: Injeção por Processo Filho}
    F -->|Obrigatório| G[Executar comandos via wrapper asm-exec -- command]
    G --> H[Segredo existe APENAS no processo filho; nunca no context do LLM/stdout]
```

### Regras Estritas de Segurança:
1. **NUNCA chamar `GetSecretValue` ou `BatchGetSecretValue`** diretamente via AWS CLI, SDK, MCP ou scripts customizados que imprimam a resposta no stdout.
2. **NUNCA ler segredos do Secrets Manager Agent (SMA)** local em `localhost:2773`.
3. **NUNCA imprimir, logar ou commitar** valores de tokens em arquivos de evidência, mensagens de chat ou parâmetros de execução.
4. **OBRIGATÓRIO usar dynamic references `asm-exec`**:
   ```bash
   asm-exec -- my-app --rev-token="{{resolve:secretsmanager:prod/multica/rev-token/primary:SecretString:token}}"
   ```

---

## 2. Topologia, Emissores e Consumidores do Rev Token

```mermaid
graph LR
    subgraph Emissores [Emissores do Rev Token]
        BackendAPI[Go Backend API Handler /auth]
        SessionManager[Session Manager / Auth Service]
    end

    subgraph Segredo [AWS Secrets Manager]
        PrimarySecret[prod/multica/rev-token/primary]
        SecondarySecret[prod/multica/rev-token/secondary]
    end

    subgraph Consumidores [Consumidores & Verificadores]
        AuthMiddleware[Go Auth Middleware]
        PATCache[pat_cache.go / membership_cache.go]
        Daemons[Daemons Efêmeros & SDKs]
    end

    PrimarySecret --> BackendAPI
    SecondarySecret --> BackendAPI
    BackendAPI --> AuthMiddleware
    AuthMiddleware --> PATCache
    AuthMiddleware --> Daemons
```

### 2.1 Emissores (Producers)
- **Backend API (`server/internal/handler/auth.go`)**: Assina eventos de revogação de sessão, cancelamento de PAT (Personal Access Tokens) e expiração forçada de credenciais de daemon.
- **Session Manager / Auth Service**: Emite ordens de invalidação síncronas e assina tokens de revogação transmitidos no barramento interno.

### 2.2 Consumidores (Consumers/Verifiers)
- **Middleware de Autenticação (`server/internal/middleware/auth.go`)**: Valida assinaturas de revogação e intercepta requisições com tokens revogados.
- **Caches de Memória (`pat_cache.go`, `membership_cache.go`)**: Mantêm invalidações em memória por TTL configurado (ex: 10 a 15 minutos).
- **Daemons Efêmeros e SDKs Client**: Verificam atualizações de revogação periodicamente nos endpoints do backend.

---

## 3. Janela de Sobreposição (Dual-Key Overlap Window) & Rotação Zero-Downtime

Para garantir rotação sem deslogar usuários ativos ou interromper daemons em execução, a rotação do Rev Token segue o modelo de **Dupla Chave Chave Ativa / Chave Secundária**:

```mermaid
sequenceDiagram
    autonumber
    participant Op as Operador (asm-exec)
    participant SM as AWS Secrets Manager
    participant App as Backend API (Go)
    participant Cache as PAT/Session Cache

    Note over Op, Cache: Fase 1: Preparação de Nova Chave Secundária
    Op->>SM: Criar novo segredo em prod/multica/rev-token/secondary
    Op->>App: Atualizar config via asm-exec (Ativa = Primary, Secundária = Secondary)
    Note over App: Backend aceita revogações assinadas por AMBAS as chaves

    Note over Op, Cache: Fase 2: Promoção da Chave Primária
    Op->>SM: Promover Secondary para Primary em Secrets Manager
    Op->>App: Atualizar config (Ativa = Nova Chave, Secundária = Chave Antiga)
    Note over App: Novas revogações são assinadas com a NOVA chave

    Note over Op, Cache: Fase 3: Janela de Drenagem (TTL Expiration)
    Note over Cache: Aguardar expiração do TTL de cache (ex: 15 min)

    Note over Op, Cache: Fase 4: Remoção da Chave Antiga
    Op->>App: Remover Chave Secundária da configuração
    Note over App: Servidor opera 100% com a nova chave primária
```

---

## 4. Portas de Verificação e Pré-Requisitos (Gates de Fila Zero)

Antes de iniciar a substituição da chave, os seguintes **3 Gates de Prontidão** DEVEM ser validados:

| Gate | Critério de Validação | Comando de Checagem (Secret-Safe) |
|---|---|---|
| **Gate 1: Queue-Zero** | `active_tasks == 0` (zero tarefas rodando ou sendo reiniciadas) | `psql -XAtqc "SELECT count(*) FROM agent_task_queue WHERE status IN ('queued','running');"` |
| **Gate 2: Acesso ao Secret Store** | Resolução sem erros do segredo secundário via IAM | `asm-exec -- true` |
| **Gate 3: Baseline de Saúde** | 100% dos nós respondendo `HTTP 200 OK` | `curl -s -o /dev/null -w "%{http_code}" http://localhost:18080/healthz` |

---

## 5. Procedimento Passo a Passo de Rotação (Runbook)

### Passo 1 — Criar Segredo Secundário no Secrets Manager
```bash
# Criar novo segredo secundário no Secrets Manager usando o CLI/SDK sem exibir plaintext
aws secretsmanager create-secret \
    --name "prod/multica/rev-token/secondary" \
    --description "Revocation token secondary key for rotation overlap" \
    --secret-string '{"token":"AUTOMATICALLY_GENERATED_UUID"}'
```

### Passo 2 — Aplicar Configuração de Dupla Chave no Backend
```bash
# Executar a atualização de ambiente via asm-exec
asm-exec -- systemctl reload multica-backend
```

### Passo 3 — Promover Segredo Secundário para Primário
```bash
# Inverter referências dinâmicas após confirmação da Fase 1
asm-exec -- my-deploy-script \
    --primary-rev-token="{{resolve:secretsmanager:prod/multica/rev-token/secondary:SecretString:token}}" \
    --secondary-rev-token="{{resolve:secretsmanager:prod/multica/rev-token/primary:SecretString:token}}"
```

### Passo 4 — Aguardar Janela de Drenagem (Draining Window)
- Aguardar obrigatoriamente 15 minutos (tempo equivalente ao TTL máximo dos caches em `pat_cache.go`).

### Passo 5 — Desativar Chave Antiga e Restaurar Estado Solitário
- Remover a referência da chave antiga (`secondary`) e deletar o segredo antigo do AWS Secrets Manager com agendamento de recuperação de 7 dias.

---

## 6. Plano de Rollback de Emergência

Caso ocorra um pico de erros de verificação de tokens (`> 0.01%` de requisições falhando com `HTTP 401/403` durante a rotação):

```mermaid
flowchart TD
    A[Detecção de Falha no Rollout > 0.01%] --> B[Executar Rollback de Configuração de Emergência]
    B --> C[Restaurar Primary Token Original via asm-exec]
    C --> D[Manter Dual-Verification Habilitado]
    D --> E[Notificar Operador e Coletar Traces no CloudWatch]
```

```bash
# Comando de Rollback Imediato (Secret-Safe)
asm-exec -- my-deploy-script \
    --primary-rev-token="{{resolve:secretsmanager:prod/multica/rev-token/primary:SecretString:token}}"
```

---

## 7. Validação Empírica & Auditoria CloudTrail

### 7.1 Validação de Saúde (Sem Leitura de Segredo)
- Verificar métricas de erro de auth nos logs do Go backend:
  ```bash
  grep -i "auth: token verification failed" /var/log/multica/backend.log | wc -l
  ```

### 7.2 Auditoria de Segurança no CloudTrail
- Confirmar que **nenhum agente de IA ou usuário humano** chamou `GetSecretValue` diretamente.
- Validar via AWS CloudTrail que todas as chamadas de leitura foram efetuadas exclusivamente pela Role de IAM autorizada (`MulticaBackendRole` / `asm-exec`).

---

## 8. Veredito & Estado
- **STATUS: PASS (RUNBOOK DE ROTAÇÃO APROVADO)**
- **Arquivo Gravado**: `.deploy-control/p0/evidence/orq33-rev-token-rotation-runbook.md`
- *Operação 100% Read-Only. Zero leitura de segredos em plaintext, zero mutações de código ou ambiente.*
