//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

// openCredentialFile opens the file with O_NOFOLLOW, which causes the kernel
// to return ELOOP if the path is a symlink — rejecting the symlink atomically
// at open time rather than in a separate Lstat that races.
func openCredentialFile(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		if isELOOP(err) {
			return nil, &credentialFileError{reason: "secret_file_is_symlink"}
		}
		return nil, &credentialFileError{reason: "secret_file_not_found"}
	}
	return f, nil
}

// isELOOP reports whether err is the ELOOP errno returned by O_NOFOLLOW on a symlink.
func isELOOP(err error) bool {
	if pe, ok := err.(*os.PathError); ok {
		return pe.Err == syscall.ELOOP
	}
	return false
}

// checkCredentialOwner validates that the file described by info is owned by
// the current process UID.
func checkCredentialOwner(info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return &credentialFileError{reason: "secret_file_stat_unsupported"}
	}
	if stat.Uid != uint32(os.Getuid()) {
		return &credentialFileError{reason: "secret_file_wrong_owner"}
	}
	return nil
}
