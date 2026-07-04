# Backlog — Detecção avançada de esgotamento (insight do operador, 2026-07-01)

Insight (Manoel): há DOIS sinais distintos do vendor sobre o limite de 5h, e eles
não são a mesma coisa. Devemos aproveitá-los como camadas adicionais de detecção.

## Sinais observados (Codex, exemplo real)
1. **Banner de aviso automático (ambiente/passivo):**
   `⚠ Heads up, you have less than 10% of your 5h limit left. Run /status ...`
   - Aparece SOZINHO na saída do agente ao se aproximar do limite.
   - Captável por screen-scrape do pane, SEM comando.
   - Valor: aviso ANTECIPADO → rotação proativa antes do hard-stop (zero interrupção).

2. **Saída do `/status` (ativo/sob demanda):**
   ```
   5h limit:  [██░░...] 10% left (resets 19:59)
   Weekly limit: 86% left (resets 14:59 on 8 Jul)
   Context window: 15% left (221K/258K)
   ```
   - Só aparece quando o comando `/status` é digitado no prompt do Codex.
   - Traz o HORÁRIO EXATO de reset (5h) + limite semanal + janela de contexto.
   - Valor: setar `CooldownUntil` EXATO (não estimar 5h) + detectar limite SEMANAL
     (a conta pode esgotar pela semanal mesmo com 5h disponível) + contexto.

## Camadas de detecção (estado atual vs proposto)
| Camada | Sinal | Fonte | Feito? |
|--------|-------|-------|--------|
| Reativa | "usage limit reached" / 429 | screen/API | ✅ detector.go |
| Proativa (ledger) | tokens_used vs janela 5h | contagem interna | ✅ proactive.go |
| Proativa (banner) | "<10% of 5h limit left" | screen-scrape ambiente | ⬜ TODO |
| Ativa (probe) | `/status` completo | injetar `/status` + parsear | ⬜ TODO |

## Itens de trabalho (fase futura)
- [ ] Banner regex por vendor: "less than N% of your 5h limit left" → sinal de
      pré-esgotamento (rotação proativa antecipada). Distinguir de "limit reached".
- [ ] Probe `/status`: mecanismo do daemon para injetar `/status` no pane e parsear:
      % restante 5h + horário de reset + limite semanal + janela de contexto.
- [ ] Usar o reset time parseado como `CooldownUntil` EXATO no rotation.Account.
- [ ] Modelar limite SEMANAL separado do 5h (duas janelas simultâneas por conta).
- [ ] Cada vendor tem seu texto próprio de banner/status → tabela de padrões por
      vendor, confirmada contra a tela real (mesma disciplina do detector atual).

## Modelos de cota REAIS por vendor (confirmado contra tela)

Cada vendor tem uma UNIDADE DE COTA diferente — não há regex único. O detector deve
ser por vendor, entendendo a régua daquele vendor.

### Codex (OpenAI) — janela de 5h (TEMPO)
- Comando: `/status`. Banner passivo: "⚠ Heads up, you have less than 10% of your 5h
  limit left. Run /status". `/status` detalha: "5h limit: 10% left (resets 19:59)" +
  limite semanal + janela de contexto.
- Régua: % restante da janela de 5h. Reset: horário do dia. Também há limite SEMANAL.

### Kiro (AWS) — créditos mensais (CRÉDITOS) — CONFIRMADO 2026-07-01
- Comando: `/usage` (probe ativo — NÃO é banner passivo).
- Saída real: "Estimated Usage | resets on 2026-08-01 | KIRO PRO+ / Credits
  (895.04 of 2000 covered in plan) / 44.8% / Overages: Disabled".
- Régua: créditos consumidos vs plano (ex.: 2000). Esgotou = créditos acabaram.
- Reset: DATA mensal (2026-08-01), não hora do dia.
- Gatilho antecipado sugerido: consumido >= 95% do plano (ou N créditos restantes).
- Parsing: extrair "X of Y covered in plan" → percent = X/Y; reset date "resets on YYYY-MM-DD".

### Antigravity (Google) — Opus 4.6 / Gemini 3.1 Pro / Gemini 3.5 Flash — CONFIRMADO 2026-07-01
- Comando: `/usage` → "Models & Quota". Régua: POR MODELO, com DUAS janelas
  simultâneas cada (per-week E per-5-hour cap). Pools separados por modelo:
  Gemini Pro, Gemini Flash, Claude/GPT.
- Saída real (exemplos):
  * "Gemini 3.5 Flash (High) [████...] 100.00% / Quota available"
  * "Claude Sonnet 4.6 (Thinking) [███...] 96.43% / 96% remaining · Refreshes in 1h 30m"
  * "Claude Opus 4.6 (Thinking) 96.43%"
  Account: mostrado no topo (ex.: beniciosmsnoel@gmail.com).
