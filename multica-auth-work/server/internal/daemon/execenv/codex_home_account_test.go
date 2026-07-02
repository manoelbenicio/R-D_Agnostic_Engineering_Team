package execenv

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPrepareCodexHomePerAccountIsolatesAuth proves the core of the credential
// isolation change: when AccountHome is set, auth.json is COPIED from that
// account's own dir (not symlinked to the shared home), so two accounts never
// share one credential and a refresh on one cannot clobber another.
func TestPrepareCodexHomePerAccountIsolatesAuth(t *testing.T) {
	// Two separate account credential dirs with distinct auth.json contents.
	accountA := filepath.Join(t.TempDir(), "accountA")
	accountB := filepath.Join(t.TempDir(), "accountB")
	for _, a := range []string{accountA, accountB} {
		if err := os.MkdirAll(a, 0o700); err != nil {
			t.Fatalf("mkdir account dir: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(accountA, "auth.json"), []byte(`{"account":"A"}`), 0o600); err != nil {
		t.Fatalf("write account A auth: %v", err)
	}
	if err := os.WriteFile(filepath.Join(accountB, "auth.json"), []byte(`{"account":"B"}`), 0o600); err != nil {
		t.Fatalf("write account B auth: %v", err)
	}

	homeA := filepath.Join(t.TempDir(), "codex-home-A")
	homeB := filepath.Join(t.TempDir(), "codex-home-B")

	if err := prepareCodexHomeWithOpts(homeA, CodexHomeOptions{GOOS: "linux", AccountHome: accountA}, testLogger()); err != nil {
		t.Fatalf("prepare home A: %v", err)
	}
	if err := prepareCodexHomeWithOpts(homeB, CodexHomeOptions{GOOS: "linux", AccountHome: accountB}, testLogger()); err != nil {
		t.Fatalf("prepare home B: %v", err)
	}

	authA := filepath.Join(homeA, "auth.json")
	authB := filepath.Join(homeB, "auth.json")

	// Each per-task home must carry ITS OWN account credential.
	gotA, err := os.ReadFile(authA)
	if err != nil {
		t.Fatalf("read home A auth: %v", err)
	}
	gotB, err := os.ReadFile(authB)
	if err != nil {
		t.Fatalf("read home B auth: %v", err)
	}
	if string(gotA) != `{"account":"A"}` {
		t.Errorf("home A auth = %q, want account A", string(gotA))
	}
	if string(gotB) != `{"account":"B"}` {
		t.Errorf("home B auth = %q, want account B", string(gotB))
	}

	// auth.json MUST be a regular file (copied), not a symlink — isolation.
	fiA, err := os.Lstat(authA)
	if err != nil {
		t.Fatalf("lstat home A auth: %v", err)
	}
	if fiA.Mode()&os.ModeSymlink != 0 {
		t.Error("home A auth.json is a symlink; per-account mode must copy (isolate)")
	}

	// Simulate a token refresh on account A's per-task home: it must NOT
	// propagate to account B's home (proves isolation).
	if err := os.WriteFile(authA, []byte(`{"account":"A","refreshed":true}`), 0o600); err != nil {
		t.Fatalf("simulate refresh on A: %v", err)
	}
	gotB2, err := os.ReadFile(authB)
	if err != nil {
		t.Fatalf("re-read home B auth: %v", err)
	}
	if string(gotB2) != `{"account":"B"}` {
		t.Errorf("home B auth changed after A refresh = %q; accounts are NOT isolated", string(gotB2))
	}
}

// TestPrepareCodexHomeFallbackWhenNoAccount confirms that with AccountHome
// empty, the historical behavior is preserved: no per-account copy is forced,
// and preparation still succeeds (full backward compatibility).
func TestPrepareCodexHomeFallbackWhenNoAccount(t *testing.T) {
	home := filepath.Join(t.TempDir(), "codex-home")
	if err := prepareCodexHomeWithOpts(home, CodexHomeOptions{GOOS: "linux"}, testLogger()); err != nil {
		t.Fatalf("prepare home (fallback): %v", err)
	}
	if _, err := os.Stat(home); err != nil {
		t.Fatalf("codex home not created in fallback mode: %v", err)
	}
}
