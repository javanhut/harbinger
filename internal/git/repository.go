package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Repository struct {
	Path string
}

func NewRepository(path string) (*Repository, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Verify it's a git repository
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = absPath
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("not a git repository: %w", err)
	}

	return &Repository{Path: absPath}, nil
}

func (r *Repository) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = r.Path
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (r *Repository) Fetch() error {
	cmd := exec.Command("git", "fetch", "--all")
	cmd.Dir = r.Path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}
	return nil
}

func (r *Repository) GetRemoteCommit(branch string) (string, error) {
	cmd := exec.Command("git", "rev-parse", fmt.Sprintf("origin/%s", branch))
	cmd.Dir = r.Path
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote commit: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (r *Repository) GetLocalCommit(branch string) (string, error) {
	cmd := exec.Command("git", "rev-parse", branch)
	cmd.Dir = r.Path
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get local commit: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (r *Repository) CheckForConflicts(targetBranch string) ([]Conflict, error) {
	// Perform a dry-run merge to check for conflicts
	cmd := exec.Command("git", "merge", "--no-commit", "--no-ff", targetBranch)
	cmd.Dir = r.Path

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if it's a merge conflict
		if strings.Contains(stderr.String(), "conflict") {
			conflicts, err := r.getConflictedFiles()
			if err != nil {
				return nil, err
			}

			// Abort the merge
			abortCmd := exec.Command("git", "merge", "--abort")
			abortCmd.Dir = r.Path
			if err := abortCmd.Run(); err != nil {
				return nil, fmt.Errorf("failed to abort merge: %w", err)
			}

			return conflicts, nil
		}
		return nil, fmt.Errorf("merge check failed: %w", err)
	}

	// No conflicts, abort the merge
	abortCmd := exec.Command("git", "merge", "--abort")
	abortCmd.Dir = r.Path
	if err := abortCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to abort merge: %w", err)
	}

	return nil, nil
}

func (r *Repository) getConflictedFiles() ([]Conflict, error) {
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=U")
	cmd.Dir = r.Path
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get conflicted files: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	conflicts := make([]Conflict, 0, len(lines))

	for _, file := range lines {
		if file == "" {
			continue
		}

		conflict, err := r.getFileConflict(file)
		if err != nil {
			return nil, err
		}
		conflicts = append(conflicts, *conflict)
	}

	return conflicts, nil
}

func (r *Repository) getFileConflict(file string) (*Conflict, error) {
	// Get the conflict markers from the file
	content, err := os.ReadFile(filepath.Join(r.Path, file))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return &Conflict{
		File:    file,
		Content: string(content),
	}, nil
}

func (r *Repository) GetConflictedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=U")
	cmd.Dir = r.Path
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get conflicted files: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, file := range lines {
		if file != "" {
			files = append(files, file)
		}
	}

	return files, nil
}

type Conflict struct {
	File    string
	Content string
}
