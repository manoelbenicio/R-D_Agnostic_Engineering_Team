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
