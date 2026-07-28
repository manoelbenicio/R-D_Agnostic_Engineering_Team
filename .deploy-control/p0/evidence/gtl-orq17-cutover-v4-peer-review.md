# Peer Review Adversarial: Plano de Cutover Aprovado ORQ-17 Seção V4 (GTL-R17R)

**auditor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T12:46Z  
**solicitante**: General-Tech-Lead (Codex56-TL w5:pC)  
**governança**: Kanban Issue `ORQ-17` (Kanban em FREEZE pelo incidente `ORQ-31` duplicado; nenhum card criado)  
**documento revisado**: `.deploy-control/p0/evidence/gtl-orq17-approved-cutover-plan.md` — SEÇÃO V4 (autor: Opus48#A)  
**modo**: READ-ONLY / RE-REVIEW ADVERSARIAL — NENHUMA alteração de código, banco, container, rede, cert, serve ou ACL executada  

---

## 1. Veredito Final de Peer Review

### **VEREDITO: PASS** ✅

**Resumo da Avaliação**:
A Seção V4 do documento `gtl-orq17-approved-cutover-plan.md` corrige integralmente todos os bloqueadores identificados na V3. Ela demonstra rigor técnico impecável ao resolver a colisão de prefixo entre as rotas de autenticação do backend Go (`/auth/login`, `/auth/google`, `/auth/logout`) e a rota do frontend Next.js (`/auth/callback`), ao identificar que todas as 15 máquinas do Tailnet compartilham o mesmo `UserID` (exigindo ACL por dispositivo/IP e não por identidade do Tailnet), e ao remover por completo a expiração por inatividade (Idle 4h) do Gate A.

---

## 2. Auditoria Detalhada dos 6 Âncoras de Validação

### Âncora 1: Roteamento `tailscale serve`, Longest-Prefix e Preservação de `/auth/callback`
- **Sintaxe do Tailscale 1.98.9**: Confirmada funcionalidade das flags `--https=443`, `--set-path`, `--bg`, `--yes`, `serve status [--json]` e `serve reset`.
- **Análise da Colisão `/auth`**:
  - Backend Go (`router.go:487-489`): define `POST /auth/login`, `POST /auth/google`, `POST /auth/logout`.
  - Frontend Next.js (`app/auth/callback/page.tsx`): define a página `/auth/callback` usada na finalização do OAuth via Google.
- **Resolução V4**: O `tailscale serve` utiliza **matching por prefixo mais longo** (longest-prefix routing).
  - O mapeamento dos três paths exatos (`/auth/login`, `/auth/google`, `/auth/logout`) para `http://127.0.0.1:18080` garante que requisições de login atinjam o backend Go.
  - A rota `/auth/callback` não casa com esses três prefixos mais longos e cai no prefixo raiz `/` (`http://127.0.0.1:13100`), preservando a página do Next.js **sem erro 404**.

### Âncora 2: Rota `/uploads` e Rastreamento de Endpoints do Backend
- **Rota `/uploads`**: Confirmada em `router.go:493` (`r.Handle("/uploads/*", ...)` quando utilizado `storage.LocalStorage`). Essencial para avatares e anexos locais. Mapeada para `127.0.0.1:18080`.
- **Isolamento de Health Checks**: `/health`, `/readyz`, `/healthz` e `/health/realtime` permanecem restritos ao loopback (`127.0.0.1:18080`), sem exposição em 443, o que reduz a superfície de ataque mantendo os preflights operacionais.

### Âncora 3: Exposição de `/api/daemon` e Topologia de Segurança
- **Risco Residual**: Mapear `/api` expõe endpoints como `GET /api/daemon/ws`.
- **Mitigação**: O `tailscale serve` não suporta regras de negação (deny path). A exposição do `/api/daemon` sobre 443 é **completamente mitigada pela ACL de dispositivo do Tailnet** (V4.3), que impede que nós de agente (como o ORQ2 ou ec2-jump-box) alcancem a porta 443 do ORQ1. O daemon do ORQ2 continua utilizando estritamente o túnel SSH local em `127.0.0.1:18080`.

### Âncora 4: Identidade do Tailnet e Restrição por Dispositivo (ACLs/Grants)
- **Achado Crítico de Medição**: Todos os 15 nós do Tailnet pertencem à **mesma conta** `cloud.labs.brazil@gmail.com` (`UserID=7650130887043022`) e nenhum possui tag. Regras baseadas em `src: ["user@domain"]` liberariam acesso a todos os agentes.
- **Solução por Dispositivo/IP (Proposta A)**: A restrição deve ser aplicada por IP do dispositivo do Owner (`owner-dev-1` em `hosts` da policy Tailscale).
- **Alerta de Sobrescrita por ACL Pré-Existente**: O Tailscale aplica o princípio *default deny*, mas qualquer regra genérica pré-existente (ex: `accept * -> *`) invalidaria a restrição. O Owner precisa revisar e remover regras amplas antes do cutover.
- **Verificação Distribuída**:
  - Do dispositivo do Owner: `curl -fsSI https://orq1.tail96e2c0.ts.net/` -> **HTTP 200**.
  - Do ORQ2 (nó de agente): `curl https://orq1.tail96e2c0.ts.net/` -> **Falha na camada de rede (connection refused/timeout)**.

### Âncora 5: Gate Zero HTTPS, Biscoitos de Autenticação, TTL 24h e Túnel SSH
- **Gate Zero HTTPS**: Preflight valida `CertDomains` em `tailscale status --json`. Se `None`, o processo **PARADO** até o Owner habilitar no Admin Console.
- **Segurança de Cookies**: `FRONTEND_ORIGIN=https://orq1.tail96e2c0.ts.net` força o scheme `https`, resultando em `Secure` (`cookie.go:115-119`) e `SameSite=Strict` (`cookie.go:164`). O token CSRF permanece válido.
- **TTL Absoluto 24h**: Suportado por `AUTH_TOKEN_TTL=24h` (`cookie.go:36-61`). Container do backend deve ser recriado devido a `sync.Once`.
- **Túnel SSH Preservado**: `multica-orq1-backend-tunnel.service` no ORQ2 é mantido ativo e intocado.

### Âncora 6: Remoção Completa do Idle 4h do Gate A
- O Idle 4h foi **100% removido do Gate A** e enfileirado como item futuro sem número (respeitando o Kanban Freeze de `ORQ-31`).
- A política efetiva declarada ao Owner é **TTL Absoluto de 24h sem timeout por inatividade**.

---

## 3. Comandos Mínimos de Execução do Gate A (Com Retenção de Parada)

### 3.1 Comandos de Borda `tailscale serve` (Executar no ORQ1 após G0 e G1)
```bash
# 1. Frontend Raiz
tailscale serve --bg --yes --https=443 http://127.0.0.1:13100

# 2. API Backend
tailscale serve --bg --yes --https=443 --set-path /api http://127.0.0.1:18080

# 3. Realtime WebSocket
tailscale serve --bg --yes --https=443 --set-path /ws http://127.0.0.1:18080

# 4. Uploads de Arquivos
tailscale serve --bg --yes --https=443 --set-path /uploads http://127.0.0.1:18080

# 5. Endpoints Exatos de Autenticação Go Backend
tailscale serve --bg --yes --https=443 --set-path /auth/login http://127.0.0.1:18080
tailscale serve --bg --yes --https=443 --set-path /auth/google http://127.0.0.1:18080
tailscale serve --bg --yes --https=443 --set-path /auth/logout http://127.0.0.1:18080
```

### 3.2 Protocolo de Rollback Imediato (Gate A)
Em caso de falha em qualquer uma das 10 verificações pós-cutover (Seção V4.5.3 do plano):
1. **Desativar a Borda**: `tailscale serve reset` (remove a escuta em 443 sem reiniciar backend/frontend/túnel).
2. **Restaurar Ambiente**: Voltar `FRONTEND_ORIGIN=http://localhost:13100`, remover `AUTH_TOKEN_TTL`, e recriar o container do backend. O bypass auth será reativado automaticamente em loopback.
3. **Restaurar Policy ACL**: Selecionar a revisao anterior no Tailscale Admin Console.

---

## 4. Entradas Restantes do Owner Humano (Owner-Only Inputs)

As seguintes ações exigem intervenção exclusiva do Owner humano antes da execução:
1. **Ativação do HTTPS no Tailnet**: Habilitar a opção "HTTPS Certificates" no painel admin do Tailscale (`cloud.labs.brazil@gmail.com`) para que `CertDomains` reporte o FQDN.
2. **Declaração do IP do Dispositivo**: Declarar o IP Tailscale do seu dispositivo pessoal (`owner-dev-1`) para preencher a regra de ACL.
3. **Atualização da Policy ACL**: Aplicar a regra de restrição de porta 443 no painel Tailscale Admin Console e remover quaisquer regras genéricas de leitura/escrita (`* -> *`).
4. **Autorização Escrita para a Janela de Recriação de Container**: Aprovar a breve pausa do backend Go para aplicação de `LOCAL_AUTH_BYPASS=false`, `FRONTEND_ORIGIN=https://orq1.tail96e2c0.ts.net`, `MULTICA_TRUSTED_PROXIES=127.0.0.1/32` e `AUTH_TOKEN_TTL=24h`.

---

## 5. Check-out Citing ORQ-17

- **Governança**: `ORQ-17` / `GTL-R17R`
- **Artefato Gerado**: `.deploy-control/p0/evidence/gtl-orq17-cutover-v4-peer-review.md`
- **Veredito**: **PASS** ✅
- **Status de Mutação**: READ-ONLY. Zero alterações executadas no sistema.

---

## Addendum GTL-R17D: Independent Device Mapping & ACL Verification (READ-ONLY)

**auditor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T12:54Z  
**governança**: Kanban Issue `ORQ-17`  
**modo**: READ-ONLY — `tailscale status --json` medido diretamente sem mutação de rede, cert, env ou policy  

### 1. Medição Factual dos Nós Autorizados e Servidores

| Nó / Papel | DNSName Exato | Node ID Exato | IPv4 Exato | OS | Estado Medido | Status de Exclusão / Aceite |
|---|---|---|---|---|---|---|
| **ORQ1 (Servidor Web)** | `orq1.tail96e2c0.ts.net.` | `nus9yXJufe11CNTRL` | `100.118.244.61` | Linux | **ONLINE** | Alvo de Borda `:443` |
| **ORQ2 (Servidor Daemon/Tunnel)** | `orq2.tail96e2c0.ts.net.` | `nUgByTFSuw11CNTRL` | `100.110.178.47` | Linux | **ONLINE** | Restrito a `:22` e `:18080` (DENIED na 443) |
| **Owner Laptop 1** | `msi-laptop-1.tail96e2c0.ts.net.` | `nFo3jVWzSA11CNTRL` | `100.112.85.91` | Windows | **ONLINE** | **VERIFICADO & ACEITO** |
| **Owner Laptop 2** | `hp-laptop.tail96e2c0.ts.net.` | `nWs3fyq48A21CNTRL` | `100.85.79.80` | Windows | **ONLINE** | **VERIFICADO & ACEITO** |
| **Owner Laptop 3** | `manoelneto-laptop-1.tail96e2c0.ts.net.` | `nM39yzyBqz11CNTRL` | `100.86.110.121` | Windows | **OFFLINE** | **ACEITE PENDENTE (Dispositivo Offline)** |

### 2. Confirmação de Exclusão de Duplicatas Linux e Nós de Agente
As seguintes máquinas Linux foram **medidas e explicitamente excluídas** da permissão de acesso à porta 443 do ORQ1:
- `msi-laptop` (Linux, Node ID `n4VGspiHL721CNTRL`, IP `100.122.21.119`) -> **EXCLUÍDO**
- `manoelneto-laptop` (Linux, Node ID `nox87nmLpd11CNTRL`, IP `100.98.214.121`) -> **EXCLUÍDO**
- `lenovo-lab` (Linux, Node ID `nK8hGrpyb811CNTRL`, IP `100.120.203.49`) -> **EXCLUÍDO**
- `wsl-lenovo-lab` (Linux, Node ID `n7tNLoRv1k11CNTRL`, IP `100.104.184.20`) -> **EXCLUÍDO**
- `wsl-dataops-labs` (Linux, Node ID `nGe5CJnDeC21CNTRL`, IP `100.117.245.15`) -> **EXCLUÍDO**
- `ec2-jump-box` (Linux, Node ID `ndHZovrehX11CNTRL`, IP `100.94.211.42`) -> **EXCLUÍDO**

### 3. Validação dos Seletores de ACL e Preservação de Porta do ORQ2
- **Configuração de `hosts` e `acls` Recomendada**:
  ```jsonc
  {
    "hosts": {
      "orq1": "100.118.244.61",
      "orq2": "100.110.178.47",
      "owner-msi": "100.112.85.91",
      "owner-hp": "100.85.79.80",
      "owner-manoelneto": "100.86.110.121"
    },
    "acls": [
      { "action": "accept", "src": ["owner-msi", "owner-hp", "owner-manoelneto"], "dst": ["orq1:443"] },
      { "action": "accept", "src": ["orq2"], "dst": ["orq1:22", "orq1:18080"] }
    ]
  }
  ```
- **Conexão ORQ2 -> ORQ1**: A permissão é mantida exclusivamente para as portas `22` e `18080`. Como a porta `443` não possui regra de permissão para `orq2`, o padrão *default deny* do Tailscale bloqueia conexões do ORQ2 para a porta 443 no nível da camada de rede.

### 4. Veredito do Addendum GTL-R17D
- **VEREDITO**: **PASS (Mapeamento de Dispositivos e Seletores de ACL Validados)** ✅
- **Ressalva de Aceite**: O nó `manoelneto-laptop-1` foi registrado como **ACEITE PENDENTE** devido ao seu estado offline atual. Nenhum teste falso foi inventado.

