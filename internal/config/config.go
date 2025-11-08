package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	DefaultMaxRetries   = 3
	DefaultBackoffBase  = 2.0
	DefaultDataDirPerms = 0755
)

type Config struct {
	MaxRetries   int     `json:"max_retries"`
	BackoffBase  float64 `json:"backoff_base"`
	DatabasePath string  `json:"-"` // Not stored in config file, but useful to have
}

var globalConfig *Config

func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(home, ".queuectl")
	return filepath.Join(configDir, "config.json"), nil
}

func getDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".queuectl"), nil
}

func Load() (*Config, error) {
	if globalConfig != nil {
		return globalConfig, nil
	}

	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	dataDir, err := getDataDir()
	if err != nil {
		return nil, err
	}

	// Default config
	cfg := &Config{
		MaxRetries:   DefaultMaxRetries,
		BackoffBase:  DefaultBackoffBase,
		DatabasePath: filepath.Join(dataDir, "jobs.db"),
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// If config doesn't exist, create the directory and we'll use defaults
		if err := os.MkdirAll(filepath.Dir(configPath), DefaultDataDirPerms); err != nil {
			return nil, err
		}
	} else {
		// If it exists, load it
		file, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(file, cfg); err != nil {
			return nil, err
		}
	}

	globalConfig = cfg
	return globalConfig, nil
}

func (c *Config) Save() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
