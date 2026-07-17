package agent

import (
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestBuildGeminiArgsBaseline(t *testing.T) {
	t.Parallel()

	args := buildGeminiArgs("write a haiku", ExecOptions{}, slog.Default())
	expected := []string{
		"-p", "write a haiku",
		"--yolo",
		"-o", "stream-json",
	}

	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, want := range expected {
		if args[i] != want {
			t.Fatalf("expected args[%d] = %q, got %q", i, want, args[i])
		}
	}
}

func TestBuildGeminiArgsWithModel(t *testing.T) {
	t.Parallel()

	args := buildGeminiArgs("hi", ExecOptions{Model: "gemini-2.5-pro"}, slog.Default())

	var foundModel bool
	for i, a := range args {
		if a == "-m" {
			if i+1 >= len(args) || args[i+1] != "gemini-2.5-pro" {
				t.Fatalf("expected -m followed by gemini-2.5-pro, got %v", args)
			}
			foundModel = true
			break
		}
	}
	if !foundModel {
		t.Fatalf("expected -m flag when Model is set, got args=%v", args)
	}
}

func TestBuildGeminiArgsWithResume(t *testing.T) {
	t.Parallel()

	args := buildGeminiArgs("hi", ExecOptions{ResumeSessionID: "3"}, slog.Default())

	var foundResume bool
	for i, a := range args {
		if a == "-r" {
			if i+1 >= len(args) || args[i+1] != "3" {
				t.Fatalf("expected -r followed by session id, got %v", args)
			}
			foundResume = true
			break
		}
	}
	if !foundResume {
		t.Fatalf("expected -r flag when ResumeSessionID is set, got args=%v", args)
	}
}

func TestBuildGeminiArgsOmitsModelWhenEmpty(t *testing.T) {
	t.Parallel()

	args := buildGeminiArgs("hi", ExecOptions{}, slog.Default())
	for _, a := range args {
		if a == "-m" {
			t.Fatalf("expected no -m flag when Model is empty, got args=%v", args)
		}
		if a == "-r" {
			t.Fatalf("expected no -r flag when ResumeSessionID is empty, got args=%v", args)
		}
	}
}

func TestBuildGeminiArgsPassesThroughCustomArgs(t *testing.T) {
	t.Parallel()

	args := buildGeminiArgs("hi", ExecOptions{
		CustomArgs: []string{"--sandbox"},
	}, slog.Default())

	if args[len(args)-1] != "--sandbox" {
		t.Fatalf("expected --sandbox at end of args, got %v", args)
	}
}

// envLookup returns the value of key in an env slice, or ("", false) if absent.
// When the key appears multiple times the last occurrence wins, mirroring how
// libc's getenv resolves duplicates on the daemon's supported platforms — the
// caller-supplied override therefore takes precedence over our default.
func envLookup(env []string, key string) (string, bool) {
	prefix := key + "="
	var value string
	var found bool
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			value = strings.TrimPrefix(entry, prefix)
			found = true
		}
	}
	return value, found
}

func TestBuildGeminiEnvSetsTrustWorkspaceDefault(t *testing.T) {
	t.Parallel()

	env := buildGeminiEnv(nil)
	got, ok := envLookup(env, "GEMINI_CLI_TRUST_WORKSPACE")
	if !ok {
		t.Fatalf("expected GEMINI_CLI_TRUST_WORKSPACE to be set, got env=%v", env)
	}
	if got != "true" {
		t.Fatalf("expected GEMINI_CLI_TRUST_WORKSPACE=true, got %q", got)
	}
}

func TestBuildGeminiEnvRespectsExplicitOverride(t *testing.T) {
	t.Parallel()

	// Users who deliberately set the value (e.g. to "false" to opt back into
	// gemini's folder-trust gate, or to a future-proofed value) must win over
	// our daemon default.
	env := buildGeminiEnv(map[string]string{"GEMINI_CLI_TRUST_WORKSPACE": "false"})
	got, ok := envLookup(env, "GEMINI_CLI_TRUST_WORKSPACE")
	if !ok {
		t.Fatalf("expected GEMINI_CLI_TRUST_WORKSPACE to be set, got env=%v", env)
	}
	if got != "false" {
		t.Fatalf("expected caller's GEMINI_CLI_TRUST_WORKSPACE=false to win, got %q", got)
	}
}

func TestBuildGeminiEnvPreservesOtherExtras(t *testing.T) {
	t.Parallel()

	env := buildGeminiEnv(map[string]string{"GEMINI_API_KEY": "secret"})
	if got, ok := envLookup(env, "GEMINI_API_KEY"); !ok || got != "secret" {
		t.Fatalf("expected GEMINI_API_KEY=secret to pass through, got %q (ok=%v)", got, ok)
	}
	if got, ok := envLookup(env, "GEMINI_CLI_TRUST_WORKSPACE"); !ok || got != "true" {
		t.Fatalf("expected default GEMINI_CLI_TRUST_WORKSPACE=true, got %q (ok=%v)", got, ok)
	}
}

func TestBuildGeminiArgsFiltersBlockedCustomArgs(t *testing.T) {
	t.Parallel()

	args := buildGeminiArgs("hi", ExecOptions{
		CustomArgs: []string{"-o", "text", "--sandbox"},
	}, slog.Default())

	// -o text should be filtered, --sandbox should pass through
	for i, a := range args {
		if a == "-o" && i+1 < len(args) && args[i+1] == "text" {
			t.Fatalf("blocked -o text should have been filtered: %v", args)
		}
	}
	if args[len(args)-1] != "--sandbox" {
		t.Fatalf("expected --sandbox to pass through, got %v", args)
	}
}

func TestWriteGeminiThinkingOverrideUsesModelScopedSystemSettings(t *testing.T) {
	t.Parallel()

	path, cleanup, err := writeGeminiThinkingOverride("gemini-3.5-flash", "high")
	if err != nil {
		t.Fatalf("writeGeminiThinkingOverride: %v", err)
	}
	defer cleanup()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated settings: %v", err)
	}
	var payload struct {
		ModelConfigs struct {
			CustomOverrides []struct {
				Match struct {
					Model string `json:"model"`
				} `json:"match"`
				ModelConfig struct {
					GenerateContentConfig struct {
						ThinkingConfig struct {
							ThinkingLevel string `json:"thinkingLevel"`
						} `json:"thinkingConfig"`
					} `json:"generateContentConfig"`
				} `json:"modelConfig"`
			} `json:"customOverrides"`
		} `json:"modelConfigs"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode generated settings: %v\n%s", err, raw)
	}
	if len(payload.ModelConfigs.CustomOverrides) != 1 {
		t.Fatalf("customOverrides length = %d, want 1", len(payload.ModelConfigs.CustomOverrides))
	}
	override := payload.ModelConfigs.CustomOverrides[0]
	if override.Match.Model != "gemini-3.5-flash" {
		t.Fatalf("override model = %q, want gemini-3.5-flash", override.Match.Model)
	}
	if got := override.ModelConfig.GenerateContentConfig.ThinkingConfig.ThinkingLevel; got != "HIGH" {
		t.Fatalf("thinkingLevel = %q, want HIGH", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat generated settings: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("generated settings mode = %o, want 600", info.Mode().Perm())
	}
}

func TestWriteGeminiThinkingOverrideRequiresExplicitModel(t *testing.T) {
	t.Parallel()
	if _, cleanup, err := writeGeminiThinkingOverride("", "high"); err == nil {
		cleanup()
		t.Fatal("expected empty model to be rejected")
	}
}
