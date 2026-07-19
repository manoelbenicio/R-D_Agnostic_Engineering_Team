//go:build linux

package observability

import (
	"fmt"
	"os/exec"
	"syscall"
)

func configureSyntheticProcess(cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	return nil
}

func terminateSyntheticProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setpgid {
		return fmt.Errorf("synthetic process group was not configured")
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
		return err
	}
	return nil
}
