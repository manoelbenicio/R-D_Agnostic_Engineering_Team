package execenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareOpenCodeHomePerAccountIsolatesDataAndConfig(t *testing.T) {
	t.Parallel()

	accountA := filepath.Join(t.TempDir(), "account-a")
	accountB := filepath.Join(t.TempDir(), "account-b")
	writeTestCredential(t, filepath.Join(accountA, ".local", "share", "opencode", "auth.json"), `{"account":"A"}`)
	writeTestCredential(t, filepath.Join(accountA, ".config", "opencode", "opencode.json"), `{"provider":"A"}`)
	writeTestCredential(t, filepath.Join(accountB, ".local", "share", "opencode", "auth.json"), `{"account":"B"}`)
	writeTestCredential(t, filepath.Join(accountB, ".config", "opencode", "opencode.json"), `{"provider":"B"}`)

	envA, err := Prepare(PrepareParams{
		WorkspacesRoot:        t.TempDir(),
		WorkspaceID:           "ws-test",
		TaskID:                "11111111-2222-3333-4444-555555555555",
		Provider:              "opencode",
		CredentialAccountHome: accountA,
		Task:                  TaskContextForEnv{IssueID: "issue-1"},
	}, testLogger())
	if err != nil {
		t.Fatalf("Prepare opencode A: %v", err)
	}
	defer envA.Cleanup(true)

	envB, err := Prepare(PrepareParams{
		WorkspacesRoot:        t.TempDir(),
		WorkspaceID:           "ws-test",
		TaskID:                "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Provider:              "opencode",
		CredentialAccountHome: accountB,
		Task:                  TaskContextForEnv{IssueID: "issue-1"},
	}, testLogger())
	if err != nil {
		t.Fatalf("Prepare opencode B: %v", err)
	}
	defer envB.Cleanup(true)

	authA := filepath.Join(envA.OpenCodeDataHome, "opencode", "auth.json")
	authB := filepath.Join(envB.OpenCodeDataHome, "opencode", "auth.json")
	configA := filepath.Join(envA.OpenCodeConfigHome, "opencode", "opencode.json")
	configB := filepath.Join(envB.OpenCodeConfigHome, "opencode", "opencode.json")
	assertFileContent(t, authA, `{"account":"A"}`)
	assertFileContent(t, authB, `{"account":"B"}`)
	assertFileContent(t, configA, `{"provider":"A"}`)
	assertFileContent(t, configB, `{"provider":"B"}`)

	envVars := envA.CredentialEnv("opencode")
	if envVars["XDG_DATA_HOME"] != envA.OpenCodeDataHome {
		t.Fatalf("XDG_DATA_HOME = %q, want %q", envVars["XDG_DATA_HOME"], envA.OpenCodeDataHome)
	}
	if envVars["XDG_CONFIG_HOME"] != envA.OpenCodeConfigHome {
		t.Fatalf("XDG_CONFIG_HOME = %q, want %q", envVars["XDG_CONFIG_HOME"], envA.OpenCodeConfigHome)
	}

	if err := os.WriteFile(authA, []byte(`{"account":"A-refreshed-in-task"}`), 0o600); err != nil {
		t.Fatalf("simulate opencode refresh on A: %v", err)
	}
	assertFileContent(t, authB, `{"account":"B"}`)
	assertFileContent(t, filepath.Join(accountA, ".local", "share", "opencode", "auth.json"), `{"account":"A"}`)

	writeTestCredential(t, filepath.Join(accountA, ".local", "share", "opencode", "auth.json"), `{"account":"A-source-refresh"}`)
	reused := Reuse(ReuseParams{
		WorkDir:               envA.WorkDir,
		Provider:              "opencode",
		CredentialAccountHome: accountA,
		Task:                  TaskContextForEnv{IssueID: "issue-1"},
	}, testLogger())
	if reused == nil {
		t.Fatal("Reuse opencode A returned nil")
	}
	assertFileContent(t, filepath.Join(reused.OpenCodeDataHome, "opencode", "auth.json"), `{"account":"A-source-refresh"}`)
	assertFileContent(t, authB, `{"account":"B"}`)
}

func TestPrepareGLMUsesOpenCodeCompatibleIsolation(t *testing.T) {
	t.Parallel()

	accountHome := t.TempDir()
	writeTestCredential(t, filepath.Join(accountHome, ".local", "share", "opencode", "auth.json"), `{"account":"glm"}`)
	writeTestCredential(t, filepath.Join(accountHome, ".config", "opencode", "opencode.json"), `{"provider":"glm"}`)

	env, err := Prepare(PrepareParams{
		WorkspacesRoot:        t.TempDir(),
		WorkspaceID:           "ws-test",
		TaskID:                "11111111-2222-3333-4444-555555555555",
		Provider:              "glm",
		CredentialAccountHome: accountHome,
		Task:                  TaskContextForEnv{IssueID: "issue-1"},
	}, testLogger())
	if err != nil {
		t.Fatalf("Prepare glm: %v", err)
	}
	defer env.Cleanup(true)

	assertFileContent(t, filepath.Join(env.OpenCodeDataHome, "opencode", "auth.json"), `{"account":"glm"}`)
	assertFileContent(t, filepath.Join(env.OpenCodeConfigHome, "opencode", "opencode.json"), `{"provider":"glm"}`)

	envVars := env.CredentialEnv("glm")
	if envVars["XDG_DATA_HOME"] != env.OpenCodeDataHome {
		t.Fatalf("glm XDG_DATA_HOME = %q, want %q", envVars["XDG_DATA_HOME"], env.OpenCodeDataHome)
	}
	if envVars["XDG_CONFIG_HOME"] != env.OpenCodeConfigHome {
		t.Fatalf("glm XDG_CONFIG_HOME = %q, want %q", envVars["XDG_CONFIG_HOME"], env.OpenCodeConfigHome)
	}
}

func TestPrepareOpenCodeHomeAcceptsXDGRotationRoot(t *testing.T) {
	t.Parallel()

	accountHome := t.TempDir()
	writeTestCredential(t, filepath.Join(accountHome, "opencode", "auth.json"), `{"account":"xdg"}`)

	env, err := Prepare(PrepareParams{
		WorkspacesRoot:        t.TempDir(),
		WorkspaceID:           "ws-test",
		TaskID:                "11111111-2222-3333-4444-555555555555",
		Provider:              "opencode",
		CredentialAccountHome: accountHome,
		Task:                  TaskContextForEnv{IssueID: "issue-1"},
	}, testLogger())
	if err != nil {
		t.Fatalf("Prepare opencode xdg root: %v", err)
	}
	defer env.Cleanup(true)

	assertFileContent(t, filepath.Join(env.OpenCodeDataHome, "opencode", "auth.json"), `{"account":"xdg"}`)
}
