# ORQ-32 — Discovery cross-fleet do "handshake token": segredo real, consumidores reais e escopo

- Card: **ORQ-32** · `b01925fe…` (Security Wave B: Handshake Token Rotation & Lifecycle)
- Auditor: Codex56#A (`w7:p3`) · UTC 2026-07-27T15:08Z
- Modo: **READ-ONLY**. Nenhum `GetSecretValue`/`BatchGetSecretValue`, nenhum `asm-exec`, nenhuma
  rotação, nenhum valor de segredo lido, nenhuma mutação de board/código/serviço.

## 0. Conclusão: o "handshake token" **não existe** como segredo de produto

Duas provas independentes:

1. **O artefato que originou o card era string de teste de pane, não credencial.**
   `RCA-HANDOVER-20260727.md:161-162` registra literalmente:
   `handshake-token.txt` 644, 24 B → *"**falso positivo** — string `HANDSHAKE-KIROTL-131232` de teste
   de pane"*; `rev-token.txt` 644, 24 B → *"**falso positivo** — `MSG-FROM-CODEX56-135612`"*.
   Os 24 bytes que eu mesmo medi na varredura das 11:02Z são exatamente o tamanho dessas strings.
2. **Nenhum consumidor no produto.** Busca em todo `multica-auth-work` (`*.go`, `*.ts`, `*.yml`,
   `*.json`):

```text
HANDSHAKE_TOKEN_PRIMARY      hits=0
HANDSHAKE_TOKEN_SECONDARY    hits=0
MULTICA_HANDSHAKE_TOKEN      hits=0
handshake_token              hits=0
prod/handshake-token         hits=0
```

Estado atual dos arquivos no host LOCAL (`dataops-lab@…245.15`), medido agora:

```text
AUSENTE /tmp/handshake-token.txt      AUSENTE /tmp/rev-token.txt
lsof → LSOF_EXIT=1 (nenhum consumidor)     find /tmp -name '*token*' → vazio
```

Eles foram contidos na Wave A (644→600) e depois **removidos por outro operador**; não há quarentena
em `~/.private-tmp` naquele host. Portanto não há nem artefato a rotacionar.

Isso converge com o peer review do runbook (`CHECKOUT__Agy-P0-A8__orq32-…-peer-review`): **BLOCK**,
por o runbook depender de `HANDSHAKE_TOKEN_PRIMARY/SECONDARY` inexistentes e provocar 401/403. Eu
chego ao mesmo BLOCK por caminho diferente: o problema não é falta de patch Go, é **ausência do
segredo**. O runbook `orq32-handshake-token-rotation-runbook.md:20,29,39-40` assume um segredo AWS
`prod/handshake-token` e três variáveis de ambiente que **não existem no código**.

## 1. Mapa real do plano de autenticação daemon↔server (arquivo:linha)

| # | credencial real | prefixo | onde é gerada | onde é validada | onde é injetada/consumida |
|---|---|---|---|---|---|
| 1 | **Daemon token** | `mdt_` | `internal/auth/jwt.go:70-76` (`GenerateDaemonToken`, `"mdt_" + 40 hex`) | `internal/middleware/daemon_auth.go:79` (`DaemonAuth(...)`), `:106-107` (ramo `mdt_`) | `cmd/server/router.go:511` monta a cadeia `DaemonAuth` para as rotas de daemon |
| 2 | **Task token (agente)** | `mat_` | `internal/auth/jwt.go:80-89` (single-purpose, bound a task) | mesmo `DaemonAuth`/PAT cache | `internal/daemon/daemon.go:3634-3643` injeta como `MULTICA_TOKEN` no processo do agente; `internal/handler/agent.go:321` e `internal/handler/daemon.go:1765` explicam o binding servidor-side; `cmd/multica/cmd_agent.go:253` exige que `MULTICA_TOKEN` seja `mat_` task-scoped; `cmd/multica/cmd_auth.go:74` lê `MULTICA_TOKEN` do ambiente |
| 3 | **User PAT / Cloud PAT** | `mul_` / `mcn_` | servidor (PAT store) | `cmd/server/router.go:391` (verificador Cloud PAT), `middleware/daemon_auth.go` (PAT cache) | `cmd/multica/cmd_login.go:35-52` e `cmd_auth.go:30` (`loginTokenPrefixes`) |
| 4 | **Gateway/OmniRoute inference key** | — (arquivo) | fora do repo | — | referência por caminho: `internal/daemon/brain/config.go:16` (`AGENT_BRAIN_GATEWAY_SECRET_FILE`), `:80-92` (`SecretFileRef`, exige caminho absoluto), `:159`, `:174` (fail-closed se `Required` e path vazio). Valor vive em `/etc/agent-brain/secrets/omniroute-inference-key` no host do daemon |
| 5 | `JWT_SECRET` | — | operação | servidor | `.env` do backend; marcador também no `/tmp/backend-recover.sh` (ORQ2, contido 0600) |
| 6 | `POSTGRES_PASSWORD` / `DATABASE_URL` | — | operação | Postgres/backend | `.env` do backend; marcadores em artefatos de /tmp já contidos |
| 7 | `OPENAI_API_KEY` | — | provedor | consumidor externo | marcador em `/tmp/arch.txt` (ORQ2 e LOCAL, contidos 0600) |

