# Runbook — Isolamento de Credencial por Terminal nas Panes do Herdr

> Documento de narrativa + causa-raiz + solução + aplicação, para replicar o fix em
> **qualquer instância do Herdr** que sofra o mesmo problema.
> Autor: Kiro (TL). Verificado na fonte em 2026-07-13.

---

## 1. Sintoma (a "problemática")

Rodando muitos agentes no Herdr (8, 10, 15… até 50), com metade **codex** e metade **outro
vendor** (Cline/GLM, Antigravity/agy, OpenCode), acontece o seguinte em cadeia:

1. Agente 1 faz login OAuth (ex.: codex) numa pane e começa a trabalhar — OK.
2. Você loga a **conta 2** do mesmo vendor em outra pane.
3. A pasta de credencial do vendor é **sobrescrita** com os dados do último login.
4. O Agente 1 **perde a sessão** ("logo demais para de trabalhar"), precisa relogar.
5. Ao relogar o Agente 1, ele derruba o Agente 2 — e assim sucessivamente. O processo
   inteiro "caga", especialmente após dezenas de logins/logout no mesmo dia.

Efeito: **impossível manter N agentes do mesmo vendor logados ao mesmo tempo**; perda de
rastreabilidade de qual conta está em qual agente.

---

## 2. Causa-raiz (o "porquê", com precisão)

Existem **dois caminhos** de credencial no sistema. Só um deles está protegido:

### 2a. Caminho do DAEMON (sessões que o multica spawna) — JÁ ISOLADO ✅
Quando o daemon do multica inicia um agente para uma task, ele monta um HOME isolado por
conta e **copia** (nunca symlink) a credencial:
- `server/internal/daemon/daemon.go` → `requiresCredentialIsolation()` cobre os 6 vendors
  (`codex, kiro, antigravity, glm, cline, opencode`, + `nim`).
- `server/internal/daemon/execenv/*_home.go` → `codex_home.go` (`seedAccountAuth` copia
  `auth.json` por conta; comentário explícito: symlink faria o refresh de uma conta
  clobberar as outras), `cline_home.go`, `antigravity_home.go`, `opencode_home.go`,
  `kiro_home.go`.
- Coberto por `runtime_isolation_test.go`.
**Este lado NÃO é o problema.**

### 2b. Caminho do LOGIN MANUAL nas panes do Herdr — QUEBRADO ❌ (a causa real)
Quando **você (ou o agente) roda o CLI direto numa pane** (`codex`, `cline`, `agy`…) e faz
**OAuth login**, o daemon não está no meio. O CLI grava na pasta **default** do vendor
(`~/.codex`, `~/.cline`, `~/.gemini`) — **compartilhada entre todas as panes** — a menos que
o shell daquela pane tenha exportado a env var de isolamento do vendor.

A única proteção existente para esse caminho era um bloco em `~/.bashrc`:
```bash
# >>> herdr codex-home isolation (opus-4.8-orchestrator) >>>
case "${HERDR_PANE_ID:-}" in
  w3:pJ) export CODEX_HOME="$HOME/.codex-a" ;;
  w3:pM) export CODEX_HOME="$HOME/.codex-b" ;;
  w3:pK) export CODEX_HOME="$HOME/.codex-c" ;;
  w3:p9) export CODEX_HOME="$HOME/.codex-d" ;;
esac
# <<< herdr codex-home isolation <<<
```
Esse bloco tem **quatro falhas fatais**:

1. **Keyed em `HERDR_PANE_ID`, que NÃO é durável.** O próprio Herdr avisa: pane IDs
   **recompactam** quando panes fecham/abrem. Depois de dezenas de logins, os IDs deslizam
   (`w3:pM/pK/p9` deixam de existir; surge `w3:p18`), então o mapa **desalinha**.
2. **Sem branch `default`.** Qualquer pane **não listada** (ex.: `w3:p18`) cai no
   `CODEX_HOME` padrão = **`~/.codex` compartilhado** → sobrescrita.
3. **Só cobre codex.** **Cline, GLM, Antigravity/agy, OpenCode ficam de fora** →
   os ~50% "outro vendor" **compartilham `~/.cline`/`~/.gemini`** sem nenhum isolamento.
4. **Contas duplicadas entre homes** (ex.: `.codex` e `.codex-c` = mesma conta) → sem
   mapeamento limpo 1 conta : 1 home; o default vive sendo reescrito.

