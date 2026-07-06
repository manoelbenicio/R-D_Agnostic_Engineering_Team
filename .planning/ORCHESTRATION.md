# ORCHESTRATION — Plano de Paralelismo para 8 Agentes

> **Autoridade:** Este documento é fonte de verdade para orquestração, propriedade de arquivo, comunicação e paralelismo.
> **Data:** 2026-07-04 · **Config GSD:** mode=yolo, profile=quality, agents=8, agent_mode=independent.

---

## 1. Hierarquia de Comando (INEGOCIÁVEL)

```
   ┌─────────────────────────────────────────────────────────────┐
   │                        DONO (Manoel)                        │
   │                  Autoriza / Bloqueia / Decide               │
   └──────────────────────────┬──────────────────────────────────┘
                              │ (canal único)
   ┌──────────────────────────▼──────────────────────────────────┐
   │    TECH LEAD (TL) — Sr. SME Solutions Architect &           │
   │    SR. Agentic Planner Orchestrator Expert                  │
   │                                                             │
   │    Papel:                                                   │
   │    • Interface ÚNICA entre o dono e o time de agentes       │
   │    • Orquestra todas as fases (P0→P10)                      │
   │    • Atribui tasks aos agentes                              │
   │    • Resolve conflitos de arquivo/merge                     │
   │    • Valida evidência (re-roda, não confia no tail)          │
   │    • Ensina o time a usar Herdr                             │
   │    • Faz sign-off de cada GATE                              │
   │                                                             │
   │    Agente: opus-4.8-orchestrator · Pane: w3:pE              │
   └──┬────┬────┬────┬────┬────┬────┬────────────────────────────┘
      │    │    │    │    │    │    │
      ▼    ▼    ▼    ▼    ▼    ▼    ▼
   Agentes executores (NUNCA falam entre si)
   Cada agente fala SOMENTE com o TL via Herdr
```

**REGRA DE OURO:** Nenhum agente executor fala diretamente com outro agente. Toda comunicação passa pelo TL.

---

## 2. Roster de Agentes (8 executores + 1 TL)

| # | Agente | Pane Herdr | Fase(s) | Especialidade |
|---|--------|------------|---------|---------------|
| TL | `opus-4.8-orchestrator` | `w3:pE` | TODAS | Orquestração, validação, sign-off |
| 1 | `Codex#5.5#A` | `w3:pJ` | P1 (Contrato) | Go, schema JSON, contrato L2 |
| 2 | `Codex#5.5#B` | `w3:pM` | P2 (Fork-map), P9 (Reset-claim) | Rust, análise de crates, prodex |
| 3 | `Codex#5.5#C` | `w3:pK` | P0 (Fundação), P3 (Integração) | Go daemon, build, sidecar lifecycle |
| 4 | `Codex#5.5#D` | `w3:p9` | P7 (Deploy/DevOps) | CI/CD, Helm, docker-compose, runbook |
| 5 | `GLM#52#A` | `w3:pT` | P6 (QA/Conformance) | Testes, evidência, conformance C1-C6 |
| 6 | `GLM#52#B` | *(ver `herdr agent list`)* | P4 (State/Security) | Postgres, redaction, audit, segurança |
| 7 | `Gemini#PRO#31` | `w3:pN` | P5 (Vendor Matrix) | Research, fonte primária, decisões |
| 8 | `Gemini#Flash35` | *(ver `herdr agent list`)* | P8 (Ops/Evidence) | Status board, evidence index |

> **Nota:** Pane IDs podem mudar ao fechar/reabrir. Sempre re-confirmar com `herdr agent list`.

---

## 3. Protocolo de Comunicação via Herdr

### 3.1 — O TL DEVE ENSINAR cada agente ao iniciar

Quando o TL ativa um agente, DEVE enviar estas instruções:

