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

	// Display header
	color.Cyan("=== Conflict Resolution (%d/%d) ===\n", current, total)
	color.Yellow("File: %s\n\n", conflict.File)

	// Parse and display conflict
	sections := parseConflict(conflict.Content)

	for _, section := range sections {
		switch section.Type {
		case "ours":
			color.Green("<<<<<<< YOURS\n")
			fmt.Print(section.Content)
			color.Green("\n")
		case "theirs":
			color.Red(">>>>>>> THEIRS\n")
			fmt.Print(section.Content)
			color.Red("\n")
		case "normal":
			fmt.Print(section.Content)
		}
	}

	// Show options
	fmt.Println("\n" + strings.Repeat("-", 50))
	fmt.Println("Choose an option:")
	fmt.Println("  [1] Accept yours")
	fmt.Println("  [2] Accept theirs")
	fmt.Println("  [3] Edit in your editor")
	fmt.Println("  [4] Skip this file")
	fmt.Print("\nYour choice: ")

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
		color.Yellow("Skipped %s\n", conflict.File)
		return nil
	default:
		color.Red("Invalid choice. Please try again.")
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
		editor = "vi" // Default to vi if no editor is set
	}

	fullPath := filepath.Join(r.repo.Path, file)
	cmd := exec.Command(editor, fullPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	// Stage the file
	cmd = exec.Command("git", "add", file)
	cmd.Dir = r.repo.Path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage file: %w", err)
	}

	color.Green("✓ Edited and staged %s\n", file)
	return nil
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
			if currentSection.Content != "" {
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

	if currentSection.Content != "" {
		sections = append(sections, currentSection)
	}

	return sections
}
