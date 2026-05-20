package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/steugen/trigger/internal"
)

const tuiWidth = 78

const (
	ansiReset       = "\033[0m"
	ansiBold        = "\033[1m"
	ansiDim         = "\033[2m"
	ansiGreen       = "\033[38;5;46m"
	ansiGreenBright = "\033[38;5;118m"
	ansiCyan        = "\033[38;5;51m"
	ansiYellow      = "\033[38;5;220m"
	ansiRed         = "\033[38;5;196m"
	ansiGray        = "\033[38;5;244m"
	ansiSilver      = "\033[38;5;252m"
	ansiBlackGreen  = "\033[30;48;5;46m"
	ansiBlackCyan   = "\033[30;48;5;51m"
	ansiBlackYellow = "\033[30;48;5;220m"
	ansiBlackRed    = "\033[30;48;5;196m"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "interactive terminal UI for local triggers",
	Long: `Open an interactive terminal UI for local triggers.

The TUI lets you:
  - list triggers stored on this machine
  - create a new trigger from a prompted command line
  - select and run an existing trigger
  - open a constrained shell with a JSON-backed command allowlist
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
	canStyle bool
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
		canStyle: canUseANSI(out),
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

		choice, err := ui.prompt("select option")
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
			} else if err := ui.pause("Press Enter to return to the console..."); err != nil {
				return err
			}
		case "2", "r", "run":
			if err := ui.runTrigger(sorted); err != nil {
				if pauseErr := ui.showError(err); pauseErr != nil {
					return pauseErr
				}
			} else if err := ui.pause("Press Enter to return to the console..."); err != nil {
				return err
			}
		case "3", "l", "list", "refresh", "":
			continue
		case "4", "s", "shell":
			if err := ui.runShell(); err != nil {
				if pauseErr := ui.showError(err); pauseErr != nil {
					return pauseErr
				}
			}
		case "5", "q", "quit", "exit":
			ui.clearScreen()
			fmt.Fprintln(ui.out, ui.note("[ session closed ]"))
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

	ui.printBanner()
	ui.printSection("CONTROL STATE")
	fmt.Fprintf(ui.out, "  %s  %s  %s\n",
		ui.badge("cache", fmt.Sprintf("%d loaded", len(triggers)), ansiBlackGreen),
		ui.badge("mode", ui.modeLabel(ui.dryRun), ui.modeColor(ui.dryRun)),
		ui.badge("verbose", onOff(ui.verbose), ui.toggleColor(ui.verbose)),
	)
	fmt.Fprintf(ui.out, "  %s %s\n\n", ui.label("config"), ui.paint(GlobalConfig.ConfigDir, ansiCyan))

	ui.printSection("LOCAL TRIGGER CACHE")
	if len(triggers) == 0 {
		fmt.Fprintf(ui.out, "  %s local stash is cold. forge something.\n", ui.paint("[empty]", ansiYellow, ansiBold))
	} else {
		for i, trigger := range triggers {
			fmt.Fprintln(ui.out, ui.formatTriggerLine(i, trigger))
		}
	}
	fmt.Fprintln(ui.out)

	ui.printSection("OPS MENU")
	fmt.Fprintf(ui.out, "  %s %s forge a new trigger into local storage\n", ui.actionLabel("1"), ui.paint("create", ansiBold, ansiGreenBright))
	fmt.Fprintf(ui.out, "  %s %s dispatch an existing trigger\n", ui.actionLabel("2"), ui.paint("run", ansiBold, ansiGreenBright))
	fmt.Fprintf(ui.out, "  %s %s reload trigger cache view\n", ui.actionLabel("3"), ui.paint("refresh", ansiBold, ansiGreenBright))
	fmt.Fprintf(ui.out, "  %s %s open the constrained local shell\n", ui.actionLabel("4"), ui.paint("shell", ansiBold, ansiGreenBright))
	fmt.Fprintf(ui.out, "  %s %s leave the console\n", ui.actionLabel("5"), ui.paint("quit", ansiBold, ansiGreenBright))
	fmt.Fprintln(ui.out)
	fmt.Fprintf(ui.out, "  %s %s\n", ui.paint("[ tip ]", ansiYellow, ansiBold), ui.note("pipes and redirects are stored as shell triggers automatically; shell allowlist lives in whitelist_shell.json."))
	fmt.Fprintln(ui.out)
}

func (ui *terminalUI) createTrigger() error {
	ui.clearScreen()
	ui.printBanner()
	ui.printSection("FORGE NEW TRIGGER")
	fmt.Fprintln(ui.out, "  Enter a trigger name and a full command line.")
	fmt.Fprintln(ui.out, "  Existing script paths are embedded automatically into the local stash.")
	fmt.Fprintln(ui.out)

	name, err := ui.promptRequired("trigger name")
	if err != nil {
		return err
	}

	commandLine, err := ui.promptRequired("command line")
	if err != nil {
		return err
	}

	var trigger *internal.Trigger
	if containsShellSyntax(commandLine) {
		trigger, err = internal.CreateShellTrigger(ui.storage, name, commandLine, internal.CreateTriggerOptions{
			Verbose: ui.verbose,
			Output:  ui.out,
		})
	} else {
		parts, err := splitCommandLine(commandLine)
		if err != nil {
			return err
		}
		if len(parts) == 0 {
			return fmt.Errorf("command line is required")
		}

		trigger, err = internal.CreateTrigger(ui.storage, name, parts[0], parts[1:], internal.CreateTriggerOptions{
			Verbose: ui.verbose,
			Output:  ui.out,
		})
	}
	if err != nil {
		return err
	}

	fmt.Fprintln(ui.out)
	if trigger.ScriptContent != "" {
		ui.printSuccess(fmt.Sprintf("trigger '%s' forged with embedded script payload", trigger.Name))
	} else if trigger.Shell {
		ui.printSuccess(fmt.Sprintf("trigger '%s' forged as a shell trigger", trigger.Name))
	} else {
		ui.printSuccess(fmt.Sprintf("trigger '%s' forged and indexed", trigger.Name))
	}
	fmt.Fprintf(ui.out, "  %s %s\n", ui.label("command"), ui.paint(ui.commandPreview(*trigger), ansiCyan))

	return nil
}

func (ui *terminalUI) runTrigger(triggers []internal.Trigger) error {
	if len(triggers) == 0 {
		ui.printWarning("no triggers registered")
		return nil
	}

	ui.clearScreen()
	ui.printBanner()
	ui.printSection("DISPATCH TRIGGER")
	fmt.Fprintln(ui.out, "  Pick a trigger by number or by exact name.")
	fmt.Fprintln(ui.out)
	for i, trigger := range triggers {
		fmt.Fprintln(ui.out, ui.formatTriggerLine(i, trigger))
	}
	fmt.Fprintln(ui.out)

	selection, err := ui.promptRequired("trigger number or name")
	if err != nil {
		return err
	}

	trigger, err := resolveTriggerSelection(triggers, selection)
	if err != nil {
		return err
	}

	argsLine, err := ui.prompt("runtime args (optional)")
	if err != nil {
		return err
	}

	runtimeArgs, err := splitCommandLine(argsLine)
	if err != nil {
		return err
	}

	payloadPath, err := ui.prompt("payload file path (optional)")
	if err != nil {
		return err
	}

	timeoutText, err := ui.prompt("timeout, e.g. 30s (optional)")
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

	dryRun, err := ui.promptYesNo("dry run", ui.dryRun)
	if err != nil {
		return err
	}

	fmt.Fprintln(ui.out)
	ui.printSection("EXECUTION TRACE")
	fmt.Fprintf(ui.out, "  %s %s\n", ui.label("target"), ui.paint(trigger.Name, ansiGreenBright, ansiBold))
	fmt.Fprintf(ui.out, "  %s %s\n", ui.label("command"), ui.paint(ui.commandPreview(*trigger), ansiCyan))
	fmt.Fprintf(ui.out, "  %s %d\n", ui.label("runtime args"), len(runtimeArgs))
	if payloadPath != "" {
		fmt.Fprintf(ui.out, "  %s %s\n", ui.label("payload"), ui.paint(payloadPath, ansiCyan))
	}
	if timeout > 0 {
		fmt.Fprintf(ui.out, "  %s %s\n", ui.label("timeout"), ui.paint(timeout.String(), ansiYellow))
	}
	fmt.Fprintf(ui.out, "  %s %s\n\n", ui.label("mode"), ui.paint(ui.modeLabel(dryRun), ui.modeTextColor(dryRun), ansiBold))

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
	fmt.Fprintf(ui.out, "%s %s %s ", ui.shellPrompt(), ui.label(label), ui.paint(">", ansiBold, ansiGreenBright))
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
		ui.printWarning("value is required")
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
	fmt.Fprintf(ui.errOut, "%s %s\n", ui.badge("fault", "execution aborted", ansiBlackRed), ui.paint(err.Error(), ansiRed, ansiBold))
	return ui.pause("Press Enter to return to the console...")
}

func (ui *terminalUI) pause(message string) error {
	fmt.Fprintln(ui.out)
	fmt.Fprintf(ui.out, "%s %s ", ui.label("[ idle ]"), ui.note(message))
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

func (ui *terminalUI) printBanner() {
	lines := []string{
		" _______  ____  ___   ____   ____  _____  ____",
		"|_   _\\ \\/ /  \\|_ _| / ___| / ___|| ____|/ ___|",
		"  | |  \\  /| |) || | | |  _  \\___ \\|  _|  \\___ \\",
		"  | |  /  \\|  _ < | | | |_| |  ___) | |___  ___) |",
		"  |_| /_/\\_\\_| \\_\\___| \\____| |____/|_____| |____/",
	}

	fmt.Fprintln(ui.out, ui.paint(strings.Repeat("=", tuiWidth), ansiGreen, ansiDim))
	for _, line := range lines {
		fmt.Fprintln(ui.out, ui.paint(line, ansiGreenBright, ansiBold))
	}
	fmt.Fprintf(ui.out, "%s %s\n", ui.badge("console", "local ops", ansiBlackCyan), ui.bannerTagline())
	fmt.Fprintln(ui.out, ui.paint(strings.Repeat("=", tuiWidth), ansiGreen, ansiDim))
	fmt.Fprintln(ui.out)
}

func (ui *terminalUI) printSection(title string) {
	header := "[ " + strings.ToUpper(title) + " ]"
	if len(header) < tuiWidth {
		header += strings.Repeat("-", tuiWidth-len(header))
	}
	fmt.Fprintln(ui.out, ui.paint(header, ansiGreen, ansiBold))
}

func (ui *terminalUI) printSuccess(message string) {
	fmt.Fprintf(ui.out, "%s %s\n", ui.badge("ok", "write complete", ansiBlackGreen), ui.paint(message, ansiGreenBright, ansiBold))
}

func (ui *terminalUI) printWarning(message string) {
	fmt.Fprintf(ui.out, "%s %s\n", ui.badge("warn", "attention", ansiBlackYellow), ui.paint(message, ansiYellow, ansiBold))
}

func (ui *terminalUI) actionLabel(id string) string {
	return ui.badge(id, "op", ansiBlackGreen)
}

func (ui *terminalUI) formatTriggerLine(index int, trigger internal.Trigger) string {
	meta := ui.badge(fmt.Sprintf("%02d", index+1), "idx", ansiBlackGreen)
	name := ui.paint(padRight(trigger.Name, 18), ansiCyan, ansiBold)
	preview := ui.note(shorten(ui.commandPreview(trigger), 42))
	line := fmt.Sprintf("  %s  %s :: %s", meta, name, preview)
	if trigger.Shell {
		line += " " + ui.badge("shell", "sh -c", ansiBlackCyan)
	}
	if trigger.ScriptContent != "" {
		line += " " + ui.badge("script", trigger.ScriptPath, ansiBlackYellow)
	}
	return line
}

func (ui *terminalUI) commandPreview(trigger internal.Trigger) string {
	if trigger.Shell {
		return trigger.CommandLine
	}
	parts := append([]string{trigger.Command}, trigger.Args...)
	return strings.Join(parts, " ")
}

func (ui *terminalUI) modeLabel(dryRun bool) string {
	if dryRun {
		return "simulate"
	}
	return "live"
}

func (ui *terminalUI) modeColor(dryRun bool) string {
	if dryRun {
		return ansiBlackYellow
	}
	return ansiBlackGreen
}

func (ui *terminalUI) modeTextColor(dryRun bool) string {
	if dryRun {
		return ansiYellow
	}
	return ansiGreenBright
}

func (ui *terminalUI) toggleColor(enabled bool) string {
	if enabled {
		return ansiBlackCyan
	}
	return ansiBlackYellow
}

func (ui *terminalUI) shellPrompt() string {
	return ui.paint("root@trigger", ansiGreenBright, ansiBold)
}

func (ui *terminalUI) label(text string) string {
	return ui.paint(text, ansiSilver, ansiBold)
}

func (ui *terminalUI) note(text string) string {
	return ui.paint(text, ansiSilver)
}

func (ui *terminalUI) bannerTagline() string {
	if !ui.canStyle {
		return "event->action relay // stash, forge, dispatch"
	}

	return ui.paint("event", ansiGreenBright, ansiBold) +
		ui.paint("->", ansiSilver, ansiBold) +
		ui.paint("action relay", ansiCyan, ansiBold) +
		ui.paint(" // ", ansiSilver) +
		ui.paint("stash", ansiYellow, ansiBold) +
		ui.paint(", ", ansiSilver) +
		ui.paint("forge", ansiGreenBright, ansiBold) +
		ui.paint(", ", ansiSilver) +
		ui.paint("dispatch", ansiCyan, ansiBold)
}

func (ui *terminalUI) badge(label string, value string, color string) string {
	raw := fmt.Sprintf(" %s:%s ", strings.ToUpper(label), value)
	if !ui.canStyle {
		return "[" + raw[1:len(raw)-1] + "]"
	}
	return ui.paint(raw, color, ansiBold)
}

func (ui *terminalUI) paint(text string, codes ...string) string {
	if !ui.canStyle || len(codes) == 0 {
		return text
	}
	return strings.Join(codes, "") + text + ansiReset
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

func canUseANSI(out io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

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

func (ui *terminalUI) runShell() error {
	whitelist, err := ui.storage.LoadShellWhitelist()
	if err != nil {
		return err
	}

	allowed := make(map[string]struct{}, len(whitelist))
	for _, command := range whitelist {
		allowed[command] = struct{}{}
	}

	ui.clearScreen()
	ui.printBanner()
	ui.printSection("TUI SHELL")
	if cwd, err := os.Getwd(); err == nil {
		fmt.Fprintf(ui.out, "  %s %s\n", ui.label("cwd"), ui.paint(cwd, ansiCyan))
	}
	fmt.Fprintf(ui.out, "  %s %s\n", ui.label("allowed"), ui.note(strings.Join(whitelist, ", ")))
	fmt.Fprintf(ui.out, "  %s %s\n", ui.label("config"), ui.note(ui.storage.ShellWhitelistPath()))
	fmt.Fprintf(ui.out, "  %s %s\n\n", ui.label("exit"), ui.note("type back, exit, or quit to return to the console"))

	for {
		line, err := ui.prompt("shell")
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Fprintln(ui.out)
				return nil
			}
			return err
		}
		if line == "" {
			continue
		}

		switch strings.ToLower(line) {
		case "back", "exit", "quit":
			return nil
		}

		if err := ui.executeShellCommand(line, allowed); err != nil {
			ui.printWarning(err.Error())
		}
	}
}

func (ui *terminalUI) executeShellCommand(line string, allowed map[string]struct{}) error {
	parts, err := splitCommandLine(line)
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		return nil
	}

	command := parts[0]
	if _, ok := allowed[command]; !ok {
		return fmt.Errorf("shell command %q is not whitelisted in %s", command, ui.storage.ShellWhitelistPath())
	}

	if command == "cd" {
		return changeDirectory(parts[1:])
	}

	cmd := exec.Command(command, parts[1:]...)
	cmd.Stdout = ui.out
	cmd.Stderr = ui.errOut
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func changeDirectory(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("cd accepts at most one path")
	}

	target := ""
	if len(args) == 0 {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		target = home
	} else {
		target = expandHomePath(args[0])
	}

	return os.Chdir(target)
}

func expandHomePath(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
		return path
	}

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return home + path[1:]
		}
	}

	return path
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

func containsShellSyntax(line string) bool {
	var (
		inSingle bool
		inDouble bool
		escaped  bool
	)

	for _, r := range line {
		switch {
		case escaped:
			escaped = false
		case r == '\\' && !inSingle:
			escaped = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case !inSingle && !inDouble:
			switch r {
			case '|', '>', '<', ';', '&':
				return true
			}
		}
	}

	return false
}

func onOff(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}

func shorten(value string, max int) string {
	if len(value) <= max || max < 4 {
		return value
	}
	return value[:max-3] + "..."
}

func padRight(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-len(value))
}
