package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"myenv/internal/config"
	"myenv/internal/core"
	"myenv/internal/runner"
)

type result struct {
	Schema  int      `json:"schema"`
	OK      bool     `json:"ok"`
	Changed bool     `json:"changed"`
	Data    any      `json:"data"`
	Error   *failure `json:"error"`
}
type failure struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	NextAction string `json:"next_action"`
}

// Execute owns terminal output and exit status; application services stay independent.
func Execute(args []string, in io.Reader, out, diagnostic io.Writer, version string) int {
	return execute(args, in, out, diagnostic, version, "")
}

// The injected directory is for isolated integrations, never a CLI override of
// another user's namespace.
func execute(args []string, in io.Reader, out, diagnostic io.Writer, version, userConfigDirectory string) int {
	return executeContext(context.Background(), args, in, out, diagnostic, version, userConfigDirectory)
}

// ExecuteContext permits callers to cancel an operation and wait for cleanup.
func ExecuteContext(ctx context.Context, args []string, in io.Reader, out, diagnostic io.Writer, version string) int {
	return executeContext(ctx, args, in, out, diagnostic, version, "")
}

func executeContext(ctx context.Context, args []string, in io.Reader, out, diagnostic io.Writer, version, userConfigDirectory string) int {
	lang, localeErr := selectLocale(args)
	t := lang.text
	display := newProgressDisplay(diagnostic)
	defer display.close()
	var language string
	var dir string
	var jsonOutput, noInput, verbose bool
	var operationError bool
	var errorData any
	var errorChanged bool
	emit := func(data any, changed bool, message string) error {
		if jsonOutput {
			return json.NewEncoder(out).Encode(result{Schema: 1, OK: true, Changed: changed, Data: data})
		}
		display.mu.Lock()
		display.clearLocked()
		defer display.mu.Unlock()
		_, err := fmt.Fprintln(out, t(message))
		return err
	}
	root := &cobra.Command{Use: "myenv", Short: "Manage project development environments", Long: "myEnv manages project development environments.\n\nWorkflow: init → sync → run. Change versions with use; diagnose with doctor.\nDevelopment build: project Node/Python environments are implemented. Windows and native Linux runtime checks have passed; performance acceptance remains open. macOS is excluded from this delivery.", Version: version, SilenceErrors: true, SilenceUsage: true, Args: cobra.NoArgs}
	root.PersistentFlags().StringVarP(&dir, "directory", "C", ".", "Project context directory")
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Emit one structured JSON result")
	root.PersistentFlags().BoolVar(&noInput, "no-input", false, "Never wait for input")
	root.PersistentFlags().BoolVar(&verbose, "verbose", false, "Write diagnostic details to stderr")
	root.PersistentFlags().StringVar(&language, "lang", "auto", "Display language (auto, zh-CN or en)")
	var stopSignals context.CancelFunc
	var diagnosticStart time.Time
	var diagnosticCommand string

	defer func() {
		if stopSignals != nil {
			stopSignals()
		}
		if !diagnosticStart.IsZero() {
			fmt.Fprintf(diagnostic, "myenv diagnostic: command=%s elapsed=%s\n", diagnosticCommand, time.Since(diagnosticStart).Round(time.Microsecond))
		}
	}()
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if !jsonOutput && !noInput && !verbose && cmd.Name() != "run" && cmd.Name() != "shell-init" && terminalANSI(diagnostic) {
			diagnostic = display
			cmd.SetErr(display)
			cmd.SetOut(&progressOutput{display: display, out: out})
		}
		if verbose {
			diagnosticStart = time.Now()
			diagnosticCommand = cmd.Name()
			fmt.Fprintf(diagnostic, "myenv diagnostic: command=%s platform=%s/%s\n", diagnosticCommand, runtime.GOOS, runtime.GOARCH)
		}
		// run owns child signal forwarding in runner. Other operations need
		// cancellation so downloads/scans stop and preparation can unwind.
		if cmd.Name() != "run" {
			operationContext, stop := signal.NotifyContext(cmd.Context(), operationSignals()...)
			stopSignals = stop
			cmd.SetContext(operationContext)
		}
		return cmd.Context().Err()
	}
	root.RunE = func(cmd *cobra.Command, args []string) error {
		operationError = true
		status, err := (&core.Service{}).Status(cmd.Context(), dir)
		if errors.Is(err, config.ErrNoProject) {
			return emit(map[string]any{"project": nil}, false, "No myEnv project found. Start with myenv init, then myenv sync and myenv run <command>.")
		}
		if err != nil {
			return err
		}
		return emit(status, false, lang.status(status))
	}
	root.AddCommand(&cobra.Command{Use: "status", Short: "Show the current project environment", Args: cobra.NoArgs, RunE: root.RunE})
	initCmd := &cobra.Command{Use: "init", Short: "Detect declarations and create myenv.yaml without installing", Long: "Detect .python-version, pyproject.toml, .node-version, .nvmrc and package.json.\nCreate myenv.yaml in the selected directory without installation or overwriting.\nMissing or conflicting declarations return NEEDS_INPUT (exit 3).", Example: "  myenv init\n  myenv -C ./project init --no-input --json", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		operationError = true
		var selectVersion config.SelectVersion
		if !jsonOutput && !noInput && terminalInput(in, out) {
			selectVersion = versionPrompt(promptInput(cmd.Context(), in), diagnostic, lang)
		}
		c, changed, err := config.InitWithInput(dir, selectVersion)
		if err != nil {
			return err
		}
		message := "Existing myenv.yaml retained."
		if changed {
			message = "Created myenv.yaml. Next: myenv sync."
		}
		return emit(map[string]any{"tools": c.Tools, "path": filepath.Join(dir, "myenv.yaml")}, changed, message)
	}}
	root.AddCommand(initCmd)
	addShellInit(root, userConfigDirectory, &jsonOutput)
	addVersions(root, &jsonOutput)
	addSystem(root, userConfigDirectory, &jsonOutput, lang, func(data any, changed bool) { operationError = true; errorData = data; errorChanged = changed })
	var locked, dryRun, allowBuild, rebuild bool
	var global bool
	var update string
	buildConfirmation := func(ctx context.Context) func(string) (bool, error) {
		if !jsonOutput && !noInput && terminalInput(in, out) {
			return buildPrompt(promptInput(ctx, in), diagnostic, lang)
		}
		return nil
	}
	syncCmd := &cobra.Command{Use: "sync", Short: "Prepare and apply the declared environment", Long: "Prepare Node/Python/Java/Go/Rust tools and the selected Python project in a new environment, retaining the previous generation on failure.\n--locked requires matching runtime and native Python locks without rewriting them.\nBuild code requires --allow-build or confirmation for this invocation.\nDry-run does not install a backend; unresolved Python selectors are shown for resolution during sync.\nNative Linux runtime checks have passed; performance acceptance remains open. macOS is excluded from this delivery.", Example: "  myenv sync\n  myenv sync --locked --no-input --json", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		operationError = true
		fmt.Fprintln(diagnostic, t("Checking environment..."))
		service, err := runtimeService(global, userConfigDirectory)
		if err != nil {
			return err
		}
		defer closeRuntimeService(service)
		result, err := service.Sync(cmd.Context(), core.SyncRequest{Directory: dir, Locked: locked, DryRun: dryRun, Rebuild: rebuild, Update: update, AllowBuild: allowBuild, ConfirmBuild: buildConfirmation(cmd.Context()), Progress: syncProgress(diagnostic, lang)})
		if err != nil {
			errorData = result
			errorChanged = result.Changed || result.LockChanged || result.NativeLockChanged
			if result.LockChanged || result.NativeLockChanged {
				return fmt.Errorf("%w; lock_changed=%t native_lock_changed=%t; the prior active environment is retained; retry sync or run --current", err, result.LockChanged, result.NativeLockChanged)
			}
			return err
		}
		message := "Environment already up to date."
		if result.Changed {
			message = "Environment applied."
		}
		if result.Plan != nil {
			message = fmt.Sprintf(t("Preview on %s; environment preparation needed: %t."), result.Plan.Platform, result.Plan.NeedsApply)
			if result.Plan.Node.Version != "" {
				message += fmt.Sprintf(" Node %s.", result.Plan.Node.Version)
			}
			if result.Plan.Python != nil {
				message += fmt.Sprintf(" Python %s.", result.Plan.Python.Version)
			}
			if result.Plan.UnresolvedPython != "" {
				message += fmt.Sprintf(t(" Python %s requires resolution during sync."), result.Plan.UnresolvedPython)
			}
			if result.Plan.NativeLockNeedsUpdate {
				message += t(" Python native lock needs checking/updating.")
			}
		}
		return emit(result, result.Changed, message)
	}}
	syncCmd.Flags().BoolVar(&locked, "locked", false, "Require existing matching locks without modifying them")
	syncCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without installing or modifying locks and environment")
	syncCmd.Flags().BoolVar(&rebuild, "rebuild", false, "Prepare a fresh generation even when quick checks find no changes")
	syncCmd.Flags().StringVar(&update, "update", "", "Refresh the selected runtime resolution (node, python, java, go, rust)")
	syncCmd.Flags().BoolVar(&allowBuild, "allow-build", false, "Allow dependency and project build code for this invocation")
	syncCmd.Flags().BoolVar(&global, "global", false, "Use the current user's default tool profile")
	syncCmd.MarkFlagsMutuallyExclusive("locked", "update")
	root.AddCommand(syncCmd)
	var useAllowBuild bool
	var useProvider string
	useCmd := &cobra.Command{Use: "use [<tool>@<version>]", Short: "Change a runtime declaration and synchronize", Long: "Edit a supported runtime declaration and invoke the same sync workflow.\nWithout an argument, an interactive terminal prompts for tool@version.\nNon-interactive, --json and --no-input invocations require an explicit selection.\nIf preparation fails, the edited declaration remains and the previous active environment is retained.", Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && !jsonOutput && !noInput && terminalInput(in, out) {
			return nil
		}
		return cobra.ExactArgs(1)(cmd, args)
	}, RunE: func(cmd *cobra.Command, args []string) error {
		operationError = true
		if len(args) == 0 {
			tool, version, err := versionPrompt(promptInput(cmd.Context(), in), diagnostic, lang)("", "Select the runtime to change.")
			if cancelErr := cmd.Context().Err(); cancelErr != nil {
				return cancelErr
			}
			if err != nil {
				return err
			}
			args = []string{tool + "@" + version}
		}
		service, err := runtimeService(global, userConfigDirectory)
		if err != nil {
			return err
		}
		defer closeRuntimeService(service)
		fmt.Fprintln(diagnostic, t("Checking environment..."))
		selection, err := providerSelection(args[0], useProvider)
		if err != nil {
			return err
		}
		result, err := service.UseWithRequest(cmd.Context(), core.UseRequest{Directory: dir, Selection: selection, AllowBuild: useAllowBuild, ConfirmBuild: buildConfirmation(cmd.Context()), Progress: syncProgress(diagnostic, lang)})
		if err != nil {
			errorData = result
			errorChanged = result.DeclarationChanged || result.Changed || result.LockChanged || result.NativeLockChanged
			return err
		}
		return emit(result, result.DeclarationChanged || result.Changed, "Declaration synchronized with the applied environment.")
	}}
	useCmd.Flags().BoolVar(&useAllowBuild, "allow-build", false, "Allow dependency and project build code for this invocation")
	useCmd.Flags().StringVar(&useProvider, "provider", "", "Python 来源：python.org 或 astral；选择写入项目声明")
	useCmd.Flags().BoolVar(&global, "global", false, "Edit and synchronize the current user's default profile")
	root.AddCommand(useCmd)
	rollbackCmd := &cobra.Command{Use: "rollback", Short: "Switch to the previous retained environment", Long: "Switch to the previous complete generation without modifying declarations or locks.\nUse run --current to run it when current declarations differ.", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		operationError = true
		generation, err := (&core.Service{Profile: global, UserConfigDirectory: userConfigDirectory}).Rollback(cmd.Context(), dir)
		if err != nil {
			return err
		}
		return emit(generation, true, "Previous environment applied. Declarations and locks are retained; use myenv run --current <command>, or sync to apply current declarations.")
	}}
	rollbackCmd.Flags().BoolVar(&global, "global", false, "Roll back the current user's default profile")
	root.AddCommand(rollbackCmd)
	var deep bool
	doctorCmd := &cobra.Command{Use: "doctor", Short: "Inspect environment and command paths", Long: "Inspect local configuration, runtime locks, entries and PATH without modifying the environment.\nUse --deep to compare generation-owned files against the publication baseline; this may be slow.\nExternal runtime contents and ACLs are outside this content check.", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		operationError = true
		service := &core.Service{Profile: global, UserConfigDirectory: userConfigDirectory}
		check := service.Doctor
		if deep {
			check = service.DoctorDeep
		}
		report, err := check(cmd.Context(), dir, os.Environ())
		if err != nil {
			return err
		}
		message := fmt.Sprintf(t("Check level: %s\nEnvironment: %s"), t(report.CheckLevel), t(report.Environment))
		if report.ProtectionDetail != "" {
			message += "\n" + t(report.ProtectionDetail)
		}
		if deep {
			message += fmt.Sprintf(t("\nContent evidence: %s\nScope: %s\n%s"), t(report.ContentEvidence), t(report.EvidenceScope), t(report.EvidenceDetail))
		}
		if report.AppliedNode != "" {
			message += fmt.Sprintf(t("\nApplied Node: %s\nParent PATH Node: %s\nRun PATH Node: %s"), report.AppliedNode, report.ParentPathNode, report.RunPathNode)
		}
		if report.AppliedPython != "" {
			message += fmt.Sprintf(t("\nApplied Python: %s\nParent PATH Python: %s\nRun PATH Python: %s"), report.AppliedPython, report.ParentPathPython, report.RunPathPython)
		}
		return emit(report, false, message+"\n"+t(report.NextAction))
	}}
	doctorCmd.Flags().BoolVar(&global, "global", false, "Inspect the current user's default profile")
	doctorCmd.Flags().BoolVar(&deep, "deep", false, "Compare generation content with its recorded baseline (may be slow)")
	root.AddCommand(doctorCmd)
	root.AddCommand(cleanCommand(&dir, &jsonOutput, userConfigDirectory, lang))
	addHelp(root, &jsonOutput, lang)
	addRun(root, &dir, &jsonOutput, userConfigDirectory, lang)
	localizeHelp(root, lang)
	root.SetIn(in)
	root.SetOut(out)
	root.SetErr(diagnostic)
	root.SetArgs(args)
	root.CompletionOptions.DisableDefaultCmd = true
	executeRoot := func() error {
		if localeErr != nil {
			return localeErr
		}
		return root.ExecuteContext(ctx)
	}
	if err := executeRoot(); err != nil {
		var child *childExit
		if errors.As(err, &child) {
			return child.code
		}
		// Cobra can stop at an unknown flag before seeing a later --json.
		// Recover output intent only on failure; normal parsing remains authoritative.
		jsonOutput = errorJSONRequested(args, jsonOutput)
		f := &failure{Code: "USAGE_ERROR", Message: err.Error(), NextAction: "Run myenv --help."}
		exit := 2
		if operationError {
			f.Code = "INVALID_CONFIG"
			f.NextAction = "Check project input files and retry."
		}
		var pathErr *os.PathError
		var executionErr *runFailure
		if errors.As(err, &executionErr) {
			f.Code = "RUN_FAILED"
			f.NextAction = executionErr.nextAction()
			exit = 1
		}
		for _, code := range []string{"INPUT_CHANGED", "LOCK_OUT_OF_DATE", "DOWNLOAD_FAILED", "CHECKSUM_MISMATCH", "SYNC_FAILED", "ENV_NOT_READY"} {
			if strings.HasPrefix(err.Error(), code+":") {
				f.Code = code
				f.NextAction = "Check the reported cause and retry myenv sync; the prior applied generation is retained."
				exit = 1
			}
		}
		if errors.As(err, &pathErr) && f.Code != "ENV_NOT_READY" {
			f.Code = "IO_ERROR"
			f.NextAction = "Check that the project directory exists and is accessible, then retry."
			exit = 1
		}
		var needs *config.NeedsInput
		var environmentErr *environmentConfigError
		if errors.As(err, &environmentErr) {
			f.NextAction = environmentErr.nextAction()
		}
		if errors.As(err, &needs) {
			f.Code = "NEEDS_INPUT"
			f.NextAction = needs.Message
			exit = 3
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			f.Code = "CANCELED"
			f.NextAction = "Inspect current status before retrying; completed changes are retained."
			exit = 130
		}
		if executionErr != nil && errors.Is(err, runner.ErrTreeUnconfirmed) {
			f.NextAction = executionErr.nextAction()
		}
		if jsonOutput {
			_ = json.NewEncoder(out).Encode(result{Schema: 1, Error: f, Data: errorData, Changed: errorChanged})
		} else {
			lang.failure(diagnostic, f)
		}
		return exit
	}
	return 0
}

func errorJSONRequested(args []string, current bool) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if arg == "-C" || arg == "--directory" || arg == "--lang" {
			i++
			continue
		}
		if arg == "--json" || arg == "--json=true" {
			current = true
		}
		if arg == "--json=false" {
			current = false
		}
		if arg == "run" {
			// User argv after the first run command belongs to the child.
			for j := i + 1; j < len(args); j++ {
				if args[j] == "--" || !strings.HasPrefix(args[j], "-") {
					return current
				}
				if args[j] == "--json" || args[j] == "--json=true" {
					current = true
				}
				if args[j] == "--json=false" {
					current = false
				}
				if args[j] == "-C" || args[j] == "--directory" || args[j] == "--lang" {
					j++
				}
			}
			return current
		}
	}
	return current
}
