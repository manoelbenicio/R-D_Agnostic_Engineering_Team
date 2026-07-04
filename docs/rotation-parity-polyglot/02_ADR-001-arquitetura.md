> **PLANO CORRIGIDO (2026-07-04) — leia antes de agir.** O plano anterior assumia o binario prodex instalado (ERRO). Correcoes herdadas:
> - **P0 FUNDACAO e pre-requisito de TUDO**: provisionar/buildar o binario prodex (source -> ~/runtime/prodex-src; instalar Rust; `cargo build --release`; verificar pin v0.246.0/7750da9b + hash; setar `MULTICA_PRODEX_ENABLED/PATH/VERSION/COMMIT` + `PRODEX_HOME`). Ref: Diligencias/00_FUNDACAO_P0.md + openspec specs/prodex-runtime-provisioning.
> - **Kill-switch + rollback: TESTADOS** (nao so documentados) antes do deploy.
> - **QA EXAUSTIVO em container ANTES do deploy** (NUNCA bypassado); deploy direto em PROD depois.
> - **OpenCode: ARQUIVADO** (sucessor Crush) -> disabled/descope (decisao F5).
> - **IPv6 desabilitado** nos builds: `docker run --sysctl net.ipv6.conf.all.disable_ipv6=1 ...`.
> Fonte de verdade: `openspec/changes/rotation-parity-polyglot/` + `.planning/` + `Diligencias/`.

# ADR-001 — Camada de Rotação/Otimização: prodex AS-IS em PROD agora; alvo polyglot (Go L4 + Rust L2)

- **Status:** ACEITO (2026-07-04)
- **Decisores:** Dono do produto (Manoel Benicio) + Codex R&D Engineering Team + Orquestração (Opus 4.8)
- **Entrada:** `docs/rotation-parity-polyglot/01_PRD.md` + parecer R&D (`feedback-codex-rd-engineering.md`, `parecer-tecnico-rd-sem-prazo-custo.md`, `plano-operacional-multiagente-opus48.md`, `agentic-execution-prompt-framework.md`)

## Contexto

Multica (L4, Go) orquestra agentes sobre CLIs de vendors. O moat é rotacionar **contas de assinatura** antes do esgotamento de quota, sem custo metered. Gaps: reset-claim headless, **otimização de contexto/token-saver mandatório**, cobertura multi-vendor não-uniforme, governança.

`prodex` (Apache-2.0, Rust) é a peça open-source mais completa: multi-conta/provider, auto-rotação pré-commit, afinidade de sessão, `redeem`/`--auto-redeem`, **Smart Context/token-saver**, modos, gateway OpenAI-compat, MCP. É Rust porque o caminho quente (proxy/Smart Context/gateway) exige ausência de GC, segurança de concorrência e cauda p99 previsível.

## Decisão

**Duas horizontes, decididos pelo dono:**

### Agora (near-term) — prodex AS-IS em PROD
- Deploy do **`prodex` AS-IS, pinado por versão/commit**, com o **Multica Go (L4) orquestrando** (lança `prodex`/`prodex s` no lugar de `codex` cru).
- Objetivo: obter **imediatamente todas as features** (rotação, reset-claim, **token-saver/Smart Context**, modos) usando as assinaturas próprias.
- **Sem fase de teste/staging dedicada** — decisão explícita do dono para otimizar tempo; **ajusta-se em PROD**.
- Guarda-corpos (config da ferramenta, não fase nova): controles **nativos** do prodex — Smart Context em shadow/canary configurável, **kill switch**, logs scrubbed.

