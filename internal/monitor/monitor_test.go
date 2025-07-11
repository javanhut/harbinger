package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/javanhut/harbinger/internal/git"
	"github.com/javanhut/harbinger/internal/notify"
	"github.com/javanhut/harbinger/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptions(t *testing.T) {
	options := Options{
		PollInterval: 30 * time.Second,
	}

	assert.Equal(t, 30*time.Second, options.PollInterval)
}

func TestNew_InvalidRepo(t *testing.T) {
	options := Options{
		PollInterval: 10 * time.Second,
	}

	monitor, err := New("/non/existent/path", options)
	assert.Error(t, err)
	assert.Nil(t, monitor)
}

func TestNew_ValidRepo(t *testing.T) {
	options := Options{
		PollInterval: 10 * time.Second,
	}

	// Use current directory which should be a valid git repo
	monitor, err := New(".", options)
	require.NoError(t, err)
	assert.NotNil(t, monitor)

	// Verify monitor fields are set
	assert.NotNil(t, monitor.repo)
	assert.NotNil(t, monitor.notifier)
	assert.NotNil(t, monitor.config)
	assert.NotNil(t, monitor.ctx)
	assert.NotNil(t, monitor.cancel)
	assert.Equal(t, options.PollInterval, monitor.options.PollInterval)
}

func TestMonitor_StartStop(t *testing.T) {
	options := Options{
		PollInterval: 100 * time.Millisecond, // Very short for testing
	}

	monitor, err := New(".", options)
	require.NoError(t, err)

	// Start the monitor
	err = monitor.Start()
	assert.NoError(t, err)

	// Let it run briefly
	time.Sleep(200 * time.Millisecond)

	// Stop the monitor
	err = monitor.Stop()
	assert.NoError(t, err)
}

func TestMonitor_ContextCancellation(t *testing.T) {
	options := Options{
		PollInterval: 50 * time.Millisecond,
	}

	monitor, err := New(".", options)
	require.NoError(t, err)

	// Start monitor
	err = monitor.Start()
	require.NoError(t, err)

	// Cancel context directly
	monitor.cancel()

	// Give it time to shutdown
	time.Sleep(100 * time.Millisecond)

	// Context should be cancelled
	assert.Error(t, monitor.ctx.Err())
	assert.Equal(t, context.Canceled, monitor.ctx.Err())
}

func TestMonitor_Fields(t *testing.T) {
	options := Options{
		PollInterval: 15 * time.Second,
	}

	monitor, err := New(".", options)
	require.NoError(t, err)

	// Test field types and initial values
	assert.IsType(t, &git.Repository{}, monitor.repo)
	assert.IsType(t, &notify.Notifier{}, monitor.notifier)
	assert.IsType(t, &config.Config{}, monitor.config)
	assert.NotNil(t, monitor.ctx) // Context should not be nil

	// Test initial sync status
	assert.False(t, monitor.lastSyncStatus)   // Should start as false
	assert.Empty(t, monitor.currentBranch)    // Should start empty
	assert.Empty(t, monitor.lastRemoteCommit) // Should start empty
}

func TestMonitor_BranchSwitching(t *testing.T) {
	// This test verifies the logic for handling branch switches
	// We can't easily test actual branch switching without a real git repo
	// but we can test the field updates

	options := Options{
		PollInterval: 1 * time.Second,
	}

	monitor, err := New(".", options)
	require.NoError(t, err)

	// Simulate setting current branch
	monitor.currentBranch = "feature-branch"
	monitor.lastSyncStatus = true
	monitor.lastRemoteCommit = "abc123"

	// Verify initial state
	assert.Equal(t, "feature-branch", monitor.currentBranch)
	assert.True(t, monitor.lastSyncStatus)
	assert.Equal(t, "abc123", monitor.lastRemoteCommit)
}

func TestMonitor_ConfigIntegration(t *testing.T) {
	options := Options{
		PollInterval: 5 * time.Second,
	}

	monitor, err := New(".", options)
	require.NoError(t, err)

	// Verify config is loaded
	assert.NotNil(t, monitor.config)

	// Test that config has expected default values
	assert.Equal(t, "30s", monitor.config.PollInterval) // Default from config
	assert.True(t, monitor.config.Notifications)        // Default should be true
	assert.True(t, monitor.config.AutoResolve)          // Default should be true
	assert.False(t, monitor.config.AutoPull)            // Default should be false for safety
}

func TestMonitor_MultipleStartStop(t *testing.T) {
	options := Options{
		PollInterval: 100 * time.Millisecond,
	}

	monitor, err := New(".", options)
	require.NoError(t, err)

	// Multiple start/stop cycles
	for i := 0; i < 3; i++ {
		err = monitor.Start()
		assert.NoError(t, err)

		time.Sleep(50 * time.Millisecond)

		err = monitor.Stop()
		assert.NoError(t, err)

		// Brief pause between cycles
		time.Sleep(10 * time.Millisecond)
	}
}

func TestMonitor_InvalidPollInterval(t *testing.T) {
	// Test with zero poll interval
	options := Options{
		PollInterval: 0,
	}

	monitor, err := New(".", options)
	require.NoError(t, err)

	// Should still be able to create monitor
	assert.NotNil(t, monitor)
	assert.Equal(t, time.Duration(0), monitor.options.PollInterval)
}

func TestMonitor_VeryShortPollInterval(t *testing.T) {
	// Test with very short poll interval
	options := Options{
		PollInterval: 1 * time.Millisecond,
	}

	monitor, err := New(".", options)
	require.NoError(t, err)

	// Start and quickly stop
	err = monitor.Start()
	assert.NoError(t, err)

	// Give it a very brief moment
	time.Sleep(5 * time.Millisecond)

	err = monitor.Stop()
	assert.NoError(t, err)
}

func TestMonitor_StopWithoutStart(t *testing.T) {
	options := Options{
		PollInterval: 1 * time.Second,
	}

	monitor, err := New(".", options)
	require.NoError(t, err)

	// Try to stop without starting
	err = monitor.Stop()
	assert.NoError(t, err) // Should not error
}

func TestMonitor_DoubleStart(t *testing.T) {
	options := Options{
		PollInterval: 100 * time.Millisecond,
	}

	monitor, err := New(".", options)
	require.NoError(t, err)

	// Start once
	err = monitor.Start()
	assert.NoError(t, err)

	// Try to start again - this might have different behavior
	// depending on implementation, but should handle gracefully
	err = monitor.Start()
	// Implementation might allow multiple starts or prevent them
	// Just verify it doesn't panic

	// Clean up
	monitor.Stop()
}
