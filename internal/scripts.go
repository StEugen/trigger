package internal

import (
	"io/ioutil"
	"path/filepath"
	"strings"
)

var scriptExtensions = []string{
	".sh", ".py", ".js", ".rb", ".php", ".pl", ".lua", ".groovy", ".swift",
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

// EmbedScript reads script content from disk
func EmbedScript(scriptPath string) (string, error) {
	content, err := ioutil.ReadFile(scriptPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// WriteEmbeddedScript writes embedded script to scripts directory and returns its path
func WriteEmbeddedScript(scriptsDir string, triggerName string, originalPath string, content string) (string, error) {
	ext := filepath.Ext(originalPath)
	filename := triggerName + ext
	fullPath := filepath.Join(scriptsDir, filename)

	if err := ioutil.WriteFile(fullPath, []byte(content), 0o755); err != nil {
		return "", err
	}

	return fullPath, nil
}
