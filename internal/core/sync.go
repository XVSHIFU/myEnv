// Package core coordinates environment operations without terminal output.
package core

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/runner"
	"myenv/internal/state"
)

type SyncRequest struct {
	Directory      string
	Locked         bool
	DryRun         bool
	Rebuild        bool
	Update         string
	ExpectedDigest string
	AllowBuild     bool
	ConfirmBuild   func(string) (bool, error)
	// Progress is synchronous; callbacks must return promptly and not mutate state.
	Progress func(SyncPhase)
}
type SyncResult struct {
	Changed           bool
	LockChanged       bool `json:"lock_changed"`
	NativeLockChanged bool `json:"native_lock_changed"`
	Generation        *state.Generation
	Plan              *SyncPlan `json:"plan,omitempty"`
}
type SyncPlan struct {
	Platform              string                        `json:"platform"`
	Tools                 map[string]config.RuntimeLock `json:"tools,omitempty"`
	Node                  config.RuntimeLock            `json:"node"`
	Python                *config.RuntimeLock           `json:"python,omitempty"`
	UnresolvedPython      string                        `json:"unresolved_python,omitempty"`
	NativeLockNeedsUpdate bool                          `json:"native_lock_needs_update,omitempty"`
	NeedsApply            bool                          `json:"needs_apply"`
}
type Service struct {
	Storage             *config.UserStorage
	Profile             bool
	UserConfigDirectory string
	Node                *backend.Node
	UV                  *backend.UV
	// UVArchive optionally supplies retained release bytes, still hash-verified.
	UVArchive string
	// UVMirror overrides only the download base; pinned version and hashes remain.
	UVMirror     string
	PythonMirror string
	// PythonDirectory permits isolated managed runtime storage in integrations.
	PythonDirectory string
}

