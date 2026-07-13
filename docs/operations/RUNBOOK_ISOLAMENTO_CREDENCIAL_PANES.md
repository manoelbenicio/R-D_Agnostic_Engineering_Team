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

## 4. Como aplicar o fix (passo a passo, replicável em qualquer instância Herdr)

> Pré-requisitos: acesso shell ao host do Herdr; `herdr` no PATH; `flock` disponível.

1. **Diagnosticar (confirmar que é o mesmo problema):**
   ```bash
   # bloco antigo (codex-only / pane_id volátil)?
   grep -n "codex-home isolation" ~/.bashrc
   # vendors não-codex compartilhando um único dir?
   ls -1d ~/.codex* ~/.cline* ~/.gemini* ~/.config/opencode 2>/dev/null
   # o default compartilhado está sendo reescrito hoje?
   stat -c '%y %n' ~/.codex/auth.json ~/.cline 2>/dev/null
   ```
   Se houver bloco `case "$HERDR_PANE_ID"`, e `~/.cline`/`~/.gemini` forem únicos → é o mesmo bug.

2. **Instalar o script de isolamento** (copiar `scripts/ops/agent-cred-isolation.sh` deste
   repo para o host) e **sourcá-lo no final** do `~/.bashrc`, **removendo** o bloco antigo
   `>>> herdr codex-home isolation >>>`:
   ```bash
   source /caminho/agent-cred-isolation.sh || return
   ```

3. **Migrar os homes atuais** rodando o modo de migração do script (copia física,
   dereferenciando symlinks, sem sobrescrever slot já inicializado):
   ```bash
   /caminho/agent-cred-isolation.sh migrate
   ```

4. **Recarregar os shells das panes** (novo `source ~/.bashrc` ou reabrir a pane) para que
   cada terminal resolva seu slot e exporte as envs.

5. **Validar (aceite empírico — só considerar aplicado se passar):**
   - Logar **2 contas do mesmo vendor** em 2 panes ao mesmo tempo → **ambas seguem válidas**;
     nenhuma sobrescreve a outra.
   - **Fechar/abrir panes** (forçar recompactação de `pane_id`) → cada terminal **mantém o
     slot** e continua logado (sem relogin).
   - Repetir para **Cline** e **agy**.
   - `agent-cred-isolation.sh status` mostra o slot certo por pane e o estado on/off de
     cada store nativo, sem ler nem imprimir tokens.

6. **Rastreabilidade:** conferir `~/.agent-cred-homes/registry.json` — 1º login do dia
   intacto; cada novo login em slot distinto.

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

## 7. Referências (fonte de verdade)
- `docs/operations/FIX_ISOLAMENTO_CREDENCIAL_CENTRAL.md` — fix central por-conta (daemon).
- `openspec/changes/agent-credential-isolation/` — proposal/design/tasks/spec/auth-inventory.
- `multica-auth-work/server/internal/daemon/execenv/*_home.go` — mapa de env por vendor.
- `multica-auth-work/server/internal/daemon/daemon.go` — `requiresCredentialIsolation()`.
- `multica-auth-work/server/internal/daemon/runtime_isolation_test.go` — cobertura de teste.

> Status do script `scripts/ops/agent-cred-isolation.sh`: **implementado e validado em
> 2026-07-13**. O gate empírico reproduzível está documentado na seção 5.
