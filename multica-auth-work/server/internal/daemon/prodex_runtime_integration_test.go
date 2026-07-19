package daemon

import (
	"strings"
	"testing"
)

// These tests are focused, fully offline unit tests for
// persist-prodex-runtime-integration tasks 1.1-1.3. They assert the behavior
// already implemented in the shared baseline (prodex.go / l2_runtime.go) and
// never touch real credentials, real environment values, or the network:
//   - all environment is set with t.Setenv (auto-restored),
//   - all executable paths are synthetic temp fixtures (writeFakeExecutable),
//   - all endpoints/tokens are obviously synthetic loopback constants.

const (
	// synthetic, loopback-only test constants (never real values)
	testL2LoopbackBaseURL = "http://127.0.0.1:43117"
	testL2SyntheticToken   = "synthetic-test-bearer-token-not-a-secret"
	testL2SyntheticTenant  = "tenant-synthetic"
	testProdexTestVersion  = "0.246.0"
	testProdexTestCommit   = "0000000000000000000000000000000000000000"
	testProdexPGURL        = "postgres://synthetic:synthetic@127.0.0.1:5432/synthetic"
)

// clearProdexRuntimeEnv neutralizes every environment key the loaders read so a
// test starts from a known, synthetic-only baseline regardless of the host env.
func clearProdexRuntimeEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"MULTICA_PRODEX_ENABLED", "MULTICA_PRODEX_REQUIRED", "MULTICA_PRODEX_PATH",
		"MULTICA_PRODEX_VERSION", "MULTICA_PRODEX_COMMIT", "MULTICA_PRODEX_CONFIG_SOURCE",
		"MULTICA_L2_ENABLED", "MULTICA_L2_BASE_URL", "MULTICA_L2_BEARER_TOKEN",
		"MULTICA_L2_SIDECAR_PATH", "MULTICA_L2_SIDECAR_ARGS", "MULTICA_L2_POLICY_ID",
		"MULTICA_L2_TENANT_ID", "MULTICA_L2_TIMEOUT", "PRODEX_PG_URL",
	} {
		t.Setenv(key, "")
	}
}

// --- Task 1.1: MULTICA_L2_SIDECAR_PATH validated independently of MULTICA_PRODEX_PATH ---

func TestLoadL2RuntimeConfigRequiresSidecarPath(t *testing.T) {
	clearProdexRuntimeEnv(t)
	t.Setenv("MULTICA_L2_ENABLED", "1")
	t.Setenv("MULTICA_L2_BASE_URL", testL2LoopbackBaseURL)
	t.Setenv("MULTICA_L2_BEARER_TOKEN", testL2SyntheticToken)
	// MULTICA_L2_SIDECAR_PATH intentionally left empty.

	_, err := loadL2RuntimeConfig()
	if err == nil {
		t.Fatal("expected missing MULTICA_L2_SIDECAR_PATH to fail closed")
	}
	if !strings.Contains(err.Error(), "MULTICA_L2_SIDECAR_PATH is required") {
		t.Fatalf("error = %q, want MULTICA_L2_SIDECAR_PATH requirement", err)
	}
}

func TestLoadL2RuntimeConfigRejectsMissingSidecarExecutable(t *testing.T) {
	clearProdexRuntimeEnv(t)
	t.Setenv("MULTICA_L2_ENABLED", "1")
	t.Setenv("MULTICA_L2_BASE_URL", testL2LoopbackBaseURL)
	t.Setenv("MULTICA_L2_BEARER_TOKEN", testL2SyntheticToken)
	// Point at a path that does not exist so lookup fails closed.
	t.Setenv("MULTICA_L2_SIDECAR_PATH", "/nonexistent/synthetic/prodex-sidecar")

	_, err := loadL2RuntimeConfig()
	if err == nil {
		t.Fatal("expected unresolved adapter executable to fail closed")
	}
	if !strings.Contains(err.Error(), "adapter executable") {
		t.Fatalf("error = %q, want adapter executable resolution failure", err)
	}
}

func TestLoadL2RuntimeConfigResolvesSidecarIndependentOfProdexPath(t *testing.T) {
	clearProdexRuntimeEnv(t)
	// Two distinct synthetic executables prove the adapter path is configured
	// and resolved on its own env key, separate from the pinned prodex binary.
	sidecar := writeFakeExecutable(t, "prodex-sidecar")
	prodex := writeFakeExecutable(t, "prodex")
	t.Setenv("MULTICA_L2_ENABLED", "1")
	t.Setenv("MULTICA_L2_BASE_URL", testL2LoopbackBaseURL)
	t.Setenv("MULTICA_L2_BEARER_TOKEN", testL2SyntheticToken)
	t.Setenv("MULTICA_L2_SIDECAR_PATH", sidecar)
	t.Setenv("MULTICA_PRODEX_PATH", prodex) // must NOT be consumed by the L2 loader
	t.Setenv("MULTICA_L2_TENANT_ID", testL2SyntheticTenant)

	cfg, err := loadL2RuntimeConfig()
	if err != nil {
		t.Fatalf("loadL2RuntimeConfig: %v", err)
	}
	if !cfg.Enabled {
		t.Fatal("l2 runtime should be enabled")
	}
	if cfg.SidecarPath != sidecar {
		t.Fatalf("SidecarPath = %q, want adapter fixture %q", cfg.SidecarPath, sidecar)
	}
	if cfg.SidecarPath == prodex {
		t.Fatal("adapter path must be resolved from MULTICA_L2_SIDECAR_PATH, not the pinned prodex path")
	}
	if cfg.TenantID != testL2SyntheticTenant {
		t.Fatalf("TenantID = %q, want %q", cfg.TenantID, testL2SyntheticTenant)
	}
}

