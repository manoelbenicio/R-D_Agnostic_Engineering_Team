package brain

import (
	"fmt"
	"strings"
	"unicode"
)

// CLIKind identifies an executable frontend. It does not identify a provider,
// credential owner, account pool, or model vendor.
type CLIKind string

const (
	CLIClaudeCode       CLIKind = "claude-code"
	CLICodex            CLIKind = "codex"
	CLIKimi             CLIKind = "kimi"
	CLIOpenAICompatible CLIKind = "openai-compatible"
	CLIAntigravity      CLIKind = "antigravity"
	CLINIM              CLIKind = "nim"
)

var supportedCLIKinds = map[CLIKind]struct{}{
	CLIClaudeCode:       {},
	CLICodex:            {},
	CLIKimi:             {},
	CLIOpenAICompatible: {},
	CLIAntigravity:      {},
	CLINIM:              {},
}

func ParseCLIKind(raw string) (CLIKind, error) {
	kind := CLIKind(strings.ToLower(strings.TrimSpace(raw)))
	if _, ok := supportedCLIKinds[kind]; !ok {
		return "", fmt.Errorf("unsupported CLI kind %q", raw)
	}
	return kind, nil
}

// RouteModel is an exact model identifier understood by the gateway. It is
// independent from CLIKind and is never used to locate provider credentials.
type RouteModel string

func ParseRouteModel(raw string) (RouteModel, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("route model is required")
	}
	if len(value) > 256 {
		return "", fmt.Errorf("route model exceeds 256 bytes")
	}
	if strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || strings.Contains(value, "//") {
		return "", fmt.Errorf("route model has an invalid path form")
	}
	for _, r := range value {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return "", fmt.Errorf("route model contains whitespace or control characters")
		}
	}
	return RouteModel(value), nil
}

// RouterOwner identifies the one component allowed to make hot-path routing,
// account selection, retry, and fallback decisions for a request.
type RouterOwner string

const (
	RouterOwnerOmniRoute       RouterOwner = "omniroute"
	RouterOwnerLegacyRustL2    RouterOwner = "rust_l2"
	RouterOwnerLegacyGo        RouterOwner = "legacy_go"
	RouterOwnerLegacyNativeCLI RouterOwner = "native_cli"
)

func ParseRouterOwner(raw string) (RouterOwner, error) {
	owner := RouterOwner(strings.ToLower(strings.TrimSpace(raw)))
	switch owner {
	case RouterOwnerOmniRoute, RouterOwnerLegacyRustL2, RouterOwnerLegacyGo, RouterOwnerLegacyNativeCLI:
		return owner, nil
	default:
		return "", fmt.Errorf("unsupported router owner %q", raw)
	}
}

// ProtocolFamily is the gateway protocol spoken by a CLI adapter.
type ProtocolFamily string

const (
	ProtocolAnthropicMessages ProtocolFamily = "anthropic-messages"
	ProtocolOpenAIResponses   ProtocolFamily = "openai-responses"
	ProtocolOpenAIChat        ProtocolFamily = "openai-chat-completions"
	ProtocolAntigravity       ProtocolFamily = "antigravity"
)
