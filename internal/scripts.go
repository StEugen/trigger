package internal

import (
	"os"
	"path/filepath"
	"strings"
)

var scriptExtensions = []string{
	".sh", ".py", ".js", ".rb", ".php", ".pl", ".lua", ".groovy", ".swift",
}

var scriptInterpreters = map[string]string{
	".sh":     "sh",
	".py":     "python3",
	".js":     "node",
	".rb":     "ruby",
	".php":    "php",
	".pl":     "perl",
	".lua":    "lua",
	".groovy": "groovy",
	".swift":  "swift",
}

// EmbeddedScriptPath returns the on-disk path used for an embedded trigger script.
func EmbeddedScriptPath(scriptsDir string, triggerName string, originalPath string) string {
	ext := filepath.Ext(originalPath)
	filename := triggerName + ext
	return filepath.Join(scriptsDir, filename)
}

// IsScriptFile checks if a file path is a script based on extension
func IsScriptFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, scriptExt := range scriptExtensions {
		if ext == scriptExt {
			return true
		}
	}
	return false
}

// HasShebang reports whether script content starts with a shebang line.
func HasShebang(content string) bool {
	return strings.HasPrefix(content, "#!")
}

// ScriptInterpreter returns the default interpreter for a known script extension.
func ScriptInterpreter(path string) (string, bool) {
	interpreter, ok := scriptInterpreters[strings.ToLower(filepath.Ext(path))]
	return interpreter, ok
}

// EmbedScript reads script content from disk
func EmbedScript(scriptPath string) (string, error) {
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// WriteEmbeddedScript writes embedded script to scripts directory and returns its path
func WriteEmbeddedScript(scriptsDir string, triggerName string, originalPath string, content string) (string, error) {
	fullPath := EmbeddedScriptPath(scriptsDir, triggerName, originalPath)

	if err := os.WriteFile(fullPath, []byte(content), 0o755); err != nil {
		return "", err
	}

	return fullPath, nil
}
