# Design & Gap/Risk Report — Isolamento de credencial por conta

> ⚠️ **NOTA IMPORTANTE (leia primeiro):** Este projeto **NÃO usa Claude Code** como
> vendor. Onde "Claude"/"CLAUDE_CONFIG_DIR" aparecer abaixo, é material de referência
> histórico e **não faz parte do roster deste projeto**. O que temos é o **Antigravity**,
> que roda o modelo **Opus 4.6** (entre outros). Os vendors reais em escopo são:
> **Codex, Kiro, Antigravity** (Fase 2) e **Kimchi, OpenCode, Cline** (Fase 3).
> Trate qualquer entrada "Claude" como NÃO aplicável.

> ESCOPO (travado pelo dono, 2026-07-01):
> - **Produto = Multica, que já funciona.** NÃO reescrever, NÃO trocar pelo AOP,
>   NÃO mexer no "cérebro" (orquestração/canvas/dispatch). Mudança **cirúrgica**.
> - **Único alvo: o mecanismo de auth OAuth** — substituir a credencial global
>   compartilhada por **separação por conta** via as env vars nativas de cada CLI.
> - **Nunca alterar o source.** Trabalhar sempre em **cópia local na pasta raiz
>   do nosso projeto**; o desenvolvimento acontece na cópia.
> - AOP serviu apenas como **referência de pesquisa** do modelo; não é alvo de
>   implementação.
> - Ponto cirúrgico já mapeado: `execenv/codex_home.go` (symlink do auth.json
>   global) + `daemon.go` (~l.3380, injeção de env por tarefa).

> Pré-implementação. Baseado em investigação empírica no servidor vivo
> (192.168.15.6, 2026-07-01) + leitura do runtime Go, do frontend e do AOP.
> Objetivo: eliminar surpresas que quebrariam a app no meio do desenvolvimento.

## 0. DESCOBERTA DECISIVA — a feature JÁ EXISTE, construída, no AOP

`C:\VMs\Projects\AOP\control-plane` já implementa **todo o modelo** (Fase 1 + Fase 2),
não é preciso projetar do zero. Módulos existentes:

- **`seats/pool.py`** — `Seat` com **isolamento de credencial via `home_dir`/`config_dir`**
  próprios; `get_env()` injeta `HOME`, `SEAT_ID`, `VENDOR`, `TENANT_ID`, `SEAT_TOKEN`;
  `SeatPool` gerencia pool por (tenant, vendor) com `ref_count`/lease anticolisão.
- **`sessions_api/service.py`** — `_prepare_isolated_paths()` (valida abs + config
  dentro do home, cria com 0700) e **`_vendor_env()`** (mapa autoritativo por vendor):
  - codex → `CODEX_HOME=config_dir`
  - claude → `CLAUDE_CONFIG_DIR=config_dir`
  - gemini → `GEMINI_CONFIG_DIR=config_dir`
  - kiro → `KIRO_HOME=home_dir` + `KIRO_CONFIG_DIR=config_dir`
  - injeta ainda `HOME`, `XDG_CONFIG_HOME`, `AOP_SEAT_CONFIG_DIR`.
- **`rotation/`** (Fase 2 completa): `models.py` (`Account`, `AccountStatus`,
  `RotationReason`, `VENDOR_PRIORITY` codex>opus>antigravity, janela 5h/cooldown),
  `detector.py`, `trigger.py`, `service.py`, `auth.py` (`DeviceLoginAuthenticator`
  logout/login/wait_authenticated), `pool.py`, `assembly.py`.
- **`seats_api/` + `sessions_api/`** (router/repository/schema) e **web** já tem
  páginas `seats` e `sessions` + `SeatCards.tsx`.
- **Spec autoritativa:** `AOP/docs/30-COMPONENTES/36-ROTACAO-CONTAS-TOKEN.md` + ADR-009.
  Cobre detecção reativa (regex por vendor na tela + HTTP 429), proativa (QuotaLedger
  95%), algoritmo de rotação (lock→mark_exhausted→snapshot→logout→select_next→login→
  restore→resume), política por prioridade de expertise, e o caso "todas esgotadas"
  (park + wake em min(cooldown_until) + alerta).

### Implicação para o nosso change
Isto **NÃO é "escrever do zero" nem "copiar do Multica"**. É **portar/reusar o modelo
do AOP** (que já é o produto agnóstico-alvo, sem branding Multica) para o runtime que
roda os agentes. O AOP é a fonte de verdade; o `auth_routes.py` do AgentVerse é a versão
antiga/incompleta.

