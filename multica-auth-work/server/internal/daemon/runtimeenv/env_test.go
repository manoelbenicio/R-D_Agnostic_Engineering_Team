package runtimeenv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

const syntheticSecret = "synthetic-omniroute-value"

func TestBuildMinimalInheritedRemovesCredentialAndRoutingSurface(t *testing.T) {
	inherited := []string{
		"PATH=/usr/bin", "LC_ALL=C", "TERM=xterm",
		"HOME=/untrusted", "ANTHROPIC_API_KEY=not-a-real-key",
		"OPENAI_BASE_URL=https://provider.invalid/v1", "KIMI_TOKEN=not-a-real-token",
		"NVIDIA_API_KEY=not-a-real-key", "AGENT_BRAIN_GATEWAY_BASE_URL=http://override.invalid",
		"SESSION_COOKIE=not-a-real-cookie", "HTTP_PROXY=http://proxy.invalid", "UNRELATED=value",
	}
	minimal, report, err := BuildMinimalInherited(inherited)
	if err != nil {
		t.Fatalf("BuildMinimalInherited returned error: %v", err)
	}
	want := []string{"LC_ALL", "PATH", "TERM"}
	if got := minimal.Keys(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("minimal inherited keys mismatch: got %v want %v", got, want)
	}
	if len(report.Removed) != len(inherited)-len(want) {
		t.Fatalf("removed count = %d, want %d", len(report.Removed), len(inherited)-len(want))
	}
	for _, removal := range report.Removed {
		if strings.Contains(removal.Key, "not-a-real") {
			t.Fatal("sanitization report exposed an environment value")
		}
	}
}

func TestValidateCustomEnvironmentRejectsNamesWithoutValues(t *testing.T) {
	value := "must-not-appear-in-error"
	err := ValidateCustomEnvironment(map[string]string{
		"SAFE_SETTING":         "enabled",
		"OPENAI_API_KEY":       value,
		"CUSTOM_REFRESH_TOKEN": value,
		"MODEL_BASE_URL":       value,
	})
	if err == nil {
		t.Fatal("expected custom environment rejection")
	}
	message := err.Error()
	if strings.Contains(message, value) {
		t.Fatal("custom environment error exposed a value")
	}
	for _, key := range []string{"OPENAI_API_KEY", "CUSTOM_REFRESH_TOKEN", "MODEL_BASE_URL"} {
		if !strings.Contains(message, key) {
			t.Fatalf("custom environment error omitted key %s", key)
		}
	}
}

func TestBuildGatewayEnvironmentClaudeAppliesTrustedValuesLast(t *testing.T) {
	_, taskHome, _ := controlledTestDirectories(t)
	secret, err := NewStableSecret(syntheticSecret)
	if err != nil {
		t.Fatalf("NewStableSecret returned error: %v", err)
	}
	environment, report, err := BuildGatewayEnvironment(ComposeOptions{
		Inherited: []string{
			"PATH=/usr/bin", "ANTHROPIC_BASE_URL=https://provider.invalid",
			"ANTHROPIC_AUTH_TOKEN=untrusted", "CLAUDECODE=1", "CLAUDE_CODE_SESSION_ID=parent",
		},
		Custom: map[string]string{"CLAUDE_CODE_MAX_OUTPUT_TOKENS": "4096", "SAFE_SETTING": "enabled"},
		Adapter: AdapterEnvironment{
			CLI: brain.CLIClaudeCode, GatewayRoot: "http://127.0.0.1:20128/",
			TaskHome: taskHome, StableSecret: secret,
		},
	})
	if err != nil {
		t.Fatalf("BuildGatewayEnvironment returned error: %v", err)
	}
	if len(report.Removed) != 4 {
		t.Fatalf("removed count = %d, want 4", len(report.Removed))
	}
	assertEnvValue(t, environment, "ANTHROPIC_BASE_URL", "http://127.0.0.1:20128")
	assertEnvValue(t, environment, "ANTHROPIC_AUTH_TOKEN", syntheticSecret)
	assertEnvValue(t, environment, "HOME", taskHome)
	assertEnvValue(t, environment, "CLAUDE_CODE_MAX_OUTPUT_TOKENS", "4096")
	for _, denied := range []string{"CLAUDECODE", "CLAUDE_CODE_SESSION_ID"} {
		if envHasKey(environment, denied) {
			t.Fatalf("internal Claude marker %s leaked", denied)
		}
	}
	if got := fmt.Sprintf("%v", environment); strings.Contains(got, syntheticSecret) {
		t.Fatal("formatted child environment exposed the stable secret")
	}
	if got := fmt.Sprintf("%+v", secret); strings.Contains(got, syntheticSecret) {
		t.Fatal("formatted stable secret exposed its value")
	}
}