### Alvo (próximo marco) — polyglot productizado
```text
Multica Go L4 (control plane, FRIO)          Rust L2 (runtime plane, QUENTE)
  - tenants, approved accounts, policies         - runtime proxy / gateway
  - workspaces, orchestration, Postgres          - session/profile affinity
  - dashboards / observability agregada          - precommit routing + fallback
  - inicia/para/monitora o L2                     - Smart Context (shadow/canary/live)
  desired state ───▶                              - reset-claim/redeem (guardado)
                ◀─── eventos (observabilidade/ledger, não redecisão)
```
- Endurecer via **fork do `prodex`** (Apache-2.0, com atribuição, rebrand do produto); partes que não atenderem invariantes são reescritas em Rust dentro do fork, preservando contratos/fixtures.
- **Fronteira:** sidecar local (HTTP/gRPC-like JSON sobre loopback, bearer efêmero, schema versionado). Não FFI; não subprocesso-por-request.
- **Invariante central — um único roteador por sessão:** Go decide desired state; Rust decide o request em voo. Eventos do Rust voltam ao Go só como observabilidade/ledger.

## Escopo e prioridades
- **Vendors:** Codex, Kiro, Antigravity, **Cline, OpenCode**. **Kimchi REMOVIDO** do escopo.
- **Reset-claim:** é **caminho frio e aleatório** (só ocorre em janela específica) → **prioridade mais baixa** que todo o resto, mas **será feito** (via `prodex redeem` + validação empírica com contas reais quando o estado ocorrer).
- **State compartilhado:** **Postgres** (SQLite proibido — histórico de lock forçou o upgrade antecipado).
- **Matriz de capabilities por provider** (não interface genérica): `launch_mode / auth_mode / quota_mode / rotation_mode / continuation_mode / smart_context_mode / reset_claim_mode`.

## Consequências
- **Reverte "tudo em Go"** no caminho quente — decisão consciente do dono; prioriza robustez/escala/performance sobre trabalho já feito.
- **Aposenta como runtime** o rotation-router Go (policy/fallback/loadbalance/proactive_reset): a autoridade runtime passa ao prodex/Rust L2. O Go retém **cadastro/policy/approved-accounts/observability** (control plane). Ver supersede em `openspec/changes/rotation-router`.
- Passamos a operar (agora) e manter (marco) um artefato **Rust** — **staffing Rust confirmado** disponível.

## Riscos e mitigações
| Risco | Mitigação |
|-------|-----------|
| Deploy direto em PROD sem staging → Smart Context pode corromper sessão | knobs nativos do prodex (shadow/canary) + kill switch + logs scrubbed + rollback documentado |
| Bus-factor 1 do prodex + churn diário | pin por versão/commit; fork Apache-2.0 pronto para assumir; SBOM/cargo-deny/gitleaks/attestation |
| Drift do Codex upstream quebrar proxy | prodex acompanha com release diária (agora); compat-watch no marco do fork |

## Alternativas consideradas
| Opção | Motivo |
|-------|--------|
| B — tudo em Go (Smart Context em Go) | recusada: risco semântico/protocolo + hot path inferior + meses |
| E — migrar Go→Rust | recusada: L4 é frio; custo sem ganho; zera validação |
| **A (agora) + D (alvo)** | **escolhida**: as-is em PROD já entrega valor; polyglot/fork endurece depois |

## Itens fora do escopo / não-aplicáveis
- **ToS jurídico:** irrelevante — **não há Claude Code** no projeto; Opus roda via **Kiro (AWS)**. Tema excluído.

## Invariantes que não podem regredir
- Isolamento OAuth/profile via `CODEX_HOME`/`XDG`/`HOME`; **fail-closed** em troca de perfil.
- Rotação **apenas pré-commit**; afinidade de continuation vence heurística.
- Sem segredo em log/trace/evidência; caminhos absolutos; container verde antes de DONE; migrations reversíveis.

## Referências
- PRD: `docs/rotation-parity-polyglot/01_PRD.md`
- OpenSpec: `openspec/changes/rotation-parity-polyglot/`
- Parecer/planos R&D (workspace do dono): `feedback-codex-rd-engineering.md`, `parecer-tecnico-rd-sem-prazo-custo.md`, `plano-operacional-multiagente-opus48.md`
- prodex: github.com/christiandoxa/prodex (Apache-2.0)