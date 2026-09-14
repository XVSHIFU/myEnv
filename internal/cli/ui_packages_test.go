package cli

import (
	"encoding/json"
	"testing"

	"myenv/internal/core"
)

func TestUIPackagePlanBinding(t *testing.T) {
	request := scopedUIRequest(UIRequest{Action: "package-plan", Directory: t.TempDir(), Global: true, Tool: "node", Package: &core.PackageRequest{Target: core.PackageTarget{Ecosystem: "node", Scope: "global_package_root", Root: t.TempDir()}, Operation: "switch", Items: []core.PackageItemRequest{{Name: "test-package", Desired: "1.0.0"}}}})
	encoded, _ := json.Marshal(core.PackagePlan{ID: "retained", Request: *request.Package})
	newController := func() *UIController {
		c := NewUIController(t.TempDir())
		c.tasks = []UITask{{ID: 7, State: "succeeded", Request: cloneUIRequest(request), Result: encoded}}
		return c
	}
	apply := cloneUIRequest(request)
	apply.Action = "package-apply"
	apply.PlanTaskID = 7
	for _, test := range []struct {
		name   string
		change func(*UIRequest)
	}{
		{"package", func(r *UIRequest) { r.Package.Items[0].Name = "other-package" }},
		{"version", func(r *UIRequest) { r.Package.Items[0].Desired = "2.0.0" }},
		{"root", func(r *UIRequest) { r.Package.Target.Root += "-other" }},
		{"manager", func(r *UIRequest) { r.Package.Target.ManagerPath = "different" }},
		{"operation", func(r *UIRequest) { r.Package.Operation = "remove" }},
		{"project", func(r *UIRequest) { r.Directory += "-other" }},
		{"scope", func(r *UIRequest) { r.Global = false }},
		{"expired", func(r *UIRequest) { r.PlanTaskID = 6 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := cloneUIRequest(apply)
			test.change(&r)
			if newController().bindPackagePlan(&r) == nil {
				t.Fatal("changed request was authorized")
			}
		})
	}
	c := newController()
	r := cloneUIRequest(apply)
	if err := c.bindPackagePlan(&r); err != nil || r.packagePlan == nil {
		t.Fatalf("retained plan not bound: %v", err)
	}
	r = cloneUIRequest(apply)
	if c.bindPackagePlan(&r) == nil {
		t.Fatal("plan replay accepted")
	}
}

func TestUIPackageRequestSnapshotsAndScope(t *testing.T) {
	input := UIRequest{Action: "package-plan", Tool: "java", Apply: true, PlanTaskID: 20, Selection: "java@8", PackageName: "stale", PackageQuery: "stale", Package: &core.PackageRequest{Items: []core.PackageItemRequest{{Name: "original"}}}}
	r := scopedUIRequest(input)
	input.Package.Items[0].Name = "caller changed"
	if r.Package.Items[0].Name != "original" || r.Apply || r.Tool != "" || r.PlanTaskID != 0 || r.PackageName != "" || r.PackageQuery != "" {
		t.Fatalf("request not isolated/scoped: %+v", r)
	}
	c := NewUIController("")
	c.tasks = []UITask{{ID: 1, Request: r}}
	current := c.Current()
	current.Request.Package.Items[0].Name = "view changed"
	if c.Current().Request.Package.Items[0].Name != "original" {
		t.Fatal("UI view can mutate retained request")
	}
	r.Action = "inventory"
	r = scopedUIRequest(r)
	if r.Package != nil {
		t.Fatal("package draft leaked to inventory")
	}
}