func TestBuildGatewayEnvironmentCodexUsesDedicatedKeyName(t *testing.T) {
	_, taskHome, codexHome := controlledTestDirectories(t)
	secret, err := NewStableSecret(syntheticSecret)
	if err != nil {
		t.Fatalf("NewStableSecret returned error: %v", err)
	}
	environment, _, err := BuildGatewayEnvironment(ComposeOptions{
		Inherited: []string{"PATH=/usr/bin", "OPENAI_API_KEY=untrusted", "CODEX_ACCESS_TOKEN=untrusted"},
		Adapter: AdapterEnvironment{
			CLI: brain.CLICodex, GatewayRoot: "http://127.0.0.1:20128",
			TaskHome: taskHome, CodexHome: codexHome,
			StableSecret: secret,
		},
	})
	if err != nil {
		t.Fatalf("BuildGatewayEnvironment returned error: %v", err)
	}
	if CodexOmniRouteAPIKeyEnv != "AGENT_BRAIN_OMNIROUTE_API_KEY" {
		t.Fatalf("unexpected dedicated Codex key name %s", CodexOmniRouteAPIKeyEnv)
	}
	assertEnvValue(t, environment, CodexOmniRouteAPIKeyEnv, syntheticSecret)
	if envHasKey(environment, "OPENAI_API_KEY") || envHasKey(environment, "CODEX_ACCESS_TOKEN") {
		t.Fatal("provider-native Codex/OpenAI credential leaked")
	}
	assertEnvValue(t, environment, "HOME", taskHome)
	assertEnvValue(t, environment, "CODEX_HOME", codexHome)
	if environment.taskHome != taskHome || environment.codexHome != codexHome {
		t.Fatal("trusted canonical homes were not preserved exactly")
	}
}

func TestBuildGatewayEnvironmentRejectsNoncanonicalTrustedHomes(t *testing.T) {
	root, taskHome, codexHome := controlledTestDirectories(t)
	alternate := filepath.Join(root, "alternate")
	if err := os.MkdirAll(alternate, 0o700); err != nil {
		t.Fatalf("create alternate synthetic directory: %v", err)
	}
	noncanonical := taskHome + string(filepath.Separator) + ".." + string(filepath.Separator) + "alternate"
	secret, _ := NewStableSecret(syntheticSecret)
	tests := []struct {
		name      string
		cli       brain.CLIKind
		taskHome  string
		codexHome string
	}{
		{name: "Claude task home traversal", cli: brain.CLIClaudeCode, taskHome: noncanonical},
		{name: "Codex task home traversal", cli: brain.CLICodex, taskHome: noncanonical, codexHome: codexHome},
		{name: "Codex home traversal", cli: brain.CLICodex, taskHome: taskHome, codexHome: noncanonical},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := BuildGatewayEnvironment(ComposeOptions{Adapter: AdapterEnvironment{
				CLI: test.cli, GatewayRoot: "http://127.0.0.1:20128",
				TaskHome: test.taskHome, CodexHome: test.codexHome, StableSecret: secret,
			}})
			if err == nil {
				t.Fatal("noncanonical trusted home was accepted")
			}
		})
	}
}

