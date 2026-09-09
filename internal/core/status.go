package core

import (
	"context"
	"os"
	"path/filepath"
	"reflect"

	"myenv/internal/config"
	"myenv/internal/runner"
	"myenv/internal/state"
)

type Status struct {
	Scope       string            `json:"scope"`
	Project     string            `json:"project"`
	Tools       map[string]string `json:"tools"`
	Environment string            `json:"environment"`
	Generation  *state.Generation `json:"generation,omitempty"`
	NextAction  string            `json:"next_action"`
}

// Status reads small local inputs and the active reference without installation,
// recovery, runtime execution, or lease acquisition.
func (s *Service) Status(ctx context.Context, directory string) (*Status, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, err := s.resolveRoot(directory)
	if err != nil {
		return nil, err
	}
	c, err := s.loadDeclaration(root)
	if err != nil {
		return nil, err
	}
	result := &Status{Project: root, Tools: c.Tools, Environment: "not_ready", NextAction: "Run myenv sync."}
	result.Scope = "project"
	if s.Profile {
		result.Scope = "profile"
	}
	defer func() { result.NextAction = s.commandHint(result.NextAction) }()
	database := filepath.Join(s.workDirectory(root), "state.db")
	if _, err = os.Stat(database); os.IsNotExist(err) {
		return result, nil
	} else if err != nil {
		return nil, err
	}
	store, err := state.OpenReadOnly(ctx, database)
	if err != nil {
		return nil, err
	}
	defer store.Close()
	result.Generation, err = store.Active(ctx)
	if err != nil {
		return nil, err
	}
	if result.Generation == nil {
		return result, nil
	}
	g := result.Generation
	result.Environment = "incomplete"
	appliedConfig, healthy := readHealthySnapshot(g, root)
	if !healthy || !entriesHealthy(g, appliedConfig.Tools) {
		return result, nil
	}
	digest, err := config.Digest(c)
	if err != nil {
		return nil, err
	}
	result.Environment = "drifted"
	result.NextAction = "Run myenv sync, or myenv run --current <command> to use the applied environment."
	if digest != g.InputDigest {
		return result, nil
	}
	lock, err := config.ReadLock(s.desiredLockPath(root))
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	platform, err := runner.Platform()
	if err != nil {
		return nil, err
	}
	appliedLock, err := config.ReadLock(filepath.Join(g.Directory, "myenv.lock"))
	if err != nil || appliedLock.ConfigDigest != g.InputDigest || !platformLockHealthy(appliedLock.Platforms[platform], appliedConfig) {
		result.Environment = "incomplete"
		result.NextAction = "Run myenv sync."
		return result, nil
	}
	inputs, err := pythonInputs(root, c, false)
	if err != nil {
		return nil, err
	}
	if lock.ConfigDigest != digest || !reflect.DeepEqual(appliedLock.Platforms[platform], lock.Platforms[platform]) || !reflect.DeepEqual(inputs, appliedLock.Platforms[platform].Python) {
		return result, nil
	}
	result.Environment = "ready"
	result.NextAction = "Run myenv run <command>."
	return result, nil
}