- Régua: % REMAINING por (modelo × variante de effort). Reset: DURAÇÃO relativa
  ("Refreshes in 1h 30m") — parsear "in Nh Nm" → now+delta.
- CRÍTICO: detecção precisa ser por (vendor × MODELO), não só vendor — um modelo
  pode estar 100% e outro em 96%. O gatilho olha o MODELO que o agente usa.
- Parsing: "<Model> (<Variant>) ... NN.NN% ... (Quota available | N% remaining ·
  Refreshes in Xh Ym)".
- Modelos observados no pool (completo): Gemini 3.5 Flash (Medium/High/Low),
  Gemini 3.1 Pro (Low/High), Claude Sonnet 4.6 (Thinking), Claude Opus 4.6 (Thinking),
  GPT-OSS 120B (Medium). "Quota available" = 100%/sem refresh; consumido mostra
  "N% remaining · Refreshes in Xh Ym" (duração relativa, conta pra baixo).

### Cline — GLM-5.2 (high) — CONFIRMADO 2026-07-01 (parcial)
- Plano: "ClinePass/glm-5.2 (high)". Régua: TOKENS (incluídos na assinatura).
- Saída real observada: "○ Plan / ████████ (0 tokens) $0.00 (included with subscription)".
- Régua: consumo de TOKENS no plano ClinePass, com custo em USD (incluído).
- Reset / limite total do plano: A CONFIRMAR (a linha mostra tokens consumidos e $,
  mas não o teto — precisamos do valor máximo do ClinePass ou do comando de uso completo).
- Isolamento de credencial: onde o Cline guarda a sessão = A CONFIRMAR (vendor novo,
  sem *_home.go na Fase 1).
- Auto-approve: "Auto-approve all enabled" observado (contexto, não cota).

## Implicação de arquitetura
- O "detector de pré-esgotamento" é POR VENDOR, com unidade de cota própria
  (tempo 5h / créditos mensais / tokens). NÃO usar um regex genérico.
- Alguns vendors usam PROBE ATIVO (`/status`, `/usage`) em vez de banner passivo →
  o daemon precisa injetar o comando periodicamente e parsear a resposta.

---

# Backlog original (banner passivo)

- `/status` consome uma interação — não pollar agressivamente; usar quando o banner
  de aviso aparecer ou perto do threshold do ledger.
- Textos dos vendors mudam → manter os padrões em config, não hardcoded.
- NUNCA logar conteúdo sensível ao parsear a tela.


## Dados OFICIAIS dos fabricantes (pesquisa direta, 2026-07-01)

### Kiro (kiro.dev/pricing + docs/billing) — CONFIRMADO
- Tiers de credito/mes: Free 50 | Pro 1.000 | Pro+ 2.000 | Pro Max/Power acima.
  (Pro+ = 2.000 bate com o /usage do operador: "895.04 of 2000".)
- Creditos NAO acumulam entre meses (reset mensal; unused e perdido).
- Consumo fracionario (incrementos de 0.01 credito).
- Overage $0.04/credito quando habilitado (operador: Overages Disabled).
- => regua = creditos consumidos vs teto do tier; threshold antecipado viavel
  (ex.: >=95% do teto). Reset = 1o dia do proximo ciclo mensal.

### Cline / ClinePass (docs.cline.bot) — DESCOBERTA IMPORTANTE
- ClinePass = $9.99/mes, entrega "2-5x os rate limits da API padrao" — e RATE-LIMIT
  (velocidade), NAO um teto fixo de tokens/mes. Nao ha "X de Y" publico.
- Fontes de terceiros classificam o teto exato como "unknown" (nao publicado).
- => Para o Cline NAO da para antecipar por % de um teto fixo. Deteccao correta =
  REATIVA (rate-limit/429/"too many requests" temporario) OU tokens do plano
  esgotados. O detector.go reativo (429) JA cobre parcialmente. Marcar percent-based
  como N/A para Cline ate haver um teto publicado/observado.


## PESQUISA PRIMÁRIA — vendors adicionais para PROD (2026-07-01)
Fonte: docs oficiais / GitHub oficial / npm oficial. Regra de ouro: nada inventado.
Cada item marcado [CONFIRMADO-PRIMARIO], [SECUNDARIO] ou [A VERIFICAR CONTRA BINARIO].

