//go:build !windows
// +build !windows

package main

import (
	"os"
	"syscall"
)

func sendStopSignal(process *os.Process) error {
	return process.Signal(syscall.SIGTERM)
}
