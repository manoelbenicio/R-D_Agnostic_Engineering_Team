# GTL-35 — Peer review independente do design de remediação de auth (ORQ-17)

- Revisor: Kiro-Opus5 (leitura e recomendação; sem poder de decisão, AA-001 §0.0)
- Documento auditado: `.deploy-control/p0/evidence/gtl-orq17-auth-remediation-design.md`
  (autor Antigravity w8:p2, 2026-07-27T11:40Z)
- Dispatch: GENERAL-TL GTL-35, 2026-07-27T11:41Z
- Modo: READ-ONLY. Nenhum código, env, rede, serviço ou worktree tocado. O worktree
  ORQ-26 (`/home/ec2-user/workspace/worktrees/gtl-orq26`) permanece congelado e não foi
  acessado nesta tarefa. Única escrita: este arquivo.

## Veredito: BLOCK

O design está direcionalmente correto (proxy HTTPS + ACL, binds em loopback, fim do túnel
SSH), mas contém **seis erros factuais sobre o código existente**, **quatro dependências
circulares** e **um risco real de lockout do owner** que precisam ser corrigidos antes de
qualquer implementação. Também classifica como decisão de negócio itens que são técnicos, e
omite decisões de negócio que realmente existem.

---

## 1. Erros factuais sobre o sistema atual

| # | Afirmação do design | Realidade no código |
|---|---|---|
| F1 | Cookie `multica_session` (§3.1) | `AuthCookieName = "multica_auth"` e `CSRFCookieName = "multica_csrf"` — `server/internal/auth/cookie.go:21-22` |
| F2 | `SameSite=Lax` (§3.1) | Hoje é `http.SameSiteStrictMode` nos quatro Set-Cookie — `cookie.go:164,181,201,211`. Implementar Lax é **downgrade** |
| F3 | Sessão validada contra a tabela `user_session` (§3.1) | Não existe `user_session`. Auth é JWT em cookie + PAT; as únicas tabelas com "session" são `chat_session` (`migrations/033_chat.up.sql:4`), `daemon_pairing_session` (`005_daemon_pairing.up.sql:1`) e `lark_chat_session_binding` (`109_lark_integration.up.sql:118`). Não há store de sessão server-side para expirar |
| F4 | CSRF via `X-Requested-With` ou `Origin` (§5) | Já existe CSRF mais forte: double-submit com token HMAC-derivado do auth token, header `X-CSRF-Token`, validado em `auth.ValidateCSRF` (`cookie.go:130-146,217-227`) e aplicado no middleware quando o auth veio de cookie (`middleware/auth.go:144-148`). Trocar por `X-Requested-With` **enfraquece** e exigiria mudança de cliente |
| F5 | `Secure` "exigido estritamente pelo navegador" (§3.1) | O flag é derivado do **scheme de `FRONTEND_ORIGIN`** — `cookie.go:114-118`. Se `FRONTEND_ORIGIN` continuar `http://localhost...`, os cookies saem sem `Secure` mesmo servindo HTTPS |
| F6 | Binds e portas a ajustar: Next `13100`, Go `18080`, PG `5432` (§6) | O compose self-host **já** publica só em loopback: Postgres `127.0.0.1:5432` (`docker-compose.selfhost.yml:33`), backend `127.0.0.1:8080` (:51), frontend `127.0.0.1:3000` (:129). As portas do design (13100/18080) não correspondem a nenhum default do repositório |

Complementos omitidos que o design precisa cobrir:

- **F7 — `X-Real-IP`/XFF são ignorados por padrão.** O design injeta `header_up X-Real-IP`,
  mas a aplicação só confia nesses headers se `MULTICA_TRUSTED_PROXIES` incluir
  `127.0.0.1/32` (`docker-compose.selfhost.yml`, bloco `MULTICA_TRUSTED_PROXIES`). Sem
  isso, rate limit e logging por IP passam a ver sempre o proxy.
- **F8 — mais variáveis dependem da nova origem** do que o design lista: além de
  `CORS_ALLOWED_ORIGINS`, também `FRONTEND_ORIGIN`, `MULTICA_APP_URL`,
  `MULTICA_PUBLIC_URL` e `GOOGLE_REDIRECT_URI`.
- **F9 — limite de upload inconsistente.** O servidor aceita 100 MB
  (`maxUploadSize = 100 << 20`, `internal/handler/file.go`); o exemplo Nginx do design usa
  `client_max_body_size 50M`, o que quebraria uploads entre 50 e 100 MB.
- **F10 — `pg_hba ... trust` (§6.3) é inaceitável.** `trust` permite que qualquer processo
  local se conecte como qualquer usuário, inclusive superusuário, sem senha; contradiz o
  objetivo "zero bypass" do próprio §1.1. Deve ser `scram-sha-256`, sem alternativa.