**Evidência empírica coletada (2026-07-13):** `~/.codex/auth.json` reescrito às 17:22 do dia
(caminho compartilhado, ativo); `~/.cline` e `~/.gemini` são diretórios **únicos** usados por
3 panes Cline + 1 agy simultaneamente.

> **Resumo da causa-raiz:** o isolamento do caminho de **login manual** estava (a) preso a um
> identificador volátil (pane_id), (b) sem fail-safe, e (c) incompleto (só codex). Basta uma
> pane cair no default compartilhado para reintroduzir a sobrescrita.

---

## 3. Solução (o "o quê")

Isolamento **físico, durável e por terminal** para **TODOS os vendors** no caminho de login
manual, reusando o **mapa de env autoritativo** que já existe no `execenv` (não reinventar):

| Vendor | Env de isolamento (fonte: `execenv/*_home.go`) | Store |
|---|---|---|
| Codex | `CODEX_HOME` | `auth.json` |
| Kiro | `XDG_DATA_HOME` (fork Amazon Q; ignora `KIRO_HOME`) | `kiro-cli/data.sqlite3` |
| OpenCode / GLM | `XDG_DATA_HOME` (+ `XDG_CONFIG_HOME`) | `opencode/auth.json` (GLM da frota roda pelo runtime compatível OpenCode) |
| Cline | `CLINE_DATA_DIR` (+ `CLINE_SANDBOX_DATA_DIR`) | data-dir próprio |
| Antigravity / agy | `HOME` (lê `~/.gemini/antigravity-cli`) | token em `~/.gemini/antigravity-cli` |

Componentes da solução:

1. **Âncora estável de terminal.** Usar o **`terminal_id` do Herdr** (ex.:
   `term_6565d1ec508f42`), estável mesmo quando o `pane_id` recompacta. Resolver no init do
   shell via `herdr pane get "$HERDR_PANE_ID"` → `.result.pane.terminal_id`. Fallback: `tty` +
   UUID persistido num marcador.
2. **Alocador de slots + registro** (`~/.agent-cred-homes/registry.json`, com `flock`):
   - Um **slot** por terminal (`slot-01/`, `slot-02/`…), cada um com subdirs por vendor.
   - **Idempotente:** terminal que já tem slot **reusa** (sem relogin).
   - **Monotônico:** terminal novo pega o **próximo slot livre**; **nunca sobrescreve o 1º
     login do dia** → rastreabilidade e auditoria.
3. **Export de env por vendor** apontando para o slot do terminal (tabela acima).
4. **Fail-safe default:** terminal **sem** mapeamento recebe **slot próprio**, **nunca** o
   diretório compartilhado do vendor. (Corrige a falha nº2 do bloco antigo.)
5. **Migração** dos homes atuais (`~/.codex*`, `~/.cline`, `~/.gemini`, `~/.config/opencode`)
   para slots **por cópia**, preservando os logins vivos.
6. **Doctor/status:** `status` mostra `pane → terminal → slot → vendor → conta → on/off`
   (o "quem está realmente on/off").

Implementação: um script sourced no `~/.bashrc` (ex.: `scripts/ops/agent-cred-isolation.sh`)
que substitui o bloco frágil. **É config de ambiente/frota — não altera código de produto do
multica.** O isolamento por-conta do daemon (§2a) permanece intacto.

---

## 3.1 Arquitetura técnica — AS-IS vs TO-BE

### AS-IS (antes) — login manual grava em diretório compartilhado
```
   Pane A (shell)          Pane B (shell)          Pane C (shell/agy)
      │  HERDR_PANE_ID         │ HERDR_PANE_ID          │
      ▼                        ▼                        ▼
   ~/.bashrc  case "$HERDR_PANE_ID"  (só codex, IDs fixos, SEM default)
      │ match → CODEX_HOME=~/.codex-a      │ no-match / não-codex
      ▼                                    ▼
   codex login ──OAuth──► ~/.codex-a    (cai no) ~/.codex  ~/.cline  ~/.gemini
                                           ▲        ▲         ▲
                                           └── COMPARTILHADO entre panes ──┘
   RESULTADO: 2º login sobrescreve o 1º → agente perde sessão → cascata.
```