### KIMCHI (getkimchi/kimchi — kimchi.dev) — VENDOR NOVO, distinto de Kimi
- [CONFIRMADO-PRIMARIO] NÃO é o Kimi/Moonshot. É "CLI coding agent, inference layer,
  and spend console all in one. Powered by open-source models." Built on pi-mono SDK,
  conecta à infra LLM da kimchi. Binário standalone (bun build --compile); Homebrew
  getkimchi/tap/kimchi. Comando: `kimchi setup` (config), `kimchi` (launch).
- [CONFIRMADO-PRIMARIO] AUTH/ISOLAMENTO: API key resolve por (1) env `KIMCHI_API_KEY`
  (precede), (2) `~/.config/kimchi/config.json` campo `api_key`. Config/sessions/models
  em `~/.config/kimchi/harness/`. Tags/estado em `~/.config/kimchi/`. Ferment em `.kimchi/`
  no projeto. => isolamento provável via `XDG_CONFIG_HOME` (redireciona ~/.config) +
  `KIMCHI_API_KEY` por conta. [A VERIFICAR CONTRA BINARIO] se há env dedicada tipo
  KIMCHI_HOME (README não cita; só XDG implícito).
- [CONFIRMADO-PRIMARIO] MODELO DE COTA = SPEND (gasto medido), não janela 5h nem
  créditos fixos. É um "spend console": cada request é tag-eado com `phase:{name}`,
  `model:{model_id}` para "usage analytics and cost attribution". `KIMCHI_TAGS` para
  atribuição. NÃO há comando `/usage` nem teto publicado no README.
- IMPLICAÇÃO: sem % de teto fixo publicado → detecção proativa por % NÃO se aplica
  direto (igual filosofia Cline). Detecção correta = REATIVA (429/erro de saldo) OU,
  se a kimchi.dev expuser API de spend console, um probe HTTP à conta (fora do README).
  [A VERIFICAR CONTRA BINARIO] existência de `kimchi ... usage`/endpoint de spend.
- É multi-model orchestrator (orchestrator/builder/reviewer/explorer/researcher), com
  modelos kimchi-dev (kimi-k2.6, minimax-m2.7, nemotron-3-ultra-fp4) + externos
  (anthropic/openai). O limite efetivo pode vir do provider externo subjacente.

### OPENCODE (opencode.ai — github.com/anomalyco/opencode) — VENDOR NOVO (meta-agente)
- [CONFIRMADO-PRIMARIO] Agente de terminal open-source (TS+Go, roda em Bun/binário Go),
  75+ providers via AI SDK + Models.dev. Install: `curl -fsSL https://opencode.ai/install|bash`
  ou `npm i -g opencode-ai` ou `brew install sst/tap/opencode`.
- [CONFIRMADO-PRIMARIO] CREDENCIAIS: `~/.local/share/opencode/auth.json` (via `/connect`).
  `opencode auth list` lista. Config em `~/.config/opencode/opencode.json(c)`.
  => isolamento via `XDG_DATA_HOME` (auth.json) + `XDG_CONFIG_HOME` (config).
  [A VERIFICAR CONTRA BINARIO] confirmar que respeita XDG_DATA_HOME/XDG_CONFIG_HOME.
- [CONFIRMADO-PRIMARIO] É META-AGENTE: a cota efetiva é a do provider subjacente
  (Anthropic/OpenAI/Bedrock/Moonshot/etc.), não do OpenCode. Config por modelo tem
  `limit.context`/`limit.output` (janela de contexto, NÃO cota de plano).
