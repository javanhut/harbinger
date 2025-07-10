
param(
    [string]$Title,
    [string]$Message
)

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

$balloon = New-Object System.Windows.Forms.NotifyIcon
$path = Join-Path $PSScriptRoot "harbinger.ico" # Assuming an icon might be placed here later
if (-not (Test-Path $path)) {
    # Fallback to a default system icon if harbinger.ico doesn't exist
    $icon = [System.Drawing.SystemIcons]::Information
} else {
    $icon = New-Object System.Drawing.Icon($path)
}
$balloon.Icon = $icon
$balloon.BalloonTipTitle = $Title
$balloon.BalloonTipText = $Message
$balloon.Visible = $true
$balloon.ShowBalloonTip(3000) # Show for 3 seconds

Start-Sleep -Milliseconds 3100 # Keep script alive long enough for notification to show
$balloon.Dispose()
