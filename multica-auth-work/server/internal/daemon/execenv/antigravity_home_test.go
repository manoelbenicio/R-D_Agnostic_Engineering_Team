package execenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareAntigravityHomePerAccountIsolatesTokenDir(t *testing.T) {
	accountA := filepath.Join(t.TempDir(), "accountA")
	accountB := filepath.Join(t.TempDir(), "accountB")
	writeTestCredential(t, filepath.Join(accountA, antigravityCredentialRelDir, "refresh.token"), "antigravity-account-A")
	writeTestCredential(t, filepath.Join(accountB, antigravityCredentialRelDir, "refresh.token"), "antigravity-account-B")

	homeA := filepath.Join(t.TempDir(), "home-A")
	homeB := filepath.Join(t.TempDir(), "home-B")

	if err := prepareAntigravityHome(homeA, AntigravityHomeOptions{AccountHome: accountA}, testLogger()); err != nil {
		t.Fatalf("prepare antigravity home A: %v", err)
	}
	if err := prepareAntigravityHome(homeB, AntigravityHomeOptions{AccountHome: accountB}, testLogger()); err != nil {
		t.Fatalf("prepare antigravity home B: %v", err)
	}

	tokenA := filepath.Join(homeA, antigravityCredentialRelDir, "refresh.token")
	tokenB := filepath.Join(homeB, antigravityCredentialRelDir, "refresh.token")
	assertFileContent(t, tokenA, "antigravity-account-A")
	assertFileContent(t, tokenB, "antigravity-account-B")
	assertNotSymlink(t, tokenA)
	assertNotSymlink(t, tokenB)

	if err := os.WriteFile(tokenA, []byte("antigravity-account-A-refreshed"), 0o600); err != nil {
		t.Fatalf("simulate antigravity refresh on A: %v", err)
	}
	assertFileContent(t, tokenB, "antigravity-account-B")
	assertFileContent(t, filepath.Join(accountA, antigravityCredentialRelDir, "refresh.token"), "antigravity-account-A")

	writeTestCredential(t, filepath.Join(accountA, antigravityCredentialRelDir, "refresh.token"), "antigravity-account-A-source-refresh")
	if err := prepareAntigravityHome(homeA, AntigravityHomeOptions{AccountHome: accountA}, testLogger()); err != nil {
		t.Fatalf("reuse antigravity home A: %v", err)
	}
	assertFileContent(t, tokenA, "antigravity-account-A-source-refresh")
	assertFileContent(t, tokenB, "antigravity-account-B")
}

func TestPrepareAntigravityHomeFallbackWhenNoAccount(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	if err := prepareAntigravityHome(home, AntigravityHomeOptions{}, testLogger()); err != nil {
		t.Fatalf("prepare antigravity fallback: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, antigravityCredentialRelDir)); !os.IsNotExist(err) {
		t.Fatalf("antigravity fallback created credential copy, stat err = %v", err)
	}
}

func writeTestCredential(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir credential parent: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write credential %s: %v", path, err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", path, string(got), want)
	}
}

func assertNotSymlink(t *testing.T, path string) {
	t.Helper()
	fi, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat %s: %v", path, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("%s is a symlink; per-account credentials must be copied", path)
	}
}
