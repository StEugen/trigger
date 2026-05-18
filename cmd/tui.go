package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/steugen/trigger/internal"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "interactive terminal UI for local triggers",
	Long: `Open an interactive terminal UI for local triggers.

The TUI lets you:
  - list triggers stored on this machine
  - create a new trigger from a prompted command line
  - select and run an existing trigger
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		storage, err := internal.NewStorage(GlobalConfig.ConfigDir)
		if err != nil {
			return err
		}

		ui := newTerminalUI(os.Stdin, cmd.OutOrStdout(), cmd.ErrOrStderr(), storage, GlobalDryRun, GlobalVerbose)
		return ui.Run()
	},
}

type terminalUI struct {
	reader   *bufio.Reader
	out      io.Writer
	errOut   io.Writer
	storage  *internal.Storage
	dryRun   bool
	verbose  bool
	canClear bool
}

func newTerminalUI(input io.Reader, out io.Writer, errOut io.Writer, storage *internal.Storage, dryRun bool, verbose bool) *terminalUI {
	return &terminalUI{
		reader:   bufio.NewReader(input),
		out:      out,
		errOut:   errOut,
		storage:  storage,
		dryRun:   dryRun,
		verbose:  verbose,
		canClear: canClearScreen(out),
	}
}

func (ui *terminalUI) Run() error {
	for {
		triggers, err := ui.storage.LoadTriggers()
		if err != nil {
			return err
		}

		sorted := sortTriggersByName(triggers)
		ui.renderHome(sorted)

		choice, err := ui.prompt("Select action")
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Fprintln(ui.out)
				return nil
			}
			return err
		}

		switch strings.ToLower(choice) {
		case "1", "c", "create":
			if err := ui.createTrigger(); err != nil {
				if pauseErr := ui.showError(err); pauseErr != nil {
					return pauseErr
				}
			} else if err := ui.pause("Press Enter to return to the menu..."); err != nil {
				return err
			}
		case "2", "r", "run":
			if err := ui.runTrigger(sorted); err != nil {
				if pauseErr := ui.showError(err); pauseErr != nil {
					return pauseErr
				}
			} else if err := ui.pause("Press Enter to return to the menu..."); err != nil {
				return err
			}
		case "3", "l", "list", "refresh", "":
			continue
		case "4", "q", "quit", "exit":
			return nil
		default:
			if err := ui.showError(fmt.Errorf("unknown action %q", choice)); err != nil {
				return err
			}
		}
	}
}

func (ui *terminalUI) renderHome(triggers []internal.Trigger) {
	ui.clearScreen()

	fmt.Fprintln(ui.out, "trigger tui")
	fmt.Fprintln(ui.out, "===========")
	fmt.Fprintln(ui.out)
	fmt.Fprintln(ui.out, "Local triggers:")
	if len(triggers) == 0 {
		fmt.Fprintln(ui.out, "  (none)")
	} else {
		for i, trigger := range triggers {
			fmt.Fprintf(ui.out, "  %d. %s\n", i+1, internal.FormatTriggerSummary(trigger))
		}
	}

	fmt.Fprintln(ui.out)
	fmt.Fprintln(ui.out, "Actions:")
	fmt.Fprintln(ui.out, "  1. Create trigger")
	fmt.Fprintln(ui.out, "  2. Run trigger")
	fmt.Fprintln(ui.out, "  3. Refresh list")
	fmt.Fprintln(ui.out, "  4. Quit")
	fmt.Fprintln(ui.out)
	if ui.dryRun {
		fmt.Fprintln(ui.out, "Run mode: dry-run enabled from global flag")
		fmt.Fprintln(ui.out)
	}
}

func (ui *terminalUI) createTrigger() error {
	ui.clearScreen()
	fmt.Fprintln(ui.out, "Create Trigger")
	fmt.Fprintln(ui.out, "==============")
	fmt.Fprintln(ui.out)

	name, err := ui.promptRequired("Trigger name")
	if err != nil {
		return err
	}

	commandLine, err := ui.promptRequired("Command line")
	if err != nil {
		return err
	}

	parts, err := splitCommandLine(commandLine)
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		return fmt.Errorf("command line is required")
	}

	trigger, err := internal.CreateTrigger(ui.storage, name, parts[0], parts[1:], internal.CreateTriggerOptions{
		Verbose: ui.verbose,
		Output:  ui.out,
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(ui.out)
	if trigger.ScriptContent != "" {
		fmt.Fprintf(ui.out, "Created trigger '%s' -> %s %v (script embedded)\n", trigger.Name, trigger.Command, trigger.Args)
	} else {
		fmt.Fprintf(ui.out, "Created trigger '%s' -> %s %v\n", trigger.Name, trigger.Command, trigger.Args)
	}

	return nil
}

func (ui *terminalUI) runTrigger(triggers []internal.Trigger) error {
	if len(triggers) == 0 {
		fmt.Fprintln(ui.out, "No triggers registered.")
		return nil
	}

	ui.clearScreen()
	fmt.Fprintln(ui.out, "Run Trigger")
	fmt.Fprintln(ui.out, "===========")
	fmt.Fprintln(ui.out)
	for i, trigger := range triggers {
		fmt.Fprintf(ui.out, "  %d. %s\n", i+1, internal.FormatTriggerSummary(trigger))
	}
	fmt.Fprintln(ui.out)

	selection, err := ui.promptRequired("Trigger number or name")
	if err != nil {
		return err
	}

	trigger, err := resolveTriggerSelection(triggers, selection)
	if err != nil {
		return err
	}

	argsLine, err := ui.prompt("Runtime args (optional)")
	if err != nil {
		return err
	}

	runtimeArgs, err := splitCommandLine(argsLine)
	if err != nil {
		return err
	}

	payloadPath, err := ui.prompt("Payload file path (optional)")
	if err != nil {
		return err
	}

	timeoutText, err := ui.prompt("Timeout, e.g. 30s (optional)")
	if err != nil {
		return err
	}

	timeout := time.Duration(0)
	if timeoutText != "" {
		timeout, err = time.ParseDuration(timeoutText)
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}
	}

	dryRun, err := ui.promptYesNo("Dry run", ui.dryRun)
	if err != nil {
		return err
	}

	fmt.Fprintln(ui.out)
	return internal.RunTrigger(ui.storage, trigger.Name, runtimeArgs, internal.RunTriggerOptions{
		PayloadPath: payloadPath,
		Timeout:     timeout,
		DryRun:      dryRun,
		Verbose:     ui.verbose,
		Stdout:      ui.out,
		Stderr:      ui.errOut,
	})
}

func (ui *terminalUI) prompt(label string) (string, error) {
	fmt.Fprintf(ui.out, "%s: ", label)
	line, err := ui.reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	line = strings.TrimSpace(line)
	if errors.Is(err, io.EOF) && line == "" {
		return "", io.EOF
	}

	return line, nil
}

func (ui *terminalUI) promptRequired(label string) (string, error) {
	for {
		value, err := ui.prompt(label)
		if err != nil {
			return "", err
		}
		if value != "" {
			return value, nil
		}
		fmt.Fprintln(ui.out, "Value is required.")
	}
}

func (ui *terminalUI) promptYesNo(label string, defaultValue bool) (bool, error) {
	suffix := "y/N"
	if defaultValue {
		suffix = "Y/n"
	}

	value, err := ui.prompt(fmt.Sprintf("%s [%s]", label, suffix))
	if err != nil {
		return false, err
	}

	switch strings.ToLower(value) {
	case "":
		return defaultValue, nil
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return false, fmt.Errorf("expected yes or no")
	}
}

func (ui *terminalUI) showError(err error) error {
	fmt.Fprintln(ui.errOut)
	fmt.Fprintf(ui.errOut, "Error: %v\n", err)
	return ui.pause("Press Enter to return to the menu...")
}

func (ui *terminalUI) pause(message string) error {
	fmt.Fprintln(ui.out)
	fmt.Fprint(ui.out, message)
	_, err := ui.reader.ReadString('\n')
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func (ui *terminalUI) clearScreen() {
	if ui.canClear {
		fmt.Fprint(ui.out, "\033[H\033[2J")
	}
}

func canClearScreen(out io.Writer) bool {
	file, ok := out.(*os.File)
	if !ok {
		return false
	}

	if os.Getenv("TERM") == "" || os.Getenv("TERM") == "dumb" {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

func sortTriggersByName(triggers []internal.Trigger) []internal.Trigger {
	sorted := append([]internal.Trigger(nil), triggers...)
	sort.Slice(sorted, func(i int, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}

func resolveTriggerSelection(triggers []internal.Trigger, selection string) (*internal.Trigger, error) {
	if index, err := strconv.Atoi(selection); err == nil {
		if index < 1 || index > len(triggers) {
			return nil, fmt.Errorf("trigger index %d is out of range", index)
		}
		return &triggers[index-1], nil
	}

	for i := range triggers {
		if triggers[i].Name == selection {
			return &triggers[i], nil
		}
	}

	return nil, fmt.Errorf("trigger '%s' not found", selection)
}

func splitCommandLine(line string) ([]string, error) {
	var (
		parts        []string
		current      strings.Builder
		inSingle     bool
		inDouble     bool
		escaped      bool
		tokenStarted bool
	)

	flush := func() {
		if !tokenStarted && current.Len() == 0 {
			return
		}
		parts = append(parts, current.String())
		current.Reset()
		tokenStarted = false
	}

	for _, r := range line {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
			tokenStarted = true
		case r == '\\' && !inSingle:
			escaped = true
			tokenStarted = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
			tokenStarted = true
		case r == '"' && !inSingle:
			inDouble = !inDouble
			tokenStarted = true
		case unicode.IsSpace(r) && !inSingle && !inDouble:
			flush()
		default:
			current.WriteRune(r)
			tokenStarted = true
		}
	}

	if escaped {
		current.WriteRune('\\')
	}

	if inSingle || inDouble {
		return nil, fmt.Errorf("unterminated quote")
	}

	flush()
	return parts, nil
}
