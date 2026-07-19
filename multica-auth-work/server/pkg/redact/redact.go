// Package redact provides functions for detecting and masking secrets
// in agent output before it reaches the database or WebSocket broadcast.
package redact

import (
	"log/slog"
	"os"
	"os/user"
	"reflect"
	"regexp"
	"strings"
)

// secretPattern pairs a compiled regex with its replacement text.
type secretPattern struct {
	re          *regexp.Regexp
	replacement string
}

// Patterns are checked in order; first match wins per position.
var patterns = []secretPattern{
	// AWS access key IDs (always start with AKIA)
	{regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`), "[REDACTED AWS KEY]"},

	// AWS secret access keys (40 char base64-ish, preceded by a common separator)
	{regexp.MustCompile(`(?i)(?:aws_secret_access_key|secret_?access_?key)\s*[=:]\s*[A-Za-z0-9/+=]{40}`), "[REDACTED AWS SECRET]"},

	// PEM private keys (multi-line)
	{regexp.MustCompile(`(?s)-----BEGIN[A-Z\s]*PRIVATE KEY-----.*?-----END[A-Z\s]*PRIVATE KEY-----`), "[REDACTED PRIVATE KEY]"},

	// GitHub tokens (classic PAT, fine-grained, OAuth, etc.)
	{regexp.MustCompile(`\b(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{36,255}\b`), "[REDACTED GITHUB TOKEN]"},

	// OpenAI / Anthropic API keys
	{regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{20,}\b`), "[REDACTED API KEY]"},

	// Slack tokens
	{regexp.MustCompile(`\bxox[bporas]-[A-Za-z0-9\-]{10,}\b`), "[REDACTED SLACK TOKEN]"},

	// GitLab personal access tokens
	{regexp.MustCompile(`\bglpat-[A-Za-z0-9_-]{20,}\b`), "[REDACTED GITLAB TOKEN]"},

	// JWT tokens (three base64url segments)
	{regexp.MustCompile(`\bey[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`), "[REDACTED JWT]"},

	// Generic "Bearer <token>" in output
	{regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9\-._~+/]+=*\b`), "Bearer [REDACTED]"},

	// Connection strings with embedded passwords
	{regexp.MustCompile(`(?i)(?:postgres|mysql|mongodb|redis|amqp)(?:ql)?://[^:\s]+:[^@\s]+@`), "[REDACTED CONNECTION STRING]@"},

	// Credential-bearing JSON fields, including provider error response bodies.
	{regexp.MustCompile(`(?i)("(?:api_key|api_secret|secret_key|secret|access_token|refresh_token|id_token|auth_token|private_key|database_url|db_password|db_url|redis_url|password|token)"\s*:\s*)"(?:\\.|[^"\\])*"`), `${1}"[REDACTED]"`},

	// Generic key=value patterns for common secret env var names
	{regexp.MustCompile(`(?i)(?:API_KEY|API_SECRET|SECRET_KEY|SECRET|ACCESS_TOKEN|AUTH_TOKEN|PRIVATE_KEY|DATABASE_URL|DB_PASSWORD|DB_URL|REDIS_URL|PASSWORD|TOKEN)\s*[=:]\s*\S+`), "[REDACTED CREDENTIAL]"},
}

// InputMap returns a copy of m with all string values passed through Text.
// Non-string values are preserved as-is.
func InputMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			out[k] = Text(s)
		} else {
			out[k] = v
		}
	}
	return out
}

// homeDir is resolved once at init for path redaction.
var homeDir string
var username string

func init() {
	homeDir, _ = os.UserHomeDir()
	if u, err := user.Current(); err == nil {
		username = u.Username
	}
}

// Text scans the input string for known secret patterns and replaces
// matches with safe placeholders. It also masks the local user's home
// directory path to prevent leaking the username.
func Text(s string) string {
	for _, p := range patterns {
		s = p.re.ReplaceAllString(s, p.replacement)
	}

	// Redact home directory paths (e.g. /Users/john/ → /Users/****/).
	if homeDir != "" && username != "" {
		masked := strings.Replace(homeDir, username, "****", 1)
		s = strings.ReplaceAll(s, homeDir, masked)
	}

	return s
}

const (
	logRedactionReplacement = "[REDACTED]"
	maxLogSanitizeDepth     = 16
)

type logVisit struct {
	kind reflect.Kind
	ptr  uintptr
}

