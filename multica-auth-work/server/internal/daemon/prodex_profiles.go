package daemon

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/multica-ai/multica/server/internal/l2runtime"
	"github.com/multica-ai/multica/server/internal/rotation"
)

var prodexProfileNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

type prodexState struct {
	Profiles map[string]struct {
		CodexHome string `json:"codex_home"`
	} `json:"profiles"`
}

func (d *Daemon) reconcileProdexProfiles(ctx context.Context) error {
	if !d.cfg.Prodex.Enabled || !d.cfg.L2Runtime.Enabled {
		return nil
	}
	if d.rotationStore == nil {
		return fmt.Errorf("prodex profile reconciliation failed closed: validated account inventory is unavailable")
	}
	accounts, err := d.rotationStore.ListAccounts(ctx, "codex", d.cfg.L2Runtime.TenantID)
	if err != nil {
		return fmt.Errorf("prodex profile reconciliation failed closed: list validated Codex accounts: %w", err)
	}
	slotsRoot := strings.TrimSpace(os.Getenv("MULTICA_AGENT_CREDENTIAL_SLOTS_ROOT"))
	if slotsRoot == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return fmt.Errorf("prodex profile reconciliation failed closed: resolve home: %w", homeErr)
		}
		slotsRoot = filepath.Join(home, ".agent-cred-homes", "slots")
	}
	slotsRoot, err = filepath.Abs(slotsRoot)
	if err != nil {
		return fmt.Errorf("prodex profile reconciliation failed closed: resolve slots root: %w", err)
	}
	if err := validateApprovedPOSIXFilesystem(slotsRoot); err != nil {
		return fmt.Errorf("prodex profile reconciliation failed closed: slots root: %w", err)
	}

	prodexHome := strings.TrimSpace(os.Getenv("PRODEX_HOME"))
	if prodexHome == "" {
		return fmt.Errorf("prodex profile reconciliation failed closed: PRODEX_HOME is required")
	}
	prodexHome, err = filepath.Abs(prodexHome)
	if err != nil {
		return fmt.Errorf("prodex profile reconciliation failed closed: resolve PRODEX_HOME: %w", err)
	}
	if err := validateDirectoryMode(prodexHome, 0o700); err != nil {
		return fmt.Errorf("prodex profile reconciliation failed closed: %w", err)
	}
	if err := validateApprovedPOSIXFilesystem(prodexHome); err != nil {
		return fmt.Errorf("prodex profile reconciliation failed closed: PRODEX_HOME: %w", err)
	}

	state, err := loadProdexState(filepath.Join(prodexHome, "state.json"))
	if err != nil {
		return fmt.Errorf("prodex profile reconciliation failed closed: %w", err)
	}
	identityOwners := make(map[[sha256.Size]byte]string, len(accounts))
	reconciled := make([]l2runtime.AccountProfile, 0, len(accounts))
	profileByHome := make(map[string]string, len(accounts))

	for _, account := range accounts {
		name := prodexProfileName(account)
		if !prodexProfileNamePattern.MatchString(name) {
			return fmt.Errorf("prodex profile reconciliation failed closed: invalid profile name %q", name)
		}
		codexHome, err := validateCodexSlotHome(slotsRoot, account.HomeDir)
		if err != nil {
			return fmt.Errorf("prodex profile %q rejected: %w", name, err)
		}
		authBytes, err := os.ReadFile(filepath.Join(codexHome, "auth.json"))
		if err != nil {
			return fmt.Errorf("prodex profile %q rejected: read slot-local auth: %w", name, err)
		}
		identity := sha256.Sum256(authBytes)
		if previous, exists := identityOwners[identity]; exists {
			return fmt.Errorf("prodex profiles %q and %q rejected: duplicate credential identity", previous, name)
		}
		identityOwners[identity] = name

		if existing, exists := state.Profiles[name]; exists {
			existingHome, err := filepath.Abs(existing.CodexHome)
			if err != nil || existingHome != codexHome {
				return fmt.Errorf("prodex profile %q points to a different credential home", name)
			}
		} else if err := addProdexProfileReference(ctx, d.cfg.Prodex.Path, prodexHome, name, codexHome); err != nil {
			return fmt.Errorf("prodex profile %q registration failed: %w", name, err)
		}

		reconciled = append(reconciled, l2runtime.AccountProfile{
			ProfileID:     name,
			Provider:      "codex",
			ProfileHome:   codexHome,
			AuthMode:      "oauth_profile",
			Status:        "approved",
			CapabilityRef: "codex.oauth_profile.v1",
		})
		profileByHome[codexHome] = name
	}
	if len(reconciled) == 0 && d.cfg.Prodex.Required {
		return fmt.Errorf("prodex profile reconciliation failed closed: validated account inventory has no Codex profiles")
	}

	d.l2ProfilesMu.Lock()
	d.reconciledL2Profiles = reconciled
	d.l2ProfileByHome = profileByHome
	d.l2ProfilesMu.Unlock()
	return nil
}

