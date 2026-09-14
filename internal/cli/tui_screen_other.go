//go:build !windows

package cli

import tea "charm.land/bubbletea/v2"

func repaintTUIPage(next tea.Cmd) tea.Cmd { return next }
