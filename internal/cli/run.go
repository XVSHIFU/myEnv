package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"myenv/internal/backend"
	"myenv/internal/core"
	"myenv/internal/runner"
)

type childExit struct{ code int }

func (e *childExit) Error() string { return fmt.Sprintf("child exited with status %d", e.code) }

type runFailure struct {
	cause  error
	global bool
}

func (e *runFailure) Error() string { return e.cause.Error() }
func (e *runFailure) Unwrap() error { return e.cause }

func (e *runFailure) nextAction() string {
	doctor := "myenv doctor"
	if e.global {
		doctor += " --global"
	}
	if errors.Is(e.cause, runner.ErrTreeUnconfirmed) {
		return "Environment protection is retained because child-process completion is unconfirmed. Inspect remaining child processes and run " + doctor + " before retrying."
	}
	return "Inspect the reported execution error and run " + doctor + " before retrying."
}

func addRun(root *cobra.Command, dir *string, jsonOutput *bool, userConfigDirectory string, languages ...locale) {
	var lang locale
	if len(languages) > 0 {
		lang = languages[0]
	}
	var current, global, holdConsole bool
	var linkerMode string
	command := &cobra.Command{Use: "run [--current] [--] <command> [args...]", Short: "Run a command in the applied environment", Long: "Run with the applied environment without installation.\nArguments after command are preserved. Managed Windows Rust defaults to its bundled LLD when no linker is configured; use --rust-linker=system to disable this adjustment. --current permits declaration drift.\nManaged node/python/npm/npx use the selected generation. Other commands are resolved\nfrom the applied PATH; absolute and relative executable paths are also accepted.", Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			scope := ""
			if global {
				scope = " --global"
			}
			if lang {
				return fmt.Errorf("请在 run 后指定要运行的命令，例如：myenv run%s java -version", scope)
			}
			return fmt.Errorf("specify a command to run, for example: myenv run%s java -version", scope)
		}
		return nil
	}, RunE: func(cmd *cobra.Command, args []string) error {
		if *jsonOutput {
			return fmt.Errorf("--json is not supported for run")
		}
		err := runUseCase(cmd.Context(), RunRequest{Directory: *dir, Namespace: userConfigDirectory, Global: global, Current: current, Linker: linkerMode, Chinese: bool(lang), Stdin: cmd.InOrStdin(), Stdout: cmd.OutOrStdout(), Stderr: cmd.ErrOrStderr()}, args)
		// GUI handoff only: wait after Finish has released the run lease and
		// restored signal handlers. Ordinary CLI invocations never pause.
		if holdConsole && terminalInput(cmd.InOrStdin(), cmd.OutOrStdout()) {
			code := 0
			if err != nil {
				code = 1
				var child *childExit
				if errors.As(err, &child) {
					code = child.code
				}
				fmt.Fprintln(cmd.ErrOrStderr(), err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\n命令已结束，退出码：%d。按 Enter 关闭此终端。\n", code)
			_, _ = bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
		}
		return err
	}}
	command.Flags().BoolVar(&current, "current", false, "Use the applied generation even if declarations changed")
	command.Flags().BoolVar(&global, "global", false, "Run with the current user's default tool profile")
	command.Flags().BoolVar(&holdConsole, "hold-console", false, "Keep the GUI command result visible until Enter")
	_ = command.Flags().MarkHidden("hold-console")
	command.Flags().SetInterspersed(false)
	command.Flags().StringVar(&linkerMode, "rust-linker", "auto", "Windows 托管 Rust 链接器：auto（未配置时使用自带 LLD）、bundled、system")
	root.AddCommand(command)
}

// RunRequest freezes execution scope and I/O before terminal ownership changes.
type RunRequest struct {
	Directory, Namespace, Linker string
	Global, Current, Chinese     bool
	Stdin                        io.Reader
	Stdout, Stderr               io.Writer
}

