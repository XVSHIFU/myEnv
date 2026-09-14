package cli

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"time"
)

func addVersions(root *cobra.Command, jsonOutput *bool) {
	var search string
	var preview bool
	var channel, date, provider string
	var major int
	command := &cobra.Command{Use: "versions <python|node|java|go|rust>", Short: "查询发行版本目录", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		// Reject irrelevant options before platform checks or any network request.
		switch args[0] {
		case "python", "node", "java", "go", "rust":
		default:
			return fmt.Errorf("unsupported catalog tool %q", args[0])
		}
		if cmd.Flags().Changed("major") && (args[0] != "java" || major <= 0) {
			return fmt.Errorf("--major applies only to a positive Java major")
		}
		if cmd.Flags().Changed("provider") && (args[0] != "python" || provider != "python.org" && provider != "astral") {
			return fmt.Errorf("versions --provider currently accepts python.org or astral for Python")
		}
		if cmd.Flags().Changed("channel") || cmd.Flags().Changed("date") {
			if args[0] != "rust" || (channel != "beta" && channel != "nightly") {
				return fmt.Errorf("--channel beta|nightly and --date apply to Rust")
			}
			if cmd.Flags().Changed("date") {
				if _, err := time.Parse("2006-01-02", date); err != nil {
					return fmt.Errorf("--date requires a valid date in YYYY-MM-DD format")
				}
			}
		}
		data, err := QueryVersions(cmd.Context(), VersionQuery{Tool: args[0], Provider: provider, Preview: preview, Major: major, Channel: channel, Date: date, Search: search})
		if err != nil {
			return err
		}
		selected, coverage, platform := data.Releases, data.Coverage, data.Platform

		if *jsonOutput {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result{Schema: 1, OK: true, Data: map[string]any{"releases": selected, "coverage": coverage}})
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s / %s：%d 个发行条目\n", args[0], platform, len(selected))
		for _, r := range selected {
			if r.DisplayVersion != "" {
				label := r.DisplayVersion
				if r.Kind == "release_page" {
					label += "（只读发布记录）"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", r.Version, label, r.Provider, r.Channel)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", r.Version, r.Provider, r.Channel)
			}
		}
		if args[0] == "java" {
			fmt.Fprintln(cmd.OutOrStdout(), "仅 Eclipse Temurin / HotSpot JDK，列出各主版本近期发行，不是完整归档；可手动指定完整历史发行 ID。Java 8 即 JDK 1.8；支持搜索 1.8、jdk1.8、java8、8u。按项目兼容要求选择，预览版不默认安装。")
		}
		if args[0] == "python" {
			fmt.Fprintln(cmd.OutOrStdout(), "默认目录与默认安装均为 CPython / Astral（固定 uv）。--provider python.org 显式查询官网：Windows 为完整 ZIP，Linux 为发布记录，不代表可安装制品。")
		}
		fmt.Fprintln(cmd.OutOrStdout(), "用 myenv use <工具>@<版本> 安装；用户默认环境加 --global。预览版须显式填写完整版本或 Rust 渠道。")
		if args[0] == "rust" {
			fmt.Fprintln(cmd.OutOrStdout(), "Rust --preview 包含当前 beta/nightly；历史日期用 --channel nightly --date YYYY-MM-DD 查询，不伪造官网未提供的全历史索引。")
		}
		return nil
	}}
	command.Flags().StringVar(&search, "search", "", "按版本文本筛选")
	command.Flags().BoolVar(&preview, "preview", false, "包含官方预览版目录和 Rust 当前 beta/nightly")
	command.Flags().StringVar(&provider, "provider", "", "Python 目录来源：python.org 或 astral（需已有固定 uv）")
	command.Flags().StringVar(&channel, "channel", "", "Rust 渠道：beta 或 nightly")
	command.Flags().StringVar(&date, "date", "", "查询指定日期的 Rust 渠道：YYYY-MM-DD")
	command.Flags().IntVar(&major, "major", 0, "仅查询指定 Java 主版本，减少官网请求")
	root.AddCommand(command)
}
