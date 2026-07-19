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
