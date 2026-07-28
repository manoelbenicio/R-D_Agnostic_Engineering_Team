# ORQ-17 — Monitor independente pós-apply do Gate A (READ-ONLY)

- Card: **ORQ-17** (`7d873133-16d5-42c6-8595-629d6fb16251`)
- Monitor: Kiro-Opus5 (sem poder de decisão, AA-001 §0.0) — independente do executor e do autor
  do preflight (Antigravity w8:p2)
- Nó auditado: **ORQ1** `100.118.244.61` / `orq1.tail96e2c0.ts.net`, Tailscale **1.98.9**
- Data UTC: 2026-07-27T15:08Z
- Modo READ-ONLY. **Não alterei serve, ACL, container, serviço ou config.** Nenhum
  `serve`/`funnel` apply, nenhum `reset`, nenhum restart, nenhum `tailscale cert`. Só leitura:
  `serve status`, `funnel status`, `status --json`, `serve reset --help`, `ss -ltn`, e requisições
  HTTP GET/HEAD idempotentes. Única escrita: este arquivo.

## VEREDITO: **PASS**, com 1 desvio de forma a registrar (não bloqueante) e 2 itens fora do meu alcance de verificação

O Gate A está aplicado e funcional. Todas as rotas do desenho respondem pelo destino correto, o
certificado casa com o FQDN, o Funnel está ausente por dois sinais independentes, e o rollback está
disponível. O desvio é que o config aplicado tem **7 mounts em vez dos 9** do preflight — e eu
**medi** que isso não cria lacuna de roteamento.

---

## 1. Serve status aplicado (medido em ORQ1)

```console
$ tailscale serve status
https://orq1.tail96e2c0.ts.net (tailnet only)
|-- /            proxy http://127.0.0.1:13100/
|-- /ws          proxy http://127.0.0.1:18080/ws
|-- /api         proxy http://127.0.0.1:18080/api
|-- /uploads     proxy http://127.0.0.1:18080/uploads
|-- /auth/login  proxy http://127.0.0.1:18080/auth/login
|-- /auth/google proxy http://127.0.0.1:18080/auth/google
|-- /auth/logout proxy http://127.0.0.1:18080/auth/logout

$ tailscale serve status --json   (resumo)
TCP=["443"]  TCP["443"]={"HTTPS":true}
Web["orq1.tail96e2c0.ts.net:443"].Handlers =
  ["/","/api","/auth/google","/auth/login","/auth/logout","/uploads","/ws"]
AllowFunnel = null
```

Listeners locais no ORQ1: `100.118.244.61:443`, `[fd7a:115c:a1e0::5034:f43e]:443`,
`127.0.0.1:13100`, `127.0.0.1:18080`. Os dois upstreams seguem **loopback-only** — nada de
frontend ou backend escutando em interface de rede.

## 2. Certificado e FQDN — **PASS**

```console
subject = CN=orq1.tail96e2c0.ts.net
issuer  = C=US, O=Let's Encrypt, CN=YE2
notBefore = Jul 27 14:07:29 2026 GMT
notAfter  = Oct 25 14:07:28 2026 GMT
X509v3 Subject Alternative Name: DNS:orq1.tail96e2c0.ts.net
```

- CN **e** SAN batem exatamente com o FQDN canônico do nó (`status --json` → `orq1.tail96e2c0.ts.net.`).
- SAN único, **sem IP literal** — coerente com o que eu havia apontado no review do design (um site
  em IP literal não teria SAN compatível). O caminho escolhido foi o nome MagicDNS, que é o correto.
- Emitido por Let's Encrypt via Tailscale, validade de 90 dias começando hoje 14:07:29Z.
- Verificação de cadeia pelo cliente: `curl` retornou `ssl_verify_result=0` (sem erro) sem qualquer
  flag de exceção, então a confiança é nativa, sem pin manual.

## 3. Rota raiz e subárvores — **PASS por medição**

Todas as probes são GET idempotentes, feitas de ORQ2 sobre o tailnet:

| rota | HTTP | tipo | destino provado |
|---|---|---|---|
| `/` | 200 | `text/html` | Next.js (`<!DOCTYPE html>` com bundle `inter_…-module__`) → `:13100` ✓ |
| `/api` (exato) | 404 | `text/plain` | backend Go (404 plano, não HTML do Next) ✓ |
| `/api/issues` | 400 | `application/json` | backend: `{"error":"workspace_id or workspace_slug is required"}` ✓ |
| `/api/issues?workspace_id=…` | 200 | `application/json` | backend devolveu payload real de issues ✓ |
| `/uploads` (exato) | 404 | `text/plain` | backend ✓ |
| `/uploads/nonexistent.png` | 404 | `text/plain` | backend (arquivo inexistente, rota certa) ✓ |
| `/auth/login` | 405 | — | backend, método não permitido em GET (rota é POST) ✓ |
| `/auth/google` | 405 | — | backend ✓ |
| `/auth/logout` | 405 | — | backend ✓ |
| `/ws` | 400 | `text/plain` | backend WS: `{"error":"workspace_id or workspace_slug required"}` ✓ |
| `/auth/callback` | 200 | `text/html` | **Next.js** — callback do Google OAuth preservado pelo catch-all, como o desenho exigia ✓ |
| `/healthz` | 404 | `text/html` | **Next.js** — não exposto pelo serve |

