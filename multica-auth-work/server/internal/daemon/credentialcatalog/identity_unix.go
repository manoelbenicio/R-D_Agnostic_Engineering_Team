//go:build !windows

package credentialcatalog

import (
	"fmt"
	"os"
	"syscall"
)

type fileIdentity struct {
	device uint64
	inode  uint64
}

func identityFromInfo(info os.FileInfo) (fileIdentity, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fileIdentity{}, fmt.Errorf("credential catalog: filesystem identity unavailable")
	}
	return fileIdentity{device: uint64(stat.Dev), inode: uint64(stat.Ino)}, nil
}

func ownedByEffectiveUser(info os.FileInfo) (bool, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false, fmt.Errorf("credential catalog: owner metadata unavailable")
	}
	return stat.Uid == uint32(os.Geteuid()), nil
}
