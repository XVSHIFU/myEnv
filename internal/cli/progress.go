package cli

import (
	"fmt"
	"io"
	"myenv/internal/core"
)

func syncProgress(out io.Writer, languages ...locale) func(core.SyncPhase) {
	var lang locale
	if len(languages) > 0 {
		lang = languages[0]
	}
	messages := map[core.SyncPhase]string{
		core.PhaseWaiting:      "Waiting for the workspace lock...",
		core.PhaseResolving:    "Resolving runtime versions...",
		core.PhaseSDK:          "Preparing SDK...",
		core.PhaseBackend:      "Preparing the Python backend...",
		core.PhaseNode:         "Preparing Node...",
		core.PhasePython:       "Preparing Python...",
		core.PhaseDependencies: "Synchronizing Python dependencies...",
		core.PhaseVerifying:    "Verifying the prepared environment...",
		core.PhasePublishing:   "Applying the prepared environment...",
	}
	return func(phase core.SyncPhase) {
		if message := messages[phase]; message != "" {
			if display, ok := out.(*progressDisplay); ok {
				display.phase(lang.text(message))
			} else {
				fmt.Fprintln(out, lang.text(message))
			}
		}
	}
}
