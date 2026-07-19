package runtimeenv

import (
	"fmt"
	"sort"
	"strings"
)

// DenyReason is a value-free policy classification safe to emit in redacted
// diagnostics. It never contains an environment value.
type DenyReason string

const (
	DenyInvalidName        DenyReason = "invalid-name"
	DenyProviderCredential DenyReason = "provider-credential"
	DenyProviderEndpoint   DenyReason = "provider-endpoint"
	DenyOAuthOrCookie      DenyReason = "oauth-or-cookie"
	DenyCredentialRoot     DenyReason = "credential-discovery-root"
	DenyGatewayOverride    DenyReason = "unsafe-gateway-override"
	DenyClaudeMarker       DenyReason = "internal-claude-marker"
	DenyNetworkOverride    DenyReason = "unsafe-network-override"
)

var deniedExactKeys = map[string]DenyReason{
	"ANTHROPIC_API_KEY":              DenyProviderCredential,
	"ANTHROPIC_AUTH_TOKEN":           DenyProviderCredential,
	"ANTHROPIC_BASE_URL":             DenyProviderEndpoint,
	"ANTHROPIC_CUSTOM_HEADERS":       DenyProviderCredential,
	"OPENAI_API_KEY":                 DenyProviderCredential,
	"OPENAI_API_KEYS":                DenyProviderCredential,
	"OPENAI_BASE_URL":                DenyProviderEndpoint,
	"CODEX_API_KEY":                  DenyProviderCredential,
	"CODEX_ACCESS_TOKEN":             DenyProviderCredential,
	"CODEX_HOME":                     DenyCredentialRoot,
	"NVIDIA_API_KEY":                 DenyProviderCredential,
	"NIM_BASE_URL":                   DenyProviderEndpoint,
	"GOOGLE_API_KEY":                 DenyProviderCredential,
	"GEMINI_API_KEY":                 DenyProviderCredential,
	"GOOGLE_APPLICATION_CREDENTIALS": DenyCredentialRoot,
	"KIRO_API_KEY":                   DenyProviderCredential,
	"HOME":                           DenyCredentialRoot,
	"USERPROFILE":                    DenyCredentialRoot,
	"XDG_DATA_HOME":                  DenyCredentialRoot,
	"XDG_CONFIG_HOME":                DenyCredentialRoot,
	"CLINE_DATA_DIR":                 DenyCredentialRoot,
	"CLINE_SANDBOX_DATA_DIR":         DenyCredentialRoot,
	"OPENCLAW_CONFIG_PATH":           DenyCredentialRoot,
	"OPENCLAW_STATE_DIR":             DenyCredentialRoot,
	"OPENCLAW_HOME":                  DenyCredentialRoot,
	"OPENCLAW_INCLUDE_ROOTS":         DenyCredentialRoot,
	"SSH_AUTH_SOCK":                  DenyProviderCredential,
	"HTTP_PROXY":                     DenyNetworkOverride,
	"HTTPS_PROXY":                    DenyNetworkOverride,
	"ALL_PROXY":                      DenyNetworkOverride,
	"NO_PROXY":                       DenyNetworkOverride,
	"SSL_CERT_FILE":                  DenyNetworkOverride,
	"SSL_CERT_DIR":                   DenyNetworkOverride,
	"NODE_EXTRA_CA_CERTS":            DenyNetworkOverride,
	"REQUESTS_CA_BUNDLE":             DenyNetworkOverride,
	"CURL_CA_BUNDLE":                 DenyNetworkOverride,
	"GIT_ASKPASS":                    DenyProviderCredential,
	"GIT_SSH_COMMAND":                DenyProviderCredential,
	"CLAUDECODE":                     DenyClaudeMarker,
	"CLAUDE_CODE_ENTRYPOINT":         DenyClaudeMarker,
	"CLAUDE_CODE_EXECPATH":           DenyClaudeMarker,
	"CLAUDE_CODE_SESSION_ID":         DenyClaudeMarker,
	"CLAUDE_CODE_SSE_PORT":           DenyClaudeMarker,
}

var providerPrefixes = []string{
	"ANTHROPIC_", "OPENAI_", "AWS_", "AZURE_", "GOOGLE_", "GCP_",
	"GEMINI_", "KIMI_", "NVIDIA_", "NIM_", "KIRO_", "CLINE_", "CODEX_",
	"OPENCLAW_",
}

var credentialFragments = []string{
	"API_KEY", "API_KEYS", "ACCESS_TOKEN", "REFRESH_TOKEN", "AUTH_TOKEN",
	"BEARER_TOKEN", "OAUTH", "COOKIE", "CREDENTIAL", "SECRET_KEY",
}

