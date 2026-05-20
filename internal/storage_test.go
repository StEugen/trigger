package internal

import (
	"os"
	"reflect"
	"testing"
)

func TestNewStorageCreatesDefaultShellWhitelist(t *testing.T) {
	storage, err := NewStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	if _, err := os.Stat(storage.ShellWhitelistPath()); err != nil {
		t.Fatalf("shell whitelist file missing: %v", err)
	}

	got, err := storage.LoadShellWhitelist()
	if err != nil {
		t.Fatalf("LoadShellWhitelist() error = %v", err)
	}

	want := []string{"cd", "ls"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadShellWhitelist() = %v, want %v", got, want)
	}
}

func TestLoadShellWhitelistSupportsArrayFormat(t *testing.T) {
	storage, err := NewStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	if err := os.WriteFile(storage.ShellWhitelistPath(), []byte(`["pwd","ls"]`), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	got, err := storage.LoadShellWhitelist()
	if err != nil {
		t.Fatalf("LoadShellWhitelist() error = %v", err)
	}

	want := []string{"pwd", "ls"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadShellWhitelist() = %v, want %v", got, want)
	}
}
