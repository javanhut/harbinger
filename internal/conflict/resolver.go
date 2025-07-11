package conflict

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/javanhut/harbinger/internal/git"
	"github.com/javanhut/harbinger/internal/ui"
)

type Resolver struct {
	repo *git.Repository
}

func NewResolver(repo *git.Repository) *Resolver {
	return &Resolver{repo: repo}
}

func (r *Resolver) ResolveConflicts(conflicts []git.Conflict) error {
	ui := ui.NewTerminalUI()

	for i, conflict := range conflicts {
		if err := r.resolveConflict(ui, conflict, i+1, len(conflicts)); err != nil {
			return err
		}
	}

	color.Green("\n✅ All conflicts resolved!")
	return nil
}

func (r *Resolver) resolveConflict(ui *ui.TerminalUI, conflict git.Conflict, current, total int) error {
	ui.Clear()

	// Display header with box
	header := fmt.Sprintf("Conflict Resolution (%d/%d)\nFile: %s", current, total, conflict.File)
	ui.DrawBox(header)
	fmt.Println()

	// Parse and display conflict with better formatting
	sections := parseConflict(conflict.Content)

	for _, section := range sections {
		switch section.Type {
		case "ours":
			color.Green("┌─ YOUR CHANGES " + strings.Repeat("─", 30) + "┐")
			color.Green("│")
			lines := strings.Split(strings.TrimSpace(section.Content), "\n")
			for _, line := range lines {
				color.Green("│ " + line)
			}
			color.Green("└" + strings.Repeat("─", 47) + "┘")
			fmt.Println()
		case "theirs":
			color.Red("┌─ THEIR CHANGES " + strings.Repeat("─", 29) + "┐")
			color.Red("│")
			lines := strings.Split(strings.TrimSpace(section.Content), "\n")
			for _, line := range lines {
				color.Red("│ " + line)
			}
			color.Red("└" + strings.Repeat("─", 47) + "┘")
			fmt.Println()
		case "normal":
			// Show context lines in a muted color
			if strings.TrimSpace(section.Content) != "" {
				color.HiBlack("Context:")
				lines := strings.Split(strings.TrimSpace(section.Content), "\n")
				for _, line := range lines {
					color.HiBlack("  " + line)
				}
				fmt.Println()
			}
		}
	}

	// Show options in a nice menu
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
	color.White("Your choice: ")

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		return r.acceptOurs(conflict.File)
	case "2":
		return r.acceptTheirs(conflict.File)
	case "3":
		return r.editInEditor(conflict.File)
	case "4":
		color.Yellow("⏭️  Skipped %s\n", conflict.File)
		return nil
	case "5":
		r.showDiff(conflict.File)
		return r.resolveConflict(ui, conflict, current, total)
	case "6":
		r.showHelp()
		return r.resolveConflict(ui, conflict, current, total)
	default:
		color.Red("❌ Invalid choice. Please try again.")
		fmt.Println()
		return r.resolveConflict(ui, conflict, current, total)
	}
}

func (r *Resolver) acceptOurs(file string) error {
	cmd := exec.Command("git", "checkout", "--ours", file)
	cmd.Dir = r.repo.Path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to accept ours: %w", err)
	}

	// Stage the file
	cmd = exec.Command("git", "add", file)
	cmd.Dir = r.repo.Path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage file: %w", err)
	}

	color.Green("✓ Accepted your changes for %s\n", file)
	return nil
}

func (r *Resolver) acceptTheirs(file string) error {
	cmd := exec.Command("git", "checkout", "--theirs", file)
	cmd.Dir = r.repo.Path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to accept theirs: %w", err)
	}

	// Stage the file
	cmd = exec.Command("git", "add", file)
	cmd.Dir = r.repo.Path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage file: %w", err)
	}

	color.Green("✓ Accepted their changes for %s\n", file)
	return nil
}

func (r *Resolver) editInEditor(file string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		// Try common editors
		for _, e := range []string{"code", "vim", "nano", "vi"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
		if editor == "" {
			return fmt.Errorf("no editor found. Please set EDITOR environment variable")
		}
	}

	fullPath := filepath.Join(r.repo.Path, file)
	color.Yellow("🖊️  Opening %s in %s...\n", file, editor)

	cmd := exec.Command(editor, fullPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	// Ask if user wants to stage the file
	fmt.Print("\n🤔 Stage this file? [Y/n]: ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "" || response == "y" || response == "yes" {
		// Stage the file
		cmd = exec.Command("git", "add", file)
		cmd.Dir = r.repo.Path
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to stage file: %w", err)
		}
		color.Green("✓ Edited and staged %s\n", file)
	} else {
		color.Yellow("✏️  Edited %s (not staged)\n", file)
	}

	return nil
}

func (r *Resolver) showDiff(file string) {
	color.Cyan("\n🔍 Showing diff for %s:\n", file)
	cmd := exec.Command("git", "diff", file)
	cmd.Dir = r.repo.Path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	fmt.Println()
	color.HiBlack("Press Enter to continue...")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
}

func (r *Resolver) showHelp() {
	color.Cyan("\n📚 Conflict Resolution Help:\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("When Git finds conflicts, you have several options:")
	fmt.Println()
	color.Green("  ✓ Accept Yours:")
	fmt.Println("    Keep your changes and discard their changes")
	fmt.Println()
	color.Red("  ✓ Accept Theirs:")
	fmt.Println("    Keep their changes and discard your changes")
	fmt.Println()
	color.Yellow("  ✏️  Edit in Editor:")
	fmt.Println("    Open the file in your editor to manually resolve")
	fmt.Println("    Remove conflict markers and keep desired changes")
	fmt.Println()
	color.HiBlack("  ⏭️  Skip:")
	fmt.Println("    Leave this file unresolved for now")
	fmt.Println()
	color.Magenta("  🔍 Show Diff:")
	fmt.Println("    View the differences between versions")
	fmt.Println()
	color.HiBlack("Press Enter to continue...")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
}

type ConflictSection struct {
	Type    string // "ours", "theirs", "normal"
	Content string
}

func parseConflict(content string) []ConflictSection {
	lines := strings.Split(content, "\n")
	sections := []ConflictSection{}

	currentSection := ConflictSection{Type: "normal", Content: ""}
	inConflict := false

	for _, line := range lines {
		if strings.HasPrefix(line, "<<<<<<<") {
			if strings.TrimSpace(currentSection.Content) != "" {
				sections = append(sections, currentSection)
			}
			currentSection = ConflictSection{Type: "ours", Content: ""}
			inConflict = true
		} else if strings.HasPrefix(line, "=======") && inConflict {
			sections = append(sections, currentSection)
			currentSection = ConflictSection{Type: "theirs", Content: ""}
		} else if strings.HasPrefix(line, ">>>>>>>") && inConflict {
			sections = append(sections, currentSection)
			currentSection = ConflictSection{Type: "normal", Content: ""}
			inConflict = false
		} else {
			currentSection.Content += line + "\n"
		}
	}

	if strings.TrimSpace(currentSection.Content) != "" {
		sections = append(sections, currentSection)
	}

	return sections
}
