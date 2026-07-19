//go:build !linux

package daemon

import (
	"fmt"
	"runtime"
)

func validateApprovedPOSIXFilesystem(_ string) error {
	return fmt.Errorf("Prodex credential filesystem validation is not implemented on %s", runtime.GOOS)
}