Conclusões medidas, não inferidas:

1. **Preservação de prefixo confirmada.** A preocupação do preflight §3.2 (strip do mount prefix)
   está resolvida na prática: `/api/issues` chegou ao backend **como** `/api/issues` — prova disso é
   a resposta específica do handler de issues, e não um 404. Mesma prova em `/uploads/...`.
2. **Subárvore funciona com o mount exato.** Os mounts `/api` e `/uploads` (sem barra) atendem
   também os subcaminhos; não houve necessidade dos mounts `/api/` e `/uploads/`.
3. **Ordem de precedência correta.** `/` catch-all não engoliu `/api/*` nem `/auth/login`, e ao
   mesmo tempo capturou `/auth/callback`, que é exatamente o comportamento desejado.

## 4. Funnel ausente — **PASS por dois sinais independentes**

1. `tailscale funnel status` imprime o mesmo bloco marcado **`(tailnet only)`**, sem qualquer
   entrada de Funnel.
2. `tailscale serve status --json` traz **`AllowFunnel = null`** — nenhum host com Funnel habilitado.

Complemento: `ShieldsUp=false`, `WantRunning=true`. O serviço está exposto **somente ao tailnet**.

## 5. Rollback disponível — **PASS**

```console
$ tailscale serve reset --help
Reset current serve config
USAGE
  tailscale serve reset
```

Comando presente e documentado na 1.98.9. **Não executei.** Snapshot para restauração, caso o reset
seja usado, está registrado na §1 (7 mounts, alvo de cada um, `TCP 443 HTTPS:true`) — é o suficiente
para reconstruir o estado atual com os mesmos comandos do preflight §4.1.

Caminho de contingência adicional, medido: o túnel SSH de ORQ2 continua ativo
(`127.0.0.1:18080` em ORQ2 é um listener do processo `ssh`), então o acesso administrativo não
depende do serve.

## 6. Desvio de forma frente ao preflight — registrar, não bloquear

O preflight §4.1 lista **9** comandos de apply; o estado aplicado tem **7** mounts. Ausentes:

- `--set-path=/api/ http://127.0.0.1:18080/api/`
- `--set-path=/uploads/ http://127.0.0.1:18080/uploads/`

Avaliação: **benigno e, na prática, preferível.** Medi que `/api/issues` e `/uploads/<arquivo>`
já são roteados corretamente pelos mounts sem barra, então os dois mounts extras seriam redundantes.
Recomendo **atualizar o preflight para 7 comandos** em vez de aplicar os 2 faltantes, para que
documento e realidade coincidam. Quem decide é o GTL; é mudança de documento, não de serve.

## 7. Itens que eu **não** posso verificar por leitura

| # | item | motivo |
|---|---|---|
| U-1 | Que a política de ACL não foi alterada | exige a admin API/console do tailnet; `serve status` não expõe ACL. O preflight registra "Full-mesh mantido, nenhuma ACL alterada" com base em confirmação do owner, e eu **não posso confirmar nem refutar** |
| U-2 | Que nenhum outro nó do tailnet expõe os mesmos serviços | verifiquei apenas ORQ2 (`No serve config`) e ORQ1; os outros peers não foram inspecionados |

Observação de escopo, sem juízo de valor: uma das probes (`/api/issues?workspace_id=…`) retornou
dados reais do board por ser GET idempotente. Não mutei nada, mas registro que o serve **já expõe
a API de produção ao tailnet inteiro** — o controle de acesso efetivo passa a ser a ACL do tailnet
(U-1) somada à autenticação da aplicação, não mais o túnel SSH. Isso reforça por que o bypass local
de auth precisa estar desligado, item que a suíte de regressão do ORQ-17 (GTL-85R2, PASS) fixa.

## Check-out — ORQ-17

- Veredito independente: **PASS**. Serve aplicado e funcional, certificado e FQDN corretos, root
  para Next.js, rotas exatas e subárvores provadas por medição, Funnel ausente por dois sinais,
  rollback disponível.
- Registrar: desvio 7 vs 9 mounts (§6, benigno, corrigir o documento); U-1 e U-2 fora do meu
  alcance de leitura (§7).
- Não alterei serve, ACL, container, serviço, config nem card; nenhum `reset` executado; nenhum
  segredo lido. Worktrees ORQ-26 e ORQ-17 intocados.
