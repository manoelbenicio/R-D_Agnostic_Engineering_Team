package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateCredentialAccountHome(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "slot-1", "home")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := validateCredentialAccountHome("agy", home, true)
	if err != nil || got != home {
		t.Fatalf("valid approved home: got=%q err=%v", got, err)
	}

	for name, path := range map[string]string{
		"empty":    "",
		"relative": "slot-1/home",
		"missing":  filepath.Join(root, "missing"),
		"root":     "/",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := validateCredentialAccountHome("kiro", path, true); !errors.Is(err, errCredentialAccountHomeUnavailable) {
				t.Fatalf("error=%v, want fail-closed sentinel", err)
			}
		})
	}

	link := filepath.Join(root, "linked-home")
	if err := os.Symlink(home, link); err != nil {
		t.Fatal(err)
	}
	if _, err := validateCredentialAccountHome("codex", link, true); !errors.Is(err, errCredentialAccountHomeUnavailable) {
		t.Fatalf("symlink error=%v, want fail-closed sentinel", err)
	}
}

func TestValidateCredentialAccountHomeLegacyPayloadDoesNotEnforce(t *testing.T) {
	got, err := validateCredentialAccountHome("kiro", "", false)
	if err != nil || got != "" {
		t.Fatalf("legacy payload got=%q err=%v", got, err)
	}
}

func TestValidateCredentialAccountHomeRejectsRequiredUnknownProvider(t *testing.T) {
	if _, err := validateCredentialAccountHome("openclaw", "/tmp/account", true); !errors.Is(err, errCredentialAccountHomeUnavailable) {
		t.Fatalf("unknown required provider error=%v", err)
	}
}
