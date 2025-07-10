package ui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

type TerminalUI struct{}

func NewTerminalUI() *TerminalUI {
	return &TerminalUI{}
}

func (t *TerminalUI) Clear() {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default:
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func (t *TerminalUI) DrawBox(content string) {
	lines := splitLines(content)
	maxLen := 0
	for _, line := range lines {
		if len(line) > maxLen {
			maxLen = len(line)
		}
	}

	// Top border
	fmt.Printf("┌%s┐\n", repeatStr("─", maxLen+2))

	// Content
	for _, line := range lines {
		fmt.Printf("│ %-*s │\n", maxLen, line)
	}

	// Bottom border
	fmt.Printf("└%s┘\n", repeatStr("─", maxLen+2))
}

func splitLines(content string) []string {
	var lines []string
	current := ""
	for _, ch := range content {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func repeatStr(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
