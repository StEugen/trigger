package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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

		if storage.Exists(name) {
			return fmt.Errorf("trigger '%s' already exists", name)
		}

		// Create trigger
		trigger := internal.Trigger{
			Name:      name,
			Command:   command,
			Args:      commandArgs,
			CreatedAt: time.Now().UTC(),
		}

		// Handle script embedding
		if internal.IsScriptFile(command) {
			if _, err := os.Stat(command); err == nil {
				content, err := internal.EmbedScript(command)
				if err != nil {
					return fmt.Errorf("failed to read script: %w", err)
				}

				scriptPath, err := internal.WriteEmbeddedScript(
					storage.ScriptsDir(),
					name,
					command,
					content,
				)
				if err != nil {
					return fmt.Errorf("failed to embed script: %w", err)
				}

				trigger.ScriptContent = content
				trigger.ScriptPath = filepath.Base(command)
				trigger.Command = scriptPath

				if GlobalVerbose {
					fmt.Printf("embedded script '%s' into trigger\n", command)
				}
			}
		}

		// Save
		triggers, err := storage.LoadTriggers()
		if err != nil {
			return err
		}

		triggers = append(triggers, trigger)
		if err := storage.SaveTriggers(triggers); err != nil {
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