```
Você é [NOME_AGENTE]. Sua ÚNICA interface é o Tech Lead (TL) no pane [PANE_TL].
REGRAS:
1. NUNCA fale com outro agente diretamente.
2. Para falar com o TL, use EXATAMENTE:
   herdr pane run [PANE_TL] 'sua mensagem aqui'
3. Para ler a resposta do TL:
   herdr pane read [PANE_TL] --source recent --lines 40
4. Para esperar o TL responder:
   herdr wait output [PANE_TL] --match "para [SEU_NOME]" --timeout 120000
5. Antes de tocar QUALQUER arquivo, faça sign-in:
   echo "SIGN-IN [SEU_NOME] $(date -u +%Y%m%dT%H%M%SZ)" >> .deploy-control/checkins/[SEU_NOME].log
6. Ao terminar uma task, faça sign-out:
   echo "SIGN-OUT [SEU_NOME] $(date -u +%Y%m%dT%H%M%SZ) DONE task [N.M]" >> .deploy-control/checkins/[SEU_NOME].log
7. NUNCA marque uma task como DONE sem evidência reproduzível.
8. Se bloqueado, PARE e avise o TL imediatamente.
```

### 3.2 — Comandos que o agente usa para falar com o TL

| Ação | Comando Herdr |
|------|---------------|
| **Enviar mensagem ao TL** | `herdr pane run w3:pE 'TL: [mensagem]'` |
| **Ler resposta do TL** | `herdr pane read w3:pE --source recent --lines 40` |
| **Esperar resposta** | `herdr wait output w3:pE --match "para [NOME]" --timeout 120000` |
| **Reportar task DONE** | `herdr pane run w3:pE 'TL: task [N.M] DONE com evidência em .deploy-control/evidence/[file]'` |
| **Reportar BLOQUEIO** | `herdr pane run w3:pE 'TL: BLOCKED em task [N.M] — razão: [motivo]'` |
| **Pedir decisão** | `herdr pane run w3:pE 'TL: DECISION NEEDED — [pergunta]'` |

### 3.3 — Comandos que o TL usa para falar com um agente

| Ação | Comando Herdr |
|------|---------------|
| **Enviar instrução** | `herdr pane run [PANE_AGENTE] 'para [NOME]: [instrução]'` |
| **Ler progresso** | `herdr pane read [PANE_AGENTE] --source recent --lines 80` |
| **Esperar agente terminar** | `herdr wait agent-status [PANE_AGENTE] --status done --timeout 300000` |
| **Ver todos os agentes** | `herdr agent list` (para atualizar pane IDs) |
| **Listar panes** | `herdr pane list` |

### 3.4 — Padrão de mensagem (convenção)

```
TL → Agente:  "para Codex#5.5#C: execute task 0.1 — mover source para ~/runtime/prodex-src"
Agente → TL:  "TL: Codex#5.5#C task 0.1 DONE — source movido, git rev-parse = 7750da9b"
Agente → TL:  "TL: Codex#5.5#C BLOCKED em task 0.3 — cargo build falhou: error[E0308]"
```

---

## 4. Propriedade de Arquivo (DISJUNTA — sem colisão)

### 4.1 — Hotspots SERIAIS (dono único, NUNCA em paralelo)

| Diretório/Arquivo | Dono exclusivo | Razão |
|-------------------|---------------|-------|
| `server/internal/daemon/daemon.go` | Codex#5.5#C | Hotspot integração |
| `server/internal/daemon/prodex.go` | Codex#5.5#C | Launcher prodex |
| `server/internal/daemon/l2_runtime.go` | Codex#5.5#C | Client L2 |
| `server/internal/daemon/execenv/` | Codex#5.5#C | Isolamento credencial |
| `server/internal/daemon/*_test.go` | Codex#5.5#C | Testes daemon |

> **REGRA:** Se outro agente precisar tocar nesses arquivos, DEVE pedir ao TL, que coordena com Codex#5.5#C para evitar conflito.

### 4.2 — Propriedade por fase (paralelo permitido)

