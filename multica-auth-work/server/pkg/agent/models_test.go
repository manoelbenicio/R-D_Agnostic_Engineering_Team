package agent

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestListModelsStaticProviders(t *testing.T) {
	ctx := context.Background()
	providers := []string{"claude", "codex", "gemini"}
	if runtime.GOOS != "windows" {
		providers = append(providers, "cursor")
	}
	for _, provider := range providers {
		got, err := ListModels(ctx, provider, "")
		if err != nil {
			t.Fatalf("ListModels(%q) error: %v", provider, err)
		}
		if len(got) == 0 {
			t.Errorf("ListModels(%q) returned no models", provider)
		}
		for i, m := range got {
			if m.ID == "" {
				t.Errorf("ListModels(%q)[%d] has empty ID", provider, i)
			}
			if m.Label == "" {
				t.Errorf("ListModels(%q)[%d] has empty Label", provider, i)
			}
		}
	}
}

func TestListModelsCopilotFallsBackToStatic(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows model discovery intentionally fails closed before process start")
	}
	// Copilot uses dynamic ACP discovery, but with no `copilot`
	// binary on PATH (the discovery LookPath fails) it must fall
	// back to copilotStaticModels() so the UI dropdown stays
	// populated. This is the "binary missing on the daemon host"
	// path we care about for self-hosted runtimes.
	ctx := context.Background()
	key := discoveryCacheKey("copilot", "/nonexistent/copilot-cli")
	modelCacheMu.Lock()
	delete(modelCache, key)
	modelCacheMu.Unlock()

	got, err := ListModels(ctx, "copilot", "/nonexistent/copilot-cli")
	if err != nil {
		t.Fatalf("ListModels(copilot) error: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected static fallback models, got empty list")
	}
	ids := map[string]bool{}
	for _, m := range got {
		ids[m.ID] = true
	}
	if !ids["gpt-5.4"] || !ids["claude-sonnet-4.6"] {
		t.Errorf("static fallback missing expected models: %+v", got)
	}
}

func TestClaudeStaticModelsExposesFable5(t *testing.T) {
	models := claudeStaticModels()
	ids := map[string]Model{}
	defaults := 0
	for _, m := range models {
		ids[m.ID] = m
		if m.Default {
			defaults++
		}
	}

	fable, ok := ids["claude-fable-5"]
	if !ok {
		t.Fatalf("missing Claude Fable 5 in: %+v", models)
	}
	if fable.Label != "Claude Fable 5" || fable.Provider != "anthropic" || fable.Default {
		t.Errorf("unexpected Fable entry: %+v", fable)
	}
	if defaults != 1 || !ids["claude-sonnet-4-6"].Default {
		t.Errorf("expected Sonnet 4.6 to remain the sole default, got defaults=%d models=%+v", defaults, models)
	}
}

func TestGeminiStaticModelsExposesAliasesAndGemini3(t *testing.T) {
	// Gemini CLI has no `models list` subcommand, so we expose the
	// CLI's own aliases (auto / pro / flash / flash-lite) plus
	// explicit version pins including Gemini 3. Regression guard
	// for multica-ai/multica#1503 — Gemini 3 must be selectable.
	models := geminiStaticModels()
	ids := map[string]Model{}
	for _, m := range models {
		ids[m.ID] = m
	}
	for _, want := range []string{
		"auto", "auto-gemini-2.5",
		"pro", "flash", "flash-lite",
		"gemini-3-pro-preview", "gemini-3-flash-preview",
		"gemini-3.1-pro-preview", "gemini-3.1-flash-lite", "gemini-3.5-flash",
		"gemini-2.5-pro", "gemini-2.5-flash", "gemini-2.5-flash-lite",
	} {
		if _, ok := ids[want]; !ok {
			t.Errorf("missing expected Gemini model %q in: %+v", want, models)
		}
	}
	auto, ok := ids["auto"]
	if !ok || !auto.Default {
		t.Errorf("expected `auto` to be the default Gemini entry, got %+v", auto)
	}
	for _, m := range models {
		if m.Provider != "google" {
			t.Errorf("all Gemini entries must carry Provider=google, got %+v", m)
		}
	}
}

func TestAnnotateGeminiThinkingUsesOfficialPerModelLevels(t *testing.T) {
	t.Parallel()
	models := []Model{
		{ID: "gemini-3.1-pro-preview"},
		{ID: "gemini-3.5-flash"},
		{ID: "gemini-2.5-pro"},
	}
	annotateGeminiThinking(models)

	if got := thinkingValues(models[0].Thinking); !reflect.DeepEqual(got, []string{"low", "medium", "high"}) {
		t.Fatalf("Gemini 3.1 Pro levels = %v, want low/medium/high", got)
	}
	if models[0].Thinking.DefaultLevel != "high" {
		t.Fatalf("Gemini 3.1 Pro default = %q, want high", models[0].Thinking.DefaultLevel)
	}
	if got := thinkingValues(models[1].Thinking); !reflect.DeepEqual(got, []string{"minimal", "low", "medium", "high"}) {
		t.Fatalf("Gemini 3.5 Flash levels = %v, want minimal/low/medium/high", got)
	}
	if models[1].Thinking.DefaultLevel != "medium" {
		t.Fatalf("Gemini 3.5 Flash default = %q, want medium", models[1].Thinking.DefaultLevel)
	}
	if models[2].Thinking != nil {
		t.Fatalf("Gemini 2.5 Pro must remain unannotated; it uses thinkingBudget, got %+v", models[2].Thinking)
	}
}

func TestAnnotateKimiThinkingExposesProcessEffortLevels(t *testing.T) {
	t.Parallel()
	models := []Model{{ID: "kimi-code/kimi-for-coding"}}
	annotateKimiThinking(models)
	want := []string{"low", "medium", "high", "xhigh", "max"}
	if got := thinkingValues(models[0].Thinking); !reflect.DeepEqual(got, want) {
		t.Fatalf("Kimi levels = %v, want %v", got, want)
	}
}

func TestClineStaticModelsExposeRequestedClinePassModels(t *testing.T) {
	t.Parallel()
	models := clineStaticModels()
	ids := map[string]Model{}
	for _, model := range models {
		ids[model.ID] = model
	}
	for _, want := range []string{"cline-pass/kimi-k2.7-code", "cline-pass/glm-5.2"} {
		if _, ok := ids[want]; !ok {
			t.Errorf("missing requested ClinePass model %q in %+v", want, models)
		}
	}
}

func TestListModelsClineFallsBackAndAnnotatesThinking(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows model discovery intentionally fails closed before process start")
	}
	ctx := context.Background()
	got, err := ListModels(ctx, "cline", "/nonexistent/cline-cli")
	if err != nil {
		t.Fatalf("ListModels(cline) error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Cline fallback models = %+v, want exactly two", got)
	}
	wantLevels := []string{"none", "low", "medium", "high", "xhigh"}
	for _, model := range got {
		if levels := thinkingValues(model.Thinking); !reflect.DeepEqual(levels, wantLevels) {
			t.Errorf("%s thinking levels = %v, want %v", model.ID, levels, wantLevels)
		}
	}
}

func TestNIMStaticModelsDefaultsToGLM52(t *testing.T) {
	t.Parallel()
	models := nimStaticModels()
	if len(models) < 1 || models[0].ID != "z-ai/glm-5.2" || !models[0].Default {
		t.Fatalf("NIM catalog must default to z-ai/glm-5.2, got %+v", models)
	}
	if models[0].Thinking != nil {
		t.Fatalf("NVIDIA GLM-5.2 endpoint does not publish a per-request effort parameter; got %+v", models[0].Thinking)
	}
}

func thinkingValues(thinking *ModelThinking) []string {
	if thinking == nil {
		return nil
	}
	values := make([]string, 0, len(thinking.SupportedLevels))
	for _, level := range thinking.SupportedLevels {
		values = append(values, level.Value)
	}
	return values
}

