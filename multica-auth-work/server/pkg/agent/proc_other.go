//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package agent

import (
	"errors"
	"os/exec"
	"syscall"
)

// hideAgentWindow is a no-op on non-Windows platforms.
func hideAgentWindow(cmd *exec.Cmd) {}

// discoveryProcessTree places short-lived catalog-discovery CLIs in their
// own process group. Descendants inherit the group, so cleanup can terminate
// the complete local tree even when the immediate child has already exited.
type discoveryProcessTree struct {
	pgid int
}

func requireDiscoveryProcessContainment() error { return nil }

func configureDiscoveryProcessTree(cmd *exec.Cmd) (*discoveryProcessTree, error) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	return &discoveryProcessTree{}, nil
}

func (tree *discoveryProcessTree) attach(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return errors.New("discovery process was not started")
	}
	tree.pgid = cmd.Process.Pid
	return nil
}

func (tree *discoveryProcessTree) terminate() error {
	if tree == nil || tree.pgid <= 0 {
		return nil
	}
	if err := syscall.Kill(-tree.pgid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}

func (tree *discoveryProcessTree) close() error { return nil }

func terminateUnattachedDiscoveryProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}
