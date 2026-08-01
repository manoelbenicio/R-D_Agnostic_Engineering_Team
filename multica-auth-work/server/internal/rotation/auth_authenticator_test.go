package rotation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCredentialAuthenticatorLoginWaitLogout(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "auth.json"), []byte(`{"opaque":true}`), 0o600); err != nil {
		t.Fatalf("write credential fixture: %v", err)
	}

	auth := NewCredentialAuthenticator()
	sessionID, err := auth.Login(context.Background(), Account{
		AccountID: "account-1",
		Vendor:    "codex",
		HomeDir:   home,
		ConfigDir: source,
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if sessionID == "" {
		t.Fatal("Login returned empty session id")
	}
	if _, err := os.Stat(filepath.Join(home, "auth.json")); err != nil {
		t.Fatalf("credential was not restored into home: %v", err)
	}

	ok, err := auth.WaitAuthenticated(context.Background(), sessionID, time.Second)
	if err != nil {
		t.Fatalf("WaitAuthenticated: %v", err)
	}
	if !ok {
		t.Fatal("WaitAuthenticated = false, want true")
	}

	if err := auth.Logout(context.Background(), Account{
		AccountID: "account-1",
		Vendor:    "codex",
		HomeDir:   home,
		ConfigDir: source,
	}); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "auth.json")); !os.IsNotExist(err) {
		t.Fatalf("credential still present after logout: %v", err)
	}
}
