//go:build windows

package execenv

import (
	"fmt"
	"os"
)

// Windows deployment is not supported for credential-isolated task homes:
// os.Open has no O_NOFOLLOW equivalent and a best-effort check would retain
// the path-swap race this helper exists to close.
func openCredentialCopySource(_ string) (*os.File, error) {
	return nil, fmt.Errorf("credential copy source is unsupported on windows")
}
