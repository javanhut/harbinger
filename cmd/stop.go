package main

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop any running harbinger background monitors",
	Long:  `Stops all harbinger monitor processes running in the background.`,
	RunE:  runStop,
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	pidFile := getPIDFile()
	// Check if PID file exists
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		fmt.Println("No background harbinger monitor found")
		return nil
	}

	// Read PID from file
	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(string(pidBytes))
	if err != nil {
		return fmt.Errorf("invalid PID in file: %w", err)
	}

	// Find process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send SIGTERM to process
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process might not exist, remove PID file
		os.Remove(pidFile)
		fmt.Println("No background harbinger monitor found")
		return nil
	}

	// Remove PID file
	os.Remove(pidFile)
	fmt.Printf("Stopped harbinger monitor (PID: %d)\n", pid)

	return nil
}
