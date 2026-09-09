package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"myenv/internal/config"
)

func addShellInit(root *cobra.Command, namespace string, jsonOutput *bool) {
	root.AddCommand(&cobra.Command{Use: "shell-init <bash|zsh|fish|powershell>", Short: "Print explicit user-profile tool wrappers for your shell", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if *jsonOutput {
			return fmt.Errorf("shell-init outputs shell code and does not support --json")
		}
		switch args[0] {
		case "bash", "zsh", "fish", "powershell":
		default:
			return fmt.Errorf("unsupported shell %q", args[0])
		}
		path, err := config.ProfilePath(namespace)
		if err != nil {
			return err
		}
		profile, err := config.LoadProfile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("ENV_NOT_READY: create a user profile with myenv use --global <tool>@<version> before shell-init: %w", err)
			}
			return err
		}
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		script := profileShellScript(args[0], executable, profile.Tools)
		_, err = fmt.Fprint(cmd.OutOrStdout(), script)
		return err
	}})
}

func profileShellScript(shell, executable string, tools map[string]string) string {
	commands := []string{}
	if tools["node"] != "" {
		commands = append(commands, "node", "npm", "npx")
	}
	if tools["python"] != "" {
		commands = append(commands, "python")
	}
	if tools["java"] != "" {
		commands = append(commands, "java", "javac", "jar")
	}
	if tools["go"] != "" {
		commands = append(commands, "go", "gofmt")
	}
	if tools["rust"] != "" {
		commands = append(commands, "rustc", "cargo", "rustdoc")
	}
	var script strings.Builder
	script.WriteString("# myEnv user-profile wrappers; no installation or shell-file changes.\n")
	for _, name := range commands {
		switch shell {
		case "bash", "zsh":
			quoted := "'" + strings.ReplaceAll(executable, "'", "'\"'\"'") + "'"
			fmt.Fprintf(&script, "%s() { command %s run --global %s \"$@\"; }\n", name, quoted, name)
		case "fish":
			quoted := "'" + strings.ReplaceAll(strings.ReplaceAll(executable, "\\", "\\\\"), "'", "\\'") + "'"
			fmt.Fprintf(&script, "function %s\n  command %s run --global %s $argv\nend\n", name, quoted, name)
		case "powershell":
			quoted := "'" + strings.ReplaceAll(executable, "'", "''") + "'"
			fmt.Fprintf(&script, "function global:%s { & %s run --global %s @args }\n", name, quoted, name)
		}
	}
	return script.String()
}
