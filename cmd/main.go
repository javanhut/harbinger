package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/javanhut/harbinger/pkg/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "harbinger",
		Short: "A Git conflict monitoring tool that notifies you when your branch needs attention",
		Long: `Harbinger monitors your Git repository in the background and notifies you when:
- Your branch is out of sync with the remote
- There are potential merge conflicts
- Remote changes might affect your work

It provides an interactive conflict resolution interface right in your terminal.`,
	}
	logsCmd = &cobra.Command{
		Use:   "logs [PID]",
		Short: "Read logs from a specific background monitor process",
		Long:  `Reads and displays the logs generated by a detached harbinger monitor process, identified by its PID.`,
		Args:  cobra.MaximumNArgs(1), // Allow 0 or 1 argument
		RunE:  runLogs,
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.harbinger.yaml)")
	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// No PID provided, list available log files
		return listAvailableLogs()
	}

	pid, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PID: %w", err)
	}

	logFile := getLogFileForPID(pid)
	f, err := os.Open(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("No log file found for PID %d at %s. Is the monitor running in detached mode?\n", pid, logFile)
			return nil
		}
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading log file: %w", err)
	}
	return nil
}

func listAvailableLogs() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Look for log files
	pattern := filepath.Join(home, ".harbinger.*.log")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find log files: %w", err)
	}

	if len(matches) == 0 {
		fmt.Println("No log files found.")
		fmt.Println("Log files are created when monitors are run with --detach flag.")
		return nil
	}

	fmt.Println("Available log files:")
	for _, logFile := range matches {
		// Extract PID from filename
		base := filepath.Base(logFile)
		parts := strings.Split(base, ".")
		if len(parts) >= 3 {
			pid := parts[len(parts)-2]
			fmt.Printf("  PID %s: %s\n", pid, logFile)
		}
	}
	fmt.Println("\nUse 'harbinger logs <PID>' to view a specific log file.")
	return nil
}

func getLogFileForPID(pid int) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Sprintf("/tmp/harbinger.%d.log", pid)
	}
	return filepath.Join(home, fmt.Sprintf(".harbinger.%d.log", pid))
}

func initConfig() {
	if cfgFile != "" {
		config.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		config.SetConfigPath(home)
		config.SetConfigName(".harbinger.yaml")
		
		// Create default config file if it doesn't exist
		configPath := filepath.Join(home, ".harbinger.yaml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			defaultConfig := &config.Config{
				PollInterval:   "30s",
				Editor:         "code",
				Notifications:  true,
				AutoResolve:    true,
				AutoSync:       false,
				IgnoreBranches: []string{"main", "master"},
			}
			if err := config.Save(defaultConfig); err != nil {
				log.Printf("Warning: Failed to create default config file: %v", err)
			} else {
				log.Printf("Created default configuration file at %s", configPath)
			}
		}
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
