package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
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

		storage, err := internal.NewStorage(GlobalConfig.ConfigDir)
		if err != nil {
			return err
		}

		trigger, _, err := storage.FindByName(runName)
		if err != nil {
			return err
		}

		// Resolve argument placeholders
		resolvedArgs := internal.ResolveArguments(trigger.Args, runArgs)

		// Determine command to run
		commandToRun := trigger.Command
		if trigger.ScriptContent != "" {
			scriptPath, err := internal.WriteEmbeddedScript(
				storage.ScriptsDir(),
				trigger.Name,
				trigger.ScriptPath,
				trigger.ScriptContent,
			)
			if err != nil {
				return fmt.Errorf("failed to write embedded script: %w", err)
			}
			commandToRun = scriptPath
		}

		if GlobalDryRun {
			fmt.Printf("[dry-run] would run: %s %v\n", commandToRun, resolvedArgs)
			return nil
		}

		if GlobalVerbose {
			fmt.Printf("running: %s %v\n", commandToRun, resolvedArgs)
		}

		// Execute command
		ctxCmd := exec.Command(commandToRun, resolvedArgs...)

		if runPayload != "" {
			b, err := ioutil.ReadFile(runPayload)
			if err != nil {
				return err
			}

			stdin, err := ctxCmd.StdinPipe()
			if err != nil {
				return err
			}

			go func() {
				defer stdin.Close()
				io.WriteString(stdin, string(b))
			}()
		}

		ctxCmd.Stdout = os.Stdout
		ctxCmd.Stderr = os.Stderr

		if runTimeout > 0 {
			if err := ctxCmd.Start(); err != nil {
				return err
			}

			c := make(chan error)
			go func() { c <- ctxCmd.Wait() }()

			select {
			case err := <-c:
				return err
			case <-time.After(runTimeout):
				ctxCmd.Process.Kill()
				return fmt.Errorf("command timed out after %s", runTimeout)
			}
		}

		return ctxCmd.Run()
	},
}

func init() {
	runCmd.Flags().StringVar(&runName, "name", "", "name of trigger to run")
	runCmd.Flags().StringSliceVar(&runArgs, "args", []string{}, "runtime arguments to replace [arg0], [arg1], etc.")
	runCmd.Flags().StringVar(&runPayload, "payload", "", "path to payload file to pass on stdin")
	runCmd.Flags().DurationVar(&runTimeout, "timeout", 0, "timeout for command (e.g. 30s)")
}
