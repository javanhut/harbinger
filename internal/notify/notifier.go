package notify

import (
	"bytes"
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
		useDesktopNotifications: checkDesktopNotificationSupport("/proc/version"),
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

func (n *Notifier) NotifyInSync(branch string) {
	title := "Branch In Sync"
	message := fmt.Sprintf("Branch '%s' is up to date with remote ✅", branch)

	n.sendNotification(title, message)
	log.Printf("✅ %s: %s", title, message)
}

func (n *Notifier) NotifyAutoPull(branch string, commitCount int) {
	title := "Auto-Pull Completed"
	message := fmt.Sprintf("Pulled %d commit(s) into branch '%s' ⬇️", commitCount, branch)

	n.sendNotification(title, message)
	log.Printf("⬇️ %s: %s", title, message)
}

func (n *Notifier) NotifyBehindRemote(branch string, commitCount int) {
	title := "Branch Behind Remote"
	message := fmt.Sprintf("Branch '%s' is %d commit(s) behind remote", branch, commitCount)

	n.sendNotification(title, message)
	log.Printf("⬆️ %s: %s", title, message)
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
		if isWSL("/proc/version") {
			n.sendWSLNotification(title, message)
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

func checkDesktopNotificationSupport(procVersionPath string) bool {
	switch runtime.GOOS {
	case "darwin":
		return true
	case "linux":
		// Check if notify-send is available or if running on WSL
		if isWSL(procVersionPath) {
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

// sendWSLNotification sends a notification through WSL to Windows
func (n *Notifier) sendWSLNotification(title, message string) {
	// Create the PowerShell script content
	scriptContent := fmt.Sprintf(`
param([string]$Title, [string]$Message)

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

$notify = New-Object System.Windows.Forms.NotifyIcon
$notify.Icon = [System.Drawing.SystemIcons]::Information
$notify.BalloonTipIcon = [System.Windows.Forms.ToolTipIcon]::Info
$notify.BalloonTipText = $Message
$notify.BalloonTipTitle = $Title
$notify.Visible = $true
$notify.ShowBalloonTip(5000)

# Keep the script running for a moment so the notification shows
Start-Sleep -Seconds 1
$notify.Dispose()
`)

	// Create temp directory for the script
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Error getting user home directory: %v", err)
		return
	}

	harbingerDir := filepath.Join(homeDir, ".harbinger")
	if err := os.MkdirAll(harbingerDir, 0755); err != nil {
		log.Printf("Error creating harbinger directory: %v", err)
		return
	}

	scriptPath := filepath.Join(harbingerDir, "notify.ps1")

	// Write the script to a temporary file
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		log.Printf("Error writing PowerShell script: %v", err)
		return
	}

	// Convert WSL path to Windows path for PowerShell
	windowsScriptPath, err := n.convertWSLPathToWindows(scriptPath)
	if err != nil {
		log.Printf("Error converting WSL path: %v", err)
		return
	}

	// Execute the PowerShell script with Windows paths
	cmd := exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-File", windowsScriptPath, "-Title", title, "-Message", message)
	if err := cmd.Run(); err != nil {
		log.Printf("Error executing PowerShell notification: %v", err)
	}
}

// convertWSLPathToWindows converts a WSL path to Windows path
func (n *Notifier) convertWSLPathToWindows(wslPath string) (string, error) {
	cmd := exec.Command("wslpath", "-w", wslPath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to convert WSL path: %w", err)
	}
	return string(bytes.TrimSpace(output)), nil
}

// isWSL checks if the current environment is Windows Subsystem for Linux
func isWSL(procVersionPath string) bool {
	if runtime.GOOS == "linux" {
		content, err := os.ReadFile(procVersionPath)
		if err != nil {
			return false
		}
		if bytes.Contains(content, []byte("microsoft")) || bytes.Contains(content, []byte("Microsoft")) {
			return true
		}
	}
	return false
}
