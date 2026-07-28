# ORQ-32 — Auditoria independente de fechamento técnico (READ-ONLY)

Data UTC: 2026-07-27T15:16Z
Objeto: `.deploy-control/p0/evidence/orq32-handshake-token-discovery-map.md`
Fontes: RCA, código/configuração versionados e evidências preservadas de ORQ-42/43/44.
Modo: nenhum segredo lido, nenhum secret store consultado, nenhuma varredura sensível filesystem-wide, nenhum build/test/restart e nenhuma mutação de board/código/configuração.

## 1. Veredito

**PASS para fechamento técnico de ORQ-32 como non-defect/falsa premissa.**

Recomendação de status: **`cancelled`**, não `done`. O card propõe rotacionar um segredo handshake que não foi identificado como objeto definido ou consumido pelo produto; portanto não houve implementação/remediação a concluir como `done`.

O PASS é limitado e preciso:

- confirma que os arquivos que originaram handshake/rev foram classificados pelo RCA como artefatos de teste de pane, não credenciais;
- confirma zero referência aos nomes handshake/rev no produto versionado e nas classes de configuração inspecionadas;
- confirma que os mecanismos reais JWT, `mat_`, `mdt_` e gateway key são objetos distintos e possuem cards separados;
- **não** afirma que nenhum objeto homônimo exista em toda conta/região de qualquer secret store externo.

Há duas erratas factuais não bloqueantes no mapa (§5): a via de validação de `mat_` e o estado produtivo atual de `mdt_`. Elas não sustentam nem enfraquecem a conclusão central sobre a inexistência de um segredo handshake/rev integrado ao produto.

## 2. Origem dos arquivos: falso positivo confirmado

`RCA-HANDOVER-20260727.md:161-162` classifica explicitamente:

- `handshake-token.txt` como string de teste de pane;
- `rev-token.txt` como mensagem/string de teste de pane.

A classificação vem do RCA que resolveu a autoria/contexto dos artefatos. Esta auditoria não abriu os arquivos, não depende de seu conteúdo atual e não repete seus textos. A conclusão relevante é de natureza: eram payloads de coordenação/teste, não material de autenticação do produto.

**Resultado:** a origem material do ORQ-32 é um falso positivo confirmado pelo RCA.

## 3. Consumidor/env/secret do produto

Foi feita busca escopada somente em `multica-auth-work`, limitada a arquivos versionados de código e configuração (`Go`, `TS/TSX`, `JS`, shell, YAML, JSON, TOML, service, env/example), pelos dez identificadores:

```text
HANDSHAKE_TOKEN_PRIMARY
HANDSHAKE_TOKEN_SECONDARY
MULTICA_HANDSHAKE_TOKEN
handshake_token
prod/handshake-token
REV_TOKEN_PRIMARY
REV_TOKEN_SECONDARY
MULTICA_REV_TOKEN
rev_token
prod/rev-token
```

**Resultado: zero matches.**

Isso confirma, no escopo verificável:

| Alegação | Resultado | Força da evidência |
|---|---|---|
| consumidor Go/TS/JS/shell | zero referência | confirmado no repo inspecionado |
| variável em YAML/JSON/TOML/service/env/example | zero referência | confirmado no repo inspecionado |
| nome de segredo referenciado pelo produto | zero referência | confirmado no repo inspecionado |
| objeto de mesmo nome em secret store externo | não consultado | não alegado |
| consumidor em outro repositório/sistema não indexado | não inspecionado | não alegado |

Portanto a formulação tecnicamente defensável é: **“nenhum segredo handshake/rev definido ou consumido pelo produto inspecionado foi identificado.”** A formulação absoluta “o segredo não existe em nenhum lugar” seria mais ampla que a evidência e não deve ser usada.

## 4. Mecanismos reais e separação de escopo

### 4.1 JWT — ORQ-42

`JWT_SECRET` é real e possui consumidores de produção separados:

- emissão de JWT: `server/internal/handler/auth.go:213,226`;
- validação HTTP: `server/internal/middleware/auth.go:295`;
- fallback de autenticação daemon: `server/internal/middleware/daemon_auth.go:234`;
- realtime/WebSocket: `server/internal/realtime/hub.go:691`;
- carregamento/validação de configuração: `server/internal/auth/jwt.go` e `server/cmd/server/main.go`.

Evidência preservada registra **ORQ-42** (`64bfcae0-b867-4812-ad33-ae03ef7f25ae`), “Rotação controlada do JWT_SECRET (pós-ORQ-30)”, separado do handshake fictício e dependente de ORQ-30.

### 4.2 Task token `mat_` / `MULTICA_TOKEN` — ORQ-36

`mat_` é real e task-scoped:

- gerado por `GenerateAgentTaskToken`: `server/internal/auth/jwt.go:80-89`;
- validado pelo middleware **regular `Auth`**, via `GetTaskTokenByHash`: `server/internal/middleware/auth.go:151-181`;
- injetado no processo do agente como `MULTICA_TOKEN`: `server/internal/daemon/daemon.go:3634-3643`;
- exigido em contexto de agente: `server/cmd/multica/cmd_agent.go:252-253`.

Seu owner permanece **ORQ-36** (`MULTICA_TOKEN Authentication & Secret Governance`). `mat_` está explicitamente fora do escopo de ORQ-43. Não deve ser reclassificado como handshake nem agrupado com `mdt_`.

### 4.3 Daemon token `mdt_` — ORQ-43

`mdt_` é um mecanismo real **parcialmente wired**:

- gerador existe: `GenerateDaemonToken`, `server/internal/auth/jwt.go:70-76`;
- schema/query de persistência existem;
- validador existe em `DaemonAuth`: `server/internal/middleware/daemon_auth.go:62-143`;
- cache/revogação existem.