### TO-BE (agora) — isolamento físico por terminal (slots)
```
   Pane (shell init)
      │  source scripts/ops/agent-cred-isolation.sh
      ▼
   [1] resolve terminal_id  ── herdr pane get $HERDR_PANE_ID ──► "herdr:<terminal_id>"
                                (fallback sem herdr) ──────────► "tty:<sha256(tty)>:<uuid>"
      ▼
   [2] flock(registry.lock) → aloca/reusa slot-NN  (monotônico; terminal conhecido reusa)
      ▼
   [3] export de env por vendor → APONTAM para ~/.agent-cred-homes/slots/slot-NN/…
      ▼
   vendor CLI login ──OAuth──► grava DENTRO do slot do terminal (nunca no compartilhado)
```
Mapa de env exportado por slot (TO-BE):

| Vendor | Env | Destino no slot |
|---|---|---|
| Codex | `CODEX_HOME` | `slot-NN/codex` |
| Kiro / OpenCode / GLM | `XDG_DATA_HOME` (+ `XDG_CONFIG_HOME`) | `slot-NN/xdg-data` (+ `xdg-config`) |
| Cline | `CLINE_DATA_DIR` (+ `CLINE_SANDBOX_DATA_DIR`) | `slot-NN/cline` (+ `cline-sandbox`) |
| Antigravity / agy | `HOME` | `slot-NN/home` (lê `~/.gemini/antigravity-cli`) |

### Componentes — quem fala com quem
```
 ┌─────────────┐  pane get   ┌──────────────────────┐  flock   ┌────────────────┐
 │ Herdr daemon│────────────►│ init script (bashrc) │─────────►│ registry.json  │
 └─────────────┘             │  agent-cred-isolation│◄─────────│ (+ registry.lock)│
                             └───────────┬──────────┘  slot NN └────────────────┘
                                         │ export env
                                         ▼
                             ┌──────────────────────┐
                             │ slots/slot-NN/{codex, │  ◄── OAuth login do vendor CLI grava aqui
                             │  cline, cline-sandbox,│
                             │  home, xdg-config,    │
                             │  xdg-data}            │
                             └──────────────────────┘

 (independente) caminho do DAEMON — sessões que o multica spawna:
 ┌───────────────┐ requiresCredentialIsolation + credentialAccountHomeForTask (FAIL-CLOSED)
 │ multica daemon│──────────────────────────────────────────────┐
 └───────────────┘                                               ▼
        Postgres (accounts/credentials/assignments) ──► execenv/*_home.go (CÓPIA, nunca symlink)
                                                          └──► home isolado por conta/tarefa
```

## 3.2 Metodologia (regras do algoritmo)
- **Chave de terminal (estável):** `herdr:<terminal_id>` (preferida — sobrevive à recompactação de
  `pane_id`); fallback `tty:<sha256(tty)>:<uuid-persistido>` quando o Herdr não responde.
- **Slot:** nome `slot-NN` (zero-padded); alocação **monotônica** via `next_slot` no registry;
  terminal já visto **reusa** seu slot (sem relogin); slot **nunca é reusado/sobrescrito** →
  preserva o 1º login e dá rastreabilidade.
- **Concorrência:** toda leitura/escrita do `registry.json` é serializada por `flock` no
  `registry.lock` (validado com 8 alocações simultâneas → slots únicos).
- **Migração:** cópia-uma-vez idempotente, dereferenciando symlinks, **sem** sobrescrever slot já
  inicializado; homes legados permanecem intactos (cópia, não move).
- **Fail-safe:** terminal sem mapeamento/errо na consulta Herdr recebe **slot privado próprio**,
  nunca o diretório compartilhado do vendor.

## 3.3 Inventário de pastas (real, verificado 2026-07-14)
Raiz de estado criada: **`~/.agent-cred-homes/`** contendo:
- `registry.json` (v1: `next_slot`, `terminals{}`, `slots{}`), `registry.lock` (flock),
  `slots/`, `fallback-terminals/`.

**7 slots criados** (`slot-01` … `slot-07`):
- **slots 01–05:** migrados dos homes codex legados (`.codex-a`, `.codex-b`, `.codex-c`,
  `.codex-d`, `.codex-slotA`) — contêm subdir `codex/`.
- **slots 06–07:** multi-vendor completos — subdirs `codex/ cline/ cline-sandbox/ home/
  xdg-config/ xdg-data/` (slot-06 = terminal Herdr; slot-07 = fallback tty).
- `fallback-terminals/<sha256(tty)>.uuid`: âncora persistida do modo fallback.
- Homes legados preservados por cópia: `~/.codex`, `~/.codex-a..d`, `~/.codex-slotA`, `~/.cline`,
  `~/.gemini` (não removidos — migração não-destrutiva).

