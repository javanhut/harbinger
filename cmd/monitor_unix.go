//go:build !windows
// +build !windows

package main

import (
	"fmt"
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

func redirectOutputToLog(cmd *exec.Cmd, logPath string) error {
	// Open log file for stdout/stderr redirection
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file for redirection: %w", err)
	}
	
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	
	// The file will be closed when the process exits
	return nil
}