| Agente | Arquivos que PODE tocar | Arquivos que NÃO pode tocar |
|--------|------------------------|-----------------------------|
| **Codex#5.5#A** (P1) | `docs/contracts/*`, `openspec/.../specs/l2-runtime-contract/`, schema.json | daemon/*, server/internal/ |
| **Codex#5.5#B** (P2) | `docs/prodex/*`, fork-map docs | daemon/*, server/internal/ |
| **Codex#5.5#C** (P0+P3) | `server/internal/daemon/*`, `server/internal/l2runtime/*`, build scripts | docs/contracts/*, docs/qa/* |
| **Codex#5.5#D** (P7) | `deploy/*`, `scripts/*`, CI configs, `.github/workflows/` | server/internal/*, docs/contracts/* |
| **GLM#52#A** (P6) | `.deploy-control/evidence/*`, `docs/qa/*`, test scripts | server/internal/*, deploy/* |
| **GLM#52#B** (P4) | `server/migrations/*`, `server/internal/storage/*`, `docs/security/*`, `docs/state/*` | daemon/*, l2runtime/* |
| **Gemini#PRO** (P5) | `docs/vendors/*` | server/*, deploy/* |
| **Gemini#Flash35** (P8) | `.deploy-control/`, status board, evidence index | server/*, deploy/* |

### 4.3 — Sign-in/Sign-out obrigatório

Antes de tocar em QUALQUER arquivo:
```bash
# Sign-in (antes de começar)
mkdir -p .deploy-control/checkins
echo "SIGN-IN $(whoami)__$(date -u +%Y%m%dT%H%M%SZ) FILES: [lista de arquivos]" \
  >> .deploy-control/checkins/$(whoami).log

# Sign-out (ao terminar)
echo "SIGN-OUT $(whoami)__$(date -u +%Y%m%dT%H%M%SZ) TASK: [N.M] RESULT: [DONE|BLOCKED]" \
  >> .deploy-control/checkins/$(whoami).log
```

---

## 5. Plano de Waves (Execução Paralela)

### Wave 0 — Fundação (BLOQUEIA TUDO) — 1 agente serial

```
┌─────────────────────────────────────┐
│  Codex#5.5#C → P0 (tasks 0.1–0.9)  │  ← BLOQUEIA todas as waves
│  Build prodex no fleet host         │
│  (192.168.1.27, NÃO neste host)     │
└────────────────┬────────────────────┘
                 │ P0 GATE verde
                 ▼
```

### Wave 1 — Paralelo máximo (5 agentes) — depende de P0

```
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ Codex#5.5#A      │  │ Codex#5.5#B      │  │ GLM#52#B         │
│ → P1 (Contrato)  │  │ → P2 (Fork-map)  │  │ → P4 (Security)  │
│ tasks 1.1–1.5    │  │ tasks 2.1–2.8    │  │ tasks 4.1–4.11   │
└────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘
         │                     │                      │
┌────────┴─────────┐  ┌───────┴──────────┐            │
│ Gemini#PRO       │  │ Gemini#Flash35   │            │
│ → P5 (Vendor)    │  │ → P8 (Ops)       │            │
│ tasks 5.1–5.4    │  │ tasks 8.1–8.2    │            │
└────────┬─────────┘  └────────┬─────────┘            │
         │                     │                      │
         └─────────┬───────────┘──────────────────────┘
                   │ Wave 1 GATES verdes
                   ▼
```

### Wave 2 — Integração (1 agente serial no daemon) — depende de P1

```
┌────────────────────────────────────────────┐
│  Codex#5.5#C → P3 (tasks 3.1–3.6)         │
│  SERIAL (hotspot daemon)                    │
│  Inclui Early Rotation (task 3.6 = REQ-40) │
└────────────────┬───────────────────────────┘
                 │ P3 GATE verde
                 ▼
```

### Wave 3 — QA Exaustivo (1-2 agentes) — depende de P3+P4+P5

