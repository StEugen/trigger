package internal

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// CreateTriggerOptions controls trigger creation behavior.
type CreateTriggerOptions struct {
	Verbose bool
	Output  io.Writer
	Now     func() time.Time
}

// RunTriggerOptions controls trigger execution behavior.
type RunTriggerOptions struct {
	PayloadPath string
	Timeout     time.Duration
	DryRun      bool
	Verbose     bool
	Stdout      io.Writer
	Stderr      io.Writer
}

// CreateTrigger creates and persists a trigger in storage.
func CreateTrigger(storage *Storage, name string, command string, commandArgs []string, opts CreateTriggerOptions) (*Trigger, error) {
	if storage.Exists(name) {
		return nil, fmt.Errorf("trigger '%s' already exists", name)
	}

	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}

	trigger := Trigger{
		Name:      name,
		Command:   command,
		Args:      commandArgs,
		CreatedAt: now().UTC(),
	}

	if IsScriptFile(command) {
		if _, err := os.Stat(command); err == nil {
			content, err := EmbedScript(command)
			if err != nil {
				return nil, fmt.Errorf("failed to read script: %w", err)
			}

			scriptPath, err := WriteEmbeddedScript(
				storage.ScriptsDir(),
				name,
				command,
				content,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to embed script: %w", err)
			}

			trigger.ScriptContent = content
			trigger.ScriptPath = filepath.Base(command)
			trigger.Command = scriptPath

			if opts.Verbose {
				fmt.Fprintf(outputOrDiscard(opts.Output), "embedded script '%s' into trigger\n", command)
			}
		}
	}

	triggers, err := storage.LoadTriggers()
	if err != nil {
		return nil, err
	}

	triggers = append(triggers, trigger)
	if err := storage.SaveTriggers(triggers); err != nil {
		return nil, err
	}

	return &trigger, nil
}

// RunTrigger resolves and executes a stored trigger.
func RunTrigger(storage *Storage, name string, runtimeArgs []string, opts RunTriggerOptions) error {
	trigger, _, err := storage.FindByName(name)
	if err != nil {
		return err
	}

	resolvedArgs := ResolveArguments(trigger.Args, runtimeArgs)

	commandToRun := trigger.Command
	commandArgs := append([]string{}, resolvedArgs...)
	if trigger.ScriptContent != "" {
		scriptPath, err := WriteEmbeddedScript(
			storage.ScriptsDir(),
			trigger.Name,
			trigger.ScriptPath,
			trigger.ScriptContent,
		)
		if err != nil {
			return fmt.Errorf("failed to write embedded script: %w", err)
		}
		commandToRun = scriptPath

		scriptName := trigger.ScriptPath
		if scriptName == "" {
			scriptName = scriptPath
		}

		if !HasShebang(trigger.ScriptContent) {
			interpreter, ok := ScriptInterpreter(scriptName)
			if !ok {
				return fmt.Errorf("embedded script '%s' requires a shebang or a known script extension", scriptName)
			}
			if _, err := exec.LookPath(interpreter); err != nil {
				return fmt.Errorf("interpreter '%s' not found for embedded script '%s': %w", interpreter, scriptName, err)
			}

			commandToRun = interpreter
			commandArgs = append([]string{scriptPath}, resolvedArgs...)
		}
	}

	stdout := outputOrStd(opts.Stdout, os.Stdout)
	stderr := outputOrStd(opts.Stderr, os.Stderr)

	if opts.DryRun {
		fmt.Fprintf(stdout, "[dry-run] would run: %s %v\n", commandToRun, commandArgs)
		return nil
	}

	if opts.Verbose {
		fmt.Fprintf(stdout, "running: %s %v\n", commandToRun, commandArgs)
	}

	ctxCmd := exec.Command(commandToRun, commandArgs...)

	if opts.PayloadPath != "" {
		b, err := os.ReadFile(opts.PayloadPath)
		if err != nil {
			return err
		}

		stdin, err := ctxCmd.StdinPipe()
		if err != nil {
			return err
		}

		go func() {
			defer stdin.Close()
			_, _ = io.WriteString(stdin, string(b))
		}()
	}

	ctxCmd.Stdout = stdout
	ctxCmd.Stderr = stderr

	if opts.Timeout > 0 {
		if err := ctxCmd.Start(); err != nil {
			return err
		}

		waitCh := make(chan error, 1)
		go func() {
			waitCh <- ctxCmd.Wait()
		}()

		select {
		case err := <-waitCh:
			return err
		case <-time.After(opts.Timeout):
			_ = ctxCmd.Process.Kill()
			return fmt.Errorf("command timed out after %s", opts.Timeout)
		}
	}

	return ctxCmd.Run()
}

// FormatTriggerSummary renders a single trigger in the same style as `trigger list`.
func FormatTriggerSummary(trigger Trigger) string {
	if trigger.ScriptContent != "" {
		return fmt.Sprintf("%s: %s %v [embedded: %s]", trigger.Name, trigger.Command, trigger.Args, trigger.ScriptPath)
	}

	return fmt.Sprintf("%s: %s %v", trigger.Name, trigger.Command, trigger.Args)
}

func outputOrDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}

	return w
}

func outputOrStd(w io.Writer, fallback *os.File) io.Writer {
	if w == nil {
		return fallback
	}

	return w
}
