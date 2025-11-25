package internal

import (
	"os"
	"path/filepath"
)

// GetConfigDir returns the configuration directory
func GetConfigDir() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "trigger"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "trigger"), nil
}

// NewConfig creates a configuration with proper directories
func NewConfig() (*Config, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	return &Config{
		ConfigDir:  configDir,
		TriggersDB: filepath.Join(configDir, "triggers.json"),
		ScriptsDir: filepath.Join(configDir, "scripts"),
	}, nil
}