## 4. Como aplicar o fix (passo a passo, replicável em qualquer instância Herdr)

> **Pré-requisitos no host da instância Herdr:** acesso shell; `herdr`, `python3`, `flock` e
> `git` no PATH. Os artefatos ficam neste repo — versione/pull, não cole à mão.

**Artefatos necessários (2 arquivos):**
- `scripts/ops/agent-cred-isolation.sh` (o fix — instalar/sourcar)
- `scripts/ops/tests/agent-cred-isolation-harness.sh` (validação)

**Passo 0 — obter os artefatos no host:**
```bash
# opção A: se o host tem o repo, basta atualizar
cd <clone-do-repo> && git pull origin main
# opção B: se não tem o repo, copiar os 2 arquivos para o host (ex.: em ~/ops/)
#   scripts/ops/agent-cred-isolation.sh  e  scripts/ops/tests/agent-cred-isolation-harness.sh
```
> Nas instruções abaixo, `SCRIPT=/caminho/para/agent-cred-isolation.sh` (ajuste ao caminho real).

**Passo 1 — Diagnosticar (confirmar que é o mesmo problema):**
```bash
grep -n "codex-home isolation" ~/.bashrc                       # bloco antigo (pane_id volátil)?
ls -1d ~/.codex* ~/.cline* ~/.gemini* ~/.config/opencode 2>/dev/null   # vendors não-codex num dir único?
stat -c '%y %n' ~/.codex/auth.json ~/.cline 2>/dev/null        # default compartilhado sendo reescrito?
```
Se houver o bloco `case "$HERDR_PANE_ID"` e `~/.cline`/`~/.gemini` forem diretórios únicos → é o mesmo bug.

**Passo 2 — Instalar (SOURCE no `~/.bashrc`, não "rodar uma vez"):**
```bash
SCRIPT=/caminho/para/agent-cred-isolation.sh
# 2a. remover o bloco antigo, se existir (backup incluso):
cp ~/.bashrc ~/.bashrc.bak.$(date +%s)
sed -i '/>>> herdr codex-home isolation/,/<<< herdr codex-home isolation/d' ~/.bashrc
# 2b. sourcar o novo script no FINAL do ~/.bashrc (idempotente: só adiciona se ainda não houver):
grep -qF "$SCRIPT" ~/.bashrc || printf '\nsource %s\n' "$SCRIPT" >> ~/.bashrc
```
> Importante: é `source` (roda no init de CADA shell/pane). Executar o script "solto" isola só
> aquele shell e não persiste.

**Passo 3 — Migrar os homes atuais (uma vez; cópia física, não-destrutiva):**
```bash
"$SCRIPT" migrate
```

**Passo 4 — Fazer as panes adotarem o isolamento:**
- **Panes NOVAS** já nascem isoladas (o `source` roda no init).
- **Panes já abertas** (e CLIs já rodando nelas) continuam no home antigo até você
  **recarregar o shell** (`source ~/.bashrc`) **e relançar o CLI** (`/quit` no codex → `codex`).

**Passo 5 — Validar (aceite empírico — só considerar resolvido se passar):**
```bash
bash /caminho/para/agent-cred-isolation-harness.sh      # espera: PASS: 6-vendor ... flock allocator
"$SCRIPT" doctor status                                  # slot certo por pane, on/off, sem tokens
python3 -m json.tool ~/.agent-cred-homes/registry.json   # 1 slot por terminal_id; 1º login intacto
```
Teste manual complementar: logar 2 contas do mesmo vendor em 2 panes ao mesmo tempo → ambas
seguem válidas; fechar/abrir panes (recompactação) → cada terminal mantém o slot sem relogin;
repetir para Cline e agy.

**Passo 6 — (SÓ se este host também roda o daemon multica) levar o fail-closed do daemon:**
O `.sh` cobre o login manual nas panes. O outro caminho (sessões que o **daemon multica** spawna)
depende do código Go (commit `a564651`, fail-closed). Se esta instância roda o multica:
```bash
cd <clone-do-repo> && git pull origin main
cd multica-auth-work && docker compose -f docker-compose.selfhost.yml up -d --build   # rebuild backend
```
Se a instância é **só panes de agente** (sem daemon multica) → pule este passo; o `.sh` basta.

**Passo 7 — Rastreabilidade:** `~/.agent-cred-homes/registry.json` mantém 1 slot por terminal,
monotônico; o 1º login do dia nunca é sobrescrito.

---

## 5. Validação e evidência reproduzível