func prodexProfileName(account rotation.Account) string {
	home := filepath.Clean(strings.TrimSpace(account.HomeDir))
	slot := filepath.Base(filepath.Dir(home))
	if strings.HasPrefix(slot, "slot-") {
		return "codex-" + slot
	}
	id := strings.ReplaceAll(strings.TrimSpace(account.AccountID), "-", "")
	if len(id) > 12 {
		id = id[:12]
	}
	return "codex-" + id
}

func loadProdexState(path string) (prodexState, error) {
	state := prodexState{Profiles: make(map[string]struct {
		CodexHome string `json:"codex_home"`
	})}
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return prodexState{}, fmt.Errorf("read Prodex state: %w", err)
	}
	if err := json.Unmarshal(body, &state); err != nil {
		return prodexState{}, fmt.Errorf("decode Prodex state: %w", err)
	}
	if state.Profiles == nil {
		state.Profiles = make(map[string]struct {
			CodexHome string `json:"codex_home"`
		})
	}
	return state, nil
}

func validateCodexSlotHome(slotsRoot, configuredHome string) (string, error) {
	home, err := filepath.Abs(strings.TrimSpace(configuredHome))
	if err != nil {
		return "", fmt.Errorf("resolve CODEX_HOME: %w", err)
	}
	rel, err := filepath.Rel(slotsRoot, home)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("CODEX_HOME is outside the approved slots root")
	}
	if filepath.Base(home) != "codex" {
		return "", fmt.Errorf("CODEX_HOME must be the slot-local codex directory")
	}
	if err := validateApprovedPOSIXFilesystem(home); err != nil {
		return "", err
	}
	if err := validateDirectoryMode(home, 0o700); err != nil {
		return "", err
	}
	authPath := filepath.Join(home, "auth.json")
	info, err := os.Stat(authPath)
	if err != nil {
		return "", fmt.Errorf("slot-local auth.json: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || info.Size() == 0 {
		return "", fmt.Errorf("slot-local auth.json must be a non-empty regular mode-0600 file")
	}
	return home, nil
}

func validateDirectoryMode(path string, want os.FileMode) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", filepath.Base(path), err)
	}
	if !info.IsDir() || info.Mode().Perm() != want {
		return fmt.Errorf("%s must be a mode-%04o directory", filepath.Base(path), want)
	}
	return nil
}

func addProdexProfileReference(ctx context.Context, prodexPath, prodexHome, name, codexHome string) error {
	cmd := exec.CommandContext(ctx, prodexPath, "profile", "add", name, "--codex-home", codexHome)
	env := envMap(os.Environ())
	env["PRODEX_HOME"] = prodexHome
	env["PRODEX_ALLOW_UNSAFE_CHILD_ENV"] = "off"
	cmd.Env = flattenEnvMap(env)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("prodex profile add failed: %w (%s)", err, redactCommandOutput(string(output)))
	}
	return nil
}

func redactCommandOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return "no output"
	}
	return "output redacted"
}

func (d *Daemon) reconciledProdexProfiles() []l2runtime.AccountProfile {
	d.l2ProfilesMu.RLock()
	defer d.l2ProfilesMu.RUnlock()
	out := make([]l2runtime.AccountProfile, len(d.reconciledL2Profiles))
	copy(out, d.reconciledL2Profiles)
	return out
}

func (d *Daemon) prodexProfileForCredentialHome(home string) (string, bool) {
	home, err := filepath.Abs(strings.TrimSpace(home))
	if err != nil {
		return "", false
	}
	d.l2ProfilesMu.RLock()
	defer d.l2ProfilesMu.RUnlock()
	name, ok := d.l2ProfileByHome[home]
	return name, ok
}
