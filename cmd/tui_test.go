package cmd

import (
	"reflect"
	"testing"

	"github.com/steugen/trigger/internal"
)

func TestSplitCommandLine(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:  "splits simple command",
			input: `echo hello world`,
			want:  []string{"echo", "hello", "world"},
		},
		{
			name:  "preserves quoted groups",
			input: `echo "hello world" 'two words'`,
			want:  []string{"echo", "hello world", "two words"},
		},
		{
			name:  "preserves empty quoted args",
			input: `printf "" ''`,
			want:  []string{"printf", "", ""},
		},
		{
			name:  "handles escaped spaces",
			input: `echo one\ two`,
			want:  []string{"echo", "one two"},
		},
		{
			name:    "rejects unterminated quote",
			input:   `echo "hello`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitCommandLine(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("splitCommandLine(%q) expected error", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("splitCommandLine(%q) error = %v", tt.input, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("splitCommandLine(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveTriggerSelection(t *testing.T) {
	triggers := []internal.Trigger{
		{Name: "alpha"},
		{Name: "beta"},
	}

	trigger, err := resolveTriggerSelection(triggers, "2")
	if err != nil {
		t.Fatalf("resolveTriggerSelection() error = %v", err)
	}
	if trigger.Name != "beta" {
		t.Fatalf("resolveTriggerSelection() selected %q, want beta", trigger.Name)
	}

	trigger, err = resolveTriggerSelection(triggers, "alpha")
	if err != nil {
		t.Fatalf("resolveTriggerSelection() by name error = %v", err)
	}
	if trigger.Name != "alpha" {
		t.Fatalf("resolveTriggerSelection() selected %q, want alpha", trigger.Name)
	}

	if _, err := resolveTriggerSelection(triggers, "3"); err == nil {
		t.Fatalf("resolveTriggerSelection() expected out-of-range error")
	}
}
