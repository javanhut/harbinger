//go:build !windows
// +build !windows

package main

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func notifySignals(sigChan chan os.Signal) {
	// POSIX systems can use a single channel for all signals.
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
}

func setPlatformProcessAttributes(cmd *exec.Cmd) {
	// On Unix-like systems, create a new process group
	// to prevent signals from being passed to the parent.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