var endpointSuffixes = []string{
	"_BASE_URL", "_API_BASE", "_API_URL", "_ENDPOINT", "_ENDPOINT_URL",
}

// EnvironmentKeyClassification reports whether a key is forbidden in an
// inherited/custom gateway-required child environment.
type EnvironmentKeyClassification struct {
	Denied bool
	Reason DenyReason
}

func ClassifyEnvironmentKey(key string) EnvironmentKeyClassification {
	if !validEnvironmentKey(key) {
		return EnvironmentKeyClassification{Denied: true, Reason: DenyInvalidName}
	}
	upper := strings.ToUpper(key)
	if reason, ok := deniedExactKeys[upper]; ok {
		return EnvironmentKeyClassification{Denied: true, Reason: reason}
	}
	if strings.HasPrefix(upper, "CLAUDECODE_") || strings.HasPrefix(upper, "CLAUDE_CODE_USE_") {
		return EnvironmentKeyClassification{Denied: true, Reason: DenyClaudeMarker}
	}
	if strings.HasPrefix(upper, "AGENT_BRAIN_") || strings.HasPrefix(upper, "MULTICA_L2_") || strings.HasPrefix(upper, "MULTICA_PRODEX_") {
		return EnvironmentKeyClassification{Denied: true, Reason: DenyGatewayOverride}
	}
	for _, prefix := range providerPrefixes {
		if strings.HasPrefix(upper, prefix) {
			for _, suffix := range endpointSuffixes {
				if strings.HasSuffix(upper, suffix) {
					return EnvironmentKeyClassification{Denied: true, Reason: DenyProviderEndpoint}
				}
			}
			return EnvironmentKeyClassification{Denied: true, Reason: DenyProviderCredential}
		}
	}
	for _, suffix := range endpointSuffixes {
		if strings.HasSuffix(upper, suffix) {
			return EnvironmentKeyClassification{Denied: true, Reason: DenyProviderEndpoint}
		}
	}
	for _, fragment := range credentialFragments {
		if strings.Contains(upper, fragment) {
			reason := DenyProviderCredential
			if strings.Contains(fragment, "OAUTH") || strings.Contains(fragment, "COOKIE") {
				reason = DenyOAuthOrCookie
			}
			return EnvironmentKeyClassification{Denied: true, Reason: reason}
		}
	}
	return EnvironmentKeyClassification{}
}

func validEnvironmentKey(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		switch {
		case r == '_':
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

func isSafeInheritedKey(key string) bool {
	upper := strings.ToUpper(key)
	switch upper {
	case "PATH", "PATHEXT", "SYSTEMROOT", "WINDIR", "COMSPEC",
		"TMPDIR", "TMP", "TEMP", "TEMPDIR", "LANG", "LANGUAGE",
		"TERM", "COLORTERM", "NO_COLOR", "FORCE_COLOR", "USER",
		"LOGNAME", "SHELL":
		return true
	}
	return strings.HasPrefix(upper, "LC_")
}

// EnvironmentViolation contains names and classifications only.
type EnvironmentViolation struct {
	Key    string
	Reason DenyReason
}

// EnvironmentPolicyError is safe to log: environment values are never stored.
type EnvironmentPolicyError struct {
	Violations []EnvironmentViolation
}

func (e *EnvironmentPolicyError) Error() string {
	if e == nil || len(e.Violations) == 0 {
		return "gateway-required environment policy violation"
	}
	parts := make([]string, 0, len(e.Violations))
	for _, violation := range e.Violations {
		parts = append(parts, fmt.Sprintf("%s(%s)", violation.Key, violation.Reason))
	}
	sort.Strings(parts)
	return "gateway-required environment rejected keys: " + strings.Join(parts, ", ")
}

// ValidateCustomEnvironment rejects provider credentials, credential roots,
// routing/auth overrides, and generic secret-bearing names without retaining
// or reporting any value.
func ValidateCustomEnvironment(custom map[string]string) error {
	violations := make([]EnvironmentViolation, 0)
	seen := map[string]string{}
	for key := range custom {
		canonical := strings.ToUpper(key)
		if prior, ok := seen[canonical]; ok && prior != key {
			violations = append(violations, EnvironmentViolation{Key: key, Reason: DenyInvalidName})
			continue
		}
		seen[canonical] = key
		classification := ClassifyEnvironmentKey(key)
		if classification.Denied {
			violations = append(violations, EnvironmentViolation{Key: key, Reason: classification.Reason})
		}
	}
	if len(violations) > 0 {
		return &EnvironmentPolicyError{Violations: violations}
	}
	return nil
}