func TestCodexStaticModelsExposesGPT55(t *testing.T) {
	// Codex CLI has no `models list` subcommand so the catalog is
	// hand-maintained. Regression guard for multica-ai/multica#2009 —
	// GPT-5.5 must be selectable, and the badge default must point at
	// the latest release rather than lagging a version behind.
	models := codexStaticModels()
	ids := map[string]Model{}
	for _, m := range models {
		ids[m.ID] = m
	}
	for _, want := range []string{
		"gpt-5.5", "gpt-5.5-mini",
		"gpt-5.4", "gpt-5.4-mini",
		"gpt-5.3-codex", "gpt-5",
		"o3", "o3-mini",
	} {
		if _, ok := ids[want]; !ok {
			t.Errorf("missing expected Codex model %q in: %+v", want, models)
		}
	}
	latest, ok := ids["gpt-5.6-sol"]
	if !ok || !latest.Default {
		t.Errorf("expected `gpt-5.6-sol` to be the default Codex entry, got %+v", latest)
	}
	defaults := 0
	for _, m := range models {
		if m.Default {
			defaults++
		}
		if m.Provider != "openai" {
			t.Errorf("all Codex entries must carry Provider=openai, got %+v", m)
		}
	}
	if defaults != 1 {
		t.Errorf("expected exactly one default Codex entry, got %d", defaults)
	}
}

func TestModelKnownIncompatibleWithProvider(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		model    string
		want     bool
	}{
		{
			name:     "claude model is incompatible with codex",
			provider: "codex",
			model:    "claude-sonnet-4-6",
			want:     true,
		},
		{
			name:     "codex model is compatible with codex",
			provider: "codex",
			model:    "gpt-5.5",
			want:     false,
		},
		{
			name:     "codex model is incompatible with claude",
			provider: "claude",
			model:    "o3",
			want:     true,
		},
		{
			name:     "exact claude model is compatible with claude",
			provider: "claude",
			model:    "claude-opus-4-7",
			want:     false,
		},
		{
			name:     "provider-prefixed openai model is incompatible with codex",
			provider: "codex",
			model:    "openai/gpt-4o",
			want:     true,
		},
		{
			name:     "provider-prefixed anthropic model is incompatible with claude",
			provider: "claude",
			model:    "anthropic/claude-opus-4.7",
			want:     true,
		},
		{
			name:     "known openai-looking model outside codex catalog is incompatible",
			provider: "codex",
			model:    "gpt-99",
			want:     true,
		},
		{
			name:     "unknown custom model is not classified",
			provider: "codex",
			model:    "private-lab-model",
			want:     false,
		},
		{
			name:     "unknown target provider does not clear",
			provider: "opencode",
			model:    "claude-sonnet-4-6",
			want:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ModelKnownIncompatibleWithProvider(tc.provider, tc.model); got != tc.want {
				t.Fatalf("ModelKnownIncompatibleWithProvider(%q, %q) = %v, want %v", tc.provider, tc.model, got, tc.want)
			}
		})
	}
}

func TestInferCopilotProvider(t *testing.T) {
	cases := map[string]string{
		"gpt-5.5":           "openai",
		"gpt-5.4-mini":      "openai",
		"gpt-5.3-codex":     "openai",
		"gpt-4.1":           "openai",
		"o1":                "openai",
		"o3":                "openai",
		"o3-mini":           "openai",
		"o4-mini":           "openai",
		"o5":                "openai", // future-proof: any o<digit>+
		"o6-mini-high":      "openai",
		"claude-opus-4.7":   "anthropic",
		"claude-sonnet-4.6": "anthropic",
		"claude-haiku-4.5":  "anthropic",
		"gemini-3-pro":      "google",
		"grok-code-fast-1":  "xai",
		"auto":              "",
		"raptor-mini":       "",
		// negative cases: must not be misidentified as OpenAI
		// reasoning series even though they start with `o`.
		"opus-fake": "",
		"omni":      "",
		"o":         "",
	}
	for id, want := range cases {
		if got := inferCopilotProvider(id); got != want {
			t.Errorf("inferCopilotProvider(%q) = %q, want %q", id, got, want)
		}
	}
}

func TestCopilotStaticModelsExposesFullCatalog(t *testing.T) {
	// GitHub Copilot CLI has no `models list` subcommand, so the
	// catalog is hand-maintained from the official supported-models
	// docs. Regression guard for multica-ai/multica#1948 — the
	// dropdown previously shipped only 2 models and used dashed IDs
	// (`claude-sonnet-4-6`) which the CLI rejects. IDs must use the
	// dotted form (`claude-sonnet-4.6`) that `copilot --model <id>`
	// actually accepts, and cover both OpenAI and Anthropic families.
	models := copilotStaticModels()
	ids := map[string]Model{}
	for _, m := range models {
		ids[m.ID] = m
	}
	for _, want := range []string{
		"gpt-5.5", "gpt-5.4", "gpt-5.4-mini",
		"gpt-5.3-codex", "gpt-5.2-codex", "gpt-5.2",
		"gpt-5-mini", "gpt-4.1",
		"claude-opus-4.7", "claude-sonnet-4.6",
		"claude-sonnet-4.5", "claude-haiku-4.5",
	} {
		if _, ok := ids[want]; !ok {
			t.Errorf("missing expected Copilot model %q in: %+v", want, models)
		}
	}
	// Dashed legacy IDs must not reappear — `copilot --model
	// claude-sonnet-4-6` errors with "Model ... is not available".
	for _, banned := range []string{"claude-sonnet-4-6", "claude-sonnet-4-5"} {
		if _, ok := ids[banned]; ok {
			t.Errorf("Copilot catalog must not use dashed model id %q; use dotted form", banned)
		}
	}
	for _, m := range models {
		switch m.Provider {
		case "openai", "anthropic":
		default:
			t.Errorf("Copilot entry %q has unexpected Provider %q", m.ID, m.Provider)
		}
		if m.Default {
			t.Errorf("Copilot entries should not set Default; account routing decides. got %+v", m)
		}
	}
}

func TestListModelsHermesWithoutBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows model discovery intentionally fails closed before process start")
	}
	// With no `hermes` binary on PATH the discovery fast-paths to
	// an empty list (the UI then falls back to creatable manual
	// entry). This test only verifies the fast-path; an actual
	// ACP session is exercised in integration.
	ctx := context.Background()
	// Prime the cache miss so we hit the live discovery function.
	key := discoveryCacheKey("hermes", "/nonexistent/hermes")
	modelCacheMu.Lock()
	delete(modelCache, key)
	modelCacheMu.Unlock()

	got, err := ListModels(ctx, "hermes", "/nonexistent/hermes")
	if err != nil {
		t.Fatalf("ListModels(hermes) error: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil slice even when binary is missing")
	}
}

func TestListModelsKiroWithoutBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows model discovery intentionally fails closed before process start")
	}
	ctx := context.Background()
	key := discoveryCacheKey("kiro", "/nonexistent/kiro-cli")
	modelCacheMu.Lock()
	delete(modelCache, key)
	modelCacheMu.Unlock()

	got, err := ListModels(ctx, "kiro", "/nonexistent/kiro-cli")
	if err != nil {
		t.Fatalf("ListModels(kiro) error: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil slice even when binary is missing")
	}
}

func TestAnnotateKiroThinkingUsesDocumentedPerModelLevels(t *testing.T) {
	t.Parallel()

	models := []Model{
		{ID: "claude-opus-4.8", Label: "Claude Opus 4.8"},
		{ID: "claude-opus-4.6", Label: "Claude Opus 4.6"},
		{ID: "claude-sonnet-4.6", Label: "Claude Sonnet 4.6"},
		{ID: "gpt-5.6-sol", Label: "GPT 5.6 Sol"},
	}
	annotateKiroThinking(models)

	values := func(model Model) []string {
		if model.Thinking == nil {
			return nil
		}
		out := make([]string, 0, len(model.Thinking.SupportedLevels))
		for _, level := range model.Thinking.SupportedLevels {
			out = append(out, level.Value)
		}
		return out
	}

	if got, want := values(models[0]), []string{"low", "medium", "high", "xhigh", "max"}; !slices.Equal(got, want) {
		t.Errorf("Opus 4.8 levels = %v, want %v", got, want)
	}
	if got, want := values(models[1]), []string{"low", "medium", "high", "max"}; !slices.Equal(got, want) {
		t.Errorf("Opus 4.6 levels = %v, want %v", got, want)
	}
	if got, want := values(models[2]), []string{"low", "medium", "high", "max"}; !slices.Equal(got, want) {
		t.Errorf("Sonnet 4.6 levels = %v, want %v", got, want)
	}
	if models[3].Thinking != nil {
		t.Errorf("undocumented Kiro effort model should stay unannotated: %+v", models[3].Thinking)
	}
}

func TestListModelsQoderWithoutBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows model discovery intentionally fails closed before process start")
	}
	ctx := context.Background()
	key := discoveryCacheKey("qoder", "/nonexistent/qodercli")
	modelCacheMu.Lock()
	delete(modelCache, key)
	modelCacheMu.Unlock()

	got, err := ListModels(ctx, "qoder", "/nonexistent/qodercli")
	if err != nil {
		t.Fatalf("ListModels(qoder) error: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil slice even when binary is missing")
	}
}

func TestListModelsUnknownProvider(t *testing.T) {
	ctx := context.Background()
	_, err := ListModels(ctx, "nonexistent", "")
	if err == nil {
		t.Fatal("ListModels(unknown) expected error")
	}
}

func TestStaticCatalogsHaveAtMostOneDefault(t *testing.T) {
	// Each catalog should tag at most one entry as the display
	// default so the UI badge is unambiguous. More than one
	// usually means a copy/paste slip when adding new models.
	catalogs := map[string][]Model{
		"claude":  claudeStaticModels(),
		"codex":   codexStaticModels(),
		"gemini":  geminiStaticModels(),
		"cursor":  cursorStaticModels(),
		"copilot": copilotStaticModels(),
	}
	for provider, models := range catalogs {
		count := 0
		for _, m := range models {
			if m.Default {
				count++
			}
		}
		if count > 1 {
			t.Errorf("%s: %d models marked Default, want 0 or 1", provider, count)
		}
	}
}

func TestParseOpenCodeModels(t *testing.T) {
	input := `PROVIDER/MODEL                     CONTEXT  MAX_OUT
openai/gpt-4o                      128000   16384
anthropic/claude-sonnet-4-6        200000   8192
openai/gpt-4o                      128000   16384
nonprefixed-line
`
	models := parseOpenCodeModels(input)
	if len(models) != 2 {
		t.Fatalf("expected 2 models (header skipped, duplicate deduped, non-slash skipped), got %d: %+v", len(models), models)
	}
	if models[0].ID != "openai/gpt-4o" || models[0].Provider != "openai" {
		t.Errorf("unexpected first model: %+v", models[0])
	}
	if models[1].ID != "anthropic/claude-sonnet-4-6" || models[1].Provider != "anthropic" {
		t.Errorf("unexpected second model: %+v", models[1])
	}
}

func TestParseOpenCodeModelsVerboseVariants(t *testing.T) {
	input := `openai/gpt-5
{
  "id": "gpt-5",
  "name": "GPT-5",
  "reasoning": true,
  "variants": {
    "high": { "reasoningEffort": "high" },
    "low": { "reasoningEffort": "low" },
    "xhigh": { "reasoningEffort": "xhigh" },
    "fast-mode": { "reasoningEffort": "low" },
    "disabled": { "disabled": true }
  }
}
anthropic/claude-sonnet-4-6
{
  "id": "claude-sonnet-4-6",
  "reasoning": true,
  "variants": {
    "max": { "thinking": { "type": "enabled", "budgetTokens": 32000 } },
    "high": { "thinking": { "type": "enabled", "budgetTokens": 16000 } }
  }
}
`
	models := parseOpenCodeModels(input)
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d: %+v", len(models), models)
	}
	if models[0].Thinking == nil {
		t.Fatalf("expected first model to expose thinking variants")
	}
	got := make([]string, 0, len(models[0].Thinking.SupportedLevels))
	for _, lvl := range models[0].Thinking.SupportedLevels {
		got = append(got, lvl.Value)
		if lvl.Value == "xhigh" && lvl.Label != "Extra high" {
			t.Errorf("xhigh label: got %q, want Extra high", lvl.Label)
		}
		if lvl.Value == "fast-mode" && lvl.Label != "Fast Mode" {
			t.Errorf("custom variant label: got %q, want Fast Mode", lvl.Label)
		}
	}
	want := []string{"low", "high", "xhigh", "fast-mode"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("variant order/values: got %v, want %v", got, want)
	}
	if models[1].Thinking == nil || len(models[1].Thinking.SupportedLevels) != 2 {
		t.Fatalf("expected second model variants, got %+v", models[1].Thinking)
	}
}

func TestParseOpenCodeModelsMalformedVerboseBlockKeepsFollowingModels(t *testing.T) {
	input := `openai/gpt-5
{
  "id": "gpt-5",
  "reasoning": true,
  "variants": {
    "high": {}
  }
anthropic/claude-sonnet-4-6
{
  "id": "claude-sonnet-4-6",
  "reasoning": true,
  "variants": {
    "high": {},
    "max": {}
  }
}
`
	models := parseOpenCodeModels(input)
	if len(models) != 2 {
		t.Fatalf("expected both model rows to survive malformed JSON, got %d: %+v", len(models), models)
	}
	if models[0].ID != "openai/gpt-5" {
		t.Fatalf("unexpected first model: %+v", models[0])
	}
	if models[0].Thinking != nil {
		t.Fatalf("malformed first JSON block should not annotate thinking: %+v", models[0].Thinking)
	}
	if models[1].ID != "anthropic/claude-sonnet-4-6" {
		t.Fatalf("unexpected second model: %+v", models[1])
	}
	if models[1].Thinking == nil || len(models[1].Thinking.SupportedLevels) != 2 {
		t.Fatalf("valid following JSON block should still annotate thinking: %+v", models[1].Thinking)
	}
}

func TestDiscoverOpenCodeModelsFallsBackWhenVerboseFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script fake binary requires a POSIX shell")
	}

	dir := t.TempDir()
	fake := filepath.Join(dir, "opencode")
	script := `#!/bin/sh
if [ "$1" = "models" ] && [ "$2" = "--verbose" ]; then
  exit 2
fi
if [ "$1" = "models" ]; then
  cat <<'EOF'
PROVIDER/MODEL                     CONTEXT  MAX_OUT
openai/gpt-4o                      128000   16384
EOF
  exit 0
fi
exit 1
`
	writeTestExecutable(t, fake, []byte(script))

	models, err := discoverOpenCodeModels(context.Background(), fake)
	if err != nil {
		t.Fatalf("discoverOpenCodeModels: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected fallback non-verbose model, got %d: %+v", len(models), models)
	}
	if models[0].ID != "openai/gpt-4o" || models[0].Thinking != nil {
		t.Fatalf("unexpected fallback model: %+v", models[0])
	}
}

