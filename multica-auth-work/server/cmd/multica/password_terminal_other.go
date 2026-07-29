//go:build !linux && !darwin && !dragonfly && !freebsd && !netbsd && !openbsd && !windows

package main

import (
	"errors"
	"os"
)

func disableTerminalEcho(*os.File) (func() error, error) {
	return nil, errors.New("hidden terminal input is unsupported on this platform")
}
