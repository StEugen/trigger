package internal

import (
	"regexp"
	"strconv"
	"strings"
)

var argPlaceholderPattern = regexp.MustCompile(`\[arg(\d+)\]`)

// ResolveArguments replaces [arg0], [arg1], etc. with runtime arguments
func ResolveArguments(args []string, runtimeArgs []string) []string {
	resolved := make([]string, len(args))
	copy(resolved, args)

	for i, arg := range resolved {
		if replacement, ok := resolveRuntimeArgPlaceholder(arg, runtimeArgs); ok {
			resolved[i] = replacement
		}
	}
	return resolved
}

// ResolveShellCommandLine replaces [argN] placeholders with shell-escaped runtime arguments.
func ResolveShellCommandLine(commandLine string, runtimeArgs []string) string {
	return argPlaceholderPattern.ReplaceAllStringFunc(commandLine, func(match string) string {
		replacement, ok := resolveRuntimeArgPlaceholder(match, runtimeArgs)
		if !ok {
			return match
		}

		return ShellQuote(replacement)
	})
}

// ShellQuote wraps values so they can be safely interpolated into `sh -c` command lines.
func ShellQuote(value string) string {
	if value == "" {
		return "''"
	}

	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func resolveRuntimeArgPlaceholder(value string, runtimeArgs []string) (string, bool) {
	if !strings.HasPrefix(value, "[arg") || !strings.HasSuffix(value, "]") {
		return "", false
	}

	idxStr := strings.TrimPrefix(strings.TrimSuffix(value, "]"), "[arg")
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 || idx >= len(runtimeArgs) {
		return "", false
	}

	return runtimeArgs[idx], true
}
