package cli

import (
	tea "charm.land/bubbletea/v2"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"myenv/internal/backend"
	"myenv/internal/core"
	"strings"
)

type tuiUpdate struct{}
type tuiModel struct {
	homeIndex      int
	cursor         int
	inputStarted   bool
	catalog        *VersionResult
	catalogRequest UIRequest
	handledTask    uint64
	initializing   bool

	versionIndex   int
	versionMode    bool
	filter         string
	height         int
	width          int
	scroll         int
	controller     *UIController
	request        UIRequest
	focus          int
	editing        bool
	input, message string
	runArgs        []string
	commandDrafts  map[string][]string
	rawResult      bool
	quit           bool
	runRequested   bool
	workbench      *core.Workbench
	mode, theme    string
	menuIndex      int
	recent         []UIDirectory
	drafts         map[string]tuiDraft
	projectKey     string
	preview        *UIRequest
	previewDigest  string
	browsePath     string
	browseDirs     []UIDirectory
	returnIndex    int
}

func (m *tuiModel) Init() tea.Cmd { return nil }
func (m *tuiModel) Update(msg tea.Msg) (model tea.Model, command tea.Cmd) {
	previous, editing, versions := m.mode, m.editing, m.versionMode
	defer func() {
		if !m.quit && !(previous == "run" && m.mode == "") && (previous != m.mode || editing != m.editing || versions != m.versionMode) {
			command = repaintTUIPage(command)
		}
	}()

	if k, ok := msg.(tea.KeyPressMsg); ok && (k.String() == "ctrl+c" || k.String() == "q" && !m.editing) {
		m.quit = true
		return m, tea.Quit
	}
	if handled, cmd := m.flowUpdate(msg); handled {
		return m, cmd
	}
	if handled, cmd := m.workbenchUpdate(msg); handled {
		return m, cmd
	}
	switch k := msg.(type) {
	case tea.PasteMsg:
		if m.editing && len(m.input)+len(k.Content) < 8192 {
			m.insertInput(k.Content)
		}
	case tea.WindowSizeMsg:
		m.height = k.Height
		m.width = k.Width
	case tea.KeyPressMsg:
		key := k.String()
		if m.editing {
			m.prepareInput()
			switch key {
			case "esc":
				m.editing = false
				m.inputStarted = false
			case "enter":
				if m.focus == -1 {
					value := strings.TrimSpace(m.input)
					if value == "" {
						m.message = "请输入期望版本"
						return m, nil
					}
					if !strings.Contains(value, "@") {
						value = m.request.Tool + "@" + value
					}
					if !strings.HasPrefix(value, m.request.Tool+"@") {
						m.message = "请输入当前语言的版本"
						return m, nil
					}
					m.request.Selection = value
					m.versionMode = false
					m.mode = "save"
					m.message = ""
				} else if m.focus == 0 {
					m.switchProject(m.input, false)
				} else if m.focus == 1 {
					m.request.Selection = m.input
					m.selectTool(strings.SplitN(m.input, "@", 2)[0])
				} else if m.focus == 2 {
					m.request.Provider = m.input
				} else if m.focus == 3 {
					candidate := m.request
					if err := json.Unmarshal([]byte(m.input), &candidate); err != nil {
						m.message = err.Error()
					} else {
						candidate.Directory = m.request.Directory
						candidate.Global = m.request.Global
						m.request = candidate
					}
				} else if m.focus >= 6 {
					i := m.focus - 6
					for len(m.runArgs) <= i {
						m.runArgs = append(m.runArgs, "")
					}
					m.runArgs[i] = m.input
				} else if m.focus == 5 {
					m.filter = m.input
					m.versionIndex = 0
				} else {
					var args []string
					if err := json.Unmarshal([]byte(m.input), &args); err != nil {
						m.message = err.Error()
					} else {
						m.runArgs = args
					}
				}
				m.editing = false
				m.inputStarted = false
			case "ctrl+u":
				m.input = ""
				m.cursor = 0
			case "left":
				m.cursor = max(0, m.cursor-1)
			case "right":
				m.cursor = min(len([]rune(m.input)), m.cursor+1)
			case "home":
				m.cursor = 0
			case "end":
				m.cursor = len([]rune(m.input))
			case "delete":
				r := []rune(m.input)
				if m.cursor < len(r) {
					m.input = string(append(r[:m.cursor], r[m.cursor+1:]...))
				}
			case "backspace":
				r := []rune(m.input)
				if m.cursor > 0 {
					m.input = string(append(r[:m.cursor-1], r[m.cursor:]...))
					m.cursor--
				}

			default:
				if k.Text != "" && len(m.input)+len(k.Text) < 8192 {
					m.insertInput(k.Text)
				}
			}
			return m, nil
		}
		if m.versionMode {
			rows := m.versionRows()
			switch key {
			case "esc", "v":
				m.versionMode = false
			case "/":
				m.editing = true
				m.focus = 5
				m.input = m.filter
			case "up":
				if m.versionIndex > 0 {
					m.versionIndex--
				}
			case "down":
				if m.versionIndex+1 < len(rows) {
					m.versionIndex++
				}
			case "enter":
				if len(rows) > 0 {
					v := rows[m.versionIndex]
					if v.Kind == "release_page" || v.URL == "" {
						m.message = "此项只是官网发布记录，没有可安装目录"
					} else {
						m.request.Selection = m.request.Tool + "@" + v.Version
						if m.request.Tool == "python" {
							if v.Provider == "astral" {
								m.request.Provider = "astral"
							} else {
								m.request.Provider = "python.org"
							}
						}
						m.versionMode = false
						m.mode = "save"
					}
				}
			}
			return m, nil
		}

	}
	return m, nil
}
func addTUI(root *cobra.Command, dir *string, jsonOutput *bool, noInput *bool, namespace string) {
	var global bool
	cmd := &cobra.Command{Use: "tui", Short: "打开交互式终端界面", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if *noInput {
			return fmt.Errorf("tui 不支持 --no-input；请使用 CLI 命令")
		}
		if *jsonOutput {
			return fmt.Errorf("tui 不支持 --json")
		}
		if !terminalInput(cmd.InOrStdin(), cmd.OutOrStdout()) {
			return fmt.Errorf("tui 需要交互式 TTY；脚本请使用 CLI 或 --json")
		}
		c := NewUIController(namespace)
		defer c.Close()
		m := &tuiModel{controller: c}
		m.switchProject(*dir, global)
		for {
			p := tea.NewProgram(m, tea.WithInput(cmd.InOrStdin()), tea.WithOutput(cmd.OutOrStdout()), tea.WithContext(cmd.Context()), tea.WithFPS(10))
			stopped := make(chan struct{})
			forwarded := make(chan struct{})
			go func() {
				defer close(forwarded)
				for {
					select {
					case <-stopped:
						return
					case <-c.Updates():
						p.Send(tuiUpdate{})
					}
				}
			}()
			_, err := p.Run()
			close(stopped)
			<-forwarded
			if err != nil {
				return err
			}
			if m.quit || !m.takeRunRequest() {
				return nil
			}
			// Program.Run has joined Tea's renderer, input reader and signal handlers.
			// Use the command context, never Tea's canceled child context; runner owns signals.
			code, runErr := RunCommand(cmd.Context(), RunRequest{Directory: m.request.Directory, Namespace: namespace, Global: m.request.Global, Linker: "auto", Chinese: true, Stdin: cmd.InOrStdin(), Stdout: cmd.OutOrStdout(), Stderr: cmd.ErrOrStderr()}, m.runArgs)
			m.message = fmt.Sprintf("上次命令退出码：%d", code)
			if runErr != nil {
				m.message += " · " + runErr.Error()
			}
			m.request.Action = ""
		}
	}}
	cmd.Flags().BoolVar(&global, "global", false, "打开用户默认环境")
	root.AddCommand(cmd)
}

// A normal Tea shutdown can come from SIGTERM as well as tea.Quit. Only the
// explicit run-confirmation action authorizes a handoff, and it is consumed once.
func (m *tuiModel) takeRunRequest() bool {
	requested := m.runRequested
	m.runRequested = false
	return requested
}

func (m *tuiModel) versionRows() []backend.CatalogRelease {
	var result VersionResult
	t := m.controller.Current()
	if m.catalog != nil {
		result = *m.catalog
		t.Request = m.catalogRequest
	} else if json.Unmarshal(t.Result, &result) != nil {
		return nil
	}
	if t.Request.Tool != m.request.Tool || t.Request.Provider != m.request.Provider || t.Request.Preview != m.request.Preview || t.Request.Major != m.request.Major || t.Request.Channel != m.request.Channel || t.Request.Date != m.request.Date {
		return nil
	}
	rows := []backend.CatalogRelease{}
	for _, row := range result.Releases {
		if backend.CatalogReleaseMatches(row, m.filter) {
			rows = append(rows, row)
		}
	}
	if m.versionIndex >= len(rows) {
		m.versionIndex = 0
	}
	return rows
}
