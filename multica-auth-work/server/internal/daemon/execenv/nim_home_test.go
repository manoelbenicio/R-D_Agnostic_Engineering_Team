package execenv

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPrepareNimHomePerAccountIsolatesCredentialDir verifies that two accounts'
// NIM credential dirs are isolated into separate per-task dirs, copied (not
// symlinked), and that a source-side refresh re-syncs only the re-prepared task.
func TestPrepareNimHomePerAccountIsolatesCredentialDir(t *testing.T) {
	accountA := filepath.Join(t.TempDir(), "accountA")
	accountB := filepath.Join(t.TempDir(), "accountB")
	writeTestCredential(t, filepath.Join(accountA, nimCredentialRelDir, "credentials.json"), `{"api_key":"nvapi-account-A"}`)
	writeTestCredential(t, filepath.Join(accountB, nimCredentialRelDir, "credentials.json"), `{"api_key":"nvapi-account-B"}`)

	homeA := filepath.Join(t.TempDir(), "home-A")
	homeB := filepath.Join(t.TempDir(), "home-B")

	if err := prepareNimHome(homeA, NimHomeOptions{AccountHome: accountA}, testLogger()); err != nil {
		t.Fatalf("prepare nim home A: %v", err)
	}
	if err := prepareNimHome(homeB, NimHomeOptions{AccountHome: accountB}, testLogger()); err != nil {
		t.Fatalf("prepare nim home B: %v", err)
	}

	credA := filepath.Join(homeA, "credentials.json")
	credB := filepath.Join(homeB, "credentials.json")
	assertFileContent(t, credA, `{"api_key":"nvapi-account-A"}`)
	assertFileContent(t, credB, `{"api_key":"nvapi-account-B"}`)
	assertNotSymlink(t, credA)
	assertNotSymlink(t, credB)

	// A refresh inside task A's isolated dir must NOT leak into B's dir or the
	// shared account-A source.
	if err := os.WriteFile(credA, []byte(`{"api_key":"nvapi-A-refreshed-in-task"}`), 0o600); err != nil {
		t.Fatalf("simulate nim refresh on A: %v", err)
	}
	assertFileContent(t, credB, `{"api_key":"nvapi-account-B"}`)
	assertFileContent(t, filepath.Join(accountA, nimCredentialRelDir, "credentials.json"), `{"api_key":"nvapi-account-A"}`)

	// A source-side refresh re-syncs into A's isolated dir on re-prepare, without
	// touching B.
	writeTestCredential(t, filepath.Join(accountA, nimCredentialRelDir, "credentials.json"), `{"api_key":"nvapi-A-source-refresh"}`)
	if err := prepareNimHome(homeA, NimHomeOptions{AccountHome: accountA}, testLogger()); err != nil {
		t.Fatalf("reuse nim home A: %v", err)
	}
	assertFileContent(t, credA, `{"api_key":"nvapi-A-source-refresh"}`)
	assertFileContent(t, credB, `{"api_key":"nvapi-account-B"}`)
}

// TestPrepareNimHomeFallbackWhenNoAccount confirms an empty AccountHome is a
// no-op (shared/global behavior) and creates no credential copy.
func TestPrepareNimHomeFallbackWhenNoAccount(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	if err := prepareNimHome(home, NimHomeOptions{}, testLogger()); err != nil {
		t.Fatalf("prepare nim fallback: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
		t.Fatalf("nim fallback created credential copy, stat err = %v", err)
	}
}

// TestPrepareNimHomeResolvesBareNimSubdir confirms the resolver accepts a bare
// nim/ subdir (no leading dot) as the credential source, mirroring cline's
// candidate layout.
func TestPrepareNimHomeResolvesBareNimSubdir(t *testing.T) {
	accountHome := t.TempDir()
	writeTestCredential(t, filepath.Join(accountHome, "nim", "credentials.json"), `{"api_key":"nvapi-bare"}`)

	home := filepath.Join(t.TempDir(), "home-bare")
	if err := prepareNimHome(home, NimHomeOptions{AccountHome: accountHome}, testLogger()); err != nil {
		t.Fatalf("prepare nim bare subdir: %v", err)
	}
	assertFileContent(t, filepath.Join(home, "credentials.json"), `{"api_key":"nvapi-bare"}`)
}

// TestPrepareNimHomeEmptyWhenNoSource confirms that an account home without any
// NIM credential marker yields an empty isolated dir (fail-closed at runtime
// rather than silently falling back to a shared/global key).
func TestPrepareNimHomeEmptyWhenNoSource(t *testing.T) {
	accountHome := t.TempDir() // no .nim/, no nim/, no markers
	home := filepath.Join(t.TempDir(), "home-empty")

	if err := prepareNimHome(home, NimHomeOptions{AccountHome: accountHome}, testLogger()); err != nil {
		t.Fatalf("prepare nim empty source: %v", err)
	}
	info, err := os.Stat(home)
	if err != nil {
		t.Fatalf("empty nim home not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("nim home %s is not a directory", home)
	}
	if mode := info.Mode().Perm(); mode != 0o700 {
		t.Fatalf("nim home mode = %o, want 0700", mode)
	}
	if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
		t.Fatalf("empty nim home should not contain a credential copy, stat err = %v", err)
	}
}

// TestPrepareNimHomeRejectsEmptyDataDir confirms a missing nimDataDir is an
// error (guard against a bad caller wiring an empty NIM_HOME).
func TestPrepareNimHomeRejectsEmptyDataDir(t *testing.T) {
	accountHome := t.TempDir()
	writeTestCredential(t, filepath.Join(accountHome, nimCredentialRelDir, "credentials.json"), `{"api_key":"nvapi-x"}`)

	if err := prepareNimHome("", NimHomeOptions{AccountHome: accountHome}, testLogger()); err == nil {
		t.Fatal("prepareNimHome with empty data dir returned nil, want error")
	}
}
