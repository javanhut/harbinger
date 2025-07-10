package notify

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type Notifier struct {
	useDesktopNotifications bool
}

func New() *Notifier {
	return &Notifier{
		useDesktopNotifications: checkDesktopNotificationSupport(),
	}
}

func (n *Notifier) NotifyRemoteChange(branch, commit string) {
	title := "Remote Branch Updated"
	message := fmt.Sprintf("Branch '%s' has new commits on remote\nLatest: %s", branch, commit[:7])

	n.sendNotification(title, message)
	log.Printf("🔄 %s: %s", title, message)
}

func (n *Notifier) NotifyOutOfSync(branch, localCommit, remoteCommit string) {
	title := "Branch Out of Sync"
	message := fmt.Sprintf("Branch '%s' is out of sync\nLocal: %s\nRemote: %s",
		branch, localCommit[:7], remoteCommit[:7])

	n.sendNotification(title, message)
	log.Printf("⚠️  %s: %s", title, message)
}

func (n *Notifier) NotifyConflicts(count int) {
	title := "Merge Conflicts Detected"
	message := fmt.Sprintf("Found %d potential merge conflicts that need resolution", count)

	n.sendNotification(title, message)
	log.Printf("❌ %s: %s", title, message)
}

func (n *Notifier) sendNotification(title, message string) {
	if !n.useDesktopNotifications {
		return
	}

	switch runtime.GOOS {
	case "darwin":
		// macOS notification
		script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
		exec.Command("osascript", "-e", script).Run()
	case "linux":
		// Linux notification (requires notify-send) or WSL notification
		if isWSL() {
			// For WSL, execute the PowerShell script on the Windows host
			// Get the absolute path to the PowerShell script from the user's home directory
			homeDir, err := os.UserHomeDir()
			if err != nil {
				log.Printf("Error getting user home directory: %v", err)
				return
			}
			scriptPath := filepath.Join(homeDir, ".harbinger", "notify.ps1")

			cmd := exec.Command("powershell.exe", "-File", scriptPath, "-Title", title, "-Message", message)
			cmd.Run()
		} else {
			exec.Command("notify-send", title, message).Run()
		}
	case "windows":
		// Windows notification (requires PowerShell)
		script := fmt.Sprintf(`
			[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
			[Windows.UI.Notifications.ToastNotification, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
			[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

			$template = @"
<toast>
	<visual>
		<binding template="ToastText02">
			<text id="1">%s</text>
			<text id="2">%s</text>
		</binding>
	</visual>
</toast>
"@

			$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
			$xml.LoadXml($template)
			$toast = New-Object Windows.UI.Notifications.ToastNotification $xml
			[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("Harbinger").Show($toast)
		`, title, message)
		exec.Command("powershell", "-Command", script).Run()
	}
}

func checkDesktopNotificationSupport() bool {
	switch runtime.GOOS {
	case "darwin":
		return true
	case "linux":
		// Check if notify-send is available or if running on WSL
		if isWSL() {
			return true // We will use PowerShell script for notifications on WSL
		}
		if err := exec.Command("which", "notify-send").Run(); err == nil {
			return true
		}
	case "windows":
		return true
	}
	return false
}

// isWSL checks if the current environment is Windows Subsystem for Linux
func isWSL() bool {
	if runtime.GOOS == "linux" {
		if _, err := exec.Command("grep", "-q", "microsoft", "/proc/version").Output(); err == nil {
			return true
		}
	}
	return false
}
