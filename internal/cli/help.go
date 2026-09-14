package cli

import (
	_ "embed"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func addHelp(root *cobra.Command, jsonOutput *bool, languages ...locale) {
	var lang locale
	if len(languages) > 0 {
		lang = languages[0]
	}
	root.Long += "\nRead the offline manual with myenv help manual."
	completion := &cobra.Command{
		Use: "completion <shell>", Short: "Generate a shell completion script",
		Long:      "Generate local completion for Bash, Zsh, Fish or PowerShell.\nNo downloads, backend execution or shell configuration changes are performed.",
		Example:   "  myenv completion powershell\n  myenv completion bash",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			if *jsonOutput {
				return fmt.Errorf("--json is not supported for completion")
			}
			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(cmd.OutOrStdout(), true)
			case "zsh":
				return root.GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return root.GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			}
			return nil
		},
	}
	root.AddCommand(completion)
	help := &cobra.Command{
		Use: "help [command | manual]", Short: "Read command help or the offline manual",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if *jsonOutput {
				return fmt.Errorf("--json is not supported for help")
			}
			if len(args) == 0 {
				return root.Help()
			}
			if args[0] == "manual" {
				return printManual(root, cmd.OutOrStdout(), lang)
			}
			for _, child := range root.Commands() {
				if child.Name() == args[0] {
					child.InitDefaultHelpFlag()
					return child.Help()
				}
			}
			return fmt.Errorf("unknown help topic %q", args[0])
		},
	}
	// Cobra adds the configured help command lazily, after localizeHelp walks
	// existing children. Translate this detached command before registration.
	if lang {
		help.Short = chineseCommands["help"][0]
		help.Long = chineseCommands["help"][1]
	}
	root.SetHelpCommand(help)
}

// The reference is rendered from the same command objects as --help/completion.
//
//go:embed manual_zh.txt
var chineseManual string

