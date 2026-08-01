package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadProdexLaunchConfigDisabled(t *testing.T) {
	t.Setenv("MULTICA_PRODEX_ENABLED", "")

	cfg, entry, err := loadProdexLaunchConfig()
	if err != nil {
		t.Fatalf("loadProdexLaunchConfig: %v", err)
	}
	if cfg.Enabled {
		t.Fatal("prodex should be disabled by default")
	}
	if entry.Path != "" {
		t.Fatalf("entry.Path = %q, want empty", entry.Path)
	}
}

func TestLoadProdexLaunchConfigRequiresVersionAndCommitPins(t *testing.T) {
	fake := writeFakeExecutable(t, "prodex")
	t.Setenv("MULTICA_PRODEX_ENABLED", "1")
	t.Setenv("MULTICA_PRODEX_PATH", fake)
	t.Setenv("MULTICA_PRODEX_VERSION", "0.246.0")
	t.Setenv("MULTICA_PRODEX_COMMIT", "")

	_, _, err := loadProdexLaunchConfig()
	if err == nil {
		t.Fatal("expected missing commit pin to fail closed")
	}
}

func TestLoadProdexLaunchConfigResolvesPinnedExecutable(t *testing.T) {
	fake := writeFakeExecutable(t, "prodex")
	t.Setenv("MULTICA_PRODEX_ENABLED", "true")
	t.Setenv("MULTICA_PRODEX_PATH", fake)
	t.Setenv("MULTICA_PRODEX_VERSION", "0.246.0")
	t.Setenv("MULTICA_PRODEX_COMMIT", "7750da9b6a5c91a6d429e18e6a4d422cab4bc144")
	t.Setenv("MULTICA_CODEX_MODEL", "gpt-5")

	cfg, entry, err := loadProdexLaunchConfig()
	if err != nil {
		t.Fatalf("loadProdexLaunchConfig: %v", err)
	}
	if !cfg.Enabled {
		t.Fatal("prodex should be enabled")
	}
	if cfg.Path != fake || entry.Path != fake {
		t.Fatalf("resolved path cfg=%q entry=%q, want %q", cfg.Path, entry.Path, fake)
	}
	if cfg.Version == "" || cfg.Commit == "" {
		t.Fatalf("pins not preserved: %+v", cfg)
	}
	if entry.Model != "gpt-5" {
		t.Fatalf("entry.Model = %q, want gpt-5", entry.Model)
	}
	if !cfg.SmartContextShadow || cfg.SmartContextCanary != "0" || !cfg.KillSwitchDefaultOn {
		t.Fatalf("unexpected guardrail defaults: %+v", cfg)
	}
}

func TestApplyProdexEnvOnlyForCodex(t *testing.T) {
	d := &Daemon{cfg: Config{Prodex: ProdexConfig{
		Enabled:             true,
		Version:             "0.246.0",
		Commit:              "7750da9",
		SmartContextShadow:  true,
		SmartContextCanary:  "1",
		KillSwitchDefaultOn: true,
	}}}
	env := map[string]string{}

	d.applyProdexEnv("claude", "/tmp/env-root", env)
	if len(env) != 0 {
		t.Fatalf("non-codex provider should not receive prodex env: %#v", env)
	}

	d.applyProdexEnv("codex", "/tmp/env-root", env)
	if env["PRODEX_HOME"] != filepath.Join("/tmp/env-root", "prodex") {
		t.Fatalf("PRODEX_HOME = %q", env["PRODEX_HOME"])
	}
	if env["MULTICA_PRODEX_ENABLED"] != "1" ||
		env["MULTICA_PRODEX_VERSION"] != "0.246.0" ||
		env["MULTICA_PRODEX_COMMIT"] != "7750da9" ||
		env["PRODEX_SMART_CONTEXT_SHADOW"] != "1" ||
		env["PRODEX_SMART_CONTEXT_CANARY_PERCENT"] != "1" ||
		env["PRODEX_KILL_SWITCH_DEFAULT_ON"] != "1" {
		t.Fatalf("missing prodex launch guard env: %#v", env)
	}
}

