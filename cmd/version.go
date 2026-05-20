package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "1.1.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}
