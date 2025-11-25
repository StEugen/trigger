package cmd

import (
	"github.com/spf13/cobra"
	"github.com/steugen/trigger/internal"
)

// Global flags
var (
	GlobalDryRun  bool
	GlobalVerbose bool
)

var GlobalConfig *internal.Config

var rootCmd = &cobra.Command{
	Use:   "trigger",
	Short: "trigger — lightweight DevSecOps CLI for event->action workflows",
	Long:  "Trigger is a CLI to register and run named triggers (commands) with optional signing and dry-run support.",
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// Initialize sets up the CLI
func Initialize() error {
	var err error
	GlobalConfig, err = internal.NewConfig()
	if err != nil {
		return err
	}

	// Add persistent flags
	rootCmd.PersistentFlags().BoolVar(&GlobalDryRun, "dry-run", false, "don't execute commands; show what would run")
	rootCmd.PersistentFlags().BoolVarP(&GlobalVerbose, "verbose", "v", false, "verbose output")

	// Add subcommands
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(signCmd)
	rootCmd.AddCommand(versionCmd)

	// Set up completion with root cmd reference
	completionCmd.RunE = getCompletionRunE(rootCmd)
	rootCmd.AddCommand(completionCmd)

	return nil
}

// GetRootCmd returns the root command
func GetRootCmd() *cobra.Command {
	return rootCmd
}
