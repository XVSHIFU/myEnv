package cli

import (
	tea "charm.land/bubbletea/v2"
	"encoding/json"
	"fmt"
)

func (m *tuiModel) prepareInput() {
	if !m.inputStarted {
		m.cursor = len([]rune(m.input))
		m.inputStarted = true
	}
}
func (m *tuiModel) insertInput(s string) {
	m.prepareInput()
	r := []rune(m.input)
	n := []rune(s)
	next := append([]rune{}, r[:m.cursor]...)
	next = append(next, n...)
	next = append(next, r[m.cursor:]...)
	m.input = string(next)
	m.cursor += len(n)
}
func (m *tuiModel) beginEdit(focus int, value string) {
	m.focus = focus
	m.input = value
	m.editing = true
	m.inputStarted = false
}
func (m *tuiModel) startFlow(r UIRequest) {
	if uiActive(m.controller.Current()) {
		m.message = "请等待当前任务收尾"
		return
	}
	m.preview = nil
	m.message = ""
	m.scroll = 0
	if _, err := m.controller.Start(r); err != nil {
		m.message = err.Error()
		return
	}
	m.mode = "task"
	m.versionMode = false
}
func (m *tuiModel) queryFlow() { r := m.request; r.Action = "versions"; m.startFlow(r) }

var maintenanceActions = []string{"sync", "doctor", "repair", "rollback", "clean", "inventory", "upgrade", "remove", "external"}
var maintenanceLabels = []string{"预览整个环境", "诊断", "重建修复（先预览）", "回退到前一环境", "清理预览", "本机环境", "选择默认工具新版本", "移除默认工具", "外部管理（高级参数）"}

