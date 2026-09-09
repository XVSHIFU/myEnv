package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"myenv/internal/core"
)

func addSystem(root *cobra.Command, namespace string, jsonOutput *bool, lang locale, record func(any, bool)) {
	system := &cobra.Command{Use: "system", Short: "查看本机环境，管理当前用户的 myenv 默认环境"}
	addExternal(system, namespace, jsonOutput, record)
	for _, name := range []string{"list", "doctor"} {
		name := name
		var extraPaths []string
		var details bool
		command := &cobra.Command{Use: name, Short: "发现已安装环境并检查可执行入口", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
			record(nil, false)
			inventory, err := (&core.Service{UserConfigDirectory: namespace}).Inventory(cmd.Context(), true, extraPaths...)
			if name == "doctor" && err == nil {
				inventory.Compilers = core.CompilerPrerequisites(cmd.Context())
			}
			record(inventory, false)
			if err != nil {
				return err
			}
			if *jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result{Schema: 1, OK: true, Data: inventory})
			}
			writeInventory(cmd.OutOrStdout(), inventory, details || name == "doctor", lang)
			for _, w := range inventory.Warnings {
				fmt.Fprintln(cmd.ErrOrStderr(), w)
			}
			for _, c := range inventory.Compilers {
				fmt.Fprintf(cmd.OutOrStdout(), "%s：%s %s\n  %s\n", c.Name, lang.sdkLabel(c.State), c.Path, c.Advice)
			}

			return nil
		}}
		command.Flags().StringArrayVar(&extraPaths, "path", nil, "补充自定义安装路径，可重复指定")
		command.Flags().BoolVar(&details, "details", false, "显示完整路径、安装 ID 与原始版本信息")
		system.AddCommand(command)
		if name == "list" {
			shortcut := &cobra.Command{Use: "list", Short: "列出本机开发环境（system list 的快捷入口）", Args: cobra.NoArgs, RunE: command.RunE}
			shortcut.Flags().AddFlagSet(command.Flags())
			root.AddCommand(shortcut)
		}
	}
	for _, action := range []string{"install", "upgrade", "repair", "remove"} {
		action := action
		var provider string
		use := action + " <tool>"
		if action == "install" {
			use = action + " <tool>@<version>"
		}
		command := &cobra.Command{Use: use, Short: "管理 myenv 拥有的当前用户默认环境", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			record(nil, false)
			service, err := runtimeService(true, namespace)
			if err != nil {
				return err
			}
			defer closeRuntimeService(service)
			var data any
			changed := false
			if action == "remove" {
				r, e := service.RemoveDefault(cmd.Context(), args[0])
				data = r
				err = e
				changed = r.Changed || r.DeclarationChanged
			} else if action == "install" {
				fmt.Fprintln(cmd.ErrOrStderr(), lang.text("Checking environment..."))
				selection, e := providerSelection(args[0], provider)
				if e != nil {
					return e
				}
				result, e := service.UseWithRequest(cmd.Context(), core.UseRequest{Selection: selection, Progress: syncProgress(cmd.ErrOrStderr(), lang)})
				data = result
				err = e
				changed = result.Changed || result.DeclarationChanged || result.LockChanged || result.NativeLockChanged
			} else {
				if strings.Contains(args[0], "@") {
					return fmt.Errorf("upgrade/repair accepts a tool name; select a version with system install")
				}
				status, e := service.Status(cmd.Context(), "")
				if e != nil {
					return e
				}
				if len(status.Tools) == 0 {
					return fmt.Errorf("no managed user profile")
				}
				if status.Tools[args[0]] == "" {
					return fmt.Errorf("tool is not declared in the myenv user profile; external installations are read-only")
				}
				request := core.SyncRequest{Progress: syncProgress(cmd.ErrOrStderr(), lang)}
				if action == "upgrade" {
					request.Update = args[0]
				} else {
					request.Rebuild = true
				}
				result, e := service.Sync(cmd.Context(), request)
				data = result
				err = e
				changed = result.Changed || result.LockChanged || result.NativeLockChanged
			}
			record(data, changed)
			if err != nil {
				return err
			}
			if *jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result{Schema: 1, OK: true, Changed: changed, Data: data})
			}
			if action == "remove" {
				fmt.Fprintln(cmd.OutOrStdout(), "已从默认环境移除；旧文件按回退与运行保护规则保留，可用 system clean 检查可回收内容。")
				return nil
			}
			writeSystemSuccess(cmd.OutOrStdout(), strings.SplitN(args[0], "@", 2)[0], changed, lang)
			return nil
		}}
		if action == "repair" {
			command.Long = "在新环境代重建当前用户默认环境，成功后切换。会重建该默认环境中的全部工具；不修复外部安装。"
		}
		if action == "install" {
			command.Flags().StringVar(&provider, "provider", "", "Python 来源：python.org 或 astral")
		}
		system.AddCommand(command)
	}
	var dryRun bool
	clean := &cobra.Command{Use: "clean", Short: "按引用与运行保护规则清理 myenv 用户默认环境", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		service := core.Service{Profile: true, UserConfigDirectory: namespace}
		data, err := service.Clean(cmd.Context(), "", dryRun, nil)
		record(data, data.Changed)
		if err != nil {
			return err
		}
		if *jsonOutput {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result{Schema: 1, OK: true, Changed: data.Changed, Data: data})
		}
		fmt.Fprintf(cmd.OutOrStdout(), "可回收：%d；已移除：%d。当前代、回退代及运行中环境保留。\n", data.Candidates, data.Removed)
		return nil
	}}
	clean.Flags().BoolVar(&dryRun, "dry-run", false, "仅预览")
	system.AddCommand(clean)
	root.AddCommand(system)
}
