package notify

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsWSL(t *testing.T) {
	if runtime.GOOS == "linux" {
		// Create a temporary directory for mock /proc/version files
		tmpDir, err := ioutil.TempDir("", "wsl_test")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Test case: Running on WSL
		wslProcVersionPath := filepath.Join(tmpDir, "proc_version_wsl")
		err = ioutil.WriteFile(wslProcVersionPath, []byte("Linux version 5.10.16.3-microsoft-standard-WSL2 (oe-user@oe-host) (GCC version 9.3.0 (Debian 9.3.0-17)) #1 SMP Fri Apr 2 22:23:43 UTC 2021"), 0644)
		assert.NoError(t, err)
		assert.True(t, isWSL(wslProcVersionPath), "Should detect WSL")

		// Test case: Not running on WSL
		nonWslProcVersionPath := filepath.Join(tmpDir, "proc_version_non_wsl")
		err = ioutil.WriteFile(nonWslProcVersionPath, []byte("Linux version 5.4.0-77-generic (buildd@lcy01-amd64-020) (gcc version 9.3.0 (Ubuntu 9.3.0-17ubuntu1~20.04)) #86-Ubuntu SMP Thu Jun 17 02:35:03 UTC 2021"), 0644)
		assert.NoError(t, err)
		assert.False(t, isWSL(nonWslProcVersionPath), "Should not detect WSL")

		// Test case: /proc/version does not exist
		assert.False(t, isWSL(filepath.Join(tmpDir, "non_existent_file")), "Should not detect WSL if file does not exist")

	} else {
		assert.False(t, isWSL(""), "Should not detect WSL on non-Linux OS")
	}
}

func TestSendNotification_WSL(t *testing.T) {
	if runtime.GOOS == "linux" && isWSL("/proc/version") {
		// This test requires a manual verification as it triggers a desktop notification.
		// It's hard to automate testing of desktop notifications.
		// You can run this test and observe if a notification appears on your Windows host.
		notifier := New()
		notifier.sendNotification("Test Title from WSL", "Test Message from WSL")
		// Add a small delay to allow the notification to appear
		// time.Sleep(2 * time.Second)
	}
}
