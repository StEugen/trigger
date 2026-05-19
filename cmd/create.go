package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steugen/trigger/internal"
)

var createCmd = &cobra.Command{
	Use:   "create NAME -- command [args]",
	Short: "create a named trigger",
	Long: `Create a named trigger with a command and optional arguments.

Supports:
  - Argument placeholders: use [arg0], [arg1], etc. in command args
  - Script embedding: if command is a script file (.sh, .py, .js, .rb, .php, .pl, .lua, .groovy, .swift),
    its content will be embedded into the trigger

Examples:
  trigger create backup -- tar -czf [arg0] /etc
  trigger create alert-slack -- ./send_slack_alert.sh
`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if len(args) < 2 {
			return fmt.Errorf("usage: trigger create NAME -- command [args]")
		}

		// Parse arguments
		sep := parseSeparator(args)
		command, commandArgs := extractCommand(args, sep)

		// Load and validate
		storage, err := internal.NewStorage(GlobalConfig.ConfigDir)
		if err != nil {
			return err
		}

		trigger, err := internal.CreateTrigger(storage, name, command, commandArgs, internal.CreateTriggerOptions{
			Verbose: GlobalVerbose,
			Output:  cmd.OutOrStdout(),
		})
		if err != nil {
			return err
		}

		if trigger.ScriptContent != "" {
			fmt.Printf("created trigger '%s' -> %s %v (script embedded)\n", name, trigger.Command, commandArgs)
		} else {
			fmt.Printf("created trigger '%s' -> %s %v\n", name, command, commandArgs)
		}

		return nil
	},
}

func parseSeparator(args []string) int {
	for i := 1; i < len(args); i++ {
		if args[i] == "--" {
			return i
		}
	}
	return -1
}

func extractCommand(args []string, sep int) (string, []string) {
	if sep >= 0 {
		if sep+1 >= len(args) {
			return "", nil
		}
		return args[sep+1], args[sep+2:]
	}
	return args[1], args[2:]
}
