# RISK REGISTER / ISSUE LIST — SEV-0 Stop-Ship (Rotation-Parity Polyglot)

> Evidência-primeiro. Repo: `R-D_Agnostic_Engineering_Team/multica-auth-work` @ `52cdd87` (main).
> Placeholders do briefing (CI logs externos, incidents, release date) **NÃO fornecidos** → sinalizados como MISSING-ARTIFACT.

## Decisão atual: **NO-SHIP** (ver gates bloqueados abaixo)

## Issues

| ID | Sev | Título | Status | Evidência (path/comando) |
|----|-----|--------|--------|--------------------------|
| **ISSUE-001** | SEV-0 | **Binário prodex não existe** (deploy impossível) | OPEN — bloqueia | source `/tmp/prodex-audit-7750da9`@7750da9b presente; `target/release/prodex` **ausente**; Rust/cargo ausente. design §2 dizia "instalação verificada" (falso) |
| **ISSUE-002** | SEV-1 | Auth clobber — workers compartilhavam `~/.codex` | **RESOLVIDO** | 4 `auth.json` idênticos (mesmo `account_id`); fix: `CODEX_HOME` isolado → 4 contas distintas verificadas |
| **ISSUE-003** | SEV-2 | Dashboard `UnicodeEncodeError` em terminal não-UTF8 | **RESOLVIDO** | `PYTHONIOENCODING=ascii` → traceback; fix reconfigure UTF-8 + `--ascii`; QA 29/29; regressão testada |
| **ISSUE-004** | SEV-1 | Plano sem fundação (0 tasks rastreáveis; binário assumido) | **RESOLVIDO** | `openspec status` = no-tasks → agora 51 tasks + P0 fundação + specs |
| **ISSUE-005** | SEV-1 | **91 arquivos uncommitted** no produto (perda/rastreabilidade) | OPEN | `git status --porcelain \| wc -l` = 91 |
| **ISSUE-006** | SEV-1 | Gates de QA marcados DONE mas **plan-only/dry-run** (paperwork) | OPEN | G1/G5/G7/G10 check-ins DONE; orquestrador admitiu "plan-done, live-gated"; só `g10-container`+`g2` têm evidência empírica |
| **ISSUE-007** | SEV-2 | CI Go: gates de test/vet/security a confirmar | TO-VERIFY | `.github/workflows/ci.yml` tem job Go "Build"; falta confirmar `go test`/`go vet`/gitleaks |
| **ISSUE-008** | SEV-2 | OpenCode ARQUIVADO no escopo (dependência morta) | OPEN | F5: projeto arquivado, sucessor Crush; decisão descope pendente |

## Hotspots de risco (do inventário — foco de forense/teste)
| Módulo | Sinal | Risco |
|--------|-------|-------|
| `server/internal/daemon/daemon.go` | **4626 LOC** (arquivo único, hot path) | alto — difícil revisar; hotspot serial |
| `server/internal/l2runtime/client.go` | 762 LOC (caminho quente p/ prodex) | alto — integração runtime |
| concorrência | **78 arquivos** com goroutine/mutex/chan | médio-alto — corrida |
| `server/internal/handler` | 143 arquivos | superfície grande |
| auth/execenv | 47 arquivos | crítico (credencial/isolamento) |
| testes | 317 `_test.go` | presença boa; **cobertura % desconhecida** (medir) |

## MISSING-ARTIFACT (bloqueia diligência completa — solicitar)
- CI run logs (links) · incidents/bugs recentes · thresholds de performance · release date/timezone.
