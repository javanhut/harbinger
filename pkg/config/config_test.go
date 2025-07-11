package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	// Reset global variables
	originalConfigPath := configPath
	originalConfigName := configName
	defer func() {
		configPath = originalConfigPath
		configName = originalConfigName
	}()

	// Clear config path to test defaults
	configPath = ""
	configName = ""

	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Test default values
	assert.Equal(t, "30s", cfg.PollInterval)
	assert.Equal(t, true, cfg.Notifications)
	assert.Equal(t, true, cfg.AutoResolve)
	assert.Equal(t, false, cfg.AutoPull) // Should default to false for safety
	assert.Nil(t, cfg.IgnoreBranches)    // Should be empty by default
}

func TestLoad_WithValidConfig(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ".harbinger.yaml")

	configContent := `poll_interval: 60s
editor: vim
notifications: false
auto_resolve: false
auto_pull: true
ignore_branches:
  - main
  - master
  - develop
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set config path and name
	originalConfigPath := configPath
	originalConfigName := configName
	defer func() {
		configPath = originalConfigPath
		configName = originalConfigName
	}()

	SetConfigPath(tmpDir)
	SetConfigName(".harbinger.yaml")

	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Test loaded values
	assert.Equal(t, "60s", cfg.PollInterval)
	assert.Equal(t, "vim", cfg.Editor)
	assert.Equal(t, false, cfg.Notifications)
	assert.Equal(t, false, cfg.AutoResolve)
	assert.Equal(t, true, cfg.AutoPull)
	assert.Equal(t, []string{"main", "master", "develop"}, cfg.IgnoreBranches)
}

func TestLoad_InvalidYAML(t *testing.T) {
	// Create temporary config file with invalid YAML
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ".harbinger.yaml")

	invalidYAML := `poll_interval: 60s
editor: vim
notifications: [invalid yaml structure
auto_resolve: false
`

	err := os.WriteFile(configFile, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	// Set config path and name
	originalConfigPath := configPath
	originalConfigName := configName
	defer func() {
		configPath = originalConfigPath
		configName = originalConfigName
	}()

	SetConfigPath(tmpDir)
	SetConfigName(".harbinger.yaml")

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_NonExistentConfig(t *testing.T) {
	// Set path to non-existent config
	originalConfigPath := configPath
	originalConfigName := configName
	defer func() {
		configPath = originalConfigPath
		configName = originalConfigName
	}()

	SetConfigPath("/non/existent/path")
	SetConfigName(".harbinger.yaml")

	cfg, err := Load()
	require.NoError(t, err) // Should not error, should use defaults
	assert.NotNil(t, cfg)

	// Should have default values
	assert.Equal(t, "30s", cfg.PollInterval)
	assert.Equal(t, true, cfg.Notifications)
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ".harbinger.yaml")

	// Set config path and name
	originalConfigPath := configPath
	originalConfigName := configName
	defer func() {
		configPath = originalConfigPath
		configName = originalConfigName
	}()

	SetConfigPath(tmpDir)
	SetConfigName(".harbinger.yaml")

	cfg := &Config{
		PollInterval:   "45s",
		Editor:         "code",
		Notifications:  true,
		AutoResolve:    false,
		AutoPull:       true,
		IgnoreBranches: []string{"main", "develop"},
	}

	err := Save(cfg)
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, configFile)

	// Load and verify content
	loadedCfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, cfg.PollInterval, loadedCfg.PollInterval)
	assert.Equal(t, cfg.Editor, loadedCfg.Editor)
	assert.Equal(t, cfg.Notifications, loadedCfg.Notifications)
	assert.Equal(t, cfg.AutoResolve, loadedCfg.AutoResolve)
	assert.Equal(t, cfg.AutoPull, loadedCfg.AutoPull)
	assert.Equal(t, cfg.IgnoreBranches, loadedCfg.IgnoreBranches)
}

func TestSave_NoConfigPath(t *testing.T) {
	// Reset global variables
	originalConfigPath := configPath
	originalConfigName := configName
	defer func() {
		configPath = originalConfigPath
		configName = originalConfigName
	}()

	configPath = ""
	configName = ""

	cfg := &Config{
		PollInterval: "30s",
	}

	err := Save(cfg)
	assert.NoError(t, err) // Should not error when no config path is set
}

func TestSetConfigFile(t *testing.T) {
	originalConfigPath := configPath
	originalConfigName := configName
	defer func() {
		configPath = originalConfigPath
		configName = originalConfigName
	}()

	testFile := "/path/to/config/.harbinger.yaml"
	SetConfigFile(testFile)

	assert.Equal(t, "/path/to/config", configPath)
	assert.Equal(t, ".harbinger.yaml", configName)
}

func TestSetConfigPath(t *testing.T) {
	originalConfigPath := configPath
	defer func() {
		configPath = originalConfigPath
	}()

	testPath := "/test/path"
	SetConfigPath(testPath)

	assert.Equal(t, testPath, configPath)
}

func TestSetConfigName(t *testing.T) {
	originalConfigName := configName
	defer func() {
		configName = originalConfigName
	}()

	testName := "test-config.yaml"
	SetConfigName(testName)

	assert.Equal(t, testName, configName)
}

func TestConfig_WithEnvironmentEditor(t *testing.T) {
	// Set EDITOR environment variable
	originalEditor := os.Getenv("EDITOR")
	defer func() {
		if originalEditor != "" {
			os.Setenv("EDITOR", originalEditor)
		} else {
			os.Unsetenv("EDITOR")
		}
	}()

	os.Setenv("EDITOR", "nano")

	// Reset global variables
	originalConfigPath := configPath
	originalConfigName := configName
	defer func() {
		configPath = originalConfigPath
		configName = originalConfigName
	}()

	configPath = ""
	configName = ""

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "nano", cfg.Editor)
}