- [CONFIRMADO-PRIMARIO] NÃO há comando de uso/quota agregado nativo: GitHub issue
  #8911 "Add OAuth rate limits and usage dashboard" está ABERTA => feature não existe.
  Rate-limit surge como erro (issue #11242 "Rate limit exceeded"). => detecção do
  OpenCode = REATIVA (429 do provider subjacente passado adiante).
- Output automação: TUI + CLI; suporta modo headless/serve e ACP. [A VERIFICAR] formato
  exato de evento de uso/erro no stdout headless.

### CLINE (github.com/cline/cline — docs.cline.bot) — CORREÇÃO de suposições antigas
- [CONFIRMADO-PRIMARIO] TEM CLI 2.0 (npm `cline`), não só extensão VSCode. Modos:
  interativo, one-shot, `--json` (NDJSON), `--yolo`, `--zen` (hub daemon).
- [CONFIRMADO-PRIMARIO] ISOLAMENTO NATIVO (melhor que supunhamos): `--data-dir <path>`
  (substitui `~/.cline`, ativa sandbox) e envs `CLINE_DATA_DIR`, `CLINE_SANDBOX=1`,
  `CLINE_SANDBOX_DATA_DIR`. NÃO precisamos criar cline_home.go — o vendor já isola.
- [CONFIRMADO-PRIMARIO] É META-AGENTE: wraps cline, openai-codex (ChatGPT sub),
  anthropic, gemini, bedrock, vertex, groq, kimi, etc. Cota efetiva = provider embaixo.
  Sinal de uso proativo = eventos NDJSON de `--json` (tokens/custo) + `-v` (stats).
- [CONFIRMADO-PRIMARIO] SEM comando `/usage` agregado headless. ClinePass = rate-limit
  (2-5x), sem teto fixo publicado (decisão anterior mantida: Cline = REATIVO/429).
  [A VERIFICAR CONTRA BINARIO] nomes exatos de campos de token/custo no NDJSON.

### KIMI / Kimi Code CLI (MoonshotAI/kimi-cli, kimi-code — kimi.com/code) — se entrar
- [CONFIRMADO-PRIMARIO] ISOLAMENTO LIMPO: `KIMI_CODE_HOME` define data root
  (`$KIMI_CODE_HOME/config.toml`), análogo a CODEX_HOME. Config default `~/.kimi-code/`.
  Credencial via config.toml `[providers.<name>]` (NÃO cai em `export KIMI_API_KEY`
  do shell — só dentro do config). Non-interativo: `-p/--prompt` + `--output-format
  text|stream-json`.
- [CONFIRMADO-PRIMARIO] Modelo atual = Kimi K2.7 Code (não K2.5). Cota = janela 5h
  rolante (família QuotaTime5h, igual Codex) + billing por token. [A VERIFICAR] string
  real de banner/erro de limite e se há `/usage`.
- ATENÇÃO: blog secundário citou install `@anthropic-ai/kimi-code` — [SECUNDARIO/SUSPEITO],
  escopo errado; usar fontes MoonshotAI oficiais.

## MATRIZ CONSOLIDADA (6 cenários PROD) — modelo de cota × isolamento × detecção
| Vendor       | Cota (régua)              | Isolamento (env)                 | Detecção proativa |
|--------------|---------------------------|----------------------------------|-------------------|
| Codex        | tempo 5h                  | CODEX_HOME                       | SIM (banner+/status) |
| Kiro (AWS)   | créditos mensais          | XDG_DATA_HOME                    | SIM (/usage) |
| Antigravity  | por-modelo 5h+semana      | HOME                             | SIM (/usage) |
| Kimi Code    | tempo 5h (+token bill)    | KIMI_CODE_HOME                   | PROVÁVEL (a confirmar cmd) |
| Cline        | provider subjacente / RL  | CLINE_DATA_DIR (nativo)          | REATIVO (429/NDJSON) |
| OpenCode     | provider subjacente       | XDG_DATA_HOME + XDG_CONFIG_HOME  | REATIVO (429) |
| KIMCHI       | SPEND (gasto medido)      | XDG_CONFIG_HOME + KIMCHI_API_KEY | REATIVO (spend/429) |

## DECISÃO DE ARQUITETURA (reforçada pela pesquisa)
1. São DUAS classes de vendor:
   - CLASSE A — cota própria por plano (Codex/Kiro/Antigravity/Kimi): detecção
     PROATIVA por % (usage.go + probe.go quando houver comando confirmado).
   - CLASSE B — META-AGENTE (Cline/OpenCode) e SPEND (KIMCHI): NÃO têm % de teto fixo;
     detecção REATIVA (429/erro do provider subjacente) via detector.go existente.
2. Isolamento: TODOS os novos têm mecanismo nativo (KIMI_CODE_HOME / CLINE_DATA_DIR /
   XDG_DATA_HOME / XDG_CONFIG_HOME). NÃO precisamos inventar *_home.go novo — mapear
   as envs reais no execenv (stream de isolamento), 1 entrada por vendor.
3. NÃO inventar comando de usage/probe de nenhum vendor novo. Onde não há comando
   confirmado, o vendor entra como CLASSE B (reativo) até confirmarmos contra o binário.


## CORREÇÃO KIMCHI (fonte: kimchi.dev homepage, 2026-07-01) — CLASSE C (spend budget)
- [CONFIRMADO-PRIMARIO] "No rate limits or token caps" (dito 2x) → NÃO existe janela 5h
  nem teto de tokens. Descarta detecção Classe A (CLI banner/%).
- [CONFIRMADO-PRIMARIO] TEM "Budget controls": spend caps por API key / user / team /
  org, com THRESHOLD % e forecast de run-rate. UI mostra "$4,820 / $6,000 cap · 80%" e
  "hits 80% budget tomorrow at current run rate". Billing $/1M tokens (kimi-k2.7 $4.50,
  minimax-m3 $1.20, nemotron-3 $0.75). Sinal vive no CONSOLE (app.kimchi.dev), NÃO na CLI.
- => KIMCHI é uma TERCEIRA CLASSE de detecção: SPEND BUDGET. Proativo é VIÁVEL, mas via
  API do console (spend vs cap por API key), não por texto de tela nem probe de CLI.
- Gatilho de rotação natural: key >= 80% do cap → rotacionar para outra API key/conta.
- [A VERIFICAR — NÃO INVENTAR] endpoint real de leitura de spend/budget em
  docs.kimchi.dev / app.kimchi.dev (a homepage mostra a UI, não o contrato de API).
  Enquanto não confirmado: KIMCHI cai em REATIVO (erro de budget/429) como fallback.
- Isolamento: KIMCHI_API_KEY por conta (cada key tem seu cap) + ~/.config/kimchi/.

## TRÊS CLASSES DE DETECÇÃO (modelo final)
- CLASSE A (plano/cota na CLI): Codex, Kiro, Antigravity, Kimi → proativo por % via CLI.
- CLASSE B (meta-agente): Cline, OpenCode → reativo 429 do provider subjacente.
- CLASSE C (spend budget via console): KIMCHI → proativo por % de $ cap via API do
  console (a confirmar endpoint), fallback reativo.


## TOPOLOGIA PROD — CONFIRMADA (Manoel, 2026-07-01)
- Cline / OpenCode / Kimchi rodam com ASSINATURA PRÓPRIA (standalone), NÃO como
  wrappers de outra conta. => cada um é um vendor de 1a classe com sua própria cota;
  a rotação acontece na conta/API-key DAQUELE vendor, não numa conta subjacente.
- Consequência por vendor:
  * Cline (ClinePass): rate-limit sem teto fixo publicado → detecção REATIVA (429).
  * OpenCode (assinatura própria — OpenCode Zen/Go): confirmar superfície de uso real
    contra o binário; até lá, REATIVO. [A VERIFICAR CONTRA BINARIO]
  * Kimchi: spend budget PRÓPRIO. Manoel confirma que o LAYOUT de token usage do Kimchi
    é MUITO SEMELHANTE ao do Codex (vai colar o texto real). => provável CLASSE A-like
    (parse de % na saída), não só console. Aguardando o texto real para extrair o padrão
    SEM inventar. Reclassificar Kimchi de "C-console" para "A-like CLI" se o layout colado
    confirmar % de uso na própria saída (igual "/status" do Codex).


## FRONTEIRA DE SPRINT (Manoel, 2026-07-01) — NÃO MISTURAR
- SPRINT ATUAL (Fase 2, em curso): EXCLUSIVO Codex + Kiro + Antigravity. Já planejado,
  código verde, integração proativa (banner+ledger) DONE. NÃO adicionar Kimchi/OpenCode/
  Cline aqui. O W-PROBE (probe ativo Kiro/Antigravity) permanece backlog do sprint atual.
- PRÓXIMO SPRINT (Fase 3): EXCLUSIVO Kimchi + OpenCode + Cline. Todo o planejamento
  desses 3 vive em docs/project/SPRINT-NEXT-vendors.md (novo). Nada de código agora.


## PESQUISA DE PRIOR-ART E SINAIS REAIS (fontes primárias, 2026-07-02)
Diretriz do dono: pesquisar SEMPRE na fonte (vendor site/GH, issues, repos de usuários,
YT) — somos early adopters, prior art é escasso e vive em issues/comunidade.

### Prior art que VALIDA nossa arquitetura (não estamos sozinhos)
- **subswap** (github.com/x0c/subswap, ATIVO v0.3.27 Jul/2026, Rust) — o mais próximo:
  * daemon de fundo que faz POLL de quota e AUTO-SWAP ao cruzar threshold  = nosso proativo.
  * isolamento por conta com AS MESMAS envs que usamos: `CODEX_HOME` (Codex),
    `CLAUDE_CONFIG_DIR` (Claude). → confirma que nossos levers estão certos.
  * INVARIANTES de design que devemos adotar:
    - "swap manual NUNCA depende de quota lookup" (escape hatch resiliente).
    - swap ATÔMICO com snapshot/rollback antes de tocar arquivos.
    - cache de quota com fallback stale (status responsivo se API lenta).
    - "adicionar provider = 1 crate + 1 linha" (nossa ideia de parser por vendor).
- Outros: `codex-switch` (Rust, proxy local + hot-load de OAuth ao bater limite),
  CCS "Account Pools" (auto-continue quando 1 conta bate limite; multi-conta p/ Gemini/
  Codex/Antigravity), OpenClaw codex-account-switcher (swap determinístico, quota-aware).
  → o padrão de mercado emergente == o que estamos construindo. Reforça a tese.

### SINAIS REAIS do Codex (openai/codex issues — validar/expandir nosso detector)
- Banner REAL de bloqueio: "Codex message usage limit reached / Please wait until HH:MM"
  (issue #23994). É DIFERENTE do "less than X% of your 5h limit left" (pré-aviso). Nosso
  detector deve casar AMBOS: pré-aviso (proativo) E "usage limit reached ... wait until"
  (reativo, com hora de reset).
- ⚠️ FALSO-POSITIVO CONFIRMADO (issue #23994): o Codex mostra "usage limit reached" MESMO
  com 5h/semana restantes, quando um teto MENSAL de créditos do ChatGPT-seat é atingido.
  Ou seja, o banner NEM SEMPRE é exaustão real de 5h → rotacionar só pela string pode
  rotacionar à toa. Nosso detector reativo deve, quando possível, cruzar com o /status
  ("5h limit: N% left") antes de tratar como esgotamento definitivo.
- Reset é por horário do dia (5h) + limite SEMANAL separado (issues #16423, #4080, #11508):
  confirma que Codex tem DUAS janelas (5h + semanal), como o Antigravity — o parser deve
  capturar as duas quando presentes.

### Item de backlog derivado
- Detector reativo: adicionar padrão "usage limit reached" + "wait until HH:MM" e a
  ressalva de falso-positivo do teto mensal de ChatGPT-seat (não é 5h).
- Robustez (inspirado no subswap): swap manual sem depender de quota; snapshot/rollback
  atômico; cache de quota com stale fallback. Avaliar p/ hardening pós-staging.
- Observabilidade do daemon (gap já achado): expor metrics server no processo daemon.


## CORREÇÃO + FATOS REAIS (operador, 2026-07-02) — contas, /status, token-reset surpresa
### Correção de um erro do Opus (registrar honestamente)
- Opus afirmou (ERRADO) que as 2 contas Codex na box eram "a mesma conta" com base em
  acct_fp igual. Isso estava INCORRETO: o operador confirma que são contas REAIS,
  DISTINTAS, PROD (emails OpenAI diferentes / assinaturas diferentes). O acct_fp que o
  Opus comparou reflete estrutura do token, NÃO prova mesma assinatura. Lição: não afirmar
  conclusão a partir de um hash — regra "nada inventado / não afirmar suposição como fato".
- => Rotação multi-quota REAL é comprovável. Com um 3º Codex (3º email real) = pool de 3
  contas reais. A oferta do operador de "3º agente Codex" era, de propósito, oferecer uma
  3ª CREDENCIAL real (3º email), não um seat de orquestração.

### Strings REAIS do /status do Codex (v0.142.5) — primárias, para o detector
- "5h limit: [███░] N% left (resets HH:MM)"        → janela 5h, reset por hora do dia.
- "Weekly limit: [███] N% left (resets HH:MM on D Mon)" → janela SEMANAL, reset com data.
- "Context window: N% left (X used / Y)"           → contexto (não é cota de plano).
- Conta/plano no header: "Account: <email> (Plus)".
- Detector deve parsear as DUAS janelas (5h + semanal) e o % de cada uma.

### COMPORTAMENTO CRÍTICO NOVO — "token reset" surpresa (mid-window)
- Alguns provedores concedem RESET DE TOKEN gratuito e ALEATÓRIO quando detectam que a
  conta está PERTO do limite — NÃO só no reset fixo de 5h. Ou seja, uma conta exausta pode
  voltar a ter cota DE FORMA IMPREVISÍVEL, no meio da janela.
- IMPACTO na arquitetura de rotação:
  * O modelo atual assume reset = window_start + 5h (ProactiveDetector). Isso é INCOMPLETO:
    a conta pode ficar available antes disso por um reset surpresa.
  * cooldown/return (H3) NÃO pode confiar só no relógio de 5h — precisa RE-SONDAR o /status
    da conta em cooldown para detectar reset antecipado e devolvê-la ao pool mais cedo.
  * Sob deploy grande, esperar 1-2+ esgotamentos por conta — cenário REAL de teste de
    rotação repetida (não sintético). Ótimo para validar rotação em ciclo.
- AÇÃO: alimentar isto em H3 (cooldown-return por re-sondagem, não só por relógio) e no
  detector (tratar reset surpresa). Registrar como requisito, não suposição.


## CORREÇÃO CODEX — é PROBE ATIVO do PAINEL DE USAGE (não só banner passivo)
Fonte: painel real do Codex CLI v0.142.5 (operador, 2026-07-02).
- O Codex CLI EXPÕE um PAINEL DE USAGE estruturado e parseável, sob demanda, contendo:
    5h limit:     [bar] N% left (resets HH:MM)
    Weekly limit: [bar] N% left (resets HH:MM on D Mon)
    Context window: N% left (X used / Y)
    Account: <email> (Plus)
  (URL de referência no header: chatgpt.com/codex/settings/usage.)
- CORREÇÃO de escopo: o mecanismo proativo CERTO para o Codex é o PROBE ATIVO deste
  painel (mesma família de /usage do Kiro e do Antigravity), NÃO esperar o banner passivo.
  O banner ("less than N% ... left" / "usage limit reached") vira FALLBACK, não o principal.
- O probe do Codex deve:
  * parsear AS DUAS janelas (5h E semanal) com % e reset de cada;
  * rotacionar quando QUALQUER janela cair abaixo do threshold (a mais apertada manda);
  * usar o mesmo painel para detectar RESET SURPRESA (conta em cooldown que recuperou cota
    aparece aqui) → devolve a conta ao pool (liga com H3 cooldown-return por re-sondagem).
- Implicação p/ usage.go / probe: o parser do Codex ganha padrões das DUAS linhas reais
  acima; o QuotaProbe do Codex passa a ter comando real (o painel de usage), saindo do
  slot "confirmar contra CLI real" para PROBE ATIVO confirmado.
- AÇÃO: (a) estender parser Codex p/ 5h+semanal; (b) definir o comando/al forma de obter o
  painel headless (confirmar CONTRA O BINÁRIO como se dispara o painel sem TUI — flag/subcmd);
  NÃO inventar a flag: verificar `codex --help`/docs antes de fixar.


## DESCOBERTA CRÍTICA — RESETS DE LIMITE CLAIMÁVEIS via /usage (Codex, 2026-07-02)
Fonte primária: tela real do Codex CLI v0.142.5 (operador).
Mensagem real observada no startup:
  "• You have 2 usage limit resets available. Run /usage to use one."

### O que isto revela (muda a lógica de rotação)
- O "token reset" NÃO é só automático/aleatório: o Codex CONCEDE CRÉDITOS DE RESET que o
  usuário CLAIMA ativamente rodando `/usage`. Uma conta exausta pode ter N resets em banco.
- `/usage` tem DUPLO papel:
  1) MOSTRA o painel de uso (5h + semanal + %/reset) → sinal do probe;
  2) CONSOME um crédito de reset → AÇÃO que restaura cota da MESMA conta.

### Impacto na arquitetura (requisito, não suposição)
- NOVA regra de decisão ANTES de rotacionar uma conta Codex exausta:
  1) Sondar `/usage`: a conta tem "N usage limit resets available"?
  2) SE sim (N>=1) → CLAIMAR um reset (rodar /usage p/ consumir) e MANTER a conta ativa
     (zero context switch, mesma sessão continua) — preferível à rotação.
  3) SE não (N==0) → ROTACIONAR para a próxima conta do pool (comportamento atual).