func TestControlledDirectoryValidationRejectsSymlinkComponents(t *testing.T) {
	root, taskHome, codexHome := controlledTestDirectories(t)
	secret, _ := NewStableSecret(syntheticSecret)
	outside := t.TempDir()

	t.Run("execution root symlink", func(t *testing.T) {
		link := filepath.Join(t.TempDir(), "root-link")
		createTestDirectorySymlink(t, root, link)
		if err := ValidateExecutionRoot(link); err == nil {
			t.Fatal("symlinked execution root was accepted")
		}
	})

	t.Run("HOME symlink", func(t *testing.T) {
		link := filepath.Join(root, "home-link")
		createTestDirectorySymlink(t, outside, link)
		if _, _, err := BuildGatewayEnvironment(ComposeOptions{Adapter: AdapterEnvironment{
			CLI: brain.CLIClaudeCode, GatewayRoot: "http://127.0.0.1:20128",
			TaskHome: link, StableSecret: secret,
		}}); err == nil {
			t.Fatal("symlinked HOME was accepted")
		}
	})

	t.Run("HOME redirected component", func(t *testing.T) {
		redirect := filepath.Join(root, "redirect")
		createTestDirectorySymlink(t, outside, redirect)
		redirectedHome := filepath.Join(redirect, "nested-home")
		if err := os.MkdirAll(filepath.Join(outside, "nested-home"), 0o700); err != nil {
			t.Fatalf("create redirected synthetic home: %v", err)
		}
		if _, _, err := BuildGatewayEnvironment(ComposeOptions{Adapter: AdapterEnvironment{
			CLI: brain.CLIClaudeCode, GatewayRoot: "http://127.0.0.1:20128",
			TaskHome: redirectedHome, StableSecret: secret,
		}}); err == nil {
			t.Fatal("HOME with a redirecting path component was accepted")
		}
	})

	t.Run("CODEX_HOME symlink", func(t *testing.T) {
		link := filepath.Join(root, "codex-link")
		createTestDirectorySymlink(t, outside, link)
		if _, _, err := BuildGatewayEnvironment(ComposeOptions{Adapter: AdapterEnvironment{
			CLI: brain.CLICodex, GatewayRoot: "http://127.0.0.1:20128",
			TaskHome: taskHome, CodexHome: link, StableSecret: secret,
		}}); err == nil {
			t.Fatal("symlinked CODEX_HOME was accepted")
		}
	})

	if err := ValidateExecutionRoot(root); err != nil {
		t.Fatalf("valid canonical execution root rejected: %v", err)
	}
	noncanonicalRoot := root + string(filepath.Separator) + "child" + string(filepath.Separator) + ".."
	if err := ValidateExecutionRoot(noncanonicalRoot); err == nil {
		t.Fatal("noncanonical execution root was accepted")
	}
	if _, _, err := BuildGatewayEnvironment(ComposeOptions{Adapter: AdapterEnvironment{
		CLI: brain.CLICodex, GatewayRoot: "http://127.0.0.1:20128",
		TaskHome: taskHome, CodexHome: codexHome, StableSecret: secret,
	}}); err != nil {
		t.Fatalf("valid canonical homes rejected: %v", err)
	}
}

func controlledTestDirectories(t *testing.T) (root, taskHome, codexHome string) {
	t.Helper()
	root = t.TempDir()
	taskHome = filepath.Join(root, "task-home")
	codexHome = filepath.Join(root, "codex-home")
	for _, directory := range []string{taskHome, codexHome} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatalf("create controlled synthetic directory: %v", err)
		}
	}
	return root, taskHome, codexHome
}

func createTestDirectorySymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("directory symlink unavailable on this platform: %v", err)
	}
}

