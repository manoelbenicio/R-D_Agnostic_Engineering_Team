# Design — Rotation Router (metodologia fundamentada em Requesty, self-hosted)

> **Fonte da metodologia:** Requesty (doc pública + console próprio do dono, acesso 2026-07-03).
> **Regra:** copiar a MECÂNICA provada; NÃO integrar/pagar o Requesty. Tudo self-hosted,
> na camada de ASSINATURA (janela 5h/semanal/créditos/resets) — onde o Requesty (metered)
> não atua. Zero custo/dependência de terceiro.
> **Fundação reusada:** internal/rotation/* + execenv + metrics + observability stack (já verdes).

---

## 1. Mapa de camadas (onde o router vive)
```
 L4 Multica (orquestração: tasks/workspaces/dispatch)        ← produto, intacto
 L3 Agente CLI (codex/kiro/antigravity/opencode/cline)       ← roda o loop agêntico
 L2 ROTATION ROUTER  ◀── ESTE DESIGN (seleção de CONTA)      ← nosso moat
 L1 Vendors/modelos                                          ← providers
```
Requesty vive em L2 mas roteando MODELOS por billing metered. Nós roteamos CONTAS de
assinatura por estado de quota. Mesma mecânica, alvo e economia diferentes.

---

## 2. Modelo de dados — RotationPolicy (extraído do console, Tela "Edit Policy")
Requesty policy = `{name, type∈{fallback|loadbalancing|latency}, items[ordenados]}`,
cada item = `{model, retries(0–10), key_source}`. Adaptação nossa (item carrega ESTADO DE
QUOTA, não preço $/token):

```
RotationPolicy {
  name        string              // alias nomeado; ref "policy/<name>" (troca sem redeploy)
  type        enum { FALLBACK, LOAD_BALANCING, LATENCY }
  workType    enum { GENERAL, HEAVY, CHEAP, REVIEW }   // taxonomia (§4)
  items       []PolicyItem        // ORDENADOS (prioridade) para FALLBACK
}
PolicyItem {
  vendor          string          // codex|kiro|antigravity|opencode|cline|...
  accountRef      string          // conta específica ou "any-of-vendor"
  retries         int             // 0–10 (default 1) — mesma escala do Requesty
  weight          int             // usado só em LOAD_BALANCING (normalizado)
  credentialSrc   enum            // ordem de fonte de credencial (análogo "key selection")
}
```
- Os 3 `type` são mutuamente exclusivos por policy (igual Requesty).
- Reordenação = prioridade do fallback. Drag-handle no console deles → campo `position`.
- **NÃO** replicar preço $/token nos itens: nosso "custo" é quota-state (§6).

---

## 3. Mecânica de FALLBACK (parâmetros reais — doc pública Requesty)
```
retry por item:        0–10 (default 1)
backoff exponencial:   500ms → 1s → 2s → 4s
jitter:                ±10% (evita thundering herd)
failover imediato em:  erro NÃO-retryable (auth inválida, request inválido)
retry em:              timeout, rate-limit(429), erro transitório 5xx
esgotou o item →       próximo item da chain; esgotou a chain → ErrNoAccountAvailable
```
- Classificação retryable-vs-não é o gate: 429/timeout/503 = retry+backoff; 401/400 = pula já.
- REUSA `detector.go` (reativo) + `token_lifecycle.go` (liveness) pra classificar.

---

## 4. Taxonomia de policies por TIPO DE TRABALHO (validada no vídeo GLM5.2+GPT5.5)
```
 GENERAL  → rotação padrão do pool (prioridade por expertise)
 HEAVY    → back-end pesado → conta/vendor mais forte (ex: Opus 4.8 via Kiro)
 CHEAP    → docs/refactors simples → assinatura mais barata (ex: GLM/Cline)
 REVIEW   → revisão alta qualidade → melhor conta disponível, sempre delegada
```
O agente/task declara o workType; o router resolve a policy correspondente. Isso é feature
de produto ("modo econômico/veloz/confiável") trocável no console, sem redeploy.

---

## 5. Estratégias de seleção (os 3 tipos, adaptados à camada de conta)
```
 FALLBACK       → tenta contas EM ORDEM; próxima se a atual falhar/esgotar (o que já temos,
                  enriquecido com retry/backoff/jitter do §3)
 LOAD_BALANCING → distribui tasks por PESO. MELHORIA NOSSA: peso por "saúde de janela" —
                  equilibra as janelas 5h pra esgotarem JUNTAS = throughput agregado máx.
                  Consistência: hashing determinístico (xxhash) sobre trace_id/agent_id →
                  mesma task fica na mesma conta (afinidade → reuso de contexto/cache).
 LATENCY        → escolhe a conta/VENDOR mais rápida agora. Métrica: TTFT + velocidade de
                  geração, janela móvel ~1h ponderando recente, scoring otimista p/ conta
                  "fria" (sem dado). MELHORIA NOSSA: latency entre VENDORS (codex vs kiro
                  vs antigravity), não só entre modelos.
```

---

## 6. Estado de quota por item (o que substitui "preço" do Requesty)
Cada conta tem régua própria (já mapeado em BACKLOG-detection.md):
```
 codex/kiro-opus/kimi   → janela 5h + semanal (%, reset HH:MM / data)  [+ reset credits]
 kiro(aws)              → créditos mensais (X de Y)
 antigravity            → por-modelo, 5h + semanal
 cline/opencode         → rate-limit / provider subjacente (reativo)
```
O router lê esse estado (via probe/painel usage — `probe_codex.go` etc.) para: (a) ordenar/
escolher; (b) decidir proativo (§7); (c) alimentar analytics (§8).

---

## 7. TETO — melhorias exclusivas (acima do Requesty; ele não pode fazer)
```
 (1) ROTAÇÃO PROATIVA
     Requesty = reativo (falha o request → troca). Nós trocamos ANTES de falhar, lendo
     quota/banner/painel de uso (warnbanner.go + usage.go + probe_codex.go). ZERO request
     falho. Requesty não vê quota de assinatura → não consegue.
 (2) CLAIM-DE-RESET antes de rotacionar
     Conta esgotada com "N usage limit resets available" → CLAIMAR reset e MANTER a conta
     (zero context switch) ANTES de partir pra próxima. Rotação vira fallback; claim é 1ª opção.
     [DEP: mecanismo headless de claim — /usage é TUI-only; CONFIRMAR CONTRA BINÁRIO. Se não
      houver headless, claim fica manual/assistido; documentar.]
 (3) LOAD-BALANCE POR SAÚDE DE JANELA (vs peso estático do Requesty)
     Objetivo: maximizar throughput agregado do pool de assinaturas (não A/B de modelo).
```

---

## 8. Observabilidade (Tela "Analytics" + reforço do que já temos)
```
 4 métricas núcleo: Cost | Request Volume | Token Usage | Latency
 × faixa de tempo (7d/30d/mês/tri/ano/custom)
 × GROUP BY dimensão: conta | vendor | task | workspace | repo | agent-version
 × filtros
 + aba SAVINGS → KPI DO MOAT: "$ economizado rotacionando assinaturas vs metered"
```
- REUSA `credential_metrics.go` (rotation_total, all_accounts_exhausted, accounts_available,
  exhaustion_detected_total, ...) + dashboards (rotation.json etc.) + gen_dashboards.py.
- ADICIONAR dimensões task/workspace/repo/agent-version (headers estilo X-Requesty-*).
- **Savings KPI** é novo e é a métrica que prova o valor do produto em número.

---

## 9. Account Registry + Governança (Tela "Model Library")
```
 Registry: cada conta/vendor com metadados
   { vendor, accountRef, modelos servidos, capabilities, contexto, região,
     quota-model (§6), estado atual de janela }
 Governança: Approved vs All — quais contas um TENANT pode usar
```
REUSA a tabela `accounts` (migration 123) + `enroll_account.sh`. Adiciona camada de
"approved-per-tenant".

---

## 10. O que REUSA vs o que é NOVO
| Área | Reusa (já verde) | Novo neste change |
|------|------------------|-------------------|
| Fallback ordenado | service.go/pool.go (priority) | retry/backoff/jitter §3 |
| Detecção | detector.go, warnbanner.go, usage.go, probe_codex.go | classificação retryable §3 |
| Liveness | token_lifecycle.go | uso na seleção |
| Policies | — | RotationPolicy §2 + taxonomia §4 (NOVO) |
| Load-balance | — | por saúde de janela §5 (NOVO) |
| Latency | — | multi-vendor §5 (NOVO) |
| Observabilidade | credential_metrics + dashboards | dimensões + Savings §8 |
| Registry | accounts + enroll | approved-per-tenant §9 |
| Proativo | proactive.go (ledger) | + banner/painel + claim-reset §7 |

---

## 11. Decisões travadas
1. Requesty = **referência**, nunca vendor. Zero custo metered em produção.
2. Router vive em L2 (conta), não substitui Multica (L4).
3. Item de policy carrega **quota-state**, não preço.
4. `contract.go` estendido, não reescrito; hotspots (service/pool/daemon) = streams seriais.
5. Claim-de-reset depende de mecanismo headless a confirmar contra o binário (não inventar).

---

## 12. Riscos / a confirmar
- **Claim-de-reset headless**: `/usage` é TUI-only; confirmar via app-server RPC ou marcar
  como assistido. NÃO inventar comando.
- **Latency multi-vendor**: medir TTFT por vendor exige instrumentar o dispatch (execenv/daemon).
- **Governança per-tenant**: exige schema novo (approved_accounts) — migration adicional.
