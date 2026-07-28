# ORQ-33 — Emenda de Desenho V3: Rotação de Segredo JWT e Remediação Segura (READ-ONLY)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data UTC:** 2026-07-27T15:17:00Z
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Governança:** Issue `ORQ-33`
- **Bases Preservadas:** V1 (`orq33-rev-token-rotation-runbook.md`) e V2 (`orq33-jwt-secret-rotation-runbook-v2.md` & `orq33-jwt-rotation-v2-independent-peer-review.md`)
- **Modo:** SOMENTE LEITURA / EMENDA DE DESENHO ARQUITETURAL V3 — Zero acesso a segredos em texto claro, zero `GetSecretValue`, zero restart de servidor, zero mutação de containers.

---

## 1. Síntese dos 9 Ajustes da Emenda V3

Esta Emenda V3 retifica as lacunas identificadas no parecer adversarial V2, alinhando o desenho com a skill `aws-secrets-manager`, o isolamento estrito de permissões (`0700`/`0600`) e a autorização explícita do Owner.

---

## 2. Detalhamento dos Requisitos da Emenda V3

### 2.1 Ajuste 1: Resolução de Variáveis no Docker Compose via Shell Wrapper e Trap
- **Limitação Técnica do `asm-exec`**: O `asm-exec` substitui referências `{{resolve:...}}` exclusivamente em argumentos de linha de comando (`argv`) e variáveis de ambiente (`env`), **NÃO inspecionando nem modificando o conteúdo de arquivos de configuração no disco**.
- **Solução Arquitetural Aprovada**:
  Geração dinâmica de arquivo de ambiente temporário com permissão `0600` envelopado por `asm-exec` e limpo obrigatoriamente via `trap` no encerramento:
  ```bash
  asm-exec -- sh -c '
    TMP_ENV=$(mktemp "$HOME/.private-tmp/jwt-env.XXXXXX")
    chmod 0600 "$TMP_ENV"
    trap "rm -f $TMP_ENV" EXIT INT TERM

    echo "JWT_SECRET={{resolve:secretsmanager:prod/jwt-secret:SecretString:secret:AWSCURRENT}}" > "$TMP_ENV"
    docker-compose -p multica -f /home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.yml --env-file "$TMP_ENV" up -d
  '
  ```

### 2.2 Ajuste 2: Trava de Rejeição de Referências Literais `{{resolve:` Não-Resolvidas
- **Risco**: Se a resolução falhar ou o placeholder for passado como string não resolvida, o container backend poderia subir utilizando o texto literal `"{{resolve:..."` como chave de assinatura HMAC-SHA256.
- **Verificação de Guarda**:
  ```bash
  RESOLVED_SECRET=$(asm-exec -- env | grep ^JWT_SECRET= | cut -d= -f2-)
  if [ -z "$RESOLVED_SECRET" ] || case "$RESOLVED_SECRET" in "{{"*) true;; *) false;; esac; then
    echo "ERRO FATAL: Referência de segredo não resolvida ou vazia ({{resolve:). Abortando."
    exit 1
  fi
  ```

### 2.3 Ajuste 3: Mapeamento de 4 Caminhos Absolutos de Docker Compose e Projeto Explícito
Para evitar ambiguidade de caminhos relativos e colisões entre ambientes:
- **Projeto Explícito**: `-p multica` em todas as invocações de `docker-compose`.
- **4 Caminhos Absolutos Mapeados**:
  1. `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.yml`
  2. `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.build.yml`
  3. `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.yml`
  4. `/home/ec2-user/workspace/worktrees/gtl-orq26/multica-auth-work/docker-compose.selfhost.yml`

### 2.4 Ajuste 4: Validação de Estado Autenticado na Rota `GET /api/me`
- **Validação de Invalidação de Sessão (Token Antigo)**:
  - Requisição para `GET /api/me` utilizando cookie/header com JWT assinado pela chave antiga DEVE retornar **`HTTP 401 Unauthorized`**.
- **Validação de Emissão de Nova Sessão (Token Novo)**:
  - Requisição para `GET /api/me` utilizando novo JWT emitido pós-recreate DEVE retornar **`HTTP 200 OK`**.

### 2.5 Ajuste 5: Isolamento Estrito de Diretórios (`0700`) e Arquivos (`0600`)
- **Diretórios Temporários**: Todos os arquivos temporários devem residir em `$HOME/.private-tmp` com permissão estrita **`0700` (`drwx------`)**.
- **Arquivos Temporários**: Arquivos contendo variáveis de ambiente resolvidas devem ser criados com `umask 077` e definidos expressamente como **`0600` (`-rw-------`)**.

### 2.6 Ajuste 6: Gate de Tamanho e Conteúdo Não-Vazio do `JWT_SECRET`
- **Verificação Pré-Flight**:
  ```bash
  SECRET_LEN=$(echo -n "$RESOLVED_SECRET" | wc -c)
  if [ "$SECRET_LEN" -lt 32 ]; then
    echo "ERRO FATAL: JWT_SECRET inseguro (tamanho $SECRET_LEN < 32 bytes). Abortando."
    exit 1
  fi
  ```

### 2.7 Ajuste 7: Disparo Correto de Rollback e Preservação de Dados
- **Gatilho de Rollback**: Disparado APENAS se os probes `/healthz`, `/readyz` ou a validação autenticada `GET /api/me` falharem pós-recreate.
- **Preservação de Dados**: O rollback restaura o container apontando para a versão anterior do segredo (`AWSPREVIOUS`) através de `docker-compose -p multica up -d --no-recreate`, **sem apagar volumes do Postgres ou dados de workspaces**.

### 2.8 Ajuste 8: Separação Formal da Autorização de Re-Pair do Daemon no ORQ2
- **Isolamento de Escopo**: O re-pareamento de daemons no host ORQ2 **NÃO FAZ PARTE** da rotação automática do segredo JWT da API.
- **Porta de Governança**: Qualquer re-pareamento ou rotação de token de daemon no ORQ2 exige um card Kanban (`ORQ-N`) separado com autorização prévia e explícita do Owner.

### 2.9 Ajuste 9: Solicitação de Peer Review Independente
- Esta Emenda V3 é submetida para auditoria adversarial independente por um peer reviewer antes de qualquer autorização de execução.

---

## 3. Declaração de FILES_LOCKED

- `multica-auth-work/server/internal/auth/jwt.go` (LOCKED — JWT Config)
- `multica-auth-work/docker-compose.selfhost.yml` (LOCKED — Compose Config)

---

## 4. Veredito Final
- **STATUS: PASS (EMENDA DE DESENHO V3 CONCLUÍDA E ENVIADA PARA PEER REVIEW)**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq33-jwt-rotation-remediation-v3-design.md`
- *Operação 100% Read-Only. Nenhuma leitura de segredo, nenhuma mutação de servidor ou container.*
