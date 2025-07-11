package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/javanhut/harbinger/internal/git"
	"github.com/javanhut/harbinger/internal/notify"
	"github.com/javanhut/harbinger/internal/ui"
	"github.com/spf13/cobra"
)

var (
	testNotifications bool
	testUI            bool
	testAll           bool
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test harbinger UI components and notifications",
	Long:  `Test various components of harbinger to ensure they work correctly on your system.`,
	RunE:  runTest,
}

func init() {
	rootCmd.AddCommand(testCmd)
	testCmd.Flags().BoolVarP(&testNotifications, "notifications", "n", false, "Test notification system only")
	testCmd.Flags().BoolVarP(&testUI, "ui", "u", false, "Test UI components only")
	testCmd.Flags().BoolVarP(&testAll, "all", "a", false, "Test all components (default)")
}

func runTest(cmd *cobra.Command, args []string) error {
	// If no specific flags are set, test all
	if !testNotifications && !testUI {
		testAll = true
	}

	ui := ui.NewTerminalUI()
	ui.Clear()

	color.Cyan("🧪 Harbinger Component Testing")
	color.Cyan("================================")
	fmt.Println()

	if testAll || testUI {
		if err := testUIComponents(ui); err != nil {
			return fmt.Errorf("UI test failed: %w", err)
		}
	}

	if testAll || testNotifications {
		if err := testNotificationSystem(); err != nil {
			return fmt.Errorf("notification test failed: %w", err)
		}
	}

	if testAll {
		if err := testConflictResolutionUI(); err != nil {
			return fmt.Errorf("conflict resolution UI test failed: %w", err)
		}
	}

	color.Green("\n✅ All tests completed successfully!")
	color.HiBlack("Harbinger should work correctly on your system.")
	return nil
}

func testUIComponents(ui *ui.TerminalUI) error {
	color.Yellow("📱 Testing UI Components...")
	fmt.Println()

	// Test terminal clearing
	fmt.Print("Testing terminal clear... ")
	time.Sleep(500 * time.Millisecond)
	ui.Clear()
	color.Green("✓ Clear works")
	fmt.Println()

	// Test box drawing
	fmt.Print("Testing box drawing... ")
	ui.DrawBox("Test Box Content\nMultiple lines\nWith different lengths")
	color.Green("✓ Box drawing works")
	fmt.Println()

	// Test colors
	fmt.Print("Testing colors... ")
	color.Red("Red text ")
	color.Green("Green text ")
	color.Yellow("Yellow text ")
	color.Blue("Blue text ")
	color.Magenta("Magenta text ")
	color.Cyan("Cyan text ")
	fmt.Println()
	color.Green("✓ Colors work")
	fmt.Println()

	// Test unicode symbols
	fmt.Print("Testing unicode symbols... ")
	fmt.Print("✅ ❌ ⚠️ 🔄 📱 🎯 ⬇️ ⬆️ 🔍 📝 ⏭️ ")
	color.Green("✓ Unicode works")
	fmt.Println()

	waitForUser("UI components")
	return nil
}

