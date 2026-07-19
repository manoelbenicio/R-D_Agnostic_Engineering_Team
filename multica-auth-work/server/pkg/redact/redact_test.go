package redact

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestRedactAWSAccessKey(t *testing.T) {
	t.Parallel()
	input := "Found key AKIAIOSFODNN7EXAMPLE in config"
	got := Text(input)
	if strings.Contains(got, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatalf("AWS key not redacted: %s", got)
	}
	if !strings.Contains(got, "[REDACTED AWS KEY]") {
		t.Fatalf("expected [REDACTED AWS KEY] placeholder, got: %s", got)
	}
}

func TestRedactAWSSecretKey(t *testing.T) {
	t.Parallel()
	input := "aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	got := Text(input)
	if strings.Contains(got, "wJalrXUtnFEMI") {
		t.Fatalf("AWS secret not redacted: %s", got)
	}
}

func TestRedactPrivateKey(t *testing.T) {
	t.Parallel()
	input := "Here is the key:\n-----BEGIN RSA PRIVATE KEY-----\nMIIEow...\n-----END RSA PRIVATE KEY-----\nDone."
	got := Text(input)
	if strings.Contains(got, "MIIEow") {
		t.Fatalf("private key content not redacted: %s", got)
	}
	if !strings.Contains(got, "[REDACTED PRIVATE KEY]") {
		t.Fatalf("expected [REDACTED PRIVATE KEY] placeholder, got: %s", got)
	}
}

func TestRedactGitHubToken(t *testing.T) {
	t.Parallel()
	input := "export GITHUB_TOKEN=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmn"
	got := Text(input)
	if strings.Contains(got, "ghp_") {
		t.Fatalf("GitHub token not redacted: %s", got)
	}
}

func TestRedactOpenAIKey(t *testing.T) {
	t.Parallel()
	input := "OPENAI_API_KEY=sk-proj-abc123def456ghi789jkl012mno345"
	got := Text(input)
	if strings.Contains(got, "sk-proj-abc123") {
		t.Fatalf("OpenAI key not redacted: %s", got)
	}
}

func TestRedactSlackToken(t *testing.T) {
	t.Parallel()
	input := "token: xoxb-123456789012-1234567890123-AbCdEfGhIjKl"
	got := Text(input)
	if strings.Contains(got, "xoxb-") {
		t.Fatalf("Slack token not redacted: %s", got)
	}
}

func TestRedactBearerToken(t *testing.T) {
	t.Parallel()
	input := "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abc123"
	got := Text(input)
	if strings.Contains(got, "eyJhbGci") {
		t.Fatalf("Bearer token not redacted: %s", got)
	}
}

func TestRedactGenericCredentials(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
	}{
		{"API_KEY", "API_KEY=mysupersecretkey123"},
		{"DATABASE_URL", "DATABASE_URL=postgres://user:pass@host/db"},
		{"DB_PASSWORD", "DB_PASSWORD: hunter2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Text(tc.input)
			if !strings.Contains(got, "[REDACTED CREDENTIAL]") {
				t.Fatalf("expected credential redaction for %s, got: %s", tc.name, got)
			}
		})
	}
}

func TestRedactHomeDirectory(t *testing.T) {
	t.Parallel()
	if homeDir == "" || username == "" {
		t.Skip("cannot determine home dir or username")
	}
	input := "Reading file at " + homeDir + "/Documents/secret.txt"
	got := Text(input)
	if strings.Contains(got, username) {
		t.Fatalf("home directory username not redacted: %s", got)
	}
	if !strings.Contains(got, "****") {
		t.Fatalf("expected **** in path, got: %s", got)
	}
}

func TestNoFalsePositivesOnNormalText(t *testing.T) {
	t.Parallel()
	inputs := []string{
		"This is a normal commit message about fixing a bug",
		"The function returns skip-navigation as the class name",
		"Created PR #42 for the authentication feature",
		"Running tests in /tmp/test-workspace/project",
		"The API endpoint /api/issues/123 was updated",
	}
	for _, input := range inputs {
		got := Text(input)
		if got != input {
			t.Fatalf("false positive redaction:\n  input:  %s\n  output: %s", input, got)
		}
	}
}

func TestRedactGitLabToken(t *testing.T) {
	t.Parallel()
	input := "GITLAB_TOKEN=glpat-AbCdEfGhIjKlMnOpQrStUvWx"
	got := Text(input)
	if strings.Contains(got, "glpat-") {
		t.Fatalf("GitLab token not redacted: %s", got)
	}
}