func (s *Service) Sync(ctx context.Context, request SyncRequest) (result SyncResult, syncErr error) {
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if request.Update != "" && !config.SupportedTool(request.Update) {
		return result, fmt.Errorf("unsupported runtime update %q; supported tools: node, python, java, go, rust", request.Update)
	}
	if request.Locked && request.Update != "" {
		return result, fmt.Errorf("--update and --locked are mutually exclusive")
	}
	root, err := s.resolveRoot(request.Directory)
	if err != nil {
		return result, err
	}
	c, err := s.loadDeclaration(root)
	if err != nil {
		return result, err
	}
	platform, err := runner.Platform()
	if err != nil {
		return result, err
	}
	if request.Update != "" && c.Tools[request.Update] == "" {
		return result, fmt.Errorf("cannot update undeclared tool %s", request.Update)
	}
	digest, err := config.Digest(c)
	if err != nil {
		return result, err
	}
	if request.ExpectedDigest != "" && request.ExpectedDigest != digest {
		return result, fmt.Errorf("INPUT_CHANGED: declaration changed after use edited it")
	}
	inputs, err := pythonInputs(root, c, request.Locked)
	if err != nil {
		return result, err
	}
	work := s.workDirectory(root)
	storage, err := s.storagePaths(work)
	if err != nil {
		return result, err
	}
	var store *state.Store
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if !request.DryRun {
		if err = os.MkdirAll(work, 0700); err != nil {
			return result, err
		}
		request.phase(PhaseWaiting)
		unlock, err := state.LockWorkspace(ctx, filepath.Join(work, "modify.lock"))
		if err != nil {
			return result, err
		}
		defer unlock()
		if err = ctx.Err(); err != nil {
			return result, err
		}
		currentConfig, err := s.loadDeclaration(root)
		if err != nil {
			return result, err
		}
		currentDigest, err := config.Digest(currentConfig)
		if err != nil {
			return result, err
		}
		if currentDigest != digest {
			return result, fmt.Errorf("INPUT_CHANGED: configuration changed while waiting for workspace lock")
		}
		currentInputs, err := pythonInputs(root, c, request.Locked)
		if err != nil {
			return result, err
		}
		if !reflect.DeepEqual(inputs, currentInputs) {
			return result, fmt.Errorf("INPUT_CHANGED: Python inputs changed while waiting for workspace lock")
		}
		store, err = state.Open(ctx, filepath.Join(work, "state.db"))
		if err != nil {
			return result, err
		}
		if _, err = store.RecoverInterrupted(ctx); err != nil {
			store.Close()
			return result, err
		}
	} else {
		database := filepath.Join(work, "state.db")
		if _, statErr := os.Stat(database); statErr == nil {
			store, err = state.OpenReadOnly(ctx, database)
			if err != nil {
				return result, err
			}
		} else if !os.IsNotExist(statErr) {
			return result, statErr
		}
	}
	var active *state.Generation
	if store != nil {
		defer store.Close()
		active, err = store.Active(ctx)
		if err != nil {
			return result, err
		}
	}
	lockPath := s.desiredLockPath(root)
	lock, err := config.ReadLock(lockPath)
	if err != nil && !os.IsNotExist(err) {
		return result, err
	}
	selected := config.PlatformLock{Tools: map[string]config.RuntimeLock{}, Python: inputs}
	if lock != nil {
		for tool := range c.Tools {
			selected.Tools[tool] = lock.Platforms[platform].Tools[tool]
		}
	}
	validTools := map[string]bool{}
	valid := true
	for tool, selector := range c.Tools {
		locked := selected.Tools[tool]
		provider, _ := config.PythonProvider(selector)
		if tool == "python" && provider == "astral" && locked.Version != "" && locked.Backend != "python-official-v1" && locked.Backend != "uv-"+backend.UVVersion && request.Update != "python" {
			return result, fmt.Errorf("LOCK_OUT_OF_DATE: locked Python backend %q is unavailable; use sync --update python to select this build's backend explicitly", locked.Backend)
		}
		constraint, _ := config.ParseConstraint(tool, selector)
		backendValid := locked.Backend == "node-official-v1" && locked.Evidence == "artifact"
		if tool == "python" {
			backendValid = locked.Backend == "uv-"+backend.UVVersion && locked.Evidence == "version"
			if provider == "python.org" {
				backendValid = locked.Backend == "python-official-v1" && locked.Evidence == "artifact"
			}
		}
		if tool == "java" || tool == "go" || tool == "rust" {
			backendValid = locked.Backend == tool+"-official-v1" && locked.Evidence == "artifact"
		}
		validTools[tool] = request.Update != tool && backendValid && constraint.Contains(locked.Version)
		valid = valid && validTools[tool]
	}
	if request.Locked && (lock == nil || lock.ConfigDigest != digest || !valid) {
		return result, fmt.Errorf("%s", s.commandHint("LOCK_OUT_OF_DATE: run myenv sync to refresh the lock"))
	}
	if request.Locked && !reflect.DeepEqual(lock.Platforms[platform], selected) {
		return result, fmt.Errorf("LOCK_OUT_OF_DATE: Python inputs or declared tools differ from the lock")
	}
	if !request.Rebuild && valid && lock != nil && lock.ConfigDigest == digest && active != nil && active.InputDigest == digest {
		appliedLock, e := config.ReadLock(filepath.Join(active.Directory, "myenv.lock"))
		if e == nil && appliedLock.ConfigDigest == digest && entriesHealthy(active, c.Tools) && snapshotHealthy(active, root) && reflect.DeepEqual(appliedLock.Platforms[platform], selected) && reflect.DeepEqual(lock.Platforms[platform], selected) {
			if request.DryRun {
				return SyncResult{Plan: syncPlan(platform, selected, false)}, nil
			}
			return SyncResult{Generation: active}, nil
		}
	}
	node := s.Node
	if node == nil {
		node = backend.NewNode()
	}
	if c.Tools["node"] != "" && !validTools["node"] {
		request.phase(PhaseResolving)
		artifact, e := node.Resolve(ctx, c.Tools["node"], platform)
		if e != nil {
			return result, e
		}
		selected.Tools["node"] = config.RuntimeLock{Version: artifact.Version, NPM: artifact.NPM, Backend: artifact.Backend, Evidence: artifact.Evidence, URL: artifact.URL, SHA256: artifact.SHA256}
	}
	for _, tool := range []string{"java", "go", "rust"} {
		if c.Tools[tool] == "" || validTools[tool] {
			continue
		}
		request.phase(PhaseResolving)
		resolved, e := (backend.Catalog{Client: node.Client}).ResolveSDK(ctx, tool, c.Tools[tool], platform)
		if e != nil {
			return result, e
		}
		selected.Tools[tool] = resolved
	}
	pythonProvider, pythonSelector := config.PythonProvider(c.Tools["python"])
	if c.Tools["python"] != "" && pythonProvider == "python.org" && !validTools["python"] {
		resolved, e := (backend.Catalog{Client: node.Client}).ResolveOfficialPython(ctx, pythonSelector, platform)
		if e != nil {
			return result, e
		}
		selected.Tools["python"] = resolved
	}
	if request.DryRun {
		plan := syncPlan(platform, selected, true)
		if c.Tools["python"] != "" && pythonProvider != "python.org" && !validTools["python"] {
			plan.Python = nil
			plan.UnresolvedPython = c.Tools["python"]
		}
		plan.NativeLockNeedsUpdate = inputs != nil && (lock == nil || !reflect.DeepEqual(lock.Platforms[platform].Python, inputs))
		return SyncResult{Plan: plan}, nil
	}
	var uv backend.UV
	if c.Python != nil && !request.AllowBuild {
		project, err := config.Within(root, c.Python.Project)
		if err != nil {
			return result, err
		}
		if err := backend.CheckPythonBuildPermission(project); err != nil {
			var needsInput *config.NeedsInput
			if !errors.As(err, &needsInput) || request.ConfirmBuild == nil {
				return result, err
			}
			allowed, confirmErr := request.ConfirmBuild(needsInput.Message)
			if confirmErr != nil {
				return result, confirmErr
			}
			if !allowed {
				return result, err
			}
			request.AllowBuild = true
		}
	}
	id := make([]byte, 16)
	if _, err = rand.Read(id); err != nil {
		return result, err
	}
	generationID := fmt.Sprintf("%x", id)
	generations := filepath.Join(work, "generations")
	if err = os.MkdirAll(generations, 0700); err != nil {
		return result, err
	}
	directory := filepath.Join(generations, generationID)
	if err = store.BeginTrackedOperation(ctx, generationID, directory, digest); err != nil {
		return result, err
	}
	defer func() {
		// Publication clears any remaining hold. Panic/Goexit during backend
		// work and an unconfirmed supervisor leave durable protection in place.
		syncErr = finishFailedPreparation(store, generationID, platform, syncErr)
	}()
	operationDirectory := filepath.Join(work, "operations", generationID)
	if err = os.MkdirAll(operationDirectory, 0700); err != nil {
		return result, err
	}
	ctx = observePreparationChildren(ctx, store, generationID, platform, operationDirectory)
	if c.Tools["python"] != "" {
		request.phase(PhaseBackend)
		uv, err = s.managedUV(ctx, work, platform)
		if err != nil {
			return result, err
		}
		if !validTools["python"] && pythonProvider != "python.org" {
			request.phase(PhaseResolving)
			release, err := uv.ResolvePython(ctx, pythonSelector, filepath.Join(storage.Cache, "uv"))
			if err != nil {
				return result, err
			}
			selected.Tools["python"] = config.RuntimeLock{Version: release.Version, Backend: release.Backend, Evidence: release.Evidence, URL: release.URL}
		}
	}
	// A missing native lock cannot be recorded as a complete input yet.
	if inputs != nil && inputs.UVLockSHA256 == "" {
		selected.Python = nil
	}
	lockNeedsWrite := lock == nil || lock.ConfigDigest != digest || !reflect.DeepEqual(lock.Platforms[platform], selected)
	if lock == nil {
		lock = &config.Lock{Schema: 1, Platforms: map[string]config.PlatformLock{}}
	}
	lock.ConfigDigest = digest
	lock.Platforms[platform] = selected
	if !request.Locked && lockNeedsWrite {
		if err = config.ReplaceLock(lockPath, lock); err != nil {
			return result, err
		}
		result.LockChanged = true
	}
	var executable string
	if c.Tools["node"] != "" {
		request.phase(PhaseNode)
		locked := selected.Tools["node"]
		artifact := backend.NodeArtifact{Version: locked.Version, NPM: locked.NPM, Platform: platform, Backend: locked.Backend, Evidence: locked.Evidence, URL: locked.URL, SHA256: locked.SHA256}
		archive, err := s.nodeArchive(ctx, node, artifact, operationDirectory)
		if err != nil {
			return result, err
		}
		defer os.Remove(archive)
		if platform == "windows-amd64" {
			executable, err = backend.PrepareNodeZIP(ctx, archive, directory, artifact)
		} else {
			executable, err = backend.PrepareNodeTarGZ(ctx, archive, directory, artifact)
		}
		if err != nil {
			return result, err
		}
	}
	for _, tool := range []string{"java", "go", "rust"} {
		if c.Tools[tool] == "" {
			continue
		}
		request.phase(PhaseSDK)
		locked := selected.Tools[tool]
		if err = backend.ValidateSDKOrigin(tool, locked.URL); err != nil {
			return result, err
		}
		downloader := &backend.Node{Client: backend.SDKClient(node.Client), SegmentedSDK: true}
		archive, e := s.nodeArchive(ctx, downloader, backend.NodeArtifact{URL: locked.URL, SHA256: locked.SHA256}, operationDirectory)
		if e != nil {
			return result, e
		}
		err = backend.PrepareSDK(ctx, archive, directory, tool, platform, locked)
		os.Remove(archive)
		if err != nil {
			return result, err
		}
	}
	var pythonExecutable string
	if c.Tools["python"] != "" {
		request.phase(PhasePython)
		var python string
		if pythonProvider == "python.org" {
			locked := selected.Tools["python"]
			if err = backend.ValidatePythonOrigin(locked.URL); err != nil {
				return result, err
			}
			downloader := &backend.Node{Client: backend.SDKClient(node.Client), SegmentedSDK: true}
			archive, e := s.nodeArchive(ctx, downloader, backend.NodeArtifact{URL: locked.URL, SHA256: locked.SHA256}, operationDirectory)
			if e != nil {
				return result, e
			}
			python, err = backend.PrepareOfficialPython(ctx, archive, directory, locked.Version)
			os.Remove(archive)
			if err == nil {
				err = uv.CreateVenv(ctx, python, filepath.Join(directory, "venv"), filepath.Join(storage.Cache, "uv"))
			}
			pythonExecutable = pythonEntry(filepath.Join(directory, "venv"), platform)
		} else {
			python, pythonExecutable, err = s.preparePython(ctx, uv, selected.Tools["python"].Version, work, directory, platform)
		}
		if err != nil {
			return result, fmt.Errorf("SYNC_FAILED: %w", err)
		}
		if c.Python != nil {
			projectConfig := c
			before, err := pythonInputs(root, c, request.Locked)
			if err != nil {
				return result, err
			}
			if !reflect.DeepEqual(before, inputs) {
				return result, fmt.Errorf("INPUT_CHANGED: Python inputs changed during runtime preparation")
			}
			defer func() {
				after, err := pythonInputs(root, projectConfig, false)
				if err == nil {
					result.NativeLockChanged = after.UVLockSHA256 != inputs.UVLockSHA256
				}
			}()
			project, err := config.Within(root, c.Python.Project)
			if err != nil {
				return result, err
			}
			projectRequest := backend.PythonProjectRequest{Project: project, Python: python, Venv: filepath.Join(directory, "venv"), Cache: filepath.Join(storage.Cache, "uv"), Groups: c.Python.Groups, Locked: request.Locked, AllowBuild: request.AllowBuild}
			if inputs.WorkspaceRoot != "" {
				projectRequest.ConfigProject, err = config.Within(root, inputs.WorkspaceRoot)
				if err != nil {
					return result, err
				}
			}
			request.phase(PhaseDependencies)
			err = uv.SyncPythonProject(ctx, projectRequest)
			var needsInput *config.NeedsInput
			if err != nil && !request.AllowBuild && request.ConfirmBuild != nil && errors.As(err, &needsInput) {
				pendingInputs, inputErr := pythonInputs(root, c, request.Locked)
				if inputErr != nil {
					return result, inputErr
				}
				if !samePythonManifest(inputs, pendingInputs) || (request.Locked && !reflect.DeepEqual(inputs, pendingInputs)) {
					return result, fmt.Errorf("INPUT_CHANGED: Python inputs changed before build confirmation")
				}
				allowed, confirmErr := request.ConfirmBuild(needsInput.Message)
				if confirmErr != nil {
					return result, confirmErr
				}
				if allowed {
					confirmedInputs, inputErr := pythonInputs(root, c, request.Locked)
					confirmedConfig, configErr := s.loadDeclaration(root)
					confirmedDigest := ""
					if configErr == nil {
						confirmedDigest, configErr = config.Digest(confirmedConfig)
					}
					confirmedLock, lockErr := config.ReadLock(lockPath)
					if inputErr != nil || configErr != nil || lockErr != nil || confirmedDigest != digest || !reflect.DeepEqual(confirmedInputs, pendingInputs) || !reflect.DeepEqual(confirmedLock, lock) {
						return result, fmt.Errorf("INPUT_CHANGED: inputs changed while waiting for build confirmation; retry sync")
					}
					projectRequest.AllowBuild = true
					err = uv.SyncPythonProject(ctx, projectRequest)
				}
			}
			if err != nil {
				return result, fmt.Errorf("SYNC_FAILED: %w", err)
			}
			after, err := pythonInputs(root, c, true)
			if err != nil {
				return result, err
			}
			if !samePythonManifest(inputs, after) || (request.Locked && !reflect.DeepEqual(inputs, after)) {
				return result, fmt.Errorf("INPUT_CHANGED: Python inputs changed during preparation")
			}
			selected.Python = after
		}
	}
	// All backend calls are finished. Only local validation and publication may
	// follow this boundary; the workspace lock still excludes another writer.
	// macOS does not yet provide the same descendant-completion guarantee.
	if platform == "windows-amd64" || platform == "linux-amd64-glibc" {
		if err = store.ConfirmOperationTreesDone(ctx, generationID); err != nil {
			return result, err
		}
	}
	request.phase(PhaseVerifying)
	c, err = s.loadDeclaration(root)
	if err != nil {
		return result, err
	}
	latestDigest, err := config.Digest(c)
	if err != nil {
		return result, err
	}
	if latestDigest != digest {
		return result, fmt.Errorf("INPUT_CHANGED: configuration changed during preparation")
	}
	latestLock, lockErr := config.ReadLock(lockPath)
	if lockErr != nil || !reflect.DeepEqual(latestLock, lock) {
		return result, fmt.Errorf("INPUT_CHANGED: runtime lock changed during preparation; review %s and retry sync", filepath.Base(s.desiredLockPath(root)))
	}
	if !reflect.DeepEqual(lock.Platforms[platform], selected) {
		lock.Platforms[platform] = selected
		if request.Locked {
			return result, fmt.Errorf("LOCK_OUT_OF_DATE: native inputs changed")
		}
		if err = config.ReplaceLock(lockPath, lock); err != nil {
			return result, err
		}
		result.LockChanged = true
	}
	finalInputs, err := pythonInputs(root, c, true)
	if err != nil {
		return result, err
	}
	if !reflect.DeepEqual(finalInputs, selected.Python) {
		return result, fmt.Errorf("INPUT_CHANGED: Python inputs changed before publication")
	}
	if len(c.Tools) == 0 {
		if err = os.MkdirAll(directory, 0700); err != nil {
			return result, err
		}
	}
	if err = config.WriteSnapshot(directory, c); err != nil {
		return result, err
	}
	if err = config.WriteNewLock(filepath.Join(directory, "myenv.lock"), lock); err != nil {
		return result, err
	}
	if err = config.MarkComplete(directory, digest); err != nil {
		return result, err
	}
	contentDigest, err := config.GenerationDigest(ctx, directory)
	if err != nil {
		return result, fmt.Errorf("SYNC_FAILED: capture generation evidence: %w", err)
	}
	// Content capture may be slow for large environments. Recheck external
	// desired inputs after it, rather than extending the prior validation window.
	latestConfig, err := s.loadDeclaration(root)
	if err != nil {
		return result, err
	}
	latestDigest, err = config.Digest(latestConfig)
	if err != nil {
		return result, err
	}
	latestLock, err = config.ReadLock(s.desiredLockPath(root))
	if err != nil {
		return result, err
	}
	latestInputs, err := pythonInputs(root, latestConfig, false)
	if err != nil {
		return result, err
	}
	if latestDigest != digest || !reflect.DeepEqual(latestLock, lock) || !reflect.DeepEqual(latestInputs, finalInputs) {
		return result, fmt.Errorf("INPUT_CHANGED: desired inputs changed during evidence capture")
	}
	generation := state.Generation{EmptyProfile: s.Profile && len(c.Tools) == 0, ID: generationID, Directory: directory, InputDigest: digest, NodeExecutable: executable, PythonExecutable: pythonExecutable}
	for _, tool := range []string{"java", "go", "rust"} {
		if c.Tools[tool] != "" {
			generation.PreparedEntry = backend.SDKEntry(directory, tool, platform)
		}
	}
	previous := ""
	if active != nil {
		previous = active.ID
	}
	request.phase(PhasePublishing)
	if err = store.PublishWithEvidence(ctx, generation, previous, contentDigest); err != nil {
		return result, err
	}
	result.Changed = true
	result.Generation = &generation
	return result, nil
}

