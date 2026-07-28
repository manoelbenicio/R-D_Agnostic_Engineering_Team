# Triagem READ-ONLY de duplicatas — JWT_SECRET, daemon token `mdt_`, gateway inference key

- Relacionado a **ORQ-32** (`b01925fe-e914-422a-812e-f63cada274dc`, 32) e **ORQ-33**
  (`dfeabbdc-33e1-4ab8-9460-27b43df227db`, 33)
- Auditor: Codex56#A (`w7:p3`) · UTC 2026-07-27T15:12Z
- Modo READ-ONLY: nenhum card criado, comentado, atribuído ou com status alterado. Toda leitura por
  `GET … ?workspace_id=…`, com checagem de `http_code == 200` e exigência de `id` + `identifier` +
  `number` presentes antes de afirmar identidade (padrão fail-closed do ORQ-38).

## 1. Base de comparação verificada

```text
GET /api/issues?workspace_id=20fce817-895d-447b-965a-49f5e279314a   -> 200
TOTAL 31 · MAX number 41 · registros sem id/identifier/number: 0
```

Varredura por palavra-chave no título (`jwt`, `mdt_`, `daemon token`, `omniroute`, `gateway`,
`inference key`, além do conjunto amplo `secret|token|credential|rotation|rotação`):

| busca | hits | cards |
|---|---:|---|
| `jwt` | 1 | ORQ-30 |
| `mdt_` / `daemon token` | **0** | — |
| `omniroute` / `gateway` / `inference key` | **0** | — |
| amplo (`secret\|token\|credential\|rotation`) | 8 | ORQ-14, ORQ-30, ORQ-32, ORQ-33, ORQ-34, ORQ-35, ORQ-36, ORQ-37 |

UUIDs verificados dos 8 candidatos:

| ORQ | UUID | number | título (66c) |
|---|---|---:|---|
| ORQ-14 | `19ab8cfe-0a95-4a79-af99-4ee384326021` | 14 | Contabilizar tokens reais em AGY, Codex e Kiro |
| ORQ-30 | `4369b017-4740-4ea4-bd7e-d9a8a86d7c03` | 30 | Persist backend restart environment and remediate JWT rotation inc… |
| ORQ-32 | `b01925fe-e914-422a-812e-f63cada274dc` | 32 | Security Wave B: Handshake Token Rotation & Lifecycle |
| ORQ-33 | `dfeabbdc-33e1-4ab8-9460-27b43df227db` | 33 | Security Wave B: Rev Token Rotation & Validation |
| ORQ-34 | `685524e4-eec7-4e7e-9e10-a4a51346fc14` | 34 | Security Wave B: OPENAI_API_KEY Secret Management & Rotation |
| ORQ-35 | `3f73ff90-55a1-4c2f-a52d-d3735580ce7e` | 35 | Security Wave B: PostgreSQL / DATABASE_URL Credential Hardening |
| ORQ-36 | `41645aaf-83ff-4f50-9322-177e51ed99c3` | 36 | Security Wave B: MULTICA_TOKEN Authentication & Secret Governance |
| ORQ-37 | `36d18727-f516-4147-9b0c-1cb7c2b91e83` | 37 | Security Wave B: MCP Authorization & Gatekeeper Token Lifecycle |

## 2. Problema (1) — rotação do `JWT_SECRET`

**Duplicata parcial: ORQ-30 existe, mas o escopo dele NÃO é a rotação.** Lido por UUID:

```text
ORQ-30 · 4369b017-… · number 30 · status in_review · priority urgent
título: "Persist backend restart environment and remediate JWT rotation incident"
descrição (verbatim, trecho): "…the live 64-character JWT was transferred internally without stdout,
argv, evidence value, or regeneration." + .env durável 0600 sob dir 0700, override de compose,
comando de recreate, rollback de um comando, dois recreates controlados aprovados.
```

Leitura: ORQ-30 entregou **persistência do ambiente** e contenção do incidente, e afirma
explicitamente que o segredo foi movido **sem regeneração**. Logo a **rotação do valor** continua
pendente e não pertence ao aceite já cumprido de um card em `in_review`.

**Decisão: 1 card novo** — "Rotate `JWT_SECRET` and invalidate pre-rotation sessions" — com
**dependência declarada de ORQ-30** (`4369b017-…`), reaproveitando o `.env` durável e o rollback de um
comando que ORQ-30 já criou. Reabrir ORQ-30 seria alterar escopo de card em revisão.

