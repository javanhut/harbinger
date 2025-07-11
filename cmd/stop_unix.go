//go:build !windows
// +build !windows

package main

import (
	"os"
	"syscall"
	"time"
)

func sendStopSignal(process *os.Process) error {
	// First check if process exists
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return err
	}

	// Send SIGTERM
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return err
	}

	// Wait a bit for graceful shutdown
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		if err := process.Signal(syscall.Signal(0)); err != nil {
			// Process no longer exists
			return nil
		}
	}

	// If still running after 1 second, force kill
	return process.Signal(syscall.SIGKILL)
}

func checkProcessExists(process *os.Process) bool {
	// On Unix, signal 0 checks if process exists without sending a signal
	err := process.Signal(syscall.Signal(0))
	return err == nil
}
