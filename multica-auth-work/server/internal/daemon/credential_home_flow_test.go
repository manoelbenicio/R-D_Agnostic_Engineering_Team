package daemon

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/execenv"
)

// currentAntigravitySlots mirrors the operator allowlist this daemon build is
// cut for. Kept as a literal so a future allowlist change has to be made
// deliberately in both the environment and this test.
const currentAntigravitySlots = "162,163,168,169"

func flowTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// seedAntigravitySlot builds a physical slot whose credential directory holds
// the OAuth token plus exactly the volatile siblings that used to break the
// recursive directory copy: a cli.log symlink pointing at a sibling file, a
// state database, and a cache subtree.
func seedAntigravitySlot(t *testing.T, slotsRoot, slot, token string) string {
	t.Helper()
	credentialDir := filepath.Join(slotsRoot, slot, "home", ".gemini", "antigravity-cli")
	if err := os.MkdirAll(filepath.Join(credentialDir, "cache"), 0o700); err != nil {
		t.Fatalf("create slot credential dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(credentialDir, "antigravity-oauth-token"), []byte(token), 0o600); err != nil {
		t.Fatalf("write slot token: %v", err)
	}
	if err := os.WriteFile(filepath.Join(credentialDir, "state.db"), []byte("volatile"), 0o600); err != nil {
		t.Fatalf("write slot state db: %v", err)
	}
	if err := os.WriteFile(filepath.Join(credentialDir, "cache", "models.json"), []byte("volatile"), 0o600); err != nil {
		t.Fatalf("write slot cache entry: %v", err)
	}
	if err := os.Symlink(filepath.Join(credentialDir, "state.db"), filepath.Join(credentialDir, "cli.log")); err != nil {
		t.Fatalf("create slot cli.log symlink: %v", err)
	}
	return filepath.Join(slotsRoot, slot, "home")
}

// TestAntigravityTaskFlowResolvesSlotHomeAndCopiesOnlyToken is the end-to-end
// proof for the wiring: the frozen agent identity plus provider resolve to a
// physical allowlisted slot home (non-empty), and feeding that home through
// execenv.Prepare produces a task HOME containing only the OAuth token. The
// cli.log symlink that used to abort the recursive directory copy is present
// in the source and must simply be ignored.
func TestAntigravityTaskFlowResolvesSlotHomeAndCopiesOnlyToken(t *testing.T) {
	stateRoot := t.TempDir()
	slotsRoot := filepath.Join(stateRoot, "slots")
	if err := os.Mkdir(slotsRoot, 0o700); err != nil {
		t.Fatalf("create slots root: %v", err)
	}
	seeded := map[string]string{}
	for _, slot := range strings.Split(currentAntigravitySlots, ",") {
		name := "slot-" + slot
		seeded[seedAntigravitySlot(t, slotsRoot, name, "token-"+slot)] = name
	}
	t.Setenv(credentialSlotsRootEnv, slotsRoot)
	t.Setenv(credentialSlotAllowlistEnv("antigravity"), currentAntigravitySlots)

	// Step 1: the frozen task identity binds to a physical allowlisted slot.
	// "agy" is the provider alias the daemon claims tasks with.
	accountHome, err := resolveCredentialAccountHome("agent-orq66", "agy")
	if err != nil {
		t.Fatalf("resolve credential account home: %v", err)
	}
	if accountHome == "" {
		t.Fatal("credential account home is empty; the task would run against the shared global HOME")
	}
	if _, ok := seeded[accountHome]; !ok {
		t.Fatalf("resolved home %q is not one of the allowlisted slot homes", accountHome)
	}
	if !strings.HasPrefix(accountHome, slotsRoot+string(filepath.Separator)) {
		t.Fatalf("resolved home %q escapes the slots root %q", accountHome, slotsRoot)
	}

	// Step 2: execenv builds the task HOME from that account home.
	env, err := execenv.Prepare(execenv.PrepareParams{
		WorkspacesRoot:        filepath.Join(stateRoot, "workspaces"),
		WorkspaceID:           "ws-orq66",
		TaskID:                "task-orq66",
		AgentName:             "opus48-a",
		Provider:              "antigravity",
		CredentialAccountHome: accountHome,
	}, flowTestLogger())
	if err != nil {
		t.Fatalf("prepare execution environment: %v", err)
	}
	if env.AntigravityHome == "" {
		t.Fatal("execenv did not create an isolated antigravity HOME")
	}

	// Step 3: only the OAuth token crossed into the task HOME.
	taskCredentialDir := filepath.Join(env.AntigravityHome, ".gemini", "antigravity-cli")
	entries, err := os.ReadDir(taskCredentialDir)
	if err != nil {
		t.Fatalf("read task credential dir: %v", err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if len(names) != 1 || names[0] != "antigravity-oauth-token" {
		t.Fatalf("task credential dir contains %v, want only [antigravity-oauth-token]", names)
	}

	tokenPath := filepath.Join(taskCredentialDir, "antigravity-oauth-token")
	info, err := os.Lstat(tokenPath)
	if err != nil {
		t.Fatalf("lstat task token: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		t.Fatalf("task token mode = %v, want a physical regular file", info.Mode())
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("task token perm = %o, want 600", perm)
	}
	got, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("read task token: %v", err)
	}
	if want := "token-" + strings.TrimPrefix(seeded[accountHome], "slot-"); string(got) != want {
		t.Fatalf("task token = %q, want %q", string(got), want)
	}
	for _, dir := range []string{
		env.AntigravityHome,
		filepath.Join(env.AntigravityHome, ".gemini"),
		taskCredentialDir,
	} {
		dirInfo, statErr := os.Stat(dir)
		if statErr != nil {
			t.Fatalf("stat %s: %v", dir, statErr)
		}
		if perm := dirInfo.Mode().Perm(); perm != 0o700 {
			t.Fatalf("%s perm = %o, want 700", dir, perm)
		}
	}
}

// TestAntigravityTaskFlowFailsClosedWithoutSlotsRoot proves the wiring aborts
// the task instead of degrading to an empty account home, which would silently
// run the CLI against the daemon's shared HOME.
func TestAntigravityTaskFlowFailsClosedWithoutSlotsRoot(t *testing.T) {
	t.Setenv(credentialSlotsRootEnv, "")
	t.Setenv(credentialSlotAllowlistEnv("antigravity"), currentAntigravitySlots)

	home, err := resolveCredentialAccountHome("agent-orq66", "agy")
	if err == nil {
		t.Fatalf("resolve succeeded with no slots root, home = %q", home)
	}
	if home != "" {
		t.Fatalf("failed resolve returned home %q, want empty", home)
	}
	if !strings.Contains(err.Error(), credentialSlotsRootEnv) {
		t.Fatalf("error = %v, want it to name %s", err, credentialSlotsRootEnv)
	}
}

// TestAntigravityTaskFlowRejectsSlotOutsideAllowlist proves a physically
// present but unlisted slot cannot serve a task.
func TestAntigravityTaskFlowRejectsSlotOutsideAllowlist(t *testing.T) {
	stateRoot := t.TempDir()
	slotsRoot := filepath.Join(stateRoot, "slots")
	if err := os.Mkdir(slotsRoot, 0o700); err != nil {
		t.Fatalf("create slots root: %v", err)
	}
	seedAntigravitySlot(t, slotsRoot, "slot-999", "token-999")
	t.Setenv(credentialSlotsRootEnv, slotsRoot)
	t.Setenv(credentialSlotAllowlistEnv("antigravity"), currentAntigravitySlots)

	if _, err := resolveCredentialAccountHome("agent-orq66", "agy"); err == nil {
		t.Fatal("resolve succeeded with only an unlisted slot present")
	}
}
