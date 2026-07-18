package deploy

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

const (
	DefaultSecretDirectory = "/etc/agent-brain/secrets"
	DefaultSecretPath      = DefaultSecretDirectory + "/omniroute-inference-key"
	ServiceOwner           = "root"
	ServiceGroup           = "agent-brain"
	MaxSecretReferenceSize = int64(16 * 1024)
)

// ReadSafetyCheck is a metadata check required before a later runtime may read
// a secret. This package only describes the checks; it never opens the path.
type ReadSafetyCheck string

const (
	CheckAbsolutePath     ReadSafetyCheck = "absolute-path"
	CheckNoSymlink        ReadSafetyCheck = "no-symlink"
	CheckRegularFile      ReadSafetyCheck = "regular-file"
	CheckExpectedOwner    ReadSafetyCheck = "expected-owner"
	CheckExpectedGroup    ReadSafetyCheck = "expected-group"
	CheckRestrictedMode   ReadSafetyCheck = "restricted-mode"
	CheckNonEmpty         ReadSafetyCheck = "non-empty"
	CheckBoundedSize      ReadSafetyCheck = "bounded-size"
	CheckNoContentInError ReadSafetyCheck = "no-content-in-error"
)

// SecretReferenceContract is intentionally incapable of carrying a secret
// value. It defines only the reference and its operational controls.
type SecretReferenceContract struct {
	EvidenceID        string
	ConfigEnvironment string
	Directory         string
	Path              string
	Owner             string
	Group             string
	DirectoryMode     fs.FileMode
	FileMode          fs.FileMode
	MaxBytes          int64
	ReadSafety        []ReadSafetyCheck
	InjectionPolicy   string
	Provisioning      string
	Rotation          string
	LoggingEvidence   string
	RevocationFailure string
	Backup            string
}

// DefaultSecretReferenceContract returns the frozen G2D reference contract.
func DefaultSecretReferenceContract() SecretReferenceContract {
	return SecretReferenceContract{
		EvidenceID:        "EV-G2D-01",
		ConfigEnvironment: brain.EnvGatewaySecretFile,
		Directory:         DefaultSecretDirectory,
		Path:              DefaultSecretPath,
		Owner:             ServiceOwner,
		Group:             ServiceGroup,
		DirectoryMode:     0o750,
		FileMode:          0o440,
		MaxBytes:          MaxSecretReferenceSize,
		ReadSafety: []ReadSafetyCheck{
			CheckAbsolutePath,
			CheckNoSymlink,
			CheckRegularFile,
			CheckExpectedOwner,
			CheckExpectedGroup,
			CheckRestrictedMode,
			CheckNonEmpty,
			CheckBoundedSize,
			CheckNoContentInError,
		},
		InjectionPolicy:   "read once in the authorized service; inject only after environment sanitization; never copy into task homes",
		Provisioning:      "authorized operator atomically installs an externally derived value; repository, image, command line, screenshots, and logs contain only the reference",
		Rotation:          "stage a restricted sibling reference target, validate metadata, atomically rename, then perform a controlled reload or restart",
		LoggingEvidence:   "record reference class, metadata result, generation, timestamp, and safe outcome code only; never record value, authorization data, fingerprint, or hash",
		RevocationFailure: "make gateway readiness false and fail closed for new inference; never restore provider-native, Prodex, or legacy router fallback",
		Backup:            "exclude plaintext from ordinary repository and configuration backups; use the approved audited secret escrow and restore process",
	}
}

// Validate verifies the reference contract without touching the filesystem.
func (c SecretReferenceContract) Validate() error {
	if c.EvidenceID != "EV-G2D-01" {
		return fmt.Errorf("unexpected secret-reference evidence id")
	}
	if c.ConfigEnvironment != brain.EnvGatewaySecretFile {
		return fmt.Errorf("secret reference must use frozen environment name")
	}
	if !filepath.IsAbs(c.Directory) || !filepath.IsAbs(c.Path) {
		return fmt.Errorf("secret directory and path must be absolute")
	}
	if filepath.Dir(filepath.Clean(c.Path)) != filepath.Clean(c.Directory) {
		return fmt.Errorf("secret path must be contained directly in the restricted directory")
	}
	if c.Owner == "" || c.Group == "" {
		return fmt.Errorf("secret reference owner and group are required")
	}
	if c.DirectoryMode.Perm() != 0o750 || c.FileMode.Perm() != 0o440 {
		return fmt.Errorf("secret reference modes must remain 0750/0440")
	}
	if c.DirectoryMode.Perm()&0o007 != 0 || c.FileMode.Perm()&0o007 != 0 {
		return fmt.Errorf("secret reference must deny world access")
	}
	if c.FileMode.Perm()&0o222 != 0 {
		return fmt.Errorf("installed secret reference must not be writable")
	}
	if c.MaxBytes <= 0 || c.MaxBytes > MaxSecretReferenceSize {
		return fmt.Errorf("secret reference size bound is invalid")
	}
	required := map[ReadSafetyCheck]bool{
		CheckAbsolutePath:     false,
		CheckNoSymlink:        false,
		CheckRegularFile:      false,
		CheckExpectedOwner:    false,
		CheckExpectedGroup:    false,
		CheckRestrictedMode:   false,
		CheckNonEmpty:         false,
		CheckBoundedSize:      false,
		CheckNoContentInError: false,
	}
	for _, check := range c.ReadSafety {
		if _, ok := required[check]; !ok {
			return fmt.Errorf("unknown read-safety check %q", check)
		}
		required[check] = true
	}
	for check, present := range required {
		if !present {
			return fmt.Errorf("missing read-safety check %q", check)
		}
	}
	if c.InjectionPolicy == "" || c.Provisioning == "" || c.Rotation == "" ||
		c.LoggingEvidence == "" || c.RevocationFailure == "" || c.Backup == "" {
		return fmt.Errorf("all secret-reference operational policies are required")
	}
	return nil
}
