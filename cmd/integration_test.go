package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/javanhut/harbinger/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_PIDFileOperations(t *testing.T) {
	// Test the full PID file lifecycle
	testRepoPath := "/test/repo/path"
	
	// Test PID file generation
	pidFile := getPIDFileForRepo(testRepoPath)
	assert.Contains(t, pidFile, ".harbinger-")
	assert.Contains(t, pidFile, ".pid")
	
	// Test hash string function
	hash1 := hashString(testRepoPath)
	hash2 := hashString(testRepoPath)
	hash3 := hashString("/different/path")
	
	assert.Equal(t, hash1, hash2, "Same path should generate same hash")
	assert.NotEqual(t, hash1, hash3, "Different paths should generate different hashes")
	
	// Test PID file writing and cleanup
	testPID := 12345
	// Set the global repoPath variable for testing
	originalRepoPath := repoPath
	repoPath = testRepoPath
	defer func() { repoPath = originalRepoPath }()
	
	err := writePIDFile(pidFile, testPID)
	require.NoError(t, err)
	defer os.Remove(pidFile)
	
	// Verify file exists and contains correct data
	data, err := os.ReadFile(pidFile)
	require.NoError(t, err)
	
	content := string(data)
	assert.Contains(t, content, "12345")
	assert.Contains(t, content, testRepoPath)
}

func TestIntegration_LogFileOperations(t *testing.T) {
	testPID := 99998
	logFile := getLogFileForPID(testPID)
	
	// Create a test log file with enough content to exceed the 1KB threshold
	logContent := `[2023-01-01T12:00:00Z] Harbinger monitor started for repository: /test/repo
[2023-01-01T12:00:00Z] Polling interval: 30s
[2023-01-01T12:00:00Z] Process ID: 99998
[2023-01-01T12:00:05Z] Repository status changed
[2023-01-01T12:00:35Z] Detected remote changes
[2023-01-01T12:00:45Z] Branch synchronization complete
[2023-01-01T12:01:00Z] Performing repository check
[2023-01-01T12:01:15Z] Fetching remote changes
[2023-01-01T12:01:30Z] Comparing local and remote branches
[2023-01-01T12:01:45Z] Found new commits on remote
[2023-01-01T12:02:00Z] Notifying user of changes
[2023-01-01T12:02:15Z] Continuing monitoring
[2023-01-01T12:02:30Z] Next check scheduled
[2023-01-01T12:02:45Z] System resources checked
[2023-01-01T12:03:00Z] Network connectivity verified
[2023-01-01T12:03:15Z] Git repository validation complete
[2023-01-01T12:03:30Z] Remote tracking branch updated
[2023-01-01T12:03:45Z] Local branch status verified
[2023-01-01T12:04:00Z] Monitoring cycle complete
[2023-01-01T12:04:15Z] Waiting for next polling interval
[2023-01-01T12:04:30Z] Background monitoring continues
[2023-01-01T12:04:45Z] All systems operational
`
	
	err := os.WriteFile(logFile, []byte(logContent), 0644)
	require.NoError(t, err)
	defer os.Remove(logFile)
	
	// Test log file cleanup - should NOT be cleaned up (has real content)
	cleanupLogFile(testPID)
	
	// File should still exist
	_, err = os.Stat(logFile)
	if err != nil {
		t.Logf("Log file was cleaned up, but expected to remain. Content was: %s", logContent)
	}
	assert.NoError(t, err, "Log file with real content should not be cleaned up")
}

func TestIntegration_LogFileCleanup(t *testing.T) {
	testPID := 99997
	logFile := getLogFileForPID(testPID)
	
	// Create a log file with only startup messages
	startupOnlyContent := `[2023-01-01T12:00:00Z] Harbinger monitor started for repository: /test/repo
[2023-01-01T12:00:00Z] Polling interval: 30s
[2023-01-01T12:00:00Z] Process ID: 99997
`
	
	err := os.WriteFile(logFile, []byte(startupOnlyContent), 0644)
	require.NoError(t, err)
	
	// Test log file cleanup - should be cleaned up (only startup messages)
	cleanupLogFile(testPID)
	
	// File should be removed
	_, err = os.Stat(logFile)
	assert.True(t, os.IsNotExist(err), "Log file with only startup messages should be cleaned up")
}