func printManual(root *cobra.Command, out io.Writer, languages ...locale) error {
	manual := `myEnv offline manual (development build)

The workflow is init → sync → run, with use for version changes and doctor for
diagnosis. This development build prepares project Node/Python/Java/Go/Rust environments.
Use myenv or myenv status to read the current project status.
--lang auto|zh-CN|en or MYENV_LANG selects human-readable help and messages.
Explicit flags override MYENV_LANG; auto uses the system language with English fallback.
JSON, generated shell scripts and child output keep their existing contracts.
Options after the run executable belong to that executable, including --lang.
The Windows setup wizard installs for the current user and can add its directory
to user PATH. Reopen the terminal after installation. A portable ZIP is also available.
Uninstall removes owned program files/PATH entries, retaining project and runtime data.
Windows and native Linux runtime checks have passed; performance acceptance remains open. macOS is excluded from this delivery.
Windows execution requires Windows 10 or newer (Server 2016 or newer) for
Job assignment at process creation. This minimum is not a tested OS matrix.
run executes the applied environment without implicit installation.
Managed node/python/npm/npx select entries from that environment. Other command
names are looked up in its PATH; executable paths may be absolute or relative
to the invocation directory. Arguments are retained. Managed Windows Rust uses bundled LLD unless a linker is explicitly configured; run --rust-linker=system disables this adjustment.
Managed Python sets VIRTUAL_ENV and clears inherited PYTHONHOME; explicit env configuration takes precedence.
run does not interpret shell pipelines or redirection; use your shell explicitly
when those features are needed.

SDK and system management
Examples are not version limits. Available versions depend on compatible upstream
archives for the current platform; availability is not a claim of native validation.
Java uses Adoptium Eclipse Temurin HotSpot JDK x64: choose available majors such as
8, 11, 17, 21 or 25, or a complete release name. Other vendors, OpenJ9 and JRE are excluded.
myenv versions java --major 21 queries a family; myenv system install java@21 installs it.
Node uses nodejs.org archives, Go uses go.dev archives, and Rust uses official full
toolchain archives (Windows MSVC / Linux GNU). For example:
  myenv versions node --search 22
  myenv system install node@22
  myenv versions go --search 1.26
  myenv system install go@1.26
  myenv versions rust --search 1.98
  myenv system install rust@1.98
Use use instead of system install for a project environment.
versions TOOL --preview includes preview records without installing them. Rust --channel and
--date query dated manifests, not a complete historical nightly index; Java --major only filters queries.
use python@3.14 --provider python.org uses full official Windows x64 runtime ZIPs;
--provider astral uses the pinned uv 0.11.26 catalog of default CPython builds.
Installation defaults to Astral, which may lag python.org. The provider is persisted.
On Windows and Linux, versions python defaults to the Astral catalog, matching installation.
It requires the existing pinned uv and never installs it implicitly for a query.
Explicit --provider python.org lists Windows full ZIPs or Linux upstream release pages.
Release pages are not installable artifacts; embedded/free-threaded/test ZIPs are excluded.
There is no supported python.org Linux binary backend; no silent fallback occurs.
system list/doctor discover existing runtimes; doctor includes compiler prerequisites.
C/C++ is detected, not automatically installed.
system install/upgrade/repair/remove manage myenv's current-user default profile.
repair rebuilds the whole profile; remove retains protected historical generations.
system external ID --manager ABSOLUTE_PATH --action ACTION defaults to a plan;
--apply invokes a verified original uv/rustup/Conda manager. Capabilities vary.
Conda base and unknown installers are protected. External changes have no myenv
rollback guarantee; Conda remove deletes the entire environment and packages.
No Anaconda installation is performed by these commands.

Configuration
Create myenv.yaml with schema: 1 and a tools mapping. Quote version values.
Supported tools are python, node, java (Temurin), go and rust. Optional python.project is a relative path
inside the workspace; python.groups selects dependency groups. env values must
be ordinary strings. Supply secrets through the calling process environment.
Python dependencies remain in pyproject.toml and uv.lock.
Python sync uses uv.toml or tool.uv from the effective project/workspace root,
without merging unrelated ancestor/user/system uv configuration. Workspace
members use the shared uv.lock at their native workspace root, which must stay
inside the myEnv workspace. Native project metadata
stays in pyproject.toml. A temporary configuration snapshot is removed after
the backend invocation; it resides beside the project to preserve relative paths.

Python synchronization
sync prepares a new venv at its final path using a fixed managed uv backend.
It records native input digests and explicitly selects python.groups without
automatically adding default groups. --locked never updates either lock file.
Astral Python runtime locks provide version evidence; Node/Java/Go/Rust and python.org
archives provide artifact evidence. A changed pyproject.toml, uv.lock or uv.toml requires synchronization.
Build code requires --allow-build or one terminal confirmation for this invocation.
Complex workspace input tracking and complete build-source coverage remain under development.
On failure, the old environment is retained even when desired locks have changed;
use run --current to run it. rollback retains declarations and native locks.

Version selectors
Bare major/minor versions select that stable release series. Node/Python numeric
ranges support >=, >, <=, <, != and ==, comma/space conjunctions and || unions.
Node also supports ^, ~ and hyphen ranges; Python supports ~=. Java/Go/Rust accept
numeric families or complete release names, with = for an exact match, not comparison ranges.
Explicit preview versions and Rust beta/nightly[-YYYY-MM-DD] are supported;
stable ranges do not implicitly select previews. Invalid selectors fail before writes.
init intersects declarations from version files and manifests; conflicting
declarations require correction. Existing myenv.yaml files are never overwritten.

Output and errors
--json on status/init/sync emits one object with schema, ok, changed, data and error.
Errors contain code, message and next_action. Diagnostics go to stderr.
Exit 0 means success, 1 means an I/O/environment failure, 2 means invalid usage/configuration, and 3 needs input.
--no-input and --json never prompt. In a terminal, init asks for a tool/version only when declarations are missing or conflicting;
CI and redirected input/output disable interaction. End input or submit an empty value to cancel.

Run exit status and signals
After a child starts, run preserves its exit status instead of mapping it to the
myEnv 1/2/3 error categories. A child that handles a signal and exits normally
keeps that exit code. On Linux, termination by a signal returns 128 plus the
signal number (SIGINT 130, SIGQUIT 131, SIGTERM 143); Windows uses the native
process exit status. Failures before launch use the myEnv error categories.
Linux forwards SIGINT, SIGTERM, SIGHUP, SIGQUIT and SIGCONT through its supervisor.
Terminal input and foreground ownership are restored when the command finishes.
Linux Ctrl-Z and shell fg/bg handling is implemented and tested with isolated
native Linux Bash/PTY fixtures. Bash, zsh and fish profile integration is tested.
A stopped caller cannot execute Go cancellation callbacks until resumed; absolute
deadlines remain enforced by the independent supervisor. macOS is excluded.
Cleanup waits for supervised descendants. If completion cannot be confirmed,
the generation remains protected from clean. A lease-finalization error is
reported on stderr without replacing a completed child's exit status.

Build and local installation
From this source tree: go build -o dist/myenv ./cmd/myenv
On Windows use dist/myenv.exe. Run by absolute path, or place it in a directory
you explicitly add to PATH. Check Get-Command myenv -All (PowerShell) or
command -v myenv (POSIX shell) before installation; do not overwrite another tool.
Remove only the installed binary to uninstall. Project files remain separate.

Completion setup (explicit, optional)
Bash: source <(myenv completion bash)
Zsh: save myenv completion zsh output as _myenv in a directory on fpath,
then initialize completion with autoload -Uz compinit; compinit.
Fish: save myenv completion fish output to ~/.config/fish/completions/myenv.fish.
PowerShell: myenv completion powershell | Out-String | Invoke-Expression
These commands enable shell completion; myEnv does not modify shell profiles.

User profile (explicit current-user scope)
myenv use --global python@3.12 creates or edits the user's profile and syncs it.
The declaration is myenv/profile.yaml under the OS user configuration directory;
only schema/tools are accepted. It is independent of project declarations.
sync, run, doctor and rollback also accept --global. For example:
  myenv run --global python --version
  myenv doctor --global
Profile state uses .myenv-profile and profile.lock next to profile.yaml.
run keeps the requested working directory. Projects do not inherit profile tools.
Explicit user-tool shell integration
PowerShell: myenv shell-init powershell | Out-String | Invoke-Expression
Bash/Zsh: eval "$(myenv shell-init bash)" (use zsh for Zsh)
Fish: myenv shell-init fish | source
This defines functions for declared profile tools (node/npm/npx and/or python).
They take precedence over PATH commands in this shell and invoke run --global
through the absolute myEnv path, preserving supervision and argument forwarding.
The command only prints code; it does not install tools or edit shell files.
Close the shell to discard the functions; re-evaluate after adding profile tools
or moving myEnv. Removing a tool from the profile does not remove an existing
shell function: start a new shell after removing tools or uninstalling myEnv.
Previously shadowed user functions are not saved or restored by shell-init.
Versions are selected afresh on each call. Child programs that
look up executables directly still use their own PATH, not these shell functions.
shell-init does not support --json. A valid user profile must already exist.

Proxies, mirrors and credentials
Backend processes inherit ordinary caller environment variables, including
HTTPS_PROXY. Managed uv removes inherited UV_* settings to keep its invocation
controlled, so exporting UV_* is not a supported myEnv configuration interface.
Set MYENV_NODE_MIRROR to an HTTP(S) distribution base URL to explicitly select
a Node source, for example https://mirror.example/node/dist. It must serve
index.json, version directories and SHASUMS256.txt in the official layout.
New resolutions record that source and its SHA-256 in the lock; existing locked
URLs are unchanged. Trust the selected source: its checksum is not a signature.
Credentials, query strings and fragments in the base URL are rejected.
Unset it to use the official Node source. MYENV_UV_MIRROR selects the pinned uv
engine download base, with <version>/<official-archive-name> beneath it; the
embedded version and SHA-256 remain mandatory. An already installed uv is reused.
MYENV_PYTHON_MIRROR selects the CPython runtime mirror passed explicitly to uv
python install --mirror. Preserve the python-build-standalone release directory
and archive layout beneath that base. Existing shared installations are reused.
This changes transport, not Python's version-only lock evidence; it does not
turn the runtime lock into an archive hash lock.
Set SSL_CERT_FILE to an absolute PEM bundle path for custom CA trust. Node and
uv-engine downloads accept a regular file up to 4 MiB; the bundle replaces their
default roots. uv processes inherit the same variable. Invalid or empty bundles
fail instead of disabling verification. SSL_CERT_DIR is not implemented by the
myEnv downloader. No certificates are installed into the system store.
Pass credentials through the calling environment instead of committing them in
myenv.yaml. Runtime download transport errors omit credential-bearing URLs.

Deep content diagnosis
myenv doctor --deep compares generation-owned files with their publication baseline.
It may be slow and never repairs files. Older generations may have no baseline.
Use myenv sync --rebuild --locked to prepare a fresh generation from current locks;
--rebuild --dry-run previews this operation. The old generation is retained.
Rebuilding does not reinstall shared base interpreters or clear shared caches.
Python bytecode caches, external runtime contents and ACLs are outside this check.
Ordinary doctor/status/run retain quick checks without scanning file contents.
Doctor also checks for retained run-protection records without changing them.
Its JSON data.run_protection is present or none; present does not prove that
the recorded processes are alive. data.protection_detail explains the next step.
Use myenv clean --dry-run to preview safe project reclamation. clean does not
accept --global. This check does not count preparation holds or reclaim records.

Shared Python storage
sync/use now keep the managed uv backend and Python runtimes under the user's
myenv data directory: LOCALAPPDATA on Windows, XDG_DATA_HOME (or ~/.local/share)
on Linux. uv caches use the OS user
cache directory under myenv. Project venv generations remain in .myenv/generations.
Existing applied environments keep their original paths; no-op sync does not move
them. Node archives are shared by SHA-256 in the user cache and copied into each operation. Use clean --cache node --dry-run to preview shared Node archive cleanup. clean --cache uv delegates cleanup to the installed pinned uv, without --force.

Manual cleanup
myenv clean --dry-run lists unused project generations and logical file bytes.
myenv clean removes those generations after rechecking protection in a transaction.
Current, previous, leased and preparing generations are retained. Use --json for
one structured result including individual items, totals and any partial failure.
Failed preparation records and their generation/download directories are also cleaned. Windows clean can recover identified leases after supervisor and Job exit; unknown and legacy leases remain protected. Use clean --cache node to remove shared Node transport archives; Use clean --cache uv to delegate cache cleanup to the installed pinned uv, without --force.
clean does not accept --global.

Command reference`
	if len(languages) > 0 && languages[0] {
		manual = chineseManual
	}
	if _, err := fmt.Fprintln(out, manual); err != nil {
		return err
	}
	var render func(*cobra.Command) error
	render = func(c *cobra.Command) error {
		if c.Hidden {
			return nil
		}
		c.InitDefaultHelpFlag()
		if _, err := fmt.Fprintf(out, "\n%s\n%s\n", c.CommandPath(), c.Long); err != nil {
			return err
		}
		if c.Example != "" {
			if _, err := fmt.Fprintln(out, c.Example); err != nil {
				return err
			}
		}
		// UsageString includes both local and inherited flags, using Cobra definitions.
		if _, err := io.WriteString(out, c.UsageString()); err != nil {
			return err
		}
		for _, child := range c.Commands() {
			if err := render(child); err != nil {
				return err
			}
		}
		return nil
	}
	return render(root)
}
