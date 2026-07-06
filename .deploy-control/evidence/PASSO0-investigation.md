# PASSO 0 - Investigacao bloqueante prodex nativo 0.246.0

Data UTC: 2026-07-05T22:28:50Z..2026-07-05T22:40:32Z
Agente: Codex#5.5#B
Check-in: `.deploy-control/Codex-5.5-B__PASSO0-RUNTIME-INVESTIGATION__20260705T222850Z.md`

## Veredito

NAO (2b).

Os subcomandos nativos do `prodex 0.246.0` expoem runtime proxy/gateway real com Smart Context e rotacao pre-commit, mas nao expoem uma sessao de runtime REAL ja mapeavel diretamente ao contrato `rpp.l2.v1`.

Motivo decisivo: a superficie nativa encontrada e OpenAI-compatible (`/v1/responses`, `/v1/chat/completions`, etc.) e/ou app-server broker skeleton JSON-RPC. Ela nao implementa localmente `/healthz`, `/readyz`, `/v1/session/start`, nem `/v1/events/stream` com semantica `rpp.l2.v1`. Essas rotas existem no `multica-auth-work/prodex-sidecar`, que o proprio codigo declara ser shim/smoke em memoria sem provider calls.

## Ambiente verificado

```text
$ which prodex
/home/dataops-lab/.nvm/versions/node/v24.17.0/bin/prodex

$ prodex --version
prodex 0.246.0

$ file /home/dataops-lab/.nvm/versions/node/v24.17.0/lib/node_modules/@christiandoxa/prodex/node_modules/@christiandoxa/prodex-linux-x64/vendor/prodex
ELF 64-bit LSB pie executable, x86-64, dynamically linked, not stripped
```

## 1. CLI nativo

### `prodex --help`

Comandos relevantes expostos:

```text
run                Run codex through prodex with quota preflight and eligible pre-commit rotation.
app-server-broker  Inspect the experimental JSON-RPC app-server broker contract.
gateway            Run a standalone OpenAI-compatible gateway backed by Prodex provider routing.
context            Audit and compact token-heavy shared Codex context files.
```

### `prodex run --help`

Evidencia de wrapper de launch/rotacao, nao de servidor `rpp.l2.v1`:

```text
Run codex through prodex with quota preflight and eligible pre-commit rotation.
--auto-rotate / --no-auto-rotate
--skip-quota-check
--dry-run
Notes:
  Eligible pre-commit rotation is allowed by default...
  If the selected profile's config.toml sets model_provider to a non-OpenAI backend,
  prodex launches Codex directly without quota preflight or the local auto-rotate proxy.
```

Dry-run sem perfil confirma que `run` precisa de perfil/Codex runtime, nao cria endpoint L2:

```text
$ PRODEX_HOME=/tmp/prodex-pass0-run prodex run --dry-run --skip-quota-check --no-auto-rotate
Error: no active profile selected and no Codex-compatible profiles are available; use `prodex use --profile <name>` or pass --profile
```

### `prodex gateway --help`

Evidencia de gateway OpenAI-compatible e Smart Context:

```text
Run a standalone OpenAI-compatible gateway backed by Prodex provider routing.
--listen <ADDR>
--provider <PROVIDER>
--base-url <URL>
--api-key <KEY>
--auth-token <TOKEN>
--smart-context        Enable Smart Context Autopilot for gateway /v1/responses and /v1/chat/completions requests
```

### `prodex app-server-broker --help` e `--json`

```text
Inspect the experimental JSON-RPC app-server broker contract.
--json
--experimental-stdio  Reserved opt-in for future stdio brokering. Currently reports unsupported
```

Contrato JSON retornado:

```json
{
  "affinity": {
    "continuation_affinity_wins": true,
    "rotate_only_before_turn_commit": true,
    "thread_session_owner_required": true
  },
  "default_mode": "direct-passthrough",
  "enabled_by_default": false,
  "lifecycle_methods": [
    "initialize",
    "initialized",
    "thread/start",
    "thread/resume",
    "thread/fork",
    "turn/start",
    "turn/cancel"
  ],
  "object": "app_server_broker.contract",
  "status": "skeleton",
  "transport": [
    "stdio-planned"
  ]
}
```

Interpretacao: ha um contrato planejado/skeleton com invariantes proximas de rotacao/afinidade, mas nao um broker stdio ativo nem endpoints HTTP `rpp.l2.v1`.

## 2. Grep em `multica-auth-work/prodex-runtime-broker/src`

Resultado:

```text
$ rg -n "smart_context|token_count|compaction|rotate" multica-auth-work/prodex-runtime-broker/src -S
rg: multica-auth-work/prodex-runtime-broker/src: IO error ... No such file or directory
```

Tambem:

```text
$ rg --files . | rg 'prodex-runtime-broker|prodex-core|prodex-context|Cargo.toml|src/.+\.rs$'
./multica-auth-work/prodex-sidecar/src/main.rs
./multica-auth-work/prodex-sidecar/Cargo.toml
```

