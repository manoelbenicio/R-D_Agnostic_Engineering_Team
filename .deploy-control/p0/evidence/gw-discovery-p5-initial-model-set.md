# P5 — Patch Proposto: `InitialModelSet` em `release.go`
**Pacote:** p5-initial-model-set  
**Arquivo:** `internal/daemon/brain/release.go`  
**Revisor:** Antigravity (Opus48#C / w8:p1)  
**Data:** 2026-07-26T22:45Z  
**Status:** PROPOSTA — NÃO APLICADA. Requer assinatura Codex56-TL + decisão escrita do owner.

---

## Contexto dos tipos reais (identity.go)

- `CLIKind` conhecidos: `CLIClaudeCode="claude-code"`, `CLICodex="codex"`, `CLIAntigravity="antigravity"`, `CLIOpenAICompatible="openai-compatible"`, `CLIKimi="kimi"`, `CLINIM="nim"`
- `ProtocolFamily`: `ProtocolAnthropicMessages`, `ProtocolOpenAIResponses`, `ProtocolOpenAIChat`, `ProtocolAntigravity`
- `RouteModel`: texto livre validado (≤256 chars, sem whitespace, sem `//`)
- kiro, cline, agy, opencode **não têm CLIKind declarado** em identity.go (gap a resolver em outro pacote)

## IDs reais confirmados no gateway (curl 2026-07-26T22:45Z)

| ID no gateway | Conta | Tier |
|---|---|---|
| `agy/claude-opus-4-6-thinking` | agy | thinking |
| `agy/claude-sonnet-4-6` | agy | — |
| `agy/gemini-3.1-pro-high` | agy | high |
| `agy/gemini-3.1-pro-low` | agy | low |
| `agy/gemini-3.5-flash-high` | agy | high |
| `agy/gemini-3.5-flash-medium` | agy | medium |
| `agy/gemini-3.5-flash-low` | agy | low |
| `agy/gemini-2.5-pro` | agy | — |
| `agy/gemini-2.5-flash-thinking` | agy | thinking |
| `aug/claude-sonnet-4.6` | aug | — |
| `aug/claude-sonnet-4.6-thinking` | aug | thinking |
| `aug/claude-opus-4.6` | aug | — |
| `aug/gpt-5.5-high` | aug | high |
| `aug/gpt-5.5-medium` | aug | medium |
| `cp/cline-pass/kimi-k2.7-code` | clinepass | — |
| `claude_code_kimi_2.7_Code` | daemon default | — |

---

## ANTES (release.go L28-L42, atual)

```go
// L28-L42
func InitialModelSet() []InitialRoute {
	return []InitialRoute{
		{
			Model:    RouteModel("agy/claude-opus-4-6-thinking"),
			CLI:      CLIClaudeCode,
			Protocol: ProtocolAnthropicMessages,
			Approval: RouteApprovalEvidenceRequired,
			Fallback: FallbackPolicy{
				SameModelAccountFallback: true,
				CrossModelFallback:       nil,
				PreCommitOnly:            true,
			},
		},
	}
}
```

**Problemas:**
1. Registra apenas 1 rota (claude-opus-4-6-thinking) — sem tiers de reasoning, sem outros modelos.
2. `CLI: CLIClaudeCode` vincula o modelo a um CLI específico, o que viola o princípio gateway-first (o daemon não deveria precisar do CLI para descobrir modelos).
3. `Approval: RouteApprovalEvidenceRequired` bloqueia todas as rotas em canary — nenhuma fica disponível para o owner criar agentes.

---

## DEPOIS (proposta)

> **Nota de design:** `InitialRoute.CLI` e `InitialRoute.Protocol` existem para que o daemon saiba *qual executor* iniciar ao receber uma task com aquele modelo. No modelo gateway-first, o dropdown da UI virá do `/v1/models` do OmniRoute — mas o daemon ainda precisa do mapa CLI↔modelo para despachar. Por isso, mantemos `InitialRoute` mas expandimos o conjunto e adicionamos o campo `ThinkingLevel` (ver abaixo). Mudança de `Approval` para `RouteApprovalCanaryOnly` remove o bloqueio de evidência e deixa as rotas acessíveis para uso imediato.

### Novo campo `ThinkingLevel` em `InitialRoute` (release.go L10-L16)

```go
// ANTES (L10-L16):
type InitialRoute struct {
	Model    RouteModel
	CLI      CLIKind
	Protocol ProtocolFamily
	Approval RouteApproval
	Fallback FallbackPolicy
}

// DEPOIS — adicionar ThinkingLevel:
type InitialRoute struct {
	Model         RouteModel
	CLI           CLIKind
	Protocol      ProtocolFamily
	Approval      RouteApproval
	Fallback      FallbackPolicy
	ThinkingLevel string // "thinking" | "high" | "medium" | "low" | "" (sem tier explícito)
}
```

### Função `InitialModelSet` expandida (L28-L42 → L28-L120 estimado)

```go
// DEPOIS:
func InitialModelSet() []InitialRoute {
	return []InitialRoute{

		// ── CLAUDE (Anthropic-Messages, conta agy) ──────────────────────────
		{
			Model:         RouteModel("agy/claude-opus-4-6-thinking"),
			CLI:           CLIClaudeCode,
			Protocol:      ProtocolAnthropicMessages,
			Approval:      RouteApprovalCanaryOnly,
			ThinkingLevel: "thinking",
			Fallback:      FallbackPolicy{SameModelAccountFallback: true},
		},
		{
			Model:         RouteModel("agy/claude-sonnet-4-6"),
			CLI:           CLIClaudeCode,
			Protocol:      ProtocolAnthropicMessages,
			Approval:      RouteApprovalCanaryOnly,
			ThinkingLevel: "",
			Fallback:      FallbackPolicy{SameModelAccountFallback: true},
		},
		// conta aug — segundo pool Anthropic
		{
			Model:         RouteModel("aug/claude-sonnet-4.6-thinking"),
			CLI:           CLIClaudeCode,
			Protocol:      ProtocolAnthropicMessages,
			Approval:      RouteApprovalCanaryOnly,
			ThinkingLevel: "thinking",
			Fallback:      FallbackPolicy{SameModelAccountFallback: true},
		},
		{
			Model:         RouteModel("aug/claude-sonnet-4.6"),
			CLI:           CLIClaudeCode,
			Protocol:      ProtocolAnthropicMessages,
			Approval:      RouteApprovalCanaryOnly,
			ThinkingLevel: "",
			Fallback:      FallbackPolicy{SameModelAccountFallback: true},
		},
		{
			Model:         RouteModel("aug/claude-opus-4.6"),
			CLI:           CLIClaudeCode,
			Protocol:      ProtocolAnthropicMessages,
			Approval:      RouteApprovalCanaryOnly,
			ThinkingLevel: "",
			Fallback:      FallbackPolicy{SameModelAccountFallback: true},
		},

		// ── GEMINI (OpenAI-Chat via agy, tiers de reasoning explícitos) ────
		{
			Model:         RouteModel("agy/gemini-3.1-pro-high"),
			CLI:           CLIAntigravity,
			Protocol:      ProtocolOpenAIChat,
			Approval:      RouteApprovalCanaryOnly,
			ThinkingLevel: "high",
			Fallback:      FallbackPolicy{SameModelAccountFallback: true, CrossModelFallback: []RouteModel{"agy/gemini-3.1-pro-low"}},
		},
		{
			Model:         RouteModel("agy/gemini-3.1-pro-low"),
			CLI:           CLIAntigravity,
			Protocol:      ProtocolOpenAIChat,
			Approval:      RouteApprovalCanaryOnly,
			ThinkingLevel: "low",
			Fallback:      FallbackPolicy{SameModelAccountFallback: true},
		},
		{
			Model:         RouteModel("agy/gemini-3.5-flash-high"),
			CLI:           CLIAntigravity,
			Protocol:      ProtocolOpenAIChat,
			Approval:      RouteApprovalCanaryOnly,
			ThinkingLevel: "high",
			Fallback:      FallbackPolicy{SameModelAccountFallback: true, CrossModelFallback: []RouteModel{"agy/gemini-3.5-flash-medium", "agy/gemini-3.5-flash-low"}},
		},
		{
			Model:         RouteModel("agy/gemini-3.5-flash-medium"),
			CLI:           CLIAntigravity,
			Protocol:      ProtocolOpenAIChat,
			Approval:      RouteApprovalCanaryOnly,
			ThinkingLevel: "medium",
			Fallback:      FallbackPolicy{SameModelAccountFallback: true, CrossModelFallback: []RouteModel{"agy/gemini-3.5-flash-low"}},
		},
		{
			Model:         RouteModel("agy/gemini-3.5-flash-low"),
			CLI:           CLIAntigravity,
			Protocol:      ProtocolOpenAIChat,
			Approval:      RouteApprovalCanaryOnly,
			ThinkingLevel: "low",
			Fallback:      FallbackPolicy{SameModelAccountFallback: true},
		},
		// gemini-3.6-flash — não confirmado no gateway hoje; OMITIDO até validação

		// ── CODEX / OpenAI (OpenAI-Responses, conta aug) ───────────────────
		{
			Model:         RouteModel("aug/gpt-5.5-high"),
			CLI:           CLICodex,
			Protocol:      ProtocolOpenAIResponses,
			Approval:      RouteApprovalCanaryOnly,
			ThinkingLevel: "high",
			Fallback:      FallbackPolicy{SameModelAccountFallback: true, CrossModelFallback: []RouteModel{"aug/gpt-5.5-medium"}},
		},
		{
			Model:         RouteModel("aug/gpt-5.5-medium"),
			CLI:           CLICodex,
			Protocol:      ProtocolOpenAIResponses,
			Approval:      RouteApprovalCanaryOnly,
			ThinkingLevel: "medium",
			Fallback:      FallbackPolicy{SameModelAccountFallback: true},
		},

		// ── KIMI (ClinePass, via clinepass pool) ────────────────────────────
		{
			Model:         RouteModel("cp/cline-pass/kimi-k2.7-code"),
			CLI:           CLIKimi,
			Protocol:      ProtocolOpenAIChat,
			Approval:      RouteApprovalCanaryOnly,
			ThinkingLevel: "",
			Fallback:      FallbackPolicy{SameModelAccountFallback: true},
		},
	}
}
```

---

## Justificativas linha a linha

| Decisão | Razão |
|---|---|
| `ThinkingLevel string` em `InitialRoute` | Owner quer ver tier de reasoning na UI; campo livre evita enum restritivo agora; pode ser tipado depois |
| `RouteApprovalCanaryOnly` em todas as novas rotas | Remove bloqueio `EvidenceRequired` que impedia qualquer uso; canary ainda valida antes de full rollout |
| `gemini-3.6-flash` omitido | ID não existe nos 327 modelos do gateway hoje (curl confirmado); adicionar ID falso causaria 404 em produção |
| `CLIAntigravity` para rotas Gemini | `agy/` é a conta Antigravity no gateway; o CLI antigravity faz OpenAI-chat compat; `ProtocolOpenAIChat` correto |
| `CLIKimi` para kimi-k2.7 | CLIKind já declarado em identity.go L16 |
| kiro, cline, opencode sem rotas próprias | Não têm `CLIKind` declarado em identity.go — precisam de entrada em outro pacote antes de aparecer aqui |
| `PreCommitOnly: true` removido das novas rotas | Era guardião para a rota single de claude-opus; com múltiplas rotas e `CanaryOnly`, o gate é o canary |
| Fallback cross-model só dentro do mesmo tier | Evitar custo surpresa: high não cai em xhigh automaticamente |

---

## Gaps que ficam fora deste patch (requerem outros pacotes)

1. **identity.go** — `kiro-cli`, `cline`, `agy`, `opencode` não têm `CLIKind` → impede rota própria
2. **config.go:145-154** — `agentBrainBuiltInCLIFor` não mapeia os novos kinds → rotas não chegam ao executor
3. **handler de modelos** — descoberta do `/v1/models` do gateway (pacote gateway-fetch, outro autor)
4. **frontend** — dropdown `thinking_level` (banco já tem coluna `agent.thinking_level`)

---

## Verificação de compilação (a executar APÓS autorização)

```bash
cd /home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server
/home/ec2-user/goroot/go/bin/go vet ./internal/daemon/brain/... 2>&1
```