func assertEnvValue(t *testing.T, environment ChildEnvironment, key, want string) {
	t.Helper()
	for _, item := range environment.Exec() {
		candidate, value, ok := strings.Cut(item, "=")
		if ok && strings.EqualFold(candidate, key) {
			if value != want {
				t.Fatalf("environment value for %s did not match", key)
			}
			return
		}
	}
	t.Fatalf("environment key %s missing", key)
}

func envHasKey(environment ChildEnvironment, key string) bool {
	for _, candidate := range environment.Keys() {
		if strings.EqualFold(candidate, key) {
			return true
		}
	}
	return false
}

func TestBuildMinimalInheritedSkipsMalformedEntries(t *testing.T) {
	inherited := []string{
		"PATH=/usr/bin",
		"LANG=en_US.UTF-8",
		"BASH_FUNC_which%%=() { builtin which; }", // invalid key (contains %)
		"no-equals-separator",                     // no '=' separator
		"SHELL=/bin/bash",
		"ANTHROPIC_API_KEY=secret123", // denied credential (must still be removed)
	}
	env, report, err := BuildMinimalInherited(inherited)
	if err != nil {
		t.Fatalf("expected skip of malformed, got hard error: %v", err)
	}

	// Valid safe entries survive
	keys := env.Keys()
	keySet := map[string]bool{}
	for _, k := range keys {
		keySet[k] = true
	}
	if !keySet["PATH"] || !keySet["LANG"] || !keySet["SHELL"] {
		t.Fatalf("valid entries missing: %v", keys)
	}

	// Malformed entries use synthetic _malformed_<index> keys (no raw values/paths leaked)
	malformedCount := 0
	for _, r := range report.Removed {
		if strings.HasPrefix(r.Key, "_malformed_") {
			malformedCount++
			if strings.Contains(r.Key, "BASH") || strings.Contains(r.Key, "which") || strings.Contains(r.Key, "equals") {
				t.Fatalf("removal key leaks original content: %s", r.Key)
			}
		}
	}
	if malformedCount != 2 {
		t.Fatalf("expected 2 malformed removals (invalid key + no '='), got %d", malformedCount)
	}

	// Denied credential entries still removed via existing path
	credRemoved := false
	for _, r := range report.Removed {
		if r.Key == "ANTHROPIC_API_KEY" {
			credRemoved = true
		}
	}
	if !credRemoved {
		t.Fatal("denied credential ANTHROPIC_API_KEY should be in removal report")
	}

	// No raw values or paths in any removal entry
	for _, r := range report.Removed {
		if strings.Contains(r.Key, "secret") || strings.Contains(r.Key, "builtin") {
			t.Fatalf("removal leaks value content: %s", r.Key)
		}
	}
}

// TestBuildMinimalInheritedSkipsMalformedEntryAtIndex63WithoutLeak reproduces the
// live terminal failure ("inherited environment entry 63 malformed"): a malformed
// exported-bash-function fragment sitting at inherited index 63. The sanitizer MUST
// skip it (not hard-fail the launch), record the removal by INDEX only, and never
// place the entry's value into the SanitizationReport.
func TestBuildMinimalInheritedSkipsMalformedEntryAtIndex63WithoutLeak(t *testing.T) {
	const secretish = "sk-should-never-appear-in-report-63"
	inherited := make([]string, 0, 65)
	inherited = append(inherited, "PATH=/usr/bin") // index 0: safe, must survive
	for i := len(inherited); i < 63; i++ {         // indices 1..62: padding
		inherited = append(inherited, fmt.Sprintf("PAD_%d=x", i))
	}
	// index 63: invalid key name (bash exported-function fragment) with a
	// secret-looking value that must NOT be echoed into any report field.
	inherited = append(inherited, "BASH_FUNC_x%%=() { "+secretish+"; }")
	if got := len(inherited) - 1; got != 63 {
		t.Fatalf("test setup error: malformed entry is at index %d, want 63", got)
	}

	env, report, err := BuildMinimalInherited(inherited)
	if err != nil {
		t.Fatalf("malformed entry at index 63 must be skipped, got hard error: %v", err)
	}

	// The safe PATH key must survive the sanitizer.
	if _, ok := env.entries["PATH"]; !ok {
		t.Fatalf("PATH dropped; safe key must survive: %v", env.Keys())
	}

	// Index 63 must be recorded by index only, and no report field may leak the value.
	foundIndex63 := false
	for _, r := range report.Removed {
		if r.Key == "_malformed_63" {
			foundIndex63 = true
		}
		if strings.Contains(r.Key, secretish) || strings.Contains(string(r.Reason), secretish) {
			t.Fatalf("SanitizationReport leaked the malformed entry value: %+v", r)
		}
	}
	if !foundIndex63 {
		t.Fatalf("malformed index 63 not recorded as _malformed_63: %v", report.Removed)
	}
}

