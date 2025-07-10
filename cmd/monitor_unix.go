//go:build !windows
// +build !windows

package main

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func setPlatformProcessAttributes(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
}

func getPIDFileDefaultPath() string {
	return "/tmp/harbinger.pid"
}

func notifySignals(sigChan chan os.Signal) {
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
}
