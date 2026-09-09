package cli

import (
	"fmt"
	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/core"
	"os"
	"path/filepath"
)

type environmentConfigError struct {
	variable string
	cause    error
}

func (e *environmentConfigError) Error() string { return fmt.Sprintf("%s: %v", e.variable, e.cause) }
func (e *environmentConfigError) Unwrap() error { return e.cause }
func (e *environmentConfigError) nextAction() string {
	return "Correct or unset " + e.variable + " in the calling environment, then retry."
}

// runtimeService is constructed only by installing commands. Help/status/run
// do not need user storage resolution. Integrations supply an isolated namespace.
func runtimeService(global bool, userConfigDirectory string) (*core.Service, error) {
	data, cache := "", ""
	if userConfigDirectory != "" {
		data = filepath.Join(userConfigDirectory, "test-storage", "data")
		cache = filepath.Join(userConfigDirectory, "test-storage", "cache")
	}
	storage, err := config.ResolveUserStorage(data, cache)
	if err != nil {
		return nil, err
	}
	service := &core.Service{Profile: global, UserConfigDirectory: userConfigDirectory, Storage: &storage}
	if certificateFile := os.Getenv("SSL_CERT_FILE"); certificateFile != "" {
		client, err := backend.DownloadClient(certificateFile)
		if err != nil {
			return nil, &environmentConfigError{"SSL_CERT_FILE", err}
		}
		service.Node = backend.NewNode()
		service.Node.Client = client
	}
	if mirror := os.Getenv("MYENV_NODE_MIRROR"); mirror != "" {
		base, err := config.MirrorBaseURL(mirror)
		if err != nil {
			return nil, &environmentConfigError{"MYENV_NODE_MIRROR", err}
		}
		if service.Node == nil {
			service.Node = backend.NewNode()
		}
		service.Node.BaseURL = base
	}
	service.UVMirror, err = config.MirrorBaseURL(os.Getenv("MYENV_UV_MIRROR"))
	if err != nil {
		return nil, &environmentConfigError{"MYENV_UV_MIRROR", err}
	}
	service.PythonMirror, err = config.MirrorBaseURL(os.Getenv("MYENV_PYTHON_MIRROR"))
	if err != nil {
		return nil, &environmentConfigError{"MYENV_PYTHON_MIRROR", err}
	}
	return service, nil
}

// Only custom transports are owned by this invocation. Do not close the shared
// default transport used by unrelated callers in an embedding process.
func closeRuntimeService(service *core.Service) {
	if service.Node != nil && service.Node.Client != nil && service.Node.Client.Transport != nil {
		service.Node.Client.CloseIdleConnections()
	}
}
