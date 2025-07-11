//go:build windows
// +build windows

package main

import (
	"os"
	"syscall"
)

func sendStopSignal(process *os.Process) error {
	// On Windows, we use Kill() as there's no SIGTERM
	return process.Kill()
}

func checkProcessExists(process *os.Process) bool {
	// On Windows, we need to check differently
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(process.Pid))
	if err != nil {
		return false
	}
	syscall.CloseHandle(handle)
	return true
}
