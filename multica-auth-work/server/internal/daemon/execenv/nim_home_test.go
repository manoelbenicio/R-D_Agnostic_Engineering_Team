package execenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareNimHomePerAccountCopiesAPIKey(t *testing.T) {
	t.Parallel()

	accountA := t.TempDir()
	accountB := t.TempDir()
	writeTestCredential(t, filepath.Join(accountA, nimCredentialFileName), "nvapi-account-a\n")
	writeTestCredential(t, filepath.Join(accountB, nimCredentialFileName), "nvapi-account-b\n")

	envA := prepareNIMTestEnv(t, accountA, "11111111-2222-3333-4444-555555555555")
	defer envA.Cleanup(true)
	envB := prepareNIMTestEnv(t, accountB, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	defer envB.Cleanup(true)

	if envA.NIMCredentialPath == envB.NIMCredentialPath {
		t.Fatalf("NIM credential paths overlap: %q", envA.NIMCredentialPath)
	}
	assertNotSymlink(t, envA.NIMCredentialPath)
	assertNotSymlink(t, envB.NIMCredentialPath)
	for _, path := range []string{envA.NIMCredentialPath, envB.NIMCredentialPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat copied NIM credential: %v", err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("copied NIM credential mode = %o, want 600", got)
		}
	}

	if got := envA.CredentialEnv("nim")["NVIDIA_API_KEY"]; got != "nvapi-account-a" {
		t.Fatalf("account A NVIDIA_API_KEY = %q", got)
	}
	if got := envB.CredentialEnv("nim")["NVIDIA_API_KEY"]; got != "nvapi-account-b" {
		t.Fatalf("account B NVIDIA_API_KEY = %q", got)
	}

	if err := os.WriteFile(envA.NIMCredentialPath, []byte("nvapi-task-refresh"), 0o600); err != nil {
		t.Fatalf("refresh task copy: %v", err)
	}
	assertFileContent(t, filepath.Join(accountA, nimCredentialFileName), "nvapi-account-a\n")
	assertFileContent(t, filepath.Join(accountB, nimCredentialFileName), "nvapi-account-b\n")
}

func TestPrepareNimHomeRefreshesOnReuse(t *testing.T) {
	t.Parallel()

	accountHome := t.TempDir()
	writeTestCredential(t, filepath.Join(accountHome, nimCredentialFileName), "nvapi-v1")
	env := prepareNIMTestEnv(t, accountHome, "11111111-2222-3333-4444-555555555555")
	defer env.Cleanup(true)

	writeTestCredential(t, filepath.Join(accountHome, nimCredentialFileName), "nvapi-v2")
	reused := Reuse(ReuseParams{
		WorkDir:               env.WorkDir,
		Provider:              "nim",
		CredentialAccountHome: accountHome,
		Task:                  TaskContextForEnv{IssueID: "issue-nim"},
	}, testLogger())
	if reused == nil {
		t.Fatal("Reuse returned nil")
	}
	if got := reused.CredentialEnv("nim")["NVIDIA_API_KEY"]; got != "nvapi-v2" {
		t.Fatalf("reused NVIDIA_API_KEY = %q, want refreshed value", got)
	}
	assertNotSymlink(t, reused.NIMCredentialPath)
}

func TestPrepareNimHomeFailsClosedForMissingOrEmptyKey(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		content *string
	}{
		{name: "missing"},
		{name: "empty", content: stringPtr("  \n")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			accountHome := t.TempDir()
			if tt.content != nil {
				writeTestCredential(t, filepath.Join(accountHome, nimCredentialFileName), *tt.content)
			}
			_, err := Prepare(PrepareParams{
				WorkspacesRoot:        t.TempDir(),
				WorkspaceID:           "ws-nim",
				TaskID:                "11111111-2222-3333-4444-555555555555",
				Provider:              "nim",
				CredentialAccountHome: accountHome,
				Task:                  TaskContextForEnv{IssueID: "issue-nim"},
			}, testLogger())
			if err == nil {
				t.Fatal("Prepare accepted an unusable NIM credential")
			}
		})
	}
}

func prepareNIMTestEnv(t *testing.T, accountHome, taskID string) *Environment {
	t.Helper()
	env, err := Prepare(PrepareParams{
		WorkspacesRoot:        t.TempDir(),
		WorkspaceID:           "ws-nim",
		TaskID:                taskID,
		Provider:              "nim",
		CredentialAccountHome: accountHome,
		Task:                  TaskContextForEnv{IssueID: "issue-nim"},
	}, testLogger())
	if err != nil {
		t.Fatalf("Prepare NIM: %v", err)
	}
	return env
}

func stringPtr(value string) *string { return &value }
