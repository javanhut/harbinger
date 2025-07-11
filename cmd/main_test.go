package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogsCommand(t *testing.T) {
	// Create a dummy log file
	pid := os.Getpid() // Use current PID for simplicity in testing
	logFilePath := getLogFileForPID(pid)

	logContent := "Line 1\nLine 2\nLine 3\n"
	err := os.WriteFile(logFilePath, []byte(logContent), 0644)
	require.NoError(t, err)
	defer os.Remove(logFilePath) // Clean up the log file

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the logs command directly
	err = runLogs(logsCmd, []string{strconv.Itoa(pid)})
	require.NoError(t, err)

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout // Restore stdout

	assert.Equal(t, logContent, string(out))
}

func TestLogsCommand_NoLogFile(t *testing.T) {
	// Ensure no log file exists for a dummy PID
	dummyPID := 99999
	logFilePath := getLogFileForPID(dummyPID)
	os.Remove(logFilePath)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the logs command directly
	err := runLogs(logsCmd, []string{strconv.Itoa(dummyPID)})
	require.NoError(t, err)

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout // Restore stdout

	expectedOutput := fmt.Sprintf("No log file found for PID %d at %s. Is the monitor running in detached mode?\n", dummyPID, logFilePath)
	assert.Equal(t, expectedOutput, string(out))
}

func TestGetLogFileForPID(t *testing.T) {
	tests := []struct {
		name string
		pid  int
	}{
		{"positive PID", 1234},
		{"zero PID", 0},
		{"negative PID", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logFile := getLogFileForPID(tt.pid)
			assert.Contains(t, logFile, fmt.Sprintf(".harbinger.%d.log", tt.pid))
		})
	}
}

func TestListAvailableLogs(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := listAvailableLogs()
	require.NoError(t, err)

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout // Restore stdout

	// Should contain the "No log files found" message when no logs exist
	assert.Contains(t, string(out), "No log files found")
}
