package runtimeenv

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

type PreservedState string

const (
	PreserveSandboxConfig PreservedState = "sandbox-config"
	PreserveSkills        PreservedState = "skills"
	PreserveSessions      PreservedState = "sessions"
	PreserveWorkspace     PreservedState = "workspace"
)

// HomeContract describes the credentialless task-home boundary. It never
// names or reads an external account home.
type HomeContract struct {
	CLI                     brain.CLIKind
	ControlledHomeRequired  bool
	ProviderAuthCopyAllowed bool
	Preserve                []PreservedState
	ForbiddenClasses        []string
}

func CredentiallessHomeContract(cli brain.CLIKind) HomeContract {
	return HomeContract{
		CLI: cli, ControlledHomeRequired: true, ProviderAuthCopyAllowed: false,
		Preserve: []PreservedState{PreserveSandboxConfig, PreserveSkills, PreserveSessions, PreserveWorkspace},
		ForbiddenClasses: []string{
			"provider-auth-file", "provider-token-directory", "provider-credential-database",
			"uncontrolled-provider-config", "provider-account-home",
		},
	}
}

type HomeEntry struct {
	RelativePath string
	Directory    bool
}

var ErrTaskHomeCredential = errors.New("task home contains forbidden provider authentication or routing state")

func ValidateTaskHomeManifest(cli brain.CLIKind, entries []HomeEntry) error {
	for _, entry := range entries {
		path := filepath.ToSlash(filepath.Clean(entry.RelativePath))
		if path == "." || filepath.IsAbs(entry.RelativePath) || path == ".." || strings.HasPrefix(path, "../") {
			return ErrTaskHomeCredential
		}
		lower := strings.ToLower(path)
		base := strings.ToLower(filepath.Base(path))
		switch {
		case base == "auth.json":
			return ErrTaskHomeCredential
		case base == "nvidia_api_key":
			return ErrTaskHomeCredential
		case strings.Contains(lower, ".gemini/antigravity-cli"):
			return ErrTaskHomeCredential
		case strings.Contains(lower, "kiro-cli/data.sqlite3"):
			return ErrTaskHomeCredential
		case lower == ".cline" || strings.Contains(lower, ".cline/") || strings.HasPrefix(lower, ".cline/"):
			return ErrTaskHomeCredential
		case base == "providers.json":
			return ErrTaskHomeCredential
		case strings.Contains(lower, "opencode/auth.json"):
			return ErrTaskHomeCredential
		case base == "openclaw-user-snapshot.json":
			return ErrTaskHomeCredential
		case cli == brain.CLICodex && base == "config.json":
			return ErrTaskHomeCredential
		}
	}
	return nil
}
