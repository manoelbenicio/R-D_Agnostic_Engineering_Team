//go:build windows

package agent

import (
	"os/exec"
	"syscall"
)

// createNewConsole allocates a hidden console for ordinary agent processes.
// This behavior is unrelated to catalog discovery containment.
const createNewConsole = 0x00000010

func hideAgentWindow(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= createNewConsole
}

// Windows model discovery is intentionally disabled until process creation and
// Job assignment can be performed atomically before user code runs. Attaching
// a process to a Job after cmd.Start leaves a race in which descendants can
// escape, so this implementation fails closed before cmd.Start.
type discoveryProcessTree struct{}

func requireDiscoveryProcessContainment() error {
	return errDiscoveryProcessContainmentUnavailable
}

func configureDiscoveryProcessTree(cmd *exec.Cmd) (*discoveryProcessTree, error) {
	return nil, errDiscoveryProcessContainmentUnavailable
}

func (tree *discoveryProcessTree) attach(cmd *exec.Cmd) error {
	return errDiscoveryProcessContainmentUnavailable
}

func (tree *discoveryProcessTree) terminate() error { return nil }

func (tree *discoveryProcessTree) close() error { return nil }

func terminateUnattachedDiscoveryProcess(cmd *exec.Cmd) error {
	return errDiscoveryProcessContainmentUnavailable
}