// RunCommand is shared by CLI and TUI, in the current process. The caller must
// release its terminal and signal handlers before calling it.
func RunCommand(ctx context.Context, request RunRequest, args []string) (int, error) {
	if len(args) == 0 {
		return 2, fmt.Errorf("请指定运行命令")
	}
	err := runUseCase(ctx, request, args)
	var child *childExit
	if errors.As(err, &child) {
		return child.code, nil
	}
	if err != nil {
		return 1, err
	}
	return 0, nil
}
func runUseCase(ctx context.Context, request RunRequest, args []string) error {
	service := core.Service{Profile: request.Global, UserConfigDirectory: request.Namespace}
	selected, err := service.SelectRun(ctx, request.Directory, request.Current)
	if err != nil {
		return err
	}
	var runErr error
	defer func() {
		if releaseErr := selected.Finish(runErr); releaseErr != nil {
			fmt.Fprintf(request.Stderr, locale(request.Chinese).text("IO_ERROR: could not finalize process lease: %v\n"), releaseErr)
		}
	}()
	cwd, err := filepath.Abs(request.Directory)
	if err != nil {
		return err
	}
	executable := selected.Generation.NodeExecutable
	argv := args[1:]
	switch args[0] {
	case "node", "node.exe":
	case "python", "python3", "python.exe", "python3.exe":
		executable = selected.Generation.PythonExecutable
	case "npm", "npx":
		if executable == "" {
			break
		}
		modules := filepath.Join(filepath.Dir(executable), "node_modules")
		if runtime.GOOS != "windows" {
			modules = filepath.Join(filepath.Dir(filepath.Dir(executable)), "lib", "node_modules")
		}
		argv = append([]string{filepath.Join(modules, "npm", "bin", args[0]+"-cli.js")}, argv...)
	default:
		executable = ""
	}
	var bins []string
	for _, entry := range []string{selected.Generation.PythonExecutable, selected.Generation.NodeExecutable} {
		if entry != "" {
			bins = append(bins, filepath.Dir(entry))
		}
	}
	platform, err := runner.Platform()
	if err != nil {
		return err
	}
	overrides := map[string]string{}
	for k, v := range selected.Config.Env {
		overrides[k] = v
	}
	parent := pythonRunEnvironment(os.Environ(), overrides, selected.Generation.PythonExecutable, runtime.GOOS == "windows")
	for _, tool := range []string{"java", "go", "rust"} {
		if selected.Config.Tools[tool] == "" {
			continue
		}
		entry := backend.SDKEntry(selected.Generation.Directory, tool, platform)
		bins = append(bins, filepath.Dir(entry))
		if tool == "java" {
			overrides["JAVA_HOME"] = filepath.Dir(filepath.Dir(entry))
		}
		if tool == "go" {
			overrides["GOROOT"] = filepath.Dir(filepath.Dir(entry))
			overrides["GOTOOLCHAIN"] = "local"
		}
	}
	env, err := runner.Environment(parent, overrides, bins, runtime.GOOS == "windows")
	if err != nil {
		return &runFailure{cause: err, global: request.Global}
	}
	if executable == "" {
		executable, err = runner.Lookup(args[0], cwd, env, runtime.GOOS == "windows")
		if err != nil {
			return err
		}
	}
	if runtime.GOOS == "windows" && selected.Config.Tools["rust"] != "" {
		argv, err = rustLinker(request.Linker, executable, filepath.Join(selected.Generation.Directory, "sdks", "rust"), cwd, argv, overrides)
		if err != nil {
			return err
		}
		env, err = runner.Environment(parent, overrides, bins, true)
		if err != nil {
			return err
		}
	}
	code, runErr := runner.Execute(ctx, runner.Process{Completion: selected.Completion, TreeID: selected.TreeID, Executable: executable, Args: argv, Directory: cwd, Environment: env, Stdin: request.Stdin, Stdout: request.Stdout, Stderr: request.Stderr})
	if runErr != nil {
		return &runFailure{cause: runErr, global: request.Global}
	}
	if code != 0 {
		return &childExit{code}
	}
	return nil
}
