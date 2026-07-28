# Peer Review Adversarial: Plano de Cutover Aprovado ORQ-17 V3 (GTL-R17)

**auditor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T12:39Z  
**solicitante**: General-Tech-Lead (Codex56-TL w5:pC)  
**governança**: Kanban Issue `ORQ-17`  
**documento revisado**: `.deploy-control/p0/evidence/gtl-orq17-approved-cutover-plan.md` (autor: Opus48#A)  
**modo**: READ-ONLY / AUDITORIA ADVERSARIAL — NENHUMA alteração de código, banco, container, rede ou cert executada  

---

## 1. Veredito Final de Peer Review

### **VEREDITO: BLOCK** ❌

**Resumo da Avaliação**:
O plano de cutover V3 (`gtl-orq17-approved-cutover-plan.md`) demonstra excelente diagnóstico de ambiente (identificando corretamente FQDN `orq1.tail96e2c0.ts.net`, a dependência do toggle de HTTPS no Tailnet Admin Console e a imutabilidade do `AUTH_TOKEN_TTL` via `sync.Once`). No entanto, o veredito é **BLOCK devido a 3 falhas críticas de roteamento, segurança de ACL e CSRF**:

1. **Roteamento Incompleto no `tailscale serve` (Bloqueador de Autenticação)**: O plano propõe mapear apenas `/` -> 13100, `/api` -> 18080 e `/ws` -> 18080. Isso **omite a rota `/auth`** (`/auth/login`, `/auth/google`, `/auth/logout`), `/health` e `/uploads`. Sem mapear `/auth` para a porta `18080`, qualquer tentativa de login no browser será enviada ao Next.js (porta 13100) resultando em **HTTP 404**, impossibilitando a autenticação do usuário.
2. **Ausência de ACL/Grants no Tailnet (Exposição Tailnet-Wide)**: O `tailscale serve` expõe a porta 443 para **todos** os nós/dispositivos do Tailnet (`cloud.labs.brazil@gmail.com`). O plano não especifica a regra de ACL/Grants necessária no painel Tailscale para restringir o acesso à porta 443 estritamente à identidade do owner.
3. **Quebra de CSRF no Modelo de Sliding-Cookie Proposto para Idle 4h**: Em `multica`, o token CSRF é gerado via HMAC a partir do `auth_token` (`cookie.go:141-145`). Reemitir o cookie de auth em voo altera o `auth_token`, invalidando imediatamente o token CSRF do cliente e causando **HTTP 403 Forbidden** em chamadas subsequentes.

---

## 2. Auditoria Detalhada dos 7 Âncoras Obrigatórios

### Âncora 1: Pró-requisito de Certificado HTTPS e Sintaxe do `tailscale serve` (`v1.98.9`)
- **Versão Medida do Tailscale**: `1.98.9` (commit `4fb758c39a`).
- **FQDN Medido**: `orq1.tail96e2c0.ts.net`.
- **Estado de Certificados**: `CertDomains: None` no `tailscale status --json`.
- **Diagnóstico**: O recurso "HTTPS Certificates" está desabilitado no Admin Console do Tailnet (`cloud.labs.brazil@gmail.com`). Certificados IP não são suportados. **Gate Zero Mantido**: Nenhuma execução pode iniciar antes que o Owner habilite HTTPS Certificates no painel Tailscale.
- **Falha de Roteamento Identificada**:
  - `router.go:487-489` define as rotas públicas de autenticação sob `/auth/...`:
    - `POST /auth/login`
    - `POST /auth/google`
    - `POST /auth/logout`
  - A sintaxe corrigida para `tailscale serve` na versão 1.98.9 deve mapear:
    - `tailscale serve --bg --https=443 / http://127.0.0.1:13100` (Next.js Frontend)
    - `tailscale serve --bg --https=443 /api http://127.0.0.1:18080` (Go API Backend)
    - `tailscale serve --bg --https=443 /auth http://127.0.0.1:18080` (Go Auth Backend - **OBRIGATÓRIO**)
    - `tailscale serve --bg --https=443 /ws http://127.0.0.1:18080` (Go Realtime WS)

### Âncora 2: Restrição de Escopo Tailnet-Wide vs Regras de ACL/Grants
- **Diagnóstico de Escopo**: Por padrão, o `tailscale serve` publica a porta 443 para **todo o Tailnet**. Outros agentes ou máquinas na mesma Tailnet conseguem acessar `orq1.tail96e2c0.ts.net:443`.
- **Correção Mínima de ACL**: O plano deve exigir a inclusão da regra no Tailscale Policy (huJSON) antes de ativar o serve:
  ```json
  {
    "acls": [
      {
        "action": "accept",
        "src": ["owner-identity@example.com"],
        "dst": ["orq1:443"]
      }
    ]
  }
  ```

### Âncora 3: Mapeamento de Rotas e Isolamento do Daemon WS (`/api/daemon/ws`)
- **Rotas de Usuário**: `/`, `/auth/...`, `/api/...`, `/ws`.
- **Daemon WebSocket**: `GET /api/daemon/ws` (manipulado por `DaemonWebSocket` em `daemon_ws.go:12`).
- **Análise de Segurança**: O endpoint `/api/daemon/ws` é protegido por `middleware.DaemonAuth` (exigindo tokens de daemon). No entanto, para defesa em profundidade, recomenda-se manter a comunicação de daemons (como o ORQ2) utilizando o túnel SSH local (`multica-orq1-backend-tunnel.service` em `127.0.0.1:18080`), sem expor chamadas de gerenciamento de daemons para clientes de borda se não for necessário.

### Âncora 4: Comportamento do Proxy Reverso (Headers, CORS, WS e Stream Uploads)
- **Trusted Proxies (`MULTICA_TRUSTED_PROXIES`)**: Ao rodar atrás do `tailscale serve`, o proxy conecta via loopback `127.0.0.1`. A variável `MULTICA_TRUSTED_PROXIES=127.0.0.1/32` deve ser configurada para que o middleware de IP (`router.go:70-90`) confie nos cabeçalhos `X-Forwarded-For` e `X-Forwarded-Proto`.
- **CORS (`FRONTEND_ORIGIN`)**: Deve ser configurado para `FRONTEND_ORIGIN=https://orq1.tail96e2c0.ts.net` (com esquema `https://` e sem barra final).
- **WebSocket Upgrade (`/ws`)**: O `tailscale serve` suporta nativamente HTTP Upgrade para WebSockets sem necessidade de diretivas extras.
- **Upload Streaming**: Uploads para `/api/runtimes/.../upload` passam sem limites de tamanho impostos pelo `tailscale serve` (ao contrário do `client_max_body_size` do Nginx).

### Âncora 5: Autenticação Fail-Closed e Comportamento dos Cookies
- **Auth Bypass Interlock**: Em `middleware/auth.go:48-52`, a checagem de bypass verifica se o Host é `localhost` ou IP de loopback. Quando `LOCAL_AUTH_BYPASS=false` e `FRONTEND_ORIGIN=https://orq1.tail96e2c0.ts.net`, o bypass é **desativado imediatamente** de forma fail-closed.
- **Bandeiras de Cookie**: O login real via `POST /auth/login` emite cookies com `HttpOnly`, `Secure` (aceito no HTTPS), e `SameSite=Lax`.

### Âncora 6: TTL de Sessão (Absoluto 24h vs Idle 4h)
- **TTL Absoluto 24h**: Totalmente suportado via `AUTH_TOKEN_TTL=24h` em `cookie.go:74`. Exige recriação/restart do container devido ao `sync.Once`.
- **Idle 4h Expiration (Independente e Prova de Inexistência)**:
  - **Prova**: Nenhuma lógica de renovação/sliding session existe em `internal/auth/` ou `internal/middleware/`. Tokens JWT são 100% stateless (`exp = iat + TTL`).
  - **Falha no Modelo de Sliding-Cookie Proposto**: Reemitir o `auth_token` JWT em voo altera o hash base do token CSRF (`cookie.go:141-145`). O cliente browser passará a enviar requisições mutantes com o token CSRF antigo, resultando em **HTTP 403 Forbidden**.
  - **Decisão Estrita**:
    - **Gate A (Rede e Cutover de Borda)**: Adota estritamente **TTL Absoluto 24h** (`AUTH_TOKEN_TTL=24h`).
    - **Gate B (Remediação de Idle 4h)**: Isolado para uma issue separada, exigindo ajuste na derivação de CSRF, testes de unidade e revisão atenta.

### Âncora 7: Preservação do Túnel e Plano de Rollback do Gate A
- **Preservação do Túnel**: O serviço `multica-orq1-backend-tunnel.service` (ORQ2 -> ORQ1 `127.0.0.1:18080`) **não deve ser alterado ou parado**. A comunicação do daemon ORQ2 continua 100% operacional via loopback SSH.
- **Plano de Rollback Atômico do Gate A**:
  1. Executar `tailscale serve reset` para desativar a borda HTTPS.
  2. Restaurar variáveis de ambiente (`LOCAL_AUTH_BYPASS=true`, remover `FRONTEND_ORIGIN` de borda).
  3. Recriar/reiniciar os containers locais.
  4. Validar que os ouvintes em loopback (`13100` e `18080`) continuam operacionais.
  5. Garantir que **nenhuma porta anônima fique exposta** e nenhuma credencial seja copiada.

---

## 3. Plano Corrigido e Dividido (Gate A vs Gate B)

### **GATE A — Cutover de Borda e Autenticação HTTPS (Sem Alteração de Código)**
- **Pré-Requisito (Owner)**: Habilitar HTTPS Certificates no painel Tailscale (`CertDomains` ativado) e configurar a regra de ACL/Grants no Tailnet para `orq1:443`.
- **Configuração de Borda**:
  ```bash
  tailscale serve --bg --https=443 / http://127.0.0.1:13100
  tailscale serve --bg --https=443 /api http://127.0.0.1:18080
  tailscale serve --bg --https=443 /auth http://127.0.0.1:18080
  tailscale serve --bg --https=443 /ws http://127.0.0.1:18080
  ```
- **Ambiente Backend**: `LOCAL_AUTH_BYPASS=false`, `FRONTEND_ORIGIN=https://orq1.tail96e2c0.ts.net`, `MULTICA_TRUSTED_PROXIES=127.0.0.1/32`, `AUTH_TOKEN_TTL=24h`.

### **GATE B — Expiração por Inatividade (Idle 4h) (Issue Separada Futura)**
- Desenho de renovação de sessão desacoplado do hash CSRF, com testes de integração e suíte de regressão.

---

## 4. Check-out Citing ORQ-17

- **Governança**: `ORQ-17` / `GTL-R17`
- **Artefato Gerado**: `.deploy-control/p0/evidence/gtl-orq17-cutover-v3-peer-review.md`
- **Veredito**: **BLOCK** (requer inclusão da rota `/auth` no `tailscale serve`, especificação de ACL no Tailnet e isolamento do Idle 4h para o Gate B).
- **Status de Mutação**: READ-ONLY. Zero alterações executadas no sistema.
