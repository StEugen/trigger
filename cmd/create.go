package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steugen/trigger/internal"
)

var createShell bool

var createCmd = &cobra.Command{
	Use:   "create NAME [--shell] -- command [args]",
	Short: "create a named trigger",
	Long: `Create a named trigger with a command and optional arguments.

Supports:
  - Argument placeholders: use [arg0], [arg1], etc. in command args
  - Shell mode: store a raw command line to be executed by sh -c (useful for pipes and redirects)
  - Script embedding: if command is a script file (.sh, .py, .js, .rb, .php, .pl, .lua, .groovy, .swift),
    its content will be embedded into the trigger

Examples:
  trigger create backup -- tar -czf [arg0] /etc
  trigger create db-dump --shell -- "pg_dump [arg0] | zstd > [arg1]"
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

		if createShell {
			commandLine := extractShellCommandLine(args, sep)
			if commandLine == "" {
				return fmt.Errorf("usage: trigger create NAME --shell -- \"command line\"")
			}

			trigger, err := internal.CreateShellTrigger(storage, name, commandLine, internal.CreateTriggerOptions{
				Verbose: GlobalVerbose,
				Output:  cmd.OutOrStdout(),
			})
			if err != nil {
				return err
			}

			fmt.Printf("created shell trigger '%s' -> %s\n", name, trigger.CommandLine)
			return nil
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

func extractShellCommandLine(args []string, sep int) string {
	if sep >= 0 {
		if sep+1 >= len(args) {
			return ""
		}
		return strings.Join(args[sep+1:], " ")
	}
	if len(args) < 2 {
		return ""
	}
	return strings.Join(args[1:], " ")
}

func init() {
	createCmd.Flags().BoolVar(&createShell, "shell", false, "store the remaining command as a shell command line")
}
