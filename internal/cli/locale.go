package cli

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"myenv/internal/core"
)

// A locale belongs to one invocation. It never wraps a child's output streams.
type locale bool

func (l locale) text(english string) string {
	if l {
		if translated, ok := chineseText[english]; ok {
			return translated
		}
	}
	return english
}

// Scan only myEnv's options, including options before run. Once the executable
// begins, every remaining argument belongs to it (even --lang or --json).
func languageOption(args []string) (string, bool) {
	value, found, inRun := "", false, false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if arg == "--lang" {
			found = true
			value = ""
			if i+1 < len(args) {
				i++
				value = args[i]
			}
			continue
		}
		if strings.HasPrefix(arg, "--lang=") {
			value, found = strings.TrimPrefix(arg, "--lang="), true
			continue
		}
		if arg == "-C" || arg == "--directory" || arg == "--update" || arg == "--cache" || arg == "--rust-linker" || arg == "--provider" {
			i++
			continue
		}
		if inRun && !strings.HasPrefix(arg, "-") {
			break
		}
		if arg == "run" {
			inRun = true
		}
	}
	return value, found
}

func selectLocale(args []string) (locale, error) {
	value, explicit := languageOption(args)
	if !explicit {
		value = os.Getenv("MYENV_LANG")
	}
	if value == "" && !explicit {
		value = "auto"
	}
	switch value {
	case "zh-CN":
		return true, nil
	case "en":
		return false, nil
	case "auto":
		return locale(systemChinese()), nil
	default:
		return false, fmt.Errorf("unsupported language %q; use --lang auto, zh-CN or en", value)
	}
}

func (l locale) status(s *core.Status) string {
	if !l {
		return fmt.Sprintf("Project: %s\nConfigured tools: %v\nEnvironment: %s\n%s", s.Project, s.Tools, s.Environment, s.NextAction)
	}
	keys := make([]string, 0, len(s.Tools))
	for k := range s.Tools {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	tools := make([]string, 0, len(keys))
	for _, k := range keys {
		tools = append(tools, k+" "+s.Tools[k])
	}
	return fmt.Sprintf("项目：%s\n配置工具：%s\n环境状态：%s\n%s", s.Project, strings.Join(tools, "，"), l.text(s.Environment), l.text(s.NextAction))
}

func (l locale) failure(out io.Writer, f *failure) {
	if !l {
		fmt.Fprintf(out, "%s: %s\n%s\n", f.Code, f.Message, f.NextAction)
		return
	}
	summaries := map[string]string{
		"USAGE_ERROR": "命令或参数不正确", "INVALID_CONFIG": "项目配置无法使用", "IO_ERROR": "文件或环境访问失败",
		"RUN_FAILED": "命令执行失败，环境保护按完成证据保留", "ENV_NOT_READY": "环境尚未就绪", "INPUT_CHANGED": "项目输入已改变",
		"LOCK_OUT_OF_DATE": "锁文件与当前声明不一致", "DOWNLOAD_FAILED": "下载失败", "CHECKSUM_MISMATCH": "下载内容校验不通过",
		"SYNC_FAILED": "同步失败", "NEEDS_INPUT": "需要补充输入", "CANCELED": "操作已取消", "CLEAN_FAILED": "清理未全部完成",
	}
	summary := summaries[f.Code]
	if summary == "" {
		summary = "操作失败"
	}
	action := l.text(f.NextAction)
	if action == f.NextAction {
		action = "请根据原始原因检查配置和环境后重试。处理建议（原文）：" + f.NextAction
	}
	fmt.Fprintf(out, "%s：%s\n原始原因：%s\n%s\n", f.Code, summary, f.Message, action)
}
