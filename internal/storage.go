package internal

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Storage manages trigger persistence
type Storage struct {
	triggersPath string
	scriptsDir   string
}

// NewStorage creates a new storage instance
func NewStorage(configDir string) (*Storage, error) {
	triggersPath := filepath.Join(configDir, "triggers.json")
	scriptsDir := filepath.Join(configDir, "scripts")

	// Create directories if they don't exist
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(scriptsDir, 0o700); err != nil {
		return nil, err
	}

	return &Storage{
		triggersPath: triggersPath,
		scriptsDir:   scriptsDir,
	}, nil
}

// LoadTriggers reads all triggers from storage
func (s *Storage) LoadTriggers() ([]Trigger, error) {
	if _, err := os.Stat(s.triggersPath); os.IsNotExist(err) {
		return []Trigger{}, nil
	}

	b, err := ioutil.ReadFile(s.triggersPath)
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
	return ioutil.WriteFile(s.triggersPath, b, 0o600)
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