func TestRedactJWT(t *testing.T) {
	t.Parallel()
	input := "token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	got := Text(input)
	if strings.Contains(got, "eyJhbGci") {
		t.Fatalf("JWT not redacted: %s", got)
	}
}

func TestRedactConnectionString(t *testing.T) {
	t.Parallel()
	input := "connecting to postgres://admin:s3cret@db.example.com:5432/mydb"
	got := Text(input)
	if strings.Contains(got, "s3cret") {
		t.Fatalf("connection string password not redacted: %s", got)
	}
}

func TestRedactPasswordEnvVar(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
	}{
		{"PASSWORD", "PASSWORD=hunter2"},
		{"SECRET", "SECRET=mysecretvalue"},
		{"TOKEN", "TOKEN=abc123xyz"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Text(tc.input)
			if !strings.Contains(got, "[REDACTED CREDENTIAL]") {
				t.Fatalf("expected credential redaction for %s, got: %s", tc.name, got)
			}
		})
	}
}

func TestInputMap(t *testing.T) {
	t.Parallel()
	m := map[string]any{
		"command":   "echo sk-proj-abc123def456ghi789jkl012mno345",
		"file_path": "/tmp/test.txt",
		"count":     42,
	}
	got := InputMap(m)
	if s, ok := got["command"].(string); ok {
		if strings.Contains(s, "sk-proj") {
			t.Fatalf("API key in input map not redacted: %s", s)
		}
	}
	// Non-string values preserved
	if got["count"] != 42 {
		t.Fatalf("non-string value altered: %v", got["count"])
	}
	// Clean strings unchanged
	if got["file_path"] != "/tmp/test.txt" {
		t.Fatalf("clean string altered: %v", got["file_path"])
	}
}

func TestInputMapNil(t *testing.T) {
	t.Parallel()
	if got := InputMap(nil); got != nil {
		t.Fatalf("expected nil, got: %v", got)
	}
}

func TestRedactMultipleSecrets(t *testing.T) {
	t.Parallel()
	input := "Keys: AKIAIOSFODNN7EXAMPLE and ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmn"
	got := Text(input)
	if strings.Contains(got, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatal("AWS key not redacted in multi-secret text")
	}
	if strings.Contains(got, "ghp_") {
		t.Fatal("GitHub token not redacted in multi-secret text")
	}
}

func TestSanitizeForLog(t *testing.T) {
	t.Parallel()
	input := map[string]any{
		"headers": map[string][]string{
			"Authorization": {"Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abc123"},
			"X-Api-Key":     {"AKIAIOSFODNN7EXAMPLE"},
			"User-Agent":    {"Go-http-client/1.1"},
		},
		"query": map[string]string{
			"secret": "synthetic-query-sentinel",
			"page":   "2",
		},
		"body": []any{
			"Some text with OPENAI_API_KEY=sk-proj-abc123def456ghi789jkl012mno345 inside",
			map[string]any{
				"nested_token": "glpat-AbCdEfGhIjKlMnOpQrStUvWx",
				"count":        42,
			},
		},
		"err": redactedError{err: &mockError{"DB_PASSWORD: synthetic-error-sentinel"}},
		"arr": []string{"normal string", "PASSWORD=synthetic-array-sentinel"},
	}

	got := SanitizeForLog(input)

	// Check if everything is redacted
	gotMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("SanitizeForLog returned %T, expected map[string]any", got)
	}

	headers := gotMap["headers"].(map[string][]string)
	if strings.Contains(headers["Authorization"][0], "eyJhbGci") {
		t.Fatalf("Bearer token in header not redacted: %s", headers["Authorization"][0])
	}
	if strings.Contains(headers["X-Api-Key"][0], "AKIAIOSFODNN7EXAMPLE") {
		t.Fatalf("AWS key in header not redacted: %s", headers["X-Api-Key"][0])
	}
	if headers["User-Agent"][0] != "Go-http-client/1.1" {
		t.Fatalf("Non-secret header altered: %s", headers["User-Agent"][0])
	}

	query := gotMap["query"].(map[string]string)
	if strings.Contains(query["secret"], "synthetic-query-sentinel") {
		t.Fatalf("query secret not redacted: %s", query["secret"])
	}
	if query["page"] != "2" {
		t.Fatalf("Non-secret query altered: %s", query["page"])
	}

	body := gotMap["body"].([]any)
	if strings.Contains(body[0].(string), "sk-proj-abc123") {
		t.Fatalf("nested OpenAI key not redacted: %s", body[0])
	}
	nestedMap := body[1].(map[string]any)
	if strings.Contains(nestedMap["nested_token"].(string), "glpat-") {
		t.Fatalf("nested GitLab token not redacted: %s", nestedMap["nested_token"])
	}
	if nestedMap["count"] != 42 {
		t.Fatalf("nested non-string value altered: %v", nestedMap["count"])
	}

	errVal := gotMap["err"].(redactedError)
	if strings.Contains(errVal.Error(), "synthetic-error-sentinel") {
		t.Fatalf("error string not redacted: %s", errVal.Error())
	}

	arr := gotMap["arr"].([]string)
	if arr[0] != "normal string" {
		t.Fatalf("normal string in array altered: %s", arr[0])
	}
	if strings.Contains(arr[1], "synthetic-array-sentinel") {
		t.Fatalf("password in array not redacted: %s", arr[1])
	}
}

