package execenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareAntigravityHomePerAccountCopiesOnlyOAuthToken(t *testing.T) {
	accountA := filepath.Join(t.TempDir(), "accountA")
	accountB := filepath.Join(t.TempDir(), "accountB")
	sourceTokenA := filepath.Join(accountA, antigravityCredentialRelPath)
	sourceTokenB := filepath.Join(accountB, antigravityCredentialRelPath)
	writeTestCredential(t, sourceTokenA, "antigravity-account-A")
	writeTestCredential(t, sourceTokenB, "antigravity-account-B")
	if err := os.Chmod(sourceTokenA, 0o644); err != nil {
		t.Fatalf("chmod source token A: %v", err)
	}

	sourceDirA := filepath.Join(accountA, antigravityCredentialRelDir)
	writeTestCredential(t, filepath.Join(sourceDirA, "state.db"), "must-not-copy")
	writeTestCredential(t, filepath.Join(sourceDirA, "cache", "models.json"), "must-not-copy")
	if err := os.Symlink(filepath.Join(sourceDirA, "state.db"), filepath.Join(sourceDirA, "cli.log")); err != nil {
		t.Fatalf("create cli.log symlink: %v", err)
	}

	homeA := filepath.Join(t.TempDir(), "home-A")
	homeB := filepath.Join(t.TempDir(), "home-B")
	for _, dir := range []string{
		homeA,
		filepath.Join(homeA, ".gemini"),
		filepath.Join(homeA, antigravityCredentialRelDir),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("pre-create permissive destination directory %s: %v", dir, err)
		}
		if err := os.Chmod(dir, 0o755); err != nil {
			t.Fatalf("chmod permissive destination directory %s: %v", dir, err)
		}
	}

	if err := prepareAntigravityHome(homeA, AntigravityHomeOptions{AccountHome: accountA}, testLogger()); err != nil {
		t.Fatalf("prepare antigravity home A: %v", err)
	}
	if err := prepareAntigravityHome(homeB, AntigravityHomeOptions{AccountHome: accountB}, testLogger()); err != nil {
		t.Fatalf("prepare antigravity home B: %v", err)
	}

	tokenA := filepath.Join(homeA, antigravityCredentialRelPath)
	tokenB := filepath.Join(homeB, antigravityCredentialRelPath)
	assertFileContent(t, tokenA, "antigravity-account-A")
	assertFileContent(t, tokenB, "antigravity-account-B")
	assertNotSymlink(t, tokenA)
	assertNotSymlink(t, tokenB)
	assertFileMode(t, tokenA, 0o600)
	assertFileMode(t, tokenB, 0o600)
	assertFileMode(t, homeA, 0o700)
	assertFileMode(t, filepath.Join(homeA, ".gemini"), 0o700)
	assertFileMode(t, filepath.Join(homeA, antigravityCredentialRelDir), 0o700)
	assertPathMissing(t, filepath.Join(homeA, antigravityCredentialRelDir, "state.db"))
	assertPathMissing(t, filepath.Join(homeA, antigravityCredentialRelDir, "cache"))
	assertPathMissing(t, filepath.Join(homeA, antigravityCredentialRelDir, "cli.log"))

	if err := os.WriteFile(tokenA, []byte("antigravity-account-A-refreshed"), 0o600); err != nil {
		t.Fatalf("simulate antigravity refresh on A: %v", err)
	}
	assertFileContent(t, tokenB, "antigravity-account-B")
	assertFileContent(t, sourceTokenA, "antigravity-account-A")

	writeTestCredential(t, sourceTokenA, "antigravity-account-A-source-refresh")
	if err := prepareAntigravityHome(homeA, AntigravityHomeOptions{AccountHome: accountA}, testLogger()); err != nil {
		t.Fatalf("reuse antigravity home A: %v", err)
	}
	assertFileContent(t, tokenA, "antigravity-account-A-source-refresh")
	assertFileContent(t, tokenB, "antigravity-account-B")
}

func TestPrepareAntigravityHomeRejectsEmptyDestination(t *testing.T) {
	accountHome := t.TempDir()
	writeTestCredential(t, filepath.Join(accountHome, antigravityCredentialRelPath), "token")
	if err := prepareAntigravityHome("", AntigravityHomeOptions{AccountHome: accountHome}, testLogger()); err == nil {
		t.Fatal("prepare antigravity home succeeded with empty destination")
	}
}

func TestPrepareAntigravityHomeReplacesDestinationSymlink(t *testing.T) {
	accountHome := t.TempDir()
	sourceToken := filepath.Join(accountHome, antigravityCredentialRelPath)
	writeTestCredential(t, sourceToken, "account-token")

	home := t.TempDir()
	tokenPath := filepath.Join(home, antigravityCredentialRelPath)
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0o700); err != nil {
		t.Fatalf("mkdir destination token parent: %v", err)
	}
	target := filepath.Join(t.TempDir(), "target")
	writeTestCredential(t, target, "must-not-change")
	if err := os.Symlink(target, tokenPath); err != nil {
		t.Fatalf("create destination symlink: %v", err)
	}

	if err := prepareAntigravityHome(home, AntigravityHomeOptions{AccountHome: accountHome}, testLogger()); err != nil {
		t.Fatalf("prepare antigravity home: %v", err)
	}
	assertNotSymlink(t, tokenPath)
	assertFileMode(t, tokenPath, 0o600)
	assertFileContent(t, tokenPath, "account-token")
	assertFileContent(t, target, "must-not-change")
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

func TestReuseAntigravityFailsClosedOnInvalidSource(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "work")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	accountHome := t.TempDir()
	if got := Reuse(ReuseParams{
		WorkDir:               workDir,
		Provider:              "antigravity",
		CredentialAccountHome: accountHome,
	}, testLogger()); got != nil {
		t.Fatal("Reuse returned an environment with a missing AGY token")
	}
}

func TestPrepareAntigravityHomeRejectsInvalidOAuthToken(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, tokenPath string)
	}{
		{
			name: "missing",
			setup: func(t *testing.T, tokenPath string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(tokenPath), 0o700); err != nil {
					t.Fatalf("mkdir token parent: %v", err)
				}
			},
		},
		{
			name: "symlink",
			setup: func(t *testing.T, tokenPath string) {
				t.Helper()
				target := filepath.Join(t.TempDir(), "token")
				writeTestCredential(t, target, "must-not-follow")
				if err := os.MkdirAll(filepath.Dir(tokenPath), 0o700); err != nil {
					t.Fatalf("mkdir token parent: %v", err)
				}
				if err := os.Symlink(target, tokenPath); err != nil {
					t.Fatalf("create token symlink: %v", err)
				}
			},
		},
		{
			name: "directory",
			setup: func(t *testing.T, tokenPath string) {
				t.Helper()
				if err := os.MkdirAll(tokenPath, 0o700); err != nil {
					t.Fatalf("mkdir token path: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accountHome := t.TempDir()
			tokenPath := filepath.Join(accountHome, antigravityCredentialRelPath)
			tt.setup(t, tokenPath)

			home := filepath.Join(t.TempDir(), "home")
			if err := prepareAntigravityHome(home, AntigravityHomeOptions{AccountHome: accountHome}, testLogger()); err == nil {
				t.Fatal("prepare antigravity home succeeded with invalid token")
			}
			assertPathMissing(t, filepath.Join(home, antigravityCredentialRelPath))
		})
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

func assertFileMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %o, want %o", path, got, want)
	}
}

func assertPathMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("%s exists or returned unexpected error: %v", path, err)
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
