package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/core"
	"myenv/internal/runner"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"
)

// UIRequest is a value snapshot, never a process-wide directory/env mutation.
type UIRequest struct {
	Rebuild        bool                 `json:"rebuild"`
	NodeMirror     string               `json:"nodeMirror"`
	UVMirror       string               `json:"uvMirror"`
	PythonMirror   string               `json:"pythonMirror"`
	Certificate    string               `json:"certificate"`
	Action         string               `json:"action"`
	Directory      string               `json:"directory"`
	Global         bool                 `json:"global"`
	Selection      string               `json:"selection"`
	Provider       string               `json:"provider"`
	Tool           string               `json:"tool"`
	Preview        bool                 `json:"preview"`
	Deep           bool                 `json:"deep"`
	Locked         bool                 `json:"locked"`
	Major          int                  `json:"major"`
	Channel        string               `json:"channel"`
	Date           string               `json:"date"`
	ID             string               `json:"id"`
	Manager        string               `json:"manager"`
	ExternalAction string               `json:"externalAction"`
	Apply          bool                 `json:"apply"`
	ExpectedDigest string               `json:"expectedDigest"`
	PlanTaskID     uint64               `json:"planTaskId"`
	Package        *core.PackageRequest `json:"package,omitempty"`
	PackageQuery   string               `json:"packageQuery,omitempty"`
	PackageName    string               `json:"packageName,omitempty"`
	externalPlan   *core.ExternalPlan
	packagePlan    *core.PackagePlan
}
type UITask struct {
	ID           uint64          `json:"id"`
	Request      UIRequest       `json:"request"`
	State        string          `json:"state"`
	Phase        string          `json:"phase"`
	Phases       []string        `json:"phases,omitempty"`
	StartedAt    int64           `json:"startedAt"`
	UpdatedAt    int64           `json:"updatedAt"`
	FinishedAt   int64           `json:"finishedAt,omitempty"`
	Progress     *UIProgress     `json:"progress,omitempty"`
	Result       json.RawMessage `json:"result,omitempty"`
	Error        *failure        `json:"error,omitempty"`
	Exit         int             `json:"exit"`
	Log          string          `json:"log,omitempty"`
	LogTruncated bool            `json:"logTruncated,omitempty"`
	View         *core.Workbench `json:"view,omitempty"`
	Summary      UISummary       `json:"summary"`
}
type UIProgress struct {
	Kind      string `json:"kind"`
	Label     string `json:"label"`
	Completed int64  `json:"completed"`
	Total     int64  `json:"total,omitempty"`
}
type UIController struct {
	mu               sync.Mutex
	namespace        string
	tasks            []UITask
	next             uint64
	cancel           context.CancelFunc
	done             chan struct{}
	confirm          chan bool
	closing          bool
	updates          chan struct{}
	notifyTimer      *time.Timer
	notifyVersion    uint64
	lastNotify       time.Time
	downloadID       uint64
	downloadClosed   bool
	usedPackagePlans map[uint64]bool
}

// External manager streams share one bounded log; never block a child on UI rendering.
type uiLog struct {
	controller *UIController
	id         uint64
}

func (w uiLog) Write(p []byte) (int, error) {
	c := w.controller
	c.mu.Lock()
	defer c.mu.Unlock()
	const limit = 64 * 1024
	n := len(p)
	t := c.activeTask(w.id)
	if t == nil || n == 0 || t.LogTruncated {
		return n, nil
	}
	left := limit - len(t.Log)
	if len(p) > left {
		p = p[:left]
		t.LogTruncated = true
	}
	t.Log += string(p)
	t.UpdatedAt = time.Now().UnixMilli()
	c.notifyStream()
	return n, nil
}

func NewUIController(namespace string) *UIController {
	return &UIController{namespace: namespace, updates: make(chan struct{}, 1)}
}
func (c *UIController) Updates() <-chan struct{} { return c.updates }

// All notification helpers are called with c.mu held. Only changed streaming
// data arms a one-shot timer; idle controllers have no polling goroutine.
func (c *UIController) notify() {
	c.notifyVersion++
	if c.notifyTimer != nil {
		c.notifyTimer.Stop()
		c.notifyTimer = nil
	}
	c.lastNotify = time.Now()
	select {
	case c.updates <- struct{}{}:
	default:
	}
}
func (c *UIController) notifyStream() {
	const interval = 100 * time.Millisecond
	delay := interval - time.Since(c.lastNotify)
	if delay <= 0 {
		c.notify()
		return
	}
	if c.notifyTimer != nil {
		return
	}
	version := c.notifyVersion
	c.notifyTimer = time.AfterFunc(delay, func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.notifyVersion == version {
			c.notify()
		}
	})
}

