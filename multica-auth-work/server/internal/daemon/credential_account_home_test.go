package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func createControlledCredentialHome(t *testing.T, provider string) (string, string) {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "slots")
	home := filepath.Join(root, "slot-1", "home")
	for _, dir := range []string{root, filepath.Dir(home), home} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	switch provider {
	case "codex":
		if err := os.WriteFile(filepath.Join(home, "auth.json"), []byte("synthetic"), 0o600); err != nil {
			t.Fatal(err)
		}
	case "kiro":
		dir := filepath.Join(home, "kiro-cli")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "data.sqlite3"), []byte("synthetic"), 0o600); err != nil {
			t.Fatal(err)
		}
	case "antigravity":
		dir := filepath.Join(home, ".gemini", "antigravity-cli")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "synthetic-token"), []byte("synthetic"), 0o600); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unsupported synthetic provider %q", provider)
	}
	t.Setenv(credentialSlotsRootEnv, root)
	return root, home
}

func TestValidateCredentialAccountHomeProviderLayouts(t *testing.T) {
	for _, provider := range []string{"codex", "kiro", "antigravity", "agy"} {
		t.Run(provider, func(t *testing.T) {
			layoutProvider := provider
			if provider == "agy" {
				layoutProvider = "antigravity"
			}
			_, home := createControlledCredentialHome(t, layoutProvider)
			got, err := validateCredentialAccountHome(provider, home, true)
			if err != nil || got != home {
				t.Fatalf("valid approved home: got=%q err=%v", got, err)
			}
		})
	}
}

func TestValidateCredentialAccountHomeFailsClosed(t *testing.T) {
	_, home := createControlledCredentialHome(t, "kiro")
	for name, path := range map[string]string{
		"empty":    "",
		"relative": "slot-1/home",
		"missing":  filepath.Join(filepath.Dir(home), "missing"),
		"root":     "/",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := validateCredentialAccountHome("kiro", path, true); !errors.Is(err, errCredentialAccountHomeUnavailable) {
				t.Fatalf("error=%v, want fail-closed sentinel", err)
			}
		})
	}
}

func TestValidateCredentialAccountHomeRejectsWrongRoot(t *testing.T) {
	root, _ := createControlledCredentialHome(t, "codex")
	otherHome := filepath.Join(t.TempDir(), "slot-2", "home")
	if err := os.MkdirAll(otherHome, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherHome, "auth.json"), []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(credentialSlotsRootEnv, root)
	if _, err := validateCredentialAccountHome("codex", otherHome, true); !errors.Is(err, errCredentialAccountHomeUnavailable) {
		t.Fatalf("wrong-root error=%v", err)
	}
}

func TestValidateCredentialAccountHomeRejectsSymlink(t *testing.T) {
	root, home := createControlledCredentialHome(t, "codex")
	link := filepath.Join(root, "slot-2", "home")
	if err := os.MkdirAll(filepath.Dir(link), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(home, link); err != nil {
		t.Fatal(err)
	}
	if _, err := validateCredentialAccountHome("codex", link, true); !errors.Is(err, errCredentialAccountHomeUnavailable) {
		t.Fatalf("symlink error=%v, want fail-closed sentinel", err)
	}
}

func TestValidateCredentialAccountHomeRejectsWorldReadableMode(t *testing.T) {
	_, home := createControlledCredentialHome(t, "kiro")
	if err := os.Chmod(filepath.Join(home, "kiro-cli", "data.sqlite3"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := validateCredentialAccountHome("kiro", home, true); !errors.Is(err, errCredentialAccountHomeUnavailable) {
		t.Fatalf("world-readable error=%v", err)
	}
}

func TestValidateCredentialAccountHomeLegacyPayloadDoesNotEnforce(t *testing.T) {
	t.Setenv(credentialAssignmentEnforcementEnv, "")
	got, err := validateCredentialAccountHome("kiro", "", false)
	if err != nil || got != "" {
		t.Fatalf("legacy payload got=%q err=%v", got, err)
	}
}

func TestValidateCredentialAccountHomeRolloutGateFailsClosed(t *testing.T) {
	t.Setenv(credentialAssignmentEnforcementEnv, "true")
	for _, provider := range []string{"kiro", "codex", "antigravity", "agy"} {
		if _, err := validateCredentialAccountHome(provider, "", false); !errors.Is(err, errCredentialAccountHomeUnavailable) {
			t.Fatalf("%s with enforcement enabled: error=%v", provider, err)
		}
	}
}

func TestValidateCredentialAccountHomeRolloutGateDoesNotEnrollOtherProviders(t *testing.T) {
	t.Setenv(credentialAssignmentEnforcementEnv, "true")
	got, err := validateCredentialAccountHome("openclaw", "", false)
	if err != nil || got != "" {
		t.Fatalf("unenrolled provider got=%q err=%v", got, err)
	}
}

func TestValidateCredentialAccountHomeRejectsRequiredUnknownProvider(t *testing.T) {
	_, home := createControlledCredentialHome(t, "codex")
	if _, err := validateCredentialAccountHome("openclaw", home, true); !errors.Is(err, errCredentialAccountHomeUnavailable) {
		t.Fatalf("unknown required provider error=%v", err)
	}
}