// TestCachedDiscoveryNegativeCache verifies that an empty discovery result is
// cached only for the short negative TTL. This prevents retry storms without
// holding a transient blank catalog for the normal positive TTL.
func TestCachedDiscoveryNegativeCache(t *testing.T) {
	const emptyKey, nonEmptyKey = "test-cache-empty", "test-cache-nonempty"
	// modelCache is a package-level global; clear our keys up front and on
	// cleanup so the test stays hermetic under `go test -count=N` (a leftover
	// non-empty entry from a prior run would otherwise skip the callback).
	resetCache := func() {
		modelCacheMu.Lock()
		delete(modelCache, emptyKey)
		delete(modelCache, nonEmptyKey)
		delete(modelCacheFlights, emptyKey)
		delete(modelCacheFlights, nonEmptyKey)
		modelCacheMu.Unlock()
	}
	resetCache()
	t.Cleanup(resetCache)

	emptyCalls := 0
	empty := func() ([]Model, error) {
		emptyCalls++
		return []Model{}, nil
	}
	for i := 0; i < 2; i++ {
		got, err := cachedDiscovery(context.Background(), emptyKey, empty)
		if err != nil {
			t.Fatalf("cachedDiscovery: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected empty result, got %+v", got)
		}
	}
	if emptyCalls != 1 {
		t.Fatalf("empty result must be negative-cached: expected fn called 1x, got %d", emptyCalls)
	}
	modelCacheMu.Lock()
	entry := modelCache[emptyKey]
	entry.refreshAfter = time.Now().Add(-time.Second)
	entry.hardExpiresAt = time.Now().Add(-time.Second)
	modelCache[emptyKey] = entry
	modelCacheMu.Unlock()
	if _, err := cachedDiscovery(context.Background(), emptyKey, empty); err != nil {
		t.Fatalf("cachedDiscovery after negative expiry: %v", err)
	}
	if emptyCalls != 2 {
		t.Fatalf("expired negative result must retry: expected fn called 2x, got %d", emptyCalls)
	}

	nonEmptyCalls := 0
	nonEmpty := func() ([]Model, error) {
		nonEmptyCalls++
		return []Model{{ID: "provider/model"}}, nil
	}
	for i := 0; i < 2; i++ {
		if _, err := cachedDiscovery(context.Background(), nonEmptyKey, nonEmpty); err != nil {
			t.Fatalf("cachedDiscovery: %v", err)
		}
	}
	if nonEmptyCalls != 1 {
		t.Fatalf("non-empty result must be cached: expected fn called 1x, got %d", nonEmptyCalls)
	}
}

func TestParsePiModels(t *testing.T) {
	input := `openai:gpt-4o
anthropic:claude-opus-4-7
openai:gpt-4o
bareword
`
	models := parsePiModels(input)
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d: %+v", len(models), models)
	}
	if models[0].ID != "openai/gpt-4o" {
		t.Errorf("expected colon normalized to slash: %+v", models[0])
	}
}

func TestParsePiModelsTableFormat(t *testing.T) {
	input := `provider             model                   context  max-out  thinking  images
bailian-coding-plan  glm-4.7                 202.8K   16.4K    no        no
bailian-coding-plan  qwen3.6-plus            1M       65.5K    no        yes
opencode             claude-sonnet-4-6       1M       64K      yes       yes
opencode             claude-sonnet-4-6:exp   1M       64K      yes       yes
opencode             claude-sonnet-4-6       1M       64K      yes       yes
bareword-only-line
`
	models := parsePiModels(input)
	if len(models) != 4 {
		t.Fatalf("expected 4 models (header skipped, duplicate deduped, bareword skipped), got %d: %+v", len(models), models)
	}
	if models[0].ID != "bailian-coding-plan/glm-4.7" || models[0].Provider != "bailian-coding-plan" {
		t.Errorf("unexpected first model: %+v", models[0])
	}
	if models[1].ID != "bailian-coding-plan/qwen3.6-plus" || models[1].Provider != "bailian-coding-plan" {
		t.Errorf("unexpected second model: %+v", models[1])
	}
	if models[2].ID != "opencode/claude-sonnet-4-6" || models[2].Provider != "opencode" {
		t.Errorf("unexpected third model: %+v", models[2])
	}
	// Colon inside a model name in column 1 must be preserved — only
	// the legacy `provider:model` form gets colon→slash normalization.
	if models[3].ID != "opencode/claude-sonnet-4-6:exp" || models[3].Provider != "opencode" {
		t.Errorf("expected ':' inside table-format model name to be preserved: %+v", models[3])
	}
}

// TestDiscoverPiModelsNonZeroExit verifies that discoverPiModels still returns
// the resolvable catalog when `pi --list-models` exits non-zero. Pi exits
// non-zero (and warns) when an agent config references stale provider/model
// patterns that no longer match the local catalog. Before the fix the daemon
// discarded the populated output on any non-zero exit and returned an empty
// list, so the UI model picker was blank even though the runtime was online and
// agents ran fine. See GitHub #3729.
func TestDiscoverPiModelsNonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake pi binary is a /bin/sh script")
	}

	const table = "provider         model        context  max-out  thinking  images\n" +
		"glm-coding-plan  glm-4.7      202.8K   16.4K    no        no"
	// The unmatched-pattern warning, with and without the `Warning:` prefix —
	// the prefix is not guaranteed across pi versions, and the bare form is
	// what slips past a naive guard into a bogus `No/models` model.
	const prefixed = `Warning: No models match pattern "opencode-go/mimo-v2-omni"`
	const bare = `No models match pattern "opencode-go/mimo-v2-pro"`

	cases := []struct {
		name   string
		script string
	}{
		{
			// Newer pi prints the catalog to stdout; the stale-pattern
			// warning goes to stderr and the process exits non-zero.
			name: "catalog on stdout",
			script: "#!/bin/sh\n" +
				"cat <<'EOF'\n" + table + "\nEOF\n" +
				"echo " + strconv.Quote(prefixed) + " >&2\n" +
				"exit 1\n",
		},
		{
			// Older pi prints the catalog (and the warning) to stderr; same
			// non-zero exit. The stderr fallback must still parse the catalog.
			name: "catalog and prefixed warning on stderr",
			script: "#!/bin/sh\n" +
				"cat >&2 <<'EOF'\n" + table + "\n" + prefixed + "\nEOF\n" +
				"exit 1\n",
		},
		{
			// Same, but the warning has no `Warning:` prefix — must not leak in
			// as a `No/models` row.
			name: "catalog and bare warning on stderr",
			script: "#!/bin/sh\n" +
				"cat >&2 <<'EOF'\n" + table + "\n" + bare + "\nEOF\n" +
				"exit 1\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakePath := filepath.Join(t.TempDir(), "pi")
			writeTestExecutable(t, fakePath, []byte(tc.script))

			models, err := discoverPiModels(context.Background(), fakePath)
			if err != nil {
				t.Fatalf("discoverPiModels: %v", err)
			}
			// Exactly the resolvable model — no warning line coined into a
			// bogus entry, no header row.
			if len(models) != 1 || models[0].ID != "glm-coding-plan/glm-4.7" {
				t.Fatalf("expected exactly [glm-coding-plan/glm-4.7] despite non-zero exit, got %+v", models)
			}
		})
	}
}

// TestDiscoverOpenCodeModelsFallsBackOnVerboseNoise verifies that a non-zero
// `opencode models --verbose` whose stdout is unparseable noise still falls
// back to the plain `opencode models` command instead of returning empty. The
// earlier fix skipped the fallback whenever verbose printed any bytes, which
// regressed this case. Mirrors the pi hardening in #3729.
func TestDiscoverOpenCodeModelsFallsBackOnVerboseNoise(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake opencode binary is a /bin/sh script")
	}

	// `opencode models --verbose` => $2 == "--verbose": emit noise + exit 1.
	// `opencode models`           => no $2: print the plain catalog.
	script := "#!/bin/sh\n" +
		"if [ \"$2\" = \"--verbose\" ]; then\n" +
		"  echo 'panic: catalog sync failed'\n" +
		"  exit 1\n" +
		"fi\n" +
		"echo 'openai/gpt-4o'\n"

	fakePath := filepath.Join(t.TempDir(), "opencode")
	writeTestExecutable(t, fakePath, []byte(script))

	models, err := discoverOpenCodeModels(context.Background(), fakePath)
	if err != nil {
		t.Fatalf("discoverOpenCodeModels: %v", err)
	}
	if len(models) != 1 || models[0].ID != "openai/gpt-4o" {
		t.Fatalf("expected fallback to plain `opencode models` to yield [openai/gpt-4o], got %+v", models)
	}
}

