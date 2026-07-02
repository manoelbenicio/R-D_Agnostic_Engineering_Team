package execenv

import "testing"

func TestVendorCredentialFallbackDoesNotSetIsolatedHomes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider string
	}{
		{name: "kiro", provider: "kiro"},
		{name: "antigravity", provider: "antigravity"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env, err := Prepare(PrepareParams{
				WorkspacesRoot: t.TempDir(),
				WorkspaceID:    "ws-test",
				TaskID:         "11111111-2222-3333-4444-555555555555",
				AgentName:      "Vendor Agent",
				Provider:       tt.provider,
				Task:           TaskContextForEnv{IssueID: "issue-1"},
			}, testLogger())
			if err != nil {
				t.Fatalf("Prepare(%s): %v", tt.provider, err)
			}
			defer env.Cleanup(true)

			assertNoIsolatedHomes(t, "Prepare", env)

			reused := Reuse(ReuseParams{
				WorkDir:  env.WorkDir,
				Provider: tt.provider,
				Task:     TaskContextForEnv{IssueID: "issue-1"},
			}, testLogger())
			if reused == nil {
				t.Fatal("Reuse returned nil")
			}

			assertNoIsolatedHomes(t, "Reuse", reused)
		})
	}
}

func assertNoIsolatedHomes(t *testing.T, phase string, env *Environment) {
	t.Helper()

	if env.CodexHome != "" {
		t.Fatalf("%s set CodexHome = %q; non-codex vendor fallback must not inject CODEX_HOME", phase, env.CodexHome)
	}
	if env.KiroDataHome != "" {
		t.Fatalf("%s set KiroDataHome = %q with empty CredentialAccountHome; daemon would inject XDG_DATA_HOME", phase, env.KiroDataHome)
	}
	if env.AntigravityHome != "" {
		t.Fatalf("%s set AntigravityHome = %q with empty CredentialAccountHome; daemon would inject HOME", phase, env.AntigravityHome)
	}
}
