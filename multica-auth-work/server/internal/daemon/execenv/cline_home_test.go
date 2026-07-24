package execenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareClineHomePerAccountIsolatesDataDir(t *testing.T) {
	t.Parallel()

	accountA := filepath.Join(t.TempDir(), "account-a")
	accountB := filepath.Join(t.TempDir(), "account-b")
	writeTestCredential(t, filepath.Join(accountA, ".cline", "data", "settings", "providers.json"), `{"account":"A"}`)
	writeTestCredential(t, filepath.Join(accountB, ".cline", "data", "settings", "providers.json"), `{"account":"B"}`)

	envA, err := Prepare(PrepareParams{
		WorkspacesRoot:        t.TempDir(),
		WorkspaceID:           "ws-test",
		TaskID:                "11111111-2222-3333-4444-555555555555",
		Provider:              "cline",
		CredentialAccountHome: accountA,
		Task:                  TaskContextForEnv{IssueID: "issue-1"},
	}, testLogger())
	if err != nil {
		t.Fatalf("Prepare cline A: %v", err)
	}
	defer envA.Cleanup(true)

	envB, err := Prepare(PrepareParams{
		WorkspacesRoot:        t.TempDir(),
		WorkspaceID:           "ws-test",
		TaskID:                "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Provider:              "cline",
		CredentialAccountHome: accountB,
		Task:                  TaskContextForEnv{IssueID: "issue-1"},
	}, testLogger())
	if err != nil {
		t.Fatalf("Prepare cline B: %v", err)
	}
	defer envB.Cleanup(true)

	providersA := filepath.Join(envA.ClineDataDir, "data", "settings", "providers.json")
	providersB := filepath.Join(envB.ClineDataDir, "data", "settings", "providers.json")
	assertFileContent(t, providersA, `{"account":"A"}`)
	assertFileContent(t, providersB, `{"account":"B"}`)
	if envA.ClineSandboxDataDir == "" {
		t.Fatal("ClineSandboxDataDir is empty")
	}

	envVars := envA.CredentialEnv("cline")
	if envVars["CLINE_DATA_DIR"] != envA.ClineDataDir {
		t.Fatalf("CLINE_DATA_DIR = %q, want %q", envVars["CLINE_DATA_DIR"], envA.ClineDataDir)
	}
	if envVars["CLINE_SANDBOX"] != "1" {
		t.Fatalf("CLINE_SANDBOX = %q, want 1", envVars["CLINE_SANDBOX"])
	}
	if envVars["CLINE_SANDBOX_DATA_DIR"] != envA.ClineSandboxDataDir {
		t.Fatalf("CLINE_SANDBOX_DATA_DIR = %q, want %q", envVars["CLINE_SANDBOX_DATA_DIR"], envA.ClineSandboxDataDir)
	}

	if err := os.WriteFile(providersA, []byte(`{"account":"A-refreshed-in-task"}`), 0o600); err != nil {
		t.Fatalf("simulate cline refresh on A: %v", err)
	}
	assertFileContent(t, providersB, `{"account":"B"}`)
	assertFileContent(t, filepath.Join(accountA, ".cline", "data", "settings", "providers.json"), `{"account":"A"}`)

	writeTestCredential(t, filepath.Join(accountA, ".cline", "data", "settings", "providers.json"), `{"account":"A-source-refresh"}`)
	reused := Reuse(ReuseParams{
		WorkDir:               envA.WorkDir,
		Provider:              "cline",
		CredentialAccountHome: accountA,
		Task:                  TaskContextForEnv{IssueID: "issue-1"},
	}, testLogger())
	if reused == nil {
		t.Fatal("Reuse cline A returned nil")
	}
	assertFileContent(t, filepath.Join(reused.ClineDataDir, "data", "settings", "providers.json"), `{"account":"A-source-refresh"}`)
	assertFileContent(t, providersB, `{"account":"B"}`)
}

