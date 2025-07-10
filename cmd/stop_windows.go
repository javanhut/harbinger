//go:build windows
// +build windows

package main

import (
	"os"
)

func sendStopSignal(process *os.Process) error {
	// On Windows, we use Kill() as there's no SIGTERM
	return process.Kill()
}
