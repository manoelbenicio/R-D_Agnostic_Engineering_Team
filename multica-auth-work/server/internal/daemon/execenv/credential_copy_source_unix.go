//go:build !windows

package execenv

import (
	"os"
	"syscall"
)

// openCredentialCopySource atomically rejects a symlink at the final path
// component. copyCredentialFile then fstats this same descriptor and compares
// it with the earlier Lstat result before reading any bytes.
func openCredentialCopySource(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
}