- Ou seja: rotação vira FALLBACK; "claim reset" é a 1ª opção quando há créditos. Isso
  maximiza uso de cada assinatura antes de trocar de conta.
- Detector/probe do Codex deve parsear também: "You have N usage limit resets available".
- cooldown-return (H3): uma conta em cooldown pode voltar via reset automático OU via
  claim — re-sondar /usage cobre ambos.

### Itens de backlog derivados
- [B-RESET-CLAIM] Lógica "claim-before-rotate" no serviço de rotação (nova política).
  Hotspot service.go/daemon.go → stream serial próprio. NÃO inventar: confirmar contra o
  binário COMO claimar headless (o painel é TUI; ver se há subcomando/flag não-interativo).
- [PROBE-CODEX] Parser do painel: 5h%+semanal%+resets_available+reset times.
- CONFIRMAR CONTRA BINÁRIO: forma headless de (a) ler o painel e (b) claimar um reset,
  já que `/usage` é slash-command de TUI (modo print é não-interativo). NÃO fixar sem verificar.


## RESOLVIDO — não foi bug (2026-07-02): logout foi MANUAL do operador
### O que aconteceu (esclarecido pelo operador)
- `~/.codex/auth.json` sumiu porque o OPERADOR fez um logout manual (ação humana), NÃO
  porque o `Logout()` da rotação apagou credencial real. Sem bug de blast-radius aqui.
