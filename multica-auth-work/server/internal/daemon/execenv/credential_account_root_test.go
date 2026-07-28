package execenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareCredentialAccountRefusesExistingTaskRoot(t *testing.T) {
	workspacesRoot := t.TempDir()
	workspaceID := "workspace-account-root"
	taskID := "12345678-account-task"
	envRoot := PredictRootDir(workspacesRoot, workspaceID, taskID)
	if err := os.MkdirAll(envRoot, 0o700); err != nil {
		t.Fatalf("create existing task root: %v", err)
	}
	sentinel := filepath.Join(envRoot, "preserve-me")
	if err := os.WriteFile(sentinel, []byte("synthetic-sentinel"), 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	_, err := Prepare(PrepareParams{
		WorkspacesRoot:        workspacesRoot,
		WorkspaceID:           workspaceID,
		TaskID:                taskID,
		Provider:              "codex",
		CredentialAccountHome: filepath.Join(t.TempDir(), "account-home"),
	}, testLogger())
	if err == nil || !strings.Contains(err.Error(), "refusing to inspect or rewrite") {
		t.Fatalf("Prepare error = %v, want existing-root refusal", err)
	}

	got, readErr := os.ReadFile(sentinel)
	if readErr != nil {
		t.Fatalf("sentinel was removed or replaced: %v", readErr)
	}
	if string(got) != "synthetic-sentinel" {
		t.Fatalf("sentinel changed unexpectedly")
	}
}