### Divergência crítica AOP-vs-realidade (validar antes de portar)
O `_vendor_env` do AOP tem as **mesmas suposições que testei como erradas no servidor**:
- **kiro** → AOP usa `KIRO_HOME`/`KIRO_CONFIG_DIR`, mas o binário real (fork Amazon Q)
  **ignora** ambos e honra **`XDG_DATA_HOME`** (comprovado). O `Seat.get_env()` seta `HOME`,
  o que pode cobrir o caso — mas `XDG_DATA_HOME` não é setado. **Testar.**
- **gemini** → AOP usa `GEMINI_CONFIG_DIR`; agente real é `agy` (antigravity) que lê
  `~/.gemini/antigravity-cli` e provavelmente só responde a `HOME`. `get_env()` seta
  `HOME` → pode funcionar, mas **não provado**.
- **codex** → `CODEX_HOME` ✓ (bate com o comprovado).
- **claude** → `CLAUDE_CONFIG_DIR`; store real no servidor **não localizado**. **Validar.**

Ou seja: o AOP já resolve a arquitetura, mas herda os **mesmos 4 pontos de validação**
de vendor da §6. A economia é enorme (arquitetura, rotação, pool, API, web prontos),
mas a validação por-vendor continua obrigatória.

## 0.5 Decisões travadas (respostas R&D + pesquisa de vendors, 2026-07-01)

- **Sem API externa a integrar.** O app de credenciais funciona igual ao sistema
  de Auth de cliente já existente (`auth_routes.py`): armazena o estado de sessão
  OAuth / config-dir por conta e o serve. Mesmo sistema, mesmos agentes, mesmos
  vendors. Login OAuth = **exatamente AS-IS** do sistema existente.
- **Armazenar/restaurar AS-IS**: formato bruto do vendor, estado idêntico. Sem
  parsing, sem transformação, sem masking no caminho de restore.
- **Não armazenar credencial já usada/expirada** (sem créditos → inútil).
- **Compartilhamento concorrente é permitido**: uma credencial válida serve
  N agentes durante seu ciclo (~5h). → **NÃO** impor lease de conta única
  (removido o risco R3/R4 de concorrência; é comportamento aceito e testado).
- **Múltiplas contas por vendor**: permitido e testado (semanas, sem issue).
- **Portabilidade confirmada por vendor** (pesquisa): Codex `auth.json` portável;
  Kiro aceita `KIRO_API_KEY` (env) além da sessão sqlite; Antigravity via token
  em `~/.gemini/antigravity-cli` (+ `HOME`). Store-as-is é válido para os três.
- **Formato estável pós-handshake**: o schema não muda entre logins; apenas o
  `access_token` interno é renovado pelo próprio CLI. Restore-as-is é suficiente.

## 1. Achado central (a premissa "uma pasta por conta" só é uniforme na aparência)

Cada CLI guarda credencial de um jeito diferente e responde a uma **alavanca de
isolamento diferente**. Um modelo único de "config_dir por conta" **não se
aplica igual** aos quatro. Verificado no binário real, em execução:

| Agente | Store real (no servidor) | Alavanca de isolamento | Verificado? |
|--------|--------------------------|------------------------|-------------|
| **Codex** | `~/.codex/auth.json` (arquivo plano, 0600) | `CODEX_HOME` → dir por conta | ✅ honra `CODEX_HOME` (mas o dir precisa pré-existir) |
| **Kiro** | `~/.local/share/kiro-cli/data.sqlite3` (tabela `auth_kv`) — é **fork do Amazon Q** | `XDG_DATA_HOME` → dir por conta | ✅ honra `XDG_DATA_HOME` (cria sqlite novo); **ignora `KIRO_HOME`** |
| **Gemini/agy** | `~/.gemini/antigravity-cli/antigravity-oauth-token` (JSON) | Provável `HOME` override; **sem flag de config-dir, sem subcomando de login** | ⚠️ não provado |
| **Claude** | Nenhum arquivo de credencial encontrado; sem keyring instalado | Desconhecida | ❌ não resolvido |

## 2. Causa-raiz da sobreposição (no runtime Go)

- `execenv/codex_home.go`: o `auth.json` é **symlinkado** do `~/.codex` global
  (`codexSymlinkedFiles = ["auth.json"]`). Mesmo o Codex, que já tem
  `CODEX_HOME` por tarefa, aponta o symlink para **um único** `auth.json`.
- `codex_user_skills.go` (comentário explícito): *"Codex is the only runtime
  whose HOME is redirected to a per-task directory"*. **Claude, Gemini e Kiro
  não recebem home isolado** — usam o HOME/dirs globais do daemon.
- **Ponto de injeção único e confirmado:** `daemon.go` (~linha 3380) monta o mapa
  `agentEnv` por tarefa e seta `CODEX_HOME`, `CURSOR_DATA_DIR`,
  `OPENCLAW_CONFIG_PATH`, etc. **É aqui** que as env vars de credencial por conta
  entram — a mudança é cirúrgica e localizada.