Mas a busca por caller encontrou:

- `GenerateDaemonToken()` somente na própria definição;
- `CreateDaemonToken` somente na query/gerado, sem caller de produção;
- código registra que `daemon_token` está hoje sem uso produtivo e que a maioria dos daemons usa PAT/JWT: `server/internal/handler/workspace_revoke.go:26-31`;
- teste de caracterização afirma que minting `mdt_` ainda não está wired: `server/internal/middleware/daemon_auth_test.go:55-57`.

Isso não cria um “handshake token”. Ao contrário, justifica **ORQ-43** (`d1149dd3-9da8-4678-a4b7-d98f3eddca14`) como card próprio para completar emissão, persistência client-side, rotação/revogação e cache. O design `orq43-daemon-token-mdt-lifecycle-design.md` já declara `mat_`/ORQ-36 fora de escopo e registra a ausência de emissor.

### 4.4 OmniRoute gateway inference key — ORQ-44

A gateway key é real e consumida por referência de arquivo, não por handshake:

- env de referência: `AGENT_BRAIN_GATEWAY_SECRET_FILE`, `server/internal/daemon/brain/config.go:16`;
- referência deve ser absoluta: `brain/config.go:80-92`;
- modo required exige referência e readiness estrito/fail-closed: `brain/config.go:159-178`;
- `gateway.Client` exige `SecretFile.Path` e `CredentialSource`: `server/internal/daemon/gateway/client.go:58-84`;
- `FileCredentialSource` é a fonte de produção: `server/internal/daemon/credential_file_source.go:26-45`;
- contrato operacional nomeia somente path/metadata, nunca valor: `server/internal/daemon/deploy/secret_reference.go`.

Evidência preservada registra **ORQ-44** (`abd12d6a-16a5-439b-bc52-74ec4bb6b231`) para ciclo de vida/rotação da gateway inference key, explicitamente distinto de MCP, `mdt_` e handshake.

## 5. Erratas do mapa auditado

### E1 — `mat_` não é validado pelo “mesmo DaemonAuth/PAT cache”

A tabela do mapa atribui a validação de `mat_` ao mesmo caminho de daemon. O código mostra outra divisão:

- `mdt_` → `middleware.DaemonAuth` + `GetDaemonTokenByHash`;
- `mat_` → `middleware.Auth` + `GetTaskTokenByHash`.

**Classificação:** errata factual não bloqueante. A separação ORQ-36 versus ORQ-43 permanece correta e deve ser preservada.

### E2 — `mdt_` não deve ser apresentado como fluxo produtivo atual comprovado

O mapa descreve ORQ2 como usando `mdt_`. Esta auditoria não leu env/cmdline/token live e não valida essa alegação operacional. O código versionado aponta o contrário como estado geral: emissão/persistência não estão wired e a maioria dos daemons usa PAT/JWT.

Formulação corrigida: **“o produto possui schema, validação, cache e revogação para `mdt_`, mas o caminho de emissão/adoção produtiva ainda não está wired; ORQ-43 é dono dessa lacuna.”**

**Classificação:** errata de estado operacional não bloqueante para fechar ORQ-32.

## 6. Solidez do fechamento

Fechar ORQ-32 é tecnicamente sólido porque as duas condições necessárias convergem:

1. o artefato originador não era credencial, conforme RCA;
2. nenhum consumidor, env name ou referência de segredo handshake/rev foi encontrado no produto inspecionado.

Manter ORQ-32 aberto para executar o runbook seria mais arriscado: o runbook pressupõe nomes e dual-token inexistentes e poderia gerar falso sucesso ou indisponibilidade sem rotacionar nenhum mecanismo real.

O fechamento não abandona trabalho real: JWT, `mdt_`, gateway key e `mat_` permanecem separados em ORQ-42, ORQ-43, ORQ-44 e ORQ-36, respectivamente.

## 7. Status e wording exatos recomendados

**Status recomendado:** `cancelled`.

**Texto exato recomendado para o registro de fechamento (não aplicado nesta auditoria):**

> ORQ-32 cancelada como non-defect por premissa inexistente no produto. O RCA (`RCA-HANDOVER-20260727.md:161-162`) classifica `handshake-token.txt` e `rev-token.txt` como artefatos de teste de pane, não credenciais. Busca escopada no código e configuração versionados encontrou zero referência aos nomes handshake/rev auditados; portanto nenhum segredo handshake/rev definido ou consumido pelo produto foi identificado, e nenhuma rotação foi necessária ou executada. Esta conclusão não afirma ausência global de eventual objeto homônimo em secret store externo. O trabalho sobre credenciais reais permanece separado: ORQ-42 (`JWT_SECRET`), ORQ-43 (`mdt_`, incluindo emissão ainda não wired), ORQ-44 (OmniRoute gateway inference key) e ORQ-36 (`mat_`/`MULTICA_TOKEN`). Evidência: `.deploy-control/p0/evidence/orq32-handshake-token-discovery-map.md` e `.deploy-control/p0/evidence/orq32-independent-closure-audit.md`.

Não usar `done`, “segredo removido”, “segredo rotacionado”, “AWS secret inexistente” ou “zero segredo em toda a frota”: nenhuma dessas afirmações foi provada ou executada por esta auditoria.

## 8. Atestação read-only

- Nenhum board/card foi lido via API, comentado, atribuído, fechado ou alterado.
- Nenhum segredo, env value, token, credential file ou secret-store value foi lido.
- Nenhuma varredura filesystem-wide, busca em `/proc`, SSH ou inspeção de container foi feita.
- Nenhum build, teste, formatter, deploy, restart, rotação ou chamada live foi executado.
- Nenhum arquivo de produto ou mapa auditado foi alterado; somente este parecer foi criado.
