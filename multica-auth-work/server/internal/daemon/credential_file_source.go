package daemon

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
	"github.com/multica-ai/multica/server/internal/daemon/gateway"
)

const (
	// credentialFileMaxBytes bounds the read to reject unreasonable files.
	credentialFileMaxBytes = 4096
	// credentialMinTrimmedLen rejects obviously empty or stub files.
	credentialMinTrimmedLen = 8
	// credentialAllowedModeMask: any group or world bit set → reject.
	// Accepts 0600 (owner rw only) or stricter (0400).
	credentialAllowedModeMask = os.FileMode(0o077)
)

// FileCredentialSource is the production gateway.CredentialSource.
// It reads a restricted secret file at call time with fail-closed
// metadata and content-shape validation. The value is never cached,
// logged, or returned outside the callback scope.
//
// Security model:
//   - File is opened with O_NOFOLLOW semantics (see platform helpers) to
//     reject symlinks atomically at open time, eliminating the TOCTOU race
//     between a separate Lstat and a subsequent ReadFile.
//   - Metadata (type, mode, owner) is validated via Fstat on the SAME open
//     file descriptor — not a re-stat of the path.
//   - Content is read from the SAME descriptor.
//   - Mode must have no group or world bits (enforces ≤ 0600).
//   - Owner must be the current process UID (platform-specific; see helpers).
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

	// openNoFollow opens the file with O_NOFOLLOW on Unix (ELOOP on symlink)
	// or a best-effort Lstat guard on Windows — see platform build-tagged files.
	f, err := openCredentialFile(ref.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Fstat the open descriptor — validates the same object we will read.
	info, err := f.Stat()
	if err != nil {
		return &credentialFileError{reason: "secret_file_stat_failed"}
	}
	if !info.Mode().IsRegular() {
		return &credentialFileError{reason: "secret_file_not_regular"}
	}
	// Reject any group or world permission bit.
	if info.Mode().Perm()&credentialAllowedModeMask != 0 {
		return &credentialFileError{reason: "secret_file_permissions_too_open"}
	}
	// Verify owner matches the running process UID (platform-specific).
	if err := checkCredentialOwner(info); err != nil {
		return err
	}
	if info.Size() > credentialFileMaxBytes {
		return &credentialFileError{reason: "secret_file_too_large"}
	}

	// Read from the same open descriptor — no reopen, no TOCTOU.
	buf := make([]byte, credentialFileMaxBytes+1)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return &credentialFileError{reason: "secret_file_read_failed"}
	}
	if n > credentialFileMaxBytes {
		return &credentialFileError{reason: "secret_file_too_large"}
	}

	value := strings.TrimSpace(string(buf[:n]))

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
