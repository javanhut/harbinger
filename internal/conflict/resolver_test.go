package conflict

import (
	"testing"

	"github.com/javanhut/harbinger/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResolver(t *testing.T) {
	repo := &git.Repository{Path: "/test/path"}
	resolver := NewResolver(repo)
	
	assert.NotNil(t, resolver)
	assert.Equal(t, repo, resolver.repo)
}

func TestParseConflict(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		expectedSections int
		checkSections   func(t *testing.T, sections []ConflictSection)
	}{
		{
			name: "simple conflict",
			content: `line before conflict
<<<<<<< HEAD
our change
=======
their change
>>>>>>> branch
line after conflict`,
			expectedSections: 4, // normal, ours, theirs, normal
			checkSections: func(t *testing.T, sections []ConflictSection) {
				assert.Equal(t, "normal", sections[0].Type)
				assert.Contains(t, sections[0].Content, "line before conflict")
				
				assert.Equal(t, "ours", sections[1].Type)
				assert.Contains(t, sections[1].Content, "our change")
				
				assert.Equal(t, "theirs", sections[2].Type)
				assert.Contains(t, sections[2].Content, "their change")
				
				assert.Equal(t, "normal", sections[3].Type)
				assert.Contains(t, sections[3].Content, "line after conflict")
			},
		},
		{
			name: "conflict with no leading context",
			content: `<<<<<<< HEAD
our change only
=======
their change only
>>>>>>> branch`,
			expectedSections: 2,
			checkSections: func(t *testing.T, sections []ConflictSection) {
				assert.Equal(t, "ours", sections[0].Type)
				assert.Contains(t, sections[0].Content, "our change only")
				
				assert.Equal(t, "theirs", sections[1].Type)
				assert.Contains(t, sections[1].Content, "their change only")
			},
		},
		{
			name: "multiple conflicts",
			content: `first line
<<<<<<< HEAD
first conflict ours
=======
first conflict theirs
>>>>>>> branch
middle line
<<<<<<< HEAD
second conflict ours
=======
second conflict theirs
>>>>>>> branch
last line`,
			expectedSections: 7,
			checkSections: func(t *testing.T, sections []ConflictSection) {
				// Verify we have alternating pattern of normal, ours, theirs
				assert.Equal(t, "normal", sections[0].Type)
				assert.Equal(t, "ours", sections[1].Type)
				assert.Equal(t, "theirs", sections[2].Type)
				assert.Equal(t, "normal", sections[3].Type)
				assert.Equal(t, "ours", sections[4].Type)
				assert.Equal(t, "theirs", sections[5].Type)
				assert.Equal(t, "normal", sections[6].Type)
			},
		},
		{
			name: "no conflicts",
			content: `just normal content
no conflict markers here
regular file content`,
			expectedSections: 1,
			checkSections: func(t *testing.T, sections []ConflictSection) {
				assert.Equal(t, "normal", sections[0].Type)
				assert.Contains(t, sections[0].Content, "just normal content")
			},
		},
		{
			name: "empty content",
			content: "",
			expectedSections: 0,
			checkSections: func(t *testing.T, sections []ConflictSection) {
				// No sections expected for empty content
			},
		},
		{
			name: "conflict with empty sections",
			content: `<<<<<<< HEAD
=======
>>>>>>> branch`,
			expectedSections: 2,
			checkSections: func(t *testing.T, sections []ConflictSection) {
				assert.Equal(t, "ours", sections[0].Type)
				assert.Equal(t, "theirs", sections[1].Type)
				// Both should have minimal content (just newlines)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sections := parseConflict(tt.content)
			assert.Len(t, sections, tt.expectedSections)
			
			if tt.checkSections != nil {
				tt.checkSections(t, sections)
			}
		})
	}
}

func TestConflictSection(t *testing.T) {
	section := ConflictSection{
		Type:    "ours",
		Content: "test content",
	}

	assert.Equal(t, "ours", section.Type)
	assert.Equal(t, "test content", section.Content)
}

func TestParseConflict_ComplexExample(t *testing.T) {
	complexContent := `package main

import "fmt"

func main() {
<<<<<<< HEAD
	fmt.Println("Hello from feature branch")
	fmt.Println("Additional line in feature")
=======
	fmt.Println("Hello from main branch")
	fmt.Println("Different additional line")
>>>>>>> main
	
	// This part is not in conflict
	fmt.Println("Common ending")
}`

	sections := parseConflict(complexContent)
	
	// Should have: normal (before), ours, theirs, normal (after)
	require.Len(t, sections, 4)
	
	// Check the structure
	assert.Equal(t, "normal", sections[0].Type)
	assert.Contains(t, sections[0].Content, "package main")
	assert.Contains(t, sections[0].Content, "func main() {")
	
	assert.Equal(t, "ours", sections[1].Type)
	assert.Contains(t, sections[1].Content, "Hello from feature branch")
	assert.Contains(t, sections[1].Content, "Additional line in feature")
	
	assert.Equal(t, "theirs", sections[2].Type)
	assert.Contains(t, sections[2].Content, "Hello from main branch")
	assert.Contains(t, sections[2].Content, "Different additional line")
	
	assert.Equal(t, "normal", sections[3].Type)
	assert.Contains(t, sections[3].Content, "Common ending")
}

func TestParseConflict_EdgeCases(t *testing.T) {
	t.Run("malformed conflict markers", func(t *testing.T) {
		content := `<<<<<<< HEAD
our change
======= missing closing marker`
		
		sections := parseConflict(content)
		// Should still parse what it can
		assert.Greater(t, len(sections), 0)
	})
	
	t.Run("nested-like markers", func(t *testing.T) {
		content := `<<<<<<< HEAD
content with <<<<<<< in it
=======
content with >>>>>>> in it  
>>>>>>> branch`
		
		sections := parseConflict(content)
		assert.Len(t, sections, 2)
		assert.Equal(t, "ours", sections[0].Type)
		assert.Equal(t, "theirs", sections[1].Type)
	})
	
	t.Run("multiple equals lines", func(t *testing.T) {
		content := `<<<<<<< HEAD
our change
=======
=======
their change
>>>>>>> branch`
		
		sections := parseConflict(content)
		// The parser should handle this gracefully
		assert.Greater(t, len(sections), 0)
	})
}

func TestResolver_Integration(t *testing.T) {
	// Create a mock repository
	repo := &git.Repository{Path: t.TempDir()}
	resolver := NewResolver(repo)
	
	// Verify resolver was created properly
	assert.NotNil(t, resolver)
	assert.Equal(t, repo, resolver.repo)
	
	// Test that we can create conflicts
	conflict := git.Conflict{
		File: "test.txt",
		Content: `line 1
<<<<<<< HEAD
our change
=======
their change
>>>>>>> branch
line 2`,
	}
	
	// Parse the conflict
	sections := parseConflict(conflict.Content)
	assert.Greater(t, len(sections), 0)
	
	// Verify we have both ours and theirs sections
	hasOurs := false
	hasTheirs := false
	for _, section := range sections {
		if section.Type == "ours" {
			hasOurs = true
		}
		if section.Type == "theirs" {
			hasTheirs = true
		}
	}
	
	assert.True(t, hasOurs, "Should have 'ours' section")
	assert.True(t, hasTheirs, "Should have 'theirs' section")
}