## 3. Problema (2) — ciclo de vida/rotação do daemon token `mdt_`

**Zero duplicata.** Nenhum título contém `mdt_` ou "daemon token". O card mais próximo é **ORQ-36**
(`41645aaf-…`), cuja descrição verbatim é: *"Remediation card for MULTICA_TOKEN: rotate authentication
tokens, audit API header propagation, and enforce secret safety rules across background services."*

Por que **não** é reuso: `MULTICA_TOKEN` é o token **task-scoped `mat_`** injetado no processo do
agente (`internal/auth/jwt.go:80-89`; `internal/daemon/daemon.go:3634-3643`;
`cmd/multica/cmd_agent.go:253`). O `mdt_` é credencial **de daemon**, gerada por
`GenerateDaemonToken` (`internal/auth/jwt.go:70-76`) e validada por outra via
(`internal/middleware/daemon_auth.go:79,106-107`, cadeia em `cmd/server/router.go:511`), com cache
próprio (`DaemonTokenCache`). São credenciais, geradores, validadores e blast radius distintos:
rotacionar `mdt_` desconecta daemons; rotacionar `mat_` afeta uma task.

**Decisão: 1 card novo** — "Daemon token (`mdt_`) lifecycle: rotation, revocation and cache
invalidation" — sem estender ORQ-36.

## 4. Problema (3) — ciclo de vida da inference key do gateway OmniRoute

**Zero duplicata.** Nenhum título contém `omniroute`, `gateway` ou `inference key`. O card mais próximo
é **ORQ-37** (`36d18727-…`, "MCP Authorization & Gatekeeper Token Lifecycle"), que trata de
`Authorization` de **MCP** — os artefatos `/tmp/mh` e `/tmp/mcp_h` do ORQ1 —, não da chave de
inferência do OmniRoute.

Por que **não** é reuso: a gateway key é consumida por **referência de arquivo** com fail-closed
(`internal/daemon/brain/config.go:16` `AGENT_BRAIN_GATEWAY_SECRET_FILE`, `:80-92` `SecretFileRef`
exigindo caminho absoluto, `:159`, `:174`), com valor em `/etc/agent-brain/secrets/omniroute-inference-key`
no host do daemon, e o container `omniroute` mantém a própria autenticação (`/v1/models` → 401 sem
chave). Escopo, host e mecanismo diferem de MCP.

**Decisão: 1 card novo** — "OmniRoute gateway inference key lifecycle: rotation, file-reference
hardening and fail-closed proof".

## 5. Mapeamento final

| problema | duplicata verificada? | decisão | vínculos por UUID |
|---|---|---|---|
| (1) rotação de `JWT_SECRET` | **parcial** — ORQ-30 cobre persistência/incidente, não rotação | **1 card novo** | depende de ORQ-30 `4369b017-4740-4ea4-bd7e-d9a8a86d7c03` |
| (2) `mdt_` lifecycle | **não** | **1 card novo** | vizinho, não duplicata: ORQ-36 `41645aaf-83ff-4f50-9322-177e51ed99c3` |
| (3) gateway inference key | **não** | **1 card novo** | vizinho, não duplicata: ORQ-37 `36d18727-f516-4147-9b0c-1cb7c2b91e83` |

Total: **3 cards novos, um por problema distinto**, zero duplicata. Nenhum deles deve nascer com
assignee (freeze de atribuição, e atribuir enfileira task paga — `handler/issue.go:2530-2536`).
A criação cabe ao registrar designado, um card por vez, com `POST` + `GET` por UUID e prova de
unicidade, como nos K05/K06/K08.

## 6. Não-alegações

- Não criei, comentei, atribuí nem alterei status/descrição de card algum.
- Toda leitura foi `GET` com `workspace_id`, exigindo `http_code 200` e os três campos; nenhum card foi
  afirmado a partir de leitura sem `workspace_id`.
- Não li valor de segredo; a classificação dos três problemas é por nome de chave e por
  arquivo:linha do código.
- Não avaliei o mérito técnico de ORQ-30 (em `in_review`) nem reexecutei os gates dele.
- Não inspecionei comentários dos cards: se houver duplicata registrada apenas em comentário, ela não
  aparece nesta triagem — fica declarado como limite.
