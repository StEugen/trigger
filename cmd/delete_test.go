package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steugen/trigger/internal"
)

func TestDeleteCmdRemovesEmbeddedScript(t *testing.T) {
	configDir := t.TempDir()
	sourceScript := filepath.Join(t.TempDir(), "source-script.sh")

	if err := os.WriteFile(sourceScript, []byte("#!/bin/sh\necho hello\n"), 0o755); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	oldConfig := GlobalConfig
	oldDryRun := GlobalDryRun
	oldVerbose := GlobalVerbose
	t.Cleanup(func() {
		GlobalConfig = oldConfig
		GlobalDryRun = oldDryRun
		GlobalVerbose = oldVerbose
		_ = deleteCmd.Flags().Set("name", "")
	})

	GlobalConfig = &internal.Config{ConfigDir: configDir}
	GlobalDryRun = false
	GlobalVerbose = false

	if err := createCmd.RunE(createCmd, []string{"cleanup-test", "--", sourceScript}); err != nil {
		t.Fatalf("createCmd.RunE() error = %v", err)
	}

	storage, err := internal.NewStorage(configDir)
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	embeddedPath := internal.EmbeddedScriptPath(storage.ScriptsDir(), "cleanup-test", sourceScript)
	if _, err := os.Stat(embeddedPath); err != nil {
		t.Fatalf("embedded script missing before delete: %v", err)
	}

	if err := deleteCmd.Flags().Set("name", "cleanup-test"); err != nil {
		t.Fatalf("deleteCmd.Flags().Set() error = %v", err)
	}

	output := captureStdout(t, func() {
		if err := deleteCmd.RunE(deleteCmd, nil); err != nil {
			t.Fatalf("deleteCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "deleted trigger 'cleanup-test'") {
		t.Fatalf("unexpected output: %q", output)
	}

	if _, err := os.Stat(embeddedPath); !os.IsNotExist(err) {
		t.Fatalf("embedded script still exists after delete: %v", err)
	}

	triggers, err := storage.LoadTriggers()
	if err != nil {
		t.Fatalf("LoadTriggers() error = %v", err)
	}

	if len(triggers) != 0 {
		t.Fatalf("expected no triggers after delete, got %d", len(triggers))
	}
}
