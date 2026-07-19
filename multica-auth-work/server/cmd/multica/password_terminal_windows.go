//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

func disableTerminalEcho(terminal *os.File) (func() error, error) {
	handle := windows.Handle(terminal.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return nil, err
	}
	if err := windows.SetConsoleMode(handle, mode&^windows.ENABLE_ECHO_INPUT); err != nil {
		return nil, err
	}
	return func() error { return windows.SetConsoleMode(handle, mode) }, nil
}
