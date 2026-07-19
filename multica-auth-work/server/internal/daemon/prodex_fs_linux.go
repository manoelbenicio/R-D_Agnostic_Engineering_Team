//go:build linux

package daemon

import (
	"fmt"

	"golang.org/x/sys/unix"
)

const (
	extFilesystemMagic = 0xef53
	xfsFilesystemMagic = 0x58465342
)

func validateApprovedPOSIXFilesystem(path string) error {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return fmt.Errorf("stat filesystem: %w", err)
	}
	switch uint64(stat.Type) {
	case extFilesystemMagic, xfsFilesystemMagic:
		return nil
	default:
		return fmt.Errorf("filesystem type 0x%x is not approved; require ext4 or xfs", uint64(stat.Type))
	}
}