func (m *tuiModel) flowUpdate(msg tea.Msg) (bool, tea.Cmd) {
	if _, ok := msg.(tuiUpdate); ok {
		t := m.controller.Current()
		if t.Request.Directory != m.request.Directory || t.Request.Global != m.request.Global {
			return false, nil
		}
		if !uiActive(t) && t.ID > m.handledTask {
			m.handledTask = t.ID
			follow := !m.editing && (m.mode == "task" || m.mode == "")
			if t.View != nil {
				m.workbench = t.View
			}
			if t.State == "succeeded" && t.Request.Action == "versions" {
				var c VersionResult
				if json.Unmarshal(t.Result, &c) == nil {
					m.catalog = &c
					m.catalogRequest = t.Request
					if follow {
						m.mode = ""
						m.versionMode = true
						m.versionIndex = 0
					}
					return true, nil
				}
			}
			if t.State != "succeeded" && t.Request.Action == "versions" {
				if follow {
					m.mode = "version-setup"
					m.menuIndex = 0
				}
				return true, nil
			}
			if m.initializing && t.Request.Action == "init" {
				m.initializing = false
				if t.State == "succeeded" {
					r := t.Request
					r.Action = "desired"
					m.startFlow(r)
					return true, nil
				}
			}
			if t.State == "succeeded" && (t.Request.Action == "desired" || t.Request.Action == "sync" && t.Request.Preview || t.Request.Action == "clean" && !t.Request.Apply || t.Request.Action == "external" && !t.Request.Apply) {
				r := t.Request
				if r.Action == "external" {
					r.PlanTaskID = t.ID
				}
				m.preview = &r
				if t.View != nil {
					m.previewDigest = t.View.Digest
				}
				if follow {
					m.mode, m.scroll = "preview", 0
				}
			}
		}
		return true, nil
	}
	k, ok := msg.(tea.KeyPressMsg)
	if !ok || m.editing {
		return false, nil
	}
	key := k.String()
	if m.mode == "version-setup" {
		switch key {
		case "esc":
			m.mode = ""
		case "up":
			m.menuIndex = (m.menuIndex + len(m.versionSetupOptions()) - 1) % len(m.versionSetupOptions())
		case "down":
			m.menuIndex = (m.menuIndex + 1) % len(m.versionSetupOptions())
		case "p":
			m.request.Preview = !m.request.Preview
		case "enter":
			switch m.menuIndex {
			case 0:
				m.queryFlow()
			case 1:
				m.beginEdit(-1, m.request.Selection)
			case 2:
				m.togglePythonSource()
			}
		case "o":
			m.togglePythonSource()
		case "i":
			m.beginEdit(-1, m.request.Selection)
		}
		return true, nil
	}
	if m.versionMode {
		switch key {
		case "i":
			m.beginEdit(-1, m.request.Selection)
			return true, nil
		case "p":
			m.request.Preview = !m.request.Preview
			m.queryFlow()
			return true, nil
		case "o":
			if m.request.Tool == "python" {
				if m.request.Provider == "astral" {
					m.request.Provider = "python.org"
				} else {
					m.request.Provider = "astral"
				}
				m.filter = ""
				m.queryFlow()
			}
			return true, nil
		case "esc":
			m.versionMode = false
			m.mode = ""
			return true, nil
		}
		return false, nil
	}
	if m.mode == "save" {
		switch key {
		case "esc":
			m.mode = ""
			m.versionMode = m.catalog != nil
			if !m.versionMode {
				m.mode = "version-setup"
			}
		case "enter":
			r := m.request
			r.Action = "desired"
			if m.workbench != nil && m.workbench.Empty && !r.Global {
				r.Action = "init"
				m.initializing = true
			}
			m.startFlow(r)
		}
		return true, nil
	}
	if m.mode == "preview" && key == "esc" {
		m.mode = ""
		m.versionMode = m.catalog != nil
		return true, nil
	}
	if m.mode == "run" && key == "esc" {
		m.mode = "command"
		return true, nil
	}
	if m.mode == "task" && key == "esc" {
		if uiActive(m.controller.Current()) {
			m.message = "任务仍在进行；按 c 取消并等待收尾"
			return true, nil
		}
		m.mode = ""
		return true, nil
	}
	if m.mode == "preview" && key == "enter" {
		return m.workbenchUpdate(tea.KeyPressMsg{Code: 's', Text: "s"})
	}
	if m.mode == "task" && key == "enter" && !uiActive(m.controller.Current()) {
		if m.preview != nil {
			m.mode = "preview"
			return true, nil
		}
		if m.controller.Current().State == "succeeded" && m.controller.Current().Request.Action == "sync" {
			m.mode = "command"
			m.menuIndex = 0
			return true, nil
		}
		m.mode = ""
		return true, nil
	}
	if m.mode == "command" {
		switch key {
		case "esc":
			m.mode = ""
		case "home":
			m.menuIndex = 0
		case "end":
			m.menuIndex = len(m.runArgs) + 1
		case "up":
			m.menuIndex = max(0, m.menuIndex-1)
		case "down":
			m.menuIndex = min(len(m.runArgs)+1, m.menuIndex+1)
		case "enter":
			if m.menuIndex < len(m.runArgs) {
				m.beginEdit(6+m.menuIndex, m.runArgs[m.menuIndex])
			} else if m.menuIndex == len(m.runArgs) {
				m.beginEdit(6+m.menuIndex, "")
			} else if !uiActive(m.controller.Current()) {
				m.mode = "run"
				m.scroll = 0
			}
		case "delete":
			if m.menuIndex > 0 && m.menuIndex < len(m.runArgs) {
				m.runArgs = append(m.runArgs[:m.menuIndex], m.runArgs[m.menuIndex+1:]...)
			}
		case "j":
			b, _ := json.Marshal(m.runArgs)
			m.beginEdit(4, string(b))
		}
		return true, nil
	}
	if m.mode == "maintenance" {
		switch key {
		case "esc":
			m.mode = ""
		case "up":
			m.menuIndex = (m.menuIndex + len(maintenanceActions) - 1) % len(maintenanceActions)
		case "down":
			m.menuIndex = (m.menuIndex + 1) % len(maintenanceActions)
		case "enter":
			r := m.request
			r.Action = maintenanceActions[m.menuIndex]
			r.Preview = false
			r.Apply = false
			if !r.Global && (r.Action == "upgrade" || r.Action == "remove") {
				m.message = "此项仅用于用户默认环境；Esc返回后按g切换作用域"
				return true, nil
			}
			switch r.Action {
			case "sync":
				r.Preview = true
			case "repair":
				r.Action = "sync"
				r.Preview = true
				r.Rebuild = true
			case "upgrade":
				m.queryFlow()
				return true, nil
			case "remove", "rollback":
				m.mode = "maint-confirm"
				return true, nil
			case "external":
				r.Apply = false
				b, _ := json.Marshal(r)
				m.beginEdit(3, string(b))
				m.mode = "external-confirm"
				return true, nil
			}
			m.startFlow(r)
		case "l":
			if m.menuIndex == 0 {
				m.request.Locked = !m.request.Locked
			}
		case "d":
			if m.menuIndex == 1 {
				m.request.Deep = !m.request.Deep
			}
		}
		return true, nil
	}
	if m.mode == "maint-confirm" || m.mode == "external-confirm" {
		if key == "esc" {
			m.mode = "maintenance"
		} else if key == "enter" {
			r := m.request
			r.Action = maintenanceActions[m.menuIndex]
			if r.Action == "external" {
				r.Apply = false
			}
			m.startFlow(r)
		}
		return true, nil
	}
	if m.mode != "" {
		return false, nil
	}
	switch key {
	case "up":
		m.homeIndex = (m.homeIndex + 7) % 8
	case "down", "tab":
		m.homeIndex = (m.homeIndex + 1) % 8
	case "enter":
		if uiActive(m.controller.Current()) {
			m.message = "请等待当前任务收尾"
			return true, nil
		}
		if m.homeIndex < 5 {
			m.selectTool(uiTools[m.homeIndex])
			m.mode = "version-setup"
			m.menuIndex = 0
		} else if m.homeIndex == 5 {
			m.mode = "command"
			m.menuIndex = 0
		} else if m.homeIndex == 6 {
			m.mode = "maintenance"
			m.menuIndex = 0
		} else {
			m.mode = "menu"
			m.menuIndex = 0
		}
	case "g":
		m.switchProject(m.request.Directory, !m.request.Global)
	case "m", "?":
		m.mode = "menu"
		m.menuIndex = 0
	case "1":
		m.beginEdit(0, m.request.Directory)
	case "c":
		if uiActive(m.controller.Current()) {
			m.controller.Cancel(m.controller.Current().ID)
		}
	default:
		return true, nil
	}
	return true, nil
}
func (m *tuiModel) flowBody() []string {
	switch m.mode {
	case "version-setup":
		body := []string{m.request.Tool + " · 选择版本（尚不保存）", fmt.Sprintf("预发布版本：%t [p]", m.request.Preview)}
		for i, label := range m.versionSetupOptions() {
			mark := "  "
			if m.menuIndex == i {
				mark = "> "
			}
			body = append(body, mark+label)
		}
		if m.request.Tool == "python" {
			body = append(body, "当前来源："+m.request.Provider, "Astral 查询需要已有 uv；首次也可手动输入后预览。", "python.org 官网发布记录不等于当前平台可安装制品。")
		}
		return append(body, "查询失败可重试或手动输入；保存后仍需预览并确认。")
	case "save":
		verb := "保存当前声明"
		if m.workbench != nil && m.workbench.Empty && !m.request.Global {
			verb = "在本目录创建 myenv.yaml"
		}
		return []string{verb, "目标：" + m.request.Selection, "来源：" + m.request.Provider, "Enter明确保存并预览整个环境；此步不安装。", "Esc返回版本列表，保留筛选与选择。"}
	case "command":
		body := []string{"逐个编辑程序和参数；不要求 JSON，不经 shell 展开。"}
		for i, a := range m.runArgs {
			mark := "  "
			if i == m.menuIndex {
				mark = "> "
			}
			label := fmt.Sprintf("参数 %d", i)
			if i == 0 {
				label = "程序"
			}
			body = append(body, mark+label+"："+a)
		}
		for i, label := range []string{"添加参数", "运行命令 → 确认并交接终端"} {
			mark := "  "
			if m.menuIndex == len(m.runArgs)+i {
				mark = "> "
			}
			body = append(body, mark+label)
		}
		return body
	case "maintenance":
		body := []string{}
		for i, label := range maintenanceLabels {
			mark := "  "
			if m.menuIndex == i {
				mark = "> "
			}
			if i == 0 {
				label += fmt.Sprintf(" · 锁定 %t [l]", m.request.Locked)
			}
			if i == 1 {
				label += fmt.Sprintf(" · 深度 %t [d]", m.request.Deep)
			}
			body = append(body, mark+label)
		}
		return body
	case "maint-confirm", "external-confirm":
		return []string{"将执行：" + maintenanceLabels[m.menuIndex], "固定目录：" + m.request.Directory, "Enter明确执行，Esc返回；删除默认工具会修改声明。"}
	}
	return nil
}
func (m *tuiModel) homeBody() []string {
	status := "读取中"
	if m.workbench != nil && m.workbench.Empty {
		status = "空项目，尚未保存配置"
	}
	if m.workbench != nil && m.workbench.Status != nil {
		status = uiEnvironment(m.workbench.Status.Environment)
	}
	body := []string{"环境：" + status, "选择语言 → 选择版本 → 保存并预览 → 确认同步"}
	for i, t := range uiTools {
		mark := "  "
		if i == m.homeIndex {
			mark = "> "
		}
		want, have := "未配置", "未安装"
		if m.workbench != nil {
			if s := m.workbench.Desired[t]; s != "" {
				want = s
			}
			l := m.workbench.Applied[t]
			if l.Version != "" {
				have = l.Version
			}
		}
		body = append(body, mark+fmt.Sprintf("%-7s 生效 %s · 配置 %s", t, have, want))
		if m.height >= 30 && m.workbench != nil {
			body = append(body, "    "+uiToolState(m.workbench, t))
		}
	}
	for i, s := range []string{"运行命令", "更多操作 / 维护", "项目、来源与设置"} {
		mark := "  "
		if m.homeIndex == 5+i {
			mark = "> "
		}
		body = append(body, mark+s)
	}
	args, _ := json.Marshal(m.runArgs)
	body = append(body, "当前命令："+string(args))
	if m.workbench != nil && m.workbench.Empty {
		body = append(body, "空目录：尚未写入配置；选择版本后明确确认创建。")
	}
	return body
}
func (m *tuiModel) inputDisplay() string {
	m.prepareInput()
	r := []rune(m.input)
	return string(r[:m.cursor]) + "▏" + string(r[m.cursor:])
}

func (m *tuiModel) togglePythonSource() {
	if m.request.Tool != "python" {
		return
	}
	if m.request.Provider == "astral" {
		m.request.Provider = "python.org"
	} else {
		m.request.Provider = "astral"
	}
	m.request.Selection = ""
	m.filter = ""
	m.catalog = nil
}

func (m *tuiModel) versionSetupOptions() []string {
	options := []string{"查询 / 重试版本目录", "手动输入期望版本"}
	if m.request.Tool == "python" {
		options = append(options, "切换来源：Astral / python.org")
	}
	return options
}

func (m *tuiModel) previewConfirm() string {
	if m.preview != nil {
		switch m.preview.Action {
		case "clean":
			return "确认清理"
		case "external":
			return "确认原管理器执行"
		}
	}
	return "确认同步"
}
func (m *tuiModel) previewTitle() string {
	if m.preview != nil {
		switch m.preview.Action {
		case "clean":
			return "清理预览 · 仅删除核心允许的对象"
		case "external":
			return "外部管理计划 · 核验后明确执行"
		}
	}
	return "环境变更 · 尚未应用"
}
