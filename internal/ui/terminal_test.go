package ui

import (
	"io"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTerminalUI(t *testing.T) {
	ui := NewTerminalUI()
	assert.NotNil(t, ui)
	assert.IsType(t, &TerminalUI{}, ui)
}

func TestTerminalUI_Clear(t *testing.T) {
	ui := NewTerminalUI()

	// Capture stdout to verify clear command is executed
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// This should not panic
	assert.NotPanics(t, func() {
		ui.Clear()
	})

	w.Close()
	_, err := io.ReadAll(r)
	os.Stdout = oldStdout

	assert.NoError(t, err)
}

func TestTerminalUI_DrawBox(t *testing.T) {
	ui := NewTerminalUI()

	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:    "single line",
			content: "Hello World",
			expected: []string{
				"┌─────────────┐",
				"│ Hello World │",
				"└─────────────┘",
			},
		},
		{
			name:    "multiple lines",
			content: "Line 1\nLine 2\nLine 3",
			expected: []string{
				"┌────────┐",
				"│ Line 1 │",
				"│ Line 2 │",
				"│ Line 3 │",
				"└────────┘",
			},
		},
		{
			name:    "empty content",
			content: "",
			expected: []string{
				"┌──┐",
				"└──┘",
			},
		},
		{
			name:    "content with varying lengths",
			content: "Short\nMuch longer line\nMed",
			expected: []string{
				"┌──────────────────┐",
				"│ Short            │",
				"│ Much longer line │",
				"│ Med              │",
				"└──────────────────┘",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			ui.DrawBox(tt.content)

			w.Close()
			output, _ := io.ReadAll(r)
			os.Stdout = oldStdout

			lines := strings.Split(strings.TrimSpace(string(output)), "\n")

			assert.Equal(t, len(tt.expected), len(lines), "Number of output lines should match expected")

			for i, expectedLine := range tt.expected {
				if i < len(lines) {
					assert.Equal(t, expectedLine, lines[i], "Line %d should match", i+1)
				}
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "single line",
			content:  "Hello World",
			expected: []string{"Hello World"},
		},
		{
			name:     "multiple lines",
			content:  "Line 1\nLine 2\nLine 3",
			expected: []string{"Line 1", "Line 2", "Line 3"},
		},
		{
			name:     "empty string",
			content:  "",
			expected: nil, // splitLines returns nil for empty string
		},
		{
			name:     "string with trailing newline",
			content:  "Line 1\nLine 2\n",
			expected: []string{"Line 1", "Line 2"},
		},
		{
			name:     "string with only newline",
			content:  "\n",
			expected: []string{""},
		},
		{
			name:     "multiple consecutive newlines",
			content:  "Line 1\n\n\nLine 2",
			expected: []string{"Line 1", "", "", "Line 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLines(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRepeatStr(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		count    int
		expected string
	}{
		{
			name:     "repeat dash",
			str:      "-",
			count:    5,
			expected: "-----",
		},
		{
			name:     "repeat space",
			str:      " ",
			count:    3,
			expected: "   ",
		},
		{
			name:     "repeat unicode",
			str:      "─",
			count:    4,
			expected: "────",
		},
		{
			name:     "zero count",
			str:      "x",
			count:    0,
			expected: "",
		},
		{
			name:     "negative count",
			str:      "x",
			count:    -1,
			expected: "",
		},
		{
			name:     "repeat multi-char string",
			str:      "ab",
			count:    3,
			expected: "ababab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repeatStr(tt.str, tt.count)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTerminalUI_Integration(t *testing.T) {
	// Integration test to verify UI components work together
	ui := NewTerminalUI()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Perform multiple UI operations
	ui.Clear()
	ui.DrawBox("Test Box\nWith Multiple Lines")

	w.Close()
	output, _ := io.ReadAll(r)
	os.Stdout = oldStdout

	outputStr := string(output)

	// Verify clear was called (platform-specific behavior)
	if runtime.GOOS == "windows" {
		// On Windows, clear might not produce visible output in test
		assert.Contains(t, outputStr, "┌")
	} else {
		// On Unix-like systems, we expect some output
		assert.NotEmpty(t, outputStr)
	}

	// Verify box drawing
	assert.Contains(t, outputStr, "┌")
	assert.Contains(t, outputStr, "└")
	assert.Contains(t, outputStr, "│ Test Box")
}