- Opus havia levantado a hipótese "nosso código apagou" — hipótese INCORRETA. Registrar
  para honestidade: pular para a causa mais alarmante antes de confirmar foi precipitado;
  a causa real era a mais simples (logout do operador).
- Recuperação: operador re-loga (`codex login`) quando precisar da conta primária.

### Ainda assim — guarda defensiva continua VÁLIDA (best practice, não incidente)
- Princípio permanece correto e barato: rotação/logout deve operar SOMENTE em cópias
  isoladas por task, nunca no ~/.codex REAL. Não é urgente (não houve bug), mas vale como
  guard defensivo de baixo custo.
- [B-ISOLATION-GUARD] REBAIXADO de "bloqueador/incidente" para HARDENING defensivo (H-tier):
  guard opcional que recusa RemoveAll/restore fora do envRoot isolado + teste. Prioridade
  normal de hardening, não emergência.



## 3 CONTAS CODEX REAIS CONFIRMADAS (operador /status, 2026-07-02) — pool real
Três emails DISTINTOS (cada um Plus, quota própria) — provado por /status:
- acct1: mbenicios.filho82@gmail.com  (5h resets 12:51; weekly 16:32 on 8 Jul)
- acct2: beniciofilho82@gmail.com     (5h resets 13:10; weekly 08:10 on 9 Jul)
- acct3: mbeniciofilho82@gmail.com    (Plus; janelas próprias)
(+ ~/.codex-charlie existe mas é distinto/stale Jun-14 — não é nenhum dos 3 acima
 necessariamente; confirmar antes de usar.)