// SanitizeSlogAttr is the central slog ReplaceAttr hook. It uses the structured
// key before sanitizing the value, so an opaque sentinel under a credential key
// is still removed. Primitive non-secret values retain their original slog kind.
func SanitizeSlogAttr(groups []string, attr slog.Attr) slog.Attr {
	if IsSensitiveKey(attr.Key) || hasSensitiveGroup(groups) {
		attr.Value = slog.StringValue(logRedactionReplacement)
		return attr
	}

	switch attr.Value.Kind() {
	case slog.KindString:
		value := attr.Value.String()
		if sanitized := Text(value); sanitized != value {
			attr.Value = slog.StringValue(sanitized)
		}
	case slog.KindAny:
		attr.Value = slog.AnyValue(SanitizeForLog(attr.Value.Any()))
	}
	return attr
}

func hasSensitiveGroup(groups []string) bool {
	for _, group := range groups {
		if IsSensitiveKey(group) {
			return true
		}
	}
	return false
}

// IsSensitiveKey reports whether a structured field name denotes credential
// material. Suffix matching covers names such as nested_token and x_api_key,
// while deliberately preserving telemetry names such as token_count.
func IsSensitiveKey(key string) bool {
	normalized := strings.Trim(strings.ToLower(key), " _-")
	normalized = strings.NewReplacer("-", "_", ".", "_").Replace(normalized)
	switch normalized {
	case "authorization", "proxy_authorization", "cookie", "set_cookie",
		"auth", "credential", "credentials",
		"password", "passwd", "secret", "token", "api_key", "apikey",
		"api_secret", "access_token", "auth_token", "private_key",
		"database_url", "db_password", "db_url", "redis_url":
		return true
	}
	for _, suffix := range []string{"_password", "_passwd", "_secret", "_token", "_api_key", "_authorization"} {
		if strings.HasSuffix(normalized, suffix) {
			return true
		}
	}
	return false
}

// SanitizeForLog recursively traverses common structured logging containers
// and errors. Traversal is depth-bounded and cycle-aware, and never mutates the
// caller's maps or slices.
func SanitizeForLog(v any) any {
	return sanitizeForLog(v, "", 0, make(map[logVisit]struct{}))
}

func sanitizeForLog(v any, key string, depth int, path map[logVisit]struct{}) any {
	if v == nil {
		return nil
	}
	if IsSensitiveKey(key) {
		return logRedactionReplacement
	}
	if depth >= maxLogSanitizeDepth {
		return logRedactionReplacement
	}

	value := reflect.ValueOf(v)
	if isNilValue(value) {
		return nil
	}
	var visit logVisit
	if value.Kind() == reflect.Map || value.Kind() == reflect.Slice {
		visit = logVisit{kind: value.Kind(), ptr: value.Pointer()}
		if visit.ptr != 0 {
			if _, exists := path[visit]; exists {
				return logRedactionReplacement
			}
			path[visit] = struct{}{}
			defer delete(path, visit)
		}
	}

	switch val := v.(type) {
	case string:
		return Text(val)
	case []string:
		arr := make([]string, len(val))
		for i, s := range val {
			arr[i] = sanitizeForLog(s, "", depth+1, path).(string)
		}
		return arr
	case []any:
		arr := make([]any, len(val))
		for i, item := range val {
			arr[i] = sanitizeForLog(item, "", depth+1, path)
		}
		return arr
	case map[string]any:
		m := make(map[string]any, len(val))
		for k, item := range val {
			m[k] = sanitizeForLog(item, k, depth+1, path)
		}
		return m
	case map[string]string:
		m := make(map[string]string, len(val))
		for k, item := range val {
			m[k] = sanitizeForLog(item, k, depth+1, path).(string)
		}
		return m
	case map[string][]string:
		m := make(map[string][]string, len(val))
		for k, items := range val {
			if IsSensitiveKey(k) {
				m[k] = []string{logRedactionReplacement}
				continue
			}
			arr := make([]string, len(items))
			for i, item := range items {
				arr[i] = sanitizeForLog(item, "", depth+1, path).(string)
			}
			m[k] = arr
		}
		return m
	case error:
		if val == nil {
			return nil
		}
		return redactedError{err: val}
	default:
		return val
	}
}

func isNilValue(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

type redactedError struct {
	err error
}

func (e redactedError) Error() string {
	if e.err == nil {
		return ""
	}
	return Text(e.err.Error())
}