## 3. Mismatch contrato-vs-realidade (vai quebrar se copiar cru)

O modelo de referência (`infra/cao/auth_routes.py` + `resolveSessionEnv` no
frontend) assume env vars que **não batem** com os binários reais:

| Provedor | Contrato atual (cao/frontend) | Realidade no servidor | Ação |
|----------|-------------------------------|-----------------------|------|
| codex | frontend só seta `OPENAI_MODEL` (**sem** `CODEX_HOME`) | runtime usa `CODEX_HOME` ✓ | frontend precisa passar o config_dir → `CODEX_HOME` |
| kiro | `KIRO_HOME` | binário **ignora** `KIRO_HOME`; usa `XDG_DATA_HOME` | trocar para `XDG_DATA_HOME` |
| gemini | `CLOUDSDK_CONFIG`/`GEMINI_CONFIG_DIR` (modelo gcloud) | agente real é `agy` (antigravity), lê `~/.gemini/antigravity-cli` | isolar via `HOME`; validar |
| claude | `CLAUDE_CONFIG_DIR` | store não localizado no servidor | confirmar onde a credencial vive antes de codar |

## 4. Riscos e mitigações (cenários de desastre)

| # | Risco | Severidade | Mitigação |
|---|-------|-----------|-----------|
| R1 | Copiar o mapa de env vars do `auth_routes.py` cru → kiro/gemini não isolam e **continuam sobrepondo** (bug silencioso: parece isolado, não está) | Alta | Usar a tabela real da §1; testar cada CLI com dir vazio (logado-out esperado) antes de confiar |
| R2 | `CODEX_HOME` apontando para dir inexistente → **erro fatal** "path does not exist", agente não sobe | Alta | `mkdir -p` do dir da conta antes do spawn (o runtime já faz `MkdirAll` no codex_home; replicar para os demais) |
| R3 | Kiro sob concorrência: duas tarefas da mesma conta compartilham `data.sqlite3` → lock/corrupção | Média | 1 dir `XDG_DATA_HOME` por conta; se a mesma conta rodar em paralelo, serializar ou dir por (conta×tarefa) |
| R4 | Claude: store real desconhecido → isolamento pode não ter efeito | Alta | **Bloquear Claude da Fase 1** até localizar o store (login numa conta e diffar o FS); não é usado no roster atual do servidor |
| R5 | Token refresh: CLIs reescrevem o auth (`last_refresh`) no dir da conta; se for symlink para global, o refresh de uma conta contamina a outra | Alta | Cada conta = **cópia real**, não symlink para o global (inverter a decisão atual do codex_home.go para o caso multi-conta) |
| R6 | De-branding: remover "MULTICA_*" (env vars como `MULTICA_AUTOPILOT_RUN_ID`) quebra o CLI que as lê | Média | De-branding só em superfície segura; env vars de contrato do runtime **não** renomear sem trocar produtor+consumidor juntos |
| R7 | agy (gemini) login é interativo/OAuth sem subcomando → rotação automática (Fase 2) pode não ser automatizável | Média | Provar o fluxo de login do agy antes de prometer rotação automática para gemini |

## 5. Decisões de design propostas

1. **Tabela de provedores corrigida** (fonte de verdade nova), substituindo o
   `PROVIDERS` do `auth_routes.py`:
   - codex → env `CODEX_HOME`, store = dir (arquivo auth.json)
   - kiro → env `XDG_DATA_HOME`, store = dir (sqlite data.sqlite3)
   - gemini/agy → env `HOME` (a validar), store = `~/.gemini/antigravity-cli`
   - claude → a definir após localizar store
2. **Cópia por conta, não symlink global** (R5): o dir da conta é a fonte; o
   runtime monta/aponta para ele. Token refresh fica contido na conta.
3. **Piloto = Codex** (store simples, env var provada). Kiro em seguida
   (XDG provado). Gemini e Claude só após validação de store/login.
4. **Injeção** no ponto único do `daemon.go` (agentEnv), estendendo o padrão
   `CODEX_HOME` já existente para os demais provedores.
5. **Fase 2 (rotação)** keyed em `status`/`expires_at` que o discovery já expõe;
   só habilitada para provedores cujo login for automatizável.

## 6. Pendências de validação antes de codar cada provedor

- [ ] Claude: logar 1 conta e diffar o FS/keyring para achar o store real.
- [ ] Gemini/agy: provar isolamento via `HOME` e mapear o fluxo de login.
- [ ] Kiro: confirmar comportamento sob 2 tarefas concorrentes da mesma conta.
- [ ] Codex: confirmar que copiar `auth.json` (em vez de symlink) não quebra
  refresh nem `config.toml` trust_level por path.
