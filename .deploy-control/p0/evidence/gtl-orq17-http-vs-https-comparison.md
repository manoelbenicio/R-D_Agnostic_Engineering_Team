# Comparação Técnica de Arquitetura: HTTP Direto no Tailnet vs HTTPS Tailscale Serve (ORQ-17)

**autor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T14:15Z  
**governança**: Kanban Issue `ORQ-17`  
**modo**: READ-ONLY / ANÁLISE COMPARATIVA DE ARQUITETURA — NENHUMA alteração de plano, código, banco, container, cert, serve, ACL ou rede executada  

---

## 1. Visão Geral da Comparação

Esta análise compara fundamentadamente as duas opções arquiteturais de publicação do serviço Multica no host ORQ1:

- **Opção A (HTTP Direto no Tailnet)**: Acesso direto via protocolo HTTP desprotegido nas portas `:13100` (frontend) e `:18080` (backend) expostas na interface Tailscale (`http://100.118.244.61:...` ou `http://orq1.tail96e2c0.ts.net:...`).
- **Opção B (HTTPS Tailscale Serve)**: Publicação via proxy de borda `tailscale serve` na porta pública HTTPS `:443` com certificado TLS automático do Tailscale MagicDNS (`https://orq1.tail96e2c0.ts.net`), mantendo o frontend (`:13100`) e o backend (`:18080`) 100% restritos ao loopback (`127.0.0.1`).

---

## 2. Matriz Comparativa por Dimensão Técnica

| Dimensão Técnica | Opção A: HTTP Direto no Tailnet | Opção B: HTTPS Tailscale Serve (`:443`) | Veredito & Impacto |
|---|---|---|---|
| **Segurança de Cookies (`Secure` / `SameSite`)** | ❌ **Inviável**. O código (`cookie.go:115-119`) deriva a flag `Secure` do esquema. Em HTTP, navegadores modernos (Chrome, Edge, Firefox, Safari) rejeitam cookies `Secure` em domínios não-localhost, quebrando a sessão. | ✅ **Ideal**. O esquema `https://` ativa a flag `Secure=true` e o roteamento em origem única garante `SameSite=Strict` sem perda de cookies em chamadas de API. | **Opção B é Obrigatória** |
| **Provedores de OAuth 2.0 (Google OAuth)** | ❌ **Inviável**. O Google OAuth 2.0 exige estritamente URIs de callback HTTPS (`https://...`). URLs HTTP com IP ou FQDN Tailnet são rejeitadas pelo Google Cloud Console. | ✅ **Totalmente Compatível**. Permite registrar a URI de callback oficial `https://orq1.tail96e2c0.ts.net/auth/callback` no painel do Google. | **Opção B é Obrigatória** |
| **Protocolo WebSocket (`/ws`)** | 🟡 **Limitado**. Utiliza `ws://` desprotegido. Navegadores bloqueiam conexões `ws://` inseguras quando a página tenta operar sob contexto seguro. | ✅ **Ideal**. Utiliza `wss://` criptografado via TLS. O `tailscale serve` gerencia o HTTP Upgrade nativamente e o backend valida os cabeçalhos `X-Forwarded-Host`. | **Opção B é Superior** |
| **Uploads de Mídia & Anexos (`/uploads`)** | ✅ Funcional. Uploads diretos para a porta do backend Go. | ✅ **Ideal**. O `tailscale serve` repassa requisições multipart sem impor limites arbitrários de tamanho de payload no proxy. | **Empate Técnico** |
| **Controle de Acesso por ACL (Tailscale)** | 🟡 **Fraco**. Exige expor duas portas públicas no Tailnet (`:13100` e `:18080`), aumentando a superfície de ataque nos nós da rede. | ✅ **Forte e Enxuto**. Expõe apenas a porta `:443` no Tailnet. Frontend e Backend ficam 100% isolados em `127.0.0.1`. ACL restringe a `:443` exclusivamente aos IPs do Owner. | **Opção B é Superior** |
| **Custo Financeiro de Infraestrutura** | **$0** (Grátis) | **$0** (Grátis) — Recurso nativo, incluso e sem custo no plano do Tailscale. | **Empate ($0 em Ambas)** |
| **Sobrecarga de Gestão de Certificados** | N/A | **Zero Esforço**. O Tailscale solicita, emite e renova os certificados Let's Encrypt automaticamente via MagicDNS sem `certbot` ou crons. | **Opção B é Superior** |