**Não há "handshake" no protocolo.** O registro/boot do daemon usa o **daemon token `mdt_`** validado
por `DaemonAuth`; o agente recebe um **`mat_` task-scoped** como `MULTICA_TOKEN`. `compatibility.go:32`
descreve a superfície como *"task-scoped opaque control token"* — nenhuma etapa de handshake com
segredo compartilhado.

## 2. Consumidores por processo (medido no fleet)

| host | processo | credencial que realmente usa | evidência |
|---|---|---|---|
| ORQ2 | daemon `multica-auth-credential-home-v1` (unit de usuário `multica-daemon-orq2-credential.service`) | `mdt_` para falar com o backend; injeta `mat_` nos agentes; lê o gateway key por **referência de arquivo** | `cgroup` do PID medido na auditoria de runtime; `brain/config.go:16` |
| ORQ1 | backend/frontend/postgres em Docker; túnel `multica-orq1-backend-tunnel.service` | `JWT_SECRET`, `DATABASE_URL`; valida `mdt_`/`mat_`/PAT | router/middleware acima |
| ORQ1 | container `omniroute` | inference key própria (auth-gated, `/v1/models` → 401 sem chave) | medição anterior de readiness |
| LOCAL | nenhum consumidor | — | `lsof` vazio; arquivos ausentes |

## 3. Decisão de escopo proposta (não decido; recomendo)

**ORQ-32 como está redigido deve ser FECHADO como não-defeito (falso positivo), não rotacionado.**
Fundamento: o objeto do card não é segredo (RCA:161-162) e não tem consumidor (0 hits no código, 0
`lsof`, arquivos já ausentes). Rotacionar um segredo inexistente introduz risco sem remover nenhum.

Reescopo sugerido, em cards próprios já existentes ou a criar pelo registrar:

1. **Fechar ORQ-32** com o motivo documentado e link para este mapa e para o peer review BLOCK.
   O mesmo vale para **ORQ-33** (`rev-token`), pelo mesmo par de linhas do RCA.
2. **Manter e priorizar os segredos reais** já cardados na Wave B: ORQ-34 (`OPENAI_API_KEY`),
   ORQ-35 (`POSTGRES_PASSWORD`/`DATABASE_URL`), ORQ-36 (`MULTICA_TOKEN`), ORQ-37 (Authorization MCP).
   Ordem por blast radius (do menor para o maior): 34 → 36 → 37 → 35.
3. **Lacuna que nenhum card cobre hoje:** ciclo de vida do **daemon token `mdt_`** e da **gateway
   inference key**. `mdt_` é gerado por `jwt.go:70-76` e cacheado (`DaemonTokenCache`), mas não vi
   card de rotação/revogação; a gateway key é referência de arquivo sem card de rotação. Recomendo
   **um** card para "rotação/revogação de daemon token `mdt_` e da chave de inferência do gateway",
   criado pelo registrar designado, sem assignee (freeze vigente).
4. **JWT_SECRET** aparece no plano da Wave A como B2 mas **não tem card**; deve entrar no mesmo lote
   de reescopo.

## 4. Não-alegações

- Não chamei `GetSecretValue`/`BatchGetSecretValue`, não usei `asm-exec`, não rotacionei, não li nem
  imprimi valor de segredo algum. Classificação por **nome/prefixo** e por contagem.
- Não afirmo que os arquivos removidos no LOCAL foram deletados com autorização: apenas que **hoje
  estão ausentes** e não há quarentena naquele host. Quem removeu e quando está fora do meu alcance.
- Não inspecionei o interior do container `omniroute` nem a configuração de sua própria chave.
- Não verifiquei se existe algum segredo `prod/handshake-token` no AWS Secrets Manager: verificar
  exigiria chamada de API de segredo, o que está proibido. Afirmo apenas que **nenhum código do
  produto o referencia**.
- Não mutei board: não fechei, não criei e não atribuí card algum. As recomendações de §3 são para o
  registrar/GTL.