func TestParseOpenclawAgents(t *testing.T) {
	input := `deepseek-v4   deepseek-v4
claude-sonnet claude-sonnet-4-6
deepseek-v4   deepseek-v4
`
	models := parseOpenclawAgents(input)
	// duplicate deduped; label includes model name.
	if len(models) != 2 {
		t.Fatalf("expected 2 agents, got %d: %+v", len(models), models)
	}
	if models[0].ID != "deepseek-v4" {
		t.Errorf("unexpected first agent: %+v", models[0])
	}
	if models[0].Label != "deepseek-v4 (deepseek-v4)" {
		t.Errorf("unexpected label: %+v", models[0])
	}
	if models[0].Provider != "openclaw" {
		t.Errorf("expected provider openclaw, got %q", models[0].Provider)
	}
}

func TestParseOpenclawAgentsRejectsDecoratedTUI(t *testing.T) {
	// Reproduces the shape of real `openclaw agents list` output
	// that leaked header tokens like "Identity:" / "Workspace:"
	// and single-character box-drawing icons into the dropdown.
	input := `╭───────────────────────────────╮
│                               │
│  ◇  Agents:                   │
│  │                            │
│  │    Identity:               │
│  │    Workspace:              │
│  │    Agent                   │
│  │                            │
╰───────────────────────────────╯
deepseek-v4   deepseek-v4
claude-sonnet claude-sonnet-4-6
`
	models := parseOpenclawAgents(input)
	if len(models) != 2 {
		t.Fatalf("expected 2 agents (decoration skipped), got %d: %+v", len(models), models)
	}
	for _, m := range models {
		if strings.HasSuffix(m.ID, ":") {
			t.Errorf("section header leaked into result: %+v", m)
		}
	}
	if models[0].ID != "deepseek-v4" || models[1].ID != "claude-sonnet" {
		t.Errorf("unexpected agents: %+v", models)
	}
}

func TestParseOpenclawAgentsJSONArray(t *testing.T) {
	input := []byte(`[
    {"name": "deepseek-v4", "model": "deepseek-v4"},
    {"name": "claude-sonnet", "model": "claude-sonnet-4-6"}
]`)
	models, ok := parseOpenclawAgentsJSON(input)
	if !ok {
		t.Fatal("expected parseOpenclawAgentsJSON to accept an array")
	}
	if len(models) != 2 {
		t.Fatalf("got %d, want 2: %+v", len(models), models)
	}
	if models[0].ID != "deepseek-v4" || models[0].Label != "deepseek-v4 (deepseek-v4)" {
		t.Errorf("unexpected first entry: %+v", models[0])
	}
}

func TestParseOpenclawAgentsJSONWrapped(t *testing.T) {
	input := []byte(`{"agents": [{"name": "foo", "model": "bar"}]}`)
	models, ok := parseOpenclawAgentsJSON(input)
	if !ok {
		t.Fatal("expected parseOpenclawAgentsJSON to accept wrapped object")
	}
	if len(models) != 1 || models[0].ID != "foo" {
		t.Errorf("unexpected: %+v", models)
	}
}

func TestOpenclawEntriesToModelsUsesIDOverName(t *testing.T) {
	// When both id and name are present, Model.ID should use the id field
	// because openclaw resolves --agent by id. Names with spaces (e.g.
	// "Sub2API OPS") would be mangled by openclaw's normalizeAgentId.
	input := []byte(`[{"id": "sub2api", "name": "Sub2API OPS", "model": "gpt-4o"}]`)
	models, ok := parseOpenclawAgentsJSON(input)
	if !ok {
		t.Fatal("expected parseOpenclawAgentsJSON to accept array")
	}
	if len(models) != 1 {
		t.Fatalf("got %d models, want 1", len(models))
	}
	if models[0].ID != "sub2api" {
		t.Errorf("Model.ID = %q, want %q (should use id, not name)", models[0].ID, "sub2api")
	}
	if models[0].Label != "Sub2API OPS (gpt-4o)" {
		t.Errorf("Model.Label = %q, want %q (should use name for display)", models[0].Label, "Sub2API OPS (gpt-4o)")
	}
}

func TestParseOpenclawAgentsJSONRejectsGarbage(t *testing.T) {
	if _, ok := parseOpenclawAgentsJSON([]byte("not json")); ok {
		t.Error("expected ok=false for non-JSON")
	}
}

func TestParseCursorModels(t *testing.T) {
	input := `Available models

auto - Auto
composer-2-fast - Composer 2 Fast (current, default)
composer-2 - Composer 2
claude-4.6-sonnet-medium - Sonnet 4.6 1M
claude-opus-4-7-high - Opus 4.7 1M
gemini-3.1-pro - Gemini 3.1 Pro
`
	models := parseCursorModels(input)
	if len(models) != 6 {
		t.Fatalf("expected 6 models, got %d: %+v", len(models), models)
	}
	ids := map[string]Model{}
	for _, m := range models {
		ids[m.ID] = m
	}
	for _, want := range []string{"auto", "composer-2-fast", "composer-2", "claude-4.6-sonnet-medium", "claude-opus-4-7-high", "gemini-3.1-pro"} {
		if _, ok := ids[want]; !ok {
			t.Errorf("missing expected model %q in: %+v", want, models)
		}
	}
	if def := ids["composer-2-fast"]; !def.Default {
		t.Errorf("composer-2-fast should be marked default, got %+v", def)
	}
	if def := ids["composer-2-fast"]; def.Label != "Composer 2 Fast" {
		t.Errorf("default label should be stripped of parenthetical, got %q", def.Label)
	}
	// Non-default entry should not carry Default=true.
	if auto := ids["auto"]; auto.Default {
		t.Errorf("non-default entry should not be flagged default: %+v", auto)
	}
}

func TestParseCursorModelsSkipsHeaderAndBlankLines(t *testing.T) {
	input := `Available models

composer-2 - Composer 2
`
	models := parseCursorModels(input)
	if len(models) != 1 || models[0].ID != "composer-2" {
		t.Fatalf("unexpected: %+v", models)
	}
}

func TestParseHermesSessionNewModels(t *testing.T) {
	// Mirrors the real shape emitted by hermes'
	// acp_adapter/server.py _build_model_state -> SessionModelState.
	raw := []byte(`{
      "sessionId": "ses_123",
      "models": {
        "availableModels": [
          {"modelId": "nous:moonshotai/kimi-k2.5", "name": "moonshotai/kimi-k2.5", "description": "Provider: Nous"},
          {"modelId": "nous:anthropic/claude-opus-4.7", "name": "anthropic/claude-opus-4.7", "description": "Provider: Nous • current"},
          {"modelId": "nous:moonshotai/kimi-k2.5", "name": "duplicate", "description": "dup"}
        ],
        "currentModelId": "nous:anthropic/claude-opus-4.7"
      }
    }`)
	models := parseACPSessionNewModels(raw)
	if len(models) != 2 {
		t.Fatalf("expected 2 models (duplicate deduped), got %d: %+v", len(models), models)
	}
	if models[0].ID != "nous:moonshotai/kimi-k2.5" || models[0].Provider != "nous" {
		t.Errorf("unexpected first model: %+v", models[0])
	}
	if models[0].Default {
		t.Errorf("non-current entry must not be marked default: %+v", models[0])
	}
	if !models[1].Default {
		t.Errorf("current entry must be marked default: %+v", models[1])
	}
	if models[1].ID != "nous:anthropic/claude-opus-4.7" {
		t.Errorf("expected current model second: %+v", models[1])
	}
}

func TestParseHermesSessionNewModelsPreservesCustomModelIDsWithColons(t *testing.T) {
	raw := []byte(`{
      "sessionId": "ses_123",
      "models": {
        "availableModels": [
          {"modelId": "custom:lfm2.5:8b", "name": "lfm2.5:8b", "description": "Provider: Custom"}
        ],
        "currentModelId": "custom:lfm2.5:8b"
      }
    }`)
	models := parseACPSessionNewModels(raw)
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d: %+v", len(models), models)
	}
	if models[0].ID != "custom:lfm2.5:8b" {
		t.Errorf("model id must be preserved verbatim, got %+v", models[0])
	}
	if models[0].Provider != "custom" {
		t.Errorf("provider should be derived from the first colon only, got %+v", models[0])
	}
	if !models[0].Default {
		t.Errorf("current custom model should be marked default: %+v", models[0])
	}
}

