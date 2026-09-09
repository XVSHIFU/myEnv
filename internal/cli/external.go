package cli

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"myenv/internal/core"
)

func addExternal(system *cobra.Command, namespace string, jsonOutput *bool, record func(any, bool)) {
	var action, manager string
	var apply, dry bool
	var paths []string
	command := &cobra.Command{Use: "external <installation-id>", Short: "通过原管理器处理外部安装；默认仅预览", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		record(nil, false)
		inventory, e := (&core.Service{UserConfigDirectory: namespace}).Inventory(cmd.Context(), false, paths...)
		if e != nil {
			return e
		}
		var row *core.Installation
		for i := range inventory.Installations {
			if inventory.Installations[i].ID == args[0] {
				row = &inventory.Installations[i]
				break
			}
		}
		if row == nil {
			return fmt.Errorf("installation ID not found; run system list with the same --path")
		}
		plan, e := core.BuildExternalPlan(cmd.Context(), *row, manager, action)
		if e != nil {
			return e
		}
		record(plan, false)
		if !apply {
			if *jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result{Schema: 1, OK: true, Data: plan})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "目标：%s\n管理器：%s\n参数：%q\n%s\n当前仅预览；核对后添加 --apply 执行。\n", row.Path, plan.Manager, plan.Args, plan.Notice)
			return nil
		}
		record(plan, true)
		output := cmd.OutOrStdout()
		if *jsonOutput {
			output = cmd.ErrOrStderr()
		}
		if e = core.ApplyExternalPlan(cmd.Context(), plan, cmd.InOrStdin(), output, cmd.ErrOrStderr()); e != nil {
			return e
		}
		if *jsonOutput {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result{Schema: 1, OK: true, Changed: true, Data: plan})
		}
		fmt.Fprintln(cmd.OutOrStdout(), "原管理器操作已完成；请重新运行 system doctor 检查。")
		return nil
	}}
	command.Flags().StringVar(&action, "action", "", "upgrade、repair 或 remove（能力取决于原管理器）")
	command.Flags().StringVar(&manager, "manager", "", "原管理器可执行文件绝对路径")
	command.Flags().StringArrayVar(&paths, "path", nil, "补充安装路径，与 system list 保持一致")
	command.Flags().BoolVar(&apply, "apply", false, "执行对外部安装的修改；默认仅预览")
	command.Flags().BoolVar(&dry, "dry-run", false, "仅预览（默认行为）")
	command.MarkFlagsMutuallyExclusive("apply", "dry-run")
	system.AddCommand(command)
}
