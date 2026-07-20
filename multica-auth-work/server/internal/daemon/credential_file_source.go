package daemon

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
	"github.com/multica-ai/multica/server/internal/daemon/gateway"
)

const (
	// credentialFileMaxBytes bounds the read to reject unreasonable files
	// (binary, accidental mount, etc.) without disclosing content shape.
	credentialFileMaxBytes = 4096
	// credentialMinTrimmedLen rejects obviously empty or stub files.
	credentialMinTrimmedLen = 8
)

// FileCredentialSource is the production gateway.CredentialSource that reads a
// restricted secret file at call time, validates its content shape without
// logging or returning the value outside the callback, and fails closed on any
// metadata or content-shape violation. The value is never cached beyond the
// callback scope.
//
// Validation (fail-closed):
//   - path must be non-empty and absolute (enforced by brain.SecretFileRef)
//   - file must exist and be a regular file (no symlink, directory, device)
//   - file permissions must not be world-readable (mode & 0o004 == 0)
//   - file size must be <= credentialFileMaxBytes
//   - content must be valid UTF-8 after trimming
//   - trimmed content must be >= credentialMinTrimmedLen
//   - trimmed content must not contain control characters (except none after trim)
//   - trimmed content must not contain whitespace (single token)
type FileCredentialSource struct{}

// compile-time interface check
var _ gateway.CredentialSource = FileCredentialSource{}

func (FileCredentialSource) WithCredential(ctx context.Context, ref brain.SecretFileRef, use func(string) error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if ref.Path == "" {
		return &credentialFileError{reason: "secret_file_ref_empty"}
	}

	// Metadata validation: regular file, not world-readable.
	info, err := os.Lstat(ref.Path)
	if err != nil {
		return &credentialFileError{reason: "secret_file_not_found"}
	}
	if !info.Mode().IsRegular() {
		return &credentialFileError{reason: "secret_file_not_regular"}
	}
	if info.Mode().Perm()&0o004 != 0 {
		return &credentialFileError{reason: "secret_file_world_readable"}
	}
	if info.Size() > credentialFileMaxBytes {
		return &credentialFileError{reason: "secret_file_too_large"}
	}

	// Read and validate content shape (never log or return raw value).
	data, err := os.ReadFile(ref.Path)
	if err != nil {
		return &credentialFileError{reason: "secret_file_read_failed"}
	}
	value := strings.TrimSpace(string(data))

	if len(value) < credentialMinTrimmedLen {
		return &credentialFileError{reason: "secret_file_content_too_short"}
	}
	if !utf8.ValidString(value) {
		return &credentialFileError{reason: "secret_file_content_not_utf8"}
	}
	if strings.ContainsAny(value, " \t\n\r") {
		return &credentialFileError{reason: "secret_file_content_has_whitespace"}
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return &credentialFileError{reason: "secret_file_content_has_control_char"}
		}
	}

	return use(value)
}

// credentialFileError is a deterministic, content-free error that never leaks
// the secret value, file content, or detailed OS error messages.
type credentialFileError struct {
	reason string
}

func (e *credentialFileError) Error() string {
	return fmt.Sprintf("credential file source failed closed: %s", e.reason)
}
