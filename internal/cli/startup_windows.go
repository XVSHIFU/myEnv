package cli

import "github.com/spf13/cobra"

func init() {
	// myEnv is a noninteractive CLI unless an operation explicitly requests
	// input. Cobra's Explorer detection enumerates processes on every command
	// and may pause a double-click launch; neither behavior belongs here.
	cobra.MousetrapHelpText = ""
}