func TestParseHermesSessionNewModelsSnakeCaseAndUnknownNames(t *testing.T) {
	raw := []byte(`{
      "session_id": "ses_123",
      "models": {
        "available_models": [
          {"model_id": "nous:moonshotai/kimi-k2.6", "name": "Unknown", "description": "Provider: Nous"},
          {"model_id": "nous:anthropic/claude-sonnet-4.6", "name": "unknown", "description": "Provider: Nous"}
        ],
        "current_model_id": "nous:moonshotai/kimi-k2.6"
      }
    }`)
	models := parseACPSessionNewModels(raw)
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d: %+v", len(models), models)
	}
	if models[0].Label != "nous:moonshotai/kimi-k2.6" {
		t.Errorf("Unknown label should fall back to model id, got %+v", models[0])
	}
	if !models[0].Default {
		t.Errorf("snake_case current_model_id should mark default: %+v", models[0])
	}
	if models[1].Label != "nous:anthropic/claude-sonnet-4.6" {
		t.Errorf("lowercase unknown label should fall back to model id, got %+v", models[1])
	}
}

func TestParseHermesSessionNewModelsMissingField(t *testing.T) {
	// session/new without the models field — older hermes or
	// failed _build_model_state — should yield nil so the caller
	// can distinguish "no catalog" from "empty catalog".
	raw := []byte(`{"sessionId": "ses_123"}`)
	if got := parseACPSessionNewModels(raw); got != nil && len(got) != 0 {
		t.Errorf("expected nil/empty, got %+v", got)
	}
}

func TestParseHermesSessionNewModelsGarbage(t *testing.T) {
	if got := parseACPSessionNewModels([]byte("not json")); got != nil {
		t.Errorf("expected nil for non-JSON, got %+v", got)
	}
}

func TestHermesModelSelectionSupported(t *testing.T) {
	// Regression guard: hermes now supports model selection via
	// the ACP session/set_model RPC, so the UI dropdown should
	// not be disabled for it.
	if !ModelSelectionSupported("hermes") {
		t.Error("hermes should be model-selection-supported now that set_session_model is wired")
	}
}

// TestAntigravityModelSelectionSupported pins that the antigravity provider
// now reports model selection as supported: agy 1.0.6 added a `--model` flag
// (MUL-3125) and buildAntigravityArgs wires opts.Model through, so the UI
// must render the live picker rather than a disabled "Managed by runtime"
// label.
func TestAntigravityModelSelectionSupported(t *testing.T) {
	if !ModelSelectionSupported("antigravity") {
		t.Error("antigravity should be model-selection-supported now that agy 1.0.6 has --model")
	}
}

// TestParseAntigravityModels covers the `agy models` line-per-name format:
// each non-blank line becomes a Model whose ID and Label are the verbatim
// display string `--model` expects, duplicates collapse, and blanks drop.
func TestParseAntigravityModels(t *testing.T) {
	t.Parallel()

	out := strings.Join([]string{
		"Gemini 3.5 Flash (Medium)",
		"Claude Opus 4.6 (Thinking)",
		"", // blank line — skipped
		"GPT-OSS 120B (Medium)",
		"Claude Opus 4.6 (Thinking)", // duplicate — collapsed
	}, "\n")

	got := parseAntigravityModels(out)
	want := []Model{
		{ID: "Gemini 3.5 Flash (Medium)", Label: "Gemini 3.5 Flash (Medium)", Provider: "antigravity"},
		{ID: "Claude Opus 4.6 (Thinking)", Label: "Claude Opus 4.6 (Thinking)", Provider: "antigravity"},
		{ID: "GPT-OSS 120B (Medium)", Label: "GPT-OSS 120B (Medium)", Provider: "antigravity"},
	}
	if len(got) != len(want) {
		t.Fatalf("parseAntigravityModels len = %d, want %d (%+v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("model[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestParseAntigravityModelsEmpty pins that empty / whitespace-only output
// yields no models (so cachedDiscovery uses only its short negative TTL,
// rather than retaining a blank catalog for the normal positive TTL).
func TestParseAntigravityModelsEmpty(t *testing.T) {
	t.Parallel()
	if got := parseAntigravityModels("   \n\t\n"); len(got) != 0 {
		t.Errorf("expected no models for blank output, got %+v", got)
	}
}

func TestDiscoverAntigravityModelsSurfacesCommandFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows model discovery intentionally fails closed before process start")
	}
	t.Parallel()
	fake := filepath.Join(t.TempDir(), "agy")
	writeTestExecutable(t, fake, []byte("#!/bin/sh\necho broken >&2\nexit 7\n"))

	_, err := discoverAntigravityModels(context.Background(), fake)
	if err == nil || !strings.Contains(err.Error(), "model discovery failed") {
		t.Fatalf("expected explicit command failure, got %v", err)
	}
}

func TestDiscoverAntigravityModelsSurfacesTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows model discovery intentionally fails closed before process start")
	}
	t.Parallel()
	fake := filepath.Join(t.TempDir(), "agy")
	writeTestExecutable(t, fake, []byte("#!/bin/sh\nexec sleep 5\n"))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := discoverAntigravityModels(ctx, fake)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected explicit timeout, got %v", err)
	}
}

func TestListModelsAntigravityCachesPerExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows model discovery intentionally fails closed before process start")
	}
	fakeDir := t.TempDir()
	first := filepath.Join(fakeDir, "agy-first")
	second := filepath.Join(fakeDir, "agy-second")
	writeTestExecutable(t, first, []byte("#!/bin/sh\necho 'First Model'\n"))
	writeTestExecutable(t, second, []byte("#!/bin/sh\necho 'Second Model'\n"))

	for _, path := range []string{first, second} {
		key := discoveryCacheKey("antigravity", path)
		modelCacheMu.Lock()
		delete(modelCache, key)
		modelCacheMu.Unlock()
		t.Cleanup(func() {
			modelCacheMu.Lock()
			delete(modelCache, key)
			modelCacheMu.Unlock()
		})
	}

	gotFirst, err := ListModels(context.Background(), "antigravity", first)
	if err != nil {
		t.Fatal(err)
	}
	gotSecond, err := ListModels(context.Background(), "antigravity", second)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotFirst) != 1 || gotFirst[0].ID != "First Model" {
		t.Fatalf("first executable models = %+v", gotFirst)
	}
	if len(gotSecond) != 1 || gotSecond[0].ID != "Second Model" {
		t.Fatalf("second executable models = %+v", gotSecond)
	}
}

func TestListModelsCursorCachesPerExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows model discovery intentionally fails closed before process start")
	}
	dir := t.TempDir()
	first := filepath.Join(dir, "cursor-first")
	second := filepath.Join(dir, "cursor-second")
	writeTestExecutable(t, first, []byte("#!/bin/sh\nprintf 'Available models\\nfirst-model - First Model (default)\\n'\n"))
	writeTestExecutable(t, second, []byte("#!/bin/sh\nprintf 'Available models\\nsecond-model - Second Model (default)\\n'\n"))

	firstKey := discoveryCacheKey("cursor", first)
	secondKey := discoveryCacheKey("cursor", second)
	modelCacheMu.Lock()
	delete(modelCache, firstKey)
	delete(modelCache, secondKey)
	delete(modelCacheFlights, firstKey)
	delete(modelCacheFlights, secondKey)
	modelCacheMu.Unlock()
	t.Cleanup(func() {
		modelCacheMu.Lock()
		delete(modelCache, firstKey)
		delete(modelCache, secondKey)
		delete(modelCacheFlights, firstKey)
		delete(modelCacheFlights, secondKey)
		modelCacheMu.Unlock()
	})

	firstModels, err := ListModels(context.Background(), "cursor", first)
	if err != nil {
		t.Fatal(err)
	}
	secondModels, err := ListModels(context.Background(), "cursor", second)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstModels) != 1 || firstModels[0].ID != "first-model" {
		t.Fatalf("first Cursor executable catalog = %+v", firstModels)
	}
	if len(secondModels) != 1 || secondModels[0].ID != "second-model" {
		t.Fatalf("second Cursor executable catalog = %+v", secondModels)
	}
}

