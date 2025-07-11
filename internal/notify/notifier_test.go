package notify

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsWSL(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
		goos     string
	}{
		{
			name:     "WSL detected with microsoft in content",
			content:  "Linux version 5.10.16.3-microsoft-standard-WSL2",
			expected: true,
			goos:     "linux",
		},
		{
			name:     "WSL detected with Microsoft (capitalized) in content",
			content:  "Linux version 5.10.16.3-Microsoft-standard-WSL2",
			expected: true,
			goos:     "linux",
		},
		{
			name:     "Regular Linux not detected as WSL",
			content:  "Linux version 5.4.0-77-generic (buildd@lcy01-amd64-020)",
			expected: false,
			goos:     "linux",
		},
		{
			name:     "Non-Linux OS not detected as WSL",
			content:  "",
			expected: false,
			goos:     "darwin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.goos != runtime.GOOS && tt.goos != "" {
				t.Skip("Test only applies to " + tt.goos)
			}

			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "proc_version")
			
			if tt.content != "" {
				err := os.WriteFile(testFile, []byte(tt.content), 0644)
				require.NoError(t, err)
			} else {
				// Test non-existent file
				testFile = filepath.Join(tmpDir, "non_existent_file")
			}

			result := isWSL(testFile)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNotifier_New(t *testing.T) {
	notifier := New()
	assert.NotNil(t, notifier)
	// We can't easily test the internal useDesktopNotifications field
	// without making it exported, but we can verify the notifier is created
}

func TestNotifier_NotificationMethods(t *testing.T) {
	notifier := New()
	
	// These tests verify the methods don't panic and can be called
	// Actual notification testing would require platform-specific mocking
	t.Run("NotifyInSync", func(t *testing.T) {
		assert.NotPanics(t, func() {
			notifier.NotifyInSync("test-branch")
		})
	})
	
	t.Run("NotifyRemoteChange", func(t *testing.T) {
		assert.NotPanics(t, func() {
			notifier.NotifyRemoteChange("test-branch", "abc123def456")
		})
	})
	
	t.Run("NotifyOutOfSync", func(t *testing.T) {
		assert.NotPanics(t, func() {
			notifier.NotifyOutOfSync("test-branch", "abc123d", "def456g")
		})
	})
	
	t.Run("NotifyBehindRemote", func(t *testing.T) {
		assert.NotPanics(t, func() {
			notifier.NotifyBehindRemote("test-branch", 3)
		})
	})
	
	t.Run("NotifyAutoPull", func(t *testing.T) {
		assert.NotPanics(t, func() {
			notifier.NotifyAutoPull("test-branch", 2)
		})
	})
	
	t.Run("NotifyConflicts", func(t *testing.T) {
		assert.NotPanics(t, func() {
			notifier.NotifyConflicts(2)
		})
	})
}

func TestConvertWSLPathToWindows(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("WSL path conversion only applies to Linux")
	}
	
	notifier := New()
	
	// We can't easily test this without actual WSL environment
	// but we can test that the method exists and handles errors
	_, err := notifier.convertWSLPathToWindows("/some/path")
	// This will likely fail in non-WSL environment, which is expected
	assert.Error(t, err)
}

func TestCheckDesktopNotificationSupport(t *testing.T) {
	tests := []struct {
		name        string
		goos        string
		expectTrue  bool
	}{
		{
			name:       "macOS should support notifications",
			goos:       "darwin", 
			expectTrue: true,
		},
		{
			name:       "Windows should support notifications",
			goos:       "windows",
			expectTrue: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.goos != runtime.GOOS {
				t.Skip("Test only applies to " + tt.goos)
			}
			
			result := checkDesktopNotificationSupport("/proc/version")
			if tt.expectTrue {
				assert.True(t, result)
			}
		})
	}
}
