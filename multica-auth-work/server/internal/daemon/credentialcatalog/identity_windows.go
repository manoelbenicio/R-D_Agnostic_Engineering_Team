//go:build windows

package credentialcatalog

import (
	"fmt"
	"os"
)

type fileIdentity struct {
	device uint64
	inode  uint64
}

func identityFromInfo(os.FileInfo) (fileIdentity, error) {
	return fileIdentity{}, fmt.Errorf("credential catalog: metadata-only filesystem identity unsupported")
}

func ownedByEffectiveUser(os.FileInfo) (bool, error) {
	return false, fmt.Errorf("credential catalog: metadata-only owner check unsupported")
}
