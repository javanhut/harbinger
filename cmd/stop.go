package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var (
	stopAll bool
)

var stopCmd = &cobra.Command{
	Use:   "stop [PID]",
	Short: "Stop harbinger background monitors",
	Long:  `Stops harbinger monitor processes running in the background. If no PID is specified, lists all running monitors.`,
	RunE:  runStop,
	Args:  cobra.MaximumNArgs(1),
}

func init() {
	rootCmd.AddCommand(stopCmd)
	stopCmd.Flags().BoolVarP(&stopAll, "all", "a", false, "Stop all running monitors")
}

func runStop(cmd *cobra.Command, args []string) error {
	if stopAll {
		return stopAllMonitors()
	}

	if len(args) == 0 {
		// List all running monitors
		return listRunningMonitors()
	}

	// Stop specific monitor by PID
	pid, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PID: %w", err)
	}

	return stopMonitorByPID(pid)
}

func listRunningMonitors() error {
	monitors := findAllMonitors()

	if len(monitors) == 0 {
		fmt.Println("No harbinger monitors are currently running")
		return nil
	}

	fmt.Println("Running harbinger monitors:")
	fmt.Println("PID\tRepository")
	fmt.Println("---\t----------")

	for _, mon := range monitors {
		fmt.Printf("%d\t%s\n", mon.PID, mon.RepoPath)
	}

	fmt.Println("\nUse 'harbinger stop <PID>' to stop a specific monitor")
	fmt.Println("Use 'harbinger stop --all' to stop all monitors")

	return nil
}

func stopAllMonitors() error {
	monitors := findAllMonitors()

	if len(monitors) == 0 {
		fmt.Println("No harbinger monitors are currently running")
		return nil
	}

	stoppedCount := 0
	for _, mon := range monitors {
		if err := stopMonitor(mon); err == nil {
			fmt.Printf("Stopped monitor %d for %s\n", mon.PID, mon.RepoPath)
			stoppedCount++
		}
	}

	fmt.Printf("\nStopped %d monitor(s)\n", stoppedCount)
	return nil
}

func stopMonitorByPID(pid int) error {
	monitors := findAllMonitors()

	for _, mon := range monitors {
		if mon.PID == pid {
			if err := stopMonitor(mon); err != nil {
				return err
			}
			fmt.Printf("Stopped harbinger monitor (PID: %d) for %s\n", pid, mon.RepoPath)
			return nil
		}
	}

	return fmt.Errorf("no harbinger monitor found with PID %d", pid)
}

func stopMonitor(mon monitorInfo) error {
	// Find process
	process, err := os.FindProcess(mon.PID)
	if err != nil {
		// Remove stale PID file
		os.Remove(mon.PIDFile)
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send stop signal to process
	if err := sendStopSignal(process); err != nil {
		// Process might not exist, remove PID file
		os.Remove(mon.PIDFile)
		return fmt.Errorf("process not running")
	}

	// Remove PID file
	os.Remove(mon.PIDFile)

	// Clean up log file if empty or only contains startup messages
	cleanupLogFile(mon.PID)

	return nil
}

type monitorInfo struct {
	PID      int
	RepoPath string
	PIDFile  string
}

func findAllMonitors() []monitorInfo {
	var monitors []monitorInfo

	home, err := os.UserHomeDir()
	if err != nil {
		home = "/tmp"
	}

	// Look for harbinger PID files
	pattern := filepath.Join(home, ".harbinger-*.pid")
	matches, _ := filepath.Glob(pattern)

	// Also check the legacy PID file
	legacyPID := filepath.Join(home, ".harbinger.pid")
	if _, err := os.Stat(legacyPID); err == nil {
		matches = append(matches, legacyPID)
	}

	for _, pidFile := range matches {
		data, err := os.ReadFile(pidFile)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		if len(lines) < 1 {
			continue
		}

		pid, err := strconv.Atoi(strings.TrimSpace(lines[0]))
		if err != nil {
			continue
		}

		repoPath := "unknown"
		if len(lines) >= 2 {
			repoPath = strings.TrimSpace(lines[1])
		}

		// Check if process is actually running
		if isProcessRunning(pid) {
			monitors = append(monitors, monitorInfo{
				PID:      pid,
				RepoPath: repoPath,
				PIDFile:  pidFile,
			})
		} else {
			// Clean up stale PID file
			os.Remove(pidFile)
		}
	}

	return monitors
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Check if process exists by trying a null signal
	return checkProcessExists(process)
}

func cleanupLogFile(pid int) {
	logFile := getLogFileForPID(pid)

	// Check if log file exists
	info, err := os.Stat(logFile)
	if err != nil {
		return // File doesn't exist
	}

	// If file is small (likely only contains startup messages), remove it
	if info.Size() < 1024 { // Less than 1KB
		os.Remove(logFile)
		return
	}

	// Check if file only contains startup messages
	file, err := os.Open(logFile)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	hasRealContent := false

	for scanner.Scan() && lineCount < 10 {
		line := scanner.Text()
		lineCount++

		// Check if line contains actual monitoring output
		if !strings.Contains(line, "monitor started") &&
			!strings.Contains(line, "Polling interval") &&
			!strings.Contains(line, "Process ID") &&
			strings.TrimSpace(line) != "" {
			hasRealContent = true
			break
		}
	}

	// If no real content, remove the file
	if !hasRealContent && lineCount < 10 {
		file.Close()
		os.Remove(logFile)
	}
}
