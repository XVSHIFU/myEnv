package core

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"myenv/internal/config"
	"myenv/internal/runner"
	"myenv/internal/state"
)

type Diagnosis struct {
	*Status
	CheckLevel       string `json:"check_level"`
	ContentEvidence  string `json:"content_evidence,omitempty"`
	EvidenceScope    string `json:"evidence_scope,omitempty"`
	EvidenceDetail   string `json:"evidence_detail,omitempty"`
	AppliedNode      string `json:"applied_node,omitempty"`
	ParentPathNode   string `json:"parent_path_node,omitempty"`
	RunPathNode      string `json:"run_path_node,omitempty"`
	AppliedPython    string `json:"applied_python,omitempty"`
	ParentPathPython string `json:"parent_path_python,omitempty"`
	RunPathPython    string `json:"run_path_python,omitempty"`
	PathDiffers      bool   `json:"path_differs"`
	RunProtection    string `json:"run_protection"`
	ProtectionDetail string `json:"protection_detail,omitempty"`
}

// DoctorDeep measures the observed generation without acquiring leases or
// changing storage. Concurrent removal is reported as unavailable evidence.
func (s *Service) DoctorDeep(ctx context.Context, directory string, environment []string) (*Diagnosis, error) {
	r, err := s.Doctor(ctx, directory, environment)
	if err != nil {
		return nil, err
	}
	r.CheckLevel = "deep"
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.ContentEvidence = "missing"
	r.EvidenceScope = "Generation-owned files and link targets; excludes Python bytecode caches, external runtime contents and ACLs."
	if r.Generation == nil {
		return r, nil
	}
	store, err := state.OpenReadOnly(ctx, filepath.Join(s.workDirectory(r.Project), "state.db"))
	if err != nil {
		return nil, err
	}
	defer store.Close()
	baseline, err := store.GenerationEvidence(ctx, r.Generation.ID)
	if err == nil && baseline == "" {
		r.EvidenceDetail = "This generation has no recorded content baseline."
		return r, nil
	}
	var actual string
	if err == nil {
		actual, err = config.GenerationDigest(ctx, r.Generation.Directory)
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		r.ContentEvidence = "unavailable"
		r.EvidenceDetail = err.Error()
		r.NextAction = "Review the evidence error and retry myenv doctor --deep."
	} else if actual != baseline {
		r.ContentEvidence = "mismatch"
		r.EvidenceDetail = "Generation content differs from its publication baseline."
		r.Environment = "content_changed"
		r.NextAction = s.commandHint("Run myenv sync --rebuild --locked to prepare a fresh generation from the current locks.")
	} else {
		r.ContentEvidence = "matched"
	}
	active, err := store.Active(ctx)
	if err != nil {
		return nil, err
	}
	if active == nil || active.ID != r.Generation.ID {
		r.ContentEvidence = "unavailable"
		r.EvidenceDetail = "The active generation changed during this check."
		r.NextAction = "Retry myenv doctor --deep."
	}
	if s.Profile {
		r.NextAction = strings.ReplaceAll(r.NextAction, "myenv doctor --deep", "myenv doctor --global --deep")
	}
	return r, nil
}

func (s *Service) Doctor(ctx context.Context, directory string, parentEnvironment []string) (*Diagnosis, error) {
	status, err := s.Status(ctx, directory)
	if err != nil {
		return nil, err
	}
	result := &Diagnosis{Status: status, CheckLevel: "quick", RunProtection: "none"}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	database := filepath.Join(s.workDirectory(status.Project), "state.db")
	if _, statErr := os.Stat(database); statErr == nil {
		store, err := state.OpenReadOnly(ctx, database)
		if err != nil {
			return nil, err
		}
		protected, err := store.HasLeases(ctx)
		closeErr := store.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if protected {
			result.RunProtection = "present"
			result.ProtectionDetail = s.commandHint("Run protection records remain; this does not prove that their processes are still running. Use myenv clean --dry-run to inspect what can be safely reclaimed.")
		}
	} else if !os.IsNotExist(statErr) {
		return nil, statErr
	}
	cwd, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	windows := runtime.GOOS == "windows"
	result.ParentPathNode, _ = runner.Lookup("node", cwd, parentEnvironment, windows)
	result.ParentPathPython, _ = runner.Lookup("python", cwd, parentEnvironment, windows)
	if status.Generation == nil {
		return result, nil
	}
	result.AppliedNode = status.Generation.NodeExecutable
	result.AppliedPython = status.Generation.PythonExecutable
	applied, err := config.ReadSnapshot(status.Generation.Directory, status.Project)
	if err != nil {
		return result, nil
	} // Status already reports the incomplete snapshot.
	var bins []string
	for _, entry := range []string{result.AppliedPython, result.AppliedNode} {
		if entry != "" {
			bins = append(bins, filepath.Dir(entry))
		}
	}
	environment, err := runner.Environment(parentEnvironment, applied.Env, bins, windows)
	if err != nil {
		return nil, err
	}
	result.RunPathNode, _ = runner.Lookup("node", cwd, environment, windows)
	result.RunPathPython, _ = runner.Lookup("python", cwd, environment, windows)
	// SameFile handles Windows path casing without changing PATH lookup rules.
	for _, paths := range [][2]string{{result.ParentPathNode, result.AppliedNode}, {result.ParentPathPython, result.AppliedPython}} {
		if paths[1] == "" {
			continue
		}
		a, aerr := os.Stat(paths[0])
		b, berr := os.Stat(paths[1])
		result.PathDiffers = result.PathDiffers || aerr != nil || berr != nil || !os.SameFile(a, b)
	}
	return result, nil
}