- **F11 — o Caddyfile de §4.1 provavelmente não carrega.** O matcher nomeado
  `@websockets` está definido **dentro** de `handle @allowed_ips`; matchers nomeados são
  declarados no nível do site block. Além disso `handle /api/*` aparece depois do
  `handle @websockets`, e o `reverse_proxy` de WS não repassa `X-Forwarded-Proto`.
  Precisa de `caddy validate` antes de qualquer ativação (não executei nada).

## 2. Dependências circulares

**CD1 — bypass ↔ `FRONTEND_ORIGIN` ↔ cookie `Secure` ↔ URL do proxy.** O bypass local só
existe enquanto `FRONTEND_ORIGIN` for loopback: `localAuthBypassEmail()` exige
`MULTICA_LOCAL_AUTH_BYPASS=true` **e** host `localhost`/IP de loopback
(`middleware/auth.go:38-55`). O mesmo `FRONTEND_ORIGIN` governa o flag `Secure`
(`cookie.go:114-118`). Logo, trocar `FRONTEND_ORIGIN` para `https://<host>:13100` executa,
num único movimento atômico: desligar o bypass, ligar `Secure`, invalidar
`GOOGLE_REDIRECT_URI` e exigir atualização de CORS/CSRF. O design trata isso como passos
independentes (§7.1 passos 2 e 3) — não são.

**CD2 — rollback reabilita instância sem autenticação.** O rollback (§7.2 passo 2) reverte
os binds e, por consequência, `FRONTEND_ORIGIN` para loopback. Com
`MULTICA_LOCAL_AUTH_BYPASS` ainda `true`, o rollback **restaura silenciosamente uma
instância sem login**. O rollback precisa fixar `MULTICA_LOCAL_AUTH_BYPASS=false` de forma
permanente e separada.

**CD3 — certificado ↔ endereço do site.** A Opção A (`tailscale cert`) emite certificado
para o **nome MagicDNS**, não para o literal IP usado como site address e ACL em §4.1
(`https://100.110.178.47:13100`). Como está escrito, Opção A é inimplementável: o browser
verá mismatch de SAN e o owner não abre a página. Ou se adota o nome MagicDNS em todo o
caminho (site address, `FRONTEND_ORIGIN`, CORS, cookies), ou se cai na Opção B com SAN de
IP e pin manual.

**CD4 — rollback depende de artefato não verificado.** `/home/dataops-lab/tunnel-multica.sh`
não foi lido (fora deste host/escopo). Se o túnel encaminha para um bind **não-loopback**,
o passo 2 do rollout (§7.1) quebra exatamente o caminho de rollback do §7.2. Verificar o
alvo do script é pré-condição do rollout.

## 3. Risco de lockout do owner

| # | Cenário | Mitigação exigida antes do rollout |
|---|---|---|
| L1 | ACL de um único `/32` Tailscale. IP muda em re-auth/recriação de nó, ou o owner entra pela LAN com outro IP ⇒ 403 sem segundo caminho | Autorizar CIDR do tailnet **ou** manter `127.0.0.1`+túnel como segundo caminho durante a janela; nunca só um `/32` |
| L2 | Bypass desligado (CD1) sem credencial válida do owner ⇒ ninguém entra | Provar login **antes**: `POST /api/auth/login` existe (`cmd/server/router.go:487`) e o provider de senha é wired por padrão (`router.go:165-168`), mas responde `503 "password login is not configured"` se `AuthProvider` for nil. Confirmar que o usuário do owner tem senha utilizável |
| L3 | Fallback por código de e-mail sem SMTP/Resend configurado | Sem `RESEND_API_KEY`/`SMTP_HOST` os códigos vão para stdout (`.env.example:22`), e existe `MULTICA_DEV_VERIFICATION_CODE`. É break-glass viável **desde que** o owner tenha acesso ao log do container — precisa estar escrito no runbook |
| L4 | Google OAuth como plano B | Não funciona com host de IP literal nem porta não padrão: o redirect URI teria de ser registrado no Google e IPs não são aceitos. Tratar Google como indisponível após a migração, salvo se houver nome DNS real |
| L5 | Cookie `Secure` + `FRONTEND_ORIGIN` inconsistentes | Se o proxy serve HTTPS mas `FRONTEND_ORIGIN` continua `http://...`, cookies saem sem `Secure` e o navegador pode recusá-los em contexto seguro dependendo de `SameSite`/host ⇒ loop de login. Trocar `FRONTEND_ORIGIN` junto com o proxy, não depois |
| L6 | `COOKIE_DOMAIN` com IP | É ignorado por decisão explícita (`cookie.go:95-113`, RFC 6265). Manter vazio para host de IP; se alguém preencher, o Set-Cookie é descartado ⇒ lockout aparente |

## 4. Negócio vs técnica — reclassificação

Corretamente de negócio no design: porta pública (§2.1.3) e política de sessão (§2.1.4),
com a ressalva de que "sessão" aqui é **TTL de JWT**, não linha de tabela (F3).

Reclassificar:

- §2.1.1 (CA do TLS) é **misto**: a escolha é de negócio, mas a Opção A restringe o
  endereço a um nome MagicDNS (CD3). Precisa vir com a decisão de nome/host.