### Restrição operacional (governa o enrollment)
- `codex login` SOBRESCREVE ~/.codex/auth.json → só 1 conta em disco por vez.
- Para o pool de 3, CADA conta precisa ser CAPTURADA em diretório próprio ANTES do
  próximo login sobrescrever. Fluxo capture-then-enroll:
    (por conta) operador faz `codex login` → script snapshota ~/.codex/auth.json para
    ~/.codex-acctN/auth.json (chmod 600, sem imprimir conteúdo) → UPSERT no pool (prioridade N).
- Resultado: 3 dirs isolados coexistindo + 3 contas enroladas = pool real de rotação
  multi-quota (quotas independentes, provado pelos resets distintos).


## VERIFICADO — contas Codex COEXISTEM via CODEX_HOME (arquitetura do pool validada)
Teste real (Opus, 2026-07-02) contra o binário codex 0.142.5:
- `codex login status` (default ~/.codex) → "Logged in using ChatGPT" (conta ativa).
- `CODEX_HOME=~/.codex-charlie codex login status` → "Logged in using ChatGPT" (OUTRA conta).
- AMBAS autenticam SIMULTÂNEA e independentemente. Logar numa NÃO invalida a outra.
=> CONFIRMA a premissa central: pool multi-conta funciona. CODEX_HOME por conta = isolamento
   real; contas capturadas em dirs próprios coexistem. `codex login status` serve como
   liveness check por conta (wire no cooldown-return / health do pool).
