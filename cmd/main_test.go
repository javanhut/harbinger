package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogsCommand(t *testing.T) {
	// Create a dummy log file
	pid := os.Getpid() // Use current PID for simplicity in testing
	logFilePath := getLogFileForPID(pid)

	logContent := "Line 1\nLine 2\nLine 3\n"
	err := ioutil.WriteFile(logFilePath, []byte(logContent), 0644)
	assert.NoError(t, err)
	defer os.Remove(logFilePath) // Clean up the log file

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the logs command
	rootCmd.SetArgs([]string{"logs", strconv.Itoa(pid)})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	w.Close()
	out, _ := ioutil.ReadAll(r)
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

	// Run the logs command
	rootCmd.SetArgs([]string{"logs", strconv.Itoa(dummyPID)})
	err := rootCmd.Execute()
	assert.NoError(t, err)

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = oldStdout // Restore stdout

	expectedOutput := fmt.Sprintf("No log file found for PID %d at %s. Is the monitor running in detached mode?\n", dummyPID, logFilePath)
	assert.Equal(t, expectedOutput, string(out))
}
