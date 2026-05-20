package internal

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Storage manages trigger persistence
type Storage struct {
	triggersPath       string
	scriptsDir         string
	shellWhitelistPath string
}

type shellWhitelistFile struct {
	Commands []string `json:"commands"`
}

var defaultShellWhitelist = []string{"cd", "ls"}

// NewStorage creates a new storage instance
func NewStorage(configDir string) (*Storage, error) {
	triggersPath := filepath.Join(configDir, "triggers.json")
	scriptsDir := filepath.Join(configDir, "scripts")
	shellWhitelistPath := filepath.Join(configDir, "whitelist_shell.json")

	// Create directories if they don't exist
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(scriptsDir, 0o700); err != nil {
		return nil, err
	}

	storage := &Storage{
		triggersPath:       triggersPath,
		scriptsDir:         scriptsDir,
		shellWhitelistPath: shellWhitelistPath,
	}

	if err := storage.ensureShellWhitelist(); err != nil {
		return nil, err
	}

	return storage, nil
}

// LoadTriggers reads all triggers from storage
func (s *Storage) LoadTriggers() ([]Trigger, error) {
	if _, err := os.Stat(s.triggersPath); os.IsNotExist(err) {
		return []Trigger{}, nil
	}

	b, err := os.ReadFile(s.triggersPath)
	if err != nil {
		return nil, err
	}

	var triggers []Trigger
	if err := json.Unmarshal(b, &triggers); err != nil {
		return nil, err
	}

	return triggers, nil
}

// SaveTriggers writes triggers to storage
func (s *Storage) SaveTriggers(triggers []Trigger) error {
	b, err := json.MarshalIndent(triggers, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.triggersPath, b, 0o600)
}

// FindByName retrieves a trigger by name
func (s *Storage) FindByName(name string) (*Trigger, int, error) {
	triggers, err := s.LoadTriggers()
	if err != nil {
		return nil, -1, err
	}

	for i := range triggers {
		if triggers[i].Name == name {
			return &triggers[i], i, nil
		}
	}

	return nil, -1, errors.New("trigger not found")
}

// Exists checks if a trigger with the given name exists
func (s *Storage) Exists(name string) bool {
	_, _, err := s.FindByName(name)
	return err == nil
}

// ScriptsDir returns the directory where embedded scripts are stored
func (s *Storage) ScriptsDir() string {
	return s.scriptsDir
}

// ShellWhitelistPath returns the whitelist file used by the TUI shell.
func (s *Storage) ShellWhitelistPath() string {
	return s.shellWhitelistPath
}

// LoadShellWhitelist reads the list of commands allowed in the TUI shell.
func (s *Storage) LoadShellWhitelist() ([]string, error) {
	if _, err := os.Stat(s.shellWhitelistPath); os.IsNotExist(err) {
		return DefaultShellWhitelist(), nil
	}

	b, err := os.ReadFile(s.shellWhitelistPath)
	if err != nil {
		return nil, err
	}

	var payload shellWhitelistFile
	if err := json.Unmarshal(b, &payload); err == nil {
		return normalizeShellWhitelist(payload.Commands), nil
	}

	var commands []string
	if err := json.Unmarshal(b, &commands); err == nil {
		return normalizeShellWhitelist(commands), nil
	}

	return nil, errors.New("invalid shell whitelist format")
}

// SaveShellWhitelist persists the list of commands allowed in the TUI shell.
func (s *Storage) SaveShellWhitelist(commands []string) error {
	payload := shellWhitelistFile{
		Commands: normalizeShellWhitelist(commands),
	}

	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.shellWhitelistPath, b, 0o600)
}

// DefaultShellWhitelist returns the built-in command allowlist for the TUI shell.
func DefaultShellWhitelist() []string {
	return append([]string(nil), defaultShellWhitelist...)
}

func (s *Storage) ensureShellWhitelist() error {
	if _, err := os.Stat(s.shellWhitelistPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	return s.SaveShellWhitelist(DefaultShellWhitelist())
}

func normalizeShellWhitelist(commands []string) []string {
	seen := make(map[string]struct{}, len(commands))
	normalized := make([]string, 0, len(commands))

	for _, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}
		if _, ok := seen[command]; ok {
			continue
		}
		seen[command] = struct{}{}
		normalized = append(normalized, command)
	}

	if len(normalized) == 0 {
		return DefaultShellWhitelist()
	}

	return normalized
}