- §2.1.2 (IP ACL) é **misto**: aprovar quem acessa é de negócio; "um único /32" é decisão
  técnica ruim (L1).
- §6.3 (`trust` no `pg_hba`) **não é opção**: técnica, obrigatória em `scram-sha-256` (F10).

Decisões de negócio ausentes que precisam de resposta escrita do owner:

1. Aceitar troca de acesso sem login por acesso com credencial (mudança de UX do owner) e
   desligar `MULTICA_LOCAL_AUTH_BYPASS` de forma definitiva.
2. Como provisionar a credencial do owner: senha via `POST /api/auth/login` ou código de
   e-mail lido do stdout do container (L2/L3).
3. Manter ou abandonar Google OAuth (exige nome DNS real) (L4).
4. Nome DNS/MagicDNS oficial do host, que determina cert, cookies, CORS e ACL (CD3).
5. Janela em que o túnel SSH continua ativo como segundo caminho (L1).

## 5. Avaliação item por item do pedido do dispatch

| Item | Resultado |
|---|---|
| Remoção de bypass | **Incompleto.** O design nunca nomeia `MULTICA_LOCAL_AUTH_BYPASS`/`MULTICA_LOCAL_AUTH_EMAIL` (`middleware/auth.go:38-55`, `docker-compose.selfhost.yml`). Correção de premissa: hoje **não** há "bypass global" exposto na LAN, porque o código exige `FRONTEND_ORIGIN` loopback; o risco real é reintroduzi-lo no rollback (CD2) |
| Proxy HTTPS + ACL | **Aprovado no conceito, reprovado na forma.** F11 (Caddyfile provavelmente inválido), CD3 (cert × IP), L1 (ACL /32) |
| Cookie | **Reprovado.** F1, F2, F5, L6 |
| CORS | **Parcial.** Falta `CORS_ALLOWED_ORIGINS` explícito e as demais variáveis de origem (F8) |
| CSRF | **Reprovado.** F4: substitui um mecanismo mais forte já implementado |
| WS | **Parcial.** Rota e upgrade corretos em conceito; F11 (matcher aninhado, sem `X-Forwarded-Proto`); validar `Origin` no hub após a troca de origem |
| Uploads | **Reprovado.** F9 (50 MB × 100 MB) |
| Binds loopback | **Já satisfeito no compose** (F6); reconfirmar para processos fora do compose, o que não inspecionei |
| Rollout | **Reprovado.** CD1 (passos 2 e 3 são atômicos), CD4 (script de túnel não verificado) |
| Rollback | **Reprovado.** CD2 (reabilita instância sem auth) |
| Negócio vs técnica | **Reprovado.** Ver §4 |

## 6. Correções mínimas exigidas para virar PASS

1. Corrigir F1-F6 e F9 no documento (nomes reais de cookie, `SameSite=Strict`, JWT em vez
   de `user_session`, manter `X-CSRF-Token`, `Secure` via `FRONTEND_ORIGIN`, portas reais,
   cap de 100 MB).
2. Substituir `trust` por `scram-sha-256` (F10).
3. Escolher nome DNS/MagicDNS e usá-lo de forma consistente em site address, cert,
   `FRONTEND_ORIGIN`, `MULTICA_APP_URL`, `MULTICA_PUBLIC_URL`, CORS e ACL (CD3).
4. Transformar o passo "troca de origem" em **um único passo atômico** com checklist de
   todas as variáveis (CD1) e um passo separado e permanente de
   `MULTICA_LOCAL_AUTH_BYPASS=false` (CD2).
5. Pré-condições verificáveis antes de desligar o bypass: credencial do owner testada,
   caminho de break-glass documentado, ACL com CIDR ou segundo caminho ativo (L1-L3).
6. Ler e registrar o alvo de `/home/dataops-lab/tunnel-multica.sh` (CD4).
7. `caddy validate` (ou `nginx -t`) na configuração final antes de qualquer ativação (F11).
8. Adicionar `MULTICA_TRUSTED_PROXIES=127.0.0.1/32` (F7).

## 7. Limites desta revisão

- Não inspecionei o estado **live** (processos, portas abertas, systemd, Tailscale, IP
  real do owner): o dispatch proíbe tocar env/rede e eu não tenho autorização para
  mutação; a checagem de binds acima vem do compose versionado, não do runtime.
- Não li `/home/dataops-lab/tunnel-multica.sh` (fora deste workspace).
- Nenhum gate executado: não há toolchain Go neste host, e instalar é AA-001 §0.1.

## Check-out

- Entregue: veredito BLOCK com 11 correções factuais, 4 dependências circulares, 6
  cenários de lockout, reclassificação negócio/técnica e lista mínima para PASS.
- Bloqueios: estado live e script de túnel não verificáveis no meu escopo.
- Estado: revisão concluída, evidência gravada, reportada a w5:pC. Worktree ORQ-26
  intocado e congelado. Aguardando redistribuição.
