# Peer Review Adversarial: Runbook de Rotação do Rev Token ORQ-33 (GTL-87)

**auditor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T13:41Z  
**governança**: Kanban Issue `ORQ-33` (UUID `dfeabbdc-33e1-4ab8-9460-27b43df227db`)  
**skill ativada**: `aws-secrets-manager` (`.agents/skills/aws-secrets-manager/SKILL.md`)  
**documento revisado**: `.deploy-control/p0/evidence/orq33-rev-token-rotation-runbook.md` (autor: Agy-P0-A8)  
**modo**: READ-ONLY / AUDITORIA ADVERSARIAL — NENHUM acesso a segredos em plaintext, zero chamadas a `GetSecretValue`, zero mutações de código ou infraestrutura  

---

## 1. Veredito Final de Peer Review

### **VEREDITO: BLOCK** ❌

**Resumo da Avaliação**:
O runbook `orq33-rev-token-rotation-runbook.md` segue com rigor exemplar as diretrizes de segurança da skill `aws-secrets-manager` (utilizando `asm-exec`, dynamic references `{{resolve:secretsmanager:...}}` e proibição total de leitura de segredos em plaintext). No entanto, o veredito é **BLOCK por 3 falhas de premissa de código e incompletude de gates**:

1. **Premissa Falsa de Suporte a Dupla Chave (Dual-Token Overlap Window)**: O runbook assume que o backend Go suporta uma janela de sobreposição com chave primária e secundária (`primary` e `secondary`). Na realidade do código (`server/internal/auth/jwt.go`), o backend suporta exclusivamente uma **única chave** `JWT_SECRET`, carregada via `sync.Once`. Não existe qualquer código de validação dual implementado. Tentar aplicar a rotação com dupla chave no backend atual resultará na ignoração dos parâmetros ou em indisponibilidade imediata de tokens.
2. **Imutabilidade por `sync.Once` no Backend**: Em `jwt.go:33-40`, o `JWT_SECRET` é lido das variáveis de ambiente uma única vez via `sync.Once`. Executar `systemctl reload` ou mudar o secret sem recriar o container Go **não atualiza o segredo em memória**.
3. **Incompletude do Gate 1 (Queue-Zero)**: O Gate 1 do runbook verifica apenas os status `'queued'` e `'running'`. A migration `109_agent_task_waiting_local_directory.up.sql` e a arquitetura do sistema definem 4 estados ativos: `queued`, `dispatched`, `running` e `waiting_local_directory`. O Gate 1 omitiu 2 estados ativos.

---

## 2. Auditoria Detalhada dos Requisitos de Segurança e Código

### 2.1 Conformidade com a Skill `aws-secrets-manager`
- **Uso de `asm-exec`**: Todos os comandos propostos utilizam o wrapper `asm-exec -- command`, garantindo que os segredos existam apenas nos processos filhos.
- **Zero Plaintext / Zero Logs**: Nenhum valor de segredo ou token em formato plaintext é exibido, logado ou armazenado nos artefatos.
- **Proibição de `GetSecretValue`**: Respeitada integralmente. A verificação do Secrets Manager ocorre por referências dinâmicas `{{resolve:secretsmanager:prod/multica/...}}`.

### 2.2 Código Implementado vs Premissas Não-Implementadas do Runbook

| Recurso de Rotação | Declarado no Runbook | Realidade do Código Go (`server/internal/auth/`) | Status da Premissa |
|---|---|---|---|
| **Dupla Chave (Primary/Secondary)** | Exige validação simultânea por chave antiga e nova | **NÃO IMPLEMENTADO**. `jwt.go` aceita apenas uma chave única `JWT_SECRET`. | ❌ **PREMISSA FALSA** |
| **Reload Dinâmico de Secret** | Supõe atualização de segredo via `systemctl reload` | **NÃO IMPLEMENTADO**. `JWTSecret()` usa `sync.Once` (`jwt.go:35`), exigindo restart do container. | ❌ **PREMISSA FALSA** |
| **Cache Draining Window (15 min)** | Supõe drenagem gradual baseada em TTL | `pat_cache.go` usa TTL de 10 min e `membership_cache.go` usa 5 min. Como a chave é única, a troca desvalida imediatamente todos os JWTs. | 🟡 RESSALVA |

### 2.3 Auditoria de Invariantes de Fila (Queue-Zero Gate)
- **Status do Gate 1 no Runbook**:
  `SELECT count(*) FROM agent_task_queue WHERE status IN ('queued','running');`
- **Correção Estrita Obrigatória**:
  O predicado correto de tarefas ativas deve cobrir os 4 estados:
  ```sql
  SELECT count(*) FROM agent_task_queue 
  WHERE status IN ('queued', 'dispatched', 'running', 'waiting_local_directory');
  ```

### 2.4 Riscos de Skew de Relógio e Replay
- Sem o suporte a validação por dupla chave no código Go, qualquer rotação do `JWT_SECRET` invalida instantaneamente todos os tokens ativos.
- O plano de rollback deve declarar explicitamente que a rotação no código atual é um **evento disruptivo de sessão** (logout de usuários e necessidade de re-emparelhamento de daemons) até que a RFC de suporte a Dual-Signing Key seja implementada no repositório Go.

---

## 3. Correções Mínimas Obrigatórias para Aprovação (PASS-Readiness)

1. **Atualizar a Premissa Técnica no Runbook**: Declarar explicitamente que o código Go atual utiliza **chave única `JWT_SECRET` com recarga via recriação de container (`sync.Once`)**, e que a rotação por dupla chave (`primary`/`secondary`) exige a implementação prévia da PR de Dual-Signing Key no backend.
2. **Corrigir a Query do Gate 1 (Queue-Zero)**: Incluir os estados `dispatched` e `waiting_local_directory` no filtro SQL.
3. **Formalizar o Procedimento de Restart**: Substituir `systemctl reload` por recriação controlada do container (`docker restart` / `systemctl restart`) para forçar o reinício da primitiva `sync.Once`.

---

## 4. Check-out Citing ORQ-33

- **Governança**: `ORQ-33` (UUID `dfeabbdc-33e1-4ab8-9460-27b43df227db`)
- **Skill Utilizada**: `aws-secrets-manager`
- **Artefato Gerado**: `.deploy-control/p0/evidence/gtl-orq33-rev-token-rotation-peer-review.md`
- **Veredito**: **BLOCK** ❌ (Aguardando atualização do runbook para alinhar o procedimento com o código Go de chave única e corrigir o Gate 1 de fila zero).
- **Status de Segredo**: ZERO valores em plaintext lidos, exibidos ou armazenados.
