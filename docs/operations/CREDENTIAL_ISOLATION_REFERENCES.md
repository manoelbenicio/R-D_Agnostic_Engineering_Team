# Referências — Solução de Isolamento de Credencial por Conta (RPP)

> Índice documentado de TODAS as referências da solução (isolamento + rotação de credencial).
> Compilado por: Kiro/Principal · Gerado em: **2026-07-06 ~15:39 (UTC‑3)**.
> Nota de timestamp: `mtime` = data de modificação do arquivo no host (fleet `manoelneto-laptop`).
> Os arquivos do RD com mtime `2026-07-06 13:06–13:07` refletem **checkout/clone recente**, não a data de autoria;
> para a autoria/histórico, use as **datas de commit** na seção 4.

---

## 1. OpenSpec — a mudança aprovada (fonte da spec)
Local: `openspec/changes/agent-credential-isolation/`

| Arquivo | mtime |
|---|---|
| `proposal.md` | 2026-07-06 13:07:41 |
| `design.md` | 2026-07-06 13:07:41 |
| `tasks.md` | 2026-07-06 13:07:41 |
| `specs/agent-credential-isolation/spec.md` | 2026-07-06 13:07:41 |
| `auth-inventory.md` | 2026-07-06 13:07:41 |

## 2. Implementação de referência JÁ CONSTRUÍDA — AOP (`/mnt/c/VMs/Projects/AOP/control-plane`)
Fonte de verdade a portar (Fase 1 isolamento + Fase 2 rotação).

| Arquivo | mtime | Papel |
|---|---|---|
| `sessions_api/service.py` | 2026-06-26 08:49:02 | `_prepare_isolated_paths()` (0700, abs, config⊂home) + `_vendor_env()` (mapa por vendor) |
| `seats/pool.py` | 2026-06-26 06:02:06 | `Seat.get_env()` (HOME/XDG_CONFIG_HOME/SEAT_ID) + `SeatPool` (ref_count/lease anticolisão) |
| `rotation/models.py` | 2026-06-27 20:16:18 | `Account`, `VENDOR_PRIORITY` (codex>opus>antigravity), janela 5h/cooldown |
| `rotation/detector.py` | 2026-06-27 20:16:12 | detecção reativa (regex por vendor + HTTP 429) — codex/glm/antigravity |
| `rotation/service.py` | 2026-06-27 15:24:38 | orquestração da rotação |
| `rotation/trigger.py` | 2026-06-27 15:36:37 | gatilhos |
| `rotation/auth.py` | 2026-06-27 15:29:10 | `DeviceLoginAuthenticator` (logout/login/wait) |
| `rotation/pool.py` | 2026-06-27 15:23:19 | seleção por prioridade de expertise |
| `docs/30-COMPONENTES/36-ROTACAO-CONTAS-TOKEN.md` | 2026-06-27 15:39:43 | **spec autoritativa** da rotação (algoritmo, detecção, "todas esgotadas") |
| ADR‑009 | **não localizado** | referenciado no `design.md`; arquivo não encontrado na busca (a confirmar) |

## 3. RD — código, plano e histórico (`/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team`)

| Arquivo | mtime | Papel |
|---|---|---|
| `multica-auth-work/server/internal/daemon/execenv/codex_home.go` | 2026-07-06 13:07:25 | ponto cirúrgico: copy-vs-symlink do `auth.json` (fallback symlink = furo) |
| `multica-auth-work/server/internal/daemon/daemon.go` | 2026-07-06 13:07:25 | injeção de env por task (`CODEX_HOME`/`XDG_DATA_HOME`/`HOME`/`PRODEX_HOME`) + `credentialAccountHomeForTask` (rotationStore Postgres) |
| `multica-auth-work/server/migrations/123_rotation.up.sql` | 2026-07-06 13:07:38 | tabelas `rotation_accounts/credentials/assignments/events` (Postgres) |
| `multica-auth-work/server/internal/daemon/runtime_isolation_test.go` | 2026-07-06 13:07:26 | teste de isolamento (gate) |
| `docs/project/01-as-is.md` | 2026-07-06 13:06:40 | problema (auth.json global symlinkado) |
| `docs/project/02-to-be.md` | 2026-07-06 13:06:40 | solução desenhada (env nativo por conta, copy) |
| `.planning/RCA-2026-05-31-001.md` | 2026-07-06 13:06:33 | RCA (⚠️ é sobre falhas de API do CAO — **não** sobre isolamento de credencial) |

## 4. Git — commits relevantes (datas de autoria)
Repo: `R-D_Agnostic_Engineering_Team`

| Commit | Data (UTC‑3) | Assunto |
|---|---|---|
| `aa62401` | 2026-07-02 10:12:55 | checkpoint: rotation platform (**isolation+rotation**+observability) + prod-readiness |
| `3cad173` | 2026-07-03 21:46:37 | plan: rotation-router change (proposal+design+tasks) |
| `360cc3e` | 2026-07-04 08:44:58 | rotation-router Wave1+2 (policy/fallback/loadbalance/proactive-reset) |
| `31d50b9` | 2026-07-05 21:24:27 | rotation-parity: Smart Context real + D2/D3 |
| `9b6c3c1` | 2026-07-05 22:13:56 | P11 matriz vendors (Codex/Kiro/Antigravity/Cline/OpenCode) |
| `612aea4` | **não encontrado** | citado no relatório P0 como origem do `CREDENTIAL_ISOLATION.md`; **não existe neste repo** (era de outro checkout) |

## 5. Cobertura por vendor (verificada no `_vendor_env`/detector do AOP em 2026-07-06)
| Vendor | Isolamento env | Detecção rotação |
|---|---|---|
| Codex | ✅ `CODEX_HOME` | ✅ |
| Kiro | ⚠️ mapeia `KIRO_HOME` mas binário honra `XDG_DATA_HOME` | ❌ |
| Antigravity | ⚠️ só `HOME` genérico | ✅ |
| GLM | ❌ `{}` | ✅ |
| Cline | ❌ `{}` | ❌ |
| OpenCode | ❌ `{}` | ❌ |

## 6. Observações de integridade (honestas)
- `mtime` do RD ≈ checkout recente (2026-07-06 13:0x); autoria real → seção 4 (commits).
- `ADR‑009` e `612aea4`: **referenciados mas NÃO localizados** neste repo — pendente confirmar origem.
- `RCA-2026-05-31-001.md` **não** documenta o isolamento de credencial (trata de API do CAO).
- Múltiplas cópias do repo existem no host (ex.: `Agentic_Agnostic_Orchestrator_Tool/R-D_Agnostic_Engineering_Team`, `Automonous_Agentic/Mapeamento_New_Features/multica-src`, `New_Product/...`) — confirmar a árvore canônica antes de qualquer implementação.
