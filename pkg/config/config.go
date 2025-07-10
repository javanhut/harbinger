package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	PollInterval   string   `yaml:"poll_interval"`
	Editor         string   `yaml:"editor"`
	Notifications  bool     `yaml:"notifications"`
	IgnoreBranches []string `yaml:"ignore_branches"`
}

var (
	configPath string
	configName string
)

func SetConfigPath(path string) {
	configPath = path
}

func SetConfigName(name string) {
	configName = name
}

func SetConfigFile(file string) {
	configPath = filepath.Dir(file)
	configName = filepath.Base(file)
}

func Load() (*Config, error) {
	cfg := &Config{
		PollInterval:  "30s",
		Editor:        os.Getenv("EDITOR"),
		Notifications: true,
	}

	if configPath == "" || configName == "" {
		return cfg, nil
	}

	configFile := filepath.Join(configPath, configName)
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, use defaults
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func Save(cfg *Config) error {
	if configPath == "" || configName == "" {
		return nil
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	configFile := filepath.Join(configPath, configName)
	return os.WriteFile(configFile, data, 0644)
}