func TestSanitizeForLogIsBoundedAndCycleSafe(t *testing.T) {
	t.Parallel()
	cycle := map[string]any{"safe": "visible"}
	cycle["self"] = cycle

	got := SanitizeForLog(cycle).(map[string]any)
	if got["safe"] != "visible" {
		t.Fatalf("safe value altered: %v", got["safe"])
	}
	if got["self"] != logRedactionReplacement {
		t.Fatalf("cycle did not fail closed: %v", got["self"])
	}

	var deep any = "visible"
	for range maxLogSanitizeDepth + 1 {
		deep = []any{deep}
	}
	if got := SanitizeForLog(deep); got == nil {
		t.Fatal("depth-bounded sanitization returned nil")
	}
}

func TestSanitizeForLogTypedNilError(t *testing.T) {
	t.Parallel()
	var typedNil *mockError
	var err error = typedNil
	if got := SanitizeForLog(err); got != nil {
		t.Fatalf("typed nil error should remain nil, got %T(%v)", got, got)
	}
}

func TestSanitizeSlogAttrUsesKeyAndPreservesSafeKinds(t *testing.T) {
	t.Parallel()
	secret := SanitizeSlogAttr(nil, slog.String("nested_token", "synthetic-opaque-sentinel"))
	if secret.Value.String() != logRedactionReplacement {
		t.Fatalf("sensitive slog attribute was not redacted: %q", secret.Value.String())
	}

	safeInt := SanitizeSlogAttr(nil, slog.Int("token_count", 42))
	if safeInt.Value.Kind() != slog.KindInt64 || safeInt.Value.Int64() != 42 {
		t.Fatalf("safe integer kind/value changed: kind=%v value=%v", safeInt.Value.Kind(), safeInt.Value)
	}
	safeString := SanitizeSlogAttr(nil, slog.String("status", "ready"))
	if safeString.Value.Kind() != slog.KindString || safeString.Value.String() != "ready" {
		t.Fatalf("safe string changed: kind=%v value=%v", safeString.Value.Kind(), safeString.Value)
	}
}

func TestSanitizeSlogAttrThroughHandler(t *testing.T) {
	t.Parallel()
	const sentinel = "synthetic-opaque-sentinel"
	var output bytes.Buffer
	handler := slog.NewJSONHandler(&output, &slog.HandlerOptions{ReplaceAttr: SanitizeSlogAttr})
	logger := slog.New(handler)
	logger.LogAttrs(context.Background(), slog.LevelInfo, "safe message",
		slog.String("token", sentinel),
		slog.Group("request", slog.String("client_secret", sentinel), slog.String("status", "ready")),
		slog.Group("credentials", slog.String("value", sentinel)),
		slog.Any("payload", map[string]any{"password": sentinel, "count": 42}),
	)

	got := output.String()
	if strings.Contains(got, sentinel) {
		t.Fatalf("central slog sanitization leaked sentinel: %s", got)
	}
	for _, safe := range []string{"safe message", "ready", `"count":42`} {
		if !strings.Contains(got, safe) {
			t.Fatalf("central slog sanitization removed safe value %q: %s", safe, got)
		}
	}
}

func TestRedactCredentialFieldsInJSONBody(t *testing.T) {
	t.Parallel()
	const sentinel = "synthetic-json-sentinel"
	got := Text(`{"access_token":"` + sentinel + `","refresh_token":"` + sentinel + `","status":"denied"}`)
	if strings.Contains(got, sentinel) {
		t.Fatalf("JSON credential field leaked: %s", got)
	}
	if !strings.Contains(got, `"status":"denied"`) {
		t.Fatalf("safe JSON field altered: %s", got)
	}
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string { return e.msg }
