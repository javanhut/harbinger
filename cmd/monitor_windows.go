//go:build windows
// +build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func notifySignals(sigChan chan os.Signal) {
	// Windows does not support signal notifications in the same way as Unix.
	// We can only listen for os.Interrupt.
	signal.Notify(sigChan, os.Interrupt)
}

func setPlatformProcessAttributes(cmd *exec.Cmd) {
	// On Windows, we can create a new process group to prevent the new process
	// from being affected by Ctrl+C events in the parent console.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
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