func TestPrepareClineHomeAcceptsDataDirAsAccountHome(t *testing.T) {
	t.Parallel()

	accountHome := t.TempDir()
	writeTestCredential(t, filepath.Join(accountHome, "data", "settings", "providers.json"), `{"account":"direct"}`)

	env, err := Prepare(PrepareParams{
		WorkspacesRoot:        t.TempDir(),
		WorkspaceID:           "ws-test",
		TaskID:                "11111111-2222-3333-4444-555555555555",
		Provider:              "cline",
		CredentialAccountHome: accountHome,
		Task:                  TaskContextForEnv{IssueID: "issue-1"},
	}, testLogger())
	if err != nil {
		t.Fatalf("Prepare cline direct data dir: %v", err)
	}
	defer env.Cleanup(true)

	assertFileContent(t, filepath.Join(env.ClineDataDir, "data", "settings", "providers.json"), `{"account":"direct"}`)
}

// --- D5: credentialless Cline carrier writer (WriteCredentiallessClineConfig) ---
// These cover the credentialless launch path only. The legacy per-account
// prepareClineHome tests above are intentionally left unchanged (regression
// guard for the credentialed path).

func TestWriteCredentiallessClineConfigMaterializesCarrier(t *testing.T) {
	t.Parallel()

	assertPerm := func(path string, want os.FileMode) {
		t.Helper()
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if info.Mode().Perm() != want {
			t.Fatalf("%s mode = %o, want %o", path, info.Mode().Perm(), want)
		}
	}

	// The writer is deliberately schema-agnostic: it writes the caller-validated
	// non-secret carrier bytes verbatim. Use a synthetic non-secret payload.
	clineDataDir := filepath.Join(t.TempDir(), "cline-data")
	raw := []byte(`{"synthetic":"non-secret-cline-carrier"}`)

	if err := WriteCredentiallessClineConfig(clineDataDir, raw); err != nil {
		t.Fatalf("WriteCredentiallessClineConfig: %v", err)
	}

	carrier := filepath.Join(clineDataDir, "settings", "providers.json")
	got, err := os.ReadFile(carrier)
	if err != nil {
		t.Fatalf("read carrier: %v", err)
	}
	if string(got) != string(raw) {
		t.Fatalf("carrier bytes = %q, want verbatim %q", got, raw)
	}
	assertPerm(carrier, 0o600)
	assertPerm(clineDataDir, 0o700)
	assertPerm(filepath.Join(clineDataDir, "settings"), 0o700)

	// Second write replaces (never appends) the carrier in place.
	raw2 := []byte(`{"synthetic":"non-secret-cline-carrier-v2"}`)
	if err := WriteCredentiallessClineConfig(clineDataDir, raw2); err != nil {
		t.Fatalf("second WriteCredentiallessClineConfig: %v", err)
	}
	got2, err := os.ReadFile(carrier)
	if err != nil {
		t.Fatalf("read carrier after replace: %v", err)
	}
	if string(got2) != string(raw2) {
		t.Fatalf("carrier after replace = %q, want %q", got2, raw2)
	}
	assertPerm(carrier, 0o600)
}

func TestWriteCredentiallessClineConfigRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	valid := []byte(`{"synthetic":"carrier"}`)
	absDir := filepath.Join(t.TempDir(), "cline-data")

	if err := WriteCredentiallessClineConfig("", valid); err == nil {
		t.Fatal("empty cline data dir must be rejected")
	}
	if err := WriteCredentiallessClineConfig("relative/cline-data", valid); err == nil {
		t.Fatal("relative cline data dir must be rejected")
	}
	if err := WriteCredentiallessClineConfig(string(filepath.Separator), valid); err == nil {
		t.Fatal("root cline data dir must be rejected")
	}
	if err := WriteCredentiallessClineConfig(absDir, nil); err == nil {
		t.Fatal("empty carrier must be rejected")
	}
	if err := WriteCredentiallessClineConfig(absDir, make([]byte, (64<<10)+1)); err == nil {
		t.Fatal("oversize carrier must be rejected")
	}
	// A rejected input must not leave a carrier behind.
	if _, err := os.Stat(filepath.Join(absDir, "settings", "providers.json")); !os.IsNotExist(err) {
		t.Fatal("carrier written despite rejected inputs")
	}
}