Conclusao local: este checkout nao contem `prodex-runtime-broker/src` nem crates fonte `prodex-core`/`prodex-context`. O que existe no repo e o sidecar shim.

## 3. Grep/inspecao de `prodex-core` e `prodex-context`

Nao ha fonte local dos crates para grep direto. O binario nativo instalado, porem, nao esta stripped e expoe simbolos desses crates/modulos.

Amostra de simbolos relevantes:

```text
prodex_context::compression::compress_context_text
prodex_context::command_output::compact_command_output_with_options
prodex_context::command_output::compact_command_output_with_intent_options
prodex_context::command_output::compact_successful_command_output_with_options
prodex_state::compact_app_state_with_policy
runtime_proxy::smart_context::token_accounting::smart_context_observed_token_accounting_with_calibration
runtime_proxy::smart_context::token_accounting::estimation::smart_context_estimate_tokens_from_body
runtime_smart_context_* modules
median_input_token_reduction_percent_vs_exact
input_token_reduction_percent=
```

`prodex capability list --json` tambem confirma Smart Context built-in:

```json
{
  "category": "runtime",
  "command": null,
  "description": "runtime proxy context compaction and rehydration",
  "name": "smart-context",
  "status": "built-in"
}
```

Conclusao: existe logica real nativa de Smart Context/compactacao/token accounting no prodex 0.246.0, mas a evidencia disponivel nao mostra essa logica exposta como sessao `rpp.l2.v1`.

## 4. Endpoints expostos por binarios

### `prodex gateway` nativo

Gateway iniciado temporariamente em loopback com upstream falso, sem provider call:

```text
$ PRODEX_HOME=/tmp/prodex-pass0 PRODEX_GATEWAY_TOKEN=pass0-test prodex gateway --listen 127.0.0.1:43119 --base-url http://127.0.0.1:9 --api-key test

[ Gateway ]
URL:           http://127.0.0.1:43119
Provider:      openai-compatible
Auth required: true
Endpoints:     /v1/responses, /v1/chat/completions, /v1/embeddings, /v1/images/*, /v1/audio/*, /v1/batches,
               /v1/rerank, /v1/a2a, /v1/messages
Models:        /v1/models
```

Consultas as rotas `rpp.l2.v1` no gateway nativo:

```text
$ curl -s -i -H 'Authorization: Bearer pass0-test' http://127.0.0.1:43119/healthz
HTTP/1.1 502 Bad Gateway
failed to proxy local provider request to http://127.0.0.1:9/v1/healthz

$ curl -s -i -H 'Authorization: Bearer pass0-test' http://127.0.0.1:43119/readyz
HTTP/1.1 502 Bad Gateway
failed to proxy local provider request to http://127.0.0.1:9/v1/readyz

$ curl -s -i -H 'Authorization: Bearer pass0-test' -H 'Content-Type: application/json' -d '{"contract_version":"rpp.l2.v1","tenant_id":"t","request_id":"pass0","session_id":"s"}' http://127.0.0.1:43119/v1/session/start
HTTP/1.1 502 Bad Gateway
failed to proxy local provider request to http://127.0.0.1:9/v1/session/start

$ curl -s -i -H 'Authorization: Bearer pass0-test' http://127.0.0.1:43119/v1/events/stream?session_id=s
HTTP/1.1 502 Bad Gateway
failed to proxy local provider request to http://127.0.0.1:9/v1/events/stream?session_id=s
```

Interpretacao: o gateway nativo nao implementa essas rotas como controle local. Ele tenta proxiar para upstream OpenAI-compatible.

### `multica-auth-work/prodex-sidecar`

O sidecar expoe as rotas do contrato, mas e shim:

```rust
//! Minimal rpp.l2.v1 sidecar binary for QA and deploy-safety smokes.
//! This broker exposes the control surface the Multica Go daemon and shell
//! smokes expect. It keeps state in memory and performs no provider calls.
```

Rotas no `route()`:

```rust
(Method::Get, "/healthz") => handle_healthz(),
(Method::Get, "/readyz") => handle_readyz(),
(Method::Get, p) if p.starts_with("/v1/events/stream") => handle_events_stream(p),
(Method::Post, "/v1/session/start") => post_json(req, handle_session_start),
```

`handle_session_start()` retorna `router_owner: "rust_l2"`, `runtime_endpoint: "loopback"`, `runtime_log_ref: "memory"` e `smart_context_mode`, mas guarda estado em memoria e nao chama provider.

## Decisao

NAO (2b): nao da para substituir o shim por `prodex run`, `prodex gateway` ou `prodex app-server-broker` diretamente como runtime `rpp.l2.v1`.

Proximo passo tecnico necessario: construir um adapter/broker real que traduza `rpp.l2.v1` para a superficie nativa do prodex runtime proxy/gateway/app-server quando aplicavel, incluindo readiness real, StartSession real, event stream real, estado persistente e prova de provider calls/Smart Context/rotacao. O `app-server-broker` pode ser uma pista futura, mas em 0.246.0 ele ainda reporta `status: skeleton` e `stdio-planned`.
