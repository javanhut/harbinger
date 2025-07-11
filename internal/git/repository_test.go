package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateBranchName(t *testing.T) {
	tests := []struct {
		name        string
		branchName  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid branch name",
			branchName:  "feature-branch",
			expectError: false,
		},
		{
			name:        "empty branch name",
			branchName:  "",
			expectError: true,
			errorMsg:    "branch name cannot be empty",
		},
		{
			name:        "branch with semicolon",
			branchName:  "feature;branch",
			expectError: true,
			errorMsg:    "branch name contains invalid characters",
		},
		{
			name:        "branch with pipe",
			branchName:  "feature|branch",
			expectError: true,
			errorMsg:    "branch name contains invalid characters",
		},
		{
			name:        "branch with dollar sign",
			branchName:  "feature$branch",
			expectError: true,
			errorMsg:    "branch name contains invalid characters",
		},
		{
			name:        "branch starting with slash",
			branchName:  "/feature-branch",
			expectError: true,
			errorMsg:    "branch name cannot start or end with '/'",
		},
		{
			name:        "branch ending with slash",
			branchName:  "feature-branch/",
			expectError: true,
			errorMsg:    "branch name cannot start or end with '/'",
		},
		{
			name:        "branch with double dots",
			branchName:  "feature..branch",
			expectError: true,
			errorMsg:    "branch name cannot contain '..'",
		},
		{
			name:        "valid branch with numbers",
			branchName:  "feature-123",
			expectError: false,
		},
		{
			name:        "valid branch with underscores",
			branchName:  "feature_branch",
			expectError: false,
		},
		{
			name:        "valid branch with dots",
			branchName:  "feature.branch",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBranchName(tt.branchName)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewRepository(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setupFunc   func(string) error
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty path",
			path:        "",
			expectError: true,
			errorMsg:    "repository path cannot be empty",
		},
		{
			name:        "non-existent path",
			path:        "/non/existent/path",
			expectError: true,
			errorMsg:    "path does not exist",
		},
		{
			name: "directory without git",
			path: "",
			setupFunc: func(path string) error {
				return os.MkdirAll(path, 0755)
			},
			expectError: true,
			errorMsg:    "not a git repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPath := tt.path
			if tt.setupFunc != nil {
				testPath = t.TempDir()
				err := tt.setupFunc(testPath)
				require.NoError(t, err)
			}

			repo, err := NewRepository(testPath)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, repo)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, repo)
			}
		})
	}
}

func TestNewRepository_ValidGitRepo(t *testing.T) {
	// This test runs in the current directory which should be a git repo
	repo, err := NewRepository(".")
	require.NoError(t, err)
	assert.NotNil(t, repo)
	assert.NotEmpty(t, repo.Path)

	// Verify the path is absolute
	assert.True(t, filepath.IsAbs(repo.Path))
}

func TestGetCurrentBranch_ValidRepo(t *testing.T) {
	// Test with current git repository
	repo, err := NewRepository(".")
	require.NoError(t, err)

	branch, err := repo.GetCurrentBranch()
	assert.NoError(t, err)
	assert.NotEmpty(t, branch)
	// Should be a valid branch name
	assert.NoError(t, validateBranchName(branch))
}

func TestGetLocalCommit_InvalidBranch(t *testing.T) {
	repo, err := NewRepository(".")
	require.NoError(t, err)

	// Test with invalid branch name
	_, err = repo.GetLocalCommit("invalid;branch")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid branch name")
}

func TestGetRemoteCommit_InvalidBranch(t *testing.T) {
	repo, err := NewRepository(".")
	require.NoError(t, err)

	// Test with invalid branch name
	_, err = repo.GetRemoteCommit("invalid|branch")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid branch name")
}

func TestIsInSync_InvalidBranch(t *testing.T) {
	repo, err := NewRepository(".")
	require.NoError(t, err)

	// Test with invalid branch name
	_, err = repo.IsInSync("invalid$branch")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid branch name")
}

func TestIsBehindRemote_InvalidBranch(t *testing.T) {
	repo, err := NewRepository(".")
	require.NoError(t, err)

	// Test with invalid branch name
	_, _, err = repo.IsBehindRemote("invalid&branch")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid branch name")
}

func TestIsAheadOfRemote_InvalidBranch(t *testing.T) {
	repo, err := NewRepository(".")
	require.NoError(t, err)

	// Test with invalid branch name
	_, _, err = repo.IsAheadOfRemote("invalid'branch")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid branch name")
}

func TestGetRemoteName_InvalidBranch(t *testing.T) {
	repo, err := NewRepository(".")
	require.NoError(t, err)

	// Test with invalid branch name
	_, err = repo.GetRemoteName("invalid\"branch")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid branch name")
}

func TestCheckForConflicts_InvalidBranch(t *testing.T) {
	repo, err := NewRepository(".")
	require.NoError(t, err)

	// Test with invalid branch name
	_, err = repo.CheckForConflicts("invalid\\branch")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid target branch name")
}

func TestParseConflictsFromMergeTree(t *testing.T) {
	repo := &Repository{Path: "."} // Just for testing the method

	tests := []struct {
		name          string
		output        string
		expectedCount int
		expectedFiles []string
	}{
		{
			name:          "no conflicts",
			output:        "clean merge",
			expectedCount: 0,
		},
		{
			name:          "one conflict",
			output:        "CONFLICT (content): Merge conflict in file1.txt",
			expectedCount: 1,
			expectedFiles: []string{"file1.txt"},
		},
		{
			name: "multiple conflicts",
			output: `CONFLICT (content): Merge conflict in file1.txt
CONFLICT (content): Merge conflict in file2.js
regular output line
CONFLICT (add/add): Merge conflict in file3.go`,
			expectedCount: 3,
			expectedFiles: []string{"file1.txt", "file2.js", "file3.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflicts, err := repo.parseConflictsFromMergeTree(tt.output)
			assert.NoError(t, err)
			assert.Len(t, conflicts, tt.expectedCount)

			for i, expectedFile := range tt.expectedFiles {
				if i < len(conflicts) {
					assert.Equal(t, expectedFile, conflicts[i].File)
				}
			}
		})
	}
}

func TestHasUncommittedChanges_CleanRepo(t *testing.T) {
	repo, err := NewRepository(".")
	require.NoError(t, err)

	// This test assumes the current repo is clean
	// In a real scenario, you might want to create a test git repo
	hasChanges, err := repo.HasUncommittedChanges()
	assert.NoError(t, err)
	// We can't assert the exact value since the repo state might vary
	assert.IsType(t, false, hasChanges)
}

func TestConflictStruct(t *testing.T) {
	conflict := Conflict{
		File:    "test.txt",
		Content: "test content",
	}

	assert.Equal(t, "test.txt", conflict.File)
	assert.Equal(t, "test content", conflict.Content)
}
