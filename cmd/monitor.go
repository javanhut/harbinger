package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/javanhut/harbinger/internal/monitor"
	"github.com/spf13/cobra"
)

var (
	pollInterval time.Duration
	repoPath     string
	detach       bool
	remoteBranch string
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
	monitorCmd.Flags().StringVarP(&remoteBranch, "remote-branch", "r", "", "Remote branch to monitor (e.g., 'main', 'develop')")
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
		RemoteBranch: remoteBranch,
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
	if remoteBranch != "" {
		args = append(args, "--remote-branch", remoteBranch)
	}

	// Start process in background
	cmd := exec.Command(exe, args...)

	setPlatformProcessAttributes(cmd)

	// Create log file in the user's home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Use a temporary PID for the log file name
	tempPID := os.Getpid()
	logPath := filepath.Join(home, fmt.Sprintf(".harbinger.temp.%d.log", tempPID))
	
	// Redirect output to log file
	if err := redirectOutputToLog(cmd, logPath); err != nil {
		return fmt.Errorf("failed to redirect output: %w", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		os.Remove(logPath)
		return fmt.Errorf("failed to start background process: %w", err)
	}

	// Now rename the log file with the actual PID
	actualLogPath := getLogFileForPID(cmd.Process.Pid)
	if err := os.Rename(logPath, actualLogPath); err != nil {
		// If rename fails, keep the temp name
		actualLogPath = logPath
	}

	// Write PID to file for later stopping
	pidFile := getPIDFileForRepoAndBranch(repoPath, remoteBranch)
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
	return getPIDFileForRepoAndBranch(repoPath, "")
}

// getPIDFileForRepoAndBranch returns a repository and branch specific PID file path
func getPIDFileForRepoAndBranch(repoPath, branch string) string {
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

	if branch != "" {
		// Sanitize branch name
		safeBranch := strings.ReplaceAll(branch, "/", "-")
		safeBranch = strings.ReplaceAll(safeBranch, ".", "-")
		return filepath.Join(home, fmt.Sprintf(".harbinger-%s-%s-%s.pid", safeRepoName, hash[:8], safeBranch))
	}

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
