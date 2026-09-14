//go:build windows

package cli

import tea "charm.land/bubbletea/v2"

// Native conhost can retain fragments across our differently structured pages.
// Invalidate only on page/input transitions, never on ticks, progress or idle.
func repaintTUIPage(next tea.Cmd) tea.Cmd { return tea.Sequence(tea.ClearScreen, next) }
