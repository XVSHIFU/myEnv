package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"myenv/internal/backend"
	"myenv/internal/core"
)

func cloneUIRequest(r UIRequest) UIRequest {
	if r.Package != nil {
		p := *r.Package
		p.Items = append([]core.PackageItemRequest(nil), p.Items...)
		r.Package = &p
	}
	return r
}

// Only retained server results can become executable plans. Every draft field
// and the selected workspace must still match, and one plan is used once.
func (c *UIController) bindPackagePlan(r *UIRequest) error {
	if c.usedPackagePlans[r.PlanTaskID] {
		return fmt.Errorf("此包管理计划已执行，请刷新并重新预览")
	}
	for _, task := range c.tasks {
		if task.ID != r.PlanTaskID || task.State != "succeeded" || task.Request.Action != "package-plan" {
			continue
		}
		want := cloneUIRequest(task.Request)
		want.Action, want.PlanTaskID = "package-apply", task.ID
		if !reflect.DeepEqual(want, *r) {
			return fmt.Errorf("包、版本或安装位置与预览不一致，请重新预览")
		}
		var plan core.PackagePlan
		if err := json.Unmarshal(task.Result, &plan); err != nil || plan.ID == "" {
			return fmt.Errorf("包管理计划不可用，请重新预览")
		}
		r.packagePlan = &plan
		if c.usedPackagePlans == nil {
			c.usedPackagePlans = map[uint64]bool{}
		}
		c.usedPackagePlans[task.ID] = true
		return nil
	}
	return fmt.Errorf("请先预览包管理操作，再确认执行；原计划可能已过期")
}

func (c *UIController) packageAction(ctx context.Context, id uint64, r UIRequest, service *core.Service) (any, error) {
	if r.Package == nil {
		return nil, fmt.Errorf("请选择包及具体安装位置")
	}
	target := r.Package.Target
	ctx = core.WithPackageProgress(ctx, func(label string) { c.phase(id, "running", label) })
	ctx = core.WithPackageLog(ctx, uiLog{c, id})
	if target.Ecosystem != "node" && target.Ecosystem != "python" {
		return nil, fmt.Errorf("当前官方目录支持 Node.js 与 Python 包")
	}
	if target.Scope == "project" {
		directory, err := filepath.Abs(target.Directory)
		same := filepath.Clean(directory) == filepath.Clean(r.Directory)
		if runtime.GOOS == "windows" {
			same = strings.EqualFold(filepath.Clean(directory), filepath.Clean(r.Directory))
		}
		if err != nil || r.Global || !same {
			return nil, fmt.Errorf("包管理项目与当前选中目录不一致，请重新打开管理")
		}
	}
	certificate := r.Certificate
	if certificate == "" {
		certificate = os.Getenv("SSL_CERT_FILE")
	}
	client, err := backend.DownloadClient(certificate)
	if err != nil {
		return nil, err
	}
	defer client.CloseIdleConnections()
	registry := backend.PackageRegistry{Client: client}
	service.Node = backend.NewNode()
	service.Node.Client = client
	switch r.Action {
	case "package-search":
		c.phase(id, "running", "查询官方包目录")
		return registry.Search(ctx, target.Ecosystem, r.PackageQuery)
	case "package-catalog":
		c.phase(id, "running", "查询官方版本与兼容要求")
		return registry.Package(ctx, target.Ecosystem, r.PackageName)
	case "package-plan":
		c.phase(id, "running", "核对安装位置并解析目标版本")
		return service.PlanPackages(ctx, *r.Package)
	case "package-apply":
		if r.packagePlan == nil {
			return nil, fmt.Errorf("请先预览包管理操作")
		}
		c.phase(id, "running", "执行已确认的包管理计划")
		return service.ApplyPackages(ctx, *r.packagePlan)
	}
	return nil, fmt.Errorf("不支持的包管理操作")
}
