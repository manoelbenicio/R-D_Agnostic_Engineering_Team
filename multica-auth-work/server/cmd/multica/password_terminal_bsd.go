//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

func disableTerminalEcho(terminal *os.File) (func() error, error) {
	fd := int(terminal.Fd())
	state, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		return nil, err
	}
	noEcho := *state
	noEcho.Lflag &^= unix.ECHO
	if err := unix.IoctlSetTermios(fd, unix.TIOCSETA, &noEcho); err != nil {
		return nil, err
	}
	return func() error { return unix.IoctlSetTermios(fd, unix.TIOCSETA, state) }, nil
}
