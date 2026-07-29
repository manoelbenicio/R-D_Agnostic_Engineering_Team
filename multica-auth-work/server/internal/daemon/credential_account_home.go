package daemon

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/multica-ai/multica/server/internal/credentialregistry"
)

var errCredentialAccountHomeUnavailable = errors.New("approved credential account home unavailable")

const credentialAssignmentEnforcementEnv = "MULTICA_CREDENTIAL_ASSIGNMENT_ENFORCED"
const credentialSlotsRootEnv = "MULTICA_CREDENTIAL_SLOTS_ROOT"

func credentialAssignmentEnforcementEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(credentialAssignmentEnforcementEnv))) {
	case "1", "true":
		return true
	default:
		return false
	}
}

// validateCredentialAccountHome is a metadata-only gate. It verifies the
// approved path itself without opening or reading any credential file.
func validateCredentialAccountHome(provider, path string, required bool) (string, error) {
	provider = credentialregistry.CanonicalProvider(provider)
	coveredProvider := credentialregistry.RequiresApprovedAssignment(provider)
	if !required {
		// During the mixed-version rollout this remains off until every server
		// can emit assignment metadata. Once enabled, a covered provider can
		// never silently fall back to the shared/global credential home.
		if coveredProvider && credentialAssignmentEnforcementEnabled() {
			return "", errCredentialAccountHomeUnavailable
		}
		return "", nil
	}
	if !coveredProvider {
		return "", errCredentialAccountHomeUnavailable
	}
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return "", errCredentialAccountHomeUnavailable
	}
	clean := filepath.Clean(path)
	if clean == string(filepath.Separator) || clean != path {
		return "", errCredentialAccountHomeUnavailable
	}
	root, err := controlledCredentialSlotsRoot()
	if err != nil || !pathIsControlledSlotHome(root, clean) {
		return "", errCredentialAccountHomeUnavailable
	}
	for _, candidate := range []string{root, filepath.Dir(clean), clean} {
		if err := validatePrivateOwnedPath(candidate, true); err != nil {
			return "", errCredentialAccountHomeUnavailable
		}
	}
	if err := validateProviderCredentialLayout(provider, clean); err != nil {
		return "", errCredentialAccountHomeUnavailable
	}
	return clean, nil
}

func controlledCredentialSlotsRoot() (string, error) {
	root := strings.TrimSpace(os.Getenv(credentialSlotsRootEnv))
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, ".agent-cred-homes", "slots")
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == string(filepath.Separator) {
		return "", errCredentialAccountHomeUnavailable
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || resolved != root {
		return "", errCredentialAccountHomeUnavailable
	}
	return root, nil
}

func pathIsControlledSlotHome(root, home string) bool {
	rel, err := filepath.Rel(root, home)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	return len(parts) == 2 && parts[0] != "" && parts[0] != "." && parts[1] == "home"
}

func validatePrivateOwnedPath(path string, wantDir bool) error {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		return errCredentialAccountHomeUnavailable
	}
	if wantDir != info.IsDir() {
		return errCredentialAccountHomeUnavailable
	}
	if info.Mode().Perm()&0o077 != 0 {
		return errCredentialAccountHomeUnavailable
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return errCredentialAccountHomeUnavailable
	}
	return nil
}

func validateProviderCredentialLayout(provider, home string) error {
	switch provider {
	case "codex":
		return validatePrivateOwnedPath(filepath.Join(home, "auth.json"), false)
	case "kiro":
		return validatePrivateOwnedPath(filepath.Join(home, "kiro-cli", "data.sqlite3"), false)
	case "antigravity":
		root := filepath.Join(home, ".gemini", "antigravity-cli")
		if err := validatePrivateOwnedPath(root, true); err != nil {
			return err
		}
		return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return errCredentialAccountHomeUnavailable
			}
			info, err := entry.Info()
			if err != nil || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
				return errCredentialAccountHomeUnavailable
			}
			stat, ok := info.Sys().(*syscall.Stat_t)
			if !ok || stat.Uid != uint32(os.Geteuid()) {
				return errCredentialAccountHomeUnavailable
			}
			if !info.IsDir() && !info.Mode().IsRegular() {
				return errCredentialAccountHomeUnavailable
			}
			return nil
		})
	default:
		return errCredentialAccountHomeUnavailable
	}
}