func TestProdexEnvKeysAreBlockedFromCustomEnv(t *testing.T) {
	for _, key := range []string{"PRODEX_HOME", "PRODEX_SMART_CONTEXT_SHADOW", "MULTICA_PRODEX_COMMIT"} {
		if !isBlockedEnvKey(key) {
			t.Fatalf("%s must be blocked from custom_env", key)
		}
	}
}

func TestL2SidecarArgsAcceptsProdexSubcommand(t *testing.T) {
	t.Setenv("MULTICA_L2_SIDECAR_ARGS", "run --profile primary -- --model gpt-5")

	args, err := l2SidecarArgs("/opt/prodex/bin/prodex")
	if err != nil {
		t.Fatalf("l2SidecarArgs: %v", err)
	}
	want := []string{"run", "--profile", "primary", "--", "--model", "gpt-5"}
	assertStringSlice(t, args, want)
}

func TestL2SidecarArgsStripsProdexCommandToken(t *testing.T) {
	t.Setenv("MULTICA_L2_SIDECAR_ARGS", "prodex run --profile primary -- --model gpt-5")

	args, err := l2SidecarArgs("/opt/prodex/bin/prodex")
	if err != nil {
		t.Fatalf("l2SidecarArgs: %v", err)
	}
	want := []string{"run", "--profile", "primary", "--", "--model", "gpt-5"}
	assertStringSlice(t, args, want)
}

func TestL2SidecarArgsStripsPinnedProdexPath(t *testing.T) {
	prodexPath := filepath.Join(t.TempDir(), "prodex")
	t.Setenv("MULTICA_L2_SIDECAR_ARGS", prodexPath+" app-server-broker --listen 127.0.0.1:43117")

	args, err := l2SidecarArgs(prodexPath)
	if err != nil {
		t.Fatalf("l2SidecarArgs: %v", err)
	}
	want := []string{"app-server-broker", "--listen", "127.0.0.1:43117"}
	assertStringSlice(t, args, want)
}

func TestL2SidecarArgsRejectsShimExecutable(t *testing.T) {
	shim := filepath.Join(t.TempDir(), "prodex-sidecar")
	t.Setenv("MULTICA_L2_SIDECAR_ARGS", shim+" --listen 127.0.0.1:43117")

	_, err := l2SidecarArgs("/opt/prodex/bin/prodex")
	if err == nil {
		t.Fatal("l2SidecarArgs error = nil, want shim executable rejection")
	}
	if !strings.Contains(err.Error(), "configured prodex") {
		t.Fatalf("error = %q, want configured prodex message", err)
	}
}

func TestProdexSidecarEnvForcesSafeChildEnvAndLoopbackNoProxy(t *testing.T) {
	t.Setenv("PRODEX_ALLOW_UNSAFE_CHILD_ENV", "on")
	t.Setenv("NO_PROXY", "example.test")
	t.Setenv("no_proxy", "localhost")

	env := envMap(prodexSidecarEnv())
	if env["PRODEX_ALLOW_UNSAFE_CHILD_ENV"] != "off" {
		t.Fatalf("PRODEX_ALLOW_UNSAFE_CHILD_ENV = %q, want off", env["PRODEX_ALLOW_UNSAFE_CHILD_ENV"])
	}
	for _, key := range []string{"NO_PROXY", "no_proxy"} {
		for _, required := range []string{"127.0.0.1", "localhost", "::1"} {
			if !csvContains(env[key], required) {
				t.Fatalf("%s = %q, missing %s", key, env[key], required)
			}
		}
	}
}

func writeFakeExecutable(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake executable: %v", err)
	}
	return path
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d; got=%#v want=%#v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q; got=%#v want=%#v", i, got[i], want[i], got, want)
		}
	}
}

func csvContains(value, want string) bool {
	for _, part := range strings.Split(value, ",") {
		if strings.TrimSpace(part) == want {
			return true
		}
	}
	return false
}
