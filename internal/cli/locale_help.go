package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"strings"
)

var chineseCommands = map[string][2]string{
	"myenv":      {"管理项目开发环境", "myEnv 管理项目及用户默认的 Python、Node、Java、Go、Rust 开发环境。\n\n日常流程：init → sync → run。用 use 修改版本，用 doctor 排查问题。\n直接运行 myenv 或 myenv status 查看状态。\n当前为开发版，Windows 和原生 Linux 功能已有验证，性能验收尚未完成；本次不支持 macOS。\n运行 myenv help manual 阅读完整离线手册。"},
	"status":     {"查看当前项目环境状态", "只读查看当前项目的配置和已应用环境，与直接运行 myenv 相同。"},
	"init":       {"识别版本声明并创建 myenv.yaml", "从 .python-version、pyproject.toml、.node-version、.nvmrc 和 package.json 识别版本。\n仅创建 myenv.yaml，不安装工具，也不覆盖已有配置。\n非交互模式下，缺少或冲突的声明返回 NEEDS_INPUT（退出码 3）。"},
	"sync":       {"准备并应用声明的环境", "在新环境中准备 Node/Python/Java/Go/Rust 及选定的 Python 项目，失败时保留上一环境。\n--locked 要求运行时和 Python 原生锁文件匹配，不改写锁文件。\n构建代码需要 --allow-build 或本次终端确认。\n--dry-run 不安装后端；尚未解析的 Python 版本在实际 sync 时确定。"},
	"use":        {"修改运行时版本并同步", "修改支持的运行时版本声明，随后执行与 sync 相同的流程。\n交互终端中省略参数会询问 tool@version；--json、--no-input 和非交互调用必须明确提供。\n准备失败时保留修改后的声明及之前已应用的环境。"},
	"run":        {"在已应用环境中执行命令", "使用已应用的环境运行命令，不隐式安装。\n保留命令参数；Windows 托管 Rust 在未配置链接器时默认使用自带 LLD。\n可用 --rust-linker=system 保留系统链接行为；--current 允许声明与已应用环境不同。\nnode/python/npm/npx 使用该环境的入口；其他命令从运行时 PATH 查找，也接受绝对或相对路径。\n管道和重定向由 Shell 处理，run 本身不解释这些语法。"},
	"rollback":   {"切换到保留的上一环境", "切换到完整的上一环境，不修改声明、锁文件、代码或数据。\n声明不匹配时，用 run --current 执行已回退的环境。"},
	"doctor":     {"检查环境和命令路径", "只读检查配置、锁文件、运行入口和 PATH。\n--deep 将环境代内容与记录的基线比较，可能较慢；不检查外部运行时内容、Python 字节码缓存或 ACL。\n检查不会修复环境；不能确认退出的进程仍受删除保护。"},
	"clean":      {"清理不再使用的项目环境", "保留当前、上一代、仍有运行保护记录及准备中的环境。\n--dry-run 仅预览候选项和逻辑字节数；实际清理持锁重新检查。\n有明确子进程完成证据的中断准备操作可恢复并清理；未知记录继续保留。\nWindows 仅在监督者与对应 Job 均确认退出后恢复相关记录。\n--cache node 清理共享 Node 下载缓存；--cache uv 调用已安装的固定 uv 清理缓存，不使用 --force。\nclean 不接受 --global，不卸载共享 Python 解释器。"},
	"help":       {"阅读命令帮助或离线手册", "myenv help <command> 查看命令帮助；myenv help manual 阅读完整手册。"},
	"completion": {"生成 Shell 补全脚本", "为 Bash、Zsh、Fish 或 PowerShell 生成补全脚本。\n仅输出脚本，不下载、不执行安装工具，也不修改 Shell 配置。生成的脚本保持原有格式。"},
	"shell-init": {"生成当前用户工具的 Shell 包装函数", "输出 bash、zsh、fish 或 powershell 的工具包装函数。\n需要先用 myenv use --global <tool>@<version> 创建用户默认环境。\n不会自动执行脚本或修改 Shell 配置；不支持 --json。"},
}

