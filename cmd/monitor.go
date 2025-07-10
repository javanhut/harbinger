package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/javanhut/harbinger/internal/monitor"
	"github.com/spf13/cobra"
)

var (
	pollInterval time.Duration
	repoPath     string
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
}

func runMonitor(cmd *cobra.Command, args []string) error {
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
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

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
