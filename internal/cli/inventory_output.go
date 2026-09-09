package cli

import (
	"fmt"
	"io"
	"myenv/internal/core"
	"regexp"
	"strings"
)

// Only the presentation extracts a compact version; inventory evidence stays intact.
var displayVersion = regexp.MustCompile(`[0-9]+\.[0-9]+(?:\.[0-9]+)*(?:[a-zA-Z][0-9]+|-[0-9A-Za-z.+-]+)?`)

func uiText(lang locale, zh, en string) string {
	if lang {
		return zh
	}
	return en
}

func writeInventory(out io.Writer, inventory core.Inventory, details bool, lang locale) {
	fmt.Fprintln(out, uiText(lang, "本机开发环境", "Local development environments"))
	available := 0
	for _, row := range inventory.Installations {
		if row.State == "available" {
			available++
		}
	}
	fmt.Fprintln(out, uiText(lang, fmt.Sprintf("发现 %d 个入口 · %d 个可用\n", len(inventory.Installations), available), fmt.Sprintf("%d entries · %d available\n", len(inventory.Installations), available)))
	groups := []struct {
		title, english string
		match          func(core.Installation) bool
	}{
		{"myenv 管理", "Managed by myenv", func(r core.Installation) bool { return r.Owner == "myenv" }},
		{"外部安装", "External installations", func(r core.Installation) bool { return r.Owner != "myenv" && r.State == "available" }},
		{"其他入口 / 需关注", "Other entries / needs attention", func(r core.Installation) bool { return r.Owner != "myenv" && r.State != "available" }},
	}
	for _, group := range groups {
		heading := false
		for _, row := range inventory.Installations {
			if !group.match(row) {
				continue
			}
			if !heading {
				fmt.Fprintln(out, paint(out, "1;36", uiText(lang, group.title, group.english)))
				heading = true
			}
			version := displayVersion.FindString(row.Version)
			if version != "" {
				version = " " + version
			}
			manager := row.Manager
			if manager == "unknown" {
				manager = uiText(lang, "来源未识别", "source unknown")
			}
			fmt.Fprintf(out, "  %s%s  [%s]  %s\n", row.Tool, version, paint(out, stateColor(row.State), lang.sdkLabel(row.State)), manager)
			if details {
				fmt.Fprintf(out, "    ID: %s\n    %s\n", row.ID, row.Path)
				if row.Version != "" {
					fmt.Fprintln(out, "    "+strings.Join(strings.Fields(row.Version), " "))
				}
			}
			if row.State == "alias" {
				fmt.Fprintln(out, uiText(lang, "    Windows 应用执行别名，尚未验证为可用运行时。", "    Windows execution alias; runtime availability is unverified."))
			} else if row.State == "shim" {
				fmt.Fprintln(out, uiText(lang, "    管理器转发入口；实际工具链单独列出。", "    Manager dispatcher; discovered toolchains are listed separately."))
			} else if row.Problem != "" {
				fmt.Fprintln(out, "    "+strings.Join(strings.Fields(row.Problem), " "))
			}
		}
		if heading {
			fmt.Fprintln(out)
		}
	}
	if !details {
		fmt.Fprintln(out, uiText(lang, "完整路径与安装 ID：myenv system list --details", "Paths and installation IDs: myenv system list --details"))
	}
	fmt.Fprintln(out, uiText(lang, "运行示例：myenv run --global java -version", "Run example: myenv run --global java -version"))
	fmt.Fprintln(out, uiText(lang, "扫描范围：PATH、已知位置及登记信息；自定义目录可用 --path 补充。", "Scope: PATH, known locations and registrations; add custom locations with --path."))
	if details {
		fmt.Fprintln(out, uiText(lang, "外部安装操作：myenv system external --help", "External installation actions: myenv system external --help"))
	}
}

func writeSystemSuccess(out io.Writer, tool string, changed bool, lang locale) {
	message := uiText(lang, "已更新用户默认环境", "User default environment updated")
	if !changed {
		message = uiText(lang, "用户默认环境已就绪，无需更改", "User default environment is already up to date")
	}
	fmt.Fprintf(out, "%s：%s\n\n", paint(out, "32", message), tool)
	command := map[string]string{"java": "java -version", "python": "python --version", "node": "node --version", "go": "go version", "rust": "rustc --version"}[tool]
	if command != "" {
		fmt.Fprintf(out, "  myenv run --global %s\n\n", command)
	}
	fmt.Fprintln(out, uiText(lang, "查看环境：myenv list\n作用范围：当前用户的 myenv 默认环境；系统 PATH 未修改。", "List environments: myenv list\nScope: current user's myenv default environment; system PATH unchanged."))
}