// The current task is the only producer allowed to mutate controller state.
// Late logs, progress and phase callbacks cannot revive a terminal task.
func (c *UIController) activeTask(id uint64) *UITask {
	if len(c.tasks) == 0 {
		return nil
	}
	t := &c.tasks[len(c.tasks)-1]
	if t.ID != id || (t.State != "running" && t.State != "confirm" && t.State != "canceling") {
		return nil
	}
	return t
}

func cloneUITask(t UITask) UITask {
	t.Request = cloneUIRequest(t.Request)
	t.Request.packagePlan = nil
	t.Result = append(json.RawMessage(nil), t.Result...)
	t.Phases = append([]string(nil), t.Phases...)
	if t.Progress != nil {
		progress := *t.Progress
		t.Progress = &progress
	}
	return t
}
func (c *UIController) Current() UITask {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.tasks) == 0 {
		return UITask{State: "idle"}
	}
	return cloneUITask(c.tasks[len(c.tasks)-1])
}
func (c *UIController) Start(r UIRequest) (uint64, error) {
	// Flags are scoped to operations, never inherited from a previous task.
	r = scopedUIRequest(r)
	directory, err := filepath.Abs(r.Directory)
	if err != nil {
		return 0, err
	}
	r.Directory = directory
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closing {
		return 0, fmt.Errorf("界面正在关闭")
	}
	if c.cancel != nil {
		return 0, fmt.Errorf("请等待当前任务收尾")
	}
	if r.Action == "external" && r.Apply {
		if err := c.bindExternalPlan(&r); err != nil {
			return 0, err
		}
	}
	if r.Action == "package-apply" {
		if err := c.bindPackagePlan(&r); err != nil {
			return 0, err
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.done = make(chan struct{})
	c.confirm = make(chan bool, 1)
	c.next++
	if len(c.tasks) == 16 {
		delete(c.usedPackagePlans, c.tasks[0].ID)
		c.tasks = c.tasks[1:]
	}
	id := c.next
	now := time.Now().UnixMilli()
	c.tasks = append(c.tasks, UITask{ID: id, Request: r, State: "running", StartedAt: now, UpdatedAt: now})
	c.downloadID, c.downloadClosed = 0, false
	c.notify()
	go c.execute(ctx, id, r)
	return id, nil
}

// Called with c.mu held. Only a retained, successful verification can authorize
// an external write. The UI cannot supply or modify an executable plan.
func (c *UIController) bindExternalPlan(r *UIRequest) error {
	for _, t := range c.tasks {
		if t.ID != r.PlanTaskID || t.State != "succeeded" || t.Request.Action != "external" || t.Request.Apply {
			continue
		}
		want := t.Request
		want.Apply, want.PlanTaskID = true, t.ID
		if !reflect.DeepEqual(want, *r) {
			return fmt.Errorf("执行目标与已核验计划不一致，请重新核验")
		}
		var plan core.ExternalPlan
		if err := json.Unmarshal(t.Result, &plan); err != nil || plan.Manager == "" {
			return fmt.Errorf("核验计划不可用，请重新核验")
		}
		r.externalPlan = &plan
		return nil
	}
	return fmt.Errorf("请先核验外部管理计划，再确认执行；历史可能已过期")
}

func scopedUIRequest(r UIRequest) UIRequest {
	r = cloneUIRequest(r)
	r.externalPlan = nil
	r.packagePlan = nil
	if strings.HasPrefix(r.Action, "package-") {
		// A package operation has its own target. Language-page drafts are not inputs.
		r = UIRequest{Action: r.Action, Directory: r.Directory, Global: r.Global, Certificate: r.Certificate, Package: r.Package, PackageQuery: r.PackageQuery, PackageName: r.PackageName, PlanTaskID: r.PlanTaskID}
		if r.Action != "package-search" {
			r.PackageQuery = ""
		}
		if r.Action != "package-catalog" {
			r.PackageName = ""
		}
	} else {
		r.Package, r.PackageQuery, r.PackageName = nil, "", ""
	}
	if (r.Action != "external" || !r.Apply) && r.Action != "package-apply" {
		r.PlanTaskID = 0
	}
	if r.Action != "sync" && r.Action != "versions" {
		r.Preview = false
	}
	if r.Action != "sync" {
		r.ExpectedDigest = ""
		r.Locked = false
		r.Rebuild = false
	}
	if r.Action != "doctor" {
		r.Deep = false
	}
	if r.Action != "clean" && r.Action != "external" {
		r.Apply = false
	}
	return r
}
func (c *UIController) Cancel(id uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil && c.next == id {
		t := &c.tasks[len(c.tasks)-1]
		t.State, t.Progress = "canceling", nil
		t.UpdatedAt = time.Now().UnixMilli()
		c.downloadClosed = true
		c.cancel()
		c.notify()
	}
}
func (c *UIController) Confirm(id uint64, allow bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.next != id || c.cancel == nil || c.tasks[len(c.tasks)-1].State != "confirm" {
		return
	}
	select {
	case c.confirm <- allow:
		t := &c.tasks[len(c.tasks)-1]
		t.State = "running"
		t.UpdatedAt = time.Now().UnixMilli()
		c.notify()
	default:
	}
}
func (c *UIController) Close() {
	c.mu.Lock()
	c.closing = true
	done := c.done
	if c.cancel != nil {
		c.cancel()
	}
	c.mu.Unlock()
	if done != nil {
		<-done
	}
}
func (c *UIController) phase(id uint64, state, phase string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := c.activeTask(id)
	if t == nil {
		return
	}
	if t.State != "canceling" {
		t.State = state
	}
	t.Phase = phase
	t.Progress = nil
	t.UpdatedAt = time.Now().UnixMilli()
	c.downloadClosed = true
	if state == "running" && phase != "" {
		seen := false
		for _, reached := range t.Phases {
			seen = seen || reached == phase
		}
		if !seen && len(t.Phases) < 16 {
			t.Phases = append(t.Phases, phase)
		}
	}
	c.notify()
}

func (c *UIController) download(id uint64, progress backend.DownloadProgress) {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := c.activeTask(id)
	if t == nil || t.State != "running" || progress.ID < c.downloadID || (progress.ID == c.downloadID && c.downloadClosed) {
		return
	}
	c.downloadID, c.downloadClosed = progress.ID, progress.Done
	if progress.Done {
		t.Progress = nil
	} else {
		t.Progress = &UIProgress{Kind: "download", Label: progress.Label, Completed: progress.Completed, Total: progress.Total}
	}
	t.UpdatedAt = time.Now().UnixMilli()
	c.notifyStream()
}

func (c *UIController) execute(ctx context.Context, id uint64, r UIRequest) {
	ctx = backend.WithDownloadProgress(ctx, func(p backend.DownloadProgress) { c.download(id, p) })
	data, err := c.action(runner.BackgroundContext(ctx), id, r)
	var view *core.Workbench
	if w, ok := data.(core.Workbench); ok {
		view = &w
	} else if r.Action != "versions" && r.Action != "inventory" && r.Action != "external" && !strings.HasPrefix(r.Action, "package-") {
		w, e := (&core.Service{Profile: r.Global, UserConfigDirectory: c.namespace}).Workbench(context.Background(), r.Directory)
		if e == nil {
			view = &w
		}
	}
	encoded, encodeErr := json.Marshal(data)
	if err == nil {
		err = encodeErr
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	t := c.activeTask(id)
	if t == nil {
		return
	}
	t.State = "succeeded"
	t.Result = encoded
	t.View = view
	if err != nil {
		t.State = "failed"
		t.Error, t.Exit = describeFailure(err, true)
		if t.Error.Code == "CANCELED" {
			t.State = "canceled"
		}
		if r.Action == "package-apply" {
			t.Error.NextAction = "已完成的包操作不会自动回退，请查看逐项结果并刷新列表。"
		}
	}
	t.Progress = nil
	t.FinishedAt = time.Now().UnixMilli()
	t.UpdatedAt = t.FinishedAt
	t.Summary = summarizeTask(*t)
	c.cancel()
	c.cancel = nil
	close(c.done)
	c.notify()
}
func (c *UIController) action(ctx context.Context, id uint64, r UIRequest) (any, error) {
	service := &core.Service{Profile: r.Global, UserConfigDirectory: c.namespace}
	// A UI project is the explicitly selected directory, not an implicit parent.
	if !r.Global {
		switch r.Action {
		case "status", "doctor", "rollback", "clean", "desired", "use", "sync", "upgrade", "repair":
			if _, err := os.Stat(filepath.Join(r.Directory, "myenv.yaml")); err != nil {
				return nil, fmt.Errorf("当前目录没有可读取的 myenv.yaml，请先初始化：%w", err)
			}
		}
	}
	confirm := func(message string) (bool, error) {
		c.phase(id, "confirm", message)
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case yes := <-c.confirm:
			c.phase(id, "running", "")
			return yes, nil
		}
	}
	progress := func(p core.SyncPhase) { c.phase(id, "running", string(p)) }
	switch r.Action {
	case "package-search", "package-catalog", "package-plan", "package-apply":
		return c.packageAction(ctx, id, r, service)
	case "status":
		return service.Status(ctx, r.Directory)
	case "workbench":
		return service.Workbench(ctx, r.Directory)
	case "init":
		if r.Global {
			return nil, fmt.Errorf("默认环境请使用安装工具")
		}
		selection, e := providerSelection(r.Selection, r.Provider)
		if e != nil {
			return nil, e
		}
		cfg, changed, err := config.InitWithInput(r.Directory, func(tool, reason string) (string, string, error) {
			name, version, ok := strings.Cut(selection, "@")
			if !ok {
				return "", "", fmt.Errorf("%s：请选择 tool@version", reason)
			}
			return name, version, nil
		})
		return map[string]any{"config": cfg, "changed": changed}, err
	case "doctor":
		if r.Deep {
			return service.DoctorDeep(ctx, r.Directory, os.Environ())
		}
		return service.Doctor(ctx, r.Directory, os.Environ())
	case "inventory":
		if r.Global {
			return service.InventoryForProject(ctx, true, "")
		}
		return service.InventoryForProject(ctx, true, r.Directory)
	case "rollback":
		return service.Rollback(ctx, r.Directory)
	case "clean":
		result := uiCleanResult{Items: []core.CleanItem{}}
		var err error
		result.CleanResult, err = service.Clean(ctx, r.Directory, !r.Apply, func(item core.CleanItem) error {
			if len(result.Items) < 200 {
				result.Items = append(result.Items, item)
			} else {
				result.Truncated = true
			}
			return nil
		})
		return result, err
	case "remove":
		if !r.Global {
			return nil, fmt.Errorf("移除工具仅适用于用户默认环境")
		}
		return service.RemoveDefault(ctx, r.Tool)
	case "versions":
		return QueryVersions(ctx, VersionQuery{Tool: r.Tool, Provider: r.Provider, Preview: r.Preview, Major: r.Major, Channel: r.Channel, Date: r.Date, Namespace: c.namespace, Certificate: r.Certificate})
	case "external":
		inventory, err := service.Inventory(ctx, true)
		if err != nil {
			return nil, err
		}
		for _, row := range inventory.Installations {
			if row.ID == r.ID {
				plan, err := core.BuildExternalPlan(ctx, row, r.Manager, r.ExternalAction)
				if err != nil {
					return nil, err
				}
				if r.Apply {
					if r.externalPlan == nil || !reflect.DeepEqual(*r.externalPlan, plan) {
						return nil, fmt.Errorf("外部环境或管理器计划已变化，请重新核验后确认")
					}
					err = core.ApplyExternalPlan(ctx, plan, strings.NewReader(""), uiLog{c, id}, uiLog{c, id})
				}
				return plan, err
			}
		}
		return nil, fmt.Errorf("安装 ID 已失效，请刷新列表")
	case "desired", "use", "sync", "upgrade", "repair":
		installing, err := runtimeService(r.Global, c.namespace)
		if err != nil {
			return nil, err
		}
		defer closeRuntimeService(installing)
		if r.Certificate != "" {
			client, e := backend.DownloadClient(r.Certificate)
			if e != nil {
				return nil, e
			}
			if installing.Node == nil {
				installing.Node = backend.NewNode()
			}
			installing.Node.Client = client
		}
		if r.NodeMirror != "" {
			base, e := config.MirrorBaseURL(r.NodeMirror)
			if e != nil {
				return nil, e
			}
			if installing.Node == nil {
				installing.Node = backend.NewNode()
			}
			installing.Node.BaseURL = base
		}
		if r.UVMirror != "" {
			base, e := config.MirrorBaseURL(r.UVMirror)
			if e != nil {
				return nil, e
			}
			installing.UVMirror = base
		}
		if r.PythonMirror != "" {
			base, e := config.MirrorBaseURL(r.PythonMirror)
			if e != nil {
				return nil, e
			}
			installing.PythonMirror = base
		}
		if r.Action == "use" || r.Action == "desired" {
			selection, err := providerSelection(r.Selection, r.Provider)
			if err != nil {
				return nil, err
			}
			return installing.UseWithRequest(ctx, core.UseRequest{Directory: r.Directory, Selection: selection, Preview: r.Action == "desired", ConfirmBuild: confirm, Progress: progress})
		}
		update := ""
		if r.Action == "upgrade" {
			update = r.Tool
		}
		return installing.Sync(ctx, core.SyncRequest{Directory: r.Directory, ExpectedDigest: r.ExpectedDigest, DryRun: r.Preview, Locked: r.Locked, Rebuild: r.Action == "repair" || r.Rebuild, Update: update, ConfirmBuild: confirm, Progress: progress})
	default:
		return nil, fmt.Errorf("不支持的界面操作 %q", r.Action)
	}
}
