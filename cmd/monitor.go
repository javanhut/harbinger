package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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

	setPlatformProcessAttributes(cmd)

	// Start the process first to get the PID
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start background process: %w", err)
	}

	// Now we have the PID, create the log file
	logFile, err := os.OpenFile(getLogFileForPID(cmd.Process.Pid), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// If we can't create log file, kill the process
		cmd.Process.Kill()
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Write initial log entry
	fmt.Fprintf(logFile, "[%s] Harbinger monitor started for repository: %s\n", time.Now().Format(time.RFC3339), repoPath)
	fmt.Fprintf(logFile, "[%s] Polling interval: %s\n", time.Now().Format(time.RFC3339), pollInterval)
	fmt.Fprintf(logFile, "[%s] Process ID: %d\n", time.Now().Format(time.RFC3339), cmd.Process.Pid)
	logFile.Close() // Close our reference; the child process will open its own

	// Write PID to file for later stopping
	pidFile := getPIDFileForRepo(repoPath)
	if err := writePIDFile(pidFile, cmd.Process.Pid); err != nil {
		log.Printf("Warning: failed to write PID file: %v", err)
	}

	fmt.Printf("Running harbinger in background with process ID: %d\n", cmd.Process.Pid)
	fmt.Printf("Monitoring repository: %s\n", repoPath)
	fmt.Printf("View logs: harbinger logs %d\n", cmd.Process.Pid)
	fmt.Printf("Stop monitor: harbinger stop %d\n", cmd.Process.Pid)

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

// getPIDFileForRepo returns a repository-specific PID file path
func getPIDFileForRepo(repoPath string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/tmp"
	}

	// Create a safe filename from the repo path
	safeRepoName := filepath.Base(repoPath)
	if safeRepoName == "." || safeRepoName == "/" {
		safeRepoName = "default"
	}

	// Include a hash of the full path to handle repos with same name
	hash := fmt.Sprintf("%08x", hashString(repoPath))

	return filepath.Join(home, fmt.Sprintf(".harbinger-%s-%s.pid", safeRepoName, hash[:8]))
}

// Simple string hash function for generating unique IDs
func hashString(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h = (h ^ uint32(s[i])) * 16777619
	}
	return h
}

func writePIDFile(path string, pid int) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create PID directory: %w", err)
	}

	// Write PID and repository path
	data := fmt.Sprintf("%d\n%s\n", pid, repoPath)
	return os.WriteFile(path, []byte(data), 0644)
}
