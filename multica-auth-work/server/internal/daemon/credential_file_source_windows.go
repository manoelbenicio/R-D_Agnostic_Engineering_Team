//go:build windows

package daemon

import "os"

// openCredentialFile on Windows fails closed unconditionally.
// FileCredentialSource is not supported on Windows because there is no
// TOCTOU-safe equivalent of O_NOFOLLOW without Win32 API calls, and the
// gateway-required daemon is not deployed to Windows targets.
func openCredentialFile(_ string) (*os.File, error) {
	return nil, &credentialFileError{reason: "platform_unsupported"}
}

// checkCredentialOwner on Windows fails closed unconditionally.
func checkCredentialOwner(_ os.FileInfo) error {
	return &credentialFileError{reason: "platform_unsupported"}
}
