package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/javanhut/harbinger/internal/monitor"
	"github.com/spf13/cobra"
)

var (
	pollInterval time.Duration
	repoPath     string
	detach       bool
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Start monitoring the current Git repository for conflicts",
	Long:  `Starts a background process that monitors your Git repository for potential conflicts and remote changes.`,
	RunE:  runMonitor,
}

func init() {
	rootCmd.AddCommand(monitorCmd)
	monitorCmd.Flags().DurationVarP(&pollInterval, "interval", "i", 30*time.Second, "Polling interval for checking remote changes")
	monitorCmd.Flags().StringVarP(&repoPath, "path", "p", ".", "Path to the Git repository to monitor")
	monitorCmd.Flags().BoolVarP(&detach, "detach", "d", false, "Run monitor in the background")
}

func runMonitor(cmd *cobra.Command, args []string) error {
	// Resolve repoPath to an absolute path
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for repository: %w", err)
	}
	repoPath = absRepoPath

	if detach {
		return runDetachedMonitor()
	}

	fmt.Println("Starting Git conflict monitor...")

	// Create monitor
	m, err := monitor.New(repoPath, monitor.Options{
		PollInterval: pollInterval,
	})
	if err != nil {
		return fmt.Errorf("failed to create monitor: %w", err)
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	notifySignals(sigChan)

	// Start monitoring
	if err := m.Start(); err != nil {
		return fmt.Errorf("failed to start monitor: %w", err)
	}

	fmt.Printf("Monitoring repository at %s (checking every %s)\n", repoPath, pollInterval)
	fmt.Println("Press Ctrl+C to stop...")

	// Wait for interrupt
	<-sigChan

	fmt.Println("\nStopping monitor...")
	if err := m.Stop(); err != nil {
		log.Printf("Error stopping monitor: %v", err)
	}

	return nil
}

func runDetachedMonitor() error {
	// Get current executable path
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Build command args without the detach flag
	args := []string{"monitor"}
	if pollInterval != 30*time.Second {
		args = append(args, "--interval", pollInterval.String())
	}
			args = append(args, "--path", repoPath)
	

	// Start process in background
	cmd := exec.Command(exe, args...)

	// Redirect stdout and stderr to a log file
	logFile, err := os.OpenFile(getLogFileForPID(cmd.Process.Pid), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close() // Close the file when the function exits

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	setPlatformProcessAttributes(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start background process: %w", err)
	}

	// Write PID to file for later stopping
	pidFile := getPIDFile()
	if err := writePIDFile(pidFile, cmd.Process.Pid); err != nil {
		log.Printf("Warning: failed to write PID file: %v", err)
	}

	fmt.Printf("Running harbinger in background with process ID: %d\n", cmd.Process.Pid)
	fmt.Println("Use 'harbinger stop' to stop the background monitor")

	return nil
}

func getPIDFileDefaultPath() string {
	return "/tmp/harbinger.pid"
}

func getPIDFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return getPIDFileDefaultPath()
	}
	return filepath.Join(home, ".harbinger.pid")
}

func writePIDFile(path string, pid int) error {
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0644)
}