func TestCachedDiscovery(t *testing.T) {
	calls := 0
	fn := func() ([]Model, error) {
		calls++
		return []Model{{ID: "x", Label: "x"}}, nil
	}
	// First call populates the cache; reset for isolation.
	modelCacheMu.Lock()
	delete(modelCache, "testkey")
	modelCacheMu.Unlock()

	if _, err := cachedDiscovery(context.Background(), "testkey", fn); err != nil {
		t.Fatal(err)
	}
	if _, err := cachedDiscovery(context.Background(), "testkey", fn); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("expected 1 underlying call due to cache, got %d", calls)
	}
}

func TestCachedDiscoveryBoundedAndDeletesExpiredEntries(t *testing.T) {
	modelCacheMu.Lock()
	modelCache = map[string]modelCacheEntry{}
	modelCacheFlights = map[string]*modelDiscoveryFlight{}
	modelCacheSequence = 0
	modelCacheMu.Unlock()
	t.Cleanup(func() {
		modelCacheMu.Lock()
		modelCache = map[string]modelCacheEntry{}
		modelCacheFlights = map[string]*modelDiscoveryFlight{}
		modelCacheSequence = 0
		modelCacheMu.Unlock()
	})

	for i := 0; i <= modelCacheMaxEntries; i++ {
		key := "bounded-" + strconv.Itoa(i)
		if _, err := cachedDiscovery(context.Background(), key, func() ([]Model, error) {
			return []Model{{ID: key, Label: key}}, nil
		}); err != nil {
			t.Fatal(err)
		}
	}

	modelCacheMu.Lock()
	if len(modelCache) != modelCacheMaxEntries {
		t.Fatalf("model cache size = %d, want %d", len(modelCache), modelCacheMaxEntries)
	}
	if _, ok := modelCache["bounded-0"]; ok {
		t.Fatal("oldest model cache entry was not evicted")
	}
	expiredKey := "bounded-" + strconv.Itoa(modelCacheMaxEntries)
	expired := modelCache[expiredKey]
	expired.refreshAfter = time.Now().Add(-time.Second)
	expired.hardExpiresAt = time.Now().Add(-time.Second)
	modelCache[expiredKey] = expired
	modelCacheMu.Unlock()

	if _, err := cachedDiscovery(context.Background(), "bounded-1", func() ([]Model, error) {
		t.Fatal("unexpired cache hit unexpectedly rediscovered")
		return nil, nil
	}); err != nil {
		t.Fatal(err)
	}
	modelCacheMu.Lock()
	_, expiredStillPresent := modelCache[expiredKey]
	modelCacheMu.Unlock()
	if expiredStillPresent {
		t.Fatal("expired model cache entry was not deleted")
	}

	oversized := make([]Model, modelCacheMaxModels+10)
	for i := range oversized {
		oversized[i] = Model{ID: "oversized-" + strconv.Itoa(i)}
	}
	got, err := cachedDiscovery(context.Background(), "oversized-models", func() ([]Model, error) {
		return oversized, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != modelCacheMaxModels {
		t.Fatalf("cached model rows = %d, want %d", len(got), modelCacheMaxModels)
	}
}

func TestCachedDiscoveryServesLastKnownUntilHardExpiry(t *testing.T) {
	const key = "two-tier-last-known"
	modelCacheMu.Lock()
	delete(modelCache, key)
	delete(modelCacheFlights, key)
	modelCacheMu.Unlock()
	t.Cleanup(func() {
		modelCacheMu.Lock()
		delete(modelCache, key)
		delete(modelCacheFlights, key)
		modelCacheMu.Unlock()
	})

	want := []Model{{ID: "provider/last-known", Label: "Last known"}}
	if _, err := cachedDiscovery(context.Background(), key, func() ([]Model, error) {
		return want, nil
	}); err != nil {
		t.Fatal(err)
	}
	modelCacheMu.Lock()
	entry := modelCache[key]
	hardExpiry := entry.hardExpiresAt
	entry.refreshAfter = time.Now().Add(-time.Second)
	modelCache[key] = entry
	modelCacheMu.Unlock()

	var refreshCalls atomic.Int32
	refresh := func() ([]Model, error) {
		refreshCalls.Add(1)
		return nil, errors.New("synthetic transient discovery failure")
	}
	got, err := cachedDiscovery(context.Background(), key, refresh)
	if err != nil {
		t.Fatalf("stale fallback returned error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stale fallback = %+v, want %+v", got, want)
	}
	if _, err := cachedDiscovery(context.Background(), key, refresh); err != nil {
		t.Fatal(err)
	}
	if got := refreshCalls.Load(); got != 1 {
		t.Fatalf("failed refresh calls = %d, want 1 during negative backoff", got)
	}

	modelCacheMu.Lock()
	entry = modelCache[key]
	if !entry.hardExpiresAt.Equal(hardExpiry) {
		modelCacheMu.Unlock()
		t.Fatalf("stale fallback extended hard expiry: got %v want %v", entry.hardExpiresAt, hardExpiry)
	}
	entry.refreshAfter = time.Now().Add(-time.Second)
	entry.hardExpiresAt = time.Now().Add(-time.Second)
	modelCache[key] = entry
	modelCacheMu.Unlock()

	if _, err := cachedDiscovery(context.Background(), key, refresh); err == nil {
		t.Fatal("hard-expired last-known catalog unexpectedly masked refresh failure")
	}
	modelCacheMu.Lock()
	_, retained := modelCache[key]
	modelCacheMu.Unlock()
	if retained {
		t.Fatal("hard-expired last-known catalog was not deleted")
	}
}

func TestExecutableDiscoveryKeysIsolateFreshAndNegativeCatalogs(t *testing.T) {
	dir := t.TempDir()
	firstExecutable := filepath.Join(dir, "cursor-first")
	secondExecutable := filepath.Join(dir, "cursor-second")
	writeTestExecutable(t, firstExecutable, []byte("synthetic executable one"))
	writeTestExecutable(t, secondExecutable, []byte("synthetic executable two"))

	firstKey := discoveryCacheKey("cursor", firstExecutable)
	secondKey := discoveryCacheKey("cursor", secondExecutable)
	if firstKey == secondKey {
		t.Fatal("distinct validated executables produced the same discovery key")
	}
	for _, key := range []string{firstKey, secondKey} {
		modelCacheMu.Lock()
		delete(modelCache, key)
		delete(modelCacheFlights, key)
		modelCacheMu.Unlock()
	}
	t.Cleanup(func() {
		modelCacheMu.Lock()
		delete(modelCache, firstKey)
		delete(modelCache, secondKey)
		delete(modelCacheFlights, firstKey)
		delete(modelCacheFlights, secondKey)
		modelCacheMu.Unlock()
	})

	firstCatalog := []Model{{ID: "synthetic/first", Label: "First"}}
	secondCatalog := []Model{{ID: "synthetic/second", Label: "Second"}}
	gotFirst, err := cachedDiscovery(context.Background(), firstKey, func() ([]Model, error) { return firstCatalog, nil })
	if err != nil || !reflect.DeepEqual(gotFirst, firstCatalog) {
		t.Fatalf("first executable fresh catalog = %+v, err=%v", gotFirst, err)
	}
	gotSecond, err := cachedDiscovery(context.Background(), secondKey, func() ([]Model, error) { return secondCatalog, nil })
	if err != nil || !reflect.DeepEqual(gotSecond, secondCatalog) {
		t.Fatalf("second executable fresh catalog = %+v, err=%v", gotSecond, err)
	}

	modelCacheMu.Lock()
	delete(modelCache, secondKey)
	modelCacheMu.Unlock()
	if got, err := cachedDiscovery(context.Background(), secondKey, func() ([]Model, error) { return []Model{}, nil }); err != nil || len(got) != 0 {
		t.Fatalf("second executable negative catalog = %+v, err=%v", got, err)
	}
	var unexpectedSecondRefresh atomic.Int32
	if got, err := cachedDiscovery(context.Background(), secondKey, func() ([]Model, error) {
		unexpectedSecondRefresh.Add(1)
		return secondCatalog, nil
	}); err != nil || len(got) != 0 {
		t.Fatalf("second executable negative cache = %+v, err=%v", got, err)
	}
	if unexpectedSecondRefresh.Load() != 0 {
		t.Fatal("negative cache did not suppress the second executable refresh")
	}
	if got, err := cachedDiscovery(context.Background(), firstKey, func() ([]Model, error) {
		return nil, errors.New("first executable unexpectedly refreshed")
	}); err != nil || !reflect.DeepEqual(got, firstCatalog) {
		t.Fatalf("second executable negative state contaminated first catalog: got=%+v err=%v", got, err)
	}
}

func TestEveryExecutableBackedProviderKeysCustomPathsIndependently(t *testing.T) {
	dir := t.TempDir()
	firstExecutable := filepath.Join(dir, "runtime-first")
	secondExecutable := filepath.Join(dir, "runtime-second")
	writeTestExecutable(t, firstExecutable, []byte("synthetic executable one"))
	writeTestExecutable(t, secondExecutable, []byte("synthetic executable two"))

	providers := []string{
		"cline", "antigravity", "cursor", "copilot", "hermes", "kimi",
		"kiro", "qoder", "opencode", "pi", "openclaw", "codebuddy",
	}
	seen := map[string]string{}
	for _, provider := range providers {
		firstKey := discoveryCacheKey(provider, firstExecutable)
		secondKey := discoveryCacheKey(provider, secondExecutable)
		if firstKey == secondKey {
			t.Errorf("provider %s collapsed two custom executable paths", provider)
		}
		if previousProvider, exists := seen[firstKey]; exists {
			t.Errorf("providers %s and %s share discovery key %q", previousProvider, provider, firstKey)
		}
		seen[firstKey] = provider
	}
}

func TestExecutableDiscoveryKeysIsolateStaleCatalogsAndRefreshErrors(t *testing.T) {
	dir := t.TempDir()
	firstExecutable := filepath.Join(dir, "codebuddy-first")
	secondExecutable := filepath.Join(dir, "codebuddy-second")
	writeTestExecutable(t, firstExecutable, []byte("synthetic executable one"))
	writeTestExecutable(t, secondExecutable, []byte("synthetic executable two"))

	firstKey := discoveryCacheKey("codebuddy", firstExecutable)
	secondKey := discoveryCacheKey("codebuddy", secondExecutable)
	firstCatalog := []Model{{ID: "synthetic/first-stale", Label: "First stale"}}
	secondCatalog := []Model{{ID: "synthetic/second-stale", Label: "Second stale"}}
	for key, catalog := range map[string][]Model{firstKey: firstCatalog, secondKey: secondCatalog} {
		modelCacheMu.Lock()
		delete(modelCache, key)
		delete(modelCacheFlights, key)
		modelCacheMu.Unlock()
		if _, err := cachedDiscovery(context.Background(), key, func() ([]Model, error) { return catalog, nil }); err != nil {
			t.Fatal(err)
		}
		modelCacheMu.Lock()
		entry := modelCache[key]
		entry.refreshAfter = time.Now().Add(-time.Second)
		modelCache[key] = entry
		modelCacheMu.Unlock()
	}
	t.Cleanup(func() {
		modelCacheMu.Lock()
		delete(modelCache, firstKey)
		delete(modelCache, secondKey)
		delete(modelCacheFlights, firstKey)
		delete(modelCacheFlights, secondKey)
		modelCacheMu.Unlock()
	})

	firstResult, firstErr := cachedDiscovery(context.Background(), firstKey, func() ([]Model, error) {
		return nil, errors.New("synthetic first executable refresh error")
	})
	secondResult, secondErr := cachedDiscovery(context.Background(), secondKey, func() ([]Model, error) {
		return nil, errors.New("synthetic second executable refresh error")
	})
	if firstErr != nil || !reflect.DeepEqual(firstResult, firstCatalog) {
		t.Fatalf("first stale fallback = %+v, err=%v", firstResult, firstErr)
	}
	if secondErr != nil || !reflect.DeepEqual(secondResult, secondCatalog) {
		t.Fatalf("second stale fallback = %+v, err=%v", secondResult, secondErr)
	}
}

func TestCachedDiscoveryBlankRefreshKeepsLastKnownWithNegativeBackoff(t *testing.T) {
	const key = "two-tier-blank-refresh"
	modelCacheMu.Lock()
	delete(modelCache, key)
	delete(modelCacheFlights, key)
	modelCacheMu.Unlock()
	t.Cleanup(func() {
		modelCacheMu.Lock()
		delete(modelCache, key)
		delete(modelCacheFlights, key)
		modelCacheMu.Unlock()
	})

	want := []Model{{ID: "provider/last-known", Label: "Last known"}}
	if _, err := cachedDiscovery(context.Background(), key, func() ([]Model, error) { return want, nil }); err != nil {
		t.Fatal(err)
	}
	modelCacheMu.Lock()
	entry := modelCache[key]
	entry.refreshAfter = time.Now().Add(-time.Second)
	modelCache[key] = entry
	modelCacheMu.Unlock()

	var calls atomic.Int32
	blank := func() ([]Model, error) {
		calls.Add(1)
		return []Model{}, nil
	}
	for i := 0; i < 2; i++ {
		got, err := cachedDiscovery(context.Background(), key, blank)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("blank refresh fallback = %+v, want %+v", got, want)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("blank refresh calls = %d, want 1 during negative backoff", got)
	}
}

func TestCachedDiscoverySingleFlight(t *testing.T) {
	const key = "single-flight"
	modelCacheMu.Lock()
	delete(modelCache, key)
	delete(modelCacheFlights, key)
	modelCacheMu.Unlock()

	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	var startOnce sync.Once
	fn := func() ([]Model, error) {
		calls.Add(1)
		startOnce.Do(func() { close(started) })
		<-release
		return []Model{{ID: "provider/model", Label: "Model"}}, nil
	}

	const callers = 8
	errs := make(chan error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cachedDiscovery(context.Background(), key, fn)
			errs <- err
		}()
	}
	<-started
	close(release)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("discovery calls = %d, want 1", got)
	}
}

