package cli

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestTUIRunHandoffRequiresExplicitConfirmation(t *testing.T) {
	for _, mode := range []string{"", "version-setup", "command", "run", "task"} {
		m := &tuiModel{mode: mode}
		if m.takeRunRequest() {
			t.Fatalf("Tea shutdown in %q authorized a command", mode)
		}
	}
	for _, key := range []tea.KeyPressMsg{
		{Code: tea.KeyEsc},
		{Code: 'q', Text: "q"},
		{Code: 'c', Mod: tea.ModCtrl},
	} {
		m := &tuiModel{mode: "run"}
		m.Update(key)
		if m.takeRunRequest() {
			t.Fatalf("cancel/quit key %s authorized a command", key.String())
		}
	}

	m := &tuiModel{mode: "run"}
	_, command := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if command == nil {
		t.Fatal("confirmed command did not leave Tea")
	}
	if _, ok := command().(tea.QuitMsg); !ok {
		t.Fatal("confirmed command must stop Tea before handing over terminal")
	}
	if !m.takeRunRequest() || m.takeRunRequest() {
		t.Fatal("explicit confirmation must authorize exactly one handoff")
	}
}
