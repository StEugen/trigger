package internal

import (
	"strconv"
	"strings"
)

// ResolveArguments replaces [arg0], [arg1], etc. with runtime arguments
func ResolveArguments(args []string, runtimeArgs []string) []string {
	resolved := make([]string, len(args))
	copy(resolved, args)

	for i, arg := range resolved {
		// Check for [argN] format
		if strings.HasPrefix(arg, "[arg") && strings.HasSuffix(arg, "]") {
			idxStr := strings.TrimPrefix(strings.TrimSuffix(arg, "]"), "[arg")
			if idx, err := strconv.Atoi(idxStr); err == nil && idx < len(runtimeArgs) {
				resolved[i] = runtimeArgs[idx]
			}
		}
	}
	return resolved
}
