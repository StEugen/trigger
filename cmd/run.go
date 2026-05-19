package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/steugen/trigger/internal"
)

var (
	runName    string
	runPayload string
	runTimeout time.Duration
	runArgs    []string
)

var runCmd = &cobra.Command{
	Use:   "run --name NAME [--args arg0 arg1 ...] [--payload file]",
	Short: "run a named trigger",
	Long: `Run a registered trigger by name.

You can provide runtime arguments that will replace [arg0], [arg1], etc. placeholders.
If a payload file is provided, its contents will be piped to the command's stdin.

Examples:
  trigger run --name backup --args ./backup.tar.gz /etc
  trigger run --name alert-slack --payload message.json --timeout 30s
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if runName == "" {
			return fmt.Errorf("--name is required")
		}

		runtimeArgs := append([]string{}, runArgs...)
		runtimeArgs = append(runtimeArgs, args...)

		storage, err := internal.NewStorage(GlobalConfig.ConfigDir)
		if err != nil {
			return err
		}

		return internal.RunTrigger(storage, runName, runtimeArgs, internal.RunTriggerOptions{
			PayloadPath: runPayload,
			Timeout:     runTimeout,
			DryRun:      GlobalDryRun,
			Verbose:     GlobalVerbose,
			Stdout:      cmd.OutOrStdout(),
			Stderr:      cmd.ErrOrStderr(),
		})
	},
}

func init() {
	runCmd.Flags().StringVar(&runName, "name", "", "name of trigger to run")
	runCmd.Flags().StringSliceVar(&runArgs, "args", []string{}, "runtime arguments to replace [arg0], [arg1], etc.")
	runCmd.Flags().StringVar(&runPayload, "payload", "", "path to payload file to pass on stdin")
	runCmd.Flags().DurationVar(&runTimeout, "timeout", 0, "timeout for command (e.g. 30s)")
}
