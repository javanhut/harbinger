//go:build windows
// +build windows

package main

import (
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
)

func setPlatformProcessAttributes(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func getPIDFileDefaultPath() string {
	// On Windows, use temp directory
	return filepath.Join(os.TempDir(), "harbinger.pid")
}

func notifySignals(sigChan chan os.Signal) {
	// On Windows, only listen for os.Interrupt (Ctrl+C)
	signal.Notify(sigChan, os.Interrupt)
}
