//go:build !windows && !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package agent

import (
	"errors"
	"os/exec"
)

// Unsupported hosts fail closed rather than falling back to parent-only
// termination, which could leave a discovery descendant running.
type discoveryProcessTree struct{}

func hideAgentWindow(cmd *exec.Cmd) {}

func requireDiscoveryProcessContainment() error {
	return errDiscoveryProcessContainmentUnavailable
}

func configureDiscoveryProcessTree(cmd *exec.Cmd) (*discoveryProcessTree, error) {
	return nil, errDiscoveryProcessContainmentUnavailable
}

func (tree *discoveryProcessTree) attach(cmd *exec.Cmd) error {
	return errors.New("catalog discovery process-tree containment is unsupported")
}

func (tree *discoveryProcessTree) terminate() error { return nil }

func (tree *discoveryProcessTree) close() error { return nil }

func terminateUnattachedDiscoveryProcess(cmd *exec.Cmd) error {
	return errors.New("catalog discovery process-tree containment is unsupported")
}