var chineseFlags = map[string]string{
	"directory": "项目上下文目录", "json": "输出一个结构化 JSON 结果（字段与消息保持英文合同）", "no-input": "不等待交互输入", "verbose": "向标准错误输出技术诊断",
	"lang": "显示语言：auto 自动、zh-CN 简体中文、en 英文", "help": "显示帮助", "version": "显示版本",
	"locked": "要求已有锁文件匹配，不修改锁文件", "dry-run": "仅预览，不修改环境、锁文件或删除内容", "rebuild": "即使快速检查无变化，也准备新环境",
	"update": "重新解析指定运行时版本（node、python、java、go、rust）", "allow-build": "允许本次以当前用户身份执行依赖和项目构建代码",
	"global": "使用当前用户的独立默认环境", "current": "使用已应用环境，允许当前声明已改变", "deep": "与记录的内容基线比较（可能较慢）", "cache": "清理指定共享缓存（node 或 uv）",
}

func localizeHelp(root *cobra.Command, lang locale) {
	if !lang {
		return
	}
	// Completion output remains based on the original English command metadata.
	var restore []func()
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		short, long := c.Short, c.Long
		restore = append(restore, func() { c.Short, c.Long = short, long })
		key := c.Name()
		if c.Parent() != nil && c.Parent() != root {
			key = strings.TrimPrefix(c.CommandPath(), root.Name()+" ")
		}
		if text, ok := chineseCommands[key]; ok {
			c.Short, c.Long = text[0], text[1]
		}
		c.InitDefaultHelpFlag()
		if c == root {
			c.InitDefaultVersionFlag()
		}
		translateFlag := func(f *pflag.Flag) {
			original := f.Usage
			restore = append(restore, func() { f.Usage = original })
			if text, ok := chineseFlags[f.Name]; ok {
				f.Usage = text
			}
		}
		c.LocalNonPersistentFlags().VisitAll(translateFlag)
		c.PersistentFlags().VisitAll(translateFlag)
		for _, child := range c.Commands() {
			walk(child)
		}
	}
	walk(root)
	for _, c := range root.Commands() {
		if c.Name() == "completion" {
			run := c.RunE
			c.RunE = func(cmd *cobra.Command, args []string) error {
				for _, restoreOne := range restore {
					restoreOne()
				}
				return run(cmd, args)
			}
		}
	}
	root.SetUsageFunc(func(c *cobra.Command) error {
		out := c.OutOrStdout()
		fmt.Fprintf(out, "用法：\n  %s\n", c.UseLine())
		if c.HasAvailableSubCommands() {
			fmt.Fprintf(out, "  %s <command>\n\n可用命令：\n", c.CommandPath())
			for _, child := range c.Commands() {
				if !child.Hidden {
					fmt.Fprintf(out, "  %-12s %s\n", child.Name(), child.Short)
				}
			}
		}
		if c.Example != "" {
			fmt.Fprintf(out, "\n示例：\n%s\n", c.Example)
		}
		flags := func(title string, fs *pflag.FlagSet) {
			if !fs.HasAvailableFlags() {
				return
			}
			fmt.Fprintf(out, "\n%s：\n", title)
			fs.VisitAll(func(f *pflag.Flag) {
				if f.Hidden {
					return
				}
				name := "--" + f.Name
				if f.Shorthand != "" {
					name = "-" + f.Shorthand + ", " + name
				}
				if f.Value.Type() != "bool" {
					name += " <" + f.Value.Type() + ">"
				}
				fmt.Fprintf(out, "  %s\n      %s", name, f.Usage)
				if f.DefValue != "" && f.DefValue != "false" {
					fmt.Fprintf(out, "（默认：%s）", f.DefValue)
				}
				fmt.Fprintln(out)
			})
		}
		flags("参数", c.LocalFlags())
		flags("全局参数", c.InheritedFlags())
		return nil
	})
	root.SetHelpFunc(func(c *cobra.Command, args []string) {
		text := c.Long
		if text == "" {
			text = c.Short
		}
		fmt.Fprintln(c.OutOrStdout(), text+"\n")
		_ = c.Usage()
	})
}
