# Design — Rotation-Parity Polyglot

> Base: proposal.md + ADR-001 + parecer R&D. Regra: nada inventado; comportamento de vendor só de
> fonte primária; sem segredo em log; caminhos absolutos; verde-em-container/PROD antes de DONE.

---

## 1. Camadas e autoridade
```
 L4  Multica (Go) — CONTROL PLANE (frio)     |  L2  prodex/Rust — RUNTIME PLANE (quente)
   tenants, approved accounts, policies       |    runtime proxy / gateway
   workspaces, orchestration, Postgres        |    session/profile affinity
   dashboards/observability agregada          |    precommit routing + fallback
   inicia/para/monitora o L2                   |    Smart Context (shadow/canary/live)
        desired state ─────────────────▶      |    reset-claim/redeem (guardado, baixa prio)
                       ◀───────────────── eventos runtime (observabilidade/ledger)
```
**Invariante central — um roteador por sessão:** o Go envia *desired state* (contas permitidas,
policies, budgets, kill switches); o `prodex`/Rust decide o *request em voo* (afinidade, fallback
pré-commit, Smart Context, redeem). Eventos do Rust voltam ao Go **apenas** como observabilidade/
ledger — nunca para redecidir request já em voo.

## 2. Horizonte AGORA — prodex AS-IS em PROD
- Multica lança `prodex run --profile <x>` / `prodex s` no lugar de `codex` cru (usa assinatura, não metered).
- `prodex` **pinado** por versão+commit; instalação verificada (integridade/attestation).
- Isolamento por perfil preservado (`$PRODEX_HOME/profiles/<name>`, `CODEX_HOME` por conta).
- Features ligadas: rotação pré-commit, afinidade, **Smart Context/token-saver**, modos, reset-claim.
- **Guarda-corpos (config nativa do prodex, não fase de teste):**
  - Smart Context em `PRODEX_SMART_CONTEXT_SHADOW=1` ou `PRODEX_SMART_CONTEXT_CANARY_PERCENT=N` no rollout inicial;
  - **kill switch** por tenant/provider/profile;
  - logs scrubbed; rollback documentado (voltar a `codex` cru se necessário).

## 3. Horizonte ALVO — Rust L2 via fork do prodex
- Fork Apache-2.0 (com atribuição, rebrand do produto). Partes fora dos invariantes → reescritas em
  Rust dentro do fork, preservando contratos/fixtures.
- Fronteira Go↔Rust: **sidecar local HTTP/gRPC-like JSON sobre loopback**, bearer efêmero de alta
  entropia, schema versionado, health/readiness. **Não** FFI; **não** subprocesso-por-request.

## 4. Contrato Go ↔ L2 (mínimo, versionado)
```
HealthCheck            → liveness/readiness do sidecar
ApplyPolicy            → Go empurra desired policy/budgets/kill-switch
RegisterAccounts       → Go empurra contas aprovadas por tenant (approved accounts)
StartSession/StopSession → Go inicia/encerra sessão de runtime
RouteDecisionEvent     → Rust reporta seleção/afinidade/fallback (observabilidade)
RuntimeEventStream     → Rust emite eventos (selection, redeem attempt, rewrite decision, spend/savings, guardrail)
KillSwitch             → Go desliga Smart Context/gateway/auto-redeem por tenant/provider/profile
```

## 5. Matriz de capabilities por provider (não interface genérica)
```
ProviderCapability {
  launch_mode:      native_cli | codex_provider_bridge | openai_compatible_api
  auth_mode:        oauth_profile | api_key | cloud_iam | cli_native_store
  quota_mode:       codex_usage | vendor_balance | rate_limit_headers | custom_probe | none
  rotation_mode:    profile_pool | key_pool | gateway_route | unsupported
  continuation_mode:response_id | session_id | cli_thread | none
  smart_context_mode: proxy_rewrite | pre_tool_output_filter | disabled_shadow_only
  reset_claim_mode: codex_redeem | unsupported
}
```
Cada vendor recebe seu perfil de capability (verificado em fonte primária). Codex é o mais maduro;
Kiro/Antigravity/Cline/OpenCode entram por seus caminhos próprios (native CLI / provider bridge /
API-compatible), não como "igual ao Codex".

## 6. Smart Context / token-saver (mandatório)
- Fornecido pelo prodex (as-is agora; fork depois). **Modelo de segurança:** campos de controle/
  continuation/tool-call são exatos; rewrite só em segmentos elegíveis; **fallback exato** quando a
  integridade estrutural/protocolo é afetada. Rollout via shadow/canary nativos.

## 7. Reset-claim (baixa prioridade — frio/aleatório)
- Via `prodex redeem <profile>` / `--auto-redeem` (guarda: só weekly-exhausted, sem outro perfil
  elegível, reset não iminente). **Eficácia validada empiricamente** com contas reais quando o estado
  ocorrer (matriz: sem crédito / com crédito / perto do reset / weekly exhausted / 5h-only / todos
  exhausted / provider não-OpenAI). Guardas: idempotência, cooldown, audit event, sem redeem em
  thin/critical se há outro perfil elegível.

## 8. State e segurança
- **Postgres** para estado compartilhado (gateway/ledger/approved-accounts). SQLite proibido.
- Sem segredo em log/trace/evidência/checkin. Auditoria de: account selection, redeem attempt,
  fallback, continuation binding, context-rewrite decision.

## 9. O que REUSA vs SUPERSEDE
| Área | Reusa (Go, frio) | Supersedido (vira runtime do prodex/Rust) |
|------|------------------|-------------------------------------------|
| Isolamento credencial | execenv (`CODEX_HOME`/`XDG`/`HOME`) | — |
| Cadastro/approved accounts | accounts + migration 124 | — |
| Observability agregada | credential_metrics + dashboards | — |
| Seleção/rotação runtime | — | policy/fallback/loadbalance/proactive_reset (Go) → prodex L2 |
| Reset-claim | — | proactive_reset gated (Go) → `prodex redeem` |
| Smart Context | (não existia) | prodex L2 |

## 10. Decisões travadas
1. `prodex` AS-IS em PROD agora; fork/polyglot no próximo marco.
2. Um roteador por sessão (Go desired-state; Rust runtime).
3. Postgres para estado compartilhado.
4. Vendors: Codex/Kiro/Antigravity/Cline/OpenCode; **Kimchi fora**.
5. Reset-claim = baixa prioridade, mas será feito.
6. Sem staging dedicado (ajusta em PROD) — mitigado por knobs nativos + kill switch.

## 11. Riscos / a validar
- Eficácia real de reset-claim e Smart Context sob carga PROD → validar com contas reais + evidência scrubbed.
- Drift do Codex upstream → prodex acompanha (agora); compat-watch no fork (marco).
- Conformance de provider parcialmente split no prodex (reconhecido nos docs dele) → promover contratos ao core no fork.
