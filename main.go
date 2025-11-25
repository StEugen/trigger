package main

import (
	"fmt"
	"os"

	"github.com/steugen/trigger/cmd"
)

func main() {
	if err := cmd.Initialize(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