### 5a. Gate empírico do caminho de login manual

Comando executado na raiz do repositório em 2026-07-13:

```bash
bash scripts/ops/tests/agent-cred-isolation-harness.sh
```

Saída esperada e obtida:

```text
PASS: 6-vendor migration, isolated dual login, Cline+agy, recompaction, fail-safe, and flock allocator
```

O harness usa processos shell separados como panes e o contrato real do `herdr pane get`.
Ele grava marcadores de login distintos nos stores nativos, sem depender de rede nem expor
segredos reais, e comprova:

| Gate | Evidência automatizada |
|---|---|
| Duas contas do mesmo vendor | panes A/B recebem `slot-01`/`slot-02`; escritas distintas em `CODEX_HOME/auth.json` permanecem intactas |
| Recompactação | `pane-a` e `pane-a-recompact` resolvem o mesmo `terminal_id`, reutilizam `slot-01` e preservam Codex, Cline e agy |
| Multi-vendor | Cline (`CLINE_DATA_DIR`) e agy (`HOME`) mantêm marcadores A/B independentes nas duas panes |
| Seis vendors | migração por cópia valida Codex, Kiro, Antigravity, GLM, Cline e OpenCode, inclusive SQLite sidecar do Kiro |
| Nunca symlink | uma origem Codex propositalmente symlinkada chega ao slot como arquivo regular e não altera a origem compartilhada |
| Fail-safe | falha na consulta Herdr aloca slots privados diferentes; nenhum vendor volta ao home compartilhado |
| Concorrência | oito alocações simultâneas produzem slots únicos e ownership consistente no registry protegido por `flock` |

### 5b. Inspeção operacional após instalar

```bash
/caminho/agent-cred-isolation.sh doctor status
python3 -m json.tool ~/.agent-cred-homes/registry.json
```

Critério: cada `terminal_id` possui exatamente um slot; a relação inversa em `slots` aponta
para o mesmo terminal; os paths exibidos ficam sob `~/.agent-cred-homes/slots/slot-NN/`.

---

## 6. Quando aplicar

Aplique **antes** de subir uma sessão multi-agente com **≥2 agentes do mesmo vendor** (ou
qualquer mix codex + outro vendor com login OAuth manual). Sinais de que a instância precisa:
- Bloco `case "$HERDR_PANE_ID"` no `~/.bashrc` (ou nenhum isolamento).
- `~/.cline` / `~/.gemini` / `~/.config/opencode` como diretório único.
- Agentes "deslogando sozinhos" após você logar outra conta.

Como o fix é **idempotente** e **fail-safe**, pode ser aplicado em qualquer instância Herdr
existente sem risco de derrubar os logins já ativos (a migração é por cópia).

---

## 7. Resolução executada neste host — registro fiel (2026-07-13)

Ordem exata do que foi feito até a resolução final (para replicar passo a passo):

1. **Diagnóstico na fonte (scan da frota):** confirmado o bloco `case "$HERDR_PANE_ID"`
   codex-only/stale no `~/.bashrc`; `~/.cline` e `~/.gemini` compartilhados; `~/.codex/auth.json`
   reescrito no próprio dia. Prova por processo (`/proc/<pid>/environ`): Codex-B em
   `CODEX_HOME=~/.codex-a`, Codex-A no default `~/.codex`.
2. **Incidente Codex-A:** token revogado (`401 refresh token was revoked`) — o próprio sintoma
   do bug (concorrência/sobrescrita de credencial).
3. **Workaround imediato (sem re-login destrutivo do B):** isolado o `CODEX_HOME` do Codex-A
   num home dedicado exclusivo `~/.codex-slotA` (`export CODEX_HOME` no shell da pane + relaunch
   do `codex`), garantindo que o login do A **não toca** o `~/.codex-a` do B. Verificado via
   `/proc/<pid>/environ`. (Isolamento permanente veio depois, pelo script — passos 4–5.)
4. **Deploy agêntico com 2 codex, arquivos disjuntos:**
   - **Codex-56-A (Go/daemon)** → commit `a564651 fix(daemon): enforce credential isolation
     fail-closed`: `credentialAccountHomeForTask` exige store+agentID+assignment+account+home_dir
     (senão erro **antes** do spawn); **cópia-não-symlink** para os 6 vendors; `custom_env`
     impedido de sobrescrever `XDG_CONFIG_HOME`/`CLINE_DATA_DIR`/`CLINE_SANDBOX*`;
     `runtime_isolation_test.go` cobre os 6 vendors.
   - **Codex-56-B (ops/shell)** → commit `e352985 feat(ops): isolate pane credential homes`:
     `scripts/ops/agent-cred-isolation.sh` + harness `scripts/ops/tests/agent-cred-isolation-harness.sh`;
     sourced no `~/.bashrc` (bloco antigo `case "$HERDR_PANE_ID"` **removido**).
