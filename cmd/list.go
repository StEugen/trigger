package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steugen/trigger/internal"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list registered triggers",
	RunE: func(cmd *cobra.Command, args []string) error {
		storage, err := internal.NewStorage(GlobalConfig.ConfigDir)
		if err != nil {
			return err
		}

		triggers, err := storage.LoadTriggers()
		if err != nil {
			return err
		}

		if len(triggers) == 0 {
			fmt.Println("no triggers registered")
			return nil
		}

		for _, t := range triggers {
			if t.ScriptContent != "" {
				fmt.Printf("- %s: %s %v [embedded: %s]\n", t.Name, t.Command, t.Args, t.ScriptPath)
			} else {
				fmt.Printf("- %s: %s %v\n", t.Name, t.Command, t.Args)
			}
		}

		return nil
	},
}
