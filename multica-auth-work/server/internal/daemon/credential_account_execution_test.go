package daemon

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/execenv"
)

func TestAgyAliasCanonicalizesBeforeCredentialExecenv(t *testing.T) {
	_, accountHome := createControlledCredentialHome(t, "antigravity")
	d := &Daemon{
		cfg: Config{Agents: map[string]AgentEntry{
			"antigravity": {Path: "/synthetic/agy"},
		}},
		logger: slog.Default(),
	}

	entry, provider, err := d.resolveTaskAgentEntry(Task{}, "agy")
	if err != nil {
		t.Fatalf("resolve agy alias: %v", err)
	}
	if provider != "antigravity" || entry.Path != "/synthetic/agy" {
		t.Fatalf("resolved provider=%q entry=%+v", provider, entry)
	}
	validatedHome, err := validateCredentialAccountHome(provider, accountHome, true)
	if err != nil {
		t.Fatalf("validate canonical account home: %v", err)
	}
	env, err := execenv.Prepare(execenv.PrepareParams{
		WorkspacesRoot:        t.TempDir(),
		WorkspaceID:           "synthetic-workspace",
		TaskID:                "synthetic-agy-alias",
		AgentName:             "Synthetic Antigravity",
		Provider:              provider,
		CredentialAccountHome: validatedHome,
		CredentiallessGateway: false,
		Task:                  execenv.TaskContextForEnv{IssueID: "synthetic-issue"},
	}, slog.Default())
	if err != nil {
		t.Fatalf("prepare canonical Antigravity execenv: %v", err)
	}
	if env.AntigravityHome == "" {
		t.Fatal("agy alias did not activate Antigravity HOME isolation")
	}
	copied := filepath.Join(env.AntigravityHome, ".gemini", "antigravity-cli", "synthetic-token")
	info, err := os.Lstat(copied)
	if err != nil {
		t.Fatalf("canonical Antigravity credential layout not copied: %v", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("copied synthetic credential metadata is not private: mode=%v", info.Mode())
	}
}
