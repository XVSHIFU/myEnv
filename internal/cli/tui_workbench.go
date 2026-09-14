package cli

import (
	tea "charm.land/bubbletea/v2"
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"image/color"
	"os"
	"path/filepath"
	"strings"
)

type tuiDraft struct {
	Request  UIRequest
	Args     []string
	Commands map[string][]string
}

func defaultToolCommand(tool string) []string {
	switch tool {
	case "java":
		return []string{"java", "-version"}
	case "go":
		return []string{"go", "version"}
	case "rust":
		return []string{"rustc", "--version"}
	default:
		return []string{tool, "--version"}
	}
}
func (m *tuiModel) selectTool(tool string) {
	if tool == m.request.Tool {
		return
	}
	if m.commandDrafts == nil {
		m.commandDrafts = map[string][]string{}
	}
	if m.request.Tool != "" {
		m.commandDrafts[m.request.Tool] = append([]string(nil), m.runArgs...)
	}
	m.request.Tool = tool
	m.request.Selection = ""
	m.catalog = nil
	m.filter = ""
	m.request.Provider = ""
	m.request.Major = 0
	m.request.Channel = ""
	m.request.Date = ""
	if tool == "python" {
		m.request.Provider = "astral"
		if m.workbench != nil && strings.HasPrefix(m.workbench.Desired[tool], "python.org/") {
			m.request.Provider = "python.org"
		}
	}
	if args, ok := m.commandDrafts[tool]; ok {
		m.runArgs = append([]string(nil), args...)
	} else {
		m.runArgs = defaultToolCommand(tool)
	}
}

var uiTools = []string{"python", "node", "java", "go", "rust"}
var uiMenu = []string{"项目目录 / 最近项目", "软件来源详情", "主题偏好", "任务与错误记录", "高级任务设置", "返回工作台"}

func (m *tuiModel) browse(path string) {
	d, err := NormalizeUIDirectory(path)
	if err != nil {
		m.message = err.Error()
		return
	}
	f, err := os.Open(d.Path)
	if err != nil {
		m.message = err.Error()
		return
	}
	defer f.Close()
	entries, err := f.ReadDir(256)
	if err != nil && len(entries) == 0 {
		m.message = "目录为空；. 选择当前目录"
	}
	m.browsePath = d.Path
	m.browseDirs = nil
	for _, e := range entries {
		if e.IsDir() {
			p := filepath.Join(d.Path, e.Name())
			m.browseDirs = append(m.browseDirs, UIDirectory{Path: p})
		}
	}
	m.mode = "browse"
	m.menuIndex = 0
	m.scroll = 0
}

