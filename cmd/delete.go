package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/steugen/trigger/internal"
)

var deleteCmd = &cobra.Command{
	Use:   "delete --name NAME",
	Short: "delete a trigger",
	Long: `Delete a trigger by name. This will remove the trigger from the triggers.json file
and delete any associated embedded scripts.

Examples:
  trigger delete --name backup
  trigger delete --name alert-slack
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return err
		}

		if name == "" {
			return fmt.Errorf("--name flag is required")
		}

		// Load storage
		storage, err := internal.NewStorage(GlobalConfig.ConfigDir)
		if err != nil {
			return err
		}

		// Find the trigger
		trigger, index, err := storage.FindByName(name)
		if err != nil {
			return fmt.Errorf("trigger '%s' not found", name)
		}

		// Delete associated script if it exists
		if trigger.ScriptPath != "" {
			scriptPath := filepath.Join(storage.ScriptsDir(), trigger.ScriptPath)
			if _, err := os.Stat(scriptPath); err == nil {
				if err := os.Remove(scriptPath); err != nil {
					return fmt.Errorf("failed to delete script: %w", err)
				}
				if GlobalVerbose {
					fmt.Printf("deleted embedded script '%s'\n", scriptPath)
				}
			}
		}

		// Load all triggers and remove the one we want to delete
		triggers, err := storage.LoadTriggers()
		if err != nil {
			return err
		}

		// Remove the trigger at the found index
		triggers = append(triggers[:index], triggers[index+1:]...)

		// Save updated triggers
		if err := storage.SaveTriggers(triggers); err != nil {
			return err
		}

		fmt.Printf("deleted trigger '%s'\n", name)
		return nil
	},
}

func init() {
	deleteCmd.Flags().StringP("name", "n", "", "name of the trigger to delete")
}
