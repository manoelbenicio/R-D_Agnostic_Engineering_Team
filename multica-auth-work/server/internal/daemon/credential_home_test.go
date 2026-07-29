package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCredentialMarker(t *testing.T, slotsRoot, slot, provider string) string {
	t.Helper()
	slotRoot := filepath.Join(slotsRoot, slot)
	home, required, _ := credentialProviderPath(slotRoot, provider)
	if err := os.MkdirAll(filepath.Dir(filepath.Join(home, required)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, required), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	return home
}

func configureCredentialResolver(t *testing.T, slotsRoot, provider, allowlist string) {
	t.Helper()
	t.Setenv(credentialSlotsRootEnv, slotsRoot)
	t.Setenv(credentialSlotAllowlistEnv(provider), allowlist)
}

func TestResolveCredentialAccountHomeUsesProviderRootAndPersistsAffinity(t *testing.T) {
	stateRoot := t.TempDir()
	slotsRoot := filepath.Join(stateRoot, "slots")
	if err := os.Mkdir(slotsRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	homeA := writeCredentialMarker(t, slotsRoot, "slot-140", "kiro")
	homeB := writeCredentialMarker(t, slotsRoot, "slot-149", "kiro")
	configureCredentialResolver(t, slotsRoot, "kiro", "140,slot-149")

	first, err := resolveCredentialAccountHome("agent-a", "kiro")
	if err != nil {
		t.Fatal(err)
	}
	second, err := resolveCredentialAccountHome("agent-a", "kiro")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("affinity changed: first=%q second=%q", first, second)
	}
	if first != homeA && first != homeB {
		t.Fatalf("resolved home %q is not a provider root", first)
	}
	info, err := os.Stat(filepath.Join(stateRoot, assignmentFileName))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("assignment mode = %o, want 600", info.Mode().Perm())
	}
}

func TestResolveCredentialAccountHomeRejectsMissingKiroCredential(t *testing.T) {
	stateRoot := t.TempDir()
	slotsRoot := filepath.Join(stateRoot, "slots")
	if err := os.MkdirAll(filepath.Join(slotsRoot, "slot-145", "xdg-data", "kiro-cli"), 0o700); err != nil {
		t.Fatal(err)
	}
	configureCredentialResolver(t, slotsRoot, "kiro", "145")

	_, err := resolveCredentialAccountHome("agent-a", "kiro")
	if err == nil || !strings.Contains(err.Error(), "required kiro artifact missing") {
		t.Fatalf("error = %v, want explicit missing Kiro credential", err)
	}
}

func TestResolveCredentialAccountHomeRejectsSymlinkSlot(t *testing.T) {
	stateRoot := t.TempDir()
	slotsRoot := filepath.Join(stateRoot, "slots")
	outside := filepath.Join(stateRoot, "outside")
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(slotsRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(slotsRoot, "slot-140")); err != nil {
		t.Fatal(err)
	}
	configureCredentialResolver(t, slotsRoot, "kiro", "140")

	if _, err := resolveCredentialAccountHome("agent-a", "kiro"); err == nil {
		t.Fatal("expected symlink slot to fail closed")
	}
}

func TestResolveCredentialAccountHomeFailsWhenPersistedSlotLeavesAllowlist(t *testing.T) {
	stateRoot := t.TempDir()
	slotsRoot := filepath.Join(stateRoot, "slots")
	if err := os.Mkdir(slotsRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	writeCredentialMarker(t, slotsRoot, "slot-140", "kiro")
	writeCredentialMarker(t, slotsRoot, "slot-149", "kiro")
	configureCredentialResolver(t, slotsRoot, "kiro", "140")
	if _, err := resolveCredentialAccountHome("agent-a", "kiro"); err != nil {
		t.Fatal(err)
	}
	t.Setenv(credentialSlotAllowlistEnv("kiro"), "149")
	if _, err := resolveCredentialAccountHome("agent-a", "kiro"); err == nil ||
		!strings.Contains(err.Error(), "persisted slot") {
		t.Fatalf("error = %v, want persisted-slot failure", err)
	}
}

func TestResolveCredentialAccountHomeIgnoresOutOfScopeProvider(t *testing.T) {
	if home, err := resolveCredentialAccountHome("agent-a", "claude"); err != nil || home != "" {
		t.Fatalf("home=%q err=%v", home, err)
	}
}

func TestResolveCredentialModelDiscoveryHomesUsesOnlyAllowlist(t *testing.T) {
	stateRoot := t.TempDir()
	slotsRoot := filepath.Join(stateRoot, "slots")
	if err := os.Mkdir(slotsRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	home141 := writeCredentialMarker(t, slotsRoot, "slot-141", "antigravity")
	home150 := writeCredentialMarker(t, slotsRoot, "slot-150", "antigravity")
	if err := os.MkdirAll(filepath.Join(slotsRoot, "slot-139", "home", ".gemini"), 0o700); err != nil {
		t.Fatal(err)
	}
	configureCredentialResolver(t, slotsRoot, "antigravity", "141,150")

	homes, err := resolveCredentialModelDiscoveryHomes("antigravity")
	if err != nil {
		t.Fatal(err)
	}
	if len(homes) != 2 || homes[0] != home141 || homes[1] != home150 {
		t.Fatalf("discovery homes = %q, want [%q %q]", homes, home141, home150)
	}
}
