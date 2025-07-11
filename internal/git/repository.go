package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Repository struct {
	Path string
}

func NewRepository(path string) (*Repository, error) {
	// Validate input path
	if path == "" {
		return nil, fmt.Errorf("repository path cannot be empty")
	}

	// Sanitize path
	cleanPath := filepath.Clean(path)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Verify the path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("path does not exist: %s", absPath)
	}

	// Verify it's a git repository
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = absPath
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("not a git repository: %w", err)
	}

	return &Repository{Path: absPath}, nil
}

// validateBranchName validates that a branch name is safe to use in git commands
func validateBranchName(branch string) error {
	if branch == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Check for dangerous characters that could be used for command injection
	dangerousChars := regexp.MustCompile(`[;&|$(){}[\]<>'"\\]`)
	if dangerousChars.MatchString(branch) {
		return fmt.Errorf("branch name contains invalid characters: %s", branch)
	}

	// Check for git ref format requirements
	if strings.HasPrefix(branch, "/") || strings.HasSuffix(branch, "/") {
		return fmt.Errorf("branch name cannot start or end with '/': %s", branch)
	}

	if strings.Contains(branch, "..") {
		return fmt.Errorf("branch name cannot contain '..': %s", branch)
	}

	return nil
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
	if err := validateBranchName(branch); err != nil {
		return "", fmt.Errorf("invalid branch name: %w", err)
	}

	cmd := exec.Command("git", "rev-parse", fmt.Sprintf("origin/%s", branch))
	cmd.Dir = r.Path
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote commit: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (r *Repository) GetLocalCommit(branch string) (string, error) {
	if err := validateBranchName(branch); err != nil {
		return "", fmt.Errorf("invalid branch name: %w", err)
	}

	cmd := exec.Command("git", "rev-parse", branch)
	cmd.Dir = r.Path
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get local commit: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (r *Repository) CheckForConflicts(targetBranch string) ([]Conflict, error) {
	if err := validateBranchName(targetBranch); err != nil {
		return nil, fmt.Errorf("invalid target branch name: %w", err)
	}

	// Use git merge-tree to check for conflicts without modifying the working tree
	// This is available in git 2.38+, fallback to merge-base method for older versions

	// First, get the merge base
	currentBranch, err := r.GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	// Try using git merge-tree (non-destructive)
	cmd := exec.Command("git", "merge-tree", "--write-tree", "--name-only", currentBranch, targetBranch)
	cmd.Dir = r.Path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		// Check if merge-tree is not available (older git version)
		if strings.Contains(stderr.String(), "unknown option") || strings.Contains(stderr.String(), "usage:") {
			// Fallback to diff-based conflict detection
			return r.checkConflictsWithDiff(targetBranch)
		}

		// Check for conflicts in the output
		if strings.Contains(stdout.String(), "CONFLICT") {
			return r.parseConflictsFromMergeTree(stdout.String())
		}

		return nil, fmt.Errorf("merge-tree failed: %w", err)
	}

	// Check output for conflicts
	if strings.Contains(stdout.String(), "CONFLICT") {
		return r.parseConflictsFromMergeTree(stdout.String())
	}

	return nil, nil
}

// checkConflictsWithDiff uses a diff-based approach for older git versions
func (r *Repository) checkConflictsWithDiff(targetBranch string) ([]Conflict, error) {
	// Get the merge base
	cmd := exec.Command("git", "merge-base", "HEAD", targetBranch)
	cmd.Dir = r.Path
	mergeBase, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get merge base: %w", err)
	}
	mergeBaseStr := strings.TrimSpace(string(mergeBase))

	// Get files changed in both branches since merge base
	cmd = exec.Command("git", "diff", "--name-only", mergeBaseStr, "HEAD")
	cmd.Dir = r.Path
	ourFiles, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get our changed files: %w", err)
	}

	cmd = exec.Command("git", "diff", "--name-only", mergeBaseStr, targetBranch)
	cmd.Dir = r.Path
	theirFiles, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get their changed files: %w", err)
	}

	// Find files changed in both branches
	ourSet := make(map[string]bool)
	for _, file := range strings.Split(string(ourFiles), "\n") {
		if file != "" {
			ourSet[file] = true
		}
	}

	var potentialConflicts []string
	for _, file := range strings.Split(string(theirFiles), "\n") {
		if file != "" && ourSet[file] {
			potentialConflicts = append(potentialConflicts, file)
		}
	}

	// For each potentially conflicting file, check if the changes actually conflict
	var conflicts []Conflict
	for _, file := range potentialConflicts {
		// Get the three-way diff to see if there are actual conflicts
		cmd = exec.Command("git", "show", mergeBaseStr+":"+file)
		cmd.Dir = r.Path
		baseContent, _ := cmd.Output() // Ignore error if file doesn't exist in base

		cmd = exec.Command("git", "show", "HEAD:"+file)
		cmd.Dir = r.Path
		ourContent, _ := cmd.Output()

		cmd = exec.Command("git", "show", targetBranch+":"+file)
		cmd.Dir = r.Path
		theirContent, _ := cmd.Output()

		// Simple conflict detection: if both branches modified the same file differently
		if !bytes.Equal(ourContent, theirContent) &&
			(!bytes.Equal(ourContent, baseContent) && !bytes.Equal(theirContent, baseContent)) {
			conflicts = append(conflicts, Conflict{
				File:    file,
				Content: fmt.Sprintf("Potential conflict in %s\n", file),
			})
		}
	}

	return conflicts, nil
}