=> Enrollment capture-then-enroll é válido: cada conta em ~/.codex-acctN + CODEX_HOME próprio.


## REQUISITO DE DEPLOY (O1) — credenciais em FS POSIX real, nunca drvfs/9p
Descoberta empírica do GLM-5.2 (B4, verificada pelo Opus 2026-07-02):
- `/mnt/c` (WSL 9p DrvFs SEM opção metadata) NÃO preserva permissões POSIX: `chmod 600`
  aparece como 777 em `ls -l`. Guardar credencial ali = permissão silenciosamente frouxa.
- Solução aplicada em staging: credenciais ficam FISICAMENTE em ext4
  (/home/dataops-lab/multica-auth-creds/<alias>) e são expostas via symlink onde o
  caminho lógico exige; o modo real 0600 é honrado no ext4. Credencial é CÓPIA (não
  symlink) por conta, conforme o modelo de isolamento do daemon (codex_home.go).
- REGRA PROD (runbook O1): o diretório de credenciais do pool DEVE residir em filesystem
  POSIX real (ext4/xfs/...), NUNCA em drvfs/9p/CIFS. Validar no deploy que `stat -c '%a'`
  do arquivo de credencial retorna 600 de fato. Caso contrário, abortar o deploy.


## FIX DE PROCESSO — board ÚNICO (split-brain corrigido, 2026-07-02)
- Detectado: existiam DOIS `.deploy-control/` — o raiz (correto) e um aninhado em
  multica-auth-work/.deploy-control/ (agentes com cwd em multica-auth-work resolviam o
  path relativo errado). Risco: um agente não ver o lock de outro → colisão.
- Corrigido: preservado o único stray (STG-ROTATE) no board raiz; board aninhado removido.
  Agora há UM board só: /mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/.
- REGRA (todos os prompts daqui pra frente): o check-in usa o CAMINHO ABSOLUTO do board:
  `/mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/<AGENT>__<STREAM>__<START_UTC>.md`
  NUNCA um `.deploy-control/` relativo ao cwd (que pode cair dentro de multica-auth-work).