// TestBuildGatewayEnvironmentClineUsesDedicatedKeyName proves the accepted
// OpenAI-compatible (Cline) route injects the controlled per-task Cline data
// dir and the stable OmniRoute secret under the dedicated CLINE_OMNIROUTE_API_KEY
// as trusted-last entries (W1 D2). Inherited/custom provider surface is dropped
// and cannot shadow the trusted values; the secret never appears in diagnostics.
func TestBuildGatewayEnvironmentClineUsesDedicatedKeyName(t *testing.T) {
	root, taskHome, _ := controlledTestDirectories(t)
	clineDataDir := filepath.Join(root, "cline-data")
	if err := os.MkdirAll(clineDataDir, 0o700); err != nil {
		t.Fatalf("create controlled cline data dir: %v", err)
	}
	secret, err := NewStableSecret(syntheticSecret)
	if err != nil {
		t.Fatalf("NewStableSecret returned error: %v", err)
	}
	environment, _, err := BuildGatewayEnvironment(ComposeOptions{
		Inherited: []string{"PATH=/usr/bin", "OPENAI_API_KEY=untrusted", "CLINE_OMNIROUTE_API_KEY=untrusted"},
		Adapter: AdapterEnvironment{
			CLI: brain.CLIOpenAICompatible, GatewayRoot: "http://127.0.0.1:20128",
			TaskHome: taskHome, ClineDataDir: clineDataDir, StableSecret: secret,
		},
	})
	if err != nil {
		t.Fatalf("BuildGatewayEnvironment returned error: %v", err)
	}
	if ClineOmniRouteAPIKeyEnv != "CLINE_OMNIROUTE_API_KEY" {
		t.Fatalf("unexpected dedicated Cline key name %s", ClineOmniRouteAPIKeyEnv)
	}
	// Trusted-last secret wins over the inherited untrusted CLINE_OMNIROUTE_API_KEY
	// (which the deny-list drops), and the controlled data dir + task home are set.
	assertEnvValue(t, environment, ClineOmniRouteAPIKeyEnv, syntheticSecret)
	assertEnvValue(t, environment, "CLINE_DATA_DIR", clineDataDir)
	assertEnvValue(t, environment, "HOME", taskHome)
	if envHasKey(environment, "OPENAI_API_KEY") {
		t.Fatal("provider-native OpenAI credential leaked into the Cline child environment")
	}
	if got := fmt.Sprintf("%v", environment); strings.Contains(got, syntheticSecret) {
		t.Fatal("formatted child environment exposed the stable secret")
	}
}

// TestClineTrustedKeysRejectedFromCustomEnvironment proves CLINE_DATA_DIR and
// CLINE_OMNIROUTE_API_KEY are legal ONLY as trusted-injected entries: supplied
// via custom/local they must be rejected by the deny-list, so the launch cannot
// be tricked into overriding the controlled Cline data dir or the stable secret.
func TestClineTrustedKeysRejectedFromCustomEnvironment(t *testing.T) {
	if err := ValidateCustomEnvironment(map[string]string{"CLINE_DATA_DIR": "/tmp/x"}); err == nil {
		t.Fatal("CLINE_DATA_DIR must be rejected from the custom environment")
	}
	if err := ValidateCustomEnvironment(map[string]string{"CLINE_OMNIROUTE_API_KEY": "x"}); err == nil {
		t.Fatal("CLINE_OMNIROUTE_API_KEY must be rejected from the custom environment")
	}
}