func syncPlan(platform string, selected config.PlatformLock, needsApply bool) *SyncPlan {
	plan := &SyncPlan{Tools: selected.Tools, Platform: platform, Node: selected.Tools["node"], NeedsApply: needsApply}
	if python, ok := selected.Tools["python"]; ok {
		plan.Python = &python
	}
	return plan
}

func runtimeMatches(active *state.Generation, platform string, desired config.RuntimeLock) bool {
	if desired.Version == "" {
		return false
	}
	applied, err := config.ReadLock(filepath.Join(active.Directory, "myenv.lock"))
	if err != nil || applied.ConfigDigest != active.InputDigest {
		return false
	}
	return applied.Platforms[platform].Tools["node"] == desired
}

func snapshotHealthy(active *state.Generation, root string) bool {
	_, healthy := readHealthySnapshot(active, root)
	return healthy
}

func readHealthySnapshot(active *state.Generation, root string) (*config.Config, bool) {
	if err := config.CheckComplete(active.Directory, active.InputDigest); err != nil {
		return nil, false
	}
	c, err := config.ReadSnapshot(active.Directory, root)
	if err != nil {
		return nil, false
	}
	digest, err := config.Digest(c)
	if err != nil || digest != active.InputDigest {
		return nil, false
	}
	return c, true
}