func TestCachedDiscoveryCancelledLeaderDoesNotPoisonWaiterOrCache(t *testing.T) {
	const key = "cancelled-flight"
	modelCacheMu.Lock()
	delete(modelCache, key)
	delete(modelCacheFlights, key)
	modelCacheMu.Unlock()

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	leaderStarted := make(chan struct{})
	leaderDone := make(chan error, 1)
	go func() {
		_, err := cachedDiscovery(leaderCtx, key, func() ([]Model, error) {
			close(leaderStarted)
			<-leaderCtx.Done()
			return nil, leaderCtx.Err()
		})
		leaderDone <- err
	}()
	<-leaderStarted

	var followerCalls atomic.Int32
	followerDone := make(chan []Model, 1)
	followerErr := make(chan error, 1)
	go func() {
		models, err := cachedDiscovery(context.Background(), key, func() ([]Model, error) {
			followerCalls.Add(1)
			return []Model{{ID: "provider/recovered", Label: "Recovered"}}, nil
		})
		followerDone <- models
		followerErr <- err
	}()

	cancelLeader()
	if err := <-leaderDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("leader error = %v, want context canceled", err)
	}
	models := <-followerDone
	if err := <-followerErr; err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].ID != "provider/recovered" {
		t.Fatalf("follower models = %+v", models)
	}
	if got := followerCalls.Load(); got != 1 {
		t.Fatalf("follower discovery calls = %d, want 1", got)
	}

	if _, err := cachedDiscovery(context.Background(), key, func() ([]Model, error) {
		t.Fatal("recovered result was not cached")
		return nil, nil
	}); err != nil {
		t.Fatal(err)
	}
}