5. **Validação pelo TL na fonte (não só o DONE do agente):**
   - Go: gate em container → `ok github.com/multica-ai/multica/server/internal/daemon`
     (`go build ./...` + testes de isolamento verdes).
   - Shell: harness → `PASS: 6-vendor migration, isolated dual login, Cline+agy, recompaction,
     fail-safe, and flock allocator`.
6. **Registro e rastreabilidade:** `.deploy-control/DEPLOY_PLAN_CRED_ISOLATION.md` (14/14 tasks ✅,
   com LIVE STATUS auto-atualizado) + check-ins START/DONE dos dois codex em `.deploy-control/`
   (`CHECKIN_Codex-56-A_*` e `CHECKIN_Codex-56-B_*`).

### 7.1 ANTES vs AGORA — estado real verificado (2026-07-14)

| Aspecto | ANTES (problemático) | AGORA (verificado no host) |
|---|---|---|
| `~/.bashrc` | bloco `case "$HERDR_PANE_ID"` (codex-only, IDs velhos `w3:pJ/pM/pK/p9`, sem default) | bloco antigo **removido**; `source .../scripts/ops/agent-cred-isolation.sh` **ativo** (linha 157) |
| Vendors não-codex | `~/.cline`, `~/.gemini` **compartilhados** (0 isolamento) | envs nativas por vendor no script (Codex/Kiro/agy/GLM/Cline/OpenCode); fail-safe default |
| Estado do isolamento | nenhum registro; panes caíam no `~/.codex` compartilhado | `~/.agent-cred-homes/` criado: `registry.json` (7 slots, `next_slot:8`), `registry.lock` (flock), `slots/`, `fallback-terminals/`; migração por cópia rodou (legacy `.codex-a/-b/-c/-d/-slotA`→slots 1–5, terminal herdr→slot 6, fallback tty→slot 7) |
| Daemon (sessões spawnadas) | isolava por conta, mas podia cair na credencial compartilhada sem atribuição | **fail-closed** endurecido (`a564651`); `runtime_isolation_test.go` 6 vendors verde |
| Codex-A (incidente) | token revogado (401), no `~/.codex` default | isolado em `~/.codex-slotA` (workaround); não toca o `~/.codex-a` do Codex-B |

**Caveat honesto (estado transitório das panes já abertas):** as panes que estavam rodando
**antes** do `source` ainda usam seus `CODEX_HOME` legados (Codex-A=`~/.codex-slotA`,
Codex-B=`~/.codex-a`, kiro=`~/.codex`) — **distintos entre si** (não há sobrescrita agora), mas
ainda **não sob `~/.agent-cred-homes/slots/slot-NN/`**. Elas adotam o slot pleno ao **recarregar
o shell (`source ~/.bashrc`) ou relançar o CLI**. **Panes novas já nascem isoladas** no slot, e o
**fail-safe** impede que qualquer login novo caia no diretório compartilhado. Os diretórios
legados permanecem porque a migração é por **cópia** (não move) — esperado e não-destrutivo.

> Nota sobre §2a: nesta resolução o lado do **daemon foi adicionalmente endurecido para
> fail-closed** (commit `a564651`) — antes ele isolava por conta mas ainda podia cair na
> credencial compartilhada sem atribuição; agora **não cai** (erra antes do spawn).

---

## 8. Referências (fonte de verdade)
- `docs/operations/FIX_ISOLAMENTO_CREDENCIAL_CENTRAL.md` — fix central por-conta (daemon).
- `openspec/changes/agent-credential-isolation/` — proposal/design/tasks/spec/auth-inventory.
- `multica-auth-work/server/internal/daemon/execenv/*_home.go` — mapa de env por vendor.
- `multica-auth-work/server/internal/daemon/daemon.go` — `requiresCredentialIsolation()`.
- `multica-auth-work/server/internal/daemon/runtime_isolation_test.go` — cobertura de teste.

> Status do script `scripts/ops/agent-cred-isolation.sh`: **implementado e validado em
> 2026-07-13**. O gate empírico reproduzível está documentado na seção 5.