func TestTrustedTelemetryInjectionAndOverrideBlocking(t *testing.T) {
	_, taskHome, _ := controlledTestDirectories(t)
	secret, err := NewStableSecret(syntheticSecret)
	if err != nil {
		t.Fatalf("NewStableSecret: %v", err)
	}

	// 1. Verify trusted injection sets exact official OTEL env
	environment, _, err := BuildGatewayEnvironment(ComposeOptions{
		Adapter: AdapterEnvironment{
			CLI:                       brain.CLIClaudeCode,
			GatewayRoot:               "http://127.0.0.1:20128",
			TaskHome:                  taskHome,
			StableSecret:              secret,
			TelemetryOTLPLogsEndpoint: "http://127.0.0.1:54321/v1/logs",
			TelemetryTaskID:           "task-77",
			TelemetryRequestID:        "abreq-9",
		},
	})
	if err != nil {
		t.Fatalf("BuildGatewayEnvironment: %v", err)
	}

	assertEnvValue(t, environment, "CLAUDE_CODE_ENABLE_TELEMETRY", "1")
	assertEnvValue(t, environment, "OTEL_LOGS_EXPORTER", "otlp")
	assertEnvValue(t, environment, "OTEL_METRICS_EXPORTER", "none")
	assertEnvValue(t, environment, "OTEL_EXPORTER_OTLP_LOGS_PROTOCOL", "http/json")
	assertEnvValue(t, environment, "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", "http://127.0.0.1:54321/v1/logs")
	assertEnvValue(t, environment, "OTEL_RESOURCE_ATTRIBUTES", "agent_brain.task_id=task-77,agent_brain.request_id=abreq-9")
	assertEnvValue(t, environment, "OTEL_LOG_USER_PROMPTS", "0")
	assertEnvValue(t, environment, "OTEL_LOG_ASSISTANT_RESPONSES", "0")
	assertEnvValue(t, environment, "OTEL_LOG_TOOL_DETAILS", "0")
	assertEnvValue(t, environment, "OTEL_LOG_TOOL_CONTENT", "0")

	if envHasKey(environment, "OTEL_LOG_RAW_API_BODIES") {
		t.Fatal("OTEL_LOG_RAW_API_BODIES must be absent/disabled")
	}

	// 2. Verify user/local/custom overrides for telemetry are rejected fail-closed
	overrideCases := []string{
		"CLAUDE_CODE_ENABLE_TELEMETRY",
		"OTEL_LOGS_EXPORTER",
		"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
		"OTEL_RESOURCE_ATTRIBUTES",
		"OTEL_LOG_USER_PROMPTS",
		"OTEL_LOG_ASSISTANT_RESPONSES",
		"OTEL_LOG_TOOL_DETAILS",
		"OTEL_LOG_TOOL_CONTENT",
		"OTEL_LOG_RAW_API_BODIES",
	}
	for _, key := range overrideCases {
		if err := ValidateCustomEnvironment(map[string]string{key: "1"}); err == nil {
			t.Fatalf("custom override for %s must be rejected fail-closed", key)
		}
	}

	// 3. Verify inherited telemetry overrides are stripped
	inheritedEnv, _, err := BuildMinimalInherited([]string{
		"PATH=/usr/bin",
		"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=http://attacker.com",
		"OTEL_LOG_USER_PROMPTS=1",
	})
	if err != nil {
		t.Fatalf("BuildMinimalInherited: %v", err)
	}
	for _, key := range inheritedEnv.Keys() {
		if strings.HasPrefix(strings.ToUpper(key), "OTEL_") {
			t.Fatalf("inherited telemetry override %s must be stripped", key)
		}
	}
}
