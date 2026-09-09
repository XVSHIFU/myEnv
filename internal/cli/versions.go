package cli

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"myenv/internal/backend"
	"myenv/internal/runner"
	"os"
	"strings"
)

func addVersions(root *cobra.Command, jsonOutput *bool) {
	var search string
	var preview bool
	var channel, date, provider string
	var major int
	command := &cobra.Command{Use: "versions <python|node|java|go|rust>", Short: "查询官方发行版本", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		platform, err := runner.Platform()
		if err != nil {
			return err
		}
		client, err := backend.DownloadClient(os.Getenv("SSL_CERT_FILE"))
		if err != nil {
			return err
		}
		defer client.CloseIdleConnections()
		if major < 0 || (major != 0 && args[0] != "java") {
			return fmt.Errorf("--major applies only to a positive Java major")
		}
		catalog := backend.Catalog{Client: client, IncludePreview: preview, JavaMajor: major}
		var rows []backend.CatalogRelease
		if channel != "" || date != "" {
			if args[0] != "rust" || (channel != "beta" && channel != "nightly") {
				return fmt.Errorf("--channel beta|nightly and --date apply to Rust")
			}
			selector := channel
			if date != "" {
				selector += "-" + date
			}
			r, e := catalog.RustChannel(cmd.Context(), selector, platform)
			err = e
			rows = []backend.CatalogRelease{r}
		} else if args[0] == "python" && (provider == "python.org" || (provider == "" && platform == "windows-amd64")) {
			rows, err = catalog.OfficialPython(cmd.Context(), platform)
		} else {
			if provider != "" {
				return fmt.Errorf("versions --provider currently accepts python.org for Python")
			}
			rows, err = catalog.List(cmd.Context(), args[0], platform)
		}
		if err != nil {
			return err
		}
		selected := []backend.CatalogRelease{}
		for _, r := range rows {
			if !preview && channel == "" && r.Channel != "stable" {
				continue
			}
			if !strings.Contains(strings.ToLower(r.Version), strings.ToLower(search)) {
				continue
			}
			selected = append(selected, r)
		}
		coverage := "current-platform upstream archives; preview installation requires an explicit preview version"
		if args[0] == "python" {
			coverage = "python.org full Windows x64 runtime archives; use --provider python.org to install; other platforms only show release pages"
		}
		if args[0] == "rust" {
			coverage += "; beta/nightly current manifests with --preview; query historical dates with --channel and --date"
		}
		if args[0] == "java" {
			coverage += "; Eclipse Temurin only, not all JDK vendors"
		}
		if *jsonOutput {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result{Schema: 1, OK: true, Data: map[string]any{"releases": selected, "coverage": coverage}})
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s / %s：%d 个发行条目\n", args[0], platform, len(selected))
		for _, r := range selected {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", r.Version, r.Provider, r.Channel)
		}
		if args[0] == "python" {
			fmt.Fprintln(cmd.OutOrStdout(), "Windows 默认列 python.org 完整运行时 ZIP；安装时添加 --provider python.org。Linux 官网发布记录不等同于可用二进制包，可显式选择 astral。")
		}
		fmt.Fprintln(cmd.OutOrStdout(), "用 myenv use <工具>@<版本> 安装；用户默认环境加 --global。预览版须显式填写完整版本或 Rust 渠道。")
		if args[0] == "rust" {
			fmt.Fprintln(cmd.OutOrStdout(), "Rust --preview 包含当前 beta/nightly；历史日期用 --channel nightly --date YYYY-MM-DD 查询，不伪造官网未提供的全历史索引。")
		}
		return nil
	}}
	command.Flags().StringVar(&search, "search", "", "按版本文本筛选")
	command.Flags().BoolVar(&preview, "preview", false, "包含官方预览版目录和 Rust 当前 beta/nightly")
	command.Flags().StringVar(&provider, "provider", "", "Python 目录来源：python.org")
	command.Flags().StringVar(&channel, "channel", "", "Rust 渠道：beta 或 nightly")
	command.Flags().StringVar(&date, "date", "", "查询指定日期的 Rust 渠道：YYYY-MM-DD")
	command.Flags().IntVar(&major, "major", 0, "仅查询指定 Java 主版本，减少官网请求")
	root.AddCommand(command)
}