func TestIntegration_CommandLineInterface(t *testing.T) {
	// Test that all main commands can be invoked without panic
	tests := []struct {
		name string
		args []string
	}{
		{"help", []string{"--help"}},
		{"version", []string{"--version"}}, // This might not exist, but shouldn't panic
		{"test help", []string{"test", "--help"}},
		{"logs help", []string{"logs", "--help"}},
		{"stop help", []string{"stop", "--help"}},
		{"monitor help", []string{"monitor", "--help"}},
		{"resolve help", []string{"resolve", "--help"}},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset rootCmd args
			rootCmd.SetArgs(tt.args)
			
			// This should not panic, even if it returns an error
			assert.NotPanics(t, func() {
				rootCmd.Execute()
			})
		})
	}
}

func TestIntegration_FilePathOperations(t *testing.T) {
	// Test cross-platform path operations
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	
	// Test various PID file paths
	testPaths := []string{
		".",
		"/tmp/test-repo",
		home + "/projects/harbinger",
		"relative/path",
	}
	
	for _, testPath := range testPaths {
		t.Run("path_"+filepath.Base(testPath), func(t *testing.T) {
			pidFile := getPIDFileForRepo(testPath)
			
			// Should always be an absolute path
			assert.True(t, filepath.IsAbs(pidFile), "PID file path should be absolute")
			
			// Should contain harbinger and .pid
			assert.Contains(t, pidFile, "harbinger")
			assert.Contains(t, pidFile, ".pid")
			
			// Should be in home directory or /tmp
			assert.True(t, 
				filepath.HasPrefix(pidFile, home) || filepath.HasPrefix(pidFile, "/tmp"),
				"PID file should be in home or /tmp directory")
		})
	}
}

func TestIntegration_HashStringFunction(t *testing.T) {
	// Test hash string function properties
	testCases := []struct {
		input    string
		expected uint32 // We can't predict exact values, but we can test properties
	}{
		{"", 2166136261}, // FNV-1a offset basis
		{"test", 0},      // We'll just check it's not the offset basis
		{"different", 0}, // We'll just check it's different from "test"
	}
	
	results := make(map[string]uint32)
	
	for _, tc := range testCases {
		result := hashString(tc.input)
		results[tc.input] = result
		
		if tc.input == "" {
			assert.Equal(t, tc.expected, result, "Empty string should return offset basis")
		} else {
			assert.NotEqual(t, uint32(2166136261), result, "Non-empty string should not return offset basis")
		}
	}
	
	// Different inputs should produce different hashes (with high probability)
	assert.NotEqual(t, results["test"], results["different"], "Different inputs should produce different hashes")
	
	// Same input should always produce same hash
	assert.Equal(t, hashString("test"), hashString("test"), "Same input should produce same hash")
}

func TestIntegration_ConfigurationFlow(t *testing.T) {
	// Test complete configuration loading and usage flow
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ".harbinger.yaml")
	
	// Create a comprehensive config
	configContent := `poll_interval: 45s
editor: code
notifications: true
auto_resolve: false
auto_pull: true
ignore_branches:
  - main
  - master
  - develop
  - staging
`
	
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)
	
	// Test config loading through the initialization process
	// Test SetConfigFile function
	config.SetConfigFile(configFile)
	
	// Load the config to verify it works
	cfg, err := config.Load()
	require.NoError(t, err)
	
	// Verify config was loaded correctly
	assert.Equal(t, "45s", cfg.PollInterval)
	assert.Equal(t, "code", cfg.Editor)
	assert.True(t, cfg.Notifications)
	assert.False(t, cfg.AutoResolve)
	assert.True(t, cfg.AutoPull)
	assert.Equal(t, []string{"main", "master", "develop", "staging"}, cfg.IgnoreBranches)
}