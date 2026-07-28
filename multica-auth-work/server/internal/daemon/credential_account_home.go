package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/multica-ai/multica/server/internal/credentialregistry"
)

var errCredentialAccountHomeUnavailable = errors.New("approved credential account home unavailable")

// validateCredentialAccountHome is a metadata-only gate. It verifies the
// approved path itself without opening or reading any credential file.
func validateCredentialAccountHome(provider, path string, required bool) (string, error) {
	if !required {
		return "", nil
	}
	if !credentialregistry.RequiresApprovedAssignment(provider) {
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
	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil || resolved != clean {
		return "", errCredentialAccountHomeUnavailable
	}
	info, err := os.Lstat(clean)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", errCredentialAccountHomeUnavailable
	}
	return clean, nil
}