// parseConflictsFromMergeTree parses conflicts from git merge-tree output
func (r *Repository) parseConflictsFromMergeTree(output string) ([]Conflict, error) {
	var conflicts []Conflict
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(line, "CONFLICT") {
			// Extract filename from conflict message
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "in" && i+1 < len(parts) {
					filename := parts[i+1]
					conflicts = append(conflicts, Conflict{
						File:    filename,
						Content: line,
					})
					break
				}
			}
		}
	}

	return conflicts, nil
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

// IsInSync checks if the local branch is in sync with the remote
func (r *Repository) IsInSync(branch string) (bool, error) {
	if err := validateBranchName(branch); err != nil {
		return false, fmt.Errorf("invalid branch name: %w", err)
	}

	// Fetch latest remote information
	if err := r.Fetch(); err != nil {
		return false, fmt.Errorf("failed to fetch: %w", err)
	}

	localCommit, err := r.GetLocalCommit(branch)
	if err != nil {
		return false, err
	}

	remoteCommit, err := r.GetRemoteCommit(branch)
	if err != nil {
		// If remote branch doesn't exist, we're in sync (nothing to sync with)
		if strings.Contains(err.Error(), "unknown revision") {
			return true, nil
		}
		return false, err
	}

	return localCommit == remoteCommit, nil
}

// IsBehindRemote checks if the local branch is behind the remote
func (r *Repository) IsBehindRemote(branch string) (bool, int, error) {
	if err := validateBranchName(branch); err != nil {
		return false, 0, fmt.Errorf("invalid branch name: %w", err)
	}

	// Check how many commits we're behind
	cmd := exec.Command("git", "rev-list", "--count", fmt.Sprintf("%s..origin/%s", branch, branch))
	cmd.Dir = r.Path
	output, err := cmd.Output()
	if err != nil {
		// If the command fails, it might be because the remote branch doesn't exist
		if strings.Contains(err.Error(), "unknown revision") {
			return false, 0, nil
		}
		return false, 0, fmt.Errorf("failed to check if behind remote: %w", err)
	}

	count := 0
	countStr := strings.TrimSpace(string(output))
	if countStr != "" {
		count, _ = strconv.Atoi(countStr)
	}

	return count > 0, count, nil
}

// IsAheadOfRemote checks if the local branch is ahead of the remote
func (r *Repository) IsAheadOfRemote(branch string) (bool, int, error) {
	if err := validateBranchName(branch); err != nil {
		return false, 0, fmt.Errorf("invalid branch name: %w", err)
	}

	// Check how many commits we're ahead
	cmd := exec.Command("git", "rev-list", "--count", fmt.Sprintf("origin/%s..%s", branch, branch))
	cmd.Dir = r.Path
	output, err := cmd.Output()
	if err != nil {
		// If the command fails, it might be because the remote branch doesn't exist
		if strings.Contains(err.Error(), "unknown revision") {
			return false, 0, nil
		}
		return false, 0, fmt.Errorf("failed to check if ahead of remote: %w", err)
	}

	count := 0
	countStr := strings.TrimSpace(string(output))
	if countStr != "" {
		count, _ = strconv.Atoi(countStr)
	}

	return count > 0, count, nil
}

// HasUncommittedChanges checks if there are uncommitted changes
func (r *Repository) HasUncommittedChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = r.Path
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check status: %w", err)
	}

	return len(strings.TrimSpace(string(output))) > 0, nil
}

// Pull performs a git pull on the current branch
func (r *Repository) Pull() error {
	// First check if we have uncommitted changes
	hasChanges, err := r.HasUncommittedChanges()
	if err != nil {
		return err
	}
	if hasChanges {
		return fmt.Errorf("cannot pull: uncommitted changes in working directory")
	}

	cmd := exec.Command("git", "pull")
	cmd.Dir = r.Path

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull: %w - %s", err, stderr.String())
	}

	return nil
}

// GetRemoteName gets the remote name for the current branch
func (r *Repository) GetRemoteName(branch string) (string, error) {
	if err := validateBranchName(branch); err != nil {
		return "", fmt.Errorf("invalid branch name: %w", err)
	}

	cmd := exec.Command("git", "config", fmt.Sprintf("branch.%s.remote", branch))
	cmd.Dir = r.Path
	output, err := cmd.Output()
	if err != nil {
		return "origin", nil // Default to origin
	}
	return strings.TrimSpace(string(output)), nil
}
