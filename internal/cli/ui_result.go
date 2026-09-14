package cli

import (
	"encoding/json"
	"fmt"
	"myenv/internal/config"
	"myenv/internal/core"
	"sort"
)

type UIChange struct {
	Tool string `json:"tool"`
	From string `json:"from"`
	To   string `json:"to"`
}
type UISummary struct {
	Title   string     `json:"title"`
	Lines   []string   `json:"lines"`
	Changes []UIChange `json:"changes,omitempty"`
}
type uiCleanResult struct {
	core.CleanResult
	Items     []core.CleanItem `json:"items"`
	Truncated bool             `json:"items_truncated"`
}

// Presentation only: source resolution and protection decisions remain in core.
func summarizeTask(t UITask) UISummary {
	s := UISummary{Title: uiTaskState(t.State), Lines: []string{}}
	if t.Error != nil {
		s.Lines = append(s.Lines, "问题："+t.Error.Message, "下一步："+t.Error.NextAction)
		return s
	}
	switch t.Request.Action {
	case "package-search":
		s.Title = "官方目录查询完成"
	case "package-catalog":
		s.Title = "官方版本查询完成"
	case "package-plan":
		var p core.PackagePlan
		_ = json.Unmarshal(t.Result, &p)
		s.Title = "包管理预览 · 尚未执行"
		s.Lines = append(s.Lines, fmt.Sprintf("%d 个包 · 确认后仅修改预览中的安装位置", len(p.Changes)))
	case "package-apply":
		var result core.PackageApplyResult
		_ = json.Unmarshal(t.Result, &result)
		s.Title = "包管理已完成"
		for _, row := range result.Outcomes {
			s.Lines = append(s.Lines, row.Name+" · "+row.State+" "+row.Version)
		}
	case "sync", "desired", "use", "repair", "upgrade":
		var r core.SyncResult
		_ = json.Unmarshal(t.Result, &r)
		if r.Plan != nil {
			s.Title = "同步预览 · 尚未应用"
			planned := map[string]config.RuntimeLock{}
			for k, v := range r.Plan.Tools {
				planned[k] = v
			}
			if r.Plan.Node.Version != "" {
				planned["node"] = r.Plan.Node
			}
			if r.Plan.Python != nil {
				planned["python"] = *r.Plan.Python
			}
			keys := map[string]bool{}
			for k := range planned {
				keys[k] = true
			}
			if t.View != nil {
				for k := range t.View.Desired {
					keys[k] = true
				}
				for k := range t.View.Applied {
					keys[k] = true
				}
			}
			names := []string{}
			for k := range keys {
				names = append(names, k)
			}
			sort.Strings(names)
			for _, k := range names {
				from, to := "未安装", "移除"
				var old config.RuntimeLock
				if t.View != nil {
					old = t.View.Applied[k]
					if old.Version != "" {
						from = old.Version
					}
					if want := t.View.Desired[k]; want != "" {
						to = want + "（待解析）"
					}
				}
				if v := planned[k]; v.Version != "" {
					to = v.Version
				}
				if from != to || old.Backend != planned[k].Backend || old.URL != planned[k].URL || old.SHA256 != planned[k].SHA256 {
					s.Changes = append(s.Changes, UIChange{k, from, to})
				}
			}
			if !r.Plan.NeedsApply {
				s.Lines = append(s.Lines, "当前环境无需切换。")
			} else {
				s.Lines = append(s.Lines, "确认后准备新环境；失败保留当前生效环境。")
			}
			if len(s.Changes) == 0 && r.Plan.NeedsApply {
				s.Lines = append(s.Lines, "工具版本不变；环境配置、依赖或重建需求仍需同步。")
			}
		} else if r.Changed {
			s.Title = "新环境已应用"
		} else {
			s.Title = "环境无需更改"
		}
	case "clean":
		var r uiCleanResult
		_ = json.Unmarshal(t.Result, &r)
		s.Title = "清理预览"
		if t.Request.Apply {
			s.Title = "清理完成"
		}
		s.Lines = append(s.Lines, fmt.Sprintf("候选 %d 项 · 已删除 %d 项 · %s %.2f MiB", r.Candidates, r.Removed, map[bool]string{true: "已释放", false: "可释放"}[t.Request.Apply], float64(r.Bytes)/(1<<20)), "当前/上一代、引用、租约与运行占用仍由核心保护；未提供完整保护对象清单。")
		for _, item := range r.Items {
			s.Lines = append(s.Lines, fmt.Sprintf("%s · %.2f MiB · %s", item.Kind, float64(item.Bytes)/(1<<20), item.Directory))
		}
		if r.Truncated {
			s.Lines = append(s.Lines, "对象列表仅显示前200项，总数和空间统计完整。")
		}
		if r.UnknownLeases > 0 {
			s.Lines = append(s.Lines, fmt.Sprintf("%d 项租约无法确认，继续保留保护。", r.UnknownLeases))
		}
	case "doctor", "status":
		var r core.Diagnosis
		_ = json.Unmarshal(t.Result, &r)
		if r.Status != nil {
			if r.Environment == "ready" && (r.CheckLevel != "deep" || r.ContentEvidence == "matched") {
				s.Title = "环境检查正常"
			} else {
				s.Title = "环境需要处理"
			}
			s.Lines = append(s.Lines, "环境状态："+uiEnvironment(r.Environment))
			if r.Environment == "ready" {
				s.Lines = append(s.Lines, "可以运行命令；如需更新配置，先查看环境变更。")
			} else {
				s.Lines = append(s.Lines, "下一步：查看环境变更；环境不完整时可返回维护预览修复。")
			}
			if r.CheckLevel == "deep" {
				s.Lines = append(s.Lines, "内容校验："+map[string]string{"matched": "与基线一致", "mismatch": "与基线不一致", "missing": "没有可比较的基线", "unavailable": "无法完成核验"}[r.ContentEvidence])
			}
			if r.EvidenceDetail != "" {
				s.Lines = append(s.Lines, "内容检查："+r.EvidenceDetail)
			}
			if r.ProtectionDetail != "" {
				s.Lines = append(s.Lines, "运行保护："+r.ProtectionDetail)
			}
		}
	case "inventory":
		var r core.Inventory
		_ = json.Unmarshal(t.Result, &r)
		s.Title = fmt.Sprintf("发现 %d 个环境入口", len(r.Installations))
		for _, v := range r.Installations {
			if len(s.Lines) >= 200 {
				break
			}
			s.Lines = append(s.Lines, v.Tool+" · "+v.State+" · "+v.Path)
		}
	case "external":
		var r core.ExternalPlan
		_ = json.Unmarshal(t.Result, &r)
		s.Title = "外部管理计划"
		if t.Request.Apply {
			s.Title = "外部管理操作完成"
		}
		s.Lines = append(s.Lines, r.Installation.Tool+" · "+r.Installation.Path, "原管理器："+r.Manager, r.Notice)
	case "init":
		s.Title = "项目配置已创建或保留"
		s.Lines = append(s.Lines, "选择期望版本并预览同步；初始化不会安装工具。")
	case "rollback":
		s.Title = "环境回退完成"
		s.Lines = append(s.Lines, "已切换生效环境；声明、项目代码和数据保持原样。", "下一步：运行检查，或预览已保存配置与当前环境的差异。")
	case "remove":
		s.Title = "工具已移出用户默认环境"
		s.Lines = append(s.Lines, "当前默认环境已更新；旧代继续遵守回滚与清理保护。")
	}
	return s
}
func uiTaskState(state string) string {
	if s, ok := map[string]string{"succeeded": "已完成", "failed": "失败", "canceled": "已取消", "canceling": "正在取消收尾", "running": "进行中", "confirm": "等待构建许可"}[state]; ok {
		return s
	}
	return "暂无任务"
}

func uiEnvironment(state string) string {
	if s, ok := map[string]string{"ready": "已应用", "drifted": "已保存，待同步", "incomplete": "环境需修复", "not_ready": "尚未安装"}[state]; ok {
		return s
	}
	return "需要检查"
}

func uiToolState(w *core.Workbench, tool string) string {
	want, have := w.Desired[tool] != "", w.Applied[tool].Version != ""
	if !want && have {
		return "配置已移除，待同步"
	}
	if !want {
		return "未配置"
	}
	if !have {
		return "待安装"
	}
	if w.Status != nil {
		return uiEnvironment(w.Status.Environment)
	}
	return "需要检查"
}

func uiPhase(phase string) string {
	if s, ok := map[string]string{"waiting": "等待环境锁", "resolving": "解析版本", "backend": "准备安装后端", "sdk": "准备工具链", "node": "准备 Node.js", "python": "准备 Python", "dependencies": "安装项目依赖", "verifying": "检查新环境", "publishing": "应用新环境"}[phase]; ok {
		return s
	}
	return phase
}
