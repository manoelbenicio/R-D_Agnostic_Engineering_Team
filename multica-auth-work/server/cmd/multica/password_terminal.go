package main

import (
	"fmt"
	"io"
	"os"
)

func readPasswordFromTerminal(terminal *os.File, prompt io.Writer) (password string, err error) {
	restore, err := disableTerminalEcho(terminal)
	if err != nil {
		return "", fmt.Errorf("password prompt requires a terminal; use --password-stdin for automation: %w", err)
	}
	if _, err := fmt.Fprint(prompt, "New password: "); err != nil {
		_ = restore()
		return "", err
	}
	defer func() {
		restoreErr := restore()
		_, _ = fmt.Fprintln(prompt)
		if err == nil && restoreErr != nil {
			err = restoreErr
		}
	}()

	return readPasswordLine(terminal)
}