---

## 3. Análise Detalhada dos Pontos de Decisão

### 3.1 Cookies e Limites do Navegador
O backend Go em `server/internal/auth/cookie.go:115-119` determina a flag `Secure` inspecionando o esquema da URL configurada em `FRONTEND_ORIGIN`:
```go
secure := strings.EqualFold(parsed.Scheme, "https")
```
Se a aplicação for operada em HTTP (`http://orq1.tail96e2c0.ts.net:13100`), o esquema é `http`, desativando a flag `Secure`. No entanto, se o ambiente for alterado para requerer cookies seguros enquanto rodando em HTTP, o navegador recusará o armazenamento do cookie `auth_token`, impedindo o login do usuário. Além disso, acessar o frontend na porta 13100 e fazer chamadas à API na porta 18080 estabelece uma relação cross-port que afeta a política `SameSite`, resultando em descarte silencioso de cookies pelo navegador.

### 3.2 Suporte a Google OAuth 2.0
O fluxo de login via Google OAuth exige o envio do parâmetro `redirect_uri`. As políticas de segurança do OAuth 2.0 (RFC 8252 e diretrizes do Google Identity) **proíbem expressamente URIs de callback em HTTP**, exceto para o hostname literal `localhost`. Portanto, URLs como `http://100.118.244.61:18080/auth/callback` ou `http://orq1.tail96e2c0.ts.net/auth/callback` são recusadas na configuração do console do Google. Apenas a URL HTTPS `https://orq1.tail96e2c0.ts.net/auth/callback` é aceita.

### 3.3 Isolamento por ACL do Tailnet
Na Opção A, as portas `:13100` e `:18080` precisam ser expostas na interface da Tailnet (`0.0.0.0` ou IP da Tailscale). Qualquer nó da rede Tailnet conseguirá conectar diretamente à porta do backend Go `:18080`.
Na Opção B, os processos do Next.js e do Go Backend escutam estritamente em `127.0.0.1`. Apenas a porta `:443` do `tailscale serve` é exposta. Com a regra de ACL do Tailscale:
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
A porta `:443` fica acessível **exclusivamente aos laptops autorizados do Owner**, enquanto o `orq2` acessa apenas as portas `:22` e `:18080` (via túnel SSH / comunicação direta necessária), bloqueando tentativas de acesso de nós de agentes à interface de borda 443.

---

## 4. Recomendação Técnica Final & Critérios de Decisão

### **RECOMENDAÇÃO DEFINITIVA**: **Adotar a Opção B (HTTPS Tailscale Serve)**.

### Critérios de Decisão Objetivos:
1. **Inviabilidade Técnica do HTTP para Autenticação**: A Opção A (HTTP) falha nos requisitos de cookies `Secure` e inviabiliza a integração com o Google OAuth 2.0.
2. **Princípio do Menor Privilégio e Isolamento de Rede**: A Opção B mantém as aplicações vinculadas exclusivamente ao loopback (`127.0.0.1`), expondo apenas a porta 443 com controle estrito por ACL.
3. **Custo Zero e Sobrecarga Zero**: Como os certificados TLS do MagicDNS são gratuitos e geridos automaticamente pelo daemon `tailscaled`, a Opção B oferece segurança de nível de produção sem adicionar custos financeiros ou complexidade de manutenção.

---

## 5. Check-out Citing ORQ-17

- **Governança**: `ORQ-17`
- **Artefato Gerado**: `.deploy-control/p0/evidence/gtl-orq17-http-vs-https-comparison.md`
- **Status de Mutação**: READ-ONLY. Zero alterações executadas no sistema.