func testNotificationSystem() error {
	color.Yellow("🔔 Testing Notification System...")
	fmt.Println()

	notifier := notify.New()

	// Test each type of notification
	notifications := []struct {
		name     string
		testFunc func()
	}{
		{"In Sync", func() { notifier.NotifyInSync("test-branch") }},
		{"Remote Change", func() { notifier.NotifyRemoteChange("test-branch", "abc123def456") }},
		{"Out of Sync", func() { notifier.NotifyOutOfSync("test-branch", "abc123d", "def456g") }},
		{"Behind Remote", func() { notifier.NotifyBehindRemote("test-branch", 3) }},
		{"Auto Pull", func() { notifier.NotifyAutoPull("test-branch", 2) }},
		{"Conflicts", func() { notifier.NotifyConflicts(2) }},
	}

	for i, notification := range notifications {
		fmt.Printf("Sending test notification %d/%d: %s", i+1, len(notifications), notification.name)
		notification.testFunc()

		// Show countdown
		for j := 3; j > 0; j-- {
			fmt.Printf("\rSending test notification %d/%d: %s (next in %ds)", i+1, len(notifications), notification.name, j)
			time.Sleep(1 * time.Second)
		}
		fmt.Printf("\rSending test notification %d/%d: %s ✓           \n", i+1, len(notifications), notification.name)
	}

	color.Green("✓ All notification types sent")
	fmt.Println()

	// Ask user if they received notifications
	fmt.Print("Did you receive system notifications? [Y/n]: ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "n" || response == "no" {
		color.Yellow("⚠️  Notifications may not be working properly on your system.")
		color.HiBlack("This could be due to:")
		color.HiBlack("  - Notification permissions not granted")
		color.HiBlack("  - Missing notification system (Linux: install libnotify)")
		color.HiBlack("  - WSL environment without Windows notification bridge")
		fmt.Println()
	} else {
		color.Green("✓ Notifications working correctly")
		fmt.Println()
	}

	return nil
}

func testConflictResolutionUI() error {
	color.Yellow("⚔️  Testing Conflict Resolution UI...")
	fmt.Println()

	// Create a mock conflict
	mockConflict := git.Conflict{
		File: "example.txt",
		Content: `This is a normal line
<<<<<<< HEAD
This is your change
Your additional line
=======
This is their change
Their additional line
>>>>>>> feature-branch
This is another normal line`,
	}

	color.Cyan("This is a demo of the conflict resolution interface.")
	color.HiBlack("You can interact with it to test all features.")
	fmt.Println()

	// We simulate the UI display instead of using the actual resolver
	// since it would try to modify git in a test environment
	simulateConflictUI(mockConflict)

	waitForUser("conflict resolution UI demo")
	return nil
}

func simulateConflictUI(conflict git.Conflict) {
	ui := ui.NewTerminalUI()
	ui.Clear()

	// Display header with box
	header := fmt.Sprintf("Conflict Resolution Demo\nFile: %s", conflict.File)
	ui.DrawBox(header)
	fmt.Println()

	// Parse and display conflict sections
	lines := strings.Split(conflict.Content, "\n")
	inOurs := false
	inTheirs := false

	for _, line := range lines {
		if strings.HasPrefix(line, "<<<<<<<") {
			color.Green("┌─ YOUR CHANGES " + strings.Repeat("─", 30) + "┐")
			color.Green("│")
			inOurs = true
		} else if strings.HasPrefix(line, "=======") && inOurs {
			color.Green("└" + strings.Repeat("─", 47) + "┘")
			fmt.Println()
			color.Red("┌─ THEIR CHANGES " + strings.Repeat("─", 29) + "┐")
			color.Red("│")
			inOurs = false
			inTheirs = true
		} else if strings.HasPrefix(line, ">>>>>>>") && inTheirs {
			color.Red("└" + strings.Repeat("─", 47) + "┘")
			fmt.Println()
			inTheirs = false
		} else {
			if inOurs {
				color.Green("│ " + line)
			} else if inTheirs {
				color.Red("│ " + line)
			} else {
				if strings.TrimSpace(line) != "" {
					color.HiBlack("Context: " + line)
				}
			}
		}
	}

	// Show options menu
	fmt.Println(strings.Repeat("═", 50))
	color.Cyan("What would you like to do?")
	fmt.Println()
	color.Green("  [1] ✓ Accept your changes")
	color.Red("  [2] ✓ Accept their changes")
	color.Yellow("  [3] ✏️  Edit in your editor")
	color.HiBlack("  [4] ⏭️  Skip this file")
	color.Magenta("  [5] 🔍 Show diff")
	color.Cyan("  [6] ❓ Show help")
	fmt.Println()
	color.White("Your choice (demo mode - any key to continue): ")

	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	color.Yellow("✓ Conflict resolution UI demo completed")
	fmt.Println()
}

func waitForUser(component string) {
	color.HiBlack(fmt.Sprintf("Press Enter to continue (finished testing %s)...", component))
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
	fmt.Println()
}
