package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/javanhut/harbinger/internal/conflict"
	"github.com/javanhut/harbinger/internal/git"
	"github.com/spf13/cobra"
)

var resolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Manually resolve merge conflicts in the current repository",
	Long:  `Launch the interactive conflict resolution UI to manually resolve any merge conflicts in the current repository.`,
	RunE:  runResolve,
}

func init() {
	rootCmd.AddCommand(resolveCmd)
}

func runResolve(cmd *cobra.Command, args []string) error {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize repository
	repo, err := git.NewRepository(wd)
	if err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}

	// Check if we're in a merge state
	gitDir := filepath.Join(wd, ".git")
	mergeHeadPath := filepath.Join(gitDir, "MERGE_HEAD")
	if _, err := os.Stat(mergeHeadPath); os.IsNotExist(err) {
		fmt.Println("No merge conflicts detected. Repository is in a clean state.")
		return nil
	}

	// Find conflicted files
	conflicts, err := findConflictedFiles(repo)
	if err != nil {
		return fmt.Errorf("failed to find conflicted files: %w", err)
	}

	if len(conflicts) == 0 {
		fmt.Println("No conflicted files found.")
		return nil
	}

	fmt.Printf("Found %d conflicted file(s):\n", len(conflicts))
	for _, conflict := range conflicts {
		fmt.Printf("  - %s\n", conflict.File)
	}
	fmt.Println()

	// Launch conflict resolution UI
	resolver := conflict.NewResolver(repo)
	if err := resolver.ResolveConflicts(conflicts); err != nil {
		return fmt.Errorf("failed to resolve conflicts: %w", err)
	}

	return nil
}

func findConflictedFiles(repo *git.Repository) ([]git.Conflict, error) {
	// Get list of conflicted files from git status
	conflictedFiles, err := repo.GetConflictedFiles()
	if err != nil {
		return nil, err
	}

	var conflicts []git.Conflict
	for _, file := range conflictedFiles {
		// Read file content
		fullPath := filepath.Join(repo.Path, file)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", file, err)
		}

		// Check if file contains conflict markers
		if strings.Contains(string(content), "<<<<<<<") {
			conflicts = append(conflicts, git.Conflict{
				File:    file,
				Content: string(content),
			})
		}
	}

	return conflicts, nil
}
