# ORQ-42 — Proposta de Runbook V4: Rotação do Segredo JWT e Remediação Definitiva (READ-ONLY)

- **Status:** **PROPOSAL** (Submetido para Revisão Independente)
- **Card:** **ORQ-42**
- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data UTC:** 2026-07-27T15:39:00Z
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Substitui:** Versão V3 (`orq42-jwt-v3-adversarial-peer-review.md`)
- **Skill Carregada:** `.agents/skills/aws-secrets-manager/SKILL.md` (zero texto claro, zero `GetSecretValue`, resolução via `asm-exec` em processo filho).
- **Modo:** SOMENTE LEITURA / PROPOSTA V4 — Zero leitura de segredo em texto claro, zero `asm-exec` executado, zero mutação em Docker/AWS/containers/quadro, zero login ou restart.

---

## 1. Síntese das 13 Retificações da Versão V4

A presente proposta V4 corrige integralmente todas as ressalvas apontadas no parecer adversarial V3, estabelecendo o procedimento definitivo e seguro para a rotação da chave JWT do backend.

---

## 2. Detalhamento Técnico da Proposta V4

### 2.1 Sintaxe Atualizada do Docker Compose (Plugin)
- **Sintaxe Obrigatória**: Uso do plugin moderno `docker compose` (separado por espaço), substituindo o comando antigo hifenizado `docker-compose`.

### 2.2 Caminhos Absolutos de Configuração no Host ORQ1
Mapeados diretamente das labels dos containers em execução no host ORQ1:
- Arquivo Principal: `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.yml`
- Arquivo de Build: `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.build.yml`

### 2.3 Nome Explícito do Projeto Docker Compose
- **Projeto**: `-p multica-dev-transition` em todas as invocações de `docker compose`.

### 2.4 Estratégia de Re-criação Estrita do Backend (`--force-recreate --no-deps backend`)
- Tanto a implantação quanto o rollback recriam **EXCLUSIVAMENTE** o container do serviço `backend`:
  ```bash
  docker compose -p multica-dev-transition -f /home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.yml up -d --force-recreate --no-deps backend
  ```
- Containers de Postgres, Redis e Web permanecem 100% intocados durante todo o processo.

### 2.5 Zero Materialização de Texto Claro na Shell Pai
- O segredo nunca é exportado ou lido na shell pai.
- A resolução ocorre **estritamente dentro da memória do processo filho** gerado por `asm-exec`.

### 2.6 Guarda de Inspeção Baseada em Efeito (Anti-Placeholder)
No processo filho, valida-se que o segredo resolvido não contém o texto literal `"{{resolve:"` nem está vazio:
```bash
if [ -z "$JWT_SECRET" ] || case "$JWT_SECRET" in "{{"*) true;; *) false;; esac; then
  echo "ERRO FATAL: Referência de segredo não resolvida ({{resolve:). Abortando."
  exit 1
fi
```

### 2.7 Checagem de Tamanho com `printf %s` no Processo Filho
- Cálculo exato sem inclusão de caracteres de quebra de linha:
```bash
SECRET_LEN=$(printf %s "$JWT_SECRET" | wc -c)
if [ "$SECRET_LEN" -lt 32 ]; then
  echo "ERRO FATAL: JWT_SECRET menor que 32 bytes (tamanho $SECRET_LEN). Abortando."
  exit 1
fi
```

### 2.8 Isolamento de Diretório Privado `0700`, `umask 077` e `trap` Aspas Simples
- Criação e validação do diretório temporário privado:
  ```bash
  mkdir -p -m 0700 "$HOME/.private-tmp"
  chmod 0700 "$HOME/.private-tmp"
  ```
- Criação do arquivo temporário com `umask 077` e remoção garantida por `trap`:
  ```bash
  TMP_ENV=$(umask 077 && mktemp "$HOME/.private-tmp/jwt-env.XXXXXX")
  chmod 0600 "$TMP_ENV"
  trap 'rm -f "$TMP_ENV"' EXIT INT TERM
  ```

### 2.9 Preservação do Contrato de Interpolação
- O processo filho grava a variável no arquivo env temporário `0600`:
  `echo "JWT_SECRET=${JWT_SECRET}" > "$TMP_ENV"`
- O arquivo é passado ao Compose via `--env-file "$TMP_ENV"`.

### 2.10 Probes de Saúde do Backend e Invalidação em `/api/me`
- **Probe Desautenticado**: `curl -sf http://localhost:8080/health` DEVE retornar `HTTP 200 OK`.
- **Probe Autenticado com Token Antigo**: `curl -sf -b "session=TOKEN_ANTIGO" http://localhost:8080/api/me` DEVE retornar `HTTP 401 Unauthorized`.
- **Probe Autenticado com Token Novo**: `curl -sf -b "session=TOKEN_NOVO" http://localhost:8080/api/me` DEVE retornar `HTTP 200 OK`.

### 2.11 Verificação da Existência do Segredo na AWS (Apenas Metadados)
- Consulta de metadados via AWS CLI sem ler o valor em texto claro:
  ```bash
  aws secretsmanager describe-secret --secret-id prod/jwt-secret --region sa-east-1
  ```

### 2.12 Re-pareamento do Daemon no Host ORQ2 como Pré-requisito da Mesma Janela
- O re-pareamento de daemons no host ORQ2 deve ser executado **na mesma janela operacional** para evitar que daemons fiquem desautenticados pós-rotação.
- **Porta de Governança**: Exige um card Kanban (`ORQ-N`) separado com autorização formal do Owner.

### 2.13 Declaração de `FILES_LOCKED`
- `multica-auth-work/server/internal/auth/jwt.go` (LOCKED — JWT Config)
- `multica-auth-work/docker-compose.selfhost.yml` (LOCKED — Compose Config)

---

## 3. Veredito Final
- **STATUS: PROPOSAL (PROPOSTA V4 CONCLUÍDA E SUBMETIDA PARA PEER REVIEW)**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq42-jwt-rotation-runbook-v4-design.md`
- *Operação 100% Read-Only. Nenhuma leitura de segredo, nenhuma mutação no Docker, AWS, containers ou quadro.*