// --- Task 1.2: adapter launch passes the pinned prodex path through its env ---

func TestProdexSidecarEnvInjectsPinnedProdexPath(t *testing.T) {
	clearProdexRuntimeEnv(t)
	prodex := writeFakeExecutable(t, "prodex")
	cfg := Config{
		Prodex:              ProdexConfig{Enabled: true, Path: prodex},
		L2Runtime:           L2RuntimeConfig{Enabled: true, BearerToken: testL2SyntheticToken},
		RotationDatabaseURL: testProdexPGURL,
	}

	env := envMap(prodexSidecarEnv(cfg))

	if env["MULTICA_PRODEX_PATH"] != prodex {
		t.Fatalf("MULTICA_PRODEX_PATH = %q, want pinned prodex %q", env["MULTICA_PRODEX_PATH"], prodex)
	}
	if env["MULTICA_L2_BEARER_TOKEN"] != testL2SyntheticToken {
		t.Fatalf("MULTICA_L2_BEARER_TOKEN = %q, want synthetic token", env["MULTICA_L2_BEARER_TOKEN"])
	}
	if env["PRODEX_PG_URL"] != testProdexPGURL {
		t.Fatalf("PRODEX_PG_URL = %q, want rotation database url", env["PRODEX_PG_URL"])
	}
	// Guardrail (Golden Rule 7): unsafe child env forwarding must be forced off.
	if env["PRODEX_ALLOW_UNSAFE_CHILD_ENV"] != "off" {
		t.Fatalf("PRODEX_ALLOW_UNSAFE_CHILD_ENV = %q, want off", env["PRODEX_ALLOW_UNSAFE_CHILD_ENV"])
	}
}

func TestL2SidecarArgsDefaultsToAdapterListenNotProdexPath(t *testing.T) {
	clearProdexRuntimeEnv(t)
	// No MULTICA_L2_SIDECAR_ARGS => default adapter listen args (not a prodex
	// executable path). Confirms the Go lifecycle launches the adapter binary
	// with adapter arguments.
	args, err := l2SidecarArgs()
	if err != nil {
		t.Fatalf("l2SidecarArgs: %v", err)
	}
	want := []string{"127.0.0.1:43117"}
	assertStringSlice(t, args, want)
}

func TestL2SidecarArgsRejectsExecutablePathFirstArg(t *testing.T) {
	clearProdexRuntimeEnv(t)
	// The adapter is the executable; args must be adapter arguments, never an
	// executable path in the first position.
	t.Setenv("MULTICA_L2_SIDECAR_ARGS", "/opt/synthetic/prodex app-server-broker")

	_, err := l2SidecarArgs()
	if err == nil {
		t.Fatal("expected executable-path first arg to be rejected")
	}
	if !strings.Contains(err.Error(), "not an executable path") {
		t.Fatalf("error = %q, want adapter-arguments rejection", err)
	}
}

// --- Task 1.3: MULTICA_PRODEX_REQUIRED fail-closed startup enforcement ---

func TestLoadProdexLaunchConfigRequiredButDisabledFailsClosed(t *testing.T) {
	clearProdexRuntimeEnv(t)
	t.Setenv("MULTICA_PRODEX_REQUIRED", "1")
	// MULTICA_PRODEX_ENABLED intentionally empty => required config downgraded.

	_, _, err := loadProdexLaunchConfig()
	if err == nil {
		t.Fatal("expected required-but-disabled prodex to fail closed")
	}
	if !strings.Contains(err.Error(), "required") || !strings.Contains(err.Error(), "MULTICA_PRODEX_ENABLED") {
		t.Fatalf("error = %q, want required/disabled fail-closed message", err)
	}
}

func TestLoadL2RuntimeConfigRequiredMissingTenantFailsClosed(t *testing.T) {
	clearProdexRuntimeEnv(t)
	sidecar := writeFakeExecutable(t, "prodex-sidecar")
	t.Setenv("MULTICA_L2_ENABLED", "1")
	t.Setenv("MULTICA_L2_BASE_URL", testL2LoopbackBaseURL)
	t.Setenv("MULTICA_L2_BEARER_TOKEN", testL2SyntheticToken)
	t.Setenv("MULTICA_L2_SIDECAR_PATH", sidecar)
	t.Setenv("MULTICA_PRODEX_REQUIRED", "1")
	// MULTICA_L2_TENANT_ID intentionally empty in required mode.

	_, err := loadL2RuntimeConfig()
	if err == nil {
		t.Fatal("expected required mode with missing tenant to fail closed")
	}
	if !strings.Contains(err.Error(), "MULTICA_L2_TENANT_ID") {
		t.Fatalf("error = %q, want missing-tenant fail-closed message", err)
	}
}

func TestLoadL2RuntimeConfigNotRequiredDefaultsTenant(t *testing.T) {
	clearProdexRuntimeEnv(t)
	sidecar := writeFakeExecutable(t, "prodex-sidecar")
	t.Setenv("MULTICA_L2_ENABLED", "1")
	t.Setenv("MULTICA_L2_BASE_URL", testL2LoopbackBaseURL)
	t.Setenv("MULTICA_L2_BEARER_TOKEN", testL2SyntheticToken)
	t.Setenv("MULTICA_L2_SIDECAR_PATH", sidecar)
	// Not required, no tenant => must fall back to "default" (no downgrade error).

	cfg, err := loadL2RuntimeConfig()
	if err != nil {
		t.Fatalf("loadL2RuntimeConfig: %v", err)
	}
	if cfg.TenantID != "default" {
		t.Fatalf("TenantID = %q, want default when not required", cfg.TenantID)
	}
}
