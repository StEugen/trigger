package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/steugen/trigger/internal"
)

func TestRunCmdUsesAllRuntimeArgs(t *testing.T) {
	configDir := t.TempDir()
	storage, err := internal.NewStorage(configDir)
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	err = storage.SaveTriggers([]internal.Trigger{
		{
			Name:      "copy",
			Command:   "cp",
			Args:      []string{"[arg0]", "[arg1]"},
			CreatedAt: time.Now().UTC(),
		},
	})
	if err != nil {
		t.Fatalf("SaveTriggers() error = %v", err)
	}

	GlobalConfig = &internal.Config{ConfigDir: configDir}
	GlobalDryRun = true
	GlobalVerbose = false
	runName = "copy"
	runArgs = []string{"source.txt"}
	runPayload = ""
	runTimeout = 0

	output := captureStdout(t, func() {
		if err := runCmd.RunE(runCmd, []string{"dest.txt"}); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "[dry-run] would run: cp [source.txt dest.txt]") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunCmdAppendsExtraArgsWithoutPlaceholders(t *testing.T) {
	configDir := t.TempDir()
	storage, err := internal.NewStorage(configDir)
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	err = storage.SaveTriggers([]internal.Trigger{
		{
			Name:      "echo-all",
			Command:   "echo",
			Args:      []string{"first", "[arg0]", "[arg1]", "[arg2]"},
			CreatedAt: time.Now().UTC(),
		},
	})
	if err != nil {
		t.Fatalf("SaveTriggers() error = %v", err)
	}

	GlobalConfig = &internal.Config{ConfigDir: configDir}
	GlobalDryRun = true
	GlobalVerbose = false
	runName = "echo-all"
	runArgs = []string{"one"}
	runPayload = ""
	runTimeout = 0

	output := captureStdout(t, func() {
		if err := runCmd.RunE(runCmd, []string{"two", "three"}); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "[dry-run] would run: echo [first one two three]") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}

	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	outputCh := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, reader)
		outputCh <- buf.String()
	}()

	fn()

	_ = writer.Close()
	result := <-outputCh
	_ = reader.Close()

	return result
}
