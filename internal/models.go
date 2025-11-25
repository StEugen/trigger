package internal

import "time"

// Trigger represents a named command that can be saved and executed.
type Trigger struct {
	Name          string            `json:"name"`
	Command       string            `json:"command"`
	Args          []string          `json:"args,omitempty"`
	Description   string            `json:"description,omitempty"`
	Meta          map[string]string `json:"meta,omitempty"`
	ScriptContent string            `json:"script_content,omitempty"`
	ScriptPath    string            `json:"script_path,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
}

// Config holds application configuration
type Config struct {
	ConfigDir  string
	TriggersDB string
	ScriptsDir string
}