func uiActive(t UITask) bool {
	return t.State == "running" || t.State == "confirm" || t.State == "canceling"
}
func (m *tuiModel) switchProject(path string, global bool) {
	if uiActive(m.controller.Current()) {
		m.message = "任务仍在收尾，不能切换项目"
		return
	}
	dir, err := NormalizeUIDirectory(path)
	if err != nil {
		m.message = err.Error()
		return
	}
	if m.drafts == nil {
		m.drafts = map[string]tuiDraft{}
	}
	if m.projectKey != "" {
		m.drafts[m.projectKey] = tuiDraft{m.request, append([]string(nil), m.runArgs...), m.commandDrafts}
	}
	key := dir.Key
	if global {
		key = "@user"
	}
	if d, ok := m.drafts[key]; ok {
		m.request = d.Request
		m.runArgs = append([]string(nil), d.Args...)
		m.commandDrafts = d.Commands
	} else {
		m.request = UIRequest{Directory: dir.Path, Global: global, Tool: "java"}
		m.runArgs = []string{"java", "-version"}
		m.commandDrafts = map[string][]string{}
	}
	m.request.Directory = dir.Path
	m.request.Global = global
	for i, t := range uiTools {
		if t == m.request.Tool {
			m.homeIndex = i
		}
	}
	m.catalog = nil
	m.versionMode = false
	m.projectKey = key
	m.workbench = nil
	m.preview = nil
	m.message = ""
	m.mode = ""
	m.scroll = 0
	if !global {
		list := []UIDirectory{dir}
		for _, p := range m.recent {
			if p.Key != dir.Key && len(list) < 12 {
				list = append(list, p)
			}
		}
		m.recent = list
	}
	request := m.request
	request.Action = "workbench"
	if _, err := m.controller.Start(request); err != nil {
		m.message = err.Error()
	}
}
func (m *tuiModel) workbenchUpdate(msg tea.Msg) (bool, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok || m.editing {
		return false, nil
	}
	key := k.String()
	if key == "j" && (m.mode == "preview" || m.mode == "task" || m.mode == "history") {
		m.rawResult = !m.rawResult
		m.scroll = 0
		return true, nil
	}
	if m.mode == "run" && key == "enter" {
		m.runRequested = true
		m.mode = ""
		return true, tea.Quit
	}
	if m.mode == "run" && key == "esc" {
		m.mode = ""
		return true, nil
	}
	if key == "c" {
		m.controller.Cancel(m.controller.Current().ID)
		return true, nil
	}
	if key == "y" || key == "n" {
		m.controller.Confirm(m.controller.Current().ID, key == "y")
		return true, nil
	}
	if key == "s" && m.mode == "preview" {
		m.mode = ""
	}
	if m.versionMode {
		return false, nil
	}
	if m.mode != "" {
		switch key {
		case "esc":
			if m.mode == "menu" {
				m.mode = ""
			} else {
				m.mode = "menu"
			}
			m.menuIndex = m.returnIndex
			m.scroll = 0
		case "up":
			if m.menuIndex > 0 {
				m.menuIndex--
			}
			if m.mode == "browse" {
				m.scroll = max(0, m.menuIndex-max(1, m.height-13))
			}
		case "down":
			m.menuIndex++
			if m.mode == "browse" {
				m.menuIndex %= max(1, len(m.browseDirs))
				m.scroll = max(0, m.menuIndex-max(1, m.height-13))
			}
		case "pgdown":
			m.scroll += max(1, m.height-9)
		case "pgup":
			m.scroll = max(0, m.scroll-max(1, m.height-9))
		case "1":
			if m.mode == "projects" {
				m.focus = 0
				m.input = m.request.Directory
				m.editing = true
			}
		case "b":
			if m.mode == "projects" {
				m.browse(m.request.Directory)
			}
		case "backspace":
			if m.mode == "browse" {
				m.browse(filepath.Dir(m.browsePath))
			}
		case ".":
			if m.mode == "browse" {
				m.switchProject(m.browsePath, false)
			}
		case "enter":
			switch m.mode {
			case "browse":
				if len(m.browseDirs) > 0 {
					m.browse(m.browseDirs[m.menuIndex%len(m.browseDirs)].Path)
				}
			case "menu":
				m.returnIndex = m.menuIndex % len(uiMenu)
				switch m.menuIndex % len(uiMenu) {
				case 0:
					m.mode = "projects"
				case 1:
					m.mode = "source"
				case 2:
					m.mode = "theme"
				case 3:
					m.mode = "history"
				case 4:
					m.mode = ""
					m.focus = 3
					m.editing = true
					b, _ := json.Marshal(m.request)
					m.input = string(b)
				case 5:
					m.mode = ""
				}
				m.menuIndex = 0
			case "projects":
				if len(m.recent) > 0 {
					m.switchProject(m.recent[m.menuIndex%len(m.recent)].Path, false)
				}
			case "theme":
				m.theme = []string{"system", "light", "dark"}[m.menuIndex%3]
			}
		}
		return true, nil
	}
	switch key {
	case "m", "?":
		m.mode = "menu"
		m.menuIndex = 0
		return true, nil
	case "s":
		if m.preview == nil || uiActive(m.controller.Current()) {
			m.message = "先完成当前目标的同步/清理预览"
			return true, nil
		}
		r := *m.preview
		r.Preview = false
		r.ExpectedDigest = m.previewDigest
		m.preview = nil
		m.mode = "task"
		if r.Action == "clean" || r.Action == "external" {
			r.Apply = true
		} else {
			r.Action = "sync"
		}
		if _, err := m.controller.Start(r); err != nil {
			m.message = err.Error()
		}
		return true, nil
	}
	return false, nil
}
func displayCell(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = ansi.Truncate(s, width, "…")
	return s + strings.Repeat(" ", max(0, width-ansi.StringWidth(s)))
}
func (m *tuiModel) View() tea.View {
	width, height := m.width, m.height
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}
	width = max(30, width)
	height = max(12, height)
	lines := []string{}
	line := func(s string) {
		cell := displayCell(s, width-2)
		if strings.HasPrefix(strings.TrimSpace(s), "> ") {
			cell = "\x1b[7m" + cell + "\x1b[27m"
		}
		lines = append(lines, "│"+cell+"│")
	}
	rule := func() { lines = append(lines, "├"+strings.Repeat("─", width-2)+"┤") }
	lines = append(lines, "┌"+strings.Repeat("─", width-2)+"┐")
	scope := "项目"
	if m.request.Global {
		scope = "用户默认"
	}
	line(" myEnv / 环境工作台    " + scope + "    [m 菜单]")
	line(" " + m.request.Directory)
	rule()
	t := m.controller.Current()
	if m.mode != "" && !m.editing {
		line(" " + map[string]string{"menu": "菜单 / 上下文操作", "projects": "项目目录 · 1 输入路径 · Enter 最近项目 · b 浏览", "browse": "浏览目录 · Enter进入 · Backspace上级 · .选择当前", "theme": "主题偏好（唯一入口）", "source": "来源详情 / 期望与生效锁", "history": "任务与错误记录", "preview": m.previewTitle(), "task": "任务详情", "run": "运行确认", "save": "保存配置确认", "version-setup": "来源与版本选择", "maintenance": "更多操作", "command": "运行命令", "maint-confirm": "维护确认", "external-confirm": "外部管理确认"}[m.mode])
		rule()
		body := m.flowBody()
		switch m.mode {
		case "browse":
			body = append(body, m.browsePath, "每层读取至多256个入口；其他目录可用路径输入。")
			for i, d := range m.browseDirs {
				mark := "  "
				if m.menuIndex%max(1, len(m.browseDirs)) == i {
					mark = "> "
				}
				body = append(body, mark+filepath.Base(d.Path))
			}
		case "preview", "task":
			body = append(body, t.Summary.Title)
			for _, change := range t.Summary.Changes {
				body = append(body, change.Tool+": "+change.From+" → "+change.To)
			}
			body = append(body, t.Summary.Lines...)
			if m.rawResult {
				b, _ := json.MarshalIndent(t.Result, "", "  ")
				body = append(body, strings.Split(string(b), "\n")...)
			}
			if t.Error != nil {
				body = append([]string{t.Error.Code + ": " + t.Error.Message, t.Error.NextAction}, body...)
			}
			if t.Log != "" {
				body = append(body, t.Log)
			}
		case "menu":
			for i, s := range uiMenu {
				mark := "  "
				if m.menuIndex%len(uiMenu) == i {
					mark = "> "
				}
				body = append(body, mark+s)
			}
		case "theme":
			for i, s := range []string{"系统：继承终端默认前景和背景", "浅色：森林绿 / 浅色背景", "暗色：森林绿 / 深色背景"} {
				mark := "  "
				if m.menuIndex%3 == i {
					mark = "> "
				}
				body = append(body, mark+s)
			}
			body = append(body, "当前："+m.theme, "系统模式不猜测终端背景，手动主题固定。")
		case "projects":
			for i, d := range m.recent {
				mark := "  "
				if m.menuIndex%max(1, len(m.recent)) == i {
					mark = "> "
				}
				body = append(body, mark+d.Path)
			}
		case "source":
			if m.workbench != nil {
				lock := m.workbench.Applied[m.request.Tool]
				body = []string{"工具：" + m.request.Tool + "  平台：" + m.workbench.Platform, "期望：" + m.workbench.Desired[m.request.Tool], "生效版本：" + lock.Version, "安装后端：" + lock.Backend, "锁定 URL：" + lock.URL, "制品 SHA256：" + lock.SHA256, "证据类型：" + lock.Evidence, "空摘要不代表已校验；当前没有签名验证证据。", "Python 默认 CPython / Astral 构建 / 固定 uv 后端。", "Java 仅 Eclipse Temurin / HotSpot JDK。", "镜像设置见高级任务参数，不倒推旧下载来源。", "当前环境目录：" + m.workbench.Path}
			}
		case "history":
			for _, r := range m.controller.History() {
				if r.Request.Global == m.request.Global && (m.request.Global || r.Request.Directory == m.request.Directory) {
					body = append(body, fmt.Sprintf("#%d %s · %s", r.ID, r.Request.Action, r.Summary.Title))
					body = append(body, r.Summary.Lines...)
					if m.rawResult {
						b, _ := json.Marshal(r.Result)
						body = append(body, string(b))
					}
				}
			}
		case "run":
			args, _ := json.Marshal(m.runArgs)
			body = []string{"将执行以下精确参数（不是 shell 脚本）：", string(args), "使用当前项目/用户默认环境；Enter交接终端，Esc返回。"}
		}
		wrapped := strings.Split(ansi.Hardwrap(strings.Join(body, "\n"), width-4, false), "\n")
		start := min(m.scroll, len(wrapped))
		if m.mode == "command" || m.mode == "maintenance" {
			for i, line := range wrapped {
				if strings.HasPrefix(line, "> ") {
					start = max(0, i-(height-11))
					break
				}
			}
		}
		for _, s := range wrapped[start:min(len(wrapped), start+height-10)] {
			line(" " + s)
		}
	} else if m.versionMode && !m.editing {
		if m.request.Tool == "java" {
			line(" Temurin近期目录 · Java 8 = JDK 1.8 · Enter选择，尚不保存")
		} else {
			line(" 版本目录 · Enter选择目标（尚不保存）")
		}
		line(fmt.Sprintf(" %s · 来源 %s · 预发布 %t · 筛选 %s", m.request.Tool, m.request.Provider, m.request.Preview, m.filter))
		rows := m.versionRows()
		visibleRows := height - 10
		start := max(0, m.versionIndex-visibleRows+1)
		for i := start; i < min(len(rows), start+visibleRows); i++ {
			mark := "  "
			if i == m.versionIndex {
				mark = "> "
			}
			label := rows[i].Version + " · " + rows[i].Kind
			if rows[i].DisplayVersion != "" {
				label = rows[i].DisplayVersion + " · " + rows[i].Version
				if rows[i].Kind == "release_page" {
					label = "只读发布记录 · " + label
				} else if rows[i].Channel == "preview" {
					label = "预览 · " + label
				}
			}
			line(mark + label)
		}
	} else if m.editing {
		line(" 编辑 · Enter保存 / Esc取消 · Ctrl+U清空")
		line(" 中文输入、终端复制/粘贴快捷键保留")
		rule()
		wrapped := strings.Split(ansi.Hardwrap(m.inputDisplay(), width-4, false), "\n")
		// Wrap the prefix including the cursor with the same terminal column rules.
		prefix := string([]rune(m.input)[:m.cursor]) + "▏"
		row := strings.Count(ansi.Hardwrap(prefix, width-4, false), "\n")
		count := max(1, height-11)
		start := max(0, min(row-count+1, len(wrapped)-count))
		for _, s := range wrapped[start:min(len(wrapped), start+count)] {
			line(" " + s)
		}
	} else {
		for _, s := range m.homeBody() {
			line(" " + s)
		}

	}
	for len(lines) < height-4 {
		line("")
	}
	if len(lines) > height-4 {
		lines = lines[:height-4]
	}
	rule()
	status := fmt.Sprintf(" 任务 %d · %s · %s", t.ID, uiTaskState(t.State), uiPhase(t.Phase))
	if m.message != "" {
		status = m.message
	} else if t.Error != nil {
		status = t.Error.Code + ": " + t.Error.Message
	}
	line(status)
	footer := "↑↓选择 Enter进入 g项目/默认 m设置 q退出"
	if m.mode != "" {
		footer = "↑↓选择 Enter打开 1输入路径 PgUp/PgDn滚动 Esc返回"
	}
	if m.mode == "preview" {
		footer = "Enter" + m.previewConfirm() + " j原始详情 PgUp/PgDn翻页 Esc返回"
	}
	if m.mode == "run" {
		footer = "Enter交接终端 PgUp/PgDn完整参数 Esc返回 Ctrl+C退出"
	}
	if m.versionMode {
		footer = "↑↓选择 Enter目标 /筛选 i手输 p预发布 Esc返回"
		if m.request.Tool == "python" {
			footer += " o来源"
		}
	}
	if m.mode == "version-setup" {
		footer = "↑↓选择 Enter操作 i手输 p预发布 Esc返回"
		if m.request.Tool == "python" {
			footer += " o来源"
		}
	}
	if m.mode == "save" {
		footer = "Enter保存配置并预览 Esc返回版本"
	}
	if m.mode == "maintenance" {
		footer = "↑↓选择 Enter操作 Esc返回"
		if m.menuIndex == 0 {
			footer += " l锁定同步"
		}
		if m.menuIndex == 1 {
			footer += " d深度检查"
		}
	}
	if m.mode == "command" {
		footer = "↑↓选择 Enter编辑/运行 Delete删除参数 j高级JSON Esc返回"
	}
	if m.mode == "task" {
		footer = "Enter返回 j详情 PgUp/PgDn翻页 Esc返回"
		if t.State == "succeeded" && t.Request.Action == "sync" {
			footer = "Enter进入运行命令 j详情 Esc返回工作台"
		}
		if uiActive(t) {
			footer = "c取消并等待收尾 j详情 PgUp/PgDn翻页"
		}
	}
	if m.mode == "maint-confirm" || m.mode == "external-confirm" {
		footer = "Enter明确执行 Esc返回"
	}
	if m.editing {
		footer = "←→ Home/End移动 Delete删除 Ctrl+U清空 Enter保存 Esc取消"
	}
	if t.State == "confirm" {
		footer = "仅当前任务：y允许构建 / n拒绝 · c取消并等待收尾"
	}
	line(" " + footer)
	lines = append(lines, "└"+strings.Repeat("─", width-2)+"┘")
	content := strings.Join(lines, "\n")
	v := tea.NewView(content)
	v.AltScreen = true
	// The default theme leaves both colours unset; terminals retain their own palette.
	if m.theme == "light" {
		v.ForegroundColor = color.RGBA{0x21, 0x3a, 0x34, 255}
		v.BackgroundColor = color.RGBA{0xf5, 0xf4, 0xef, 255}
	} else if m.theme == "dark" {
		v.ForegroundColor = color.RGBA{0xed, 0xf4, 0xed, 255}
		v.BackgroundColor = color.RGBA{0x10, 0x1e, 0x1b, 255}
	}
	return v
}
