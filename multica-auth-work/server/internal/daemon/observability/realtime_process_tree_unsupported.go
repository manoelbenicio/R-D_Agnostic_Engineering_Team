//go:build !linux

package observability

import "os/exec"

func configureSyntheticProcess(cmd *exec.Cmd) error {
	return &UnsupportedHostMetricError{Metric: "process-tree", Reason: "process-group containment is unsupported on this platform"}
}

func terminateSyntheticProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
