package internal

import "testing"

func TestResolveShellCommandLineQuotesRuntimeArgs(t *testing.T) {
	got := ResolveShellCommandLine("pg_dump [arg0] | zstd > [arg1]", []string{"db name", "dump file.sql.zst"})
	want := "pg_dump 'db name' | zstd > 'dump file.sql.zst'"

	if got != want {
		t.Fatalf("ResolveShellCommandLine() = %q, want %q", got, want)
	}
}

func TestShellQuoteEscapesSingleQuotes(t *testing.T) {
	got := ShellQuote("it's fine")
	want := `'it'"'"'s fine'`

	if got != want {
		t.Fatalf("ShellQuote() = %q, want %q", got, want)
	}
}