```
┌─────────────────────────────────────────────┐
│  GLM#52#A → P6 (tasks 6.0a–6.9)            │
│  Smoke S1-S5 + Conformance C1-C6            │
│  + suporte de Codex#5.5#C para evidência    │
│    de container (rodar testes no fleet)      │
└────────────────┬────────────────────────────┘
                 │ P6 GATE verde (TODOS C1-C6 + S1-S5)
                 ▼
```

### Wave 4 — Deploy + Reset-claim (2 agentes paralelo) — depende de P6

```
┌──────────────────────┐  ┌──────────────────────┐
│ Codex#5.5#D          │  │ Codex#5.5#B           │
│ → P7 (Deploy)        │  │ → P9 (Reset-claim)    │
│ tasks 7.1–7.7        │  │ tasks 9.1–9.3         │
│ GATED ao dono        │  │ empírico, não bloqueia │
└────────┬─────────────┘  └────────┬──────────────┘
         │                         │
         └────────┬────────────────┘
                  │
                  ▼
```

### Wave 5 — Meta/Reconciliação (1 agente)

```
┌────────────────────────────────────┐
│  TL (opus) → P10 (tasks 10.1–10.2)│
│  Arquivar rotation-router          │
│  Reconciliar docs/board            │
└────────────────────────────────────┘
```

---

## 6. Eficiência Máxima — Métricas de Paralelismo

| Wave | Agentes ativos | Paralelismo | Duração estimada |
|------|---------------|-------------|------------------|
| 0 | 1 (Codex#5.5#C) | Serial | 4h |
| 1 | 5 (A+B+GLM#52B+Gemini+Flash) | **5x paralelo** | 6h (mais longo: P4) |
| 2 | 1 (Codex#5.5#C) | Serial (hotspot) | 4h |
| 3 | 1-2 (GLM#52A + suporte) | Semi-paralelo | 8h |
| 4 | 2 (D + B) | 2x paralelo | 4h |
| 5 | 1 (TL) | Serial | 1h |
| **TOTAL** | | | **~27h** (vs ~50h serial) |

**Ganho de eficiência:** ~46% redução vs execução puramente serial.

---

## 7. Regras Anti-Colisão (INEGOCIÁVEIS)

1. **Nunca 2 agentes no mesmo arquivo** — propriedade disjunta (§4).
2. **Hotspot daemon = dono único serial** — só Codex#5.5#C toca.
3. **Sign-in/out obrigatório** — antes e depois de tocar arquivo.
4. **Merge via TL** — se dois agentes precisam do mesmo arquivo, TL sequencializa.
5. **Build = fleet host** — NUNCA neste host (orquestração). Sempre `ssh 192.168.1.27`.
6. **DONE = evidência** — TL re-roda e confirma; não confia no tail (ERR-08).
7. **IPv6 OFF** — `--sysctl net.ipv6.conf.all.disable_ipv6=1` em todo `docker run`.
8. **Sem segredo em log** — scrubbing antes de commitar qualquer evidência.

---

## 8. Protocolo de Escalonamento

```
Agente encontra problema
       │
       ▼
É bloqueio? ──NÃO──→ Resolve sozinho, reporta ao TL quando DONE
       │
      SIM
       │
       ▼
herdr pane run w3:pE "TL: BLOCKED em task N.M — razão"
       │
       ▼
TL avalia: ──É decisão do dono?──SIM──→ TL escala ao dono
       │                                      │
       │                                      ▼
       │                              Dono decide
       │                                      │
      NÃO                                     │
       │                                      │
       ▼                                      ▼
TL resolve e responde ao agente ◄─────────────┘
```

---

## 9. Checklist do TL ao Iniciar Cada Wave

- [ ] Confirmar pane IDs com `herdr agent list`
- [ ] Enviar instruções de comunicação (§3.1) para cada agente da wave
- [ ] Confirmar que agentes anteriores fizeram sign-out
- [ ] Confirmar GATE da wave anterior está verde com evidência
- [ ] Atribuir tasks específicas a cada agente
- [ ] Confirmar que nenhum arquivo tem 2 donos na wave
- [ ] Monitorar progresso a cada ~30min via `herdr pane read`
